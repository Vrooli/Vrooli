package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	apiVersion  = "1.0.0"
	serviceName = "visited-tracker"
	defaultPort = "20251"
)

// Logger provides structured logging
type Logger struct {
	*log.Logger
}

func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[visited-tracker-api] ", log.LstdFlags|log.Lshortfile),
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
	}

	json.NewEncoder(w).Encode(errorResp)
}

// Campaign represents a visited tracker campaign
type Campaign struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	Name             string                 `json:"name" db:"name"`
	Description      string                 `json:"description" db:"description"`
	FilePatterns     []string               `json:"file_patterns" db:"file_patterns"`
	WorkingDirectory string                 `json:"working_directory" db:"working_directory"`
	ScenarioName     string                 `json:"scenario_name" db:"scenario_name"`
	Metadata         map[string]interface{} `json:"metadata" db:"metadata"`
	Status           string                 `json:"status" db:"status"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
}

// FileEntry represents a file within a campaign
type FileEntry struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	CampaignID    uuid.UUID              `json:"campaign_id" db:"campaign_id"`
	FilePath      string                 `json:"file_path" db:"file_path"`
	AbsolutePath  string                 `json:"absolute_path" db:"absolute_path"`
	Status        string                 `json:"status" db:"status"`
	ErrorMessage  *string                `json:"error_message,omitempty" db:"error_message"`
	Attempts      int                    `json:"attempts" db:"attempts"`
	LastAttemptAt *time.Time             `json:"last_attempt_at,omitempty" db:"last_attempt_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	Metadata      map[string]interface{} `json:"metadata" db:"metadata"`
}

// CreateCampaignRequest represents campaign creation input
type CreateCampaignRequest struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	FilePatterns     []string               `json:"file_patterns"`
	WorkingDirectory string                 `json:"working_directory"`
	ScenarioName     string                 `json:"scenario_name"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// UpdateFileStatusRequest represents file status update input
type UpdateFileStatusRequest struct {
	Status       string                 `json:"status"`
	ErrorMessage string                 `json:"error_message"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Database connection
var db *sql.DB
var logger *Logger

func initDB() error {
	// Get database connection details from environment
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5433"
	}

	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "vrooli"
	}

	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		return fmt.Errorf("POSTGRES_PASSWORD environment variable is required")
	}

	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		dbname = "vrooli"
	}

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established")

	// Initialize schema
	if err = initSchema(); err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}

