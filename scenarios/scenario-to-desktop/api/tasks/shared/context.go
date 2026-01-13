package shared

import (
	"encoding/json"
	"fmt"
	"strings"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

	"scenario-to-desktop-api/pipeline"
)

// DefaultIncludeContexts lists context items included by default if none specified.
var DefaultIncludeContexts = []string{
	"task-metadata",
	"error-info",
	"safety-rules",
	"diagnostic-checklist",
	"output-format",
	"pipeline-config",
	"pipeline-results",
	"build-logs",
	"architecture-guide",
}

// ContainsContext checks if a context key is in the include list.
func ContainsContext(includeContexts []string, key string) bool {
	for _, c := range includeContexts {
		if c == key {
			return true
		}
	}
	return false
}

// SafeDeref safely dereferences a string pointer, returning a default if nil.
func SafeDeref(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}

// ExtractErrorSummary extracts a one-line summary of the pipeline error.
func ExtractErrorSummary(pipelineStatus *pipeline.Status) string {
	if pipelineStatus.Error == "" {
		return ""
	}

	msg := pipelineStatus.Error

	// Common patterns in build errors
	if idx := strings.Index(msg, "npm ERR!"); idx != -1 {
		end := strings.Index(msg[idx:], "\n")
		if end != -1 {
			return msg[idx : idx+end]
		}
		return msg[idx:min(idx+80, len(msg))]
	}

	// For other errors, take first line or first 100 chars
	if idx := strings.Index(msg, "\n"); idx != -1 {
		return msg[:idx]
	}
	if len(msg) > 100 {
		return msg[:100] + "..."
	}
	return msg
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BuildTaskMetadataAttachment creates the task metadata context.
func BuildTaskMetadataAttachment(pipelineStatus *pipeline.Status, mode string) *domainpb.ContextAttachment {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("pipeline_id: %s\n", pipelineStatus.PipelineID))
	content.WriteString(fmt.Sprintf("scenario: %s\n", pipelineStatus.ScenarioName))
	content.WriteString(fmt.Sprintf("status: %s\n", pipelineStatus.Status))
	content.WriteString(fmt.Sprintf("mode: %s\n", mode))
	if pipelineStatus.CurrentStage != "" {
		content.WriteString(fmt.Sprintf("current_stage: %s\n", pipelineStatus.CurrentStage))
	}

	// Find failed stage
	failedStage := ""
	for stageName, result := range pipelineStatus.Stages {
		if result.Status == pipeline.StatusFailed {
			failedStage = stageName
			break
		}
	}
	if failedStage != "" {
		content.WriteString(fmt.Sprintf("failed_stage: %s\n", failedStage))
	}

	summary := fmt.Sprintf("Pipeline %s", pipelineStatus.ScenarioName)
	if failedStage != "" {
		summary = fmt.Sprintf("Pipeline %s failed at %s", pipelineStatus.ScenarioName, failedStage)
	}

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "task-metadata",
		Tags:     []string{"metadata", "task"},
		Label:    "Task Metadata",
		Summary:  summary,
		Format:   "yaml",
		Priority: "high",
		Content:  content.String(),
	}
}

// BuildErrorInfoAttachment creates the error information context with summarization.
func BuildErrorInfoAttachment(pipelineStatus *pipeline.Status) *domainpb.ContextAttachment {
	var content strings.Builder
	var summary string

	// Find the failed stage and its error
	for stageName, result := range pipelineStatus.Stages {
		if result.Status == pipeline.StatusFailed {
			content.WriteString(fmt.Sprintf("Failed Stage: %s\n\n", stageName))
			if result.Error != "" {
				content.WriteString(fmt.Sprintf("Stage Error:\n%s\n", result.Error))
			}
			break
		}
	}

	if pipelineStatus.Error != "" {
		summary = ExtractErrorSummary(pipelineStatus)
		if content.Len() > 0 {
			content.WriteString("\n")
		}
		content.WriteString(fmt.Sprintf("Pipeline Error:\n%s", pipelineStatus.Error))
	}

	if content.Len() == 0 {
		return nil
	}

	if summary == "" {
		summary = "Pipeline error details"
	}

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "error-info",
		Tags:     []string{"error", "diagnosis"},
		Label:    "Error Information",
		Summary:  summary,
		Format:   "text",
		Priority: "high",
		Content:  content.String(),
	}
}

