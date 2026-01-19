package middleware

import (
	"strings"

	"github.com/arcaintelligence/arca-gateway/internal/auth"
	"github.com/arcaintelligence/arca-gateway/internal/models"
	"github.com/arcaintelligence/arca-gateway/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Context keys para armazenar dados do usuário autenticado
const (
	ContextKeyClaims   = "claims"
	ContextKeyUserID   = "user_id"
	ContextKeyTenantID = "tenant_id"
	ContextKeyClientID = "client_id"
	ContextKeyRole     = "role"
	ContextKeyScopes   = "scopes"
)

// AuthMiddleware middleware de autenticação JWT
type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

// NewAuthMiddleware cria um novo middleware de autenticação
func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

// Authenticate middleware que requer autenticação
func (m *AuthMiddleware) Authenticate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extrair token do header
		authHeader := c.Get("Authorization")
		tokenString, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			return response.Unauthorized(c, "Missing or invalid authorization token")
		}

		// Validar token
		claims, err := m.jwtManager.ValidateToken(tokenString)
		if err != nil {
			switch err {
			case auth.ErrExpiredToken:
				return response.Unauthorized(c, "Token has expired")
			case auth.ErrInvalidToken, auth.ErrInvalidClaims:
				return response.Unauthorized(c, "Invalid token")
			default:
				return response.Unauthorized(c, "Authentication failed")
			}
		}

		// Verificar se é token de acesso
		if claims.TokenType != auth.TokenTypeAccess && claims.TokenType != auth.TokenTypeAPI {
			return response.Unauthorized(c, "Invalid token type")
		}

		// Armazenar claims no contexto
		c.Locals(ContextKeyClaims, claims)
		c.Locals(ContextKeyUserID, claims.UserID)
		c.Locals(ContextKeyTenantID, claims.TenantID)
		c.Locals(ContextKeyRole, claims.Role)
		c.Locals(ContextKeyScopes, claims.Scopes)

		// Extrair client_id do header se presente
		if clientID := c.Get("X-Client-ID"); clientID != "" {
			if parsedID, err := uuid.Parse(clientID); err == nil {
				c.Locals(ContextKeyClientID, parsedID)
			}
		}

		return c.Next()
	}
}

// OptionalAuth middleware que tenta autenticar mas não falha se não houver token
func (m *AuthMiddleware) OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		tokenString, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			return c.Next()
		}

		claims, err := m.jwtManager.ValidateToken(tokenString)
		if err != nil {
			return c.Next()
		}

		c.Locals(ContextKeyClaims, claims)
		c.Locals(ContextKeyUserID, claims.UserID)
		c.Locals(ContextKeyTenantID, claims.TenantID)
		c.Locals(ContextKeyRole, claims.Role)
		c.Locals(ContextKeyScopes, claims.Scopes)

		return c.Next()
	}
}

// RequireRole middleware que requer um role específico
func (m *AuthMiddleware) RequireRole(roles ...models.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := GetClaims(c)
		if claims == nil {
			return response.Unauthorized(c, "Authentication required")
		}

		for _, role := range roles {
			if claims.Role == role {
				return c.Next()
			}
		}

		return response.Forbidden(c, "Insufficient permissions")
	}
}

// RequireScope middleware que requer um scope específico
func RequireScope(scopes ...models.Scope) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := GetClaims(c)
		if claims == nil {
			return response.Unauthorized(c, "Authentication required")
		}

		// Admin tem acesso a tudo
		if claims.IsAdmin() {
			return c.Next()
		}

		if !claims.HasAnyScope(scopes...) {
			return response.Forbidden(c, "Missing required scope: "+scopesToString(scopes))
		}

		return c.Next()
	}
}

// RequireAllScopes middleware que requer todos os scopes especificados
func RequireAllScopes(scopes ...models.Scope) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := GetClaims(c)
		if claims == nil {
			return response.Unauthorized(c, "Authentication required")
		}

		// Admin tem acesso a tudo
		if claims.IsAdmin() {
			return c.Next()
		}

		if !claims.HasAllScopes(scopes...) {
			return response.Forbidden(c, "Missing required scopes")
		}

		return c.Next()
	}
}

// RequireTenantAccess middleware que verifica acesso ao tenant
func RequireTenantAccess() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := GetClaims(c)
		if claims == nil {
			return response.Unauthorized(c, "Authentication required")
		}

		// Extrair tenant_id do path ou header
		tenantIDStr := c.Params("tenant_id")
		if tenantIDStr == "" {
			tenantIDStr = c.Get("X-Tenant-ID")
		}

		if tenantIDStr == "" {
			// Se não especificado, usar o tenant do token
			return c.Next()
		}

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			return response.BadRequest(c, "Invalid tenant ID")
		}

		// Admin pode acessar qualquer tenant
		if claims.IsAdmin() {
			c.Locals(ContextKeyTenantID, tenantID)
			return c.Next()
		}

		// Usuário só pode acessar seu próprio tenant
		if claims.TenantID != tenantID {
			return response.Forbidden(c, "Access denied to this tenant")
		}

		return c.Next()
	}
}

// RequireClientAccess middleware que verifica acesso ao cliente
func RequireClientAccess() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := GetClaims(c)
		if claims == nil {
			return response.Unauthorized(c, "Authentication required")
		}

		clientIDStr := c.Params("client_id")
		if clientIDStr == "" {
			clientIDStr = c.Get("X-Client-ID")
		}

		if clientIDStr == "" {
			return c.Next()
		}

		clientID, err := uuid.Parse(clientIDStr)
		if err != nil {
			return response.BadRequest(c, "Invalid client ID")
		}

		c.Locals(ContextKeyClientID, clientID)

		// TODO: Verificar se o cliente pertence ao tenant do usuário
		// Isso requer acesso ao repositório de clientes

		return c.Next()
	}
}

// =============================================================================
// HELPERS
// =============================================================================

// GetClaims retorna os claims do contexto
func GetClaims(c *fiber.Ctx) *auth.Claims {
	claims, ok := c.Locals(ContextKeyClaims).(*auth.Claims)
	if !ok {
		return nil
	}
	return claims
}

// GetUserID retorna o user_id do contexto
func GetUserID(c *fiber.Ctx) uuid.UUID {
	userID, ok := c.Locals(ContextKeyUserID).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return userID
}

// GetTenantID retorna o tenant_id do contexto
func GetTenantID(c *fiber.Ctx) uuid.UUID {
	tenantID, ok := c.Locals(ContextKeyTenantID).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return tenantID
}

// GetClientID retorna o client_id do contexto (pode ser nil)
func GetClientID(c *fiber.Ctx) *uuid.UUID {
	clientID, ok := c.Locals(ContextKeyClientID).(uuid.UUID)
	if !ok || clientID == uuid.Nil {
		return nil
	}
	return &clientID
}

// GetRole retorna o role do contexto
func GetRole(c *fiber.Ctx) models.Role {
	role, ok := c.Locals(ContextKeyRole).(models.Role)
	if !ok {
		return ""
	}
	return role
}

// GetScopes retorna os scopes do contexto
func GetScopes(c *fiber.Ctx) []models.Scope {
	scopes, ok := c.Locals(ContextKeyScopes).([]models.Scope)
	if !ok {
		return nil
	}
	return scopes
}

// scopesToString converte scopes para string
func scopesToString(scopes []models.Scope) string {
	strs := make([]string, len(scopes))
	for i, s := range scopes {
		strs[i] = string(s)
	}
	return strings.Join(strs, ", ")
}
