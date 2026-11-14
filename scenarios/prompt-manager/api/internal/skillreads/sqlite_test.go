package skillreads

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

func newSQLFixture(t testing.TB) (*Service, *sql.DB) {
	t.Helper()
	dsn, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "reads.db"), storage.SQLiteTuning{})
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	return NewService(NewSQLRepository(db)), db
}

func TestSkillReadStoreBoundsAndPrunesByRetention(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	store, _ := newSQLFixture(t)
	store.now = func() time.Time { return now }
	store.maxEntries = 2
	if err := store.Append(Read{ID: "old", At: now.Add(-40 * 24 * time.Hour).Format(time.RFC3339), SkillID: "old"}); err != nil {
		t.Fatal(err)
	}
	got, err := store.ReadSince(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("retention left stale entry before cap eviction: %#v", got)
	}
	for _, id := range []string{"one", "two", "three"} {
		if err := store.Append(Read{ID: id, SkillID: id}); err != nil {
			t.Fatal(err)
		}
	}

	got, err = store.ReadSince(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "two" || got[1].ID != "three" {
		t.Fatalf("bounded reads = %#v, want latest two entries", got)
	}
}

func TestSkillReadStoreWindowReturnsAnEmptyCollectionWhenNoRowsMatch(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	store, _ := newSQLFixture(t)
	store.now = func() time.Time { return now }
	store.retention = 365 * 24 * time.Hour
	if err := store.Append(Read{ID: "old", At: now.Add(-24 * time.Hour).Format(time.RFC3339)}); err != nil {
		t.Fatal(err)
	}
	got, err := store.ReadSince(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("window reads = %#v, want non-nil empty collection", got)
	}
}

func TestSkillReadsPreserveOrderAttributionAndInvalidTimestamps(t *testing.T) {
	s, _ := newSQLFixture(t)
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	s.maxEntries = 2
	reads := []Read{
		{ID: "same", SkillID: "first", At: now.Format(time.RFC3339)},
		{ID: "old", SkillID: "expired", At: now.Add(-31 * 24 * time.Hour).Format(time.RFC3339)},
		{ID: "same", SkillID: "second", At: "invalid", Caller: "operator-direct", CallerKind: "operator-direct", AgentRunID: "run", ViaDiscovery: true, DiscoveryCallID: "call"},
	}
	for _, read := range reads {
		if err := s.Append(read); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.ReadSince(time.Hour)
	if err != nil || len(got) != 2 || got[0] != reads[0] || got[1] != reads[2] {
		t.Fatalf("retained ordered history=%+v err=%v", got, err)
	}
	// A late old timestamp does not consume the cap or evict a useful event.
	if err := s.Append(reads[1]); err != nil {
		t.Fatal(err)
	}
	got, err = s.ReadSince(0)
	if err != nil || len(got) != 2 || got[0] != reads[0] || got[1] != reads[2] {
		t.Fatalf("retention/cap=%+v %v", got, err)
	}
}

func TestSkillReadsConcurrentAppendsAndRestart(t *testing.T) {
	s, db := newSQLFixture(t)
	const total = 100
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := s.Append(Read{ID: fmt.Sprint(i), SkillID: "skill"}); err != nil {
				t.Errorf("append %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
	// A new repository/service has no process-local log state to lose.
	restarted := NewService(NewSQLRepository(db))
	got, err := restarted.ReadSince(0)
	seen := map[string]bool{}
	for _, read := range got {
		seen[read.ID] = true
	}
	if err != nil || len(got) != total || len(seen) != total {
		t.Fatalf("lost concurrent durable writes: rows=%d unique=%d err=%v", len(got), len(seen), err)
	}
}

func TestSkillReadsRetentionFailureRollsBackAppend(t *testing.T) {
	s, db := newSQLFixture(t)
	if err := s.Append(Read{ID: "before"}); err != nil {
		t.Fatal(err)
	}
	s.maxEntries = 1
	if _, err := db.Exec("CREATE TRIGGER refuse_read_delete BEFORE DELETE ON skill_read_telemetry BEGIN SELECT RAISE(ABORT, 'fixture refusal'); END"); err != nil {
		t.Fatal(err)
	}
	if err := s.Append(Read{ID: "after"}); err == nil {
		t.Fatal("append ignored failed cap enforcement")
	}
	got, err := s.ReadSince(0)
	if err != nil || len(got) != 1 || got[0].ID != "before" {
		t.Fatalf("partial transaction survived: %+v %v", got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.WithContext(ctx).Append(Read{ID: "cancelled"}); err == nil {
		t.Fatal("cancelled append succeeded")
	}
}

func TestSkillReadsRouteAppendAndReadToLeasedPool(t *testing.T) {
	routed, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { routed.Close() })
	if err := database.EnsureSchemas(context.Background(), routed.Primary(), database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	testPath := filepath.Join(t.TempDir(), "leased.db")
	routed.SetTestPoolInitializer(func(ctx context.Context, db *sql.DB) error {
		return database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema))
	})
	if err := routed.InstallTestPool(context.Background(), testPath, "read-lease", 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { routed.ClearTestPool("read-lease") })
	s := NewService(NewSQLRepository(routed))
	if err := s.Append(Read{ID: "live"}); err != nil {
		t.Fatal(err)
	}
	isolated := s.WithContext(database.WithTestMode(context.Background()))
	if err := isolated.Append(Read{ID: "test"}); err != nil {
		t.Fatal(err)
	}
	live, err := s.ReadSince(0)
	if err != nil || len(live) != 1 || live[0].ID != "live" {
		t.Fatalf("live rows=%+v %v", live, err)
	}
	test, err := isolated.ReadSince(0)
	if err != nil || len(test) != 1 || test[0].ID != "test" {
		t.Fatalf("isolated rows=%+v %v", test, err)
	}
}

func BenchmarkSkillReadTelemetrySQLAtCapacity(b *testing.B) {
	s, db := newSQLFixture(b)
	tx, err := db.Begin()
	if err != nil {
		b.Fatal(err)
	}
	stmt, err := tx.Prepare("INSERT INTO skill_read_telemetry(observed_at,observed_at_nanos,payload) VALUES (?,?,?)")
	if err != nil {
		b.Fatal(err)
	}
	now := time.Now().UTC()
	payload := `{"id":"benchmark","at":"` + now.Format(time.RFC3339) + `","skillId":"skill","caller":"agent-member:worker","callerKind":"agent-member","agentRunId":"run"}`
	for i := 0; i < MaxEntries; i++ {
		if _, err := stmt.Exec(now.Unix(), now.Nanosecond(), payload); err != nil {
			b.Fatal(err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := s.Append(Read{ID: fmt.Sprint(i), At: now.Format(time.RFC3339), SkillID: "skill", Caller: "agent-member:worker", CallerKind: "agent-member", AgentRunID: "run"}); err != nil {
			b.Fatal(err)
		}
	}
}

func TestSkillReadsTimestampRangeAndNanosecondBoundary(t *testing.T) {
	s, _ := newSQLFixture(t)
	now := time.Date(2026, 11, 5, 12, 0, 0, 123, time.UTC)
	s.now = func() time.Time { return now }
	cutoff := now.Add(-Retention)
	for _, read := range []Read{
		{ID: "future", At: "9999-01-01T00:00:00Z"},
		{ID: "exact", At: cutoff.Format(time.RFC3339Nano)},
		{ID: "expired", At: cutoff.Add(-time.Nanosecond).Format(time.RFC3339Nano)},
	} {
		if err := s.Append(read); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.ReadSince(Retention)
	if err != nil || len(got) != 2 || got[0].ID != "future" || got[1].ID != "exact" {
		t.Fatalf("timestamp overflow or boundary changed: %+v %v", got, err)
	}
}
