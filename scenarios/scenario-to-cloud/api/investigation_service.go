package main

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

	"scenario-to-cloud/agentmanager"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/persistence"
)

// InvestigationService handles deployment investigation logic.
type InvestigationService struct {
	repo        *persistence.Repository
	agentSvc    *agentmanager.AgentService
	progressHub *ProgressHub
}

// NewInvestigationService creates a new investigation service.
func NewInvestigationService(
	repo *persistence.Repository,
	agentSvc *agentmanager.AgentService,
	hub *ProgressHub,
) *InvestigationService {
	return &InvestigationService{
		repo:        repo,
		agentSvc:    agentSvc,
		progressHub: hub,
	}
}

// TriggerInvestigationRequest contains parameters for triggering an investigation.
type TriggerInvestigationRequest struct {
	DeploymentID string
	AutoFix      bool
	Note         string
}

// TriggerInvestigation starts a new investigation for a failed deployment.
// Returns the investigation immediately; the actual investigation runs in the background.
func (s *InvestigationService) TriggerInvestigation(ctx context.Context, req TriggerInvestigationRequest) (*domain.Investigation, error) {
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
		ID:           uuid.New().String(),
		DeploymentID: req.DeploymentID,
		Status:       domain.InvestigationStatusPending,
		Progress:     0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateInvestigation(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to create investigation: %w", err)
	}

	// Start background investigation
	go s.runInvestigation(inv.ID, deployment, req.AutoFix, req.Note)

	return inv, nil
}

