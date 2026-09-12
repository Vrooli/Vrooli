//go:build ignore

// Seed the isolated Offer Desk database for the routed-database proof.
package main

import (
	"database/sql"
	"log"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	dsn := strings.TrimSpace(os.Getenv("PLAYBOOKS_SQLITE_DSN"))
	if dsn == "" { dsn = strings.TrimSpace(os.Getenv("PLAYBOOKS_SQLITE_PATH")) }
	if dsn == "" { log.Fatal("seed: PLAYBOOKS_SQLITE_DSN or PLAYBOOKS_SQLITE_PATH is required") }
	db, err := sql.Open("sqlite", dsn)
	if err != nil { log.Fatal(err) }
	defer db.Close()
	_, err = db.Exec(`INSERT INTO nodes (id, kind, name, status, trigger_id, created_at, actual_account_id, release_rank, deliverable_class, finish_bar)
VALUES ('bas-routed-smoke-001', 1, 'routed-smoke-001', 4, '', CURRENT_TIMESTAMP, '', 0, 0, 0)
ON CONFLICT(id) DO NOTHING`)
	if err != nil { log.Fatalf("seed routed offer: %v", err) }
	log.Println("seed: inserted routed-smoke-001 into the leased Offer Desk database")
}
