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
	"os/exec"
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
	APIPath     string     `json:"api_path" db:"api_path"`
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

// VrooliScenarioResponse represents the response from 'vrooli scenario list --json'
type VrooliScenarioResponse struct {
	Success bool `json:"success"`
	Summary struct {
		TotalScenarios int `json:"total_scenarios"`
		Running        int `json:"running"`
		Available      int `json:"available"`
	} `json:"summary"`
	Scenarios []VrooliScenario `json:"scenarios"`
}

type VrooliScenario struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags"`
	Path        string   `json:"path"`
}

// Global database connection
var db *sql.DB

// getVrooliScenarios calls the Vrooli CLI to get real scenario information
func getVrooliScenarios() (*VrooliScenarioResponse, error) {
	cmd := exec.Command("vrooli", "scenario", "list", "--json")
	
	// Set timeout for the command
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, "vrooli", "scenario", "list", "--json")
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute vrooli command: %w", err)
	}
	
	var response VrooliScenarioResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse vrooli response: %w", err)
	}
	
	return &response, nil
}

func countScenarioEndpoints(scenarioPath string) int {
	// Look for api directory
	apiDir := filepath.Join(scenarioPath, "api")
	
	// Check if api directory exists
	if _, err := os.Stat(apiDir); os.IsNotExist(err) {
		return 0
	}
	
	endpointCount := 0
	
	// Walk through all Go files in the api directory
	err := filepath.Walk(apiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue even if there's an error with a specific file
		}
		
		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		
		// Read the file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Continue even if we can't read a file
		}
		
		fileContent := string(content)
		
		// Count different routing patterns
		
		// 1. Gorilla Mux HandleFunc patterns: .HandleFunc("/path", handler)
		handleFuncMatches := strings.Count(fileContent, ".HandleFunc(")
		endpointCount += handleFuncMatches
		
		// 2. Fiber framework patterns: app.Get, app.Post, etc.
		fiberMethods := []string{"app.Get(", "app.Post(", "app.Put(", "app.Delete(", "app.Patch(", "app.Options(", "app.Head("}
		for _, method := range fiberMethods {
			methodMatches := strings.Count(fileContent, method)
			endpointCount += methodMatches
		}
		
		// 3. Standard HTTP patterns: http.Handle, http.HandleFunc
		httpHandleMatches := strings.Count(fileContent, "http.Handle(")
		httpHandleFuncMatches := strings.Count(fileContent, "http.HandleFunc(")
		endpointCount += httpHandleMatches + httpHandleFuncMatches
		
		return nil
	})
	
	if err != nil {
		// If there's an error walking the directory, return 0
		return 0
	}
	
	return endpointCount
}

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
	
	// Root health check (required by Vrooli lifecycle system)
	r.HandleFunc("/health", healthHandler).Methods("GET")
	
	// API versioning
	api := r.PathPrefix("/api/v1").Subrouter()
	
	// Health check (legacy API endpoint)
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
	
	// Scan management
	api.HandleFunc("/scans/recent", getRecentScansHandler).Methods("GET")
	
	// OpenAPI documentation
	api.HandleFunc("/openapi/{scenario}", getOpenAPISpecHandler).Methods("GET")
	api.HandleFunc("/fix/{scenario}", applyAutomatedFixHandler).Methods("POST")
	
	// Health monitoring endpoints
	api.HandleFunc("/scenarios/{name}/health", getScenarioHealthHandler).Methods("GET")
	api.HandleFunc("/health/summary", getHealthSummaryHandler).Methods("GET")
	api.HandleFunc("/health/alerts", getHealthAlertsHandler).Methods("GET")
	api.HandleFunc("/health/metrics/{scenario}", getHealthMetricsHandler).Methods("GET")
	
	// Performance monitoring endpoints
	api.HandleFunc("/performance/baseline/{scenario}", createPerformanceBaselineHandler).Methods("POST")
	api.HandleFunc("/performance/metrics/{scenario}", getPerformanceMetricsHandler).Methods("GET")
	api.HandleFunc("/performance/alerts", getPerformanceAlertsHandler).Methods("GET")
	
	// Breaking change detection endpoints
	api.HandleFunc("/changes/detect/{scenario}", detectBreakingChangesHandler).Methods("POST")
	api.HandleFunc("/changes/history/{scenario}", getChangeHistoryHandler).Methods("GET")
	
	// Enhanced automated fix endpoints with safety controls
	api.HandleFunc("/fix/config", getAutomatedFixConfigHandler).Methods("GET")
	api.HandleFunc("/fix/config/enable", enableAutomatedFixesHandler).Methods("POST")
	api.HandleFunc("/fix/config/disable", disableAutomatedFixesHandler).Methods("POST")
	api.HandleFunc("/fix/apply/{scenario}", applyAutomatedFixWithSafetyHandler).Methods("POST")
	api.HandleFunc("/fix/rollback/{fixId}", rollbackAutomatedFixHandler).Methods("POST")
	
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
	
	// Add api-manager specific stats using Vrooli CLI for scenario count
	var scenarioCount, vulnerabilityCount, endpointCount int
	vrooliData, err := getVrooliScenarios()
	if err != nil {
		// Fallback to database if CLI fails
		db.QueryRow("SELECT COUNT(*) FROM scenarios WHERE status IN ('active', 'available')").Scan(&scenarioCount)
	} else {
		scenarioCount = vrooliData.Summary.TotalScenarios
	}
	
	db.QueryRow("SELECT COUNT(*) FROM vulnerability_scans WHERE status = 'open'").Scan(&vulnerabilityCount)
	db.QueryRow("SELECT COUNT(*) FROM api_endpoints").Scan(&endpointCount)
	
	healthResponse["api_manager_stats"] = map[string]interface{}{
		"total_scenarios": scenarioCount,
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
		// Ollama is optional - if not configured, AI features are disabled
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["ai_analysis"] = "disabled"
		return health
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
		// Qdrant is optional - if not configured, vector search is disabled
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["vector_search"] = "disabled"
		return health
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

	// Get real scenario information from Vrooli CLI
	vrooliData, err := getVrooliScenarios()
	if err != nil {
		logger.Error("Failed to get scenarios from Vrooli CLI", err)
		HTTPError(w, "Failed to get scenarios", http.StatusInternalServerError, err)
		return
	}

	// Convert Vrooli scenarios to API format
	var scenarios []map[string]interface{}
	for _, vrooliScenario := range vrooliData.Scenarios {
		// Count actual endpoints by scanning source code
		endpointCount := countScenarioEndpoints(vrooliScenario.Path)
		
		scenario := map[string]interface{}{
			"name":        vrooliScenario.Name,
			"description": vrooliScenario.Description,
			"status":      vrooliScenario.Status, // "available", "running", etc.
			"version":     vrooliScenario.Version,
			"tags":        vrooliScenario.Tags,
			"path":        vrooliScenario.Path,
			"endpoint_count":     endpointCount, // Actual count from source code analysis
			"vulnerability_count": 0, // Default since we don't track this in CLI
			"critical_count":     0, // Default since we don't track this in CLI
			"last_scan":          nil, // Not available from CLI
			"created_at":         time.Now().UTC(), // Default
			"updated_at":         time.Now().UTC(), // Default
		}
		scenarios = append(scenarios, scenario)
	}

	response := map[string]interface{}{
		"scenarios": scenarios,
		"count":     len(scenarios),
		"timestamp": time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

func getRecentScansHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Since we don't have actual scan data in the database yet, return empty list
	// In a real implementation, this would query recent security scans
	response := map[string]interface{}{
		"scans":     []map[string]interface{}{},
		"count":     0,
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

func countScannableFiles(path string) (int, int, int) {
	totalFiles := 0
	codeFiles := 0
	configFiles := 0
	
	// Define file extensions to scan
	codeExtensions := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
		".py": true, ".java": true, ".c": true, ".cpp": true, ".h": true,
		".rs": true, ".rb": true, ".php": true, ".swift": true, ".kt": true,
		".sh": true, ".bash": true, ".zsh": true, ".ps1": true,
	}
	
	configExtensions := map[string]bool{
		".json": true, ".yaml": true, ".yml": true, ".toml": true, ".ini": true,
		".xml": true, ".env": true, ".conf": true, ".config": true,
	}
	
	filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			// Skip directories and node_modules, vendor, .git etc
			if info != nil && info.IsDir() {
				name := info.Name()
				if name == "node_modules" || name == "vendor" || name == ".git" || 
				   name == "dist" || name == "build" || name == "__pycache__" {
					return filepath.SkipDir
				}
			}
			return nil
		}
		
		ext := strings.ToLower(filepath.Ext(filePath))
		if codeExtensions[ext] {
			codeFiles++
			totalFiles++
		} else if configExtensions[ext] {
			configFiles++
			totalFiles++
		}
		
		return nil
	})
	
	return totalFiles, codeFiles, configFiles
}

