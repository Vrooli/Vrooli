// Package scheduler provides workflow scheduling capabilities using cron expressions.
// It manages scheduled workflow executions, supporting timezone-aware scheduling,
// hot-reloading of schedules, and graceful shutdown.
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

// WorkflowExecutor defines the interface for executing workflows.
// This is implemented by the workflow service.
type WorkflowExecutor interface {
	ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error)
}

// ScheduleNotifier defines the interface for notifying about schedule events.
// This is used to push WebSocket notifications.
type ScheduleNotifier interface {
	BroadcastScheduleEvent(event ScheduleEvent)
}

// ScheduleEvent represents a schedule-related event for WebSocket notification.
type ScheduleEvent struct {
	Type         string    `json:"type"`
	ScheduleID   uuid.UUID `json:"schedule_id"`
	WorkflowID   uuid.UUID `json:"workflow_id"`
	ScheduleName string    `json:"schedule_name,omitempty"`
	ExecutionID  uuid.UUID `json:"execution_id,omitempty"`
	NextRunAt    string    `json:"next_run_at,omitempty"`
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// Event types for schedule notifications
const (
	EventTypeScheduleStarted   = "schedule_started"
	EventTypeScheduleCompleted = "schedule_completed"
	EventTypeScheduleFailed    = "schedule_failed"
	EventTypeScheduleUpdated   = "schedule_updated"
)

// Scheduler manages workflow scheduling using cron expressions.
type Scheduler struct {
	repo     database.Repository
	executor WorkflowExecutor
	notifier ScheduleNotifier
	log      *logrus.Logger

	cron      *cron.Cron
	entries   map[uuid.UUID]cron.EntryID // scheduleID -> cron entry ID
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
}

// Config holds scheduler configuration options.
type Config struct {
	// ExecutionTimeout is the max time allowed for a single scheduled execution.
	// Default: 30 minutes
	ExecutionTimeout time.Duration

	// RecoveryInterval is how often to check for missed schedules.
	// Default: 5 minutes
	RecoveryInterval time.Duration
}

// DefaultConfig returns the default scheduler configuration.
func DefaultConfig() *Config {
	return &Config{
		ExecutionTimeout: 30 * time.Minute,
		RecoveryInterval: 5 * time.Minute,
	}
}

// New creates a new Scheduler instance.
func New(repo database.Repository, executor WorkflowExecutor, notifier ScheduleNotifier, log *logrus.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	// Create cron scheduler with seconds precision and location support
	c := cron.New(
		cron.WithSeconds(),
		cron.WithChain(
			cron.SkipIfStillRunning(cron.DefaultLogger),
			cron.Recover(cron.DefaultLogger),
		),
	)

	return &Scheduler{
		repo:     repo,
		executor: executor,
		notifier: notifier,
		log:      log,
		cron:     c,
		entries:  make(map[uuid.UUID]cron.EntryID),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start initializes and starts the scheduler service.
// It loads all active schedules from the database and begins scheduling.
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	s.log.Info("Starting scheduler service...")

	// Load all active schedules from database
	schedules, err := s.repo.GetActiveSchedules(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to load active schedules: %w", err)
	}

	s.log.WithField("count", len(schedules)).Info("Loading active schedules")

	// Register each schedule with the cron scheduler
	for _, schedule := range schedules {
		if err := s.registerSchedule(schedule); err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"schedule_id":   schedule.ID,
				"schedule_name": schedule.Name,
			}).Error("Failed to register schedule")
			// Continue with other schedules even if one fails
			continue
		}
	}

	// Start the cron scheduler
	s.cron.Start()
	s.isRunning = true

	s.log.WithField("registered_count", len(s.entries)).Info("Scheduler service started")
	return nil
}

// Stop gracefully stops the scheduler service.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	s.log.Info("Stopping scheduler service...")

	// Stop accepting new jobs and wait for running jobs to complete
	ctx := s.cron.Stop()
	<-ctx.Done()

	// Cancel the context
	s.cancel()

	s.isRunning = false
	s.log.Info("Scheduler service stopped")
	return nil
}

// RegisterSchedule adds or updates a schedule in the cron scheduler.
// This is called when a schedule is created or updated via the API.
func (s *Scheduler) RegisterSchedule(schedule *database.WorkflowSchedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entry if present
	if entryID, exists := s.entries[schedule.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.entries, schedule.ID)
	}

	// Only register active schedules
	if !schedule.IsActive {
		s.log.WithFields(logrus.Fields{
			"schedule_id":   schedule.ID,
			"schedule_name": schedule.Name,
		}).Debug("Schedule is inactive, not registering")
		return nil
	}

	return s.registerSchedule(schedule)
}

// UnregisterSchedule removes a schedule from the cron scheduler.
// This is called when a schedule is deleted via the API.
func (s *Scheduler) UnregisterSchedule(scheduleID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.entries[scheduleID]; exists {
		s.cron.Remove(entryID)
		delete(s.entries, scheduleID)
		s.log.WithField("schedule_id", scheduleID).Debug("Unregistered schedule")
	}

	return nil
}

