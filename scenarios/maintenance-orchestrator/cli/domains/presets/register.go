package presets

import (
	"fmt"
	"os"
	"strings"

	"maintenance-orchestrator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `preset` subcommand group covering listing, active
// queries, apply (toggle), and create operations for maintenance presets.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "preset",
		Description: "Manage maintenance-orchestrator presets",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List available presets", Run: func(args []string) error { return runList(core, args) }},
			{Name: "active", Description: "List presets currently active", Run: func(args []string) error { return runActive(core, args) }},
			{Name: "apply", Description: "Toggle a preset (activate/deactivate)", Run: func(args []string) error { return runApply(core, args) }},
			{Name: "create", Description: "Create a new preset", Run: func(args []string) error { return runCreate(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("preset list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/presets", nil)
	if err != nil {
		return err
	}
	var resp support.PresetsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	rows := make([]string, 0, len(resp.Presets))
	for _, p := range resp.Presets {
		state := ""
		if p.IsActive {
			state = " [ACTIVE]"
		}
		if p.IsDefault {
			state += " [DEFAULT]"
		}
		rows = append(rows, fmt.Sprintf("%s%s - %s", p.ID, state, p.Description))
	}
	if len(rows) == 0 {
		rows = []string{"(no presets)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Presets: %d", len(resp.Presets))},
		ResultsHeading: "Presets",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s preset apply <preset-id>", support.CLIName),
			fmt.Sprintf("%s preset active", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runActive(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("preset active")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/presets/active", nil)
	if err != nil {
		return err
	}
	var resp support.ActivePresetsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	rows := make([]string, 0, len(resp.ActivePresets))
	for _, p := range resp.ActivePresets {
		rows = append(rows, fmt.Sprintf("%s - %s", p.ID, p.Description))
	}
	if len(rows) == 0 {
		rows = []string{"(no presets active)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Active presets: %d", len(resp.ActivePresets))},
		ResultsHeading: "Active presets",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s preset list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runApply(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("preset apply")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: preset apply <preset-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/presets/"+id+"/apply", nil, nil)
	if err != nil {
		return err
	}
	var resp support.ApplyPresetResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{
		fmt.Sprintf("Activated: %d scenarios", len(resp.Activated)),
		fmt.Sprintf("Deactivated: %d scenarios", len(resp.Deactivated)),
	}
	if len(resp.Activated) > 0 {
		changes = append(changes, "  activated: "+strings.Join(resp.Activated, ", "))
	}
	if len(resp.Deactivated) > 0 {
		changes = append(changes, "  deactivated: "+strings.Join(resp.Deactivated, ", "))
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Applied preset %s", id)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s orchestrator overview", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("preset create")
	name := fs.String("name", "", "Preset name (required)")
	description := fs.String("description", "", "Preset description")
	fromCurrent := fs.Bool("from-current-state", false, "Capture current scenario states into the preset")
	bodyFile := fs.String("body-file", "", "Path to JSON file containing {\"states\": {...}, \"tags\": [...]} payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("--name is required")
	}

	payload := map[string]interface{}{
		"name":        *name,
		"description": *description,
	}
	if *fromCurrent {
		payload["fromCurrentState"] = true
	} else {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return fmt.Errorf("--body-file is required unless --from-current-state is set: %w", err)
		}
		var parsed struct {
			States map[string]bool `json:"states"`
			Tags   []string        `json:"tags"`
		}
		if err := support.Decode(raw, &parsed); err != nil {
			return fmt.Errorf("parse body-file: %w", err)
		}
		if len(parsed.States) == 0 {
			return fmt.Errorf("body-file must contain a non-empty \"states\" object")
		}
		payload["states"] = parsed.States
		if len(parsed.Tags) > 0 {
			payload["tags"] = parsed.Tags
		}
	}

	body, err := core.Request("POST", "/presets", nil, payload)
	if err != nil {
		return err
	}

	var resp struct {
		Preset support.Preset `json:"preset"`
	}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{fmt.Sprintf("ID: %s", resp.Preset.ID)}
	if len(resp.Preset.States) > 0 {
		changes = append(changes, fmt.Sprintf("Scenarios captured: %d", len(resp.Preset.States)))
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created preset %s", resp.Preset.Name)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s preset apply %s", support.CLIName, resp.Preset.ID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
