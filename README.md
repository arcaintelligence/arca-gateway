<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.24"/>
  <img src="https://img.shields.io/badge/Fiber-v2-00ACD7?style=for-the-badge&logo=go&logoColor=white" alt="Fiber"/>
  <img src="https://img.shields.io/badge/License-Proprietary-red?style=for-the-badge" alt="License"/>
  <img src="https://img.shields.io/badge/Status-Production-green?style=for-the-badge" alt="Status"/>
</p>

<h1 align="center">
  <br>
  🛡️ ARCA Gateway
  <br>
</h1>

<h4 align="center">High-Performance API Gateway for Digital Risk Protection</h4>

<p align="center">
  <strong>ARCA Intelligence</strong> — Rio de Janeiro, Brasil
</p>

<p align="center">
  <a href="#visão-geral">Visão Geral</a> •
  <a href="#arquitetura">Arquitetura</a> •
  <a href="#instalação">Instalação</a> •
  <a href="#api-reference">API Reference</a> •
  <a href="#segurança">Segurança</a> •
  <a href="#observabilidade">Observabilidade</a>
</p>

---

## Visão Geral

O **ARCA Gateway** é o ponto de entrada único da plataforma ARCA Intelligence, responsável por autenticação, autorização, rate limiting e roteamento de requisições para o AGNO Control Plane. Construído com Go e Fiber (fasthttp), oferece performance excepcional e segurança enterprise-grade.

### Principais Funcionalidades

| Funcionalidade | Descrição |
|----------------|-----------|
| **JWT Authentication** | Autenticação multi-nível com roles e scopes granulares |
| **MCP Integration** | Integração nativa com AGNO via Model Context Protocol |
| **Rate Limiting** | Limitação por tenant e por rota com backend Redis |
| **Security Layer** | CORS, Helmet, mTLS, IP filtering e audit logging |
| **Observability** | Métricas Prometheus, tracing distribuído e logs estruturados |
| **Multi-Tenant** | Isolamento completo entre tenants com tenant_id tracking |

---

