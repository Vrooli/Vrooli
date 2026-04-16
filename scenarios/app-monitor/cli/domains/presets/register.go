package presets

import (
	"fmt"
	"os"
	"strings"

	"app-monitor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `preset` subcommand group for workspace preset CRUD.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "preset",
		Description: "Manage workspace presets",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List presets", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one preset", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Description: "Create a preset", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Description: "Update a preset", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Description: "Delete a preset", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("preset list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/workspace/presets", nil)
	if err != nil {
		return err
	}
	var presets []support.Preset
	if err := support.Decode(body, &presets); err != nil {
		return err
	}

	rows := make([]string, 0, len(presets))
	for _, p := range presets {
		rows = append(rows, fmt.Sprintf("%s | %s | %s", support.ShortID(p.ID), p.Name, p.Description))
	}
	if len(rows) == 0 {
		rows = []string{"(no presets)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Presets: %d", len(presets))},
		ResultsHeading: "Presets",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s preset get <preset-id>", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("preset get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: preset get <preset-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/workspace/presets/"+id, nil)
	if err != nil {
		return err
	}
	var preset support.Preset
	if err := support.Decode(body, &preset); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", preset.ID),
		fmt.Sprintf("Name: %s", preset.Name),
	}
	if preset.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", preset.Description))
	}
	if preset.CreatedAt != nil {
		results = append(results, fmt.Sprintf("Created: %s", support.FormatTimeValue(*preset.CreatedAt)))
	}
	if len(preset.Layout) > 0 {
		results = append(results, fmt.Sprintf("Layout: %s", string(preset.Layout)))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Preset: %s", preset.Name)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s preset update %s --layout-file ./layout.json", support.CLIName, preset.ID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("preset create")
	name := fs.String("name", "", "Preset name (required)")
	description := fs.String("description", "", "Preset description")
	layoutFile := fs.String("layout-file", "", "Path to JSON layout file (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("--name is required")
	}

	layout, err := support.ReadJSONFile(*layoutFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/workspace/presets", nil, map[string]interface{}{
		"name":        *name,
		"description": *description,
		"layout":      layout,
	})
	if err != nil {
		return err
	}
	var preset support.Preset
	if err := support.Decode(body, &preset); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created preset %s", preset.Name)},
		Changes:     []string{fmt.Sprintf("ID: %s", preset.ID)},
		NextCommand: []string{fmt.Sprintf("%s preset get %s", support.CLIName, preset.ID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("preset update")
	name := fs.String("name", "", "New preset name")
	description := fs.String("description", "", "New description")
	layoutFile := fs.String("layout-file", "", "Path to JSON layout file")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: preset update <preset-id> [--name ...] [--description ...] [--layout-file ...]")
	}
	id := fs.Arg(0)

	payload := map[string]interface{}{}
	if strings.TrimSpace(*name) != "" {
		payload["name"] = *name
	}
	if strings.TrimSpace(*description) != "" {
		payload["description"] = *description
	}
	if strings.TrimSpace(*layoutFile) != "" {
		layout, err := support.ReadJSONFile(*layoutFile, true)
		if err != nil {
			return err
		}
		payload["layout"] = layout
	}
	if len(payload) == 0 {
		return fmt.Errorf("no fields to update — provide --name, --description, or --layout-file")
	}

	if _, err := core.Request("PUT", "/workspace/presets/"+id, nil, payload); err != nil {
		return err
	}

	changes := make([]string, 0, len(payload))
	for k, v := range payload {
		if k == "layout" {
			changes = append(changes, "layout: (updated)")
			continue
		}
		changes = append(changes, fmt.Sprintf("%s: %s", k, support.RenderValue(v)))
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated preset %s", id)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s preset get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("preset delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: preset delete <preset-id>")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/workspace/presets/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted preset %s", id)},
		NextCommand: []string{fmt.Sprintf("%s preset list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
