package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) runCohortReport(args []string) error {
	fs := flag.NewFlagSet("run cohort-report", flag.ContinueOnError)
	runIDs := fs.String("run-ids", "", "comma-separated run UUIDs (1-100)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*runIDs) == "" {
		return fmt.Errorf("usage: agent-manager run cohort-report --run-ids <id1,id2> [--json]")
	}
	body, err := a.services.Runs.CohortReport(*runIDs)
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
