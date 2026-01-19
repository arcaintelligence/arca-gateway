package middleware

import "github.com/arcaintelligence/arca-gateway/internal/models"

// Scope constants for use in middleware
const (
	ScopeHuntingRead  = models.ScopeHuntingRead
	ScopeHuntingWrite = models.ScopeHuntingWrite
	ScopeMonitorRead  = models.ScopeMonitorRead
	ScopeMonitorWrite = models.ScopeMonitorWrite
	ScopeAnalyzeRead  = models.ScopeAnalyzeRead
	ScopeAnalyzeWrite = models.ScopeAnalyzeWrite
	ScopeAlertsRead   = models.ScopeAlertsRead
	ScopeAlertsWrite  = models.ScopeAlertsWrite
	ScopeClientsRead  = models.ScopeClientsRead
	ScopeClientsWrite = models.ScopeClientsWrite
	ScopeBrandsRead   = models.ScopeBrandsRead
	ScopeBrandsWrite  = models.ScopeBrandsWrite
	ScopeReportsRead  = models.ScopeReportsRead
	ScopeReportsWrite = models.ScopeReportsWrite
	ScopeAdminRead    = models.ScopeAdminRead
	ScopeAdminWrite   = models.ScopeAdminWrite
)
