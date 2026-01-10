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
	investigationTag                = "agent-manager-investigation"
	investigationApplyTag           = "agent-manager-investigation-apply"
	investigationProfileKey         = "agent-manager-investigation"
	investigationApplyProfileKey    = "agent-manager-investigation-apply"
	investigationEventLimit         = 500
	investigationReportTimeout      = 10 * time.Minute
	investigationApplyReportTimeout = 15 * time.Minute
)

// NOTE: InvestigationDepth type is defined in domain/investigation.go
// Use domain.InvestigationDepth, domain.InvestigationDepthQuick, etc.

// ScenarioContext contains information about a scenario being investigated.
type ScenarioContext struct {
	ProjectRoot    string   `json:"projectRoot"`
	ScenarioName   string   `json:"scenarioName,omitempty"`
	RunIDs         []string `json:"runIds"`
	KeyFiles       []string `json:"keyFiles,omitempty"`
	StructureHints []string `json:"structureHints,omitempty"`
}

// extractScenarioRoots extracts unique project roots from the investigated runs' tasks.
// Returns a map of projectRoot -> list of runIDs that use that root.
func (o *Orchestrator) extractScenarioRoots(ctx context.Context, runIDs []uuid.UUID) (map[string][]uuid.UUID, error) {
	roots := make(map[string][]uuid.UUID)

	for _, runID := range runIDs {
		run, err := o.GetRun(ctx, runID)
		if err != nil {
			return nil, err
		}

		task, err := o.GetTask(ctx, run.TaskID)
		if err != nil {
			return nil, err
		}

		projectRoot := task.ProjectRoot
		if projectRoot == "" {
			projectRoot = o.config.DefaultProjectRoot // Fallback
		}

		roots[projectRoot] = append(roots[projectRoot], runID)
	}

	return roots, nil
}

// determineInvestigationWorkDir determines the appropriate working directory for investigation.
// If all runs share the same projectRoot, use that. Otherwise, use the common ancestor.
func determineInvestigationWorkDir(roots map[string][]uuid.UUID, fallback string) string {
	if len(roots) == 0 {
		return fallback
	}

	// If single root, use it directly
	if len(roots) == 1 {
		for root := range roots {
			return root
		}
	}

	// Multiple roots - find common ancestor
	var paths []string
	for root := range roots {
		paths = append(paths, root)
	}

	return findCommonAncestor(paths, fallback)
}

// findCommonAncestor finds the common ancestor directory of multiple paths.
func findCommonAncestor(paths []string, fallback string) string {
	if len(paths) == 0 {
		return fallback
	}

	parts := strings.Split(paths[0], string(os.PathSeparator))

	for _, path := range paths[1:] {
		otherParts := strings.Split(path, string(os.PathSeparator))
		minLen := len(parts)
		if len(otherParts) < minLen {
			minLen = len(otherParts)
		}

		commonLen := 0
		for i := 0; i < minLen; i++ {
			if parts[i] == otherParts[i] {
				commonLen = i + 1
			} else {
				break
			}
		}
		parts = parts[:commonLen]
	}

	if len(parts) == 0 {
		return fallback
	}

	return strings.Join(parts, string(os.PathSeparator))
}

// buildScenarioContextAttachment creates a context attachment describing the scenarios.
func buildScenarioContextAttachment(roots map[string][]uuid.UUID) domain.ContextAttachment {
	var contexts []ScenarioContext

	for root, runIDs := range roots {
		scenarioCtx := ScenarioContext{
			ProjectRoot: root,
			RunIDs:      make([]string, len(runIDs)),
		}

		for i, id := range runIDs {
			scenarioCtx.RunIDs[i] = id.String()
		}

		// Extract scenario name from path
		scenarioCtx.ScenarioName = extractScenarioName(root)

		// Add key files to explore
		scenarioCtx.KeyFiles = detectKeyFiles(root)

		// Add structure hints
		scenarioCtx.StructureHints = getStructureHints(root)

		contexts = append(contexts, scenarioCtx)
	}

	content, _ := json.MarshalIndent(contexts, "", "  ")

	return domain.ContextAttachment{
		Type:    "note",
		Key:     "scenario-context",
		Label:   "Scenario Context",
		Content: string(content),
		Tags:    []string{"scenario", "context", "investigation"},
	}
}

