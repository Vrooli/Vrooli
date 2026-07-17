// Package orchestration provides helpers for investigation runs.
//
// Investigation runs are normal runs tagged for analysis of other runs.
// They use a dedicated profile key and standard task/run creation flow.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

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

	investigationEventLimit = 500

	// maxInvestigationContextBytes caps the rendered context snapshot that
	// becomes the workflow input. The run node renders this input into the agent
	// prompt, which the runtime bounds at workflowruntime.MaxRenderedPromptBytes
	// (64 KiB); we cap below that to leave room for the skill instructions.
	maxInvestigationContextBytes = 56000
)

// NOTE: InvestigationDepth type is defined in domain/investigation.go
// Use domain.InvestigationDepth, domain.InvestigationDepthQuick, etc.

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
func buildInvestigationMetadataAttachment(runIDs []uuid.UUID, depth domain.InvestigationDepth) domain.ContextAttachment {
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
	sb.WriteString(fmt.Sprintf("**Investigated At**: %s\n", time.Now().UTC().Format(time.RFC3339)))

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
	metadataAttachment := buildInvestigationMetadataAttachment(req.RunIDs, depth)
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

	selected := req.Selected
	if selected == nil {
		selected = []string{}
	}
	note := req.CustomContext
	if len(note) > 8192 {
		note = note[:8192]
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
	now := time.Now()
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
	// Depth controls which attachments are included to manage context budget:
	//   quick:    run overview + event timeline + custom context
	//   standard: + agent setup + diff
	//   deep:     + historical context
	includeAgentSetup := depth != domain.InvestigationDepthQuick
	includeDiff := depth != domain.InvestigationDepthQuick
	includeHistory := depth == domain.InvestigationDepthDeep

	attachments := make([]domain.ContextAttachment, 0, len(runIDs)*4+3)

	for _, runID := range runIDs {
		run, err := o.GetRun(ctx, runID)
		if err != nil {
			return nil, err
		}

		// Fetch the task for this run (provides description of what the run was supposed to do).
		var task *domain.Task
		task, _ = o.GetTask(ctx, run.TaskID)

		// Fetch the agent profile (provides runner config, tools, etc.).
		var profile *domain.AgentProfile
		if run.AgentProfileID != nil {
			profile, _ = o.GetProfile(ctx, *run.AgentProfileID)
		}

		short := shortID(runID)

		// 1. Human-readable run overview (always included).
		attachments = append(attachments, buildRunOverview(run, task, profile, short))

		// 2. Agent setup context (standard and deep only).
		if includeAgentSetup && profile != nil {
			projectRoot := ""
			if task != nil {
				projectRoot = task.ProjectRoot
			}
			if projectRoot == "" {
				projectRoot = o.config.DefaultProjectRoot
			}
			if att, ok := buildAgentSetupAttachment(profile, projectRoot, short); ok {
				attachments = append(attachments, att)
			}
		}

		// 3. Curated event timeline (always included).
		if o.events != nil {
			events, err := o.GetRunEvents(ctx, runID, event.GetOptions{
				AfterSequence: -1,
				Limit:         investigationEventLimit,
			})
			if err != nil {
				return nil, err
			}
			attachments = append(attachments, buildRunTimeline(events, run, short))
		}

		// 4. Diff (standard and deep only).
		if includeDiff {
			diff, err := o.GetRunDiff(ctx, runID)
			if err == nil && diff != nil {
				diffJSON, err := marshalJSON(diff)
				if err != nil {
					return nil, err
				}
				attachments = append(attachments, domain.ContextAttachment{
					Type:     "note",
					Key:      fmt.Sprintf("run-diff-%s", short),
					Label:    fmt.Sprintf("Run Diff %s", short),
					Content:  diffJSON,
					Format:   "json",
					Priority: "medium",
					Summary:  fmt.Sprintf("Code changes from run %s (%d files, %d bytes)", short, run.ChangedFiles, run.TotalSizeBytes),
					Tags:     []string{"run", "diff", "investigation"},
				})
			}
		}

		// 5. Historical context (deep only).
		if includeHistory && run.AgentProfileID != nil {
			if att, ok := o.buildHistoricalContext(ctx, run, short); ok {
				attachments = append(attachments, att)
			}
		}
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

// =============================================================================
// HUMAN-READABLE ATTACHMENT BUILDERS
// =============================================================================

// buildRunOverview creates a human-readable markdown summary of a run,
// replacing the raw JSON dump of the entire Run struct.
func buildRunOverview(run *domain.Run, task *domain.Task, profile *domain.AgentProfile, short string) domain.ContextAttachment {
	var sb strings.Builder

	// Basic identification.
	sb.WriteString(fmt.Sprintf("**Run ID**: `%s`\n", run.ID))
	if run.Tag != "" {
		sb.WriteString(fmt.Sprintf("**Tag**: `%s`\n", run.Tag))
	}
	sb.WriteString(fmt.Sprintf("**Status**: %s\n", run.Status))

	// Timing.
	if run.StartedAt != nil && run.EndedAt != nil {
		duration := run.EndedAt.Sub(*run.StartedAt).Round(time.Second)
		sb.WriteString(fmt.Sprintf("**Duration**: %s (started %s, ended %s)\n",
			duration, run.StartedAt.Format(time.RFC3339), run.EndedAt.Format(time.RFC3339)))
	} else if run.StartedAt != nil {
		sb.WriteString(fmt.Sprintf("**Started At**: %s\n", run.StartedAt.Format(time.RFC3339)))
	}

	// Exit / error info.
	if run.ExitCode != nil {
		sb.WriteString(fmt.Sprintf("**Exit Code**: %d\n", *run.ExitCode))
	}
	if run.ErrorMsg != "" {
		sb.WriteString(fmt.Sprintf("**Error**: %s\n", run.ErrorMsg))
	}

	// Task context — what was the run supposed to do?
	sb.WriteString("\n### Task\n\n")
	if task != nil {
		sb.WriteString(fmt.Sprintf("**Title**: %s\n", task.Title))
		if task.Description != "" {
			// Truncate very long descriptions (prompts can be huge).
			desc := task.Description
			if len(desc) > 500 {
				desc = desc[:500] + "\n\n... (truncated, use `agent-manager task get " + task.ID.String() + "` for full description)"
			}
			sb.WriteString(fmt.Sprintf("**Description**:\n%s\n", desc))
		}
		sb.WriteString(fmt.Sprintf("**Project Root**: `%s`\n", task.ProjectRoot))
	} else {
		sb.WriteString(fmt.Sprintf("**Task ID**: `%s` (details unavailable)\n", run.TaskID))
	}

	// Progress and resource usage.
	sb.WriteString("\n### Execution Details\n\n")
	sb.WriteString(fmt.Sprintf("**Phase**: %s\n", run.Phase))
	sb.WriteString(fmt.Sprintf("**Progress**: %d%%\n", run.ProgressPercent))
	if run.LastHeartbeat != nil {
		sb.WriteString(fmt.Sprintf("**Last Heartbeat**: %s\n", run.LastHeartbeat.Format(time.RFC3339)))
		if run.EndedAt != nil {
			gap := run.EndedAt.Sub(*run.LastHeartbeat).Round(time.Second)
			sb.WriteString(fmt.Sprintf("**Heartbeat-to-End Gap**: %s\n", gap))
		}
	}

	if run.Summary != nil {
		s := run.Summary
		if s.TurnsUsed > 0 {
			sb.WriteString(fmt.Sprintf("**Turns Used**: %d\n", s.TurnsUsed))
		}
		if s.TokensUsed > 0 {
			sb.WriteString(fmt.Sprintf("**Tokens Used**: %d\n", s.TokensUsed))
		}
		if s.CostEstimate > 0 {
			sb.WriteString(fmt.Sprintf("**Cost Estimate**: $%.4f\n", s.CostEstimate))
		}
		if s.Description != "" {
			sb.WriteString(fmt.Sprintf("**Run Summary**: %s\n", s.Description))
		}
	}

	sb.WriteString(fmt.Sprintf("**Files Changed**: %d\n", run.ChangedFiles))

	// Runner configuration.
	if run.ResolvedConfig != nil {
		c := run.ResolvedConfig
		sb.WriteString("\n### Runner Configuration\n\n")
		sb.WriteString(fmt.Sprintf("**Runner**: %s\n", c.RunnerType))
		if c.Model != "" {
			sb.WriteString(fmt.Sprintf("**Model**: %s\n", c.Model))
		}
		if c.RoleRef != "" {
			sb.WriteString(fmt.Sprintf("**Role**: %s\n", c.RoleRef))
		}
		if c.MaxTurns > 0 {
			sb.WriteString(fmt.Sprintf("**Max Turns**: %d\n", c.MaxTurns))
		}
		if c.Timeout > 0 {
			sb.WriteString(fmt.Sprintf("**Timeout**: %s\n", c.Timeout))
		}
	}

	// Profile reference.
	if profile != nil {
		sb.WriteString("\n### Agent Profile\n\n")
		sb.WriteString(fmt.Sprintf("**Name**: %s\n", profile.Name))
		sb.WriteString(fmt.Sprintf("**Key**: `%s`\n", profile.ProfileKey))
		if profile.Description != "" {
			sb.WriteString(fmt.Sprintf("**Description**: %s\n", profile.Description))
		}
	}

	return domain.ContextAttachment{
		Type:     "note",
		Key:      fmt.Sprintf("run-overview-%s", short),
		Label:    fmt.Sprintf("Run Overview %s", short),
		Content:  sb.String(),
		Format:   "markdown",
		Priority: "high",
		Summary:  fmt.Sprintf("Run %s: %s (status=%s)", short, run.Tag, run.Status),
		Tags:     []string{"run", "overview", "investigation"},
	}
}

// buildRunTimeline creates a curated markdown timeline from run events,
// replacing the raw JSON array of up to 500 event objects.
func buildRunTimeline(events []*domain.RunEvent, run *domain.Run, short string) domain.ContextAttachment {
	var sb strings.Builder

	if len(events) == 0 {
		sb.WriteString("No events recorded for this run.\n")
		return domain.ContextAttachment{
			Type:     "note",
			Key:      fmt.Sprintf("run-timeline-%s", short),
			Label:    fmt.Sprintf("Run Timeline %s", short),
			Content:  sb.String(),
			Format:   "markdown",
			Priority: "medium",
			Summary:  "No events recorded",
			Tags:     []string{"run", "timeline", "investigation"},
		}
	}

	// Compute the base time for relative timestamps.
	baseTime := events[0].Timestamp

	sb.WriteString("| # | Time | Type | Summary |\n")
	sb.WriteString("|---|------|------|---------|\n")

	var errorEvents []string
	var lastFailureIdx int

	for i, evt := range events {
		elapsed := evt.Timestamp.Sub(baseTime).Round(time.Second)
		summary := formatEventSummary(evt)
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n",
			i+1, formatDuration(elapsed), evt.EventType, summary))

		if evt.EventType == domain.EventTypeError {
			lastFailureIdx = i + 1
			errorEvents = append(errorEvents, summary)
		}
	}

	// Failure analysis section.
	if len(errorEvents) > 0 {
		sb.WriteString("\n### Failure Points\n\n")
		for i, errSummary := range errorEvents {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, errSummary))
		}
		if lastFailureIdx > 0 {
			sb.WriteString(fmt.Sprintf("\nLast error occurred at event #%d of %d total events.\n", lastFailureIdx, len(events)))
		}
	}

	// Event statistics.
	stats := computeEventStats(events)
	sb.WriteString("\n### Event Statistics\n\n")
	sb.WriteString(fmt.Sprintf("**Total Events**: %d\n", len(events)))
	if stats.toolCalls > 0 {
		sb.WriteString(fmt.Sprintf("**Tool Calls**: %d (%d succeeded, %d failed)\n",
			stats.toolCalls, stats.toolSuccesses, stats.toolFailures))
	}
	if stats.statusChanges > 0 {
		sb.WriteString(fmt.Sprintf("**Status Changes**: %d\n", stats.statusChanges))
	}
	if stats.errors > 0 {
		sb.WriteString(fmt.Sprintf("**Errors**: %d\n", stats.errors))
	}
	if stats.totalCostUSD > 0 {
		sb.WriteString(fmt.Sprintf("**Total Cost**: $%.4f\n", stats.totalCostUSD))
	}
	if stats.totalTokens > 0 {
		sb.WriteString(fmt.Sprintf("**Total Tokens**: %d (input: %d, output: %d)\n",
			stats.totalTokens, stats.inputTokens, stats.outputTokens))
	}

	sb.WriteString(fmt.Sprintf("\n*For complete event data: `agent-manager run events %s`*\n", run.ID))

	return domain.ContextAttachment{
		Type:     "note",
		Key:      fmt.Sprintf("run-timeline-%s", short),
		Label:    fmt.Sprintf("Run Timeline %s", short),
		Content:  sb.String(),
		Format:   "markdown",
		Priority: "high",
		Summary:  fmt.Sprintf("%d events, %d errors, %d tool calls", len(events), stats.errors, stats.toolCalls),
		Tags:     []string{"run", "timeline", "investigation"},
	}
}

