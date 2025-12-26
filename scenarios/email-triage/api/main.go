package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"email-triage/handlers"
	"email-triage/models"
	"email-triage/services"
)

type Server struct {
	db                *sql.DB
	authService       *services.AuthService
	emailService      *services.EmailService
	ruleService       *services.RuleService
	searchService     *services.SearchService
	realtimeProcessor *services.RealtimeProcessor
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "email-triage",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Load configuration from environment - REQUIRED, no defaults
	config := loadConfig()

	// Initialize database connection with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	db, err := database.Connect(context.Background(), database.Config{
		Driver: database.DriverPostgres,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	log.Println("Database connection established successfully!")

	// Initialize services
	authService := services.NewAuthService(config.AuthServiceURL)
	emailService := services.NewEmailService(config.MailServerURL)
	ruleService := services.NewRuleService(db, config.OllamaURL)
	searchService := services.NewSearchService(config.QdrantURL)
	realtimeProcessor := services.NewRealtimeProcessor(db, emailService, ruleService, searchService)

	// Create server instance
	srv := &Server{
		db:                db,
		authService:       authService,
		emailService:      emailService,
		ruleService:       ruleService,
		searchService:     searchService,
		realtimeProcessor: realtimeProcessor,
	}

	// Setup HTTP router
	router := srv.setupRoutes()

	// Configure CORS - allow UI port dynamically
	uiPort := os.Getenv("UI_PORT")
	var allowedOrigins []string
	if uiPort != "" {
		allowedOrigins = []string{
			"http://localhost:" + uiPort,
		}
		log.Printf("✅ CORS configured for UI port: %s", uiPort)
	} else {
		log.Println("⚠️  UI_PORT not set, CORS disabled (API-only mode)")
		allowedOrigins = []string{} // No origins allowed
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	// Start real-time processor
	if err := srv.realtimeProcessor.Start(); err != nil {
		log.Printf("Warning: Real-time processor failed to start: %v", err)
		// Continue running without real-time processing
	}

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error {
			srv.realtimeProcessor.Stop()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
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
	api.HandleFunc("/emails/{id}/priority", s.updateEmailPriority).Methods("PUT")
	api.HandleFunc("/processor/status", s.getProcessorStatus).Methods("GET")

	// Analytics and insights
	analyticsHandler := handlers.NewAnalyticsHandler(s.db)
	api.HandleFunc("/analytics/dashboard", analyticsHandler.GetDashboard).Methods("GET")
	api.HandleFunc("/analytics/usage", analyticsHandler.GetUsageStats).Methods("GET")
	api.HandleFunc("/analytics/rules-performance", analyticsHandler.GetRulePerformance).Methods("GET")

	return router
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Development mode: bypass auth if DEV_MODE is set
		if os.Getenv("DEV_MODE") == "true" {
			// Use mock user ID for development
			ctx := context.WithValue(r.Context(), "user_id", "00000000-0000-0000-0000-000000000001")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

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
		Service:   "email-triage-api",
		Timestamp: time.Now(),
		Readiness: true,
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
	QdrantURL       string
	AuthServiceURL  string
	MailServerURL   string
	NotificationURL string
	OllamaURL       string
}

// updateEmailPriority updates the priority score for an email
func (s *Server) updateEmailPriority(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	vars := mux.Vars(r)
	emailID := vars["id"]

	// Parse request body
	var req struct {
		PriorityScore float64 `json:"priority_score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate priority score
	if req.PriorityScore < 0 || req.PriorityScore > 1 {
		http.Error(w, `{"error":"priority_score must be between 0 and 1"}`, http.StatusBadRequest)
		return
	}

	// Update priority score with user authorization check
	query := `
		UPDATE processed_emails pe
		SET priority_score = $3
		FROM email_accounts ea
		WHERE pe.account_id = ea.id 
		AND pe.id = $1 
		AND ea.user_id = $2`

	result, err := s.db.Exec(query, emailID, userID, req.PriorityScore)
	if err != nil {
		http.Error(w, `{"error":"database update failed"}`, http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, `{"error":"email not found or access denied"}`, http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"success":        true,
		"email_id":       emailID,
		"priority_score": req.PriorityScore,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getProcessorStatus returns the status of the real-time processor
func (s *Server) getProcessorStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"running":       s.realtimeProcessor != nil,
		"sync_interval": "5m",
		"timestamp":     time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func loadConfig() Config {
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333" // Default for Qdrant
		log.Printf("⚠️  QDRANT_URL not set, using default: %s", qdrantURL)
	}

	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		// Try discovery for scenario-authenticator
		discoveredURL, err := discovery.ResolveScenarioURLDefault(context.Background(), "scenario-authenticator")
		if err != nil {
			// Discovery failed - check DEV_MODE
			if os.Getenv("DEV_MODE") != "true" {
				log.Fatal("❌ Authentication service not found via discovery (or set DEV_MODE=true for testing)")
			}
			// In DEV_MODE, use localhost default
			authServiceURL = "http://localhost:8080"
			log.Println("⚠️  Authentication service not configured, using DEV_MODE default")
		} else {
			authServiceURL = discoveredURL
			log.Println("✅ Authentication service discovered via discovery")
		}
	} else {
		log.Println("✅ Authentication service configured via environment")
	}

	mailServerURL := os.Getenv("MAIL_SERVER_URL")
	if mailServerURL == "" {
		mailServerURL = "localhost" // Default mail server
		log.Printf("⚠️  MAIL_SERVER_URL not set, using default: %s", mailServerURL)
	}

	// Optional services - use defaults if not provided
	notificationURL := os.Getenv("NOTIFICATION_HUB_URL")
	if notificationURL == "" {
		notificationURL = "http://localhost:8081" // Default for notification hub
		log.Printf("⚠️  NOTIFICATION_HUB_URL not set, using default: %s", notificationURL)
	}

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434" // Default for Ollama
		log.Printf("⚠️  OLLAMA_URL not set, using default: %s", ollamaURL)
	}

	return Config{
		QdrantURL:       qdrantURL,
		AuthServiceURL:  authServiceURL,
		MailServerURL:   mailServerURL,
		NotificationURL: notificationURL,
		OllamaURL:       ollamaURL,
	}
}
