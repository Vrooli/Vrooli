package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/database"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

var (
	terminalExecutionStatuses = map[string]struct{}{
		"completed": {},
		"failed":    {},
		"cancelled": {},
	}
	adhocExecutionCleanupInterval = 5 * time.Second
	adhocExecutionRetentionPeriod = 10 * time.Minute
	adhocExecutionCleanupTimeout  = 6 * time.Hour
)

// IsTerminalExecutionStatus reports whether the supplied status represents a terminal execution state.
func IsTerminalExecutionStatus(status string) bool {
	if strings.TrimSpace(status) == "" {
		return false
	}
	_, ok := terminalExecutionStatuses[strings.ToLower(status)]
	return ok
}

// ExecuteWorkflow executes a workflow
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
	// Verify workflow exists
	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	if workflow.ProjectID != nil {
		if err := s.syncProjectWorkflows(ctx, *workflow.ProjectID); err != nil {
			return nil, err
		}
		workflow, err = s.repo.GetWorkflow(ctx, workflowID)
		if err != nil {
			return nil, err
		}
	}

	if err := s.ensureWorkflowChangeMetadata(ctx, workflow); err != nil {
		return nil, err
	}

	execution := &database.Execution{
		ID:              uuid.New(),
		WorkflowID:      workflowID,
		WorkflowVersion: workflow.Version,
		Status:          "pending",
		TriggerType:     "manual",
		Parameters:      database.JSONMap(parameters),
		StartedAt:       time.Now(),
		Progress:        0,
		CurrentStep:     "Initializing",
	}

	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return nil, err
	}

	// Start async execution
	s.startExecutionRunner(execution, workflow)

	return execution, nil
}

// ExecuteAdhocWorkflow executes a workflow definition without persisting it to the database.
// This is useful for testing scenarios where workflows should be ephemeral and not pollute
// the database with test data. The workflow definition is validated and executed directly,
// with execution records still persisted for telemetry and replay purposes.
func (s *WorkflowService) ExecuteAdhocWorkflow(ctx context.Context, flowDefinition map[string]any, parameters map[string]any, name string) (*database.Execution, error) {
	// Validate workflow definition structure
	if flowDefinition == nil {
		return nil, errors.New("flow_definition is required")
	}

	nodes, hasNodes := flowDefinition["nodes"]
	if !hasNodes {
		return nil, errors.New("flow_definition must contain 'nodes' array")
	}

	nodesArray, ok := nodes.([]interface{})
	if !ok {
		return nil, errors.New("'nodes' must be an array")
	}

	if len(nodesArray) == 0 {
		return nil, errors.New("workflow must contain at least one node")
	}

	_, hasEdges := flowDefinition["edges"]
	if !hasEdges {
		return nil, errors.New("flow_definition must contain 'edges' array")
	}

	// Create ephemeral workflow (temporarily persisted to satisfy FK constraint)
	// This workflow will be auto-deleted when execution is cleaned up (ON DELETE CASCADE)
	// Add timestamp to name to avoid unique constraint violations on (name, folder_path)
	ephemeralID := uuid.New()
	ephemeralName := fmt.Sprintf("%s [adhoc-%s]", name, ephemeralID.String()[:8])

	ephemeralWorkflow := &database.Workflow{
		ID:             ephemeralID,
		Name:           ephemeralName,
		FlowDefinition: database.JSONMap(flowDefinition),
		Version:        0, // Version 0 indicates adhoc/ephemeral workflow
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		// ProjectID is intentionally nil - adhoc workflows have no project context
	}

	// Temporarily persist ephemeral workflow to satisfy executions.workflow_id FK constraint
	// This allows executions table to maintain referential integrity while still being ephemeral
	if err := s.repo.CreateWorkflow(ctx, ephemeralWorkflow); err != nil {
		return nil, fmt.Errorf("failed to create ephemeral workflow: %w", err)
	}

	// Create execution record (persists for telemetry/replay)
	execution := &database.Execution{
		ID:              uuid.New(),
		WorkflowID:      ephemeralWorkflow.ID, // Reference ephemeral workflow ID
		WorkflowVersion: 0,                    // 0 indicates adhoc execution
		Status:          "pending",
		TriggerType:     "adhoc", // Special trigger type for ephemeral workflows
		Parameters:      database.JSONMap(parameters),
		StartedAt:       time.Now(),
		Progress:        0,
		CurrentStep:     "Initializing",
	}

	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Execute asynchronously (same pattern as normal workflow execution)
	// The executeWorkflowAsync method works with any workflow struct,
	// whether persisted or ephemeral
	s.startExecutionRunner(execution, ephemeralWorkflow)
	go s.scheduleAdhocWorkflowCleanup(ephemeralWorkflow.ID, execution.ID)

	return execution, nil
}

