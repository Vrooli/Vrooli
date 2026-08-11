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
		{Name: "patch", Description: "Atomically apply an RFC 7386 operator-state merge patch from --body-file", Run: func(args []string) error { return runPatch(core, args) }},
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

func runPatch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("operator patch")
	bodyFile := fs.String("body-file", "", "Path to an RFC 7386 operator-state merge patch")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	response, err := core.Request("PATCH", "/v2/operator-state", nil, body)
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

	patchBody, err := json.Marshal(map[string]any{"host_safeguards": map[string]any{*name: map[string]any{"config": map[string]any{*key: value}}}})
	if err != nil {
		return fmt.Errorf("encode safeguard patch: %w", err)
	}
	response, err := core.Request("PATCH", "/v2/operator-state", nil, patchBody)
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(response, '\n'))
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Safeguard config committed atomically"}, NextCommand: []string{support.CLIName + " operator show"}})
}
