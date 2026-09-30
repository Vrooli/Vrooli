package storagehealth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	coredb "github.com/vrooli/api-core/database"
	corestorage "github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

// openFileDB opens a real WAL-mode file database under t.TempDir with the same
// DSN pragmas the production pool uses. incremental creates it in incremental
// auto-vacuum mode from the start.
func openFileDB(t *testing.T, incremental bool) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agent-manager.db")
	dsn, err := corestorage.SQLiteDSNAt(path, corestorage.SQLiteTuning{PageSizeBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(3)
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	mustExec(t, conn, "CREATE TABLE blobs (id INTEGER PRIMARY KEY, body BLOB NOT NULL)")
	if incremental {
		// The DSN's journal_mode(WAL) writes the header on open, so the file
		// already exists and only a VACUUM applies the mode, as in production.
		mustExec(t, conn, "PRAGMA auto_vacuum=INCREMENTAL")
		mustExec(t, conn, "VACUUM")
	}
	return db, path
}

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func mustExec(t *testing.T, db execer, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query, args...); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

// fill writes rows of 64 KiB, then deletes the given fraction, leaving their
// pages on the freelist.
func fill(t *testing.T, db *sql.DB, rows int, deleteEvery int) {
	t.Helper()
	body := make([]byte, 64<<10)
	for i := 0; i < rows; i++ {
		mustExec(t, db, "INSERT INTO blobs (body) VALUES (?)", body)
	}
	if deleteEvery > 0 {
		mustExec(t, db, "DELETE FROM blobs WHERE id % ? != 0", deleteEvery)
	}
}

func testPolicy() Policy {
	p := DefaultPolicy()
	p.FreeTriggerBytes, p.FreeFloorBytes, p.SpaceMarginBytes = 1<<20, 0, 1<<20
	p.BatchPages, p.BatchPause, p.MaintainEvery = 64, time.Millisecond, time.Hour
	return p
}

type drainedFence struct {
	mu    sync.Mutex
	state FenceState
	err   error
}

func (f *drainedFence) observe(context.Context) (FenceState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state, f.err
}

type pauseRecorder struct {
	mu     sync.Mutex
	events []string
}

func (p *pauseRecorder) pause(context.Context) (func(), error) {
	p.mu.Lock()
	p.events = append(p.events, "pause")
	p.mu.Unlock()
	return func() { p.mu.Lock(); p.events = append(p.events, "resume"); p.mu.Unlock() }, nil
}

func newService(t *testing.T, db *sql.DB, path string, fence *drainedFence, pause Pauser) *Service {
	t.Helper()
	opts := Options{DB: func() *sql.DB { return db }, Path: path, Policy: testPolicy(), Pause: pause, FreeSpace: func(string) (uint64, error) { return 1 << 40, nil }}
	if fence != nil {
		opts.Fence = fence.observe
	}
	svc, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestAssessPlacesTheFileAgainstItsBudget(t *testing.T) {
	p := DefaultPolicy()
	const gib = int64(1 << 30)
	for _, tc := range []struct {
		name       string
		stats      Stats
		wantLevel  Level
		wantAction Action
	}{
		{"unbudgeted idle", Stats{FileBytes: gib, LiveBytes: gib, AutoVacuum: "incremental"}, LevelUnbudgeted, ActionNone},
		{"live incident shape", Stats{FileBytes: 293 * gib, LiveBytes: 9 * gib, FreeBytes: 284 * gib, BudgetBytes: 6 * gib, AutoVacuum: "none"}, LevelAlarm, ActionCompactionRequired},
		{"freed pages online", Stats{FileBytes: 3 * gib, LiveBytes: 2 * gib, FreeBytes: gib, BudgetBytes: 12 * gib, AutoVacuum: "incremental"}, LevelOK, ActionIncrementalVacuum},
		{"small slack stays", Stats{FileBytes: 2 * gib, LiveBytes: 2 * gib, FreeBytes: 32 << 20, BudgetBytes: 12 * gib, AutoVacuum: "incremental"}, LevelOK, ActionNone},
		{"pressure returns small slack", Stats{FileBytes: 9 * gib, LiveBytes: 9 * gib, FreeBytes: 32 << 20, BudgetBytes: 12 * gib, AutoVacuum: "incremental"}, LevelReclaim, ActionIncrementalVacuum},
		{"live rows near budget", Stats{FileBytes: 11 * gib, LiveBytes: 11 * gib, BudgetBytes: 12 * gib, AutoVacuum: "incremental"}, LevelAlarm, ActionRetentionOrBudget},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := tc.stats
			assess(&st, p)
			if st.Level != tc.wantLevel || st.Action != tc.wantAction {
				t.Fatalf("level=%s action=%s, want %s/%s", st.Level, st.Action, tc.wantLevel, tc.wantAction)
			}
		})
	}
}

