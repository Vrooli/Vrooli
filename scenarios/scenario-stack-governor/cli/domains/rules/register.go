package rules

import (
	"fmt"
	"strings"

	"scenario-stack-governor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "rules",
		Description: "Rule inventory and enablement management",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List governance rules", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", NeedsAPI: true, Description: "Show one rule", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "enable", NeedsAPI: true, Description: "Enable one or more rules", Run: func(args []string) error { return runSetEnabled(core, args, true) }},
			{Name: "disable", NeedsAPI: true, Description: "Disable one or more rules", Run: func(args []string) error { return runSetEnabled(core, args, false) }},
			{Name: "reset", NeedsAPI: true, Description: "Reset rule enablement to defaults", Run: func(args []string) error { return runReset(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("rules list")
	category := fs.String("category", "", "Filter by rule category")
	severity := fs.String("severity", "", "Filter by rule severity")
	onlyEnabled := fs.Bool("enabled", false, "Show only enabled rules")
	onlyDisabled := fs.Bool("disabled", false, "Show only disabled rules")
	onlyFixable := fs.Bool("fixable", false, "Show only fixable rules")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *onlyEnabled && *onlyDisabled {
		return fmt.Errorf("choose at most one of --enabled or --disabled")
	}

	var resp support.RulesResponse
	if err := support.GetJSON(core, "/rules", nil, &resp); err != nil {
		return err
	}
	support.SortRules(resp.Rules)

	filtered := make([]support.Rule, 0, len(resp.Rules))
	enabledCount := 0
	fixableCount := 0
	for _, rule := range resp.Rules {
		if rule.Enabled {
			enabledCount++
		}
		if rule.Fixable {
			fixableCount++
		}
		if *category != "" && !strings.EqualFold(rule.Category, *category) {
			continue
		}
		if *severity != "" && !strings.EqualFold(rule.Severity, *severity) {
			continue
		}
		if *onlyEnabled && !rule.Enabled {
			continue
		}
		if *onlyDisabled && rule.Enabled {
			continue
		}
		if *onlyFixable && !rule.Fixable {
			continue
		}
		filtered = append(filtered, rule)
	}

	results := make([]string, 0, len(filtered))
	for _, rule := range filtered {
		status := "disabled"
		if rule.Enabled {
			status = "enabled"
		}
		fixable := "manual-only"
		if rule.Fixable {
			fixable = "fixable"
		}
		results = append(results, fmt.Sprintf("[%s/%s/%s] %s - %s", rule.Severity, rule.Category, fixable, rule.ID, statusTitle(rule, status)))
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Rules returned: %d", len(filtered)),
			fmt.Sprintf("Enabled rules: %d", enabledCount),
			fmt.Sprintf("Fixable rules: %d", fixableCount),
		},
		Results:        results,
		RetrievalHints: []string{"scenario-stack-governor rules get <rule-id>", "scenario-stack-governor run --scenario <scenario>"},
	}
	return support.PrintList(*jsonOutput, resp, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("rules get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: rules get <rule-id> [--json]")
	}

	var resp support.RulesResponse
	if err := support.GetJSON(core, "/rules", nil, &resp); err != nil {
		return err
	}
	rule := support.FindRuleByID(resp.Rules, fs.Arg(0))
	if rule == nil {
		return fmt.Errorf("unknown rule %q", fs.Arg(0))
	}

	status := "disabled"
	if rule.Enabled {
		status = "enabled"
	}
	results := []string{
		fmt.Sprintf("Title: %s", rule.Title),
		fmt.Sprintf("Category: %s", rule.Category),
		fmt.Sprintf("Severity: %s", rule.Severity),
		fmt.Sprintf("Default enabled: %t", rule.DefaultEnabled),
		fmt.Sprintf("Fixable: %t", rule.Fixable),
		fmt.Sprintf("Summary: %s", rule.Summary),
		fmt.Sprintf("Why important: %s", rule.WhyImportant),
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Rule: %s", rule.ID), fmt.Sprintf("Status: %s", status)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{"scenario-stack-governor rules enable " + rule.ID, "scenario-stack-governor run --scenario <scenario>"},
	}
	return support.PrintList(*jsonOutput, rule, report)
}

func runSetEnabled(core *cliapp.ScenarioApp, args []string, enabled bool) error {
	verb := "enable"
	if !enabled {
		verb = "disable"
	}
	fs := support.NewFlagSet("rules " + verb)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: rules %s <rule-id> [<rule-id> ...] [--json]", verb)
	}

	var rulesResp support.RulesResponse
	if err := support.GetJSON(core, "/rules", nil, &rulesResp); err != nil {
		return err
	}
	cfg := rulesResp.Config
	if cfg.EnabledRules == nil {
		cfg.EnabledRules = map[string]bool{}
	}

	targetIDs := support.ParseMultiValue("", nil, fs.Args())
	changed := make([]string, 0, len(targetIDs))
	for _, id := range targetIDs {
		rule := support.FindRuleByID(rulesResp.Rules, id)
		if rule == nil {
			return fmt.Errorf("unknown rule %q", id)
		}
		cfg.EnabledRules[id] = enabled
		changed = append(changed, fmt.Sprintf("%s -> %t", id, enabled))
	}

	var updated support.RulesConfig
	if err := support.RequestJSON(core, "PUT", "/config", nil, cfg, &updated); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated %d rule(s)", len(targetIDs))},
		Changes:     changed,
		NextCommand: []string{"scenario-stack-governor rules list", "scenario-stack-governor run"},
	}
	return support.PrintMutation(*jsonOutput, updated, report)
}

func runReset(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("rules reset")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var resp support.RulesResponse
	if err := support.GetJSON(core, "/rules", nil, &resp); err != nil {
		return err
	}
	cfg := support.RulesConfig{
		Version:      resp.Config.Version,
		EnabledRules: map[string]bool{},
	}
	for _, id := range support.DefaultEnabledRuleIDs(resp.Rules) {
		cfg.EnabledRules[id] = true
	}

	var updated support.RulesConfig
	if err := support.RequestJSON(core, "PUT", "/config", nil, cfg, &updated); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Reset rule enablement to defaults"},
		Changes: []string{
			fmt.Sprintf("Enabled rules after reset: %d", len(updated.EnabledRules)),
		},
		NextCommand: []string{"scenario-stack-governor rules list", "scenario-stack-governor run"},
	}
	return support.PrintMutation(*jsonOutput, updated, report)
}

func statusTitle(rule support.Rule, status string) string {
	title := strings.TrimSpace(rule.Title)
	if title == "" {
		return status
	}
	return title + " (" + status + ")"
}
