package handlers

import (
	"net/http"

	"github.com/arcaintelligence/arca-gateway/internal/mcp"
	"github.com/arcaintelligence/arca-gateway/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// OnboardingHandler handler para operações de onboarding
type OnboardingHandler struct {
	mcpClient *mcp.MCPClient
}

// NewOnboardingHandler cria um novo handler de onboarding
func NewOnboardingHandler(mcpClient *mcp.MCPClient) *OnboardingHandler {
	return &OnboardingHandler{
		mcpClient: mcpClient,
	}
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// OnboardingRegisterRequest request para registro de cliente via onboarding
type OnboardingRegisterRequest struct {
	CompanyName string `json:"company_name" validate:"required,min=2,max=200"`
	Email       string `json:"email" validate:"required,email"`
	Plan        string `json:"plan" validate:"required,oneof=starter professional enterprise"`
	Phone       string `json:"phone,omitempty"`
	Website     string `json:"website,omitempty"`
}

// VerifyEmailRequest request para verificação de email
type VerifyEmailRequest struct {
	ClientID string `json:"client_id" validate:"required"`
	Code     string `json:"code" validate:"required,len=6"`
}

// BrandCreateRequest request para criação de marca
type BrandCreateRequest struct {
	Name               string              `json:"name" validate:"required,min=2,max=100"`
	Domain             string              `json:"domain" validate:"required"`
	Keywords           []string            `json:"keywords,omitempty"`
	OfficialApps       *OfficialAppsConfig `json:"official_apps,omitempty"`
	OfficialDevelopers []string            `json:"official_developers,omitempty"`
	SocialMedia        *SocialMediaConfig  `json:"social_media,omitempty"`
}

// OfficialAppsConfig configuração de apps oficiais
type OfficialAppsConfig struct {
	GooglePlay []string `json:"google_play,omitempty"`
	AppleStore []string `json:"apple_store,omitempty"`
}

// SocialMediaConfig configuração de redes sociais
type SocialMediaConfig struct {
	Instagram string `json:"instagram,omitempty"`
	Facebook  string `json:"facebook,omitempty"`
	Twitter   string `json:"twitter,omitempty"`
	LinkedIn  string `json:"linkedin,omitempty"`
	TikTok    string `json:"tiktok,omitempty"`
	YouTube   string `json:"youtube,omitempty"`
}

// StartMonitoringRequest request para iniciar monitoramento
type StartMonitoringRequest struct {
	Frequency     string            `json:"frequency" validate:"required,oneof=realtime hourly daily weekly"`
	Channels      []string          `json:"channels,omitempty"`
	AlertSettings map[string]string `json:"alert_settings,omitempty"`
}

// =============================================================================
// HANDLERS
// =============================================================================

// Register registra um novo cliente
func (h *OnboardingHandler) Register(c *fiber.Ctx) error {
	var req OnboardingRegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	// Validar campos obrigatórios
	if req.CompanyName == "" || req.Email == "" || req.Plan == "" {
		return response.BadRequest(c, "Missing required fields: company_name, email and plan are required")
	}

	// Chamar Core Python via MCP
	mcpReq := &mcp.MCPRequest{
		RequestID: uuid.New().String(),
		TenantID:  uuid.New(),
		Tool:      "onboarding",
		Action:    "register",
		Params: map[string]interface{}{
			"company_name": req.CompanyName,
			"email":        req.Email,
			"plan":         req.Plan,
			"phone":        req.Phone,
			"website":      req.Website,
		},
	}

	resp, err := h.mcpClient.ProxyRequest(c.Context(), http.MethodPost, "/v1/onboarding/register", mcpReq)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "MCP_ERROR", "Failed to register client: "+err.Error())
	}

	return response.Success(c, resp.Data)
}

// VerifyEmail verifica o email do cliente
func (h *OnboardingHandler) VerifyEmail(c *fiber.Ctx) error {
	var req VerifyEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	mcpReq := &mcp.MCPRequest{
		RequestID: uuid.New().String(),
		TenantID:  uuid.New(),
		Tool:      "onboarding",
		Action:    "verify_email",
		Params: map[string]interface{}{
			"client_id": req.ClientID,
			"code":      req.Code,
		},
	}

	resp, err := h.mcpClient.ProxyRequest(c.Context(), http.MethodPost, "/v1/onboarding/verify-email", mcpReq)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "MCP_ERROR", "Failed to verify email: "+err.Error())
	}

	return response.Success(c, resp.Data)
}

