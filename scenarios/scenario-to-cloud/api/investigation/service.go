package investigation

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

	"scenario-to-cloud/agentmanager"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/persistence"
)

// ProgressBroadcaster defines the interface for broadcasting progress events.
type ProgressBroadcaster interface {
	BroadcastInvestigation(deploymentID, invID, eventType string, progress float64, message string)
}

// Service handles deployment investigation logic.
type Service struct {
	repo        *persistence.Repository
	agentSvc    *agentmanager.AgentService
	progressHub ProgressBroadcaster
}

// NewService creates a new investigation service.
func NewService(
	repo *persistence.Repository,
	agentSvc *agentmanager.AgentService,
	hub ProgressBroadcaster,
) *Service {
	return &Service{
		repo:        repo,
		agentSvc:    agentSvc,
		progressHub: hub,
	}
}

// TriggerInvestigationRequest contains parameters for triggering an investigation.
type TriggerInvestigationRequest struct {
	DeploymentID    string
	AutoFix         bool
	Note            string
	IncludeContexts []string // Context items to include (e.g., "error-info", "deployment-manifest")
}

// TriggerInvestigation starts a new investigation for a failed deployment.
// Returns the investigation immediately; the actual investigation runs in the background.
func (s *Service) TriggerInvestigation(ctx context.Context, req TriggerInvestigationRequest) (*domain.Investigation, error) {
	// Validate deployment exists and has failed
	deployment, err := s.repo.GetDeployment(ctx, req.DeploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}
	if deployment == nil {
		return nil, fmt.Errorf("deployment not found: %s", req.DeploymentID)
	}
	if deployment.Status != domain.StatusFailed {
		return nil, fmt.Errorf("deployment status is %s, expected failed", deployment.Status)
	}

	// Check if agent-manager is available
	if !s.agentSvc.IsAvailable(ctx) {
		return nil, fmt.Errorf("agent-manager is not available; please ensure it is running")
	}

	// Check for existing active investigation
	active, err := s.repo.GetActiveInvestigation(ctx, req.DeploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active investigation: %w", err)
	}
	if active != nil {
		return nil, fmt.Errorf("investigation %s is already in progress", active.ID)
	}

	// Create investigation record
	now := time.Now()
	inv := &domain.Investigation{
		ID:              uuid.New().String(),
		DeploymentID:    req.DeploymentID,
		DeploymentRunID: deployment.RunID,
		Status:          domain.InvestigationStatusPending,
		Progress:        0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreateInvestigation(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to create investigation: %w", err)
	}

	// Start background investigation
	go s.runInvestigation(inv.ID, deployment, req.AutoFix, req.Note, req.IncludeContexts)

	return inv, nil
}

// runInvestigation executes the investigation in the background.
func (s *Service) runInvestigation(
	invID string,
	deployment *domain.Deployment,
	autoFix bool,
	note string,
	includeContexts []string,
) {
	ctx := context.Background()

	// Update status to running
	if err := s.repo.UpdateInvestigationStatus(ctx, invID, domain.InvestigationStatusRunning); err != nil {
		log.Printf("[investigation] failed to update status to running: %v", err)
		return
	}

	// Broadcast investigation started
	s.broadcastProgress(deployment.ID, invID, "investigation_started", 0, "Starting deployment investigation...")

	// Build the investigation prompt and context attachments
	prompt, contextAttachments, err := buildPromptAndContext(deployment, autoFix, note, includeContexts)
	if err != nil {
		s.handleInvestigationError(ctx, invID, deployment.ID, fmt.Sprintf("failed to build prompt: %v", err))
		return
	}

	// Update progress
	if err := s.repo.UpdateInvestigationProgress(ctx, invID, 10); err != nil {
		log.Printf("[investigation] failed to update progress to 10: %v", err)
	}
	s.broadcastProgress(deployment.ID, invID, "investigation_progress", 10, "Agent executing investigation...")

	// Execute via agent-manager
	workingDir := os.Getenv("VROOLI_ROOT")
	if workingDir == "" {
		workingDir = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	// Use a timeout context for the agent execution (1 hour - agents can take a while)
	execCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	runID, err := s.agentSvc.ExecuteAsync(execCtx, agentmanager.ExecuteRequest{
		InvestigationID:    invID,
		AdditionalTag:      "scenario-to-cloud-investigation",
		Prompt:             prompt,
		WorkingDir:         workingDir,
		ContextAttachments: contextAttachments,
	})
	if err != nil {
		s.handleInvestigationError(ctx, invID, deployment.ID, fmt.Sprintf("failed to start agent: %v", err))
		return
	}

	// Store the run ID
	if err := s.repo.UpdateInvestigationRunID(ctx, invID, runID); err != nil {
		log.Printf("[investigation] failed to store run ID: %v", err)
	}

	// Poll for completion
	if err := s.repo.UpdateInvestigationProgress(ctx, invID, 20); err != nil {
		log.Printf("[investigation] failed to update progress to 20: %v", err)
	}
	s.broadcastProgress(deployment.ID, invID, "investigation_progress", 20, "Agent investigating VPS...")

	result, err := s.pollForCompletion(execCtx, invID, deployment.ID, runID)
	if err != nil {
		s.handleInvestigationError(ctx, invID, deployment.ID, fmt.Sprintf("agent execution failed: %v", err))
		return
	}

	// Store findings
	details := domain.InvestigationDetails{
		Source:         "agent-manager",
		RunID:          result.RunID,
		DurationSecs:   result.DurationSeconds,
		TokensUsed:     result.TokensUsed,
		CostEstimate:   result.CostEstimate,
		OperationMode:  "report-only",
		TriggerReason:  "user_requested",
		DeploymentStep: getStringPtr(deployment.ErrorStep),
	}
	if autoFix {
		details.OperationMode = "auto-fix"
	}

	detailsJSON, _ := json.Marshal(details)

	findings := result.Output

	// Handle cases where findings are empty
	if findings == "" {
		var errorMsg string
		if result.ErrorMessage != "" {
			// Agent had an error
			errorMsg = fmt.Sprintf("Investigation encountered an error: %s", result.ErrorMessage)
		} else {
			// Agent completed but produced no output - this is unexpected
			errorMsg = "Agent completed but did not produce any findings. The agent may have encountered an issue or the run summary was not generated."
		}
		// Store the error with details so we preserve execution metadata (tokens, duration, etc.)
		if err := s.repo.UpdateInvestigationErrorWithDetails(ctx, invID, errorMsg, detailsJSON); err != nil {
			log.Printf("[investigation] failed to store error with details: %v", err)
		}
		s.broadcastProgress(deployment.ID, invID, "investigation_failed", 0, errorMsg)
		log.Printf("[investigation] investigation %s failed: %s", invID, errorMsg)
		return
	}

	if err := s.repo.UpdateInvestigationFindings(ctx, invID, findings, detailsJSON); err != nil {
		log.Printf("[investigation] failed to store findings: %v", err)
	}

	// Broadcast completion
	s.broadcastProgress(deployment.ID, invID, "investigation_completed", 100, "Investigation complete")

	log.Printf("[investigation] completed investigation %s (run=%s, duration=%ds, tokens=%d)",
		invID, result.RunID, result.DurationSeconds, result.TokensUsed)
}

// pollForCompletion polls the agent run until it completes or times out.
func (s *Service) pollForCompletion(
	ctx context.Context,
	invID, deploymentID, runID string,
) (*agentmanager.ExecuteResult, error) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	progressSteps := []int{30, 40, 50, 60, 70, 80, 90}
	stepIdx := 0

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

			// Update progress periodically
			if stepIdx < len(progressSteps) {
				progress := progressSteps[stepIdx]
				if err := s.repo.UpdateInvestigationProgress(ctx, invID, progress); err != nil {
					log.Printf("[investigation] failed to update progress to %d: %v", progress, err)
				}
				s.broadcastProgress(deploymentID, invID, "investigation_progress", float64(progress), "Agent investigating...")
				stepIdx++
			}

			// Check for terminal status using the proto enum constants
			switch run.Status {
			case domainpb.RunStatus_RUN_STATUS_COMPLETE:
				result := &agentmanager.ExecuteResult{
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
				return nil, fmt.Errorf("agent run failed: %s", run.ErrorMsg)
			case domainpb.RunStatus_RUN_STATUS_CANCELLED:
				return nil, fmt.Errorf("agent run was cancelled")
			}
		}
	}
}

