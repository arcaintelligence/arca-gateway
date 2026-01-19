# =============================================================================
# ARCA Gateway - Multi-stage Dockerfile
# =============================================================================

# Stage 1: Build
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o /app/arca-gateway \
    ./cmd/server

# Stage 2: Runtime
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 arca && \
    adduser -u 1000 -G arca -s /bin/sh -D arca

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/arca-gateway /app/arca-gateway

# Copy config files (if any)
# COPY --from=builder /app/config /app/config

# Change ownership
RUN chown -R arca:arca /app

# Switch to non-root user
USER arca

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run binary
ENTRYPOINT ["/app/arca-gateway"]
