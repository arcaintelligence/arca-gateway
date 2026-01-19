# ARCA Gateway

High-performance API Gateway for ARCA Intelligence Platform, built with Go and Fiber.

## Features

- **High Performance**: Built with Fiber (fasthttp), capable of handling millions of requests per minute
- **JWT Authentication**: Multi-level permissions with scopes for fine-grained access control
- **MCP Integration**: Native integration with AGNO Control Plane via MCP protocol
- **Rate Limiting**: Per-tenant and per-route rate limiting with Redis backend
- **Security**: CORS, Helmet, mTLS, IP filtering, and audit logging
- **Observability**: Structured logging, Prometheus metrics, and distributed tracing

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (Site ARCA)                     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    ARCA Gateway (Go/Fiber)                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                  Security Layer                      │   │
│  │  - JWT Auth with Scopes                             │   │
│  │  - Rate Limiting (Redis)                            │   │
│  │  - CORS, Helmet, mTLS                               │   │
│  └─────────────────────────────────────────────────────┘   │
│                          │                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                  Routing Layer                       │   │
│  │  - /v1/auth/*     → Auth Handler                    │   │
│  │  - /v1/clients/*  → Client Handler                  │   │
│  │  - /v1/hunting/*  → Hunting Handler (MCP)           │   │
│  │  - /v1/monitor/*  → Monitor Handler (MCP)           │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              AGNO Control Plane (Python/MCP)                │
│              MCP Server + Agents + Policies                 │
└─────────────────────────────────────────────────────────────┘
```

## Permission System

### Roles

| Role | Description |
|------|-------------|
| `admin` | Full access to all resources |
| `manager` | Manage tenants, clients, and users |
| `analyst` | Execute hunting and analysis operations |
| `viewer` | Read-only access |
| `api` | Programmatic access (webhooks, integrations) |

### Scopes

| Scope | Description |
|-------|-------------|
| `hunting:read` | View hunting results |
| `hunting:write` | Execute hunting operations |
| `monitor:read` | View monitoring jobs |
| `monitor:write` | Create/manage monitoring jobs |
| `analyze:read` | View analysis results |
| `analyze:write` | Execute analysis operations |
| `alerts:read` | View alerts |
| `alerts:write` | Manage alerts |
| `clients:read` | View clients |
| `clients:write` | Manage clients |
| `brands:read` | View brands |
| `brands:write` | Manage brands |
| `reports:read` | View reports |
| `reports:write` | Generate reports |
| `admin:read` | View admin settings |
| `admin:write` | Manage admin settings |

## API Endpoints

### Authentication

```
POST /v1/auth/register    # Register new tenant
POST /v1/auth/login       # Login
POST /v1/auth/refresh     # Refresh access token
POST /v1/auth/logout      # Logout
GET  /v1/auth/me          # Get current user
POST /v1/auth/api-key     # Generate API key
```

### Clients & Brands

```
GET    /v1/clients                           # List clients
POST   /v1/clients                           # Create client
GET    /v1/clients/:id                       # Get client
PUT    /v1/clients/:id                       # Update client
DELETE /v1/clients/:id                       # Delete client

GET    /v1/clients/:id/brands                # List brands
POST   /v1/clients/:id/brands                # Create brand
GET    /v1/clients/:id/brands/:brand_id      # Get brand
PUT    /v1/clients/:id/brands/:brand_id      # Update brand
DELETE /v1/clients/:id/brands/:brand_id      # Delete brand
POST   /v1/clients/:id/brands/:brand_id/monitoring/start  # Start monitoring
POST   /v1/clients/:id/brands/:brand_id/monitoring/stop   # Stop monitoring
```

### Hunting & Analysis

```
POST /v1/hunting/hunt         # Execute hunting
POST /v1/hunting/scan         # Scan URL
POST /v1/hunting/analyze      # Analyze URL
POST /v1/hunting/leaks/search # Search leaks
```

### Monitoring

```
POST /v1/monitor/jobs              # Create monitor job
POST /v1/monitor/jobs/:id/stop     # Stop monitor job
```

## Configuration

Environment variables:

```bash
# Server
ENVIRONMENT=development
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# JWT
JWT_SECRET=your-super-secret-key
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=7d

# MCP
MCP_BASE_URL=http://localhost:8000

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=arca
DB_PASSWORD=arca_password
DB_NAME=arca
```

## Development

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Redis
- PostgreSQL

### Running Locally

```bash
# Clone repository
git clone https://github.com/arcaintelligence/arca-gateway.git
cd arca-gateway

# Install dependencies
go mod download

# Run server
go run cmd/server/main.go
```

### Running with Docker

```bash
# Build and run
docker-compose up -d

# View logs
docker-compose logs -f gateway

# Stop
docker-compose down
```

### Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...
```

## Security

- All endpoints require JWT authentication (except `/health`, `/v1/auth/login`, `/v1/auth/register`)
- Tokens expire after 15 minutes (access) and 7 days (refresh)
- Rate limiting is enforced per tenant
- All requests are logged for audit
- Sensitive data is never logged

## License

Proprietary - ARCA Intelligence
