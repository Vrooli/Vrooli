package store

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "vrooli-events"

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Store",
		Commands: []cliapp.Command{
			{Name: "stats", NeedsAPI: true, Description: "Show event store statistics", Run: func(args []string) error { return runStats(core, args) }},
		},
	}
}

func runStats(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := core.GetRoot("/health", nil)
	if err != nil {
		return err
	}

	var health struct {
		Status      string `json:"status"`
		Subscribers int    `json:"subscribers"`
		Store       struct {
			TotalEvents       int     `json:"totalEvents"`
			TotalPayloadBytes float64 `json:"totalPayloadBytes"`
		} `json:"store"`
	}
	if err := json.Unmarshal(body, &health); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Store status: " + health.Status,
			fmt.Sprintf("Subscribers: %d", health.Subscribers),
		},
		ResultsHeading: "Store Metrics",
		Results: []string{
			fmt.Sprintf("Events: %d", health.Store.TotalEvents),
			fmt.Sprintf("Payload size: %.1f MB", health.Store.TotalPayloadBytes/1024/1024),
		},
		RetrievalHints: []string{
			cliName + " query --limit 20",
			cliName + " status",
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
