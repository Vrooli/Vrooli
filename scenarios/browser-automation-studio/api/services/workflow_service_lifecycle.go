package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/database"
)

// StopExecution stops a running execution
func (s *WorkflowService) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	// Get current execution
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return err
	}

	// Only stop if currently running
	if execution.Status != "running" && execution.Status != "pending" {
		return nil // Already stopped/completed
	}

	// Signal the async runner to stop as early as possible
	s.cancelExecutionByID(execution.ID)

	// Update execution status
	execution.Status = "cancelled"
	now := time.Now()
	execution.CompletedAt = &now

	if err := s.repo.UpdateExecution(ctx, execution); err != nil {
		return err
	}

	// Log the cancellation
	logEntry := &database.ExecutionLog{
		ExecutionID: execution.ID,
		Level:       "info",
		StepName:    "execution_cancelled",
		Message:     "Execution cancelled by user request",
	}
	if err := s.repo.CreateExecutionLog(ctx, logEntry); err != nil && s.log != nil {
		s.log.WithError(err).WithField("execution_id", execution.ID).Warn("Failed to persist cancellation log entry")
	}

	s.log.WithField("execution_id", executionID).Info("Execution stopped by user request")
	return nil
}

// startExecutionRunner launches the async execution goroutine with cancellation support.
func (s *WorkflowService) startExecutionRunner(execution *database.Execution, workflow *database.Workflow) {
	ctx, cancel := context.WithCancel(context.Background())
	s.storeExecutionCancel(execution.ID, cancel)
	go s.executeWorkflowAsync(ctx, execution, workflow)
}

// storeExecutionCancel stores a cancel function for later invocation.
func (s *WorkflowService) storeExecutionCancel(executionID uuid.UUID, cancel context.CancelFunc) {
	if cancel == nil {
		return
	}
	s.executionCancels.Store(executionID, cancel)
}

// cancelExecutionByID retrieves and invokes the stored cancel function for an execution.
func (s *WorkflowService) cancelExecutionByID(executionID uuid.UUID) {
	value, ok := s.executionCancels.LoadAndDelete(executionID)
	if !ok {
		return
	}
	if cancel, valid := value.(context.CancelFunc); valid && cancel != nil {
		cancel()
	}
}

