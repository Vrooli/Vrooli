package resources

import (
	"fmt"

	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "resources",
		Description: "Resource-level secret metadata and strategy management",
		Subcommands: []cliapp.Command{
			{Name: "get", NeedsAPI: true, Description: "Show one resource's secret inventory", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "update-secret", NeedsAPI: true, Description: "Update secret metadata on one resource", Run: func(args []string) error { return runUpdateSecret(core, args) }},
			{Name: "set-strategy", NeedsAPI: true, Description: "Set the deployment strategy for one resource secret", Run: func(args []string) error { return runSetStrategy(core, args) }},
		},
	}
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resources get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: resources get <resource>")
	}

	var resp support.ResourceDetail
	if err := support.GetJSON(core, "/resources/"+fs.Arg(0), nil, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Secrets)+len(resp.OpenVulnerabilities))
	for _, secret := range resp.Secrets {
		results = append(results, fmt.Sprintf("%s | %s | required=%t | %s | validation=%s",
			secret.SecretKey, secret.Classification, secret.Required, secret.SecretType, secret.ValidationState))
	}
	for _, vuln := range resp.OpenVulnerabilities {
		results = append(results, fmt.Sprintf("Open vulnerability: %s | %s | %s:%d", vuln.ID, vuln.Severity, vuln.FilePath, vuln.LineNumber))
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Resource: %s", resp.ResourceName),
			fmt.Sprintf("Valid secrets: %d/%d", resp.ValidSecrets, resp.TotalSecrets),
			fmt.Sprintf("Missing secrets: %d", resp.MissingSecrets),
		},
		ResultsHeading: "Resource Details",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " resources update-secret " + resp.ResourceName + " <secret> --description \"...\"", support.CLIName + " resources set-strategy " + resp.ResourceName + " <secret> --tier tier-2-desktop --handling-strategy prompt"},
	}
	return support.PrintList(*jsonOutput, resp, report)
}

func runUpdateSecret(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resources update-secret")
	classification := fs.String("classification", "", "Classification: infrastructure|service|user")
	description := fs.String("description", "", "Secret description")
	required := fs.String("required", "", "Set required to true or false")
	ownerTeam := fs.String("owner-team", "", "Owning team")
	ownerContact := fs.String("owner-contact", "", "Owning contact")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: resources update-secret <resource> <secret> [flags]")
	}

	payload := map[string]any{}
	if *classification != "" {
		payload["classification"] = *classification
	}
	if *description != "" {
		payload["description"] = *description
	}
	if *required != "" {
		payload["required"] = *required == "true"
	}
	if *ownerTeam != "" {
		payload["owner_team"] = *ownerTeam
	}
	if *ownerContact != "" {
		payload["owner_contact"] = *ownerContact
	}
	if len(payload) == 0 {
		return fmt.Errorf("at least one update flag is required")
	}

	resource, secret := fs.Arg(0), fs.Arg(1)
	var resp support.ResourceSecretDetail
	if err := support.RequestJSON(core, "PATCH", "/resources/"+resource+"/secrets/"+secret, nil, payload, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Secret metadata updated", fmt.Sprintf("Resource/secret: %s/%s", resource, resp.SecretKey)},
		Changes: []string{
			"Classification: " + resp.Classification,
			fmt.Sprintf("Required: %t", resp.Required),
			"Owner team: " + support.Fallback(resp.OwnerTeam, "unset"),
			"Owner contact: " + support.Fallback(resp.OwnerContact, "unset"),
		},
		NextCommand: []string{support.CLIName + " resources get " + resource},
	}
	return support.PrintMutation(*jsonOutput, resp, report)
}

func runSetStrategy(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resources set-strategy")
	tier := fs.String("tier", "", "Tier to update")
	handling := fs.String("handling-strategy", "", "Handling strategy: strip|generate|prompt|delegate")
	fallback := fs.String("fallback-strategy", "", "Fallback strategy")
	requiresUserInput := fs.Bool("requires-user-input", false, "Whether this secret requires user input")
	promptLabel := fs.String("prompt-label", "", "Prompt label")
	promptDescription := fs.String("prompt-description", "", "Prompt description")
	generatorTemplate := fs.String("generator-template", "", "Generator template JSON object")
	bundleHints := fs.String("bundle-hints", "", "Bundle hints JSON object")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: resources set-strategy <resource> <secret> --tier <tier> --handling-strategy <strategy>")
	}
	if *tier == "" || *handling == "" {
		return fmt.Errorf("--tier and --handling-strategy are required")
	}

	generator, err := support.ParseJSONMap(*generatorTemplate)
	if err != nil {
		return err
	}
	hints, err := support.ParseJSONMap(*bundleHints)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"tier":                *tier,
		"handling_strategy":   *handling,
		"requires_user_input": *requiresUserInput,
	}
	if *fallback != "" {
		payload["fallback_strategy"] = *fallback
	}
	if *promptLabel != "" {
		payload["prompt_label"] = *promptLabel
	}
	if *promptDescription != "" {
		payload["prompt_description"] = *promptDescription
	}
	if generator != nil {
		payload["generator_template"] = generator
	}
	if hints != nil {
		payload["bundle_hints"] = hints
	}

	resource, secret := fs.Arg(0), fs.Arg(1)
	var resp support.ResourceSecretDetail
	if err := support.RequestJSON(core, "POST", "/resources/"+resource+"/secrets/"+secret+"/strategy", nil, payload, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Secret deployment strategy updated", fmt.Sprintf("Resource/secret: %s/%s", resource, resp.SecretKey)},
		Changes:     []string{fmt.Sprintf("Tier: %s", *tier), fmt.Sprintf("Handling strategy: %s", resp.TierStrategies[*tier])},
		NextCommand: []string{support.CLIName + " resources get " + resource, support.CLIName + " deployment plan --scenario <scenario> --tier " + *tier},
	}
	return support.PrintMutation(*jsonOutput, resp, report)
}
