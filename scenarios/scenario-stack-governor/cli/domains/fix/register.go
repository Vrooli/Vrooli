package fix

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
		Title: "Governance Fixes",
		Commands: []cliapp.Command{
			{Name: "fix", NeedsAPI: true, Description: "Apply or preview automated rule fixes for one or more scenarios", Run: func(args []string) error { return run(core, args) }},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("fix")
	var scenarios cliutil.StringList
	var rules cliutil.StringList
	fs.Var(&scenarios, "scenario", "Scenario to fix (repeatable)")
	fs.Var(&rules, "rule", "Rule ID to fix (repeatable)")
	scenarioCSV := fs.String("scenarios", "", "Comma-separated scenarios to fix")
	ruleCSV := fs.String("rules", "", "Comma-separated rule IDs to fix")
	dryRun := fs.Bool("dry-run", false, "Preview changes without writing files")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	request := support.FixRequest{
		ScenarioNames: support.ParseMultiValue(*scenarioCSV, scenarios.Values(), fs.Args()),
		RuleIDs:       support.ParseMultiValue(*ruleCSV, rules.Values(), nil),
		DryRun:        *dryRun,
	}
	if len(request.ScenarioNames) == 0 {
		return fmt.Errorf("at least one scenario is required. Usage: fix --scenario <name> [--rule <rule-id>] [--dry-run]")
	}

	var resp support.FixResponse
	if err := support.RequestJSON(core, "POST", "/fix", nil, request, &resp); err != nil {
		return err
	}

	fixedCount := 0
	errorCount := 0
	changedFiles := map[string]struct{}{}
	changes := make([]string, 0, len(resp.Results))
	sort.SliceStable(resp.Results, func(i, j int) bool {
		if resp.Results[i].ScenarioName != resp.Results[j].ScenarioName {
			return resp.Results[i].ScenarioName < resp.Results[j].ScenarioName
		}
		if resp.Results[i].RuleID != resp.Results[j].RuleID {
			return resp.Results[i].RuleID < resp.Results[j].RuleID
		}
		return resp.Results[i].FilePath < resp.Results[j].FilePath
	})

	for _, result := range resp.Results {
		if result.Fixed {
			fixedCount++
		}
		if strings.TrimSpace(result.Error) != "" {
			errorCount++
		}
		if path := strings.TrimSpace(result.FilePath); path != "" {
			changedFiles[path] = struct{}{}
		}
		changes = append(changes, summarizeFixResult(result))
	}

	action := "Applied"
	if *dryRun {
		action = "Previewed"
	}
	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("%s fixes for %d scenario(s)", action, len(request.ScenarioNames)),
			fmt.Sprintf("Repo root: %s", resp.RepoRoot),
		},
		Changes: []string{
			fmt.Sprintf("Rule results returned: %d", len(resp.Results)),
			fmt.Sprintf("Successful fixes: %d", fixedCount),
			fmt.Sprintf("Results with errors: %d", errorCount),
			fmt.Sprintf("Files touched: %d", len(changedFiles)),
			strings.Join(changes, "\n"),
		},
		NextCommand: nextCommands(*dryRun, request),
	}
	return support.PrintMutation(*jsonOutput, resp, report)
}

func summarizeFixResult(result support.FixResult) string {
	status := "skipped"
	if result.Fixed {
		status = "fixed"
	}
	if strings.TrimSpace(result.Error) != "" {
		status = "error"
	}

	parts := []string{fmt.Sprintf("[%s] %s / %s", strings.ToUpper(status), result.ScenarioName, result.RuleID)}
	if path := strings.TrimSpace(result.FilePath); path != "" {
		parts = append(parts, path)
	}
	if len(result.Changes) > 0 {
		changeLines := make([]string, 0, len(result.Changes))
		for _, change := range result.Changes {
			text := strings.TrimSpace(change.Detail)
			if text == "" {
				text = change.Type
			}
			changeLines = append(changeLines, "- "+text)
		}
		parts = append(parts, strings.Join(changeLines, "\n"))
	}
	if errText := strings.TrimSpace(result.Error); errText != "" {
		parts = append(parts, "- error: "+errText)
	}
	if result.Diff != nil {
		parts = append(parts, "- diff available in --json output")
	}
	return strings.Join(parts, "\n")
}

func nextCommands(dryRun bool, request support.FixRequest) []string {
	if len(request.ScenarioNames) == 0 {
		return []string{"scenario-stack-governor run"}
	}
	first := request.ScenarioNames[0]
	if dryRun {
		return []string{
			"scenario-stack-governor fix --scenario " + first,
			"scenario-stack-governor run --scenario " + first,
		}
	}
	return []string{
		"scenario-stack-governor run --scenario " + first,
		"git diff -- scenarios/" + first,
	}
}
