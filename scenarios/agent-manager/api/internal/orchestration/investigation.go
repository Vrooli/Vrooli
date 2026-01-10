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

// InvestigationContext contains information about the investigation scope.
type InvestigationContext struct {
	ProjectRoot string   `json:"projectRoot"`
	ScopePaths  []string `json:"scopePaths,omitempty"`
	RunIDs      []string `json:"runIds"`
}

// buildInvestigationContextAttachment creates a context attachment describing the investigation scope.
func buildInvestigationContextAttachment(projectRoot string, scopePaths []string, runIDs []uuid.UUID) domain.ContextAttachment {
	runIDStrings := make([]string, len(runIDs))
	for i, id := range runIDs {
		runIDStrings[i] = id.String()
	}

	ctx := InvestigationContext{
		ProjectRoot: projectRoot,
		ScopePaths:  scopePaths,
		RunIDs:      runIDStrings,
	}

	content, _ := json.MarshalIndent(ctx, "", "  ")

	return domain.ContextAttachment{
		Type:    "note",
		Key:     "investigation-context",
		Label:   "Investigation Context",
		Content: string(content),
		Tags:    []string{"context", "investigation"},
	}
}

// InvestigationMetadata contains dynamic investigation parameters passed as context.
type InvestigationMetadata struct {
	Depth          string   `json:"depth"`
	DepthGuidance  string   `json:"depthGuidance"`
	RunIDs         []string `json:"runIds"`
	TotalRunCount  int      `json:"totalRunCount"`
	InvestigatedAt string   `json:"investigatedAt"`
}

