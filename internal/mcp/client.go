package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMCPUnavailable  = errors.New("MCP service unavailable")
	ErrMCPTimeout      = errors.New("MCP request timeout")
	ErrMCPUnauthorized = errors.New("MCP unauthorized")
	ErrMCPForbidden    = errors.New("MCP forbidden - tool not allowed")
	ErrMCPNotFound     = errors.New("MCP resource not found")
	ErrMCPRateLimit    = errors.New("MCP rate limit exceeded")
)

// MCPClient cliente para comunicação com AGNO Control Plane
type MCPClient struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
}

// MCPConfig configuração do cliente MCP
type MCPConfig struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
}

// NewMCPClient cria um novo cliente MCP
func NewMCPClient(config MCPConfig) *MCPClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}

	return &MCPClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		maxRetries: config.MaxRetries,
		retryDelay: config.RetryDelay,
	}
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// MCPRequest estrutura de request para o MCP
type MCPRequest struct {
	// Identificadores
	RequestID string    `json:"request_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	ClientID  *uuid.UUID `json:"client_id,omitempty"`
	UserID    uuid.UUID `json:"user_id"`
	
	// Operação
	Tool      string                 `json:"tool"`
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params"`
	
	// Metadata
	Scopes    []string `json:"scopes"`
	Priority  string   `json:"priority,omitempty"`
	Async     bool     `json:"async,omitempty"`
}

