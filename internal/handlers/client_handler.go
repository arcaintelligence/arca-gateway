package handlers

import (
	"time"

	"github.com/arcaintelligence/arca-gateway/internal/middleware"
	"github.com/arcaintelligence/arca-gateway/internal/models"
	"github.com/arcaintelligence/arca-gateway/internal/services"
	"github.com/arcaintelligence/arca-gateway/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ClientHandler handlers de clientes
type ClientHandler struct {
	clientService *services.ClientService
	brandService  *services.BrandService
}

// NewClientHandler cria um novo handler de clientes
func NewClientHandler(clientService *services.ClientService, brandService *services.BrandService) *ClientHandler {
	return &ClientHandler{
		clientService: clientService,
		brandService:  brandService,
	}
}

// CreateClientRequest request para criar cliente
type CreateClientRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Industry    string                `json:"industry,omitempty"`
	Settings    models.ClientSettings `json:"settings,omitempty"`
}

// CreateBrandRequest request para criar marca
type CreateBrandRequest struct {
	Name          string             `json:"name"`
	PrimaryDomain string             `json:"primary_domain"`
	Config        models.BrandConfig `json:"config,omitempty"`
}

// ClientResponse response de cliente
type ClientResponse struct {
	ID          uuid.UUID             `json:"id"`
	TenantID    uuid.UUID             `json:"tenant_id"`
	Name        string                `json:"name"`
	Slug        string                `json:"slug"`
	Description string                `json:"description,omitempty"`
	Industry    string                `json:"industry,omitempty"`
	Status      models.Status         `json:"status"`
	Settings    models.ClientSettings `json:"settings"`
	BrandsCount int                   `json:"brands_count"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// BrandResponse response de marca
type BrandResponse struct {
	ID              uuid.UUID          `json:"id"`
	ClientID        uuid.UUID          `json:"client_id"`
	TenantID        uuid.UUID          `json:"tenant_id"`
	Name            string             `json:"name"`
	PrimaryDomain   string             `json:"primary_domain"`
	Status          models.Status      `json:"status"`
	Config          models.BrandConfig `json:"config"`
	MonitoringJobID *uuid.UUID         `json:"monitoring_job_id,omitempty"`
	LastScanAt      *time.Time         `json:"last_scan_at,omitempty"`
	ThreatsFound    int                `json:"threats_found"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// =============================================================================
// CLIENT HANDLERS
// =============================================================================

// ListClients lista todos os clientes do tenant
func (h *ClientHandler) ListClients(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == uuid.Nil {
		return response.Unauthorized(c, "Authentication required")
	}

	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 20)

	clients, total, err := h.clientService.ListByTenant(c.Context(), tenantID, page, perPage)
	if err != nil {
		return response.InternalServerError(c, "Failed to list clients")
	}

	clientResponses := make([]ClientResponse, len(clients))
	for i, client := range clients {
		brandsCount, _ := h.brandService.CountByClient(c.Context(), client.ID)
		clientResponses[i] = ClientResponse{
			ID:          client.ID,
			TenantID:    client.TenantID,
			Name:        client.Name,
			Slug:        client.Slug,
			Description: client.Description,
			Industry:    client.Industry,
			Status:      client.Status,
			Settings:    client.Settings,
			BrandsCount: brandsCount,
			CreatedAt:   client.CreatedAt,
			UpdatedAt:   client.UpdatedAt,
		}
	}

	return response.Paginated(c, clientResponses, page, perPage, total)
}

// GetClient retorna um cliente específico
func (h *ClientHandler) GetClient(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	clientID, err := uuid.Parse(c.Params("client_id"))
	if err != nil {
		return response.BadRequest(c, "Invalid client ID")
	}

	client, err := h.clientService.GetByID(c.Context(), clientID, tenantID)
	if err != nil {
		return response.NotFound(c, "Client not found")
	}

	brandsCount, _ := h.brandService.CountByClient(c.Context(), client.ID)

	return response.Success(c, ClientResponse{
		ID:          client.ID,
		TenantID:    client.TenantID,
		Name:        client.Name,
		Slug:        client.Slug,
		Description: client.Description,
		Industry:    client.Industry,
		Status:      client.Status,
		Settings:    client.Settings,
		BrandsCount: brandsCount,
		CreatedAt:   client.CreatedAt,
		UpdatedAt:   client.UpdatedAt,
	})
}

