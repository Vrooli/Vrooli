package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
	"react-component-library/internal/versionledger"
)

func main() {
	root := flag.String("root", "", "repository root (defaults to the current directory)")
	databasePath := flag.String("database", "", "SQLite database path (defaults to the Vrooli scenario database)")
	apply := flag.Bool("apply", false, "apply only safe orphan pruning and mirror-confirmed re-recording")
	acceptCurrent := flag.Bool("accept-current", false, "accept current authored bytes after independent provenance review")
	pruneMissing := flag.Bool("prune-missing", false, "prune authored rows whose files were deliberately removed after claim-retirement review")
	flag.Parse()
	resolvedRoot := *root
	if resolvedRoot == "" {
		var err error
		resolvedRoot, err = os.Getwd()
		if err != nil {
			fail(err)
		}
	}
	dbPath := *databasePath
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fail(err)
		}
		dbPath = filepath.Join(home, ".vrooli/data/vrooli/react-component-library/react-component-library.db")
	}
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		fail(err)
	}
	defer db.Close()
	repair, err := versionledger.RepairReleaseHashLedgerWithOptions(resolvedRoot, db, *apply, versionledger.LedgerRepairOptions{AcceptCurrent: *acceptCurrent, PruneMissing: *pruneMissing})
	if err != nil {
		fail(err)
	}
	encoded, err := json.MarshalIndent(repair, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(encoded))
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
