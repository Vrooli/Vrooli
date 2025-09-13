package main

import (
	"net/http"
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
}

// NewServer creates a new server instance
func NewServer(config *Config, db *Database, logger *Logger) *Server {
	// Create connection manager for WebSocket
	connectionManager := NewConnectionManager(logger)
	go connectionManager.Start()

	server := &Server{
		config:            config,
		db:                db,
		logger:            logger,
		connectionManager: connectionManager,
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

	// Widget endpoints
	api.HandleFunc("/chatbots/{id}/widget", s.WidgetHandler).Methods("GET")
	api.HandleFunc("/widget.js", s.ServeWidgetJS).Methods("GET") // Serve widget JavaScript file
}

// WidgetHandler returns the widget embed code for a chatbot
func (s *Server) WidgetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatbotID := vars["id"]

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