package scan

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"tidiness-manager/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	maturityreport "github.com/vrooli/maturity-go/report"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tidiness-manager/v1/validation"
)

const cliName = "tidiness-manager"

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Scanning",
		Commands: []cliapp.Command{
			{
				Name:        "scan",
				NeedsAPI:    true,
				Description: "Run tidiness, light, or smart scans",
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scan")
	scanType := fs.String("type", "tidiness", "Scan type: tidiness, light, or smart")
	timeout := fs.Int("timeout", 120, "Light scan timeout in seconds")
	var files cliutil.StringList
	fs.Var(&files, "file", "File to include in smart scan (repeatable)")
	forceRescan := fs.Bool("force-rescan", false, "Re-analyze files already seen in this smart scan session")
	campaignID := fs.Int("campaign-id", 0, "Attach smart scan results to a campaign")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	target := "."
	if fs.NArg() > 0 {
		target = fs.Arg(0)
	}

	switch strings.ToLower(strings.TrimSpace(*scanType)) {
	case "tidiness", "maintainability":
		return runTidiness(core, target, *timeout, *jsonOutput)
	case "light":
		return runLight(core, target, *timeout, *jsonOutput)
	case "smart":
		return runSmart(core, target, files.Values(), *forceRescan, *campaignID, *jsonOutput)
	default:
		return fmt.Errorf("unsupported scan type %q", *scanType)
	}
}

func runTidiness(core *cliapp.ScenarioApp, target string, timeout int, jsonOutput bool) error {
	scenario := support.ScenarioName(target)
	if strings.TrimSpace(scenario) == "" {
		return fmt.Errorf("scenario name is required for tidiness scans")
	}
	if timeout <= 0 {
		timeout = 120
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, time.Duration(timeout)*time.Second)
	client := scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)
	resp, err := client.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: scenario,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate tidiness for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg
	if jsonOutput {
		return cliapp.PrintProtoJSON(os.Stdout, msg)
	}
	var native validationv1.TidinessScanResponse
	if detail := msg.GetNativeDetail(); detail != nil {
		if err := detail.UnmarshalTo(&native); err != nil {
			return fmt.Errorf("unpack tidiness native detail: %w", err)
		}
	}

	findings := native.GetFindings()
	sort.SliceStable(findings, func(i, j int) bool {
		return support.SeverityRank(findings[i].GetSeverity()) < support.SeverityRank(findings[j].GetSeverity())
	})

	assessment := msg.GetAssessment()
	summary := native.GetSummary()
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", msg.GetScenario()),
			fmt.Sprintf("Status: %s", statusLabel(msg.GetStatus())),
			fmt.Sprintf("Findings: %d", summary.GetTotalFindings()),
			fmt.Sprintf("Long files: %d", summary.GetLongFiles()),
			fmt.Sprintf("Complexity: %d", summary.GetComplexity()),
			fmt.Sprintf("Duplication: %d", summary.GetDuplication()),
			fmt.Sprintf("Tech debt: %d", summary.GetTechDebt()),
		},
		ResultsHeading: "Tidiness Findings",
		Results:        tidinessRows(findings),
		RetrievalHints: []string{
			fmt.Sprintf("%s scan %s --type tidiness", cliName, scenario),
			fmt.Sprintf("%s recommend-refactors %s --limit 10", cliName, scenario),
		},
	}
	if assessment != nil {
		maturity := maturityreport.BuildMaturityListReport(assessment)
		report.Summary = append(report.Summary, maturity.Summary...)
		report.RetrievalHints = append(report.RetrievalHints, maturity.RetrievalHints...)
	}

	if err := cliapp.RenderListReport(os.Stdout, report); err != nil {
		return err
	}
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		return fmt.Errorf("scenario %s did not pass tidiness validation (%d error finding(s))", msg.GetScenario(), severityCount(assessment, "SEVERITY_ERROR"))
	}
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR {
		return fmt.Errorf("scenario %s tidiness validation errored", msg.GetScenario())
	}
	return nil
}

