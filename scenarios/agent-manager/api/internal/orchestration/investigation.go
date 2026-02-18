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
	// Use domain constants for tag values to ensure consistency across packages.
	// See domain.InvestigationTag and domain.InvestigationApplyTag.
	investigationTag             = domain.InvestigationTag
	investigationApplyTag        = domain.InvestigationApplyTag
	investigationProfileKey      = domain.InvestigationTag      // Profile key matches tag
	investigationApplyProfileKey = domain.InvestigationApplyTag // Profile key matches tag

	investigationEventLimit         = 500
	investigationReportTimeout      = 10 * time.Minute
	investigationApplyReportTimeout = 15 * time.Minute
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

	sb.WriteString("\n### CLI Commands for Additional Data\n\n")
	sb.WriteString("```bash\n")
	for _, id := range runIDs {
		sb.WriteString(fmt.Sprintf("agent-manager run get %s      # Full run details\n", id))
		sb.WriteString(fmt.Sprintf("agent-manager run events %s   # All events with tool calls\n", id))
		sb.WriteString(fmt.Sprintf("agent-manager run diff %s     # Code changes made\n", id))
	}
	sb.WriteString("```\n")

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

	// Build attachments - all dynamic data goes here, NOT in the prompt
	attachments, err := o.buildInvestigationAttachments(ctx, req.RunIDs, req.CustomContext)
	if err != nil {
		return nil, err
	}

	// Add investigation metadata attachment (depth, run IDs, etc.)
	metadataAttachment := buildInvestigationMetadataAttachment(req.RunIDs, depth)
	attachments = append([]domain.ContextAttachment{metadataAttachment}, attachments...)

	// Add investigation context attachment (explicit project root and scope paths)
	investigationCtx := buildInvestigationContextAttachment(projectRoot, req.ScopePaths, req.RunIDs)
	attachments = append([]domain.ContextAttachment{investigationCtx}, attachments...)

	// Read prompt from prompt-manager skill, fall back to settings/hardcoded default
	prompt, err := o.readInvestigationSkill(ctx, "agent-manager-process-investigation")
	if err != nil {
		prompt = settings.PromptTemplate
	}

	// Create task with explicit project root
	task, err := o.createInvestigationTask(ctx, "Investigation", prompt, attachments, projectRoot)
	if err != nil {
		return nil, err
	}

	run, err := o.createInvestigationRunWithProfile(
		ctx,
		task.ID,
		investigationTag,
		investigationProfileRef(),
		req.RunIDs,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return o.attachRunActions(ctx, run), nil
}

// CreateInvestigationApplyRun creates a new run that applies investigation recommendations.
func (o *Orchestrator) CreateInvestigationApplyRun(
	ctx context.Context,
	investigationRunID uuid.UUID,
	customContext string,
) (*domain.Run, error) {
	run, err := o.GetRun(ctx, investigationRunID)
	if err != nil {
		return nil, err
	}
	if allowed, reason := domain.CanApplyInvestigationRun(run, o.investigationTagAllowlist(ctx)); !allowed {
		return nil, domain.NewValidationError("investigationRunId", reason)
	}

	task, err := o.GetTask(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}

	// Fetch investigation settings to get the configurable apply prompt template
	var settings *domain.InvestigationSettings
	if o.investigationSettings != nil {
		settings, err = o.investigationSettings.Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get investigation settings: %w", err)
		}
	}
	if settings == nil {
		settings = domain.DefaultInvestigationSettings()
	}

	attachments, err := o.buildApplyAttachments(ctx, run, task, customContext)
	if err != nil {
		return nil, err
	}

	applyTemplate, err := o.readInvestigationSkill(ctx, "agent-manager-process-investigation-apply")
	if err != nil {
		applyTemplate = settings.ApplyPromptTemplate
	}
	prompt := buildApplyPrompt(applyTemplate, investigationRunID, customContext)
	// Use the original task's project root for the apply run
	applyTask, err := o.createInvestigationTask(ctx, "Apply Investigation", prompt, attachments, task.ProjectRoot)
	if err != nil {
		return nil, err
	}

	// Apply runs use a different profile with write capabilities
	applyRun, err := o.createInvestigationRunWithProfile(
		ctx,
		applyTask.ID,
		investigationApplyTag,
		applyInvestigationProfileRef(),
		nil,
		&investigationRunID,
	)
	if err != nil {
		return nil, err
	}
	return o.attachRunActions(ctx, applyRun), nil
}

// readInvestigationSkill reads a skill from prompt-manager, returning an error if unavailable.
func (o *Orchestrator) readInvestigationSkill(ctx context.Context, skillID string) (string, error) {
	if o.promptClient == nil {
		return "", fmt.Errorf("no prompt client configured")
	}
	content, err := o.promptClient.ReadSkill(ctx, skillID, nil, false)
	if err != nil {
		return "", err
	}
	return content, nil
}

