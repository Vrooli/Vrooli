package fix

import (
	"fmt"
	"strings"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

	"scenario-to-desktop-api/domain"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/tasks/shared"
)

// BuildPromptAndContext generates the fix prompt and context attachments for one iteration.
// This is a pure function: deterministic output for same input.
func BuildPromptAndContext(input shared.TaskInput) (shared.PromptResult, error) {
	if input.Request == nil {
		return shared.PromptResult{}, fmt.Errorf("request is required")
	}
	if input.Pipeline == nil {
		return shared.PromptResult{}, fmt.Errorf("pipeline is required")
	}

	// Build base prompt for this iteration
	prompt := buildIterationPrompt(input)

	// Build context attachments
	attachments := buildAttachments(input)

	return shared.PromptResult{
		Prompt:      prompt,
		Attachments: attachments,
	}, nil
}

// buildIterationPrompt creates the prompt for one fix iteration.
func buildIterationPrompt(input shared.TaskInput) string {
	var sb strings.Builder

	maxIterations := input.Request.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 5
	}

	// Header
	sb.WriteString(fmt.Sprintf("# Fix Task - Iteration %d/%d\n\n", input.Iteration, maxIterations))

	// Mission
	sb.WriteString("## Mission\n")
	sb.WriteString(fmt.Sprintf("Fix the desktop build pipeline failure for **%s**.\n\n", input.Pipeline.ScenarioName))

	// Previous iterations summary
	if len(input.PreviousIterations) > 0 {
		sb.WriteString("## Previous Iterations\n")
		for _, iter := range input.PreviousIterations {
			sb.WriteString(fmt.Sprintf("### Iteration %d\n", iter.Number))
			if iter.DiagnosisSummary != "" {
				sb.WriteString(fmt.Sprintf("- Diagnosis: %s\n", iter.DiagnosisSummary))
			}
			if iter.ChangesSummary != "" {
				sb.WriteString(fmt.Sprintf("- Changes: %s\n", iter.ChangesSummary))
			}
			if iter.RebuildTriggered {
				sb.WriteString("- Rebuild: Triggered\n")
			}
			sb.WriteString(fmt.Sprintf("- Verify: %s\n", iter.VerifyResult))
			sb.WriteString(fmt.Sprintf("- Outcome: %s\n\n", iter.Outcome))
		}
	}

	// Failed stage info
	failedStage := ""
	for stageName, result := range input.Pipeline.Stages {
		if result.Status == pipeline.StatusFailed {
			failedStage = stageName
			break
		}
	}
	if failedStage != "" {
		sb.WriteString(fmt.Sprintf("## Failed Stage: %s\n\n", failedStage))
	}

	// Focus areas
	sb.WriteString("## Focus Scope\n")
	if input.Request.Focus.Harness {
		sb.WriteString("- **Harness** (scenario-to-desktop): Examine build pipeline, template generation, Electron config\n")
	}
	if input.Request.Focus.Subject {
		sb.WriteString(fmt.Sprintf("- **Subject** (%s): Examine scenario code, service.json, dependencies\n", input.Pipeline.ScenarioName))
	}
	sb.WriteString("\n")

	// Permissions
	sb.WriteString("## Permissions\n")
	sb.WriteString("You are authorized to make the following types of changes:\n\n")

	if input.Request.Permissions.Immediate {
		sb.WriteString("### Immediate Fixes (ALLOWED)\n")
		sb.WriteString("- Trigger pipeline rebuild\n")
		sb.WriteString("- Clear build caches (node_modules, dist)\n")
		sb.WriteString("- Fix file permissions\n")
		sb.WriteString("- Kill stuck build processes\n\n")
	} else {
		sb.WriteString("### Immediate Fixes (NOT ALLOWED)\n")
		sb.WriteString("Do NOT run commands that change build state.\n\n")
	}

	if input.Request.Permissions.Permanent {
		sb.WriteString("### Permanent Fixes (ALLOWED)\n")
		sb.WriteString("- Modify scenario's .vrooli/service.json\n")
		sb.WriteString("- Fix package.json dependencies\n")
		sb.WriteString("- Update electron-builder configuration\n")
		sb.WriteString("- Leave changes uncommitted for user review\n\n")
	} else {
		sb.WriteString("### Permanent Fixes (NOT ALLOWED)\n")
		sb.WriteString("Do NOT modify code or configuration files.\n\n")
	}

	if input.Request.Permissions.Prevention {
		sb.WriteString("### Prevention (ALLOWED)\n")
		sb.WriteString("- Add preflight validation checks\n")
		sb.WriteString("- Improve error messages\n")
		sb.WriteString("- Add build verification steps\n")
		sb.WriteString("- Document fixes\n\n")
	} else {
		sb.WriteString("### Prevention (NOT ALLOWED)\n")
		sb.WriteString("Do NOT add preventive measures.\n\n")
	}

	// Build commands
	sb.WriteString("## Build Commands\n\n")
	sb.WriteString("### To trigger a full rebuild (after permanent fixes):\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST http://localhost:15020/api/v1/pipeline -H 'Content-Type: application/json' -d '{\"scenario_name\": \"%s\"}'\n", input.Pipeline.ScenarioName))
	sb.WriteString("# Poll for completion:\n")
	sb.WriteString(fmt.Sprintf("curl -s http://localhost:15020/api/v1/pipeline/%s | jq .status\n", input.Pipeline.PipelineID))
	sb.WriteString("```\n\n")

	sb.WriteString("### To rebuild just the desktop app (for immediate fixes):\n")
	sb.WriteString("```bash\n")
	sb.WriteString("cd <desktop-output-dir>\n")
	sb.WriteString("rm -rf node_modules dist\n")
	sb.WriteString("npm install && npm run build\n")
	sb.WriteString("```\n\n")

	// Verification
	sb.WriteString("## Verification\n")
	sb.WriteString("After applying fixes, verify success:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Check if artifacts were created\n")
	sb.WriteString("ls -la <desktop-output-dir>/dist/\n\n")
	sb.WriteString("# Check for expected outputs (dmg, exe, AppImage)\n")
	sb.WriteString("find <desktop-output-dir>/dist -name '*.dmg' -o -name '*.exe' -o -name '*.AppImage'\n\n")
	sb.WriteString("# Run smoke test if available\n")
	sb.WriteString("curl -X POST http://localhost:15020/api/v1/smoketest/run -H 'Content-Type: application/json' -d '{\"desktop_path\": \"<desktop-output-dir>\"}'\n")
	sb.WriteString("```\n\n")

	// Required output format
	sb.WriteString("## Required Output Format\n")
	sb.WriteString("At the end of your work, provide this JSON block:\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{
  "iteration_report": {
    "diagnosis": "Brief description of what you found",
    "changes_made": ["list", "of", "changes"],
    "rebuild_triggered": true,
    "verification_result": "pass|fail|skip",
    "outcome": "success|continue|gave_up",
    "notes": "Any additional context"
  }
}
`)
	sb.WriteString("```\n\n")

	sb.WriteString("**outcome values:**\n")
	sb.WriteString("- `success`: The fix worked, build completed successfully\n")
	sb.WriteString("- `continue`: More work needed, continue to next iteration\n")
	sb.WriteString("- `gave_up`: The issue cannot be fixed with available permissions/scope\n\n")

	// Source investigation findings if available
	if input.SourceFindings != nil && *input.SourceFindings != "" {
		sb.WriteString("## Source Investigation Findings\n")
		sb.WriteString("Use these findings to guide your fix approach:\n\n")
		sb.WriteString("<investigation_findings>\n")
		sb.WriteString(*input.SourceFindings)
		sb.WriteString("\n</investigation_findings>\n\n")
	}

	return sb.String()
}

