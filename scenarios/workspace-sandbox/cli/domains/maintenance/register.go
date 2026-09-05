package maintenance

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"workspace-sandbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "maintenance",
		Description: "Inspect driver state and garbage-collection behavior",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "driver", Description: "Show driver information", Run: func(args []string) error { return runDriver(deps, args) }},
			{Name: "gc", Description: "Run sandbox garbage collection", Run: func(args []string) error { return runGC(deps, args, false) }},
			{Name: "preview", Description: "Preview garbage collection candidates", Run: func(args []string) error { return runGC(deps, args, true) }},
		},
	}
}

func runDriver(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("maintenance driver", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := deps.ScenarioApp().Get("/driver/info", nil)
	if err != nil {
		return err
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary:        []string{"Driver information"},
		ResultsHeading: "Details",
		Results:        support.SortedMapLines(payload),
		RetrievalHints: []string{support.CLIName + " status", support.CLIName + " maintenance gc --dry-run"},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGC(deps support.Dependencies, args []string, forceDryRun bool) error {
	fs := flag.NewFlagSet("maintenance gc", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "Preview instead of collecting")
	maxAge := fs.String("max-age", "", "Maximum sandbox age")
	idle := fs.String("idle", "", "Maximum idle time")
	limit := fs.Int("limit", 0, "Maximum number of sandboxes to evaluate")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	effectiveDryRun := forceDryRun || *dryRun
	reqBody := map[string]any{"dryRun": effectiveDryRun}
	policy := map[string]any{}
	if *maxAge != "" {
		policy["maxAge"] = *maxAge
	}
	if *idle != "" {
		policy["idleTimeout"] = *idle
	}
	if len(policy) > 0 {
		reqBody["policy"] = policy
	}
	if *limit > 0 {
		reqBody["limit"] = *limit
	}

	endpoint := "/gc"
	if effectiveDryRun {
		endpoint = "/gc/preview"
	}
	body, err := deps.ScenarioApp().Request("POST", endpoint, nil, reqBody)
	if err != nil {
		return err
	}

	var resp support.GCResult
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	duration := resp.CompletedAt.Sub(resp.StartedAt).Round(time.Millisecond)
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Dry run: %t", resp.DryRun),
			fmt.Sprintf("Sandboxes matched: %d", resp.TotalCollected),
			fmt.Sprintf("Bytes reclaimed: %s", support.FormatBytes(resp.TotalBytesReclaimed)),
			fmt.Sprintf("Completed in: %s", duration),
		},
		Results:        renderGCRows(resp.Collected),
		RetrievalHints: []string{support.CLIName + " sandbox list", support.CLIName + " maintenance gc --limit 10"},
	}
	for _, item := range resp.Errors {
		report.RetrievalHints = append(report.RetrievalHints, fmt.Sprintf("Warning for %s: %s", support.TruncateID(item.SandboxID), item.Error))
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderGCRows(sandboxes []support.GCCollectedSandbox) []string {
	if len(sandboxes) == 0 {
		return nil
	}
	rows := make([]string, 0, len(sandboxes))
	for _, sandbox := range sandboxes {
		rows = append(rows, fmt.Sprintf(
			"%s | %s | size=%s | created=%s | reason=%s",
			support.TruncateID(sandbox.ID),
			sandbox.Status,
			support.FormatBytes(sandbox.SizeBytes),
			sandbox.CreatedAt.Format("2006-01-02 15:04"),
			sandbox.Reason,
		))
	}
	return rows
}
