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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
	"github.com/rs/cors"
)

// Schedule represents a workflow schedule
type Schedule struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	CronExpression  string                 `json:"cron_expression"`
	Timezone        string                 `json:"timezone"`
	TargetType      string                 `json:"target_type"`
	TargetURL       string                 `json:"target_url,omitempty"`
	TargetWorkflowID string                `json:"target_workflow_id,omitempty"`
	TargetMethod    string                 `json:"target_method"`
	TargetHeaders   map[string]interface{} `json:"target_headers,omitempty"`
	TargetPayload   map[string]interface{} `json:"target_payload,omitempty"`
	Status          string                 `json:"status"`
	Enabled         bool                   `json:"enabled"`
	OverlapPolicy   string                 `json:"overlap_policy"`
	MaxRetries      int                    `json:"max_retries"`
	RetryStrategy   string                 `json:"retry_strategy"`
	TimeoutSeconds  int                    `json:"timeout_seconds"`
	CatchUpMissed   bool                   `json:"catch_up_missed"`
	Tags            []string               `json:"tags,omitempty"`
	Priority        int                    `json:"priority"`
	Owner           string                 `json:"owner,omitempty"`
	Team            string                 `json:"team,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	LastExecutedAt  *time.Time             `json:"last_executed_at,omitempty"`
	NextExecutionAt *time.Time             `json:"next_execution_at,omitempty"`
}

// Execution represents a schedule execution
type Execution struct {
	ID             string     `json:"id"`
	ScheduleID     string     `json:"schedule_id"`
	ScheduledTime  time.Time  `json:"scheduled_time"`
	StartTime      *time.Time `json:"start_time,omitempty"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	DurationMs     int        `json:"duration_ms,omitempty"`
	Status         string     `json:"status"`
	AttemptCount   int        `json:"attempt_count"`
	ResponseCode   int        `json:"response_code,omitempty"`
	ResponseBody   string     `json:"response_body,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	IsManualTrigger bool      `json:"is_manual_trigger"`
	IsCatchUp      bool       `json:"is_catch_up"`
	TriggeredBy    string     `json:"triggered_by,omitempty"`
}

// ScheduleMetrics represents performance metrics for a schedule
type ScheduleMetrics struct {
	ScheduleID         string     `json:"schedule_id"`
	TotalExecutions    int        `json:"total_executions"`
	SuccessCount       int        `json:"success_count"`
	FailureCount       int        `json:"failure_count"`
	SuccessRate        float64    `json:"success_rate"`
	AvgDurationMs      int        `json:"avg_duration_ms"`
	HealthScore        int        `json:"health_score"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastSuccessAt      *time.Time `json:"last_success_at,omitempty"`
	LastFailureAt      *time.Time `json:"last_failure_at,omitempty"`
}

// CronPreset represents a reusable cron expression
type CronPreset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Expression  string `json:"expression"`
	Category    string `json:"category"`
	IsSystem    bool   `json:"is_system"`
	UsageCount  int    `json:"usage_count"`
}

// App represents the main application
type App struct {
	DB        *sql.DB
	Router    *mux.Router
	Scheduler *Scheduler
}

// Initialize sets up the database connection and routes
func (a *App) Initialize() {
	var err error
	
	// Database configuration - support both POSTGRES_URL and individual components
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		// Try to build from individual components
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Missing database configuration. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}
	
	// Connect to database
	a.DB, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to open database connection:", err)
	}
	
	// Configure connection pool
	a.DB.SetMaxOpenConns(25)
	a.DB.SetMaxIdleConns(5)
	a.DB.SetConnMaxLifetime(5 * time.Minute)
	
	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	
	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Println("📊 Database URL configured")
	
	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = a.DB.Ping()
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
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
	
	log.Println("Connected to PostgreSQL database")
	
	// Initialize scheduler
	a.Scheduler = NewScheduler(a.DB)
	
	// Initialize router
	a.Router = mux.NewRouter()
	a.setRoutes()
}

