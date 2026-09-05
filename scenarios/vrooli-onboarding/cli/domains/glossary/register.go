package glossary

import (
	"fmt"
	"os"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `vrooli-onboarding glossary` as a flat command since
// `/api/v1/glossary` is a single read-only endpoint. Filtering happens
// server-side via the --query flag.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Glossary",
		Commands: []cliapp.Command{
			{
				Name:        "glossary",
				Description: "Look up Vrooli glossary terms",
				NeedsAPI:    true,
				Run:         func(args []string) error { return run(core, args) },
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("glossary")
	query := fs.String("query", "", "Optional search term (server-side filter)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	q := support.BuildQuery(map[string]string{"q": *query})
	body, err := core.Get("/glossary", q)
	if err != nil {
		return err
	}
	var resp support.GlossaryResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Glossary entries: %d", resp.Count)}
	if resp.Query != "" {
		summary = append(summary, fmt.Sprintf("Query: %q", resp.Query))
	}
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Terms",
		Results:        rows(resp.Entries),
		RetrievalHints: []string{
			fmt.Sprintf("%s glossary --query postgres", support.CLIName),
			fmt.Sprintf("%s resources list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func rows(entries []support.GlossaryEntry) []string {
	if len(entries) == 0 {
		return []string{"(no matching glossary entries)"}
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, fmt.Sprintf("%s [%s] -> %s", e.Term, e.Category, e.Description))
	}
	return out
}
