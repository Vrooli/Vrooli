package scores

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"scenario-completeness-scoring/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `score` subcommand group. The API surface covers
// listing, per-scenario detail, forced calculation, validation analysis,
// history, trends, what-if, recommendations, and a bulk refresh. Each command
// is a thin wrapper around the corresponding API endpoint.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "score",
		Description: "Calculate and inspect scenario completeness scores",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List completeness scores for all scenarios", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one scenario's score", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "calculate", Description: "Force recalculation and persist a history snapshot", Run: func(args []string) error { return runCalculate(core, args) }},
			{Name: "validation", Description: "Show validation-analysis details for a scenario", Run: func(args []string) error { return runValidation(core, args) }},
			{Name: "history", Description: "List score history for a scenario", Run: func(args []string) error { return runHistory(core, args) }},
			{Name: "trends", Description: "Show trend analysis for a scenario", Run: func(args []string) error { return runTrends(core, args) }},
			{Name: "what-if", Description: "Run a what-if analysis (requires --body-file)", Run: func(args []string) error { return runWhatIf(core, args) }},
			{Name: "recommend", Description: "List prioritized improvement recommendations", Run: func(args []string) error { return runRecommend(core, args) }},
			{Name: "refresh-all", Description: "Trigger a bulk refresh of all scenario scores", Run: func(args []string) error { return runRefreshAll(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("score list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/scores", nil)
	if err != nil {
		return err
	}
	var resp support.ScoreListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenarios scored: %d", len(resp.Scenarios)),
			fmt.Sprintf("Total available: %d", resp.Total),
		},
		ResultsHeading: "Scores",
		Results:        scoreRows(resp.Scenarios),
		RetrievalHints: []string{
			fmt.Sprintf("%s score get <scenario>", support.CLIName),
			fmt.Sprintf("%s score recommend <scenario>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("score get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: score get <scenario>")
	}
	scenario := fs.Arg(0)

	body, err := core.Get("/scores/"+scenario, nil)
	if err != nil {
		return err
	}
	var summary support.ScoreSummary
	if err := support.Decode(body, &summary); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Scenario: %s", summary.Scenario),
		fmt.Sprintf("Category: %s", summary.Category),
		fmt.Sprintf("Score: %.2f", summary.Score),
		fmt.Sprintf("Base score: %.2f", summary.BaseScore),
		fmt.Sprintf("Validation penalty: %.2f", summary.ValidationPenalty),
		fmt.Sprintf("Classification: %s", summary.Classification),
	}
	if summary.CalculatedAt != "" {
		results = append(results, fmt.Sprintf("Calculated at: %s", support.FormatTime(summary.CalculatedAt)))
	}
	if len(summary.Breakdown) > 0 {
		results = append(results, "Breakdown:")
		for _, row := range support.MapRows(summary.Breakdown) {
			results = append(results, "  "+row)
		}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Score for %s: %.2f (%s)", summary.Scenario, summary.Score, summary.Classification)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s score validation %s", support.CLIName, scenario),
			fmt.Sprintf("%s score recommend %s", support.CLIName, scenario),
			fmt.Sprintf("%s score history %s", support.CLIName, scenario),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCalculate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("score calculate")
	source := fs.String("source", "", "Source identifier for history tracking")
	var tags cliutil.StringList
	fs.Var(&tags, "tag", "Tag to associate with the snapshot (repeatable)")
	bodyFile := fs.String("body-file", "", "Optional JSON file with a custom payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: score calculate <scenario> [--source name] [--tag v] [--body-file path]")
	}
	scenario := fs.Arg(0)

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		body := map[string]interface{}{}
		if trimmed := strings.TrimSpace(*source); trimmed != "" {
			body["source"] = trimmed
		}
		if values := tags.Values(); len(values) > 0 {
			body["tags"] = values
		}
		payload = body
	}

	respBody, err := core.Request("POST", "/scores/"+scenario+"/calculate", nil, payload)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(respBody)
	if message == "" {
		message = fmt.Sprintf("Score recalculated for %s", scenario)
	}

	changes := []string{"Snapshot persisted to history."}
	if trimmed := strings.TrimSpace(*source); trimmed != "" {
		changes = append(changes, "Source: "+trimmed)
	}
	if values := tags.Values(); len(values) > 0 {
		changes = append(changes, "Tags: "+strings.Join(values, ", "))
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s score get %s", support.CLIName, scenario),
			fmt.Sprintf("%s score history %s", support.CLIName, scenario),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runValidation(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("score validation")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: score validation <scenario>")
	}
	scenario := fs.Arg(0)

	body, err := core.Get("/scores/"+scenario+"/validation-analysis", nil)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Validation analysis for %s", scenario)},
		ResultsHeading: "Analysis payload",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{
			fmt.Sprintf("%s score get %s", support.CLIName, scenario),
			fmt.Sprintf("%s score recommend %s", support.CLIName, scenario),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runHistory(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("score history")
	limit := fs.Int("limit", 30, "Number of entries to show")
	source := fs.String("source", "", "Filter by source")
	var tags cliutil.StringList
	fs.Var(&tags, "tag", "Filter by tag (repeatable)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: score history <scenario> [--limit N] [--source s] [--tag t]")
	}
	scenario := fs.Arg(0)

	query := support.BuildQuery(map[string]string{
		"limit":  strconv.Itoa(*limit),
		"source": *source,
	})
	for _, t := range tags.Values() {
		query.Add("tag", t)
	}
	body, err := core.Get("/scores/"+scenario+"/history", query)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("History for %s (limit=%d)", scenario, *limit),
		},
		ResultsHeading: "History payload",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{
			fmt.Sprintf("%s score trends %s", support.CLIName, scenario),
			fmt.Sprintf("%s score calculate %s", support.CLIName, scenario),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runTrends(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("score trends")
	limit := fs.Int("limit", 30, "Number of entries to analyze")
	source := fs.String("source", "", "Filter by source")
	var tags cliutil.StringList
	fs.Var(&tags, "tag", "Filter by tag (repeatable)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: score trends <scenario> [--limit N] [--source s] [--tag t]")
	}
	scenario := fs.Arg(0)

	query := support.BuildQuery(map[string]string{
		"limit":  strconv.Itoa(*limit),
		"source": *source,
	})
	for _, t := range tags.Values() {
		query.Add("tag", t)
	}
	body, err := core.Get("/scores/"+scenario+"/trends", query)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Trend analysis for %s", scenario)},
		ResultsHeading: "Trend payload",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{
			fmt.Sprintf("%s score history %s", support.CLIName, scenario),
			fmt.Sprintf("%s score recommend %s", support.CLIName, scenario),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runWhatIf(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("score what-if")
	bodyFile := fs.String("body-file", "", "Path to a JSON file describing the changes payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: score what-if <scenario> [--body-file path]")
	}
	scenario := fs.Arg(0)

	var payload interface{}
	if raw, err := support.ReadJSONFile(*bodyFile, false); err != nil {
		return err
	} else if raw != nil {
		payload = raw
	} else {
		payload = map[string]interface{}{"changes": []interface{}{}}
	}

	body, err := core.Request("POST", "/scores/"+scenario+"/what-if", nil, payload)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("What-if analysis for %s", scenario)},
		ResultsHeading: "Analysis payload",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{
			fmt.Sprintf("%s score get %s", support.CLIName, scenario),
			fmt.Sprintf("%s score recommend %s", support.CLIName, scenario),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRecommend(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("score recommend")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: score recommend <scenario>")
	}
	scenario := fs.Arg(0)

	body, err := core.Get("/recommendations/"+scenario, nil)
	if err != nil {
		return err
	}
	var resp support.RecommendationsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Recommendations for %s", fallback(resp.Scenario, scenario)),
	}
	if resp.CurrentScore != nil {
		summary = append(summary, fmt.Sprintf("Current score: %s", support.RenderValue(resp.CurrentScore)))
	}
	if resp.PotentialScore != nil {
		summary = append(summary, fmt.Sprintf("Potential score: %s", support.RenderValue(resp.PotentialScore)))
	}
	if resp.TotalImpact != nil {
		summary = append(summary, fmt.Sprintf("Total impact: %s", support.RenderValue(resp.TotalImpact)))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Recommendations",
		Results:        recommendationRows(resp.Recommendations),
		RetrievalHints: []string{
			fmt.Sprintf("%s score get %s", support.CLIName, scenario),
			fmt.Sprintf("%s score what-if %s --body-file changes.json", support.CLIName, scenario),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRefreshAll(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("score refresh-all")
	bodyFile := fs.String("body-file", "", "Optional JSON file describing refresh parameters")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if raw, err := support.ReadJSONFile(*bodyFile, false); err != nil {
		return err
	} else if raw != nil {
		payload = raw
	} else {
		payload = map[string]interface{}{}
	}

	body, err := core.Request("POST", "/scores/refresh-all", nil, payload)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = "Bulk refresh issued"
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{"All scenario scores were re-computed and persisted."},
		NextCommand: []string{
			fmt.Sprintf("%s score list", support.CLIName),
			fmt.Sprintf("%s analysis trends", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func scoreRows(items []support.ScoreListItem) []string {
	if len(items) == 0 {
		return []string{"(no scored scenarios)"}
	}
	rows := make([]string, 0, len(items))
	for _, it := range items {
		partial := ""
		if it.Partial {
			partial = " (partial)"
		}
		rows = append(rows, fmt.Sprintf("%-32s %-12s score=%6.2f classification=%s%s",
			it.Scenario, it.Category, it.Score, it.Classification, partial))
	}
	return rows
}

func recommendationRows(recs []support.Recommendation) []string {
	if len(recs) == 0 {
		return []string{"(no recommendations)"}
	}
	rows := make([]string, 0, len(recs))
	for _, r := range recs {
		rows = append(rows, fmt.Sprintf("%s (impact %.2f)", r.Message, r.Impact))
	}
	return rows
}

func fallback(value, fallbackValue string) string {
	if strings.TrimSpace(value) == "" {
		return fallbackValue
	}
	return value
}