// handleInvestigationError updates the investigation with an error.
func (s *Service) handleInvestigationError(ctx context.Context, invID, deploymentID, errorMsg string) {
	if err := s.repo.UpdateInvestigationError(ctx, invID, errorMsg); err != nil {
		log.Printf("[investigation] failed to store error: %v", err)
	}
	s.broadcastProgress(deploymentID, invID, "investigation_failed", 0, errorMsg)
	log.Printf("[investigation] investigation %s failed: %s", invID, errorMsg)
}

// broadcastProgress sends a progress event via the progress hub.
func (s *Service) broadcastProgress(deploymentID, invID, eventType string, progress float64, message string) {
	s.progressHub.BroadcastInvestigation(deploymentID, invID, eventType, progress, message)
}

// GetInvestigation retrieves an investigation by ID scoped to a deployment.
func (s *Service) GetInvestigation(ctx context.Context, deploymentID, id string) (*domain.Investigation, error) {
	return s.repo.GetInvestigationForDeployment(ctx, deploymentID, id)
}

// ListInvestigations retrieves investigations for a deployment.
func (s *Service) ListInvestigations(ctx context.Context, deploymentID string, limit int) ([]*domain.Investigation, error) {
	return s.repo.ListInvestigations(ctx, deploymentID, limit)
}

