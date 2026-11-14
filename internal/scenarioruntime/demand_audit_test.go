package scenarioruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

func TestDemandAuditQueryIndexRepairsExistingRegistryWithoutChangingCompatibility(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "runtime.db")
	store, err := NewSQLiteStore(ctx, Config{DBPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordEvent(ctx, Event{Scenario: "demo", EventType: "unrelated"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `DROP INDEX idx_runtime_events_scenario_type`); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// A read-only observer can still inspect a registry created by an older
	// binary. It must not need write permission merely to add an optimization.
	observer, err := NewSQLiteStore(ctx, Config{DBPath: path, ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := observer.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = NewSQLiteStore(ctx, Config{DBPath: path})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var version, count int
	if err := store.db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil || version != SchemaVersion {
		t.Fatalf("compatibility version=%d err=%v", version, err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT count(*) FROM runtime_events WHERE scenario='demo'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("existing history count=%d err=%v", count, err)
	}
	rows, err := store.db.QueryContext(ctx, `EXPLAIN QUERY PLAN SELECT rowid FROM runtime_events
WHERE scenario = ? AND event_type = ? ORDER BY rowid DESC LIMIT -1 OFFSET ?`, "demo", demandAuditEventType, DemandAuditRetention)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	scoped := false
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(detail, "(scenario=? AND event_type=?)") {
			scoped = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !scoped {
		t.Fatal("retention must seek the selected scenario and event family, not scan fleet history")
	}
}

func BenchmarkDemandAuditRetentionFleet(b *testing.B) {
	for _, indexed := range []bool{false, true} {
		name := "fleet_scan"
		if indexed {
			name = "scenario_seek"
		}
		b.Run(name, func(b *testing.B) {
			ctx := context.Background()
			store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(b.TempDir(), "runtime.db")})
			if err != nil {
				b.Fatal(err)
			}
			defer store.Close()
			if !indexed {
				if _, err := store.db.ExecContext(ctx, `DROP INDEX idx_runtime_events_scenario_type`); err != nil {
					b.Fatal(err)
				}
			}
			// A hundred independent owners, each with its retained 1,000 rows.
			// Only one owner's newest transition is being recorded/pruned.
			_, err = store.db.ExecContext(ctx, `WITH RECURSIVE n(x) AS (
SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<100000
) INSERT INTO runtime_events(event_id, scenario, event_type, created_at, details_json)
SELECT printf('seed-%d',x), printf('owner-%d',x%100), 'demand_transition', '2026-11-05T00:00:00Z', '{}' FROM n`)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := store.withTx(ctx, func(tx *sql.Tx) error {
					return store.auditDemandTx(ctx, tx, DemandTransition{Operation: "renewed", Scenario: "owner-0", Variant: "live"})
				}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestDemandTransitionHistoryTracksOnlyCommittedChanges(t *testing.T) {
	ctx := context.Background()
	clock := testenv.NewClock(time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clock})
	})
	lease := DemandLease{LeaseID: "job", Scenario: "demo", ConsumerID: "worker", Kind: DemandLeaseJob, MetadataJSON: `{"private":"not-audit-data"}`}
	if _, err := store.AcquireDemandLease(ctx, lease, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RenewDemandLease(ctx, lease.LeaseID, time.Minute); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := store.ReleaseDemandLease(ctx, lease.LeaseID, "done"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.RenewDemandLease(ctx, lease.LeaseID, time.Minute); !errors.Is(err, ErrDemandLeaseConflict) {
		t.Fatalf("renew released=%v", err)
	}
	if _, err := store.AcquireDemandLease(ctx, lease, time.Minute); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Minute)
	for range 2 {
		if _, err := store.ExpireDemandLeases(ctx, clock.Now()); err != nil {
			t.Fatal(err)
		}
	}
	events, err := store.ListDemandTransitions(ctx, "demo", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"expired", "acquired", "released", "renewed", "acquired"}
	if len(events) != len(want) {
		t.Fatalf("events=%+v", events)
	}
	for i, e := range events {
		if e.Operation != want[i] || e.Variant != DefaultVariant || e.SchemaVersion != 1 || e.LeaseID != "job" {
			t.Fatalf("event %d=%+v", i, e)
		}
	}
	data, _ := json.Marshal(events)
	if strings.Contains(string(data), "not-audit-data") {
		t.Fatal("consumer metadata leaked into audit")
	}
	if _, err := store.db.ExecContext(ctx, `CREATE TRIGGER audit_failure BEFORE INSERT ON runtime_events WHEN NEW.event_type = 'demand_transition' BEGIN SELECT RAISE(ABORT, 'audit persistence failed'); END`); err != nil {
		t.Fatal(err)
	}
	lease.LeaseID = "uncommitted"
	if _, err := store.AcquireDemandLease(ctx, lease, time.Minute); err == nil {
		t.Fatal("acquisition survived failed audit persistence")
	}
	if _, err := store.getDemandLease(ctx, lease.LeaseID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("uncommitted lease=%v", err)
	}
}

func TestDemandTransitionRetentionIsBoundedAndVariantScoped(t *testing.T) {
	ctx := context.Background()
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path}) })
	if _, err := store.RecordEvent(ctx, Event{Scenario: "demo", EventType: "unrelated"}); err != nil {
		t.Fatal(err)
	}
	err := store.withTx(ctx, func(tx *sql.Tx) error {
		for i := 0; i < DemandAuditRetention+5; i++ {
			variant := "live"
			if i%2 == 0 {
				variant = "shadow"
			}
			if err := store.auditDemandTx(ctx, tx, DemandTransition{Operation: "renewed", Scenario: "demo", Variant: variant}); err != nil {
				return err
			}
		}
		return store.auditDemandTx(ctx, tx, DemandTransition{Operation: "acquired", Scenario: "other", Variant: "live"})
	})
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := store.db.QueryRowContext(ctx, `SELECT count(*) FROM runtime_events WHERE scenario='demo' AND event_type=?`, demandAuditEventType).Scan(&count); err != nil || count != DemandAuditRetention {
		t.Fatalf("retention count=%d err=%v", count, err)
	}
	for _, variant := range []string{"live", "shadow"} {
		events, err := store.ListDemandTransitions(ctx, "demo", variant, 1)
		if err != nil || len(events) != 1 || events[0].Variant != variant {
			t.Fatalf("filtered history=%+v %v", events, err)
		}
	}
	if err := store.db.QueryRowContext(ctx, `SELECT count(*) FROM runtime_events WHERE event_type='unrelated' OR scenario='other'`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("retention removed other history: count=%d err=%v", count, err)
	}
}

func TestDemandTeardownAuditIsOrderedAndReplaySafe(t *testing.T) {
	ctx := context.Background()
	store, instance := demandStopFixture(t)
	for range 2 {
		if err := store.BeginDemandStop(ctx, instance.InstanceID, instance.Generation); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.CompleteDemandStop(ctx, instance.InstanceID, instance.Generation); err != nil {
		t.Fatal(err)
	}
	events, err := store.ListDemandTransitions(ctx, instance.Scenario, instance.Variant, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("events=%+v", events)
	}
	for i, op := range []string{"stopped", "stop_signaling", "stop_claimed"} {
		if events[i].Operation != op || events[i].InstanceID != instance.InstanceID || events[i].Generation != instance.Generation {
			t.Fatalf("event=%+v", events[i])
		}
	}
}