// eventStats aggregates statistics from a run's event stream.
type eventStats struct {
	toolCalls     int
	toolSuccesses int
	toolFailures  int
	statusChanges int
	errors        int
	totalCostUSD  float64
	inputTokens   int
	outputTokens  int
	totalTokens   int
}

func computeEventStats(events []*domain.RunEvent) eventStats {
	var s eventStats
	for _, evt := range events {
		switch evt.EventType {
		case domain.EventTypeToolCall:
			s.toolCalls++
		case domain.EventTypeToolResult:
			if data, ok := evt.Data.(*domain.ToolResultEventData); ok {
				if data.Success {
					s.toolSuccesses++
				} else {
					s.toolFailures++
				}
			}
		case domain.EventTypeStatus:
			s.statusChanges++
		case domain.EventTypeError:
			s.errors++
		case domain.EventTypeMetric:
			if data, ok := evt.Data.(*domain.CostEventData); ok {
				s.totalCostUSD += data.TotalCostUSD
				s.inputTokens += data.InputTokens
				s.outputTokens += data.OutputTokens
				s.totalTokens += data.InputTokens + data.OutputTokens
			}
		}
	}
	return s
}

// formatEventSummary returns a concise human-readable summary for a single event.
func formatEventSummary(evt *domain.RunEvent) string {
	switch data := evt.Data.(type) {
	case *domain.LogEventData:
		msg := data.Message
		if len(msg) > 120 {
			msg = msg[:120] + "..."
		}
		return fmt.Sprintf("[%s] %s", data.Level, msg)

	case *domain.MessageEventData:
		msg := data.Content
		if len(msg) > 120 {
			msg = msg[:120] + "..."
		}
		return fmt.Sprintf("%s: %s", data.Role, msg)

	case *domain.ToolCallEventData:
		inputSummary := summarizeToolInput(data.ToolName, data.Input)
		return fmt.Sprintf("Call `%s`: %s", data.ToolName, inputSummary)

	case *domain.ToolResultEventData:
		if !data.Success {
			errMsg := data.Error
			if len(errMsg) > 100 {
				errMsg = errMsg[:100] + "..."
			}
			return fmt.Sprintf("Result `%s`: FAILED — %s", data.ToolName, errMsg)
		}
		outputLen := len(data.Output)
		if outputLen > 80 {
			return fmt.Sprintf("Result `%s`: OK (%d chars)", data.ToolName, outputLen)
		}
		if outputLen > 0 {
			return fmt.Sprintf("Result `%s`: %s", data.ToolName, data.Output)
		}
		return fmt.Sprintf("Result `%s`: OK", data.ToolName)

	case *domain.StatusEventData:
		if data.Reason != "" {
			return fmt.Sprintf("%s → %s (%s)", data.OldStatus, data.NewStatus, data.Reason)
		}
		return fmt.Sprintf("%s → %s", data.OldStatus, data.NewStatus)

	case *domain.ErrorEventData:
		msg := data.Message
		if len(msg) > 120 {
			msg = msg[:120] + "..."
		}
		if data.Code != "" {
			return fmt.Sprintf("[%s] %s", data.Code, msg)
		}
		return msg

	case *domain.RateLimitEventData:
		return fmt.Sprintf("Rate limit (%s): %s", data.LimitType, data.Message)

	case *domain.CostEventData:
		return fmt.Sprintf("Cost: $%.4f (in=%d, out=%d tokens)", data.TotalCostUSD, data.InputTokens, data.OutputTokens)

	case *domain.ArtifactEventData:
		return fmt.Sprintf("Artifact [%s]: %s", data.Type, data.Path)

	case *domain.ProgressEventData:
		if data.CurrentAction != "" {
			return fmt.Sprintf("Progress: %d%% — %s", data.PercentComplete, data.CurrentAction)
		}
		return fmt.Sprintf("Progress: %d%% (phase=%s)", data.PercentComplete, data.Phase)

	case *domain.MessageDeletedEventData:
		return fmt.Sprintf("Message deleted: %s", data.TargetEventID)

	default:
		// Fallback for unknown payload types.
		if evt.Data != nil {
			raw, _ := json.Marshal(evt.Data)
			s := string(raw)
			if len(s) > 120 {
				s = s[:120] + "..."
			}
			return s
		}
		return "(no data)"
	}
}

