package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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
	// Load configuration from environment
	config := loadConfig()
	
	// Initialize database connection
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	
	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}
	log.Println("Connected to PostgreSQL database")
	
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
	
	// Create HTTP server
	srv := &http.Server{
		Addr:         ":3200",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	// Start server in goroutine
	go func() {
		log.Printf("Email Triage API server starting on port 3200")
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
	return Config{
		DatabaseURL:     getEnvOrDefault("DATABASE_URL", "postgres://postgres:password@localhost/email_triage?sslmode=disable"),
		QdrantURL:       getEnvOrDefault("QDRANT_URL", "http://localhost:6333"),
		AuthServiceURL:  getEnvOrDefault("AUTH_SERVICE_URL", "http://localhost:8080"),
		MailServerURL:   getEnvOrDefault("MAIL_SERVER_URL", "localhost"),
		NotificationURL: getEnvOrDefault("NOTIFICATION_HUB_URL", "http://localhost:8081"),
		OllamaURL:       getEnvOrDefault("OLLAMA_URL", "http://localhost:11434"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}