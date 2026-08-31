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
	flag.Parse()
	resolved := *root
	if resolved == "" {
		var err error
		resolved, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	dbPath := *databasePath
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		dbPath = filepath.Join(home, ".vrooli/data/vrooli/react-component-library/react-component-library.db")
	}
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()
	audit, err := versionledger.AuditReleaseHashLedgerWithDatabase(resolved, db)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoded, err := json.MarshalIndent(audit, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}
