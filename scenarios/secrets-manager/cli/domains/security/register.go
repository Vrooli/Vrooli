package security

import (
	"fmt"
	"net/url"
	"strings"

	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "security",
		Description: "Vulnerability scanning, compliance, and remediation workflows",
		Subcommands: []cliapp.Command{
			{Name: "vulnerabilities", Aliases: []string{"list"}, NeedsAPI: true, Description: "List vulnerabilities with optional filters", Run: func(args []string) error { return runVulnerabilities(core, args) }},
			{Name: "scan", NeedsAPI: true, Description: "Run a filesystem security scan and summarize the results", Run: func(args []string) error { return runScan(core, args) }},
			{Name: "compliance", NeedsAPI: true, Description: "Show aggregate compliance and remediation posture", Run: func(args []string) error { return runCompliance(core, args) }},
			{Name: "set-status", NeedsAPI: true, Description: "Update the tracked status for one vulnerability", Run: func(args []string) error { return runSetStatus(core, args) }},
			{Name: "fix", NeedsAPI: true, Description: "Trigger the vulnerability fixer workflow for selected findings", Run: func(args []string) error { return runFix(core, args) }},
		},
	}
}

func runVulnerabilities(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("security vulnerabilities")
	component := fs.String("component", "", "Filter by component name")
	componentType := fs.String("component-type", "", "Filter by component type (scenario|resource)")
	severity := fs.String("severity", "", "Filter by severity")
	quick := fs.Bool("quick", false, "Use quick scan mode")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := makeQueryMap(
		"component", *component,
		"component_type", *componentType,
		"severity", *severity,
	)
	if *quick {
		query.Set("quick", "true")
	}

	var resp support.VulnerabilityResponse
	if err := support.GetJSON(core, "/security/vulnerabilities", query, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Vulnerabilities))
	for _, vuln := range resp.Vulnerabilities {
		results = append(results, fmt.Sprintf("%s | %s | %s/%s | %s:%d | status=%s | auto-fix=%t",
			vuln.ID, vuln.Severity, vuln.ComponentType, vuln.ComponentName, vuln.FilePath, vuln.LineNumber, support.Fallback(vuln.Status, "open"), vuln.CanAutoFix))
		results = append(results, "  "+support.JoinNonEmpty(vuln.Title, vuln.Type))
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Vulnerabilities: %d", resp.TotalCount),
			fmt.Sprintf("Risk score: %d", resp.ScanMetadata.RiskScore),
			fmt.Sprintf("Scan ID: %s", resp.ScanMetadata.ScanID),
		},
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " security scan", support.CLIName + " security set-status <id> --status in_progress"},
	}
	return support.PrintList(*jsonOutput, resp, report)
}

func runScan(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("security scan")
	component := fs.String("component", "", "Filter by component name")
	componentType := fs.String("component-type", "", "Filter by component type (scenario|resource)")
	severity := fs.String("severity", "", "Filter vulnerabilities by severity")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := makeQueryMap(
		"component", *component,
		"component_type", *componentType,
		"severity", *severity,
	)

	var resp support.SecurityScanResponse
	if err := support.GetJSON(core, "/security/scan", query, &resp); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Scan ID: %s", resp.ScanID),
			fmt.Sprintf("Risk score: %d", resp.RiskScore),
			fmt.Sprintf("Vulnerabilities: %d", len(resp.Vulnerabilities)),
			fmt.Sprintf("Files scanned: %d", resp.ScanMetrics.FilesScanned),
			fmt.Sprintf("Scan complete: %s", support.BoolLabel(resp.ScanMetrics.ScanComplete, "yes", "no")),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Scope",
				Items: []string{
					fmt.Sprintf("Component filter: %s", support.Fallback(resp.ComponentFilter, "all")),
					fmt.Sprintf("Component type: %s", support.Fallback(resp.ComponentType, "all")),
					fmt.Sprintf("Resources scanned: %d", resp.ComponentsSummary.ResourcesScanned),
					fmt.Sprintf("Scenarios scanned: %d", resp.ComponentsSummary.ScenariosScanned),
				},
			},
			{
				Heading: "Performance",
				Items: []string{
					fmt.Sprintf("Total scan time: %dms", resp.ScanMetrics.TotalScanTimeMS),
					fmt.Sprintf("Files skipped: %d", resp.ScanMetrics.FilesSkipped),
					fmt.Sprintf("Timeout occurred: %s", support.BoolLabel(resp.ScanMetrics.TimeoutOccurred, "yes", "no")),
				},
			},
		},
		NextSteps: []string{
			support.CLIName + " security vulnerabilities",
			support.CLIName + " security compliance",
			support.CLIName + " security fix --severity critical",
		},
	}
	if len(resp.ScanMetrics.ScanErrors) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Scan Errors", Items: resp.ScanMetrics.ScanErrors})
	}
	return support.PrintOperational(*jsonOutput, resp, report)
}

