package workflows

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"agent-metareasoning-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `agent-metareasoning-manager workflows` with list and
// semantic-search subcommands. Both endpoints are read-only on the API side
// (`GET /workflows`, `POST /workflows/search`).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "workflows",
		Description: "Discover and search metareasoning workflows",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List discovered workflows", Run: func(args []string) error { return runList(core, args) }},
			{Name: "search", Description: "Semantic search across discovered workflows", Run: func(args []string) error { return runSearch(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("workflows list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/workflows", nil)
	if err != nil {
		return err
	}
	var workflows []support.Workflow
	if err := support.Decode(body, &workflows); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Discovered workflows: %d", len(workflows))},
		ResultsHeading: "Workflows",
		Results:        workflowRows(workflows),
		RetrievalHints: []string{
			fmt.Sprintf("%s workflows search <query>", support.CLIName),
			fmt.Sprintf("%s analyze --type <type> --input <input>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("workflows search")
	query := fs.String("query", "", "Search query (alternative to positional)")
	limit := fs.Int("limit", 0, "Maximum results (0 = API default)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	q := strings.TrimSpace(*query)
	if q == "" && fs.NArg() > 0 {
		q = strings.Join(fs.Args(), " ")
	}
	if q == "" {
		return fmt.Errorf("usage: workflows search <query> [--limit N]")
	}

	payload := map[string]interface{}{"query": q}
	if *limit > 0 {
		payload["limit"] = *limit
	}

	body, err := core.Request("POST", "/workflows/search", nil, payload)
	if err != nil {
		return err
	}
	var workflows []support.Workflow
	if err := support.Decode(body, &workflows); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Query: %q", q), fmt.Sprintf("Matches: %d", len(workflows))}
	if *limit > 0 {
		summary = append(summary, fmt.Sprintf("Limit: %d", *limit))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Matching workflows",
		Results:        workflowRows(workflows),
		RetrievalHints: []string{
			fmt.Sprintf("%s workflows list", support.CLIName),
			fmt.Sprintf("%s analyze --type <type> --input <input>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func workflowRows(workflows []support.Workflow) []string {
	if len(workflows) == 0 {
		return []string{"(no workflows)"}
	}
	rows := make([]string, 0, len(workflows))
	for _, w := range workflows {
		parts := []string{fmt.Sprintf("%s (%s)", w.Name, support.ShortID(w.ID))}
		if w.Platform != "" {
			parts = append(parts, "platform="+w.Platform)
		}
		if w.PlatformID != "" {
			parts = append(parts, "platform_id="+w.PlatformID)
		}
		if w.Category != "" {
			parts = append(parts, "category="+w.Category)
		}
		parts = append(parts, "usage="+strconv.Itoa(w.UsageCount))
		if len(w.Tags) > 0 {
			parts = append(parts, "tags="+strings.Join(w.Tags, ","))
		}
		rows = append(rows, strings.Join(parts, " | "))
	}
	return rows
}
