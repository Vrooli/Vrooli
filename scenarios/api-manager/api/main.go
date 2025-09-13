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
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	apiVersion  = "1.0.0"
	serviceName = "api-manager"
	defaultPort = "8421"

	// Database limits
	maxDBConnections   = 25
	maxIdleConnections = 5
	connMaxLifetime    = 5 * time.Minute

	// Timeouts
	httpTimeout = 30 * time.Second
)

// Logger provides structured logging
type Logger struct {
	*log.Logger
}

func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[api-manager] ", log.LstdFlags|log.Lshortfile),
	}
}

func (l *Logger) Error(msg string, err error) {
	l.Printf("ERROR: %s: %v", msg, err)
}

func (l *Logger) Info(msg string) {
	l.Printf("INFO: %s", msg)
}

// HTTPError sends structured error response
func HTTPError(w http.ResponseWriter, message string, statusCode int, err error) {
	logger := NewLogger()
	if err != nil {
		logger.Error(fmt.Sprintf("HTTP %d: %s", statusCode, message), err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := map[string]interface{}{
		"error":     message,
		"status":    statusCode,
		"timestamp": time.Now().UTC(),
		"service":   serviceName,
		"version":   apiVersion,
	}

	json.NewEncoder(w).Encode(errorResp)
}

// Scenario represents a Vrooli scenario with API
type Scenario struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Path        string     `json:"path" db:"path"`
	Description string     `json:"description" db:"description"`
	Status      string     `json:"status" db:"status"`
	APIPort     *int       `json:"api_port,omitempty" db:"api_port"`
	APIVersion  string     `json:"api_version,omitempty" db:"api_version"`
	LastScanned *time.Time `json:"last_scanned,omitempty" db:"last_scanned"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// APIEndpoint represents a discovered API endpoint
type APIEndpoint struct {
	ID                   uuid.UUID   `json:"id"`
	ScenarioID          uuid.UUID   `json:"scenario_id"`
	Method              string      `json:"method"`
	Path                string      `json:"path"`
	HandlerFunction     string      `json:"handler_function,omitempty"`
	LineNumber          *int        `json:"line_number,omitempty"`
	FilePath            string      `json:"file_path,omitempty"`
	Description         string      `json:"description,omitempty"`
	Parameters          interface{} `json:"parameters,omitempty"`
	Responses           interface{} `json:"responses,omitempty"`
	SecurityRequirements interface{} `json:"security_requirements,omitempty"`
	CreatedAt           time.Time   `json:"created_at"`
}

// VulnerabilityScan represents a security vulnerability found during scanning
type VulnerabilityScan struct {
	ID            uuid.UUID  `json:"id"`
	ScenarioID    uuid.UUID  `json:"scenario_id"`
	ScanType      string     `json:"scan_type"`
	Severity      string     `json:"severity"`
	Category      string     `json:"category"`
	Title         string     `json:"title"`
	Description   string     `json:"description,omitempty"`
	FilePath      string     `json:"file_path,omitempty"`
	LineNumber    *int       `json:"line_number,omitempty"`
	CodeSnippet   string     `json:"code_snippet,omitempty"`
	Recommendation string    `json:"recommendation,omitempty"`
	Status        string     `json:"status"`
	FixedAt       *time.Time `json:"fixed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// Global database connection
var db *sql.DB

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start <scenario-name>

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	logger := NewLogger()
	logger.Info(fmt.Sprintf("Starting %s v%s", serviceName, apiVersion))

	// Initialize database
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Setup routes
	r := mux.NewRouter()
	
	// API versioning
	api := r.PathPrefix("/api/v1").Subrouter()
	
	// Health check
	api.HandleFunc("/health", healthHandler).Methods("GET")
	
	// Scenario management
	api.HandleFunc("/scenarios", getScenariosHandler).Methods("GET")
	api.HandleFunc("/scenarios/{name}", getScenarioHandler).Methods("GET")
	api.HandleFunc("/scenarios/{name}/scan", scanScenarioHandler).Methods("POST")
	api.HandleFunc("/scenarios/{name}/security-audit", securityAuditHandler).Methods("POST")
	api.HandleFunc("/scenarios/{name}/endpoints", getScenarioEndpointsHandler).Methods("GET")
	
	// Vulnerability management
	api.HandleFunc("/vulnerabilities", getVulnerabilitiesHandler).Methods("GET")
	api.HandleFunc("/vulnerabilities/{scenario_name}", getScenarioVulnerabilitiesHandler).Methods("GET")
	
	// System operations
	api.HandleFunc("/system/discover", discoverScenariosHandler).Methods("POST")
	api.HandleFunc("/system/status", getSystemStatusHandler).Methods("GET")
	api.HandleFunc("/system/validate-lifecycle", validateLifecycleProtectionHandler).Methods("GET")

	// Enable CORS
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		logger.Error("❌ API_PORT environment variable is required", nil)
		os.Exit(1)
	}
	logger.Info(fmt.Sprintf("API endpoints available at: http://localhost:%s/api/v1/", port))

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func initDB() (*sql.DB, error) {
	// ALL database configuration MUST come from environment - no defaults
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	// Validate required environment variables
	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		return nil, fmt.Errorf("❌ Missing required database configuration. Please set: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(maxDBConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxLifetime(connMaxLifetime)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	
	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📊 Connecting to: %s:%s/%s as user %s", dbHost, dbPort, dbName, dbUser)
	
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
		return nil, fmt.Errorf("❌ Database connection failed after %d attempts: %w", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
	return db, nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	overallStatus := "healthy"
	var errors []map[string]interface{}
	readiness := true
	
	// Schema-compliant health response structure
	healthResponse := map[string]interface{}{
		"status":    overallStatus,
		"service":   "api-manager-api",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true,
		"version":   apiVersion,
		"dependencies": map[string]interface{}{},
	}
	
	// Check database connectivity
	dbHealth := checkDatabaseHealth()
	healthResponse["dependencies"].(map[string]interface{})["database"] = dbHealth
	if dbHealth["status"] != "healthy" {
		overallStatus = "degraded"
		if dbHealth["status"] == "unhealthy" {
			readiness = false
			overallStatus = "unhealthy"
		}
		if dbHealth["error"] != nil {
			errors = append(errors, dbHealth["error"].(map[string]interface{}))
		}
	}
	
	// Check scanner functionality
	scannerHealth := checkScannerHealth()
	healthResponse["dependencies"].(map[string]interface{})["scanner"] = scannerHealth
	if scannerHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if scannerHealth["error"] != nil {
			errors = append(errors, scannerHealth["error"].(map[string]interface{}))
		}
	}
	
	// Check filesystem access (scenarios directory)
	fsHealth := checkFilesystemHealth()
	healthResponse["dependencies"].(map[string]interface{})["filesystem"] = fsHealth
	if fsHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if fsHealth["error"] != nil {
			errors = append(errors, fsHealth["error"].(map[string]interface{}))
		}
	}
	
	// Check optional Ollama AI service
	ollamaHealth := checkOllamaHealth()
	healthResponse["dependencies"].(map[string]interface{})["ollama"] = ollamaHealth
	if ollamaHealth["status"] == "unhealthy" {
		// Ollama is optional, so only degrade if configured but failing
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if ollamaHealth["error"] != nil {
			errors = append(errors, ollamaHealth["error"].(map[string]interface{}))
		}
	}
	
	// Check optional Qdrant vector database
	qdrantHealth := checkQdrantHealth()
	healthResponse["dependencies"].(map[string]interface{})["qdrant"] = qdrantHealth
	if qdrantHealth["status"] == "unhealthy" {
		// Qdrant is optional, so only degrade if configured but failing
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if qdrantHealth["error"] != nil {
			errors = append(errors, qdrantHealth["error"].(map[string]interface{}))
		}
	}
	
	// Update final status
	healthResponse["status"] = overallStatus
	healthResponse["readiness"] = readiness
	
	// Add errors if any
	if len(errors) > 0 {
		healthResponse["errors"] = errors
	}
	
	// Add metrics
	healthResponse["metrics"] = map[string]interface{}{
		"total_dependencies":   5,
		"healthy_dependencies": countHealthyDependencies(healthResponse["dependencies"].(map[string]interface{})),
		"response_time_ms":     time.Since(start).Milliseconds(),
	}
	
	// Add api-manager specific stats
	var scenarioCount, vulnerabilityCount, endpointCount int
	db.QueryRow("SELECT COUNT(*) FROM scenarios WHERE status = 'active'").Scan(&scenarioCount)
	db.QueryRow("SELECT COUNT(*) FROM vulnerability_scans WHERE status = 'open'").Scan(&vulnerabilityCount)
	db.QueryRow("SELECT COUNT(*) FROM api_endpoints").Scan(&endpointCount)
	
	healthResponse["api_manager_stats"] = map[string]interface{}{
		"active_scenarios": scenarioCount,
		"open_vulnerabilities": vulnerabilityCount,
		"tracked_endpoints": endpointCount,
	}
	
	// Return appropriate HTTP status
	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(healthResponse)
}

// Health check helper methods
func checkDatabaseHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}
	
	if db == nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "DATABASE_NOT_INITIALIZED",
			"message": "Database connection not initialized",
			"category": "configuration",
			"retryable": false,
		}
		return health
	}
	
	// Test database connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "DATABASE_CONNECTION_FAILED",
			"message": "Failed to ping database: " + err.Error(),
			"category": "resource",
			"retryable": true,
		}
		return health
	}
	health["checks"].(map[string]interface{})["ping"] = "ok"
	
	// Test scenarios table
	var tableExists bool
	tableQuery := `SELECT EXISTS(
		SELECT 1 FROM information_schema.tables 
		WHERE table_name = 'scenarios'
	)`
	if err := db.QueryRowContext(ctx, tableQuery).Scan(&tableExists); err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "DATABASE_SCHEMA_CHECK_FAILED",
			"message": "Failed to verify scenarios table: " + err.Error(),
			"category": "resource",
			"retryable": true,
		}
	} else if !tableExists {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "DATABASE_SCHEMA_MISSING",
			"message": "scenarios table does not exist",
			"category": "configuration",
			"retryable": false,
		}
	} else {
		health["checks"].(map[string]interface{})["scenarios_table"] = "ok"
	}
	
	// Test vulnerability_scans table
	tableQuery = `SELECT EXISTS(
		SELECT 1 FROM information_schema.tables 
		WHERE table_name = 'vulnerability_scans'
	)`
	if err := db.QueryRowContext(ctx, tableQuery).Scan(&tableExists); err != nil {
		if health["status"] == "healthy" {
			health["status"] = "degraded"
		}
		if health["error"] == nil {
			health["error"] = map[string]interface{}{
				"code": "DATABASE_VULN_TABLE_CHECK_FAILED",
				"message": "Failed to verify vulnerability_scans table: " + err.Error(),
				"category": "resource",
				"retryable": true,
			}
		}
	} else if tableExists {
		health["checks"].(map[string]interface{})["vulnerability_scans_table"] = "ok"
	}
	
	// Test api_endpoints table
	tableQuery = `SELECT EXISTS(
		SELECT 1 FROM information_schema.tables 
		WHERE table_name = 'api_endpoints'
	)`
	if err := db.QueryRowContext(ctx, tableQuery).Scan(&tableExists); err != nil {
		if health["status"] == "healthy" {
			health["status"] = "degraded"
		}
		if health["error"] == nil {
			health["error"] = map[string]interface{}{
				"code": "DATABASE_ENDPOINTS_TABLE_CHECK_FAILED",
				"message": "Failed to verify api_endpoints table: " + err.Error(),
				"category": "resource",
				"retryable": true,
			}
		}
	} else if tableExists {
		health["checks"].(map[string]interface{})["api_endpoints_table"] = "ok"
	}
	
	// Check active connections
	var openConnections int
	openConnections = db.Stats().OpenConnections
	health["checks"].(map[string]interface{})["open_connections"] = openConnections
	health["checks"].(map[string]interface{})["max_connections"] = maxDBConnections
	
	if openConnections > maxDBConnections * 9 / 10 { // 90% threshold
		if health["status"] == "healthy" {
			health["status"] = "degraded"
		}
		if health["error"] == nil {
			health["error"] = map[string]interface{}{
				"code": "DATABASE_CONNECTION_POOL_HIGH",
				"message": fmt.Sprintf("Connection pool usage high: %d/%d", openConnections, maxDBConnections),
				"category": "resource",
				"retryable": false,
			}
		}
	}
	
	return health
}

func checkScannerHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}
	
	// Check if scanner can be initialized
	scanner := NewVulnerabilityScanner(db)
	if scanner == nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "SCANNER_INITIALIZATION_FAILED",
			"message": "Failed to initialize vulnerability scanner",
			"category": "internal",
			"retryable": false,
		}
		return health
	}
	health["checks"].(map[string]interface{})["scanner_initialized"] = "ok"
	
	// Check recent scan activity
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	var lastScanTime *time.Time
	scanQuery := `SELECT MAX(created_at) FROM vulnerability_scans`
	db.QueryRowContext(ctx, scanQuery).Scan(&lastScanTime)
	
	if lastScanTime != nil {
		health["checks"].(map[string]interface{})["last_scan"] = lastScanTime.Format(time.RFC3339)
		hoursSinceLastScan := time.Since(*lastScanTime).Hours()
		if hoursSinceLastScan > 24 {
			health["checks"].(map[string]interface{})["scan_staleness_warning"] = fmt.Sprintf("No scans in %.0f hours", hoursSinceLastScan)
		}
	} else {
		health["checks"].(map[string]interface{})["last_scan"] = "never"
	}
	
	// Check open vulnerabilities count
	var criticalCount, highCount int
	db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vulnerability_scans WHERE status = 'open' AND severity = 'CRITICAL'`).Scan(&criticalCount)
	db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vulnerability_scans WHERE status = 'open' AND severity = 'HIGH'`).Scan(&highCount)
	
	health["checks"].(map[string]interface{})["critical_vulnerabilities"] = criticalCount
	health["checks"].(map[string]interface{})["high_vulnerabilities"] = highCount
	
	if criticalCount > 0 {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "CRITICAL_VULNERABILITIES_FOUND",
			"message": fmt.Sprintf("%d critical vulnerabilities require immediate attention", criticalCount),
			"category": "security",
			"retryable": false,
		}
	}
	
	return health
}

func checkFilesystemHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}
	
	// Check VROOLI_ROOT
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}
	
	// Check scenarios directory
	scenariosDir := filepath.Join(vrooliRoot, "scenarios")
	if info, err := os.Stat(scenariosDir); err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "SCENARIOS_DIR_NOT_ACCESSIBLE",
			"message": "Cannot access scenarios directory: " + err.Error(),
			"category": "configuration",
			"retryable": false,
		}
		return health
	} else if !info.IsDir() {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "SCENARIOS_PATH_NOT_DIRECTORY",
			"message": "Scenarios path exists but is not a directory",
			"category": "configuration",
			"retryable": false,
		}
		return health
	}
	health["checks"].(map[string]interface{})["scenarios_directory"] = "accessible"
	
	// Count scenarios
	scenarioCount := 0
	entries, err := os.ReadDir(scenariosDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				// Check if it has an api directory
				apiDir := filepath.Join(scenariosDir, entry.Name(), "api")
				if _, err := os.Stat(apiDir); err == nil {
					scenarioCount++
				}
			}
		}
		health["checks"].(map[string]interface{})["discovered_scenarios"] = scenarioCount
	}
	
	// Check write permissions
	testFile := filepath.Join(scenariosDir, ".api-manager-health-check")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "SCENARIOS_DIR_NOT_WRITABLE",
			"message": "Cannot write to scenarios directory: " + err.Error(),
			"category": "permission",
			"retryable": false,
		}
	} else {
		os.Remove(testFile)
		health["checks"].(map[string]interface{})["write_permission"] = "ok"
	}
	
	return health
}

