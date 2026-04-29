// Tests that auto-heal failure history persists across simulated API
// restarts (Round 3 Phase 6). Uses the real SQLite-backed repository
// so the schema, the upsert SQL, and the in-memory cache all stay in
// lockstep.

package sandbox

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/repository"
	"workspace-sandbox/internal/types"

	_ "modernc.org/sqlite"
)

func newDurabilityRepo(t *testing.T) *repository.SandboxRepository {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(repository.SchemaSQL); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return repository.NewSandboxRepository(db, clock.System{})
}

// TestHealTracker_PersistsAcrossRestart records a failure, simulates
// a restart by constructing a fresh tracker against the same repo,
// and verifies the failure count survived. Pre-Phase-6 this returned
// 0 (silent reset) — the load-bearing assertion of this round.
func TestHealTracker_PersistsAcrossRestart(t *testing.T) {
	repo := newDurabilityRepo(t)
	ctx := context.Background()

	// Seed a sandbox so the foreign-key constraint is satisfied.
	sb := &types.Sandbox{
		ID:            uuid.New(),
		ScopePath:     "/p/s",
		ProjectRoot:   "/p",
		ReservedPath:  "/p/s",
		ReservedPaths: []string{"/p/s"},
		Status:        types.StatusActive,
		DriverID:      "fuse-overlayfs",
		DriverVersion: "1.0",
		OwnerType:     types.OwnerTypeAgent,
	}
	if err := repo.Create(ctx, sb); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Round 1: record three failures.
	tracker := newHealTracker().withRepo(repo)
	for i := 0; i < 3; i++ {
		tracker.recordFailure(sb.ID, time.Now(), "stale mount")
	}
	if got := tracker.get(sb.ID); got == nil || got.consecutiveFailures != 3 {
		t.Fatalf("pre-restart count = %v, want 3", got)
	}

	// Simulated restart: throw away the in-memory map, reload from repo.
	tracker = newHealTracker().withRepo(repo)
	if err := tracker.loadFromRepo(ctx); err != nil {
		t.Fatalf("loadFromRepo: %v", err)
	}
	got := tracker.get(sb.ID)
	if got == nil {
		t.Fatal("post-restart heal state was lost; expected count to survive")
	}
	if got.consecutiveFailures != 3 {
		t.Errorf("post-restart count = %d, want 3", got.consecutiveFailures)
	}
}

// TestHealTracker_ResetClearsRepo confirms that a successful heal
// removes the durable row, not just the in-memory entry. Without
// this, a sandbox that healed on the previous run would carry old
// failure history into the next reboot and be denied retries.
func TestHealTracker_ResetClearsRepo(t *testing.T) {
	repo := newDurabilityRepo(t)
	ctx := context.Background()

	sb := &types.Sandbox{
		ID:            uuid.New(),
		ScopePath:     "/p/s",
		ProjectRoot:   "/p",
		ReservedPath:  "/p/s",
		ReservedPaths: []string{"/p/s"},
		Status:        types.StatusActive,
		DriverID:      "fuse-overlayfs",
		DriverVersion: "1.0",
		OwnerType:     types.OwnerTypeAgent,
	}
	if err := repo.Create(ctx, sb); err != nil {
		t.Fatalf("Create: %v", err)
	}

	tracker := newHealTracker().withRepo(repo)
	tracker.recordFailure(sb.ID, time.Now(), "x")
	tracker.reset(sb.ID)

	row, err := repo.GetHealState(ctx, sb.ID)
	if err != nil {
		t.Fatalf("GetHealState: %v", err)
	}
	if row != nil {
		t.Errorf("expected heal_state cleared after reset, got %+v", row)
	}
}

// TestHealTracker_MetricsExposeActiveCount verifies that the
// observability surface (active sandboxes + max consecutive failures)
// reflects in-memory state. Pins the Phase-6 metrics contract.
func TestHealTracker_MetricsExposeActiveCount(t *testing.T) {
	tracker := newHealTracker()
	tracker.recordFailure(uuid.New(), time.Now(), "")
	tracker.recordFailure(uuid.New(), time.Now(), "")
	id3 := uuid.New()
	tracker.recordFailure(id3, time.Now(), "")
	tracker.recordFailure(id3, time.Now(), "")
	tracker.recordFailure(id3, time.Now(), "")

	if got := tracker.activeCount(); got != 3 {
		t.Errorf("activeCount = %d, want 3", got)
	}
	if got := tracker.maxFailuresSeen(); got != 3 {
		t.Errorf("maxFailuresSeen = %d, want 3", got)
	}
}
