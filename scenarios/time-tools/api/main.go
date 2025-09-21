package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

const (
	apiVersion  = "1.0.0"
	serviceName = "time-tools"
)

var logger *log.Logger

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("%s %s %s", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start time-tools

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Initialize logger
	logger = log.New(os.Stdout, "[time-tools] ", log.LstdFlags)
	
	// Initialize database (optional for time-tools)
	if err := initDB(logger); err != nil {
		logger.Printf("Warning: Database initialization failed: %v", err)
	}
	if hasDatabase() {
		defer db.Close()
	}
	
	// Setup routes
	router := mux.NewRouter()
	router.Use(corsMiddleware)
	router.Use(loggingMiddleware)
	
	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	
	// Health check
	api.HandleFunc("/health", healthHandler).Methods("GET")
	
	// Time operations
	api.HandleFunc("/time/convert", timezoneConvertHandler).Methods("POST")
	api.HandleFunc("/time/duration", durationCalculateHandler).Methods("POST")
	api.HandleFunc("/time/format", formatTimeHandler).Methods("POST")
	
	// Scheduling operations  
	api.HandleFunc("/schedule/optimal", scheduleOptimalHandler).Methods("POST")
	api.HandleFunc("/schedule/conflicts", conflictDetectHandler).Methods("POST")
	
	// Event management
	api.HandleFunc("/events", createEventHandler).Methods("POST")
	api.HandleFunc("/events", listEventsHandler).Methods("GET")
	
	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		logger.Fatal("PORT environment variable not set")
	}
	
	addr := ":" + port
	logger.Printf("Starting %s API server on %s", serviceName, addr)
	logger.Printf("API Version: %s", apiVersion)
	
	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Fatal(err)
	}
}