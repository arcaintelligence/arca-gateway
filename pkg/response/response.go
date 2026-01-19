package response

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Response estrutura padrão de resposta da API
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// ErrorInfo informações de erro
type ErrorInfo struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// Meta informações de paginação e metadata
type Meta struct {
	Page       int   `json:"page,omitempty"`
	PerPage    int   `json:"per_page,omitempty"`
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

// PaginatedData dados com paginação
type PaginatedData struct {
	Items interface{} `json:"items"`
	Meta  Meta        `json:"meta"`
}

// =============================================================================
// SUCCESS RESPONSES
// =============================================================================

// Success retorna uma resposta de sucesso
func Success(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Success:   true,
		Data:      data,
		RequestID: c.Get("X-Request-ID"),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// Created retorna uma resposta de recurso criado
func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Success:   true,
		Data:      data,
		RequestID: c.Get("X-Request-ID"),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// Accepted retorna uma resposta de requisição aceita (async)
func Accepted(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusAccepted).JSON(Response{
		Success:   true,
		Data:      data,
		RequestID: c.Get("X-Request-ID"),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// NoContent retorna uma resposta sem conteúdo
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// Paginated retorna uma resposta paginada
func Paginated(c *fiber.Ctx, items interface{}, page, perPage int, total int64) error {
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return c.Status(fiber.StatusOK).JSON(Response{
		Success: true,
		Data: PaginatedData{
			Items: items,
			Meta: Meta{
				Page:       page,
				PerPage:    perPage,
				Total:      total,
				TotalPages: totalPages,
			},
		},
		RequestID: c.Get("X-Request-ID"),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// =============================================================================
// ERROR RESPONSES
// =============================================================================

// Error retorna uma resposta de erro genérica
func Error(c *fiber.Ctx, statusCode int, code, message string) error {
	return c.Status(statusCode).JSON(Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
		RequestID: c.Get("X-Request-ID"),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// ErrorWithDetails retorna uma resposta de erro com detalhes
func ErrorWithDetails(c *fiber.Ctx, statusCode int, code, message string, details map[string]string) error {
	return c.Status(statusCode).JSON(Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
		RequestID: c.Get("X-Request-ID"),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// BadRequest retorna erro 400
func BadRequest(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusBadRequest, "BAD_REQUEST", message)
}

// BadRequestWithDetails retorna erro 400 com detalhes
func BadRequestWithDetails(c *fiber.Ctx, message string, details map[string]string) error {
	return ErrorWithDetails(c, fiber.StatusBadRequest, "BAD_REQUEST", message, details)
}

// Unauthorized retorna erro 401
func Unauthorized(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden retorna erro 403
func Forbidden(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusForbidden, "FORBIDDEN", message)
}

// NotFound retorna erro 404
func NotFound(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusNotFound, "NOT_FOUND", message)
}

// Conflict retorna erro 409
func Conflict(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusConflict, "CONFLICT", message)
}

// UnprocessableEntity retorna erro 422
func UnprocessableEntity(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusUnprocessableEntity, "UNPROCESSABLE_ENTITY", message)
}

// TooManyRequests retorna erro 429
func TooManyRequests(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusTooManyRequests, "TOO_MANY_REQUESTS", message)
}

// InternalServerError retorna erro 500
func InternalServerError(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message)
}

// ServiceUnavailable retorna erro 503
func ServiceUnavailable(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", message)
}

// =============================================================================
// VALIDATION HELPERS
// =============================================================================

// ValidationError estrutura para erros de validação
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors retorna erros de validação
func ValidationErrors(c *fiber.Ctx, errors []ValidationError) error {
	details := make(map[string]string)
	for _, err := range errors {
		details[err.Field] = err.Message
	}
	return ErrorWithDetails(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", details)
}

// =============================================================================
// ASYNC RESPONSE HELPERS
// =============================================================================

// AsyncJobResponse resposta para jobs assíncronos
type AsyncJobResponse struct {
	JobID     uuid.UUID `json:"job_id"`
	Status    string    `json:"status"`
	StatusURL string    `json:"status_url"`
	Message   string    `json:"message,omitempty"`
}

// AsyncJob retorna resposta de job assíncrono
func AsyncJob(c *fiber.Ctx, jobID uuid.UUID, statusURL string) error {
	return Accepted(c, AsyncJobResponse{
		JobID:     jobID,
		Status:    "pending",
		StatusURL: statusURL,
		Message:   "Job accepted and queued for processing",
	})
}

// =============================================================================
// HEALTH CHECK RESPONSE
// =============================================================================

// HealthResponse resposta de health check
type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services,omitempty"`
}

// Health retorna resposta de health check
func Health(c *fiber.Ctx, version string, services map[string]string) error {
	status := "healthy"
	for _, svcStatus := range services {
		if svcStatus != "healthy" {
			status = "degraded"
			break
		}
	}

	return c.Status(fiber.StatusOK).JSON(HealthResponse{
		Status:    status,
		Version:   version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  services,
	})
}