// CreateBrand cria uma nova marca para o cliente
func (h *OnboardingHandler) CreateBrand(c *fiber.Ctx) error {
	var req BrandCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	// Obter client_id do contexto (JWT) ou header
	clientID := ""
	if cid, ok := c.Locals("client_id").(string); ok && cid != "" {
		clientID = cid
	} else {
		clientID = c.Get("X-Client-ID")
	}

	// Preparar params
	params := map[string]interface{}{
		"name":      req.Name,
		"domain":    req.Domain,
		"client_id": clientID,
	}

	if len(req.Keywords) > 0 {
		params["keywords"] = req.Keywords
	}
	if req.OfficialApps != nil {
		params["official_apps"] = map[string]interface{}{
			"google_play": req.OfficialApps.GooglePlay,
			"apple_store": req.OfficialApps.AppleStore,
		}
	}
	if len(req.OfficialDevelopers) > 0 {
		params["official_developers"] = req.OfficialDevelopers
	}
	if req.SocialMedia != nil {
		params["social_media"] = map[string]interface{}{
			"instagram": req.SocialMedia.Instagram,
			"facebook":  req.SocialMedia.Facebook,
			"twitter":   req.SocialMedia.Twitter,
			"linkedin":  req.SocialMedia.LinkedIn,
			"tiktok":    req.SocialMedia.TikTok,
			"youtube":   req.SocialMedia.YouTube,
		}
	}

	var clientUUID *uuid.UUID
	if clientID != "" {
		if parsed, err := uuid.Parse(clientID); err == nil {
			clientUUID = &parsed
		}
	}

	mcpReq := &mcp.MCPRequest{
		RequestID: uuid.New().String(),
		TenantID:  uuid.New(),
		ClientID:  clientUUID,
		Tool:      "onboarding",
		Action:    "create_brand",
		Params:    params,
	}

	resp, err := h.mcpClient.ProxyRequest(c.Context(), http.MethodPost, "/v1/brands", mcpReq)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "MCP_ERROR", "Failed to create brand: "+err.Error())
	}

	return response.Created(c, resp.Data)
}

// GetBrand obtém detalhes de uma marca
func (h *OnboardingHandler) GetBrand(c *fiber.Ctx) error {
	brandID := c.Params("brand_id")
	clientID := ""
	if cid, ok := c.Locals("client_id").(string); ok && cid != "" {
		clientID = cid
	} else {
		clientID = c.Get("X-Client-ID")
	}

	var clientUUID *uuid.UUID
	if clientID != "" {
		if parsed, err := uuid.Parse(clientID); err == nil {
			clientUUID = &parsed
		}
	}

	mcpReq := &mcp.MCPRequest{
		RequestID: uuid.New().String(),
		TenantID:  uuid.New(),
		ClientID:  clientUUID,
		Tool:      "onboarding",
		Action:    "get_monitoring_status",
		Params: map[string]interface{}{
			"brand_id":  brandID,
			"client_id": clientID,
		},
	}

	resp, err := h.mcpClient.ProxyRequest(c.Context(), http.MethodGet, "/v1/brands/"+brandID, mcpReq)
	if err != nil {
		return response.NotFound(c, "Brand not found: "+err.Error())
	}

	return response.Success(c, resp.Data)
}

// ListBrands lista todas as marcas do cliente
func (h *OnboardingHandler) ListBrands(c *fiber.Ctx) error {
	clientID := ""
	if cid, ok := c.Locals("client_id").(string); ok && cid != "" {
		clientID = cid
	} else {
		clientID = c.Get("X-Client-ID")
	}

	var clientUUID *uuid.UUID
	if clientID != "" {
		if parsed, err := uuid.Parse(clientID); err == nil {
			clientUUID = &parsed
		}
	}

	mcpReq := &mcp.MCPRequest{
		RequestID: uuid.New().String(),
		TenantID:  uuid.New(),
		ClientID:  clientUUID,
		Tool:      "onboarding",
		Action:    "list_brands",
		Params: map[string]interface{}{
			"client_id": clientID,
		},
	}

	resp, err := h.mcpClient.ProxyRequest(c.Context(), http.MethodGet, "/v1/brands", mcpReq)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "MCP_ERROR", "Failed to list brands: "+err.Error())
	}

	return response.Success(c, resp.Data)
}