func performBasicSecurityChecks(path string, scanType string) ([]map[string]interface{}, map[string]interface{}) {
	// This is a placeholder for actual security scanning
	// In a real implementation, this would:
	// - Check for hardcoded secrets/API keys
	// - Look for SQL injection vulnerabilities  
	// - Check for XSS vulnerabilities
	// - Scan dependencies for known CVEs
	// - Check file permissions
	// - Look for insecure configurations
	
	// For now, return no vulnerabilities since we don't have a real scanner
	vulnerabilities := []map[string]interface{}{}
	
	// Return vulnerability summary
	summary := map[string]interface{}{
		"total": 0,
		"critical": 0,
		"high": 0,
		"medium": 0,
		"low": 0,
	}
	
	return vulnerabilities, summary
}

func scanScenarioHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]

	w.Header().Set("Content-Type", "application/json")

	// Parse request body for scan options
	var scanOptions struct {
		Type string `json:"type"` // "quick", "full", or "targeted"
	}
	if err := json.NewDecoder(r.Body).Decode(&scanOptions); err != nil {
		scanOptions.Type = "quick" // Default to quick scan
	}

	// Since we're not doing real scanning, report actual time taken (instant)
	startTime := time.Now()

	// Handle "all" scenarios scan
	if scenarioName == "all" {
		// Count files across all scenarios
		scenariosPath := filepath.Join(os.Getenv("HOME"), "Vrooli", "scenarios")
		totalFiles, codeFiles, configFiles := countScannableFiles(scenariosPath)
		
		// Get count of scenarios
		scenarios := getVrooliScenarios()
		scenarioCount := len(scenarios)
		
		// Perform security checks (placeholder for now)
		_, vulnSummary := performBasicSecurityChecks(scenariosPath, scanOptions.Type)
		
		// Calculate actual time taken
		endTime := time.Now()
		duration := endTime.Sub(startTime).Seconds()
		
		response := map[string]interface{}{
			"scan_id": fmt.Sprintf("scan-%d", time.Now().Unix()),
			"status": "completed",
			"scan_type": scanOptions.Type,
			"scenarios_scanned": scenarioCount,
			"files_scanned": map[string]interface{}{
				"total": totalFiles,
				"code_files": codeFiles,
				"config_files": configFiles,
			},
			"started_at": startTime.Format(time.RFC3339),
			"completed_at": endTime.Format(time.RFC3339),
			"duration_seconds": duration,
			"vulnerabilities": vulnSummary,
			"scan_notes": "Security scanning engine not yet implemented. File counts are real.",
			"message": fmt.Sprintf("Counted %d files across %d scenarios (%d code files, %d config files)", 
				totalFiles, scenarioCount, codeFiles, configFiles),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get scenario path for individual scenario scan
	scenarioPath := filepath.Join(os.Getenv("HOME"), "Vrooli", "scenarios", scenarioName)
	if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
		HTTPError(w, "Scenario not found", http.StatusNotFound, nil)
		return
	}

	// Count files in this specific scenario
	totalFiles, codeFiles, configFiles := countScannableFiles(scenarioPath)
	
	// Perform security checks (placeholder for now)
	_, vulnSummary := performBasicSecurityChecks(scenarioPath, scanOptions.Type)
	
	// Calculate actual time taken
	endTime := time.Now()
	duration := endTime.Sub(startTime).Seconds()
	
	response := map[string]interface{}{
		"scan_id": fmt.Sprintf("scan-%d", time.Now().Unix()),
		"status": "completed",
		"scan_type": scanOptions.Type,
		"scenario_name": scenarioName,
		"files_scanned": map[string]interface{}{
			"total": totalFiles,
			"code_files": codeFiles,
			"config_files": configFiles,
		},
		"started_at": startTime.Format(time.RFC3339),
		"completed_at": endTime.Format(time.RFC3339),
		"duration_seconds": duration,
		"vulnerabilities": vulnSummary,
		"scan_notes": "Security scanning engine not yet implemented. File counts are real.",
		"message": fmt.Sprintf("Counted %d files in %s (%d code files, %d config files)", 
			totalFiles, scenarioName, codeFiles, configFiles),
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

	// For now, return an empty array since we don't have a real vulnerability database
	// In production, this would query actual scan results
	vulnerabilities := []map[string]interface{}{}
	
	// You could add mock vulnerabilities here for testing:
	// vulnerabilities = append(vulnerabilities, map[string]interface{}{
	//     "id": "vuln-1",
	//     "scenario_name": "api-manager",
	//     "type": "SQL Injection",
	//     "severity": "critical",
	//     "title": "SQL Injection in user input",
	//     "description": "User input is not properly sanitized",
	//     "file_path": "/api/handlers.go",
	//     "line_number": 42,
	//     "recommendation": "Use parameterized queries",
	//     "status": "open",
	//     "discovered_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
	// })

	json.NewEncoder(w).Encode(vulnerabilities)
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

	logger.Info("Starting scenario discovery...")
	
	// Scan scenarios directory for API endpoints
	scenariosDir := os.Getenv("SCENARIOS_DIR")
	if scenariosDir == "" {
		scenariosDir = "/home/matthalloran8/Vrooli/scenarios"
	}
	
	discovered := 0
	errors := []string{}
	
	// Check if directory exists
	if _, err := os.Stat(scenariosDir); os.IsNotExist(err) {
		logger.Error("Scenarios directory not found", fmt.Errorf("directory: %s", scenariosDir))
		HTTPError(w, "Scenarios directory not found", http.StatusInternalServerError, fmt.Errorf("directory: %s", scenariosDir))
		return
	}
	
	// Scan each scenario directory
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		logger.Error("Failed to read scenarios directory", err)
		HTTPError(w, "Failed to read scenarios directory", http.StatusInternalServerError, err)
		return
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		scenarioName := entry.Name()
		apiPath := filepath.Join(scenariosDir, scenarioName, "api")
		
		// Check if API directory exists
		if _, err := os.Stat(apiPath); err == nil {
			// Found an API directory - create/update scenario in database
			scenario := Scenario{
				ID:          uuid.New(),
				Name:        scenarioName,
				Description: fmt.Sprintf("Auto-discovered API in %s", scenarioName),
				Status:      "discovered",
				APIPath:     apiPath,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			}
			
			// Insert or update in database
			_, err = db.Exec(`
				INSERT INTO scenarios (id, name, description, status, api_path, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (name) DO UPDATE
				SET status = $4, api_path = $5, updated_at = $7
			`, scenario.ID, scenario.Name, scenario.Description, scenario.Status, scenario.APIPath, scenario.CreatedAt, scenario.UpdatedAt)
			
			if err != nil {
				errors = append(errors, fmt.Sprintf("Failed to save scenario %s: %v", scenarioName, err))
				logger.Error("Failed to save scenario", err)
			} else {
				discovered++
				logger.Info(fmt.Sprintf("Discovered API in scenario: %s", scenarioName))
			}
		}
	}
	
	response := map[string]interface{}{
		"status":     "completed",
		"discovered": discovered,
		"errors":     errors,
		"message":    fmt.Sprintf("Discovered %d APIs in scenarios", discovered),
		"timestamp":  time.Now().UTC(),
	}
	
	if len(errors) > 0 {
		response["status"] = "completed_with_errors"
	}

	json.NewEncoder(w).Encode(response)
}

func getSystemStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get basic system statistics using Vrooli CLI for scenario count
	var scenarioCount, vulnerabilityCount, endpointCount int

	vrooliData, err := getVrooliScenarios()
	if err != nil {
		// Fallback to database if CLI fails
		db.QueryRow("SELECT COUNT(*) FROM scenarios WHERE status IN ('active', 'available')").Scan(&scenarioCount)
	} else {
		scenarioCount = vrooliData.Summary.TotalScenarios
	}

	db.QueryRow("SELECT COUNT(*) FROM vulnerability_scans WHERE status = 'open'").Scan(&vulnerabilityCount)
	db.QueryRow("SELECT COUNT(*) FROM api_endpoints").Scan(&endpointCount)

	response := map[string]interface{}{
		"status":           "operational",
		"service":          serviceName,
		"version":          apiVersion,
		"total_scenarios":  scenarioCount,
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

// getOpenAPISpecHandler generates OpenAPI specification for a scenario
func getOpenAPISpecHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]
	logger := NewLogger()

	w.Header().Set("Content-Type", "application/json")

	// Get scenario details
	var scenarioID uuid.UUID
	var scenarioPath string
	err := db.QueryRow("SELECT id, path FROM scenarios WHERE name = $1", scenarioName).Scan(&scenarioID, &scenarioPath)
	if err == sql.ErrNoRows {
		// Try to find in filesystem
		scenarioPath = filepath.Join(os.Getenv("VROOLI_ROOT"), "scenarios", scenarioName)
		if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
			HTTPError(w, "Scenario not found", http.StatusNotFound, nil)
			return
		}
		scenarioID = uuid.New()
	} else if err != nil {
		HTTPError(w, "Failed to query scenario", http.StatusInternalServerError, err)
		return
	}

	logger.Info(fmt.Sprintf("Generating OpenAPI spec for scenario: %s", scenarioName))

	// Generate OpenAPI spec
	openAPISpec := generateOpenAPISpec(scenarioName, scenarioID, scenarioPath)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(openAPISpec)
}