// summarizeToolInput extracts the most useful field from tool call input.
func summarizeToolInput(toolName string, input map[string]interface{}) string {
	if input == nil {
		return "(no input)"
	}

	// Common patterns: path, command, file_path, pattern, query.
	for _, key := range []string{"file_path", "path", "command", "pattern", "query", "url"} {
		if v, ok := input[key]; ok {
			s := fmt.Sprintf("%v", v)
			if len(s) > 100 {
				s = s[:100] + "..."
			}
			return s
		}
	}

	// Fallback: show first key-value pair.
	for k, v := range input {
		s := fmt.Sprintf("%s=%v", k, v)
		if len(s) > 100 {
			s = s[:100] + "..."
		}
		return s
	}
	return "(empty)"
}

// formatDuration formats a duration as MM:SS or H:MM:SS.
func formatDuration(d time.Duration) string {
	totalSec := int(d.Seconds())
	if totalSec < 0 {
		totalSec = 0
	}
	hours := totalSec / 3600
	minutes := (totalSec % 3600) / 60
	seconds := totalSec % 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// buildAgentSetupAttachment creates a context attachment with prompt-manager
// entity paths so the investigation agent can explore the agent's configuration.
// Always returns an attachment with profile metadata. If the agent directory
// exists in prompt-manager's store, file paths are included. If not, the
// attachment notes the absence so the investigator knows to look elsewhere
// (e.g., dynamically generated prompts, agent-manager profile settings).
func buildAgentSetupAttachment(profile *domain.AgentProfile, projectRoot string, short string) (domain.ContextAttachment, bool) {
	storeRoot := filepath.Join(projectRoot, "scenarios", "prompt-manager", "store")
	agentDir := filepath.Join(storeRoot, "agents", profile.ProfileKey)

	var sb strings.Builder

	// Always include profile metadata — useful even without on-disk files.
	sb.WriteString(fmt.Sprintf("### Agent Profile: `%s`\n\n", profile.ProfileKey))
	sb.WriteString(fmt.Sprintf("**Name**: %s\n", profile.Name))
	if profile.Description != "" {
		sb.WriteString(fmt.Sprintf("**Description**: %s\n", profile.Description))
	}
	sb.WriteString(fmt.Sprintf("**Role**: %s\n", profile.RoleRef))
	if len(profile.AllowedTools) > 0 {
		sb.WriteString(fmt.Sprintf("**Allowed Tools**: %s\n", strings.Join(profile.AllowedTools, ", ")))
	}
	if len(profile.DeniedTools) > 0 {
		sb.WriteString(fmt.Sprintf("**Denied Tools**: %s\n", strings.Join(profile.DeniedTools, ", ")))
	}
	sb.WriteString("\n")

	agentDirExists := false
	if _, err := os.Stat(agentDir); err == nil {
		agentDirExists = true
		sb.WriteString(fmt.Sprintf("**Agent Directory**: `%s/`\n", agentDir))

		// List known agent files with descriptions.
		agentFiles := []struct {
			name string
			desc string
		}{
			{"agent.json", "Agent metadata and configuration"},
			{"SOUL.md", "Core identity, boundaries, and domain focus"},
			{"AGENTS.md", "Workflow procedures and coordination"},
			{"TOOLS.md", "Available skills and resource access"},
		}
		for _, f := range agentFiles {
			path := filepath.Join(agentDir, f.name)
			if _, err := os.Stat(path); err == nil {
				sb.WriteString(fmt.Sprintf("- `%s` — %s\n", f.name, f.desc))
			}
		}
	} else {
		sb.WriteString(fmt.Sprintf("**No agent directory** at `%s/`\n", agentDir))
		sb.WriteString("This agent's prompt may be generated dynamically (e.g., by a spawn template) rather than stored as files.\n")
		sb.WriteString("Check the task description and runner configuration for how this agent receives its instructions.\n")
	}

	// Discover team memberships by scanning the relations directory.
	relationsDir := filepath.Join(storeRoot, "relations", "team-member")
	suffix := "__" + profile.ProfileKey + ".json"
	if entries, err := os.ReadDir(relationsDir); err == nil {
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), suffix) {
				continue
			}
			// Extract team ID from filename: {team-id}__{agent-id}.json
			teamID := strings.TrimSuffix(entry.Name(), suffix)
			teamDir := filepath.Join(storeRoot, "teams", teamID)
			memberDir := filepath.Join(teamDir, "members", profile.ProfileKey)

			sb.WriteString(fmt.Sprintf("\n### Team: `%s`\n\n", teamID))
			sb.WriteString(fmt.Sprintf("**Team Directory**: `%s/`\n", teamDir))

			// List team files.
			teamFiles := []struct {
				name string
				desc string
			}{
				{"team.json", "Team configuration and spawn mode"},
				{"org.json", "Organizational hierarchy (reporting structure)"},
				{"roles.json", "Role definitions within the team"},
				{filepath.Join("shared", "TEAM.md"), "Team mission, strategy, and deployment model"},
			}
			for _, f := range teamFiles {
				path := filepath.Join(teamDir, f.name)
				if _, err := os.Stat(path); err == nil {
					sb.WriteString(fmt.Sprintf("- `%s` — %s\n", f.name, f.desc))
				}
			}

			// List member files.
			if _, err := os.Stat(memberDir); err == nil {
				sb.WriteString(fmt.Sprintf("\n**Member Directory**: `%s/`\n", memberDir))

				memberFiles := []struct {
					name string
					desc string
				}{
					{"heartbeat.json", "Execution schedule and last execution status"},
					{"HEARTBEAT.md", "Checklist of tasks for scheduled runs"},
					{"RESPONSIBILITIES.md", "Role-specific duties and deliverables"},
				}
				for _, f := range memberFiles {
					path := filepath.Join(memberDir, f.name)
					if _, err := os.Stat(path); err == nil {
						sb.WriteString(fmt.Sprintf("- `%s` — %s\n", f.name, f.desc))
					}
				}

				// Note log directory if it exists.
				logsDir := filepath.Join(memberDir, "logs")
				if entries, err := os.ReadDir(logsDir); err == nil && len(entries) > 0 {
					sb.WriteString(fmt.Sprintf("- `logs/` — %d execution log(s)\n", len(entries)))
				}
			}

			sb.WriteString(fmt.Sprintf("\n**Relation**: `%s`\n", filepath.Join(relationsDir, entry.Name())))
		}
	}

	if agentDirExists {
		sb.WriteString("\n*Read these files to understand the agent's identity, instructions, tools, team context, and scheduled responsibilities.*\n")
	}

	return domain.ContextAttachment{
		Type:     "note",
		Key:      fmt.Sprintf("agent-setup-%s", short),
		Label:    fmt.Sprintf("Agent Setup %s", short),
		Content:  sb.String(),
		Format:   "markdown",
		Priority: "high",
		Summary:  fmt.Sprintf("Profile metadata and prompt-manager paths for agent %q", profile.ProfileKey),
		Tags:     []string{"agent", "setup", "investigation"},
	}, true
}

