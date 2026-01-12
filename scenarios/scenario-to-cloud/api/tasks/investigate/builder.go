package investigate

import (
	"fmt"
	"strings"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/tasks/shared"
)

// BuildPromptAndContext generates the investigation prompt and context attachments.
// This is a pure function: deterministic output for same input.
func BuildPromptAndContext(input shared.TaskInput) (shared.PromptResult, error) {
	if input.Request == nil {
		return shared.PromptResult{}, fmt.Errorf("request is required")
	}
	if input.Deployment == nil {
		return shared.PromptResult{}, fmt.Errorf("deployment is required")
	}
	if input.Manifest == nil {
		return shared.PromptResult{}, fmt.Errorf("manifest is required")
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
	sb.WriteString(fmt.Sprintf("Investigate deployment failure for %s.\n\n", input.Manifest.Scenario.ID))

	// Error summary if available
	errorSummary := shared.ExtractErrorSummary(input.Deployment)
	if errorSummary != "" {
		sb.WriteString(fmt.Sprintf("Error: %s\n\n", errorSummary))
	}

	// Focus areas
	sb.WriteString("Focus areas: ")
	var focusAreas []string
	if input.Request.Focus.Harness {
		focusAreas = append(focusAreas, "deployment harness (scenario-to-cloud)")
	}
	if input.Request.Focus.Subject {
		focusAreas = append(focusAreas, fmt.Sprintf("target scenario (%s)", input.Manifest.Scenario.ID))
	}
	sb.WriteString(strings.Join(focusAreas, ", "))
	sb.WriteString("\n\n")

	// Effort-based instructions
	switch input.Request.Effort {
	case domain.EffortChecks:
		sb.WriteString("Mode: Quick checks - perform basic health checks and diagnostics.\n")
		sb.WriteString("- Check service status\n")
		sb.WriteString("- Verify port availability\n")
		sb.WriteString("- Check recent error logs (last 50 lines)\n")
	case domain.EffortLogs:
		sb.WriteString("Mode: Log analysis - standard diagnostic procedures.\n")
		sb.WriteString("- SSH into the VPS and check logs\n")
		sb.WriteString("- Analyze systemd service status\n")
		sb.WriteString("- Check ~/.vrooli/logs/ for resource and scenario logs\n")
		sb.WriteString("- Identify root cause and report findings\n")
	case domain.EffortTrace:
		sb.WriteString("Mode: Full trace - comprehensive analysis.\n")
		sb.WriteString("- SSH into the VPS for detailed investigation\n")
		sb.WriteString("- Trace through deployment steps\n")
		sb.WriteString("- Analyze all relevant logs and configurations\n")
		sb.WriteString("- Check service.json dependencies\n")
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

	vps := input.Manifest.Target.VPS
	sshUser, sshPort := shared.GetSSHDetails(vps)

	// Task Metadata (high priority)
	if shared.ContainsContext(includeContexts, "task-metadata") {
		mode := fmt.Sprintf("investigate:%s", input.Request.Effort)
		attachments = append(attachments, shared.BuildTaskMetadataAttachment(input.Deployment, input.Manifest, mode))
	}

	// Error Information (high priority)
	if shared.ContainsContext(includeContexts, "error-info") {
		if att := shared.BuildErrorInfoAttachment(input.Deployment); att != nil {
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

	// VPS Connection (medium priority)
	if shared.ContainsContext(includeContexts, "vps-connection") {
		attachments = append(attachments, shared.BuildVPSConnectionAttachment(vps, sshUser, sshPort, input.Manifest))
	}

	// Deployment Manifest (medium priority)
	if shared.ContainsContext(includeContexts, "deployment-manifest") {
		attachments = append(attachments, shared.BuildManifestAttachment(input.Manifest))
	}

	// Deploy Results (medium priority)
	if shared.ContainsContext(includeContexts, "deploy-results") && input.Deployment.DeployResult.Valid {
		attachments = append(attachments, shared.BuildDeployResultsAttachment(input.Deployment))
	}

	// Deployment History (low priority)
	if shared.ContainsContext(includeContexts, "deployment-history") {
		if att := shared.BuildDeploymentHistoryAttachment(input.Deployment); att != nil {
			attachments = append(attachments, att)
		}
	}

	// Preflight Results (low priority)
	if shared.ContainsContext(includeContexts, "preflight-results") {
		if att := shared.BuildPreflightResultsAttachment(input.Deployment); att != nil {
			attachments = append(attachments, att)
		}
	}

	// Setup Results (low priority)
	if shared.ContainsContext(includeContexts, "setup-results") {
		if att := shared.BuildSetupResultsAttachment(input.Deployment); att != nil {
			attachments = append(attachments, att)
		}
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
		attachments = append(attachments, buildHarnessFocusAttachment())
	}

	if input.Request.Focus.Subject {
		attachments = append(attachments, buildSubjectFocusAttachment(input.Manifest.Scenario.ID))
	}

	return attachments
}

// buildHarnessFocusAttachment creates focus guidance for harness investigation.
func buildHarnessFocusAttachment() *domainpb.ContextAttachment {
	content := `Harness Investigation Focus

When investigating the deployment harness (scenario-to-cloud), examine:

1. Deployment Pipeline
   - Caddy setup and TLS configuration
   - Resource startup order and timing
   - Health check execution

2. Key Paths
   - ~/Vrooli/scenarios/scenario-to-cloud/
   - Deployment manifest generation
   - VPS setup scripts

3. Common Harness Issues
   - Port conflicts in Caddy configuration
   - SSH connection failures
   - Resource dependency resolution
   - Bundling or transfer errors

4. Logs to Check
   - ~/.vrooli/logs/scenario-to-cloud*.log
   - Caddy logs: journalctl -u caddy
   - Deployment execution logs`

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "focus-harness",
		Tags:     []string{"focus", "harness"},
		Label:    "Harness Focus Guide",
		Summary:  "Investigation focus: deployment infrastructure (scenario-to-cloud)",
		Format:   "markdown",
		Priority: "medium",
		Content:  content,
	}
}

// buildSubjectFocusAttachment creates focus guidance for subject investigation.
func buildSubjectFocusAttachment(scenarioID string) *domainpb.ContextAttachment {
	content := fmt.Sprintf(`Subject Investigation Focus

When investigating the target scenario (%s), examine:

1. Scenario Configuration
   - .vrooli/service.json dependencies
   - Port declarations and health endpoints
   - Environment configuration

2. Key Paths
   - ~/Vrooli/scenarios/%s/
   - .vrooli/service.json
   - Application entry points

3. Common Subject Issues
   - Missing or incorrect dependencies
   - Configuration errors
   - Application startup failures
   - Health check endpoint issues

4. Logs to Check
   - ~/.vrooli/logs/%s*.log
   - Application-specific logs
   - systemd service logs if applicable`, scenarioID, scenarioID, scenarioID)

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "focus-subject",
		Tags:     []string{"focus", "subject"},
		Label:    "Subject Focus Guide",
		Summary:  fmt.Sprintf("Investigation focus: target scenario (%s)", scenarioID),
		Format:   "markdown",
		Priority: "medium",
		Content:  content,
	}
}
