// Command prune-facet-views removes the legacy "rule" and "entities" facet
// texts and their embeddings. Entries written before the single-view change
// carry three facet texts that differ only by a fixed prefix on the body;
// recall reads one vector per entry now, so the extra rows are dead weight.
//
// It is a dry run unless --apply is given: the default prints exactly what
// would be removed and changes nothing. The topic row and topic vector of every
// entry are never touched, and an entry with no topic row keeps every row it
// has. Re-running it is safe.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"source-ledger/internal/journal"

	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

func main() {
	dbPath := flag.String("db", "", "path of the Source Ledger SQLite database to prune (required)")
	apply := flag.Bool("apply", false, "delete the measured rows; without it nothing is changed")
	batch := flag.Int("batch", 500, "facet texts deleted per transaction")
	flag.Parse()

	// A standalone process does not inherit the scenario's storage resolution,
	// so the path is explicit and must already exist: a typo must not create an
	// empty database that then reports nothing to prune.
	if *dbPath == "" {
		fatal(fmt.Errorf("--db is required"))
	}
	if _, err := os.Stat(*dbPath); err != nil {
		fatal(err)
	}
	dsn, err := storage.SQLiteDSNAt(*dbPath, storage.SQLiteTuning{})
	if err != nil {
		fatal(err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx := context.Background()

	planned, err := journal.MeasureRedundantFacetViews(ctx, db)
	if err != nil {
		fatal(err)
	}
	report := struct {
		Applied bool                         `json:"applied"`
		Planned journal.RedundantFacetViews  `json:"planned"`
		Removed *journal.RedundantFacetViews `json:"removed,omitempty"`
	}{Applied: *apply, Planned: planned}
	if *apply {
		removed, err := journal.PruneRedundantFacetViews(ctx, db, *batch)
		report.Removed = &removed
		if err != nil {
			emit(report)
			fatal(err)
		}
	}
	emit(report)
}

func emit(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "prune-facet-views:", err)
	os.Exit(1)
}