// buildAttachments creates the context attachments for a fix iteration.
func buildAttachments(input shared.TaskInput) []*domainpb.ContextAttachment {
	var attachments []*domainpb.ContextAttachment

	// Task Metadata (high priority)
	mode := fmt.Sprintf("fix:%s", input.Request.Permissions.String())
	attachments = append(attachments, shared.BuildTaskMetadataAttachment(input.Pipeline, mode))

	// Error Information (high priority)
	if att := shared.BuildErrorInfoAttachment(input.Pipeline); att != nil {
		attachments = append(attachments, att)
	}

	// Safety Rules (high priority)
	attachments = append(attachments, shared.BuildSafetyRulesAttachment())

	// Generator Connection / Filesystem paths (high priority for fix tasks)
	connAtt := shared.BuildGeneratorConnectionAttachment(input.Pipeline)
	connAtt.Priority = "high" // Elevate for fix tasks
	attachments = append(attachments, connAtt)

	// Pipeline Configuration (medium priority)
	if input.Pipeline.Config != nil {
		attachments = append(attachments, shared.BuildPipelineConfigAttachment(input.Pipeline.Config))
	}

	// Pipeline Results (high priority)
	attachments = append(attachments, shared.BuildPipelineResultsAttachment(input.Pipeline))

	// Build Logs (high priority for build failures)
	if att := shared.BuildBuildLogsAttachment(input.Pipeline); att != nil {
		attachments = append(attachments, att)
	}

	// Focus-specific attachments
	attachments = append(attachments, buildFocusAttachments(input)...)

	// Iteration state attachment
	if len(input.PreviousIterations) > 0 {
		attachments = append(attachments, buildIterationStateAttachment(input))
	}

	// User Note (medium priority if provided)
	if input.Request.Note != "" {
		if att := shared.BuildUserNoteAttachment(input.Request.Note); att != nil {
			attachments = append(attachments, att)
		}
	}

	return attachments
}

