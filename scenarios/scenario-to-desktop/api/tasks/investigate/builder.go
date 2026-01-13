package investigate

import (
	"fmt"
	"strings"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

	"scenario-to-desktop-api/domain"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/tasks/shared"
)

// BuildPromptAndContext generates the investigation prompt and context attachments.
// This is a pure function: deterministic output for same input.
func BuildPromptAndContext(input shared.TaskInput) (shared.PromptResult, error) {
	if input.Request == nil {
		return shared.PromptResult{}, fmt.Errorf("request is required")
	}
	if input.Pipeline == nil {
		return shared.PromptResult{}, fmt.Errorf("pipeline is required")
	}

	// Determine include contexts
	includeContexts := input.Request.IncludeContexts
	if len(includeContexts) == 0 {
		includeContexts = shared.DefaultIncludeContexts
	}

	// Build base prompt based on effort and focus
	prompt := buildBasePrompt(input)

	// Build context attachments
	attachments := buildAttachments(input, includeContexts)

	return shared.PromptResult{
		Prompt:      prompt,
		Attachments: attachments,
	}, nil
}

// buildBasePrompt creates the concise base prompt for investigation.
func buildBasePrompt(input shared.TaskInput) string {
	var sb strings.Builder

	// Core task
	sb.WriteString(fmt.Sprintf("Investigate desktop build pipeline failure for %s.\n\n", input.Pipeline.ScenarioName))

	// Error summary if available
	errorSummary := shared.ExtractErrorSummary(input.Pipeline)
	if errorSummary != "" {
		sb.WriteString(fmt.Sprintf("Error: %s\n\n", errorSummary))
	}

	// Find failed stage
	failedStage := ""
	for stageName, result := range input.Pipeline.Stages {
		if result.Status == pipeline.StatusFailed {
			failedStage = stageName
			break
		}
	}
	if failedStage != "" {
		sb.WriteString(fmt.Sprintf("Failed Stage: %s\n\n", failedStage))
	}

	// Focus areas
	sb.WriteString("Focus areas: ")
	var focusAreas []string
	if input.Request.Focus.Harness {
		focusAreas = append(focusAreas, "build pipeline harness (scenario-to-desktop)")
	}
	if input.Request.Focus.Subject {
		focusAreas = append(focusAreas, fmt.Sprintf("target scenario (%s)", input.Pipeline.ScenarioName))
	}
	sb.WriteString(strings.Join(focusAreas, ", "))
	sb.WriteString("\n\n")

	// Effort-based instructions
	switch input.Request.Effort {
	case domain.EffortChecks:
		sb.WriteString("Mode: Quick checks - perform basic validation.\n")
		sb.WriteString("- Verify output files exist and have expected sizes\n")
		sb.WriteString("- Check for obvious configuration errors\n")
		sb.WriteString("- Review last few lines of build logs\n")
	case domain.EffortLogs:
		sb.WriteString("Mode: Log analysis - standard diagnostic procedures.\n")
		sb.WriteString("- Analyze build logs for error patterns\n")
		sb.WriteString("- Check npm install output for dependency issues\n")
		sb.WriteString("- Review electron-builder configuration\n")
		sb.WriteString("- Identify root cause and report findings\n")
	case domain.EffortTrace:
		sb.WriteString("Mode: Full trace - comprehensive analysis.\n")
		sb.WriteString("- Deep dive into all pipeline stages\n")
		sb.WriteString("- Trace through template generation\n")
		sb.WriteString("- Analyze all relevant logs and configurations\n")
		sb.WriteString("- Check scenario service.json and dependencies\n")
		sb.WriteString("- Identify root cause with supporting evidence\n")
	}
	sb.WriteString("\n")

	// Task mode
	sb.WriteString("Task: Report only - analyze and report, do not make changes.\n")

	return sb.String()
}

// buildAttachments creates the context attachments based on configuration.
func buildAttachments(input shared.TaskInput, includeContexts []string) []*domainpb.ContextAttachment {
	var attachments []*domainpb.ContextAttachment

	// Task Metadata (high priority)
	if shared.ContainsContext(includeContexts, "task-metadata") {
		mode := fmt.Sprintf("investigate:%s", input.Request.Effort)
		attachments = append(attachments, shared.BuildTaskMetadataAttachment(input.Pipeline, mode))
	}

	// Error Information (high priority)
	if shared.ContainsContext(includeContexts, "error-info") {
		if att := shared.BuildErrorInfoAttachment(input.Pipeline); att != nil {
			attachments = append(attachments, att)
		}
	}

	// Safety Rules (high priority)
	if shared.ContainsContext(includeContexts, "safety-rules") {
		attachments = append(attachments, shared.BuildSafetyRulesAttachment())
	}

	// Diagnostic Checklist (medium priority, only for logs/trace effort)
	if shared.ContainsContext(includeContexts, "diagnostic-checklist") {
		if input.Request.Effort == domain.EffortLogs || input.Request.Effort == domain.EffortTrace {
			attachments = append(attachments, shared.BuildDiagnosticChecklistAttachment())
		}
	}

	// Output Format (medium priority)
	if shared.ContainsContext(includeContexts, "output-format") {
		attachments = append(attachments, shared.BuildOutputFormatAttachment(false))
	}

	// Pipeline Configuration (medium priority)
	if shared.ContainsContext(includeContexts, "pipeline-config") && input.Pipeline.Config != nil {
		attachments = append(attachments, shared.BuildPipelineConfigAttachment(input.Pipeline.Config))
	}

	// Pipeline Results (high priority)
	if shared.ContainsContext(includeContexts, "pipeline-results") {
		attachments = append(attachments, shared.BuildPipelineResultsAttachment(input.Pipeline))
	}

	// Build Logs (high priority for build failures)
	if shared.ContainsContext(includeContexts, "build-logs") {
		if att := shared.BuildBuildLogsAttachment(input.Pipeline); att != nil {
			attachments = append(attachments, att)
		}
	}

	// Generator Connection / Filesystem paths (medium priority)
	if shared.ContainsContext(includeContexts, "generator-connection") {
		attachments = append(attachments, shared.BuildGeneratorConnectionAttachment(input.Pipeline))
	}

	// Architecture Guide (low priority, only for trace effort)
	if shared.ContainsContext(includeContexts, "architecture-guide") {
		if input.Request.Effort == domain.EffortTrace {
			attachments = append(attachments, shared.BuildArchitectureGuideAttachment())
		}
	}

	// Focus-specific attachments
	attachments = append(attachments, buildFocusAttachments(input)...)

	// User Note (medium priority if provided)
	if input.Request.Note != "" {
		if att := shared.BuildUserNoteAttachment(input.Request.Note); att != nil {
			attachments = append(attachments, att)
		}
	}

	return attachments
}

// buildFocusAttachments adds focus-specific context attachments.
func buildFocusAttachments(input shared.TaskInput) []*domainpb.ContextAttachment {
	var attachments []*domainpb.ContextAttachment

	if input.Request.Focus.Harness {
		attachments = append(attachments, shared.BuildHarnessFocusAttachment())
	}

	if input.Request.Focus.Subject {
		attachments = append(attachments, shared.BuildSubjectFocusAttachment(input.Pipeline.ScenarioName))
	}

	return attachments
}