func (s *WorkflowService) scheduleAdhocWorkflowCleanup(workflowID, executionID uuid.UUID) {
	if s == nil || s.repo == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), adhocExecutionCleanupTimeout)
	defer cancel()

	ticker := time.NewTicker(adhocExecutionCleanupInterval)
	defer ticker.Stop()

	var terminalObservedAt time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		execution, err := s.repo.GetExecution(ctx, executionID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				terminalObservedAt = time.Now()
				break
			}
			if s.log != nil {
				s.log.WithError(err).WithField("execution_id", executionID).Warn("adhoc cleanup: unable to load execution status")
			}
			continue
		}
		if execution == nil {
			terminalObservedAt = time.Now()
			break
		}
		if IsTerminalExecutionStatus(execution.Status) {
			if terminalObservedAt.IsZero() {
				terminalObservedAt = time.Now()
			}
			if time.Since(terminalObservedAt) >= adhocExecutionRetentionPeriod {
				break
			}
			continue
		}
		terminalObservedAt = time.Time{}
	}

	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cleanupCancel()
	if err := s.repo.DeleteExecution(cleanupCtx, executionID); err != nil && s.log != nil {
		s.log.WithError(err).WithField("execution_id", executionID).Warn("adhoc cleanup: failed to delete execution")
	}
	if err := s.repo.DeleteWorkflow(cleanupCtx, workflowID); err != nil && s.log != nil {
		s.log.WithError(err).WithField("workflow_id", workflowID).Warn("adhoc cleanup: failed to delete workflow")
	}
}

// DescribeExecutionExport returns the current replay export status for an execution.
func (s *WorkflowService) DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*ExecutionExportPreview, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, database.ErrNotFound) {
			return nil, database.ErrNotFound
		}
		return nil, err
	}

	var workflow *database.Workflow
	if wf, wfErr := s.repo.GetWorkflow(ctx, execution.WorkflowID); wfErr == nil {
		workflow = wf
	} else if !errors.Is(wfErr, database.ErrNotFound) {
		return nil, wfErr
	}

	timeline, err := s.GetExecutionTimeline(ctx, executionID)
	if err != nil {
		return nil, err
	}

	capturedFrames := len(timeline.Frames)
	assetCount := 0
	for _, frame := range timeline.Frames {
		if frame.Screenshot != nil {
			assetCount++
		}
		assetCount += len(frame.Artifacts)
	}
	totalDurationMs := 0
	if timeline != nil {
		for _, frame := range timeline.Frames {
			if frame.TotalDurationMs > 0 {
				totalDurationMs += frame.TotalDurationMs
			} else if frame.DurationMs > 0 {
				totalDurationMs += frame.DurationMs
			}
		}
	}
	specID := execution.ID.String()

	if capturedFrames == 0 {
		status := strings.ToLower(strings.TrimSpace(execution.Status))
		previewStatus := "pending"
		message := "Replay export pending – timeline frames not captured yet"
		if status == "failed" {
			previewStatus = "unavailable"
			message = "Replay export unavailable – execution failed before capturing any steps"
		} else if status == "completed" {
			previewStatus = "unavailable"
			message = "Replay export unavailable – workflow finished without timeline frames"
		}
		preview := &ExecutionExportPreview{
			ExecutionID:         execution.ID,
			SpecID:              specID,
			Status:              previewStatus,
			Message:             message,
			CapturedFrameCount:  capturedFrames,
			AvailableAssetCount: assetCount,
			TotalDurationMs:     totalDurationMs,
			Package:             nil,
		}
		if s.log != nil {
			s.log.WithFields(logrus.Fields{
				"execution_id":      execution.ID,
				"workflow_id":       execution.WorkflowID,
				"export_status":     previewStatus,
				"captured_frames":   capturedFrames,
				"available_assets":  assetCount,
				"timeline_total_ms": totalDurationMs,
			}).Debug("DescribeExecutionExport returning preview")
		}
		return preview, nil
	}

	exportPackage, err := BuildReplayMovieSpec(execution, workflow, timeline)
	if err != nil {
		return nil, err
	}

	frameCount := exportPackage.Summary.FrameCount
	if frameCount == 0 {
		frameCount = len(timeline.Frames)
	}
	message := fmt.Sprintf("Replay export ready (%d frames, %dms)", frameCount, exportPackage.Summary.TotalDurationMs)
	assetCount = len(exportPackage.Assets)
	if exportPackage.Summary.TotalDurationMs > 0 {
		totalDurationMs = exportPackage.Summary.TotalDurationMs
	}
	if exportPackage.Execution.ExecutionID != uuid.Nil {
		specID = exportPackage.Execution.ExecutionID.String()
	}

	preview := &ExecutionExportPreview{
		ExecutionID:         execution.ID,
		SpecID:              specID,
		Status:              "ready",
		Message:             message,
		CapturedFrameCount:  frameCount,
		AvailableAssetCount: assetCount,
		TotalDurationMs:     totalDurationMs,
		Package:             exportPackage,
	}

	if s.log != nil {
		s.log.WithFields(logrus.Fields{
			"execution_id":      execution.ID,
			"workflow_id":       execution.WorkflowID,
			"export_status":     "ready",
			"captured_frames":   frameCount,
			"available_assets":  assetCount,
			"timeline_total_ms": totalDurationMs,
		}).Debug("DescribeExecutionExport returning preview")
	}

	return preview, nil
}

