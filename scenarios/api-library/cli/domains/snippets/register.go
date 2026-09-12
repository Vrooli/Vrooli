package snippets

import (
	"fmt"
	"os"
	"strings"

	"api-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `snippets` subcommands against the global snippet endpoints
// (popular, get, vote). Per-API snippet creation/listing lives under `apis`.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "snippets",
		Description: "Browse integration snippets",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "popular", Description: "List popular snippets across all APIs", Run: func(args []string) error { return runPopular(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show a single snippet", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "vote", Description: "Vote on a snippet's helpfulness", Run: func(args []string) error { return runVote(core, args) }},
		},
	}
}

func runPopular(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("snippets popular")
	limit := fs.String("limit", "", "Maximum number of snippets to return")
	language := fs.String("language", "", "Filter by language")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	query := support.BuildQuery(map[string]string{
		"limit":    *limit,
		"language": *language,
	})
	raw, err := core.Get("/snippets/popular", query)
	if err != nil {
		return err
	}
	var env support.SnippetsEnvelope
	if err := support.Decode(raw, &env); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Popular snippets: %d", env.Count)},
		ResultsHeading: "Snippets",
		Results:        snippetRows(env.Snippets),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("snippets get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: snippets get <snippet-id>")
	}
	id := fs.Arg(0)
	raw, err := core.Get("/snippets/"+id, nil)
	if err != nil {
		return err
	}
	var snippet support.Snippet
	if err := support.Decode(raw, &snippet); err != nil {
		return err
	}
	results := []string{
		fmt.Sprintf("ID: %s", snippet.ID),
		fmt.Sprintf("Title: %s", snippet.Title),
		fmt.Sprintf("API: %s / %s", orDash(snippet.APIName), orDash(snippet.APIProvider)),
		fmt.Sprintf("Language: %s", orDash(snippet.Language)),
		fmt.Sprintf("Framework: %s", orDash(snippet.Framework)),
		fmt.Sprintf("Type: %s", orDash(snippet.SnippetType)),
		fmt.Sprintf("Helpful: %d / %d", snippet.HelpfulCount, snippet.HelpfulCount+snippet.NotHelpfulCount),
	}
	if snippet.Description != "" {
		results = append(results, "", "Description:", snippet.Description)
	}
	if snippet.Code != "" {
		results = append(results, "", "Code:", snippet.Code)
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Snippet: %s", snippet.Title)},
		ResultsHeading: "Details",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runVote(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("snippets vote")
	helpful := fs.Bool("helpful", false, "Mark the snippet helpful")
	notHelpful := fs.Bool("not-helpful", false, "Mark the snippet not helpful")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the vote payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: snippets vote <snippet-id> (--helpful | --not-helpful)")
	}
	id := fs.Arg(0)

	var body interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		body = raw
	} else {
		switch {
		case *helpful && *notHelpful:
			return fmt.Errorf("--helpful and --not-helpful are mutually exclusive")
		case *helpful:
			body = map[string]interface{}{"helpful": true}
		case *notHelpful:
			body = map[string]interface{}{"helpful": false}
		default:
			return fmt.Errorf("vote direction is required: --helpful or --not-helpful")
		}
	}
	raw, err := core.Request("POST", "/snippets/"+id+"/vote", nil, body)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	_ = support.Decode(raw, &resp)
	message := support.RenderValue(resp["message"])
	if message == "" || message == "null" {
		message = "Vote recorded"
	}
	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("POST /snippets/%s/vote", id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func snippetRows(list []support.Snippet) []string {
	if len(list) == 0 {
		return []string{"(no snippets)"}
	}
	rows := make([]string, 0, len(list))
	for _, s := range list {
		rows = append(rows, fmt.Sprintf("%s (%s) | api=%s | language=%s | helpful=%d | uses=%d",
			s.Title, support.ShortID(s.ID), orDash(s.APIName), orDash(s.Language), s.HelpfulCount, s.UsageCount))
	}
	return rows
}

func orDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}
