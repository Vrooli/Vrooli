package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	_ "github.com/lib/pq"
)

// Health response type
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Service   string    `json:"service"`
}

var tidinessProcessor *TidinessProcessor

func main() {
	// Initialize database connection (optional)
	var db *sql.DB
	var err error

	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	if dbHost != "" && dbPort != "" && dbUser != "" && dbPassword != "" && dbName != "" {
		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)

		db, err = sql.Open("postgres", connStr)
		if err == nil {
			// Configure connection pool
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(5)
			db.SetConnMaxLifetime(5 * time.Minute)
			
			// Implement exponential backoff for database connection
			maxRetries := 10
			baseDelay := 1 * time.Second
			maxDelay := 30 * time.Second
			
			log.Println("🔄 Attempting database connection with exponential backoff...")
			
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
				
				time.Sleep(actualDelay)
			}
			
			if pingErr != nil {
				log.Printf("⚠️  Database connection failed after %d attempts: %v (continuing without database)", maxRetries, pingErr)
				db = nil
			}
		} else {
			log.Printf("⚠️  Database connection failed: %v (continuing without database)", err)
			db = nil
		}
	} else {
		log.Println("📝 Database not configured (continuing without database)")
	}

	// Initialize Tidiness Processor
	tidinessProcessor = NewTidinessProcessor(db)
	log.Println("🧹 Tidiness processor initialized")

	// Get port from environment or use default
	port := os.Getenv("TIDINESS_API_PORT")
	if port == "" {
		port = "20400"
	}

	// Create router
	r := mux.NewRouter()

	// Health check endpoints
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/v1/health/scan", healthScanHandler).Methods("GET")

	// Tidiness management endpoints
	r.HandleFunc("/api/v1/scan", scanCodeHandler).Methods("POST")
	r.HandleFunc("/api/v1/analyze", analyzePatternHandler).Methods("POST")
	r.HandleFunc("/api/v1/cleanup", executeCleanupHandler).Methods("POST")

	// Processor endpoints (direct n8n workflow replacements)
	r.HandleFunc("/code-scanner", codeScannerHandler).Methods("POST")
	r.HandleFunc("/pattern-analyzer", patternAnalyzerHandler).Methods("POST")
	r.HandleFunc("/cleanup-executor", cleanupExecutorHandler).Methods("POST")

	// Set up CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(r)

	// Start server
	log.Printf("🧹 Code Tidiness Manager API starting on port %s", port)
	log.Printf("📊 Health check: http://localhost:%s/health", port)
	log.Printf("🚀 API endpoints: http://localhost:%s/api/v1/", port)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

// healthHandler handles basic health checks
func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Service:   "code-tidiness-manager-api",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// healthScanHandler tests scanning capability
func healthScanHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "tidiness-scanning",
		"capabilities": []string{
			"code-scanning",
			"pattern-analysis",
			"cleanup-execution",
			"safety-validation",
		},
		"supported_scan_types": []string{
			"backup_files",
			"temp_files",
			"empty_dirs", 
			"large_files",
		},
		"supported_analysis_types": []string{
			"duplicate_detection",
			"unused_imports",
			"dead_code",
			"hardcoded_values",
			"todo_comments",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// scanCodeHandler handles code scanning requests
func scanCodeHandler(w http.ResponseWriter, r *http.Request) {
	var req CodeScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := tidinessProcessor.ScanCode(ctx, req)
	if err != nil {
		log.Printf("Code scan error: %v", err)
		http.Error(w, "Code scan failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	if result.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	
	json.NewEncoder(w).Encode(result)
}

// analyzePatternHandler handles pattern analysis requests
func analyzePatternHandler(w http.ResponseWriter, r *http.Request) {
	var req PatternAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := tidinessProcessor.AnalyzePatterns(ctx, req)
	if err != nil {
		log.Printf("Pattern analysis error: %v", err)
		http.Error(w, "Pattern analysis failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	if result.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	
	json.NewEncoder(w).Encode(result)
}

// executeCleanupHandler handles cleanup execution requests
func executeCleanupHandler(w http.ResponseWriter, r *http.Request) {
	var req CleanupExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := tidinessProcessor.ExecuteCleanup(ctx, req)
	if err != nil {
		log.Printf("Cleanup execution error: %v", err)
		http.Error(w, "Cleanup execution failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	statusCode := http.StatusOK
	if !result.Success {
		statusCode = http.StatusInternalServerError
	}
	w.WriteHeader(statusCode)
	
	json.NewEncoder(w).Encode(result)
}

// Direct n8n workflow replacement handlers

// codeScannerHandler replaces code-scanner.json workflow
func codeScannerHandler(w http.ResponseWriter, r *http.Request) {
	var req CodeScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendTidinessError(w, "Invalid request body", "validation_error", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := tidinessProcessor.ScanCode(ctx, req)
	if err != nil {
		log.Printf("Code scanner error: %v", err)
		sendTidinessError(w, "Code scanning failed", "scan_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	if result.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	
	json.NewEncoder(w).Encode(result)
}

// patternAnalyzerHandler replaces pattern-analyzer.json workflow
func patternAnalyzerHandler(w http.ResponseWriter, r *http.Request) {
	var req PatternAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendTidinessError(w, "Invalid request body", "validation_error", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := tidinessProcessor.AnalyzePatterns(ctx, req)
	if err != nil {
		log.Printf("Pattern analyzer error: %v", err)
		sendTidinessError(w, "Pattern analysis failed", "analysis_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	if result.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	
	json.NewEncoder(w).Encode(result)
}

// cleanupExecutorHandler replaces cleanup-executor.json workflow
func cleanupExecutorHandler(w http.ResponseWriter, r *http.Request) {
	var req CleanupExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendTidinessError(w, "Invalid request body", "validation_error", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := tidinessProcessor.ExecuteCleanup(ctx, req)
	if err != nil {
		log.Printf("Cleanup executor error: %v", err)
		sendTidinessError(w, "Cleanup execution failed", "execution_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	statusCode := http.StatusOK
	if !result.Success {
		statusCode = http.StatusInternalServerError
	}
	w.WriteHeader(statusCode)
	
	json.NewEncoder(w).Encode(result)
}

// sendTidinessError sends a structured error response
func sendTidinessError(w http.ResponseWriter, message, errorType string, statusCode int) {
	response := map[string]interface{}{
		"success": false,
		"error": TidinessError{
			Message:   message,
			Type:      errorType,
			Timestamp: time.Now().Format(time.RFC3339),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}