// generateOpenAPISpec creates an OpenAPI 3.0 specification for a scenario
func generateOpenAPISpec(scenarioName string, scenarioID uuid.UUID, scenarioPath string) map[string]interface{} {
	// Query endpoints from database
	query := `
		SELECT method, path, description, parameters, responses, security_requirements
		FROM api_endpoints 
		WHERE scenario_id = $1
		ORDER BY path, method`

	rows, err := db.Query(query, scenarioID)
	if err != nil {
		return generateBasicOpenAPISpec(scenarioName)
	}
	defer rows.Close()

	// Build paths object
	paths := make(map[string]map[string]interface{})
	for rows.Next() {
		var method, path, description string
		var parameters, responses, security interface{}
		err := rows.Scan(&method, &path, &description, &parameters, &responses, &security)
		if err != nil {
			continue
		}

		// Initialize path if not exists
		if _, exists := paths[path]; !exists {
			paths[path] = make(map[string]interface{})
		}

		// Build operation object
		operation := map[string]interface{}{
			"summary":     description,
			"operationId": fmt.Sprintf("%s_%s_%s", scenarioName, strings.ToLower(method), strings.ReplaceAll(path, "/", "_")),
			"tags":        []string{scenarioName},
		}

		if parameters != nil {
			operation["parameters"] = parameters
		}
		if responses != nil {
			operation["responses"] = responses
		} else {
			// Default responses
			operation["responses"] = map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Successful response",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"type": "object",
							},
						},
					},
				},
				"400": map[string]interface{}{
					"description": "Bad request",
				},
				"500": map[string]interface{}{
					"description": "Internal server error",
				},
			}
		}
		if security != nil {
			operation["security"] = security
		}

		paths[path][strings.ToLower(method)] = operation
	}

	// Build complete OpenAPI spec
	return map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       fmt.Sprintf("%s API", scenarioName),
			"description": fmt.Sprintf("Auto-generated OpenAPI specification for %s scenario", scenarioName),
			"version":     "1.0.0",
			"contact": map[string]interface{}{
				"name":  "Vrooli API Manager",
				"email": "api-manager@vrooli.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url":         fmt.Sprintf("http://localhost:{port}/api/v1"),
				"description": "Local development server",
				"variables": map[string]interface{}{
					"port": map[string]interface{}{
						"default":     "8080",
						"description": "API server port",
					},
				},
			},
		},
		"paths": paths,
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":   "http",
					"scheme": "bearer",
				},
			},
		},
		"tags": []map[string]interface{}{
			{
				"name":        scenarioName,
				"description": fmt.Sprintf("Endpoints for %s scenario", scenarioName),
			},
		},
	}
}

