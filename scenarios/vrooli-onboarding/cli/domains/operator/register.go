package operator

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes the V2 onboarding control plane. It owns no state: the API
// persists operator decisions and derives scenarios/readiness from manifests.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "operator", Description: "Inspect and commit V2 operator state", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "show", Description: "Show persisted operator state", Run: func(args []string) error { return runShow(core, args) }},
		{Name: "apply", Description: "Atomically apply operator state from --body-file", Run: func(args []string) error { return runApply(core, args) }},
		{Name: "set-safeguard-config", Description: "Set one typed safeguard config value", Run: func(args []string) error { return runSetSafeguardConfig(core, args) }},
		{Name: "scenarios", Description: "Show manifest-derived scenario choices", Run: func(args []string) error { return runGet(core, args, "/v2/scenarios") }},
		{Name: "readiness", Description: "Show metadata-safe composed readiness", Run: func(args []string) error { return runGet(core, args, "/v2/readiness") }},
	}}
}

func runShow(core *cliapp.ScenarioApp, args []string) error {
	return runGet(core, args, "/operator-state")
}

func runGet(core *cliapp.ScenarioApp, args []string, path string) error {
	fs := support.NewFlagSet("operator")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(body, '\n'))
		return err
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return fmt.Errorf("decode operator response: %w", err)
	}
	encoded, _ := json.MarshalIndent(value, "", "  ")
	_, err = fmt.Fprintln(os.Stdout, string(encoded))
	return err
}

func runApply(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("operator apply")
	bodyFile := fs.String("body-file", "", "Path to an operator-state JSON document")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	response, err := core.Request("PUT", "/operator-state", nil, body)
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(response, '\n'))
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Operator state committed atomically"}, NextCommand: []string{support.CLIName + " operator readiness", support.CLIName + " operator scenarios"}})
}

func runSetSafeguardConfig(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("operator set-safeguard-config")
	name := fs.String("name", "", "Safeguard name")
	key := fs.String("key", "", "Config key")
	valueJSON := fs.String("value-json", "", "JSON-encoded config value")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*key) == "" || strings.TrimSpace(*valueJSON) == "" {
		return fmt.Errorf("--name, --key, and --value-json are required")
	}
	var value any
	if err := json.Unmarshal([]byte(*valueJSON), &value); err != nil {
		return fmt.Errorf("parse --value-json: %w", err)
	}

	body, err := core.Get("/operator-state", nil)
	if err != nil {
		return err
	}
	var state map[string]any
	if err := json.Unmarshal(body, &state); err != nil {
		return fmt.Errorf("decode operator state: %w", err)
	}
	hostSafeguards, ok := state["host_safeguards"].(map[string]any)
	if !ok {
		hostSafeguards = map[string]any{}
		state["host_safeguards"] = hostSafeguards
	}
	entry, ok := hostSafeguards[*name].(map[string]any)
	if !ok {
		entry = map[string]any{}
		hostSafeguards[*name] = entry
	}
	config, ok := entry["config"].(map[string]any)
	if !ok {
		config = map[string]any{}
		entry["config"] = config
	}
	config[*key] = value

	requestBody, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode operator state: %w", err)
	}
	response, err := core.Request("PUT", "/operator-state", nil, requestBody)
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(response, '\n'))
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Safeguard config committed atomically"}, NextCommand: []string{support.CLIName + " operator show"}})
}
