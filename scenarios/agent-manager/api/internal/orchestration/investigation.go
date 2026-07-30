// Package orchestration provides helpers for investigation runs.
//
// Investigation runs are normal runs tagged for analysis of other runs.
// They use a dedicated profile key and standard task/run creation flow.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/findings"
	"agent-manager/internal/runreport"

	"github.com/google/uuid"
)

const (
	// investigationWorkflowOwner/Key select the self-registered declared
	// workflow that drives the investigate → await-approval → apply flow. Its
	// definition lives in scenarios/agent-manager/.vrooli/agent-manager/investigate.json
	// and is activated at startup through the self-registration seam.
	investigationWorkflowOwner  = agentManagerSelfScenario
	investigationWorkflowKey    = "agent-manager/investigate"
	investigationApprovalSignal = "investigation.approval"

	// Node ids inside investigate.json used to recover the dispatched runs for
	// the REST/UI surface.
	investigationInvestigateNodeID = "investigate"
	investigationApplyNodeID       = "apply"

	// maxInvestigationContextBytes caps the rendered context snapshot that
	// becomes the workflow input. The run node renders this input into the agent
	// prompt, which the runtime bounds at workflowruntime.MaxRenderedPromptBytes
	// (64 KiB); we cap below that to leave room for the skill instructions.
	maxInvestigationContextBytes = 56000
)

// buildInvestigationContextAttachment creates a human-readable context attachment
// describing the investigation scope and how to fetch additional data.
func buildInvestigationContextAttachment(projectRoot string, scopePaths []string, runIDs []uuid.UUID) domain.ContextAttachment {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Project Root**: `%s`\n", projectRoot))

	if len(scopePaths) > 0 {
		sb.WriteString("**Scope Paths**:\n")
		for _, p := range scopePaths {
			sb.WriteString(fmt.Sprintf("- `%s`\n", p))
		}
	}

	sb.WriteString(fmt.Sprintf("\n**Runs Under Investigation**: %d\n", len(runIDs)))
	for _, id := range runIDs {
		sb.WriteString(fmt.Sprintf("- `%s`\n", id.String()))
	}

	sb.WriteString("\n*For additional detail beyond the attached context, use: `agent-manager run events <run-id>`*\n")

	return domain.ContextAttachment{
		Type:     "note",
		Key:      "investigation-context",
		Label:    "Investigation Context",
		Content:  sb.String(),
		Format:   "markdown",
		Priority: "high",
		Summary:  fmt.Sprintf("Scope and CLI commands for %d run(s) under investigation", len(runIDs)),
		Tags:     []string{"context", "investigation"},
	}
}

