// Go seed for the routed-e2e path in scenario-completeness-scoring.
//
// test-genie's playbooks phase invokes this with SQLITE_PATH pointing at the
// per-run test database. The seed applies the score_snapshots schema and
// inserts one deterministic row. The routed-database BAS case then navigates to
// the fleet view and asserts that the seeded scenario is visible, which only
// passes when the UI -> API request reads from the routed test pool.

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

	_ "modernc.org/sqlite"
)

const scoreSnapshotsSchema = `
CREATE TABLE IF NOT EXISTS score_snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  scenario TEXT NOT NULL,
  category TEXT NOT NULL DEFAULT 'utility',
  digest TEXT NOT NULL,
  composite INTEGER NOT NULL,
  classification TEXT NOT NULL,
  working_rung TEXT NOT NULL DEFAULT '',
  breakdown_json TEXT NOT NULL DEFAULT '{}',
  importance REAL,
  source TEXT NOT NULL DEFAULT 'sweeper',
  created_at TEXT NOT NULL,
  UNIQUE (scenario, digest)
);

CREATE INDEX IF NOT EXISTS idx_score_snapshots_scenario_created_at
  ON score_snapshots (scenario, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_score_snapshots_scenario_digest
  ON score_snapshots (scenario, digest);

CREATE INDEX IF NOT EXISTS idx_score_snapshots_list_composite
  ON score_snapshots (composite DESC, scenario ASC);

CREATE INDEX IF NOT EXISTS idx_score_snapshots_list_created_at
  ON score_snapshots (created_at DESC, scenario ASC);
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
	if _, err := db.ExecContext(ctx, scoreSnapshotsSchema); err != nil {
		log.Fatalf("seed: apply score_snapshots schema: %v", err)
	}

	const fixtureScenario = "routed-smoke-001"
	const fixtureDigest = "bas:routed-smoke-001"
	now := time.Now().UTC().Format(time.RFC3339Nano)
	const insertSQL = `
INSERT INTO score_snapshots
  (scenario, category, digest, composite, classification, working_rung, breakdown_json, importance, source, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(scenario, digest) DO NOTHING
`
	if _, err := db.ExecContext(ctx, insertSQL,
		fixtureScenario,
		"utility",
		fixtureDigest,
		91,
		"strong",
		"R3 Operational",
		`{"basSeed":true}`,
		0.95,
		"bas-seed",
		now,
	); err != nil {
		log.Fatalf("seed: insert score snapshot: %v", err)
	}

	seedState := map[string]any{
		"routed_smoke_scenario": fixtureScenario,
		"routed_smoke_digest":   fixtureDigest,
	}
	outDir := os.Getenv("PLAYBOOKS_SEED_STATE_DIR")
	if outDir == "" {
		outDir = "coverage/runtime"
	}
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		log.Fatalf("seed: mkdir seed-state dir: %v", err)
	}
	outPath := filepath.Join(outDir, "seed-state.json")
	body, err := json.MarshalIndent(seedState, "", "  ")
	if err != nil {
		log.Fatalf("seed: marshal seed-state: %v", err)
	}
	if err := os.WriteFile(outPath, body, 0o600); err != nil {
		log.Fatalf("seed: write %s: %v", outPath, err)
	}

	fmt.Printf("seed: inserted score snapshot scenario=%s digest=%s; seed-state written to %s\n", fixtureScenario, fixtureDigest, outPath)
}

// resolveDSN returns the isolated database the playbooks isolation manager
// leased for this run. Only the playbooks-scoped variables are read: the
// generic SQLITE_PATH / SQLITE_DB pair used to be accepted here, which made a
// seed script write into whatever database happened to be named in the
// environment.
func resolveDSN() string {
	for _, key := range []string{"PLAYBOOKS_SQLITE_DSN", "PLAYBOOKS_SQLITE_PATH"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}