// CreateClient cria um novo cliente
func (h *ClientHandler) CreateClient(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == uuid.Nil {
		return response.Unauthorized(c, "Authentication required")
	}

	var req CreateClientRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Name == "" {
		return response.BadRequest(c, "Name is required")
	}

	// Configurações padrão
	if req.Settings.ScanFrequency == "" {
		req.Settings.ScanFrequency = "daily"
	}
	if req.Settings.Priority == "" {
		req.Settings.Priority = "medium"
	}

	client := &models.Client{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        req.Name,
		Slug:        generateSlug(req.Name),
		Description: req.Description,
		Industry:    req.Industry,
		Status:      models.StatusActive,
		Settings:    req.Settings,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.clientService.Create(c.Context(), client); err != nil {
		return response.InternalServerError(c, "Failed to create client")
	}

	return response.Created(c, ClientResponse{
		ID:          client.ID,
		TenantID:    client.TenantID,
		Name:        client.Name,
		Slug:        client.Slug,
		Description: client.Description,
		Industry:    client.Industry,
		Status:      client.Status,
		Settings:    client.Settings,
		BrandsCount: 0,
		CreatedAt:   client.CreatedAt,
		UpdatedAt:   client.UpdatedAt,
	})
}

// UpdateClient atualiza um cliente
func (h *ClientHandler) UpdateClient(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	clientID, err := uuid.Parse(c.Params("client_id"))
	if err != nil {
		return response.BadRequest(c, "Invalid client ID")
	}

	client, err := h.clientService.GetByID(c.Context(), clientID, tenantID)
	if err != nil {
		return response.NotFound(c, "Client not found")
	}

	var req CreateClientRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Name != "" {
		client.Name = req.Name
		client.Slug = generateSlug(req.Name)
	}
	if req.Description != "" {
		client.Description = req.Description
	}
	if req.Industry != "" {
		client.Industry = req.Industry
	}
	client.Settings = req.Settings
	client.UpdatedAt = time.Now()

	if err := h.clientService.Update(c.Context(), client); err != nil {
		return response.InternalServerError(c, "Failed to update client")
	}

	brandsCount, _ := h.brandService.CountByClient(c.Context(), client.ID)

	return response.Success(c, ClientResponse{
		ID:          client.ID,
		TenantID:    client.TenantID,
		Name:        client.Name,
		Slug:        client.Slug,
		Description: client.Description,
		Industry:    client.Industry,
		Status:      client.Status,
		Settings:    client.Settings,
		BrandsCount: brandsCount,
		CreatedAt:   client.CreatedAt,
		UpdatedAt:   client.UpdatedAt,
	})
}

// DeleteClient remove um cliente
func (h *ClientHandler) DeleteClient(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	clientID, err := uuid.Parse(c.Params("client_id"))
	if err != nil {
		return response.BadRequest(c, "Invalid client ID")
	}

	if err := h.clientService.Delete(c.Context(), clientID, tenantID); err != nil {
		return response.InternalServerError(c, "Failed to delete client")
	}

	return response.NoContent(c)
}

// =============================================================================
// BRAND HANDLERS
// =============================================================================

// ListBrands lista todas as marcas de um cliente
func (h *ClientHandler) ListBrands(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	clientID, err := uuid.Parse(c.Params("client_id"))
	if err != nil {
		return response.BadRequest(c, "Invalid client ID")
	}

	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 20)

	brands, total, err := h.brandService.ListByClient(c.Context(), clientID, tenantID, page, perPage)
	if err != nil {
		return response.InternalServerError(c, "Failed to list brands")
	}

	brandResponses := make([]BrandResponse, len(brands))
	for i, brand := range brands {
		brandResponses[i] = BrandResponse{
			ID:              brand.ID,
			ClientID:        brand.ClientID,
			TenantID:        brand.TenantID,
			Name:            brand.Name,
			PrimaryDomain:   brand.PrimaryDomain,
			Status:          brand.Status,
			Config:          brand.Config,
			MonitoringJobID: brand.MonitoringJobID,
			LastScanAt:      brand.LastScanAt,
			ThreatsFound:    brand.ThreatsFound,
			CreatedAt:       brand.CreatedAt,
			UpdatedAt:       brand.UpdatedAt,
		}
	}

	return response.Paginated(c, brandResponses, page, perPage, total)
}

