package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	apiVersion  = "1.0.0"
	serviceName = "scenario-auditor"

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
		Logger: log.New(os.Stdout, "[scenario-auditor] ", log.LstdFlags|log.Lshortfile),
	}
}

func (l *Logger) Error(msg string, err error) {
	l.Printf("ERROR: %s: %v", msg, err)
}

func (l *Logger) Info(msg string) {
	l.Printf("INFO: %s", msg)
}

func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	if len(fields) > 0 {
		l.Printf("WARN: %s %+v", msg, fields)
	} else {
		l.Printf("WARN: %s", msg)
	}
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
	ScenarioID           uuid.UUID   `json:"scenario_id"`
	Method               string      `json:"method"`
	Path                 string      `json:"path"`
	HandlerFunction      string      `json:"handler_function,omitempty"`
	LineNumber           *int        `json:"line_number,omitempty"`
	FilePath             string      `json:"file_path,omitempty"`
	Description          string      `json:"description,omitempty"`
	Parameters           interface{} `json:"parameters,omitempty"`
	Responses            interface{} `json:"responses,omitempty"`
	SecurityRequirements interface{} `json:"security_requirements,omitempty"`
	CreatedAt            time.Time   `json:"created_at"`
}

// VulnerabilityScan represents a security vulnerability found during scanning
type VulnerabilityScan struct {
	ID             uuid.UUID  `json:"id"`
	ScenarioID     uuid.UUID  `json:"scenario_id"`
	ScanType       string     `json:"scan_type"`
	Severity       string     `json:"severity"`
	Category       string     `json:"category"`
	Title          string     `json:"title"`
	Description    string     `json:"description,omitempty"`
	FilePath       string     `json:"file_path,omitempty"`
	LineNumber     *int       `json:"line_number,omitempty"`
	CodeSnippet    string     `json:"code_snippet,omitempty"`
	Recommendation string     `json:"recommendation,omitempty"`
	Status         string     `json:"status"`
	FixedAt        *time.Time `json:"fixed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
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

// Global agent manager (used by Claude Fix and Automated Fix features)
var agentManager = NewAgentManager()

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
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start scenario-auditor

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Log startup immediately
	fmt.Fprintf(os.Stderr, "[STARTUP] scenario-auditor main() started at %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(os.Stderr, "[STARTUP] VROOLI_LIFECYCLE_MANAGED check passed\n")

	// Debug: Print all environment variables
	fmt.Fprintf(os.Stderr, "[STARTUP] Environment variables:\n")
	for _, env := range os.Environ() {
		if strings.Contains(env, "POSTGRES") || strings.Contains(env, "API_PORT") || strings.Contains(env, "VROOLI") {
			fmt.Fprintf(os.Stderr, "  %s\n", env)
		}
	}

	logger := NewLogger()
	logger.Info(fmt.Sprintf("Starting %s v%s", serviceName, apiVersion))
	fmt.Fprintf(os.Stderr, "[STARTUP] Logger initialized\n")

	// Initialize database
	fmt.Fprintf(os.Stderr, "[STARTUP] Initializing database...\n")
	var err error
	db, err = initDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[STARTUP] Database initialization FAILED: %v\n", err)
		fmt.Fprintf(os.Stderr, "[STARTUP] Continuing without database - some features will be limited\n")
		db = nil // Ensure db is nil so handlers can check
	} else {
		fmt.Fprintf(os.Stderr, "[STARTUP] Database initialized successfully\n")
		defer db.Close()
	}

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
	// Use enhanced scanner with real vulnerability detection
	api.HandleFunc("/scenarios/scan/jobs/{jobId}", getSecurityScanStatusHandler).Methods("GET")
	api.HandleFunc("/scenarios/scan/jobs/{jobId}/cancel", cancelSecurityScanHandler).Methods("POST")
	api.HandleFunc("/scenarios/{name}/scan", enhancedScanScenarioHandler).Methods("POST")
	api.HandleFunc("/scenarios/{name}/security-audit", securityAuditHandler).Methods("POST")
	api.HandleFunc("/scenarios/{name}/endpoints", getScenarioEndpointsHandler).Methods("GET")

	// Vulnerability management
	// Keep original vulnerabilities endpoint as fallback
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
	api.HandleFunc("/fix/history", getAutomatedFixHistoryHandler).Methods("GET")
	api.HandleFunc("/fix/rollback/{fixId}", rollbackAutomatedFixHandler).Methods("POST")
	api.HandleFunc("/fix/jobs", listAutomatedFixJobsHandler).Methods("GET")
	api.HandleFunc("/fix/jobs/{jobId}", getAutomatedFixJobHandler).Methods("GET")
	api.HandleFunc("/fix/jobs/{jobId}/cancel", cancelAutomatedFixJobHandler).Methods("POST")

	// Standards compliance endpoints
	api.HandleFunc("/standards/check/jobs/{jobId}", getStandardsCheckStatusHandler).Methods("GET")
	api.HandleFunc("/standards/check/jobs/{jobId}/cancel", cancelStandardsCheckHandler).Methods("POST")
	api.HandleFunc("/standards/check/{name}", enhancedStandardsCheckHandler).Methods("POST")
	api.HandleFunc("/standards/violations", getStandardsViolationsHandler).Methods("GET")

	// Claude Fix endpoints
	api.HandleFunc("/claude/fix", triggerClaudeFixHandler).Methods("POST")
	api.HandleFunc("/claude/fix/preview", previewClaudeFixHandler).Methods("POST")
	api.HandleFunc("/claude/fix/{fixId}/status", getClaudeFixStatusHandler).Methods("GET")

	// DEPRECATED: Agent management endpoints (replaced by app-issue-tracker integration)
	// api.HandleFunc("/agents", getAgentsHandler).Methods("GET")
	// api.HandleFunc("/agents", startAgentHandler).Methods("POST")
	// api.HandleFunc("/rules/{ruleId}/agents", startAgentHandler).Methods("POST")
	// api.HandleFunc("/agents/{agentId}/stop", stopAgentHandler).Methods("POST")
	// api.HandleFunc("/agents/{agentId}/logs", getAgentLogsHandler).Methods("GET")

	// NEW: Rules management endpoints for scenario-auditor
	// IMPORTANT: Specific routes must come before parameterized routes to avoid conflicts
	api.HandleFunc("/rules", getRulesHandler).Methods("GET")
	api.HandleFunc("/rules/test-cache", clearTestCacheHandler).Methods("DELETE")
	api.HandleFunc("/rules/test-coverage", getTestCoverageHandler).Methods("GET")
	api.HandleFunc("/rules/categories", getRuleCategoriesHandler).Methods("GET")
	api.HandleFunc("/rules/create", createRuleHandler).Methods("POST")
	api.HandleFunc("/rules/ai/edit/{ruleId}", editRuleWithAIHandler).Methods("POST")
	api.HandleFunc("/rules/report-issue", reportIssueHandler).Methods("POST")
	// Parameterized routes come last
	api.HandleFunc("/rules/{ruleId}", getRuleHandler).Methods("GET")
	api.HandleFunc("/rules/{ruleId}", updateRuleHandler).Methods("PUT")
	api.HandleFunc("/rules/{ruleId}/toggle", toggleRuleHandler).Methods("POST")
	api.HandleFunc("/rules/{ruleId}/test", testRuleHandler).Methods("POST")
	api.HandleFunc("/rules/{ruleId}/scenario-test", testRuleOnScenarioHandler).Methods("POST")
	api.HandleFunc("/rules/{ruleId}/validate", validateRuleHandler).Methods("POST")
	api.HandleFunc("/rules/{ruleId}/test-cache", clearTestCacheHandler).Methods("DELETE")

	// System operations
	api.HandleFunc("/system/discover", discoverScenariosHandler).Methods("POST")
	api.HandleFunc("/system/status", getSystemStatusHandler).Methods("GET")
	api.HandleFunc("/system/validate-lifecycle", validateLifecycleProtectionHandler).Methods("GET")

	// Enable CORS
	uiPort := os.Getenv("UI_PORT")
	if uiPort == "" {
		uiPort = "36224" // Default UI port
	}
	allowedOrigin := fmt.Sprintf("http://localhost:%s", uiPort)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Get port from environment - REQUIRED, no defaults
	fmt.Fprintf(os.Stderr, "[STARTUP] Getting API_PORT from environment...\n")
	port := os.Getenv("API_PORT")
	if port == "" {
		fmt.Fprintf(os.Stderr, "[STARTUP] API_PORT is empty! Environment variables:\n")
		for _, env := range os.Environ() {
			if strings.Contains(env, "PORT") || strings.Contains(env, "VROOLI") {
				fmt.Fprintf(os.Stderr, "  %s\n", env)
			}
		}
		logger.Error("❌ API_PORT environment variable is required", nil)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "[STARTUP] API_PORT=%s\n", port)
	logger.Info(fmt.Sprintf("API endpoints available at: http://localhost:%s/api/v1/", port))

	fmt.Fprintf(os.Stderr, "[STARTUP] Starting HTTP server on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		fmt.Fprintf(os.Stderr, "[STARTUP] HTTP server FAILED to start: %v\n", err)
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
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add RANDOM jitter to prevent thundering herd
		// Using UnixNano for pseudo-randomness (avoids need for rand.Seed)
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(time.Now().UnixNano() % int64(jitterRange))
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
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
		"status":       overallStatus,
		"service":      "scenario-auditor-api",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"readiness":    true,
		"version":      apiVersion,
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

	// Add scenario-auditor specific stats using Vrooli CLI for scenario count
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

	healthResponse["scenario_auditor_stats"] = map[string]interface{}{
		"total_scenarios":      scenarioCount,
		"open_vulnerabilities": vulnerabilityCount,
		"tracked_endpoints":    endpointCount,
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
			"code":      "DATABASE_NOT_INITIALIZED",
			"message":   "Database connection not initialized",
			"category":  "configuration",
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
			"code":      "DATABASE_CONNECTION_FAILED",
			"message":   "Failed to ping database: " + err.Error(),
			"category":  "resource",
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
			"code":      "DATABASE_SCHEMA_CHECK_FAILED",
			"message":   "Failed to verify scenarios table: " + err.Error(),
			"category":  "resource",
			"retryable": true,
		}
	} else if !tableExists {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "DATABASE_SCHEMA_MISSING",
			"message":   "scenarios table does not exist",
			"category":  "configuration",
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
				"code":      "DATABASE_VULN_TABLE_CHECK_FAILED",
				"message":   "Failed to verify vulnerability_scans table: " + err.Error(),
				"category":  "resource",
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
				"code":      "DATABASE_ENDPOINTS_TABLE_CHECK_FAILED",
				"message":   "Failed to verify api_endpoints table: " + err.Error(),
				"category":  "resource",
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

	if openConnections > maxDBConnections*9/10 { // 90% threshold
		if health["status"] == "healthy" {
			health["status"] = "degraded"
		}
		if health["error"] == nil {
			health["error"] = map[string]interface{}{
				"code":      "DATABASE_CONNECTION_POOL_HIGH",
				"message":   fmt.Sprintf("Connection pool usage high: %d/%d", openConnections, maxDBConnections),
				"category":  "resource",
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
	// NOTE: Scanner is currently a placeholder (returns nil) which is expected behavior
	// This should NOT be reported as unhealthy since it's intentional, not a failure
	scanner := NewVulnerabilityScanner(db)
	if scanner == nil {
		health["checks"].(map[string]interface{})["scanner_status"] = "not_implemented"
		health["checks"].(map[string]interface{})["scanner_note"] = "Vulnerability scanner is placeholder - standards enforcement works independently"
	} else {
		health["checks"].(map[string]interface{})["scanner_initialized"] = "ok"
	}

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
			"code":      "CRITICAL_VULNERABILITIES_FOUND",
			"message":   fmt.Sprintf("%d critical vulnerabilities require immediate attention", criticalCount),
			"category":  "security",
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
			"code":      "SCENARIOS_DIR_NOT_ACCESSIBLE",
			"message":   "Cannot access scenarios directory: " + err.Error(),
			"category":  "configuration",
			"retryable": false,
		}
		return health
	} else if !info.IsDir() {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "SCENARIOS_PATH_NOT_DIRECTORY",
			"message":   "Scenarios path exists but is not a directory",
			"category":  "configuration",
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
	testFile := filepath.Join(scenariosDir, ".scenario-auditor-health-check")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "SCENARIOS_DIR_NOT_WRITABLE",
			"message":   "Cannot write to scenarios directory: " + err.Error(),
			"category":  "permission",
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

// Placeholder functions that need to be implemented
func NewVulnerabilityScanner(db *sql.DB) *VulnerabilityScanner {
	// Placeholder - return nil for now, will be implemented when copying scanner
	return nil
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

type VulnerabilityScanner struct {
	// Placeholder - will be implemented when adding new functionality
}

// Placeholder functions for missing handlers - will be implemented
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
			"name":                vrooliScenario.Name,
			"description":         vrooliScenario.Description,
			"status":              vrooliScenario.Status, // "available", "running", etc.
			"version":             vrooliScenario.Version,
			"tags":                vrooliScenario.Tags,
			"path":                vrooliScenario.Path,
			"endpoint_count":      endpointCount,    // Actual count from source code analysis
			"vulnerability_count": 0,                // Default since we don't track this in CLI
			"critical_count":      0,                // Default since we don't track this in CLI
			"last_scan":           nil,              // Not available from CLI
			"created_at":          time.Now().UTC(), // Default
			"updated_at":          time.Now().UTC(), // Default
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

func getScenarioHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
}

func securityAuditHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
}

func getScenarioEndpointsHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
}

func getVulnerabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get scenario filter from query params
	scenario := r.URL.Query().Get("scenario")

	// Get vulnerabilities from in-memory store
	vulnerabilities := vulnStore.GetVulnerabilities(scenario)

	// Convert to the expected format
	response := map[string]interface{}{
		"vulnerabilities": vulnerabilities,
		"count":           len(vulnerabilities),
		"timestamp":       time.Now().UTC(),
	}

	// Add stats if requested
	if r.URL.Query().Get("include_stats") == "true" {
		response["stats"] = vulnStore.GetStats()
	}

	json.NewEncoder(w).Encode(response)
}

func getScenarioVulnerabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
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

func getOpenAPISpecHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
}

func applyAutomatedFixHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
}

func getScenarioHealthHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
}

func getHealthSummaryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get real scenario information from Vrooli CLI
	vrooliData, err := getVrooliScenarios()
	var totalScenarios, runningScenarios, availableScenarios int
	if err != nil {
		// Fallback to database if CLI fails
		if db != nil {
			db.QueryRow("SELECT COUNT(*) FROM scenarios WHERE status IN ('active', 'available')").Scan(&totalScenarios)
		} else {
			totalScenarios = 0
		}
		runningScenarios = 0
		availableScenarios = totalScenarios
	} else if vrooliData != nil {
		totalScenarios = vrooliData.Summary.TotalScenarios
		runningScenarios = vrooliData.Summary.Running
		availableScenarios = vrooliData.Summary.Available
	} else {
		// Fallback if vrooliData is nil
		totalScenarios = 0
		runningScenarios = 0
		availableScenarios = 0
	}

	// Get vulnerability summary from database or in-memory store
	var totalVulns, criticalVulns, highVulns int
	hasScans := false

	if db != nil {
		db.QueryRow("SELECT COUNT(*) FROM vulnerability_scans WHERE status = 'open'").Scan(&totalVulns)
		db.QueryRow("SELECT COUNT(*) FROM vulnerability_scans WHERE status = 'open' AND severity = 'CRITICAL'").Scan(&criticalVulns)
		db.QueryRow("SELECT COUNT(*) FROM vulnerability_scans WHERE status = 'open' AND severity = 'HIGH'").Scan(&highVulns)
		hasScans = totalVulns > 0
	} else {
		// Use in-memory vulnerability store
		vulns := vulnStore.GetVulnerabilities("")
		totalVulns = len(vulns)
		for _, v := range vulns {
			if strings.ToUpper(v.Severity) == "CRITICAL" {
				criticalVulns++
			} else if strings.ToUpper(v.Severity) == "HIGH" {
				highVulns++
			}
		}
		hasScans = totalVulns > 0
	}

	// Gather standards violations from the in-memory store so the UI reflects reality.
	totalViolations := 0
	criticalViolations := 0
	highViolations := 0
	standardsViolations := standardsStore.GetViolations("all")
	if len(standardsViolations) == 0 {
		standardsViolations = standardsStore.GetViolations("")
	}
	if len(standardsViolations) > 0 {
		totalViolations = len(standardsViolations)
		for _, violation := range standardsViolations {
			switch strings.ToUpper(strings.TrimSpace(violation.Severity)) {
			case "CRITICAL":
				criticalViolations++
			case "HIGH":
				highViolations++
			}
		}
	}

	// Count scenarios with critical vulnerabilities
	criticalScenarios := 0
	if hasScans && db != nil {
		db.QueryRow(`
			SELECT COUNT(DISTINCT scenario_id) 
			FROM vulnerability_scans 
			WHERE status = 'open' AND severity = 'CRITICAL'
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
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"scenarios": totalScenarios, // For backwards compatibility with UI
		"scenarios_detail": map[string]interface{}{
			"total":     totalScenarios,
			"available": availableScenarios,
			"running":   runningScenarios,
			"healthy":   healthyScenarios,
			"critical":  criticalScenarios,
		},
		"vulnerabilities": totalVulns, // For backwards compatibility with UI
		"vulnerabilities_detail": map[string]interface{}{
			"total":    totalVulns,
			"critical": criticalVulns,
			"high":     highVulns,
		},
		"standards_violations": totalViolations, // For UI sidebar display
		"standards_violations_detail": map[string]interface{}{
			"total":    totalViolations,
			"critical": criticalViolations,
			"high":     highViolations,
		},
		"endpoints": map[string]interface{}{
			"total":       totalEndpoints,
			"monitored":   monitoredEndpoints,
			"unmonitored": 0, // All endpoints are considered monitored
		},
		"system_health_score": healthScore,
		"scan_status": map[string]interface{}{
			"has_scans":   hasScans,
			"total_scans": totalVulns, // Using total vulnerabilities as proxy for scan activity
			"last_scan":   nil,        // Could be populated from database if needed
			"message":     "",
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func getHealthAlertsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	alerts := []map[string]interface{}{}

	// Check for critical vulnerabilities
	var criticalVulns int
	if db != nil {
		db.QueryRow("SELECT COUNT(*) FROM vulnerability_scans WHERE status = 'open' AND severity = 'CRITICAL'").Scan(&criticalVulns)
	} else {
		// Use in-memory store
		vulns := vulnStore.GetVulnerabilities("")
		for _, v := range vulns {
			if strings.ToUpper(v.Severity) == "CRITICAL" {
				criticalVulns++
			}
		}
	}

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
	var staleScenarioNames []string

	if db != nil {
		// First get the count
		db.QueryRow(`
			SELECT COUNT(*) 
			FROM scenarios 
			WHERE status = 'active' AND (last_scanned IS NULL OR last_scanned < NOW() - INTERVAL '48 hours')
		`).Scan(&staleScenarios)

		// Then get the first 5 scenario names for display
		if staleScenarios > 0 {
			rows, err := db.Query(`
				SELECT name 
				FROM scenarios 
				WHERE status = 'active' AND (last_scanned IS NULL OR last_scanned < NOW() - INTERVAL '48 hours')
				ORDER BY COALESCE(last_scanned, '1970-01-01'::timestamp) ASC
				LIMIT 5
			`)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var name string
					if err := rows.Scan(&name); err == nil {
						staleScenarioNames = append(staleScenarioNames, name)
					}
				}
			}
		}

		// Build the message with scenario names if we found stale scenarios
		if staleScenarios > 0 {
			message := "Scenarios haven't been scanned in 48+ hours"
			if len(staleScenarioNames) > 0 {
				message = fmt.Sprintf("Not scanned recently: %s", strings.Join(staleScenarioNames, ", "))
				if staleScenarios > 5 {
					message += fmt.Sprintf(" and %d more", staleScenarios-5)
				}
			}

			alerts = append(alerts, map[string]interface{}{
				"id":          uuid.New(),
				"level":       "warning",
				"category":    "maintenance",
				"title":       fmt.Sprintf("%d Stale Scenarios", staleScenarios),
				"message":     message,
				"action":      "Run discovery and scanning on outdated scenarios",
				"created":     time.Now().UTC(),
				"scenarios":   staleScenarioNames, // Include the list for the UI
				"total_count": staleScenarios,
			})
		}
	}

	response := map[string]interface{}{
		"alerts":    alerts,
		"count":     len(alerts),
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func createPerformanceBaselineHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
}

func getPerformanceMetricsHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
}

func getPerformanceAlertsHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
}

func detectBreakingChangesHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
}

func getChangeHistoryHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
}

func getAutomatedFixConfigHandler(w http.ResponseWriter, r *http.Request) {
	cfg := automatedFixStore.GetConfig()
	response := map[string]interface{}{
		"enabled":            cfg.Enabled,
		"violation_types":    cfg.ViolationTypes,
		"severities":         cfg.Severities,
		"strategy":           cfg.Strategy,
		"loop_delay_seconds": cfg.LoopDelay,
		"timeout_seconds":    cfg.TimeoutSeconds,
		"max_fixes":          cfg.MaxFixes,
		"model":              cfg.Model,
		"updated_at":         cfg.UpdatedAt,
		"safety_status":      computeSafetyStatus(cfg),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger := NewLogger()
		logger.Error("Failed to encode automated fix config", err)
	}
}

func enableAutomatedFixesHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ViolationTypes []string `json:"violation_types"`
		Severities     []string `json:"severities"`
		Strategy       string   `json:"strategy"`
		LoopDelay      *int     `json:"loop_delay_seconds"`
		TimeoutSeconds *int     `json:"timeout_seconds"`
		MaxFixes       *int     `json:"max_fixes"`
		Model          string   `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		HTTPError(w, "Invalid configuration payload", http.StatusBadRequest, err)
		return
	}

	cfg := automatedFixStore.Enable(AutomatedFixConfigInput{
		ViolationTypes: req.ViolationTypes,
		Severities:     req.Severities,
		Strategy:       req.Strategy,
		LoopDelay:      req.LoopDelay,
		TimeoutSeconds: req.TimeoutSeconds,
		MaxFixes:       req.MaxFixes,
		Model:          req.Model,
	})
	summary := triggerAutomatedFixJobs(cfg)
	response := map[string]interface{}{
		"enabled":            cfg.Enabled,
		"violation_types":    cfg.ViolationTypes,
		"severities":         cfg.Severities,
		"strategy":           cfg.Strategy,
		"loop_delay_seconds": cfg.LoopDelay,
		"timeout_seconds":    cfg.TimeoutSeconds,
		"max_fixes":          cfg.MaxFixes,
		"model":              cfg.Model,
		"updated_at":         cfg.UpdatedAt,
		"safety_status":      computeSafetyStatus(cfg),
		"trigger_summary":    summary,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger := NewLogger()
		logger.Error("Failed to encode automated fix enable response", err)
	}
}

func disableAutomatedFixesHandler(w http.ResponseWriter, r *http.Request) {
	cfg := automatedFixStore.Disable()
	automatedFixRunner.RequestStopAll()
	response := map[string]interface{}{
		"enabled":            cfg.Enabled,
		"violation_types":    cfg.ViolationTypes,
		"severities":         cfg.Severities,
		"strategy":           cfg.Strategy,
		"loop_delay_seconds": cfg.LoopDelay,
		"timeout_seconds":    cfg.TimeoutSeconds,
		"max_fixes":          cfg.MaxFixes,
		"model":              cfg.Model,
		"updated_at":         cfg.UpdatedAt,
		"safety_status":      computeSafetyStatus(cfg),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger := NewLogger()
		logger.Error("Failed to encode automated fix disable response", err)
	}
}

func applyAutomatedFixWithSafetyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := strings.TrimSpace(vars["scenario"])
	if scenarioName == "" {
		HTTPError(w, "scenario name is required", http.StatusBadRequest, nil)
		return
	}

	if filepath.IsAbs(scenarioName) || scenarioName == "." || scenarioName == ".." {
		HTTPError(w, "invalid scenario name", http.StatusBadRequest, nil)
		return
	}

	if strings.ContainsAny(scenarioName, "/\\:") {
		HTTPError(w, "scenario name cannot contain path separators", http.StatusBadRequest, nil)
		return
	}

	cfg := automatedFixStore.GetConfig()
	if !cfg.Enabled {
		HTTPError(w, "Automated fixes are currently disabled", http.StatusConflict, nil)
		return
	}

	// Allow per-request overrides without mutating the persistent config.
	var overrides struct {
		ViolationTypes []string `json:"violation_types"`
		Severities     []string `json:"severities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&overrides); err != nil && !errors.Is(err, io.EOF) {
		HTTPError(w, "Invalid override payload", http.StatusBadRequest, err)
		return
	}

	activeTypes := cfg.ViolationTypes
	if len(overrides.ViolationTypes) > 0 {
		if normalised := normaliseViolationTypes(overrides.ViolationTypes); len(normalised) > 0 {
			activeTypes = normalised
		}
	}

	activeSeverities := cfg.Severities
	if len(overrides.Severities) > 0 {
		if normalised := normaliseSeverities(overrides.Severities); len(normalised) > 0 {
			activeSeverities = normalised
		}
	}

	scenarioRoot := getScenarioRoot()
	scenariosDir := filepath.Clean(filepath.Join(scenarioRoot, ".."))
	absBase, err := filepath.Abs(scenariosDir)
	if err != nil {
		HTTPError(w, "Failed to resolve scenarios root", http.StatusInternalServerError, err)
		return
	}

	scenarioPath := filepath.Clean(filepath.Join(absBase, scenarioName))
	if !isSubpath(absBase, scenarioPath) {
		HTTPError(w, "scenario name resolved outside of scenarios directory", http.StatusBadRequest, nil)
		return
	}

	info, err := os.Stat(scenarioPath)
	if err != nil {
		if os.IsNotExist(err) {
			HTTPError(w, fmt.Sprintf("scenario %s not found", scenarioName), http.StatusNotFound, nil)
			return
		}
		HTTPError(w, "Failed to resolve scenario path", http.StatusInternalServerError, err)
		return
	}
	if !info.IsDir() {
		HTTPError(w, fmt.Sprintf("scenario path %s is not a directory", scenarioPath), http.StatusBadRequest, nil)
		return
	}

	safetyStatus := computeSafetyStatus(cfg)

	job, err := automatedFixRunner.Start(AutomatedFixJobOptions{
		Scenario:         scenarioName,
		ActiveTypes:      activeTypes,
		ActiveSeverities: activeSeverities,
		Strategy:         cfg.Strategy,
		LoopDelaySeconds: cfg.LoopDelay,
		TimeoutSeconds:   cfg.TimeoutSeconds,
		MaxFixes:         cfg.MaxFixes,
		Model:            cfg.Model,
	})
	if err != nil {
		HTTPError(w, err.Error(), http.StatusBadRequest, err)
		return
	}

	snapshot := job.Snapshot()
	response := map[string]interface{}{
		"success":       true,
		"scenario":      scenarioName,
		"job":           snapshot,
		"job_id":        snapshot.ID,
		"safety_status": safetyStatus,
		"configuration": map[string]interface{}{
			"violation_types": activeTypes,
			"severities":      activeSeverities,
			"model":           cfg.Model,
		},
		"message": "Automated fix job started",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger := NewLogger()
		logger.Error("Failed to encode automated fix response", err)
	}
}