// setRoutes configures all API routes
func (a *App) setRoutes() {
	// Health check
	a.Router.HandleFunc("/health", a.healthCheck).Methods("GET")
	
	// System status endpoints
	a.Router.HandleFunc("/api/system/db-status", a.dbStatus).Methods("GET")
	a.Router.HandleFunc("/api/system/redis-status", a.redisStatus).Methods("GET")
	
	// Schedule management
	a.Router.HandleFunc("/api/schedules", a.getSchedules).Methods("GET")
	a.Router.HandleFunc("/api/schedules", a.createSchedule).Methods("POST")
	a.Router.HandleFunc("/api/schedules/{id}", a.getSchedule).Methods("GET")
	a.Router.HandleFunc("/api/schedules/{id}", a.updateSchedule).Methods("PUT")
	a.Router.HandleFunc("/api/schedules/{id}", a.deleteSchedule).Methods("DELETE")
	a.Router.HandleFunc("/api/schedules/{id}/enable", a.enableSchedule).Methods("POST")
	a.Router.HandleFunc("/api/schedules/{id}/disable", a.disableSchedule).Methods("POST")
	a.Router.HandleFunc("/api/schedules/{id}/trigger", a.triggerSchedule).Methods("POST")
	
	// Execution management
	a.Router.HandleFunc("/api/executions", a.getExecutions).Methods("GET")
	a.Router.HandleFunc("/api/schedules/{id}/executions", a.getScheduleExecutions).Methods("GET")
	a.Router.HandleFunc("/api/executions/{id}", a.getExecution).Methods("GET")
	a.Router.HandleFunc("/api/executions/{id}/retry", a.retryExecution).Methods("POST")
	
	// Analytics
	a.Router.HandleFunc("/api/schedules/{id}/metrics", a.getScheduleMetrics).Methods("GET")
	a.Router.HandleFunc("/api/schedules/{id}/next-runs", a.getNextRuns).Methods("GET")
	a.Router.HandleFunc("/api/dashboard/stats", a.getDashboardStats).Methods("GET")
	
	// Cron utilities
	a.Router.HandleFunc("/api/cron/validate", a.validateCron).Methods("GET")
	a.Router.HandleFunc("/api/cron/presets", a.getCronPresets).Methods("GET")
	a.Router.HandleFunc("/api/timezones", a.getTimezones).Methods("GET")
	
	// Serve API documentation
	a.Router.HandleFunc("/docs", a.serveDocs).Methods("GET")
}

// Run starts the HTTP server
func (a *App) Run() {
	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
	
	// Start the scheduler
	if err := a.Scheduler.Start(); err != nil {
		log.Fatal("Failed to start scheduler:", err)
	}
	defer a.Scheduler.Stop()
	
	// Configure CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	
	handler := c.Handler(a.Router)
	
	log.Printf("Workflow Scheduler API running on port %s", port)
	log.Printf("API Documentation: http://localhost:%s/docs", port)
	
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

// Removed getEnv function - no defaults allowed

// Health check endpoint
func (a *App) healthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "workflow-scheduler",
		"version": "1.0.0",
		"time":    time.Now().UTC(),
	}
	respondJSON(w, http.StatusOK, response)
}

// Database status endpoint
func (a *App) dbStatus(w http.ResponseWriter, r *http.Request) {
	if err := a.DB.Ping(); err != nil {
		respondError(w, http.StatusServiceUnavailable, "Database unavailable")
		return
	}
	
	var count int
	err := a.DB.QueryRow("SELECT COUNT(*) FROM schedules").Scan(&count)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	response := map[string]interface{}{
		"status":         "healthy",
		"schedule_count": count,
	}
	respondJSON(w, http.StatusOK, response)
}

// Redis status endpoint (placeholder)
func (a *App) redisStatus(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual Redis connectivity check
	response := map[string]interface{}{
		"status": "healthy",
		"info":   "Redis connection check not yet implemented",
	}
	respondJSON(w, http.StatusOK, response)
}