// buildInvestigationMetadataAttachment creates a context attachment with investigation parameters.
// This contains all the dynamic data that used to be embedded in the prompt.
func buildInvestigationMetadataAttachment(runIDs []uuid.UUID, depth domain.InvestigationDepth) domain.ContextAttachment {
	// Build depth guidance based on investigation depth
	var depthGuidance string
	switch depth {
	case domain.InvestigationDepthQuick:
		depthGuidance = "QUICK mode: Perform rapid behavioral analysis focusing on the most obvious agent mistakes. Time target: 2-3 minutes. Focus on the failing action and immediate decision that led to it."
	case domain.InvestigationDepthDeep:
		depthGuidance = "DEEP mode: Perform thorough behavioral analysis of the agent's decision-making process. Take time to understand the full context the agent had, trace its reasoning, and identify patterns."
	default: // Standard
		depthGuidance = "STANDARD mode: Perform balanced behavioral analysis with targeted review of key decision points."
	}

	// Convert UUIDs to strings
	runIDStrings := make([]string, len(runIDs))
	for i, id := range runIDs {
		runIDStrings[i] = id.String()
	}

	metadata := InvestigationMetadata{
		Depth:          string(depth),
		DepthGuidance:  depthGuidance,
		RunIDs:         runIDStrings,
		TotalRunCount:  len(runIDs),
		InvestigatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	content, _ := json.MarshalIndent(metadata, "", "  ")

	return domain.ContextAttachment{
		Type:    "note",
		Key:     "investigation-metadata",
		Label:   "Investigation Metadata",
		Content: string(content),
		Tags:    []string{"metadata", "investigation", "config"},
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

	// Use the stored prompt template (user-editable, no variable substitution)
	prompt := settings.PromptTemplate

	// Create task with explicit project root
	task, err := o.createInvestigationTask(ctx, "Investigation", prompt, attachments, projectRoot)
	if err != nil {
		return nil, err
	}

	return o.createInvestigationRunWithProfile(ctx, task.ID, investigationTag, investigationProfileRef())
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
	if run.Tag != investigationTag {
		return nil, domain.NewValidationError("investigationRunId", "run is not an investigation run")
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

	prompt := buildApplyPrompt(settings.ApplyPromptTemplate, investigationRunID, customContext)
	// Use the original task's project root for the apply run
	applyTask, err := o.createInvestigationTask(ctx, "Apply Investigation", prompt, attachments, task.ProjectRoot)
	if err != nil {
		return nil, err
	}

	// Apply runs use a different profile with write capabilities
	return o.createInvestigationRunWithProfile(ctx, applyTask.ID, investigationApplyTag, applyInvestigationProfileRef())
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

func (o *Orchestrator) createInvestigationRunWithProfile(ctx context.Context, taskID uuid.UUID, tag string, profileRef *ProfileRef) (*domain.Run, error) {
	sandboxConfig := &domain.SandboxConfig{NoLock: true}
	if o.config.DefaultSandboxConfig != nil {
		clone := *o.config.DefaultSandboxConfig
		clone.NoLock = true
		sandboxConfig = &clone
	}

	return o.CreateRun(ctx, CreateRunRequest{
		TaskID:        taskID,
		ProfileRef:    profileRef,
		Tag:           tag,
		Force:         true,
		SandboxConfig: sandboxConfig,
	})
}

func (o *Orchestrator) buildInvestigationAttachments(
	ctx context.Context,
	runIDs []uuid.UUID,
	customContext string,
) ([]domain.ContextAttachment, error) {
	attachments := make([]domain.ContextAttachment, 0, len(runIDs)*2+1)
	for _, runID := range runIDs {
		run, err := o.GetRun(ctx, runID)
		if err != nil {
			return nil, err
		}

		runJSON, err := marshalJSON(run)
		if err != nil {
			return nil, err
		}

		short := shortID(runID)
		attachments = append(attachments, noteAttachment(
			fmt.Sprintf("run-summary-%s", short),
			fmt.Sprintf("Run Summary %s", short),
			runJSON,
			[]string{"run", "summary", "investigation"},
		))

		if o.events != nil {
			events, err := o.GetRunEvents(ctx, runID, event.GetOptions{Limit: investigationEventLimit})
			if err != nil {
				return nil, err
			}
			eventsJSON, err := marshalJSON(events)
			if err != nil {
				return nil, err
			}
			attachments = append(attachments, noteAttachment(
				fmt.Sprintf("run-events-%s", short),
				fmt.Sprintf("Run Events %s", short),
				eventsJSON,
				[]string{"run", "events", "investigation"},
			))
		}

		diff, err := o.GetRunDiff(ctx, runID)
		if err == nil && diff != nil {
			diffJSON, err := marshalJSON(diff)
			if err != nil {
				return nil, err
			}
			attachments = append(attachments, noteAttachment(
				fmt.Sprintf("run-diff-%s", short),
				fmt.Sprintf("Run Diff %s", short),
				diffJSON,
				[]string{"run", "diff", "investigation"},
			))
		}
	}

	if strings.TrimSpace(customContext) != "" {
		attachments = append(attachments, noteAttachment(
			"user-context",
			"Additional Context",
			customContext,
			[]string{"user", "context"},
		))
	}

	return attachments, nil
}

func (o *Orchestrator) buildApplyAttachments(
	ctx context.Context,
	investigationRun *domain.Run,
	task *domain.Task,
	customContext string,
) ([]domain.ContextAttachment, error) {
	attachments := make([]domain.ContextAttachment, 0, len(task.ContextAttachments)+3)
	attachments = append(attachments, task.ContextAttachments...)

	short := shortID(investigationRun.ID)
	runJSON, err := marshalJSON(investigationRun)
	if err != nil {
		return nil, err
	}
	attachments = append(attachments, noteAttachment(
		fmt.Sprintf("investigation-run-summary-%s", short),
		fmt.Sprintf("Investigation Run Summary %s", short),
		runJSON,
		[]string{"run", "investigation"},
	))

	if o.events != nil {
		events, err := o.GetRunEvents(ctx, investigationRun.ID, event.GetOptions{Limit: investigationEventLimit})
		if err != nil {
			return nil, err
		}
		eventsJSON, err := marshalJSON(events)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, noteAttachment(
			fmt.Sprintf("investigation-run-events-%s", short),
			fmt.Sprintf("Investigation Run Events %s", short),
			eventsJSON,
			[]string{"run", "events", "investigation"},
		))
	}

	diff, err := o.GetRunDiff(ctx, investigationRun.ID)
	if err == nil && diff != nil {
		diffJSON, err := marshalJSON(diff)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, noteAttachment(
			fmt.Sprintf("investigation-run-diff-%s", short),
			fmt.Sprintf("Investigation Run Diff %s", short),
			diffJSON,
			[]string{"run", "diff", "investigation"},
		))
	}

	if strings.TrimSpace(customContext) != "" {
		attachments = append(attachments, noteAttachment(
			"user-context",
			"Additional Context",
			customContext,
			[]string{"user", "context"},
		))
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
