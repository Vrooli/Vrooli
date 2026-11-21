package overrides

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
		Name:        "overrides",
		Description: "Scenario-specific secret strategy overrides",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List overrides for one scenario, optionally filtered to a tier", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get one override record", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "set", NeedsAPI: true, Description: "Create or update one override", Run: func(args []string) error { return runSet(core, args) }},
			{Name: "delete", NeedsAPI: true, Description: "Delete one override and fall back to default strategy", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "effective", NeedsAPI: true, Description: "Show effective strategies after override merging", Run: func(args []string) error { return runEffective(core, args) }},
			{Name: "copy-from-tier", NeedsAPI: true, Description: "Copy overrides between tiers for one scenario", Run: func(args []string) error { return runCopyFromTier(core, args) }},
			{Name: "copy-from-scenario", NeedsAPI: true, Description: "Copy overrides from another scenario", Run: func(args []string) error { return runCopyFromScenario(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("overrides list")
	tier := fs.String("tier", "", "Optional tier filter")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: overrides list <scenario> [--tier <tier>]")
	}
	scenario := fs.Arg(0)

	path := "/scenarios/" + scenario + "/overrides"
	if strings.TrimSpace(*tier) != "" {
		path += "/" + *tier
	}
	var resp support.OverridesListResponse
	if err := support.GetJSON(core, path, nil, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Overrides))
	for _, item := range resp.Overrides {
		results = append(results, fmt.Sprintf("%s/%s | tier=%s | strategy=%s | reason=%s",
			item.ResourceName, item.SecretKey, item.Tier, deref(item.HandlingStrategy), deref(item.OverrideReason)))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scenario: %s", scenario), fmt.Sprintf("Overrides: %d", resp.Count)},
		ResultsHeading: "Overrides",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " overrides effective " + scenario + " --tier " + support.Fallback(*tier, "tier-2-desktop")},
	}
	return support.PrintList(*jsonOutput, resp, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("overrides get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 4 {
		return fmt.Errorf("usage: overrides get <scenario> <tier> <resource> <secret>")
	}

	path := fmt.Sprintf("/scenarios/%s/overrides/%s/%s/%s", fs.Arg(0), fs.Arg(1), fs.Arg(2), fs.Arg(3))
	var resp support.ScenarioSecretOverride
	if err := support.GetJSON(core, path, nil, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scenario: %s", resp.ScenarioName), fmt.Sprintf("Resource/secret: %s/%s", resp.ResourceName, resp.SecretKey)},
		ResultsHeading: "Override",
		Results: []string{
			"Tier: " + resp.Tier,
			"Handling strategy: " + deref(resp.HandlingStrategy),
			"Fallback strategy: " + deref(resp.FallbackStrategy),
			"Reason: " + deref(resp.OverrideReason),
		},
		RetrievalHints: []string{support.CLIName + " overrides delete " + fs.Arg(0) + " " + fs.Arg(1) + " " + fs.Arg(2) + " " + fs.Arg(3)},
	}
	return support.PrintList(*jsonOutput, resp, report)
}

func runSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("overrides set")
	handling := fs.String("handling-strategy", "", "Override handling strategy")
	fallbackStrategy := fs.String("fallback-strategy", "", "Override fallback strategy")
	requiresUserInput := fs.String("requires-user-input", "", "Override requires_user_input to true or false")
	promptLabel := fs.String("prompt-label", "", "Override prompt label")
	promptDescription := fs.String("prompt-description", "", "Override prompt description")
	overrideReason := fs.String("reason", "", "Human explanation for the override")
	generatorTemplate := fs.String("generator-template", "", "Generator template JSON object")
	bundleHints := fs.String("bundle-hints", "", "Bundle hints JSON object")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 4 {
		return fmt.Errorf("usage: overrides set <scenario> <tier> <resource> <secret> [flags]")
	}

	payload := map[string]any{}
	if *handling != "" {
		payload["handling_strategy"] = *handling
	}
	if *fallbackStrategy != "" {
		payload["fallback_strategy"] = *fallbackStrategy
	}
	if *requiresUserInput != "" {
		payload["requires_user_input"] = *requiresUserInput == "true"
	}
	if *promptLabel != "" {
		payload["prompt_label"] = *promptLabel
	}
	if *promptDescription != "" {
		payload["prompt_description"] = *promptDescription
	}
	if *overrideReason != "" {
		payload["override_reason"] = *overrideReason
	}
	generator, err := support.ParseJSONMap(*generatorTemplate)
	if err != nil {
		return err
	}
	if generator != nil {
		payload["generator_template"] = generator
	}
	hints, err := support.ParseJSONMap(*bundleHints)
	if err != nil {
		return err
	}
	if hints != nil {
		payload["bundle_hints"] = hints
	}
	if len(payload) == 0 {
		return fmt.Errorf("at least one override field is required")
	}

	path := fmt.Sprintf("/scenarios/%s/overrides/%s/%s/%s", fs.Arg(0), fs.Arg(1), fs.Arg(2), fs.Arg(3))
	var resp support.ScenarioSecretOverride
	if err := support.RequestJSON(core, "POST", path, nil, payload, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Override saved", fmt.Sprintf("Scenario: %s", resp.ScenarioName)},
		Changes:     []string{"Resource/secret: " + resp.ResourceName + "/" + resp.SecretKey, "Tier: " + resp.Tier, "Handling strategy: " + deref(resp.HandlingStrategy)},
		NextCommand: []string{support.CLIName + " overrides get " + fs.Arg(0) + " " + fs.Arg(1) + " " + fs.Arg(2) + " " + fs.Arg(3), support.CLIName + " overrides effective " + fs.Arg(0) + " --tier " + fs.Arg(1)},
	}
	return support.PrintMutation(*jsonOutput, resp, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("overrides delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 4 {
		return fmt.Errorf("usage: overrides delete <scenario> <tier> <resource> <secret>")
	}

	path := fmt.Sprintf("/scenarios/%s/overrides/%s/%s/%s", fs.Arg(0), fs.Arg(1), fs.Arg(2), fs.Arg(3))
	var resp support.OverrideMutationResponse
	if err := support.RequestJSON(core, "DELETE", path, nil, nil, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Override deleted", support.Fallback(resp.Message, "Override removed")},
		Changes:     []string{"Success: " + support.BoolLabel(resp.Success, "yes", "no")},
		NextCommand: []string{support.CLIName + " overrides effective " + fs.Arg(0) + " --tier " + fs.Arg(1)},
	}
	return support.PrintMutation(*jsonOutput, resp, report)
}

