package scoring

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"scenario-completeness-scoring/cli/format"
	"scenario-completeness-scoring/cli/internal/support"
	"scenario-completeness-scoring/cli/models"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "scenario-completeness-scoring"

type scoresListResponse struct {
	Scenarios []struct {
		Scenario       string  `json:"scenario"`
		Category       string  `json:"category"`
		Score          float64 `json:"score"`
		Classification string  `json:"classification"`
		Partial        bool    `json:"partial"`
	} `json:"scenarios"`
	Total int `json:"total"`
}

type recommendationsResponse struct {
	Scenario        string                  `json:"scenario"`
	CurrentScore    interface{}             `json:"current_score"`
	TotalImpact     interface{}             `json:"total_impact"`
	PotentialScore  interface{}             `json:"potential_score"`
	Recommendations []models.Recommendation `json:"recommendations"`
}

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Scoring",
		Commands: []cliapp.Command{
			{Name: "scores", NeedsAPI: true, Description: "List completeness scores for all scenarios", Run: func(args []string) error { return runScores(core, args) }},
			{Name: "score", NeedsAPI: true, Description: "Show detailed score for a scenario", Run: func(args []string) error { return runScore(core, args) }},
			{Name: "calculate", NeedsAPI: true, Description: "Force score recalculation and save history", Run: func(args []string) error { return runCalculate(core, args) }},
			{Name: "history", NeedsAPI: true, Description: "View score history", Run: func(args []string) error { return runHistory(core, args) }},
			{Name: "trends", NeedsAPI: true, Description: "View trend analysis for a scenario", Run: func(args []string) error { return runTrends(core, args) }},
			{Name: "what-if", NeedsAPI: true, Description: "Run hypothetical improvement analysis", Run: func(args []string) error { return runWhatIf(core, args) }},
			{Name: "recommend", NeedsAPI: true, Description: "Get prioritized improvement recommendations", Run: func(args []string) error { return runRecommend(core, args) }},
		},
	}
}

