package models

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// ENUMS E CONSTANTES
// =============================================================================

// Role representa o nível de permissão do usuário
type Role string

const (
	RoleAdmin   Role = "admin"   // Acesso total
	RoleManager Role = "manager" // Gerencia tenants e clientes
	RoleAnalyst Role = "analyst" // Executa hunting e análises
	RoleViewer  Role = "viewer"  // Apenas visualização
	RoleAPI     Role = "api"     // Acesso programático (webhooks)
)

// Scope representa permissões granulares para features/produtos
type Scope string

const (
	// Hunting scopes
	ScopeHuntingRead  Scope = "hunting:read"
	ScopeHuntingWrite Scope = "hunting:write"

	// Monitor scopes
	ScopeMonitorRead  Scope = "monitor:read"
	ScopeMonitorWrite Scope = "monitor:write"

	// Analyze scopes
	ScopeAnalyzeRead  Scope = "analyze:read"
	ScopeAnalyzeWrite Scope = "analyze:write"

	// Alerts scopes
	ScopeAlertsRead  Scope = "alerts:read"
	ScopeAlertsWrite Scope = "alerts:write"

	// Clients scopes
	ScopeClientsRead  Scope = "clients:read"
	ScopeClientsWrite Scope = "clients:write"

	// Brands scopes
	ScopeBrandsRead  Scope = "brands:read"
	ScopeBrandsWrite Scope = "brands:write"

	// Reports scopes
	ScopeReportsRead  Scope = "reports:read"
	ScopeReportsWrite Scope = "reports:write"

	// Admin scopes
	ScopeAdminRead  Scope = "admin:read"
	ScopeAdminWrite Scope = "admin:write"
)

// Status representa o status de uma entidade
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusPending  Status = "pending"
	StatusSuspended Status = "suspended"
)

// =============================================================================
// MODELOS PRINCIPAIS
// =============================================================================

// Tenant representa uma organização/empresa na plataforma
type Tenant struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Slug        string            `json:"slug" db:"slug"`
	Email       string            `json:"email" db:"email"`
	Plan        string            `json:"plan" db:"plan"`
	Status      Status            `json:"status" db:"status"`
	Settings    TenantSettings    `json:"settings" db:"settings"`
	Quotas      TenantQuotas      `json:"quotas" db:"quotas"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// TenantSettings configurações específicas do tenant
type TenantSettings struct {
	AllowedScopes    []Scope  `json:"allowed_scopes"`
	AllowedTools     []string `json:"allowed_tools"`
	WebhookURL       string   `json:"webhook_url,omitempty"`
	SlackWebhook     string   `json:"slack_webhook,omitempty"`
	EmailNotify      bool     `json:"email_notify"`
	MaxConcurrentJobs int     `json:"max_concurrent_jobs"`
}

// TenantQuotas quotas de uso do tenant
type TenantQuotas struct {
	MaxClients       int `json:"max_clients"`
	MaxBrands        int `json:"max_brands"`
	MaxScansPerDay   int `json:"max_scans_per_day"`
	MaxAlertsPerDay  int `json:"max_alerts_per_day"`
	MaxUsersPerTenant int `json:"max_users_per_tenant"`
	StorageLimitMB   int `json:"storage_limit_mb"`
}

// User representa um usuário da plataforma
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Name         string    `json:"name" db:"name"`
	Role         Role      `json:"role" db:"role"`
	Scopes       []Scope   `json:"scopes" db:"scopes"`
	Status       Status    `json:"status" db:"status"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Client representa um cliente do tenant (empresa monitorada)
