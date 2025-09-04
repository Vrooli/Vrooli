package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	// API version
	apiVersion  = "1.0.0"
	serviceName = "app-personalizer"

	// Defaults
	defaultPort = "8300"

	// Timeouts
	httpTimeout = 30 * time.Second

	// Database limits
	maxDBConnections  = 25
	maxIdleConnections = 5
	connMaxLifetime   = 5 * time.Minute
)

// Logger provides structured logging
type Logger struct {
	*log.Logger
}

// NewLogger creates a structured logger
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[app-personalizer-api] ", log.LstdFlags|log.Lshortfile),
	}
}

func (l *Logger) Error(msg string, err error) {
	l.Printf("ERROR: %s: %v", msg, err)
}

func (l *Logger) Warn(msg string, err error) {
	l.Printf("WARN: %s: %v", msg, err)
}

func (l *Logger) Info(msg string) {
	l.Printf("INFO: %s", msg)
}

// HTTPError sends structured error response
func HTTPError(w http.ResponseWriter, message string, statusCode int, err error) {
	logger := NewLogger()
	logger.Error(fmt.Sprintf("HTTP %d: %s", statusCode, message), err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := map[string]interface{}{
		"error":     message,
		"status":    statusCode,
		"timestamp": time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(errorResp)
}

// AppRegistry represents registered apps for personalization
type AppRegistry struct {
	ID                       uuid.UUID              `json:"id"`
	AppName                  string                 `json:"app_name"`
	AppPath                  string                 `json:"app_path"`
	AppType                  string                 `json:"app_type"`
	Framework                string                 `json:"framework"`
	Version                  string                 `json:"version"`
	PersonalizationPoints    map[string]interface{} `json:"personalization_points"`
	SupportedPersonalizations []string               `json:"supported_personalizations"`
	LastAnalyzed             *time.Time             `json:"last_analyzed,omitempty"`
	CreatedAt                time.Time              `json:"created_at"`
}

// Personalization represents a personalization record
type Personalization struct {
	ID                   uuid.UUID              `json:"id"`
	AppID                uuid.UUID              `json:"app_id"`
	PersonaID            *uuid.UUID             `json:"persona_id,omitempty"`
	BrandID              *uuid.UUID             `json:"brand_id,omitempty"`
	PersonalizationName  string                 `json:"personalization_name"`
	Description          string                 `json:"description"`
	DeploymentMode       string                 `json:"deployment_mode"`
	Modifications        map[string]interface{} `json:"modifications"`
	OriginalAppPath      string                 `json:"original_app_path"`
	PersonalizedAppPath  string                 `json:"personalized_app_path"`
	BackupPath           string                 `json:"backup_path"`
	Status               string                 `json:"status"`
	ValidationResults    map[string]interface{} `json:"validation_results,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	AppliedAt            *time.Time             `json:"applied_at,omitempty"`
}

// AppPersonalizerService handles app personalization operations
type AppPersonalizerService struct {
	db         *sql.DB
	n8nBaseURL string
	minioURL   string
	httpClient *http.Client
	logger     *Logger
}

// NewAppPersonalizerService creates a new app personalizer service
func NewAppPersonalizerService(db *sql.DB, n8nURL, minioURL string) *AppPersonalizerService {
	return &AppPersonalizerService{
		db:         db,
		n8nBaseURL: n8nURL,
		minioURL:   minioURL,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		logger: NewLogger(),
	}
}

// RegisterAppRequest represents app registration request
type RegisterAppRequest struct {
	AppName   string `json:"app_name"`
	AppPath   string `json:"app_path"`
	AppType   string `json:"app_type"`
	Framework string `json:"framework"`
	Version   string `json:"version,omitempty"`
}

// RegisterApp registers an app for personalization
func (s *AppPersonalizerService) RegisterApp(w http.ResponseWriter, r *http.Request) {
	var req RegisterAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}

	if req.AppName == "" || req.AppPath == "" || req.AppType == "" || req.Framework == "" {
		HTTPError(w, "Missing required fields: app_name, app_path, app_type, framework", http.StatusBadRequest, nil)
		return
	}

	// Check if app path exists
	if _, err := os.Stat(req.AppPath); os.IsNotExist(err) {
		HTTPError(w, fmt.Sprintf("App path does not exist: %s", req.AppPath), http.StatusBadRequest, err)
		return
	}

	// Insert app registry
	appID := uuid.New()
	_, err := s.db.Exec(`
		INSERT INTO app_registry (id, app_name, app_path, app_type, framework, version)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		appID, req.AppName, req.AppPath, req.AppType, req.Framework, req.Version)

	if err != nil {
		HTTPError(w, "Failed to register app", http.StatusInternalServerError, err)
		return
	}

	s.logger.Info(fmt.Sprintf("Registered app: %s (%s)", req.AppName, appID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"app_id":     appID,
		"message":    "App registered successfully",
		"timestamp": time.Now().UTC(),
	})
}

// ListApps returns all registered apps
func (s *AppPersonalizerService) ListApps(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT id, app_name, app_path, app_type, framework, version, 
		       personalization_points, supported_personalizations, last_analyzed, created_at
		FROM app_registry
		ORDER BY created_at DESC`)
	if err != nil {
		HTTPError(w, "Failed to query apps", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var apps []AppRegistry
	for rows.Next() {
		var app AppRegistry
		var personalizationPoints, supportedPersonalizations sql.NullString

		err := rows.Scan(&app.ID, &app.AppName, &app.AppPath, &app.AppType, &app.Framework,
			&app.Version, &personalizationPoints, &supportedPersonalizations,
			&app.LastAnalyzed, &app.CreatedAt)
		if err != nil {
			continue
		}

		// Parse JSON fields
		if personalizationPoints.Valid {
			json.Unmarshal([]byte(personalizationPoints.String), &app.PersonalizationPoints)
		}
		if supportedPersonalizations.Valid {
			json.Unmarshal([]byte(supportedPersonalizations.String), &app.SupportedPersonalizations)
		}

		apps = append(apps, app)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apps)
}

// AnalyzeAppRequest represents app analysis request
type AnalyzeAppRequest struct {
	AppID uuid.UUID `json:"app_id"`
}

// AnalyzeApp analyzes an app for personalization points
func (s *AppPersonalizerService) AnalyzeApp(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}

	// Get app details
	var appPath, framework string
	err := s.db.QueryRow(`
		SELECT app_path, framework FROM app_registry WHERE id = $1`,
		req.AppID).Scan(&appPath, &framework)
	if err != nil {
		HTTPError(w, "App not found", http.StatusNotFound, err)
		return
	}

	// Analyze app structure
	personalizationPoints := s.analyzeAppStructure(appPath, framework)

	// Update app registry with analysis results
	pointsJSON, _ := json.Marshal(personalizationPoints)
	_, err = s.db.Exec(`
		UPDATE app_registry 
		SET personalization_points = $1, last_analyzed = CURRENT_TIMESTAMP
		WHERE id = $2`,
		string(pointsJSON), req.AppID)

	if err != nil {
		s.logger.Error("Failed to update analysis results", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"app_id":               req.AppID,
		"personalization_points": personalizationPoints,
		"analyzed_at":          time.Now().UTC(),
	})
}

// analyzeAppStructure performs static analysis of app structure
func (s *AppPersonalizerService) analyzeAppStructure(appPath, framework string) map[string]interface{} {
	points := map[string]interface{}{
		"ui_theme":     []string{},
		"content":      []string{},
		"branding":     []string{},
		"behavior":     []string{},
		"configuration": []string{},
	}

	// Framework-specific analysis
	switch framework {
	case "react":
		s.analyzeReactApp(appPath, points)
	case "vue":
		s.analyzeVueApp(appPath, points)
	case "next.js":
		s.analyzeNextApp(appPath, points)
	default:
		s.analyzeGenericApp(appPath, points)
	}

	return points
}

func (s *AppPersonalizerService) analyzeReactApp(appPath string, points map[string]interface{}) {
	// Common React personalization points
	potentialFiles := []struct {
		path     string
		category string
	}{
		{"src/styles/theme.js", "ui_theme"},
		{"src/styles/theme.ts", "ui_theme"},
		{"src/theme.js", "ui_theme"},
		{"tailwind.config.js", "ui_theme"},
		{"src/styles/variables.css", "ui_theme"},
		{"src/config/app.js", "configuration"},
		{"src/config/app.ts", "configuration"},
		{"src/data/defaults.js", "content"},
		{"src/data/defaults.ts", "content"},
		{"public/manifest.json", "branding"},
		{"public/favicon.ico", "branding"},
		{"src/config/brand.js", "branding"},
		{"src/services/chatService.js", "behavior"},
		{"src/utils/personality.js", "behavior"},
	}

	for _, file := range potentialFiles {
		fullPath := filepath.Join(appPath, file.path)
		if _, err := os.Stat(fullPath); err == nil {
			if points[file.category] == nil {
				points[file.category] = []string{}
			}
			points[file.category] = append(points[file.category].([]string), file.path)
		}
	}
}

func (s *AppPersonalizerService) analyzeVueApp(appPath string, points map[string]interface{}) {
	// Vue-specific analysis - similar to React but with Vue conventions
	s.analyzeGenericApp(appPath, points)
}

func (s *AppPersonalizerService) analyzeNextApp(appPath string, points map[string]interface{}) {
	// Next.js-specific analysis
	s.analyzeReactApp(appPath, points) // Next.js builds on React
}

func (s *AppPersonalizerService) analyzeGenericApp(appPath string, points map[string]interface{}) {
	// Generic analysis for any app type
	commonFiles := []struct {
		pattern  string
		category string
	}{
		{"**/config.json", "configuration"},
		{"**/settings.json", "configuration"},
		{"**/theme.*", "ui_theme"},
		{"**/styles.*", "ui_theme"},
		{"**/manifest.json", "branding"},
	}

	for _, file := range commonFiles {
		matches, _ := filepath.Glob(filepath.Join(appPath, file.pattern))
		for _, match := range matches {
			relPath, _ := filepath.Rel(appPath, match)
			if points[file.category] == nil {
				points[file.category] = []string{}
			}
			points[file.category] = append(points[file.category].([]string), relPath)
		}
	}
}

// PersonalizeRequest represents personalization request
type PersonalizeRequest struct {
	AppID              uuid.UUID              `json:"app_id"`
	PersonaID          *uuid.UUID             `json:"persona_id,omitempty"`
	BrandID            *uuid.UUID             `json:"brand_id,omitempty"`
	PersonalizationType string                 `json:"personalization_type"`
	DeploymentMode     string                 `json:"deployment_mode"`
	TwinAPIToken       string                 `json:"twin_api_token,omitempty"`
	BrandAPIKey        string                 `json:"brand_api_key,omitempty"`
	CustomModifications map[string]interface{} `json:"custom_modifications,omitempty"`
}

// PersonalizeApp initiates app personalization through n8n workflow
func (s *AppPersonalizerService) PersonalizeApp(w http.ResponseWriter, r *http.Request) {
	var req PersonalizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}

	// Get app details
	var appPath, appName string
	err := s.db.QueryRow(`
		SELECT app_path, app_name FROM app_registry WHERE id = $1`,
		req.AppID).Scan(&appPath, &appName)
	if err != nil {
		HTTPError(w, "App not found", http.StatusNotFound, err)
		return
	}

	// Create personalization record
	personalizationID := uuid.New()
	_, err = s.db.Exec(`
		INSERT INTO personalizations (id, app_id, persona_id, brand_id, personalization_name,
		                            deployment_mode, modifications, original_app_path, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		personalizationID, req.AppID, req.PersonaID, req.BrandID,
		fmt.Sprintf("%s-%s", appName, req.PersonalizationType),
		req.DeploymentMode, "{}", appPath, "pending")

	if err != nil {
		HTTPError(w, "Failed to create personalization record", http.StatusInternalServerError, err)
		return
	}

	// Prepare n8n workflow payload
	workflowPayload := map[string]interface{}{
		"app_id":              req.AppID,
		"app_path":            appPath,
		"personalization_id":  personalizationID,
		"personalization_type": req.PersonalizationType,
		"deployment_mode":     req.DeploymentMode,
	}

	if req.PersonaID != nil {
		workflowPayload["persona_id"] = *req.PersonaID
		workflowPayload["twin_api_token"] = req.TwinAPIToken
	}

	if req.BrandID != nil {
		workflowPayload["brand_id"] = *req.BrandID
		workflowPayload["brand_api_key"] = req.BrandAPIKey
	}

	if req.CustomModifications != nil {
		workflowPayload["custom_modifications"] = req.CustomModifications
	}

	// Trigger n8n personalization workflow
	err = s.triggerPersonalizationWorkflow(workflowPayload)
	if err != nil {
		HTTPError(w, "Failed to trigger personalization workflow", http.StatusInternalServerError, err)
		return
	}

	s.logger.Info(fmt.Sprintf("Started personalization: %s for app %s", personalizationID, req.AppID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"personalization_id": personalizationID,
		"status":            "pending",
		"message":           "Personalization started",
		"timestamp":         time.Now().UTC(),
	})
}

func (s *AppPersonalizerService) triggerPersonalizationWorkflow(payload map[string]interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Post(
		fmt.Sprintf("%s/webhook/personalize", s.n8nBaseURL),
		"application/json",
		bytes.NewBuffer(payloadBytes),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("workflow returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// BackupAppRequest represents backup request
type BackupAppRequest struct {
	AppPath    string `json:"app_path"`
	BackupType string `json:"backup_type"`
}

// BackupApp creates a backup of an app before personalization
func (s *AppPersonalizerService) BackupApp(w http.ResponseWriter, r *http.Request) {
	var req BackupAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}

	if req.AppPath == "" {
		HTTPError(w, "Missing required field: app_path", http.StatusBadRequest, nil)
		return
	}

	backupPath, err := s.createAppBackup(req.AppPath, req.BackupType)
	if err != nil {
		HTTPError(w, "Failed to create backup", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"backup_path": backupPath,
		"backup_type": req.BackupType,
		"timestamp":   time.Now().UTC(),
	})
}

func (s *AppPersonalizerService) createAppBackup(appPath, backupType string) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	appName := filepath.Base(appPath)
	backupName := fmt.Sprintf("%s-%s-%s.tar.gz", appName, backupType, timestamp)
	backupPath := filepath.Join("/tmp/app-backups", backupName)

	// Ensure backup directory exists
	os.MkdirAll(filepath.Dir(backupPath), 0755)

	// Create tar.gz backup
	cmd := exec.Command("tar", "-czf", backupPath, "-C", filepath.Dir(appPath), filepath.Base(appPath))
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// ValidateAppRequest represents validation request
type ValidateAppRequest struct {
	AppPath string   `json:"app_path"`
	Tests   []string `json:"tests"`
}

// ValidateApp runs validation tests on an app
func (s *AppPersonalizerService) ValidateApp(w http.ResponseWriter, r *http.Request) {
	var req ValidateAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}

	if req.AppPath == "" {
		HTTPError(w, "Missing required field: app_path", http.StatusBadRequest, nil)
		return
	}

	results := make(map[string]interface{})

	for _, test := range req.Tests {
		result := s.runValidationTest(req.AppPath, test)
		results[test] = result
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"app_path":           req.AppPath,
		"validation_results": results,
		"timestamp":          time.Now().UTC(),
	})
}

func (s *AppPersonalizerService) runValidationTest(appPath, test string) map[string]interface{} {
	result := map[string]interface{}{
		"success": false,
		"output":  "",
		"error":   "",
	}

	var cmd *exec.Cmd
	switch test {
	case "build":
		cmd = exec.Command("npm", "run", "build")
	case "lint":
		cmd = exec.Command("npm", "run", "lint")
	case "test":
		cmd = exec.Command("npm", "test")
	case "startup":
		// Quick syntax check
		cmd = exec.Command("node", "-c", "package.json")
	default:
		result["error"] = fmt.Sprintf("Unknown test type: %s", test)
		return result
	}

	cmd.Dir = appPath
	output, err := cmd.CombinedOutput()
	
	result["output"] = string(output)
	if err != nil {
		result["error"] = err.Error()
	} else {
		result["success"] = true
	}

	return result
}

// Health endpoint
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": serviceName,
		"version": apiVersion,
	})
}