// executeWorkflowAsync executes a workflow asynchronously
func (s *WorkflowService) executeWorkflowAsync(ctx context.Context, execution *database.Execution, workflow *database.Workflow) {
	if ctx == nil {
		ctx = context.Background()
	}
	defer s.cancelExecutionByID(execution.ID)

	s.log.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"workflow_id":  execution.WorkflowID,
	}).Info("Starting async workflow execution")

	persistenceCtx := context.Background()

	// Update status to running and broadcast
	execution.Status = "running"
	now := time.Now()
	execution.LastHeartbeat = &now
	if err := s.repo.UpdateExecution(persistenceCtx, execution); err != nil {
		s.log.WithError(err).Error("Failed to update execution status to running")
		return
	}

	var err error
	selection := autoengine.FromEnv()
	eventSink := autoevents.NewWSHubSink(s.wsHub, s.log, autocontracts.DefaultEventBufferLimits)
	seq := autoevents.NewPerExecutionSequencer()
	publishLifecycle := func(kind autocontracts.EventKind, payload any) {
		if eventSink == nil {
			return
		}
		env := autocontracts.EventEnvelope{
			SchemaVersion:  autocontracts.EventEnvelopeSchemaVersion,
			PayloadVersion: autocontracts.PayloadVersion,
			Kind:           kind,
			ExecutionID:    execution.ID,
			WorkflowID:     execution.WorkflowID,
			Sequence:       seq.Next(execution.ID),
			Timestamp:      time.Now().UTC(),
			Payload:        payload,
		}
		_ = eventSink.Publish(ctx, env)
	}

	if s.log != nil && workflow != nil {
		if unsupported := unsupportedAutomationNodes(workflow.FlowDefinition); len(unsupported) > 0 {
			s.log.WithFields(logrus.Fields{
				"workflow_id":       workflow.ID,
				"unsupported_nodes": unsupported,
			}).Warn("Executing workflow with nodes that exceed current automation coverage; execution may fail")
		}
	}

	publishLifecycle(autocontracts.EventKindExecutionStarted, map[string]any{"status": "running"})
	defer closeEventSink(eventSink, execution.ID)

	err = s.executeWithAutomationEngine(ctx, execution, workflow, selection, eventSink)

	var capErr *autoexecutor.CapabilityError

	switch {
	case err == nil:
		execution.Status = "completed"
		execution.Progress = 100
		execution.CurrentStep = "Completed"
		now := time.Now()
		execution.CompletedAt = &now
		execution.Result = database.JSONMap{
			"success": true,
			"message": "Workflow completed successfully",
		}

		s.log.WithField("execution_id", execution.ID).Info("Workflow execution completed successfully")

		publishLifecycle(autocontracts.EventKindExecutionCompleted, map[string]any{"status": "completed"})

	case errors.Is(err, context.Canceled):
		s.log.WithField("execution_id", execution.ID).Info("Workflow execution cancelled")
		execution.Status = "cancelled"
		execution.Error.Valid = false
		now := time.Now()
		execution.CompletedAt = &now
		s.recordExecutionMarker(ctx, execution.ID, autocontracts.StepFailure{
			Kind:    autocontracts.FailureKindCancelled,
			Message: "execution cancelled",
			Source:  autocontracts.FailureSourceExecutor,
		})

		logEntry := &database.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       "info",
			StepName:    "execution_cancelled",
			Message:     "Workflow execution cancelled",
		}
		if err := s.repo.CreateExecutionLog(persistenceCtx, logEntry); err != nil && s.log != nil {
			s.log.WithError(err).WithField("execution_id", execution.ID).Warn("Failed to persist cancellation log entry")
		}

		publishLifecycle(autocontracts.EventKindExecutionCancelled, map[string]any{"status": "cancelled"})

	case errors.Is(err, context.DeadlineExceeded):
		s.log.WithField("execution_id", execution.ID).Warn("Workflow execution timed out")
		execution.Status = "failed"
		execution.Error.Valid = true
		execution.Error.String = "execution timed out"
		now := time.Now()
		execution.CompletedAt = &now
		s.recordExecutionMarker(ctx, execution.ID, autocontracts.StepFailure{
			Kind:    autocontracts.FailureKindTimeout,
			Message: "execution timed out",
			Source:  autocontracts.FailureSourceExecutor,
		})

		logEntry := &database.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       "error",
			StepName:    "execution_timeout",
			Message:     "Execution timed out",
		}
		if persistErr := s.repo.CreateExecutionLog(persistenceCtx, logEntry); persistErr != nil && s.log != nil {
			s.log.WithError(persistErr).WithField("execution_id", execution.ID).Warn("Failed to persist timeout log entry")
		}

		publishLifecycle(autocontracts.EventKindExecutionFailed, map[string]any{"status": "failed", "error": execution.Error.String})

	case errors.As(err, &capErr):
		failure := applyCapabilityError(execution, capErr)
		s.recordExecutionMarker(ctx, execution.ID, failure)

		logEntry := &database.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       "error",
			StepName:    "capability_mismatch",
			Message:     failure.Message,
		}
		if persistErr := s.repo.CreateExecutionLog(persistenceCtx, logEntry); persistErr != nil && s.log != nil {
			s.log.WithError(persistErr).WithField("execution_id", execution.ID).Warn("Failed to persist capability mismatch log entry")
		}

		if s.log != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"execution_id": execution.ID,
				"missing":      capErr.Missing,
				"warnings":     capErr.Warnings,
				"engine":       capErr.Engine,
			}).Warn("Workflow failed: engine missing required capabilities")
		}

		publishLifecycle(autocontracts.EventKindExecutionFailed, map[string]any{
			"status":   "failed",
			"error":    failure.Message,
			"missing":  capErr.Missing,
			"warnings": capErr.Warnings,
		})

	default:
		s.log.WithError(err).Error("Workflow execution failed")

		execution.Status = "failed"
		execution.Error.Valid = true
		execution.Error.String = err.Error()
		now := time.Now()
		execution.CompletedAt = &now

		logEntry := &database.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       "error",
			StepName:    "execution_failed",
			Message:     "Workflow execution failed: " + err.Error(),
		}
		if persistErr := s.repo.CreateExecutionLog(persistenceCtx, logEntry); persistErr != nil && s.log != nil {
			s.log.WithError(persistErr).WithField("execution_id", execution.ID).Warn("Failed to persist failure log entry")
		}

		publishLifecycle(autocontracts.EventKindExecutionFailed, map[string]any{"status": "failed", "error": err.Error()})

	}

	if err := s.repo.UpdateExecution(persistenceCtx, execution); err != nil {
		s.log.WithError(err).Error("Failed to update final execution status")
	}
}

// recordExecutionMarker best-effort persists a crash/timeout/cancel marker via recorder when available.
func (s *WorkflowService) recordExecutionMarker(ctx context.Context, executionID uuid.UUID, failure autocontracts.StepFailure) {
	if s == nil || s.artifactRecorder == nil {
		return
	}
	_ = s.artifactRecorder.MarkCrash(context.WithoutCancel(ctx), executionID, failure)
}

// applyCapabilityError marks the execution as failed due to an engine capability
// mismatch and returns the structured failure marker for persistence.
func applyCapabilityError(execution *database.Execution, capErr *autoexecutor.CapabilityError) autocontracts.StepFailure {
	now := time.Now()
	execution.Status = "failed"
	execution.CompletedAt = &now
	execution.Error.Valid = true

	missing := fmt.Sprintf("%v", capErr.Missing)
	warn := fmt.Sprintf("%v", capErr.Warnings)
	message := fmt.Sprintf("engine %s missing required capabilities: %s", capErr.Engine, missing)
	if warn != "[]" && warn != "" {
		message = fmt.Sprintf("%s (warnings: %s)", message, warn)
	}
	if len(capErr.Reasons) > 0 {
		// Surface a concise set of sources for UI/CLI surfaces.
		var parts []string
		for key, vals := range capErr.Reasons {
			if len(vals) == 0 {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s via %v", key, vals))
		}
		if len(parts) > 0 {
			message = fmt.Sprintf("%s | sources: %v", message, parts)
		}
	}
	execution.Error.String = message

	return autocontracts.StepFailure{
		Kind:    autocontracts.FailureKindOrchestration,
		Code:    "capability_mismatch",
		Message: message,
		Source:  autocontracts.FailureSourceExecutor,
		Details: map[string]any{
			"missing":  capErr.Missing,
			"warnings": capErr.Warnings,
			"engine":   capErr.Engine,
			"reasons":  capErr.Reasons,
		},
	}
}

type closableEventSink interface {
	CloseExecution(uuid.UUID)
}

func closeEventSink(sink autoevents.Sink, executionID uuid.UUID) {
	if closable, ok := sink.(closableEventSink); ok && closable != nil {
		closable.CloseExecution(executionID)
	}
}