func initSchema() error {
	schema := `
	-- Campaigns table: stores campaign metadata and configuration
	CREATE TABLE IF NOT EXISTS campaigns (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		description TEXT,
		file_patterns JSONB NOT NULL,
		working_directory VARCHAR(1000) NOT NULL,
		scenario_name VARCHAR(255),
		metadata JSONB NOT NULL DEFAULT '{}',
		status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed', 'failed')),
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		completed_at TIMESTAMP WITH TIME ZONE
	);

	-- File entries table: tracks individual file status within campaigns
	CREATE TABLE IF NOT EXISTS file_entries (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
		file_path VARCHAR(1000) NOT NULL,
		absolute_path VARCHAR(1000) NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'failed', 'skipped')),
		error_message TEXT,
		attempts INTEGER NOT NULL DEFAULT 0,
		last_attempt_at TIMESTAMP WITH TIME ZONE,
		completed_at TIMESTAMP WITH TIME ZONE,
		metadata JSONB NOT NULL DEFAULT '{}'
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status);
	CREATE INDEX IF NOT EXISTS idx_campaigns_scenario ON campaigns(scenario_name);
	CREATE INDEX IF NOT EXISTS idx_campaigns_created_at ON campaigns(created_at DESC);

	CREATE INDEX IF NOT EXISTS idx_file_entries_campaign_id ON file_entries(campaign_id);
	CREATE INDEX IF NOT EXISTS idx_file_entries_status ON file_entries(status);
	CREATE INDEX IF NOT EXISTS idx_file_entries_campaign_status ON file_entries(campaign_id, status);
	CREATE INDEX IF NOT EXISTS idx_file_entries_file_path ON file_entries(campaign_id, file_path);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	logger.Info("Database schema initialized")
	return nil
}

// resolveFilePatterns expands glob patterns to actual file paths
func resolveFilePatterns(patterns []string, workingDir string) ([]string, error) {
	var files []string

	for _, pattern := range patterns {
		var fullPattern string
		if filepath.IsAbs(pattern) {
			fullPattern = pattern
		} else {
			fullPattern = filepath.Join(workingDir, pattern)
		}

		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
		}

		for _, match := range matches {
			// Only include files, not directories
			if info, err := os.Stat(match); err == nil && !info.IsDir() {
				relPath, err := filepath.Rel(workingDir, match)
				if err != nil {
					relPath = match
				}
				files = append(files, relPath)
			}
		}
	}

	return files, nil
}

// API Handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":    "healthy",
		"service":   serviceName,
		"version":   apiVersion,
		"timestamp": time.Now().UTC(),
		"database":  "connected",
	}

	json.NewEncoder(w).Encode(response)
}

func createCampaignHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON input", http.StatusBadRequest, err)
		return
	}

	// Validate required fields
	if req.Name == "" {
		HTTPError(w, "Campaign name is required", http.StatusBadRequest, nil)
		return
	}

	if len(req.FilePatterns) == 0 {
		HTTPError(w, "At least one file pattern is required", http.StatusBadRequest, nil)
		return
	}

	if req.WorkingDirectory == "" {
		req.WorkingDirectory, _ = os.Getwd()
	}

	// Resolve file patterns to actual files
	files, err := resolveFilePatterns(req.FilePatterns, req.WorkingDirectory)
	if err != nil {
		HTTPError(w, "Failed to resolve file patterns", http.StatusBadRequest, err)
		return
	}

	if len(files) == 0 {
		HTTPError(w, "No files matched the specified patterns", http.StatusBadRequest, nil)
		return
	}

	// Create campaign
	campaignID := uuid.New()
	metadataJSON, _ := json.Marshal(req.Metadata)
	patternsJSON, _ := json.Marshal(req.FilePatterns)

	_, err = db.Exec(`
		INSERT INTO campaigns (id, name, description, file_patterns, working_directory, scenario_name, metadata, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'active', NOW(), NOW())
	`, campaignID, req.Name, req.Description, patternsJSON, req.WorkingDirectory, req.ScenarioName, metadataJSON)

	if err != nil {
		HTTPError(w, "Failed to create campaign", http.StatusInternalServerError, err)
		return
	}

	// Create file entries
	for _, filePath := range files {
		absolutePath := filepath.Join(req.WorkingDirectory, filePath)
		fileID := uuid.New()

		_, err = db.Exec(`
			INSERT INTO file_entries (id, campaign_id, file_path, absolute_path, status, attempts, metadata)
			VALUES ($1, $2, $3, $4, 'pending', 0, '{}')
		`, fileID, campaignID, filePath, absolutePath)

		if err != nil {
			logger.Error("Failed to create file entry", err)
			// Continue with other files
		}
	}

	// Return campaign with file count
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := map[string]interface{}{
		"id":         campaignID,
		"file_count": len(files),
		"campaign": Campaign{
			ID:               campaignID,
			Name:             req.Name,
			Description:      req.Description,
			FilePatterns:     req.FilePatterns,
			WorkingDirectory: req.WorkingDirectory,
			ScenarioName:     req.ScenarioName,
			Metadata:         req.Metadata,
			Status:           "active",
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		},
	}

	json.NewEncoder(w).Encode(response)
	logger.Info(fmt.Sprintf("Created campaign %s with %d files", req.Name, len(files)))
}

func listCampaignsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	status := r.URL.Query().Get("status")
	scenario := r.URL.Query().Get("scenario")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	// Build query
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 0

	if status != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	if scenario != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND scenario_name = $%d", argCount)
		args = append(args, scenario)
	}

	offset := (page - 1) * limit
	argCount++
	limitClause := fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argCount)
	args = append(args, limit)

	argCount++
	limitClause += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, offset)

	query := fmt.Sprintf(`
		SELECT id, name, description, file_patterns, working_directory, scenario_name, 
		       metadata, status, created_at, updated_at, completed_at
		FROM campaigns 
		%s %s
	`, whereClause, limitClause)

	rows, err := db.Query(query, args...)
	if err != nil {
		HTTPError(w, "Failed to query campaigns", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var campaigns []Campaign
	for rows.Next() {
		var c Campaign
		var patternsJSON, metadataJSON string

		err := rows.Scan(&c.ID, &c.Name, &c.Description, &patternsJSON, &c.WorkingDirectory,
			&c.ScenarioName, &metadataJSON, &c.Status, &c.CreatedAt, &c.UpdatedAt, &c.CompletedAt)
		if err != nil {
			logger.Error("Failed to scan campaign row", err)
			continue
		}

		json.Unmarshal([]byte(patternsJSON), &c.FilePatterns)
		json.Unmarshal([]byte(metadataJSON), &c.Metadata)
		campaigns = append(campaigns, c)
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM campaigns %s", whereClause)
	var total int
	db.QueryRow(countQuery, args[:len(args)-2]...).Scan(&total) // exclude limit and offset

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"campaigns": campaigns,
		"total":     total,
		"page":      page,
		"limit":     limit,
	}

	json.NewEncoder(w).Encode(response)
}

func getCampaignHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		HTTPError(w, "Invalid campaign ID", http.StatusBadRequest, err)
		return
	}

	var c Campaign
	var patternsJSON, metadataJSON string

	err = db.QueryRow(`
		SELECT id, name, description, file_patterns, working_directory, scenario_name,
		       metadata, status, created_at, updated_at, completed_at
		FROM campaigns WHERE id = $1
	`, campaignID).Scan(&c.ID, &c.Name, &c.Description, &patternsJSON, &c.WorkingDirectory,
		&c.ScenarioName, &metadataJSON, &c.Status, &c.CreatedAt, &c.UpdatedAt, &c.CompletedAt)

	if err == sql.ErrNoRows {
		HTTPError(w, "Campaign not found", http.StatusNotFound, nil)
		return
	} else if err != nil {
		HTTPError(w, "Failed to get campaign", http.StatusInternalServerError, err)
		return
	}

	json.Unmarshal([]byte(patternsJSON), &c.FilePatterns)
	json.Unmarshal([]byte(metadataJSON), &c.Metadata)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func listFilesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		HTTPError(w, "Invalid campaign ID", http.StatusBadRequest, err)
		return
	}

	// Parse query parameters
	statusFilter := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")

	limit := 1000
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Build query
	whereClause := "WHERE campaign_id = $1"
	args := []interface{}{campaignID}
	argCount := 1

	if statusFilter != "" {
		statuses := strings.Split(statusFilter, ",")
		argCount++
		whereClause += fmt.Sprintf(" AND status = ANY($%d)", argCount)
		args = append(args, statuses)
	}

	argCount++
	query := fmt.Sprintf(`
		SELECT id, campaign_id, file_path, absolute_path, status, error_message,
		       attempts, last_attempt_at, completed_at, metadata
		FROM file_entries %s 
		ORDER BY file_path 
		LIMIT $%d
	`, whereClause, argCount)
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		HTTPError(w, "Failed to query files", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var files []FileEntry
	for rows.Next() {
		var f FileEntry
		var metadataJSON string

		err := rows.Scan(&f.ID, &f.CampaignID, &f.FilePath, &f.AbsolutePath, &f.Status,
			&f.ErrorMessage, &f.Attempts, &f.LastAttemptAt, &f.CompletedAt, &metadataJSON)
		if err != nil {
			logger.Error("Failed to scan file row", err)
			continue
		}

		json.Unmarshal([]byte(metadataJSON), &f.Metadata)
		files = append(files, f)
	}

	// Get total and remaining counts
	var total, remaining int
	db.QueryRow("SELECT COUNT(*) FROM file_entries WHERE campaign_id = $1", campaignID).Scan(&total)
	db.QueryRow("SELECT COUNT(*) FROM file_entries WHERE campaign_id = $1 AND status NOT IN ('completed', 'skipped')", campaignID).Scan(&remaining)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"files":     files,
		"total":     total,
		"remaining": remaining,
	}

	json.NewEncoder(w).Encode(response)
}

func updateFileStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		HTTPError(w, "Invalid campaign ID", http.StatusBadRequest, err)
		return
	}

	fileID, err := uuid.Parse(vars["file_id"])
	if err != nil {
		HTTPError(w, "Invalid file ID", http.StatusBadRequest, err)
		return
	}

	var req UpdateFileStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON input", http.StatusBadRequest, err)
		return
	}

	// Validate status
	validStatuses := []string{"pending", "in_progress", "completed", "failed", "skipped"}
	isValid := false
	for _, status := range validStatuses {
		if req.Status == status {
			isValid = true
			break
		}
	}
	if !isValid {
		HTTPError(w, "Invalid status value", http.StatusBadRequest, nil)
		return
	}

	// Update file entry
	metadataJSON, _ := json.Marshal(req.Metadata)
	now := time.Now()

	var completedAt *time.Time
	if req.Status == "completed" {
		completedAt = &now
	}

	var errorMessage *string
	if req.ErrorMessage != "" {
		errorMessage = &req.ErrorMessage
	}

	_, err = db.Exec(`
		UPDATE file_entries 
		SET status = $1, error_message = $2, last_attempt_at = $3, completed_at = $4, 
		    metadata = $5, attempts = attempts + 1
		WHERE id = $6 AND campaign_id = $7
	`, req.Status, errorMessage, now, completedAt, metadataJSON, fileID, campaignID)

	if err != nil {
		HTTPError(w, "Failed to update file status", http.StatusInternalServerError, err)
		return
	}

	// Get updated file entry
	var f FileEntry
	var metadataStr string
	err = db.QueryRow(`
		SELECT id, campaign_id, file_path, absolute_path, status, error_message,
		       attempts, last_attempt_at, completed_at, metadata
		FROM file_entries WHERE id = $1
	`, fileID).Scan(&f.ID, &f.CampaignID, &f.FilePath, &f.AbsolutePath, &f.Status,
		&f.ErrorMessage, &f.Attempts, &f.LastAttemptAt, &f.CompletedAt, &metadataStr)

	if err != nil {
		HTTPError(w, "Failed to retrieve updated file", http.StatusInternalServerError, err)
		return
	}

	json.Unmarshal([]byte(metadataStr), &f.Metadata)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success": true,
		"file":    f,
	}

	json.NewEncoder(w).Encode(response)
}

func updateFileByPathHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		HTTPError(w, "Invalid campaign ID", http.StatusBadRequest, err)
		return
	}

	filePath := vars["file_path"]
	if filePath == "" {
		HTTPError(w, "File path is required", http.StatusBadRequest, nil)
		return
	}

	var req UpdateFileStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON input", http.StatusBadRequest, err)
		return
	}

	// Find file by path
	var fileID uuid.UUID
	err = db.QueryRow("SELECT id FROM file_entries WHERE campaign_id = $1 AND file_path = $2", campaignID, filePath).Scan(&fileID)
	if err == sql.ErrNoRows {
		HTTPError(w, "File not found in campaign", http.StatusNotFound, nil)
		return
	} else if err != nil {
		HTTPError(w, "Failed to find file", http.StatusInternalServerError, err)
		return
	}

	// Update file entry directly (can't reuse handler since request body is consumed)
	// Validate status
	validStatuses := []string{"pending", "in_progress", "completed", "failed", "skipped"}
	isValid := false
	for _, status := range validStatuses {
		if req.Status == status {
			isValid = true
			break
		}
	}
	if !isValid {
		HTTPError(w, "Invalid status value", http.StatusBadRequest, nil)
		return
	}

	// Update file entry
	metadataJSON, _ := json.Marshal(req.Metadata)
	now := time.Now()

	var completedAt *time.Time
	if req.Status == "completed" {
		completedAt = &now
	}

	var errorMessage *string
	if req.ErrorMessage != "" {
		errorMessage = &req.ErrorMessage
	}

	_, err = db.Exec(`
		UPDATE file_entries 
		SET status = $1, error_message = $2, last_attempt_at = $3, completed_at = $4, 
		    metadata = $5, attempts = attempts + 1
		WHERE id = $6 AND campaign_id = $7
	`, req.Status, errorMessage, now, completedAt, metadataJSON, fileID, campaignID)

	if err != nil {
		HTTPError(w, "Failed to update file status", http.StatusInternalServerError, err)
		return
	}

	// Get updated file entry
	var f FileEntry
	var metadataStr string
	err = db.QueryRow(`
		SELECT id, campaign_id, file_path, absolute_path, status, error_message,
		       attempts, last_attempt_at, completed_at, metadata
		FROM file_entries WHERE id = $1
	`, fileID).Scan(&f.ID, &f.CampaignID, &f.FilePath, &f.AbsolutePath, &f.Status,
		&f.ErrorMessage, &f.Attempts, &f.LastAttemptAt, &f.CompletedAt, &metadataStr)

	if err != nil {
		HTTPError(w, "Failed to retrieve updated file", http.StatusInternalServerError, err)
		return
	}

	json.Unmarshal([]byte(metadataStr), &f.Metadata)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success": true,
		"file":    f,
	}

	json.NewEncoder(w).Encode(response)
}

func main() {
	logger = NewLogger()
	logger.Info(fmt.Sprintf("Starting %s API v%s", serviceName, apiVersion))

	// Initialize database
	if err := initDB(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// Setup router
	r := mux.NewRouter()

	// Health endpoint
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// Campaign endpoints
	r.HandleFunc("/api/v1/campaigns", createCampaignHandler).Methods("POST")
	r.HandleFunc("/api/v1/campaigns", listCampaignsHandler).Methods("GET")
	r.HandleFunc("/api/v1/campaigns/{id}", getCampaignHandler).Methods("GET")

	// File endpoints
	r.HandleFunc("/api/v1/campaigns/{id}/files", listFilesHandler).Methods("GET")
	r.HandleFunc("/api/v1/campaigns/{id}/files/{file_id}/status", updateFileStatusHandler).Methods("PATCH")
	r.HandleFunc("/api/v1/campaigns/{id}/files/by-path/{file_path:.*}/status", updateFileByPathHandler).Methods("PATCH")

	// CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Get port from environment
	port := getEnv("API_PORT", getEnv("PORT", "8080"))
	logger.Info(fmt.Sprintf("API endpoints: http://localhost:%s/api/v1/campaigns", port))

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
