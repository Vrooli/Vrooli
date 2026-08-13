package main

import (
	"context"
	"database/sql"
	"flag"
	"log"

	"react-component-library/internal/components"
	"react-component-library/internal/componenttests"

	apidb "github.com/vrooli/api-core/database"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := flag.String("db", "../data/react-component-library.db", "SQLite database path")
	flag.Parse()
	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(components.Schema)); err != nil {
		log.Fatal(err)
	}
	if err := componenttests.BackfillRollups(context.Background(), db); err != nil {
		log.Fatal(err)
	}
	log.Println("backfilled component version test rollups")
}
