// Go seed for the routed-e2e path in development-toolchain-validator.
//
// test-genie's playbooks phase invokes this with SQLITE_PATH pointing
// at the per-run test database. The seed:
//
//  1. Applies the `goldens` schema (copy of api/internal/golden/schema.sql)
//     because RoutedDB.InstallTestPool installs a fresh pool but does not
//     run app schemas against it.
//  2. Inserts one Golden row with slug "routed-smoke-001".
//  3. Writes coverage/runtime/seed-state.json so BAS can reference the
//     fixture identifier via initial_params.
//
// The BAS playbook in bas/flows/routed-smoke.json then navigates to the
// goldens index (/) and asserts the slug is visible — which is only true
// if the UI→API request was routed to the test pool. If routing fails,
// the slug is absent from the primary pool and the assertion fails.

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
	"time"

	"github.com/google/uuid"

	_ "modernc.org/sqlite"
)

// goldensSchema mirrors api/internal/golden/schema.sql. Kept in sync
// manually — see SEAMS doc for the goldens table contract. Idempotent
// (CREATE TABLE IF NOT EXISTS), so re-runs on the same DSN are no-ops.
const goldensSchema = `
CREATE TABLE IF NOT EXISTS goldens (
  id                      TEXT PRIMARY KEY,
  slug                    TEXT NOT NULL UNIQUE,
  template_id             TEXT NOT NULL,
  template_version_pinned TEXT NOT NULL,
  path                    TEXT NOT NULL,
  created_at              TEXT NOT NULL,
  last_regenerated_at     TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_goldens_slug ON goldens(slug);
`

func main() {
	dsn := resolveDSN()
	if dsn == "" {
		log.Fatalf("seed: no SQLITE_PATH / SQLITE_DB in env")
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("seed: open sqlite: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	if _, err := db.ExecContext(ctx, goldensSchema); err != nil {
		log.Fatalf("seed: apply goldens schema: %v", err)
	}

	fixtureSlug := "routed-smoke-001"
	fixtureID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	const insertSQL = `
INSERT INTO goldens (id, slug, template_id, template_version_pinned, path, created_at, last_regenerated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(slug) DO NOTHING
`
	if _, err := db.ExecContext(ctx, insertSQL,
		fixtureID, fixtureSlug,
		"routed-smoke-template", "0.0.0",
		"docs/routed-smoke/golden.md",
		now, now,
	); err != nil {
		log.Fatalf("seed: insert golden: %v", err)
	}

	// Write coverage/runtime/seed-state.json so the playbooks runner picks
	// up these values and forwards them to BAS via initial_params.
	seedState := map[string]any{
		"routed_smoke_golden_slug": fixtureSlug,
		"routed_smoke_golden_id":   fixtureID,
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

	fmt.Printf("seed: inserted golden slug=%s id=%s; seed-state written to %s\n", fixtureSlug, fixtureID, outPath)
}

func resolveDSN() string {
	for _, key := range []string{"SQLITE_PATH", "SQLITE_DB"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}
