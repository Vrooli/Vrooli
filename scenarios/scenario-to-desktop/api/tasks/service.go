package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

	"scenario-to-desktop-api/agentmanager"
	"scenario-to-desktop-api/domain"
	"scenario-to-desktop-api/persistence"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/tasks/fix"
	"scenario-to-desktop-api/tasks/investigate"
)

// PipelineStore defines the interface for accessing pipeline status.
type PipelineStore interface {
	Get(pipelineID string) (*pipeline.Status, bool)
}

// Service orchestrates task execution for both investigation and fix workflows.
type Service struct {
	invStore      *persistence.InvestigationStore
	pipelineStore PipelineStore
	agentSvc      *agentmanager.AgentService
	progressHub   ProgressBroadcaster
	handlers      *HandlerRegistry
}

// NewService creates a new task service with registered handlers.
func NewService(
	invStore *persistence.InvestigationStore,
	pipelineStore PipelineStore,
	agentSvc *agentmanager.AgentService,
	hub ProgressBroadcaster,
) *Service {
	s := &Service{
		invStore:      invStore,
		pipelineStore: pipelineStore,
		agentSvc:      agentSvc,
		progressHub:   hub,
		handlers:      NewHandlerRegistry(),
	}

	// Register handlers
	s.handlers.Register(investigate.NewHandler())
	s.handlers.Register(fix.NewHandler())

	return s
}

