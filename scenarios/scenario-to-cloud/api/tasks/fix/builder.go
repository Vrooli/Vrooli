package fix

import (
	"fmt"
	"strings"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/tasks/shared"
)

// BuildPromptAndContext generates the fix prompt and context attachments for one iteration.
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
	sb.WriteString(fmt.Sprintf("Fix the deployment failure for **%s**.\n\n", input.Manifest.Scenario.ID))

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
			if iter.DeployTriggered {
				sb.WriteString("- Deploy: Triggered\n")
			}
			sb.WriteString(fmt.Sprintf("- Verify: %s\n", iter.VerifyResult))
			sb.WriteString(fmt.Sprintf("- Outcome: %s\n\n", iter.Outcome))
		}
	}

	// Focus areas
	sb.WriteString("## Focus Scope\n")
	if input.Request.Focus.Harness {
		sb.WriteString("- **Harness** (scenario-to-cloud): Examine deployment infrastructure, Caddy config, resource startup\n")
	}
	if input.Request.Focus.Subject {
		sb.WriteString(fmt.Sprintf("- **Subject** (%s): Examine scenario code, service.json, runtime configuration\n", input.Manifest.Scenario.ID))
	}
	sb.WriteString("\n")

	// Permissions
	sb.WriteString("## Permissions\n")
	sb.WriteString("You are authorized to make the following types of changes:\n\n")

	if input.Request.Permissions.Immediate {
		sb.WriteString("### Immediate Fixes (ALLOWED)\n")
		sb.WriteString("- Restart services via SSH\n")
		sb.WriteString("- Clear disk space, logs, or temp files\n")
		sb.WriteString("- Fix file permissions\n")
		sb.WriteString("- Kill stuck processes\n\n")
	} else {
		sb.WriteString("### Immediate Fixes (NOT ALLOWED)\n")
		sb.WriteString("Do NOT run commands that change VPS state.\n\n")
	}

	if input.Request.Permissions.Permanent {
		sb.WriteString("### Permanent Fixes (ALLOWED)\n")
		sb.WriteString("- Modify .vrooli/service.json\n")
		sb.WriteString("- Fix configuration files\n")
		sb.WriteString("- Update deployment scripts\n")
		sb.WriteString("- Leave changes uncommitted for user review\n\n")
	} else {
		sb.WriteString("### Permanent Fixes (NOT ALLOWED)\n")
		sb.WriteString("Do NOT modify code or configuration files.\n\n")
	}

	if input.Request.Permissions.Prevention {
		sb.WriteString("### Prevention (ALLOWED)\n")
		sb.WriteString("- Add health checks\n")
		sb.WriteString("- Improve error messages\n")
		sb.WriteString("- Add monitoring or alerts\n")
		sb.WriteString("- Document fixes\n\n")
	} else {
		sb.WriteString("### Prevention (NOT ALLOWED)\n")
		sb.WriteString("Do NOT add preventive measures.\n\n")
	}

	// Deployment commands
	vps := input.Manifest.Target.VPS
	sshUser, sshPort := shared.GetSSHDetails(vps)

	sb.WriteString("## Deployment Commands\n\n")
	sb.WriteString("### To trigger a full redeploy (after permanent fixes):\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST http://localhost:15010/api/v1/deployments/%s/execute\n", input.Deployment.ID))
	sb.WriteString("# Poll for completion:\n")
	sb.WriteString(fmt.Sprintf("curl -s http://localhost:15010/api/v1/deployments/%s | jq .deployment.status\n", input.Deployment.ID))
	sb.WriteString("```\n\n")

	sb.WriteString("### To quick restart via SSH (for immediate fixes):\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("ssh -i %s -p %d %s@%s \"cd %s && vrooli scenario stop %s && vrooli scenario start %s\"\n",
		vps.KeyPath, sshPort, sshUser, vps.Host, vps.Workdir, input.Manifest.Scenario.ID, input.Manifest.Scenario.ID))
	sb.WriteString("```\n\n")

	// Verification
	sb.WriteString("## Verification Commands\n")
	sb.WriteString("After applying fixes, verify success:\n")
	sb.WriteString("```bash\n")
	if input.Manifest.Edge.Domain != "" {
		sb.WriteString(fmt.Sprintf("# Health check\ncurl -fsS https://%s/health\n\n", input.Manifest.Edge.Domain))
	}
	sb.WriteString(fmt.Sprintf("# Service status via SSH\nssh -i %s -p %d %s@%s \"systemctl status caddy --no-pager && vrooli scenario status %s\"\n",
		vps.KeyPath, sshPort, sshUser, vps.Host, input.Manifest.Scenario.ID))
	sb.WriteString("```\n\n")

	// Required output format
	sb.WriteString("## Required Output Format\n")
	sb.WriteString("At the end of your work, provide this JSON block:\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{
  "iteration_report": {
    "diagnosis": "Brief description of what you found",
    "changes_made": ["list", "of", "changes"],
    "deploy_triggered": true,
    "verification_result": "pass|fail|skip",
    "outcome": "success|continue|gave_up",
    "notes": "Any additional context"
  }
}
`)
	sb.WriteString("```\n\n")

	sb.WriteString("**outcome values:**\n")
	sb.WriteString("- `success`: The fix worked, deployment is healthy\n")
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

	vps := input.Manifest.Target.VPS
	sshUser, sshPort := shared.GetSSHDetails(vps)

	// Task Metadata (high priority)
	mode := fmt.Sprintf("fix:%s", input.Request.Permissions.String())
	attachments = append(attachments, shared.BuildTaskMetadataAttachment(input.Deployment, input.Manifest, mode))

	// Error Information (high priority)
	if att := shared.BuildErrorInfoAttachment(input.Deployment); att != nil {
		attachments = append(attachments, att)
	}

	// Safety Rules (high priority)
	attachments = append(attachments, shared.BuildSafetyRulesAttachment())

	// VPS Connection (high priority for fix tasks)
	connAtt := shared.BuildVPSConnectionAttachment(vps, sshUser, sshPort, input.Manifest)
	connAtt.Priority = "high" // Elevate for fix tasks
	attachments = append(attachments, connAtt)

	// Deployment Manifest (medium priority)
	attachments = append(attachments, shared.BuildManifestAttachment(input.Manifest))

	// Deploy Results (medium priority)
	if input.Deployment.DeployResult.Valid {
		attachments = append(attachments, shared.BuildDeployResultsAttachment(input.Deployment))
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
		attachments = append(attachments, buildSubjectFocusAttachment(input.Manifest.Scenario.ID, input.Request.Permissions))
	}

	return attachments
}

