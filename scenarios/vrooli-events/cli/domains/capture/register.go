// Package capture exposes scenario-authored receipt declaration reconciliation.
package capture

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Receipt Capture",
		Commands: []cliapp.Command{
			{Name: "capture-preview", NeedsAPI: true, Description: "Validate and preview a scenario's receipt-capture declaration", Run: func(args []string) error { return reconcile(core, args, true) }},
			{Name: "capture-reconcile", NeedsAPI: true, Description: "Validate and reconcile a scenario's receipt-capture declaration", Run: func(args []string) error { return reconcile(core, args, false) }},
		},
	}
}

func reconcile(core *cliapp.ScenarioApp, args []string, preview bool) error {
	fs := flag.NewFlagSet("capture-reconcile", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "declaring scenario slug (required)")
	validateOnly := fs.Bool("validate-only", false, "validate declaration without writes")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *scenario == "" {
		return fmt.Errorf("usage: vrooli-events %s --scenario SCENARIO [--validate-only] [--json]", map[bool]string{true: "capture-preview", false: "capture-reconcile"}[preview])
	}
	body, err := core.Request("POST", "/receipt-capture-policies/reconcile", nil, map[string]any{"scenario": *scenario, "dryRun": preview, "validateOnly": *validateOnly})
	if err != nil {
		return err
	}
	var response struct {
		Scenario  string `json:"scenario"`
		Validated bool   `json:"validated"`
		Policies  int    `json:"policies"`
		Created   int    `json:"created"`
		Updated   int    `json:"updated"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse receipt reconciliation response: %w", err)
	}
	report := cliapp.MutationReport{Result: []string{"Receipt capture declaration validated", "Scenario: " + response.Scenario}, Changes: []string{fmt.Sprintf("Policies: %d", response.Policies)}}
	if !preview && !*validateOnly {
		report.Result = []string{"Receipt capture declaration reconciled", "Scenario: " + response.Scenario}
		report.Changes = append(report.Changes, fmt.Sprintf("Created: %d", response.Created), fmt.Sprintf("Updated: %d", response.Updated))
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