// buildFocusAttachments adds focus-specific context for fix tasks.
func buildFocusAttachments(input shared.TaskInput) []*domainpb.ContextAttachment {
	var attachments []*domainpb.ContextAttachment

	if input.Request.Focus.Harness {
		attachments = append(attachments, buildHarnessFocusAttachment(input.Request.Permissions))
	}

	if input.Request.Focus.Subject {
		attachments = append(attachments, buildSubjectFocusAttachment(input.Pipeline.ScenarioName, input.Request.Permissions))
	}

	return attachments
}

// buildHarnessFocusAttachment creates focus guidance for harness fixes.
func buildHarnessFocusAttachment(perms domain.FixPermissions) *domainpb.ContextAttachment {
	var content strings.Builder
	content.WriteString("Harness Fix Focus\n\n")
	content.WriteString("When fixing the build pipeline harness (scenario-to-desktop):\n\n")

	if perms.Immediate {
		content.WriteString("**Immediate Fixes:**\n")
		content.WriteString("- Clear node_modules: `rm -rf <desktop>/node_modules`\n")
		content.WriteString("- Clear build cache: `rm -rf <desktop>/dist`\n")
		content.WriteString("- Reinstall dependencies: `cd <desktop> && npm install`\n")
		content.WriteString("- Rebuild: `cd <desktop> && npm run build`\n\n")
	}

	if perms.Permanent {
		content.WriteString("**Permanent Fixes:**\n")
		content.WriteString("- Edit template files in scenario-to-desktop/runtime/templates/\n")
		content.WriteString("- Fix electron-builder.config.js generation\n")
		content.WriteString("- Update package.json template\n")
		content.WriteString("- Fix icon generation logic\n\n")
	}

	if perms.Prevention {
		content.WriteString("**Prevention:**\n")
		content.WriteString("- Add preflight checks for npm/node version\n")
		content.WriteString("- Improve error messages in build stage\n")
		content.WriteString("- Add validation for electron-builder config\n\n")
	}

	content.WriteString("**Key Paths:**\n")
	content.WriteString("- ~/Vrooli/scenarios/scenario-to-desktop/\n")
	content.WriteString("- ~/Vrooli/scenarios/scenario-to-desktop/runtime/templates/\n")
	content.WriteString("- <scenario>/desktop/ (generated output)\n")

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "focus-harness-fix",
		Tags:     []string{"focus", "harness", "fix"},
		Label:    "Harness Fix Guide",
		Summary:  "Fix guidance for build pipeline infrastructure",
		Format:   "markdown",
		Priority: "medium",
		Content:  content.String(),
	}
}