// Get all schedules
func (a *App) getSchedules(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`
		SELECT id, name, description, cron_expression, timezone, 
		       target_type, target_url, status, enabled, priority,
		       owner, team, created_at, updated_at, last_executed_at
		FROM schedules 
		ORDER BY created_at DESC
	`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	
	schedules := []Schedule{}
	for rows.Next() {
		var s Schedule
		var description, targetURL, owner, team sql.NullString
		var lastExecutedAt sql.NullTime
		
		err := rows.Scan(
			&s.ID, &s.Name, &description, &s.CronExpression, &s.Timezone,
			&s.TargetType, &targetURL, &s.Status, &s.Enabled, &s.Priority,
			&owner, &team, &s.CreatedAt, &s.UpdatedAt, &lastExecutedAt,
		)
		if err != nil {
			log.Printf("Error scanning schedule: %v", err)
			continue
		}
		
		s.Description = description.String
		s.TargetURL = targetURL.String
		s.Owner = owner.String
		s.Team = team.String
		if lastExecutedAt.Valid {
			s.LastExecutedAt = &lastExecutedAt.Time
		}
		
		schedules = append(schedules, s)
	}
	
	respondJSON(w, http.StatusOK, schedules)
}

// Create new schedule
func (a *App) createSchedule(w http.ResponseWriter, r *http.Request) {
	var s Schedule
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&s); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()
	
	// Validate cron expression
	if !isValidCron(s.CronExpression) {
		respondError(w, http.StatusBadRequest, "Invalid cron expression")
		return
	}
	
	// Generate UUID if not provided
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	
	// Set defaults
	if s.Timezone == "" {
		s.Timezone = "UTC"
	}
	if s.TargetMethod == "" {
		s.TargetMethod = "POST"
	}
	if s.Status == "" {
		s.Status = "active"
	}
	if s.OverlapPolicy == "" {
		s.OverlapPolicy = "skip"
	}
	if s.Priority == 0 {
		s.Priority = 5
	}
	
	// Insert into database
	_, err := a.DB.Exec(`
		INSERT INTO schedules (
			id, name, description, cron_expression, timezone,
			target_type, target_url, target_method, status, enabled,
			overlap_policy, max_retries, retry_strategy, timeout_seconds,
			catch_up_missed, priority, owner, team
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`,
		s.ID, s.Name, s.Description, s.CronExpression, s.Timezone,
		s.TargetType, s.TargetURL, s.TargetMethod, s.Status, s.Enabled,
		s.OverlapPolicy, s.MaxRetries, s.RetryStrategy, s.TimeoutSeconds,
		s.CatchUpMissed, s.Priority, s.Owner, s.Team,
	)
	
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	// Log audit event
	a.logAuditEvent(s.ID, "created", r.Header.Get("X-User"), r.RemoteAddr)
	
	respondJSON(w, http.StatusCreated, s)
}

// Get single schedule
func (a *App) getSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	var s Schedule
	var description, targetURL, owner, team sql.NullString
	var lastExecutedAt, nextExecutionAt sql.NullTime
	
	err := a.DB.QueryRow(`
		SELECT id, name, description, cron_expression, timezone,
		       target_type, target_url, target_method, status, enabled,
		       overlap_policy, max_retries, retry_strategy, timeout_seconds,
		       catch_up_missed, priority, owner, team,
		       created_at, updated_at, last_executed_at, next_execution_at
		FROM schedules WHERE id = $1
	`, id).Scan(
		&s.ID, &s.Name, &description, &s.CronExpression, &s.Timezone,
		&s.TargetType, &targetURL, &s.TargetMethod, &s.Status, &s.Enabled,
		&s.OverlapPolicy, &s.MaxRetries, &s.RetryStrategy, &s.TimeoutSeconds,
		&s.CatchUpMissed, &s.Priority, &owner, &team,
		&s.CreatedAt, &s.UpdatedAt, &lastExecutedAt, &nextExecutionAt,
	)
	
	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Schedule not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	s.Description = description.String
	s.TargetURL = targetURL.String
	s.Owner = owner.String
	s.Team = team.String
	if lastExecutedAt.Valid {
		s.LastExecutedAt = &lastExecutedAt.Time
	}
	if nextExecutionAt.Valid {
		s.NextExecutionAt = &nextExecutionAt.Time
	}
	
	respondJSON(w, http.StatusOK, s)
}

