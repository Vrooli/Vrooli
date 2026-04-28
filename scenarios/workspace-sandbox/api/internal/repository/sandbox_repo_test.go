// Behavior-driven repository tests. These run against a real on-disk SQLite
// database (modernc.org/sqlite) created in t.TempDir(), so query-shape
// regressions surface as real failures rather than as drifted mock
// expectations.

package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"

	_ "modernc.org/sqlite"
)

// newTestDB returns a fresh, fully-initialized SQLite handle backed by a file
// in the test's temp dir. The handle is closed automatically via t.Cleanup.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	dsn := path +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(5000)" +
		"&_txlock=immediate"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Single-connection pool mirrors production main.go.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(SchemaSQL); err != nil {
		db.Close()
		t.Fatalf("apply schema: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func newTestRepo(t *testing.T) *SandboxRepository {
	return NewSandboxRepository(newTestDB(t))
}

func newTestSandbox() *types.Sandbox {
	return &types.Sandbox{
		ID:            uuid.New(),
		ScopePath:     "/project/src",
		ReservedPath:  "/project/src",
		ReservedPaths: []string{"/project/src"},
		ProjectRoot:   "/project",
		Owner:         "test-agent",
		OwnerType:     types.OwnerTypeAgent,
		Status:        types.StatusActive,
		Driver:        "fuse-overlayfs",
		DriverVersion: "1.0.0",
		Tags:          []string{"unit", "test"},
		Metadata:      map[string]any{"key": "value"},
		Behavior: types.SandboxBehavior{
			Acceptance: types.AcceptanceConfig{Mode: "allowlist"},
		},
	}
}

// ---------------------------------------------------------------------------
// Sanity / interface checks
// ---------------------------------------------------------------------------

func TestSandboxRepository_ImplementsRepository(t *testing.T) {
	var _ Repository = (*SandboxRepository)(nil)
	var _ TxRepository = (*TxSandboxRepository)(nil)
}

func TestSchemaSQL_AppliesCleanly(t *testing.T) {
	if strings.TrimSpace(SchemaSQL) == "" {
		t.Fatal("SchemaSQL is empty")
	}
	db := newTestDB(t) // applying twice is a smoke test of idempotency
	if _, err := db.Exec(SchemaSQL); err != nil {
		t.Fatalf("re-applying schema failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CRUD round-trip
// ---------------------------------------------------------------------------

func TestCreate_AndGet_RoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)
	in := newTestSandbox()

	if err := repo.Create(ctx, in); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, in.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil for known sandbox")
	}
	if got.ID != in.ID {
		t.Errorf("ID mismatch: got %s want %s", got.ID, in.ID)
	}
	if got.ScopePath != in.ScopePath {
		t.Errorf("ScopePath mismatch: got %q want %q", got.ScopePath, in.ScopePath)
	}
	if got.Status != in.Status {
		t.Errorf("Status mismatch: got %q want %q", got.Status, in.Status)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "unit" || got.Tags[1] != "test" {
		t.Errorf("Tags mismatch: %v", got.Tags)
	}
	if got.Metadata["key"] != "value" {
		t.Errorf("Metadata mismatch: %v", got.Metadata)
	}
	if got.Behavior.Acceptance.Mode != "allowlist" {
		t.Errorf("Behavior mismatch: %v", got.Behavior)
	}
	if len(got.ReservedPaths) != 1 || got.ReservedPaths[0] != "/project/src" {
		t.Errorf("ReservedPaths mismatch: %v", got.ReservedPaths)
	}
	if got.Version != 1 {
		t.Errorf("Version got %d want 1", got.Version)
	}
	if got.CreatedAt.IsZero() || got.LastUsedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Error("expected non-zero timestamps")
	}
}

func TestGet_MissingReturnsNil(t *testing.T) {
	got, err := newTestRepo(t).Get(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Errorf("Get on missing sandbox returned %v, want nil", got)
	}
}

func TestUpdate_BumpsVersionAndPersistsFields(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)
	s := newTestSandbox()
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	s.Status = types.StatusStopped
	s.SizeBytes = 4096
	s.ActivePIDs = []int{42, 99}
	if err := repo.Update(ctx, s); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if s.Version != 2 {
		t.Errorf("Version got %d want 2", s.Version)
	}

	got, err := repo.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != types.StatusStopped {
		t.Errorf("Status got %q want stopped", got.Status)
	}
	if got.SizeBytes != 4096 {
		t.Errorf("SizeBytes got %d want 4096", got.SizeBytes)
	}
	if len(got.ActivePIDs) != 2 || got.ActivePIDs[0] != 42 || got.ActivePIDs[1] != 99 {
		t.Errorf("ActivePIDs mismatch: %v", got.ActivePIDs)
	}
}