func getAutomatedFixHistoryHandler(w http.ResponseWriter, r *http.Request) {
	history := automatedFixStore.History()
	response := map[string]interface{}{
		"fixes": history,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger := NewLogger()
		logger.Error("Failed to encode automated fix history", err)
	}
}

func rollbackAutomatedFixHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fixID := strings.TrimSpace(vars["fixId"])
	if fixID == "" {
		HTTPError(w, "fixId is required", http.StatusBadRequest, nil)
		return
	}

	record, err := automatedFixStore.RecordRollback(fixID)
	if err != nil {
		HTTPError(w, err.Error(), http.StatusNotFound, nil)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Fix marked as rolled back. Use git history if you need to undo file changes.",
		"fix":     record,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger := NewLogger()
		logger.Error("Failed to encode rollback response", err)
	}
}

func listAutomatedFixJobsHandler(w http.ResponseWriter, r *http.Request) {
	jobs := automatedFixRunner.List()
	response := map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger := NewLogger()
		logger.Error("Failed to encode automated fix jobs response", err)
	}
}

func getAutomatedFixJobHandler(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(mux.Vars(r)["jobId"])
	if jobID == "" {
		HTTPError(w, "jobId is required", http.StatusBadRequest, nil)
		return
	}
	job, ok := automatedFixRunner.Get(jobID)
	if !ok {
		HTTPError(w, "automation job not found", http.StatusNotFound, nil)
		return
	}
	response := map[string]interface{}{
		"job": job,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger := NewLogger()
		logger.Error("Failed to encode automated fix job response", err)
	}
}

