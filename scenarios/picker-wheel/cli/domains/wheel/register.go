package wheel

import (
	"fmt"
	"os"
	"strings"

	"picker-wheel/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `wheel` subcommand group covering list/get/create/delete.
// The API is the source of truth for wheels; this package is a thin wrapper that
// formats responses through the standard output contracts.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "wheel",
		Description: "List, inspect, create, and delete wheels",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List available wheels", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one wheel", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Description: "Create a wheel (body via --body-file)", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a wheel", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wheel list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/wheels", nil)
	if err != nil {
		return err
	}
	var wheels []support.Wheel
	if err := support.Decode(body, &wheels); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Wheels: %d", len(wheels))},
		ResultsHeading: "Wheels",
		Results:        wheelRows(wheels),
		RetrievalHints: []string{
			fmt.Sprintf("%s wheel get <wheel-id>", support.CLIName),
			fmt.Sprintf("%s spin --body-file payload.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wheel get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: wheel get <wheel-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/wheels/"+id, nil)
	if err != nil {
		return err
	}
	var wheel support.Wheel
	if err := support.Decode(body, &wheel); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", wheel.ID),
		fmt.Sprintf("Name: %s", wheel.Name),
	}
	if wheel.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", wheel.Description))
	}
	if wheel.Theme != "" {
		results = append(results, fmt.Sprintf("Theme: %s", wheel.Theme))
	}
	results = append(results, fmt.Sprintf("Times used: %d", wheel.TimesUsed))
	results = append(results, fmt.Sprintf("Created: %s", support.FormatTimeValue(wheel.CreatedAt)))
	if len(wheel.Options) > 0 {
		results = append(results, fmt.Sprintf("Options (%d):", len(wheel.Options)))
		for _, opt := range wheel.Options {
			results = append(results, "  - "+formatOption(opt))
		}
	} else {
		results = append(results, "Options: (none)")
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Wheel: %s", wheel.Name)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s spin --body-file payload.json", support.CLIName),
			fmt.Sprintf("%s wheel delete %s", support.CLIName, wheel.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wheel create")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the wheel payload (name, description, options, theme)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/wheels", nil, payload)
	if err != nil {
		return err
	}
	var wheel support.Wheel
	if err := support.Decode(body, &wheel); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Wheel created: %s", wheel.ID)},
		Changes: []string{fmt.Sprintf("Name: %s (%d options)", wheel.Name, len(wheel.Options))},
		NextCommand: []string{
			fmt.Sprintf("%s wheel get %s", support.CLIName, wheel.ID),
			fmt.Sprintf("%s wheel list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("wheel delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: wheel delete <wheel-id>")
	}
	id := fs.Arg(0)

	// DELETE returns 204 No Content on success.
	body, err := core.Request("DELETE", "/wheels/"+id, nil, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Wheel %s deleted", id)
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("Wheel %s: deleted", id)},
		NextCommand: []string{
			fmt.Sprintf("%s wheel list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func wheelRows(wheels []support.Wheel) []string {
	if len(wheels) == 0 {
		return []string{"No wheels available"}
	}
	rows := make([]string, 0, len(wheels))
	for _, w := range wheels {
		theme := w.Theme
		if theme == "" {
			theme = "-"
		}
		rows = append(rows, fmt.Sprintf("%s (%s) | theme=%s | options=%d | used=%d",
			w.Name, support.ShortID(w.ID), theme, len(w.Options), w.TimesUsed))
	}
	return rows
}

func formatOption(opt support.Option) string {
	parts := []string{opt.Label}
	if opt.Weight > 0 {
		parts = append(parts, fmt.Sprintf("weight=%g", opt.Weight))
	}
	if opt.Color != "" {
		parts = append(parts, fmt.Sprintf("color=%s", opt.Color))
	}
	return strings.Join(parts, " | ")
}