// StopInvestigation attempts to stop a running investigation.
func (s *Service) StopInvestigation(ctx context.Context, deploymentID, id string) error {
	inv, err := s.repo.GetInvestigationForDeployment(ctx, deploymentID, id)
	if err != nil {
		return fmt.Errorf("failed to get investigation: %w", err)
	}
	if inv == nil {
		return fmt.Errorf("investigation not found: %s", id)
	}

	if inv.Status != domain.InvestigationStatusRunning && inv.Status != domain.InvestigationStatusPending {
		return fmt.Errorf("investigation is not running (status=%s)", inv.Status)
	}

	// Stop the agent run if we have a run ID
	if inv.AgentRunID != nil && *inv.AgentRunID != "" {
		if err := s.agentSvc.StopRun(ctx, *inv.AgentRunID); err != nil {
			log.Printf("[investigation] failed to stop agent run: %v", err)
		}
	}

	// Update status to cancelled
	if err := s.repo.UpdateInvestigationStatus(ctx, id, domain.InvestigationStatusCancelled); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Broadcast cancellation
	s.broadcastProgress(inv.DeploymentID, id, "investigation_cancelled", 0, "Investigation cancelled by user")

	return nil
}

func getStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ApplyFixesRequest contains parameters for applying fixes from an investigation.
type ApplyFixesRequest struct {
	InvestigationID string
	DeploymentID    string
	Immediate       bool
	Permanent       bool
	Prevention      bool
	Note            string
}