// buildSubjectFocusAttachment creates focus guidance for subject fixes.
func buildSubjectFocusAttachment(scenarioName string, perms domain.FixPermissions) *domainpb.ContextAttachment {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Subject Fix Focus (%s)\n\n", scenarioName))
	content.WriteString("When fixing the target scenario:\n\n")

	if perms.Immediate {
		content.WriteString("**Immediate Fixes:**\n")
		content.WriteString("- Clean and rebuild: `rm -rf <desktop>/node_modules && npm install`\n")
		content.WriteString("- Fix runtime errors in bundled content\n")
		content.WriteString("- Clear scenario caches\n\n")
	}

	if perms.Permanent {
		content.WriteString("**Permanent Fixes:**\n")
		content.WriteString(fmt.Sprintf("- Edit .vrooli/service.json in scenarios/%s/\n", scenarioName))
		content.WriteString("- Fix missing dependencies in scenario's package.json\n")
		content.WriteString("- Update scenario configuration\n")
		content.WriteString("- Fix icon or asset paths\n\n")
	}

	if perms.Prevention {
		content.WriteString("**Prevention:**\n")
		content.WriteString("- Add build validation in service.json\n")
		content.WriteString("- Document required dependencies\n")
		content.WriteString("- Add pre-bundle checks\n\n")
	}

	content.WriteString("**Key Paths:**\n")
	content.WriteString(fmt.Sprintf("- ~/Vrooli/scenarios/%s/\n", scenarioName))
	content.WriteString(fmt.Sprintf("- ~/Vrooli/scenarios/%s/.vrooli/service.json\n", scenarioName))
	content.WriteString(fmt.Sprintf("- ~/Vrooli/scenarios/%s/ui/ (if applicable)\n", scenarioName))

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "focus-subject-fix",
		Tags:     []string{"focus", "subject", "fix"},
		Label:    "Subject Fix Guide",
		Summary:  fmt.Sprintf("Fix guidance for target scenario (%s)", scenarioName),
		Format:   "markdown",
		Priority: "medium",
		Content:  content.String(),
	}
}

// buildIterationStateAttachment creates a context attachment with iteration history.
func buildIterationStateAttachment(input shared.TaskInput) *domainpb.ContextAttachment {
	var content strings.Builder
	content.WriteString("Previous Iteration Results\n\n")

	for _, iter := range input.PreviousIterations {
		content.WriteString(fmt.Sprintf("Iteration %d:\n", iter.Number))
		content.WriteString(fmt.Sprintf("  Started: %s\n", iter.StartedAt.Format("15:04:05")))
		if !iter.EndedAt.IsZero() {
			content.WriteString(fmt.Sprintf("  Ended: %s\n", iter.EndedAt.Format("15:04:05")))
		}
		if iter.DiagnosisSummary != "" {
			content.WriteString(fmt.Sprintf("  Diagnosis: %s\n", iter.DiagnosisSummary))
		}
		if iter.ChangesSummary != "" {
			content.WriteString(fmt.Sprintf("  Changes: %s\n", iter.ChangesSummary))
		}
		content.WriteString(fmt.Sprintf("  Rebuild Triggered: %v\n", iter.RebuildTriggered))
		content.WriteString(fmt.Sprintf("  Verify Result: %s\n", iter.VerifyResult))
		content.WriteString(fmt.Sprintf("  Outcome: %s\n\n", iter.Outcome))
	}

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "iteration-state",
		Tags:     []string{"iteration", "history"},
		Label:    "Iteration History",
		Summary:  fmt.Sprintf("%d previous iteration(s)", len(input.PreviousIterations)),
		Format:   "yaml",
		Priority: "high",
		Content:  content.String(),
	}
}
