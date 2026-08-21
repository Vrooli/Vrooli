// Command import-review-evidence performs the one-time, bounded migration from
// durable review round files into the canonical evidence ledger. It is safe to
// repeat: observation identifiers and the parity audit are deterministic.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/evidence"
	"swarm-manager/internal/runtimepaths"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

func main() {
	apply := flag.Bool("apply", false, "write observations and a parity audit; required for migration")
	flag.Parse()

	dataRoot, err := runtimepaths.DataPath("")
	if err != nil {
		fatal(fmt.Errorf("resolve data root: %w", err))
	}
	sources, err := evidence.LoadReviewRoundSources(dataRoot)
	if err != nil {
		fatal(err)
	}
	evidenceCount := 0
	for _, source := range sources {
		evidenceCount += len(source.Round.Evidence)
	}
	if !*apply {
		fmt.Printf("review-evidence import dry run: %d rounds, %d evidence items. Re-run with --apply to write the ledger.\n", len(sources), evidenceCount)
		return
	}

	dsn, err := eventDBDSN()
	if err != nil {
		fatal(err)
	}
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: dsn, MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		fatal(fmt.Errorf("open evidence ledger: %w", err))
	}
	defer db.Close()
	if err := eventlog.NewSQLiteRepository(db).InitSchema(context.Background()); err != nil {
		fatal(fmt.Errorf("initialize evidence ledger schema: %w", err))
	}
	audit, err := evidence.NewLedger(db).ImportReviewRounds(context.Background(), sources)
	if err != nil {
		fatal(fmt.Errorf("import review evidence: %w", err))
	}
	fmt.Printf("review-evidence import complete: %d rounds, %d source evidence, %d projected evidence, parity=%t\n", len(sources), audit.SourceCount, audit.ProjectionCount, audit.SourceCount == audit.ProjectionCount)
}

func eventDBDSN() (string, error) {
	if path := os.Getenv("SWARM_MANAGER_SQLITE_PATH"); path != "" {
		return storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	}
	path, err := runtimepaths.DataPath("events.db")
	if err != nil {
		return "", err
	}
	return storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "import-review-evidence:", err)
	os.Exit(1)
}