type Client struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	TenantID    uuid.UUID      `json:"tenant_id" db:"tenant_id"`
	Name        string         `json:"name" db:"name"`
	Slug        string         `json:"slug" db:"slug"`
	Description string         `json:"description,omitempty" db:"description"`
	Industry    string         `json:"industry,omitempty" db:"industry"`
	Status      Status         `json:"status" db:"status"`
	Settings    ClientSettings `json:"settings" db:"settings"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// ClientSettings configurações específicas do cliente
type ClientSettings struct {
	AlertEmail       string   `json:"alert_email,omitempty"`
	AlertWebhook     string   `json:"alert_webhook,omitempty"`
	ScanFrequency    string   `json:"scan_frequency"` // hourly, daily, weekly
	Priority         string   `json:"priority"`       // low, medium, high, critical
	AutoTakedown     bool     `json:"auto_takedown"`
	WhitelistDomains []string `json:"whitelist_domains,omitempty"`
}

// Brand representa uma marca/domínio monitorado
type Brand struct {
	ID              uuid.UUID     `json:"id" db:"id"`
	ClientID        uuid.UUID     `json:"client_id" db:"client_id"`
	TenantID        uuid.UUID     `json:"tenant_id" db:"tenant_id"`
	Name            string        `json:"name" db:"name"`
	PrimaryDomain   string        `json:"primary_domain" db:"primary_domain"`
	Status          Status        `json:"status" db:"status"`
	Config          BrandConfig   `json:"config" db:"config"`
	MonitoringJobID *uuid.UUID    `json:"monitoring_job_id,omitempty" db:"monitoring_job_id"`
	LastScanAt      *time.Time    `json:"last_scan_at,omitempty" db:"last_scan_at"`
	ThreatsFound    int           `json:"threats_found" db:"threats_found"`
	CreatedAt       time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at" db:"updated_at"`
}

// BrandConfig configurações de monitoramento da marca
type BrandConfig struct {
	// Domínios e variações
	AdditionalDomains  []string `json:"additional_domains,omitempty"`
	KnownVariations    []string `json:"known_variations,omitempty"`
	Keywords           []string `json:"keywords,omitempty"`
	
	// Configurações de scan
	ScanFrequencyMins  int      `json:"scan_frequency_mins"`
	EnableLeakSearch   bool     `json:"enable_leak_search"`
	EnableDomainWatch  bool     `json:"enable_domain_watch"`
	EnableDeepAnalysis bool     `json:"enable_deep_analysis"`
	
	// Alertas
	AlertSeverityMin   string   `json:"alert_severity_min"` // info, low, medium, high, critical
	AlertChannels      []string `json:"alert_channels"`     // email, slack, webhook, sms
	
	// Whitelist
	WhitelistDomains   []string `json:"whitelist_domains,omitempty"`
	WhitelistIPs       []string `json:"whitelist_ips,omitempty"`
}

// =============================================================================
// MODELOS DE OPERAÇÃO
// =============================================================================

// MonitoringJob representa um job de monitoramento
type MonitoringJob struct {
	ID           uuid.UUID         `json:"id" db:"id"`
	BrandID      uuid.UUID         `json:"brand_id" db:"brand_id"`
	ClientID     uuid.UUID         `json:"client_id" db:"client_id"`
	TenantID     uuid.UUID         `json:"tenant_id" db:"tenant_id"`
	Status       string            `json:"status" db:"status"` // running, paused, stopped, error
	Config       MonitoringConfig  `json:"config" db:"config"`
	Stats        MonitoringStats   `json:"stats" db:"stats"`
	LastRunAt    *time.Time        `json:"last_run_at,omitempty" db:"last_run_at"`
	NextRunAt    *time.Time        `json:"next_run_at,omitempty" db:"next_run_at"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
}

// MonitoringConfig configuração do job de monitoramento
type MonitoringConfig struct {
	IntervalMins       int      `json:"interval_mins"`
	EnabledChecks      []string `json:"enabled_checks"` // phishing, leak, domain, ssl
	MaxConcurrentScans int      `json:"max_concurrent_scans"`
}

// MonitoringStats estatísticas do job de monitoramento
type MonitoringStats struct {
	TotalScans      int       `json:"total_scans"`
	ThreatsFound    int       `json:"threats_found"`
	LastThreatAt    *time.Time `json:"last_threat_at,omitempty"`
	AvgScanDuration float64   `json:"avg_scan_duration_ms"`
	ErrorCount      int       `json:"error_count"`
}

// Alert representa um alerta gerado
type Alert struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	BrandID     uuid.UUID    `json:"brand_id" db:"brand_id"`
	ClientID    uuid.UUID    `json:"client_id" db:"client_id"`
	TenantID    uuid.UUID    `json:"tenant_id" db:"tenant_id"`
	Type        string       `json:"type" db:"type"`           // phishing, leak, domain, ssl
	Severity    string       `json:"severity" db:"severity"`   // info, low, medium, high, critical
	Title       string       `json:"title" db:"title"`
	Description string       `json:"description" db:"description"`
	Details     AlertDetails `json:"details" db:"details"`
	Status      string       `json:"status" db:"status"`       // new, acknowledged, resolved, false_positive
	ResolvedAt  *time.Time   `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy  *uuid.UUID   `json:"resolved_by,omitempty" db:"resolved_by"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

// AlertDetails detalhes específicos do alerta
type AlertDetails struct {
	URL           string   `json:"url,omitempty"`
	Domain        string   `json:"domain,omitempty"`
	IP            string   `json:"ip,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
	IOCs          []string `json:"iocs,omitempty"`
	Confidence    float64  `json:"confidence"`
	AnalysisID    string   `json:"analysis_id,omitempty"`
	ScreenshotURL string   `json:"screenshot_url,omitempty"`
}