// buildHistoricalContext queries recent runs with the same agent profile
// to provide pattern comparison for the investigation agent.
// Returns (attachment, true) if historical data is found, (zero, false) otherwise.
func (o *Orchestrator) buildHistoricalContext(ctx context.Context, currentRun *domain.Run, short string) (domain.ContextAttachment, bool) {
	if currentRun.AgentProfileID == nil {
		return domain.ContextAttachment{}, false
	}

	// Fetch recent runs with the same agent profile (up to 10 for pattern analysis).
	recentRuns, err := o.ListRuns(ctx, RunListOptions{
		ListOptions:    ListOptions{Limit: 10},
		AgentProfileID: currentRun.AgentProfileID,
	})
	if err != nil || len(recentRuns) == 0 {
		return domain.ContextAttachment{}, false
	}

	// Filter out the current run and investigation/apply runs.
	var history []*domain.Run
	for _, r := range recentRuns {
		if r.ID == currentRun.ID {
			continue
		}
		// Skip investigation and apply runs — they're meta-runs, not comparable.
		if strings.Contains(r.Tag, "investigation") {
			continue
		}
		history = append(history, r)
	}

	if len(history) == 0 {
		return domain.ContextAttachment{}, false
	}

	var sb strings.Builder
	sb.WriteString("### Recent Runs (same agent profile)\n\n")
	sb.WriteString("| Run ID | Date | Tag | Status | Duration | Error |\n")
	sb.WriteString("|--------|------|-----|--------|----------|-------|\n")

	var successCount, failCount int
	for _, r := range history {
		runShort := shortID(r.ID)
		date := r.CreatedAt.Format("Jan 02 15:04")
		duration := "—"
		if r.StartedAt != nil && r.EndedAt != nil {
			duration = r.EndedAt.Sub(*r.StartedAt).Round(time.Second).String()
		}
		errMsg := "—"
		if r.ErrorMsg != "" {
			errMsg = r.ErrorMsg
			if len(errMsg) > 60 {
				errMsg = errMsg[:60] + "..."
			}
		}
		tag := r.Tag
		if len(tag) > 30 {
			tag = tag[:30] + "..."
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
			runShort, date, tag, r.Status, duration, errMsg))

		if r.Status == domain.RunStatusComplete {
			successCount++
		} else if r.Status == domain.RunStatusFailed {
			failCount++
		}
	}

	// Pattern summary.
	total := len(history)
	sb.WriteString(fmt.Sprintf("\n**Pattern**: %d of last %d runs succeeded, %d failed\n", successCount, total, failCount))
	if failCount > 0 && successCount > 0 {
		sb.WriteString("*Compare successful vs failed runs to identify what changed.*\n")
	} else if failCount == total {
		sb.WriteString("*All recent runs failed — likely a persistent issue rather than a transient one.*\n")
	}

	return domain.ContextAttachment{
		Type:     "note",
		Key:      fmt.Sprintf("run-history-%s", short),
		Label:    fmt.Sprintf("Historical Context %s", short),
		Content:  sb.String(),
		Format:   "markdown",
		Priority: "medium",
		Summary:  fmt.Sprintf("%d recent runs: %d succeeded, %d failed", total, successCount, failCount),
		Tags:     []string{"run", "history", "investigation"},
	}, true
}

func marshalJSON(value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func shortID(id uuid.UUID) string {
	value := id.String()
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func taskShortID() string {
	return shortID(uuid.New())
}