// TriggerTask starts a new task (investigate or fix).
func (s *Service) TriggerTask(ctx context.Context, req domain.CreateTaskRequest) (*domain.Investigation, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get handler for task type
	handler, ok := s.handlers.Get(req.TaskType)
	if !ok {
		return nil, fmt.Errorf("unsupported task type: %s", req.TaskType)
	}

	// Validate pipeline exists
	pipelineStatus, ok := s.pipelineStore.Get(req.PipelineID)
	if !ok {
		return nil, fmt.Errorf("pipeline not found: %s", req.PipelineID)
	}

	// Check if agent-manager is available
	if !s.agentSvc.IsAvailable(ctx) {
		return nil, fmt.Errorf("agent-manager is not available; please ensure it is running")
	}

	// Check for existing active investigation
	active, err := s.invStore.GetActive(req.PipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active investigation: %w", err)
	}
	if active != nil {
		return nil, fmt.Errorf("task %s is already in progress", active.ID)
	}

	// For fix tasks, get source investigation
	var sourceInv *domain.Investigation
	var sourceFindings *string
	if req.TaskType == domain.TaskTypeFix {
		if req.SourceInvestigationID == "" {
			return nil, fmt.Errorf("source_investigation_id is required for fix tasks")
		}
		sourceInv, err = s.invStore.GetForPipeline(req.PipelineID, req.SourceInvestigationID)
		if err != nil {
			return nil, fmt.Errorf("failed to get source investigation: %w", err)
		}
		if sourceInv == nil {
			return nil, fmt.Errorf("source investigation not found: %s", req.SourceInvestigationID)
		}
		if sourceInv.Status != domain.InvestigationStatusCompleted {
			return nil, fmt.Errorf("source investigation status is %s, expected completed", sourceInv.Status)
		}
		if sourceInv.Findings == nil || *sourceInv.Findings == "" {
			return nil, fmt.Errorf("source investigation has no findings")
		}
		sourceFindings = sourceInv.Findings
	}

	// Create investigation record
	now := time.Now()
	inv := &domain.Investigation{
		ID:         uuid.New().String(),
		PipelineID: req.PipelineID,
		Status:     domain.InvestigationStatusPending,
		Progress:   0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.invStore.Create(inv); err != nil {
		return nil, fmt.Errorf("failed to create investigation: %w", err)
	}

	// Create cancellation context
	taskCtx, cancel := context.WithCancel(context.Background())
	s.invStore.SetCancel(inv.ID, cancel)

	// Start background task
	go s.runTask(taskCtx, inv.ID, pipelineStatus, &req, handler, sourceFindings)

	return inv, nil
}

// runTask executes the task in the background.
func (s *Service) runTask(
	ctx context.Context,
	invID string,
	pipelineStatus *pipeline.Status,
	req *domain.CreateTaskRequest,
	handler TaskHandler,
	sourceFindings *string,
) {
	// Clean up cancel function when done
	defer s.invStore.ClearCancel(invID)

	// Update status to running
	if err := s.invStore.UpdateStatus(invID, domain.InvestigationStatusRunning); err != nil {
		log.Printf("[task] failed to update status to running: %v", err)
		return
	}

	// Dispatch to appropriate runner
	switch req.TaskType {
	case domain.TaskTypeInvestigate:
		s.runInvestigation(ctx, invID, pipelineStatus, req, handler)
	case domain.TaskTypeFix:
		s.runFixLoop(ctx, invID, pipelineStatus, req, handler, sourceFindings)
	}
}

// runInvestigation executes a single-shot investigation task.
func (s *Service) runInvestigation(
	ctx context.Context,
	invID string,
	pipelineStatus *pipeline.Status,
	req *domain.CreateTaskRequest,
	handler TaskHandler,
) {
	// Broadcast start
	s.broadcastProgress(pipelineStatus.PipelineID, invID, EventInvestigationStarted, 0, "Starting investigation...")

	// Build prompt and context
	input := TaskInput{
		Pipeline:  pipelineStatus,
		Request:   req,
		Iteration: 1,
	}

	result, err := handler.BuildPromptAndContext(ctx, input)
	if err != nil {
		s.handleTaskError(ctx, invID, pipelineStatus.PipelineID, fmt.Sprintf("failed to build prompt: %v", err))
		return
	}

	// Update progress
	s.updateProgress(ctx, invID, pipelineStatus.PipelineID, 10, "Agent executing investigation...")

	// Execute via agent-manager
	agentResult, err := s.executeAgent(ctx, invID, handler.AgentTag(), result.Prompt, result.Attachments)
	if err != nil {
		s.handleTaskError(ctx, invID, pipelineStatus.PipelineID, fmt.Sprintf("agent execution failed: %v", err))
		return
	}

	// Store results
	s.storeInvestigationResults(ctx, invID, pipelineStatus, req, agentResult)
}

// runFixLoop executes the iterative fix loop.
func (s *Service) runFixLoop(
	ctx context.Context,
	invID string,
	pipelineStatus *pipeline.Status,
	req *domain.CreateTaskRequest,
	handler TaskHandler,
	sourceFindings *string,
) {
	// Determine pipeline API URL for verification
	pipelineAPIURL := "http://localhost:15020"

	// Create loop state
	loopState := fix.NewLoopState(fix.DefaultLoopConfig(req.MaxIterations, pipelineAPIURL))

	// Broadcast start
	s.broadcastProgress(pipelineStatus.PipelineID, invID, EventFixStarted, 0, "Starting fix loop...")

	// Main loop
	for loopState.ShouldContinue() {
		select {
		case <-ctx.Done():
			// Task was cancelled
			loopState.FinalStatus = FixStatusUserStopped
			s.storeFixResults(ctx, invID, pipelineStatus, req, loopState, sourceFindings)
			return
		default:
		}

		iteration := loopState.StartIteration()

		// Broadcast iteration start
		msg := fmt.Sprintf("Iteration %d/%d: Diagnosing...", iteration, loopState.Config.MaxIterations)
		s.broadcastProgress(pipelineStatus.PipelineID, invID, EventFixIterationStarted, float64(iteration*10), msg)

		// Build prompt for this iteration
		input := TaskInput{
			Pipeline:           pipelineStatus,
			Request:            req,
			SourceFindings:     sourceFindings,
			Iteration:          iteration,
			PreviousIterations: loopState.Iterations,
		}

		result, err := handler.BuildPromptAndContext(ctx, input)
		if err != nil {
			s.handleTaskError(ctx, invID, pipelineStatus.PipelineID, fmt.Sprintf("failed to build iteration prompt: %v", err))
			return
		}

		// Execute iteration
		iterationTag := fmt.Sprintf("%s-iter%d", handler.AgentTag(), iteration)
		agentResult, err := s.executeAgent(ctx, invID, iterationTag, result.Prompt, result.Attachments)
		if err != nil {
			// Record failed iteration but continue if possible
			record := domain.FixIterationRecord{
				Number:      iteration,
				StartedAt:   time.Now(),
				EndedAt:     time.Now(),
				Outcome:     "error",
				VerifyResult: "skip",
			}
			loopState.RecordIteration(record)
			loopState.FinalStatus = FixStatusTimeout
			s.handleTaskError(ctx, invID, pipelineStatus.PipelineID, fmt.Sprintf("iteration %d failed: %v", iteration, err))
			return
		}

		// Extract and record iteration
		record := fix.ExtractIterationRecord(iteration, agentResult)
		record.StartedAt = time.Now().Add(-time.Duration(agentResult.DurationSeconds) * time.Second)
		record.EndedAt = time.Now()
		loopState.RecordIteration(record)

		// Broadcast iteration result
		s.broadcastProgress(pipelineStatus.PipelineID, invID, EventFixIterationComplete,
			float64(20+iteration*15), fmt.Sprintf("Iteration %d complete: %s", iteration, record.Outcome))

		// Check if we should stop
		shouldContinue, _ := handler.ShouldContinue(ctx, nil, agentResult)
		if !shouldContinue {
			// Check if build artifacts exist to confirm success
			artifactsExist, _ := fix.CheckBuildArtifacts(ctx, pipelineAPIURL, pipelineStatus.PipelineID, 10*time.Second)
			loopState.FinalStatus = fix.DetermineOutcome(loopState, agentResult, artifactsExist)
			break
		}

		// Update loop state in database
		s.updateLoopState(ctx, invID, loopState)
	}

	// Determine final status if not set
	if loopState.FinalStatus == "" {
		if loopState.CurrentIteration >= loopState.Config.MaxIterations {
			loopState.FinalStatus = FixStatusMaxIterations
		}
	}

	// Store final results
	s.storeFixResults(ctx, invID, pipelineStatus, req, loopState, sourceFindings)
}

// executeAgent runs an agent and waits for completion.
func (s *Service) executeAgent(
	ctx context.Context,
	invID, tag, prompt string,
	attachments []*domainpb.ContextAttachment,
) (*AgentResult, error) {
	workingDir := os.Getenv("VROOLI_ROOT")
	if workingDir == "" {
		workingDir = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	// Use a timeout context (1 hour)
	execCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	runID, err := s.agentSvc.ExecuteAsync(execCtx, agentmanager.ExecuteRequest{
		InvestigationID:    invID,
		AdditionalTag:      tag,
		Prompt:             prompt,
		WorkingDir:         workingDir,
		ContextAttachments: attachments,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start agent: %w", err)
	}

	// Store run ID
	if err := s.invStore.UpdateRunID(invID, runID); err != nil {
		log.Printf("[task] failed to store run ID: %v", err)
	}

	// Poll for completion
	return s.pollForCompletion(execCtx, runID)
}

// pollForCompletion polls the agent run until it completes.
func (s *Service) pollForCompletion(ctx context.Context, runID string) (*AgentResult, error) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			run, err := s.agentSvc.GetRunStatus(ctx, runID)
			if err != nil {
				return nil, fmt.Errorf("failed to get run status: %w", err)
			}
			if run == nil {
				return nil, fmt.Errorf("run not found: %s", runID)
			}

			switch run.Status {
			case domainpb.RunStatus_RUN_STATUS_COMPLETE:
				result := &AgentResult{
					RunID:   runID,
					Success: true,
				}
				if run.Summary != nil {
					result.Output = run.Summary.Description
					result.TokensUsed = run.Summary.TokensUsed
					result.CostEstimate = run.Summary.CostEstimate
				}
				if run.StartedAt != nil && run.EndedAt != nil {
					duration := run.EndedAt.AsTime().Sub(run.StartedAt.AsTime())
					result.DurationSeconds = int(duration.Seconds())
				}
				return result, nil

			case domainpb.RunStatus_RUN_STATUS_FAILED:
				return &AgentResult{
					RunID:        runID,
					Success:      false,
					ErrorMessage: run.ErrorMsg,
				}, nil

			case domainpb.RunStatus_RUN_STATUS_CANCELLED:
				return nil, fmt.Errorf("agent run was cancelled")
			}
		}
	}
}

