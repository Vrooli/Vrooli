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

// investigationFrictionMethodologyExtract contains the useful subset of the
// conversation-friction-analysis skill for investigation context. The full skill
// is ~270 lines; we keep only root-cause attribution, priority scoring, severity
// definitions, and friction signal patterns. Everything else (generic workflow,
// output template, retirement mapping) either duplicates the investigation
// skill's own structure or adds noise.
const investigationFrictionMethodologyExtract = `

---

## Reference: Friction Analysis Concepts

### Root-Cause Attribution Layers

Classify each friction event into one primary layer:
- **CLI/tool output**: weak next actions, poor defaults, selector/ID confusion
- **Tool capability**: missing command for repeated manual pattern
- **Skill design**: ambiguity, missing guardrails, scattered long-tail details
- **Docs/discovery**: source of truth hard to find, stale references
- **Process/policy**: no clear escalation path, conflicting governance rules
- **Intent/inputs**: missing prerequisites or unstable objectives

If multiple layers apply, identify a primary cause and contributing causes.

### Priority Scoring

For each recommendation, score:
- **impact** (1-5): expected reduction in future friction
- **recurrence** (1-5): how often this likely repeats
- **cost** (1-5): effort/risk to implement

Priority = (impact × recurrence) − cost

Prefer fixes that remove repeated manual interpretation, improve CLI output contracts, or reduce policy ambiguity.

### Severity Definitions

| Severity | Definition | Typical action |
|---|---|---|
| Critical | Blocks delivery, risks unsafe action, or causes repeated hard failure | Immediate policy/tooling fix |
| Major | Causes frequent retries/guessing and unstable execution | Patch skill/tool output soon |
| Gap | Capability implied but not operationally enabled | Add capability or explicit handoff |
| Minor | Clarity/friction improvements with low immediate risk | Queue for batch improvement |

"Forces the agent to guess next action" is at least Major.

### Friction Signal Patterns

Common signals to look for in event timelines:
- Repeated clarification on the same point
- Conflicting instructions across skills/docs
- Command examples that fail or force guessing
- Output that is non-actionable for next step decisions
- Repeated "manual interpretation" loops
- If the same pattern appears 2+ times, treat it as systemic
`

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

	// Append user-uploaded image attachments. The CreateRun pipeline resolves
	// these to file paths and hands them to the runner (see orchestration/service.go
	// image-resolution block); we just need to record the references on the task.
	for _, id := range req.AttachmentIDs {
		if strings.TrimSpace(id) == "" {
			continue
		}
		attachments = append(attachments, domain.ContextAttachment{
			Type:         "image",
			AttachmentID: id,
			Label:        "Uploaded image",
			Key:          "investigation-image-" + id,
			Tags:         []string{"image", "investigation"},
		})
	}

	// Read prompt from prompt-manager skill, fall back to settings/hardcoded default
	prompt, err := o.readInvestigationSkill(ctx, "agent-manager-process-investigation")
	if err != nil {
		prompt = settings.PromptTemplate
	}

	// Append a trimmed extract of the friction-analysis methodology.
	// The full conversation-friction-analysis skill is ~270 lines of generic
	// methodology — most of it (workflow steps, output template, scope boundaries,
	// retirement mapping) duplicates or conflicts with the investigation skill's
	// own structure. We inline only the parts that add value:
	//   - Root-cause attribution layers (helps classify findings)
	//   - Priority scoring formula (helps rank recommendations)
	//   - Severity model (shared vocabulary with investigation output)
	//   - Friction signal patterns (helps spot issues in timelines)
	prompt = prompt + investigationFrictionMethodologyExtract

	// Create task with explicit project root
	task, err := o.createInvestigationTask(ctx, "Investigation", prompt, attachments, projectRoot)
	if err != nil {
		return nil, err
	}

	run, err := o.createInvestigationRunWithProfile(
		ctx,
		task.ID,
		investigationTag,
		investigationProfileRefWithOverrides(req.RoleRef),
		req.RunIDs,
		nil,
		req.Environment,
	)
	if err != nil {
		return nil, err
	}
	return o.attachRunActions(ctx, run), nil
}