// StartMonitoring inicia o monitoramento de uma marca
func (h *OnboardingHandler) StartMonitoring(c *fiber.Ctx) error {
	brandID := c.Params("brand_id")
	clientID := ""
	if cid, ok := c.Locals("client_id").(string); ok && cid != "" {
		clientID = cid
	} else {
		clientID = c.Get("X-Client-ID")
	}

	var req StartMonitoringRequest
	if err := c.BodyParser(&req); err != nil {
		// Se não houver body, usar defaults
		req.Frequency = "hourly"
		req.Channels = []string{"web", "social"}
	}

	var clientUUID *uuid.UUID
	if clientID != "" {
		if parsed, err := uuid.Parse(clientID); err == nil {
			clientUUID = &parsed
		}
	}

	mcpReq := &mcp.MCPRequest{
		RequestID: uuid.New().String(),
		TenantID:  uuid.New(),
		ClientID:  clientUUID,
		Tool:      "monitoring",
		Action:    "start",
		Params: map[string]interface{}{
			"brand_id":       brandID,
			"client_id":      clientID,
			"frequency":      req.Frequency,
			"channels":       req.Channels,
			"alert_settings": req.AlertSettings,
		},
	}

	resp, err := h.mcpClient.ProxyRequest(c.Context(), http.MethodPost, "/v1/brands/"+brandID+"/monitoring/start", mcpReq)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "MCP_ERROR", "Failed to start monitoring: "+err.Error())
	}

	return response.Success(c, resp.Data)
}

// StopMonitoring para o monitoramento de uma marca
func (h *OnboardingHandler) StopMonitoring(c *fiber.Ctx) error {
	brandID := c.Params("brand_id")
	clientID := ""
	if cid, ok := c.Locals("client_id").(string); ok && cid != "" {
		clientID = cid
	} else {
		clientID = c.Get("X-Client-ID")
	}

	var clientUUID *uuid.UUID
	if clientID != "" {
		if parsed, err := uuid.Parse(clientID); err == nil {
			clientUUID = &parsed
		}
	}

	mcpReq := &mcp.MCPRequest{
		RequestID: uuid.New().String(),
		TenantID:  uuid.New(),
		ClientID:  clientUUID,
		Tool:      "monitoring",
		Action:    "stop",
		Params: map[string]interface{}{
			"brand_id":  brandID,
			"client_id": clientID,
		},
	}

	resp, err := h.mcpClient.ProxyRequest(c.Context(), http.MethodPost, "/v1/brands/"+brandID+"/monitoring/stop", mcpReq)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "MCP_ERROR", "Failed to stop monitoring: "+err.Error())
	}

	return response.Success(c, resp.Data)
}

// GetMonitoringStatus obtém o status do monitoramento
func (h *OnboardingHandler) GetMonitoringStatus(c *fiber.Ctx) error {
	brandID := c.Params("brand_id")
	clientID := ""
	if cid, ok := c.Locals("client_id").(string); ok && cid != "" {
		clientID = cid
	} else {
		clientID = c.Get("X-Client-ID")
	}

	var clientUUID *uuid.UUID
	if clientID != "" {
		if parsed, err := uuid.Parse(clientID); err == nil {
			clientUUID = &parsed
		}
	}

	mcpReq := &mcp.MCPRequest{
		RequestID: uuid.New().String(),
		TenantID:  uuid.New(),
		ClientID:  clientUUID,
		Tool:      "monitoring",
		Action:    "status",
		Params: map[string]interface{}{
			"brand_id":  brandID,
			"client_id": clientID,
		},
	}

	resp, err := h.mcpClient.ProxyRequest(c.Context(), http.MethodGet, "/v1/brands/"+brandID+"/monitoring/status", mcpReq)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "MCP_ERROR", "Failed to get monitoring status: "+err.Error())
	}

	return response.Success(c, resp.Data)
}

// GetThreats obtém ameaças detectadas
func (h *OnboardingHandler) GetThreats(c *fiber.Ctx) error {
	clientID := ""
	if cid, ok := c.Locals("client_id").(string); ok && cid != "" {
		clientID = cid
	} else {
		clientID = c.Get("X-Client-ID")
	}
	brandID := c.Query("brand_id")
	status := c.Query("status")
	severity := c.Query("severity")

	var clientUUID *uuid.UUID
	if clientID != "" {
		if parsed, err := uuid.Parse(clientID); err == nil {
			clientUUID = &parsed
		}
	}

	mcpReq := &mcp.MCPRequest{
		RequestID: uuid.New().String(),
		TenantID:  uuid.New(),
		ClientID:  clientUUID,
		Tool:      "threats",
		Action:    "list",
		Params: map[string]interface{}{
			"client_id": clientID,
			"brand_id":  brandID,
			"status":    status,
			"severity":  severity,
		},
	}

	resp, err := h.mcpClient.ProxyRequest(c.Context(), http.MethodGet, "/v1/threats", mcpReq)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "MCP_ERROR", "Failed to get threats: "+err.Error())
	}

	return response.Success(c, resp.Data)
}
