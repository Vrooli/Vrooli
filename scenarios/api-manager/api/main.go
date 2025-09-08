package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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

	// Get port from environment
	port := getEnv("API_PORT", getEnv("PORT", defaultPort))
	logger.Info(fmt.Sprintf("API endpoints available at: http://localhost:%s/api/v1/", port))

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func initDB() (*sql.DB, error) {
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbPort := getEnv("POSTGRES_PORT", "5433")
	dbName := getEnv("POSTGRES_DB", "vrooli_api_manager")
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPassword := getEnv("POSTGRES_PASSWORD", "")

	connStr := fmt.Sprintf("host=%s port=%s dbname=%s user=%s sslmode=disable",
		dbHost, dbPort, dbName, dbUser)
	
	if dbPassword != "" {
		connStr += " password=" + dbPassword
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(maxDBConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxLifetime(connMaxLifetime)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	health := map[string]interface{}{
		"status":    "ok",
		"service":   serviceName,
		"version":   apiVersion,
		"timestamp": time.Now().UTC(),
		"database":  "connected",
	}

	// Test database connection
	if err := db.Ping(); err != nil {
		health["status"] = "error"
		health["database"] = "disconnected"
		health["error"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(health)
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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}