// buildInvestigationMetadataAttachment creates a human-readable context attachment
// with investigation parameters including depth guidance.
func buildInvestigationMetadataAttachment(runIDs []uuid.UUID, depth domain.InvestigationDepth, now time.Time) domain.ContextAttachment {
	var depthGuidance string
	switch depth {
	case domain.InvestigationDepthQuick:
		depthGuidance = "QUICK mode: Rapid failure categorization with minimal spot-checking of primary category. Time target: 2-3 minutes."
	case domain.InvestigationDepthDeep:
		depthGuidance = "DEEP mode: Full categorization with thorough investigation of all applicable categories. Take time to explore codebase, configs, and prompts comprehensively."
	default: // Standard
		depthGuidance = "STANDARD mode: Full categorization with targeted investigation of primary failure category."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Depth**: %s\n", depth))
	sb.WriteString(fmt.Sprintf("**Guidance**: %s\n", depthGuidance))
	sb.WriteString(fmt.Sprintf("**Runs**: %d\n", len(runIDs)))
	sb.WriteString(fmt.Sprintf("**Investigated At**: %s\n", now.UTC().Format(time.RFC3339)))

	return domain.ContextAttachment{
		Type:     "note",
		Key:      "investigation-metadata",
		Label:    "Investigation Metadata",
		Content:  sb.String(),
		Format:   "markdown",
		Priority: "high",
		Summary:  fmt.Sprintf("%s depth investigation of %d run(s)", depth, len(runIDs)),
		Tags:     []string{"metadata", "investigation", "config"},
	}
}

// CreateInvestigationRun creates a new investigation run for the given run IDs.
func (o *Orchestrator) CreateInvestigationRun(
	ctx context.Context,
	req CreateInvestigationRequest,
) (*domain.Run, error) {
	if len(req.RunIDs) == 0 {
		return nil, domain.NewValidationError("runIds", "at least one run ID is required")
	}

	// Fetch investigation settings (includes user-editable prompt template)
	settings, err := o.GetInvestigationSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get investigation settings: %w", err)
	}

	// Determine depth - use request depth, fall back to settings default, then standard
	depth := req.Depth
	if depth == "" {
		depth = settings.DefaultDepth
	}
	if depth == "" {
		depth = domain.InvestigationDepthStandard
	}
	if !depth.IsValid() {
		return nil, domain.NewValidationError("depth", "must be 'quick', 'standard', or 'deep'")
	}

	// Use explicit ProjectRoot from request, falling back to config default
	projectRoot := strings.TrimSpace(req.ProjectRoot)
	if projectRoot == "" {
		projectRoot = o.config.DefaultProjectRoot
	}

	// Build attachments - all dynamic data goes here, NOT in the prompt.
	// Depth controls which attachments are included (quick = fewer, deep = all).
	attachments, err := o.buildInvestigationAttachments(ctx, req.RunIDs, req.CustomContext, depth)
	if err != nil {
		return nil, err
	}

	// Add investigation metadata attachment (depth, run IDs, etc.)
	metadataAttachment := buildInvestigationMetadataAttachment(req.RunIDs, depth, o.now())
	attachments = append([]domain.ContextAttachment{metadataAttachment}, attachments...)

	// Add investigation context attachment (explicit project root and scope paths)
	investigationCtx := buildInvestigationContextAttachment(projectRoot, req.ScopePaths, req.RunIDs)
	attachments = append([]domain.ContextAttachment{investigationCtx}, attachments...)

	// The depth-gated attachment builders now produce the typed workflow input
	// snapshot (rendered markdown), not a prompt string. The declared workflow's
	// investigate run node resolves its prompt from the prompt-manager skill via
	// promptRef and renders this context into it.
	renderedContext := renderInvestigationContext(attachments)

	runIDs := make([]string, len(req.RunIDs))
	for i, id := range req.RunIDs {
		runIDs[i] = id.String()
	}
	input, err := json.Marshal(map[string]any{
		"context":     renderedContext,
		"depth":       string(depth),
		"runIds":      runIDs,
		"projectRoot": projectRoot,
	})
	if err != nil {
		return nil, err
	}

	execution, err := o.StartWorkflowExecution(ctx, StartWorkflowExecutionRequest{
		Owner:          investigationWorkflowOwner,
		WorkflowKey:    investigationWorkflowKey,
		Input:          input,
		IdempotencyKey: "investigation/" + uuid.NewString(),
	})
	if err != nil {
		return nil, err
	}

	run, err := o.workflowNodeRun(ctx, execution.ID, investigationInvestigateNodeID)
	if err != nil {
		return nil, err
	}
	return o.attachRunActions(ctx, run), nil
}

