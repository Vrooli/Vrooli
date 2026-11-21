// Command reconcile-storage-hijack moves rows that one scenario wrote into
// another scenario's database back to the database that owns them.
//
// It exists because scenarios used to resolve their SQLite path from a generic
// environment variable above their own identity, so a supervisor that restarted
// a sick scenario handed it the supervisor's database. The resolution defect is
// fixed in api-core/storage; this repairs the data it produced.
//
// Usage:
//
//	reconcile-storage-hijack \
//	  --source scenarios/vrooli-autoheal/data/autoheal.sqlite \
//	  --target scenarios/test-genie/data/test-genie.db \
//	  --tables suite_executions,suite_execution_stages,suite_execution_phases
//
// It reports what it would move and writes NOTHING unless --apply is passed.
// It never deletes from the source: run it, verify the counts, restart the
// owning scenario onto its own database, and only then consider removing the
// rows from the source by hand. Deleting first would leave the scenario still
// writing into the source, so the rows would simply come back.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	"storage-manager/internal/migration/reconcile"
)

func main() {
	source := flag.String("source", "", "path to the database rows were wrongly written into")
	target := flag.String("target", "", "path to the database that owns the rows")
	tables := flag.String("tables", "", "comma-separated tables to move, parents before children")
	apply := flag.Bool("apply", false, "actually write; without it nothing is modified")
	flag.Parse()

	if strings.TrimSpace(*source) == "" || strings.TrimSpace(*target) == "" || strings.TrimSpace(*tables) == "" {
		fmt.Fprintln(os.Stderr, "--source, --target and --tables are all required")
		flag.Usage()
		os.Exit(2)
	}

	list := make([]string, 0)
	for _, t := range strings.Split(*tables, ",") {
		if t = strings.TrimSpace(t); t != "" {
			list = append(list, t)
		}
	}

	dsn, err := storage.SQLiteDSNAt(*target, storage.SQLiteTuning{})
	if err != nil {
		fail("build target dsn: %v", err)
	}
	// Opened through api-core/database rather than sql.Open so this maintenance
	// path gets the same retry and backoff behaviour as a scenario's own
	// connection. The pool is capped at one connection because the ATTACH below
	// is per-connection state: a second connection would not see the attached
	// source at all.
	routed, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		fail("open target: %v", err)
	}
	defer routed.Close()

	res, err := reconcile.Run(context.Background(), routed.Primary(), reconcile.Options{
		SourcePath: *source,
		Tables:     list,
		DryRun:     !*apply,
	})
	if err != nil {
		fail("reconcile: %v", err)
	}

	mode := "DRY RUN — nothing was written"
	if *apply {
		mode = "APPLIED"
	}
	fmt.Printf("%s\n  source: %s\n  target: %s\n\n", mode, *source, *target)
	fmt.Printf("%-28s %8s %8s %8s %8s %9s %10s\n", "TABLE", "SOURCE", "BEFORE", "AFTER", "MOVED", "IDENTICAL", "CONFLICTED")
	for _, t := range res.Tables {
		if t.Skipped {
			fmt.Printf("%-28s %8s  skipped: %s\n", t.Table, "-", t.Reason)
			continue
		}
		fmt.Printf("%-28s %8d %8d %8d %8d %9d %10d\n",
			t.Table, t.SourceRows, t.TargetRowsBefore, t.TargetRowsAfter, t.Inserted, t.Identical, t.Conflicted)
		if len(t.DroppedColumns) > 0 {
			fmt.Printf("%-28s   schema drift, columns not copied: %s\n", "", strings.Join(t.DroppedColumns, ", "))
		}
		if t.Conflicted > 0 {
			fmt.Printf("%-28s   %d row(s) collide with a DIFFERENT target row on a unique constraint and need a human decision\n", "", t.Conflicted)
		}
	}
	fmt.Printf("\ntotal rows moved: %d\n", res.TotalInserted())
	if !*apply {
		fmt.Println("re-run with --apply to move them")
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