// Update schedule
func (a *App) updateSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	var s Schedule
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&s); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()
	
	// Validate cron expression if provided
	if s.CronExpression != "" && !isValidCron(s.CronExpression) {
		respondError(w, http.StatusBadRequest, "Invalid cron expression")
		return
	}
	
	// Update in database
	_, err := a.DB.Exec(`
		UPDATE schedules SET
			name = $2, description = $3, cron_expression = $4, timezone = $5,
			target_type = $6, target_url = $7, target_method = $8,
			status = $9, enabled = $10, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`,
		id, s.Name, s.Description, s.CronExpression, s.Timezone,
		s.TargetType, s.TargetURL, s.TargetMethod, s.Status, s.Enabled,
	)
	
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	// Log audit event
	a.logAuditEvent(id, "updated", r.Header.Get("X-User"), r.RemoteAddr)
	
	s.ID = id
	respondJSON(w, http.StatusOK, s)
}

// Delete schedule
func (a *App) deleteSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	result, err := a.DB.Exec("DELETE FROM schedules WHERE id = $1", id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Schedule not found")
		return
	}
	
	// Log audit event
	a.logAuditEvent(id, "deleted", r.Header.Get("X-User"), r.RemoteAddr)
	
	respondJSON(w, http.StatusOK, map[string]string{"result": "success"})
}

// Enable schedule
func (a *App) enableSchedule(w http.ResponseWriter, r *http.Request) {
	a.toggleSchedule(w, r, true)
}

// Disable schedule
func (a *App) disableSchedule(w http.ResponseWriter, r *http.Request) {
	a.toggleSchedule(w, r, false)
}

// Toggle schedule enabled status
func (a *App) toggleSchedule(w http.ResponseWriter, r *http.Request, enabled bool) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	_, err := a.DB.Exec(
		"UPDATE schedules SET enabled = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1",
		id, enabled,
	)
	
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	action := "disabled"
	if enabled {
		action = "enabled"
	}
	
	// Log audit event
	a.logAuditEvent(id, action, r.Header.Get("X-User"), r.RemoteAddr)
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"result":  "success",
		"enabled": enabled,
	})
}

// Manually trigger schedule
func (a *App) triggerSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	// Get schedule details
	schedule, err := a.Scheduler.getScheduleByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "Schedule not found")
		} else {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	
	// Queue manual execution
	execID := uuid.New().String()
	execution := &ScheduleExecution{
		Schedule:      schedule,
		ExecutionID:   execID,
		ScheduledTime: time.Now(),
		IsManual:      true,
		IsCatchUp:     false,
		TriggeredBy:   r.Header.Get("X-User"),
	}
	
	// Send to execution queue
	select {
	case a.Scheduler.executions <- execution:
		// Log audit event
		a.logAuditEvent(id, "triggered", r.Header.Get("X-User"), r.RemoteAddr)
		
		respondJSON(w, http.StatusOK, map[string]string{
			"result":       "success",
			"execution_id": execID,
			"message":      "Schedule triggered successfully",
		})
	case <-time.After(5 * time.Second):
		respondError(w, http.StatusServiceUnavailable, "Execution queue is full, please try again")
	}
}

// Get all executions
func (a *App) getExecutions(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`
		SELECT e.id, e.schedule_id, s.name, e.scheduled_time, 
		       e.start_time, e.end_time, e.duration_ms, e.status,
		       e.response_code, e.is_manual_trigger
		FROM executions e
		JOIN schedules s ON e.schedule_id = s.id
		ORDER BY e.scheduled_time DESC
		LIMIT 100
	`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	
	executions := []map[string]interface{}{}
	for rows.Next() {
		var id, scheduleID, scheduleName, status string
		var scheduledTime time.Time
		var startTime, endTime sql.NullTime
		var durationMs, responseCode sql.NullInt64
		var isManualTrigger bool
		
		err := rows.Scan(
			&id, &scheduleID, &scheduleName, &scheduledTime,
			&startTime, &endTime, &durationMs, &status,
			&responseCode, &isManualTrigger,
		)
		if err != nil {
			log.Printf("Error scanning execution: %v", err)
			continue
		}
		
		exec := map[string]interface{}{
			"id":               id,
			"schedule_id":      scheduleID,
			"schedule_name":    scheduleName,
			"scheduled_time":   scheduledTime,
			"status":           status,
			"is_manual_trigger": isManualTrigger,
		}
		
		if startTime.Valid {
			exec["start_time"] = startTime.Time
		}
		if endTime.Valid {
			exec["end_time"] = endTime.Time
		}
		if durationMs.Valid {
			exec["duration_ms"] = durationMs.Int64
		}
		if responseCode.Valid {
			exec["response_code"] = responseCode.Int64
		}
		
		executions = append(executions, exec)
	}
	
	respondJSON(w, http.StatusOK, executions)
}

