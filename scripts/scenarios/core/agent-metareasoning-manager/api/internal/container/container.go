package container

import (
	"database/sql"
	"log"
	"os"
	"time"

	"metareasoning-api/internal/domain/ai"
	"metareasoning-api/internal/domain/common"
	"metareasoning-api/internal/domain/execution"
	"metareasoning-api/internal/domain/integration"
	"metareasoning-api/internal/domain/workflow"
	"metareasoning-api/internal/infrastructure/cache"
	"metareasoning-api/internal/infrastructure/database"
	"metareasoning-api/internal/infrastructure/http"
	"metareasoning-api/internal/pkg/interfaces"
	"metareasoning-api/internal/pkg/validation"

	_ "github.com/lib/pq"
)

// Container holds all application dependencies
type Container struct {
	DB              *sql.DB
	Config          *Config
	Cache           interfaces.CacheManager
	CircuitBreakers interfaces.CircuitBreakerManager
	HTTPClients     interfaces.HTTPClientFactory
	
	// Repositories
	WorkflowRepo    workflow.Repository
	ExecutionRepo   workflow.ExecutionRepository
	
	// Services
	WorkflowService     workflow.Service
	ExecutionEngine     execution.ExecutionEngine
	AIGenerationService ai.GenerationService
	IntegrationService  integration.Service
	RequestBinder       validation.RequestBinder
}

// Config holds application configuration
type Config struct {
	Port              string
	DatabaseURL       string
	N8nBase          string
	WindmillBase     string
	WindmillWorkspace string
	OllamaBase       string
}

// NewContainer creates and initializes a new dependency container
func NewContainer() (*Container, error) {
	config := loadConfig()
	
	// Initialize database
	db, err := initDatabase(config.DatabaseURL)
	if err != nil {
		return nil, err
	}
	
	// Initialize cache
	cacheManager := cache.NewMemoryCache()
	
	// Initialize circuit breakers
	circuitBreakers := http.NewCircuitBreakerManager()
	
	// Initialize HTTP clients
	httpClients := http.NewHTTPClientFactory()
	
	container := &Container{
		DB:              db,
		Config:          config,
		Cache:           cacheManager,
		CircuitBreakers: circuitBreakers,
		HTTPClients:     httpClients,
	}
	
	// Initialize repositories
	container.WorkflowRepo = database.NewPostgresWorkflowRepository(db)
	container.ExecutionRepo = database.NewPostgresExecutionRepository(db)
	
	// Initialize execution engine with strategies
	container.ExecutionEngine = execution.NewDefaultExecutionEngine()
	
	// Register execution strategies
	n8nStrategy := execution.NewN8nStrategy(config.N8nBase, httpClients, circuitBreakers)
	container.ExecutionEngine.RegisterStrategy(common.PlatformN8n, n8nStrategy)
	
	windmillStrategy := execution.NewWindmillStrategy(config.WindmillBase, config.WindmillWorkspace, httpClients, circuitBreakers)
	container.ExecutionEngine.RegisterStrategy(common.PlatformWindmill, windmillStrategy)
	
	// Initialize services
	container.WorkflowService = workflow.NewDefaultService(container.WorkflowRepo, container.ExecutionRepo, container.ExecutionEngine)
	container.AIGenerationService = ai.NewOllamaGenerationService(config.OllamaBase, httpClients, circuitBreakers)
	integrationConfig := &integration.Config{
		N8nBase:           config.N8nBase,
		WindmillBase:      config.WindmillBase,
		WindmillWorkspace: config.WindmillWorkspace,
	}
	container.IntegrationService = integration.NewDefaultIntegrationService(container.WorkflowRepo, container.ExecutionEngine, integrationConfig)
	
	// Initialize validation
	validator := validation.NewDefaultValidator()
	container.RequestBinder = validation.NewDefaultRequestBinder(validator)
	
	// Apply performance optimizations
	optimizeDatabaseConnections(db)
	
	log.Println("Dependency container initialized successfully")
	
	return container, nil
}

// Close gracefully shuts down all resources
func (c *Container) Close() error {
	log.Println("Shutting down dependency container...")
	
	if c.DB != nil {
		if err := c.DB.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
			return err
		}
	}
	
	if c.Cache != nil {
		c.Cache.Close()
	}
	
	log.Println("Dependency container shut down complete")
	return nil
}

// loadConfig loads configuration from environment variables
func loadConfig() *Config {
	return &Config{
		Port:              getEnv("PORT", "8093"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://vrooli:vrooli@localhost:5432/metareasoning?sslmode=disable"),
		N8nBase:          getEnv("N8N_BASE_URL", "http://localhost:5678"),
		WindmillBase:     getEnv("WINDMILL_BASE_URL", "http://localhost:8000"),
		WindmillWorkspace: getEnv("WINDMILL_WORKSPACE", "demo"),
		OllamaBase:       getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
	}
}

// initDatabase initializes the database connection
func initDatabase(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	
	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}
	
	return db, nil
}

// optimizeDatabaseConnections sets optimal connection pool settings
func optimizeDatabaseConnections(db *sql.DB) {
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(30 * time.Second)
	
	log.Printf("Database connection pool optimized: MaxOpen=50, MaxIdle=10, MaxLifetime=10m")
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}