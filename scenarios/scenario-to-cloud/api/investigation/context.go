package investigation

import (
	"encoding/json"
	"fmt"
	"strings"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

	"scenario-to-cloud/domain"
)

// Default context items to include if none specified
var defaultIncludeContexts = []string{
	"task-metadata",
	"error-info",
	"safety-rules",
	"diagnostic-checklist",
	"output-format",
	"deployment-manifest",
	"vps-connection",
	"deploy-results",
	"architecture-guide",
}

// containsContext checks if a context key is in the include list.
func containsContext(includeContexts []string, key string) bool {
	for _, c := range includeContexts {
		if c == key {
			return true
		}
	}
	return false
}

// buildPromptAndContext constructs the base prompt and context attachments for investigation.
// Returns the base prompt (task instructions) and context attachments (selectable data).
//
// The prompt structure follows these principles:
// 1. Base prompt is concise - just the core task
// 2. All metadata, rules, and reference material are in context attachments
// 3. Each attachment has summary, format, and priority for AI optimization
func buildPromptAndContext(
	deployment *domain.Deployment,
	autoFix bool,
	note string,
	includeContexts []string,
) (string, []*domainpb.ContextAttachment, error) {
	// Use defaults if none specified
	if len(includeContexts) == 0 {
		includeContexts = defaultIncludeContexts
	}

	// Parse the manifest to get VPS details
	var m domain.CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &m); err != nil {
		return "", nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if m.Target.VPS == nil {
		return "", nil, fmt.Errorf("deployment has no VPS target")
	}

	vps := m.Target.VPS
	sshPort := vps.Port
	if sshPort == 0 {
		sshPort = 22
	}
	sshUser := vps.User
	if sshUser == "" {
		sshUser = "root"
	}

	// Extract error summary for the base prompt
	errorSummary := extractErrorSummary(deployment)

	// Build concise base prompt
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Investigate deployment failure for %s.\n\n", m.Scenario.ID))

	if errorSummary != "" {
		sb.WriteString(fmt.Sprintf("Error: %s\n\n", errorSummary))
	}

	sb.WriteString("SSH into the VPS, check logs, identify root cause, and report findings.\n\n")

	if autoFix {
		sb.WriteString("Mode: auto-fix (you may attempt safe fixes after diagnosis)\n")
	} else {
		sb.WriteString("Mode: report-only (analyze and report, do not make changes)\n")
	}

	basePrompt := sb.String()

	// Build context attachments
	var attachments []*domainpb.ContextAttachment

	// Task Metadata (high priority - read first)
	if containsContext(includeContexts, "task-metadata") {
		attachments = append(attachments, buildTaskMetadataAttachment(deployment, &m, autoFix))
	}

	// Error Information (high priority)
	if containsContext(includeContexts, "error-info") {
		if att := buildErrorInfoAttachment(deployment); att != nil {
			attachments = append(attachments, att)
		}
	}

	// Safety Rules (high priority - must read before running commands)
	if containsContext(includeContexts, "safety-rules") {
		attachments = append(attachments, buildSafetyRulesAttachment())
	}

	// Diagnostic Checklist (medium priority)
	if containsContext(includeContexts, "diagnostic-checklist") {
		attachments = append(attachments, buildDiagnosticChecklistAttachment())
	}

	// Output Format (medium priority)
	if containsContext(includeContexts, "output-format") {
		attachments = append(attachments, buildOutputFormatAttachment(autoFix))
	}

	// VPS Connection (medium priority)
	if containsContext(includeContexts, "vps-connection") {
		attachments = append(attachments, buildVPSConnectionAttachment(vps, sshUser, sshPort, &m))
	}

	// Deployment Manifest (medium priority)
	if containsContext(includeContexts, "deployment-manifest") {
		attachments = append(attachments, buildManifestAttachment(&m))
	}

	// Deploy Results (medium priority)
	if containsContext(includeContexts, "deploy-results") && deployment.DeployResult.Valid {
		attachments = append(attachments, buildDeployResultsAttachment(deployment))
	}

	// Deployment History (low priority)
	if containsContext(includeContexts, "deployment-history") && deployment.DeploymentHistory.Valid {
		attachments = append(attachments, &domainpb.ContextAttachment{
			Type:     "note",
			Key:      "deployment-history",
			Tags:     []string{"timeline", "events"},
			Label:    "Deployment History",
			Summary:  "Timeline of deployment events",
			Format:   "json",
			Priority: "low",
			Content:  string(deployment.DeploymentHistory.Data),
		})
	}

	// Preflight Results (low priority)
	if containsContext(includeContexts, "preflight-results") && deployment.PreflightResult.Valid {
		attachments = append(attachments, &domainpb.ContextAttachment{
			Type:     "note",
			Key:      "preflight-results",
			Tags:     []string{"preflight", "checks"},
			Label:    "Preflight Results",
			Summary:  "Pre-deployment validation checks",
			Format:   "json",
			Priority: "low",
			Content:  string(deployment.PreflightResult.Data),
		})
	}

	// Setup Results (low priority)
	if containsContext(includeContexts, "setup-results") && deployment.SetupResult.Valid {
		attachments = append(attachments, &domainpb.ContextAttachment{
			Type:     "note",
			Key:      "setup-results",
			Tags:     []string{"setup", "installation"},
			Label:    "Setup Results",
			Summary:  "VPS setup and installation outputs",
			Format:   "json",
			Priority: "low",
			Content:  string(deployment.SetupResult.Data),
		})
	}

	// Architecture Guide (low priority - reference material)
	if containsContext(includeContexts, "architecture-guide") {
		attachments = append(attachments, buildArchitectureGuideAttachment())
	}

	// User Note (medium priority if provided)
	if note != "" {
		attachments = append(attachments, &domainpb.ContextAttachment{
			Type:     "note",
			Key:      "user-note",
			Tags:     []string{"user", "context"},
			Label:    "User Note",
			Summary:  "Additional context from the user",
			Format:   "text",
			Priority: "medium",
			Content:  note,
		})
	}

	return basePrompt, attachments, nil
}

