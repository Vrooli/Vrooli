package modules

import (
	"context"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestInvocationReadModelSchemaOmitsRetiredProjectionTablesAndUsesAggregateIndexes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, err := sqlx.Connect("sqlite", "file:invocation-read-model-schema?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	defer db.Close()

	if err := coredb.EnsureSchemas(ctx, db, AllSchemas()...); err != nil {
		t.Fatalf("apply schemas: %v", err)
	}
	for _, table := range []string{"investigation_invocation_facts", "investigation_friction_episodes", "investigation_self_report_spans"} {
		var count int
		if err := db.Get(&count, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table); err != nil || count != 0 {
			t.Fatalf("retired projection table %s present: count=%d err=%v", table, count, err)
		}
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
