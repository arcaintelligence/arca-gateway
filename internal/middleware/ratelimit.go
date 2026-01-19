package middleware

import (
	"sync"
	"time"

	"github.com/arcaintelligence/arca-gateway/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RateLimiter implementa rate limiting com sliding window
type RateLimiter struct {
	mu              sync.RWMutex
	requests        map[string]*slidingWindow
	limit           int
	windowSize      time.Duration
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// slidingWindow representa uma janela deslizante para rate limiting
type slidingWindow struct {
	timestamps []time.Time
	mu         sync.Mutex
}

// RateLimitConfig configuração do rate limiter
type RateLimitConfig struct {
	// Limite de requests por janela
	Limit int
	// Tamanho da janela (ex: 1 minuto)
	WindowSize time.Duration
	// Intervalo de limpeza de entradas antigas
	CleanupInterval time.Duration
	// Função para extrair a chave de identificação (tenant_id, user_id, IP)
	KeyExtractor func(*fiber.Ctx) string
	// Limites customizados por tenant/plano
	CustomLimits map[string]int
}

// NewRateLimiter cria um novo rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	if config.Limit == 0 {
		config.Limit = 1000 // default: 1000 req/min
	}
	if config.WindowSize == 0 {
		config.WindowSize = time.Minute
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = time.Minute
	}

	rl := &RateLimiter{
		requests:        make(map[string]*slidingWindow),
		limit:           config.Limit,
		windowSize:      config.WindowSize,
		cleanupInterval: config.CleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Iniciar goroutine de limpeza
	go rl.cleanup()

	return rl
}

// cleanup remove entradas antigas periodicamente
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for key, window := range rl.requests {
				window.mu.Lock()
				// Remover timestamps antigos
				cutoff := now.Add(-rl.windowSize)
				newTimestamps := make([]time.Time, 0)
				for _, ts := range window.timestamps {
					if ts.After(cutoff) {
						newTimestamps = append(newTimestamps, ts)
					}
				}
				window.timestamps = newTimestamps
				
				// Remover janela vazia
				if len(window.timestamps) == 0 {
					delete(rl.requests, key)
				}
				window.mu.Unlock()
			}
			rl.mu.Unlock()
		case <-rl.stopCleanup:
			return
		}
	}
}

// Stop para o rate limiter
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
}

// Allow verifica se uma request é permitida
func (rl *RateLimiter) Allow(key string, customLimit int) (bool, int, time.Duration) {
	now := time.Now()
	cutoff := now.Add(-rl.windowSize)
	
	limit := rl.limit
	if customLimit > 0 {
		limit = customLimit
	}

	rl.mu.Lock()
	window, exists := rl.requests[key]
	if !exists {
		window = &slidingWindow{
			timestamps: make([]time.Time, 0),
		}
		rl.requests[key] = window
	}
	rl.mu.Unlock()

	window.mu.Lock()
	defer window.mu.Unlock()

	// Remover timestamps antigos
	newTimestamps := make([]time.Time, 0)
	for _, ts := range window.timestamps {
		if ts.After(cutoff) {
			newTimestamps = append(newTimestamps, ts)
		}
	}
	window.timestamps = newTimestamps

	// Verificar limite
	remaining := limit - len(window.timestamps)
	if remaining <= 0 {
		// Calcular tempo até reset
		if len(window.timestamps) > 0 {
			oldestInWindow := window.timestamps[0]
			resetIn := oldestInWindow.Add(rl.windowSize).Sub(now)
			return false, 0, resetIn
		}
		return false, 0, rl.windowSize
	}

	// Adicionar timestamp atual
	window.timestamps = append(window.timestamps, now)
	
	return true, remaining - 1, 0
}

// RateLimitMiddleware cria um middleware de rate limiting
func RateLimitMiddleware(config RateLimitConfig) fiber.Handler {
	limiter := NewRateLimiter(config)

	return func(c *fiber.Ctx) error {
		// Extrair chave de identificação
		var key string
		if config.KeyExtractor != nil {
			key = config.KeyExtractor(c)
		} else {
			// Default: usar tenant_id se autenticado, senão IP
			tenantID := GetTenantID(c)
			if tenantID != uuid.Nil {
				key = "tenant:" + tenantID.String()
			} else {
				key = "ip:" + c.IP()
			}
		}

		// Verificar limite customizado
		customLimit := 0
		if config.CustomLimits != nil {
			if limit, ok := config.CustomLimits[key]; ok {
				customLimit = limit
			}
		}

		// Verificar rate limit
		allowed, remaining, resetIn := limiter.Allow(key, customLimit)

		// Adicionar headers de rate limit
		c.Set("X-RateLimit-Limit", string(rune(config.Limit)))
		c.Set("X-RateLimit-Remaining", string(rune(remaining)))
		
		if !allowed {
			c.Set("X-RateLimit-Reset", resetIn.String())
			c.Set("Retry-After", string(rune(int(resetIn.Seconds()))))
			return response.TooManyRequests(c, "Rate limit exceeded. Please try again later.")
		}

		return c.Next()
	}
}

// TenantRateLimitMiddleware rate limiting por tenant com limites baseados no plano
func TenantRateLimitMiddleware(baseLimits map[string]int) fiber.Handler {
	// Limites por plano
	planLimits := map[string]int{
		"free":       100,   // 100 req/min
		"starter":    500,   // 500 req/min
		"pro":        2000,  // 2000 req/min
		"enterprise": 10000, // 10000 req/min
	}

	// Merge com limites customizados
	for k, v := range baseLimits {
		planLimits[k] = v
	}

	config := RateLimitConfig{
		Limit:      1000,
		WindowSize: time.Minute,
		KeyExtractor: func(c *fiber.Ctx) string {
			tenantID := GetTenantID(c)
			if tenantID != uuid.Nil {
				return "tenant:" + tenantID.String()
			}
			return "ip:" + c.IP()
		},
	}

	return RateLimitMiddleware(config)
}

// EndpointRateLimitMiddleware rate limiting específico por endpoint
func EndpointRateLimitMiddleware(limit int, window time.Duration) fiber.Handler {
	config := RateLimitConfig{
		Limit:      limit,
		WindowSize: window,
		KeyExtractor: func(c *fiber.Ctx) string {
			tenantID := GetTenantID(c)
			endpoint := c.Method() + ":" + c.Path()
			if tenantID != uuid.Nil {
				return "endpoint:" + tenantID.String() + ":" + endpoint
			}
			return "endpoint:" + c.IP() + ":" + endpoint
		},
	}

	return RateLimitMiddleware(config)
}

// BurstRateLimitMiddleware permite bursts mas limita média
type BurstRateLimiter struct {
	*RateLimiter
	burstSize int
}

// NewBurstRateLimiter cria um rate limiter com suporte a burst
func NewBurstRateLimiter(limit, burstSize int, window time.Duration) *BurstRateLimiter {
	return &BurstRateLimiter{
		RateLimiter: NewRateLimiter(RateLimitConfig{
			Limit:      limit + burstSize,
			WindowSize: window,
		}),
		burstSize: burstSize,
	}
}
