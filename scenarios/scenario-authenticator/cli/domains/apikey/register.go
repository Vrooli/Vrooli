package apikey

import (
	"fmt"
	"os"
	"strings"

	"scenario-authenticator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `apikey` subcommand group covering list/create/revoke/validate.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "apikey",
		Description: "Manage per-user API keys",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List API keys for the caller", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Description: "Create a new API key", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "revoke", Aliases: []string{"rm"}, Description: "Revoke an API key by ID", Run: func(args []string) error { return runRevoke(core, args) }},
			{Name: "validate", Description: "Validate an API key value", Run: func(args []string) error { return runValidate(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apikey list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/apikeys", nil)
	if err != nil {
		return err
	}

	var keys []support.APIKey
	if err := support.Decode(body, &keys); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("API keys: %d", len(keys))},
		ResultsHeading: "Keys",
		Results:        apiKeyRows(keys),
		RetrievalHints: []string{
			fmt.Sprintf("%s apikey create --name <label>", support.CLIName),
			fmt.Sprintf("%s apikey revoke <id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apikey create")
	name := fs.String("name", "", "Human-readable label (required)")
	rateLimit := fs.Int("rate-limit", 0, "Requests per minute (0 uses server default)")
	expiresIn := fs.Int("expires-in", 0, "Days until expiration (0 = never)")
	permissions := fs.String("permissions", "", "Comma-separated permission strings (e.g. read:own,write:own)")
	bodyFile := fs.String("body-file", "", "Optional JSON file overriding the computed payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if strings.TrimSpace(*name) == "" {
			return fmt.Errorf("usage: apikey create --name <label> [--rate-limit N] [--expires-in DAYS] [--permissions a,b] | --body-file path")
		}
		built := map[string]interface{}{"name": *name}
		if *rateLimit > 0 {
			built["rate_limit"] = *rateLimit
		}
		if *expiresIn > 0 {
			built["expires_in"] = *expiresIn
		}
		if strings.TrimSpace(*permissions) != "" {
			perms := splitCSV(*permissions)
			if len(perms) > 0 {
				built["permissions"] = perms
			}
		}
		payload = built
	}

	body, err := core.Request("POST", "/apikeys", nil, payload)
	if err != nil {
		return err
	}

	var key support.APIKey
	if err := support.Decode(body, &key); err != nil {
		return err
	}

	result := []string{
		fmt.Sprintf("ID: %s", key.ID),
		fmt.Sprintf("Name: %s", key.Name),
		fmt.Sprintf("Key: %s", key.Key),
		fmt.Sprintf("Permissions: %s", support.JoinStrings(key.Permissions, "-")),
		fmt.Sprintf("Rate limit: %d/min", key.RateLimit),
	}
	if key.ExpiresAt != nil {
		result = append(result, fmt.Sprintf("Expires: %s", support.FormatTimePtr(key.ExpiresAt)))
	}

	report := cliapp.MutationReport{
		Result:      result,
		Changes:     []string{fmt.Sprintf("API key %q created", key.Name)},
		NextCommand: []string{fmt.Sprintf("%s apikey list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runRevoke(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apikey revoke")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apikey revoke <key-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("DELETE", "/apikeys/"+id, nil, nil)
	if err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("API key %s revoked", id)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("API key %s revoked", id)},
		NextCommand: []string{fmt.Sprintf("%s apikey list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runValidate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apikey validate")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apikey validate <api-key>")
	}
	apiKey := fs.Arg(0)

	body, err := core.Request("POST", "/apikeys/validate", nil, map[string]string{"api_key": apiKey})
	if err != nil {
		return err
	}

	var resp support.APIKeyValidation
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	if !resp.Valid {
		report := cliapp.OperationalReport{
			Status: []string{"API key is invalid"},
		}
		if resp.Error != "" {
			report.Status = append(report.Status, fmt.Sprintf("Error: %s", resp.Error))
		}
		if *jsonOutput {
			return cliapp.PrintReportJSON(os.Stdout, report)
		}
		return cliapp.RenderOperationalReport(os.Stdout, report)
	}

	results := []string{
		fmt.Sprintf("User ID: %s", resp.UserID),
		fmt.Sprintf("Permissions: %s", support.JoinStrings(resp.Permissions, "-")),
		fmt.Sprintf("Rate limit: %d/min", resp.RateLimit),
	}

	report := cliapp.ListReport{
		Summary:        []string{"API key is valid"},
		ResultsHeading: "Key details",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func apiKeyRows(keys []support.APIKey) []string {
	if len(keys) == 0 {
		return []string{"No API keys"}
	}
	rows := make([]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, fmt.Sprintf("%s (%s) | perms=%s | rate=%d/min | last_used=%s | expires=%s",
			k.Name,
			support.ShortID(k.ID),
			support.JoinStrings(k.Permissions, "-"),
			k.RateLimit,
			support.FormatTimePtr(k.LastUsed),
			support.FormatTimePtr(k.ExpiresAt),
		))
	}
	return rows
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