func (o *Orchestrator) investigationTagAllowlist(ctx context.Context) []domain.InvestigationTagRule {
	settings, err := o.GetInvestigationSettings(ctx)
	if err != nil || settings == nil {
		return domain.DefaultInvestigationTagAllowlist()
	}
	return domain.NormalizeInvestigationTagAllowlist(settings.InvestigationTagAllowlist)
}

func (o *Orchestrator) recommendationQueueFilter(ctx context.Context) func(*domain.Run) bool {
	allowlist := o.investigationTagAllowlist(ctx)
	return func(run *domain.Run) bool {
		return domain.MatchesInvestigationTag(run.Tag, allowlist)
	}
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

func (o *Orchestrator) createInvestigationRunWithProfile(
	ctx context.Context,
	taskID uuid.UUID,
	tag string,
	profileRef *ProfileRef,
	sourceRunIDs []uuid.UUID,
	sourceInvestigationRunID *uuid.UUID,
) (*domain.Run, error) {
	sandboxConfig := &domain.SandboxConfig{NoLock: true}
	if o.config.DefaultSandboxConfig != nil {
		clone := *o.config.DefaultSandboxConfig
		clone.NoLock = true
		sandboxConfig = &clone
	}

	return o.CreateRun(ctx, CreateRunRequest{
		TaskID:                   taskID,
		ProfileRef:               profileRef,
		Tag:                      tag,
		Force:                    true,
		SandboxConfig:            sandboxConfig,
		SourceRunIDs:             sourceRunIDs,
		SourceInvestigationRunID: sourceInvestigationRunID,
	})
}

func (o *Orchestrator) buildInvestigationAttachments(
	ctx context.Context,
	runIDs []uuid.UUID,
	customContext string,
) ([]domain.ContextAttachment, error) {
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

		// 1. Human-readable run overview (replaces raw JSON dump).
		attachments = append(attachments, buildRunOverview(run, task, profile, short))

		// 2. Agent setup context (prompt-manager paths for Agent, Team, Member).
		if profile != nil {
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

		// 3. Curated event timeline (replaces raw event JSON array).
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

		// 4. Diff (kept but with proper metadata).
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

		// 5. Historical context (recent runs with same agent profile).
		if run.AgentProfileID != nil {
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
		if c.ModelPreset != "" {
			sb.WriteString(fmt.Sprintf("**Model Preset**: %s\n", c.ModelPreset))
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
// Returns (attachment, true) if the agent directory exists, (zero, false) otherwise.
func buildAgentSetupAttachment(profile *domain.AgentProfile, projectRoot string, short string) (domain.ContextAttachment, bool) {
	storeRoot := filepath.Join(projectRoot, "scenarios", "prompt-manager", "store")
	agentDir := filepath.Join(storeRoot, "agents", profile.ProfileKey)

	// Check if the agent directory actually exists before including paths.
	if _, err := os.Stat(agentDir); err != nil {
		return domain.ContextAttachment{}, false
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### Agent: `%s`\n\n", profile.ProfileKey))
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

	sb.WriteString("\n*Read these files to understand the agent's identity, instructions, tools, team context, and scheduled responsibilities.*\n")

	return domain.ContextAttachment{
		Type:     "note",
		Key:      fmt.Sprintf("agent-setup-%s", short),
		Label:    fmt.Sprintf("Agent Setup %s", short),
		Content:  sb.String(),
		Format:   "markdown",
		Priority: "high",
		Summary:  fmt.Sprintf("Prompt-manager paths for agent %q and its team memberships", profile.ProfileKey),
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

func (o *Orchestrator) buildApplyAttachments(
	ctx context.Context,
	investigationRun *domain.Run,
	task *domain.Task,
	customContext string,
) ([]domain.ContextAttachment, error) {
	// Include the original investigation's context attachments (already human-readable
	// if the investigation was created with the new format).
	attachments := make([]domain.ContextAttachment, 0, len(task.ContextAttachments)+4)
	attachments = append(attachments, task.ContextAttachments...)

	short := shortID(investigationRun.ID)

	// Investigation run overview (human-readable, not raw JSON).
	var investigationProfile *domain.AgentProfile
	if investigationRun.AgentProfileID != nil {
		investigationProfile, _ = o.GetProfile(ctx, *investigationRun.AgentProfileID)
	}
	var investigationTask *domain.Task
	investigationTask, _ = o.GetTask(ctx, investigationRun.TaskID)
	attachments = append(attachments, buildRunOverview(investigationRun, investigationTask, investigationProfile, "inv-"+short))

	// Investigation run timeline.
	if o.events != nil {
		events, err := o.GetRunEvents(ctx, investigationRun.ID, event.GetOptions{
			AfterSequence: -1,
			Limit:         investigationEventLimit,
		})
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, buildRunTimeline(events, investigationRun, "inv-"+short))
	}

	// Investigation run diff.
	diff, err := o.GetRunDiff(ctx, investigationRun.ID)
	if err == nil && diff != nil {
		diffJSON, err := marshalJSON(diff)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, domain.ContextAttachment{
			Type:     "note",
			Key:      fmt.Sprintf("investigation-run-diff-%s", short),
			Label:    fmt.Sprintf("Investigation Run Diff %s", short),
			Content:  diffJSON,
			Format:   "json",
			Priority: "medium",
			Summary:  fmt.Sprintf("Code changes from investigation run %s", short),
			Tags:     []string{"run", "diff", "investigation"},
		})
	}

	if strings.TrimSpace(customContext) != "" {
		attachments = append(attachments, domain.ContextAttachment{
			Type:     "note",
			Key:      "user-context",
			Label:    "Additional Context",
			Content:  customContext,
			Format:   "markdown",
			Priority: "medium",
			Summary:  "User-provided additional context for this apply run",
			Tags:     []string{"user", "context"},
		})
	}

	return attachments, nil
}

func buildApplyPrompt(template string, investigationRunID uuid.UUID, customContext string) string {
	var sb strings.Builder

	// Use the configurable template as the base
	sb.WriteString(template)
	sb.WriteString("\n\n")

	// Add dynamic investigation run reference
	sb.WriteString("---\n\n")
	sb.WriteString("## Investigation Run Reference\n\n")
	sb.WriteString(fmt.Sprintf("**Investigation Run ID:** `%s`\n\n", investigationRunID))
	sb.WriteString("Use these commands to fetch additional investigation data:\n")
	sb.WriteString(fmt.Sprintf("```bash\nagent-manager run get %s\nagent-manager run events %s\nagent-manager run diff %s\n```\n\n", investigationRunID, investigationRunID, investigationRunID))

	// Add custom context if provided
	if strings.TrimSpace(customContext) != "" {
		sb.WriteString("## User-Provided Context\n\n")
		sb.WriteString(customContext)
		sb.WriteString("\n")
	}

	return sb.String()
}

func investigationProfileRef() *ProfileRef {
	return &ProfileRef{
		ProfileKey: investigationProfileKey,
		Defaults:   defaultInvestigationProfile(),
	}
}

func applyInvestigationProfileRef() *ProfileRef {
	return &ProfileRef{
		ProfileKey: investigationApplyProfileKey,
		Defaults:   defaultApplyInvestigationProfile(),
	}
}

func defaultInvestigationProfile() *domain.AgentProfile {
	return &domain.AgentProfile{
		Name:        "Agent-Manager Investigation",
		ProfileKey:  investigationProfileKey,
		Description: "Agent profile for behavioral analysis of failed agent runs (read-only)",
		RunnerType:  domain.RunnerTypeCodex,
		ModelPreset: domain.ModelPresetSmart,
		MaxTurns:    75, // Increased for active exploration
		Timeout:     investigationReportTimeout,
		AllowedTools: []string{
			// File exploration
			"read_file",
			"list_files",
			"glob",
			"grep",
			// Code analysis
			"analyze_code",
			// Command execution for investigation
			"execute_command",
			// Search capabilities
			"web_search",
		},
		SkipPermissionPrompt: true,
		RequiresSandbox:      true,
		RequiresApproval:     false,
		CreatedBy:            "agent-manager",
	}
}

func defaultApplyInvestigationProfile() *domain.AgentProfile {
	return &domain.AgentProfile{
		Name:        "Agent-Manager Apply Investigation",
		ProfileKey:  investigationApplyProfileKey,
		Description: "Agent profile for applying investigation recommendations (has write capabilities)",
		RunnerType:  domain.RunnerTypeCodex,
		ModelPreset: domain.ModelPresetSmart,
		MaxTurns:    100, // More turns for implementing fixes
		Timeout:     investigationApplyReportTimeout,
		AllowedTools: []string{
			// File exploration (same as investigation)
			"read_file",
			"list_files",
			"glob",
			"grep",
			// Code analysis
			"analyze_code",
			// Command execution
			"execute_command",
			// Search capabilities
			"web_search",
			// Write capabilities (NEW - for applying fixes)
			"edit_file",
			"write_file",
			"create_file",
			"delete_file",
		},
		SkipPermissionPrompt: true,
		RequiresSandbox:      true,
		RequiresApproval:     true, // Require approval for changes
		CreatedBy:            "agent-manager",
	}
}

// getBuiltInProfileDefaults returns built-in default profile settings for known profile keys.
// Returns nil if the profile key is not a known built-in profile.
func getBuiltInProfileDefaults(profileKey string) *domain.AgentProfile {
	switch profileKey {
	case investigationProfileKey:
		return defaultInvestigationProfile()
	case investigationApplyProfileKey:
		return defaultApplyInvestigationProfile()
	default:
		return nil
	}
}

func noteAttachment(key, label, content string, tags []string) domain.ContextAttachment {
	return domain.ContextAttachment{
		Type:    "note",
		Key:     key,
		Label:   label,
		Content: content,
		Tags:    tags,
	}
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