// extractErrorSummary extracts a one-line summary of the error.
func extractErrorSummary(deployment *domain.Deployment) string {
	if deployment.ErrorMessage == nil || *deployment.ErrorMessage == "" {
		return ""
	}

	msg := *deployment.ErrorMessage

	// Try to extract the most relevant part
	// Common patterns: "Caddy ACME validation...", "Cannot negotiate ALPN..."
	if idx := strings.Index(msg, "Cannot negotiate"); idx != -1 {
		// Extract the specific error
		end := strings.Index(msg[idx:], "\"}")
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

// buildTaskMetadataAttachment creates the task metadata context.
func buildTaskMetadataAttachment(deployment *domain.Deployment, m *domain.CloudManifest, autoFix bool) *domainpb.ContextAttachment {
	mode := "report-only"
	if autoFix {
		mode = "auto-fix"
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("deployment_id: %s\n", deployment.ID))
	content.WriteString(fmt.Sprintf("scenario: %s\n", m.Scenario.ID))
	content.WriteString(fmt.Sprintf("status: %s\n", deployment.Status))
	content.WriteString(fmt.Sprintf("mode: %s\n", mode))
	if deployment.ErrorStep != nil {
		content.WriteString(fmt.Sprintf("failed_step: %s\n", *deployment.ErrorStep))
	}

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "task-metadata",
		Tags:     []string{"metadata", "task"},
		Label:    "Task Metadata",
		Summary:  fmt.Sprintf("Deployment %s failed at %s", m.Scenario.ID, safeDeref(deployment.ErrorStep, "unknown step")),
		Format:   "yaml",
		Priority: "high",
		Content:  content.String(),
	}
}

func safeDeref(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}

// buildErrorInfoAttachment creates the error information context with summarization.
func buildErrorInfoAttachment(deployment *domain.Deployment) *domainpb.ContextAttachment {
	var content strings.Builder
	var summary string

	if deployment.ErrorStep != nil {
		content.WriteString(fmt.Sprintf("Failed Step: %s\n\n", *deployment.ErrorStep))
	}

	if deployment.ErrorMessage != nil && *deployment.ErrorMessage != "" {
		msg := *deployment.ErrorMessage

		// Parse and summarize the error
		summary = extractErrorSummary(deployment)

		// Check if it's a structured error with logs
		if strings.Contains(msg, " | ") {
			// Split pipe-separated log entries for readability
			content.WriteString("Error Details:\n")
			parts := strings.Split(msg, " | ")
			for _, part := range parts {
				content.WriteString(fmt.Sprintf("  - %s\n", strings.TrimSpace(part)))
			}
		} else {
			content.WriteString(fmt.Sprintf("Error Message:\n%s", msg))
		}
	}

	if content.Len() == 0 {
		return nil
	}

	if summary == "" {
		summary = "Deployment error details"
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

// buildSafetyRulesAttachment creates the safety rules context.
func buildSafetyRulesAttachment() *domainpb.ContextAttachment {
	content := `Before running any log or systemd query, verify:
- Max lines: <= 200
- Max bytes: <= 512KB
- Filter scope: single unit/path (no global/system-wide listings)

Command Guidelines:
- systemctl: Always use --no-pager --full --lines=50 or systemctl show -p ActiveState,SubState
- journalctl: Never exceed 200 lines. Use -n 200 max
- Log files: Use tail -n 200 or head -n 200, never unbounded cat
- SSH: Echo target paths before executing find/cat/grep/tar

Example safe commands:
- systemctl status <service> --no-pager --full --lines=50
- journalctl -u <service> --no-pager -n 200
- find "$HOME/.vrooli/logs" -maxdepth 3 -type f -name "*.log" -printf '%p\t%s\n' | head -200
- tail -n 200 <logfile>`

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

// buildDiagnosticChecklistAttachment creates the diagnostic checklist.
func buildDiagnosticChecklistAttachment() *domainpb.ContextAttachment {
	content := `Answer these questions during investigation:

1. Is the failing dependency in the manifest?
   → Check deployment-manifest attachment

2. If yes, did the resource/scenario start step succeed?
   → Check ~/.vrooli/logs/ for resource and scenario start logs

3. If no, is it declared in the scenario's service.json?
   → Check <workdir>/scenarios/<scenario>/.vrooli/service.json

4. Is the Vrooli CLI working?
   → Run: vrooli --version && vrooli resource list

5. Is this a configuration issue or transient failure?
   → Would a restart fix it, or is there a deeper problem?

6. What domain does this belong to?
   → scenario-to-cloud (deployment harness) OR target scenario (the app)?
   → Cite specific evidence (paths, error strings, service names)`

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "diagnostic-checklist",
		Tags:     []string{"checklist", "diagnosis"},
		Label:    "Diagnostic Checklist",
		Summary:  "6 questions to identify root cause and responsible domain",
		Format:   "text",
		Priority: "medium",
		Content:  content,
	}
}

// buildOutputFormatAttachment creates the output format specification.
func buildOutputFormatAttachment(autoFix bool) *domainpb.ContextAttachment {
	var content strings.Builder
	content.WriteString(`Report structure:

### Scope Decision
- Chosen domain: scenario-to-cloud OR target scenario
- Evidence: at least two concrete evidence points
- Confidence: high/medium/low

### Root Cause
What caused the failure. Distinguish symptoms from causes.
Example: "postgres not running" is a symptom; "postgres not in manifest" is a cause.

### Evidence
- Answers to diagnostic questions
- Relevant log snippets
- Command outputs confirming the issue

### Impact
What is broken or not working.

### Immediate Fix
Commands to run on this VPS to restore service (hotfix).`)

	if autoFix {
		content.WriteString(`
If you applied fixes, list what you did.`)
	}

	content.WriteString(`

### Permanent Fix
Code/configuration changes needed so this doesn't recur:
- Missing dependencies in .vrooli/service.json
- Manifest generation fixes
- Health checks or startup delays
- VPS setup process fixes

### Prevention
Monitoring, alerts, or pipeline improvements to catch this earlier.`)

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

// buildVPSConnectionAttachment creates the VPS connection details.
func buildVPSConnectionAttachment(vps *domain.ManifestVPS, sshUser string, sshPort int, m *domain.CloudManifest) *domainpb.ContextAttachment {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("ssh -i %s -p %d %s@%s \"<command>\"\n\n", vps.KeyPath, sshPort, sshUser, vps.Host))
	content.WriteString(fmt.Sprintf("host: %s\n", vps.Host))
	content.WriteString(fmt.Sprintf("user: %s\n", sshUser))
	content.WriteString(fmt.Sprintf("port: %d\n", sshPort))
	content.WriteString(fmt.Sprintf("key_path: %s\n", vps.KeyPath))
	if vps.Workdir != "" {
		content.WriteString(fmt.Sprintf("workdir: %s\n", vps.Workdir))
	}
	if m.Edge.Domain != "" {
		content.WriteString(fmt.Sprintf("domain: %s\n", m.Edge.Domain))
	}
	if len(m.Ports) > 0 {
		content.WriteString("\nports:\n")
		for name, port := range m.Ports {
			content.WriteString(fmt.Sprintf("  %s: %d\n", name, port))
		}
	}

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "vps-connection",
		Tags:     []string{"ssh", "access"},
		Label:    "VPS Connection",
		Summary:  fmt.Sprintf("SSH to %s@%s:%d", sshUser, vps.Host, sshPort),
		Format:   "yaml",
		Priority: "medium",
		Content:  content.String(),
	}
}

// buildManifestAttachment creates the deployment manifest context with summary.
func buildManifestAttachment(m *domain.CloudManifest) *domainpb.ContextAttachment {
	// Build summary of key manifest details
	var summaryParts []string
	if len(m.Dependencies.Resources) > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("resources: %s", strings.Join(m.Dependencies.Resources, ", ")))
	}
	if len(m.Ports) > 0 {
		var portStrs []string
		for name, port := range m.Ports {
			portStrs = append(portStrs, fmt.Sprintf("%s=%d", name, port))
		}
		summaryParts = append(summaryParts, strings.Join(portStrs, ", "))
	}
	summary := strings.Join(summaryParts, "; ")
	if summary == "" {
		summary = "Deployment configuration and dependencies"
	}

	manifestJSON, _ := json.MarshalIndent(m, "", "  ")

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "deployment-manifest",
		Tags:     []string{"config", "dependencies"},
		Label:    "Deployment Manifest",
		Summary:  summary,
		Format:   "json",
		Priority: "medium",
		Content:  string(manifestJSON),
	}
}

