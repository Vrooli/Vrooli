package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// Scheduler manages all scheduled workflows
type Scheduler struct {
	db         *sql.DB
	cron       *cron.Cron
	jobs       map[string]cron.EntryID // schedule_id -> cron_entry_id
	jobsMutex  sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	httpClient *http.Client
	executions chan *ScheduleExecution
	workers    int
}

// ScheduleExecution represents a pending execution
type ScheduleExecution struct {
	Schedule      *Schedule
	ExecutionID   string
	ScheduledTime time.Time
	IsManual      bool
	IsCatchUp     bool
	TriggeredBy   string
	RetryCount    int
}

// NewScheduler creates a new scheduler instance
func NewScheduler(db *sql.DB) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		db:         db,
		cron:       cron.New(), // Standard 5-field cron (minute hour day month weekday)
		jobs:       make(map[string]cron.EntryID),
		ctx:        ctx,
		cancel:     cancel,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		executions: make(chan *ScheduleExecution, 1000),
		workers:    5,
	}
}

// Start begins the scheduler
func (s *Scheduler) Start() error {
	log.Println("Starting workflow scheduler...")
	
	// Start worker pool for executing schedules
	for i := 0; i < s.workers; i++ {
		go s.executionWorker()
	}
	
	// Load active schedules from database
	if err := s.loadSchedules(); err != nil {
		return fmt.Errorf("failed to load schedules: %w", err)
	}
	
	// Start the cron scheduler
	s.cron.Start()
	
	// Start monitoring routine
	go s.monitorRoutine()
	
	// Start missed schedule handler
	go s.missedScheduleHandler()
	
	// Start retry handler
	go s.retryHandler()
	
	log.Printf("Scheduler started with %d active schedules", len(s.jobs))
	return nil
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	log.Println("Stopping workflow scheduler...")
	
	// Stop accepting new jobs
	s.cancel()
	
	// Stop cron scheduler
	ctx := s.cron.Stop()
	<-ctx.Done()
	
	// Wait for pending executions to complete
	close(s.executions)
	
	log.Println("Scheduler stopped")
}

