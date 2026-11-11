package scan

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
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
	budgetAudit := fs.Bool("budget-audit", false, "Print declared and observed seam and metric budgets")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	target := "."
	if fs.NArg() > 0 {
		target = fs.Arg(0)
	}
	if *budgetAudit {
		return runBudgetAudit(core, target, *timeout, *jsonOutput)
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

type budgetAuditReport struct {
	Scenario string              `json:"scenario"`
	Seams    []seamBudgetAudit   `json:"seams"`
	Metrics  []metricBudgetAudit `json:"metrics"`
	Blocking bool                `json:"blocking"`
}

type seamBudgetAudit struct {
	ID               string   `json:"id"`
	DeclaredBudget   int      `json:"declared_budget"`
	DeclaredBaseline *int     `json:"declared_baseline"`
	Observed         int      `json:"observed"`
	Reserve          int      `json:"reserve"`
	ReserveReason    string   `json:"reserve_reason,omitempty"`
	Verdicts         []string `json:"verdicts"`
}

type metricBudgetAudit struct {
	Name             string   `json:"name"`
	DeclaredBudget   *int     `json:"declared_budget"`
	DeclaredBaseline *int     `json:"declared_baseline"`
	Observed         int      `json:"observed"`
	Reserve          int      `json:"reserve"`
	Verdicts         []string `json:"verdicts"`
}

func runBudgetAudit(core *cliapp.ScenarioApp, target string, timeout int, jsonOutput bool) error {
	scenarioPath, err := support.ScenarioPath(target)
	if err != nil {
		return err
	}
	if timeout <= 0 {
		timeout = 600
	}
	originalHTTPClient, originalAPIClient := core.HTTPClient, core.APIClient
	core.HTTPClient = core.HTTPClient.CloneWithTimeout(time.Duration(timeout) * time.Second)
	core.APIClient = nil
	defer func() {
		core.HTTPClient, core.APIClient = originalHTTPClient, originalAPIClient
	}()
	body, err := core.Request("POST", "/budget-audit", nil, map[string]any{
		"scenario":      support.ScenarioName(target),
		"scenario_path": scenarioPath,
		"timeout_sec":   timeout,
	})
	if err != nil {
		return err
	}
	var report budgetAuditReport
	if err := support.Decode(body, &report); err != nil {
		return err
	}
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return err
		}
	} else {
		rows := make([]string, 0, len(report.Seams)+len(report.Metrics))
		for _, seam := range report.Seams {
			rows = append(rows, fmt.Sprintf("seam %-40s budget=%d observed=%d reserve=%d verdict=%s", seam.ID, seam.DeclaredBudget, seam.Observed, seam.Reserve, strings.Join(seam.Verdicts, ",")))
		}
		for _, metric := range report.Metrics {
			rows = append(rows, fmt.Sprintf("metric %-38s budget=%s baseline=%s observed=%d verdict=%s", metric.Name, optionalInt(metric.DeclaredBudget), optionalInt(metric.DeclaredBaseline), metric.Observed, strings.Join(metric.Verdicts, ",")))
		}
		if err := cliapp.RenderListReport(os.Stdout, cliapp.ListReport{Summary: []string{fmt.Sprintf("Scenario: %s", report.Scenario), fmt.Sprintf("Blocking verdicts: %t", report.Blocking)}, ResultsHeading: "Budget Audit", Results: rows}); err != nil {
			return err
		}
	}
	if report.Blocking {
		return fmt.Errorf("budget audit reported blocking verdicts")
	}
	return nil
}