// storeInvestigationResults stores the results of an investigation task.
func (s *Service) storeInvestigationResults(
	ctx context.Context,
	invID string,
	pipelineStatus *pipeline.Status,
	req *domain.CreateTaskRequest,
	result *AgentResult,
) {
	// Find failed stage
	failedStage := ""
	for stageName, stageResult := range pipelineStatus.Stages {
		if stageResult.Status == pipeline.StatusFailed {
			failedStage = stageName
			break
		}
	}

	details := domain.ExtendedInvestigationDetails{
		Source:        "agent-manager",
		RunID:         result.RunID,
		DurationSecs:  result.DurationSeconds,
		TokensUsed:    result.TokensUsed,
		CostEstimate:  result.CostEstimate,
		OperationMode: fmt.Sprintf("investigate:%s", req.Effort),
		TriggerReason: "user_requested",
		FailedStage:   failedStage,
		TaskType:      req.TaskType,
		Focus:         req.Focus,
		Effort:        req.Effort,
	}

	detailsJSON, _ := json.Marshal(details)

	findings := result.Output

	// Handle empty findings
	if findings == "" {
		var errorMsg string
		if result.ErrorMessage != "" {
			errorMsg = fmt.Sprintf("Investigation encountered an error: %s", result.ErrorMessage)
		} else {
			errorMsg = "Agent completed but did not produce any findings."
		}
		if err := s.invStore.UpdateErrorWithDetails(invID, errorMsg, detailsJSON); err != nil {
			log.Printf("[task] failed to store error with details: %v", err)
		}
		s.broadcastProgress(pipelineStatus.PipelineID, invID, EventInvestigationFailed, 0, errorMsg)
		return
	}

	if err := s.invStore.UpdateFindings(invID, findings, detailsJSON); err != nil {
		log.Printf("[task] failed to store findings: %v", err)
	}

	s.broadcastProgress(pipelineStatus.PipelineID, invID, EventInvestigationCompleted, 100, "Investigation complete")
	log.Printf("[task] completed investigation %s (run=%s, duration=%ds, tokens=%d)",
		invID, result.RunID, result.DurationSeconds, result.TokensUsed)
}