## Arquitetura

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              ARCA Platform                                   │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ARCA Gateway (Go/Fiber)                              │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                          Security Layer                                │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │  │
│  │  │ JWT Auth    │  │ Rate Limit  │  │ CORS/Helmet │  │ Audit Log   │  │  │
│  │  │ + Scopes    │  │ + Redis     │  │ + mTLS      │  │ + Tracing   │  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                     │                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                          Routing Layer                                 │  │
│  │  /v1/auth/*     → Auth Handler (login, register, tokens)              │  │
│  │  /v1/clients/*  → Client Handler (CRUD, brands)                       │  │
│  │  /v1/hunting/*  → Hunting Handler (hunt, scan, analyze) → MCP         │  │
│  │  /v1/monitor/*  → Monitor Handler (jobs, alerts) → MCP                │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    AGNO Control Plane (Python/FastAPI)                       │
│                    MCP Server + AI Agents + Policies                         │
│                    Memory + RAG + Knowledge Base                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           AWS Infrastructure                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐             │
│  │   OpenSearch    │  │       S3        │  │    CloudWatch   │             │
│  │   (Memory)      │  │   (Artifacts)   │  │   (Monitoring)  │             │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Fluxo de Requisição

```
1. Cliente → ARCA Gateway (:8080)
2. Security Layer: JWT validation + Rate limit check
3. Routing Layer: Route to appropriate handler
4. Handler: Validate request + Build MCP request
5. MCP Client: Send to AGNO Control Plane (:8001)
6. AGNO: Process with AI/ML + Memory + Knowledge
7. Response: Return through gateway with audit log
```

---

## Estrutura do Projeto

```
arca-gateway/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── auth/
│   │   └── jwt.go               # JWT token management
│   ├── config/
│   │   └── config.go            # Configuration loader
│   ├── handlers/
│   │   ├── auth_handler.go      # Authentication endpoints
│   │   ├── client_handler.go    # Client/Brand management
│   │   └── hunting_handler.go   # Hunting/Scan/Monitor
│   ├── mcp/
│   │   └── client.go            # MCP client for AGNO
│   ├── middleware/
│   │   ├── auth.go              # JWT authentication
│   │   ├── metrics.go           # Prometheus metrics
│   │   ├── ratelimit.go         # Rate limiting
│   │   ├── scopes.go            # Scope validation
│   │   └── security.go          # CORS, Helmet, etc.
│   ├── models/
│   │   └── models.go            # Domain models
│   └── services/
│       └── services.go          # Business logic
├── pkg/
│   ├── logger/
│   │   └── logger.go            # Structured logging
│   └── response/
│       └── response.go          # Standard responses
├── config/
│   └── prometheus.yml           # Prometheus config
├── Dockerfile                   # Multi-stage build
├── docker-compose.yml           # Full stack
├── go.mod
└── go.sum
```

---

## Instalação

### Pré-requisitos

- Go 1.22+
- Docker & Docker Compose
- Redis (para rate limiting)
- PostgreSQL (para persistência)

### Desenvolvimento Local

```bash
# Clonar repositório
git clone https://github.com/arcaintelligence/arca-gateway.git
cd arca-gateway

# Instalar dependências
go mod download

# Configurar variáveis de ambiente
export ENVIRONMENT=development
export SERVER_PORT=8080
export JWT_SECRET=your-super-secret-key
export MCP_BASE_URL=http://localhost:8001

# Executar servidor
go run cmd/server/main.go
```

### Docker Compose (Stack Completa)

```bash
# Iniciar todos os serviços
docker-compose up -d

# Verificar logs
docker-compose logs -f gateway

# Verificar health
curl http://localhost:8080/health

# Parar serviços
docker-compose down
```

### Build para Produção

```bash
# Build otimizado
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o arca-gateway \
    ./cmd/server

# Docker build
docker build -t arca-gateway:latest .
```

---

## Configuração

### Variáveis de Ambiente

| Variável | Descrição | Default |
|----------|-----------|---------|
| `ENVIRONMENT` | Ambiente (development/staging/production) | development |
| `SERVER_HOST` | Host do servidor | 0.0.0.0 |
| `SERVER_PORT` | Porta do servidor | 8080 |
| `JWT_SECRET` | Chave secreta para JWT | - |
| `JWT_ACCESS_EXPIRY` | Expiração do access token | 15m |
| `JWT_REFRESH_EXPIRY` | Expiração do refresh token | 7d |
| `MCP_BASE_URL` | URL do AGNO Control Plane | http://localhost:8001 |
| `MCP_TIMEOUT` | Timeout para requisições MCP | 30s |
| `REDIS_HOST` | Host do Redis | localhost |
| `REDIS_PORT` | Porta do Redis | 6379 |
| `DB_HOST` | Host do PostgreSQL | localhost |
| `DB_PORT` | Porta do PostgreSQL | 5432 |
| `DB_USER` | Usuário do PostgreSQL | arca |
| `DB_PASSWORD` | Senha do PostgreSQL | - |
| `DB_NAME` | Nome do banco | arca |

---

## API Reference

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "services": {
    "gateway": "healthy",
    "mcp": "healthy"
  },
  "timestamp": "2026-01-20T15:00:00Z"
}
```

---

### Authentication

#### Login

```http
POST /v1/auth/login
Content-Type: application/json

{
  "email": "user@company.com",
  "password": "secure_password"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

#### Register Tenant

```http
POST /v1/auth/register
Content-Type: application/json

{
  "company_name": "Empresa LTDA",
  "email": "admin@empresa.com.br",
  "password": "secure_password",
  "plan": "enterprise"
}
```

#### Refresh Token

```http
POST /v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### Get Current User

```http
GET /v1/auth/me
Authorization: Bearer {access_token}
```

---

### Clients & Brands

#### List Clients

```http
GET /v1/clients
Authorization: Bearer {access_token}
```

**Required Scope:** `clients:read`

#### Create Client

```http
POST /v1/clients
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "name": "Cliente Importante",
  "document": "12.345.678/0001-90",
  "contact_email": "contato@cliente.com.br"
}
```

**Required Scope:** `clients:write`

#### Create Brand

```http
POST /v1/clients/{client_id}/brands
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "name": "Marca Principal",
  "domain": "marca.com.br",
  "keywords": ["marca", "produto", "empresa"]
}
```

**Required Scope:** `brands:write`

#### Start Brand Monitoring

```http
POST /v1/clients/{client_id}/brands/{brand_id}/monitoring/start
Authorization: Bearer {access_token}
```

**Required Scope:** `monitor:write`

---

### Hunting & Analysis

#### Execute Hunt

```http
POST /v1/hunting/hunt
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "target": "marca.com.br",
  "include_leaks": true,
  "deep_analysis": true,
  "keywords": ["marca", "produto"],
  "client_id": "uuid-do-cliente"
}
```

**Required Scope:** `hunting:write`

**Response:**
```json
{
  "success": true,
  "data": {
    "hunt_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "tenant-uuid",
    "client_id": "client-uuid",
    "target": "marca.com.br",
    "status": "completed",
    "results": {
      "phishing_sites": [],
      "domain_variations": [],
      "social_media": [],
      "leaks": []
    },
    "timestamp": "2026-01-20T15:00:00Z"
  }
}
```

#### Scan URL

```http
POST /v1/hunting/scan
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "url": "https://suspicious-site.com",
  "capture_types": ["png", "pdf", "har"],
  "follow_redirects": true
}
```

**Required Scope:** `hunting:write`

#### Analyze URL

```http
POST /v1/hunting/analyze
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "url": "https://suspicious-site.com",
  "include_leaks": true,
  "deep_analysis": true
}
```

**Required Scope:** `analyze:write`

#### Search Leaks

```http
POST /v1/hunting/leaks/search
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "query": "empresa.com.br",
  "type": "domain",
  "max_results": 100
}
```

**Required Scope:** `hunting:read`

---

### Monitoring

#### Create Monitor Job

```http
POST /v1/monitor/jobs
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "brand_id": "brand-uuid",
  "target": "marca.com.br",
  "interval_mins": 60,
  "enabled_checks": ["phishing", "domain", "ssl", "social"]
}
```

**Required Scope:** `monitor:write`

#### Stop Monitor Job

```http
POST /v1/monitor/jobs/{job_id}/stop
Authorization: Bearer {access_token}
```

**Required Scope:** `monitor:write`

---

## Segurança

### Sistema de Roles

| Role | Descrição | Permissões |
|------|-----------|------------|
| `admin` | Administrador do tenant | Acesso total |
| `manager` | Gerente de operações | Gerenciar clientes, usuários, marcas |
| `analyst` | Analista de segurança | Executar hunting, análises, monitoramento |
| `viewer` | Visualizador | Apenas leitura |
| `api` | Acesso programático | Webhooks e integrações |

### Sistema de Scopes

| Scope | Descrição |
|-------|-----------|
| `hunting:read` | Visualizar resultados de hunting |
| `hunting:write` | Executar operações de hunting |
| `monitor:read` | Visualizar jobs de monitoramento |
| `monitor:write` | Criar/gerenciar jobs de monitoramento |
| `analyze:read` | Visualizar resultados de análise |
| `analyze:write` | Executar análises |
| `alerts:read` | Visualizar alertas |
| `alerts:write` | Gerenciar alertas |
| `clients:read` | Visualizar clientes |
| `clients:write` | Gerenciar clientes |
| `brands:read` | Visualizar marcas |
| `brands:write` | Gerenciar marcas |
| `reports:read` | Visualizar relatórios |
| `reports:write` | Gerar relatórios |
| `admin:read` | Visualizar configurações admin |
| `admin:write` | Gerenciar configurações admin |

### Rate Limiting

```yaml
# Limites por tenant
default:
  requests_per_minute: 1000
  burst: 100

# Limites por rota
/v1/hunting/hunt:
  requests_per_minute: 20
  burst: 40

/v1/hunting/scan:
  requests_per_minute: 30
  burst: 60
```

### Headers de Segurança

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

---

## Observabilidade

### Métricas Prometheus

```
# Métricas disponíveis em /metrics
arca_gateway_requests_total{method, path, status}
arca_gateway_request_duration_seconds{method, path}
arca_gateway_active_connections
arca_gateway_mcp_requests_total{tool, action, status}
arca_gateway_mcp_request_duration_seconds{tool, action}
```

### Logs Estruturados

```json
{
  "level": "info",
  "timestamp": "2026-01-20T15:00:00Z",
  "request_id": "req-uuid",
  "tenant_id": "tenant-uuid",
  "user_id": "user-uuid",
  "method": "POST",
  "path": "/v1/hunting/hunt",
  "status": 200,
  "duration_ms": 150,
  "ip": "192.168.1.1"
}
```

### Grafana Dashboards

O docker-compose inclui Grafana pré-configurado com dashboards para:

- Request rate e latência
- Error rate por endpoint
- Rate limit hits
- MCP request metrics
- Resource utilization

---

## Integração com AGNO

### MCP Client

O gateway se comunica com o AGNO Control Plane via HTTP usando o protocolo MCP:

```go
// Exemplo de request MCP
mcpReq := &mcp.MCPRequest{
    RequestID: "unique-request-id",
    TenantID:  tenantUUID,
    ClientID:  clientUUID,
    UserID:    userUUID,
    Tool:      "hunting",
    Action:    "hunt",
    Params: map[string]interface{}{
        "target":        "domain.com",
        "include_leaks": true,
    },
    Scopes: []string{"hunting:write"},
}
```

### Retry e Circuit Breaker

```go
// Configuração de retry
MCPConfig{
    BaseURL:    "http://agno:8001",
    Timeout:    30 * time.Second,
    MaxRetries: 3,
    RetryDelay: 1 * time.Second,
}
```

---

## Deployment

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: arca-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: arca-gateway
  template:
    metadata:
      labels:
        app: arca-gateway
    spec:
      containers:
      - name: gateway
        image: arca-gateway:latest
        ports:
        - containerPort: 8080
        env:
        - name: ENVIRONMENT
          value: "production"
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: arca-secrets
              key: jwt-secret
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

### AWS ECS

```json
{
  "family": "arca-gateway",
  "containerDefinitions": [
    {
      "name": "gateway",
      "image": "arca-gateway:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {"name": "ENVIRONMENT", "value": "production"}
      ],
      "secrets": [
        {
          "name": "JWT_SECRET",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:639135896229:secret:arca/jwt-secret"
        }
      ],
      "healthCheck": {
        "command": ["CMD-SHELL", "wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1"],
        "interval": 30,
        "timeout": 5,
        "retries": 3
      }
    }
  ]
}
```

---

## Repositórios Relacionados

| Repositório | Descrição |
|-------------|-----------|
| [agno-python](https://github.com/arcaintelligence/agno-python) | AGNO Control Plane - AI/ML, RAG, Memory |
| [agno-arca](https://github.com/arcaintelligence/agno-arca) | ARCA Go - Business Logic Layer |
| [arca-gateway](https://github.com/arcaintelligence/arca-gateway) | API Gateway (este repositório) |

---

## Roadmap

- [ ] GraphQL endpoint
- [ ] WebSocket support para real-time alerts
- [ ] API versioning (v2)
- [ ] OpenAPI/Swagger documentation
- [ ] gRPC support para comunicação interna
- [ ] Distributed tracing com Jaeger
- [ ] A/B testing infrastructure

---

## Contribuição

Este é um projeto proprietário da ARCA Intelligence. Para contribuições internas:

1. Crie uma branch a partir de `develop`
2. Implemente as mudanças com testes
3. Abra um Pull Request para `develop`
4. Aguarde code review

---

## Licença

**Proprietary** - ARCA Intelligence © 2026

Todos os direitos reservados. Este software é confidencial e de propriedade exclusiva da ARCA Intelligence.

---

<p align="center">
  <strong>ARCA Intelligence</strong><br>
  Digital Risk Protection • Brand Protection • Fraud Prevention<br>
  Rio de Janeiro, Brasil
</p>
