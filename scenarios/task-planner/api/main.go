package main

import (
	"github.com/vrooli/api-core/preflight"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
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
	db         *sql.DB
	qdrantURL  string
	httpClient *http.Client
	logger     *Logger
}

// NewTaskPlannerService creates a new service
func NewTaskPlannerService(db *sql.DB, qdrantURL string) *TaskPlannerService {
	return &TaskPlannerService{
		db:        db,
		qdrantURL: qdrantURL,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		logger:    NewLogger(),
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
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "task-planner",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
	
	// Get Qdrant URL - REQUIRED
	qdrantURL := os.Getenv("QDRANT_BASE_URL")
	if qdrantURL == "" {
		log.Fatal("❌ QDRANT_BASE_URL environment variable is required")
	}
	
	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}
	
	// Connect to database
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		logger := NewLogger()
		logger.Error("Failed to open database connection", err)
		os.Exit(1)
	}
	defer db.Close()
	
	// Configure connection pool
	db.SetMaxOpenConns(maxDBConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxLifetime(connMaxLifetime)
	
	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	
	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📆 Database URL configured")
	
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
		
		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(rand.Float64() * jitterRange)
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
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
	
	// Initialize service
	service := NewTaskPlannerService(db, qdrantURL)
	
	// Setup routes
	r := mux.NewRouter()
	
	// API endpoints
	r.HandleFunc("/health", Health).Methods("GET")
	r.HandleFunc("/api/apps", service.GetApps).Methods("GET")
	r.HandleFunc("/api/tasks", service.GetTasks).Methods("GET")
	r.HandleFunc("/api/parse-text", service.ParseText).Methods("POST")
	r.HandleFunc("/api/tasks/{taskId}/research", service.ResearchTask).Methods("POST")
	r.HandleFunc("/api/tasks/status", service.UpdateTaskStatus).Methods("PUT")
	r.HandleFunc("/api/tasks/status-history", service.GetTaskStatusHistory).Methods("GET")
	
	// Start server
	log.Printf("Starting Task Planner API on port %s", port)
	log.Printf("  Qdrant URL: %s", qdrantURL)
	log.Printf("  Database: Connected")
	
	logger := NewLogger()
	logger.Info(fmt.Sprintf("Server starting on port %s", port))
	
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Error("Server failed", err)
		os.Exit(1)
	}
}

// getEnv removed to prevent hardcoded defaults