// GetBrand retorna uma marca específica
func (h *ClientHandler) GetBrand(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	brandID, err := uuid.Parse(c.Params("brand_id"))
	if err != nil {
		return response.BadRequest(c, "Invalid brand ID")
	}

	brand, err := h.brandService.GetByID(c.Context(), brandID, tenantID)
	if err != nil {
		return response.NotFound(c, "Brand not found")
	}

	return response.Success(c, BrandResponse{
		ID:              brand.ID,
		ClientID:        brand.ClientID,
		TenantID:        brand.TenantID,
		Name:            brand.Name,
		PrimaryDomain:   brand.PrimaryDomain,
		Status:          brand.Status,
		Config:          brand.Config,
		MonitoringJobID: brand.MonitoringJobID,
		LastScanAt:      brand.LastScanAt,
		ThreatsFound:    brand.ThreatsFound,
		CreatedAt:       brand.CreatedAt,
		UpdatedAt:       brand.UpdatedAt,
	})
}

// CreateBrand cria uma nova marca e inicia monitoramento
func (h *ClientHandler) CreateBrand(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	clientID, err := uuid.Parse(c.Params("client_id"))
	if err != nil {
		return response.BadRequest(c, "Invalid client ID")
	}

	// Verificar se cliente existe
	_, err = h.clientService.GetByID(c.Context(), clientID, tenantID)
	if err != nil {
		return response.NotFound(c, "Client not found")
	}

	var req CreateBrandRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Name == "" || req.PrimaryDomain == "" {
		return response.BadRequest(c, "Name and primary_domain are required")
	}

	// Configurações padrão
	if req.Config.ScanFrequencyMins == 0 {
		req.Config.ScanFrequencyMins = 60 // 1 hora
	}
	if req.Config.AlertSeverityMin == "" {
		req.Config.AlertSeverityMin = "medium"
	}
	if len(req.Config.AlertChannels) == 0 {
		req.Config.AlertChannels = []string{"email"}
	}

	brand := &models.Brand{
		ID:            uuid.New(),
		ClientID:      clientID,
		TenantID:      tenantID,
		Name:          req.Name,
		PrimaryDomain: req.PrimaryDomain,
		Status:        models.StatusActive,
		Config:        req.Config,
		ThreatsFound:  0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.brandService.Create(c.Context(), brand); err != nil {
		return response.InternalServerError(c, "Failed to create brand")
	}

	// TODO: Iniciar job de monitoramento automaticamente
	// jobID := h.monitorService.StartMonitoring(brand)
	// brand.MonitoringJobID = &jobID

	return response.Created(c, BrandResponse{
		ID:              brand.ID,
		ClientID:        brand.ClientID,
		TenantID:        brand.TenantID,
		Name:            brand.Name,
		PrimaryDomain:   brand.PrimaryDomain,
		Status:          brand.Status,
		Config:          brand.Config,
		MonitoringJobID: brand.MonitoringJobID,
		LastScanAt:      brand.LastScanAt,
		ThreatsFound:    brand.ThreatsFound,
		CreatedAt:       brand.CreatedAt,
		UpdatedAt:       brand.UpdatedAt,
	})
}

// UpdateBrand atualiza uma marca
func (h *ClientHandler) UpdateBrand(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	brandID, err := uuid.Parse(c.Params("brand_id"))
	if err != nil {
		return response.BadRequest(c, "Invalid brand ID")
	}

	brand, err := h.brandService.GetByID(c.Context(), brandID, tenantID)
	if err != nil {
		return response.NotFound(c, "Brand not found")
	}

	var req CreateBrandRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Name != "" {
		brand.Name = req.Name
	}
	if req.PrimaryDomain != "" {
		brand.PrimaryDomain = req.PrimaryDomain
	}
	brand.Config = req.Config
	brand.UpdatedAt = time.Now()

	if err := h.brandService.Update(c.Context(), brand); err != nil {
		return response.InternalServerError(c, "Failed to update brand")
	}

	return response.Success(c, BrandResponse{
		ID:              brand.ID,
		ClientID:        brand.ClientID,
		TenantID:        brand.TenantID,
		Name:            brand.Name,
		PrimaryDomain:   brand.PrimaryDomain,
		Status:          brand.Status,
		Config:          brand.Config,
		MonitoringJobID: brand.MonitoringJobID,
		LastScanAt:      brand.LastScanAt,
		ThreatsFound:    brand.ThreatsFound,
		CreatedAt:       brand.CreatedAt,
		UpdatedAt:       brand.UpdatedAt,
	})
}

// DeleteBrand remove uma marca
func (h *ClientHandler) DeleteBrand(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	brandID, err := uuid.Parse(c.Params("brand_id"))
	if err != nil {
		return response.BadRequest(c, "Invalid brand ID")
	}

	// TODO: Parar job de monitoramento antes de deletar

	if err := h.brandService.Delete(c.Context(), brandID, tenantID); err != nil {
		return response.InternalServerError(c, "Failed to delete brand")
	}

	return response.NoContent(c)
}

// StartMonitoring inicia o monitoramento de uma marca
func (h *ClientHandler) StartMonitoring(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	brandID, err := uuid.Parse(c.Params("brand_id"))
	if err != nil {
		return response.BadRequest(c, "Invalid brand ID")
	}

	brand, err := h.brandService.GetByID(c.Context(), brandID, tenantID)
	if err != nil {
		return response.NotFound(c, "Brand not found")
	}

	if brand.MonitoringJobID != nil {
		return response.Conflict(c, "Monitoring already running")
	}

	// TODO: Chamar MCP para iniciar monitoramento
	jobID := uuid.New()
	brand.MonitoringJobID = &jobID
	brand.Status = models.StatusActive
	brand.UpdatedAt = time.Now()

	if err := h.brandService.Update(c.Context(), brand); err != nil {
		return response.InternalServerError(c, "Failed to start monitoring")
	}

	return response.Success(c, fiber.Map{
		"message": "Monitoring started",
		"job_id":  jobID,
	})
}

// StopMonitoring para o monitoramento de uma marca
func (h *ClientHandler) StopMonitoring(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	brandID, err := uuid.Parse(c.Params("brand_id"))
	if err != nil {
		return response.BadRequest(c, "Invalid brand ID")
	}

	brand, err := h.brandService.GetByID(c.Context(), brandID, tenantID)
	if err != nil {
		return response.NotFound(c, "Brand not found")
	}

	if brand.MonitoringJobID == nil {
		return response.BadRequest(c, "Monitoring not running")
	}

	// TODO: Chamar MCP para parar monitoramento
	brand.MonitoringJobID = nil
	brand.Status = models.StatusInactive
	brand.UpdatedAt = time.Now()

	if err := h.brandService.Update(c.Context(), brand); err != nil {
		return response.InternalServerError(c, "Failed to stop monitoring")
	}

	return response.Success(c, fiber.Map{
		"message": "Monitoring stopped",
	})
}