func main() {
	// Load configuration
	port := getEnv("API_PORT", getEnv("PORT", ""))

	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nURL = "http://localhost:5678"
	}

	minioURL := os.Getenv("MINIO_ENDPOINT")
	if minioURL == "" {
		minioURL = "localhost:9000"
	}

	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/app_personalizer?sslmode=disable"
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger := NewLogger()
		logger.Error("Failed to connect to database", err)
		os.Exit(1)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(maxDBConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxLifetime(connMaxLifetime)

	// Test database connection
	if err := db.Ping(); err != nil {
		logger := NewLogger()
		logger.Error("Failed to ping database", err)
		os.Exit(1)
	}

	log.Println("Connected to database")

	// Initialize service
	service := NewAppPersonalizerService(db, n8nURL, minioURL)

	// Setup routes
	r := mux.NewRouter()

	// API endpoints
	r.HandleFunc("/health", Health).Methods("GET")
	r.HandleFunc("/api/apps", service.ListApps).Methods("GET")
	r.HandleFunc("/api/apps/register", service.RegisterApp).Methods("POST")
	r.HandleFunc("/api/apps/analyze", service.AnalyzeApp).Methods("POST")
	r.HandleFunc("/api/personalize", service.PersonalizeApp).Methods("POST")
	r.HandleFunc("/api/backup", service.BackupApp).Methods("POST")
	r.HandleFunc("/api/validate", service.ValidateApp).Methods("POST")

	// Start server
	log.Printf("Starting App Personalizer API on port %s", port)
	log.Printf("  n8n URL: %s", n8nURL)
	log.Printf("  MinIO URL: %s", minioURL)
	log.Printf("  Database: %s", dbURL)

	logger := NewLogger()
	logger.Info(fmt.Sprintf("Server starting on port %s", port))

	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Error("Server failed", err)
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
