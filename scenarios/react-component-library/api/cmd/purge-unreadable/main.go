package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
	"react-component-library/internal/versionledger"
)

func main() {
	libraryID := flag.String("library-id", "", "exact library id")
	version := flag.String("version", "", "exact version")
	confirm := flag.Bool("confirm", false, "confirm the named destructive repair")
	root := flag.String("root", "", "repository root (defaults to the current directory)")
	databasePath := flag.String("database", "", "SQLite database path (defaults to the Vrooli scenario database)")
	flag.Parse()
	if *libraryID == "" || *version == "" {
		fail("--library-id and --version are required")
	}
	resolvedRoot := *root
	if resolvedRoot == "" {
		var err error
		resolvedRoot, err = os.Getwd()
		if err != nil {
			fail(err.Error())
		}
	}
	dbPath := *databasePath
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fail(err.Error())
		}
		dbPath = filepath.Join(home, ".vrooli/data/vrooli/react-component-library/react-component-library.db")
	}
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		fail(err.Error())
	}
	defer db.Close()
	repo := versionledger.NewRepository(db, filepath.Join(resolvedRoot, "scenarios/react-component-library/library"))
	if err := repo.PurgeUnreadableVersion(context.Background(), *libraryID, *version, *confirm); err != nil {
		fail(err.Error())
	}
	fmt.Printf("purged unreadable version %s@%s\n", *libraryID, *version)
}

func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