// buildDeployResultsAttachment creates the deploy results context with summarization.
func buildDeployResultsAttachment(deployment *domain.Deployment) *domainpb.ContextAttachment {
	// Parse the deploy result to extract key information
	var result struct {
		OK         bool   `json:"ok"`
		Error      string `json:"error"`
		FailedStep string `json:"failed_step"`
		Timestamp  string `json:"timestamp"`
		Steps      []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Command     string `json:"command"`
		} `json:"steps"`
	}

	rawData := string(deployment.DeployResult.Data)
	if err := json.Unmarshal(deployment.DeployResult.Data, &result); err != nil {
		// If we can't parse it, return raw data
		return &domainpb.ContextAttachment{
			Type:     "note",
			Key:      "deploy-results",
			Tags:     []string{"deployment", "execution"},
			Label:    "Deploy Results",
			Summary:  "Raw deployment execution output",
			Format:   "json",
			Priority: "medium",
			Content:  rawData,
		}
	}

	// Build a structured, readable summary
	var content strings.Builder
	content.WriteString(fmt.Sprintf("status: %s\n", boolToStatus(result.OK)))
	if result.FailedStep != "" {
		content.WriteString(fmt.Sprintf("failed_step: %s\n", result.FailedStep))
	}
	if result.Timestamp != "" {
		content.WriteString(fmt.Sprintf("timestamp: %s\n", result.Timestamp))
	}
	content.WriteString("\n")

	// Summarize the error
	if result.Error != "" {
		content.WriteString("error_summary:\n")
		// Split pipe-separated log entries
		if strings.Contains(result.Error, " | ") {
			parts := strings.Split(result.Error, " | ")
			for _, part := range parts {
				content.WriteString(fmt.Sprintf("  - %s\n", strings.TrimSpace(part)))
			}
		} else {
			content.WriteString(fmt.Sprintf("  %s\n", result.Error))
		}
		content.WriteString("\n")
	}

	// List steps with their status
	if len(result.Steps) > 0 {
		content.WriteString("steps_executed:\n")
		for _, step := range result.Steps {
			status := "✓"
			if step.ID == result.FailedStep {
				status = "✗ FAILED"
			}
			content.WriteString(fmt.Sprintf("  - [%s] %s: %s\n", status, step.ID, step.Title))
		}
	}

	summary := fmt.Sprintf("Failed at %s", result.FailedStep)
	if result.FailedStep == "" {
		summary = "Deployment execution results"
	}

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "deploy-results",
		Tags:     []string{"deployment", "execution"},
		Label:    "Deploy Results",
		Summary:  summary,
		Format:   "yaml",
		Priority: "medium",
		Content:  content.String(),
	}
}

