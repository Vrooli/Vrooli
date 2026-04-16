package bookmarks

import (
	"fmt"
	"os"

	"bookmark-intelligence-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `bookmark` subcommand group for process/query/sync.
// Query filters are forwarded to the API unchanged — no client-side filtering.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "bookmark",
		Description: "Process, query, and sync bookmarks",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "process", Description: "Process new bookmarks with AI", Run: func(args []string) error { return runProcess(core, args) }},
			{Name: "query", Aliases: []string{"list", "ls"}, Description: "Query bookmarks", Run: func(args []string) error { return runQuery(core, args) }},
			{Name: "sync", Description: "Synchronize bookmarks from all platforms", Run: func(args []string) error { return runSync(core, args) }},
		},
	}
}

func runProcess(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("bookmark process")
	bodyFile := fs.String("body-file", "", "Optional JSON file with the request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, false)
	if err != nil {
		return err
	}
	if payload == nil {
		payload = []byte("{}")
	}

	body, err := core.Request("POST", "/bookmarks/process", nil, payload)
	if err != nil {
		return err
	}

	return renderCountResult(body, "Bookmark processing completed", "processed_count", "Processed", *jsonOutput)
}

func runQuery(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("bookmark query")
	category := fs.String("category", "", "Filter by category")
	platform := fs.String("platform", "", "Filter by platform")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"category": *category,
		"platform": *platform,
	})
	body, err := core.Get("/bookmarks/query", query)
	if err != nil {
		return err
	}

	// The query endpoint returns { bookmarks, total_count, categories } — keep
	// it as a free-form map since shape may evolve.
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	total := 0
	if v, ok := resp["total_count"].(float64); ok {
		total = int(v)
	}

	rows := []string{}
	if list, ok := resp["bookmarks"].([]interface{}); ok {
		for _, item := range list {
			m, ok := item.(map[string]interface{})
			if !ok {
				rows = append(rows, support.RenderValue(item))
				continue
			}
			title, _ := m["title"].(string)
			plat, _ := m["platform"].(string)
			cat, _ := m["category"].(string)
			id, _ := m["id"].(string)
			rows = append(rows, fmt.Sprintf("%s | %s | platform=%s | category=%s", support.ShortID(id), title, plat, cat))
		}
	}
	if len(rows) == 0 {
		rows = []string{"(no bookmarks matched)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Bookmarks: %d", total)},
		ResultsHeading: "Results",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s bookmark query --category <name>", support.CLIName),
			fmt.Sprintf("%s bookmark query --platform <name>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSync(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("bookmark sync")
	bodyFile := fs.String("body-file", "", "Optional JSON file with the request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, false)
	if err != nil {
		return err
	}
	if payload == nil {
		payload = []byte("{}")
	}

	body, err := core.Request("POST", "/bookmarks/sync", nil, payload)
	if err != nil {
		return err
	}

	return renderCountResult(body, "Bookmark sync completed", "processed_count", "Processed", *jsonOutput)
}

// renderCountResult renders a {success, <countKey>, message} style response as
// a MutationReport. The API returns plain JSON (no envelope) for these.
func renderCountResult(body []byte, fallbackResult, countKey, countLabel string, asJSON bool) error {
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	result := fallbackResult
	if msg, ok := resp["message"].(string); ok && msg != "" {
		result = msg
	}

	changes := []string{}
	if v, ok := resp[countKey].(float64); ok {
		changes = append(changes, fmt.Sprintf("%s: %d", countLabel, int(v)))
	}

	report := cliapp.MutationReport{
		Result:      []string{result},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s bookmark query", support.CLIName)},
	}
	if asJSON {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
