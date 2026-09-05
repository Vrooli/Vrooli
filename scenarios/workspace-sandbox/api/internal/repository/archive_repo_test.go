package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"

	"github.com/vrooli/api-core/schedule"
)

// newTestArchiveRepo wires an archive repo + a sandbox repo against a
// fresh test DB. Most tests need both because archives have a foreign
// key to sandboxes(id).
func newTestArchiveRepo(t *testing.T) (*SandboxArchiveRepository, *SandboxRepository, *sql.DB) {
	t.Helper()
	db := newTestDB(t)
	clk := schedule.System()
	return NewArchiveRepository(db, clk), NewSandboxRepository(db, clk), db
}

// seedSandbox creates a sandbox and transitions it to terminalStatus
// without going through the service-layer state machine. Tests need
// the sandbox row to satisfy the FK; the snapshot service exercises
// the real transition.
func seedSandbox(t *testing.T, repo *SandboxRepository, terminalStatus types.Status) *types.Sandbox {
	t.Helper()
	s := newTestSandbox()
	s.Status = types.StatusActive
	if err := repo.Create(context.Background(), s); err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	s.Status = terminalStatus
	if err := repo.Update(context.Background(), s); err != nil {
		t.Fatalf("update sandbox: %v", err)
	}
	return s
}

func makeArchive(sandboxID uuid.UUID) *types.DiffArchive {
	return &types.DiffArchive{
		SandboxID:     sandboxID,
		SnapshotAt:    time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC),
		ArchiveState:  types.ArchiveStateComplete,
		SandboxStatus: types.StatusApproved,
		Files: []types.ArchivedFileEntry{
			{Path: "a.txt", ChangeType: types.ChangeTypeAdded, Size: 3, BlobSHA256: strings.Repeat("a", 64)},
			{Path: "b.txt", ChangeType: types.ChangeTypeModified, Size: 7, BlobSHA256: strings.Repeat("b", 64)},
		},
		Stats: types.DiffStats{
			FilesChanged: 2, FilesAdded: 1, FilesModified: 1, TotalBytes: 10,
		},
		UnifiedDiffSHA256: strings.Repeat("c", 64),
		TotalBlobBytes:    1234,
		ProjectRoot:       "/project",
		Owner:             "agent-7",
		AgentManagerRunID: "run-abc",
	}
}

func TestArchiveRepository_InsertAndGet_Roundtrip(t *testing.T) {
	ar, sr, _ := newTestArchiveRepo(t)
	s := seedSandbox(t, sr, types.StatusApproved)
	a := makeArchive(s.ID)

	if err := ar.Insert(context.Background(), nil, a); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := ar.Get(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("get returned nil archive after insert")
	}
	if got.SandboxID != a.SandboxID ||
		got.ArchiveState != a.ArchiveState ||
		got.SandboxStatus != a.SandboxStatus ||
		got.UnifiedDiffSHA256 != a.UnifiedDiffSHA256 ||
		got.TotalBlobBytes != a.TotalBlobBytes ||
		got.ProjectRoot != a.ProjectRoot ||
		got.Owner != a.Owner ||
		got.AgentManagerRunID != a.AgentManagerRunID {
		t.Errorf("scalar mismatch: got=%+v want=%+v", *got, *a)
	}
	if !got.SnapshotAt.Equal(a.SnapshotAt) {
		t.Errorf("snapshot_at: got %v, want %v", got.SnapshotAt, a.SnapshotAt)
	}
	if len(got.Files) != len(a.Files) {
		t.Fatalf("files len: got %d, want %d", len(got.Files), len(a.Files))
	}
	for i := range got.Files {
		if got.Files[i] != a.Files[i] {
			t.Errorf("file[%d]: got %+v, want %+v", i, got.Files[i], a.Files[i])
		}
	}
	if got.Stats != a.Stats {
		t.Errorf("stats: got %+v, want %+v", got.Stats, a.Stats)
	}
}

func TestArchiveRepository_Get_NotFound(t *testing.T) {
	ar, _, _ := newTestArchiveRepo(t)
	got, err := ar.Get(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing archive, got %+v", got)
	}
}