// BuildSafetyRulesAttachment creates the safety rules context.
func BuildSafetyRulesAttachment() *domainpb.ContextAttachment {
	content := `Before running any log or file query, verify:
- Max lines: <= 200
- Max bytes: <= 512KB
- Filter scope: specific files/directories (no system-wide searches)

Command Guidelines:
- Log files: Use tail -n 200 or head -n 200, never unbounded cat
- Process status: Use specific process names, not broad ps queries
- Find: Use -maxdepth to limit recursion
- npm/node: Always check for package.json first

Example safe commands:
- tail -n 200 <build-log>
- ps aux | grep -E "(electron|npm|node)" | head -20
- find <output-dir> -maxdepth 3 -type f -name "*.dmg" -o -name "*.exe"
- npm run build 2>&1 | tail -n 200`

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "safety-rules",
		Tags:     []string{"rules", "safety"},
		Label:    "Safety Rules",
		Summary:  "Command safety limits: max 200 lines, 512KB, bounded queries only",
		Format:   "text",
		Priority: "high",
		Content:  content,
	}
}

// BuildDiagnosticChecklistAttachment creates the diagnostic checklist.
func BuildDiagnosticChecklistAttachment() *domainpb.ContextAttachment {
	content := `Answer these questions during investigation:

1. Which stage failed?
   → Check task-metadata and pipeline-results attachments

2. Was the bundle created correctly?
   → Check if bundle stage completed, verify manifest

3. Did preflight validation pass?
   → Check preflight results for missing dependencies or secrets

4. Was the desktop wrapper generated?
   → Check for package.json, main.js, preload.js in output directory

5. Did npm install succeed?
   → Check build logs for dependency resolution errors

6. Did electron-builder run?
   → Check build logs for packaging errors, missing icons, signing issues

7. Did the smoke test pass?
   → If applicable, check if the app launched and responded

8. What domain does this belong to?
   → scenario-to-desktop (build harness) OR target scenario (the app)?
   → Cite specific evidence (paths, error strings, stage outputs)`

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "diagnostic-checklist",
		Tags:     []string{"checklist", "diagnosis"},
		Label:    "Diagnostic Checklist",
		Summary:  "8 questions to identify root cause and responsible domain",
		Format:   "text",
		Priority: "medium",
		Content:  content,
	}
}

// BuildOutputFormatAttachment creates the output format specification for investigations.
func BuildOutputFormatAttachment(autoFix bool) *domainpb.ContextAttachment {
	var content strings.Builder
	content.WriteString(`Report structure:

### Scope Decision
- Chosen domain: scenario-to-desktop OR target scenario
- Evidence: at least two concrete evidence points
- Confidence: high/medium/low

### Root Cause
What caused the failure. Distinguish symptoms from causes.
Example: "npm install failed" is a symptom; "missing peer dependency in package.json" is a cause.

### Evidence
- Answers to diagnostic questions
- Relevant log snippets
- Command outputs confirming the issue

### Impact
What is broken or not working.

### Immediate Fix
Commands to run locally to restore the build (hotfix).`)

	if autoFix {
		content.WriteString(`
If you applied fixes, list what you did.`)
	}

	content.WriteString(`

### Permanent Fix
Code/configuration changes needed so this doesn't recur:
- Missing dependencies in package.json
- Electron-builder configuration fixes
- Icon or asset generation issues
- Template generation fixes

### Prevention
Validation checks or pipeline improvements to catch this earlier.`)

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "output-format",
		Tags:     []string{"format", "report"},
		Label:    "Report Format",
		Summary:  "Required sections: Scope, Root Cause, Evidence, Impact, Fixes, Prevention",
		Format:   "markdown",
		Priority: "medium",
		Content:  content.String(),
	}
}

