package run

import (
	"fmt"
	"sort"
	"strings"

	"scenario-stack-governor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Governance Runs",
		Commands: []cliapp.Command{
			{Name: "run", Aliases: []string{"audit", "check"}, NeedsAPI: true, Description: "Run enabled governance rules across the repo or selected scenarios", Run: func(args []string) error { return run(core, args) }},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("run")
	var scenarios cliutil.StringList
	fs.Var(&scenarios, "scenario", "Scenario to evaluate (repeatable)")
	scenarioCSV := fs.String("scenarios", "", "Comma-separated scenarios to evaluate")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	request := support.RunRequest{
		ScenarioNames: support.ParseMultiValue(*scenarioCSV, scenarios.Values(), fs.Args()),
	}

	var resp support.RunResponse
	if err := support.RequestJSON(core, "POST", "/run", nil, request, &resp); err != nil {
		return err
	}
	support.SortRuleResults(resp.Results)

	rulesWithFindings := 0
	totalErrors := 0
	totalWarnings := 0
	for _, result := range resp.Results {
		if !result.Passed {
			rulesWithFindings++
		}
		totalErrors += result.ErrorCount
		totalWarnings += result.WarnCount
	}

	triage := []cliapp.TriageGroup{
		{Heading: "Rule Results", Items: renderRuleResults(resp.Results)},
	}
	if resp.TimedOut {
		triage = append(triage, cliapp.TriageGroup{
			Heading: "Timeout",
			Items:   []string{"The run hit the server timeout before all rules completed. Re-run against fewer scenarios if you need a narrower slice."},
		})
	}

	nextSteps := []string{"scenario-stack-governor rules list", "scenario-stack-governor scenarios list"}
	if len(request.ScenarioNames) > 0 {
		nextSteps = append(nextSteps, "scenario-stack-governor fix --scenario "+request.ScenarioNames[0]+" --dry-run")
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Repo root: %s", resp.RepoRoot),
			fmt.Sprintf("Rules evaluated: %d", len(resp.Results)),
			fmt.Sprintf("Rules with findings: %d", rulesWithFindings),
			fmt.Sprintf("Errors: %d", totalErrors),
			fmt.Sprintf("Warnings: %d", totalWarnings),
		},
		Triage:    triage,
		NextSteps: nextSteps,
	}
	return support.PrintOperational(*jsonOutput, resp, report)
}

func renderRuleResults(results []support.RuleResult) []string {
	lines := make([]string, 0, len(results))
	for _, result := range results {
		line := fmt.Sprintf("[%s] %s - %d errors, %d warnings", support.StatusWord(result.Passed), result.RuleID, result.ErrorCount, result.WarnCount)
		summaries := findingSummaries(result.Findings)
		if len(summaries) > 0 {
			line += "\n" + strings.Join(summaries, "\n")
		}
		lines = append(lines, line)
	}
	return lines
}

func findingSummaries(findings []support.Finding) []string {
	if len(findings) == 0 {
		return nil
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if support.SeverityRank(findings[i].Level) != support.SeverityRank(findings[j].Level) {
			return support.SeverityRank(findings[i].Level) < support.SeverityRank(findings[j].Level)
		}
		if findings[i].ScenarioName != findings[j].ScenarioName {
			return findings[i].ScenarioName < findings[j].ScenarioName
		}
		return findings[i].Message < findings[j].Message
	})

	limit := len(findings)
	if limit > 3 {
		limit = 3
	}
	lines := make([]string, 0, limit+1)
	for _, finding := range findings[:limit] {
		prefix := strings.ToUpper(strings.TrimSpace(finding.Level))
		detail := strings.TrimSpace(finding.Message)
		if scenario := strings.TrimSpace(finding.ScenarioName); scenario != "" {
			detail = scenario + ": " + detail
		}
		lines = append(lines, "- ["+prefix+"] "+detail)
	}
	if len(findings) > limit {
		lines = append(lines, fmt.Sprintf("- ... %d more finding(s)", len(findings)-limit))
	}
	return lines
}