func TestDelete_MarksDeletedAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)
	s := newTestSandbox()
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, s.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err := repo.Get(ctx, s.ID)
	if err != nil || got == nil {
		t.Fatalf("Get after Delete: %v %v", err, got)
	}
	if got.Status != types.StatusDeleted {
		t.Errorf("Status after delete got %q want deleted", got.Status)
	}
	if got.DeletedAt == nil {
		t.Error("DeletedAt was not set")
	}

	if err := repo.Delete(ctx, s.ID); err == nil {
		t.Error("Delete on already-deleted sandbox should error")
	}
}

// ---------------------------------------------------------------------------
// Optimistic locking
// ---------------------------------------------------------------------------

func TestUpdateWithVersionCheck_RejectsStaleVersion(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)
	s := newTestSandbox()
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// First update succeeds.
	first := *s
	first.Status = types.StatusStopped
	if err := repo.UpdateWithVersionCheck(ctx, &first, 1); err != nil {
		t.Fatalf("first UpdateWithVersionCheck: %v", err)
	}

	// Second update with stale version should fail with concurrent-mod error.
	stale := *s
	stale.Status = types.StatusError
	err := repo.UpdateWithVersionCheck(ctx, &stale, 1)
	if err == nil {
		t.Fatal("expected concurrent-modification error, got nil")
	}
	if !strings.Contains(err.Error(), "concurrent") && !strings.Contains(err.Error(), "version") {
		t.Errorf("error %q did not mention concurrent/version", err)
	}
}

// ---------------------------------------------------------------------------
// List / filtering
// ---------------------------------------------------------------------------

