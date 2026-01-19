package services

import (
	"context"
	"errors"
	"sync"

	"github.com/arcaintelligence/arca-gateway/internal/models"
	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrForbidden     = errors.New("access forbidden")
)

// =============================================================================
// USER SERVICE (In-Memory para demo)
// =============================================================================

// UserService serviço de usuários
type UserService struct {
	mu      sync.RWMutex
	users   map[uuid.UUID]*models.User
	tenants map[uuid.UUID]*models.Tenant
	byEmail map[string]*models.User
}

// NewUserService cria um novo serviço de usuários
func NewUserService() *UserService {
	return &UserService{
		users:   make(map[uuid.UUID]*models.User),
		tenants: make(map[uuid.UUID]*models.Tenant),
		byEmail: make(map[string]*models.User),
	}
}

// GetByID busca usuário por ID
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	return user, nil
}

// GetByEmail busca usuário por email
func (s *UserService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.byEmail[email]
	if !ok {
		return nil, ErrNotFound
	}
	return user, nil
}

// Create cria um novo usuário
func (s *UserService) Create(ctx context.Context, user *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byEmail[user.Email]; exists {
		return ErrAlreadyExists
	}

	s.users[user.ID] = user
	s.byEmail[user.Email] = user
	return nil
}

// CreateWithTenant cria um novo tenant e usuário
func (s *UserService) CreateWithTenant(ctx context.Context, tenant *models.Tenant, user *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byEmail[user.Email]; exists {
		return ErrAlreadyExists
	}

	s.tenants[tenant.ID] = tenant
	s.users[user.ID] = user
	s.byEmail[user.Email] = user
	return nil
}

// Update atualiza um usuário
func (s *UserService) Update(ctx context.Context, user *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.ID]; !exists {
		return ErrNotFound
	}

	s.users[user.ID] = user
	s.byEmail[user.Email] = user
	return nil
}

// =============================================================================
// CLIENT SERVICE (In-Memory para demo)
// =============================================================================

// ClientService serviço de clientes
type ClientService struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]*models.Client
}

// NewClientService cria um novo serviço de clientes
func NewClientService() *ClientService {
	return &ClientService{
		clients: make(map[uuid.UUID]*models.Client),
	}
}

// GetByID busca cliente por ID
func (s *ClientService) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, ok := s.clients[id]
	if !ok {
		return nil, ErrNotFound
	}

	if client.TenantID != tenantID {
		return nil, ErrForbidden
	}

	return client, nil
}

// ListByTenant lista clientes por tenant
func (s *ClientService) ListByTenant(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]*models.Client, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var clients []*models.Client
	for _, client := range s.clients {
		if client.TenantID == tenantID {
			clients = append(clients, client)
		}
	}

	total := int64(len(clients))

	// Paginação simples
	start := (page - 1) * perPage
	end := start + perPage
	if start > len(clients) {
		return []*models.Client{}, total, nil
	}
	if end > len(clients) {
		end = len(clients)
	}

	return clients[start:end], total, nil
}

// Create cria um novo cliente
func (s *ClientService) Create(ctx context.Context, client *models.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[client.ID] = client
	return nil
}

// Update atualiza um cliente
func (s *ClientService) Update(ctx context.Context, client *models.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[client.ID]; !exists {
		return ErrNotFound
	}

	s.clients[client.ID] = client
	return nil
}

// Delete remove um cliente
func (s *ClientService) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, ok := s.clients[id]
	if !ok {
		return ErrNotFound
	}

	if client.TenantID != tenantID {
		return ErrForbidden
	}

	delete(s.clients, id)
	return nil
}

// =============================================================================
// BRAND SERVICE (In-Memory para demo)
// =============================================================================

// BrandService serviço de marcas
type BrandService struct {
	mu     sync.RWMutex
	brands map[uuid.UUID]*models.Brand
}

// NewBrandService cria um novo serviço de marcas
func NewBrandService() *BrandService {
	return &BrandService{
		brands: make(map[uuid.UUID]*models.Brand),
	}
}

// GetByID busca marca por ID
func (s *BrandService) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Brand, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	brand, ok := s.brands[id]
	if !ok {
		return nil, ErrNotFound
	}

	if brand.TenantID != tenantID {
		return nil, ErrForbidden
	}

	return brand, nil
}

// ListByClient lista marcas por cliente
func (s *BrandService) ListByClient(ctx context.Context, clientID, tenantID uuid.UUID, page, perPage int) ([]*models.Brand, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var brands []*models.Brand
	for _, brand := range s.brands {
		if brand.ClientID == clientID && brand.TenantID == tenantID {
			brands = append(brands, brand)
		}
	}

	total := int64(len(brands))

	start := (page - 1) * perPage
	end := start + perPage
	if start > len(brands) {
		return []*models.Brand{}, total, nil
	}
	if end > len(brands) {
		end = len(brands)
	}

	return brands[start:end], total, nil
}

// CountByClient conta marcas por cliente
func (s *BrandService) CountByClient(ctx context.Context, clientID uuid.UUID) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, brand := range s.brands {
		if brand.ClientID == clientID {
			count++
		}
	}
	return count, nil
}

// Create cria uma nova marca
func (s *BrandService) Create(ctx context.Context, brand *models.Brand) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.brands[brand.ID] = brand
	return nil
}

// Update atualiza uma marca
func (s *BrandService) Update(ctx context.Context, brand *models.Brand) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.brands[brand.ID]; !exists {
		return ErrNotFound
	}

	s.brands[brand.ID] = brand
	return nil
}

// Delete remove uma marca
func (s *BrandService) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	brand, ok := s.brands[id]
	if !ok {
		return ErrNotFound
	}

	if brand.TenantID != tenantID {
		return ErrForbidden
	}

	delete(s.brands, id)
	return nil
}
