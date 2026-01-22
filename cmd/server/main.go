package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"database/sql"
	"github.com/arcaintelligence/arca-gateway/internal/auth"
	"github.com/arcaintelligence/arca-gateway/internal/config"
	"github.com/arcaintelligence/arca-gateway/internal/handlers"
	"github.com/arcaintelligence/arca-gateway/internal/mcp"
	"github.com/arcaintelligence/arca-gateway/internal/middleware"
	"github.com/arcaintelligence/arca-gateway/internal/services"
	"github.com/arcaintelligence/arca-gateway/pkg/response"
	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
)

const (
	version = "1.0.0"
	banner  = `
   _    ____   ____    _    
  / \  |  _ \ / ___|  / \   
 / _ \ | |_) | |     / _ \  
/ ___ \|  _ <| |___ / ___ \ 
/_/   \_\_| \_\\____/_/   \_\
                            
ARCA Gateway v%s
High-Performance API Gateway for ARCA Intelligence Platform
`
)

func main() {
	// Banner
	fmt.Printf(banner, version)

	// Carregar configuração
	cfg := config.Load()
	log.Printf("Environment: %s", cfg.Server.Environment)

	// Conectar ao Banco de Dados
	dbConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name, cfg.Database.SSLMode)
	
	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database successfully")

	// Criar JWT Manager
	jwtManager := auth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
		cfg.JWT.Issuer,
		cfg.JWT.Audience,
	)

	// Criar MCP Client
	mcpClient := mcp.NewMCPClient(mcp.MCPConfig{
		BaseURL:    cfg.MCP.BaseURL,
		Timeout:    cfg.MCP.Timeout,
		MaxRetries: cfg.MCP.MaxRetries,
		RetryDelay: cfg.MCP.RetryDelay,
	})

	// Criar Services
	userService := services.NewUserService(db)
	clientService := services.NewClientService(db)
	brandService := services.NewBrandService(db)

	// Criar Handlers
	authHandler := handlers.NewAuthHandler(jwtManager, userService)
	clientHandler := handlers.NewClientHandler(clientService, brandService)
	huntingHandler := handlers.NewHuntingHandler(mcpClient)
	onboardingHandler := handlers.NewOnboardingHandler(mcpClient)

	// Criar Auth Middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtManager)

	// Criar Fiber App
	app := fiber.New(fiber.Config{
		AppName:               "ARCA Gateway",
		ServerHeader:          "ARCA",
		ReadTimeout:           cfg.Server.ReadTimeout,
		WriteTimeout:          cfg.Server.WriteTimeout,
		IdleTimeout:           cfg.Server.IdleTimeout,
		DisableStartupMessage: false,
		Prefork:               cfg.Server.Prefork,
		ErrorHandler:          errorHandler,
	})

	// Setup Security Middlewares
	middleware.SetupSecurityMiddlewares(app, middleware.SecurityConfig{
		AllowOrigins:     cfg.CORS.AllowOrigins,
		AllowMethods:     cfg.CORS.AllowMethods,
		AllowHeaders:     cfg.CORS.AllowHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
		Environment:      cfg.Server.Environment,
	})

	// Audit Middleware
	app.Use(middleware.AuditMiddleware())

	// ==========================================================================
	// ROUTES
	// ==========================================================================

	// Health Check
	app.Get("/health", func(c *fiber.Ctx) error {
		services := map[string]string{
			"gateway": "healthy",
		}

		// Check MCP
		if err := mcpClient.HealthCheck(c.Context()); err != nil {
			services["mcp"] = "unhealthy"
		} else {
			services["mcp"] = "healthy"
		}

		return response.Health(c, version, services)
	})

	// API v1
	v1 := app.Group("/v1")

	// Auth routes (public)
	authRoutes := v1.Group("/auth")
	authRoutes.Post("/login", authHandler.Login)
	authRoutes.Post("/register", authHandler.Register)
	authRoutes.Post("/refresh", authHandler.RefreshToken)

	// Onboarding routes (public - registro inicial)
	onboardingRoutes := v1.Group("/onboarding")
	onboardingRoutes.Post("/register", onboardingHandler.Register)
	onboardingRoutes.Post("/verify-email", onboardingHandler.VerifyEmail)

	// Brand routes (protected - via onboarding handler que faz proxy para Core Python)
	brandRoutesNew := v1.Group("/brands", authMiddleware.Authenticate())
	brandRoutesNew.Get("/", onboardingHandler.ListBrands)
	brandRoutesNew.Post("/", onboardingHandler.CreateBrand)
	brandRoutesNew.Get("/:brand_id", onboardingHandler.GetBrand)
	brandRoutesNew.Post("/:brand_id/monitoring/start", onboardingHandler.StartMonitoring)
	brandRoutesNew.Post("/:brand_id/monitoring/stop", onboardingHandler.StopMonitoring)
	brandRoutesNew.Get("/:brand_id/monitoring/status", onboardingHandler.GetMonitoringStatus)

	// Threats routes (protected)
	threatsRoutes := v1.Group("/threats", authMiddleware.Authenticate())
	threatsRoutes.Get("/", onboardingHandler.GetThreats)

	// Auth routes (protected)
	authProtected := authRoutes.Group("", authMiddleware.Authenticate())
	authProtected.Post("/logout", authHandler.Logout)
	authProtected.Get("/me", authHandler.Me)
	authProtected.Post("/api-key", authHandler.GenerateAPIKey)

	// Client routes (protected)
	clientRoutes := v1.Group("/clients", authMiddleware.Authenticate())
	clientRoutes.Use(middleware.RequireScope(middleware.ScopeClientsRead))
	clientRoutes.Get("/", clientHandler.ListClients)
	clientRoutes.Get("/:client_id", clientHandler.GetClient)
	clientRoutes.Post("/", middleware.RequireScope(middleware.ScopeClientsWrite), clientHandler.CreateClient)
	clientRoutes.Put("/:client_id", middleware.RequireScope(middleware.ScopeClientsWrite), clientHandler.UpdateClient)
	clientRoutes.Delete("/:client_id", middleware.RequireScope(middleware.ScopeClientsWrite), clientHandler.DeleteClient)

	// Brand routes (nested under clients)
	brandRoutes := clientRoutes.Group("/:client_id/brands")
	brandRoutes.Use(middleware.RequireScope(middleware.ScopeBrandsRead))
	brandRoutes.Get("/", clientHandler.ListBrands)
	brandRoutes.Get("/:brand_id", clientHandler.GetBrand)
	brandRoutes.Post("/", middleware.RequireScope(middleware.ScopeBrandsWrite), clientHandler.CreateBrand)
	brandRoutes.Put("/:brand_id", middleware.RequireScope(middleware.ScopeBrandsWrite), clientHandler.UpdateBrand)
	brandRoutes.Delete("/:brand_id", middleware.RequireScope(middleware.ScopeBrandsWrite), clientHandler.DeleteBrand)
	brandRoutes.Post("/:brand_id/monitoring/start", middleware.RequireScope(middleware.ScopeMonitorWrite), clientHandler.StartMonitoring)
	brandRoutes.Post("/:brand_id/monitoring/stop", middleware.RequireScope(middleware.ScopeMonitorWrite), clientHandler.StopMonitoring)

	// Hunting routes (protected)
	huntingRoutes := v1.Group("/hunting", authMiddleware.Authenticate())
	huntingRoutes.Post("/hunt", huntingHandler.Hunt)
	huntingRoutes.Post("/scan", huntingHandler.ScanURL)
	huntingRoutes.Post("/analyze", huntingHandler.AnalyzeURL)
	huntingRoutes.Post("/leaks/search", huntingHandler.SearchLeaks)

	// Monitor routes (protected)
	monitorRoutes := v1.Group("/monitor", authMiddleware.Authenticate())
	monitorRoutes.Post("/jobs", huntingHandler.CreateMonitorJob)
	monitorRoutes.Post("/jobs/:job_id/stop", huntingHandler.StopMonitorJob)

	// ==========================================================================
	// START SERVER
	// ==========================================================================

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
		log.Printf("Starting server on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

// errorHandler handler de erros global
func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"success":   false,
		"error": fiber.Map{
			"code":    code,
			"message": message,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