func runEffective(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("overrides effective")
	tier := fs.String("tier", "", "Tier to inspect")
	var resources cliutil.StringList
	fs.Var(&resources, "resource", "Resource filter; repeatable")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: overrides effective <scenario> --tier <tier> [--resource <resource>]")
	}
	if *tier == "" {
		return fmt.Errorf("--tier is required")
	}

	query := make(url.Values)
	if values := resources.Values(); len(values) > 0 {
		query["resources"] = []string{strings.Join(values, ",")}
	}

	var resp support.EffectiveStrategiesResponse
	path := fmt.Sprintf("/scenarios/%s/effective/%s", fs.Arg(0), *tier)
	if err := support.GetJSON(core, path, query, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Strategies))
	for _, item := range resp.Strategies {
		results = append(results, fmt.Sprintf("%s/%s | %s | override=%t | prompt=%t",
			item.ResourceName, item.SecretKey, item.HandlingStrategy, item.IsOverridden, item.RequiresUserInput))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scenario: %s", resp.Scenario), fmt.Sprintf("Tier: %s", resp.Tier), fmt.Sprintf("Strategies: %d", resp.Count)},
		ResultsHeading: "Effective Strategies",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " deployment plan --scenario " + resp.Scenario + " --tier " + resp.Tier},
	}
	return support.PrintList(*jsonOutput, resp, report)
}

func runCopyFromTier(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("overrides copy-from-tier")
	sourceTier := fs.String("source-tier", "", "Source tier")
	targetTier := fs.String("target-tier", "", "Target tier")
	overwrite := fs.Bool("overwrite", false, "Overwrite existing overrides")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: overrides copy-from-tier <scenario> --source-tier <tier> --target-tier <tier>")
	}
	if *sourceTier == "" || *targetTier == "" {
		return fmt.Errorf("--source-tier and --target-tier are required")
	}

	payload := map[string]any{"source_tier": *sourceTier, "target_tier": *targetTier, "overwrite": *overwrite}
	var resp support.OverrideMutationResponse
	path := fmt.Sprintf("/scenarios/%s/overrides/copy-from-tier", fs.Arg(0))
	if err := support.RequestJSON(core, "POST", path, nil, payload, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Overrides copied from tier", fmt.Sprintf("Copied: %d", resp.Copied)},
		Changes:     []string{"Source tier: " + resp.SourceTier, "Target tier: " + resp.TargetTier, fmt.Sprintf("Overwrite: %t", resp.Overwrite)},
		NextCommand: []string{support.CLIName + " overrides list " + fs.Arg(0) + " --tier " + *targetTier},
	}
	return support.PrintMutation(*jsonOutput, resp, report)
}

func runCopyFromScenario(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("overrides copy-from-scenario")
	sourceScenario := fs.String("source-scenario", "", "Source scenario")
	tier := fs.String("tier", "", "Tier to copy")
	overwrite := fs.Bool("overwrite", false, "Overwrite existing overrides")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: overrides copy-from-scenario <target-scenario> --source-scenario <scenario> --tier <tier>")
	}
	if *sourceScenario == "" || *tier == "" {
		return fmt.Errorf("--source-scenario and --tier are required")
	}

	payload := map[string]any{"source_scenario": *sourceScenario, "tier": *tier, "overwrite": *overwrite}
	var resp support.OverrideMutationResponse
	path := fmt.Sprintf("/scenarios/%s/overrides/copy-from-scenario", fs.Arg(0))
	if err := support.RequestJSON(core, "POST", path, nil, payload, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Overrides copied from scenario", fmt.Sprintf("Copied: %d", resp.Copied)},
		Changes:     []string{"Source scenario: " + resp.SourceScenario, "Target scenario: " + resp.TargetScenario, "Tier: " + resp.Tier},
		NextCommand: []string{support.CLIName + " overrides list " + fs.Arg(0) + " --tier " + *tier},
	}
	return support.PrintMutation(*jsonOutput, resp, report)
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