func runLight(core *cliapp.ScenarioApp, target string, timeout int, jsonOutput bool) error {
	path, err := support.ScenarioPath(target)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/scan/light", nil, map[string]interface{}{
		"scenario_path": path,
		"timeout_sec":   timeout,
	})
	if err != nil {
		return err
	}

	var result support.LightScanResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Light scan completed for %s", result.Scenario),
			fmt.Sprintf("Duration: %dms", result.DurationMS),
			fmt.Sprintf("Files scanned: %d", result.TotalFiles),
			fmt.Sprintf("Total lines: %d", result.TotalLines),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Static Analysis", Items: []string{
				fmt.Sprintf("Lint issues: %d", result.LintIssuesCount),
				fmt.Sprintf("Type issues: %d", result.TypeIssuesCount),
				commandSummary("make lint", result.LintOutput),
				commandSummary("make type", result.TypeOutput),
			}},
			{Heading: "File Size", Items: longFileRows(result.LongFiles, result.LongFilesCount)},
		},
		NextSteps: []string{
			fmt.Sprintf("%s issues %s --limit 20", cliName, result.Scenario),
			fmt.Sprintf("%s recommend-refactors %s --limit 5", cliName, result.Scenario),
		},
	}

	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runSmart(core *cliapp.ScenarioApp, target string, files []string, forceRescan bool, campaignID int, jsonOutput bool) error {
	scenario := support.ScenarioName(target)
	if strings.TrimSpace(scenario) == "" {
		return fmt.Errorf("scenario name is required for smart scans")
	}
	files = support.NormalizePathList(files)
	if len(files) == 0 {
		return fmt.Errorf("at least one --file is required for smart scans")
	}

	payload := support.SmartScanRequest{
		Scenario:    scenario,
		Files:       files,
		ForceRescan: forceRescan,
	}
	if campaignID > 0 {
		payload.CampaignID = &campaignID
	}

	body, err := core.Request("POST", "/scan/smart", nil, payload)
	if err != nil {
		return err
	}

	var result support.SmartScanResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Smart scan completed for %s", scenario),
			fmt.Sprintf("Session: %s", result.SessionID),
			fmt.Sprintf("Files analyzed: %d", result.FilesAnalyzed),
			fmt.Sprintf("Issues found: %d", result.IssuesFound),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Batches", Items: batchRows(result.BatchResults)},
			{Heading: "Errors", Items: result.Errors},
		},
		NextSteps: []string{
			fmt.Sprintf("%s issues %s --category ai --limit 20", cliName, scenario),
			fmt.Sprintf("%s recommend-refactors %s --limit 10", cliName, scenario),
		},
	}

	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func commandSummary(name string, run *support.CommandRun) string {
	if run == nil {
		return fmt.Sprintf("%s: not available", name)
	}
	if run.Skipped {
		return fmt.Sprintf("%s: skipped (%s)", name, run.SkipReason)
	}
	status := "failed"
	if run.Success {
		status = "passed"
	}
	return fmt.Sprintf("%s: %s (exit %d, %dms)", name, status, run.ExitCode, run.DurationMS)
}

func longFileRows(files []support.LongFile, count int) []string {
	if len(files) == 0 {
		return []string{fmt.Sprintf("Long files: %d", count)}
	}
	rows := []string{fmt.Sprintf("Long files: %d", count)}
	limit := len(files)
	if limit > 5 {
		limit = 5
	}
	for _, file := range files[:limit] {
		rows = append(rows, fmt.Sprintf("%s (%d lines, threshold %d)", file.Path, file.Lines, file.Threshold))
	}
	return rows
}

func batchRows(batches []support.BatchResult) []string {
	if len(batches) == 0 {
		return []string{"No smart-scan batches executed"}
	}
	rows := make([]string, 0, len(batches))
	sort.SliceStable(batches, func(i, j int) bool { return batches[i].BatchID < batches[j].BatchID })
	for _, batch := range batches {
		line := fmt.Sprintf("Batch %d: %d files, %d issues", batch.BatchID, len(batch.Files), len(batch.Issues))
		if strings.TrimSpace(batch.Error) != "" {
			line += " | error: " + batch.Error
		}
		if strings.TrimSpace(batch.Duration) != "" {
			line += " | duration: " + batch.Duration
		}
		rows = append(rows, line)
	}
	return rows
}

func tidinessRows(findings []*validationv1.TidinessFinding) []string {
	rows := make([]string, 0, len(findings))
	for _, finding := range findings {
		loc := finding.GetFilePath()
		if finding.GetLineNumber() > 0 {
			loc = fmt.Sprintf("%s:%d", loc, finding.GetLineNumber())
		}
		line := fmt.Sprintf("[%s] %s", strings.ToUpper(finding.GetSeverity()), finding.GetTitle())
		if strings.TrimSpace(loc) != "" {
			line += " -> " + loc
		}
		rows = append(rows, line)
	}
	if len(rows) == 0 {
		rows = append(rows, "No tidiness findings detected")
	}
	return rows
}

func statusLabel(status scenariovalidationv1.ValidationStatus) string {
	switch status {
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED:
		return "passed"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED:
		return "failed"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED:
		return "degraded"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR:
		return "error"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_SKIPPED:
		return "skipped"
	default:
		return "unspecified"
	}
}

func severityCount(a interface{ GetFindingsBySeverity() map[string]int32 }, severity string) int {
	if a == nil {
		return 0
	}
	total := 0
	want := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(severity)), "FINDING_")
	for key, count := range a.GetFindingsBySeverity() {
		normalized := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(key)), "FINDING_")
		if normalized == want {
			total += int(count)
		}
	}
	return total
}
