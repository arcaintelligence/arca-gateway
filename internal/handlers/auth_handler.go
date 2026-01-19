package handlers

import (
	"time"

	"github.com/arcaintelligence/arca-gateway/internal/auth"
	"github.com/arcaintelligence/arca-gateway/internal/models"
	"github.com/arcaintelligence/arca-gateway/internal/services"
	"github.com/arcaintelligence/arca-gateway/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handlers de autenticação
type AuthHandler struct {
	jwtManager  *auth.JWTManager
	userService *services.UserService
}

// NewAuthHandler cria um novo handler de autenticação
func NewAuthHandler(jwtManager *auth.JWTManager, userService *services.UserService) *AuthHandler {
	return &AuthHandler{
		jwtManager:  jwtManager,
		userService: userService,
	}
}

// LoginRequest request de login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse response de login
type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
	User         UserResponse `json:"user"`
}

// UserResponse response de usuário
type UserResponse struct {
	ID       uuid.UUID      `json:"id"`
	TenantID uuid.UUID      `json:"tenant_id"`
	Email    string         `json:"email"`
	Name     string         `json:"name"`
	Role     models.Role    `json:"role"`
	Scopes   []models.Scope `json:"scopes"`
}

// RegisterRequest request de registro
type RegisterRequest struct {
	TenantName string `json:"tenant_name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Name       string `json:"name"`
}

// RefreshRequest request de refresh token
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Login autentica um usuário
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return response.BadRequest(c, "Email and password are required")
	}

	user, err := h.userService.GetByEmail(c.Context(), req.Email)
	if err != nil {
		return response.Unauthorized(c, "Invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return response.Unauthorized(c, "Invalid credentials")
	}

	if user.Status != models.StatusActive {
		return response.Forbidden(c, "Account is not active")
	}

	accessToken, refreshToken, err := h.jwtManager.GenerateTokenPair(user)
	if err != nil {
		return response.InternalServerError(c, "Failed to generate tokens")
	}

	now := time.Now()
	user.LastLoginAt = &now
	_ = h.userService.Update(c.Context(), user)

	return response.Success(c, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		User: UserResponse{
			ID:       user.ID,
			TenantID: user.TenantID,
			Email:    user.Email,
			Name:     user.Name,
			Role:     user.Role,
			Scopes:   user.Scopes,
		},
	})
}

// Register registra um novo tenant e usuário admin
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.TenantName == "" || req.Email == "" || req.Password == "" || req.Name == "" {
		return response.BadRequest(c, "All fields are required")
	}

	existingUser, _ := h.userService.GetByEmail(c.Context(), req.Email)
	if existingUser != nil {
		return response.Conflict(c, "Email already registered")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return response.InternalServerError(c, "Failed to process password")
	}

	tenant := &models.Tenant{
		ID:     uuid.New(),
		Name:   req.TenantName,
		Slug:   generateSlug(req.TenantName),
		Email:  req.Email,
		Plan:   "free",
		Status: models.StatusActive,
		Settings: models.TenantSettings{
			AllowedScopes:     models.GetDefaultScopesForRole(models.RoleAdmin),
			AllowedTools:      []string{"site_scan", "leak_search", "ai_analyze"},
			EmailNotify:       true,
			MaxConcurrentJobs: 5,
		},
		Quotas: models.TenantQuotas{
			MaxClients:        10,
			MaxBrands:         20,
			MaxScansPerDay:    100,
			MaxAlertsPerDay:   50,
			MaxUsersPerTenant: 5,
			StorageLimitMB:    1024,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	user := &models.User{
		ID:           uuid.New(),
		TenantID:     tenant.ID,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Name:         req.Name,
		Role:         models.RoleAdmin,
		Scopes:       models.GetDefaultScopesForRole(models.RoleAdmin),
		Status:       models.StatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.userService.CreateWithTenant(c.Context(), tenant, user); err != nil {
		return response.InternalServerError(c, "Failed to create account")
	}

	accessToken, refreshToken, err := h.jwtManager.GenerateTokenPair(user)
	if err != nil {
		return response.InternalServerError(c, "Failed to generate tokens")
	}

	return response.Created(c, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		User: UserResponse{
			ID:       user.ID,
			TenantID: user.TenantID,
			Email:    user.Email,
			Name:     user.Name,
			Role:     user.Role,
			Scopes:   user.Scopes,
		},
	})
}

// RefreshToken renova o access token
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.RefreshToken == "" {
		return response.BadRequest(c, "Refresh token is required")
	}

	accessToken, err := h.jwtManager.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		return response.Unauthorized(c, "Invalid or expired refresh token")
	}

	return response.Success(c, fiber.Map{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   900,
	})
}

// Logout invalida o token
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"message": "Logged out successfully",
	})
}

// Me retorna informações do usuário autenticado
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	claims := getClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	user, err := h.userService.GetByID(c.Context(), claims.UserID)
	if err != nil {
		return response.NotFound(c, "User not found")
	}

	return response.Success(c, UserResponse{
		ID:       user.ID,
		TenantID: user.TenantID,
		Email:    user.Email,
		Name:     user.Name,
		Role:     user.Role,
		Scopes:   user.Scopes,
	})
}

// GenerateAPIKey gera uma API key para o usuário
func (h *AuthHandler) GenerateAPIKey(c *fiber.Ctx) error {
	claims := getClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	if claims.Role != models.RoleAdmin && claims.Role != models.RoleManager {
		return response.Forbidden(c, "Only admin and manager can generate API keys")
	}

	user, err := h.userService.GetByID(c.Context(), claims.UserID)
	if err != nil {
		return response.NotFound(c, "User not found")
	}

	expiry := 365 * 24 * time.Hour
	apiKey, err := h.jwtManager.GenerateAPIToken(user, expiry)
	if err != nil {
		return response.InternalServerError(c, "Failed to generate API key")
	}

	return response.Success(c, fiber.Map{
		"api_key":    apiKey,
		"expires_in": int(expiry.Hours()),
		"message":    "Store this API key securely",
	})
}

func getClaims(c *fiber.Ctx) *auth.Claims {
	claims, ok := c.Locals("claims").(*auth.Claims)
	if !ok {
		return nil
	}
	return claims
}

func generateSlug(name string) string {
	slug := ""
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			slug += string(char)
		} else if char >= 'A' && char <= 'Z' {
			slug += string(char + 32)
		} else if char == ' ' || char == '-' {
			slug += "-"
		}
	}
	return slug
}
