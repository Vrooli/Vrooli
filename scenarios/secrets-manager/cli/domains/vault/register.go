package vault

import (
	"fmt"
	"net/url"

	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

type statusResponse struct {
	VaultID          string `json:"vault_id"`
	WorkspaceID      string `json:"workspace_id"`
	Status           string `json:"status"`
	KeyAvailable     bool   `json:"key_available"`
	SupportsRecovery bool   `json:"supports_recovery"`
}

type item struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Name     string   `json:"name"`
	Username string   `json:"username,omitempty"`
	URI      string   `json:"uri,omitempty"`
	Tags     []string `json:"tags"`
	Revision int      `json:"revision"`
	Trashed  bool     `json:"trashed"`
}

type listResponse struct {
	Items []item `json:"items"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "vault",
		Description: "Metadata-safe password-manager vault operations",
		Subcommands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Show lock and key availability without values", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "items", NeedsAPI: true, Description: "List encrypted vault item metadata", Run: func(args []string) error { return runItems(core, args) }},
			{Name: "unlock", NeedsAPI: true, Description: "Unlock a vault through the configured authority", Run: func(args []string) error { return runState(core, args, "unlock") }},
			{Name: "lock", NeedsAPI: true, Description: "Lock a vault", Run: func(args []string) error { return runState(core, args, "lock") }},
		},
	}
}

func parseVaultArgs(name string, args []string) (*supportFlagValues, error) {
	values := &supportFlagValues{vault: "personal"}
	fs := support.NewFlagSet(name)
	fs.StringVar(&values.vault, "vault", values.vault, "Vault identifier")
	values.json, values.format = support.JSONFlags(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return nil, err
	}
	return values, nil
}

type supportFlagValues struct {
	vault  string
	json   *bool
	format *string
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	values, err := parseVaultArgs("vault status", args)
	if err != nil {
		return err
	}
	var response statusResponse
	if err := support.GetJSON(core, "/vaults/"+url.PathEscape(values.vault)+"/status", nil, &response); err != nil {
		return err
	}
	return support.PrintOperational(*values.json, response, cliapp.OperationalReport{Status: []string{fmt.Sprintf("Vault: %s", response.VaultID), fmt.Sprintf("Status: %s", response.Status), fmt.Sprintf("Key available: %t", response.KeyAvailable), fmt.Sprintf("Recovery supported: %t", response.SupportsRecovery)}})
}

func runItems(core *cliapp.ScenarioApp, args []string) error {
	values, err := parseVaultArgs("vault items", args)
	if err != nil {
		return err
	}
	var response listResponse
	if err := support.GetJSON(core, "/vaults/"+url.PathEscape(values.vault)+"/items", nil, &response); err != nil {
		return err
	}
	results := make([]string, 0, len(response.Items))
	for _, entry := range response.Items {
		results = append(results, fmt.Sprintf("%s | %s | %s | revision=%d", entry.Name, entry.Type, entry.URI, entry.Revision))
	}
	return support.PrintList(*values.json, response, cliapp.ListReport{Summary: []string{fmt.Sprintf("Vault items: %d", len(response.Items)), "Secret fields are never returned by this command"}, ResultsHeading: "Vault Inventory", Results: results})
}

func runState(core *cliapp.ScenarioApp, args []string, state string) error {
	values, err := parseVaultArgs("vault "+state, args)
	if err != nil {
		return err
	}
	var response map[string]any
	if err := support.RequestJSON(core, "POST", "/vaults/"+url.PathEscape(values.vault)+"/"+state, nil, map[string]any{}, &response); err != nil {
		return err
	}
	return support.PrintMutation(*values.json, response, cliapp.MutationReport{Result: []string{fmt.Sprintf("Vault %s is now %s", values.vault, state)}})
}
