package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// Server encapsulates all server dependencies and state
type Server struct {
	db                 *sql.DB
	dbService          *DatabaseService
	landingPageService *LandingPageService
	claudeService      *ClaudeCodeService
	router             *mux.Router
}

// NewServer creates a new server instance with all dependencies initialized
func NewServer() (*Server, error) {
	s := &Server{}

	// Initialize database
	if err := s.initDatabase(); err != nil {
		return nil, fmt.Errorf("database initialization failed: %w", err)
	}

	// Initialize services
	s.dbService = NewDatabaseService(s.db)
	s.landingPageService = NewLandingPageService(s.dbService, DefaultTemplatesPath)
	s.claudeService = NewClaudeCodeService("", s.dbService)

	// Setup router
	s.setupRouter()

	return s, nil
}

// Close gracefully shuts down the server and releases resources
func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Database initialization
func (s *Server) initDatabase() error {
	postgresURL := buildPostgresURL()
	if postgresURL == "" {
		return fmt.Errorf("❌ Missing database configuration. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
	}

	var err error
	s.db, err = sql.Open("postgres", postgresURL)
	if err != nil {
		return fmt.Errorf("Failed to open database connection: %v", err)
	}

	// Set connection pool settings
	s.db.SetMaxOpenConns(DBMaxOpenConns)
	s.db.SetMaxIdleConns(DBMaxIdleConns)
	s.db.SetConnMaxLifetime(DBConnMaxLifetime)

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📆 Database URL configured")

	// Use the shared retry logic
	if err := pingDatabaseWithRetry(s.db); err != nil {
		return err
	}

	log.Println("🎉 Database connection pool established successfully!")
	return nil
}

// setupRouter configures all HTTP routes and middleware
func (s *Server) setupRouter() {
	s.router = mux.NewRouter()

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", s.healthHandler).Methods("GET")
	api.HandleFunc("/scenarios/scan", s.scanScenariosHandler).Methods("POST")
	api.HandleFunc("/landing-pages/generate", s.generateLandingPageHandler).Methods("POST")
	api.HandleFunc("/landing-pages/{id}/deploy", s.deployLandingPageHandler).Methods("POST")
	api.HandleFunc("/templates", s.getTemplatesHandler).Methods("GET")
	api.HandleFunc("/analytics/dashboard", s.getDashboardHandler).Methods("GET")

	// Health check endpoint (also at root level)
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")
}

// Start starts the HTTP server on the specified port
func (s *Server) Start(port string) error {
	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(s.router)

	log.Printf("SaaS Landing Manager API starting on port %s", port)
	return http.ListenAndServe(":"+port, handler)
}

// Helper functions for database connection
func buildPostgresURL() string {
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL != "" {
		return postgresURL
	}

	// Build from individual components - REQUIRED, no defaults
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		return "" // Return empty string to signal missing configuration
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)
}

func pingDatabaseWithRetry(db *sql.DB) error {
	maxRetries := DBMaxRetries
	baseDelay := DBBaseRetryDelay
	maxDelay := DBMaxRetryDelay

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			return nil
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * DBJitterRatio
		jitter := time.Duration(rand.Float64() * jitterRange)
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%DBRetryLogEveryN == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		if attempt < maxRetries-1 {
			time.Sleep(actualDelay)
		}
	}

	return fmt.Errorf("❌ Database connection failed after %d attempts: %w", maxRetries, pingErr)
}

func runLifecycleCommand(command string, args []string) error {
	// For lifecycle commands, we need a temporary database connection
	tempDB, err := sql.Open("postgres", buildPostgresURL())
	if err != nil {
		return fmt.Errorf("Failed to open database connection: %w", err)
	}
	defer tempDB.Close()

	// Implement retry logic for connection
	if err := pingDatabaseWithRetry(tempDB); err != nil {
		return err
	}

	dbService := NewDatabaseService(tempDB)

	switch command {
	case "scan-scenarios":
		forceRescan := false
		scenarioFilter := ""
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--force", "-f":
				forceRescan = true
			case "--filter":
				if i+1 < len(args) {
					scenarioFilter = args[i+1]
					i++
				}
			}
		}

		scenariosPath := os.Getenv("SCENARIOS_PATH")
		if scenariosPath == "" {
			scenariosPath = DefaultScenariosPath
		}

		detectionService := NewSaaSDetectionService(scenariosPath, dbService)
		response, err := detectionService.ScanScenarios(forceRescan, scenarioFilter)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		output, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(output))
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start saas-landing-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		if err := runLifecycleCommand(os.Args[1], os.Args[2:]); err != nil {
			log.Fatalf("%v", err)
		}
		return
	}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	// Create server instance
	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	// Start server
	if err := server.Start(port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
