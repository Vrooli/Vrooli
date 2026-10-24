package graph

import (
	"fmt"
	"strings"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

// runRebuild drives `graph rebuild [scenario] [--apply] [--force]`, the manual
// override for the unified-graph ingest. It is dry-run by default; --apply
// persists. Without a scenario it rebuilds the whole fleet.
func runRebuild(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("graph rebuild")
	var scenario string
	var apply bool
	var force bool
	var jsonOutput bool
	fs.StringVar(&scenario, "scenario", "", "Rebuild a single scenario's edges")
	fs.BoolVar(&apply, "apply", false, "Persist the rebuilt edges (default: dry-run)")
	fs.BoolVar(&force, "force", false, "Ignore freshness gating (reserved; full rebuild always re-ingests)")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) > 1 {
		return fmt.Errorf("usage: %s graph rebuild [scenario] [--apply] [--force] [--json]", support.AppName)
	}
	if len(positionals) == 1 {
		if strings.TrimSpace(scenario) != "" {
			return fmt.Errorf("provide scenario either positionally or with --scenario, not both")
		}
		scenario = positionals[0]
	}

	query := support.BuildQuery(map[string]string{
		"apply":    support.BoolWord(apply, "true", "false"),
		"scenario": scenario,
	})
	body, err := core.Request("POST", "/graph/rebuild", query, nil)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintAPIJSON(body)
	}

	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	summary := []string{
		fmt.Sprintf("Scope: %s", support.String(resp["scope"])),
		fmt.Sprintf("Applied: %t", apply),
	}
	if scenario != "" {
		summary = append(summary, fmt.Sprintf("Scenario: %s", scenario))
		summary = append(summary, fmt.Sprintf("Edges: %d", support.Int(resp["edges_persisted"])))
	} else {
		summary = append(summary,
			fmt.Sprintf("Scenarios analyzed: %d", support.Int(resp["scenarios_analyzed"])),
			fmt.Sprintf("Edges: %d (scenario=%d resource=%d)",
				support.Int(resp["edges_persisted"]),
				support.Int(resp["scenario_edges"]),
				support.Int(resp["resource_edges"])),
		)
	}
	report := cliapp.MutationReport{
		Result:  summary,
		Changes: rebuildChanges(apply, resp),
		NextCommand: []string{
			fmt.Sprintf("%s graph rebuild --apply", support.AppName),
			fmt.Sprintf("%s graph sweeper status", support.AppName),
		},
	}
	return support.PrintMutation(false, report, nil)
}

func rebuildChanges(apply bool, resp map[string]interface{}) []string {
	if !apply {
		return []string{"Dry-run: no edges persisted. Re-run with --apply to write."}
	}
	changes := []string{"Persisted unified graph edges."}
	if degraded := support.Strings(resp["degraded_sources"]); len(degraded) > 0 {
		changes = append(changes, fmt.Sprintf("Degraded sources (last-good retained): %s", strings.Join(degraded, ", ")))
	}
	return changes
}

// runSweeper drives `graph sweeper status [--json]`.
func runSweeper(core *cliapp.ScenarioApp, args []string) error {
	if len(args) == 0 || !strings.EqualFold(strings.TrimSpace(args[0]), "status") {
		return fmt.Errorf("usage: %s graph sweeper status [--json]", support.AppName)
	}
	fs := support.NewFlagSet("graph sweeper status")
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args[1:]); err != nil {
		return err
	}

	body, err := core.Get("/graph/sweeper/status", nil)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintAPIJSON(body)
	}

	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	edges := support.Map(resp["edges"])
	summary := []string{
		fmt.Sprintf("Enabled: %t", support.Bool(resp["enabled"])),
		fmt.Sprintf("Interval: %s", support.String(resp["interval"])),
		fmt.Sprintf("Concurrency: %d", support.Int(resp["concurrency"])),
		fmt.Sprintf("Cycle budget: %s", support.String(resp["cycle_budget"])),
		fmt.Sprintf("Breaker: %s", support.String(resp["breaker_state"])),
		fmt.Sprintf("Edges: %d (scenario=%d resource=%d stale=%d)",
			support.Int(edges["total"]),
			support.Int(edges["scenario"]),
			support.Int(edges["resource"]),
			support.Int(edges["stale"])),
	}
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Last cycle",
		Results:        lastCycleRows(resp["last_cycle"]),
		RetrievalHints: []string{
			fmt.Sprintf("%s graph sweeper status --json", support.AppName),
			fmt.Sprintf("%s graph rebuild --apply", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
}

func lastCycleRows(value interface{}) []string {
	cycle := support.Map(value)
	if len(cycle) == 0 {
		return []string{"No sweep cycle has completed yet."}
	}
	return []string{
		fmt.Sprintf("scanned=%d skipped_fresh=%d ingested=%d degraded=%d failed=%d budget_hit=%t",
			support.Int(cycle["scanned"]),
			support.Int(cycle["skipped_fresh"]),
			support.Int(cycle["ingested"]),
			support.Int(cycle["degraded"]),
			support.Int(cycle["failed"]),
			support.Bool(cycle["budget_hit"])),
	}
}