func runScores(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOut, err := support.ParseFlags("scores", args)
	if err != nil {
		return err
	}
	_ = fs

	body, err := core.Get("/scores", nil)
	if err != nil {
		return err
	}

	var parsed scoresListResponse
	if err := support.Decode(body, &parsed); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenarios scored: %d", len(parsed.Scenarios)),
			fmt.Sprintf("Total available: %d", parsed.Total),
		},
		Results:        renderScores(parsed),
		RetrievalHints: []string{cliName + " score <scenario>", cliName + " recommend <scenario>"},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runScore(core *cliapp.ScenarioApp, args []string) error {
	args = support.NormalizeInterspersedFlags(args)
	fs := flag.NewFlagSet("score", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	verbose := fs.Bool("verbose", false, "Show detailed breakdown")
	fs.BoolVar(verbose, "v", false, "Show detailed breakdown")
	metrics := fs.Bool("metrics", false, "Include raw metric counters")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *metrics {
		*verbose = true
	}
	if err := support.RequireArg(fs, "score <scenario> [--json] [--verbose] [--metrics]"); err != nil {
		return err
	}

	body, err := core.Get("/scores/"+fs.Arg(0), nil)
	if err != nil {
		return err
	}

	var resp models.ScoreResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	format.FormatValidationIssues(resp.ValidationAnalysis, *verbose)
	format.FormatScoreSummary(resp)
	format.FormatBaseMetrics(resp.Breakdown)
	format.FormatActionPlan(resp)

	if *metrics {
		fmt.Println()
		fmt.Println("Metrics:")
		cliutil.PrintJSONMap(resp.Metrics, 2)
	}

	format.FormatComparisonContext(resp.ValidationAnalysis, resp.Score)
	return nil
}

func runCalculate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("calculate", flag.ContinueOnError)
	source := fs.String("source", "", "Source identifier for history tracking")
	jsonOutput := cliutil.JSONFlag(fs)
	var tags cliutil.StringList
	fs.Var(&tags, "tag", "Tag to associate with snapshot (repeatable)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := support.RequireArg(fs, "calculate <scenario> [--source name] [--tag value] [--json]"); err != nil {
		return err
	}

	payload := map[string]interface{}{}
	if *source != "" {
		payload["source"] = *source
	}
	if values := tags.Values(); len(values) > 0 {
		payload["tags"] = values
	}
	if _, err := core.Request("POST", "/scores/"+fs.Arg(0)+"/calculate", nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Score recalculated", "Scenario: " + fs.Arg(0)},
		Changes: []string{
			"Snapshot persisted to history.",
		},
		NextCommand: []string{
			cliName + " history " + fs.Arg(0),
			cliName + " score " + fs.Arg(0),
		},
	}
	if *source != "" {
		report.Changes = append(report.Changes, "Source: "+*source)
	}
	if values := tags.Values(); len(values) > 0 {
		report.Changes = append(report.Changes, "Tags: "+fmt.Sprintf("%v", values))
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runHistory(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	limit := fs.Int("limit", 30, "Number of entries to show")
	source := fs.String("source", "", "Filter by source")
	jsonOutput := cliutil.JSONFlag(fs)
	var tags cliutil.StringList
	fs.Var(&tags, "tag", "Filter by tag (repeatable)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := support.RequireArg(fs, "history <scenario> [--limit N] [--source name] [--tag value] [--json]"); err != nil {
		return err
	}

	query := historyQuery(*limit, *source, tags.Values())
	body, err := core.Get("/scores/"+fs.Arg(0)+"/history", query)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Scenario history loaded",
			"Scenario: " + fs.Arg(0),
		},
		ResultsHeading: "History Payload",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{cliName + " trends " + fs.Arg(0), cliName + " calculate " + fs.Arg(0)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runTrends(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("trends", flag.ContinueOnError)
	limit := fs.Int("limit", 30, "Number of entries to analyze")
	source := fs.String("source", "", "Filter by source")
	jsonOutput := cliutil.JSONFlag(fs)
	var tags cliutil.StringList
	fs.Var(&tags, "tag", "Filter by tag")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := support.RequireArg(fs, "trends <scenario> [--limit N] [--source name] [--tag value] [--json]"); err != nil {
		return err
	}

	query := historyQuery(*limit, *source, tags.Values())
	body, err := core.Get("/scores/"+fs.Arg(0)+"/trends", query)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Trend analysis loaded",
			"Scenario: " + fs.Arg(0),
		},
		ResultsHeading: "Trend Payload",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{cliName + " history " + fs.Arg(0), cliName + " recommend " + fs.Arg(0)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runWhatIf(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("what-if", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	changesFile := fs.String("file", "", "Path to JSON file describing changes")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := support.RequireArg(fs, "what-if <scenario> [--file path] [--json]"); err != nil {
		return err
	}

	payload := map[string]interface{}{"changes": []interface{}{}}
	if *changesFile != "" {
		data, err := os.ReadFile(*changesFile)
		if err != nil {
			return fmt.Errorf("read changes file: %w", err)
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("parse changes file: %w", err)
		}
	}

	body, err := core.Request("POST", "/scores/"+fs.Arg(0)+"/what-if", nil, payload)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			"What-if analysis complete",
			"Scenario: " + fs.Arg(0),
		},
		ResultsHeading: "Analysis Payload",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{cliName + " score " + fs.Arg(0), cliName + " recommend " + fs.Arg(0)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRecommend(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOutput, err := support.ParseFlags("recommend", args)
	if err != nil {
		return err
	}
	if err := support.RequireArg(fs, "recommend <scenario> [--json]"); err != nil {
		return err
	}

	body, err := core.Get("/recommendations/"+fs.Arg(0), nil)
	if err != nil {
		return err
	}

	var response recommendationsResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Recommendations generated",
			"Scenario: " + fallbackString(response.Scenario, fs.Arg(0)),
			"Current score: " + support.StringValue(response.CurrentScore),
			"Potential score: " + support.StringValue(response.PotentialScore),
		},
		ResultsHeading: "Recommendations",
		Results:        renderRecommendations(response.Recommendations),
		RetrievalHints: []string{cliName + " score " + fallbackString(response.Scenario, fs.Arg(0)), cliName + " what-if " + fallbackString(response.Scenario, fs.Arg(0))},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderScores(parsed scoresListResponse) []string {
	lines := make([]string, 0, len(parsed.Scenarios))
	for _, item := range parsed.Scenarios {
		partial := ""
		if item.Partial {
			partial = " (partial)"
		}
		lines = append(lines, fmt.Sprintf("%-32s %-8s score=%5.2f classification=%s%s",
			item.Scenario, item.Category, item.Score, item.Classification, partial))
	}
	return lines
}

func renderRecommendations(recommendations []models.Recommendation) []string {
	lines := make([]string, 0, len(recommendations))
	for _, rec := range recommendations {
		lines = append(lines, fmt.Sprintf("%s (impact %.0f)", rec.Message, rec.Impact))
	}
	return lines
}

func fallbackString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func historyQuery(limit int, source string, tags []string) map[string][]string {
	query := map[string][]string{}
	if limit > 0 {
		query["limit"] = []string{fmt.Sprintf("%d", limit)}
	}
	if source != "" {
		query["source"] = []string{source}
	}
	for _, tag := range tags {
		query["tag"] = append(query["tag"], tag)
	}
	return query
}