func countHealthyDependencies(deps map[string]interface{}) int {
	count := 0
	for _, dep := range deps {
		if depMap, ok := dep.(map[string]interface{}); ok {
			if status, exists := depMap["status"]; exists && status == "healthy" {
				count++
			}
		}
	}
	return count
}

func checkOllamaHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "not_configured",
		"checks": map[string]interface{}{},
	}

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	// Test Ollama connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ollamaURL+"/api/tags", nil)
	if err != nil {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["ai_analysis"] = "disabled"
		return health
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["ai_analysis"] = "disabled"
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		health["status"] = "healthy"
		health["checks"].(map[string]interface{})["connectivity"] = "ok"
		
		// Parse response to check available models
		var response struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err == nil {
			health["checks"].(map[string]interface{})["available_models"] = len(response.Models)
			
			// Check for required models for API analysis
			requiredModels := []string{"llama3.1:8b", "nomic-embed-text"}
			availableModels := make(map[string]bool)
			for _, model := range response.Models {
				availableModels[model.Name] = true
			}
			
			missingModels := []string{}
			for _, required := range requiredModels {
				if !availableModels[required] {
					missingModels = append(missingModels, required)
				}
			}
			
			if len(missingModels) > 0 {
				health["status"] = "degraded"
				health["error"] = map[string]interface{}{
					"code":      "OLLAMA_MODELS_MISSING",
					"message":   fmt.Sprintf("Missing required models for API analysis: %v", missingModels),
					"category":  "configuration",
					"retryable": false,
				}
			} else {
				health["checks"].(map[string]interface{})["required_models"] = "available"
				health["checks"].(map[string]interface{})["ai_analysis"] = "enabled"
			}
		}
	} else {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "OLLAMA_UNHEALTHY",
			"message":   fmt.Sprintf("Ollama returned status %d", resp.StatusCode),
			"category":  "resource",
			"retryable": true,
		}
	}

	return health
}

func checkQdrantHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "not_configured",
		"checks": map[string]interface{}{},
	}

	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}

	// Test Qdrant connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", qdrantURL+"/health", nil)
	if err != nil {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["vector_search"] = "disabled"
		return health
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["vector_search"] = "disabled"
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		health["status"] = "healthy"
		health["checks"].(map[string]interface{})["connectivity"] = "ok"
		health["checks"].(map[string]interface{})["vector_search"] = "enabled"
	} else {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "QDRANT_UNHEALTHY",
			"message":   fmt.Sprintf("Qdrant returned status %d", resp.StatusCode),
			"category":  "resource",
			"retryable": true,
		}
	}

	return health
}

func getScenariosHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()
	w.Header().Set("Content-Type", "application/json")

	query := `
		SELECT id, name, path, description, status, api_port, api_version, last_scanned, created_at, updated_at
		FROM scenarios 
		WHERE status = 'active'
		ORDER BY name`

	rows, err := db.Query(query)
	if err != nil {
		HTTPError(w, "Failed to query scenarios", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var scenarios []Scenario
	for rows.Next() {
		var s Scenario
		err := rows.Scan(&s.ID, &s.Name, &s.Path, &s.Description, &s.Status, 
			&s.APIPort, &s.APIVersion, &s.LastScanned, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			logger.Error("Failed to scan scenario row", err)
			continue
		}
		scenarios = append(scenarios, s)
	}

	response := map[string]interface{}{
		"scenarios": scenarios,
		"count":     len(scenarios),
		"timestamp": time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

func getScenarioHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]

	w.Header().Set("Content-Type", "application/json")

	query := `
		SELECT id, name, path, description, status, api_port, api_version, last_scanned, created_at, updated_at
		FROM scenarios 
		WHERE name = $1`

	var s Scenario
	err := db.QueryRow(query, scenarioName).Scan(&s.ID, &s.Name, &s.Path, &s.Description, 
		&s.Status, &s.APIPort, &s.APIVersion, &s.LastScanned, &s.CreatedAt, &s.UpdatedAt)
	
	if err == sql.ErrNoRows {
		HTTPError(w, "Scenario not found", http.StatusNotFound, nil)
		return
	} else if err != nil {
		HTTPError(w, "Failed to query scenario", http.StatusInternalServerError, err)
		return
	}

	json.NewEncoder(w).Encode(s)
}

func scanScenarioHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]

	w.Header().Set("Content-Type", "application/json")

	// Get scenario path
	var scenarioPath string
	err := db.QueryRow("SELECT path FROM scenarios WHERE name = $1", scenarioName).Scan(&scenarioPath)
	if err == sql.ErrNoRows {
		// Try to find the scenario in the filesystem
		scenarioPath = filepath.Join(os.Getenv("HOME"), "Vrooli", "scenarios", scenarioName)
		if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
			HTTPError(w, "Scenario not found", http.StatusNotFound, nil)
			return
		}
	} else if err != nil {
		HTTPError(w, "Failed to query scenario", http.StatusInternalServerError, err)
		return
	}

	// Create scanner and perform scan
	scanner := NewVulnerabilityScanner(db)
	result, err := scanner.ScanScenario(scenarioPath, scenarioName)
	if err != nil {
		HTTPError(w, "Scan failed", http.StatusInternalServerError, err)
		return
	}

	// Count critical vulnerabilities
	criticalCount := 0
	httpLeakCount := 0
	for _, vuln := range result.Vulnerabilities {
		if vuln.Severity == "CRITICAL" {
			criticalCount++
			if vuln.Type == "HTTP_RESPONSE_BODY_LEAK" {
				httpLeakCount++
			}
		}
	}

	response := map[string]interface{}{
		"scenario":            scenarioName,
		"status":             "scan_completed",
		"vulnerabilities":    result.Vulnerabilities,
		"total_issues":       len(result.Vulnerabilities),
		"critical_issues":    criticalCount,
		"http_leak_issues":   httpLeakCount,
		"scan_time":          result.ScanTime,
		"message":            fmt.Sprintf("Found %d vulnerabilities, %d CRITICAL HTTP response body leaks", len(result.Vulnerabilities), httpLeakCount),
		"timestamp":          time.Now().UTC(),
	}

	// Add urgent warning if HTTP leaks found
	if httpLeakCount > 0 {
		response["urgent_warning"] = fmt.Sprintf("CRITICAL: %d unclosed HTTP response bodies detected! This is causing TCP connection exhaustion. Fix immediately!", httpLeakCount)
		response["immediate_action"] = "Run 'pkill -f " + scenarioName + "' to restart this API and clear leaked connections"
	}

	json.NewEncoder(w).Encode(response)
}

func securityAuditHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	logger := NewLogger()

	w.Header().Set("Content-Type", "application/json")

	// Get scan type from request body (optional)
	var requestBody struct {
		ScanType string `json:"scan_type"`
		FilePath string `json:"file_path"`
	}
	json.NewDecoder(r.Body).Decode(&requestBody)

	// Get scenario path
	var scenarioPath string
	err := db.QueryRow("SELECT path FROM scenarios WHERE name = $1", scenarioName).Scan(&scenarioPath)
	if err == sql.ErrNoRows {
		// Try to find the scenario in the filesystem
		scenarioPath = filepath.Join(os.Getenv("HOME"), "Vrooli", "scenarios", scenarioName)
		if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
			HTTPError(w, "Scenario not found", http.StatusNotFound, nil)
			return
		}
	} else if err != nil {
		HTTPError(w, "Failed to query scenario", http.StatusInternalServerError, err)
		return
	}

	// If specific file provided, validate it exists
	if requestBody.FilePath != "" {
		fullPath := filepath.Join(scenarioPath, requestBody.FilePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			HTTPError(w, "Specified file not found", http.StatusNotFound, nil)
			return
		}
	}

	logger.Info(fmt.Sprintf("Starting security audit for scenario: %s", scenarioName))

	// Create scanner and perform security audit
	scanner := NewVulnerabilityScanner(db)
	result, err := scanner.SecurityAudit(scenarioPath, scenarioName)
	if err != nil {
		HTTPError(w, "Security audit failed", http.StatusInternalServerError, err)
		return
	}

	// Count issues by severity
	severityCount := map[string]int{
		"CRITICAL": 0,
		"HIGH":     0,
		"MEDIUM":   0,
		"LOW":      0,
	}
	for _, issue := range result.Issues {
		if _, ok := severityCount[issue.Severity]; ok {
			severityCount[issue.Severity]++
		}
	}

	// Generate recommendations
	recommendations := []string{
		"Review SQL query patterns for injection risks",
		"Implement input validation middleware",
		"Add rate limiting to all endpoints",
		"Review CORS configuration",
		"Implement proper error handling",
	}

	response := map[string]interface{}{
		"status":         "completed",
		"scenario":       scenarioName,
		"audit_time":     result.AuditTime,
		"issues":         result.Issues,
		"total_issues":   len(result.Issues),
		"severity_count": severityCount,
		"recommendations": recommendations,
		"timestamp":      time.Now().UTC(),
	}

	// Add critical warning if high-severity issues found
	if severityCount["CRITICAL"] > 0 || severityCount["HIGH"] > 0 {
		response["warning"] = fmt.Sprintf("Found %d CRITICAL and %d HIGH severity issues requiring immediate attention", 
			severityCount["CRITICAL"], severityCount["HIGH"])
	}

	json.NewEncoder(w).Encode(response)
}

func getScenarioEndpointsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]

	w.Header().Set("Content-Type", "application/json")

	// First, get scenario ID
	var scenarioID uuid.UUID
	err := db.QueryRow("SELECT id FROM scenarios WHERE name = $1", scenarioName).Scan(&scenarioID)
	if err == sql.ErrNoRows {
		HTTPError(w, "Scenario not found", http.StatusNotFound, nil)
		return
	} else if err != nil {
		HTTPError(w, "Failed to query scenario", http.StatusInternalServerError, err)
		return
	}

	// Get endpoints
	query := `
		SELECT id, scenario_id, method, path, handler_function, line_number, 
			   file_path, description, parameters, responses, security_requirements, created_at
		FROM api_endpoints 
		WHERE scenario_id = $1
		ORDER BY method, path`

	rows, err := db.Query(query, scenarioID)
	if err != nil {
		HTTPError(w, "Failed to query endpoints", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var endpoints []APIEndpoint
	for rows.Next() {
		var e APIEndpoint
		err := rows.Scan(&e.ID, &e.ScenarioID, &e.Method, &e.Path, &e.HandlerFunction,
			&e.LineNumber, &e.FilePath, &e.Description, &e.Parameters, &e.Responses,
			&e.SecurityRequirements, &e.CreatedAt)
		if err != nil {
			continue
		}
		endpoints = append(endpoints, e)
	}

	response := map[string]interface{}{
		"scenario":  scenarioName,
		"endpoints": endpoints,
		"count":     len(endpoints),
		"timestamp": time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

func getVulnerabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := `
		SELECT v.id, v.scenario_id, v.scan_type, v.severity, v.category, v.title,
			   v.description, v.file_path, v.line_number, v.code_snippet,
			   v.recommendation, v.status, v.fixed_at, v.created_at, s.name as scenario_name
		FROM vulnerability_scans v
		JOIN scenarios s ON v.scenario_id = s.id
		WHERE v.status = 'open'
		ORDER BY 
			CASE v.severity WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END,
			v.created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		HTTPError(w, "Failed to query vulnerabilities", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var vulnerabilities []map[string]interface{}
	for rows.Next() {
		var v VulnerabilityScan
		var scenarioName string
		err := rows.Scan(&v.ID, &v.ScenarioID, &v.ScanType, &v.Severity, &v.Category,
			&v.Title, &v.Description, &v.FilePath, &v.LineNumber, &v.CodeSnippet,
			&v.Recommendation, &v.Status, &v.FixedAt, &v.CreatedAt, &scenarioName)
		if err != nil {
			continue
		}

		vulnMap := map[string]interface{}{
			"id":             v.ID,
			"scenario_name":  scenarioName,
			"scan_type":      v.ScanType,
			"severity":       v.Severity,
			"category":       v.Category,
			"title":          v.Title,
			"description":    v.Description,
			"file_path":      v.FilePath,
			"line_number":    v.LineNumber,
			"code_snippet":   v.CodeSnippet,
			"recommendation": v.Recommendation,
			"status":         v.Status,
			"created_at":     v.CreatedAt,
		}
		vulnerabilities = append(vulnerabilities, vulnMap)
	}

	response := map[string]interface{}{
		"vulnerabilities": vulnerabilities,
		"count":          len(vulnerabilities),
		"timestamp":      time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

func getScenarioVulnerabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario_name"]

	w.Header().Set("Content-Type", "application/json")

	query := `
		SELECT v.id, v.scenario_id, v.scan_type, v.severity, v.category, v.title,
			   v.description, v.file_path, v.line_number, v.code_snippet,
			   v.recommendation, v.status, v.fixed_at, v.created_at
		FROM vulnerability_scans v
		JOIN scenarios s ON v.scenario_id = s.id
		WHERE s.name = $1
		ORDER BY 
			CASE v.severity WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END,
			v.created_at DESC`

	rows, err := db.Query(query, scenarioName)
	if err != nil {
		HTTPError(w, "Failed to query vulnerabilities", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var vulnerabilities []VulnerabilityScan
	for rows.Next() {
		var v VulnerabilityScan
		err := rows.Scan(&v.ID, &v.ScenarioID, &v.ScanType, &v.Severity, &v.Category,
			&v.Title, &v.Description, &v.FilePath, &v.LineNumber, &v.CodeSnippet,
			&v.Recommendation, &v.Status, &v.FixedAt, &v.CreatedAt)
		if err != nil {
			continue
		}
		vulnerabilities = append(vulnerabilities, v)
	}

	response := map[string]interface{}{
		"scenario":       scenarioName,
		"vulnerabilities": vulnerabilities,
		"count":          len(vulnerabilities),
		"timestamp":      time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

func discoverScenariosHandler(w http.ResponseWriter, r *http.Request) {
	logger := NewLogger()
	w.Header().Set("Content-Type", "application/json")

	// TODO: Implement scenario discovery logic
	// This will scan the scenarios directory and discover APIs
	
	logger.Info("Starting scenario discovery...")
	
	response := map[string]interface{}{
		"status":    "discovery_initiated",
		"message":   "Scenario discovery functionality will be implemented",
		"timestamp": time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

func getSystemStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get basic system statistics
	var scenarioCount, vulnerabilityCount, endpointCount int

	db.QueryRow("SELECT COUNT(*) FROM scenarios WHERE status = 'active'").Scan(&scenarioCount)
	db.QueryRow("SELECT COUNT(*) FROM vulnerability_scans WHERE status = 'open'").Scan(&vulnerabilityCount)
	db.QueryRow("SELECT COUNT(*) FROM api_endpoints").Scan(&endpointCount)

	response := map[string]interface{}{
		"status":           "operational",
		"service":          serviceName,
		"version":          apiVersion,
		"active_scenarios": scenarioCount,
		"open_vulnerabilities": vulnerabilityCount,
		"tracked_endpoints": endpointCount,
		"database_status":  "connected",
		"timestamp":        time.Now().UTC(),
	}

	if err := db.Ping(); err != nil {
		response["status"] = "degraded"
		response["database_status"] = "disconnected"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// validateScenarioLifecycleProtection checks if all scenario APIs have lifecycle protection
func validateScenarioLifecycleProtection() ([]map[string]interface{}, error) {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		return nil, fmt.Errorf("VROOLI_ROOT environment variable not set")
	}
	
	scenariosDir := filepath.Join(vrooliRoot, "scenarios")
	
	var results []map[string]interface{}
	var unprotectedScenarios []string
	var protectedScenarios []string
	
	// Find all scenario API main.go files
	err := filepath.Walk(scenariosDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Look for main.go files in api directories
		if info.Name() == "main.go" && strings.Contains(path, "/api/") {
			// Extract scenario name from path
			scenarioName := filepath.Base(filepath.Dir(filepath.Dir(path)))
			
			// Read the main.go file
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				results = append(results, map[string]interface{}{
					"scenario": scenarioName,
					"status": "error",
					"message": fmt.Sprintf("Could not read file: %v", readErr),
					"path": path,
				})
				return nil
			}
			
			// Check if it contains lifecycle protection
			contentStr := string(content)
			if strings.Contains(contentStr, "VROOLI_LIFECYCLE_MANAGED") {
				protectedScenarios = append(protectedScenarios, scenarioName)
				results = append(results, map[string]interface{}{
					"scenario": scenarioName,
					"status": "protected",
					"message": "Has lifecycle protection",
					"path": path,
				})
			} else {
				unprotectedScenarios = append(unprotectedScenarios, scenarioName)
				results = append(results, map[string]interface{}{
					"scenario": scenarioName,
					"status": "unprotected",
					"message": "Missing VROOLI_LIFECYCLE_MANAGED check",
					"path": path,
				})
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// Add summary information
	summary := map[string]interface{}{
		"scenario": "SUMMARY",
		"status": "summary",
		"message": fmt.Sprintf("Found %d scenarios: %d protected, %d unprotected", 
			len(protectedScenarios)+len(unprotectedScenarios), 
			len(protectedScenarios), 
			len(unprotectedScenarios)),
		"protected_count": len(protectedScenarios),
		"unprotected_count": len(unprotectedScenarios),
		"protected_scenarios": protectedScenarios,
		"unprotected_scenarios": unprotectedScenarios,
	}
	
	// Insert summary at the beginning
	results = append([]map[string]interface{}{summary}, results...)
	
	return results, nil
}

// validateLifecycleProtectionHandler handles GET /system/validate-lifecycle
func validateLifecycleProtectionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	results, err := validateScenarioLifecycleProtection()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error": err.Error(),
		})
		return
	}
	
	// Extract summary from first result
	summary := results[0]
	protectedCount := summary["protected_count"].(int)
	unprotectedCount := summary["unprotected_count"].(int)
	
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"summary": summary,
			"scenarios": results[1:], // Skip the summary item
			"compliance_rate": float64(protectedCount) / float64(protectedCount+unprotectedCount) * 100,
			"recommendations": []string{
				"All scenario APIs should include lifecycle protection to prevent direct execution",
				"Use the standard pattern: check VROOLI_LIFECYCLE_MANAGED environment variable",
				"Add protection at the very beginning of main() function",
			},
		},
	}
	
	json.NewEncoder(w).Encode(response)
}

// Removed getEnv function - no defaults allowed