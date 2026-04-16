package codegen

import (
	"fmt"
	"os"
	"strings"

	"api-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `codegen` subcommands for the code-generator integration
// endpoints: spec, search, templates.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "codegen",
		Description: "Code-generator integration endpoints",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "spec", Description: "Fetch the code-generator spec for an API", Run: func(args []string) error { return runSpec(core, args) }},
			{Name: "search", Description: "Search APIs optimised for code generation (requires --body-file)", Run: func(args []string) error { return runSearch(core, args) }},
			{Name: "templates", Description: "List code-generator templates for a language", Run: func(args []string) error { return runTemplates(core, args) }},
		},
	}
}

func runSpec(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("codegen spec")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: codegen spec <api-id>")
	}
	id := fs.Arg(0)
	raw, err := core.Get("/codegen/apis/"+id, nil)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(raw, &resp); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("CodeGen spec for %s", id)},
		ResultsHeading: "Spec",
		Results:        support.MapRows(resp),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("codegen search")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the codegen search body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	var body interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		body = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: codegen search <query> | codegen search --body-file PATH")
		}
		body = map[string]interface{}{"query": strings.Join(fs.Args(), " ")}
	}
	raw, err := core.Request("POST", "/codegen/search", nil, body)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(raw, &resp); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{"CodeGen search results"},
		ResultsHeading: "Results",
		Results:        support.MapRows(resp),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runTemplates(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("codegen templates")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: codegen templates <language>")
	}
	lang := fs.Arg(0)
	raw, err := core.Get("/codegen/templates/"+lang, nil)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(raw, &resp); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Templates for language=%s", lang)},
		ResultsHeading: "Templates",
		Results:        support.MapRows(resp),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