// buildHarnessFocusAttachment creates focus guidance for harness fixes.
func buildHarnessFocusAttachment(perms domain.FixPermissions) *domainpb.ContextAttachment {
	var content strings.Builder
	content.WriteString("Harness Fix Focus\n\n")
	content.WriteString("When fixing the deployment harness (scenario-to-cloud):\n\n")

	if perms.Immediate {
		content.WriteString("**Immediate Fixes:**\n")
		content.WriteString("- Restart Caddy: `systemctl restart caddy`\n")
		content.WriteString("- Clear Caddy cache: `rm -rf /var/cache/caddy/*`\n")
		content.WriteString("- Restart failed resources: `vrooli resource stop <name> && vrooli resource start <name>`\n\n")
	}

	if perms.Permanent {
		content.WriteString("**Permanent Fixes:**\n")
		content.WriteString("- Edit deployment manifest generation in scenario-to-cloud\n")
		content.WriteString("- Fix port allocation in Caddyfile templates\n")
		content.WriteString("- Update resource dependency ordering\n\n")
	}

	if perms.Prevention {
		content.WriteString("**Prevention:**\n")
		content.WriteString("- Add health check retries\n")
		content.WriteString("- Improve error messages in deployment steps\n")
		content.WriteString("- Add deployment pre-flight checks\n\n")
	}

	content.WriteString("**Key Paths:**\n")
	content.WriteString("- ~/Vrooli/scenarios/scenario-to-cloud/\n")
	content.WriteString("- ~/.vrooli/logs/scenario-to-cloud*.log\n")

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "focus-harness-fix",
		Tags:     []string{"focus", "harness", "fix"},
		Label:    "Harness Fix Guide",
		Summary:  "Fix guidance for deployment infrastructure",
		Format:   "markdown",
		Priority: "medium",
		Content:  content.String(),
	}
}

// buildSubjectFocusAttachment creates focus guidance for subject fixes.
func buildSubjectFocusAttachment(scenarioID string, perms domain.FixPermissions) *domainpb.ContextAttachment {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Subject Fix Focus (%s)\n\n", scenarioID))
	content.WriteString("When fixing the target scenario:\n\n")

	if perms.Immediate {
		content.WriteString("**Immediate Fixes:**\n")
		content.WriteString(fmt.Sprintf("- Restart scenario: `vrooli scenario stop %s && vrooli scenario start %s`\n", scenarioID, scenarioID))
		content.WriteString("- Clear application cache/temp files\n")
		content.WriteString("- Kill stuck application processes\n\n")
	}

	if perms.Permanent {
		content.WriteString("**Permanent Fixes:**\n")
		content.WriteString(fmt.Sprintf("- Edit .vrooli/service.json in scenarios/%s/\n", scenarioID))
		content.WriteString("- Fix missing dependencies\n")
		content.WriteString("- Update port declarations or health endpoints\n\n")
	}

	if perms.Prevention {
		content.WriteString("**Prevention:**\n")
		content.WriteString("- Add health check endpoint if missing\n")
		content.WriteString("- Document configuration requirements\n")
		content.WriteString("- Add startup validation\n\n")
	}

	content.WriteString("**Key Paths:**\n")
	content.WriteString(fmt.Sprintf("- ~/Vrooli/scenarios/%s/\n", scenarioID))
	content.WriteString(fmt.Sprintf("- ~/Vrooli/scenarios/%s/.vrooli/service.json\n", scenarioID))
	content.WriteString(fmt.Sprintf("- ~/.vrooli/logs/%s*.log\n", scenarioID))

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "focus-subject-fix",
		Tags:     []string{"focus", "subject", "fix"},
		Label:    "Subject Fix Guide",
		Summary:  fmt.Sprintf("Fix guidance for target scenario (%s)", scenarioID),
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
		content.WriteString(fmt.Sprintf("  Deploy Triggered: %v\n", iter.DeployTriggered))
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