// Get executions for specific schedule
func (a *App) getScheduleExecutions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["id"]
	
	rows, err := a.DB.Query(`
		SELECT id, scheduled_time, start_time, end_time, 
		       duration_ms, status, response_code, error_message
		FROM executions
		WHERE schedule_id = $1
		ORDER BY scheduled_time DESC
		LIMIT 50
	`, scheduleID)
	
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	
	executions := []Execution{}
	for rows.Next() {
		var e Execution
		var startTime, endTime sql.NullTime
		var durationMs, responseCode sql.NullInt64
		var errorMessage sql.NullString
		
		err := rows.Scan(
			&e.ID, &e.ScheduledTime, &startTime, &endTime,
			&durationMs, &e.Status, &responseCode, &errorMessage,
		)
		if err != nil {
			log.Printf("Error scanning execution: %v", err)
			continue
		}
		
		e.ScheduleID = scheduleID
		if startTime.Valid {
			e.StartTime = &startTime.Time
		}
		if endTime.Valid {
			e.EndTime = &endTime.Time
		}
		if durationMs.Valid {
			e.DurationMs = int(durationMs.Int64)
		}
		if responseCode.Valid {
			e.ResponseCode = int(responseCode.Int64)
		}
		e.ErrorMessage = errorMessage.String
		
		executions = append(executions, e)
	}
	
	respondJSON(w, http.StatusOK, executions)
}

// Get single execution
func (a *App) getExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	var e Execution
	var startTime, endTime sql.NullTime
	var durationMs, responseCode sql.NullInt64
	var responseBody, errorMessage sql.NullString
	
	err := a.DB.QueryRow(`
		SELECT id, schedule_id, scheduled_time, start_time, end_time,
		       duration_ms, status, attempt_count, response_code,
		       response_body, error_message, is_manual_trigger, is_catch_up
		FROM executions WHERE id = $1
	`, id).Scan(
		&e.ID, &e.ScheduleID, &e.ScheduledTime, &startTime, &endTime,
		&durationMs, &e.Status, &e.AttemptCount, &responseCode,
		&responseBody, &errorMessage, &e.IsManualTrigger, &e.IsCatchUp,
	)
	
	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Execution not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	if startTime.Valid {
		e.StartTime = &startTime.Time
	}
	if endTime.Valid {
		e.EndTime = &endTime.Time
	}
	if durationMs.Valid {
		e.DurationMs = int(durationMs.Int64)
	}
	if responseCode.Valid {
		e.ResponseCode = int(responseCode.Int64)
	}
	e.ResponseBody = responseBody.String
	e.ErrorMessage = errorMessage.String
	
	respondJSON(w, http.StatusOK, e)
}

// Retry failed execution
func (a *App) retryExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	// TODO: Implement actual retry logic via n8n webhook
	
	// Update execution status
	_, err := a.DB.Exec(
		"UPDATE executions SET status = 'pending', attempt_count = attempt_count + 1 WHERE id = $1",
		id,
	)
	
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]string{"result": "success"})
}

// Get schedule metrics
func (a *App) getScheduleMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	var m ScheduleMetrics
	var lastSuccessAt, lastFailureAt sql.NullTime
	
	err := a.DB.QueryRow(`
		SELECT schedule_id, total_executions, success_count, failure_count,
		       success_rate, avg_duration_ms, health_score, consecutive_failures,
		       last_success_at, last_failure_at
		FROM schedule_metrics WHERE schedule_id = $1
	`, id).Scan(
		&m.ScheduleID, &m.TotalExecutions, &m.SuccessCount, &m.FailureCount,
		&m.SuccessRate, &m.AvgDurationMs, &m.HealthScore, &m.ConsecutiveFailures,
		&lastSuccessAt, &lastFailureAt,
	)
	
	if err == sql.ErrNoRows {
		// No metrics yet, return empty metrics
		m = ScheduleMetrics{
			ScheduleID:      id,
			TotalExecutions: 0,
			SuccessRate:     0,
			HealthScore:     100,
		}
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	if lastSuccessAt.Valid {
		m.LastSuccessAt = &lastSuccessAt.Time
	}
	if lastFailureAt.Valid {
		m.LastFailureAt = &lastFailureAt.Time
	}
	
	respondJSON(w, http.StatusOK, m)
}