// BuildPipelineConfigAttachment creates the pipeline configuration context.
func BuildPipelineConfigAttachment(cfg *pipeline.Config) *domainpb.ContextAttachment {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("scenario: %s\n", cfg.ScenarioName))
	content.WriteString(fmt.Sprintf("deployment_mode: %s\n", cfg.GetDeploymentMode()))
	content.WriteString(fmt.Sprintf("template: %s\n", cfg.GetTemplateType()))

	if len(cfg.Platforms) > 0 {
		content.WriteString(fmt.Sprintf("platforms: [%s]\n", strings.Join(cfg.Platforms, ", ")))
	}
	if cfg.SkipPreflight {
		content.WriteString("skip_preflight: true\n")
	}
	if cfg.SkipSmokeTest {
		content.WriteString("skip_smoke_test: true\n")
	}
	if cfg.Clean {
		content.WriteString("clean: true\n")
	}
	if cfg.Sign {
		content.WriteString("sign: true\n")
	}
	if cfg.ProxyURL != "" {
		content.WriteString(fmt.Sprintf("proxy_url: %s\n", cfg.ProxyURL))
	}

	summary := fmt.Sprintf("%s: %s, template=%s", cfg.ScenarioName, cfg.GetDeploymentMode(), cfg.GetTemplateType())

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "pipeline-config",
		Tags:     []string{"config", "pipeline"},
		Label:    "Pipeline Configuration",
		Summary:  summary,
		Format:   "yaml",
		Priority: "medium",
		Content:  content.String(),
	}
}

// BuildPipelineResultsAttachment creates the pipeline stage results context.
func BuildPipelineResultsAttachment(pipelineStatus *pipeline.Status) *domainpb.ContextAttachment {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("pipeline_status: %s\n", pipelineStatus.Status))
	content.WriteString(fmt.Sprintf("progress: %.0f%%\n\n", pipelineStatus.Progress()*100))

	content.WriteString("stages:\n")
	for _, stageName := range pipelineStatus.StageOrder {
		result, ok := pipelineStatus.Stages[stageName]
		if !ok {
			content.WriteString(fmt.Sprintf("  - %s: pending\n", stageName))
			continue
		}

		statusIcon := "?"
		switch result.Status {
		case pipeline.StatusCompleted:
			statusIcon = "completed"
		case pipeline.StatusFailed:
			statusIcon = "FAILED"
		case pipeline.StatusRunning:
			statusIcon = "running"
		case pipeline.StatusSkipped:
			statusIcon = "skipped"
		case pipeline.StatusPending:
			statusIcon = "pending"
		}

		if result.CompletedAt > 0 {
			duration := float64(result.CompletedAt-result.StartedAt) / 1000.0
			content.WriteString(fmt.Sprintf("  - %s: %s (%.1fs)\n", stageName, statusIcon, duration))
		} else {
			content.WriteString(fmt.Sprintf("  - %s: %s\n", stageName, statusIcon))
		}

		if result.Error != "" {
			// Truncate long errors
			errMsg := result.Error
			if len(errMsg) > 200 {
				errMsg = errMsg[:200] + "..."
			}
			content.WriteString(fmt.Sprintf("      error: %s\n", errMsg))
		}
	}

	// Find failed stage for summary
	failedStage := ""
	for stageName, result := range pipelineStatus.Stages {
		if result.Status == pipeline.StatusFailed {
			failedStage = stageName
			break
		}
	}

	summary := fmt.Sprintf("Status: %s", pipelineStatus.Status)
	if failedStage != "" {
		summary = fmt.Sprintf("Failed at %s stage", failedStage)
	}

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "pipeline-results",
		Tags:     []string{"pipeline", "execution"},
		Label:    "Pipeline Results",
		Summary:  summary,
		Format:   "yaml",
		Priority: "high",
		Content:  content.String(),
	}
}

