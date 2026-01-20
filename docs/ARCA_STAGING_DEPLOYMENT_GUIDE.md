# ARCA Intelligence Platform - Guia de Deploy para Staging

## Visão Geral

A plataforma ARCA é composta por dois componentes principais:

1. **Gateway (Go/Fiber)** - API Gateway de alta performance
2. **Core Python (FastAPI)** - Motor de IA com MCP (Model Context Protocol)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           ARCA Architecture                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌──────────┐     ┌─────────────────┐     ┌─────────────────────────────┐ │
│   │  Client  │────▶│  Gateway (Go)   │────▶│    Core Python (FastAPI)   │ │
│   │   App    │     │  Port: 8080     │     │    Port: 8001              │ │
│   └──────────┘     │                 │     │                             │ │
│                    │  • JWT Auth     │     │  • MCP Server               │ │
│                    │  • Rate Limit   │     │  • AI Agents                │ │
│                    │  • Validation   │     │  • Threat Detection         │ │
│                    │  • MCP Format   │     │  • Knowledge Base           │ │
│                    └─────────────────┘     └─────────────────────────────┘ │
│                                                      │                      │
│                                                      ▼                      │
│                    ┌─────────────────────────────────────────────────────┐ │
│                    │              External Integrations                   │ │
│                    │  • Google Play Scraper  • Apple Store Scraper       │ │
│                    │  • IntelX API           • DNS Resolution            │ │
│                    └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 1. Repositórios GitHub

| Componente | Repositório | Branch |
|------------|-------------|--------|
| Gateway | `arcaintelligence/arca-gateway` | master |
| Core Python | `arcaintelligence/arca-core-python` | main |

---

## 2. Variáveis de Ambiente

### 2.1 Gateway (Go/Fiber)

```bash
# Servidor
PORT=8080
ENV=staging

# JWT (DEVE ser igual ao Core Python)
JWT_SECRET=your-super-secret-key-change-in-production

# Core Python URL
MCP_CORE_URL=http://localhost:8001

# Rate Limiting
RATE_LIMIT_MAX=100
RATE_LIMIT_WINDOW=60
```

### 2.2 Core Python (FastAPI)

```bash
# Servidor
PORT=8001
ENV=staging

# JWT (DEVE ser igual ao Gateway)
JWT_SECRET=your-super-secret-key-change-in-production

# IntelX API (para detecção de vazamentos)
INTELX_API_KEY=8ecd4da1-85b8-45ea-be04-fcd966309bb7

# OpenAI (para agentes AI)
OPENAI_API_KEY=sk-xxx

# Database (opcional - atualmente usa in-memory)
DATABASE_URL=postgresql://user:pass@host:5432/arca

# Redis (opcional - para cache)
REDIS_URL=redis://localhost:6379
```

---

## 3. Instalação e Execução

### 3.1 Core Python

```bash
# Clonar repositório
gh repo clone arcaintelligence/arca-core-python
cd arca-core-python

# Criar ambiente virtual
python3 -m venv venv
source venv/bin/activate

# Instalar dependências
pip install -r requirements.txt

# Configurar variáveis de ambiente
export JWT_SECRET="your-super-secret-key-change-in-production"
export INTELX_API_KEY="8ecd4da1-85b8-45ea-be04-fcd966309bb7"

# Executar
cd src/arca/api
python app.py
```

### 3.2 Gateway (Go/Fiber)

```bash
# Clonar repositório
gh repo clone arcaintelligence/arca-gateway
cd arca-gateway

# Compilar
go build -o bin/arca-gateway cmd/server/main.go

# Configurar variáveis de ambiente
export JWT_SECRET="your-super-secret-key-change-in-production"
export MCP_CORE_URL="http://localhost:8001"

# Executar
./bin/arca-gateway
```

---

## 4. Endpoints da API

### 4.1 Onboarding (Público)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/v1/onboarding/register` | Registrar nova empresa |
| POST | `/v1/onboarding/verify-email` | Verificar código de email |

### 4.2 Brands (Autenticado)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/v1/brands` | Criar nova marca |
| GET | `/v1/brands` | Listar marcas |
| GET | `/v1/brands/:id` | Detalhes da marca |

### 4.3 Monitoring (Autenticado)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/v1/brands/:id/monitoring/start` | Iniciar monitoramento |
| GET | `/v1/brands/:id/monitoring/status` | Status do monitoramento |
| GET | `/v1/brands/:id/threats` | Listar ameaças detectadas |

### 4.4 MCP Tools (Interno)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/v1/mcp/call` | Executar ferramenta MCP |

---

## 5. Fluxo de Autenticação

### 5.1 Registro e Verificação

```bash
# 1. Registrar empresa
curl -X POST http://localhost:8080/v1/onboarding/register \
  -H "Content-Type: application/json" \
  -d '{
    "company_name": "Minha Empresa",
    "email": "admin@minhaempresa.com",
    "plan": "enterprise"
  }'

# Resposta: { "client_id": "cli_xxx", "_dev_code": "123456" }

# 2. Verificar email
curl -X POST http://localhost:8080/v1/onboarding/verify-email \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "cli_xxx",
    "code": "123456"
  }'

# Resposta: { "token": "eyJhbGci..." }
```

### 5.2 Usar Token JWT

```bash
# Headers obrigatórios para endpoints autenticados
Authorization: Bearer <token>
X-Client-ID: cli_xxx
```

---

## 6. Formato MCP

Todas as requisições internas seguem o formato MCP:

