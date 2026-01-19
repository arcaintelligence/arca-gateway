package handlers

import (
	"github.com/arcaintelligence/arca-gateway/internal/mcp"
	"github.com/arcaintelligence/arca-gateway/internal/middleware"
	"github.com/arcaintelligence/arca-gateway/internal/models"
	"github.com/arcaintelligence/arca-gateway/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// HuntingHandler handlers de hunting e análise
type HuntingHandler struct {
	mcpClient *mcp.MCPClient
}

// NewHuntingHandler cria um novo handler de hunting
func NewHuntingHandler(mcpClient *mcp.MCPClient) *HuntingHandler {
	return &HuntingHandler{
		mcpClient: mcpClient,
	}
}

// HuntRequest request de hunting
type HuntRequest struct {
	Target       string   `json:"target"`
	IncludeLeaks bool     `json:"include_leaks"`
	DeepAnalysis bool     `json:"deep_analysis"`
	Keywords     []string `json:"keywords,omitempty"`
	ClientID     *string  `json:"client_id,omitempty"`
}

// ScanRequest request de scan
type ScanURLRequest struct {
	URL             string   `json:"url"`
	CaptureTypes    []string `json:"capture_types,omitempty"`
	FollowRedirects bool     `json:"follow_redirects"`
	ClientID        *string  `json:"client_id,omitempty"`
}

// AnalyzeRequest request de análise
type AnalyzeURLRequest struct {
	URL          string  `json:"url"`
	IncludeLeaks bool    `json:"include_leaks"`
	DeepAnalysis bool    `json:"deep_analysis"`
	ClientID     *string `json:"client_id,omitempty"`
}

// LeakSearchRequest request de busca de vazamentos
type LeakSearchReq struct {
	Query      string  `json:"query"`
	Type       string  `json:"type"`
	MaxResults int     `json:"max_results,omitempty"`
	ClientID   *string `json:"client_id,omitempty"`
}

// =============================================================================
// HUNTING HANDLERS
// =============================================================================

// Hunt executa uma operação de hunting
func (h *HuntingHandler) Hunt(c *fiber.Ctx) error {
	claims := getClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	// Verificar scope
	if !claims.HasAnyScope(models.ScopeHuntingWrite) && !claims.IsAdmin() {
		return response.Forbidden(c, "Missing scope: hunting:write")
	}

	var req HuntRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Target == "" {
		return response.BadRequest(c, "Target is required")
	}

	// Preparar client_id
	var clientID *uuid.UUID
	if req.ClientID != nil {
		parsed, err := uuid.Parse(*req.ClientID)
		if err == nil {
			clientID = &parsed
		}
	}

	// Criar request MCP
	mcpReq := &mcp.MCPRequest{
		RequestID: c.Get("X-Request-ID"),
		TenantID:  claims.TenantID,
		ClientID:  clientID,
		UserID:    claims.UserID,
		Scopes:    scopesToStrings(claims.Scopes),
	}

	huntReq := &mcp.HuntRequest{
		Target:       req.Target,
		IncludeLeaks: req.IncludeLeaks,
		DeepAnalysis: req.DeepAnalysis,
		Keywords:     req.Keywords,
	}

	// Executar hunting via MCP
	result, err := h.mcpClient.Hunt(c.Context(), mcpReq, huntReq)
	if err != nil {
		return handleMCPError(c, err)
	}

	return response.Success(c, result)
}

// ScanURL executa um scan de URL
func (h *HuntingHandler) ScanURL(c *fiber.Ctx) error {
	claims := getClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	if !claims.HasAnyScope(models.ScopeHuntingWrite) && !claims.IsAdmin() {
		return response.Forbidden(c, "Missing scope: hunting:write")
	}

	var req ScanURLRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.URL == "" {
		return response.BadRequest(c, "URL is required")
	}

	// Capture types padrão
	if len(req.CaptureTypes) == 0 {
		req.CaptureTypes = []string{"png", "html"}
	}

	var clientID *uuid.UUID
	if req.ClientID != nil {
		parsed, err := uuid.Parse(*req.ClientID)
		if err == nil {
			clientID = &parsed
		}
	}

	mcpReq := &mcp.MCPRequest{
		RequestID: c.Get("X-Request-ID"),
		TenantID:  claims.TenantID,
		ClientID:  clientID,
		UserID:    claims.UserID,
		Scopes:    scopesToStrings(claims.Scopes),
	}

	scanReq := &mcp.ScanRequest{
		URL:             req.URL,
		CaptureTypes:    req.CaptureTypes,
		FollowRedirects: req.FollowRedirects,
	}

	result, err := h.mcpClient.ScanURL(c.Context(), mcpReq, scanReq)
	if err != nil {
		return handleMCPError(c, err)
	}

	return response.Success(c, result)
}

// AnalyzeURL executa análise de URL
func (h *HuntingHandler) AnalyzeURL(c *fiber.Ctx) error {
	claims := getClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	if !claims.HasAnyScope(models.ScopeAnalyzeWrite) && !claims.IsAdmin() {
		return response.Forbidden(c, "Missing scope: analyze:write")
	}

	var req AnalyzeURLRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.URL == "" {
		return response.BadRequest(c, "URL is required")
	}

	var clientID *uuid.UUID
	if req.ClientID != nil {
		parsed, err := uuid.Parse(*req.ClientID)
		if err == nil {
			clientID = &parsed
		}
	}

	mcpReq := &mcp.MCPRequest{
		RequestID: c.Get("X-Request-ID"),
		TenantID:  claims.TenantID,
		ClientID:  clientID,
		UserID:    claims.UserID,
		Scopes:    scopesToStrings(claims.Scopes),
	}

	analyzeReq := &mcp.AnalyzeRequest{
		URL:          req.URL,
		IncludeLeaks: req.IncludeLeaks,
		DeepAnalysis: req.DeepAnalysis,
	}

	result, err := h.mcpClient.AnalyzeURL(c.Context(), mcpReq, analyzeReq)
	if err != nil {
		return handleMCPError(c, err)
	}

	return response.Success(c, result)
}

// SearchLeaks busca vazamentos
func (h *HuntingHandler) SearchLeaks(c *fiber.Ctx) error {
	claims := getClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	if !claims.HasAnyScope(models.ScopeHuntingRead, models.ScopeHuntingWrite) && !claims.IsAdmin() {
		return response.Forbidden(c, "Missing scope: hunting:read or hunting:write")
	}

	var req LeakSearchReq
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Query == "" {
		return response.BadRequest(c, "Query is required")
	}

	if req.Type == "" {
		req.Type = "domain"
	}

	if req.MaxResults == 0 {
		req.MaxResults = 100
	}

	var clientID *uuid.UUID
	if req.ClientID != nil {
		parsed, err := uuid.Parse(*req.ClientID)
		if err == nil {
			clientID = &parsed
		}
	}

	mcpReq := &mcp.MCPRequest{
		RequestID: c.Get("X-Request-ID"),
		TenantID:  claims.TenantID,
		ClientID:  clientID,
		UserID:    claims.UserID,
		Scopes:    scopesToStrings(claims.Scopes),
	}

	searchReq := &mcp.LeakSearchRequest{
		Query:      req.Query,
		Type:       req.Type,
		MaxResults: req.MaxResults,
	}

	result, err := h.mcpClient.SearchLeaks(c.Context(), mcpReq, searchReq)
	if err != nil {
		return handleMCPError(c, err)
	}

	return response.Success(c, result)
}

// =============================================================================
// MONITOR HANDLERS
// =============================================================================

// CreateMonitorJob cria um job de monitoramento
func (h *HuntingHandler) CreateMonitorJob(c *fiber.Ctx) error {
	claims := getClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	if !claims.HasAnyScope(models.ScopeMonitorWrite) && !claims.IsAdmin() {
		return response.Forbidden(c, "Missing scope: monitor:write")
	}

	var req struct {
		BrandID       string   `json:"brand_id"`
		Target        string   `json:"target"`
		IntervalMins  int      `json:"interval_mins"`
		EnabledChecks []string `json:"enabled_checks"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.BrandID == "" || req.Target == "" {
		return response.BadRequest(c, "brand_id and target are required")
	}

	brandID, err := uuid.Parse(req.BrandID)
	if err != nil {
		return response.BadRequest(c, "Invalid brand_id")
	}

	if req.IntervalMins == 0 {
		req.IntervalMins = 60
	}

	if len(req.EnabledChecks) == 0 {
		req.EnabledChecks = []string{"phishing", "domain", "ssl"}
	}

	mcpReq := &mcp.MCPRequest{
		RequestID: c.Get("X-Request-ID"),
		TenantID:  claims.TenantID,
		UserID:    claims.UserID,
		Scopes:    scopesToStrings(claims.Scopes),
	}

	monitorReq := &mcp.MonitorJobRequest{
		BrandID:       brandID,
		Target:        req.Target,
		IntervalMins:  req.IntervalMins,
		EnabledChecks: req.EnabledChecks,
	}

	result, err := h.mcpClient.CreateMonitorJob(c.Context(), mcpReq, monitorReq)
	if err != nil {
		return handleMCPError(c, err)
	}

	return response.Created(c, result)
}

// StopMonitorJob para um job de monitoramento
func (h *HuntingHandler) StopMonitorJob(c *fiber.Ctx) error {
	claims := getClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	if !claims.HasAnyScope(models.ScopeMonitorWrite) && !claims.IsAdmin() {
		return response.Forbidden(c, "Missing scope: monitor:write")
	}

	jobID, err := uuid.Parse(c.Params("job_id"))
	if err != nil {
		return response.BadRequest(c, "Invalid job_id")
	}

	mcpReq := &mcp.MCPRequest{
		RequestID: c.Get("X-Request-ID"),
		TenantID:  claims.TenantID,
		UserID:    claims.UserID,
		Scopes:    scopesToStrings(claims.Scopes),
	}

	if err := h.mcpClient.StopMonitorJob(c.Context(), mcpReq, jobID); err != nil {
		return handleMCPError(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Monitor job stopped",
		"job_id":  jobID,
	})
}

// =============================================================================
// HELPERS
// =============================================================================

func scopesToStrings(scopes []models.Scope) []string {
	result := make([]string, len(scopes))
	for i, s := range scopes {
		result[i] = string(s)
	}
	return result
}

func handleMCPError(c *fiber.Ctx, err error) error {
	switch err {
	case mcp.ErrMCPUnauthorized:
		return response.Unauthorized(c, "MCP authentication failed")
	case mcp.ErrMCPForbidden:
		return response.Forbidden(c, "Tool not allowed by policy")
	case mcp.ErrMCPNotFound:
		return response.NotFound(c, "Resource not found")
	case mcp.ErrMCPRateLimit:
		return response.TooManyRequests(c, "Rate limit exceeded")
	case mcp.ErrMCPUnavailable:
		return response.ServiceUnavailable(c, "MCP service unavailable")
	default:
		return response.InternalServerError(c, "MCP request failed: "+err.Error())
	}
}
