package catalog

import (
	"fmt"
	"os"

	"api-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `catalog` subcommands for browsing metadata that's global
// to the library: categories, tags, and configured APIs.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "catalog",
		Description: "Browse global catalog metadata",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "categories", Description: "List categories with counts", Run: func(args []string) error { return runCategories(core, args) }},
			{Name: "tags", Description: "List tags with counts", Run: func(args []string) error { return runTags(core, args) }},
			{Name: "configured", Description: "List APIs with configured credentials", Run: func(args []string) error { return runConfigured(core, args) }},
		},
	}
}

func runCategories(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("catalog categories")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	raw, err := core.Get("/categories", nil)
	if err != nil {
		return err
	}
	var cats []support.CategoryCount
	if err := support.Decode(raw, &cats); err != nil {
		return err
	}
	rows := make([]string, 0, len(cats))
	for _, c := range cats {
		rows = append(rows, fmt.Sprintf("%s -- %d", c.Category, c.Count))
	}
	if len(rows) == 0 {
		rows = []string{"(no categories)"}
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Categories: %d", len(cats))},
		ResultsHeading: "Categories",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runTags(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("catalog tags")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	raw, err := core.Get("/tags", nil)
	if err != nil {
		return err
	}
	var tags []support.TagCount
	if err := support.Decode(raw, &tags); err != nil {
		return err
	}
	rows := make([]string, 0, len(tags))
	for _, t := range tags {
		rows = append(rows, fmt.Sprintf("%s -- %d", t.Tag, t.Count))
	}
	if len(rows) == 0 {
		rows = []string{"(no tags)"}
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Tags: %d", len(tags))},
		ResultsHeading: "Tags",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runConfigured(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("catalog configured")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	raw, err := core.Get("/configured", nil)
	if err != nil {
		return err
	}
	var list []support.ConfiguredAPI
	if err := support.Decode(raw, &list); err != nil {
		return err
	}
	rows := make([]string, 0, len(list))
	for _, c := range list {
		ts := ""
		if c.ConfigurationDate != nil {
			ts = support.FormatTimeValue(*c.ConfigurationDate)
		}
		rows = append(rows, fmt.Sprintf("%s (%s) | env=%s | configured=%s", c.Name, support.ShortID(c.ID), c.Environment, ts))
	}
	if len(rows) == 0 {
		rows = []string{"(no configured APIs)"}
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Configured APIs: %d", len(list))},
		ResultsHeading: "Configured APIs",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
