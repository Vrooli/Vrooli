package patterns

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"scalable-app-cookbook/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `patterns` subcommand group covering search/get/recipes/chapters/stats.
// The API owns ranking and filtering; this package is a thin wrapper that formats
// responses through the standard output contracts.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "patterns",
		Description: "Search and inspect architectural patterns",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "search", Description: "Search architectural patterns", Run: func(args []string) error { return runSearch(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Get a pattern by id or title", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "recipes", Description: "List recipes for a pattern", Run: func(args []string) error { return runRecipes(core, args) }},
			{Name: "chapters", Description: "List cookbook chapters", Run: func(args []string) error { return runChapters(core, args) }},
			{Name: "stats", Description: "Show cookbook statistics", Run: func(args []string) error { return runStats(core, args) }},
		},
	}
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("patterns search")
	chapter := fs.String("chapter", "", "Filter by cookbook chapter")
	section := fs.String("section", "", "Filter by chapter section")
	level := fs.String("level", "", "Filter by maturity level (L0-L4)")
	tags := fs.String("tags", "", "Filter by comma-separated pattern tags")
	limit := fs.Int("limit", 50, "Maximum patterns to return (1-100)")
	offset := fs.Int("offset", 0, "Pagination offset")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := ""
	if fs.NArg() > 0 {
		query = strings.Join(fs.Args(), " ")
	}

	params := support.BuildQuery(map[string]string{
		"query":          query,
		"chapter":        *chapter,
		"section":        *section,
		"maturity_level": *level,
		"tags":           *tags,
		"limit":          strconv.Itoa(*limit),
		"offset":         strconv.Itoa(*offset),
	})

	body, err := core.Get("/patterns/search", params)
	if err != nil {
		return err
	}
	var resp support.PatternSearchResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Found %d of %d pattern(s)", len(resp.Patterns), resp.Total)}
	if query != "" {
		summary = append(summary, fmt.Sprintf("Query: %s", query))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Patterns",
		Results:        patternRows(resp.Patterns),
		RetrievalHints: []string{
			fmt.Sprintf("%s patterns get <id>", support.CLIName),
			fmt.Sprintf("%s patterns recipes <id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("patterns get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: patterns get <pattern-id-or-title>")
	}
	id := strings.Join(fs.Args(), " ")

	body, err := core.Get("/patterns/"+id, nil)
	if err != nil {
		return err
	}
	var pattern support.Pattern
	if err := support.Decode(body, &pattern); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Pattern: %s", pattern.Title)},
		ResultsHeading: "Details",
		Results:        patternDetailRows(pattern),
		RetrievalHints: []string{
			fmt.Sprintf("%s patterns recipes %s", support.CLIName, pattern.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRecipes(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("patterns recipes")
	recipeType := fs.String("recipe-type", "", "Filter recipes by type (greenfield|brownfield|migration)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: patterns recipes <pattern-id-or-title> [--recipe-type type]")
	}
	id := strings.Join(fs.Args(), " ")

	query := support.BuildQuery(map[string]string{"type": *recipeType})
	body, err := core.Get("/patterns/"+id+"/recipes", query)
	if err != nil {
		return err
	}
	var resp support.PatternRecipesResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Pattern: %s", resp.Pattern.Title),
		fmt.Sprintf("Recipes: %d", len(resp.Recipes)),
	}
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Recipes",
		Results:        recipeRows(resp.Recipes),
		RetrievalHints: []string{
			fmt.Sprintf("%s recipes get <recipe-id>", support.CLIName),
			fmt.Sprintf("%s recipes generate <recipe-id> <language>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runChapters(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("patterns chapters")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/patterns/chapters", nil)
	if err != nil {
		return err
	}
	var chapters []support.Chapter
	if err := support.Decode(body, &chapters); err != nil {
		return err
	}

	rows := make([]string, 0, len(chapters))
	for _, c := range chapters {
		rows = append(rows, fmt.Sprintf("%s: %d patterns", c.Name, c.PatternCount))
	}
	if len(rows) == 0 {
		rows = []string{"(no chapters found)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Chapters: %d", len(chapters))},
		ResultsHeading: "Cookbook chapters",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s patterns search --chapter \"<name>\"", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runStats(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("patterns stats")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/patterns/stats", nil)
	if err != nil {
		return err
	}
	var stats support.StatsResponse
	if err := support.Decode(body, &stats); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Total patterns: %d", stats.Statistics.TotalPatterns),
		fmt.Sprintf("Total recipes: %d", stats.Statistics.TotalRecipes),
		fmt.Sprintf("Total implementations: %d", stats.Statistics.TotalImplementations),
		fmt.Sprintf("Total chapters: %d", stats.Statistics.TotalChapters),
	}
	if len(stats.MaturityLevels) > 0 {
		results = append(results, "--- Maturity levels ---")
		results = append(results, intMapRows(stats.MaturityLevels)...)
	}
	if len(stats.Languages) > 0 {
		results = append(results, "--- Languages ---")
		results = append(results, intMapRows(stats.Languages)...)
	}

	report := cliapp.ListReport{
		Summary:        []string{"Scalable App Cookbook statistics"},
		ResultsHeading: "Statistics",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s patterns chapters", support.CLIName),
			fmt.Sprintf("%s patterns search <query>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func patternRows(patterns []support.Pattern) []string {
	if len(patterns) == 0 {
		return []string{"(no patterns matched)"}
	}
	rows := make([]string, 0, len(patterns))
	for _, p := range patterns {
		line := fmt.Sprintf("%s (%s) | %s", p.Title, support.ShortID(p.ID), p.MaturityLevel)
		if p.Chapter != "" {
			line += fmt.Sprintf(" | chapter=%s", p.Chapter)
		}
		if p.Section != "" {
			line += fmt.Sprintf(" | section=%s", p.Section)
		}
		if p.RecipeCount > 0 {
			line += fmt.Sprintf(" | recipes=%d", p.RecipeCount)
		}
		if p.ImplCount > 0 {
			line += fmt.Sprintf(" | impls=%d", p.ImplCount)
		}
		if len(p.Tags) > 0 {
			line += fmt.Sprintf(" | tags=%s", strings.Join(p.Tags, ","))
		}
		rows = append(rows, line)
	}
	return rows
}

func patternDetailRows(p support.Pattern) []string {
	rows := []string{
		fmt.Sprintf("ID: %s", p.ID),
		fmt.Sprintf("Title: %s", p.Title),
		fmt.Sprintf("Chapter: %s", p.Chapter),
		fmt.Sprintf("Section: %s", p.Section),
		fmt.Sprintf("Maturity level: %s", p.MaturityLevel),
	}
	if len(p.Tags) > 0 {
		rows = append(rows, fmt.Sprintf("Tags: %s", strings.Join(p.Tags, ", ")))
	}
	if len(p.RefPatterns) > 0 {
		rows = append(rows, fmt.Sprintf("Reference patterns: %s", strings.Join(p.RefPatterns, ", ")))
	}
	if p.WhatAndWhy != "" {
		rows = append(rows, fmt.Sprintf("What & why: %s", p.WhatAndWhy))
	}
	if p.WhenToUse != "" {
		rows = append(rows, fmt.Sprintf("When to use: %s", p.WhenToUse))
	}
	if p.Tradeoffs != "" {
		rows = append(rows, fmt.Sprintf("Trade-offs: %s", p.Tradeoffs))
	}
	if p.FailureModes != "" {
		rows = append(rows, fmt.Sprintf("Failure modes: %s", p.FailureModes))
	}
	if p.CostLevers != "" {
		rows = append(rows, fmt.Sprintf("Cost levers: %s", p.CostLevers))
	}
	if p.CreatedAt != "" {
		rows = append(rows, fmt.Sprintf("Created: %s", support.FormatTime(p.CreatedAt)))
	}
	if p.UpdatedAt != "" {
		rows = append(rows, fmt.Sprintf("Updated: %s", support.FormatTime(p.UpdatedAt)))
	}
	return rows
}

func recipeRows(recipes []support.Recipe) []string {
	if len(recipes) == 0 {
		return []string{"(no recipes available)"}
	}
	rows := make([]string, 0, len(recipes))
	for _, r := range recipes {
		line := fmt.Sprintf("%s (%s) | type=%s | steps=%d | timeout=%ds",
			r.Title, support.ShortID(r.ID), r.Type, len(r.Steps), r.TimeoutSec)
		if len(r.Prerequisites) > 0 {
			line += fmt.Sprintf(" | prereqs=%s", strings.Join(r.Prerequisites, ","))
		}
		if len(r.Artifacts) > 0 {
			line += fmt.Sprintf(" | artifacts=%s", strings.Join(r.Artifacts, ","))
		}
		rows = append(rows, line)
	}
	return rows
}

func intMapRows(data map[string]int) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := make([]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, fmt.Sprintf("%s: %d", k, data[k]))
	}
	return rows
}