func cancelAutomatedFixJobHandler(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(mux.Vars(r)["jobId"])
	if jobID == "" {
		HTTPError(w, "jobId is required", http.StatusBadRequest, nil)
		return
	}
	if err := automatedFixRunner.Cancel(jobID); err != nil {
		HTTPError(w, err.Error(), http.StatusNotFound, err)
		return
	}
	response := map[string]interface{}{
		"success": true,
		"message": "Automation job cancelled",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger := NewLogger()
		logger.Error("Failed to encode automation cancel response", err)
	}
}

type automationTriggerSummary struct {
	CandidateCount int               `json:"candidate_count"`
	JobsStarted    int               `json:"jobs_started"`
	Skipped        map[string]string `json:"skipped,omitempty"`
}

func triggerAutomatedFixJobs(cfg AutomatedFixConfig) automationTriggerSummary {
	logger := NewLogger()
	summary := automationTriggerSummary{
		Skipped: make(map[string]string),
	}

	severityAllow := make(map[string]struct{}, len(cfg.Severities))
	for _, severity := range cfg.Severities {
		severityAllow[strings.ToLower(strings.TrimSpace(severity))] = struct{}{}
	}

	typeAllow := make(map[string]struct{}, len(cfg.ViolationTypes))
	for _, vt := range cfg.ViolationTypes {
		value := strings.ToLower(strings.TrimSpace(vt))
		if value != "" {
			typeAllow[value] = struct{}{}
		}
	}

	candidates := collectAutomationScenarioCandidates(typeAllow, severityAllow)
	summary.CandidateCount = len(candidates)
	if len(candidates) == 0 {
		logger.Info("Automated fixes enabled but no matching scenarios were found for the current configuration")
		return summary
	}

	logger.Info(fmt.Sprintf("Automated fixes enabled for %d scenario(s); starting background jobs", len(candidates)))

	for _, scenario := range candidates {
		_, err := automatedFixRunner.Start(AutomatedFixJobOptions{
			Scenario:         scenario,
			ActiveTypes:      append([]string(nil), cfg.ViolationTypes...),
			ActiveSeverities: append([]string(nil), cfg.Severities...),
			Strategy:         cfg.Strategy,
			LoopDelaySeconds: cfg.LoopDelay,
			TimeoutSeconds:   cfg.TimeoutSeconds,
			MaxFixes:         cfg.MaxFixes,
			Model:            cfg.Model,
		})
		if err != nil {
			lower := strings.ToLower(err.Error())
			if strings.Contains(lower, "already running") || strings.Contains(lower, "no matching violations") {
				summary.Skipped[scenario] = err.Error()
				continue
			}
			logger.Error(fmt.Sprintf("Failed to start automated fixes for scenario %s", scenario), err)
			summary.Skipped[scenario] = err.Error()
			continue
		}
		summary.JobsStarted++
	}

	if len(summary.Skipped) == 0 {
		summary.Skipped = nil
	}

	return summary
}