func boolToStatus(ok bool) string {
	if ok {
		return "success"
	}
	return "failed"
}

// buildArchitectureGuideAttachment creates the architecture reference.
func buildArchitectureGuideAttachment() *domainpb.ContextAttachment {
	content := `Vrooli Deployment Architecture

Dependencies Flow:
1. Scenario declares deps in .vrooli/service.json (dependencies.resources, dependencies.scenarios)
2. scenario-dependency-analyzer builds the manifest
3. vps_deploy.go executes: Caddy setup → Resources → Dependent scenarios → Target scenario → Health checks

Common Failure Patterns:
- Resource missing from manifest: Check .vrooli/service.json or analyzer
- Resource in manifest but failed: Check ~/.vrooli/logs/ for start errors
- Resource started but not ready: Timing/health check issue
- VPS missing Vrooli: Mini-install incomplete or corrupted

Scope Determination:
- scenario-to-cloud issues: bundling, VPS setup, deployment steps, CLI/API wiring
- Target scenario issues: application runtime, configuration, dependencies`

	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "architecture-guide",
		Tags:     []string{"documentation", "reference"},
		Label:    "Architecture Guide",
		Summary:  "Vrooli deployment flow and common failure patterns",
		Format:   "markdown",
		Priority: "low",
		Content:  content,
	}
}