// loadSchedules loads all active schedules from the database
func (s *Scheduler) loadSchedules() error {
	// First check if the schedules table exists
	var tableExists bool
	checkQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'schedules'
		)
	`
	err := s.db.QueryRow(checkQuery).Scan(&tableExists)
	if err != nil || !tableExists {
		log.Println("⚠️ Schedules table does not exist yet - starting with empty scheduler")
		return nil
	}
	
	query := `
		SELECT id, name, cron_expression, timezone, target_type, target_url, 
		       target_workflow_id, target_method, target_headers, target_payload,
		       enabled, overlap_policy, max_retries, retry_strategy, 
		       timeout_seconds, catch_up_missed, priority
		FROM schedules
		WHERE enabled = true AND status = 'active'
	`
	
	rows, err := s.db.Query(query)
	if err != nil {
		log.Printf("⚠️ Error querying schedules: %v - starting with empty scheduler", err)
		return nil
	}
	defer rows.Close()
	
	for rows.Next() {
		var schedule Schedule
		var headers, payload []byte
		var targetWorkflowID sql.NullString

		err := rows.Scan(
			&schedule.ID, &schedule.Name, &schedule.CronExpression,
			&schedule.Timezone, &schedule.TargetType, &schedule.TargetURL,
			&targetWorkflowID, &schedule.TargetMethod,
			&headers, &payload, &schedule.Enabled, &schedule.OverlapPolicy,
			&schedule.MaxRetries, &schedule.RetryStrategy, &schedule.TimeoutSeconds,
			&schedule.CatchUpMissed, &schedule.Priority,
		)
		if err != nil {
			log.Printf("Error loading schedule: %v", err)
			continue
		}

		// Handle nullable target_workflow_id
		if targetWorkflowID.Valid {
			schedule.TargetWorkflowID = targetWorkflowID.String
		}

		// Parse JSON fields
		if headers != nil {
			json.Unmarshal(headers, &schedule.TargetHeaders)
		}
		if payload != nil {
			json.Unmarshal(payload, &schedule.TargetPayload)
		}
		
		// Add to scheduler
		if err := s.AddSchedule(&schedule); err != nil {
			log.Printf("Failed to add schedule %s: %v", schedule.Name, err)
		}
	}
	
	return nil
}

// AddSchedule adds a new schedule to the cron scheduler
func (s *Scheduler) AddSchedule(schedule *Schedule) error {
	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()
	
	// Remove existing schedule if present
	if entryID, exists := s.jobs[schedule.ID]; exists {
		s.cron.Remove(entryID)
	}
	
	// Create cron job
	entryID, err := s.cron.AddFunc(schedule.CronExpression, func() {
		s.executeSchedule(schedule.ID)
	})
	
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	
	s.jobs[schedule.ID] = entryID
	
	// Calculate and update next execution time
	s.updateNextExecutionTime(schedule.ID)
	
	return nil
}

// RemoveSchedule removes a schedule from the cron scheduler
func (s *Scheduler) RemoveSchedule(scheduleID string) {
	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()
	
	if entryID, exists := s.jobs[scheduleID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, scheduleID)
	}
}

// executeSchedule queues a schedule for execution
func (s *Scheduler) executeSchedule(scheduleID string) {
	// Fetch schedule details
	schedule, err := s.getScheduleByID(scheduleID)
	if err != nil {
		log.Printf("Failed to fetch schedule %s: %v", scheduleID, err)
		return
	}
	
	// Check overlap policy
	if schedule.OverlapPolicy == "skip" && s.isScheduleRunning(scheduleID) {
		log.Printf("Skipping execution of %s - already running", schedule.Name)
		return
	}
	
	// Queue execution
	execution := &ScheduleExecution{
		Schedule:      schedule,
		ExecutionID:   uuid.New().String(),
		ScheduledTime: time.Now(),
		IsManual:      false,
		IsCatchUp:     false,
		TriggeredBy:   "scheduler",
	}
	
	select {
	case s.executions <- execution:
		log.Printf("Queued execution for schedule: %s", schedule.Name)
	case <-time.After(5 * time.Second):
		log.Printf("Failed to queue execution for schedule: %s (timeout)", schedule.Name)
	}
}

// executionWorker processes scheduled executions
func (s *Scheduler) executionWorker() {
	for execution := range s.executions {
		s.processExecution(execution)
	}
}

// processExecution executes a scheduled workflow
func (s *Scheduler) processExecution(exec *ScheduleExecution) {
	log.Printf("Executing schedule: %s (ID: %s)", exec.Schedule.Name, exec.ExecutionID)
	
	// Record execution start
	startTime := time.Now()
	s.recordExecutionStart(exec.ExecutionID, exec.Schedule.ID, exec.ScheduledTime, exec.IsManual, exec.IsCatchUp, exec.TriggeredBy)
	
	// Execute based on target type
	var err error
	var responseCode int
	var responseBody string
	
	switch exec.Schedule.TargetType {
	case "http", "webhook":
		responseCode, responseBody, err = s.executeHTTPTarget(exec.Schedule)
	case "n8n_workflow":
		responseCode, responseBody, err = s.executeN8nWorkflow(exec.Schedule)
	case "scenario":
		responseCode, responseBody, err = s.executeScenario(exec.Schedule)
	default:
		err = fmt.Errorf("unsupported target type: %s", exec.Schedule.TargetType)
	}
	
	// Record execution result
	endTime := time.Now()
	duration := int(endTime.Sub(startTime).Milliseconds())
	
	if err != nil {
		// Handle failure
		s.recordExecutionFailure(exec.ExecutionID, duration, err.Error())
		
		// Check if retry is needed
		if exec.RetryCount < exec.Schedule.MaxRetries {
			s.scheduleRetry(exec)
		} else {
			// Max retries reached, send notification
			s.sendFailureNotification(exec.Schedule, err)
		}
	} else {
		// Success
		s.recordExecutionSuccess(exec.ExecutionID, duration, responseCode, responseBody)
		s.updateScheduleMetrics(exec.Schedule.ID, true)
	}
	
	// Update next execution time
	s.updateNextExecutionTime(exec.Schedule.ID)
}

// executeHTTPTarget executes an HTTP/webhook target
func (s *Scheduler) executeHTTPTarget(schedule *Schedule) (int, string, error) {
	// Prepare request
	var body []byte
	if schedule.TargetPayload != nil {
		body, _ = json.Marshal(schedule.TargetPayload)
	}
	
	req, err := http.NewRequestWithContext(s.ctx, schedule.TargetMethod, schedule.TargetURL, bytes.NewBuffer(body))
	if err != nil {
		return 0, "", err
	}
	
	// Add headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range schedule.TargetHeaders {
		req.Header.Set(key, fmt.Sprintf("%v", value))
	}
	
	// Set timeout
	timeout := time.Duration(schedule.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	
	client := &http.Client{Timeout: timeout}
	
	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	
	// Read response
	respBody := make([]byte, 0)
	if resp.ContentLength > 0 && resp.ContentLength < 1024*1024 { // Limit to 1MB
		respBody, _ = io.ReadAll(resp.Body)
	}
	
	// Check status code
	if resp.StatusCode >= 400 {
		return resp.StatusCode, string(respBody), fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	
	return resp.StatusCode, string(respBody), nil
}

// executeN8nWorkflow executes an n8n workflow (legacy support)
func (s *Scheduler) executeN8nWorkflow(schedule *Schedule) (int, string, error) {
	// For backwards compatibility, convert to HTTP webhook call
	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		return 0, "", fmt.Errorf("N8N_BASE_URL not configured")
	}
	
	webhookURL := fmt.Sprintf("%s/webhook/%s", n8nURL, schedule.TargetWorkflowID)
	
	// Create a copy of the schedule with updated URL
	httpSchedule := *schedule
	httpSchedule.TargetType = "http"
	httpSchedule.TargetURL = webhookURL
	httpSchedule.TargetMethod = "POST"
	
	return s.executeHTTPTarget(&httpSchedule)
}

// executeScenario executes another Vrooli scenario
func (s *Scheduler) executeScenario(schedule *Schedule) (int, string, error) {
	// Call scenario API endpoint
	scenarioURL := fmt.Sprintf("http://localhost:%s/api/execute", schedule.TargetWorkflowID)
	
	httpSchedule := *schedule
	httpSchedule.TargetType = "http"
	httpSchedule.TargetURL = scenarioURL
	httpSchedule.TargetMethod = "POST"
	
	return s.executeHTTPTarget(&httpSchedule)
}

// monitorRoutine monitors schedule health and sends alerts
func (s *Scheduler) monitorRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkScheduleHealth()
		}
	}
}

// checkScheduleHealth checks for unhealthy schedules
func (s *Scheduler) checkScheduleHealth() {
	query := `
		SELECT s.id, s.name, sm.consecutive_failures, sm.last_failure_at
		FROM schedules s
		JOIN schedule_metrics sm ON s.id = sm.schedule_id
		WHERE s.enabled = true 
		  AND sm.consecutive_failures >= 3
		  AND sm.last_failure_at > NOW() - INTERVAL '1 hour'
	`
	
	rows, err := s.db.Query(query)
	if err != nil {
		log.Printf("Failed to check schedule health: %v", err)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var scheduleID, name string
		var failures int
		var lastFailure time.Time
		
		if err := rows.Scan(&scheduleID, &name, &failures, &lastFailure); err != nil {
			continue
		}
		
		log.Printf("ALERT: Schedule '%s' has %d consecutive failures (last: %v)", 
			name, failures, lastFailure)
		
		// Send alert notification
		s.sendHealthAlert(scheduleID, name, failures)
	}
}

// missedScheduleHandler handles schedules that were missed (e.g., during downtime)
func (s *Scheduler) missedScheduleHandler() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.catchUpMissedSchedules()
		}
	}
}

// catchUpMissedSchedules executes schedules that were missed
func (s *Scheduler) catchUpMissedSchedules() {
	query := `
		SELECT s.id, s.name, s.last_executed_at, s.next_execution_at
		FROM schedules s
		WHERE s.enabled = true 
		  AND s.catch_up_missed = true
		  AND s.next_execution_at < NOW() - INTERVAL '2 minutes'
		  AND (s.last_executed_at IS NULL OR s.last_executed_at < s.next_execution_at)
	`
	
	rows, err := s.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var scheduleID, name string
		var lastExecuted, nextExecution *time.Time
		
		if err := rows.Scan(&scheduleID, &name, &lastExecuted, &nextExecution); err != nil {
			continue
		}
		
		log.Printf("Catching up missed execution for schedule: %s", name)
		
		// Queue catch-up execution
		schedule, _ := s.getScheduleByID(scheduleID)
		if schedule != nil {
			execution := &ScheduleExecution{
				Schedule:      schedule,
				ExecutionID:   uuid.New().String(),
				ScheduledTime: *nextExecution,
				IsManual:      false,
				IsCatchUp:     true,
				TriggeredBy:   "catch_up_handler",
			}
			
			select {
			case s.executions <- execution:
			default:
			}
		}
	}
}

// retryHandler handles failed executions that need retry
func (s *Scheduler) retryHandler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processRetries()
		}
	}
}

// processRetries finds and retries failed executions
func (s *Scheduler) processRetries() {
	query := `
		SELECT e.id, e.schedule_id, e.attempt_count, s.max_retries, s.retry_strategy
		FROM executions e
		JOIN schedules s ON e.schedule_id = s.id
		WHERE e.status = 'failed'
		  AND e.attempt_count < s.max_retries
		  AND e.end_time > NOW() - INTERVAL '1 hour'
		  AND NOT EXISTS (
		    SELECT 1 FROM executions e2 
		    WHERE e2.schedule_id = e.schedule_id 
		    AND e2.start_time > e.end_time
		  )
		ORDER BY e.end_time ASC
		LIMIT 10
	`
	
	rows, err := s.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var executionID, scheduleID, retryStrategy string
		var attemptCount, maxRetries int
		
		if err := rows.Scan(&executionID, &scheduleID, &attemptCount, &maxRetries, &retryStrategy); err != nil {
			continue
		}
		
		// Calculate retry delay
		delay := s.calculateRetryDelay(attemptCount, retryStrategy)
		
		// Schedule retry
		time.AfterFunc(delay, func() {
			schedule, _ := s.getScheduleByID(scheduleID)
			if schedule != nil {
				execution := &ScheduleExecution{
					Schedule:      schedule,
					ExecutionID:   uuid.New().String(),
					ScheduledTime: time.Now(),
					IsManual:      false,
					IsCatchUp:     false,
					TriggeredBy:   "retry_handler",
					RetryCount:    attemptCount + 1,
				}
				
				select {
				case s.executions <- execution:
				default:
				}
			}
		})
	}
}

// scheduleRetry schedules a retry for a failed execution
func (s *Scheduler) scheduleRetry(exec *ScheduleExecution) {
	delay := s.calculateRetryDelay(exec.RetryCount, exec.Schedule.RetryStrategy)
	
	time.AfterFunc(delay, func() {
		exec.RetryCount++
		select {
		case s.executions <- exec:
			log.Printf("Scheduled retry #%d for %s", exec.RetryCount, exec.Schedule.Name)
		default:
		}
	})
}

// calculateRetryDelay calculates the delay before retry based on strategy
func (s *Scheduler) calculateRetryDelay(attemptCount int, strategy string) time.Duration {
	switch strategy {
	case "exponential":
		// 2^attempt seconds (2s, 4s, 8s, 16s, ...)
		return time.Duration(1<<uint(attemptCount)) * time.Second
	case "linear":
		// attempt * 10 seconds (10s, 20s, 30s, ...)
		return time.Duration(attemptCount*10) * time.Second
	case "fixed":
		// Always 30 seconds
		return 30 * time.Second
	default:
		return 10 * time.Second
	}
}

// Helper methods for database operations

func (s *Scheduler) getScheduleByID(scheduleID string) (*Schedule, error) {
	var schedule Schedule
	var headers, payload []byte
	var targetWorkflowID sql.NullString

	query := `
		SELECT id, name, cron_expression, timezone, target_type, target_url,
		       target_workflow_id, target_method, target_headers, target_payload,
		       enabled, overlap_policy, max_retries, retry_strategy,
		       timeout_seconds, catch_up_missed, priority
		FROM schedules WHERE id = $1
	`

	err := s.db.QueryRow(query, scheduleID).Scan(
		&schedule.ID, &schedule.Name, &schedule.CronExpression,
		&schedule.Timezone, &schedule.TargetType, &schedule.TargetURL,
		&targetWorkflowID, &schedule.TargetMethod,
		&headers, &payload, &schedule.Enabled, &schedule.OverlapPolicy,
		&schedule.MaxRetries, &schedule.RetryStrategy, &schedule.TimeoutSeconds,
		&schedule.CatchUpMissed, &schedule.Priority,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable target_workflow_id
	if targetWorkflowID.Valid {
		schedule.TargetWorkflowID = targetWorkflowID.String
	}

	if headers != nil {
		json.Unmarshal(headers, &schedule.TargetHeaders)
	}
	if payload != nil {
		json.Unmarshal(payload, &schedule.TargetPayload)
	}

	return &schedule, nil
}

func (s *Scheduler) isScheduleRunning(scheduleID string) bool {
	var count int
	query := `
		SELECT COUNT(*) FROM executions 
		WHERE schedule_id = $1 AND status = 'running'
	`
	s.db.QueryRow(query, scheduleID).Scan(&count)
	return count > 0
}

func (s *Scheduler) recordExecutionStart(executionID, scheduleID string, scheduledTime time.Time, isManual, isCatchUp bool, triggeredBy string) {
	query := `
		INSERT INTO executions (id, schedule_id, scheduled_time, start_time, status, 
		                        attempt_count, is_manual_trigger, is_catch_up, triggered_by)
		VALUES ($1, $2, $3, NOW(), 'running', 1, $4, $5, $6)
	`
	s.db.Exec(query, executionID, scheduleID, scheduledTime, isManual, isCatchUp, triggeredBy)
}

func (s *Scheduler) recordExecutionSuccess(executionID string, duration, responseCode int, responseBody string) {
	query := `
		UPDATE executions 
		SET end_time = NOW(), duration_ms = $2, status = 'success',
		    response_code = $3, response_body = $4
		WHERE id = $1
	`
	s.db.Exec(query, executionID, duration, responseCode, responseBody)
}

func (s *Scheduler) recordExecutionFailure(executionID string, duration int, errorMessage string) {
	query := `
		UPDATE executions 
		SET end_time = NOW(), duration_ms = $2, status = 'failed',
		    error_message = $3
		WHERE id = $1
	`
	s.db.Exec(query, executionID, duration, errorMessage)
}

func (s *Scheduler) updateScheduleMetrics(scheduleID string, success bool) {
	// Update or insert metrics
	if success {
		query := `
			INSERT INTO schedule_metrics (schedule_id, total_executions, success_count, consecutive_failures, last_success_at)
			VALUES ($1, 1, 1, 0, NOW())
			ON CONFLICT (schedule_id) DO UPDATE SET
				total_executions = schedule_metrics.total_executions + 1,
				success_count = schedule_metrics.success_count + 1,
				consecutive_failures = 0,
				last_success_at = NOW()
		`
		s.db.Exec(query, scheduleID)
	} else {
		query := `
			INSERT INTO schedule_metrics (schedule_id, total_executions, failure_count, consecutive_failures, last_failure_at)
			VALUES ($1, 1, 1, 1, NOW())
			ON CONFLICT (schedule_id) DO UPDATE SET
				total_executions = schedule_metrics.total_executions + 1,
				failure_count = schedule_metrics.failure_count + 1,
				consecutive_failures = schedule_metrics.consecutive_failures + 1,
				last_failure_at = NOW()
		`
		s.db.Exec(query, scheduleID)
	}
}

func (s *Scheduler) updateNextExecutionTime(scheduleID string) {
	s.jobsMutex.RLock()
	entryID, exists := s.jobs[scheduleID]
	s.jobsMutex.RUnlock()
	
	if !exists {
		return
	}
	
	entry := s.cron.Entry(entryID)
	if entry.ID != 0 {
		nextTime := entry.Next
		query := `UPDATE schedules SET next_execution_at = $2 WHERE id = $1`
		s.db.Exec(query, scheduleID, nextTime)
	}
}

func (s *Scheduler) sendFailureNotification(schedule *Schedule, err error) {
	// Send notification via notification-hub or log
	log.Printf("ALERT: Schedule '%s' failed after %d retries: %v", 
		schedule.Name, schedule.MaxRetries, err)
	
	// TODO: Integrate with notification-hub scenario
}

func (s *Scheduler) sendHealthAlert(scheduleID, name string, failures int) {
	// Send health alert notification
	log.Printf("HEALTH ALERT: Schedule '%s' (ID: %s) has %d consecutive failures", 
		name, scheduleID, failures)
	
	// TODO: Integrate with notification-hub scenario
}