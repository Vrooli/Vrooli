package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// Server represents the main application server
type Server struct {
	config            *Config
	db                *Database
	logger            *Logger
	connectionManager *ConnectionManager
	wsHandler         *WebSocketHandler
	router            *mux.Router
	eventPublisher    *EventPublisher
	baseURL           string
	// Health check cache
	healthCache      map[string]interface{}
	healthCacheTime  time.Time
	healthCacheMutex sync.RWMutex
}

// NewServer creates a new server instance
func NewServer(config *Config, db *Database, logger *Logger) *Server {
	// Create connection manager for WebSocket
	connectionManager := NewConnectionManager(logger)
	go connectionManager.Start()

	// Create event publisher
	eventPublisher := NewEventPublisher(logger)

	server := &Server{
		config:            config,
		db:                db,
		logger:            logger,
		connectionManager: connectionManager,
		eventPublisher:    eventPublisher,
		baseURL:           fmt.Sprintf("http://localhost:%s", config.APIPort),
		healthCache:       make(map[string]interface{}),
	}

	// Create WebSocket handler
	server.wsHandler = NewWebSocketHandler(server, connectionManager, logger)

	// Setup routes
	server.setupRoutes()

	return server
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	s.router = mux.NewRouter()

	// Apply middleware
	s.router.Use(func(next http.Handler) http.Handler {
		return RecoveryMiddleware(s.logger, next)
	})
	s.router.Use(func(next http.Handler) http.Handler {
		return LoggingMiddleware(s.logger, next)
	})
	s.router.Use(CORSMiddleware)

	// Apply authentication middleware globally
	authMiddleware := NewAuthenticationMiddleware(s.db, s.logger)
	s.router.Use(authMiddleware.Middleware)

	// Health check
	s.router.HandleFunc("/health", s.HealthHandler).Methods("GET", "OPTIONS")

	// API v1 routes
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.Use(ContentTypeMiddleware)

	// Apply rate limiting to API routes
	rateLimiter := NewRateLimiter(100, time.Minute)
	api.Use(rateLimiter.Middleware)

	// Chatbot management endpoints
	api.HandleFunc("/chatbots", s.CreateChatbotHandler).Methods("POST")
	api.HandleFunc("/chatbots", s.ListChatbotsHandler).Methods("GET")
	api.HandleFunc("/chatbots/{id}", s.GetChatbotHandler).Methods("GET")
	api.HandleFunc("/chatbots/{id}", s.UpdateChatbotHandler).Methods("PUT", "PATCH")
	api.HandleFunc("/chatbots/{id}", s.DeleteChatbotHandler).Methods("DELETE")

	// Chat functionality
	api.HandleFunc("/chat/{id}", s.ChatHandler).Methods("POST")
	api.HandleFunc("/ws/{id}", s.wsHandler.HandleWebSocket) // WebSocket endpoint

	// Analytics
	api.HandleFunc("/analytics/{id}", s.AnalyticsHandler).Methods("GET")

	// Escalations
	api.HandleFunc("/chatbots/{id}/escalations", s.GetEscalationsHandler).Methods("GET")
	api.HandleFunc("/escalations/{id}", s.UpdateEscalationHandler).Methods("PATCH")

	// Widget endpoints
	api.HandleFunc("/chatbots/{id}/widget", s.WidgetHandler).Methods("GET")
	api.HandleFunc("/widget.js", s.ServeWidgetJS).Methods("GET") // Serve widget JavaScript file

	// Multi-tenant endpoints
	api.HandleFunc("/tenants", s.CreateTenantHandler).Methods("POST")
	api.HandleFunc("/tenants/{id}", s.GetTenantHandler).Methods("GET")

	// A/B Testing endpoints
	api.HandleFunc("/chatbots/{chatbot_id}/ab-tests", s.CreateABTestHandler).Methods("POST")
	api.HandleFunc("/ab-tests/{test_id}/start", s.StartABTestHandler).Methods("POST")
	api.HandleFunc("/ab-tests/{test_id}/results", s.GetABTestResultsHandler).Methods("GET")

	// CRM Integration endpoints
	api.HandleFunc("/crm-integrations", s.CreateCRMIntegrationHandler).Methods("POST")
	api.HandleFunc("/conversations/{conversation_id}/sync-crm", s.SyncCRMLeadHandler).Methods("POST")
}

// WidgetHandler returns the widget embed code for a chatbot
func (s *Server) WidgetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatbotID := vars["id"]

	// Validate UUID format first
	if !isValidUUID(chatbotID) {
		http.Error(w, "Invalid chatbot ID format", http.StatusBadRequest)
		return
	}

	chatbot, err := s.db.GetChatbot(chatbotID)
	if err != nil {
		http.Error(w, "Chatbot not found", http.StatusNotFound)
		return
	}

	embedCode := s.GenerateWidgetEmbedCode(chatbot.ID, chatbot.WidgetConfig)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(embedCode))
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := ":" + s.config.APIPort
	s.logger.Printf("🚀 Server starting on port %s", s.config.APIPort)
	s.logger.Printf("📊 Health check: http://localhost:%s/health", s.config.APIPort)
	s.logger.Printf("🤖 Ollama URL: %s", s.config.OllamaURL)

	server := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	s.logger.Println("Shutting down server...")
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