func collectAutomationScenarioCandidates(typeAllow map[string]struct{}, severityAllow map[string]struct{}) []string {
	if len(typeAllow) == 0 {
		return nil
	}

	seen := make(map[string]struct{})

	if _, ok := typeAllow["standards"]; ok {
		for _, scenario := range standardsStore.ListScenarios() {
			if len(filterStandardsBySeverity(scenario, severityAllow)) > 0 {
				seen[scenario] = struct{}{}
			}
		}
	}

	if _, ok := typeAllow["security"]; ok {
		for _, scenario := range vulnStore.ListScenarios() {
			if len(filterVulnerabilitiesBySeverity(scenario, severityAllow)) > 0 {
				seen[scenario] = struct{}{}
			}
		}
	}

	if len(seen) == 0 {
		return nil
	}

	result := make([]string, 0, len(seen))
	for scenario := range seen {
		result = append(result, scenario)
	}
	sort.Strings(result)
	return result
}

func containsViolationType(types []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, value := range types {
		if strings.ToLower(strings.TrimSpace(value)) == target {
			return true
		}
	}
	return false
}

func isSubpath(base, target string) bool {
	base = filepath.Clean(base)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	if rel == ".." {
		return false
	}
	prefix := ".." + string(os.PathSeparator)
	if strings.HasPrefix(rel, prefix) {
		return false
	}
	return true
}

