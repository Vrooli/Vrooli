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
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

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

	// Include the full deployment manifest for reference
	sb.WriteString("## Deployment Manifest\n")
	sb.WriteString("This is the manifest that was used for deployment. Use it to verify dependencies.\n")
	sb.WriteString("```json\n")
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")
	sb.WriteString(string(manifestJSON))
	sb.WriteString("\n```\n\n")

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

	// Add Vrooli architecture context
	sb.WriteString("## Vrooli Deployment Architecture\n")
	sb.WriteString("Understanding how Vrooli deployments work will help you diagnose the root cause:\n\n")
	sb.WriteString("### How Dependencies Work\n")
	sb.WriteString("1. Each scenario declares its dependencies in `.vrooli/service.json` under `dependencies.resources` and `dependencies.scenarios`\n")
	sb.WriteString("2. The `scenario-dependency-analyzer` scans the scenario and its dependencies to build the manifest\n")
	sb.WriteString("3. The deployment pipeline (`vps_deploy.go`) executes these steps in order:\n")
	sb.WriteString("   - Install and configure Caddy (reverse proxy)\n")
	sb.WriteString("   - Start each resource in `manifest.dependencies.resources` via `vrooli resource start <name>`\n")
	sb.WriteString("   - Start each dependent scenario in `manifest.dependencies.scenarios`\n")
	sb.WriteString("   - Start the target scenario with port assignments\n")
	sb.WriteString("   - Run health checks\n\n")
	sb.WriteString("### Common Failure Patterns\n")
	sb.WriteString("- **Resource missing from manifest**: The scenario's `.vrooli/service.json` may not declare the resource, or the analyzer didn't pick it up\n")
	sb.WriteString("- **Resource in manifest but failed to start**: The `vrooli resource start` command failed on the VPS (check logs in `~/.vrooli/logs/`)\n")
	sb.WriteString("- **Resource started but not ready**: The scenario started before the resource was fully initialized (timing/health check issue)\n")
	sb.WriteString("- **VPS missing Vrooli installation**: The mini-Vrooli install may be incomplete or corrupted\n\n")

	sb.WriteString("## Diagnostic Questions\n")
	sb.WriteString("As you investigate, answer these questions to identify the root cause:\n\n")
	sb.WriteString("1. **Is the failing dependency in the manifest?** Check `dependencies.resources` and `dependencies.scenarios` above\n")
	sb.WriteString("2. **If yes, did the resource/scenario start step succeed?** Check `~/.vrooli/logs/` for resource and scenario start logs\n")
	sb.WriteString("3. **If no, is it declared in the scenario's service.json?** Check `<workdir>/scenarios/<scenario>/.vrooli/service.json`\n")
	sb.WriteString("4. **Is the Vrooli CLI working?** Run `vrooli --version` and `vrooli resource list`\n")
	sb.WriteString("5. **Is this a configuration issue or transient failure?** Would a simple restart fix it, or is there a deeper problem?\n\n")

	sb.WriteString("## Your Task\n")
	sb.WriteString("1. SSH into the VPS to investigate the deployment failure\n")
	sb.WriteString("2. Check relevant logs:\n")
	sb.WriteString("   - Vrooli logs: `~/.vrooli/logs/` (resource and scenario logs)\n")
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
	sb.WriteString("5. Answer the diagnostic questions above\n")
	sb.WriteString("6. Identify the root cause of the failure\n")

	if autoFix {
		sb.WriteString("7. If safe, attempt to fix the issue (restart services, clear disk, etc.)\n")
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
	sb.WriteString("What caused the deployment to fail. Be specific - distinguish between symptoms (e.g., \"postgres not running\") and actual causes (e.g., \"postgres not in manifest\" or \"postgres start command failed\").\n\n")
	sb.WriteString("### Evidence\n")
	sb.WriteString("Logs, error messages, and system state that support your conclusion. Include:\n")
	sb.WriteString("- Answers to the diagnostic questions above\n")
	sb.WriteString("- Relevant log snippets\n")
	sb.WriteString("- Command outputs that confirm the issue\n\n")
	sb.WriteString("### Impact\n")
	sb.WriteString("What is broken or not working as a result.\n\n")
	sb.WriteString("### Immediate Fix\n")
	sb.WriteString("Commands to run RIGHT NOW on this VPS to restore service. These are hotfixes to unblock the current deployment.")
	if autoFix {
		sb.WriteString(" (If you already applied fixes, list what you did.)")
	}
	sb.WriteString("\n\n")
	sb.WriteString("### Permanent Fix\n")
	sb.WriteString("What needs to change in code, configuration, or the deployment manifest so this issue does NOT occur on fresh VPS deployments. This might include:\n")
	sb.WriteString("- Adding missing dependencies to `.vrooli/service.json`\n")
	sb.WriteString("- Fixing the manifest generation process\n")
	sb.WriteString("- Adding health checks or startup delays\n")
	sb.WriteString("- Fixing the VPS setup/installation process\n\n")
	sb.WriteString("### Prevention\n")
	sb.WriteString("Recommendations for monitoring, alerts, or deployment pipeline improvements that would catch this issue earlier or prevent it entirely.\n")

	return sb.String(), nil
}