// MCPResponse estrutura de response do MCP
type MCPResponse struct {
	Success   bool                   `json:"success"`
	RequestID string                 `json:"request_id"`
	JobID     string                 `json:"job_id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     *MCPError              `json:"error,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// MCPError estrutura de erro do MCP
type MCPError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// =============================================================================
// HUNTING OPERATIONS
// =============================================================================

// HuntRequest request para hunting
type HuntRequest struct {
	Target       string   `json:"target"`
	IncludeLeaks bool     `json:"include_leaks"`
	DeepAnalysis bool     `json:"deep_analysis"`
	Keywords     []string `json:"keywords,omitempty"`
}

// HuntResponse response de hunting
type HuntResponse struct {
	HuntID    uuid.UUID              `json:"hunt_id"`
	TenantID  uuid.UUID              `json:"tenant_id"`
	ClientID  *uuid.UUID             `json:"client_id,omitempty"`
	Target    string                 `json:"target"`
	Status    string                 `json:"status"`
	Results   map[string]interface{} `json:"results,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// Hunt executa uma operação de hunting
func (c *MCPClient) Hunt(ctx context.Context, req *MCPRequest, huntReq *HuntRequest) (*HuntResponse, error) {
	req.Tool = "hunting"
	req.Action = "hunt"
	req.Params = map[string]interface{}{
		"target":        huntReq.Target,
		"include_leaks": huntReq.IncludeLeaks,
		"deep_analysis": huntReq.DeepAnalysis,
		"keywords":      huntReq.Keywords,
	}

	resp, err := c.execute(ctx, "/v1/hunt", req)
	if err != nil {
		return nil, err
	}

	huntResp := &HuntResponse{
		HuntID:    uuid.New(),
		TenantID:  req.TenantID,
		ClientID:  req.ClientID,
		Target:    huntReq.Target,
		Status:    "completed",
		Results:   resp.Data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if resp.JobID != "" {
		huntResp.HuntID = uuid.MustParse(resp.JobID)
		huntResp.Status = "processing"
	}

	return huntResp, nil
}

// =============================================================================
// SCAN OPERATIONS
// =============================================================================

// ScanRequest request para scan
type ScanRequest struct {
	URL          string   `json:"url"`
	CaptureTypes []string `json:"capture_types"` // png, pdf, har
	FollowRedirects bool  `json:"follow_redirects"`
}

// ScanResponse response de scan
type ScanResponse struct {
	ScanID     uuid.UUID              `json:"scan_id"`
	TenantID   uuid.UUID              `json:"tenant_id"`
	ClientID   *uuid.UUID             `json:"client_id,omitempty"`
	URL        string                 `json:"url"`
	Status     string                 `json:"status"`
	Results    map[string]interface{} `json:"results,omitempty"`
	Artifacts  []string               `json:"artifacts,omitempty"`
	Timestamp  string                 `json:"timestamp"`
}

// ScanURL executa um scan de URL
func (c *MCPClient) ScanURL(ctx context.Context, req *MCPRequest, scanReq *ScanRequest) (*ScanResponse, error) {
	req.Tool = "scanner"
	req.Action = "site_scan"
	req.Params = map[string]interface{}{
		"url":              scanReq.URL,
		"capture_types":    scanReq.CaptureTypes,
		"follow_redirects": scanReq.FollowRedirects,
	}

	resp, err := c.execute(ctx, "/v1/scan", req)
	if err != nil {
		return nil, err
	}

	scanResp := &ScanResponse{
		ScanID:    uuid.New(),
		TenantID:  req.TenantID,
		ClientID:  req.ClientID,
		URL:       scanReq.URL,
		Status:    "completed",
		Results:   resp.Data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if resp.JobID != "" {
		scanResp.ScanID = uuid.MustParse(resp.JobID)
	}

	return scanResp, nil
}

// =============================================================================
// MONITOR OPERATIONS
// =============================================================================

// MonitorJobRequest request para criar job de monitoramento
type MonitorJobRequest struct {
	BrandID       uuid.UUID `json:"brand_id"`
	Target        string    `json:"target"`
	IntervalMins  int       `json:"interval_mins"`
	EnabledChecks []string  `json:"enabled_checks"`
}

// MonitorJobResponse response de job de monitoramento
type MonitorJobResponse struct {
	JobID     uuid.UUID `json:"job_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	ClientID  *uuid.UUID `json:"client_id,omitempty"`
	BrandID   uuid.UUID `json:"brand_id"`
	Status    string    `json:"status"`
	NextRunAt string    `json:"next_run_at,omitempty"`
	Timestamp string    `json:"timestamp"`
}

// CreateMonitorJob cria um job de monitoramento
func (c *MCPClient) CreateMonitorJob(ctx context.Context, req *MCPRequest, monitorReq *MonitorJobRequest) (*MonitorJobResponse, error) {
	req.Tool = "monitor"
	req.Action = "create_job"
	req.Params = map[string]interface{}{
		"brand_id":       monitorReq.BrandID.String(),
		"target":         monitorReq.Target,
		"interval_mins":  monitorReq.IntervalMins,
		"enabled_checks": monitorReq.EnabledChecks,
	}

	resp, err := c.execute(ctx, "/v1/monitor/jobs", req)
	if err != nil {
		return nil, err
	}

	jobID := uuid.New()
	if resp.JobID != "" {
		parsedID, err := uuid.Parse(resp.JobID)
		if err == nil {
			jobID = parsedID
		}
	}

	return &MonitorJobResponse{
		JobID:     jobID,
		TenantID:  req.TenantID,
		ClientID:  req.ClientID,
		BrandID:   monitorReq.BrandID,
		Status:    "running",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// StopMonitorJob para um job de monitoramento
func (c *MCPClient) StopMonitorJob(ctx context.Context, req *MCPRequest, jobID uuid.UUID) error {
	req.Tool = "monitor"
	req.Action = "stop_job"
	req.Params = map[string]interface{}{
		"job_id": jobID.String(),
	}

	_, err := c.execute(ctx, fmt.Sprintf("/v1/monitor/jobs/%s/stop", jobID), req)
	return err
}

// =============================================================================
// ANALYZE OPERATIONS
// =============================================================================

// AnalyzeRequest request para análise
type AnalyzeRequest struct {
	URL          string `json:"url"`
	Domain       string `json:"domain,omitempty"`
	IncludeLeaks bool   `json:"include_leaks"`
	DeepAnalysis bool   `json:"deep_analysis"`
}

// AnalyzeResponse response de análise
type AnalyzeResponse struct {
	AnalysisID uuid.UUID              `json:"analysis_id"`
	TenantID   uuid.UUID              `json:"tenant_id"`
	ClientID   *uuid.UUID             `json:"client_id,omitempty"`
	URL        string                 `json:"url"`
	Status     string                 `json:"status"`
	Analysis   map[string]interface{} `json:"analysis,omitempty"`
	Timestamp  string                 `json:"timestamp"`
}

// AnalyzeURL executa análise de URL
func (c *MCPClient) AnalyzeURL(ctx context.Context, req *MCPRequest, analyzeReq *AnalyzeRequest) (*AnalyzeResponse, error) {
	req.Tool = "analyzer"
	req.Action = "analyze_url"
	req.Params = map[string]interface{}{
		"url":           analyzeReq.URL,
		"include_leaks": analyzeReq.IncludeLeaks,
		"deep_analysis": analyzeReq.DeepAnalysis,
	}

	resp, err := c.execute(ctx, "/v1/analyze", req)
	if err != nil {
		return nil, err
	}

	analysisID := uuid.New()
	if resp.JobID != "" {
		parsedID, err := uuid.Parse(resp.JobID)
		if err == nil {
			analysisID = parsedID
		}
	}

	return &AnalyzeResponse{
		AnalysisID: analysisID,
		TenantID:   req.TenantID,
		ClientID:   req.ClientID,
		URL:        analyzeReq.URL,
		Status:     "completed",
		Analysis:   resp.Data,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// =============================================================================
// LEAK SEARCH OPERATIONS
// =============================================================================

// LeakSearchRequest request para busca de vazamentos
type LeakSearchRequest struct {
	Query   string `json:"query"`
	Type    string `json:"type"` // domain, email, keyword
	MaxResults int `json:"max_results"`
}

// LeakSearchResponse response de busca de vazamentos
type LeakSearchResponse struct {
	SearchID  uuid.UUID              `json:"search_id"`
	TenantID  uuid.UUID              `json:"tenant_id"`
	ClientID  *uuid.UUID             `json:"client_id,omitempty"`
	Query     string                 `json:"query"`
	Results   []map[string]interface{} `json:"results"`
	Total     int                    `json:"total"`
	Timestamp string                 `json:"timestamp"`
}

// SearchLeaks busca vazamentos
func (c *MCPClient) SearchLeaks(ctx context.Context, req *MCPRequest, searchReq *LeakSearchRequest) (*LeakSearchResponse, error) {
	req.Tool = "leaks"
	req.Action = "leak_search"
	req.Params = map[string]interface{}{
		"query":       searchReq.Query,
		"type":        searchReq.Type,
		"max_results": searchReq.MaxResults,
	}

	resp, err := c.execute(ctx, "/v1/leaks/search", req)
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0)
	if data, ok := resp.Data["results"].([]interface{}); ok {
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				results = append(results, m)
			}
		}
	}

	return &LeakSearchResponse{
		SearchID:  uuid.New(),
		TenantID:  req.TenantID,
		ClientID:  req.ClientID,
		Query:     searchReq.Query,
		Results:   results,
		Total:     len(results),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// =============================================================================
// PROXY METHODS
// =============================================================================

// ProxyRequest faz proxy de uma request genérica para o Core Python
func (c *MCPClient) ProxyRequest(ctx context.Context, endpoint string, req *MCPRequest) (*MCPResponse, error) {
	return c.execute(ctx, endpoint, req)
}

// =============================================================================
// INTERNAL METHODS
// =============================================================================

// execute executa uma request para o MCP com retry
func (c *MCPClient) execute(ctx context.Context, endpoint string, req *MCPRequest) (*MCPResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay * time.Duration(attempt))
		}

		resp, err := c.doRequest(ctx, endpoint, req)
		if err != nil {
			lastErr = err
			// Não fazer retry para erros de autorização/forbidden
			if errors.Is(err, ErrMCPUnauthorized) || errors.Is(err, ErrMCPForbidden) {
				return nil, err
			}
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("MCP request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// doRequest executa uma request HTTP para o MCP
func (c *MCPClient) doRequest(ctx context.Context, endpoint string, req *MCPRequest) (*MCPResponse, error) {
	// Serializar request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Criar HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Request-ID", req.RequestID)
	httpReq.Header.Set("X-Tenant-ID", req.TenantID.String())
	if req.ClientID != nil {
		httpReq.Header.Set("X-Client-ID", req.ClientID.String())
	}

	// Executar request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMCPUnavailable, err)
	}
	defer httpResp.Body.Close()

	// Ler response
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Verificar status code
	switch httpResp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		// OK
	case http.StatusUnauthorized:
		return nil, ErrMCPUnauthorized
	case http.StatusForbidden:
		return nil, ErrMCPForbidden
	case http.StatusNotFound:
		return nil, ErrMCPNotFound
	case http.StatusTooManyRequests:
		return nil, ErrMCPRateLimit
	default:
		return nil, fmt.Errorf("MCP returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	// Deserializar response
	var mcpResp MCPResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !mcpResp.Success && mcpResp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s - %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}

	return &mcpResp, nil
}

// HealthCheck verifica se o MCP está disponível
func (c *MCPClient) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ErrMCPUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrMCPUnavailable
	}

	return nil
}