// buildFixPrompt constructs the prompt for applying fixes from an investigation.
func buildFixPrompt(
	originalInv *domain.Investigation,
	deployment *domain.Deployment,
	req ApplyFixesRequest,
) (string, error) {
	// Parse the manifest to get VPS details
	var m domain.CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &m); err != nil {
		return "", fmt.Errorf("failed to parse manifest: %w", err)
	}

	if m.Target.VPS == nil {
		return "", fmt.Errorf("deployment has no VPS target")
	}

	vps := m.Target.VPS
	sshPort := vps.Port
	if sshPort == 0 {
		sshPort = 22
	}
	sshUser := vps.User
	if sshUser == "" {
		sshUser = "root"
	}

	var sb strings.Builder

	sb.WriteString("# Fix Application Task\n\n")
	sb.WriteString("## Scope & Triage (read carefully)\n")
	sb.WriteString("Fixes may belong to one of two domains:\n")
	sb.WriteString("- **Deployment harness (scenario-to-cloud)**\n")
	sb.WriteString(fmt.Sprintf("- **Target scenario (%s)**\n", m.Scenario.ID))
	sb.WriteString("\n")
	sb.WriteString("Apply fixes ONLY in the domain supported by the investigation findings. Do not switch scopes without evidence.\n")
	sb.WriteString("\n")
	sb.WriteString("### Guardrails (mandatory)\n")
	sb.WriteString("- State the chosen domain explicitly before making changes.\n")
	sb.WriteString("- Cite at least two concrete evidence points from the investigation report.\n")
	sb.WriteString("- If evidence is mixed, stop and request clarification instead of guessing.\n\n")

	// Selected fixes section
	sb.WriteString("## Selected Fixes to Apply\n")
	if req.Immediate {
		sb.WriteString("- [x] **Immediate Fix** - Apply commands to fix the VPS right now\n")
	} else {
		sb.WriteString("- [ ] Immediate Fix - DO NOT APPLY\n")
	}
	if req.Permanent {
		sb.WriteString("- [x] **Permanent Fix** - Apply code/configuration changes\n")
	} else {
		sb.WriteString("- [ ] Permanent Fix - DO NOT APPLY\n")
	}
	if req.Prevention {
		sb.WriteString("- [x] **Prevention** - Implement monitoring/pipeline improvements\n")
	} else {
		sb.WriteString("- [ ] Prevention - DO NOT APPLY\n")
	}
	sb.WriteString("\n")

	// Instructions based on selected fixes
	sb.WriteString("## Instructions\n")
	sb.WriteString("Apply ONLY the selected fix types from the investigation results below.\n\n")

	if req.Immediate {
		sb.WriteString("### For Immediate Fixes\n")
		sb.WriteString("- SSH into the VPS and run the recommended commands\n")
		sb.WriteString("- Verify the fix worked by checking service status\n")
		sb.WriteString("- Report what commands you ran and their results\n\n")
	}

	if req.Permanent {
		sb.WriteString("### For Permanent Fixes\n")
		sb.WriteString("- Make the recommended code or configuration changes\n")
		sb.WriteString("- This may include editing `.vrooli/service.json`, fixing manifest generation, etc.\n")
		sb.WriteString("- Leave changes uncommitted for user review (do NOT commit)\n")
		sb.WriteString("- Report what files you modified and why\n\n")
	}

	if req.Prevention {
		sb.WriteString("### For Prevention\n")
		sb.WriteString("- Implement the recommended monitoring, alerts, or pipeline improvements\n")
		sb.WriteString("- This may involve adding health checks, deployment gates, or documentation\n")
		sb.WriteString("- Report what preventive measures you implemented\n\n")
	}

	// VPS connection info
	sb.WriteString("## VPS Connection\n")
	sb.WriteString("To apply fixes on the VPS, use SSH commands:\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("ssh -i %s -p %d %s@%s \"<command>\"\n", vps.KeyPath, sshPort, sshUser, vps.Host))
	sb.WriteString("```\n\n")

	sb.WriteString("## Deployment Configuration\n")
	sb.WriteString(fmt.Sprintf("- VPS Host: %s\n", vps.Host))
	sb.WriteString(fmt.Sprintf("- SSH User: %s\n", sshUser))
	sb.WriteString(fmt.Sprintf("- SSH Port: %d\n", sshPort))
	sb.WriteString(fmt.Sprintf("- SSH Key Path: %s\n", vps.KeyPath))
	if vps.Workdir != "" {
		sb.WriteString(fmt.Sprintf("- VPS Workdir: %s\n", vps.Workdir))
	}
	sb.WriteString(fmt.Sprintf("- Scenario: %s\n", m.Scenario.ID))
	sb.WriteString("\n")

	// User note if provided
	if req.Note != "" {
		sb.WriteString("## User Note\n")
		sb.WriteString(req.Note)
		sb.WriteString("\n\n")
	}

	// Include the original investigation results
	sb.WriteString("## Investigation Results\n")
	sb.WriteString("The following investigation report contains the findings and recommended fixes.\n")
	sb.WriteString("Apply ONLY the fixes that are selected above.\n\n")
	sb.WriteString("<investigation_results>\n")
	sb.WriteString(*originalInv.Findings)
	sb.WriteString("\n</investigation_results>\n\n")

	// Include deployment context
	sb.WriteString("<deployment_context>\n")
	manifestJSON, _ := json.MarshalIndent(m, "", "  ")
	sb.WriteString(string(manifestJSON))
	sb.WriteString("\n</deployment_context>\n\n")

	// Report format
	sb.WriteString("## Report Format\n")
	sb.WriteString("Please provide a structured report with:\n\n")
	sb.WriteString("### Scope Decision\n")
	sb.WriteString("- **Chosen domain**: scenario-to-cloud OR target scenario\n")
	sb.WriteString("- **Evidence**: at least two concrete evidence points\n")
	sb.WriteString("- **Confidence**: high/medium/low\n\n")
	sb.WriteString("### Fixes Applied\n")
	sb.WriteString("List each fix you applied, what you did, and the result.\n\n")
	sb.WriteString("### Verification\n")
	sb.WriteString("How you verified each fix worked (command outputs, status checks, etc.)\n\n")
	sb.WriteString("### Issues Encountered\n")
	sb.WriteString("Any problems you ran into while applying fixes, and how you resolved them.\n\n")
	sb.WriteString("### Next Steps\n")
	sb.WriteString("Any remaining manual steps the user needs to take (e.g., review uncommitted changes, restart deployment).\n")

	return sb.String(), nil
}