```json
{
  "request_id": "uuid",
  "tenant_id": "cli_xxx",
  "user_id": "user_xxx",
  "tool": "onboarding",
  "action": "register",
  "params": {
    "company_name": "...",
    "email": "..."
  },
  "scopes": ["client:read", "client:write"]
}
```

---

## 7. Hierarquia de IDs

```
Client (cli_xxx)           # Tenant/Empresa
  └── Brand (brand_xxx)    # Marca da empresa
        └── Job (job_xxx)  # Job de monitoramento
              └── Threat (threat_xxx)  # Ameaça detectada
```

---

## 8. Capacidades de Detecção

### 8.1 Typosquatting de Domínios
- Gera 100+ variações do domínio oficial
- Verifica resolução DNS
- Detecta domínios ativos que podem ser phishing

### 8.2 Apps Falsos
- **Google Play**: Busca por keywords e nome da marca
- **Apple Store**: Busca via iTunes Search API
- Filtra apps oficiais (por package name ou developer)
- Alerta apenas apps suspeitos

### 8.3 Vazamento de Credenciais
- Integração com IntelX API
- Busca por domínio e keywords
- Detecta exposição de credenciais em data breaches

### 8.4 Knowledge Base
- 208 entradas de indicadores de ameaça
- Técnicas de ataque conhecidas
- Padrões de marca para referência

---

## 9. Testes de Integração

### 9.1 Teste Completo do Fluxo

```bash
#!/bin/bash
# test_full_flow.sh

echo "1. Registrar cliente..."
REG=$(curl -s -X POST http://localhost:8080/v1/onboarding/register \
  -H "Content-Type: application/json" \
  -d '{"company_name":"Test Corp","email":"test@testcorp.com","plan":"enterprise"}')
CLI=$(echo "$REG" | jq -r '.data.client_id')
CODE=$(echo "$REG" | jq -r '.data._dev_code')
echo "   Client: $CLI"

echo "2. Verificar email..."
VER=$(curl -s -X POST http://localhost:8080/v1/onboarding/verify-email \
  -H "Content-Type: application/json" \
  -d "{\"client_id\":\"$CLI\",\"code\":\"$CODE\"}")
TOKEN=$(echo "$VER" | jq -r '.data.token')
echo "   Token obtido"

echo "3. Criar marca..."
BRD=$(curl -s -X POST http://localhost:8080/v1/brands \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Client-ID: $CLI" \
  -d '{"name":"Test Corp","domain":"testcorp.com","keywords":["test","corp"]}')
BRAND=$(echo "$BRD" | jq -r '.data.brand.id')
echo "   Brand: $BRAND"

echo "4. Iniciar monitoramento..."
curl -s -X POST "http://localhost:8080/v1/brands/$BRAND/monitoring/start" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Client-ID: $CLI" \
  -d '{"frequency":"hourly","channels":["web","appstore"]}'

echo "✅ Fluxo completo executado!"
```

---

## 10. Deploy com Docker (Recomendado)

### 10.1 Dockerfile - Core Python

```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY src/ ./src/
WORKDIR /app/src/arca/api

ENV PORT=8001
EXPOSE 8001

CMD ["python", "app.py"]
```

### 10.2 Dockerfile - Gateway

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /arca-gateway cmd/server/main.go

FROM alpine:latest
COPY --from=builder /arca-gateway /arca-gateway

ENV PORT=8080
EXPOSE 8080

CMD ["/arca-gateway"]
```

### 10.3 Docker Compose

```yaml
version: '3.8'

services:
  core-python:
    build:
      context: ./arca-core-python
    environment:
      - JWT_SECRET=${JWT_SECRET}
      - INTELX_API_KEY=${INTELX_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    ports:
      - "8001:8001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8001/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  gateway:
    build:
      context: ./arca-gateway
    environment:
      - JWT_SECRET=${JWT_SECRET}
      - MCP_CORE_URL=http://core-python:8001
    ports:
      - "8080:8080"
    depends_on:
      core-python:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

---

## 11. Checklist de Deploy

- [ ] Variáveis de ambiente configuradas
- [ ] JWT_SECRET igual em Gateway e Core Python
- [ ] IntelX API Key válida
- [ ] Portas 8001 e 8080 disponíveis
- [ ] Teste de health check em ambos serviços
- [ ] Teste de fluxo completo (register → verify → brand → monitoring)
- [ ] Logs configurados para observabilidade
- [ ] Rate limiting ajustado para produção

---

## 12. Troubleshooting

### Erro: "Invalid token"
- Verificar se JWT_SECRET é igual em Gateway e Core Python

### Erro: "Connection refused to Core Python"
- Verificar se Core Python está rodando na porta 8000
- Verificar variável MCP_CORE_URL no Gateway

### Erro: "No threats detected"
- Verificar se IntelX API Key está configurada
- Verificar conectividade com APIs externas (Google Play, Apple Store)

### Logs
```bash
# Core Python
tail -f /tmp/arca_app.log

# Gateway
# Logs vão para stdout por padrão
```

---

## 13. Próximos Passos (Roadmap)

1. **Banco de Dados Persistente** - Migrar de in-memory para PostgreSQL
2. **Redis Cache** - Cache de resultados de scan
3. **Webhooks** - Notificações em tempo real de novas ameaças
4. **Dashboard Web** - Interface visual para operadores
5. **APISIX Gateway** - Substituir Gateway Go por APISIX em produção
6. **Kubernetes** - Deploy em cluster K8s com auto-scaling

---

**Última atualização**: 20 de Janeiro de 2026
**Versão**: 1.0.0
