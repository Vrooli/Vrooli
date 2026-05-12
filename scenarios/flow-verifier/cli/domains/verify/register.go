// Package verify is the CLI's verification command surface. `run`
// regenerates artifacts; `check` runs the full pipeline + lint and
// fails on staleness. Both drive api/internal/pipeline in-process and
// record one history row per flow via the runs domain.
package verify

import (
	"context"
	"database/sql"
	"fmt"

	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/cli-core/cliapp"
	// modernc.org/sqlite registers itself as the "sqlite" driver via init();
	// the CLI opens the verification-history DB directly and needs the driver
	// loaded even though it doesn't reference an exported symbol.
	_ "modernc.org/sqlite"

	"flow-verifier/internal/clock"
	localdb "flow-verifier/internal/database"
	"flow-verifier/internal/pipeline"
	"flow-verifier/internal/runs"
)

// Register returns the `verify` subcommand group.
func Register(_ *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	rootFlag := cliapp.Flag{Name: "root", Description: "Repository root to scan (default: cwd)", Default: "."}
	flowFlag := cliapp.Flag{Name: "flow", Description: "Restrict to a single flow id"}
	return cliapp.SubcommandGroup{
		Name:        "verify",
		Description: "Generate and check formal-temporal-model artifacts via Quint",
		Subcommands: []cliapp.Command{
			{
				Name:        "run",
				Description: "Regenerate artifacts (model.qnt, runtime, replay helper) for every flow",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag, flowFlag}},
				RunCtx:      runRun,
			},
			{
				Name:        "check",
				Description: "Verify every flow: lint + freshness + Quint check (fatal lint, no --no-lint)",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag, flowFlag}},
				RunCtx:      runCheck,
			},
		},
	}
}

func runRun(ctx cliapp.RunContext) error {
	return invokeVerify(ctx, pipeline.ModeGenerate)
}

func runCheck(ctx cliapp.RunContext) error {
	return invokeVerify(ctx, pipeline.ModeCheck)
}

func invokeVerify(ctx cliapp.RunContext, mode pipeline.Mode) error {
	bg := context.Background()
	rec, db, err := openRecorder(bg)
	if err != nil {
		return err
	}
	defer db.Close()

	_, runErr := pipeline.Verify(bg, pipeline.VerifyOptions{
		Root:     ctx.Flag("root"),
		FlowID:   ctx.Flag("flow"),
		Mode:     mode,
		Stdout:   ctx.Stdout(),
		Recorder: rec,
	})
	return runErr
}

type runsRecorder struct{ svc *runs.Service }

func (r *runsRecorder) Record(ctx context.Context, e pipeline.RunEntry) error {
	_, err := r.svc.Record(ctx, runs.Run{
		FlowID:           e.FlowID,
		FlowPath:         e.FlowPath,
		Root:             e.Root,
		Mode:             pipelineModeToRunsMode(e.Mode),
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

func pipelineModeToRunsMode(m pipeline.Mode) runs.Mode {
	if m == pipeline.ModeGenerate {
		return runs.ModeRun
	}
	return runs.ModeCheck
}

func openRecorder(ctx context.Context) (pipeline.Recorder, *sql.DB, error) {
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
	return &runsRecorder{svc: runs.NewService(runs.NewSQLiteRepository(db, clock.System{}))}, db, nil
}