func TestArchiveRepository_Insert_Validations(t *testing.T) {
	ar, sr, _ := newTestArchiveRepo(t)
	s := seedSandbox(t, sr, types.StatusApproved)

	cases := []struct {
		name string
		mut  func(*types.DiffArchive)
		want string
	}{
		{
			name: "nil archive",
			mut:  nil,
			want: "nil archive",
		},
		{
			name: "missing id",
			mut:  func(a *types.DiffArchive) { a.SandboxID = uuid.Nil },
			want: "sandbox_id required",
		},
		{
			name: "invalid archive state",
			mut:  func(a *types.DiffArchive) { a.ArchiveState = "weird" },
			want: "invalid archive_state",
		},
		{
			name: "non-terminal sandbox status",
			mut:  func(a *types.DiffArchive) { a.SandboxStatus = types.StatusActive },
			want: "sandbox_status must be terminal",
		},
		{
			name: "error sandbox status (terminal but not archive-bearing)",
			mut:  func(a *types.DiffArchive) { a.SandboxStatus = types.StatusError },
			want: "is not an archive-bearing terminal state",
		},
		{
			name: "not_captured with files",
			mut: func(a *types.DiffArchive) {
				a.ArchiveState = types.ArchiveStateNotCaptured
				a.Files = []types.ArchivedFileEntry{{Path: "ghost"}}
				a.UnifiedDiffSHA256 = ""
				a.TotalBlobBytes = 0
			},
			want: "not_captured archive has files",
		},
		{
			name: "not_captured with unified diff",
			mut: func(a *types.DiffArchive) {
				a.ArchiveState = types.ArchiveStateNotCaptured
				a.Files = nil
				a.UnifiedDiffSHA256 = strings.Repeat("d", 64)
				a.TotalBlobBytes = 0
			},
			want: "not_captured archive has unified diff",
		},
		{
			name: "not_captured with non-zero bytes",
			mut: func(a *types.DiffArchive) {
				a.ArchiveState = types.ArchiveStateNotCaptured
				a.Files = nil
				a.UnifiedDiffSHA256 = ""
				a.TotalBlobBytes = 1
			},
			want: "non-zero total_blob_bytes",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var a *types.DiffArchive
			if tc.mut != nil {
				a = makeArchive(s.ID)
				tc.mut(a)
			}
			err := ar.Insert(context.Background(), nil, a)
			if err == nil {
				t.Fatalf("expected error containing %q; got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want substring %q", err, tc.want)
			}
		})
	}
}

func TestArchiveRepository_Insert_NotCaptured_Allowed(t *testing.T) {
	ar, sr, _ := newTestArchiveRepo(t)
	s := seedSandbox(t, sr, types.StatusDeleted)

	a := &types.DiffArchive{
		SandboxID:     s.ID,
		ArchiveState:  types.ArchiveStateNotCaptured,
		SandboxStatus: types.StatusDeleted,
		ProjectRoot:   "/project",
		Owner:         "system",
	}
	if err := ar.Insert(context.Background(), nil, a); err != nil {
		t.Fatalf("insert not_captured: %v", err)
	}
	got, err := ar.Get(context.Background(), s.ID)
	if err != nil || got == nil {
		t.Fatalf("get: %v / %+v", err, got)
	}
	if got.ArchiveState != types.ArchiveStateNotCaptured {
		t.Errorf("archive_state = %q, want not_captured", got.ArchiveState)
	}
	if len(got.Files) != 0 {
		t.Errorf("files = %v, want empty", got.Files)
	}
}

func TestArchiveRepository_Insert_IsIdempotentOnSandboxID(t *testing.T) {
	ar, sr, _ := newTestArchiveRepo(t)
	s := seedSandbox(t, sr, types.StatusApproved)

	a1 := makeArchive(s.ID)
	a1.UnifiedDiffSHA256 = strings.Repeat("1", 64)
	if err := ar.Insert(context.Background(), nil, a1); err != nil {
		t.Fatalf("insert #1: %v", err)
	}

	a2 := makeArchive(s.ID)
	a2.UnifiedDiffSHA256 = strings.Repeat("2", 64) // different content
	a2.SnapshotAt = a1.SnapshotAt.Add(time.Minute)
	if err := ar.Insert(context.Background(), nil, a2); err != nil {
		t.Fatalf("insert #2: %v", err)
	}

	got, err := ar.Get(context.Background(), s.ID)
	if err != nil || got == nil {
		t.Fatalf("get: %v / %+v", err, got)
	}
	if got.UnifiedDiffSHA256 != a2.UnifiedDiffSHA256 {
		t.Errorf("expected re-insert to overwrite; got unified hash %q, want %q",
			got.UnifiedDiffSHA256, a2.UnifiedDiffSHA256)
	}
}