func TestMaintainReturnsFreedPagesInIncrementalMode(t *testing.T) {
	db, path := openFileDB(t, true)
	fill(t, db, 200, 4)
	svc := newService(t, db, path, nil, nil)
	before, err := svc.Stats(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if before.Action != ActionIncrementalVacuum || before.FreeBytes < 8<<20 {
		t.Fatalf("fixture did not free pages: %+v", before)
	}

	if err := svc.Maintain(t.Context()); err != nil {
		t.Fatal(err)
	}

	after, err := svc.Stats(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if after.FreelistCount != 0 {
		t.Fatalf("freelist=%d after maintenance, want 0", after.FreelistCount)
	}
	mustExec(t, db, "PRAGMA wal_checkpoint(TRUNCATE)")
	if size := fileSize(path); size >= before.FileBytes {
		t.Fatalf("file did not shrink: %d -> %d", before.FileBytes, size)
	}
	var rows int
	if err := db.QueryRow("SELECT COUNT(*) FROM blobs").Scan(&rows); err != nil || rows != 50 {
		t.Fatalf("rows=%d err=%v, want 50 kept", rows, err)
	}
}

func TestMaintainBoundsOnePassByItsTimeBudget(t *testing.T) {
	db, path := openFileDB(t, true)
	fill(t, db, 200, 4)
	svc := newService(t, db, path, nil, nil)
	svc.opts.Policy.BatchPages, svc.opts.Policy.TickBudget, svc.opts.Policy.BatchPause = 8, 5*time.Millisecond, 5*time.Millisecond
	before, _ := svc.Stats(t.Context())

	if err := svc.Maintain(t.Context()); err != nil {
		t.Fatal(err)
	}

	after, _ := svc.Stats(t.Context())
	if after.FreelistCount == 0 || after.FreelistCount >= before.FreelistCount {
		t.Fatalf("one bounded pass should return some but not all pages: %d -> %d", before.FreelistCount, after.FreelistCount)
	}
}

// TestMaintainFinishesADrainBelowTheTrigger: once a drain starts, later passes
// run it down to the floor even when the remainder no longer exceeds the
// trigger that started it.
func TestMaintainFinishesADrainBelowTheTrigger(t *testing.T) {
	db, path := openFileDB(t, true)
	fill(t, db, 200, 4)
	svc := newService(t, db, path, nil, nil)
	svc.opts.Policy.BatchPages, svc.opts.Policy.MinBatchPages, svc.opts.Policy.BatchHoldTarget = 8, 8, 0
	svc.opts.Policy.TickBudget, svc.opts.Policy.BatchPause = 5*time.Millisecond, 5*time.Millisecond

	if err := svc.Maintain(t.Context()); err != nil {
		t.Fatal(err)
	}
	partial, _ := svc.Stats(t.Context())
	if partial.FreelistCount == 0 {
		t.Fatal("the first pass was meant to stop partway")
	}
	svc.opts.Policy.FreeTriggerBytes = partial.FreeBytes + 1
	svc.opts.Policy.TickBudget = 10 * time.Second
	if st, _ := svc.Stats(t.Context()); st.Action != ActionNone {
		t.Fatalf("remainder should be below the trigger: %s", st.Action)
	}

	if err := svc.Maintain(t.Context()); err != nil {
		t.Fatal(err)
	}

	if st, _ := svc.Stats(t.Context()); st.FreelistCount != 0 {
		t.Fatalf("drain stopped at %d free pages above a zero floor", st.FreelistCount)
	}
}

func TestMaintainNeverRewritesANoneModeDatabase(t *testing.T) {
	db, path := openFileDB(t, false)
	fill(t, db, 100, 4)
	svc := newService(t, db, path, nil, nil)
	before, _ := svc.Stats(t.Context())

	if err := svc.Maintain(t.Context()); err != nil {
		t.Fatal(err)
	}

	after, _ := svc.Stats(t.Context())
	if after.Action != ActionCompactionRequired || after.FreelistCount != before.FreelistCount || after.AutoVacuum != "none" {
		t.Fatalf("online maintenance must only report a none-mode file: %+v", after)
	}
}

func TestCompactRefusesUnlessTheFenceIsClosedAndDrained(t *testing.T) {
	for _, tc := range []struct {
		name  string
		fence *drainedFence
	}{
		{"no observer", nil},
		{"open", &drainedFence{state: FenceState{Revision: 3}}},
		{"closed not drained", &drainedFence{state: FenceState{Closed: true, Revision: 4}}},
		{"unobservable", &drainedFence{err: errors.New("inventory incomplete")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, path := openFileDB(t, false)
			fill(t, db, 20, 2)
			pauses := &pauseRecorder{}
			svc := newService(t, db, path, tc.fence, pauses.pause)

			_, err := svc.Compact(t.Context(), "owner", "test")

			if !errors.Is(err, ErrFenceNotDrained) {
				t.Fatalf("err=%v, want ErrFenceNotDrained", err)
			}
			if st, _ := svc.Stats(t.Context()); st.AutoVacuum != "none" || len(pauses.events) != 0 {
				t.Fatalf("refused compaction changed state: %+v %v", st, pauses.events)
			}
		})
	}
}

func TestCompactRefusesWhenDiskSpaceIsShort(t *testing.T) {
	db, path := openFileDB(t, false)
	fill(t, db, 50, 2)
	svc := newService(t, db, path, &drainedFence{state: FenceState{Closed: true, Drained: true}}, nil)
	svc.opts.FreeSpace = func(string) (uint64, error) { return 1 << 20, nil }

	_, err := svc.Compact(t.Context(), "owner", "test")

	if !errors.Is(err, ErrInsufficientSpace) {
		t.Fatalf("err=%v, want ErrInsufficientSpace", err)
	}
}

func TestCompactConvertsToIncrementalShrinksAndKeepsRows(t *testing.T) {
	db, path := openFileDB(t, false)
	fill(t, db, 200, 4)
	mustExec(t, db, "PRAGMA wal_checkpoint(TRUNCATE)")
	pauses := &pauseRecorder{}
	svc := newService(t, db, path, &drainedFence{state: FenceState{Closed: true, Drained: true, Revision: 9}}, pauses.pause)

	receipt, err := svc.Compact(t.Context(), "owner", "rollout")
	if err != nil {
		t.Fatal(err)
	}

	if receipt.AutoVacuumBefore != "none" || receipt.AutoVacuumAfter != "incremental" || receipt.QuickCheck != "ok" {
		t.Fatalf("receipt=%+v", receipt)
	}
	if receipt.After.FreelistCount != 0 || receipt.After.FileBytes >= receipt.Before.FileBytes || receipt.ReclaimedBytes <= 0 {
		t.Fatalf("compaction did not return space: %+v", receipt)
	}
	if receipt.After.WALBytes != 0 {
		t.Fatalf("WAL not truncated after compaction: %d", receipt.After.WALBytes)
	}
	if strings.Join(pauses.events, ",") != "pause,resume" {
		t.Fatalf("background writers not paused and resumed once: %v", pauses.events)
	}
	var rows int
	if err := db.QueryRow("SELECT COUNT(*) FROM blobs").Scan(&rows); err != nil || rows != 50 {
		t.Fatalf("rows=%d err=%v", rows, err)
	}
	var tempStore int
	if err := db.QueryRow("PRAGMA temp_store").Scan(&tempStore); err != nil || tempStore != 2 {
		t.Fatalf("pool temp_store=%d err=%v, want MEMORY restored", tempStore, err)
	}
	if status := svc.CompactionStatus(); status.State != CompactionSucceeded || status.Receipt == nil {
		t.Fatalf("status=%+v", status)
	}
	// After conversion, later deletions are returned online.
	fill(t, db, 40, 2)
	if err := svc.Maintain(t.Context()); err != nil {
		t.Fatal(err)
	}
	if st, _ := svc.Stats(t.Context()); st.FreelistCount != 0 {
		t.Fatalf("freelist=%d after converted maintenance", st.FreelistCount)
	}
}

func TestCompactAbandonsWhenTheFenceChangesWhilePausing(t *testing.T) {
	db, path := openFileDB(t, false)
	fill(t, db, 20, 2)
	fence := &drainedFence{state: FenceState{Closed: true, Drained: true, Revision: 5}}
	svc := newService(t, db, path, fence, func(context.Context) (func(), error) {
		fence.mu.Lock()
		fence.state = FenceState{Revision: 6}
		fence.mu.Unlock()
		return func() {}, nil
	})

	_, err := svc.Compact(t.Context(), "owner", "test")

	if !errors.Is(err, ErrFenceNotDrained) {
		t.Fatalf("err=%v", err)
	}
	if st, _ := svc.Stats(t.Context()); st.AutoVacuum != "none" {
		t.Fatalf("compaction ran after the fence reopened: %s", st.AutoVacuum)
	}
}

func TestStartCompactionFinishesInTheBackground(t *testing.T) {
	db, path := openFileDB(t, false)
	fill(t, db, 60, 3)
	svc := newService(t, db, path, &drainedFence{state: FenceState{Closed: true, Drained: true}}, nil)

	started, err := svc.StartCompaction(t.Context(), "owner", "rollout")
	if err != nil || started.State != CompactionRunning {
		t.Fatalf("started=%+v err=%v", started, err)
	}
	if _, err := svc.StartCompaction(t.Context(), "owner", "again"); !errors.Is(err, ErrBusy) {
		// The first run may already have finished on a fast machine.
		if svc.CompactionStatus().State == CompactionRunning {
			t.Fatalf("concurrent compaction admitted: %v", err)
		}
	}
	deadline := time.Now().Add(10 * time.Second)
	for svc.CompactionStatus().State == CompactionRunning && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if status := svc.CompactionStatus(); status.State != CompactionSucceeded {
		t.Fatalf("status=%+v", status)
	}
}

func TestReclaimDryRunPreviewsAndRealRunReturnsPages(t *testing.T) {
	db, path := openFileDB(t, true)
	fill(t, db, 120, 3)
	svc := newService(t, db, path, nil, nil)

	preview, err := svc.Reclaim(t.Context(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.DryRun || preview.After != nil || preview.Before.FreeBytes == 0 {
		t.Fatalf("preview=%+v", preview)
	}
	if st, _ := svc.Stats(t.Context()); st.FreelistCount != preview.Before.FreelistCount {
		t.Fatal("dry run changed the file")
	}

	receipt, err := svc.Reclaim(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Complete || receipt.After == nil || receipt.After.FreelistCount != 0 || receipt.Performed != PerformedIncrementalVacuum || receipt.CompactionRequired {
		t.Fatalf("receipt=%+v", receipt)
	}
	if receipt.BytesBefore != preview.Before.OccupiedBytes || receipt.BytesAfter != receipt.After.OccupiedBytes || receipt.ProjectedBytesAfter != receipt.BytesAfter {
		t.Fatalf("byte fields inconsistent: %+v", receipt)
	}
}

// TestReclaimStaysWithinItsBudget is the storage-manager backstop contract:
// the request returns after ReclaimBudget with the remainder left for later
// passes, never running until every page is back.
func TestReclaimStaysWithinItsBudget(t *testing.T) {
	db, path := openFileDB(t, true)
	fill(t, db, 200, 4)
	svc := newService(t, db, path, nil, nil)
	svc.opts.Policy.BatchPages, svc.opts.Policy.MinBatchPages, svc.opts.Policy.BatchHoldTarget = 4, 4, 0
	svc.opts.Policy.ReclaimBudget, svc.opts.Policy.BatchPause = 20*time.Millisecond, 10*time.Millisecond

	started := time.Now()
	receipt, err := svc.Reclaim(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}

	if time.Since(started) > 2*time.Second {
		t.Fatalf("reclaim ran %s past a 20ms budget", time.Since(started))
	}
	if receipt.Complete || receipt.After == nil || receipt.After.FreeBytes == 0 || !strings.Contains(receipt.Guidance, "remain") {
		t.Fatalf("bounded reclaim should leave work for later passes: %+v", receipt)
	}
}

func TestVacuumAdaptsTheBatchToTheHoldTarget(t *testing.T) {
	db, path := openFileDB(t, true)
	fill(t, db, 200, 4)
	svc := newService(t, db, path, nil, nil)
	// A clock that reports every batch as slow must shrink the batch to the
	// minimum, so later transactions each move MinBatchPages.
	var ticks int64
	svc.opts.Clock = func() time.Time { ticks++; return time.Unix(0, 0).Add(time.Duration(ticks) * time.Second) }
	svc.opts.Policy.BatchPages, svc.opts.Policy.MinBatchPages, svc.opts.Policy.BatchHoldTarget = 64, 8, 100*time.Millisecond
	before, _ := svc.Stats(t.Context())

	if _, err := svc.vacuumTo(t.Context(), 0, 12*time.Second); err != nil {
		t.Fatal(err)
	}

	after, _ := svc.Stats(t.Context())
	moved := before.FreelistCount - after.FreelistCount
	if moved <= 0 || moved >= 64*3 {
		t.Fatalf("moved %d pages; slow batches should shrink toward the minimum", moved)
	}
}

func TestReclaimOnANoneModeFileOnlyGivesGuidance(t *testing.T) {
	db, path := openFileDB(t, false)
	fill(t, db, 60, 2)
	svc := newService(t, db, path, nil, nil)

	receipt, err := svc.Reclaim(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Action != ActionCompactionRequired || receipt.After != nil || !strings.Contains(receipt.Guidance, "storage compact") {
		t.Fatalf("receipt=%+v", receipt)
	}
	if !receipt.CompactionRequired || receipt.Performed != PerformedNone || receipt.BytesAfter != receipt.BytesBefore || receipt.ProjectedBytesAfter >= receipt.BytesBefore {
		t.Fatalf("backstop receipt must report the fenced compaction and its projection: %+v", receipt)
	}
	if st, _ := svc.Stats(t.Context()); st.AutoVacuum != "none" {
		t.Fatal("the unauthenticated reclaim path rewrote the file")
	}
}

func TestLoadDeclaredBudgetReadsTheOwnerManifest(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios", "agent-manager", ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"service":{"name":"agent-manager"},"storage":{"rationale":"test","entries":{"data":{"rung":"owned","kind":"dir","class":"data","regenerable":false,"rationale":"db","budget":{"max_bytes":"12GiB","rationale":"alarm"}}}}}`
	if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	budget, err := LoadDeclaredBudget(root, "agent-manager", "data")

	if err != nil || budget.Bytes != 12<<30 || !strings.Contains(budget.Source, "service.json#storage.entries.data") {
		t.Fatalf("budget=%+v err=%v", budget, err)
	}
	if _, err := LoadDeclaredBudget(root, "agent-manager", "absent"); err == nil {
		t.Fatal("missing entry accepted")
	}
}

func TestHTTPContract(t *testing.T) {
	db, path := openFileDB(t, true)
	fill(t, db, 60, 3)
	fence := &drainedFence{state: FenceState{Closed: true}}
	svc := newService(t, db, path, fence, nil)
	authorized := true
	handler, err := NewHandler(svc, func(*http.Request) (string, int, error) {
		if !authorized {
			return "", http.StatusUnauthorized, errors.New("owner credential required")
		}
		return "owner", 0, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	serve := func(method, target, body string, test bool) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, target, strings.NewReader(body))
		if test {
			request = request.WithContext(coredb.WithTestMode(request.Context()))
		}
		response := httptest.NewRecorder()
		switch target {
		case "/api/v1/storage/health":
			handler.Health(response, request)
		case "/api/v1/storage/reclaim":
			handler.Reclaim(response, request)
		default:
			handler.Compact(response, request)
		}
		return response
	}

	if got := serve(http.MethodGet, "/api/v1/storage/health", "", false); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"freelistCount"`) || !strings.Contains(got.Body.String(), `"compaction"`) {
		t.Fatalf("health=%d %s", got.Code, got.Body.String())
	}
	if got := serve(http.MethodGet, "/api/v1/storage/health", "", true); got.Code != http.StatusConflict {
		t.Fatalf("test-mode health=%d", got.Code)
	}
	if got := serve(http.MethodPost, "/api/v1/storage/reclaim", `{"dry_run":true}`, false); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"dryRun":true`) {
		t.Fatalf("dry reclaim=%d %s", got.Code, got.Body.String())
	}
	if got := serve(http.MethodPost, "/api/v1/storage/reclaim", `{"dry_run":true,"force":1}`, false); got.Code != http.StatusBadRequest {
		t.Fatalf("unknown field=%d", got.Code)
	}
	if got := serve(http.MethodPost, "/api/v1/storage/reclaim", ``, false); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"complete":true`) {
		t.Fatalf("reclaim=%d %s", got.Code, got.Body.String())
	}
	authorized = false
	if got := serve(http.MethodPost, "/api/v1/storage/compact", `{"reason":"x"}`, false); got.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized compact=%d", got.Code)
	}
	authorized = true
	if got := serve(http.MethodPost, "/api/v1/storage/compact", `{}`, false); got.Code != http.StatusBadRequest {
		t.Fatalf("reasonless compact=%d", got.Code)
	}
	if got := serve(http.MethodPost, "/api/v1/storage/compact", `{"reason":"rollout"}`, false); got.Code != http.StatusConflict {
		t.Fatalf("undrained compact=%d %s", got.Code, got.Body.String())
	}
	fence.mu.Lock()
	fence.state.Drained = true
	fence.mu.Unlock()
	if got := serve(http.MethodPost, "/api/v1/storage/compact", `{"reason":"rollout"}`, false); got.Code != http.StatusAccepted || !strings.Contains(got.Body.String(), `"running"`) {
		t.Fatalf("compact=%d %s", got.Code, got.Body.String())
	}
	deadline := time.Now().Add(10 * time.Second)
	for svc.CompactionStatus().State == CompactionRunning && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if status := svc.CompactionStatus(); status.State != CompactionSucceeded || status.Actor != "owner" {
		t.Fatalf("status=%+v", status)
	}
}
