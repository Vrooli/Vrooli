package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/vrooli/cli-core/cliutil"
	"scenario-completeness-scoring/cli/format"
)

func (a *App) cmdScores(args []string) error {
	fs := flag.NewFlagSet("scores", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	body, parsed, err := a.fetchScoresList()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	if len(parsed.Scenarios) == 0 {
		fmt.Println("No scenarios found.")
		return nil
	}
	for _, item := range parsed.Scenarios {
		partial := ""
		if item.Partial {
			partial = " (partial)"
		}
		fmt.Printf("%-32s %-8s score=%5.2f classification=%s%s\n",
			item.Scenario, item.Category, item.Score, item.Classification, partial)
	}
	return nil
}

func (a *App) cmdScore(args []string) error {
	fs := flag.NewFlagSet("score", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	verbose := fs.Bool("verbose", false, "Show detailed breakdown")
	fs.BoolVar(verbose, "v", false, "Show detailed breakdown")
	metrics := fs.Bool("metrics", false, "Include raw metric counters")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *metrics {
		*verbose = true
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: score <scenario> [--json] [--verbose] [--metrics]")
	}
	scenarioName := fs.Arg(0)
	a.warnIfBinaryStale()
	path := fmt.Sprintf("/api/v1/scores/%s", scenarioName)
	body, err := a.api.Get(path, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	resp, err := a.parseScore(body)
	if err != nil {
		return err
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

func (a *App) fetchScoresList() ([]byte, scoresListResponse, error) {
	body, err := a.api.Get("/api/v1/scores", nil)
	if err != nil {
		return nil, scoresListResponse{}, err
	}
	var resp scoresListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, scoresListResponse{}, fmt.Errorf("parse response: %w", err)
	}
	return body, resp, nil
}

func (a *App) parseScore(body []byte) (format.ScoreResponse, error) {
	var resp format.ScoreResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, fmt.Errorf("parse response: %w", err)
	}
	return resp, nil
}

func (a *App) cmdCalculate(args []string) error {
	fs := flag.NewFlagSet("calculate", flag.ContinueOnError)
	source := fs.String("source", "", "Source identifier for history tracking")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	var tags multiValue
	fs.Var(&tags, "tag", "Tag to associate with snapshot (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: calculate <scenario> [--source name] [--tag value] [--json]")
	}
	scenarioName := fs.Arg(0)
	payload := map[string]interface{}{}
	if *source != "" {
		payload["source"] = *source
	}
	if len(tags) > 0 {
		payload["tags"] = []string(tags)
	}
	path := fmt.Sprintf("/api/v1/scores/%s/calculate", scenarioName)
	body, err := a.api.Request(http.MethodPost, path, nil, payload)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Recalculated %s\n", scenarioName)
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdHistory(args []string) error {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	limit := fs.Int("limit", 30, "Number of entries to show")
	source := fs.String("source", "", "Filter by source")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	var tags multiValue
	fs.Var(&tags, "tag", "Filter by tag (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: history <scenario> [--limit N] [--source name] [--tag value] [--json]")
	}
	scenarioName := fs.Arg(0)
	query := url.Values{}
	if *limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", *limit))
	}
	if *source != "" {
		query.Set("source", *source)
	}
	for _, tag := range tags {
		query.Add("tag", tag)
	}
	path := fmt.Sprintf("/api/v1/scores/%s/history", scenarioName)
	body, err := a.api.Get(path, query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdTrends(args []string) error {
	fs := flag.NewFlagSet("trends", flag.ContinueOnError)
	limit := fs.Int("limit", 30, "Number of entries to analyze")
	source := fs.String("source", "", "Filter by source")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	var tags multiValue
	fs.Var(&tags, "tag", "Filter by tag")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: trends <scenario> [--limit N] [--source name] [--tag value] [--json]")
	}
	scenarioName := fs.Arg(0)
	query := url.Values{}
	if *limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", *limit))
	}
	if *source != "" {
		query.Set("source", *source)
	}
	for _, tag := range tags {
		query.Add("tag", tag)
	}
	path := fmt.Sprintf("/api/v1/scores/%s/trends", scenarioName)
	body, err := a.api.Get(path, query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdWhatIf(args []string) error {
	fs := flag.NewFlagSet("what-if", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	changesFile := fs.String("file", "", "Path to JSON file describing changes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: what-if <scenario> [--file path] [--json]")
	}
	scenarioName := fs.Arg(0)
	payload := map[string]interface{}{
		"changes": []interface{}{},
	}
	if *changesFile != "" {
		data, err := os.ReadFile(*changesFile)
		if err != nil {
			return fmt.Errorf("read changes file: %w", err)
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("parse changes file: %w", err)
		}
	}
	path := fmt.Sprintf("/api/v1/scores/%s/what-if", scenarioName)
	body, err := a.api.Request(http.MethodPost, path, nil, payload)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	cliutil.PrintJSON(body)
	return nil
}
