package scan

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"tidiness-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "tidiness-manager"

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Scanning",
		Commands: []cliapp.Command{
			{
				Name:        "scan",
				NeedsAPI:    true,
				Description: "Run light, smart, or type-safety scans",
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scan")
	scanType := fs.String("type", "light", "Scan type: light, smart, or type-safety")
	timeout := fs.Int("timeout", 120, "Light scan timeout in seconds")
	var files cliutil.StringList
	fs.Var(&files, "file", "File to include in smart scan (repeatable)")
	forceRescan := fs.Bool("force-rescan", false, "Re-analyze files already seen in this smart scan session")
	campaignID := fs.Int("campaign-id", 0, "Attach smart scan results to a campaign")
	includePatterns := fs.Bool("include-patterns", false, "Include dangerous pattern summary in type-safety scans")
	fix := fs.Bool("fix", false, "Apply supported fixes for the selected scan type")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	target := "."
	if fs.NArg() > 0 {
		target = fs.Arg(0)
	}

	switch strings.ToLower(strings.TrimSpace(*scanType)) {
	case "light":
		return runLight(core, target, *timeout, *jsonOutput)
	case "smart":
		return runSmart(core, target, files.Values(), *forceRescan, *campaignID, *jsonOutput)
	case "type-safety", "type", "typesafety":
		return runTypeSafety(core, target, *includePatterns, *fix, *jsonOutput)
	default:
		return fmt.Errorf("unsupported scan type %q", *scanType)
	}
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

func runTypeSafety(core *cliapp.ScenarioApp, target string, includePatterns bool, fix bool, jsonOutput bool) error {
	scenario := support.ScenarioName(target)
	if strings.TrimSpace(scenario) == "" {
		return fmt.Errorf("scenario name is required for type-safety scans")
	}

	endpoint := "/scan/type-safety"
	if fix {
		endpoint = "/scan/type-safety/fix"
	}

	body, err := core.Request("POST", endpoint, nil, map[string]interface{}{
		"scenario_name":    scenario,
		"include_patterns": includePatterns,
	})
	if err != nil {
		return err
	}

	var result support.TypeSafetyConfigResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	violations := result.Violations
	sort.SliceStable(violations, func(i, j int) bool {
		return support.SeverityRank(violations[i].Severity) < support.SeverityRank(violations[j].Severity)
	})

	summary := []string{
		fmt.Sprintf("Scenario: %s", scenario),
		fmt.Sprintf("Violations: %d", len(violations)),
		support.StatusLine(result.TSConfigFound, "tsconfig.json detected", ""),
		support.StatusLine(result.ESLintConfigFound, "ESLint config detected", ""),
	}
	if result.TSConfigFound {
		summary = append(summary,
			support.StatusLine(result.TSConfigStrict, "TypeScript strict mode", ""),
			support.StatusLine(result.TSConfigNoUnchecked, "noUncheckedIndexedAccess", ""),
		)
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Violations",
		Results:        typeSafetyRows(violations, result.PatternSummary),
		RetrievalHints: []string{fmt.Sprintf("%s scan %s --type type-safety --include-patterns", cliName, scenario)},
	}
	if fix {
		report.RetrievalHints = append(report.RetrievalHints, fmt.Sprintf("%s scan %s --type type-safety", cliName, scenario))
	}

	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
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

func typeSafetyRows(violations []support.TypeSafetyViolation, patterns *support.TypeSafetyPatternSummary) []string {
	rows := make([]string, 0, len(violations)+4)
	for _, violation := range violations {
		rows = append(rows, fmt.Sprintf("[%s] %s", strings.ToUpper(violation.Severity), violation.Title))
	}
	if patterns != nil {
		rows = append(rows,
			fmt.Sprintf("Pattern totals: as-any=%d, assertions=%d, ts-ignore=%d, non-null=%d",
				patterns.AsAnyCount, patterns.AsTypeAssertionCount, patterns.TsIgnoreCount, patterns.NonNullAssertionCount),
		)
		limit := len(patterns.TopFiles)
		if limit > 3 {
			limit = 3
		}
		for _, top := range patterns.TopFiles[:limit] {
			rows = append(rows, fmt.Sprintf("Pattern hotspot: %s (%d markers)", top.FilePath, top.Total))
		}
	}
	if len(rows) == 0 {
		rows = append(rows, "No type-safety violations detected")
	}
	return rows
}
