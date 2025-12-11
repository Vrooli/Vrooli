package workflow

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

// ResumeExecution resumes an interrupted execution from its last checkpoint.
// It creates a new execution that starts from where the original left off.
// Returns the new execution or an error if the original cannot be resumed.
func (s *WorkflowService) ResumeExecution(ctx context.Context, originalExecutionID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
	// Verify the execution can be resumed
	originalExecution, lastStepIndex, err := s.repo.GetResumableExecution(ctx, originalExecutionID)
	if err != nil {
		return nil, fmt.Errorf("execution cannot be resumed: %w", err)
	}

	// Verify the workflow still exists
	workflow, err := s.repo.GetWorkflow(ctx, originalExecution.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	// Sync project workflows if applicable
	if workflow.ProjectID != nil {
		if err := s.syncProjectWorkflows(ctx, *workflow.ProjectID); err != nil {
			return nil, err
		}
		workflow, err = s.repo.GetWorkflow(ctx, originalExecution.WorkflowID)
		if err != nil {
			return nil, err
		}
	}

	// Restore variables from completed steps
	initialVars, err := s.extractVariablesFromCompletedSteps(ctx, originalExecutionID)
	if err != nil {
		s.log.WithError(err).WithField("original_execution_id", originalExecutionID).Warn("Failed to extract variables from completed steps; resuming with empty state")
		initialVars = make(map[string]any)
	}

	// Merge any new parameters provided by the caller
	if parameters != nil {
		for k, v := range parameters {
			initialVars[k] = v
		}
	}

	// Create a new execution for the resumed run
	newExecution := &database.Execution{
		ID:              uuid.New(),
		WorkflowID:      originalExecution.WorkflowID,
		WorkflowVersion: workflow.Version,
		Status:          "pending",
		TriggerType:     "resume",
		TriggerMetadata: database.JSONMap{
			"resumed_from_execution_id": originalExecutionID.String(),
			"resume_from_step_index":    lastStepIndex,
		},
		Parameters:  database.JSONMap(initialVars),
		StartedAt:   time.Now(),
		Progress:    originalExecution.Progress,
		CurrentStep: fmt.Sprintf("Resuming from step %d", lastStepIndex+1),
	}

	if err := s.repo.CreateExecution(ctx, newExecution); err != nil {
		return nil, err
	}

	// Start async execution with resume parameters
	s.startResumedExecutionRunner(newExecution, workflow, lastStepIndex, initialVars, originalExecutionID)

	s.log.WithFields(logrus.Fields{
		"original_execution_id": originalExecutionID,
		"new_execution_id":      newExecution.ID,
		"resume_from_step":      lastStepIndex,
		"workflow_id":           workflow.ID,
	}).Info("Resumed execution from checkpoint")

	return newExecution, nil
}

// extractVariablesFromCompletedSteps reconstructs the variable state from completed execution steps.
// It looks for steps that stored results (via storeResult parameter) and extracts their output values.
func (s *WorkflowService) extractVariablesFromCompletedSteps(ctx context.Context, executionID uuid.UUID) (map[string]any, error) {
	steps, err := s.repo.GetCompletedSteps(ctx, executionID)
	if err != nil {
		return nil, err
	}

	vars := make(map[string]any)
	for _, step := range steps {
		// Check if the step stored a result
		if step.Output != nil {
			// Look for extracted data that was stored with a storeResult key
			if stored, ok := step.Output["stored_as"]; ok {
				if key, ok := stored.(string); ok && key != "" {
					if value, ok := step.Output["value"]; ok {
						vars[key] = value
					}
				}
			}
			// Also check for raw extracted data
			if extracted, ok := step.Output["extracted_data"]; ok {
				if storeKey, ok := step.Metadata["store_result"]; ok {
					if key, ok := storeKey.(string); ok && key != "" {
						vars[key] = extracted
					}
				}
			}
		}
	}

	return vars, nil
}

// startResumedExecutionRunner launches the async execution goroutine for a resumed execution.
func (s *WorkflowService) startResumedExecutionRunner(execution *database.Execution, workflow *database.Workflow, startFromStepIndex int, initialVars map[string]any, resumedFromID uuid.UUID) {
	ctx, cancel := context.WithCancel(context.Background())
	s.storeExecutionCancel(execution.ID, cancel)
	go s.executeResumedWorkflowAsync(ctx, execution, workflow, startFromStepIndex, initialVars, resumedFromID)
}

// executeResumedWorkflowAsync executes a workflow asynchronously from a specific step index.
func (s *WorkflowService) executeResumedWorkflowAsync(ctx context.Context, execution *database.Execution, workflow *database.Workflow, startFromStepIndex int, initialVars map[string]any, resumedFromID uuid.UUID) {
	if ctx == nil {
		ctx = context.Background()
	}
	defer s.cancelExecutionByID(execution.ID)

	s.log.WithFields(logrus.Fields{
		"execution_id":          execution.ID,
		"workflow_id":           execution.WorkflowID,
		"resumed_from_id":       resumedFromID,
		"start_from_step_index": startFromStepIndex,
	}).Info("Starting resumed workflow execution")

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
	eventSink := s.newEventSink()
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

	publishLifecycle(autocontracts.EventKindExecutionStarted, map[string]any{
		"status":                "running",
		"resumed":               true,
		"resumed_from_id":       resumedFromID.String(),
		"start_from_step_index": startFromStepIndex,
	})
	defer closeEventSink(eventSink, execution.ID)

	err = s.executeResumedWithAutomationEngine(ctx, execution, workflow, selection, eventSink, startFromStepIndex, initialVars, resumedFromID)

	var capErr *autoexecutor.CapabilityError

	switch {
	case err == nil:
		execution.Status = "completed"
		execution.Progress = 100
		execution.CurrentStep = "Completed"
		now := time.Now()
		execution.CompletedAt = &now
		execution.Result = database.JSONMap{
			"success":         true,
			"message":         "Resumed workflow completed successfully",
			"resumed_from_id": resumedFromID.String(),
		}

		s.log.WithField("execution_id", execution.ID).Info("Resumed workflow execution completed successfully")

		publishLifecycle(autocontracts.EventKindExecutionCompleted, map[string]any{"status": "completed", "resumed": true})

	case errors.Is(err, context.Canceled):
		s.log.WithField("execution_id", execution.ID).Info("Resumed workflow execution cancelled")
		execution.Status = "cancelled"
		execution.Error.Valid = false
		now := time.Now()
		execution.CompletedAt = &now
		s.recordExecutionMarker(ctx, execution.ID, autocontracts.StepFailure{
			Kind:    autocontracts.FailureKindCancelled,
			Message: "execution cancelled",
			Source:  autocontracts.FailureSourceExecutor,
		})

		publishLifecycle(autocontracts.EventKindExecutionCancelled, map[string]any{"status": "cancelled", "resumed": true})

	case errors.Is(err, context.DeadlineExceeded):
		s.log.WithField("execution_id", execution.ID).Warn("Resumed workflow execution timed out")
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

		publishLifecycle(autocontracts.EventKindExecutionFailed, map[string]any{"status": "failed", "error": execution.Error.String, "resumed": true})

	case errors.As(err, &capErr):
		failure := applyCapabilityError(execution, capErr)
		s.recordExecutionMarker(ctx, execution.ID, failure)

		if s.log != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"execution_id": execution.ID,
				"missing":      capErr.Missing,
				"warnings":     capErr.Warnings,
				"engine":       capErr.Engine,
			}).Warn("Resumed workflow failed: engine missing required capabilities")
		}

		publishLifecycle(autocontracts.EventKindExecutionFailed, map[string]any{
			"status":   "failed",
			"error":    failure.Message,
			"missing":  capErr.Missing,
			"warnings": capErr.Warnings,
			"resumed":  true,
		})

	default:
		s.log.WithError(err).Error("Resumed workflow execution failed")

		execution.Status = "failed"
		execution.Error.Valid = true
		execution.Error.String = err.Error()
		now := time.Now()
		execution.CompletedAt = &now

		publishLifecycle(autocontracts.EventKindExecutionFailed, map[string]any{"status": "failed", "error": err.Error(), "resumed": true})
	}

	if err := s.repo.UpdateExecution(persistenceCtx, execution); err != nil {
		s.log.WithError(err).Error("Failed to update final execution status")
	}
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

	workflowName := "unknown"
	if workflow != nil {
		workflowName = workflow.Name
	}

	s.log.WithFields(logrus.Fields{
		"execution_id":  execution.ID,
		"workflow_id":   execution.WorkflowID,
		"workflow_name": workflowName,
	}).Info("Workflow execution started")

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
	eventSink := s.newEventSink()
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
		execution.CurrentStep = "All steps completed"
		now := time.Now()
		execution.CompletedAt = &now
		execution.Result = database.JSONMap{
			"success": true,
			"message": "Workflow completed successfully",
		}

		s.log.WithFields(logrus.Fields{
			"execution_id":  execution.ID,
			"workflow_id":   execution.WorkflowID,
			"workflow_name": workflowName,
			"duration_ms":   now.Sub(execution.StartedAt).Milliseconds(),
		}).Info("Workflow execution completed successfully")

		publishLifecycle(autocontracts.EventKindExecutionCompleted, map[string]any{"status": "completed"})

	case errors.Is(err, context.Canceled):
		execution.Status = "cancelled"
		execution.CurrentStep = "Cancelled by user"
		execution.Error.Valid = false
		now := time.Now()
		execution.CompletedAt = &now
		s.recordExecutionMarker(ctx, execution.ID, autocontracts.StepFailure{
			Kind:    autocontracts.FailureKindCancelled,
			Message: "execution cancelled",
			Source:  autocontracts.FailureSourceExecutor,
		})

		s.log.WithFields(logrus.Fields{
			"execution_id":  execution.ID,
			"workflow_id":   execution.WorkflowID,
			"workflow_name": workflowName,
			"duration_ms":   now.Sub(execution.StartedAt).Milliseconds(),
		}).Info("Workflow execution cancelled by user")

		logEntry := &database.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       "info",
			StepName:    "execution_cancelled",
			Message:     "Workflow execution cancelled by user request",
		}
		if err := s.repo.CreateExecutionLog(persistenceCtx, logEntry); err != nil && s.log != nil {
			s.log.WithError(err).WithField("execution_id", execution.ID).Warn("Failed to persist cancellation log entry")
		}

		publishLifecycle(autocontracts.EventKindExecutionCancelled, map[string]any{"status": "cancelled"})

	case errors.Is(err, context.DeadlineExceeded):
		execution.Status = "failed"
		execution.CurrentStep = "Timed out"
		execution.Error.Valid = true
		execution.Error.String = "execution timed out - consider increasing timeout or breaking workflow into smaller steps"
		now := time.Now()
		execution.CompletedAt = &now
		s.recordExecutionMarker(ctx, execution.ID, autocontracts.StepFailure{
			Kind:    autocontracts.FailureKindTimeout,
			Message: "execution timed out",
			Source:  autocontracts.FailureSourceExecutor,
		})

		s.log.WithFields(logrus.Fields{
			"execution_id":  execution.ID,
			"workflow_id":   execution.WorkflowID,
			"workflow_name": workflowName,
			"duration_ms":   now.Sub(execution.StartedAt).Milliseconds(),
		}).Warn("Workflow execution timed out")

		logEntry := &database.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       "error",
			StepName:    "execution_timeout",
			Message:     "Workflow execution timed out - consider increasing timeout or breaking workflow into smaller steps",
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
		execution.Status = "failed"
		execution.CurrentStep = "Failed"
		execution.Error.Valid = true
		execution.Error.String = err.Error()
		now := time.Now()
		execution.CompletedAt = &now

		s.log.WithError(err).WithFields(logrus.Fields{
			"execution_id":  execution.ID,
			"workflow_id":   execution.WorkflowID,
			"workflow_name": workflowName,
			"duration_ms":   now.Sub(execution.StartedAt).Milliseconds(),
		}).Error("Workflow execution failed")

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