// InvestigationMetadata contains dynamic investigation parameters passed as context.
type InvestigationMetadata struct {
	Depth           string   `json:"depth"`
	DepthGuidance   string   `json:"depthGuidance"`
	RunIDs          []string `json:"runIds"`
	ScenarioCount   int      `json:"scenarioCount"`
	TotalRunCount   int      `json:"totalRunCount"`
	InvestigatedAt  string   `json:"investigatedAt"`
}

// buildInvestigationMetadataAttachment creates a context attachment with investigation parameters.
// This contains all the dynamic data that used to be embedded in the prompt.
func buildInvestigationMetadataAttachment(runIDs []uuid.UUID, depth domain.InvestigationDepth, roots map[string][]uuid.UUID) domain.ContextAttachment {
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
		ScenarioCount:  len(roots),
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

// extractScenarioName extracts the scenario name from a project root path.
func extractScenarioName(projectRoot string) string {
	// Extract scenario name from path like "/home/.../Vrooli/scenarios/agent-manager"
	parts := strings.Split(projectRoot, string(os.PathSeparator))
	for i, part := range parts {
		if part == "scenarios" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	// Fallback to last path component
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

// detectKeyFiles finds important files in a project root.
func detectKeyFiles(projectRoot string) []string {
	keyFiles := []string{
		"CLAUDE.md",
		"README.md",
		"service.json",
		"Makefile",
		".vrooli/service.json",
	}

	var found []string
	for _, file := range keyFiles {
		path := filepath.Join(projectRoot, file)
		if _, err := os.Stat(path); err == nil {
			found = append(found, file)
		}
	}

	// Check for common subdirectories
	for _, subdir := range []string{"api", "ui", "cli", "docs"} {
		path := filepath.Join(projectRoot, subdir)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			found = append(found, subdir+"/")
		}
	}

	return found
}

// getStructureHints provides hints about the project structure.
func getStructureHints(projectRoot string) []string {
	var hints []string

	// Check for Go API
	if _, err := os.Stat(filepath.Join(projectRoot, "api", "main.go")); err == nil {
		hints = append(hints, "Go API in api/ - check api/internal/ for handlers and services")
	}

	// Check for React UI
	if _, err := os.Stat(filepath.Join(projectRoot, "ui", "package.json")); err == nil {
		hints = append(hints, "React UI in ui/ - check ui/src/ for components and hooks")
	}

	// Check for CLI
	if _, err := os.Stat(filepath.Join(projectRoot, "cli")); err == nil {
		hints = append(hints, "CLI in cli/ - check for command implementations")
	}

	// Check for CLAUDE.md
	if _, err := os.Stat(filepath.Join(projectRoot, "CLAUDE.md")); err == nil {
		hints = append(hints, "Has CLAUDE.md - read this first for project context")
	}

	return hints
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

	// Extract scenario roots from investigated runs
	roots, err := o.extractScenarioRoots(ctx, req.RunIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to extract scenario roots: %w", err)
	}

	// Determine working directory
	workDir := determineInvestigationWorkDir(roots, o.config.DefaultProjectRoot)

	// Build attachments - all dynamic data goes here, NOT in the prompt
	attachments, err := o.buildInvestigationAttachments(ctx, req.RunIDs, req.CustomContext)
	if err != nil {
		return nil, err
	}

	// Add investigation metadata attachment (depth, run IDs, etc.)
	metadataAttachment := buildInvestigationMetadataAttachment(req.RunIDs, depth, roots)
	attachments = append([]domain.ContextAttachment{metadataAttachment}, attachments...)

	// Add scenario context attachment
	scenarioCtx := buildScenarioContextAttachment(roots)
	attachments = append([]domain.ContextAttachment{scenarioCtx}, attachments...)

	// Use the stored prompt template (user-editable, no variable substitution)
	prompt := settings.PromptTemplate

	// Create task with correct working directory
	task, err := o.createInvestigationTask(ctx, "Investigation", prompt, attachments, workDir)
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