// runInvestigation executes the investigation in the background.
func (s *InvestigationService) runInvestigation(
	invID string,
	deployment *domain.Deployment,
	autoFix bool,
	note string,
) {
	ctx := context.Background()

	// Update status to running
	if err := s.repo.UpdateInvestigationStatus(ctx, invID, domain.InvestigationStatusRunning); err != nil {
		log.Printf("[investigation] failed to update status to running: %v", err)
		return
	}

	// Broadcast investigation started
	s.broadcastProgress(deployment.ID, invID, "investigation_started", 0, "Starting deployment investigation...")

	// Build the investigation prompt
	prompt, err := s.buildPrompt(deployment, autoFix, note)
	if err != nil {
		s.handleInvestigationError(ctx, invID, deployment.ID, fmt.Sprintf("failed to build prompt: %v", err))
		return
	}

	// Update progress
	s.repo.UpdateInvestigationProgress(ctx, invID, 10)
	s.broadcastProgress(deployment.ID, invID, "investigation_progress", 10, "Agent executing investigation...")

	// Execute via agent-manager
	workingDir := os.Getenv("VROOLI_ROOT")
	if workingDir == "" {
		workingDir = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	// Use a timeout context for the agent execution (10 minutes)
	execCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	runID, err := s.agentSvc.ExecuteAsync(execCtx, agentmanager.ExecuteRequest{
		InvestigationID: invID,
		Prompt:          prompt,
		WorkingDir:      workingDir,
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
	s.repo.UpdateInvestigationProgress(ctx, invID, 20)
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
func (s *InvestigationService) pollForCompletion(
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
				s.repo.UpdateInvestigationProgress(ctx, invID, progress)
				s.broadcastProgress(deploymentID, invID, "investigation_progress", float64(progress), "Agent investigating...")
				stepIdx++
			}

			// Check for terminal status using the proto enum values
			switch run.Status {
			case 3: // RUN_STATUS_COMPLETE
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
			case 4: // RUN_STATUS_FAILED
				return nil, fmt.Errorf("agent run failed: %s", run.ErrorMsg)
			case 5: // RUN_STATUS_CANCELLED
				return nil, fmt.Errorf("agent run was cancelled")
			}
		}
	}
}

// handleInvestigationError updates the investigation with an error.
func (s *InvestigationService) handleInvestigationError(ctx context.Context, invID, deploymentID, errorMsg string) {
	if err := s.repo.UpdateInvestigationError(ctx, invID, errorMsg); err != nil {
		log.Printf("[investigation] failed to store error: %v", err)
	}
	s.broadcastProgress(deploymentID, invID, "investigation_failed", 0, errorMsg)
	log.Printf("[investigation] investigation %s failed: %s", invID, errorMsg)
}

// broadcastProgress sends a progress event via the progress hub.
func (s *InvestigationService) broadcastProgress(deploymentID, invID, eventType string, progress float64, message string) {
	s.progressHub.Broadcast(deploymentID, ProgressEvent{
		Type:      eventType,
		Step:      "investigation",
		StepTitle: "Investigation",
		Progress:  progress,
		Message:   fmt.Sprintf("[%s] %s", invID[:8], message),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// buildPrompt constructs the investigation prompt with deployment context.
func (s *InvestigationService) buildPrompt(deployment *domain.Deployment, autoFix bool, note string) (string, error) {
	// Parse the manifest to get VPS details
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		return "", fmt.Errorf("failed to parse manifest: %w", err)
	}

	if manifest.Target.VPS == nil {
		return "", fmt.Errorf("deployment has no VPS target")
	}

	vps := manifest.Target.VPS
	sshPort := vps.Port
	if sshPort == 0 {
		sshPort = 22
	}
	sshUser := vps.User
	if sshUser == "" {
		sshUser = "root"
	}

	// Build the prompt
	var sb strings.Builder

	sb.WriteString("# Deployment Investigation\n\n")
	sb.WriteString("## Context\n")
	sb.WriteString(fmt.Sprintf("- Deployment ID: %s\n", deployment.ID))
	sb.WriteString(fmt.Sprintf("- Scenario: %s\n", manifest.Scenario.ID))
	sb.WriteString(fmt.Sprintf("- Deployment Status: %s\n", deployment.Status))

	if autoFix {
		sb.WriteString("- Operation Mode: **auto-fix** (you may attempt safe fixes)\n")
	} else {
		sb.WriteString("- Operation Mode: **report-only** (analyze only, do not make changes)\n")
	}
	sb.WriteString("\n")

	sb.WriteString("## Error Information\n")
	if deployment.ErrorStep != nil {
		sb.WriteString(fmt.Sprintf("- Failed Step: %s\n", *deployment.ErrorStep))
	}
	if deployment.ErrorMessage != nil {
		sb.WriteString(fmt.Sprintf("- Error Message:\n```\n%s\n```\n", *deployment.ErrorMessage))
	}
	sb.WriteString("\n")

	sb.WriteString("## VPS Connection\n")
	sb.WriteString("To investigate the VPS, use SSH commands:\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("ssh -i %s -p %d %s@%s \"<command>\"\n", vps.KeyPath, sshPort, sshUser, vps.Host))
	sb.WriteString("```\n\n")

	sb.WriteString("## Deployment Configuration\n")
	sb.WriteString(fmt.Sprintf("- VPS Host: %s\n", vps.Host))
	sb.WriteString(fmt.Sprintf("- SSH User: %s\n", sshUser))
	sb.WriteString(fmt.Sprintf("- SSH Port: %d\n", sshPort))
	sb.WriteString(fmt.Sprintf("- SSH Key Path: %s\n", vps.KeyPath))
	if vps.Workdir != "" {
		sb.WriteString(fmt.Sprintf("- VPS Workdir: %s\n", vps.Workdir))
	}
	if manifest.Edge.Domain != "" {
		sb.WriteString(fmt.Sprintf("- Domain: %s\n", manifest.Edge.Domain))
	}
	sb.WriteString("\n")

	if len(manifest.Ports) > 0 {
		sb.WriteString("## Expected Ports\n")
		for name, port := range manifest.Ports {
			sb.WriteString(fmt.Sprintf("- %s: %d\n", name, port))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Your Task\n")
	sb.WriteString("1. SSH into the VPS to investigate the deployment failure\n")
	sb.WriteString("2. Check relevant logs:\n")
	sb.WriteString("   - `journalctl -u <service>` for systemd services\n")
	sb.WriteString("   - Docker logs: `docker logs <container>`\n")
	sb.WriteString("   - Application logs in the workdir\n")
	sb.WriteString("3. Verify service status and port bindings:\n")
	sb.WriteString("   - `systemctl status <service>`\n")
	sb.WriteString("   - `docker ps -a`\n")
	sb.WriteString("   - `ss -tlnp | grep <port>`\n")
	sb.WriteString("4. Check system resources:\n")
	sb.WriteString("   - `df -h` for disk space\n")
	sb.WriteString("   - `free -m` for memory\n")
	sb.WriteString("   - `top -bn1 | head -20` for CPU usage\n")
	sb.WriteString("5. Identify the root cause of the failure\n")

	if autoFix {
		sb.WriteString("6. If safe, attempt to fix the issue (restart services, clear disk, etc.)\n")
	}
	sb.WriteString("\n")

	if note != "" {
		sb.WriteString("## User Note\n")
		sb.WriteString(note)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Report Format\n")
	sb.WriteString("Please provide a structured report with:\n\n")
	sb.WriteString("### Root Cause\n")
	sb.WriteString("What caused the deployment to fail\n\n")
	sb.WriteString("### Evidence\n")
	sb.WriteString("Logs, error messages, and system state that support your conclusion\n\n")
	sb.WriteString("### Impact\n")
	sb.WriteString("What is broken or not working as a result\n\n")
	sb.WriteString("### Resolution\n")
	sb.WriteString("Steps to fix the issue")
	if autoFix {
		sb.WriteString(" (or confirmation of what was auto-fixed)")
	}
	sb.WriteString("\n\n")
	sb.WriteString("### Prevention\n")
	sb.WriteString("Recommendations to prevent this issue in the future\n")

	return sb.String(), nil
}

// GetInvestigation retrieves an investigation by ID.
func (s *InvestigationService) GetInvestigation(ctx context.Context, id string) (*domain.Investigation, error) {
	return s.repo.GetInvestigation(ctx, id)
}

// ListInvestigations retrieves investigations for a deployment.
func (s *InvestigationService) ListInvestigations(ctx context.Context, deploymentID string, limit int) ([]*domain.Investigation, error) {
	return s.repo.ListInvestigations(ctx, deploymentID, limit)
}

// StopInvestigation attempts to stop a running investigation.
func (s *InvestigationService) StopInvestigation(ctx context.Context, id string) error {
	inv, err := s.repo.GetInvestigation(ctx, id)
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
