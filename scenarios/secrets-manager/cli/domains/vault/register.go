package vault

import (
	"fmt"

	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "vault",
		Description: "Vault coverage, validation, and provisioning",
		Subcommands: []cliapp.Command{
			{Name: "status", Aliases: []string{"list"}, NeedsAPI: true, Description: "Show vault coverage and missing secrets", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "validate", NeedsAPI: true, Description: "Validate secrets for all resources or one resource", Run: func(args []string) error { return runValidate(core, args) }},
			{Name: "provision", NeedsAPI: true, Description: "Store secrets locally and optionally sync them to Vault", Run: func(args []string) error { return runProvision(core, args) }},
		},
	}
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("vault status")
	resource := fs.String("resource", "", "Filter to one resource")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var resp support.VaultSecretsStatus
	query := support.Query("resource", *resource)
	if err := support.GetJSON(core, "/vault/secrets/status", query, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.ResourceStatuses))
	for _, item := range resp.ResourceStatuses {
		results = append(results, fmt.Sprintf("%s | %s | found %d/%d | missing %d | optional %d | checked %s",
			item.ResourceName, item.HealthStatus, item.SecretsFound, item.SecretsTotal, item.SecretsMissing, item.SecretsOptional, support.FormatTime(item.LastChecked)))
	}
	if len(resp.MissingSecrets) > 0 {
		results = append(results, "")
		for _, missing := range resp.MissingSecrets {
			results = append(results, fmt.Sprintf("Missing: %s/%s | required=%t | %s", missing.ResourceName, missing.SecretName, missing.Required, missing.Description))
		}
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Configured resources: %d/%d", resp.ConfiguredResources, resp.TotalResources),
			fmt.Sprintf("Missing secrets: %d", len(resp.MissingSecrets)),
			fmt.Sprintf("Last updated: %s", support.FormatTime(resp.LastUpdated)),
		},
		ResultsHeading: "Vault Coverage",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " vault validate", support.CLIName + " vault provision --resource <resource> --secret KEY=value"},
	}
	return support.PrintList(*jsonOutput, resp, report)
}

func runValidate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("vault validate")
	resource := fs.String("resource", "", "Validate one resource instead of the whole inventory")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body := map[string]string{}
	if *resource != "" {
		body["resource"] = *resource
	}

	var resp support.ValidationResponse
	if err := support.RequestRootJSON(core, "POST", "/secrets/validate", nil, body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.HealthSummary)+len(resp.MissingSecrets)+len(resp.InvalidSecrets))
	for _, item := range resp.HealthSummary {
		results = append(results, fmt.Sprintf("%s | valid %d/%d | missing required %d | invalid %d | last validation %s",
			item.ResourceName, item.ValidSecrets, item.TotalSecrets, item.MissingRequiredSecrets, item.InvalidSecrets, support.FormatTimePtr(item.LastValidation)))
	}
	for _, item := range resp.MissingSecrets {
		results = append(results, fmt.Sprintf("Missing validation: %s | %s", item.ID, item.ValidationStatus))
	}
	for _, item := range resp.InvalidSecrets {
		results = append(results, fmt.Sprintf("Invalid validation: %s | %s | %s", item.ID, item.ValidationStatus, item.ErrorMessage))
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Validation ID: %s", resp.ValidationID),
			fmt.Sprintf("Valid secrets: %d/%d", resp.ValidSecrets, resp.TotalSecrets),
			fmt.Sprintf("Missing validations: %d", len(resp.MissingSecrets)),
			fmt.Sprintf("Invalid validations: %d", len(resp.InvalidSecrets)),
		},
		ResultsHeading: "Validation Details",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " vault status", support.CLIName + " resources get <resource>"},
	}
	return support.PrintList(*jsonOutput, resp, report)
}

func runProvision(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("vault provision")
	resource := fs.String("resource", "", "Resource to sync into Vault (optional for local-only storage)")
	var secrets cliutil.StringList
	fs.Var(&secrets, "secret", "Secret assignment in KEY=VALUE form; repeatable")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	values, err := support.ParseKV(secrets.Values())
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return fmt.Errorf("at least one --secret KEY=VALUE is required")
	}

	payload := map[string]any{
		"resource_name": *resource,
		"secrets":       values,
	}

	var resp support.ProvisionResponse
	if err := support.RequestJSON(core, "POST", "/vault/secrets/provision", nil, payload, &resp); err != nil {
		return err
	}

	changes := []string{
		fmt.Sprintf("Local store updated: %d", resp.StoredSecrets),
		fmt.Sprintf("Vault stored: %d", resp.VaultStored),
	}
	for _, detail := range resp.Details {
		line := fmt.Sprintf("%s -> %s (%s)", detail.EnvKey, detail.VaultPath, detail.Status)
		if detail.Error != "" {
			line += " | " + detail.Error
		}
		changes = append(changes, line)
	}
	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Provision success: %t", resp.Success),
			fmt.Sprintf("Resource: %s", support.Fallback(*resource, "local-only")),
		},
		Changes:     changes,
		NextCommand: []string{support.CLIName + " vault status", support.CLIName + " vault validate" + support.OptionalResourceFlag(*resource)},
	}
	if resp.Message != "" {
		report.Result = append(report.Result, "Message: "+resp.Message)
	}
	return support.PrintMutation(*jsonOutput, resp, report)
}
