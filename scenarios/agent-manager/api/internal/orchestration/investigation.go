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

// InvestigationDepth controls how thorough the investigation should be.
type InvestigationDepth string

const (
	// InvestigationDepthQuick performs rapid analysis with minimal exploration.
	InvestigationDepthQuick InvestigationDepth = "quick"
	// InvestigationDepthStandard performs balanced analysis with targeted exploration.
	InvestigationDepthStandard InvestigationDepth = "standard"
	// InvestigationDepthDeep performs thorough exploration of the codebase.
	InvestigationDepthDeep InvestigationDepth = "deep"
)

// IsValid checks if the investigation depth is valid.
func (d InvestigationDepth) IsValid() bool {
	switch d {
	case InvestigationDepthQuick, InvestigationDepthStandard, InvestigationDepthDeep, "":
		return true
	default:
		return false
	}
}

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

	// Default depth to standard
	depth := req.Depth
	if depth == "" {
		depth = InvestigationDepthStandard
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

	// Build attachments
	attachments, err := o.buildInvestigationAttachments(ctx, req.RunIDs, req.CustomContext)
	if err != nil {
		return nil, err
	}

	// Add scenario context attachment as first attachment
	scenarioCtx := buildScenarioContextAttachment(roots)
	attachments = append([]domain.ContextAttachment{scenarioCtx}, attachments...)

	// Build directive prompt with depth awareness
	prompt := buildInvestigationPrompt(req.RunIDs, req.CustomContext, depth, roots)

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

	attachments, err := o.buildApplyAttachments(ctx, run, task, customContext)
	if err != nil {
		return nil, err
	}

	prompt := buildApplyPrompt(investigationRunID, customContext)
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

func buildInvestigationPrompt(runIDs []uuid.UUID, customContext string, depth InvestigationDepth, roots map[string][]uuid.UUID) string {
	var sb strings.Builder
	sb.WriteString("# Agent-Manager Investigation\n\n")

	// Set expectations based on depth
	switch depth {
	case InvestigationDepthQuick:
		sb.WriteString("## Investigation Mode: QUICK\n")
		sb.WriteString("Perform a rapid analysis focusing on the most obvious issues.\n")
		sb.WriteString("Time target: 2-3 minutes. Focus on the error messages and immediate causes.\n\n")
	case InvestigationDepthDeep:
		sb.WriteString("## Investigation Mode: DEEP\n")
		sb.WriteString("Perform a thorough investigation exploring all relevant code paths.\n")
		sb.WriteString("Take time to understand the full context before diagnosing. Explore related files and dependencies.\n\n")
	default: // Standard
		sb.WriteString("## Investigation Mode: STANDARD\n")
		sb.WriteString("Perform a balanced investigation with targeted exploration.\n\n")
	}

	// Directive instructions
	sb.WriteString("## Your Mission\n")
	sb.WriteString("You are an expert debugger. Your job is to ACTIVELY INVESTIGATE why the runs below failed.\n")
	sb.WriteString("**Do NOT just analyze the provided data - EXPLORE THE CODEBASE to find root causes.**\n\n")

	// Scenario locations to explore
	sb.WriteString("## Scenario Locations to Explore\n")
	for root, ids := range roots {
		scenarioName := extractScenarioName(root)
		sb.WriteString(fmt.Sprintf("### %s\n", scenarioName))
		sb.WriteString(fmt.Sprintf("- **Path**: `%s`\n", root))
		sb.WriteString(fmt.Sprintf("- **Runs**: %d\n", len(ids)))

		// Add key files if they exist
		keyFiles := detectKeyFiles(root)
		if len(keyFiles) > 0 {
			sb.WriteString(fmt.Sprintf("- **Key files**: %s\n", strings.Join(keyFiles, ", ")))
		}

		// Add structure hints
		hints := getStructureHints(root)
		for _, hint := range hints {
			sb.WriteString(fmt.Sprintf("  - %s\n", hint))
		}
		sb.WriteString("\n")
	}

	// Runs under investigation
	sb.WriteString("## Runs Under Investigation\n")
	for _, id := range runIDs {
		sb.WriteString(fmt.Sprintf("- `%s`\n", id))
	}
	sb.WriteString("\n")

	// Active investigation steps
	sb.WriteString("## Required Investigation Steps\n")
	sb.WriteString("1. **Read the scenario's CLAUDE.md or README.md** to understand the project structure\n")
	sb.WriteString("2. **Analyze the error messages** in the attached run events - find the actual failure\n")
	sb.WriteString("3. **Trace the error to source code** - use grep/read to find the failing code path\n")
	sb.WriteString("4. **Check related files** - look at imports, dependencies, callers, and configuration\n")
	sb.WriteString("5. **Identify the root cause** - distinguish symptoms from underlying issues\n\n")

	// Common patterns to check
	sb.WriteString("## Common Failure Patterns to Check\n")
	sb.WriteString("- **Log/Output Issues**: Large outputs breaking scanners, missing newlines, buffering problems\n")
	sb.WriteString("- **Path Issues**: Relative vs absolute paths, working directory assumptions\n")
	sb.WriteString("- **Timeout Issues**: Operations taking longer than expected\n")
	sb.WriteString("- **Tool Issues**: Missing tools, wrong tool usage, tool not trusted by agent\n")
	sb.WriteString("- **Prompt Issues**: Agent not understanding instructions, missing context\n")
	sb.WriteString("- **State Issues**: Stale data, cache invalidation, missing initialization\n\n")

	// How to use CLI
	sb.WriteString("## How to Fetch Additional Run Data\n")
	sb.WriteString("If you need full details beyond the attachments, use the agent-manager CLI:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("agent-manager run get <run-id>      # Full run details\n")
	sb.WriteString("agent-manager run events <run-id>  # All events with tool calls\n")
	sb.WriteString("agent-manager run diff <run-id>    # Code changes made\n")
	sb.WriteString("```\n\n")

	// Custom context from user
	if strings.TrimSpace(customContext) != "" {
		sb.WriteString("## Additional Context from User\n")
		sb.WriteString(customContext)
		sb.WriteString("\n\n")
	}

	// Report format
	sb.WriteString("## Required Report Format\n")
	sb.WriteString("Provide your findings in this structure:\n\n")
	sb.WriteString("### 1. Executive Summary\n")
	sb.WriteString("One-paragraph summary of what went wrong and why.\n\n")
	sb.WriteString("### 2. Root Cause Analysis\n")
	sb.WriteString("- **Primary cause** with file:line references\n")
	sb.WriteString("- **Contributing factors**\n")
	sb.WriteString("- **Evidence** (run IDs, event sequences, code snippets)\n\n")
	sb.WriteString("### 3. Recommendations\n")
	sb.WriteString("- **Immediate fix** (copy-pasteable code if possible)\n")
	sb.WriteString("- **Preventive measures**\n")
	sb.WriteString("- **Monitoring suggestions**\n")

	return sb.String()
}

func buildApplyPrompt(investigationRunID uuid.UUID, customContext string) string {
	var sb strings.Builder
	sb.WriteString("# Apply Investigation Recommendations\n\n")
	sb.WriteString("You are applying recommendations from a prior investigation run.\n\n")
	sb.WriteString(fmt.Sprintf("Investigation Run ID: %s\n\n", investigationRunID))

	sb.WriteString("## How to Fetch Full Investigation Data\n")
	sb.WriteString("If you need full details beyond the attachments, use the agent-manager CLI:\n")
	sb.WriteString(fmt.Sprintf("- agent-manager run get %s\n", investigationRunID))
	sb.WriteString(fmt.Sprintf("- agent-manager run events %s\n", investigationRunID))
	sb.WriteString(fmt.Sprintf("- agent-manager run diff %s\n\n", investigationRunID))

	if strings.TrimSpace(customContext) != "" {
		sb.WriteString("## Additional Context\n")
		sb.WriteString(customContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Your Task\n")
	sb.WriteString("1. Review the investigation report output.\n")
	sb.WriteString("2. Apply the recommended fixes explicitly called out by the user context.\n")
	sb.WriteString("3. If a fix requires code changes, implement them.\n")
	sb.WriteString("4. Summarize what you changed and why.\n")

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
		Description: "Agent profile for active agent-manager investigations (read-only analysis)",
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
