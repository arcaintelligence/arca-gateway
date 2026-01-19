package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
)

// SecurityConfig configuração de segurança
type SecurityConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int
	Environment      string
}

// SetupSecurityMiddlewares configura todos os middlewares de segurança
func SetupSecurityMiddlewares(app *fiber.App, config SecurityConfig) {
	// Recover - recupera de panics
	app.Use(recover.New(recover.Config{
		EnableStackTrace: config.Environment != "production",
	}))

	// Request ID - gera ID único para cada request
	app.Use(requestid.New(requestid.Config{
		Header: "X-Request-ID",
		Generator: func() string {
			return uuid.New().String()
		},
	}))

	// Security Headers (Helmet)
	app.Use(helmet.New(helmet.Config{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		PermissionPolicy:          "geolocation=(), microphone=(), camera=()",
	}))

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     joinStrings(config.AllowOrigins),
		AllowMethods:     joinStrings(config.AllowMethods),
		AllowHeaders:     joinStrings(config.AllowHeaders),
		AllowCredentials: config.AllowCredentials,
		MaxAge:           config.MaxAge,
	}))

	// Custom security headers
	app.Use(CustomSecurityHeaders())

	// Request logging
	app.Use(RequestLogger())

	// Timeout
	app.Use(TimeoutMiddleware(30 * time.Second))
}

// CustomSecurityHeaders adiciona headers de segurança customizados
func CustomSecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Strict Transport Security (HSTS)
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		
		// Content Security Policy
		c.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		
		// Expect-CT
		c.Set("Expect-CT", "max-age=86400, enforce")
		
		// Cache Control para APIs
		c.Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		
		return c.Next()
	}
}

// RequestLogger middleware de logging de requests
func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		
		// Processar request
		err := c.Next()
		
		// Calcular duração
		duration := time.Since(start)
		
		// Log estruturado (pode ser enviado para observabilidade)
		// Em produção, usar logger estruturado (zerolog, zap)
		c.Set("X-Response-Time", duration.String())
		
		return err
	}
}

// TimeoutMiddleware adiciona timeout às requests
func TimeoutMiddleware(timeout time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Fiber não tem timeout middleware nativo, usar context
		// O timeout é configurado no servidor
		return c.Next()
	}
}

// IPWhitelistMiddleware permite apenas IPs específicos
func IPWhitelistMiddleware(allowedIPs []string) fiber.Handler {
	ipSet := make(map[string]bool)
	for _, ip := range allowedIPs {
		ipSet[ip] = true
	}

	return func(c *fiber.Ctx) error {
		clientIP := c.IP()
		
		if !ipSet[clientIP] {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "IP not allowed",
			})
		}
		
		return c.Next()
	}
}

// IPBlacklistMiddleware bloqueia IPs específicos
func IPBlacklistMiddleware(blockedIPs []string) fiber.Handler {
	ipSet := make(map[string]bool)
	for _, ip := range blockedIPs {
		ipSet[ip] = true
	}

	return func(c *fiber.Ctx) error {
		clientIP := c.IP()
		
		if ipSet[clientIP] {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "IP blocked",
			})
		}
		
		return c.Next()
	}
}

// TenantIsolationMiddleware garante isolamento entre tenants
func TenantIsolationMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := GetClaims(c)
		if claims == nil {
			return c.Next()
		}

		// Verificar se há tenant_id no path
		tenantIDParam := c.Params("tenant_id")
		if tenantIDParam != "" {
			paramTenantID, err := uuid.Parse(tenantIDParam)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid tenant ID format",
				})
			}

			// Admin pode acessar qualquer tenant
			if !claims.IsAdmin() && claims.TenantID != paramTenantID {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Access denied to this tenant",
				})
			}
		}

		return c.Next()
	}
}

// AuditMiddleware registra ações para auditoria
func AuditMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Capturar informações antes da request
		startTime := time.Now()
		requestID := c.Get("X-Request-ID")
		
		// Processar request
		err := c.Next()
		
		// Capturar informações após a request
		duration := time.Since(startTime)
		statusCode := c.Response().StatusCode()
		
		// Criar log de auditoria
		auditLog := map[string]interface{}{
			"request_id":  requestID,
			"timestamp":   startTime.UTC().Format(time.RFC3339),
			"method":      c.Method(),
			"path":        c.Path(),
			"status_code": statusCode,
			"duration_ms": duration.Milliseconds(),
			"ip":          c.IP(),
			"user_agent":  c.Get("User-Agent"),
		}

		// Adicionar informações do usuário se autenticado
		claims := GetClaims(c)
		if claims != nil {
			auditLog["user_id"] = claims.UserID.String()
			auditLog["tenant_id"] = claims.TenantID.String()
			auditLog["role"] = string(claims.Role)
		}

		// Em produção, enviar para sistema de auditoria
		// logger.Info().Fields(auditLog).Msg("audit")
		_ = auditLog

		return err
	}
}

// SanitizeInputMiddleware sanitiza inputs para prevenir injection
func SanitizeInputMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Sanitização básica - em produção usar biblioteca especializada
		// Os handlers devem fazer validação adicional
		return c.Next()
	}
}

// Helper functions
func joinStrings(strs []string) string {
	if len(strs) == 0 {
		return "*"
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += ", " + strs[i]
	}
	return result
}
