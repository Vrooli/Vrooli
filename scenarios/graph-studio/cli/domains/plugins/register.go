package plugins

import (
	"fmt"
	"os"

	"graph-studio/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `graph-studio plugins` as a flat command. Plugins is a
// single, read-only surface in the API (`GET /api/v1/plugins`). The optional
// --category flag mirrors the query parameter the API already supports.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Plugins",
		Commands: []cliapp.Command{
			{
				Name:        "plugins",
				Description: "List available graph plugins",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runList(core, args) },
			},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("plugins")
	category := fs.String("category", "", "Filter plugins by category")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{"category": *category})
	body, err := core.Get("/plugins", query)
	if err != nil {
		return err
	}
	var resp support.ListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	var plugins []support.Plugin
	if len(resp.Data) > 0 {
		if err := support.Decode(resp.Data, &plugins); err != nil {
			return err
		}
	}

	summary := []string{fmt.Sprintf("Plugins: %d", resp.Total)}
	if *category != "" {
		summary = append(summary, fmt.Sprintf("Category filter: %s", *category))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Plugins",
		Results:        pluginRows(plugins),
		RetrievalHints: []string{
			fmt.Sprintf("%s plugins --category mind-maps", support.CLIName),
			fmt.Sprintf("%s graphs list --type <plugin-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func pluginRows(plugins []support.Plugin) []string {
	if len(plugins) == 0 {
		return []string{"(no plugins registered)"}
	}
	rows := make([]string, 0, len(plugins))
	for _, p := range plugins {
		enabled := "disabled"
		if p.Enabled {
			enabled = "enabled"
		}
		row := fmt.Sprintf("%s | category=%s | %s", p.ID, p.Category, enabled)
		if p.Name != "" && p.Name != p.ID {
			row = fmt.Sprintf("%s (%s) | category=%s | %s", p.ID, p.Name, p.Category, enabled)
		}
		if p.Description != "" {
			row += " | " + p.Description
		}
		rows = append(rows, row)
	}
	return rows
}
