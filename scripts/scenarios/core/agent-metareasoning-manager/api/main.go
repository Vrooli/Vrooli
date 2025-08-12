// Metareasoning API - Modular implementation
// Clean architecture with separated concerns

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

var (
	config *Config
)

func main() {
	log.Println("Starting Metareasoning API v3.0.0...")
	
	// Load configuration
	config = LoadConfig()
	log.Printf("Configuration loaded - Port: %s", config.Port)
	
	// Initialize database
	if err := InitDatabase(config.DatabaseURL); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer CloseDatabase()
	log.Println("Database connected")
	
	// Initialize services
	workflowService := NewWorkflowService(config)
	
	// Initialize handlers
	handler := NewHandler(workflowService)
	
	// Setup routes
	router := setupRoutes(handler)
	
	// Apply performance optimizations
	ApplyPerformanceOptimizations()
	
	// Apply middleware (order matters for security and performance)
	router.Use(recoveryMiddleware)           // First: catch panics
	router.Use(authMiddleware)               // Second: reject unauthorized requests early 
	router.Use(corsMiddleware)               // Third: handle CORS for authorized requests
	router.Use(loggingMiddleware)            // Fourth: log authorized requests
	router.Use(PerformanceMiddleware)        // Fifth: add performance timing
	router.Use(CacheMiddleware(300))         // Sixth: cache GET requests for 5 minutes
	router.Use(CompressionMiddleware)        // Last: compress responses
	
	// Start server
	server := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		
		log.Println("Shutting down server...")
		
		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		// Graceful shutdown allowing ongoing requests to complete
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Error during graceful shutdown: %v", err)
			// Force close if graceful shutdown fails
			if closeErr := server.Close(); closeErr != nil {
				log.Printf("Error during force close: %v", closeErr)
			}
		} else {
			log.Println("Server shutdown gracefully")
		}
	}()
	
	// Start listening
	log.Printf("Server listening on port %s", config.Port)
	log.Printf("API documentation available at http://localhost:%s/docs", config.Port)
	
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRoutes(h *Handler) *mux.Router {
	router := mux.NewRouter()
	
	// Health & System endpoints
	router.HandleFunc("/health", OptimizedHealthHandler).Methods("GET")         // Fast health check
	router.HandleFunc("/health/full", h.HealthHandler).Methods("GET")          // Full health check
	router.HandleFunc("/platforms", h.GetPlatformsHandler).Methods("GET")
	router.HandleFunc("/models", h.GetModelsHandler).Methods("GET")
	router.HandleFunc("/stats", h.GetStatsHandler).Methods("GET")
	router.HandleFunc("/circuit-breakers", h.GetCircuitBreakerStatsHandler).Methods("GET")
	
	// Workflow CRUD endpoints
	router.HandleFunc("/workflows", h.ListWorkflowsHandler).Methods("GET")
	router.HandleFunc("/workflows", h.CreateWorkflowHandler).Methods("POST")
	router.HandleFunc("/workflows/{id}", h.GetWorkflowHandler).Methods("GET")
	router.HandleFunc("/workflows/{id}", h.UpdateWorkflowHandler).Methods("PUT")
	router.HandleFunc("/workflows/{id}", h.DeleteWorkflowHandler).Methods("DELETE")
	
	// Workflow execution endpoints
	router.HandleFunc("/workflows/{id}/execute", h.ExecuteWorkflowHandler).Methods("POST")
	router.HandleFunc("/analyze/{type}", h.AnalyzeHandler).Methods("POST")
	
	// Advanced workflow endpoints
	router.HandleFunc("/workflows/search", h.SearchWorkflowsHandler).Methods("GET")
	router.HandleFunc("/workflows/generate", h.GenerateWorkflowHandler).Methods("POST")
	router.HandleFunc("/workflows/import", h.ImportWorkflowHandler).Methods("POST")
	router.HandleFunc("/workflows/{id}/export", h.ExportWorkflowHandler).Methods("GET")
	router.HandleFunc("/workflows/{id}/clone", h.CloneWorkflowHandler).Methods("POST")
	
	// Metrics & History endpoints
	router.HandleFunc("/workflows/{id}/history", h.GetHistoryHandler).Methods("GET")
	router.HandleFunc("/workflows/{id}/metrics", h.GetMetricsHandler).Methods("GET")
	
	return router
}