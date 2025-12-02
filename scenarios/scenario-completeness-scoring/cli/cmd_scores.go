package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliutil"
	"scenario-completeness-scoring/cli/format"
)

func (a *App) cmdScores(args []string) error {
	fs := flag.NewFlagSet("scores", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	parsed, body, err := a.services.Scoring.ScoresList()
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
	jsonOutput := cliutil.JSONFlag(fs)
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
	resp, body, err := a.services.Scoring.Score(scenarioName)
	if err != nil {
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

func (a *App) cmdCalculate(args []string) error {
	fs := flag.NewFlagSet("calculate", flag.ContinueOnError)
	source := fs.String("source", "", "Source identifier for history tracking")
	jsonOutput := cliutil.JSONFlag(fs)
	var tags cliutil.StringList
	fs.Var(&tags, "tag", "Tag to associate with snapshot (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: calculate <scenario> [--source name] [--tag value] [--json]")
	}
	scenarioName := fs.Arg(0)
	body, err := a.services.Scoring.Calculate(scenarioName, *source, tags.Values())
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
	jsonOutput := cliutil.JSONFlag(fs)
	var tags cliutil.StringList
	fs.Var(&tags, "tag", "Filter by tag (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: history <scenario> [--limit N] [--source name] [--tag value] [--json]")
	}
	scenarioName := fs.Arg(0)
	body, err := a.services.Scoring.History(scenarioName, *limit, *source, tags.Values())
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
	jsonOutput := cliutil.JSONFlag(fs)
	var tags cliutil.StringList
	fs.Var(&tags, "tag", "Filter by tag")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: trends <scenario> [--limit N] [--source name] [--tag value] [--json]")
	}
	scenarioName := fs.Arg(0)
	body, err := a.services.Scoring.Trends(scenarioName, *limit, *source, tags.Values())
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
	jsonOutput := cliutil.JSONFlag(fs)
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
	body, err := a.services.Scoring.WhatIf(scenarioName, payload)
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