// =============================================================================
// MODELOS DE AUDITORIA
// =============================================================================

// AuditLog registro de auditoria
type AuditLog struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	TenantID   uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	UserID     *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
	Action     string                 `json:"action" db:"action"`
	Resource   string                 `json:"resource" db:"resource"`
	ResourceID *uuid.UUID             `json:"resource_id,omitempty" db:"resource_id"`
	Details    map[string]interface{} `json:"details,omitempty" db:"details"`
	IP         string                 `json:"ip" db:"ip"`
	UserAgent  string                 `json:"user_agent" db:"user_agent"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// =============================================================================
// HELPERS
// =============================================================================

// GetDefaultScopesForRole retorna os scopes padrão para um role
func GetDefaultScopesForRole(role Role) []Scope {
	switch role {
	case RoleAdmin:
		return []Scope{
			ScopeHuntingRead, ScopeHuntingWrite,
			ScopeMonitorRead, ScopeMonitorWrite,
			ScopeAnalyzeRead, ScopeAnalyzeWrite,
			ScopeAlertsRead, ScopeAlertsWrite,
			ScopeClientsRead, ScopeClientsWrite,
			ScopeBrandsRead, ScopeBrandsWrite,
			ScopeReportsRead, ScopeReportsWrite,
			ScopeAdminRead, ScopeAdminWrite,
		}
	case RoleManager:
		return []Scope{
			ScopeHuntingRead, ScopeHuntingWrite,
			ScopeMonitorRead, ScopeMonitorWrite,
			ScopeAnalyzeRead, ScopeAnalyzeWrite,
			ScopeAlertsRead, ScopeAlertsWrite,
			ScopeClientsRead, ScopeClientsWrite,
			ScopeBrandsRead, ScopeBrandsWrite,
			ScopeReportsRead, ScopeReportsWrite,
		}
	case RoleAnalyst:
		return []Scope{
			ScopeHuntingRead, ScopeHuntingWrite,
			ScopeMonitorRead,
			ScopeAnalyzeRead, ScopeAnalyzeWrite,
			ScopeAlertsRead, ScopeAlertsWrite,
			ScopeClientsRead,
			ScopeBrandsRead,
			ScopeReportsRead,
		}
	case RoleViewer:
		return []Scope{
			ScopeHuntingRead,
			ScopeMonitorRead,
			ScopeAnalyzeRead,
			ScopeAlertsRead,
			ScopeClientsRead,
			ScopeBrandsRead,
			ScopeReportsRead,
		}
	case RoleAPI:
		return []Scope{
			ScopeHuntingRead, ScopeHuntingWrite,
			ScopeMonitorRead, ScopeMonitorWrite,
			ScopeAlertsRead,
		}
	default:
		return []Scope{}
	}
}

// HasScope verifica se o usuário tem um scope específico
func (u *User) HasScope(scope Scope) bool {
	for _, s := range u.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasAnyScope verifica se o usuário tem pelo menos um dos scopes
func (u *User) HasAnyScope(scopes ...Scope) bool {
	for _, scope := range scopes {
		if u.HasScope(scope) {
			return true
		}
	}
	return false
}

// HasAllScopes verifica se o usuário tem todos os scopes
func (u *User) HasAllScopes(scopes ...Scope) bool {
	for _, scope := range scopes {
		if !u.HasScope(scope) {
			return false
		}
	}
	return true
}
