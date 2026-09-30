package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) runCohortReport(args []string) error {
	fs := flag.NewFlagSet("run cohort-report", flag.ContinueOnError)
	runIDs := fs.String("run-ids", "", "comma-separated run UUIDs (1-100)")
	cohort := fs.String("cohort", "", "durable cohort name")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*runIDs) == "" && strings.TrimSpace(*cohort) == "" {
		return fmt.Errorf("usage: agent-manager run cohort-report (--run-ids <id1,id2> | --cohort name) [--json]")
	}
	var body []byte
	var err error
	if *cohort != "" {
		body, err = a.services.Runs.CohortReportByName(*cohort)
	} else {
		body, err = a.services.Runs.CohortReport(*runIDs)
	}
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	var report struct {
		ClassifierVersion string                         `json:"classifierVersion"`
		RunIDs            []string                       `json:"runIds"`
		Availability      struct{ State, Reason string } `json:"availability"`
		Signals           []struct {
			Kind                 string
			Count, Impact        int
			Confidence           string
			RepresentativeRunIDs []string `json:"representativeRunIds"`
		} `json:"signals"`
	}
	if err := json.Unmarshal(body, &report); err != nil {
		return fmt.Errorf("decode cohort report: %w", err)
	}
	fmt.Printf("Cohort: %d runs | classifier: %s | evidence: %s\n", len(report.RunIDs), report.ClassifierVersion, report.Availability.State)
	if report.Availability.Reason != "" {
		fmt.Printf("Evidence reason: %s\n", report.Availability.Reason)
	}
	for _, signal := range report.Signals {
		fmt.Printf("- %s: count=%d impact=%d confidence=%s evidence=%s\n", signal.Kind, signal.Count, signal.Impact, signal.Confidence, strings.Join(signal.RepresentativeRunIDs, ","))
	}
	return nil
}

func (a *App) runCohortCompare(args []string) error {
	fs := flag.NewFlagSet("run cohort-compare", flag.ContinueOnError)
	left := fs.String("left-filter-json", "{}", "JSON invocation filter for the left population")
	right := fs.String("right-filter-json", "{}", "JSON invocation filter for the right population")
	changeBinding := fs.String("change-binding", "", "change label, plan slug, or commit range")
	limit := fs.Int("limit", 100, "maximum fingerprint signals")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	var leftFilter, rightFilter map[string]any
	if err := json.Unmarshal([]byte(*left), &leftFilter); err != nil {
		return fmt.Errorf("decode left filter: %w", err)
	}
	if err := json.Unmarshal([]byte(*right), &rightFilter); err != nil {
		return fmt.Errorf("decode right filter: %w", err)
	}
	payload, err := json.Marshal(map[string]any{"left": leftFilter, "right": rightFilter, "changeBinding": *changeBinding})
	if err != nil {
		return err
	}
	body, err := a.services.Runs.CompareEpisodeCohorts(payload, *limit)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(string(body))
	return nil
}

func (a *App) runGoalCohort(args []string) error {
	fs := flag.NewFlagSet("run goal-cohort", flag.ContinueOnError)
	cohort := fs.String("cohort", "", "durable cohort name")
	limit := fs.Int("limit", 100, "maximum runs to score")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*cohort) == "" {
		return fmt.Errorf("usage: agent-manager run goal-cohort --cohort name [--json]")
	}
	if *limit < 1 || *limit > 5000 {
		return fmt.Errorf("--limit must be between 1 and 5000")
	}
	body, err := a.services.Runs.GoalCohort(*cohort, *limit)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else {
		fmt.Println(string(body))
	}
	return nil
}

func (a *App) runEpisodeTrend(args []string) error {
	fs := flag.NewFlagSet("measures episode-trend", flag.ContinueOnError)
	from := fs.String("from", "", "RFC3339 inclusive time")
	to := fs.String("to", "", "RFC3339 exclusive time")
	bucket := fs.String("bucket", "24h", "positive Go duration bucket")
	limit := fs.Int("limit", 100, "maximum buckets")
	cohort := fs.String("cohort", "", "durable cohort name")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	values := url.Values{"bucket": []string{*bucket}, "limit": []string{strconv.Itoa(*limit)}}
	if *from != "" {
		values.Set("from", *from)
	}
	if *to != "" {
		values.Set("to", *to)
	}
	if *cohort != "" {
		values.Set("cohort", *cohort)
	}
	body, err := a.services.Runs.EpisodeTrend(values)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(string(body))
	return nil
}

func (a *App) runPublishRecurringFriction(args []string) error {
	fs := flag.NewFlagSet("run publish-recurring-friction", flag.ContinueOnError)
	cap := fs.Int("cap", 25, "daily filing cap")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Runs.PublishRecurringFriction(url.Values{"cap": []string{strconv.Itoa(*cap)}})
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(string(body))
	return nil
}