func optionalInt(value *int) string {
	if value == nil {
		return "undeclared"
	}
	return fmt.Sprintf("%d", *value)
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
	report.Summary = append(report.Summary, duplicationOpportunitySummary(findings)...)

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

func duplicationOpportunitySummary(findings []*validationv1.TidinessFinding) []string {
	type opportunity struct {
		class string
		debt  float64
	}
	byFamily := map[string]*opportunity{}
	for _, finding := range findings {
		if finding.GetRuleId() != "DUPLICATED_CODE" || finding.GetEvidence() == nil {
			continue
		}
		fields := finding.GetEvidence().GetFields()
		class := fields["duplication_class"].GetStringValue()
		debt := fields["duplication_line_debt"].GetNumberValue()
		if class == "" || debt <= 0 {
			continue
		}
		family := path.Dir(finding.GetFilePath())
		if family == "." || family == "" {
			family = finding.GetFilePath()
		}
		item := byFamily[family]
		if item == nil {
			item = &opportunity{class: class}
			byFamily[family] = item
		}
		item.debt += debt
		if class == "high-leverage" {
			item.class = class
		}
	}
	type ranked struct {
		family string
		opportunity
	}
	rankedItems := make([]ranked, 0, len(byFamily))
	for family, item := range byFamily {
		rankedItems = append(rankedItems, ranked{family: family, opportunity: *item})
	}
	sort.Slice(rankedItems, func(i, j int) bool {
		if rankedItems[i].debt != rankedItems[j].debt {
			return rankedItems[i].debt > rankedItems[j].debt
		}
		return rankedItems[i].class == "high-leverage"
	})
	if len(rankedItems) == 0 {
		return nil
	}
	if len(rankedItems) > 5 {
		rankedItems = rankedItems[:5]
	}
	rows := []string{"Top duplication opportunities:"}
	for _, item := range rankedItems {
		rows = append(rows, fmt.Sprintf("- %s (%s, %.0f line-debt)", item.family, item.class, item.debt))
	}
	return rows
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
			fmt.Sprintf("%s files: %d", scanFileLabel(result.Incremental), result.TotalFiles),
			fmt.Sprintf("%s lines: %d", scanLineLabel(result.Incremental), result.TotalLines),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Static Analysis", Items: []string{
				fmt.Sprintf("Lint issues: %d", result.LintIssuesCount),
				fmt.Sprintf("Type issues: %d", result.TypeIssuesCount),
				commandSummary("make lint", result.LintOutput),
				commandSummary("make type", result.TypeOutput),
			}},
			{Heading: "File Size", Items: longFileRows(result.LongFiles, result.LongFilesCount)},
			{Heading: "Language Metrics", Items: languageMetricRows(result.LanguageMetrics)},
		},
		NextSteps: []string{
			fmt.Sprintf("%s issues %s --limit 20", cliName, result.Scenario),
			fmt.Sprintf("%s recommend-refactors %s --limit 5", cliName, result.Scenario),
		},
	}

	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, result)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func scanFileLabel(incremental bool) string {
	if incremental {
		return "Changed files"
	}
	return "Files scanned"
}

func scanLineLabel(incremental bool) string {
	if incremental {
		return "Changed-file lines"
	}
	return "Total lines"
}

func languageMetricRows(metrics map[string]support.LightLanguageMetrics) []string {
	languages := make([]string, 0, len(metrics))
	for language := range metrics {
		languages = append(languages, language)
	}
	sort.Strings(languages)
	rows := make([]string, 0, len(languages))
	for _, language := range languages {
		metric := metrics[language]
		row := fmt.Sprintf("%s: %d files, %d lines", language, metric.FileCount, metric.TotalLines)
		if len(metric.CodeMetrics) > 0 {
			var counts struct {
				Todo         int     `json:"todo_count"`
				Fixme        int     `json:"fixme_count"`
				Hack         int     `json:"hack_count"`
				AvgImports   float64 `json:"avg_imports_per_file"`
				MaxImports   int     `json:"max_imports_in_file"`
				AvgFunctions float64 `json:"avg_functions_per_file"`
				MaxFunctions int     `json:"max_functions_in_file"`
			}
			if json.Unmarshal(metric.CodeMetrics, &counts) == nil {
				row += fmt.Sprintf("; TODO/FIXME/HACK %d/%d/%d; imports avg/max %.1f/%d; functions avg/max %.1f/%d",
					counts.Todo, counts.Fixme, counts.Hack,
					counts.AvgImports, counts.MaxImports,
					counts.AvgFunctions, counts.MaxFunctions)
			}
		}
		var complexity struct {
			AverageComplexity   float64 `json:"average_complexity"`
			MaxComplexity       int     `json:"max_complexity"`
			HighComplexityCount int     `json:"high_complexity_count"`
			TotalFunctions      int     `json:"total_functions"`
			Skipped             bool    `json:"skipped"`
			SkipReason          string  `json:"skip_reason"`
			Tool                string  `json:"tool"`
		}
		if len(metric.Complexity) > 0 && json.Unmarshal(metric.Complexity, &complexity) == nil {
			if complexity.Skipped {
				row += "; complexity skipped: " + complexity.SkipReason
			} else {
				row += fmt.Sprintf("; complexity functions/avg/max/high %d/%.1f/%d/%d via %s",
					complexity.TotalFunctions, complexity.AverageComplexity, complexity.MaxComplexity, complexity.HighComplexityCount, complexity.Tool)
			}
		}
		var duplicates struct {
			TotalDuplicates int    `json:"total_duplicates"`
			TotalLines      int    `json:"total_lines"`
			Skipped         bool   `json:"skipped"`
			SkipReason      string `json:"skip_reason"`
			Tool            string `json:"tool"`
		}
		if len(metric.Duplicates) > 0 && json.Unmarshal(metric.Duplicates, &duplicates) == nil {
			if duplicates.Skipped {
				row += "; duplication skipped: " + duplicates.SkipReason
			} else {
				row += fmt.Sprintf("; %d duplicate groups / %d lines via %s", duplicates.TotalDuplicates, duplicates.TotalLines, duplicates.Tool)
			}
		}
		rows = append(rows, row)
	}
	return rows
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
		if strings.TrimSpace(batch.Duration.String()) != "" {
			line += " | duration: " + batch.Duration.String()
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
	return maturityreport.CountFindingsBySeverity(a.GetFindingsBySeverity(), severity)
}
