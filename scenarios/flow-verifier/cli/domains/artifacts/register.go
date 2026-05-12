// Package artifacts is the CLI's codegen-lifecycle command surface.
// `status` reports what's on disk; `generate` (re)materialises a flow's
// generated/ tree; `clear` removes it. All three drive
// api/internal/artifacts in-process so the CLI keeps working when the
// scenario API isn't started.
package artifacts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/cli-core/cliapp"
	_ "modernc.org/sqlite"

	"flow-verifier/internal/artifacts"
	"flow-verifier/internal/clock"
	localdb "flow-verifier/internal/database"
	"flow-verifier/internal/pipeline"
	"flow-verifier/internal/runs"
)

// Register returns the `artifacts` subcommand group.
func Register(_ *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	rootFlag := cliapp.Flag{Name: "root", Description: "Repository root to scan (default: cwd)", Default: "."}
	flowFlag := cliapp.Flag{Name: "flow", Description: "Flow id to target"}
	scenarioFlag := cliapp.Flag{Name: "scenario", Description: "Scenario id (path) to target every flow inside"}
	allFlag := cliapp.Flag{Name: "all", Description: "Apply to every discovered flow under --root", Default: "false"}
	formatFlag := cliapp.Flag{Name: "format", Description: "Output format: text (default) or json", Default: "text"}
	yesFlag := cliapp.Flag{Name: "yes", Description: "Skip the confirmation prompt for bulk clears", Default: "false"}

	return cliapp.SubcommandGroup{
		Name:        "artifacts",
		Description: "Inspect, generate, or clear a flow's generated/ tree",
		Subcommands: []cliapp.Command{
			{
				Name:        "status",
				Description: "Inspect the on-disk generated/ tree for one flow or every flow under root",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag, flowFlag, scenarioFlag, allFlag, formatFlag}},
				RunCtx:      runStatus,
			},
			{
				Name:        "generate",
				Description: "Generate or regenerate one flow's artifacts (or every flow's with --scenario or --all)",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag, flowFlag, scenarioFlag, allFlag, formatFlag}},
				RunCtx:      runGenerate,
			},
			{
				Name:        "clear",
				Description: "Remove one flow's generated/ tree (--scenario / --all require --yes)",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag, flowFlag, scenarioFlag, allFlag, yesFlag, formatFlag}},
				RunCtx:      runClear,
			},
		},
	}
}

func runStatus(ctx cliapp.RunContext) error {
	svc := artifacts.NewService(nil) // status doesn't generate, so no generator needed
	root := ctx.Flag("root")
	flow := ctx.Flag("flow")
	format := ctx.Flag("format")

	if isAll(ctx) || ctx.Flag("scenario") != "" {
		reports, err := svc.StatusForScenario(root)
		if err != nil {
			return err
		}
		return printReports(ctx, reports, format)
	}
	if flow == "" {
		return fmt.Errorf("--flow is required (or pass --all / --scenario)")
	}
	rep, err := svc.Status(root, flow)
	if err != nil {
		return err
	}
	return printReports(ctx, []artifacts.Report{rep}, format)
}

func runGenerate(ctx cliapp.RunContext) error {
	bg := context.Background()
	gen, db, err := openGenerator(bg)
	if err != nil {
		return err
	}
	defer db.Close()
	svc := artifacts.NewService(gen)
	root := ctx.Flag("root")
	flow := ctx.Flag("flow")
	format := ctx.Flag("format")

	if isAll(ctx) || ctx.Flag("scenario") != "" {
		reports, err := svc.GenerateForScenario(bg, root)
		if err != nil {
			return err
		}
		return printReports(ctx, reports, format)
	}
	if flow == "" {
		return fmt.Errorf("--flow is required (or pass --all / --scenario)")
	}
	rep, err := svc.Generate(bg, root, flow)
	if err != nil {
		return err
	}
	return printReports(ctx, []artifacts.Report{rep}, format)
}

func runClear(ctx cliapp.RunContext) error {
	svc := artifacts.NewService(nil)
	root := ctx.Flag("root")
	flow := ctx.Flag("flow")
	format := ctx.Flag("format")
	bulk := isAll(ctx) || ctx.Flag("scenario") != ""
	if bulk && !truthy(ctx.Flag("yes")) {
		return fmt.Errorf("bulk clear requires --yes")
	}

	if bulk {
		results, err := svc.ClearForScenario(root)
		if err != nil {
			return err
		}
		return printClearResults(ctx, results, format)
	}
	if flow == "" {
		return fmt.Errorf("--flow is required (or pass --all / --scenario)")
	}
	res, err := svc.Clear(root, flow)
	if err != nil {
		return err
	}
	return printClearResults(ctx, []artifacts.ClearResult{res}, format)
}

func isAll(ctx cliapp.RunContext) bool { return truthy(ctx.Flag("all")) }

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y", "on":
		return true
	}
	return false
}

func printReports(ctx cliapp.RunContext, reports []artifacts.Report, format string) error {
	out := ctx.Stdout()
	if format == "json" {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(reports)
	}
	for _, r := range reports {
		fmt.Fprintf(out, "%s\n  status %s\n  generatedDir %s\n", r.FlowID, r.Status, r.GeneratedDir)
		if len(r.Missing) > 0 {
			fmt.Fprintf(out, "  missing %s\n", strings.Join(r.Missing, ", "))
		}
	}
	return nil
}

func printClearResults(ctx cliapp.RunContext, results []artifacts.ClearResult, format string) error {
	out := ctx.Stdout()
	if format == "json" {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	for _, r := range results {
		fmt.Fprintf(out, "%s removed %d file(s)\n", r.FlowID, len(r.Removed))
		for _, f := range r.Removed {
			fmt.Fprintf(out, "  %s\n", f)
		}
	}
	return nil
}

// pipelineGenerator wraps pipeline.Verify(ModeGenerate) with the runs
// recorder. Mirrors the API handler so generate runs land in the same
// history table whether invoked over HTTP or via the CLI.
type pipelineGenerator struct {
	rec pipeline.Recorder
}

func (g pipelineGenerator) Generate(ctx context.Context, root, flowID string) error {
	_, err := pipeline.Verify(ctx, pipeline.VerifyOptions{
		Root:     root,
		FlowID:   flowID,
		Mode:     pipeline.ModeGenerate,
		Recorder: g.rec,
	})
	return err
}

type runsRecorder struct{ svc *runs.Service }

func (r *runsRecorder) Record(ctx context.Context, e pipeline.RunEntry) error {
	_, err := r.svc.Record(ctx, runs.Run{
		FlowID:           e.FlowID,
		FlowPath:         e.FlowPath,
		Root:             e.Root,
		Mode:             runs.ModeRun,
		Status:           runs.Status(e.Status),
		Output:           e.Output,
		ErrorMessage:     e.ErrorMessage,
		FailureReason:    e.FailureReason,
		MissingArtifacts: e.MissingArtifacts,
		StartedAt:        e.StartedAt,
		FinishedAt:       e.FinishedAt,
	})
	return err
}

func openGenerator(ctx context.Context) (artifacts.Generator, *sql.DB, error) {
	dsn, err := localdb.DefaultDSN()
	if err != nil {
		return nil, nil, err
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := apidb.EnsureSchemas(ctx, db,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(runs.Schema),
	); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("ensure runs schema: %w", err)
	}
	rec := &runsRecorder{svc: runs.NewService(runs.NewSQLiteRepository(db, clock.System{}))}
	return pipelineGenerator{rec: rec}, db, nil
}
