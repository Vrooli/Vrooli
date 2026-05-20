// Go seed for the routed-e2e path in development-toolchain-validator.
//
// test-genie's playbooks phase invokes this with SQLITE_PATH (or
// POSTGRES_URL) pointing at the per-run test database. The seed opens
// that DSN, inserts a single fixture row, and writes coverage/runtime/
// seed-state.json so BAS can echo the fixture identifier back through
// initial_params.
//
// This file is the routed-path counterpart to a bash seed; it stays Go
// so the same code works on Linux, macOS, and Windows runners.

//go:build ignore

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	dsn := resolveDSN()
	if dsn == "" {
		log.Fatalf("seed: no SQLITE_PATH / SQLITE_DB / POSTGRES_URL / DATABASE_URL in env")
	}

	driver := "sqlite"
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		driver = "postgres"
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		log.Fatalf("seed: open %s: %v", driver, err)
	}
	defer db.Close()

	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS routed_smoke_fixture (id TEXT PRIMARY KEY, marker TEXT NOT NULL)`); err != nil {
		log.Fatalf("seed: create routed_smoke_fixture: %v", err)
	}

	fixtureID := "routed-smoke-001"
	fixtureMarker := "ROUTED_TEST_POOL_VISIBLE"
	if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO routed_smoke_fixture (id, marker) VALUES (?, ?)`, fixtureID, fixtureMarker); err != nil {
		log.Fatalf("seed: insert fixture: %v", err)
	}

	// Write coverage/runtime/seed-state.json so the playbooks runner picks
	// up these values and forwards them to BAS via initial_params. The
	// path follows test-genie's seed-state convention.
	seedState := map[string]any{
		"routed_smoke_fixture_id":     fixtureID,
		"routed_smoke_fixture_marker": fixtureMarker,
	}
	outDir := os.Getenv("PLAYBOOKS_SEED_STATE_DIR")
	if outDir == "" {
		outDir = "coverage/runtime"
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("seed: mkdir seed-state dir: %v", err)
	}
	outPath := filepath.Join(outDir, "seed-state.json")
	body, err := json.MarshalIndent(seedState, "", "  ")
	if err != nil {
		log.Fatalf("seed: marshal seed-state: %v", err)
	}
	if err := os.WriteFile(outPath, body, 0o644); err != nil {
		log.Fatalf("seed: write %s: %v", outPath, err)
	}

	fmt.Printf("seed: inserted routed_smoke_fixture id=%s marker=%s; seed-state written to %s\n", fixtureID, fixtureMarker, outPath)
}

func resolveDSN() string {
	for _, key := range []string{"SQLITE_PATH", "SQLITE_DB", "POSTGRES_URL", "DATABASE_URL"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}