// Get next N runs for a schedule
func (a *App) getNextRuns(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	// Get schedule details
	var cronExpr, timezone string
	err := a.DB.QueryRow(
		"SELECT cron_expression, timezone FROM schedules WHERE id = $1",
		id,
	).Scan(&cronExpr, &timezone)
	
	if err != nil {
		respondError(w, http.StatusNotFound, "Schedule not found")
		return
	}
	
	// Calculate next runs
	nextRuns := calculateNextRuns(cronExpr, timezone, 5)
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"schedule_id": id,
		"next_runs":   nextRuns,
	})
}

// Get dashboard statistics
func (a *App) getDashboardStats(w http.ResponseWriter, r *http.Request) {
	var stats struct {
		TotalSchedules     int `json:"total_schedules"`
		ActiveSchedules    int `json:"active_schedules"`
		TotalExecutions    int `json:"total_executions"`
		SuccessfulExecutions int `json:"successful_executions"`
		FailedExecutions   int `json:"failed_executions"`
		AvgSuccessRate     float64 `json:"avg_success_rate"`
	}
	
	// Get schedule counts
	a.DB.QueryRow("SELECT COUNT(*) FROM schedules").Scan(&stats.TotalSchedules)
	a.DB.QueryRow("SELECT COUNT(*) FROM schedules WHERE status = 'active' AND enabled = true").Scan(&stats.ActiveSchedules)
	
	// Get execution counts
	a.DB.QueryRow("SELECT COUNT(*) FROM executions").Scan(&stats.TotalExecutions)
	a.DB.QueryRow("SELECT COUNT(*) FROM executions WHERE status = 'success'").Scan(&stats.SuccessfulExecutions)
	a.DB.QueryRow("SELECT COUNT(*) FROM executions WHERE status = 'failed'").Scan(&stats.FailedExecutions)
	
	// Get average success rate
	a.DB.QueryRow("SELECT AVG(success_rate) FROM schedule_metrics").Scan(&stats.AvgSuccessRate)
	
	respondJSON(w, http.StatusOK, stats)
}

// Validate cron expression
func (a *App) validateCron(w http.ResponseWriter, r *http.Request) {
	expression := r.URL.Query().Get("expression")
	if expression == "" {
		respondError(w, http.StatusBadRequest, "Missing expression parameter")
		return
	}
	
	valid := isValidCron(expression)
	nextRuns := []time.Time{}
	
	if valid {
		nextRuns = calculateNextRuns(expression, "UTC", 3)
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"expression": expression,
		"valid":      valid,
		"next_runs":  nextRuns,
	})
}

// Get cron presets
func (a *App) getCronPresets(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`
		SELECT id, name, description, expression, category, is_system
		FROM cron_presets
		ORDER BY category, name
	`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	
	presets := []CronPreset{}
	for rows.Next() {
		var p CronPreset
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, 
			&p.Expression, &p.Category, &p.IsSystem,
		)
		if err != nil {
			log.Printf("Error scanning preset: %v", err)
			continue
		}
		presets = append(presets, p)
	}
	
	respondJSON(w, http.StatusOK, presets)
}

// Get supported timezones
func (a *App) getTimezones(w http.ResponseWriter, r *http.Request) {
	// Return common timezones
	timezones := []map[string]string{
		{"value": "UTC", "label": "UTC"},
		{"value": "America/New_York", "label": "Eastern Time"},
		{"value": "America/Chicago", "label": "Central Time"},
		{"value": "America/Denver", "label": "Mountain Time"},
		{"value": "America/Los_Angeles", "label": "Pacific Time"},
		{"value": "Europe/London", "label": "London"},
		{"value": "Europe/Paris", "label": "Paris"},
		{"value": "Asia/Tokyo", "label": "Tokyo"},
		{"value": "Asia/Shanghai", "label": "Shanghai"},
		{"value": "Australia/Sydney", "label": "Sydney"},
	}
	respondJSON(w, http.StatusOK, timezones)
}

