package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/arcaintelligence/arca-gateway/internal/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrForbidden     = errors.New("access forbidden")
)

// =============================================================================
// USER SERVICE (PostgreSQL)
// =============================================================================

type UserService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `SELECT id, tenant_id, email, password_hash, name, role, status, created_at, updated_at FROM users WHERE id = $1`
	
	var user models.User
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.PasswordHash, &user.Name, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, tenant_id, email, password_hash, name, role, status, created_at, updated_at FROM users WHERE email = $1`
	
	var user models.User
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.PasswordHash, &user.Name, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) Create(ctx context.Context, user *models.User) error {
	query := `INSERT INTO users (id, tenant_id, email, password_hash, name, role, status, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err := s.db.ExecContext(ctx, query,
		user.ID, user.TenantID, user.Email, user.PasswordHash, user.Name, user.Role, user.Status, user.CreatedAt, user.UpdatedAt,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (s *UserService) CreateWithTenant(ctx context.Context, tenant *models.Tenant, user *models.User) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create Tenant
	queryTenant := `INSERT INTO tenants (id, name, plan, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = tx.ExecContext(ctx, queryTenant, tenant.ID, tenant.Name, tenant.Plan, tenant.Status, tenant.CreatedAt, tenant.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	// Create User
	queryUser := `INSERT INTO users (id, tenant_id, email, password_hash, name, role, status, created_at, updated_at) 
				  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err = tx.ExecContext(ctx, queryUser,
		user.ID, user.TenantID, user.Email, user.PasswordHash, user.Name, user.Role, user.Status, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return tx.Commit()
}

func (s *UserService) Update(ctx context.Context, user *models.User) error {
	query := `UPDATE users SET name = $1, role = $2, status = $3, updated_at = $4 WHERE id = $5`
	
	res, err := s.db.ExecContext(ctx, query, user.Name, user.Role, user.Status, time.Now(), user.ID)
	if err != nil {
		return err
	}
	
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// =============================================================================
// CLIENT SERVICE (PostgreSQL)
// =============================================================================

type ClientService struct {
	db *sql.DB
}

func NewClientService(db *sql.DB) *ClientService {
	return &ClientService{db: db}
}

func (s *ClientService) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Client, error) {
	query := `SELECT id, tenant_id, name, industry, status, created_at, updated_at FROM clients WHERE id = $1 AND tenant_id = $2`
	
	var client models.Client
	err := s.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&client.ID, &client.TenantID, &client.Name, &client.Industry, &client.Status, &client.CreatedAt, &client.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (s *ClientService) ListByTenant(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]*models.Client, int64, error) {
	offset := (page - 1) * perPage
	
	// Count total
	var total int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM clients WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	
	// List items
	query := `SELECT id, tenant_id, name, industry, status, created_at, updated_at 
			  FROM clients WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	
	rows, err := s.db.QueryContext(ctx, query, tenantID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	
	var clients []*models.Client
	for rows.Next() {
		var c models.Client
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Industry, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		clients = append(clients, &c)
	}
	
	return clients, total, nil
}

func (s *ClientService) Create(ctx context.Context, client *models.Client) error {
	query := `INSERT INTO clients (id, tenant_id, name, industry, status, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	_, err := s.db.ExecContext(ctx, query,
		client.ID, client.TenantID, client.Name, client.Industry, client.Status, client.CreatedAt, client.UpdatedAt,
	)
	return err
}

func (s *ClientService) Update(ctx context.Context, client *models.Client) error {
	query := `UPDATE clients SET name = $1, industry = $2, status = $3, updated_at = $4 WHERE id = $5 AND tenant_id = $6`
	
	res, err := s.db.ExecContext(ctx, query,
		client.Name, client.Industry, client.Status, time.Now(), client.ID, client.TenantID,
	)
	if err != nil {
		return err
	}
	
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ClientService) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `DELETE FROM clients WHERE id = $1 AND tenant_id = $2`
	
	res, err := s.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return err
	}
	
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// =============================================================================
// BRAND SERVICE (PostgreSQL)
// =============================================================================

type BrandService struct {
	db *sql.DB
}

func NewBrandService(db *sql.DB) *BrandService {
	return &BrandService{db: db}
}

func (s *BrandService) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Brand, error) {
	query := `SELECT id, tenant_id, client_id, name, domain, industry, monitoring_enabled, created_at, updated_at 
			  FROM brands WHERE id = $1 AND tenant_id = $2`
	
	var brand models.Brand
	err := s.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&brand.ID, &brand.TenantID, &brand.ClientID, &brand.Name, &brand.PrimaryDomain, &brand.Industry, &brand.MonitoringEnabled, &brand.CreatedAt, &brand.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &brand, nil
}

func (s *BrandService) ListByClient(ctx context.Context, clientID, tenantID uuid.UUID, page, perPage int) ([]*models.Brand, int64, error) {
	offset := (page - 1) * perPage
	
	// Count total
	var total int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM brands WHERE client_id = $1 AND tenant_id = $2`, clientID, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	
	// List items
	query := `SELECT id, tenant_id, client_id, name, domain, industry, monitoring_enabled, created_at, updated_at 
			  FROM brands WHERE client_id = $1 AND tenant_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	
	rows, err := s.db.QueryContext(ctx, query, clientID, tenantID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	
	var brands []*models.Brand
	for rows.Next() {
		var b models.Brand
		if err := rows.Scan(&b.ID, &b.TenantID, &b.ClientID, &b.Name, &b.PrimaryDomain, &b.Industry, &b.MonitoringEnabled, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, 0, err
		}
		brands = append(brands, &b)
	}
	
	return brands, total, nil
}

func (s *BrandService) CountByClient(ctx context.Context, clientID uuid.UUID) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM brands WHERE client_id = $1`, clientID).Scan(&count)
	return count, err
}

func (s *BrandService) Create(ctx context.Context, brand *models.Brand) error {
	query := `INSERT INTO brands (id, tenant_id, client_id, name, domain, industry, monitoring_enabled, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err := s.db.ExecContext(ctx, query,
		brand.ID, brand.TenantID, brand.ClientID, brand.Name, brand.PrimaryDomain, brand.Industry, brand.MonitoringEnabled, brand.CreatedAt, brand.UpdatedAt,
	)
	return err
}

func (s *BrandService) Update(ctx context.Context, brand *models.Brand) error {
	query := `UPDATE brands SET name = $1, domain = $2, industry = $3, monitoring_enabled = $4, updated_at = $5 
			  WHERE id = $6 AND tenant_id = $7`
	
	res, err := s.db.ExecContext(ctx, query,
		brand.Name, brand.PrimaryDomain, brand.Industry, brand.MonitoringEnabled, time.Now(), brand.ID, brand.TenantID,
	)
	if err != nil {
		return err
	}
	
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *BrandService) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `DELETE FROM brands WHERE id = $1 AND tenant_id = $2`
	
	res, err := s.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return err
	}
	
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