// GetInvestigation retrieves an investigation by ID scoped to a deployment.
func (s *InvestigationService) GetInvestigation(ctx context.Context, deploymentID, id string) (*domain.Investigation, error) {
	return s.repo.GetInvestigationForDeployment(ctx, deploymentID, id)
}

// ListInvestigations retrieves investigations for a deployment.
func (s *InvestigationService) ListInvestigations(ctx context.Context, deploymentID string, limit int) ([]*domain.Investigation, error) {
	return s.repo.ListInvestigations(ctx, deploymentID, limit)
}

// StopInvestigation attempts to stop a running investigation.
func (s *InvestigationService) StopInvestigation(ctx context.Context, deploymentID, id string) error {
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
func (s *InvestigationService) ApplyFixes(ctx context.Context, req ApplyFixesRequest) (*domain.Investigation, error) {
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
func (s *InvestigationService) runFixApplication(
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
	prompt, err := s.buildFixPrompt(originalInv, deployment, req)
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

// buildFixPrompt constructs the prompt for applying fixes from an investigation.
func (s *InvestigationService) buildFixPrompt(
	originalInv *domain.Investigation,
	deployment *domain.Deployment,
	req ApplyFixesRequest,
) (string, error) {
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

	var sb strings.Builder

	sb.WriteString("# Fix Application Task\n\n")

	// Selected fixes section
	sb.WriteString("## Selected Fixes to Apply\n")
	if req.Immediate {
		sb.WriteString("- [x] **Immediate Fix** - Apply commands to fix the VPS right now\n")
	} else {
		sb.WriteString("- [ ] Immediate Fix - DO NOT APPLY\n")
	}
	if req.Permanent {
		sb.WriteString("- [x] **Permanent Fix** - Apply code/configuration changes\n")
	} else {
		sb.WriteString("- [ ] Permanent Fix - DO NOT APPLY\n")
	}
	if req.Prevention {
		sb.WriteString("- [x] **Prevention** - Implement monitoring/pipeline improvements\n")
	} else {
		sb.WriteString("- [ ] Prevention - DO NOT APPLY\n")
	}
	sb.WriteString("\n")

	// Instructions based on selected fixes
	sb.WriteString("## Instructions\n")
	sb.WriteString("Apply ONLY the selected fix types from the investigation results below.\n\n")

	if req.Immediate {
		sb.WriteString("### For Immediate Fixes\n")
		sb.WriteString("- SSH into the VPS and run the recommended commands\n")
		sb.WriteString("- Verify the fix worked by checking service status\n")
		sb.WriteString("- Report what commands you ran and their results\n\n")
	}

	if req.Permanent {
		sb.WriteString("### For Permanent Fixes\n")
		sb.WriteString("- Make the recommended code or configuration changes\n")
		sb.WriteString("- This may include editing `.vrooli/service.json`, fixing manifest generation, etc.\n")
		sb.WriteString("- Leave changes uncommitted for user review (do NOT commit)\n")
		sb.WriteString("- Report what files you modified and why\n\n")
	}

	if req.Prevention {
		sb.WriteString("### For Prevention\n")
		sb.WriteString("- Implement the recommended monitoring, alerts, or pipeline improvements\n")
		sb.WriteString("- This may involve adding health checks, deployment gates, or documentation\n")
		sb.WriteString("- Report what preventive measures you implemented\n\n")
	}

	// VPS connection info
	sb.WriteString("## VPS Connection\n")
	sb.WriteString("To apply fixes on the VPS, use SSH commands:\n")
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
	sb.WriteString(fmt.Sprintf("- Scenario: %s\n", manifest.Scenario.ID))
	sb.WriteString("\n")

	// User note if provided
	if req.Note != "" {
		sb.WriteString("## User Note\n")
		sb.WriteString(req.Note)
		sb.WriteString("\n\n")
	}

	// Include the original investigation results
	sb.WriteString("## Investigation Results\n")
	sb.WriteString("The following investigation report contains the findings and recommended fixes.\n")
	sb.WriteString("Apply ONLY the fixes that are selected above.\n\n")
	sb.WriteString("<investigation_results>\n")
	sb.WriteString(*originalInv.Findings)
	sb.WriteString("\n</investigation_results>\n\n")

	// Include deployment context
	sb.WriteString("<deployment_context>\n")
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")
	sb.WriteString(string(manifestJSON))
	sb.WriteString("\n</deployment_context>\n\n")

	// Report format
	sb.WriteString("## Report Format\n")
	sb.WriteString("Please provide a structured report with:\n\n")
	sb.WriteString("### Fixes Applied\n")
	sb.WriteString("List each fix you applied, what you did, and the result.\n\n")
	sb.WriteString("### Verification\n")
	sb.WriteString("How you verified each fix worked (command outputs, status checks, etc.)\n\n")
	sb.WriteString("### Issues Encountered\n")
	sb.WriteString("Any problems you ran into while applying fixes, and how you resolved them.\n\n")
	sb.WriteString("### Next Steps\n")
	sb.WriteString("Any remaining manual steps the user needs to take (e.g., review uncommitted changes, restart deployment).\n")

	return sb.String(), nil
}