// Serve API documentation
func (a *App) serveDocs(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Workflow Scheduler API Documentation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
        h2 { color: #666; margin-top: 30px; }
        .endpoint { background: #f5f5f5; padding: 10px; margin: 10px 0; border-left: 4px solid #007bff; }
        .method { font-weight: bold; color: #28a745; }
        .path { color: #333; }
        code { background: #f0f0f0; padding: 2px 5px; }
    </style>
</head>
<body>
    <h1>Workflow Scheduler API</h1>
    <p>Enterprise-grade cron scheduling platform for Vrooli workflows</p>
    
    <h2>Schedule Management</h2>
    <div class="endpoint">
        <span class="method">GET</span> <span class="path">/api/schedules</span> - List all schedules
    </div>
    <div class="endpoint">
        <span class="method">POST</span> <span class="path">/api/schedules</span> - Create new schedule
    </div>
    <div class="endpoint">
        <span class="method">GET</span> <span class="path">/api/schedules/{id}</span> - Get schedule details
    </div>
    <div class="endpoint">
        <span class="method">PUT</span> <span class="path">/api/schedules/{id}</span> - Update schedule
    </div>
    <div class="endpoint">
        <span class="method">DELETE</span> <span class="path">/api/schedules/{id}</span> - Delete schedule
    </div>
    
    <h2>Execution Management</h2>
    <div class="endpoint">
        <span class="method">GET</span> <span class="path">/api/executions</span> - List recent executions
    </div>
    <div class="endpoint">
        <span class="method">GET</span> <span class="path">/api/schedules/{id}/executions</span> - Get schedule executions
    </div>
    <div class="endpoint">
        <span class="method">POST</span> <span class="path">/api/schedules/{id}/trigger</span> - Manually trigger schedule
    </div>
    
    <h2>Analytics</h2>
    <div class="endpoint">
        <span class="method">GET</span> <span class="path">/api/schedules/{id}/metrics</span> - Get performance metrics
    </div>
    <div class="endpoint">
        <span class="method">GET</span> <span class="path">/api/dashboard/stats</span> - Get dashboard statistics
    </div>
    
    <h2>Utilities</h2>
    <div class="endpoint">
        <span class="method">GET</span> <span class="path">/api/cron/validate?expression={cron}</span> - Validate cron expression
    </div>
    <div class="endpoint">
        <span class="method">GET</span> <span class="path">/api/cron/presets</span> - Get cron presets
    </div>
    <div class="endpoint">
        <span class="method">GET</span> <span class="path">/api/timezones</span> - Get supported timezones
    </div>
</body>
</html>
	`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

// Helper functions

func (a *App) logAuditEvent(scheduleID, action, user, ip string) {
	_, err := a.DB.Exec(`
		INSERT INTO audit_log (schedule_id, action, performed_by, ip_address)
		VALUES ($1, $2, $3, $4)
	`, scheduleID, action, user, ip)
	if err != nil {
		log.Printf("Failed to log audit event: %v", err)
	}
}

func isValidCron(expression string) bool {
	// Special expressions
	specialExpressions := []string{"@hourly", "@daily", "@weekly", "@monthly", "@yearly"}
	for _, special := range specialExpressions {
		if expression == special {
			return true
		}
	}
	
	// Try to parse as standard cron
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(expression)
	return err == nil
}

func calculateNextRuns(cronExpr, timezone string, count int) []time.Time {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return []time.Time{}
	}
	
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	
	nextRuns := make([]time.Time, 0, count)
	next := time.Now().In(loc)
	
	for i := 0; i < count; i++ {
		next = schedule.Next(next)
		nextRuns = append(nextRuns, next)
	}
	
	return nextRuns
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Error marshalling JSON"}`))
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
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

	app := &App{}
	app.Initialize()
	app.Run()
}