// registerSchedule adds a schedule to the cron scheduler (internal, not thread-safe).
func (s *Scheduler) registerSchedule(schedule *database.WorkflowSchedule) error {
	// Parse the timezone
	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		loc = time.UTC
		s.log.WithFields(logrus.Fields{
			"schedule_id": schedule.ID,
			"timezone":    schedule.Timezone,
		}).Warn("Invalid timezone, using UTC")
	}

	// Convert cron expression to robfig/cron format
	// robfig/cron v3 uses 6-field format: second minute hour day month weekday
	// If we have a 5-field expression, prepend "0" for seconds
	cronExpr := schedule.CronExpression
	if len(splitCronFields(cronExpr)) == 5 {
		cronExpr = "0 " + cronExpr
	}

	// Create the job function
	job := s.createJob(schedule)

	// Add the job to the cron scheduler with timezone
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSchedule, err := parser.Parse(cronExpr)
	if err != nil {
		return fmt.Errorf("failed to parse cron expression %q: %w", cronExpr, err)
	}

	// Wrap the schedule with timezone location
	tzSchedule := &tzCronSchedule{
		Schedule: cronSchedule,
		loc:      loc,
	}

	entryID := s.cron.Schedule(tzSchedule, cron.FuncJob(job))
	s.entries[schedule.ID] = entryID

	// Update next_run_at in database
	nextRun := tzSchedule.Next(time.Now())
	if err := s.repo.UpdateScheduleNextRun(s.ctx, schedule.ID, nextRun); err != nil {
		s.log.WithError(err).WithField("schedule_id", schedule.ID).Warn("Failed to update next_run_at")
	}

	s.log.WithFields(logrus.Fields{
		"schedule_id":     schedule.ID,
		"schedule_name":   schedule.Name,
		"workflow_id":     schedule.WorkflowID,
		"cron_expression": schedule.CronExpression,
		"timezone":        schedule.Timezone,
		"next_run":        nextRun.Format(time.RFC3339),
	}).Info("Registered schedule")

	return nil
}

// createJob creates the function that will be executed on schedule.
func (s *Scheduler) createJob(schedule *database.WorkflowSchedule) func() {
	scheduleID := schedule.ID
	workflowID := schedule.WorkflowID
	scheduleName := schedule.Name
	params := schedule.Parameters

	return func() {
		s.log.WithFields(logrus.Fields{
			"schedule_id":   scheduleID,
			"schedule_name": scheduleName,
			"workflow_id":   workflowID,
		}).Info("Executing scheduled workflow")

		// Build execution parameters
		execParams := make(map[string]any)
		if params != nil {
			for k, v := range params {
				execParams[k] = v
			}
		}
		// Add metadata about the trigger
		execParams["_trigger_type"] = "scheduled"
		execParams["_schedule_id"] = scheduleID.String()
		execParams["_schedule_name"] = scheduleName
		execParams["_triggered_at"] = time.Now().Format(time.RFC3339)

		// Notify that execution is starting
		if s.notifier != nil {
			s.notifier.BroadcastScheduleEvent(ScheduleEvent{
				Type:         EventTypeScheduleStarted,
				ScheduleID:   scheduleID,
				WorkflowID:   workflowID,
				ScheduleName: scheduleName,
				Timestamp:    time.Now(),
			})
		}

		// Execute the workflow
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		execution, err := s.executor.ExecuteWorkflow(ctx, workflowID, execParams)
		now := time.Now()

		// Update last_run_at regardless of success/failure
		if updateErr := s.repo.UpdateScheduleLastRun(context.Background(), scheduleID, now); updateErr != nil {
			s.log.WithError(updateErr).WithField("schedule_id", scheduleID).Warn("Failed to update last_run_at")
		}

		// Update next_run_at
		if schedule, getErr := s.repo.GetSchedule(context.Background(), scheduleID); getErr == nil && schedule != nil {
			if nextRun, calcErr := CalculateNextRun(schedule.CronExpression, schedule.Timezone); calcErr == nil {
				if updateErr := s.repo.UpdateScheduleNextRun(context.Background(), scheduleID, nextRun); updateErr != nil {
					s.log.WithError(updateErr).WithField("schedule_id", scheduleID).Warn("Failed to update next_run_at")
				}
			}
		}

		if err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"schedule_id":   scheduleID,
				"schedule_name": scheduleName,
				"workflow_id":   workflowID,
			}).Error("Scheduled workflow execution failed")

			// Notify about failure
			if s.notifier != nil {
				s.notifier.BroadcastScheduleEvent(ScheduleEvent{
					Type:         EventTypeScheduleFailed,
					ScheduleID:   scheduleID,
					WorkflowID:   workflowID,
					ScheduleName: scheduleName,
					Error:        err.Error(),
					Timestamp:    time.Now(),
				})
			}
			return
		}

		s.log.WithFields(logrus.Fields{
			"schedule_id":   scheduleID,
			"schedule_name": scheduleName,
			"workflow_id":   workflowID,
			"execution_id":  execution.ID,
		}).Info("Scheduled workflow execution completed")

		// Notify about success
		if s.notifier != nil {
			s.notifier.BroadcastScheduleEvent(ScheduleEvent{
				Type:         EventTypeScheduleCompleted,
				ScheduleID:   scheduleID,
				WorkflowID:   workflowID,
				ScheduleName: scheduleName,
				ExecutionID:  execution.ID,
				Timestamp:    time.Now(),
			})
		}
	}
}

// GetScheduleInfo returns information about a registered schedule.
func (s *Scheduler) GetScheduleInfo(scheduleID uuid.UUID) (entryID cron.EntryID, registered bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entryID, registered = s.entries[scheduleID]
	return
}

// IsRunning returns whether the scheduler is currently running.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// RegisteredCount returns the number of registered schedules.
func (s *Scheduler) RegisteredCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// tzCronSchedule wraps a cron.Schedule to apply timezone.
type tzCronSchedule struct {
	cron.Schedule
	loc *time.Location
}

// Next returns the next activation time in the configured timezone.
func (s *tzCronSchedule) Next(t time.Time) time.Time {
	// Convert to target timezone for calculation
	localTime := t.In(s.loc)
	// Get next time in local timezone
	nextLocal := s.Schedule.Next(localTime)
	// Return in UTC
	return nextLocal.UTC()
}