// CreateInvestigationApplyRun records the operator's approval decision on a
// completed investigation and, when the decision is "completed", launches the
// declared apply run bound to the structured investigation findings and the
// approved-recommendation selection. Rejection/abstention terminate the
// workflow without an apply run. There is no separate apply task or prompt
// assembly here: approval is a signal into the investigate workflow, and the
// apply node resolves its prompt and structured bindings from the definition.
func (o *Orchestrator) CreateInvestigationApplyRun(
	ctx context.Context,
	req CreateInvestigationApplyRequest,
) (*domain.Run, error) {
	investigationRunID := req.InvestigationRunID

	run, err := o.GetRun(ctx, investigationRunID)
	if err != nil {
		return nil, err
	}
	if allowed, reason := domain.CanApplyInvestigationRun(run, o.investigationTagAllowlist(ctx)); !allowed {
		return nil, domain.NewValidationError("investigationRunId", reason)
	}

	executionID, err := o.workflowExecutions.ExecutionIDForRun(ctx, investigationRunID)
	if err != nil {
		return nil, err
	}
	if executionID == uuid.Nil {
		return nil, domain.NewValidationError("investigationRunId", "run is not part of an investigation workflow")
	}

	decision := strings.TrimSpace(req.Decision)
	if decision == "" {
		decision = "completed"
	}
	if o.findings != nil {
		if err := o.findings.SetDecision(ctx, investigationRunID, decision); err != nil {
			return nil, err
		}
	}

	selected := req.Selected
	if selected == nil {
		selected = []string{}
	}
	note := req.CustomContext
	const maxApprovalContextBytes = 8 * 1024
	if len(note) > maxApprovalContextBytes {
		note = note[:maxApprovalContextBytes]
	}
	payload, err := json.Marshal(map[string]any{
		"decision": decision,
		"selected": selected,
		"note":     note,
	})
	if err != nil {
		return nil, err
	}

	if _, err = o.SignalWorkflowExecution(ctx, WorkflowExecutionSignalRequest{
		ExecutionID:    executionID,
		Signal:         investigationApprovalSignal,
		Payload:        payload,
		IdempotencyKey: "investigation-approval/" + executionID.String(),
	}); err != nil {
		return nil, err
	}

	// Only the "completed" decision launches an apply run; rejection/abstention
	// terminate the workflow, so we hand the investigation run back unchanged.
	if decision != "completed" {
		return o.attachRunActions(ctx, run), nil
	}

	applyRun, err := o.workflowNodeRun(ctx, executionID, investigationApplyNodeID)
	if err != nil {
		return nil, err
	}
	return o.attachRunActions(ctx, applyRun), nil
}

// workflowNodeRun resolves the run dispatched for the newest attempt of the
// named node through the first-class execution-runs projection. It backs the
// REST/UI surface, which still returns the investigation and apply runs even
// though they are workflow node attempts rather than directly created runs.
func (o *Orchestrator) workflowNodeRun(ctx context.Context, executionID uuid.UUID, nodeID string) (*domain.Run, error) {
	attempts, err := o.ListWorkflowExecutionRuns(ctx, executionID)
	if err != nil {
		return nil, err
	}
	var latest *domain.WorkflowNodeAttempt
	for _, attempt := range attempts {
		if attempt.NodeID != nodeID || attempt.RunID == nil {
			continue
		}
		if latest == nil || attempt.CreatedAt.After(latest.CreatedAt) {
			latest = attempt
		}
	}
	if latest == nil {
		return nil, domain.NewStateError("WorkflowExecution", "running", "node-run",
			fmt.Sprintf("node %q has not dispatched a run yet", nodeID))
	}
	return o.GetRun(ctx, *latest.RunID)
}

// renderInvestigationContext flattens the depth-gated context attachments into a
// single markdown block that becomes the workflow input. Image attachments are
// skipped (they carry no renderable text). The result is capped at
// maxInvestigationContextBytes so the run node's rendered prompt stays within
// the runtime's prompt-size bound.
func renderInvestigationContext(attachments []domain.ContextAttachment) string {
	var sb strings.Builder
	for _, att := range attachments {
		if strings.TrimSpace(att.Content) == "" {
			continue
		}
		label := att.Label
		if label == "" {
			label = att.Key
		}
		sb.WriteString("\n## ")
		sb.WriteString(label)
		sb.WriteString("\n\n")
		sb.WriteString(att.Content)
		sb.WriteString("\n")
	}
	rendered := sb.String()
	if len(rendered) > maxInvestigationContextBytes {
		rendered = rendered[:maxInvestigationContextBytes] + "\n\n... (context truncated to fit the run prompt budget)\n"
	}
	return rendered
}

