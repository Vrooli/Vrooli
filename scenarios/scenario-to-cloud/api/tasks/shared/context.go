package shared

import (
	"encoding/json"
	"fmt"
	"strings"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

	"scenario-to-cloud/domain"
)

// DefaultIncludeContexts lists context items included by default if none specified.
var DefaultIncludeContexts = []string{
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

// ContainsContext checks if a context key is in the include list.
func ContainsContext(includeContexts []string, key string) bool {
	for _, c := range includeContexts {
		if c == key {
			return true
		}
	}
	return false
}

// ParseManifest extracts the CloudManifest from a deployment.
func ParseManifest(deployment *domain.Deployment) (*domain.CloudManifest, error) {
	var m domain.CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	if m.Target.VPS == nil {
		return nil, fmt.Errorf("deployment has no VPS target")
	}
	return &m, nil
}

// GetSSHDetails extracts SSH connection details from a VPS target.
func GetSSHDetails(vps *domain.ManifestVPS) (user string, port int) {
	port = vps.Port
	if port == 0 {
		port = 22
	}
	user = vps.User
	if user == "" {
		user = "root"
	}
	return user, port
}

// ExtractErrorSummary extracts a one-line summary of the error.
func ExtractErrorSummary(deployment *domain.Deployment) string {
	if deployment.ErrorMessage == nil || *deployment.ErrorMessage == "" {
		return ""
	}

	msg := *deployment.ErrorMessage

	// Try to extract the most relevant part
	// Common patterns: "Caddy ACME validation...", "Cannot negotiate ALPN..."
	if idx := strings.Index(msg, "Cannot negotiate"); idx != -1 {
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

// SafeDeref safely dereferences a string pointer, returning a default if nil.
func SafeDeref(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}

// BuildTaskMetadataAttachment creates the task metadata context.
func BuildTaskMetadataAttachment(deployment *domain.Deployment, m *domain.CloudManifest, mode string) *domainpb.ContextAttachment {
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
		Summary:  fmt.Sprintf("Deployment %s failed at %s", m.Scenario.ID, SafeDeref(deployment.ErrorStep, "unknown step")),
		Format:   "yaml",
		Priority: "high",
		Content:  content.String(),
	}
}

// BuildErrorInfoAttachment creates the error information context with summarization.
func BuildErrorInfoAttachment(deployment *domain.Deployment) *domainpb.ContextAttachment {
	var content strings.Builder
	var summary string

	if deployment.ErrorStep != nil {
		content.WriteString(fmt.Sprintf("Failed Step: %s\n\n", *deployment.ErrorStep))
	}

	if deployment.ErrorMessage != nil && *deployment.ErrorMessage != "" {
		msg := *deployment.ErrorMessage
		summary = ExtractErrorSummary(deployment)

		// Check if it's a structured error with logs
		if strings.Contains(msg, " | ") {
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

// BuildSafetyRulesAttachment creates the safety rules context.
func BuildSafetyRulesAttachment() *domainpb.ContextAttachment {
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

// BuildDiagnosticChecklistAttachment creates the diagnostic checklist.
func BuildDiagnosticChecklistAttachment() *domainpb.ContextAttachment {
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

// BuildOutputFormatAttachment creates the output format specification for investigations.
func BuildOutputFormatAttachment(autoFix bool) *domainpb.ContextAttachment {
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

// BuildVPSConnectionAttachment creates the VPS connection details.
func BuildVPSConnectionAttachment(vps *domain.ManifestVPS, sshUser string, sshPort int, m *domain.CloudManifest) *domainpb.ContextAttachment {
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

// BuildManifestAttachment creates the deployment manifest context with summary.
func BuildManifestAttachment(m *domain.CloudManifest) *domainpb.ContextAttachment {
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

// BuildDeployResultsAttachment creates the deploy results context with summarization.
func BuildDeployResultsAttachment(deployment *domain.Deployment) *domainpb.ContextAttachment {
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

	var content strings.Builder
	content.WriteString(fmt.Sprintf("status: %s\n", boolToStatus(result.OK)))
	if result.FailedStep != "" {
		content.WriteString(fmt.Sprintf("failed_step: %s\n", result.FailedStep))
	}
	if result.Timestamp != "" {
		content.WriteString(fmt.Sprintf("timestamp: %s\n", result.Timestamp))
	}
	content.WriteString("\n")

	if result.Error != "" {
		content.WriteString("error_summary:\n")
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

// BuildArchitectureGuideAttachment creates the architecture reference.
func BuildArchitectureGuideAttachment() *domainpb.ContextAttachment {
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

// BuildDeploymentHistoryAttachment creates a context attachment for deployment history.
func BuildDeploymentHistoryAttachment(deployment *domain.Deployment) *domainpb.ContextAttachment {
	if !deployment.DeploymentHistory.Valid {
		return nil
	}
	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "deployment-history",
		Tags:     []string{"timeline", "events"},
		Label:    "Deployment History",
		Summary:  "Timeline of deployment events",
		Format:   "json",
		Priority: "low",
		Content:  string(deployment.DeploymentHistory.Data),
	}
}

// BuildPreflightResultsAttachment creates a context attachment for preflight results.
func BuildPreflightResultsAttachment(deployment *domain.Deployment) *domainpb.ContextAttachment {
	if !deployment.PreflightResult.Valid {
		return nil
	}
	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "preflight-results",
		Tags:     []string{"preflight", "checks"},
		Label:    "Preflight Results",
		Summary:  "Pre-deployment validation checks",
		Format:   "json",
		Priority: "low",
		Content:  string(deployment.PreflightResult.Data),
	}
}

// BuildSetupResultsAttachment creates a context attachment for setup results.
func BuildSetupResultsAttachment(deployment *domain.Deployment) *domainpb.ContextAttachment {
	if !deployment.SetupResult.Valid {
		return nil
	}
	return &domainpb.ContextAttachment{
		Type:     "note",
		Key:      "setup-results",
		Tags:     []string{"setup", "installation"},
		Label:    "Setup Results",
		Summary:  "VPS setup and installation outputs",
		Format:   "json",
		Priority: "low",
		Content:  string(deployment.SetupResult.Data),
	}
}
