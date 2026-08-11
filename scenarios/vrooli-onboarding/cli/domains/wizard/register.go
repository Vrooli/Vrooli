package wizard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Selection is the stable automation document. It names operator intent and
// deliberately does not mirror the internal operator-state schema.
type Selection struct {
	Scenarios         []string        `json:"scenarios"`
	OptionalResources []string        `json:"optional_resources,omitempty"`
	Resources         map[string]bool `json:"resources,omitempty"`
	Host              struct {
		Tools      []string `json:"tools,omitempty"`
		Safeguards []string `json:"safeguards,omitempty"`
	} `json:"host,omitempty"`
	HostTools      map[string]bool `json:"host_tools,omitempty"`
	HostSafeguards map[string]bool `json:"host_safeguards,omitempty"`
	OperatingMode  map[string]struct {
		AutoRestart bool `json:"auto_restart"`
	} `json:"operating_mode,omitempty"`
	Apply bool `json:"apply,omitempty"`
}

// selectionPatch is the one translation from the stable automation document
// to operator-state fields. Keeping it pure makes the interactive and
// declarative surfaces provably share the same write contract.
func selectionPatch(selection Selection) map[string]any {
	patch := map[string]any{"scenarios": map[string]any{}}
	scenarios := patch["scenarios"].(map[string]any)
	for _, name := range selection.Scenarios {
		scenarios[name] = map[string]any{"enabled": true}
	}
	if len(selection.Resources) > 0 || len(selection.OptionalResources) > 0 {
		resources := map[string]any{}
		for name, enabled := range selection.Resources {
			resources[name] = map[string]any{"enabled": enabled}
		}
		for _, name := range selection.OptionalResources {
			resources[name] = map[string]any{"enabled": true}
		}
		patch["resources"] = resources
	}
	if len(selection.Host.Tools) > 0 || len(selection.HostTools) > 0 {
		hostTools := map[string]any{}
		for _, name := range selection.Host.Tools {
			hostTools[name] = map[string]any{"opted_in": true}
		}
		for name, optedIn := range selection.HostTools {
			hostTools[name] = map[string]any{"opted_in": optedIn}
		}
		patch["host_tools"] = hostTools
	}
	if len(selection.Host.Safeguards) > 0 || len(selection.HostSafeguards) > 0 {
		hostSafeguards := map[string]any{}
		for _, name := range selection.Host.Safeguards {
			hostSafeguards[name] = map[string]any{"opted_in": true}
		}
		for name, optedIn := range selection.HostSafeguards {
			hostSafeguards[name] = map[string]any{"opted_in": optedIn}
		}
		patch["host_safeguards"] = hostSafeguards
	}
	for name, mode := range selection.OperatingMode {
		scenarios[name] = map[string]any{"enabled": true, "auto_restart": mode.AutoRestart}
	}
	return patch
}

func ensureModeMap(value map[string]struct {
	AutoRestart bool `json:"auto_restart"`
}) map[string]struct {
	AutoRestart bool `json:"auto_restart"`
} {
	if value == nil {
		return make(map[string]struct {
			AutoRestart bool `json:"auto_restart"`
		})
	}
	return value
}

func hostNames(names []string) string { return strings.Join(names, ", ") }

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "wizard", Description: "Configure an installation through the onboarding API", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "status", Description: "Show current readiness and committed state", Run: func(args []string) error { return get(core, args, "/v2/readiness") }},
		{Name: "apply", Description: "Commit a selection document and apply it", Run: func(args []string) error { return apply(core, args) }},
		{Name: "export", Description: "Export the current manifest-derived selection", Run: func(args []string) error { return exportSelection(core, args) }},
		{Name: "run", Description: "Walk the same eight capability steps used by the UI", Run: func(args []string) error { return runWizard(core, args) }},
	}}
}

