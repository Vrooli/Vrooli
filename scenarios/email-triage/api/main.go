package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	_ "github.com/lib/pq"
	
	"email-triage/handlers"
	"email-triage/models"
	"email-triage/services"
)

type Server struct {
	db           *sql.DB
	authService  *services.AuthService
	emailService *services.EmailService
	ruleService  *services.RuleService
	searchService *services.SearchService
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start email-triage

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Load configuration from environment - REQUIRED, no defaults
	config := loadConfig()
	
	// Database configuration - support both DATABASE_URL and individual components
	databaseURL := config.DatabaseURL
	if databaseURL == "" {
		// Try to build from individual components
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Missing database configuration. Provide DATABASE_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		databaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}
	
	// Initialize database connection
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()
	
	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	
	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	
	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📆 Database URL configured")
	
	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		// Provide detailed status every few attempts
		if attempt > 0 && attempt % 3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt * 2) * baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}
		
		time.Sleep(actualDelay)
	}
	
	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
	
	// Initialize services
	authService := services.NewAuthService(config.AuthServiceURL)
	emailService := services.NewEmailService(config.MailServerURL)
	ruleService := services.NewRuleService(db, config.OllamaURL)
	searchService := services.NewSearchService(config.QdrantURL)
	
	// Create server instance
	server := &Server{
		db:            db,
		authService:   authService,
		emailService:  emailService,
		ruleService:   ruleService,
		searchService: searchService,
	}
	
	// Setup HTTP router
	router := server.setupRoutes()
	
	// Configure CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3201", "http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		AllowCredentials: true,
	})
	
	handler := c.Handler(router)
	
	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
	
	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	// Start server in goroutine
	go func() {
		log.Printf("Email Triage API server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()
	
	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down server...")
	
	// Gracefully shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	
	log.Println("Server exited")
}

func (s *Server) setupRoutes() *mux.Router {
	router := mux.NewRouter()
	
	// Health check endpoints
	router.HandleFunc("/health", s.healthCheck).Methods("GET")
	router.HandleFunc("/health/database", s.healthCheckDatabase).Methods("GET")
	router.HandleFunc("/health/qdrant", s.healthCheckQdrant).Methods("GET")
	
	// API routes with authentication middleware
	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(s.authMiddleware)
	
	// Email account management
	accountHandler := handlers.NewAccountHandler(s.db, s.emailService)
	api.HandleFunc("/accounts", accountHandler.CreateAccount).Methods("POST")
	api.HandleFunc("/accounts", accountHandler.ListAccounts).Methods("GET")
	api.HandleFunc("/accounts/{id}", accountHandler.GetAccount).Methods("GET")
	api.HandleFunc("/accounts/{id}", accountHandler.UpdateAccount).Methods("PUT")
	api.HandleFunc("/accounts/{id}", accountHandler.DeleteAccount).Methods("DELETE")
	api.HandleFunc("/accounts/{id}/sync", accountHandler.SyncAccount).Methods("POST")
	
	// Triage rule management
	ruleHandler := handlers.NewRuleHandler(s.ruleService)
	api.HandleFunc("/rules", ruleHandler.CreateRule).Methods("POST")
	api.HandleFunc("/rules", ruleHandler.ListRules).Methods("GET")
	api.HandleFunc("/rules/{id}", ruleHandler.GetRule).Methods("GET")
	api.HandleFunc("/rules/{id}", ruleHandler.UpdateRule).Methods("PUT")
	api.HandleFunc("/rules/{id}", ruleHandler.DeleteRule).Methods("DELETE")
	api.HandleFunc("/rules/{id}/test", ruleHandler.TestRule).Methods("POST")
	
	// Email processing and search
	emailHandler := handlers.NewEmailHandler(s.db, s.searchService)
	api.HandleFunc("/emails/search", emailHandler.SearchEmails).Methods("GET")
	api.HandleFunc("/emails/{id}", emailHandler.GetEmail).Methods("GET")
	api.HandleFunc("/emails/{id}/actions", emailHandler.ApplyActions).Methods("POST")
	api.HandleFunc("/emails/sync", emailHandler.ForceSync).Methods("POST")
	
	// Analytics and insights
	analyticsHandler := handlers.NewAnalyticsHandler(s.db)
	api.HandleFunc("/analytics/dashboard", analyticsHandler.GetDashboard).Methods("GET")
	api.HandleFunc("/analytics/usage", analyticsHandler.GetUsageStats).Methods("GET")
	api.HandleFunc("/analytics/rules-performance", analyticsHandler.GetRulePerformance).Methods("GET")
	
	return router
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}
		
		// Validate token with scenario-authenticator service
		userID, err := s.authService.ValidateToken(authHeader)
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}
		
		// Add user ID to request context
		ctx := context.WithValue(r.Context(), "user_id", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	health := models.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services: map[string]string{
			"database": "connected",
			"auth":     "available",
			"qdrant":   "connected",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (s *Server) healthCheckDatabase(w http.ResponseWriter, r *http.Request) {
	err := s.db.Ping()
	status := map[string]interface{}{
		"connected": err == nil,
		"timestamp": time.Now(),
	}
	
	if err != nil {
		status["error"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) healthCheckQdrant(w http.ResponseWriter, r *http.Request) {
	healthy := s.searchService.HealthCheck()
	status := map[string]interface{}{
		"connected": healthy,
		"timestamp": time.Now(),
	}
	
	if !healthy {
		status["error"] = "qdrant connection failed"
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

type Config struct {
	DatabaseURL      string
	QdrantURL        string
	AuthServiceURL   string
	MailServerURL    string
	NotificationURL  string
	OllamaURL        string
}

func loadConfig() Config {
	// All configuration is REQUIRED - no defaults
	databaseURL := os.Getenv("DATABASE_URL")
	// DatabaseURL can be empty - will try individual components if so
	
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		log.Fatal("❌ QDRANT_URL environment variable is required")
	}
	
	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		log.Fatal("❌ AUTH_SERVICE_URL environment variable is required")
	}
	
	mailServerURL := os.Getenv("MAIL_SERVER_URL")
	if mailServerURL == "" {
		log.Fatal("❌ MAIL_SERVER_URL environment variable is required")
	}
	
	notificationURL := os.Getenv("NOTIFICATION_HUB_URL")
	if notificationURL == "" {
		log.Fatal("❌ NOTIFICATION_HUB_URL environment variable is required")
	}
	
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		log.Fatal("❌ OLLAMA_URL environment variable is required")
	}
	
	return Config{
		DatabaseURL:     databaseURL,
		QdrantURL:       qdrantURL,
		AuthServiceURL:  authServiceURL,
		MailServerURL:   mailServerURL,
		NotificationURL: notificationURL,
		OllamaURL:       ollamaURL,
	}
}