// CreateInvestigationApplyRun creates a new run that applies investigation recommendations.
func (o *Orchestrator) CreateInvestigationApplyRun(
	ctx context.Context,
	req CreateInvestigationApplyRequest,
) (*domain.Run, error) {
	investigationRunID := req.InvestigationRunID
	customContext := req.CustomContext

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

	// Append user-uploaded image attachments for this apply step (on top of any
	// images that were already attached to the investigation task and copied in
	// via buildApplyAttachments). The CreateRun pipeline resolves these to file
	// paths for the runner.
	for _, id := range req.AttachmentIDs {
		if strings.TrimSpace(id) == "" {
			continue
		}
		attachments = append(attachments, domain.ContextAttachment{
			Type:         "image",
			AttachmentID: id,
			Label:        "Uploaded image",
			Key:          "apply-image-" + id,
			Tags:         []string{"image", "investigation-apply"},
		})
	}

	applyTemplate, err := o.readInvestigationSkill(ctx, "agent-manager-process-investigation-apply")
	if err != nil {
		applyTemplate = settings.ApplyPromptTemplate
	}

	// Pre-fetch supporting methodology skill for the apply agent.
	if supporting := o.readSupportingSkills(ctx, []string{
		"skill-principles",
	}); supporting != "" {
		applyTemplate = applyTemplate + "\n\n---\n\n## Reference Methodology\n\n" + supporting
	}

	prompt := buildApplyPrompt(applyTemplate, investigationRunID, customContext)
	// Use the original task's project root for the apply run
	applyTask, err := o.createInvestigationTask(ctx, "Apply Investigation", prompt, attachments, task.ProjectRoot)
	if err != nil {
		return nil, err
	}

	// Apply runs use a different profile with write capabilities. Caller-supplied
	// runner/preset overrides are honored; nil preserves the default apply profile.
	applyRun, err := o.createInvestigationRunWithProfile(
		ctx,
		applyTask.ID,
		investigationApplyTag,
		applyInvestigationProfileRefWithOverrides(req.RoleRef),
		nil,
		&investigationRunID,
		req.Environment,
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

// readSupportingSkills fetches multiple skills from prompt-manager and returns
// them concatenated with headers. Any individual skill that fails to load is
// silently skipped. Returns empty string if none could be loaded.
func (o *Orchestrator) readSupportingSkills(ctx context.Context, skillIDs []string) string {
	if o.promptClient == nil {
		return ""
	}

	var parts []string
	for _, id := range skillIDs {
		content, err := o.promptClient.ReadSkill(ctx, id, nil, false)
		if err != nil || strings.TrimSpace(content) == "" {
			continue
		}
		parts = append(parts, content)
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n---\n\n")
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
	environment map[string]string,
) (*domain.Run, error) {
	// Investigations are diagnostic by intent — the deliverable is a written
	// report, not repo mutations. ManualReview=true defers apply at run end
	// so any inadvertent file changes persist as pending-review for operator
	// approval (via GCT, agent-manager run-detail, or workspace-sandbox UI)
	// rather than auto-applying. NoLock=true (the contract default) lets
	// investigations run concurrently with other work over the same scope;
	// per the auditability contract, locking and acceptance/apply are
	// orthogonal. See workspace-sandbox/docs/AUDITABILITY_CONTRACT.md.
	sandboxConfig := domain.DefaultSandboxConfig()
	sandboxConfig.ManualReview = true

	return o.CreateRun(ctx, CreateRunRequest{
		TaskID:                   taskID,
		ProfileRef:               profileRef,
		Tag:                      tag,
		Force:                    true,
		SandboxConfig:            sandboxConfig,
		SourceRunIDs:             sourceRunIDs,
		SourceInvestigationRunID: sourceInvestigationRunID,
		Environment:              environment,
	})
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

// investigationProfileRefWithOverrides applies a caller-provided portable role
// on top of the default investigation profile. Nil preserves the default.
func investigationProfileRefWithOverrides(roleRef *string) *ProfileRef {
	defaults := defaultInvestigationProfile()
	applyInvestigationOverrides(defaults, roleRef)
	return &ProfileRef{
		ProfileKey: investigationProfileKey,
		Defaults:   defaults,
	}
}

// applyInvestigationProfileRefWithOverrides mirrors investigationProfileRefWithOverrides
// for the apply flow.
func applyInvestigationProfileRefWithOverrides(roleRef *string) *ProfileRef {
	defaults := defaultApplyInvestigationProfile()
	applyInvestigationOverrides(defaults, roleRef)
	return &ProfileRef{
		ProfileKey: investigationApplyProfileKey,
		Defaults:   defaults,
	}
}

// applyInvestigationOverrides mutates the profile with a caller-provided role.
func applyInvestigationOverrides(profile *domain.AgentProfile, roleRef *string) {
	if profile == nil {
		return
	}
	if roleRef != nil {
		profile.RoleRef = strings.TrimSpace(*roleRef)
	}
}

func defaultInvestigationProfile() *domain.AgentProfile {
	return &domain.AgentProfile{
		Name:        "Agent-Manager Investigation",
		ProfileKey:  investigationProfileKey,
		Description: "Agent profile for behavioral analysis of failed agent runs (read-only)",
		RoleRef:     "code.smart",
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
		// Read-only by intent, but the tool surface (`execute_command`,
		// `web_search`) doesn't hard-prevent writes. ManualReview=true is
		// defense-in-depth: if the agent does mutate files, those changes
		// land as pending-review provenance for an operator instead of
		// silently auto-applying. Mirrors the apply-investigation profile
		// and the per-run override in createInvestigationRunWithProfile.
		SandboxConfig: func() *domain.SandboxConfig {
			cfg := domain.DefaultSandboxConfig()
			cfg.ManualReview = true
			return cfg
		}(),
		NetworkAccess: domain.NetworkAccessLocalhost,
		CreatedBy:     "agent-manager",
	}
}

func defaultApplyInvestigationProfile() *domain.AgentProfile {
	return &domain.AgentProfile{
		Name:        "Agent-Manager Apply Investigation",
		ProfileKey:  investigationApplyProfileKey,
		Description: "Agent profile for applying investigation recommendations (has write capabilities)",
		RoleRef:     "code.smart",
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
		SandboxConfig: func() *domain.SandboxConfig {
			cfg := domain.DefaultSandboxConfig()
			cfg.ManualReview = true
			return cfg
		}(),
		NetworkAccess: domain.NetworkAccessLocalhost,
		CreatedBy:     "agent-manager",
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