func (o *Orchestrator) investigationTagAllowlist(ctx context.Context) []domain.InvestigationTagRule {
	settings, err := o.GetInvestigationSettings(ctx)
	if err != nil || settings == nil {
		return domain.DefaultInvestigationTagAllowlist()
	}
	return domain.NormalizeInvestigationTagAllowlist(settings.InvestigationTagAllowlist)
}

func (o *Orchestrator) createInvestigationTask(
	ctx context.Context,
	titlePrefix string,
	prompt string,
	attachments []domain.ContextAttachment,
	projectRoot string,
) (*domain.Task, error) {
	now := o.now()
	if projectRoot == "" {
		projectRoot = strings.TrimSpace(o.config.DefaultProjectRoot)
	}
	task := &domain.Task{
		ID:                 uuid.New(),
		Title:              fmt.Sprintf("%s %s", titlePrefix, taskShortID()),
		Description:        prompt,
		ScopePath:          ".",
		ProjectRoot:        projectRoot,
		Status:             domain.TaskStatusQueued,
		ContextAttachments: attachments,
		CreatedAt:          now,
		UpdatedAt:          now,
		CreatedBy:          "agent-manager",
	}

	return o.CreateTask(ctx, task)
}

func (o *Orchestrator) buildInvestigationAttachments(
	ctx context.Context,
	runIDs []uuid.UUID,
	customContext string,
	depth domain.InvestigationDepth,
) ([]domain.ContextAttachment, error) {
	// The shared report contains every cheap discriminator but intentionally no
	// message or unified-diff payload. Depth remains workflow guidance rather
	// than selecting different evidence sets, which makes a quick diagnosis as
	// trustworthy as a deep one and avoids budget-dependent blind spots.
	attachments := make([]domain.ContextAttachment, 0, len(runIDs)+2)
	reports := make([]*runreport.RunReport, 0, len(runIDs))

	for _, runID := range runIDs {
		report, err := o.BuildRunReport(ctx, runID)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
		attachments = append(attachments, domain.ContextAttachment{Type: "note", Key: "run-report-" + shortID(runID), Label: "Run Report " + shortID(runID), Content: runreport.Text(report), Format: "markdown", Priority: "high", Summary: "Bounded diagnostics; use run inspection commands for payloads", Tags: []string{"run", "report", "investigation"}})
		if o.findings != nil {
			prior, listErr := o.findings.List(ctx, findings.Filter{RunID: &runID, Limit: 20})
			if listErr == nil && len(prior) > 0 {
				attachments = append(attachments, recurrenceAttachment(runID, prior))
			}
		}
	}
	// Put the compact cohort brief ahead of the per-run projections. An
	// investigator therefore starts with ranked recurrence evidence and uses
	// the existing per-run report and explicit CLI drill-down only when a
	// discriminator requires it.
	if len(reports) > 1 {
		cohortBytes, err := json.Marshal(runreport.BuildCohort(reports))
		if err != nil {
			return nil, fmt.Errorf("encode cohort projection: %w", err)
		}
		attachments = append([]domain.ContextAttachment{{
			Type: "note", Key: "cohort-brief", Label: "Cohort Brief",
			Content: string(cohortBytes), Format: "json", Priority: "high",
			Summary: "Bounded ranked cohort signals with representative run IDs",
			Tags:    []string{"cohort", "projection", "investigation"},
		}}, attachments...)
	}

	if strings.TrimSpace(customContext) != "" {
		attachments = append(attachments, domain.ContextAttachment{
			Type:     "note",
			Key:      "user-context",
			Label:    "Additional Context",
			Content:  customContext,
			Format:   "markdown",
			Priority: "medium",
			Summary:  "User-provided additional context for this investigation",
			Tags:     []string{"user", "context"},
		})
	}

	return attachments, nil
}