func filterVulnerabilitiesBySeverity(scenario string, allowed map[string]struct{}) []StoredVulnerability {
	vulnerabilities := vulnStore.GetVulnerabilities(scenario)
	if len(vulnerabilities) == 0 {
		return nil
	}
	var filtered []StoredVulnerability
	for _, vuln := range vulnerabilities {
		severity := strings.ToLower(strings.TrimSpace(vuln.Severity))
		if len(allowed) > 0 {
			if _, ok := allowed[severity]; !ok {
				continue
			}
		}
		filtered = append(filtered, vuln)
	}
	return filtered
}

func collectVulnerabilityIDs(vulnerabilities []StoredVulnerability) []string {
	ids := make([]string, 0, len(vulnerabilities))
	seen := make(map[string]struct{}, len(vulnerabilities))
	for _, vuln := range vulnerabilities {
		id := strings.TrimSpace(vuln.ID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func extractVulnerabilitySeverities(vulnerabilities []StoredVulnerability) []string {
	if len(vulnerabilities) == 0 {
		return nil
	}
	result := make([]string, 0, len(vulnerabilities))
	for _, vuln := range vulnerabilities {
		result = append(result, vuln.Severity)
	}
	return result
}

func filterStandardsBySeverity(scenario string, allowed map[string]struct{}) []StandardsViolation {
	violations := standardsStore.GetViolations(scenario)
	if len(violations) == 0 {
		return nil
	}
	var filtered []StandardsViolation
	for _, violation := range violations {
		severity := strings.ToLower(strings.TrimSpace(violation.Severity))
		if len(allowed) > 0 {
			if _, ok := allowed[severity]; !ok {
				continue
			}
		}
		filtered = append(filtered, violation)
	}
	return filtered
}

func collectStandardsIDs(violations []StandardsViolation) []string {
	ids := make([]string, 0, len(violations))
	seen := make(map[string]struct{}, len(violations))
	for _, violation := range violations {
		id := strings.TrimSpace(violation.ID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func extractStandardsSeverities(violations []StandardsViolation) []string {
	if len(violations) == 0 {
		return nil
	}
	result := make([]string, 0, len(violations))
	for _, violation := range violations {
		result = append(result, violation.Severity)
	}
	return result
}

func discoverScenariosHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
}

func getSystemStatusHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
}

func validateLifecycleProtectionHandler(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, "Not implemented yet", http.StatusNotImplemented, nil)
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
	// Round to 1 decimal place
	return math.Round(weightedScore*10) / 10
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
	// Round to 1 decimal place
	return math.Round(healthScore*10) / 10
}
