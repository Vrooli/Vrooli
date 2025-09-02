package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	// API version
	apiVersion = "2.0.0"
	serviceName = "task-planner"
	
	// Defaults
	defaultPort = "8090"
	
	// Timeouts
	httpTimeout = 120 * time.Second
	
	// Database limits
	maxDBConnections = 25
	maxIdleConnections = 5
	connMaxLifetime = 5 * time.Minute
)

// Logger provides structured logging
type Logger struct {
	*log.Logger
}

// NewLogger creates a structured logger
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[task-planner-api] ", log.LstdFlags|log.Lshortfile),
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

// TaskPlannerService handles coordination with resources
type TaskPlannerService struct {
	db          *sql.DB
	n8nBaseURL  string
	windmillURL string
	qdrantURL   string
	httpClient  *http.Client
	logger      *Logger
}

// NewTaskPlannerService creates a new service
func NewTaskPlannerService(db *sql.DB, n8nURL, windmillURL, qdrantURL string) *TaskPlannerService {
	return &TaskPlannerService{
		db:          db,
		n8nBaseURL:  n8nURL,
		windmillURL: windmillURL,
		qdrantURL:   qdrantURL,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		logger:      NewLogger(),
	}
}

// Task represents a task from the database
type Task struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	Tags        []string  `json:"tags"`
	AppID       uuid.UUID `json:"app_id"`
	AppName     string    `json:"app_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// App represents an application from the database
type App struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	DisplayName  string    `json:"display_name"`
	Type         string    `json:"type"`
	TotalTasks   int       `json:"total_tasks"`
	CompletedTasks int     `json:"completed_tasks"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ParseTextRequest represents a text parsing request
type ParseTextRequest struct {
	AppID       string `json:"app_id"`
	RawText     string `json:"raw_text"`
	APIToken    string `json:"api_token"`
	InputType   string `json:"input_type"`
	SubmittedBy string `json:"submitted_by"`
}

// TaskRequest represents task operation requests
type TaskRequest struct {
	TaskID              string `json:"task_id,omitempty"`
	ToStatus            string `json:"to_status,omitempty"`
	Reason              string `json:"reason,omitempty"`
	Notes               string `json:"notes,omitempty"`
	ForceRefresh        bool   `json:"force_refresh,omitempty"`
	OverrideStaging     bool   `json:"override_staging,omitempty"`
	ImplementationNotes string `json:"implementation_notes,omitempty"`
}

// Health endpoint
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": serviceName,
		"version": apiVersion,
	})
}

// GetApps returns all applications
func (s *TaskPlannerService) GetApps(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT id, name, display_name, type, total_tasks, completed_tasks, created_at, updated_at
		FROM apps 
		ORDER BY display_name
	`)
	if err != nil {
		HTTPError(w, "Failed to fetch apps", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var apps []App
	for rows.Next() {
		var app App
		err := rows.Scan(&app.ID, &app.Name, &app.DisplayName, &app.Type, 
			&app.TotalTasks, &app.CompletedTasks, &app.CreatedAt, &app.UpdatedAt)
		if err != nil {
			s.logger.Error("Failed to scan app row", err)
			continue
		}
		apps = append(apps, app)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"apps":  apps,
		"count": len(apps),
	})
}

// GetTasks returns tasks with optional filtering
func (s *TaskPlannerService) GetTasks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	appID := query.Get("app_id")
	status := query.Get("status")
	priority := query.Get("priority")
	limit := query.Get("limit")
	offset := query.Get("offset")

	if limit == "" {
		limit = "50"
	}
	if offset == "" {
		offset = "0"
	}

	whereClause := "1=1"
	var args []interface{}
	argIndex := 1

	if appID != "" {
		whereClause += fmt.Sprintf(" AND t.app_id = $%d", argIndex)
		args = append(args, appID)
		argIndex++
	}

	if status != "" {
		whereClause += fmt.Sprintf(" AND t.status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if priority != "" {
		whereClause += fmt.Sprintf(" AND t.priority = $%d", argIndex)
		args = append(args, priority)
		argIndex++
	}

	args = append(args, limit, offset)

	sqlQuery := fmt.Sprintf(`
		SELECT t.id, t.title, t.description, t.status, t.priority, t.tags,
		       t.app_id, a.display_name as app_name, t.created_at, t.updated_at
		FROM tasks t
		JOIN apps a ON t.app_id = a.id
		WHERE %s
		ORDER BY t.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		HTTPError(w, "Failed to fetch tasks", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var tagsStr sql.NullString
		
		err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Status, 
			&task.Priority, &tagsStr, &task.AppID, &task.AppName, 
			&task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			s.logger.Error("Failed to scan task row", err)
			continue
		}

		// Parse tags array
		if tagsStr.Valid {
			task.Tags = strings.Split(strings.Trim(tagsStr.String, "{}"), ",")
		}

		tasks = append(tasks, task)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	})
}