func TestList_FiltersByStatusAndOwner(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	for i, owner := range []string{"a", "a", "b"} {
		s := newTestSandbox()
		s.ID = uuid.New()
		s.Owner = owner
		s.ScopePath = "/p/s" + string(rune('0'+i))
		s.ReservedPath = s.ScopePath
		s.ReservedPaths = []string{s.ScopePath}
		s.ProjectRoot = "/p"
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}

	res, err := repo.List(ctx, &types.ListFilter{Owner: "a"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if res.TotalCount != 2 || len(res.Sandboxes) != 2 {
		t.Errorf("expected 2 sandboxes for owner a, got total=%d len=%d", res.TotalCount, len(res.Sandboxes))
	}
	for _, s := range res.Sandboxes {
		if s.Owner != "a" {
			t.Errorf("unexpected owner %q in filter result", s.Owner)
		}
	}
}

// ---------------------------------------------------------------------------
// CheckScopeOverlap (Go-side replacement for the SQL function)
// ---------------------------------------------------------------------------

func TestCheckScopeOverlap_DetectsAncestorAndDescendant(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	existing := newTestSandbox()
	existing.ScopePath = "/p/src/components"
	existing.ReservedPath = "/p/src/components"
	existing.ReservedPaths = []string{"/p/src/components"}
	existing.ProjectRoot = "/p"
	if err := repo.Create(ctx, existing); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Ancestor scope conflicts.
	conflicts, err := repo.CheckScopeOverlap(ctx, "/p/src", "/p", nil)
	if err != nil {
		t.Fatalf("CheckScopeOverlap (ancestor): %v", err)
	}
	if len(conflicts) == 0 {
		t.Error("expected ancestor conflict, got none")
	}

	// Descendant scope conflicts.
	conflicts, err = repo.CheckScopeOverlap(ctx, "/p/src/components/widget", "/p", nil)
	if err != nil {
		t.Fatalf("CheckScopeOverlap (descendant): %v", err)
	}
	if len(conflicts) == 0 {
		t.Error("expected descendant conflict, got none")
	}

	// Sibling scope is fine.
	conflicts, err = repo.CheckScopeOverlap(ctx, "/p/docs", "/p", nil)
	if err != nil {
		t.Fatalf("CheckScopeOverlap (sibling): %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts for sibling, got %v", conflicts)
	}

	// Different project is fine.
	conflicts, err = repo.CheckScopeOverlap(ctx, "/p/src", "/other-project", nil)
	if err != nil {
		t.Fatalf("CheckScopeOverlap (cross-project): %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts cross-project, got %v", conflicts)
	}
}

// TestCheckScopeOverlap_ConcurrentCreates exercises the BEGIN IMMEDIATE
// serialization wired by the SQLite DSN _txlock=immediate parameter. Two
// concurrent transactions racing to claim the same reserved path should
// serialize: only the first should observe a clean overlap-check, every
// other should see the first's row.
func TestCheckScopeOverlap_ConcurrentCreates(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	const concurrency = 4
	var wg sync.WaitGroup
	successes := make([]bool, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tx, err := repo.BeginTx(ctx)
			if err != nil {
				return
			}
			conflicts, err := tx.CheckScopeOverlap(ctx, "/p/shared", "/p", nil)
			if err != nil {
				_ = tx.Rollback()
				return
			}
			if len(conflicts) > 0 {
				_ = tx.Rollback()
				return
			}
			s := newTestSandbox()
			s.ID = uuid.New()
			s.ScopePath = "/p/shared"
			s.ReservedPath = "/p/shared"
			s.ReservedPaths = []string{"/p/shared"}
			s.ProjectRoot = "/p"
			s.IdempotencyKey = "" // unique
			if err := tx.Create(ctx, s); err != nil {
				_ = tx.Rollback()
				return
			}
			if err := tx.Commit(); err != nil {
				return
			}
			successes[i] = true
		}(i)
	}
	wg.Wait()

	count := 0
	for _, ok := range successes {
		if ok {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 successful create under concurrency, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

func TestGetStats_CountsByStatus(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	// Create then Update because the canonical insert does not persist
	// size_bytes (the driver sets it post-mount via Update).
	make := func(status types.Status, size int64) {
		s := newTestSandbox()
		s.ID = uuid.New()
		s.Status = status
		s.ScopePath = "/p/" + string(status)
		s.ReservedPath = s.ScopePath
		s.ReservedPaths = []string{s.ScopePath}
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Create %s: %v", status, err)
		}
		s.SizeBytes = size
		if err := repo.Update(ctx, s); err != nil {
			t.Fatalf("Update %s: %v", status, err)
		}
	}
	make(types.StatusActive, 100)
	make(types.StatusActive, 200)
	make(types.StatusStopped, 50)
	make(types.StatusError, 25)

	stats, err := repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalCount != 4 {
		t.Errorf("TotalCount got %d want 4", stats.TotalCount)
	}
	if stats.ActiveCount != 2 {
		t.Errorf("ActiveCount got %d want 2", stats.ActiveCount)
	}
	if stats.StoppedCount != 1 {
		t.Errorf("StoppedCount got %d want 1", stats.StoppedCount)
	}
	if stats.ErrorCount != 1 {
		t.Errorf("ErrorCount got %d want 1", stats.ErrorCount)
	}
	if stats.TotalSizeBytes != 375 {
		t.Errorf("TotalSizeBytes got %d want 375", stats.TotalSizeBytes)
	}
}

// ---------------------------------------------------------------------------
// Idempotency
// ---------------------------------------------------------------------------

func TestFindByIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	s := newTestSandbox()
	s.IdempotencyKey = "client-key-1"
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindByIdempotencyKey(ctx, "client-key-1")
	if err != nil {
		t.Fatalf("FindByIdempotencyKey: %v", err)
	}
	if got == nil || got.ID != s.ID {
		t.Errorf("FindByIdempotencyKey returned %v, want id %s", got, s.ID)
	}

	missing, err := repo.FindByIdempotencyKey(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("FindByIdempotencyKey missing: %v", err)
	}
	if missing != nil {
		t.Errorf("FindByIdempotencyKey for missing key returned %v, want nil", missing)
	}

	none, err := repo.FindByIdempotencyKey(ctx, "")
	if err != nil || none != nil {
		t.Errorf("FindByIdempotencyKey('') = %v, %v; want nil, nil", none, err)
	}
}

// ---------------------------------------------------------------------------
// Provenance / applied_changes
// ---------------------------------------------------------------------------

func TestRecordAppliedChanges_AndGetPendingChanges(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	s := newTestSandbox()
	s.ProjectRoot = "/proj"
	s.ScopePath = "/proj/src"
	s.ReservedPath = "/proj/src"
	s.ReservedPaths = []string{"/proj/src"}
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	c := &types.AppliedChange{
		ID:                uuid.New(),
		SandboxID:         s.ID,
		SandboxOwner:      "agent-1",
		SandboxOwnerType:  string(types.OwnerTypeAgent),
		FilePath:          "/proj/src/foo.go",
		ProjectRoot:       "/proj",
		ChangeType:        "modified",
		FileSize:          42,
		AgentManagerRunID: "run-abc",
		RunOutcome:        "success",
		ProvenanceState:   string(types.ProvenanceFileStateApplied),
		ConversationID:    "conv-xyz",
		CostUSD:           0.42,
	}
	if err := repo.RecordAppliedChanges(ctx, []*types.AppliedChange{c}); err != nil {
		t.Fatalf("RecordAppliedChanges: %v", err)
	}

	files, err := repo.GetPendingChangeFiles(ctx, "/proj", nil)
	if err != nil {
		t.Fatalf("GetPendingChangeFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 pending file, got %d", len(files))
	}
	got := files[0]
	if got.RunOutcome != "success" || got.ConversationID != "conv-xyz" || got.CostUSD != 0.42 {
		t.Errorf("provenance fields not round-tripped: %+v", got)
	}

	groups, err := repo.GetPendingChangesByRun(ctx, "/proj")
	if err != nil {
		t.Fatalf("GetPendingChangesByRun: %v", err)
	}
	if len(groups) != 1 || groups[0].RunID != "run-abc" {
		t.Errorf("expected one group keyed by run-abc, got %v", groups)
	}
	if len(groups[0].Files) != 1 || groups[0].Files[0].State != types.ProvenanceFileStateApplied {
		t.Errorf("file state mismatch: %v", groups[0].Files)
	}
}

func TestMarkChangesCommitted(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	s := newTestSandbox()
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	c := &types.AppliedChange{
		ID:           uuid.New(),
		SandboxID:    s.ID,
		SandboxOwner: "a",
		FilePath:     "/project/src/x.go",
		ProjectRoot:  "/project",
		ChangeType:   "modified",
	}
	if err := repo.RecordAppliedChanges(ctx, []*types.AppliedChange{c}); err != nil {
		t.Fatalf("RecordAppliedChanges: %v", err)
	}

	if err := repo.MarkChangesCommitted(ctx, []uuid.UUID{c.ID}, "abc123", "ship it"); err != nil {
		t.Fatalf("MarkChangesCommitted: %v", err)
	}

	prov, err := repo.GetFileProvenance(ctx, "/project/src/x.go", "/project", 10)
	if err != nil {
		t.Fatalf("GetFileProvenance: %v", err)
	}
	if len(prov) != 1 {
		t.Fatalf("expected 1 record, got %d", len(prov))
	}
	if prov[0].CommittedAt == nil {
		t.Error("CommittedAt should be set")
	}
	if prov[0].CommitHash != "abc123" {
		t.Errorf("CommitHash got %q want abc123", prov[0].CommitHash)
	}
}

// ---------------------------------------------------------------------------
// Audit log
// ---------------------------------------------------------------------------

func TestLogAuditEvent_AndGetAuditLog(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)
	s := newTestSandbox()
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	id := s.ID
	for _, et := range []string{"created", "mounted", "stopped"} {
		if err := repo.LogAuditEvent(ctx, &types.AuditEvent{
			SandboxID: &id,
			EventType: et,
			Actor:     "tester",
			Details:   map[string]any{"step": et},
		}); err != nil {
			t.Fatalf("LogAuditEvent %s: %v", et, err)
		}
		// Slight separation so event_time DESC ordering is deterministic.
		time.Sleep(2 * time.Millisecond)
	}

	events, total, err := repo.GetAuditLog(ctx, &id, 10, 0)
	if err != nil {
		t.Fatalf("GetAuditLog: %v", err)
	}
	if total != 3 || len(events) != 3 {
		t.Errorf("expected 3 events, got total=%d len=%d", total, len(events))
	}
	if events[0].EventType != "stopped" {
		t.Errorf("first event should be most recent (stopped), got %q", events[0].EventType)
	}
	if events[0].Details["step"] != "stopped" {
		t.Errorf("Details did not round-trip: %v", events[0].Details)
	}
}