// GetExecutionScreenshots gets screenshots for an execution
func (s *WorkflowService) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return s.repo.GetExecutionScreenshots(ctx, executionID)
}

// GetExecution gets an execution by ID
func (s *WorkflowService) GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error) {
	return s.repo.GetExecution(ctx, id)
}

// ListExecutions lists executions with optional workflow filtering
func (s *WorkflowService) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return s.repo.ListExecutions(ctx, workflowID, limit, offset)
}

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

	// Broadcast cancellation
	s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
		Type:        "cancelled",
		ExecutionID: execution.ID,
		Status:      "cancelled",
		Progress:    execution.Progress,
		CurrentStep: execution.CurrentStep,
		Message:     "Execution cancelled by user",
	})

	s.log.WithField("execution_id", executionID).Info("Execution stopped by user request")
	return nil
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

	s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
		Type:        "progress",
		ExecutionID: execution.ID,
		Status:      "running",
		Progress:    0,
		CurrentStep: "Initializing",
		Message:     "Workflow execution started",
	})

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

		s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
			Type:        "completed",
			ExecutionID: execution.ID,
			Status:      "completed",
			Progress:    100,
			CurrentStep: "Completed",
			Message:     "Workflow completed successfully",
			Data: map[string]any{
				"success": true,
				"result":  execution.Result,
			},
		})

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

		s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
			Type:        "cancelled",
			ExecutionID: execution.ID,
			Status:      "cancelled",
			Progress:    execution.Progress,
			CurrentStep: execution.CurrentStep,
			Message:     "Workflow execution cancelled",
		})

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

		s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
			Type:        "failed",
			ExecutionID: execution.ID,
			Status:      "failed",
			Progress:    execution.Progress,
			CurrentStep: execution.CurrentStep,
			Message:     "Execution timed out",
		})

		publishLifecycle(autocontracts.EventKindExecutionFailed, map[string]any{"status": "failed", "error": execution.Error.String})

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

		s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
			Type:        "failed",
			ExecutionID: execution.ID,
			Status:      "failed",
			Progress:    execution.Progress,
			CurrentStep: execution.CurrentStep,
			Message:     "Workflow execution failed: " + err.Error(),
		})

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

func (s *WorkflowService) startExecutionRunner(execution *database.Execution, workflow *database.Workflow) {
	ctx, cancel := context.WithCancel(context.Background())
	s.storeExecutionCancel(execution.ID, cancel)
	go s.executeWorkflowAsync(ctx, execution, workflow)
}

