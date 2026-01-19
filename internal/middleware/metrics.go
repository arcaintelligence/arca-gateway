package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "arca_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "arca_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "arca_http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "arca_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// Business metrics
	activeUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "arca_active_users",
			Help: "Number of active users",
		},
	)

	huntingOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "arca_hunting_operations_total",
			Help: "Total number of hunting operations",
		},
		[]string{"tenant_id", "operation", "status"},
	)

	monitoringJobs = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "arca_monitoring_jobs_active",
			Help: "Number of active monitoring jobs",
		},
		[]string{"tenant_id"},
	)

	threatsDetected = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "arca_threats_detected_total",
			Help: "Total number of threats detected",
		},
		[]string{"tenant_id", "severity", "type"},
	)

	mcpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "arca_mcp_requests_total",
			Help: "Total number of MCP requests",
		},
		[]string{"tool", "action", "status"},
	)

	mcpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "arca_mcp_request_duration_seconds",
			Help:    "MCP request duration in seconds",
			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		},
		[]string{"tool", "action"},
	)

	rateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "arca_rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"tenant_id", "path"},
	)

	authFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "arca_auth_failures_total",
			Help: "Total number of authentication failures",
		},
		[]string{"reason"},
	)
)

// MetricsMiddleware middleware para coletar métricas
func MetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Route().Path
		method := c.Method()

		// Request size
		reqSize := float64(len(c.Body()))
		httpRequestSize.WithLabelValues(method, path).Observe(reqSize)

		// Process request
		err := c.Next()

		// Response metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Response().StatusCode())
		respSize := float64(len(c.Response().Body()))

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path).Observe(duration)
		httpResponseSize.WithLabelValues(method, path).Observe(respSize)

		return err
	}
}

// MetricsHandler retorna o handler do Prometheus
func MetricsHandler() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.Handler())
}

// =============================================================================
// BUSINESS METRICS HELPERS
// =============================================================================

// RecordHuntingOperation registra uma operação de hunting
func RecordHuntingOperation(tenantID, operation, status string) {
	huntingOperations.WithLabelValues(tenantID, operation, status).Inc()
}

// RecordMCPRequest registra uma requisição MCP
func RecordMCPRequest(tool, action, status string, duration time.Duration) {
	mcpRequestsTotal.WithLabelValues(tool, action, status).Inc()
	mcpRequestDuration.WithLabelValues(tool, action).Observe(duration.Seconds())
}

// RecordThreatDetected registra uma ameaça detectada
func RecordThreatDetected(tenantID, severity, threatType string) {
	threatsDetected.WithLabelValues(tenantID, severity, threatType).Inc()
}

// RecordRateLimitHit registra um hit de rate limit
func RecordRateLimitHit(tenantID, path string) {
	rateLimitHits.WithLabelValues(tenantID, path).Inc()
}

// RecordAuthFailure registra uma falha de autenticação
func RecordAuthFailure(reason string) {
	authFailures.WithLabelValues(reason).Inc()
}

// SetActiveUsers define o número de usuários ativos
func SetActiveUsers(count float64) {
	activeUsers.Set(count)
}

// SetActiveMonitoringJobs define o número de jobs de monitoramento ativos
func SetActiveMonitoringJobs(tenantID string, count float64) {
	monitoringJobs.WithLabelValues(tenantID).Set(count)
}
