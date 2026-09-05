package control

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// CommandGroups exposes the flat control-plane commands whose names are part
// of the onboarding automation contract.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		{Title: "Selection", Commands: []cliapp.Command{{Name: "closure", Description: "Show the transitive selection closure", NeedsAPI: true, Run: func(args []string) error { return support.GetJSON(core, "onboarding control", args, "/v2/closure") }}}},
		{Title: "Apply", Commands: []cliapp.Command{{Name: "apply", Description: "Apply committed onboarding state; wizard commit applies a selection document first", NeedsAPI: true, Run: func(args []string) error { return apply(core, args) }}}},
	}
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		{Name: "scenarios", Description: "Inspect manifest-derived scenario choices", NeedsAPI: true, Subcommands: []cliapp.Command{{Name: "list", Description: "List scenarios and their dependencies", Run: func(args []string) error { return support.GetJSON(core, "onboarding control", args, "/v2/scenarios") }}}},
		{Name: "union", Description: "Export the deployment union for the current selection", NeedsAPI: true, Subcommands: []cliapp.Command{{Name: "export", Description: "Write the deployment union JSON", Run: func(args []string) error { return exportUnion(core, args) }}}},
		{Name: "host", Description: "Inspect host tools and safeguards", NeedsAPI: true, Subcommands: []cliapp.Command{
			{Name: "list", Description: "List host requirements", Run: func(args []string) error {
				return support.GetJSON(core, "onboarding control", args, "/v2/host-requirements")
			}},
			{Name: "set-config", Description: "Set one safeguard configuration value", Run: func(args []string) error { return setConfig(core, args) }},
			{Name: "set-recipient", Description: "Set the recipient subject this host's notifications go to", Run: func(args []string) error { return setRecipient(core, args) }},
		}},
	}
}

func apply(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apply")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Request("POST", "/v2/apply", nil, []byte("{}"))
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(body, '\n'))
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Onboarding selection applied"}, NextCommand: []string{support.CLIName + " readiness"}})
}

func exportUnion(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("union export")
	output := fs.String("output", "", "Output JSON path")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*output) == "" {
		return fmt.Errorf("--output is required")
	}
	body, err := core.Get("/v2/union", nil)
	if err != nil {
		return err
	}
	if err := support.WriteOutput(*output, append(body, '\n')); err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, "Deployment union exported to", *output)
	return err
}

func setConfig(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("host set-config")
	name := fs.String("name", "", "Safeguard name")
	key := fs.String("key", "", "Configuration key")
	value := fs.String("value-json", "", "JSON-encoded configuration value")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*key) == "" || strings.TrimSpace(*value) == "" {
		return fmt.Errorf("--name, --key, and --value-json are required")
	}
	var decoded any
	if err := json.Unmarshal([]byte(*value), &decoded); err != nil {
		return fmt.Errorf("parse --value-json: %w", err)
	}
	body, _ := json.Marshal(map[string]any{"host_safeguards": map[string]any{*name: map[string]any{"config": map[string]any{*key: decoded}}}})
	if _, err := core.Request("PATCH", "/v2/operator-state", nil, body); err != nil {
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Host safeguard configuration committed"}, NextCommand: []string{support.CLIName + " host list"}})
}

func setRecipient(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("host set-recipient")
	subject := fs.String("subject", "", "Recipient subject registered with notification-hub")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*subject) == "" {
		return fmt.Errorf("--subject is required")
	}
	body, _ := json.Marshal(map[string]any{"notifications": map[string]any{"recipient": strings.TrimSpace(*subject)}})
	if _, err := core.Request("PATCH", "/v2/operator-state", nil, body); err != nil {
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Notification recipient committed to operator state"}, NextCommand: []string{"notification-hub recipients address-upsert --help"}})
}
