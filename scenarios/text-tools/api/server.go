package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Server represents the Text Tools API server
type Server struct {
	config          *Config
	db              *DatabaseConnection
	resourceManager *ResourceManager
	router          *mux.Router
	server          *http.Server
}

// NewServer creates a new server instance
func NewServer(config *Config) *Server {
	return &Server{
		config: config,
	}
}

// Initialize sets up the server components
func (s *Server) Initialize() error {
	// Initialize database connection with retry logic
	dbConfig := NewDatabaseConfig()
	dbConn, err := NewDatabaseConnection(dbConfig)
	if err != nil {
		log.Printf("Warning: Database initialization failed: %v", err)
		log.Println("Running in degraded mode without database persistence")
	} else {
		s.db = dbConn
		log.Println("Database connection initialized")
	}

	// Initialize resource manager
	s.resourceManager = NewResourceManager(s.config)
	s.resourceManager.Start()
	log.Println("Resource manager initialized")

	// Setup router
	s.router = s.setupRouter()

	// Apply middleware
	handler := applyMiddleware(s.router)

	// Configure HTTP server
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return nil
}

// setupRouter configures all routes
func (s *Server) setupRouter() *mux.Router {
	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/health", s.HealthHandler).Methods("GET")
	
	// Resource status endpoint
	router.HandleFunc("/resources", s.ResourcesHandler).Methods("GET")

	// API v1 endpoints
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/text/diff", s.DiffHandlerV1).Methods("POST")
	v1.HandleFunc("/text/search", s.SearchHandlerV1).Methods("POST")
	v1.HandleFunc("/text/transform", s.TransformHandlerV1).Methods("POST")
	v1.HandleFunc("/text/extract", s.ExtractHandlerV1).Methods("POST")
	v1.HandleFunc("/text/analyze", s.AnalyzeHandlerV1).Methods("POST")

	// API v2 endpoints (prepared for future migration)
	v2 := router.PathPrefix("/api/v2").Subrouter()
	v2.HandleFunc("/text/diff", s.DiffHandlerV2).Methods("POST")
	v2.HandleFunc("/text/search", s.SearchHandlerV2).Methods("POST")
	v2.HandleFunc("/text/transform", s.TransformHandlerV2).Methods("POST")
	v2.HandleFunc("/text/extract", s.ExtractHandlerV2).Methods("POST")
	v2.HandleFunc("/text/analyze", s.AnalyzeHandlerV2).Methods("POST")
	v2.HandleFunc("/text/pipeline", s.PipelineHandler).Methods("POST")

	// Documentation endpoint
	router.PathPrefix("/docs").Handler(http.StripPrefix("/docs", http.FileServer(http.Dir("./docs"))))

	return router
}

// Start begins serving HTTP requests
func (s *Server) Start() error {
	log.Printf("Text Tools API listening on port %s", s.config.Port)
	log.Printf("Health endpoint: http://localhost:%s/health", s.config.Port)
	log.Printf("API v1 endpoint: http://localhost:%s/api/v1/text", s.config.Port)
	log.Printf("API v2 endpoint: http://localhost:%s/api/v2/text", s.config.Port)

	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down Text Tools API...")

	// Stop resource manager
	if s.resourceManager != nil {
		s.resourceManager.Stop()
	}

	// Close database connection if exists
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}

	// Shutdown HTTP server
	return s.server.Shutdown(ctx)
}