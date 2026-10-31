package permissionpolicy

import (
	"context"
	"reflect"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestSQLiteAuditStoreRoundTripsOnlyReconcileMetadata(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE permission_policy_reconcile_audit (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			catalog_digest TEXT NOT NULL,
			started_at TEXT NOT NULL,
			finished_at TEXT NOT NULL,
			explicitly_authorized INTEGER NOT NULL,
			success INTEGER NOT NULL,
			hard_enforcement_satisfied INTEGER NOT NULL,
			missing_hard_enforcement_rule_ids TEXT NOT NULL,
			resource_results TEXT NOT NULL
		)
	`); err != nil {
		t.Fatalf("create audit table: %v", err)
	}

	stamp := time.Date(2026, 7, 10, 22, 0, 0, 123, time.UTC)
	want := ReconcileResult{
		CatalogDigest:                 "sha256:test",
		StartedAt:                     stamp,
		FinishedAt:                    stamp.Add(time.Second),
		ExplicitlyAuthorized:          true,
		Success:                       false,
		HardEnforcementSatisfied:      false,
		MissingHardEnforcementRuleIDs: []string{"deny-root"},
		Resources: []ResourcePlan{{
			Runner:              domain.RunnerTypeCodex,
			Scope:               "user",
			Status:              "reconciled",
			Installed:           true,
			DesiredDigest:       "resource-digest",
			NativePaths:         []string{"/tmp/codex-config"},
			Enforcement:         EnforcementPosture{Permissions: "intent_only", Caveats: []string{"intent only"}},
			UnsupportedMatchers: []Matcher{},
		}},
	}
	store := NewSQLiteAuditStore(db)
	if err := store.RecordReconcile(context.Background(), want); err != nil {
		t.Fatalf("RecordReconcile: %v", err)
	}
	got, err := store.LastReconcile(context.Background())
	if err != nil {
		t.Fatalf("LastReconcile: %v", err)
	}
	if !reflect.DeepEqual(got, want.Clone()) {
		t.Fatalf("stored result = %#v, want %#v", got, want)
	}
}
