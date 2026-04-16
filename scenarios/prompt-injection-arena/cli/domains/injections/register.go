package injections

import (
	"fmt"
	"os"
	"strconv"

	"prompt-injection-arena/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `injections` subcommand group covering the /api/v1/injections*
// surface: listing the technique library, adding new techniques, and
// retrieving semantically-similar techniques via the vector search backend.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "injections",
		Description: "Browse, add, and search injection techniques",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"library", "ls"}, Description: "List injection techniques", Run: func(args []string) error { return runList(core, args) }},
			{Name: "add", Description: "Add a new injection technique", Run: func(args []string) error { return runAdd(core, args) }},
			{Name: "similar", Description: "Find similar injection techniques (vector search)", Run: func(args []string) error { return runSimilar(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("injections list")
	category := fs.String("category", "", "Filter by category")
	active := fs.String("active", "", "Filter by active flag (true|false)")
	limit := fs.Int("limit", 100, "Maximum rows to return")
	offset := fs.Int("offset", 0, "Row offset for pagination")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"category": *category,
		"active":   *active,
		"limit":    strconv.Itoa(*limit),
		"offset":   strconv.Itoa(*offset),
	})
	body, err := core.Get("/injections/library", query)
	if err != nil {
		return err
	}

	var resp support.InjectionLibraryResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Showing %d of %d techniques", len(resp.Techniques), resp.TotalCount),
	}
	if *category != "" {
		summary = append(summary, fmt.Sprintf("Category filter: %s", *category))
	}
	if len(resp.Categories) > 0 {
		summary = append(summary, fmt.Sprintf("Available categories: %d", len(resp.Categories)))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Techniques",
		Results:        techniqueRows(resp.Techniques),
		RetrievalHints: []string{
			fmt.Sprintf("%s injections similar --query \"<text>\"", support.CLIName),
			fmt.Sprintf("%s injections list --category <name>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runAdd(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("injections add")
	name := fs.String("name", "", "Technique name (required)")
	category := fs.String("category", "", "Technique category (required)")
	example := fs.String("example", "", "Example prompt (required)")
	description := fs.String("description", "", "Free-text description")
	difficulty := fs.Float64("difficulty", 0.5, "Difficulty score, 0.0-1.0")
	source := fs.String("source", "CLI user submission", "Source attribution")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request body (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var body interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		body = raw
	} else {
		if *name == "" || *category == "" || *example == "" {
			return fmt.Errorf("usage: injections add --name NAME --category CATEGORY --example TEXT [--difficulty 0.0-1.0] [--description TEXT] [--body-file PATH]")
		}
		body = map[string]interface{}{
			"name":               *name,
			"category":           *category,
			"example_prompt":     *example,
			"description":        *description,
			"difficulty_score":   *difficulty,
			"source_attribution": *source,
		}
	}

	raw, err := core.Request("POST", "/injections", nil, body)
	if err != nil {
		return err
	}

	var resp support.AddInjectionResponse
	_ = support.Decode(raw, &resp)

	message := resp.Message
	if message == "" {
		message = support.EnvelopeMessage(raw)
	}
	if message == "" {
		message = "Injection technique created"
	}

	changes := []string{}
	if resp.ID != "" {
		changes = append(changes, fmt.Sprintf("Created technique %s", resp.ID))
	}
	if resp.Technique.Name != "" {
		changes = append(changes, fmt.Sprintf("Name: %s", resp.Technique.Name))
	}
	if resp.Technique.Category != "" {
		changes = append(changes, fmt.Sprintf("Category: %s", resp.Technique.Category))
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s injections list --category %s", support.CLIName, resp.Technique.Category),
			fmt.Sprintf("%s vector index --injection-id %s", support.CLIName, resp.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSimilar(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("injections similar")
	query := fs.String("query", "", "Query text (required)")
	limit := fs.Int("limit", 10, "Maximum hits to return")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *query == "" {
		return fmt.Errorf("usage: injections similar --query TEXT [--limit N]")
	}

	values := support.BuildQuery(map[string]string{
		"query": *query,
		"limit": strconv.Itoa(*limit),
	})
	body, err := core.Get("/injections/similar", values)
	if err != nil {
		return err
	}

	var resp support.SimilarInjectionsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Query: %q | hits: %d", resp.Query, len(resp.Results))},
		ResultsHeading: "Similar techniques",
		Results:        similarRows(resp.Results),
		RetrievalHints: []string{
			fmt.Sprintf("%s vector search --body-file payload.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func techniqueRows(items []support.InjectionTechnique) []string {
	if len(items) == 0 {
		return []string{"(no techniques matched)"}
	}
	rows := make([]string, 0, len(items))
	for _, t := range items {
		rows = append(rows, fmt.Sprintf("%s (%s) | %s | difficulty=%.2f | success=%.2f",
			t.Name, support.ShortID(t.ID), t.Category, t.DifficultyScore, t.SuccessRate))
	}
	return rows
}

func similarRows(items []support.SimilarInjectionResult) []string {
	if len(items) == 0 {
		return []string{"(no similar techniques found)"}
	}
	rows := make([]string, 0, len(items))
	for _, r := range items {
		name := r.Name
		if name == "" {
			if v, ok := r.Payload["name"].(string); ok {
				name = v
			}
		}
		cat := r.Category
		if cat == "" {
			if v, ok := r.Payload["category"].(string); ok {
				cat = v
			}
		}
		rows = append(rows, fmt.Sprintf("%s (%s) | %s | score=%.3f",
			name, support.ShortID(r.ID), cat, r.Score))
	}
	return rows
}