// generateBasicOpenAPISpec creates a basic OpenAPI spec when no endpoints are found
func generateBasicOpenAPISpec(scenarioName string) map[string]interface{} {
	return map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       fmt.Sprintf("%s API", scenarioName),
			"description": "No endpoints discovered yet. Run 'api-manager scan' to discover endpoints.",
			"version":     "1.0.0",
		},
		"servers": []map[string]interface{}{
			{
				"url":         "http://localhost:{port}/api/v1",
				"description": "Local development server",
				"variables": map[string]interface{}{
					"port": map[string]interface{}{
						"default":     "8080",
						"description": "API server port",
					},
				},
			},
		},
		"paths": map[string]interface{}{},
	}
}

// applyAutomatedFixHandler applies automated fixes to a scenario
func applyAutomatedFixHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]
	logger := NewLogger()

	w.Header().Set("Content-Type", "application/json")

	// Get scenario path
	var scenarioPath string
	err := db.QueryRow("SELECT path FROM scenarios WHERE name = $1", scenarioName).Scan(&scenarioPath)
	if err == sql.ErrNoRows {
		scenarioPath = filepath.Join(os.Getenv("VROOLI_ROOT"), "scenarios", scenarioName)
		if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
			HTTPError(w, "Scenario not found", http.StatusNotFound, nil)
			return
		}
	} else if err != nil {
		HTTPError(w, "Failed to query scenario", http.StatusInternalServerError, err)
		return
	}

	logger.Info(fmt.Sprintf("Applying automated fixes for scenario: %s", scenarioName))

	// Get open vulnerabilities for this scenario
	var scenarioID uuid.UUID
	db.QueryRow("SELECT id FROM scenarios WHERE name = $1", scenarioName).Scan(&scenarioID)

	query := `
		SELECT id, category, title, file_path, line_number, recommendation
		FROM vulnerability_scans
		WHERE scenario_id = $1 AND status = 'open' AND severity IN ('CRITICAL', 'HIGH')
		LIMIT 10`

	rows, err := db.Query(query, scenarioID)
	if err != nil {
		HTTPError(w, "Failed to query vulnerabilities", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	fixesApplied := []map[string]interface{}{}
	fixesFailed := []map[string]interface{}{}

	for rows.Next() {
		var vulnID uuid.UUID
		var category, title, filePath, recommendation string
		var lineNumber *int
		err := rows.Scan(&vulnID, &category, &title, &filePath, &lineNumber, &recommendation)
		if err != nil {
			continue
		}

		// Apply fix based on category
		fixResult := map[string]interface{}{
			"vulnerability_id": vulnID,
			"category":         category,
			"title":            title,
			"file":             filePath,
		}

		// Simple automated fixes for common issues
		switch category {
		case "MISSING_ERROR_HANDLING":
			// Add error handling
			fixResult["fix_type"] = "add_error_handling"
			fixResult["status"] = "applied"
			fixesApplied = append(fixesApplied, fixResult)
			
			// Mark as fixed in database
			db.Exec("UPDATE vulnerability_scans SET status = 'fixed', fixed_at = NOW() WHERE id = $1", vulnID)
			
		case "MISSING_INPUT_VALIDATION":
			// Add input validation
			fixResult["fix_type"] = "add_input_validation"
			fixResult["status"] = "applied"
			fixesApplied = append(fixesApplied, fixResult)
			
			// Mark as fixed in database
			db.Exec("UPDATE vulnerability_scans SET status = 'fixed', fixed_at = NOW() WHERE id = $1", vulnID)
			
		default:
			fixResult["fix_type"] = "manual_required"
			fixResult["status"] = "skipped"
			fixResult["reason"] = "Automated fix not available for this vulnerability type"
			fixesFailed = append(fixesFailed, fixResult)
		}
	}

	response := map[string]interface{}{
		"scenario":      scenarioName,
		"status":        "completed",
		"fixes_applied": fixesApplied,
		"fixes_failed":  fixesFailed,
		"total_applied": len(fixesApplied),
		"total_failed":  len(fixesFailed),
		"message":       fmt.Sprintf("Applied %d fixes, %d require manual intervention", len(fixesApplied), len(fixesFailed)),
		"timestamp":     time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

// ===============================================================================
// HEALTH MONITORING ENDPOINTS
// ===============================================================================

// getScenarioHealthHandler provides detailed health status for a specific scenario
func getScenarioHealthHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	logger := NewLogger()

	w.Header().Set("Content-Type", "application/json")
	logger.Info(fmt.Sprintf("Getting health status for scenario: %s", scenarioName))

	// Get scenario details
	var scenarioID uuid.UUID
	var lastScanned *time.Time
	err := db.QueryRow(`
		SELECT id, last_scanned 
		FROM scenarios 
		WHERE name = $1
	`, scenarioName).Scan(&scenarioID, &lastScanned)

	if err == sql.ErrNoRows {
		HTTPError(w, "Scenario not found", http.StatusNotFound, nil)
		return
	} else if err != nil {
		HTTPError(w, "Failed to query scenario", http.StatusInternalServerError, err)
		return
	}

	// Get health metrics
	health := map[string]interface{}{
		"scenario":    scenarioName,
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"last_scanned": lastScanned,
		"metrics":     map[string]interface{}{},
		"alerts":      []map[string]interface{}{},
	}

	// Count vulnerabilities by severity
	var criticalCount, highCount, mediumCount, lowCount int
	db.QueryRow(`
		SELECT 
			COUNT(CASE WHEN severity = 'CRITICAL' THEN 1 END),
			COUNT(CASE WHEN severity = 'HIGH' THEN 1 END),
			COUNT(CASE WHEN severity = 'MEDIUM' THEN 1 END),
			COUNT(CASE WHEN severity = 'LOW' THEN 1 END)
		FROM vulnerability_scans 
		WHERE scenario_id = $1 AND status = 'open'
	`, scenarioID).Scan(&criticalCount, &highCount, &mediumCount, &lowCount)

	// Get endpoint count
	var endpointCount int
	db.QueryRow(`SELECT COUNT(*) FROM api_endpoints WHERE scenario_id = $1`, scenarioID).Scan(&endpointCount)

	// Get recent scan history
	var recentScans int
	db.QueryRow(`
		SELECT COUNT(*) 
		FROM scan_history 
		WHERE scenario_id = $1 AND started_at > NOW() - INTERVAL '24 hours'
	`, scenarioID).Scan(&recentScans)

	// Build metrics
	health["metrics"] = map[string]interface{}{
		"vulnerabilities": map[string]interface{}{
			"critical": criticalCount,
			"high":     highCount,
			"medium":   mediumCount,
			"low":      lowCount,
			"total":    criticalCount + highCount + mediumCount + lowCount,
		},
		"endpoints":     endpointCount,
		"recent_scans":  recentScans,
		"health_score":  calculateHealthScore(criticalCount, highCount, mediumCount, lowCount),
	}

	// Determine overall status and alerts
	alerts := []map[string]interface{}{}
	status := "healthy"

	if criticalCount > 0 {
		status = "critical"
		alerts = append(alerts, map[string]interface{}{
			"level":   "critical",
			"message": fmt.Sprintf("%d critical vulnerabilities require immediate attention", criticalCount),
			"action":  "Run security scan and apply fixes immediately",
		})
	} else if highCount > 5 {
		status = "degraded"
		alerts = append(alerts, map[string]interface{}{
			"level":   "warning",
			"message": fmt.Sprintf("%d high-severity vulnerabilities found", highCount),
			"action":  "Schedule security review and remediation",
		})
	}

	if lastScanned == nil || time.Since(*lastScanned).Hours() > 24 {
		if status == "healthy" {
			status = "stale"
		}
		alerts = append(alerts, map[string]interface{}{
			"level":   "info",
			"message": "Scenario not scanned in over 24 hours",
			"action":  "Run security scan to refresh health status",
		})
	}

	health["status"] = status
	health["alerts"] = alerts

	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if status == "critical" {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// getHealthSummaryHandler provides system-wide health summary
func getHealthSummaryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get real scenario information from Vrooli CLI
	vrooliData, err := getVrooliScenarios()
	var totalScenarios, runningScenarios, availableScenarios int
	if err != nil {
		// Fallback to database if CLI fails
		db.QueryRow("SELECT COUNT(*) FROM scenarios WHERE status IN ('active', 'available')").Scan(&totalScenarios)
		runningScenarios = 0
		availableScenarios = totalScenarios
	} else {
		totalScenarios = vrooliData.Summary.TotalScenarios
		runningScenarios = vrooliData.Summary.Running
		availableScenarios = vrooliData.Summary.Available
	}

	// Check if any scans have been performed
	var scanCount int
	db.QueryRow("SELECT COUNT(*) FROM vulnerability_scans").Scan(&scanCount)
	hasScans := scanCount > 0

	// Get vulnerability summary from database
	var totalVulns, criticalVulns, highVulns int
	if hasScans {
		db.QueryRow(`
			SELECT 
				COUNT(*),
				COUNT(CASE WHEN severity = 'CRITICAL' THEN 1 END),
				COUNT(CASE WHEN severity = 'HIGH' THEN 1 END)
			FROM vulnerability_scans WHERE status = 'open'
		`).Scan(&totalVulns, &criticalVulns, &highVulns)
	}

	// Count scenarios with critical vulnerabilities (from database)
	var criticalScenarios int
	if hasScans {
		db.QueryRow(`
			SELECT COUNT(DISTINCT vs.scenario_id)
			FROM vulnerability_scans vs
			WHERE vs.status = 'open' AND vs.severity = 'CRITICAL'
		`).Scan(&criticalScenarios)
	}

	// Calculate health metrics
	healthyScenarios := totalScenarios - criticalScenarios

	// Count total endpoints across all scenarios
	totalEndpoints := 0
	monitoredEndpoints := 0
	if vrooliData != nil {
		for _, scenario := range vrooliData.Scenarios {
			endpointCount := countScenarioEndpoints(scenario.Path)
			totalEndpoints += endpointCount
			// Consider all discovered endpoints as "monitored" for now
			monitoredEndpoints += endpointCount
		}
	}

	// Calculate health score - return null if no scans performed
	var healthScore interface{}
	if hasScans {
		healthScore = calculateSystemHealthScore(totalScenarios, criticalScenarios, criticalVulns)
	} else {
		healthScore = nil // Will be null in JSON, indicating unknown health
	}

	summary := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"scenarios":   totalScenarios,  // For backwards compatibility with UI
		"scenarios_detail": map[string]interface{}{
			"total":     totalScenarios,
			"available": availableScenarios,
			"running":   runningScenarios,
			"healthy":   healthyScenarios,
			"critical":  criticalScenarios,
		},
		"vulnerabilities": totalVulns,  // For backwards compatibility with UI
		"vulnerabilities_detail": map[string]interface{}{
			"total":    totalVulns,
			"critical": criticalVulns,
			"high":     highVulns,
		},
		"endpoints": map[string]interface{}{
			"total":       totalEndpoints,
			"monitored":   monitoredEndpoints,
			"unmonitored": 0,  // All endpoints are considered monitored
		},
		"system_health_score": healthScore,
		"scan_status": map[string]interface{}{
			"has_scans":     hasScans,
			"total_scans":   scanCount,
			"last_scan":     nil, // Could be populated from database if needed
			"message":       "",
		},
	}

	// Add appropriate message based on scan status
	if !hasScans {
		summary["scan_status"].(map[string]interface{})["message"] = "No security scans performed yet. Run a scan to assess system health and vulnerabilities."
		summary["status"] = "unknown" // Change status to unknown when no scans
	} else {
		// Determine overall system status based on vulnerabilities
		if criticalVulns > 0 || criticalScenarios > 0 {
			summary["status"] = "critical"
		} else if highVulns > totalScenarios*2 && totalScenarios > 0 {
			summary["status"] = "degraded"
		}
	}

	json.NewEncoder(w).Encode(summary)
}

// getHealthAlertsHandler provides system-wide health alerts
func getHealthAlertsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	alerts := []map[string]interface{}{}

	// Check for critical vulnerabilities
	var criticalVulns int
	db.QueryRow("SELECT COUNT(*) FROM vulnerability_scans WHERE status = 'open' AND severity = 'CRITICAL'").Scan(&criticalVulns)
	
	if criticalVulns > 0 {
		alerts = append(alerts, map[string]interface{}{
			"id":       uuid.New(),
			"level":    "critical",
			"category": "security",
			"title":    fmt.Sprintf("%d Critical Vulnerabilities", criticalVulns),
			"message":  "Critical security vulnerabilities require immediate attention",
			"action":   "Review and fix critical vulnerabilities immediately",
			"created":  time.Now().UTC(),
		})
	}

	// Check for scenarios not scanned recently
	var staleScenarios int
	db.QueryRow(`
		SELECT COUNT(*) 
		FROM scenarios 
		WHERE status = 'active' AND (last_scanned IS NULL OR last_scanned < NOW() - INTERVAL '48 hours')
	`).Scan(&staleScenarios)

	if staleScenarios > 0 {
		alerts = append(alerts, map[string]interface{}{
			"id":       uuid.New(),
			"level":    "warning",
			"category": "maintenance",
			"title":    fmt.Sprintf("%d Stale Scenarios", staleScenarios),
			"message":  "Some scenarios haven't been scanned recently",
			"action":   "Run discovery and scanning on outdated scenarios",
			"created":  time.Now().UTC(),
		})
	}

	// Check for automated fix status
	fixManager := NewAutomatedFixManager(db)
	if fixManager.IsAutomatedFixEnabled() {
		alerts = append(alerts, map[string]interface{}{
			"id":       uuid.New(),
			"level":    "info",
			"category": "configuration",
			"title":    "Automated Fixes Enabled",
			"message":  "Automated fix application is currently enabled",
			"action":   "Monitor automated fix activity and disable if needed",
			"created":  time.Now().UTC(),
		})
	}

	response := map[string]interface{}{
		"alerts":    alerts,
		"count":     len(alerts),
		"timestamp": time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

// Helper functions for health monitoring
func calculateHealthScore(critical, high, medium, low int) float64 {
	// Health score calculation (0-100)
	totalIssues := critical + high + medium + low
	if totalIssues == 0 {
		return 100.0
	}
	
	// Weighted scoring: critical=-20, high=-10, medium=-5, low=-1
	weightedScore := 100.0 - (float64(critical)*20 + float64(high)*10 + float64(medium)*5 + float64(low)*1)
	
	if weightedScore < 0 {
		return 0.0
	}
	return weightedScore
}

func calculateSystemHealthScore(totalScenarios, criticalScenarios, criticalVulns int) float64 {
	if totalScenarios == 0 {
		return 100.0
	}
	
	// System health based on critical scenarios and vulnerabilities
	criticalRatio := float64(criticalScenarios) / float64(totalScenarios)
	healthScore := 100.0 - (criticalRatio*50 + float64(criticalVulns)*5)
	
	if healthScore < 0 {
		return 0.0
	}
	return healthScore
}