// BuildBuildLogsAttachment creates the build logs context.
func BuildBuildLogsAttachment(pipelineStatus *pipeline.Status) *domainpb.ContextAttachment {
	buildResult, ok := pipelineStatus.Stages[pipeline.StageBuild]
	if !ok {
		return nil
	}

	var content strings.Builder
	if len(buildResult.Logs) > 0 {
		// Take last 100 lines of logs
		logs := buildResult.Logs
		if len(logs) > 100 {
			logs = logs[len(logs)-100:]
			content.WriteString("... (truncated, showing last 100 lines)\n\n")
		}
		for _, line := range logs {
			content.WriteString(line + "\n")
		}
	}

	if buildResult.Error != "" {
		if content.Len() > 0 {
			content.WriteString("\n")
		}
		content.WriteString("Build Error:\n")
		content.WriteString(buildResult.Error)
	}

	if content.Len() == 0 {
		return nil
	}

	summary := "Build stage output"
	if buildResult.Status == pipeline.StatusFailed {
		summary = "Build stage failed - see logs for details"
	}

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "build-logs",
		Tags:     []string{"build", "logs"},
		Label:    "Build Logs",
		Summary:  summary,
		Format:   "text",
		Priority: "high",
		Content:  content.String(),
	}
}

// BuildGeneratorConnectionAttachment creates the local filesystem context.
func BuildGeneratorConnectionAttachment(pipelineStatus *pipeline.Status) *domainpb.ContextAttachment {
	var content strings.Builder

	// Get paths from config and stage results
	if pipelineStatus.Config != nil {
		content.WriteString(fmt.Sprintf("scenario: %s\n", pipelineStatus.Config.ScenarioName))
		content.WriteString(fmt.Sprintf("deployment_mode: %s\n", pipelineStatus.Config.GetDeploymentMode()))
	}

	content.WriteString("\nKey Paths:\n")
	content.WriteString("  vrooli_root: $VROOLI_ROOT or ~/Vrooli\n")
	content.WriteString("  scenario_path: <vrooli_root>/scenarios/<scenario_name>\n")
	content.WriteString("  desktop_output: <scenario_path>/desktop\n")
	content.WriteString("  build_output: <desktop_output>/dist\n")
	content.WriteString("  build_log: <desktop_output>/build.log\n")

	// Add artifact paths if available
	if len(pipelineStatus.FinalArtifacts) > 0 {
		content.WriteString("\nBuild Artifacts:\n")
		for platform, path := range pipelineStatus.FinalArtifacts {
			content.WriteString(fmt.Sprintf("  %s: %s\n", platform, path))
		}
	}

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "generator-connection",
		Tags:     []string{"paths", "filesystem"},
		Label:    "Generator Connection",
		Summary:  "Local filesystem paths for desktop generation",
		Format:   "yaml",
		Priority: "medium",
		Content:  content.String(),
	}
}

// BuildArchitectureGuideAttachment creates the architecture reference.
func BuildArchitectureGuideAttachment() *domainpb.ContextAttachment {
	content := `scenario-to-desktop Pipeline Architecture

Pipeline Stages:
1. bundle: Analyze scenario, resolve dependencies, create deployment manifest
2. preflight: Validate configuration, check prerequisites, verify secrets
3. generate: Create Electron wrapper (package.json, main.js, preload.js, icons)
4. build: Run npm install, execute electron-builder, produce artifacts
5. smoketest: Launch app, verify startup, check basic functionality

Common Failure Patterns:
- Bundle fails: Missing service.json, invalid scenario structure
- Preflight fails: Missing secrets, unsatisfied dependencies
- Generate fails: Template errors, icon generation issues
- Build fails: npm install errors, electron-builder config, signing issues
- Smoketest fails: App crash on startup, missing resources

Deployment Modes:
- bundled: Full offline app with embedded UI (default)
- proxy: Thin client connecting to external URL

Scope Determination:
- scenario-to-desktop issues: Pipeline stages, Electron config, build process
- Target scenario issues: Application code, dependencies, runtime behavior`

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "architecture-guide",
		Tags:     []string{"documentation", "reference"},
		Label:    "Architecture Guide",
		Summary:  "Pipeline stages and common failure patterns",
		Format:   "markdown",
		Priority: "low",
		Content:  content,
	}
}

// BuildUserNoteAttachment creates a context attachment for user-provided notes.
func BuildUserNoteAttachment(note string) *domainpb.ContextAttachment {
	if note == "" {
		return nil
	}
	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "user-note",
		Tags:     []string{"user", "context"},
		Label:    "User Note",
		Summary:  "Additional context from the user",
		Format:   "text",
		Priority: "medium",
		Content:  note,
	}
}