func (s *WorkflowService) storeExecutionCancel(executionID uuid.UUID, cancel context.CancelFunc) {
	if cancel == nil {
		return
	}
	s.executionCancels.Store(executionID, cancel)
}

func (s *WorkflowService) cancelExecutionByID(executionID uuid.UUID) {
	value, ok := s.executionCancels.LoadAndDelete(executionID)
	if !ok {
		return
	}
	if cancel, valid := value.(context.CancelFunc); valid && cancel != nil {
		cancel()
	}
}

// unsupportedAutomationNodes returns node types that the new automation
// executor cannot yet orchestrate. Results are used for logging/alerting only;
// execution continues on the automation stack.
func unsupportedAutomationNodes(flowDefinition database.JSONMap) []string {
	if flowDefinition == nil {
		return nil
	}

	nodes, ok := flowDefinition["nodes"].([]any)
	if !ok || len(nodes) == 0 {
		return nil
	}

	supportedLoopTypes := map[string]struct{}{
		"repeat":  {},
		"foreach": {},
		"forEach": {},
		"while":   {},
	}

	unsupported := map[string]struct{}{}
	add := func(kind string) {
		if strings.TrimSpace(kind) == "" {
			return
		}
		unsupported[kind] = struct{}{}
	}

	for _, raw := range nodes {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		nodeType := strings.ToLower(strings.TrimSpace(extractString(node, "type")))
		data, _ := node["data"].(map[string]any)

		switch nodeType {
		case "loop":
			loopType := strings.TrimSpace(extractString(data, "loopType"))
			if loopType == "" {
				add("loop:missing_type")
				continue
			}
			if _, ok := supportedLoopTypes[loopType]; !ok {
				add(fmt.Sprintf("loop:%s", strings.ToLower(loopType)))
			}
		}
	}

	if len(unsupported) == 0 {
		return nil
	}

	out := make([]string, 0, len(unsupported))
	for kind := range unsupported {
		out = append(out, kind)
	}
	sort.Strings(out)
	return out
}

func extractString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// executeWithAutomationEngine is the sole execution path; the legacy
// browserless.Client is quarantined for parity tests and is not invoked here.
func (s *WorkflowService) executeWithAutomationEngine(ctx context.Context, execution *database.Execution, workflow *database.Workflow, selection autoengine.SelectionConfig, eventSink autoevents.Sink) error {
	if s == nil {
		return errors.New("workflow service not configured")
	}

	engineName := selection.Resolve("")

	compiler := s.planCompiler
	if compiler == nil {
		compiler = autoexecutor.PlanCompilerForEngine(engineName)
	}

	plan, _, err := autoexecutor.BuildContractsPlanWithCompiler(ctx, execution.ID, workflow, compiler)
	if err != nil {
		return err
	}

	if s.executor == nil {
		s.executor = autoexecutor.NewSimpleExecutor(nil)
	}
	if s.engineFactory == nil {
		factory, engErr := autoengine.DefaultFactory(s.log)
		if engErr != nil {
			return engErr
		}
		s.engineFactory = factory
	}
	if s.artifactRecorder == nil {
		s.artifactRecorder = autorecorder.NewDBRecorder(s.repo, nil, s.log)
	}

	if eventSink == nil {
		eventSink = autoevents.NewWSHubSink(s.wsHub, s.log, autocontracts.DefaultEventBufferLimits)
	}

	req := autoexecutor.Request{
		Plan:              plan,
		EngineName:        engineName,
		EngineFactory:     s.engineFactory,
		Recorder:          s.artifactRecorder,
		EventSink:         eventSink,
		HeartbeatInterval: 2 * time.Second,
		WorkflowResolver:  s.repo,
		PlanCompiler:      compiler,
		MaxSubflowDepth:   5,
	}
	return s.executor.Execute(ctx, req)
}

type closableEventSink interface {
	CloseExecution(uuid.UUID)
}

func closeEventSink(sink autoevents.Sink, executionID uuid.UUID) {
	if closable, ok := sink.(closableEventSink); ok && closable != nil {
		closable.CloseExecution(executionID)
	}
}