// ParseText delegates text parsing to n8n workflow
func (s *TaskPlannerService) ParseText(w http.ResponseWriter, r *http.Request) {
	var req ParseTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}

	if req.AppID == "" || req.RawText == "" || req.APIToken == "" {
		HTTPError(w, "Missing required fields: app_id, raw_text, api_token", http.StatusBadRequest, nil)
		return
	}

	// Set defaults
	if req.InputType == "" {
		req.InputType = "markdown"
	}
	if req.SubmittedBy == "" {
		req.SubmittedBy = "api"
	}

	// Call n8n text parser workflow
	webhookURL := fmt.Sprintf("%s/webhook/task-parser", s.n8nBaseURL)
	reqBody, _ := json.Marshal(req)

	resp, err := s.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		HTTPError(w, "Failed to call text parser workflow", http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		HTTPError(w, "Text parser workflow failed", resp.StatusCode, nil)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		HTTPError(w, "Failed to decode workflow response", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ResearchTask delegates task research to n8n workflow
func (s *TaskPlannerService) ResearchTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}

	// Call n8n research workflow
	webhookURL := fmt.Sprintf("%s/webhook/task-researcher", s.n8nBaseURL)
	payload := map[string]interface{}{
		"task_id":       taskID,
		"force_refresh": req.ForceRefresh,
	}

	reqBody, _ := json.Marshal(payload)
	resp, err := s.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		HTTPError(w, "Failed to call research workflow", http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		HTTPError(w, "Research workflow failed", resp.StatusCode, nil)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		HTTPError(w, "Failed to decode workflow response", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ImplementTask delegates task implementation to n8n workflow
func (s *TaskPlannerService) ImplementTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}

	// Call n8n implementation workflow
	webhookURL := fmt.Sprintf("%s/webhook/task-implementer", s.n8nBaseURL)
	payload := map[string]interface{}{
		"task_id":               taskID,
		"override_staging":      req.OverrideStaging,
		"implementation_notes":  req.ImplementationNotes,
	}

	reqBody, _ := json.Marshal(payload)
	resp, err := s.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		HTTPError(w, "Failed to call implementation workflow", http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		HTTPError(w, "Implementation workflow failed", resp.StatusCode, nil)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		HTTPError(w, "Failed to decode workflow response", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// UpdateTaskStatus delegates status updates to n8n workflow
func (s *TaskPlannerService) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}

	if req.ToStatus == "" {
		HTTPError(w, "Missing required field: to_status", http.StatusBadRequest, nil)
		return
	}

	// Validate status
	validStatuses := []string{"unstructured", "backlog", "staged", "in_progress", "completed", "cancelled", "failed"}
	isValid := false
	for _, status := range validStatuses {
		if req.ToStatus == status {
			isValid = true
			break
		}
	}
	if !isValid {
		HTTPError(w, "Invalid status", http.StatusBadRequest, nil)
		return
	}

	// Set default reason
	if req.Reason == "" {
		req.Reason = "Manual status update via API"
	}

	// Call n8n status update workflow
	webhookURL := fmt.Sprintf("%s/webhook/status-monitor", s.n8nBaseURL)
	payload := map[string]interface{}{
		"task_id":      taskID,
		"to_status":    req.ToStatus,
		"reason":       req.Reason,
		"notes":        req.Notes,
		"triggered_by": "api",
	}

	reqBody, _ := json.Marshal(payload)
	resp, err := s.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		HTTPError(w, "Failed to call status update workflow", http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		HTTPError(w, "Status update workflow failed", resp.StatusCode, nil)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		HTTPError(w, "Failed to decode workflow response", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// getResourcePort queries the port registry for a resource's port
func getResourcePort(resourceName string) string {
	cmd := exec.Command("bash", "-c", fmt.Sprintf(
		"source ${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh && ports::get_resource_port %s",
		resourceName,
	))
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to get port for %s, using default: %v", resourceName, err)
		// Fallback to defaults
		defaults := map[string]string{
			"n8n": "5678",
			"windmill": "5681",
			"postgres": "5433",
			"qdrant": "6333",
		}
		if port, ok := defaults[resourceName]; ok {
			return port
		}
		return "8080" // Generic fallback
	}
	return strings.TrimSpace(string(output))
}

func main() {
	// Load configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("API_PORT")
		if port == "" {
			port = defaultPort
		}
	}
	
	// Use port registry for resource ports
	n8nPort := getResourcePort("n8n")
	windmillPort := getResourcePort("windmill")
	postgresPort := getResourcePort("postgres")
	qdrantPort := getResourcePort("qdrant")
	
	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nURL = fmt.Sprintf("http://localhost:%s", n8nPort)
	}
	
	windmillURL := os.Getenv("WINDMILL_BASE_URL")
	if windmillURL == "" {
		windmillURL = fmt.Sprintf("http://localhost:%s", windmillPort)
	}
	
	qdrantURL := os.Getenv("QDRANT_BASE_URL")
	if qdrantURL == "" {
		qdrantURL = fmt.Sprintf("http://localhost:%s", qdrantPort)
	}
	
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = fmt.Sprintf("postgres://postgres:postgres@localhost:%s/task_planner?sslmode=disable", postgresPort)
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
	service := NewTaskPlannerService(db, n8nURL, windmillURL, qdrantURL)
	
	// Setup routes
	r := mux.NewRouter()
	
	// API endpoints
	r.HandleFunc("/health", Health).Methods("GET")
	r.HandleFunc("/api/apps", service.GetApps).Methods("GET")
	r.HandleFunc("/api/tasks", service.GetTasks).Methods("GET")
	r.HandleFunc("/api/parse-text", service.ParseText).Methods("POST")
	r.HandleFunc("/api/tasks/{taskId}/research", service.ResearchTask).Methods("POST")
	r.HandleFunc("/api/tasks/{taskId}/implement", service.ImplementTask).Methods("POST")
	r.HandleFunc("/api/tasks/{taskId}/status", service.UpdateTaskStatus).Methods("PUT")
	
	// Start server
	log.Printf("Starting Task Planner API on port %s", port)
	log.Printf("  n8n URL: %s", n8nURL)
	log.Printf("  Windmill URL: %s", windmillURL)
	log.Printf("  Qdrant URL: %s", qdrantURL)
	log.Printf("  Database: %s", dbURL)
	
	logger := NewLogger()
	logger.Info(fmt.Sprintf("Server starting on port %s", port))
	
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Error("Server failed", err)
		os.Exit(1)
	}
}