// BuildHarnessFocusAttachment creates the harness focus context.
func BuildHarnessFocusAttachment() *domainpb.ContextAttachment {
	content := `Focus: scenario-to-desktop Pipeline (Harness)

Areas to investigate:
- Bundle stage: Dependency analysis, manifest generation
- Preflight stage: Configuration validation, prerequisites
- Generate stage: Template processing, file generation
- Build stage: npm install, electron-builder execution
- Smoketest stage: App launch verification

Key files:
- scenarios/scenario-to-desktop/api/*.go - Pipeline logic
- scenarios/scenario-to-desktop/runtime/templates/ - Electron templates
- scenarios/scenario-to-desktop/runtime/assets/ - Default icons

Common harness issues:
- Template processing errors
- Icon conversion failures
- electron-builder configuration problems
- Missing build dependencies`

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "harness-focus",
		Tags:     []string{"focus", "harness"},
		Label:    "Harness Focus",
		Summary:  "Investigating scenario-to-desktop pipeline infrastructure",
		Format:   "markdown",
		Priority: "medium",
		Content:  content,
	}
}

// BuildSubjectFocusAttachment creates the subject focus context.
func BuildSubjectFocusAttachment(scenarioName string) *domainpb.ContextAttachment {
	content := fmt.Sprintf(`Focus: Target Scenario (%s)

Areas to investigate:
- Scenario structure: .vrooli/service.json, dependencies
- Application code: Runtime errors, missing resources
- Configuration: Environment variables, secrets
- Dependencies: Package versions, peer dependencies

Key files:
- scenarios/%s/.vrooli/service.json - Scenario configuration
- scenarios/%s/ui/ - UI code (if applicable)
- scenarios/%s/api/ - API code (if applicable)

Common scenario issues:
- Missing or misconfigured dependencies
- Runtime errors in application code
- Incorrect service.json configuration
- Missing environment variables`, scenarioName, scenarioName, scenarioName, scenarioName)

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "subject-focus",
		Tags:     []string{"focus", "subject"},
		Label:    "Subject Focus",
		Summary:  fmt.Sprintf("Investigating target scenario: %s", scenarioName),
		Format:   "markdown",
		Priority: "medium",
		Content:  content,
	}
}

// BuildSourceInvestigationAttachment creates context from a source investigation.
func BuildSourceInvestigationAttachment(sourceFindings string) *domainpb.ContextAttachment {
	if sourceFindings == "" {
		return nil
	}
	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "source-investigation",
		Tags:     []string{"investigation", "findings"},
		Label:    "Source Investigation",
		Summary:  "Findings from the previous investigation to guide fixes",
		Format:   "markdown",
		Priority: "high",
		Content:  sourceFindings,
	}
}

// BuildStageDetailsAttachment creates detailed context for a specific stage.
func BuildStageDetailsAttachment(stageName string, result *pipeline.StageResult) *domainpb.ContextAttachment {
	if result == nil {
		return nil
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("Stage: %s\n", stageName))
	content.WriteString(fmt.Sprintf("Status: %s\n", result.Status))

	if result.Error != "" {
		content.WriteString(fmt.Sprintf("\nError:\n%s\n", result.Error))
	}

	if result.Details != nil {
		detailsJSON, err := json.MarshalIndent(result.Details, "", "  ")
		if err == nil {
			content.WriteString(fmt.Sprintf("\nDetails:\n%s\n", string(detailsJSON)))
		}
	}

	if len(result.Logs) > 0 {
		content.WriteString("\nLogs:\n")
		logs := result.Logs
		if len(logs) > 50 {
			logs = logs[len(logs)-50:]
			content.WriteString("... (truncated, showing last 50 lines)\n")
		}
		for _, line := range logs {
			content.WriteString(line + "\n")
		}
	}

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      fmt.Sprintf("stage-%s-details", stageName),
		Tags:     []string{"stage", stageName},
		Label:    fmt.Sprintf("%s Stage Details", strings.Title(stageName)),
		Summary:  fmt.Sprintf("%s stage: %s", stageName, result.Status),
		Format:   "yaml",
		Priority: "medium",
		Content:  content.String(),
	}
}
