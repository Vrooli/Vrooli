package scenarioruntime

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func beginProvenance(t *testing.T, s *SQLiteStore, id string) EditorLeaseProvenance {
	t.Helper()
	b, err := s.CreateEditorLeaseBinding(context.Background(), EditorLease{SessionID: "session", PID: 123, HostBootID: "boot", Scope: "intended.scope"}, EditorLeaseProvenance{BindingID: id, PID: 123, UID: 1000, HostBootID: "boot", ProcessStart: "12345", IntendedScope: "intended.scope"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func successfulContainment() EditorLeaseProvenance {
	return EditorLeaseProvenance{ContainmentState: EditorContainmentEstablished, Scope: "actual.scope", Method: "systemd", EvidencePID: 124, EvidenceUID: 1000, EvidenceBootID: "boot", EvidenceProcessStart: "12346"}
}
func TestEditorProvenanceRebindAndLateEvidence(t *testing.T) {
	s, clk := newEditorLeaseStore(t)
	ctx := context.Background()
	old, err := s.CreateEditorLease(ctx, EditorLease{SessionID: "session", PID: 12, Scope: "historical.scope", ContainmentMethod: "none"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	history, err := s.ListEditorLeaseProvenance(ctx, "session")
	if err != nil || len(history) != 0 || old.BindingID != "" {
		t.Fatalf("invented historical provenance: %+v %v", history, err)
	}
	clk.Advance(time.Minute)
	a := beginProvenance(t, s, "a")
	current, _ := s.GetEditorLease(ctx, "session")
	if !current.CreatedAt.Equal(old.CreatedAt) || !a.CreatedAt.After(old.CreatedAt) || current.Scope != "intended.scope" {
		t.Fatalf("wrong creation/intent projection: %+v %+v", current, a)
	}
	if ok, err := s.RecordEditorLeaseContainment(ctx, "session", "a", successfulContainment()); !ok || err != nil {
		t.Fatalf("record: %v %v", ok, err)
	}
	current, _ = s.GetEditorLease(ctx, "session")
	if current.Scope != "intended.scope" || current.ContainmentMethod != "systemd" {
		t.Fatalf("outcome changed stable scope: %+v", current)
	}
	clk.Advance(time.Second)
	beginProvenance(t, s, "b")
	prior, _ := s.GetEditorLeaseProvenance(ctx, "a")
	if prior.State != EditorBindingSuperseded || prior.EndedAt == nil || prior.Scope != "actual.scope" || prior.PID != 123 || prior.EvidencePID != 124 {
		t.Fatalf("lost predecessor: %+v", prior)
	}
	if ok, err := s.RecordEditorLeaseContainment(ctx, "session", "a", successfulContainment()); ok || err != nil {
		t.Fatalf("late evidence: %v %v", ok, err)
	}
	if ok, err := s.HeartbeatEditorLeaseBinding(ctx, "session", "a", time.Hour); ok || err != nil {
		t.Fatalf("late heartbeat: %v %v", ok, err)
	}
	if ok, err := s.StopEditorLeaseBinding(ctx, "session", "a", "old"); ok || err != nil {
		t.Fatalf("late stop: %v %v", ok, err)
	}
	// Duplicate IDs must roll back supersession and lease replacement together.
	if _, err := s.CreateEditorLeaseBinding(ctx, EditorLease{SessionID: "session", PID: 123, HostBootID: "boot", Scope: "intended.scope"}, a, time.Minute); err == nil {
		t.Fatal("reused binding ID")
	}
	current, _ = s.GetEditorLease(ctx, "session")
	b, _ := s.GetEditorLeaseProvenance(ctx, "b")
	if current.BindingID != "b" || b.State != EditorBindingActive {
		t.Fatal("duplicate ID changed current binding")
	}
	if _, err := s.CreateEditorLease(ctx, current, time.Minute); err != nil {
		t.Fatal(err)
	}
	current, _ = s.GetEditorLease(ctx, "session")
	b, _ = s.GetEditorLeaseProvenance(ctx, "b")
	if current.BindingID != "" || b.State != EditorBindingSuperseded {
		t.Fatal("generic rebind retained binding")
	}
}

func TestEditorProvenanceOutcomeValidationAndConcurrency(t *testing.T) {
	s, _ := newEditorLeaseStore(t)
	ctx := context.Background()
	beginProvenance(t, s, "attempt")
	for _, bad := range []EditorLeaseProvenance{
		{ContainmentState: EditorContainmentEstablished, Scope: "scope", Method: "none"},
		{ContainmentState: EditorContainmentFailed, Scope: "scope"},
		{ContainmentState: EditorContainmentUnavailable, EvidencePID: 123},
		{ContainmentState: EditorContainmentIntended},
	} {
		if _, err := s.RecordEditorLeaseContainment(ctx, "session", "attempt", bad); err == nil {
			t.Fatalf("accepted invalid evidence: %+v", bad)
		}
	}
	var wg sync.WaitGroup
	results := make(chan bool, 12)
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := s.RecordEditorLeaseContainment(ctx, "session", "attempt", successfulContainment())
			results <- ok
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	wins := 0
	for ok := range results {
		if ok {
			wins++
		}
	}
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if wins != 1 {
		t.Fatalf("outcome writers = %d", wins)
	}
	if ok, err := s.RecordEditorLeaseContainment(ctx, "session", "attempt", EditorLeaseProvenance{ContainmentState: EditorContainmentFailed, Failure: "late failure"}); ok || err != nil {
		t.Fatalf("terminal outcome overwritten %v %v", ok, err)
	}
}

func TestEditorProvenanceTerminalAndHeartbeat(t *testing.T) {
	for _, end := range []string{"fenced", "expired"} {
		t.Run(end, func(t *testing.T) {
			s, clk := newEditorLeaseStore(t)
			ctx := context.Background()
			beginProvenance(t, s, "attempt")
			if ok, err := s.RecordEditorLeaseContainment(ctx, "session", "attempt", successfulContainment()); !ok || err != nil {
				t.Fatal(err)
			}
			clk.Advance(time.Second)
			if ok, err := s.HeartbeatEditorLeaseBinding(ctx, "session", "attempt", time.Minute); !ok || err != nil {
				t.Fatalf("heartbeat %v %v", ok, err)
			}
			switch end {
			case "fenced":
				if ok, err := s.StopEditorLeaseBinding(ctx, "session", "attempt", "done"); !ok || err != nil {
					t.Fatal(err)
				}
			case "expired":
				clk.Advance(2 * time.Minute)
				out, err := s.ExpireStaleEditorLeases(ctx, clk.Now(), StartingLeaseGuard{CurrentBootID: "new-boot"})
				if err != nil || len(out) != 1 {
					t.Fatalf("expire %+v %v", out, err)
				}
			}
			b, err := s.GetEditorLeaseProvenance(ctx, "attempt")
			if err != nil || b.State != EditorBindingTerminal || b.EndedAt == nil || b.Scope != "actual.scope" {
				t.Fatalf("terminal evidence %+v %v", b, err)
			}
			if ok, err := s.HeartbeatEditorLeaseBinding(ctx, "session", "attempt", time.Hour); ok || err != nil {
				t.Fatalf("terminal heartbeat %v %v", ok, err)
			}
			if ok, err := s.RecordEditorLeaseContainment(ctx, "session", "attempt", successfulContainment()); ok || err != nil {
				t.Fatalf("terminal evidence write %v %v", ok, err)
			}
		})
	}
}

func TestEditorProvenanceUpgradeAndReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "runtime.db")
	s, err := NewSQLiteStore(ctx, Config{DBPath: path})
	if err != nil {
		t.Fatal(err)
	}
	old, err := s.CreateEditorLease(ctx, EditorLease{SessionID: "historical", PID: 999, Scope: "legacy.scope", ContainmentMethod: "none"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	// Construct a real schema-15 fixture by removing precisely the additive schema-16 objects.
	if _, err = db.Exec(`DROP TABLE runtime_editor_lease_bindings; ALTER TABLE runtime_editor_leases DROP COLUMN binding_id; PRAGMA user_version=15;`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err = NewSQLiteStore(ctx, Config{DBPath: path})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetEditorLease(ctx, "historical")
	if err != nil || !reflect.DeepEqual(got, old) {
		t.Fatalf("upgrade rewrote history %+v %v", got, err)
	}
	version, err := readSchemaVersion(ctx, s.db)
	if err != nil || version != 16 {
		t.Fatalf("upgrade stamp = %d, %v; want 16", version, err)
	}
	history, err := s.ListEditorLeaseProvenance(ctx, "historical")
	if err != nil || len(history) != 0 {
		t.Fatalf("backfilled unknown history %+v %v", history, err)
	}
	beginProvenance(t, s, "attempt")
	if ok, err := s.RecordEditorLeaseContainment(ctx, "session", "attempt", EditorLeaseProvenance{ContainmentState: EditorContainmentUnavailable, Failure: "no user manager"}); !ok || err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = NewSQLiteStore(ctx, Config{DBPath: path})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	b, err := s.GetEditorLeaseProvenance(ctx, "attempt")
	if err != nil || b.ContainmentState != EditorContainmentUnavailable || b.Failure != "no user manager" || b.ProcessStart != "12345" {
		t.Fatalf("reopen lost attempt %+v %v", b, err)
	}
}

func TestEditorProvenanceConcurrentBeginAndFallback(t *testing.T) {
	s, _ := newEditorLeaseStore(t)
	ctx := context.Background()
	beginProvenance(t, s, "failed")
	if ok, err := s.RecordEditorLeaseContainment(ctx, "session", "failed", EditorLeaseProvenance{ContainmentState: EditorContainmentFailed, Failure: "exec failed"}); !ok || err != nil {
		t.Fatal(err)
	}
	if ok, err := s.RecordEditorLeaseContainment(ctx, "session", "failed", successfulContainment()); ok || err != nil {
		t.Fatalf("failed attempt promoted: %v %v", ok, err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for _, id := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, err := s.CreateEditorLeaseBinding(ctx, EditorLease{SessionID: "session", PID: 123, HostBootID: "boot"}, EditorLeaseProvenance{BindingID: id, PID: 123, HostBootID: "boot", ProcessStart: "12345"}, time.Minute)
			errs <- err
		}(id)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	all, err := s.ListEditorLeaseProvenance(ctx, "session")
	if err != nil || len(all) != 9 {
		t.Fatalf("history %d %v", len(all), err)
	}
	current, err := s.GetEditorLease(ctx, "session")
	if err != nil {
		t.Fatal(err)
	}
	active := 0
	for _, b := range all {
		if b.State == EditorBindingActive {
			active++
			if current.BindingID != b.BindingID {
				t.Fatal("lease points outside active binding")
			}
		}
	}
	if active != 1 {
		t.Fatalf("active attempts %d", active)
	}
	failed, err := s.GetEditorLeaseProvenance(ctx, "failed")
	if err != nil || failed.Failure != "exec failed" || failed.State != EditorBindingSuperseded {
		t.Fatalf("lost failure %+v %v", failed, err)
	}
}

func TestEditorProvenanceRejectsCrossIdentityEvidence(t *testing.T) {
	s, _ := newEditorLeaseStore(t)
	ctx := context.Background()
	beginProvenance(t, s, "attempt")
	for _, mutate := range []func(*EditorLeaseProvenance){func(e *EditorLeaseProvenance) { e.EvidenceBootID = "another-boot" }, func(e *EditorLeaseProvenance) { e.EvidenceUID = 1001 }} {
		e := successfulContainment()
		mutate(&e)
		if ok, err := s.RecordEditorLeaseContainment(ctx, "session", "attempt", e); ok || err != nil {
			t.Fatalf("cross-identity attestation: %v %v", ok, err)
		}
	}
	b, err := s.GetEditorLeaseProvenance(ctx, "attempt")
	if err != nil || b.ContainmentState != EditorContainmentIntended {
		t.Fatalf("invalid evidence consumed attempt: %+v %v", b, err)
	}
	if ok, err := s.RecordEditorLeaseContainment(ctx, "session", "attempt", successfulContainment()); !ok || err != nil {
		t.Fatalf("valid evidence rejected: %v %v", ok, err)
	}
	if _, err := s.CreateEditorLeaseBinding(ctx, EditorLease{SessionID: "session", PID: 123, HostBootID: "boot", Scope: "other.scope"}, EditorLeaseProvenance{BindingID: "mismatch", PID: 123, HostBootID: "boot", ProcessStart: "12345", IntendedScope: "intended.scope"}, time.Minute); err == nil {
		t.Fatal("mismatched scope intent accepted")
	}
}

// This fixture deliberately stamps 16 independently of the compiled reader.
// An isolated Go overlay lowering only SchemaVersion to 15 exercises the same
// version refusal as an older reader without changing the working source.
func TestEditorProvenanceSchema16ReaderCompatibility(t *testing.T) {
	if SchemaVersion != 16 && SchemaVersion != 15 {
		t.Fatalf("qualification requires reader 15 or 16, got %d", SchemaVersion)
	}
	for _, readOnly := range []bool{true, false} {
		t.Run(map[bool]string{true: "read_only", false: "read_write"}[readOnly], func(t *testing.T) {
			ctx := context.Background()
			dbPath := filepath.Join(t.TempDir(), "schema16.db")
			db, err := sql.Open("sqlite", dbPath)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err = db.ExecContext(ctx, schemaSQL); err != nil {
				t.Fatal(err)
			}
			if _, err = db.ExecContext(ctx, `PRAGMA user_version=16;
 INSERT INTO runtime_editor_leases (session_id,harness,agent,pid,host_boot_id,working_dir,scope,containment_method,claims_json,status,created_at,last_heartbeat_at,heartbeat_deadline_at,owner_kind,owner_ref,owner_evidence_json,owner_source,owner_recorded_at)
 VALUES ('historical','codex','codex',999,'old-boot','/fixture/repo','legacy.scope','none','["internal/"]','active','2026-09-02T15:00:00Z','2026-09-02T15:01:00Z','2026-09-02T15:02:00Z','terminal','fixture-owner','{"fixture":true}','launch','2026-09-02T15:00:00Z');`); err != nil {
				t.Fatal(err)
			}
			before, err := scanEditorLease(db.QueryRowContext(ctx, editorLeaseSelectSQL+` WHERE session_id='historical'`))
			if err != nil {
				t.Fatal(err)
			}
			var schemaBefore int
			if err = db.QueryRowContext(ctx, `PRAGMA schema_version`).Scan(&schemaBefore); err != nil {
				t.Fatal(err)
			}
			store, openErr := NewSQLiteStore(ctx, Config{DBPath: dbPath, ReadOnly: readOnly})
			if SchemaVersion == 15 {
				if store != nil {
					store.Close()
					t.Fatal("older reader returned a store")
				}
				var compatibility *SchemaCompatibilityError
				if !errors.As(openErr, &compatibility) || compatibility.DatabaseVersion != 16 || compatibility.BinaryVersion != 15 {
					t.Fatalf("older reader must reject schema 16: %T %v", openErr, openErr)
				}
				var schemaAfter int
				if err = db.QueryRowContext(ctx, `PRAGMA schema_version`).Scan(&schemaAfter); err != nil {
					t.Fatal(err)
				}
				if schemaAfter != schemaBefore {
					t.Fatalf("rejected reader mutated schema: %d -> %d", schemaBefore, schemaAfter)
				}
			} else {
				if openErr != nil {
					t.Fatal(openErr)
				}
				got, err := store.GetEditorLease(ctx, "historical")
				if err != nil || !reflect.DeepEqual(got, before) || got.BindingID != "" {
					t.Fatalf("current reader changed historical lease: %+v %v", got, err)
				}
				if err = store.Close(); err != nil {
					t.Fatal(err)
				}
			}
			after, err := scanEditorLease(db.QueryRowContext(ctx, editorLeaseSelectSQL+` WHERE session_id='historical'`))
			if err != nil || !reflect.DeepEqual(after, before) {
				t.Fatalf("reader mutated historical row: %+v %v", after, err)
			}
			version, err := readSchemaVersion(ctx, db)
			if err != nil || version != 16 {
				t.Fatalf("reader changed schema stamp: %d %v", version, err)
			}
			var bindings, leases int
			if err = db.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM runtime_editor_lease_bindings),(SELECT COUNT(*) FROM runtime_editor_leases)`).Scan(&bindings, &leases); err != nil {
				t.Fatal(err)
			}
			if bindings != 0 || leases != 1 {
				t.Fatalf("reader changed history: bindings=%d leases=%d", bindings, leases)
			}
			t.Logf("reader=%d database=16 read_only=%t: historical row and schema stamp preserved", SchemaVersion, readOnly)
		})
	}
}

// A legacy writer may still be alive after this session gains a bound attempt.
// It cannot use its session-only authority against the newer incarnation.
func TestEditorProvenanceRejectsLateLegacyWriters(t *testing.T) {
	s, clk := newEditorLeaseStore(t)
	ctx := context.Background()
	if _, err := s.CreateEditorLease(ctx, EditorLease{SessionID: "session", PID: 123, HostBootID: "boot"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	beginProvenance(t, s, "replacement")
	before, err := s.GetEditorLease(ctx, "session")
	if err != nil {
		t.Fatal(err)
	}
	ledgerBefore, err := s.GetEditorLeaseProvenance(ctx, "replacement")
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(time.Second)
	if _, err := s.HeartbeatEditorLease(ctx, "session", time.Hour); !errors.Is(err, ErrNotFound) {
		t.Fatalf("legacy heartbeat against binding: %v", err)
	}
	if _, err := s.StopEditorLease(ctx, "session", "late legacy stop"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("legacy stop against binding: %v", err)
	}
	after, err := s.GetEditorLease(ctx, "session")
	if err != nil || !reflect.DeepEqual(after, before) {
		t.Fatalf("legacy writer changed bound lease: before=%+v after=%+v err=%v", before, after, err)
	}
	ledgerAfter, err := s.GetEditorLeaseProvenance(ctx, "replacement")
	if err != nil || !reflect.DeepEqual(ledgerAfter, ledgerBefore) {
		t.Fatalf("legacy writer changed ledger: %+v %v", ledgerAfter, err)
	}
	if ok, err := s.HeartbeatEditorLeaseBinding(ctx, "session", "replacement", time.Hour); !ok || err != nil {
		t.Fatalf("fenced heartbeat failed: %v %v", ok, err)
	}
	if ok, err := s.StopEditorLeaseBinding(ctx, "session", "replacement", "current stop"); !ok || err != nil {
		t.Fatalf("fenced stop failed: %v %v", ok, err)
	}
}
