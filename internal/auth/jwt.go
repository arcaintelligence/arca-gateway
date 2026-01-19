package auth

import (
	"errors"
	"time"

	"github.com/arcaintelligence/arca-gateway/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidClaims    = errors.New("invalid token claims")
	ErrMissingToken     = errors.New("missing authorization token")
	ErrInvalidSignature = errors.New("invalid token signature")
)

// TokenType representa o tipo de token
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
	TokenTypeAPI     TokenType = "api"
)

// Claims representa os claims customizados do JWT
type Claims struct {
	jwt.RegisteredClaims
	
	// Identificadores
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	
	// Permissões
	Role   models.Role    `json:"role"`
	Scopes []models.Scope `json:"scopes"`
	
	// Metadata
	TokenType TokenType `json:"token_type"`
	Email     string    `json:"email,omitempty"`
	Name      string    `json:"name,omitempty"`
}

// JWTManager gerencia operações com JWT
type JWTManager struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	issuer        string
	audience      string
}

// NewJWTManager cria um novo gerenciador JWT
func NewJWTManager(secret string, accessExpiry, refreshExpiry time.Duration, issuer, audience string) *JWTManager {
	return &JWTManager{
		secret:        []byte(secret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		issuer:        issuer,
		audience:      audience,
	}
}

// GenerateAccessToken gera um token de acesso
func (m *JWTManager) GenerateAccessToken(user *models.User) (string, error) {
	return m.generateToken(user, TokenTypeAccess, m.accessExpiry)
}

// GenerateRefreshToken gera um token de refresh
func (m *JWTManager) GenerateRefreshToken(user *models.User) (string, error) {
	return m.generateToken(user, TokenTypeRefresh, m.refreshExpiry)
}

// GenerateAPIToken gera um token de API (longa duração)
func (m *JWTManager) GenerateAPIToken(user *models.User, expiry time.Duration) (string, error) {
	return m.generateToken(user, TokenTypeAPI, expiry)
}

// GenerateTokenPair gera um par de tokens (access + refresh)
func (m *JWTManager) GenerateTokenPair(user *models.User) (accessToken, refreshToken string, err error) {
	accessToken, err = m.GenerateAccessToken(user)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = m.GenerateRefreshToken(user)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// generateToken gera um token JWT
func (m *JWTManager) generateToken(user *models.User, tokenType TokenType, expiry time.Duration) (string, error) {
	now := time.Now()
	
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   user.ID.String(),
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
		UserID:    user.ID,
		TenantID:  user.TenantID,
		Role:      user.Role,
		Scopes:    user.Scopes,
		TokenType: tokenType,
		Email:     user.Email,
		Name:      user.Name,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ValidateToken valida um token JWT e retorna os claims
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verificar algoritmo de assinatura
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignature
		}
		return m.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// RefreshAccessToken gera um novo access token a partir de um refresh token
func (m *JWTManager) RefreshAccessToken(refreshToken string) (string, error) {
	claims, err := m.ValidateToken(refreshToken)
	if err != nil {
		return "", err
	}

	if claims.TokenType != TokenTypeRefresh {
		return "", ErrInvalidToken
	}

	// Criar um user temporário com os dados do token
	user := &models.User{
		ID:       claims.UserID,
		TenantID: claims.TenantID,
		Email:    claims.Email,
		Name:     claims.Name,
		Role:     claims.Role,
		Scopes:   claims.Scopes,
	}

	return m.GenerateAccessToken(user)
}

// ExtractTokenFromHeader extrai o token do header Authorization
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", ErrMissingToken
	}

	// Formato esperado: "Bearer <token>"
	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) {
		return "", ErrInvalidToken
	}

	if authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", ErrInvalidToken
	}

	return authHeader[len(bearerPrefix):], nil
}

// HasScope verifica se os claims contêm um scope específico
func (c *Claims) HasScope(scope models.Scope) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasAnyScope verifica se os claims contêm pelo menos um dos scopes
func (c *Claims) HasAnyScope(scopes ...models.Scope) bool {
	for _, scope := range scopes {
		if c.HasScope(scope) {
			return true
		}
	}
	return false
}

// HasAllScopes verifica se os claims contêm todos os scopes
func (c *Claims) HasAllScopes(scopes ...models.Scope) bool {
	for _, scope := range scopes {
		if !c.HasScope(scope) {
			return false
		}
	}
	return true
}

// IsAdmin verifica se o usuário é admin
func (c *Claims) IsAdmin() bool {
	return c.Role == models.RoleAdmin
}

// CanManage verifica se o usuário pode gerenciar recursos
func (c *Claims) CanManage() bool {
	return c.Role == models.RoleAdmin || c.Role == models.RoleManager
}
