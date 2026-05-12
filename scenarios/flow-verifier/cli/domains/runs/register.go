// Package runs is the CLI's verification-history command surface, backed
// by the SQLite store the runs API domain owns. Opens the same database
// file the API uses (resolved via internal/database.DefaultDSN), reads
// in-process via runs.Service.
package runs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/cli-core/cliapp"
	// modernc.org/sqlite registers itself as the "sqlite" driver via init();
	// the CLI opens the verification-history DB directly and needs the driver
	// loaded even though it doesn't reference an exported symbol.
	_ "modernc.org/sqlite"

	"flow-verifier/internal/clock"
	localdb "flow-verifier/internal/database"
	"flow-verifier/internal/runs"
)

// Register returns the `runs` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	_ = core
	return cliapp.SubcommandGroup{
		Name:        "runs",
		Description: "Browse persisted verification run history",
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List recent verification runs",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "flow", Description: "Restrict to a single flow id"},
						{Name: "limit", Description: "Maximum rows to return (default 50)"},
					},
				},
				RunCtx: runList,
			},
			{
				Name:        "show",
				Description: "Show one verification run (with counterexample on failure)",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "run-id", Required: true, Description: "Run id"}},
				},
				RunCtx: runShow,
			},
		},
	}
}

func openService(ctx context.Context) (*runs.Service, *sql.DB, error) {
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
	return runs.NewService(runs.NewSQLiteRepository(db, clock.System{})), db, nil
}

func runList(ctx cliapp.RunContext) error {
	bg := context.Background()
	svc, db, err := openService(bg)
	if err != nil {
		return err
	}
	defer db.Close()

	q := runs.ListQuery{FlowID: ctx.Flag("flow")}
	if raw := ctx.Flag("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %q", raw)
		}
		q.Limit = n
	}
	rows, err := svc.List(bg, q)
	if err != nil {
		return err
	}
	out := ctx.Stdout()
	if len(rows) == 0 {
		fmt.Fprintln(out, "no verification runs recorded")
		return nil
	}
	for _, r := range rows {
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n",
			r.FinishedAt.Format("2006-01-02T15:04:05Z"),
			r.Status,
			r.Mode,
			r.FlowID,
			r.ID,
		)
	}
	return nil
}

func runShow(ctx cliapp.RunContext) error {
	bg := context.Background()
	svc, db, err := openService(bg)
	if err != nil {
		return err
	}
	defer db.Close()

	id := ctx.Positional("run-id")
	row, err := svc.Get(bg, id)
	if err != nil {
		var nf runs.ErrNotFound
		if errors.As(err, &nf) {
			return fmt.Errorf("unknown run id %s", id)
		}
		return err
	}
	out := ctx.Stdout()
	fmt.Fprintf(out, "id:         %s\n", row.ID)
	fmt.Fprintf(out, "flow:       %s\n", row.FlowID)
	fmt.Fprintf(out, "path:       %s\n", row.FlowPath)
	fmt.Fprintf(out, "root:       %s\n", row.Root)
	fmt.Fprintf(out, "mode:       %s\n", row.Mode)
	fmt.Fprintf(out, "status:     %s\n", row.Status)
	fmt.Fprintf(out, "started:    %s\n", row.StartedAt.Format("2006-01-02T15:04:05.000Z"))
	fmt.Fprintf(out, "finished:   %s\n", row.FinishedAt.Format("2006-01-02T15:04:05.000Z"))
	fmt.Fprintf(out, "durationMs: %d\n", row.DurationMs)
	if row.ErrorMessage != "" {
		fmt.Fprintf(out, "error:      %s\n", row.ErrorMessage)
	}
	if row.Counterexample != "" {
		fmt.Fprintf(out, "counterexample:\n%s\n", row.Counterexample)
	}
	if row.Output != "" {
		fmt.Fprintf(out, "output:\n%s", row.Output)
	}
	return nil
}