func TestArchiveRepository_Insert_TransactionAtomicity(t *testing.T) {
	ar, sr, db := newTestArchiveRepo(t)
	ctx := context.Background()
	s := seedSandbox(t, sr, types.StatusApproved)

	// Open a tx, insert, then rollback. After rollback the row must
	// be absent. We do NOT poll Get during the open tx because the
	// production pool is sized to a single connection (mirroring main),
	// and a Get on a different goroutine would deadlock waiting for
	// the same connection — that's by design (SQLite serializes
	// writes through one connection).
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begintx: %v", err)
	}
	a := makeArchive(s.ID)
	if err := ar.Insert(ctx, tx, a); err != nil {
		_ = tx.Rollback()
		t.Fatalf("insert in tx: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	got, err := ar.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("get post-rollback: %v", err)
	}
	if got != nil {
		t.Errorf("rollback failed to discard insert: %+v", got)
	}

	// Now open a fresh tx, insert, commit — row must persist.
	tx2, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begintx 2: %v", err)
	}
	if err := ar.Insert(ctx, tx2, a); err != nil {
		_ = tx2.Rollback()
		t.Fatalf("insert in tx2: %v", err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	got, err = ar.Get(ctx, s.ID)
	if err != nil || got == nil {
		t.Errorf("expected row visible after commit; got=%+v err=%v", got, err)
	}
}

func TestArchiveRepository_List_FiltersAndPagination(t *testing.T) {
	ar, sr, _ := newTestArchiveRepo(t)
	ctx := context.Background()

	// Seed three archives with different statuses, projects, snapshot_at.
	type seed struct {
		status types.Status
		proj   string
		owner  string
		runID  string
		when   time.Time
		bytes  int64
	}
	seeds := []seed{
		{types.StatusApproved, "/p1", "alice", "run-1", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 100},
		{types.StatusRejected, "/p1", "bob", "run-2", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), 200},
		{types.StatusDeleted, "/p2", "alice", "run-3", time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), 300},
		{types.StatusApproved, "/p2", "carol", "run-4", time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), 400},
	}
	for _, sd := range seeds {
		s := seedSandbox(t, sr, sd.status)
		a := &types.DiffArchive{
			SandboxID:         s.ID,
			SnapshotAt:        sd.when,
			ArchiveState:      types.ArchiveStateComplete,
			SandboxStatus:     sd.status,
			ProjectRoot:       sd.proj,
			Owner:             sd.owner,
			AgentManagerRunID: sd.runID,
			TotalBlobBytes:    sd.bytes,
			Stats:             types.DiffStats{FilesChanged: 1},
		}
		if err := ar.Insert(ctx, nil, a); err != nil {
			t.Fatalf("seed insert: %v", err)
		}
	}

	t.Run("default sort newest-first", func(t *testing.T) {
		got, total, err := ar.List(ctx, types.ArchiveListFilter{})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if total != 4 {
			t.Errorf("total = %d, want 4", total)
		}
		if len(got) != 4 {
			t.Fatalf("len = %d, want 4", len(got))
		}
		// Newest first.
		for i := 0; i < len(got)-1; i++ {
			if got[i].SnapshotAt.Before(got[i+1].SnapshotAt) {
				t.Errorf("default order not desc: [%d]=%v before [%d]=%v",
					i, got[i].SnapshotAt, i+1, got[i+1].SnapshotAt)
			}
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		got, total, err := ar.List(ctx, types.ArchiveListFilter{
			Statuses: []types.Status{types.StatusApproved},
		})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if total != 2 || len(got) != 2 {
			t.Errorf("approved count: total=%d len=%d, want 2/2", total, len(got))
		}
		for _, a := range got {
			if a.SandboxStatus != types.StatusApproved {
				t.Errorf("non-approved leaked: %+v", a.SandboxStatus)
			}
		}
	})

	t.Run("filter by project_root", func(t *testing.T) {
		got, total, err := ar.List(ctx, types.ArchiveListFilter{ProjectRoot: "/p2"})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if total != 2 || len(got) != 2 {
			t.Errorf("/p2 count: total=%d len=%d, want 2/2", total, len(got))
		}
	})

	t.Run("filter by owner", func(t *testing.T) {
		got, total, err := ar.List(ctx, types.ArchiveListFilter{Owner: "alice"})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if total != 2 || len(got) != 2 {
			t.Errorf("owner=alice: total=%d len=%d, want 2/2", total, len(got))
		}
	})

	t.Run("filter by agent_manager_run_id", func(t *testing.T) {
		got, total, err := ar.List(ctx, types.ArchiveListFilter{AgentManagerRunID: "run-3"})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if total != 1 || len(got) != 1 {
			t.Errorf("run-3: total=%d len=%d, want 1/1", total, len(got))
		}
	})

	t.Run("date range", func(t *testing.T) {
		got, total, err := ar.List(ctx, types.ArchiveListFilter{
			SnapshotAtFrom: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			SnapshotAtTo:   time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if total != 2 || len(got) != 2 {
			t.Errorf("date range: total=%d len=%d, want 2/2", total, len(got))
		}
	})

	t.Run("free-text search hits owner", func(t *testing.T) {
		got, total, err := ar.List(ctx, types.ArchiveListFilter{Search: "ali"})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if total != 2 || len(got) != 2 {
			t.Errorf("search=ali: total=%d len=%d, want 2/2", total, len(got))
		}
	})

	t.Run("free-text search escapes wildcards", func(t *testing.T) {
		// run-2 contains a literal '-'; "%" should be treated literally.
		got, _, err := ar.List(ctx, types.ArchiveListFilter{Search: "%"})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("'%%' should match no rows after escape; got %d", len(got))
		}
	})

	t.Run("sort by total_blob_bytes desc", func(t *testing.T) {
		got, _, err := ar.List(ctx, types.ArchiveListFilter{
			SortBy:   "total_blob_bytes",
			SortDesc: true,
		})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		for i := 0; i < len(got)-1; i++ {
			if got[i].TotalBlobBytes < got[i+1].TotalBlobBytes {
				t.Errorf("size desc violated at [%d]/[%d]: %d / %d",
					i, i+1, got[i].TotalBlobBytes, got[i+1].TotalBlobBytes)
			}
		}
	})

	t.Run("invalid sort_by rejected", func(t *testing.T) {
		_, _, err := ar.List(ctx, types.ArchiveListFilter{SortBy: "drop_table"})
		if err == nil || !strings.Contains(err.Error(), "invalid sort_by") {
			t.Errorf("expected invalid sort_by error; got %v", err)
		}
	})

	t.Run("invalid status rejected", func(t *testing.T) {
		_, _, err := ar.List(ctx, types.ArchiveListFilter{
			Statuses: []types.Status{types.StatusActive},
		})
		if err == nil || !strings.Contains(err.Error(), "invalid status filter") {
			t.Errorf("expected invalid status error; got %v", err)
		}
	})

	t.Run("limit and offset paginate", func(t *testing.T) {
		got, total, err := ar.List(ctx, types.ArchiveListFilter{Limit: 2, Offset: 0})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if total != 4 {
			t.Errorf("total = %d, want 4", total)
		}
		if len(got) != 2 {
			t.Errorf("page 1 len = %d, want 2", len(got))
		}

		page2, _, err := ar.List(ctx, types.ArchiveListFilter{Limit: 2, Offset: 2})
		if err != nil {
			t.Fatalf("list page 2: %v", err)
		}
		if len(page2) != 2 {
			t.Errorf("page 2 len = %d, want 2", len(page2))
		}
		// No overlap.
		seen := map[uuid.UUID]bool{}
		for _, a := range got {
			seen[a.SandboxID] = true
		}
		for _, a := range page2 {
			if seen[a.SandboxID] {
				t.Errorf("pagination overlap: %v on both pages", a.SandboxID)
			}
		}
	})

	t.Run("limit clamped to maximum", func(t *testing.T) {
		_, _, err := ar.List(ctx, types.ArchiveListFilter{Limit: maxArchiveListLimit + 50})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		// No result count assertion (we only have 4 rows seeded);
		// the assertion is "no error from a too-large limit."
	})
}

func TestArchiveRepository_Delete_Idempotent(t *testing.T) {
	ar, sr, _ := newTestArchiveRepo(t)
	s := seedSandbox(t, sr, types.StatusApproved)
	a := makeArchive(s.ID)
	if err := ar.Insert(context.Background(), nil, a); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := ar.Delete(context.Background(), s.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, _ := ar.Get(context.Background(), s.ID)
	if got != nil {
		t.Errorf("expected row gone after delete")
	}
	// Idempotent.
	if err := ar.Delete(context.Background(), s.ID); err != nil {
		t.Errorf("idempotent delete: %v", err)
	}
	// Missing id rejected.
	if err := ar.Delete(context.Background(), uuid.Nil); err == nil {
		t.Errorf("delete(nil id) should error")
	}
}

func TestArchiveRepository_SumSizeBytes(t *testing.T) {
	ar, sr, _ := newTestArchiveRepo(t)
	ctx := context.Background()

	for _, p := range []struct {
		proj  string
		bytes int64
	}{
		{"/a", 100},
		{"/a", 200},
		{"/b", 50},
	} {
		s := seedSandbox(t, sr, types.StatusApproved)
		a := makeArchive(s.ID)
		a.ProjectRoot = p.proj
		a.TotalBlobBytes = p.bytes
		if err := ar.Insert(ctx, nil, a); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	totalA, err := ar.SumSizeBytes(ctx, "/a")
	if err != nil {
		t.Fatalf("sum a: %v", err)
	}
	if totalA != 300 {
		t.Errorf("/a sum = %d, want 300", totalA)
	}
	totalB, err := ar.SumSizeBytes(ctx, "/b")
	if err != nil {
		t.Fatalf("sum b: %v", err)
	}
	if totalB != 50 {
		t.Errorf("/b sum = %d, want 50", totalB)
	}
	totalAll, err := ar.SumSizeBytes(ctx, "")
	if err != nil {
		t.Fatalf("sum all: %v", err)
	}
	if totalAll != 350 {
		t.Errorf("all sum = %d, want 350", totalAll)
	}
	// Zero rows.
	if _, _, err := newTestArchiveRepo(t); err != nil {
		// just ensures helper compiles
		_ = err
	}
}

func TestArchiveRepository_OldestN(t *testing.T) {
	ar, sr, _ := newTestArchiveRepo(t)
	ctx := context.Background()

	times := []time.Time{
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}
	ids := make([]uuid.UUID, len(times))
	for i, t0 := range times {
		s := seedSandbox(t, sr, types.StatusApproved)
		a := makeArchive(s.ID)
		a.SnapshotAt = t0
		ids[i] = s.ID
		if err := ar.Insert(ctx, nil, a); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	got, err := ar.OldestN(ctx, 2)
	if err != nil {
		t.Fatalf("oldest: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if !got[0].SnapshotAt.Equal(times[0]) || !got[1].SnapshotAt.Equal(times[1]) {
		t.Errorf("oldest order: got [%v, %v], want [%v, %v]",
			got[0].SnapshotAt, got[1].SnapshotAt, times[0], times[1])
	}

	// n=0 returns nil, no error.
	got, err = ar.OldestN(ctx, 0)
	if err != nil || got != nil {
		t.Errorf("OldestN(0) = (%v, %v); want (nil, nil)", got, err)
	}
}

func TestArchiveRepository_FK_Cascade_OnSandboxDelete(t *testing.T) {
	ar, sr, db := newTestArchiveRepo(t)
	ctx := context.Background()
	s := seedSandbox(t, sr, types.StatusApproved)

	a := makeArchive(s.ID)
	if err := ar.Insert(ctx, nil, a); err != nil {
		t.Fatalf("insert: %v", err)
	}
	// Bypass repo.Delete to test the FK cascade directly.
	if _, err := db.ExecContext(ctx, "DELETE FROM sandboxes WHERE id = ?", uuidText(s.ID)); err != nil {
		t.Fatalf("delete sandbox: %v", err)
	}
	got, err := ar.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != nil {
		t.Errorf("FK cascade did not remove archive row: %+v", got)
	}
}

func TestArchiveRepository_Insert_RejectsNonexistentSandboxID(t *testing.T) {
	ar, _, _ := newTestArchiveRepo(t)
	a := makeArchive(uuid.New())
	err := ar.Insert(context.Background(), nil, a)
	if err == nil {
		t.Fatal("expected FK error for unseeded sandbox_id")
	}
	// Just ensure it surfaces as an error; SQLite emits a CHECK/FK
	// violation message that we don't depend on textually.
	_ = errors.Unwrap(err)
}