// storeFixResults stores the results of a fix task.
func (s *Service) storeFixResults(
	ctx context.Context,
	invID string,
	pipelineStatus *pipeline.Status,
	req *domain.CreateTaskRequest,
	loopState *fix.LoopState,
	sourceFindings *string,
) {
	// Find failed stage
	failedStage := ""
	for stageName, stageResult := range pipelineStatus.Stages {
		if stageResult.Status == pipeline.StatusFailed {
			failedStage = stageName
			break
		}
	}

	details := domain.ExtendedInvestigationDetails{
		Source:        "agent-manager",
		OperationMode: fmt.Sprintf("fix:%s", req.Permissions.String()),
		TriggerReason: "user_requested_fix",
		FailedStage:   failedStage,
		TaskType:      req.TaskType,
		Focus:         req.Focus,
		Permissions:   req.Permissions,
		FixState:      ptrTo(loopState.ToFixIterationState()),
	}
	if req.SourceInvestigationID != "" {
		details.SourceInvestigationID = req.SourceInvestigationID
	}
	if sourceFindings != nil {
		details.SourceFindings = *sourceFindings
	}

	detailsJSON, _ := json.Marshal(details)

	// Build findings summary
	findings := buildFixFindingsSummary(loopState)

	// Determine final status
	switch loopState.FinalStatus {
	case FixStatusSuccess:
		if err := s.invStore.UpdateFindings(invID, findings, detailsJSON); err != nil {
			log.Printf("[task] failed to store fix findings: %v", err)
		}
		s.broadcastProgress(pipelineStatus.PipelineID, invID, EventFixCompleted, 100,
			fix.TerminationReason(loopState.FinalStatus, loopState.CurrentIteration, loopState.Config.MaxIterations))

	default:
		errorMsg := fix.TerminationReason(loopState.FinalStatus, loopState.CurrentIteration, loopState.Config.MaxIterations)
		if err := s.invStore.UpdateErrorWithDetails(invID, errorMsg, detailsJSON); err != nil {
			log.Printf("[task] failed to store fix error: %v", err)
		}
		s.broadcastProgress(pipelineStatus.PipelineID, invID, EventFixFailed, 0, errorMsg)
	}

	log.Printf("[task] completed fix task %s (iterations=%d, status=%s)",
		invID, loopState.CurrentIteration, loopState.FinalStatus)
}

