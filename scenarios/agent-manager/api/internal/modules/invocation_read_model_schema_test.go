package modules

import (
	"context"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestInvocationReadModelSchemaPreservesLegacyFactsAndUsesAggregateIndexes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, err := sqlx.Connect("sqlite", "file:invocation-read-model-schema?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE investigation_invocation_facts (
		run_id TEXT NOT NULL, call_event_id TEXT NOT NULL, result_event_id TEXT,
		tool_call_id TEXT, tool_name TEXT NOT NULL, executable TEXT, command_path TEXT,
		ownership TEXT NOT NULL, catalog_snapshot TEXT, outcome TEXT NOT NULL,
		retry_of_call_event_id TEXT, help_recovery INTEGER NOT NULL DEFAULT 0,
		fingerprint TEXT NOT NULL, availability TEXT NOT NULL, classifier_version TEXT NOT NULL,
		PRIMARY KEY (run_id, call_event_id)
	)`); err != nil {
		t.Fatalf("create legacy facts: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO investigation_invocation_facts (run_id, call_event_id, tool_name, ownership, outcome, fingerprint, availability, classifier_version) VALUES ('run-1', 'event-1', 'shell', 'external', 'success', 'fingerprint', 'available', 'invocation-fact.v1')`); err != nil {
		t.Fatalf("seed legacy facts: %v", err)
	}
	if err := coredb.EnsureSchemas(ctx, db, AllSchemas()...); err != nil {
		t.Fatalf("apply schemas over legacy facts: %v", err)
	}

	var legacyCount int
	if err := db.Get(&legacyCount, `SELECT COUNT(*) FROM investigation_invocation_facts`); err != nil {
		t.Fatalf("read legacy facts: %v", err)
	}
	if legacyCount != 1 {
		t.Fatalf("legacy facts changed during schema application: got %d, want 1", legacyCount)
	}

	for _, tc := range []struct {
		name  string
		query string
		index string
	}{
		{"time", `SELECT * FROM invocation_read_model_facts WHERE occurred_at >= ? AND occurred_at < ?`, "idx_invocation_read_model_occurred_at"},
		{"ownership", `SELECT * FROM invocation_read_model_facts WHERE ownership = ? AND occurred_at >= ?`, "idx_invocation_read_model_ownership"},
		{"outcome", `SELECT * FROM invocation_read_model_facts WHERE outcome = ? AND occurred_at >= ?`, "idx_invocation_read_model_outcome"},
		{"executable", `SELECT * FROM invocation_read_model_facts WHERE executable = ? AND occurred_at >= ?`, "idx_invocation_read_model_executable"},
		{"fingerprint", `SELECT * FROM invocation_read_model_facts WHERE fingerprint = ? AND occurred_at >= ?`, "idx_invocation_read_model_fingerprint"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := db.Queryx(`EXPLAIN QUERY PLAN `+tc.query, "x", "y")
			if err != nil {
				t.Fatalf("explain query plan: %v", err)
			}
			defer rows.Close()
			var details []string
			for rows.Next() {
				var id, parent, notUsed int
				var detail string
				if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
					t.Fatalf("scan query plan: %v", err)
				}
				details = append(details, detail)
			}
			if err := rows.Err(); err != nil {
				t.Fatalf("iterate query plan: %v", err)
			}
			if !strings.Contains(strings.Join(details, "\n"), tc.index) {
				t.Fatalf("query did not use %s: %v", tc.index, details)
			}
		})
	}
	for _, tc := range []struct {
		query string
		index string
	}{
		{`SELECT * FROM invocation_read_model_runs WHERE occurred_at >= ? AND occurred_at < ?`, "idx_invocation_read_model_runs_occurred_at"},
		{`SELECT * FROM invocation_read_model_runs WHERE status = ? AND occurred_at >= ?`, "idx_invocation_read_model_runs_status"},
		{`SELECT * FROM invocation_read_model_runs WHERE profile_id = ? AND occurred_at >= ?`, "idx_invocation_read_model_runs_profile"},
	} {
		rows, err := db.Queryx(`EXPLAIN QUERY PLAN `+tc.query, "x", "y")
		if err != nil {
			t.Fatalf("explain run fact query: %v", err)
		}
		var details []string
		for rows.Next() {
			var id, parent, notUsed int
			var detail string
			if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
				rows.Close()
				t.Fatalf("scan run fact plan: %v", err)
			}
			details = append(details, detail)
		}
		rows.Close()
		if !strings.Contains(strings.Join(details, "\n"), tc.index) {
			t.Fatalf("run-fact query did not use %s: %v", tc.index, details)
		}
	}
}
