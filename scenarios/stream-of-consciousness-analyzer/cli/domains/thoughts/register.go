package thoughts

import (
	"flag"
	"fmt"
	"net/url"
	"os"

	"stream-of-consciousness-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "stream-of-consciousness-analyzer"

type thought struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "thought",
		Description: "Manage thought nodes",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List thoughts", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get a thought", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create a thought", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", NeedsAPI: true, Description: "Update a thought", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", NeedsAPI: true, Description: "Delete a thought", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("thought list", flag.ContinueOnError)
	schemeID := fs.String("scheme", "", "Filter by scheme ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *schemeID != "" {
		query.Set("scheme_id", *schemeID)
	}

	body, err := core.Get("/thoughts", query)
	if err != nil {
		return err
	}

	var thoughts []thought
	if err := support.Unmarshal(body, &thoughts); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Total thoughts: %d", len(thoughts))}
	if *schemeID != "" {
		summary = append(summary, "Filtered by scheme: "+*schemeID)
	}
	report := cliapp.ListReport{
		Summary: summary,
		Results: renderList(thoughts),
		RetrievalHints: []string{
			cliName + " thought get <thought-id>",
			cliName + " edge list <thought-id>",
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOut, err := support.ParseFlags("thought get", args)
	if err != nil {
		return err
	}
	if err := support.RequireArg(fs, "thought get <id> [--json]"); err != nil {
		return err
	}

	body, err := core.Get("/thoughts/"+fs.Arg(0), nil)
	if err != nil {
		return err
	}

	var item thought
	if err := support.Unmarshal(body, &item); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Thought loaded", "Thought ID: " + item.ID},
		ResultsHeading: "Details",
		Results: []string{
			"Title: " + item.Title,
			"Body: " + item.Body,
		},
		RetrievalHints: []string{
			cliName + " thought update " + item.ID + " --title \"...\"",
			cliName + " edge list " + item.ID,
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("thought create", flag.ContinueOnError)
	title := fs.String("title", "", "Thought title (required)")
	bodyFlag := fs.String("body", "", "Thought body")
	schemeID := fs.String("scheme", "", "Scheme ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *title == "" {
		return fmt.Errorf("usage: thought create --title TITLE [--body BODY] [--scheme ID]")
	}

	input := map[string]any{
		"title": *title,
		"body":  *bodyFlag,
	}
	if *schemeID != "" {
		input["scheme_id"] = *schemeID
	}

	body, err := core.Request("POST", "/thoughts", nil, input)
	if err != nil {
		return err
	}

	var item thought
	if err := support.Unmarshal(body, &item); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Thought created", "Thought ID: " + item.ID},
		Changes: []string{
			"Title: " + item.Title,
		},
		NextCommand: []string{
			cliName + " thought get " + item.ID,
			cliName + " edge create " + item.ID + " --target <thought-id>",
		},
	}
	if *schemeID != "" {
		report.Changes = append(report.Changes, "Scheme: "+*schemeID)
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("thought update", flag.ContinueOnError)
	title := fs.String("title", "", "New title")
	bodyFlag := fs.String("body", "", "New body")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := support.RequireArg(fs, "thought update <id> [--title TITLE] [--body BODY] [--json]"); err != nil {
		return err
	}

	input := map[string]any{}
	if *title != "" {
		input["title"] = *title
	}
	if *bodyFlag != "" {
		input["body"] = *bodyFlag
	}
	if len(input) == 0 {
		return fmt.Errorf("at least one of --title or --body is required")
	}

	body, err := core.Request("PUT", "/thoughts/"+fs.Arg(0), nil, input)
	if err != nil {
		return err
	}

	var item thought
	if err := support.Unmarshal(body, &item); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Thought updated", "Thought ID: " + item.ID},
		Changes: []string{
			"Title: " + item.Title,
			"Body: " + support.Truncate(item.Body, 80),
		},
		NextCommand: []string{
			cliName + " thought get " + item.ID,
			cliName + " suggestion generate <scheme-id>",
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOut, err := support.ParseFlags("thought delete", args)
	if err != nil {
		return err
	}
	if err := support.RequireArg(fs, "thought delete <id> [--json]"); err != nil {
		return err
	}

	if _, err := core.Request("DELETE", "/thoughts/"+fs.Arg(0), nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Thought deleted", "Thought ID: " + fs.Arg(0)},
		Changes:     []string{"Removed the thought node from the graph."},
		NextCommand: []string{cliName + " thought list"},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderList(thoughts []thought) []string {
	lines := make([]string, 0, len(thoughts))
	for _, item := range thoughts {
		lines = append(lines, fmt.Sprintf("%s  %s", item.ID, item.Title))
	}
	return lines
}
