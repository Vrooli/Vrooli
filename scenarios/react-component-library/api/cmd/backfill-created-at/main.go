package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	"react-component-library/internal/components"
)

func main() {
	dbPath := flag.String("db", "scenarios/react-component-library/data/react-component-library.db", "SQLite database path")
	repoRoot := flag.String("repo", ".", "repository root used for git history")
	flag.Parse()

	dsn, err := storage.SQLiteDSNAt(*dbPath, storage.SQLiteTuning{})
	if err != nil {
		log.Fatalf("build database dsn: %v", err)
	}
	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	count, err := components.BackfillCreatedAt(context.Background(), db.Primary(), *repoRoot)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stdout, "backfilled created_at for %d versions\n", count)
}