func runWizard(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard run")
	interactive := fs.Bool("interactive", false, "Walk all eight onboarding steps in the terminal")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if !*interactive {
		return get(core, args, "/v2/scenarios")
	}
	reader := bufio.NewReader(os.Stdin)
	read := func(step int, prompt string) (string, error) {
		if _, err := fmt.Fprintf(os.Stdout, "Step %d — %s\n> ", step, prompt); err != nil {
			return "", err
		}
		line, err := reader.ReadString('\n')
		return strings.TrimSpace(line), err
	}
	scenarios, err := core.Get("/v2/scenarios", nil)
	if err != nil {
		return err
	}
	var scenarioResponse struct {
		Scenarios []struct {
			Name string `json:"name"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(scenarios, &scenarioResponse); err != nil {
		return fmt.Errorf("decode scenarios: %w", err)
	}
	selection := Selection{}
	selectedLine, err := read(1, "select scenario names (comma separated)")
	if err != nil {
		return err
	}
	known := map[string]bool{}
	names := make([]string, 0, len(scenarioResponse.Scenarios))
	for _, scenario := range scenarioResponse.Scenarios {
		known[scenario.Name] = true
		names = append(names, scenario.Name)
	}
	_, _ = fmt.Fprintln(os.Stdout, "Available scenarios:", strings.Join(names, ", "))
	for _, name := range strings.Split(selectedLine, ",") {
		name = strings.TrimSpace(name)
		if name != "" && known[name] {
			selection.Scenarios = append(selection.Scenarios, name)
		}
	}
	resourcesBody, err := core.Get("/v2/resources", nil)
	if err != nil {
		return err
	}
	var resourceResponse struct {
		Optional []struct {
			Name string `json:"name"`
		} `json:"optional"`
		Standalone []struct {
			Name string `json:"name"`
		} `json:"standalone"`
	}
	if err := json.Unmarshal(resourcesBody, &resourceResponse); err != nil {
		return fmt.Errorf("decode resources: %w", err)
	}
	optionalNames := make([]string, 0, len(resourceResponse.Optional)+len(resourceResponse.Standalone))
	for _, resource := range resourceResponse.Optional {
		optionalNames = append(optionalNames, resource.Name)
	}
	for _, resource := range resourceResponse.Standalone {
		optionalNames = append(optionalNames, resource.Name)
	}
	sort.Strings(optionalNames)
	_, _ = fmt.Fprintln(os.Stdout, "Optional/standalone resources:", strings.Join(optionalNames, ", "))
	resourceLine, err := read(2, "select optional or standalone resources (comma separated; press enter for none)")
	if err != nil {
		return err
	}
	knownResources := map[string]bool{}
	for _, name := range optionalNames {
		knownResources[name] = true
	}
	for _, name := range strings.Split(resourceLine, ",") {
		name = strings.TrimSpace(name)
		if name != "" && knownResources[name] {
			selection.OptionalResources = append(selection.OptionalResources, name)
		}
	}
	if _, err := read(3, "provision credentials separately with credentials provision; press enter to continue"); err != nil {
		return err
	}
	if _, err := read(4, "integration binding is deferred; press enter to continue"); err != nil {
		return err
	}
	hostBody, err := core.Get("/v2/host-requirements", nil)
	if err != nil {
		return err
	}
	var hostResponse struct {
		Tools []struct {
			Name     string `json:"name"`
			Required bool   `json:"required"`
		} `json:"tools"`
		Safeguards []struct {
			Name     string `json:"name"`
			Required bool   `json:"required"`
		} `json:"safeguards"`
	}
	if err := json.Unmarshal(hostBody, &hostResponse); err != nil {
		return fmt.Errorf("decode host requirements: %w", err)
	}
	toolNames := make([]string, 0, len(hostResponse.Tools))
	for _, item := range hostResponse.Tools {
		toolNames = append(toolNames, item.Name)
	}
	_, _ = fmt.Fprintln(os.Stdout, "Host tools:", hostNames(toolNames))
	toolLine, err := read(5, "select optional host tools (comma separated; required tools are automatic)")
	if err != nil {
		return err
	}
	for _, name := range strings.Split(toolLine, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			selection.Host.Tools = append(selection.Host.Tools, name)
		}
	}
	safeguardNames := make([]string, 0, len(hostResponse.Safeguards))
	for _, item := range hostResponse.Safeguards {
		safeguardNames = append(safeguardNames, item.Name)
	}
	_, _ = fmt.Fprintln(os.Stdout, "Host safeguards:", hostNames(safeguardNames))
	safeguardLine, err := read(5, "select optional host safeguards (comma separated; press enter for none)")
	if err != nil {
		return err
	}
	for _, name := range strings.Split(safeguardLine, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			selection.Host.Safeguards = append(selection.Host.Safeguards, name)
		}
	}
	modeLine, err := read(6, "choose operating mode: enter scenario names for auto-restart (comma separated)")
	if err != nil {
		return err
	}
	for _, name := range strings.Split(modeLine, ",") {
		name = strings.TrimSpace(name)
		if name != "" && known[name] {
			selection.OperatingMode = ensureModeMap(selection.OperatingMode)
			selection.OperatingMode[name] = struct {
				AutoRestart bool `json:"auto_restart"`
			}{AutoRestart: true}
		}
	}
	if _, err := read(7, "review and apply the selection? press enter to continue"); err != nil {
		return err
	}
	selection.Apply = true
	patch := selectionPatch(selection)
	patchBody, _ := json.Marshal(patch)
	if _, err := core.Request("PATCH", "/v2/operator-state", nil, patchBody); err != nil {
		return err
	}
	applyResult, err := core.Request("POST", "/v2/apply", nil, []byte("{}"))
	if err != nil {
		return err
	}
	if _, err := read(8, "validation is complete; press enter to print readiness"); err != nil {
		return err
	}
	readinessResult, err := core.Get("/v2/readiness", nil)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(applyResult), string(readinessResult))
	return err
}

func get(core *cliapp.ScenarioApp, args []string, path string) error {
	fs := support.NewFlagSet("wizard")
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
		return fmt.Errorf("decode wizard response: %w", err)
	}
	pretty, _ := json.MarshalIndent(value, "", "  ")
	_, err = fmt.Fprintln(os.Stdout, string(pretty))
	return err
}

func apply(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard apply")
	selectionPath := fs.String("selection", "", "Path to a selection document")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*selectionPath) == "" {
		return fmt.Errorf("--selection is required")
	}
	body, err := support.ReadJSONFile(*selectionPath, true)
	if err != nil {
		return err
	}
	var selection Selection
	if err := json.Unmarshal(body, &selection); err != nil {
		return fmt.Errorf("decode selection: %w", err)
	}
	patch := selectionPatch(selection)
	patchBody, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	if _, err := core.Request("PATCH", "/v2/operator-state", nil, patchBody); err != nil {
		return err
	}
	result, err := core.Request("POST", "/v2/apply", nil, []byte("{}"))
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(result, '\n'))
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Selection committed and applied"}, NextCommand: []string{support.CLIName + " wizard status"}})
}

func exportSelection(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wizard export")
	outputPath := fs.String("output", "", "Output selection document path")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*outputPath) == "" {
		return fmt.Errorf("--output is required")
	}
	body, err := core.Get("/v2/scenarios", nil)
	if err != nil {
		return err
	}
	var response struct {
		Scenarios []struct {
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("decode scenarios: %w", err)
	}
	selection := Selection{Scenarios: []string{}}
	for _, scenario := range response.Scenarios {
		if scenario.Enabled {
			selection.Scenarios = append(selection.Scenarios, scenario.Name)
		}
	}
	data, _ := json.MarshalIndent(selection, "", "  ")
	if err := os.WriteFile(*outputPath, append(data, '\n'), 0o600); err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, "Selection exported to", *outputPath)
	return err
}
