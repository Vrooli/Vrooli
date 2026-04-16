package component

import (
	"fmt"
	"os"
	"strings"

	"react-component-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `component` subcommand group covering CRUD, search,
// content, and versions for components in the library.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "component",
		Description: "Manage React components in the library",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List components", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one component", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "search", Description: "Search components", Run: func(args []string) error { return runSearch(core, args) }},
			{Name: "create", Description: "Create a component", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Description: "Update a component", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "content-get", Description: "Show a component's source content", Run: func(args []string) error { return runContentGet(core, args) }},
			{Name: "content-update", Description: "Update a component's source content", Run: func(args []string) error { return runContentUpdate(core, args) }},
			{Name: "versions", Description: "List versions of a component", Run: func(args []string) error { return runVersions(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("component list")
	search := fs.String("search", "", "Filter by display name or description substring")
	category := fs.String("category", "", "Filter by category")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"search":   *search,
		"category": *category,
	})
	body, err := core.Get("/components", query)
	if err != nil {
		return err
	}
	var components []support.Component
	if err := support.Decode(body, &components); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Components: %d", len(components))},
		ResultsHeading: "Components",
		Results:        componentRows(components),
		RetrievalHints: []string{
			fmt.Sprintf("%s component get <component-id>", support.CLIName),
			fmt.Sprintf("%s component search --query <text>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("component get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: component get <component-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/components/"+id, nil)
	if err != nil {
		return err
	}
	var c support.Component
	if err := support.Decode(body, &c); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", c.ID),
		fmt.Sprintf("Library: %s", c.LibraryID),
		fmt.Sprintf("Name: %s", c.DisplayName),
		fmt.Sprintf("Version: %s", c.Version),
	}
	if c.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", c.Description))
	}
	if c.FilePath != "" {
		results = append(results, fmt.Sprintf("File: %s", c.FilePath))
	}
	if c.SourcePath != "" {
		results = append(results, fmt.Sprintf("Source: %s", c.SourcePath))
	}
	if c.Category != nil && strings.TrimSpace(*c.Category) != "" {
		results = append(results, fmt.Sprintf("Category: %s", *c.Category))
	}
	if len(c.Tags) > 0 {
		results = append(results, fmt.Sprintf("Tags: %s", strings.Join(c.Tags, ", ")))
	}
	if len(c.TechStack) > 0 {
		results = append(results, fmt.Sprintf("Tech stack: %s", strings.Join(c.TechStack, ", ")))
	}
	results = append(results,
		fmt.Sprintf("Created: %s", support.FormatTimeValue(c.CreatedAt)),
		fmt.Sprintf("Updated: %s", support.FormatTimeValue(c.UpdatedAt)),
	)

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Component: %s", c.DisplayName)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s component content-get %s", support.CLIName, c.ID),
			fmt.Sprintf("%s component versions %s", support.CLIName, c.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("component search")
	query := fs.String("query", "", "Free-text search query (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*query) == "" {
		return fmt.Errorf("--query is required")
	}

	body, err := core.Get("/components/search", support.BuildQuery(map[string]string{"query": *query}))
	if err != nil {
		return err
	}
	var components []support.Component
	if err := support.Decode(body, &components); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Matches for %q: %d", *query, len(components))},
		ResultsHeading: "Components",
		Results:        componentRows(components),
		RetrievalHints: []string{fmt.Sprintf("%s component get <component-id>", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("component create")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full component payload (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/components", nil, payload)
	if err != nil {
		return err
	}
	var c support.Component
	if err := support.Decode(body, &c); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created component %s", c.DisplayName)},
		Changes:     []string{fmt.Sprintf("ID: %s", c.ID), fmt.Sprintf("Version: %s", c.Version)},
		NextCommand: []string{fmt.Sprintf("%s component get %s", support.CLIName, c.ID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("component update")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the patch payload (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: component update <component-id> --body-file <path>")
	}
	id := fs.Arg(0)

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("PUT", "/components/"+id, nil, payload)
	if err != nil {
		return err
	}
	var c support.Component
	if err := support.Decode(body, &c); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated component %s", c.DisplayName)},
		Changes:     []string{fmt.Sprintf("ID: %s", c.ID), fmt.Sprintf("Updated: %s", support.FormatTimeValue(c.UpdatedAt))},
		NextCommand: []string{fmt.Sprintf("%s component get %s", support.CLIName, c.ID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runContentGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("component content-get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: component content-get <component-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/components/"+id+"/content", nil)
	if err != nil {
		return err
	}
	var content support.ComponentContent
	if err := support.Decode(body, &content); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Content for component %s", id)},
		ResultsHeading: "Source",
		Results:        []string{content.Content},
		RetrievalHints: []string{fmt.Sprintf("%s component content-update %s --body-file ./content.json", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runContentUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("component content-update")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the content payload (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: component content-update <component-id> --body-file <path>")
	}
	id := fs.Arg(0)

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	if _, err := core.Request("PUT", "/components/"+id+"/content", nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated content for component %s", id)},
		NextCommand: []string{fmt.Sprintf("%s component content-get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runVersions(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("component versions")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: component versions <component-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/components/"+id+"/versions", nil)
	if err != nil {
		return err
	}
	var versions []support.ComponentVersion
	if err := support.Decode(body, &versions); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Versions for %s: %d", id, len(versions))},
		ResultsHeading: "Versions",
		Results:        versionRows(versions),
		RetrievalHints: []string{fmt.Sprintf("%s component get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func componentRows(components []support.Component) []string {
	if len(components) == 0 {
		return []string{"(no components)"}
	}
	rows := make([]string, 0, len(components))
	for _, c := range components {
		category := ""
		if c.Category != nil && strings.TrimSpace(*c.Category) != "" {
			category = *c.Category
		}
		rows = append(rows, fmt.Sprintf("%s | %s | v%s | category=%s",
			support.ShortID(c.ID), c.DisplayName, c.Version, category))
	}
	return rows
}

func versionRows(versions []support.ComponentVersion) []string {
	if len(versions) == 0 {
		return []string{"(no versions)"}
	}
	rows := make([]string, 0, len(versions))
	for _, v := range versions {
		changelog := ""
		if v.Changelog != nil {
			changelog = strings.TrimSpace(*v.Changelog)
		}
		rows = append(rows, fmt.Sprintf("%s | v%s | %s | %s",
			support.ShortID(v.ID), v.Version, support.FormatTimeValue(v.CreatedAt), changelog))
	}
	return rows
}