// buildFixFindingsSummary creates a human-readable summary of the fix loop.
func buildFixFindingsSummary(loopState *fix.LoopState) string {
	var sb strings.Builder
	sb.WriteString("# Fix Task Summary\n\n")
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", loopState.FinalStatus))
	sb.WriteString(fmt.Sprintf("**Iterations:** %d/%d\n\n", loopState.CurrentIteration, loopState.Config.MaxIterations))

	for _, iter := range loopState.Iterations {
		sb.WriteString(fmt.Sprintf("## Iteration %d\n", iter.Number))
		if iter.DiagnosisSummary != "" {
			sb.WriteString(fmt.Sprintf("**Diagnosis:** %s\n", iter.DiagnosisSummary))
		}
		if iter.ChangesSummary != "" {
			sb.WriteString(fmt.Sprintf("**Changes:** %s\n", iter.ChangesSummary))
		}
		sb.WriteString(fmt.Sprintf("**Rebuild Triggered:** %v\n", iter.RebuildTriggered))
		sb.WriteString(fmt.Sprintf("**Verification:** %s\n", iter.VerifyResult))
		sb.WriteString(fmt.Sprintf("**Outcome:** %s\n\n", iter.Outcome))
	}

	return sb.String()
}

// updateLoopState updates the loop state in the database.
func (s *Service) updateLoopState(ctx context.Context, invID string, loopState *fix.LoopState) {
	stateJSON, err := loopState.ToJSON()
	if err != nil {
		log.Printf("[task] failed to serialize loop state: %v", err)
		return
	}

	// Update the investigation details with the current loop state
	_ = s.invStore.Update(invID, func(inv *domain.Investigation) {
		inv.Details = stateJSON
	})
}

// handleTaskError updates the investigation with an error.
func (s *Service) handleTaskError(ctx context.Context, invID, pipelineID, errorMsg string) {
	if err := s.invStore.UpdateError(invID, errorMsg); err != nil {
		log.Printf("[task] failed to store error: %v", err)
	}
	s.broadcastProgress(pipelineID, invID, EventTaskFailed, 0, errorMsg)
	log.Printf("[task] task %s failed: %s", invID, errorMsg)
}

// updateProgress updates progress and broadcasts.
func (s *Service) updateProgress(ctx context.Context, invID, pipelineID string, progress int, message string) {
	if err := s.invStore.UpdateProgress(invID, progress); err != nil {
		log.Printf("[task] failed to update progress: %v", err)
	}
	s.broadcastProgress(pipelineID, invID, EventTaskProgress, float64(progress), message)
}

// broadcastProgress sends a progress event via the progress hub.
func (s *Service) broadcastProgress(pipelineID, invID, eventType string, progress float64, message string) {
	if s.progressHub != nil {
		s.progressHub.BroadcastInvestigation(pipelineID, invID, eventType, progress, message)
	}
}

// GetTask returns a task by ID.
func (s *Service) GetTask(ctx context.Context, pipelineID, taskID string) (*domain.Investigation, error) {
	return s.invStore.GetForPipeline(pipelineID, taskID)
}

// ListTasks returns tasks for a pipeline.
func (s *Service) ListTasks(ctx context.Context, pipelineID string, limit int) ([]*domain.Investigation, error) {
	return s.invStore.List(pipelineID, limit)
}

// StopTask stops a running task.
func (s *Service) StopTask(ctx context.Context, pipelineID, taskID string) error {
	inv, err := s.invStore.GetForPipeline(pipelineID, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if inv == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if inv.Status != domain.InvestigationStatusRunning && inv.Status != domain.InvestigationStatusPending {
		return fmt.Errorf("task is not running (status=%s)", inv.Status)
	}

	// Cancel the task context
	if cancel := s.invStore.TakeCancel(taskID); cancel != nil {
		cancel()
	}

	// Stop the agent run if we have a run ID
	if inv.AgentRunID != nil && *inv.AgentRunID != "" {
		if err := s.agentSvc.StopRun(ctx, *inv.AgentRunID); err != nil {
			log.Printf("[task] failed to stop agent run: %v", err)
		}
	}

	// Update status
	if err := s.invStore.UpdateStatus(taskID, domain.InvestigationStatusCancelled); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	s.broadcastProgress(pipelineID, taskID, EventTaskStopped, 0, "Task stopped by user")
	return nil
}

// IsAgentAvailable checks if agent-manager is available.
func (s *Service) IsAgentAvailable(ctx context.Context) bool {
	return s.agentSvc.IsAvailable(ctx)
}

// GetAgentManagerURL returns the agent-manager URL.
func (s *Service) GetAgentManagerURL(ctx context.Context) (string, error) {
	return s.agentSvc.ResolveURL(ctx)
}

// ptrTo returns a pointer to the value.
func ptrTo[T any](v T) *T {
	return &v
}