// ApplyFixes spawns an agent to apply selected fixes from a completed investigation.
// Returns a new investigation record tracking the fix application.
func (s *Service) ApplyFixes(ctx context.Context, req ApplyFixesRequest) (*domain.Investigation, error) {
	// Validate at least one fix type is selected
	if !req.Immediate && !req.Permanent && !req.Prevention {
		return nil, fmt.Errorf("at least one fix type must be selected")
	}

	// Get the original investigation
	originalInv, err := s.repo.GetInvestigationForDeployment(ctx, req.DeploymentID, req.InvestigationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get investigation: %w", err)
	}
	if originalInv == nil {
		return nil, fmt.Errorf("investigation not found: %s", req.InvestigationID)
	}

	// Ensure the investigation is completed with findings
	if originalInv.Status != domain.InvestigationStatusCompleted {
		return nil, fmt.Errorf("investigation status is %s, expected completed", originalInv.Status)
	}
	if originalInv.Findings == nil || *originalInv.Findings == "" {
		return nil, fmt.Errorf("investigation has no findings to apply fixes from")
	}

	// Get the deployment for context
	deployment, err := s.repo.GetDeployment(ctx, originalInv.DeploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}
	if deployment == nil {
		return nil, fmt.Errorf("deployment not found: %s", originalInv.DeploymentID)
	}

	// Check if agent-manager is available
	if !s.agentSvc.IsAvailable(ctx) {
		return nil, fmt.Errorf("agent-manager is not available; please ensure it is running")
	}

	// Check for existing active investigation/fix application
	active, err := s.repo.GetActiveInvestigation(ctx, originalInv.DeploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active investigation: %w", err)
	}
	if active != nil {
		return nil, fmt.Errorf("investigation %s is already in progress", active.ID)
	}

	// Create a new investigation record for the fix application
	now := time.Now()
	fixInv := &domain.Investigation{
		ID:              uuid.New().String(),
		DeploymentID:    originalInv.DeploymentID,
		DeploymentRunID: deployment.RunID,
		Status:          domain.InvestigationStatusPending,
		Progress:        0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreateInvestigation(ctx, fixInv); err != nil {
		return nil, fmt.Errorf("failed to create fix investigation: %w", err)
	}

	// Start background fix application
	go s.runFixApplication(fixInv.ID, originalInv, deployment, req)

	return fixInv, nil
}

// runFixApplication executes the fix application in the background.
func (s *Service) runFixApplication(
	fixInvID string,
	originalInv *domain.Investigation,
	deployment *domain.Deployment,
	req ApplyFixesRequest,
) {
	ctx := context.Background()

	// Update status to running
	if err := s.repo.UpdateInvestigationStatus(ctx, fixInvID, domain.InvestigationStatusRunning); err != nil {
		log.Printf("[fix-application] failed to update status to running: %v", err)
		return
	}

	// Broadcast fix started
	s.broadcastProgress(deployment.ID, fixInvID, "fix_started", 0, "Applying selected fixes...")

	// Build the fix prompt
	prompt, err := buildFixPrompt(originalInv, deployment, req)
	if err != nil {
		s.handleInvestigationError(ctx, fixInvID, deployment.ID, fmt.Sprintf("failed to build fix prompt: %v", err))
		return
	}

	// Update progress
	if err := s.repo.UpdateInvestigationProgress(ctx, fixInvID, 10); err != nil {
		log.Printf("[fix-application] failed to update progress to 10: %v", err)
	}
	s.broadcastProgress(deployment.ID, fixInvID, "fix_progress", 10, "Agent applying fixes...")

	// Execute via agent-manager
	workingDir := os.Getenv("VROOLI_ROOT")
	if workingDir == "" {
		workingDir = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	// Use a timeout context for the agent execution (1 hour - agents can take a while)
	execCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	runID, err := s.agentSvc.ExecuteAsync(execCtx, agentmanager.ExecuteRequest{
		InvestigationID: fixInvID,
		AdditionalTag:   "scenario-to-cloud-apply-investigation",
		Prompt:          prompt,
		WorkingDir:      workingDir,
	})
	if err != nil {
		s.handleInvestigationError(ctx, fixInvID, deployment.ID, fmt.Sprintf("failed to start fix agent: %v", err))
		return
	}

	// Store the run ID
	if err := s.repo.UpdateInvestigationRunID(ctx, fixInvID, runID); err != nil {
		log.Printf("[fix-application] failed to store run ID: %v", err)
	}

	// Build fix types string for details (do this early so we can save details even on error)
	var fixTypes []string
	if req.Immediate {
		fixTypes = append(fixTypes, "immediate")
	}
	if req.Permanent {
		fixTypes = append(fixTypes, "permanent")
	}
	if req.Prevention {
		fixTypes = append(fixTypes, "prevention")
	}

	// Poll for completion
	if err := s.repo.UpdateInvestigationProgress(ctx, fixInvID, 20); err != nil {
		log.Printf("[fix-application] failed to update progress to 20: %v", err)
	}
	s.broadcastProgress(deployment.ID, fixInvID, "fix_progress", 20, "Agent working on fixes...")

	result, err := s.pollForCompletion(execCtx, fixInvID, deployment.ID, runID)
	if err != nil {
		// Build details even on error so source_investigation_id is preserved for grouping
		details := domain.InvestigationDetails{
			Source:                "agent-manager",
			RunID:                 runID,
			OperationMode:         "fix-application:" + strings.Join(fixTypes, ","),
			TriggerReason:         "user_requested_fix",
			DeploymentStep:        getStringPtr(deployment.ErrorStep),
			SourceInvestigationID: originalInv.ID,
			SourceFindings:        getStringPtr(originalInv.Findings),
		}
		detailsJSON, _ := json.Marshal(details)
		errorMsg := fmt.Sprintf("fix application failed: %v", err)
		if err := s.repo.UpdateInvestigationErrorWithDetails(ctx, fixInvID, errorMsg, detailsJSON); err != nil {
			log.Printf("[fix-application] failed to store error with details: %v", err)
		}
		s.broadcastProgress(deployment.ID, fixInvID, "fix_failed", 0, errorMsg)
		log.Printf("[fix-application] fix %s failed: %s", fixInvID, errorMsg)
		return
	}

	// Store results
	details := domain.InvestigationDetails{
		Source:                "agent-manager",
		RunID:                 result.RunID,
		DurationSecs:          result.DurationSeconds,
		TokensUsed:            result.TokensUsed,
		CostEstimate:          result.CostEstimate,
		OperationMode:         "fix-application:" + strings.Join(fixTypes, ","),
		TriggerReason:         "user_requested_fix",
		DeploymentStep:        getStringPtr(deployment.ErrorStep),
		SourceInvestigationID: originalInv.ID,
		SourceFindings:        getStringPtr(originalInv.Findings),
	}

	detailsJSON, _ := json.Marshal(details)

	findings := result.Output

	// Handle cases where findings are empty
	if findings == "" {
		var errorMsg string
		if result.ErrorMessage != "" {
			errorMsg = fmt.Sprintf("Fix application encountered an error: %s", result.ErrorMessage)
		} else {
			errorMsg = "Agent completed but did not produce any output. The fixes may not have been applied correctly."
		}
		if err := s.repo.UpdateInvestigationErrorWithDetails(ctx, fixInvID, errorMsg, detailsJSON); err != nil {
			log.Printf("[fix-application] failed to store error with details: %v", err)
		}
		s.broadcastProgress(deployment.ID, fixInvID, "fix_failed", 0, errorMsg)
		log.Printf("[fix-application] fix %s failed: %s", fixInvID, errorMsg)
		return
	}

	if err := s.repo.UpdateInvestigationFindings(ctx, fixInvID, findings, detailsJSON); err != nil {
		log.Printf("[fix-application] failed to store findings: %v", err)
	}

	// Broadcast completion
	s.broadcastProgress(deployment.ID, fixInvID, "fix_completed", 100, "Fixes applied successfully")

	log.Printf("[fix-application] completed fix application %s (run=%s, duration=%ds, tokens=%d)",
		fixInvID, result.RunID, result.DurationSeconds, result.TokensUsed)
}
