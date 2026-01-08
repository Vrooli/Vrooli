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
	"error-info",
	"deployment-manifest",
	"vps-connection",
	"deployment-history",
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

	// Build the base prompt (task instructions - always included)
	var sb strings.Builder

	sb.WriteString("# Deployment Investigation\n\n")
	sb.WriteString("## Context\n")
	sb.WriteString(fmt.Sprintf("- Deployment ID: %s\n", deployment.ID))
	sb.WriteString(fmt.Sprintf("- Scenario: %s\n", m.Scenario.ID))
	sb.WriteString(fmt.Sprintf("- Deployment Status: %s\n", deployment.Status))

	if autoFix {
		sb.WriteString("- Operation Mode: **auto-fix** (you may attempt safe fixes)\n")
	} else {
		sb.WriteString("- Operation Mode: **report-only** (analyze only, do not make changes)\n")
	}
	sb.WriteString("\n")

	sb.WriteString("## Diagnostic Questions\n")
	sb.WriteString("As you investigate, answer these questions to identify the root cause:\n\n")
	sb.WriteString("1. **Is the failing dependency in the manifest?** Check the deployment-manifest context attachment\n")
	sb.WriteString("2. **If yes, did the resource/scenario start step succeed?** Check `~/.vrooli/logs/` for resource and scenario start logs\n")
	sb.WriteString("3. **If no, is it declared in the scenario's service.json?** Check `<workdir>/scenarios/<scenario>/.vrooli/service.json`\n")
	sb.WriteString("4. **Is the Vrooli CLI working?** Run `vrooli --version` and `vrooli resource list`\n")
	sb.WriteString("5. **Is this a configuration issue or transient failure?** Would a simple restart fix it, or is there a deeper problem?\n\n")

	sb.WriteString("## Your Task\n")
	sb.WriteString("1. SSH into the VPS to investigate the deployment failure (see vps-connection context)\n")
	sb.WriteString("2. Check relevant logs:\n")
	sb.WriteString("   - Vrooli logs: `~/.vrooli/logs/` (resource and scenario logs)\n")
	sb.WriteString("   - `journalctl -u <service>` for systemd services\n")
	sb.WriteString("   - Docker logs: `docker logs <container>`\n")
	sb.WriteString("   - Application logs in the workdir\n")
	sb.WriteString("3. Verify service status and port bindings:\n")
	sb.WriteString("   - `systemctl status <service>`\n")
	sb.WriteString("   - `docker ps -a`\n")
	sb.WriteString("   - `ss -tlnp | grep <port>`\n")
	sb.WriteString("4. Check system resources:\n")
	sb.WriteString("   - `df -h` for disk space\n")
	sb.WriteString("   - `free -m` for memory\n")
	sb.WriteString("   - `top -bn1 | head -20` for CPU usage\n")
	sb.WriteString("5. Answer the diagnostic questions above\n")
	sb.WriteString("6. Identify the root cause of the failure\n")

	if autoFix {
		sb.WriteString("7. If safe, attempt to fix the issue (restart services, clear disk, etc.)\n")
	}
	sb.WriteString("\n")

	sb.WriteString("## Report Format\n")
	sb.WriteString("Please provide a structured report with:\n\n")
	sb.WriteString("### Root Cause\n")
	sb.WriteString("What caused the deployment to fail. Be specific - distinguish between symptoms (e.g., \"postgres not running\") and actual causes (e.g., \"postgres not in manifest\" or \"postgres start command failed\").\n\n")
	sb.WriteString("### Evidence\n")
	sb.WriteString("Logs, error messages, and system state that support your conclusion. Include:\n")
	sb.WriteString("- Answers to the diagnostic questions above\n")
	sb.WriteString("- Relevant log snippets\n")
	sb.WriteString("- Command outputs that confirm the issue\n\n")
	sb.WriteString("### Impact\n")
	sb.WriteString("What is broken or not working as a result.\n\n")
	sb.WriteString("### Immediate Fix\n")
	sb.WriteString("Commands to run RIGHT NOW on this VPS to restore service. These are hotfixes to unblock the current deployment.")
	if autoFix {
		sb.WriteString(" (If you already applied fixes, list what you did.)")
	}
	sb.WriteString("\n\n")
	sb.WriteString("### Permanent Fix\n")
	sb.WriteString("What needs to change in code, configuration, or the deployment manifest so this issue does NOT occur on fresh VPS deployments. This might include:\n")
	sb.WriteString("- Adding missing dependencies to `.vrooli/service.json`\n")
	sb.WriteString("- Fixing the manifest generation process\n")
	sb.WriteString("- Adding health checks or startup delays\n")
	sb.WriteString("- Fixing the VPS setup/installation process\n\n")
	sb.WriteString("### Prevention\n")
	sb.WriteString("Recommendations for monitoring, alerts, or deployment pipeline improvements that would catch this issue earlier or prevent it entirely.\n")

	basePrompt := sb.String()

	// Build context attachments based on selection
	var attachments []*domainpb.ContextAttachment

	// Error Information
	if containsContext(includeContexts, "error-info") {
		var errorContent strings.Builder
		if deployment.ErrorStep != nil {
			errorContent.WriteString(fmt.Sprintf("Failed Step: %s\n", *deployment.ErrorStep))
		}
		if deployment.ErrorMessage != nil {
			errorContent.WriteString(fmt.Sprintf("Error Message:\n%s", *deployment.ErrorMessage))
		}
		if errorContent.Len() > 0 {
			attachments = append(attachments, &domainpb.ContextAttachment{
				Type:    "note",
				Key:     "error-info",
				Tags:    []string{"error", "diagnosis"},
				Label:   "Error Information",
				Content: errorContent.String(),
			})
		}
	}

	// Deployment Manifest
	if containsContext(includeContexts, "deployment-manifest") {
		manifestJSON, _ := json.MarshalIndent(m, "", "  ")
		attachments = append(attachments, &domainpb.ContextAttachment{
			Type:    "note",
			Key:     "deployment-manifest",
			Tags:    []string{"config", "dependencies"},
			Label:   "Deployment Manifest",
			Content: string(manifestJSON),
		})
	}

	// VPS Connection Details
	if containsContext(includeContexts, "vps-connection") {
		var vpsContent strings.Builder
		vpsContent.WriteString("SSH Command:\n")
		vpsContent.WriteString(fmt.Sprintf("ssh -i %s -p %d %s@%s \"<command>\"\n\n", vps.KeyPath, sshPort, sshUser, vps.Host))
		vpsContent.WriteString(fmt.Sprintf("Host: %s\n", vps.Host))
		vpsContent.WriteString(fmt.Sprintf("User: %s\n", sshUser))
		vpsContent.WriteString(fmt.Sprintf("Port: %d\n", sshPort))
		vpsContent.WriteString(fmt.Sprintf("Key Path: %s\n", vps.KeyPath))
		if vps.Workdir != "" {
			vpsContent.WriteString(fmt.Sprintf("Workdir: %s\n", vps.Workdir))
		}
		if m.Edge.Domain != "" {
			vpsContent.WriteString(fmt.Sprintf("Domain: %s\n", m.Edge.Domain))
		}
		if len(m.Ports) > 0 {
			vpsContent.WriteString("\nExpected Ports:\n")
			for name, port := range m.Ports {
				vpsContent.WriteString(fmt.Sprintf("- %s: %d\n", name, port))
			}
		}
		attachments = append(attachments, &domainpb.ContextAttachment{
			Type:    "note",
			Key:     "vps-connection",
			Tags:    []string{"ssh", "access"},
			Label:   "VPS Connection Details",
			Content: vpsContent.String(),
		})
	}

	// Deployment History
	if containsContext(includeContexts, "deployment-history") && deployment.DeploymentHistory.Valid {
		attachments = append(attachments, &domainpb.ContextAttachment{
			Type:    "note",
			Key:     "deployment-history",
			Tags:    []string{"timeline", "events"},
			Label:   "Deployment History",
			Content: string(deployment.DeploymentHistory.Data),
		})
	}

	// Preflight Results
	if containsContext(includeContexts, "preflight-results") && deployment.PreflightResult.Valid {
		attachments = append(attachments, &domainpb.ContextAttachment{
			Type:    "note",
			Key:     "preflight-results",
			Tags:    []string{"preflight", "checks"},
			Label:   "Preflight Results",
			Content: string(deployment.PreflightResult.Data),
		})
	}

	// Setup Results
	if containsContext(includeContexts, "setup-results") && deployment.SetupResult.Valid {
		attachments = append(attachments, &domainpb.ContextAttachment{
			Type:    "note",
			Key:     "setup-results",
			Tags:    []string{"setup", "installation"},
			Label:   "Setup Results",
			Content: string(deployment.SetupResult.Data),
		})
	}

	// Deploy Results
	if containsContext(includeContexts, "deploy-results") && deployment.DeployResult.Valid {
		attachments = append(attachments, &domainpb.ContextAttachment{
			Type:    "note",
			Key:     "deploy-results",
			Tags:    []string{"deployment", "execution"},
			Label:   "Deploy Results",
			Content: string(deployment.DeployResult.Data),
		})
	}

	// Architecture Guide
	if containsContext(includeContexts, "architecture-guide") {
		var archContent strings.Builder
		archContent.WriteString("## Vrooli Deployment Architecture\n")
		archContent.WriteString("Understanding how Vrooli deployments work will help you diagnose the root cause:\n\n")
		archContent.WriteString("### How Dependencies Work\n")
		archContent.WriteString("1. Each scenario declares its dependencies in `.vrooli/service.json` under `dependencies.resources` and `dependencies.scenarios`\n")
		archContent.WriteString("2. The `scenario-dependency-analyzer` scans the scenario and its dependencies to build the manifest\n")
		archContent.WriteString("3. The deployment pipeline (`vps_deploy.go`) executes these steps in order:\n")
		archContent.WriteString("   - Install and configure Caddy (reverse proxy)\n")
		archContent.WriteString("   - Start each resource in `manifest.dependencies.resources` via `vrooli resource start <name>`\n")
		archContent.WriteString("   - Start each dependent scenario in `manifest.dependencies.scenarios`\n")
		archContent.WriteString("   - Start the target scenario with port assignments\n")
		archContent.WriteString("   - Run health checks\n\n")
		archContent.WriteString("### Common Failure Patterns\n")
		archContent.WriteString("- **Resource missing from manifest**: The scenario's `.vrooli/service.json` may not declare the resource, or the analyzer didn't pick it up\n")
		archContent.WriteString("- **Resource in manifest but failed to start**: The `vrooli resource start` command failed on the VPS (check logs in `~/.vrooli/logs/`)\n")
		archContent.WriteString("- **Resource started but not ready**: The scenario started before the resource was fully initialized (timing/health check issue)\n")
		archContent.WriteString("- **VPS missing Vrooli installation**: The mini-Vrooli install may be incomplete or corrupted\n")

		attachments = append(attachments, &domainpb.ContextAttachment{
			Type:    "note",
			Key:     "architecture-guide",
			Tags:    []string{"documentation", "reference"},
			Label:   "Vrooli Architecture Guide",
			Content: archContent.String(),
		})
	}

	// User Note
	if note != "" {
		attachments = append(attachments, &domainpb.ContextAttachment{
			Type:    "note",
			Key:     "user-note",
			Tags:    []string{"user", "context"},
			Label:   "User Note",
			Content: note,
		})
	}

	return basePrompt, attachments, nil
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