func runCompliance(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("security compliance")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var resp support.ComplianceResponse
	if err := support.GetJSON(core, "/security/compliance", nil, &resp); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Overall score: %d", resp.OverallScore),
			fmt.Sprintf("Credential coverage health: %d", resp.CredentialCoverageHealth),
			fmt.Sprintf("Security score: %d", resp.RemediationProgress.SecurityScore),
			fmt.Sprintf("Total vulnerabilities: %d", resp.TotalVulnerabilities),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Coverage",
				Items: []string{
					fmt.Sprintf("Configured resources: %d/%d", resp.ConfiguredResources, resp.TotalResources),
					fmt.Sprintf("Configured components: %d", resp.RemediationProgress.ConfiguredComponents),
				},
			},
			{
				Heading: "Severity Mix",
				Items: []string{
					fmt.Sprintf("Critical: %d", resp.RemediationProgress.CriticalIssues),
					fmt.Sprintf("High: %d", resp.RemediationProgress.HighIssues),
					fmt.Sprintf("Medium: %d", resp.RemediationProgress.MediumIssues),
					fmt.Sprintf("Low: %d", resp.RemediationProgress.LowIssues),
				},
			},
		},
		NextSteps: []string{
			support.CLIName + " security vulnerabilities --severity critical",
			support.CLIName + " credentials status",
			support.CLIName + " deployment readiness --scenario <scenario>",
		},
	}
	return support.PrintOperational(*jsonOutput, resp, report)
}

func runSetStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("security set-status")
	status := fs.String("status", "", "New status: open|in_progress|resolved|accepted|regressed")
	assignedTo := fs.String("assigned-to", "", "Optional assignee")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: security set-status <vulnerability-id> --status <status>")
	}
	if strings.TrimSpace(*status) == "" {
		return fmt.Errorf("--status is required")
	}

	id := fs.Arg(0)
	payload := map[string]any{"status": *status}
	if strings.TrimSpace(*assignedTo) != "" {
		payload["assigned_to"] = *assignedTo
	}

	var resp support.VulnerabilityStatusResponse
	if err := support.RequestJSON(core, "POST", "/security/vulnerabilities/"+id+"/status", nil, payload, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Vulnerability status updated", "ID: " + resp.ID},
		Changes:     []string{"Status: " + resp.Status, "Assigned to: " + support.Fallback(*assignedTo, "unchanged")},
		NextCommand: []string{support.CLIName + " security vulnerabilities", support.CLIName + " security vulnerabilities --severity critical"},
	}
	return support.PrintMutation(*jsonOutput, resp, report)
}

func runFix(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("security fix")
	var ids cliutil.StringList
	fs.Var(&ids, "id", "Vulnerability ID to include; repeatable")
	component := fs.String("component", "", "Select vulnerabilities by component")
	componentType := fs.String("component-type", "", "Select vulnerabilities by component type")
	severity := fs.String("severity", "", "Select vulnerabilities by severity")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	selected, err := selectVulnerabilities(core, ids.Values(), *component, *componentType, *severity)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		return fmt.Errorf("no vulnerabilities selected")
	}

	payload := map[string]any{"vulnerabilities": selected}
	var resp support.VulnerabilityFixResponse
	if err := support.RequestJSON(core, "POST", "/security/vulnerabilities/fix", nil, payload, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			"Vulnerability fix workflow accepted",
			"Fix request ID: " + resp.FixRequestID,
		},
		Changes: []string{
			fmt.Sprintf("Selected vulnerabilities: %d", resp.Vulnerabilities),
			"Status: " + resp.Status,
		},
		NextCommand: []string{support.CLIName + " security vulnerabilities", "Inspect secrets-manager API logs for fixer progress"},
	}
	if resp.Message != "" {
		report.Result = append(report.Result, "Message: "+resp.Message)
	}
	return support.PrintMutation(*jsonOutput, resp, report)
}

func selectVulnerabilities(core *cliapp.ScenarioApp, ids []string, component, componentType, severity string) ([]support.SecurityVulnerability, error) {
	query := makeQueryMap("component", component, "component_type", componentType, "severity", severity)
	var resp support.VulnerabilityResponse
	if err := support.GetJSON(core, "/security/vulnerabilities", query, &resp); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return resp.Vulnerabilities, nil
	}

	set := map[string]struct{}{}
	for _, id := range ids {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}
	selected := make([]support.SecurityVulnerability, 0, len(set))
	for _, item := range resp.Vulnerabilities {
		if _, ok := set[item.ID]; ok {
			selected = append(selected, item)
		}
	}
	return selected, nil
}

func makeQueryMap(parts ...string) url.Values {
	values := url.Values{}
	for i := 0; i+1 < len(parts); i += 2 {
		key := strings.TrimSpace(parts[i])
		value := strings.TrimSpace(parts[i+1])
		if key == "" || value == "" {
			continue
		}
		values.Set(key, value)
	}
	if len(values) == 0 {
		return nil
	}
	return values
}
