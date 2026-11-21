package reviewjobs

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

func queueStore(t *testing.T) (*Store, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	store, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	return store, db
}

func TestWorktreeQueueStateTransitionGraphRejectsInvalidEdges(t *testing.T) {
	states := []WorktreeQueueState{WorktreeQueueQueued, WorktreeQueueStaged, WorktreeQueueFailed, WorktreeQueueStale, WorktreeQueueCommitPending, WorktreeQueueCommitted, WorktreeQueueRemoved, WorktreeQueueAwaitingUnstage}
	allowed := map[WorktreeQueueState]map[WorktreeQueueState]bool{
		WorktreeQueueQueued:          {WorktreeQueueStaged: true, WorktreeQueueFailed: true, WorktreeQueueStale: true, WorktreeQueueCommitPending: true, WorktreeQueueRemoved: true, WorktreeQueueAwaitingUnstage: true},
		WorktreeQueueStaged:          {WorktreeQueueQueued: true, WorktreeQueueCommitPending: true, WorktreeQueueRemoved: true},
		WorktreeQueueFailed:          {WorktreeQueueQueued: true, WorktreeQueueRemoved: true},
		WorktreeQueueStale:           {WorktreeQueueQueued: true, WorktreeQueueRemoved: true},
		WorktreeQueueCommitPending:   {WorktreeQueueQueued: true, WorktreeQueueStaged: true, WorktreeQueueCommitted: true},
		WorktreeQueueCommitted:       {},
		WorktreeQueueRemoved:         {},
		WorktreeQueueAwaitingUnstage: {WorktreeQueueQueued: true, WorktreeQueueFailed: true, WorktreeQueueStale: true, WorktreeQueueRemoved: true},
	}
	for _, from := range states {
		if !from.Valid() {
			t.Errorf("declared state %d is invalid", from)
		}
		for _, to := range states {
			if got, want := from.CanTransitionTo(to), allowed[from][to]; got != want {
				t.Errorf("transition %d -> %d allowed=%v, want %v", from, to, got, want)
			}
		}
	}
	for _, invalid := range []WorktreeQueueState{0, 9, -1} {
		if invalid.Valid() || invalid.CanTransitionTo(WorktreeQueueQueued) {
			t.Errorf("invalid state %d was accepted", invalid)
		}
	}
}

func TestCapturedSelectionUsesOrderedQueueWithoutReviewJob(t *testing.T) {
	store, _ := queueStore(t)
	ctx := context.Background()
	entry := WorktreeQueueEntry{ID: "selection-a", RepositoryID: "repo", SnapshotDigest: "sha256:snapshot", Paths: []string{"a.go"}, RequestKey: "selection:selection-a", State: WorktreeQueueAwaitingUnstage, SelectionID: "selection-a", SelectionSource: 2, SelectionPayload: `{"immutable":"bytes"}`}
	queued, rev, err := store.EnqueueCapturedSelection(ctx, entry)
	if err != nil {
		t.Fatal(err)
	}
	if queued.JobID != "" || queued.State != WorktreeQueueAwaitingUnstage || rev != 1 {
		t.Fatalf("selection was not queue-only awaiting unstage: %#v rev=%d", queued, rev)
	}
	listed, err := store.ListWorktreeQueue(ctx, "repo")
	if err != nil || len(listed.Entries) != 1 {
		t.Fatalf("list captured selection: %#v %v", listed, err)
	}
	got := listed.Entries[0]
	if got.Sequence != queued.Sequence || got.SelectionID != entry.SelectionID || got.SelectionSource != entry.SelectionSource || got.SelectionPayload != entry.SelectionPayload || got.JobID != "" {
		t.Fatalf("queue list lost immutable selection: %#v", got)
	}
	finalized, err := store.FinalizeCapturedSelection(ctx, "repo", entry.ID, entry.SnapshotDigest, entry.SelectionPayload)
	if err != nil {
		t.Fatal(err)
	}
	if len(finalized.Entries) != 1 || finalized.Entries[0].State != WorktreeQueueQueued || finalized.Entries[0].Sequence != got.Sequence {
		t.Fatalf("finalize changed queue position or state: %#v", finalized)
	}
	if _, err = store.FinalizeCapturedSelection(ctx, "repo", entry.ID, "sha256:wrong", entry.SelectionPayload); !errors.Is(err, ErrQueueConflict) {
		t.Fatalf("digest mismatch should conflict, got %v", err)
	}
}

func TestWorktreeQueueStoreRejectsUnknownAndInvalidTransitions(t *testing.T) {
	ctx := context.Background()
	store, _ := queueStore(t)
	entry, revision, err := store.EnqueueWorktreeQueue(ctx, WorktreeQueueEntry{ID: "entry", RepositoryID: "repo", JobID: "job", SnapshotDigest: "digest", Paths: []string{"a.go"}, Mode: 1, RequestKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []WorktreeQueueState{0, 9, -1, WorktreeQueueFailed, WorktreeQueueCommitted} {
		if _, _, err := store.UpdateWorktreeQueueEntry(ctx, "repo", entry.ID, entry.ID, revision, invalid, ""); err == nil {
			t.Errorf("invalid transition to state %d was accepted", invalid)
		}
	}
	queue, err := store.ListWorktreeQueue(ctx, "repo")
	if err != nil || queue.Revision != revision || queue.Entries[0].State != WorktreeQueueQueued {
		t.Fatalf("invalid updates mutated durable queue: %#v, %v", queue, err)
	}
}

func TestWorktreeQueueRejectsAmbiguousPathsAndCorruptRequestReplay(t *testing.T) {
	ctx := context.Background()
	store, db := queueStore(t)
	base := WorktreeQueueEntry{ID: "entry", RepositoryID: "repo", JobID: "job", SnapshotDigest: "digest", Mode: 1, RequestKey: "key"}
	for _, paths := range [][]string{{}, {"a.go", "a.go"}, {"a.go", ""}, {"bad\x00path"}, {"../outside"}, {"/tmp/outside"}, {"a/../outside"}, {string([]byte{0xff})}} {
		candidate := base
		candidate.Paths = paths
		if _, _, err := store.EnqueueWorktreeQueue(ctx, candidate); err == nil {
			t.Errorf("accepted invalid path list %#v", paths)
		}
	}

	base.Paths = []string{"a.go"}
	created, revision, err := store.EnqueueWorktreeQueue(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.ReplaceWorktreeQueueEntry(ctx, base.RepositoryID, created.ID, revision, WorktreeQueueEntry{RepositoryID: base.RepositoryID, JobID: "replacement", SnapshotDigest: "replacement", Paths: []string{"../outside"}, Mode: 1, RequestKey: "replacement-key"}); err == nil {
		t.Fatal("queue replacement accepted a traversal path")
	}
	invalidJob := Job{ID: "invalid-job", RepositoryID: "other-repo", IdempotencyKey: "invalid-job-key", InputDigest: "digest"}
	invalidEntry := WorktreeQueueEntry{ID: "invalid-entry", RepositoryID: "other-repo", JobID: invalidJob.ID, SnapshotDigest: "digest", Paths: []string{"../outside"}, Mode: 1, RequestKey: "invalid-entry-key"}
	if _, _, _, err = store.CreateWorktreeJobAndEnqueue(ctx, invalidJob, invalidEntry); err == nil {
		t.Fatal("transactional queue admission accepted a traversal path")
	}
	var orphanJobs int
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM gct_review_jobs WHERE idempotency_key=?`, invalidJob.IdempotencyKey).Scan(&orphanJobs); err != nil || orphanJobs != 0 {
		t.Fatalf("invalid queue admission left an orphan suggestion job: count=%d err=%v", orphanJobs, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE gct_worktree_suggestion_queue_entries SET paths_json=? WHERE id=?`, `["a\u0000b"]`, base.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.EnqueueWorktreeQueue(ctx, base); err == nil {
		t.Fatal("request replay silently accepted a corrupt NUL-containing path")
	}
	if _, err := store.ListWorktreeQueue(ctx, base.RepositoryID); err == nil {
		t.Fatal("queue list silently returned a corrupt NUL-containing path")
	}
	if _, _, err := store.UpdateWorktreeQueueEntry(ctx, base.RepositoryID, base.ID, base.ID, 1, WorktreeQueueStaged, ""); err == nil {
		t.Fatal("queue mutation silently returned a corrupt NUL-containing path")
	}
	if _, err := db.ExecContext(ctx, `UPDATE gct_worktree_suggestion_queue_entries SET paths_json='not-json' WHERE id=?`, base.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.EnqueueWorktreeQueue(ctx, base); err == nil {
		t.Fatal("request replay silently accepted malformed persisted paths")
	}
}

func TestWorktreeQueueMigrationFoldsMembershipFlagsWithoutLosingRows(t *testing.T) {
	ctx := context.Background()
	store, db := queueStore(t)
	_, err := db.Exec(`CREATE TABLE gct_worktree_suggestion_queue_entries (
		id TEXT PRIMARY KEY, repository_id TEXT NOT NULL, job_id TEXT NOT NULL, snapshot_digest TEXT NOT NULL,
		paths_json TEXT NOT NULL, mode INTEGER NOT NULL, request_key TEXT NOT NULL, sequence INTEGER NOT NULL,
		state INTEGER NOT NULL, failure_reason TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
		active INTEGER NOT NULL DEFAULT 1, disposition INTEGER NOT NULL DEFAULT 0)`)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		id          string
		state       WorktreeQueueState
		sequence    int
		active      int
		disposition int
	}{
		{id: "live", state: WorktreeQueueStaged, sequence: 1, active: 1},
		{id: "committed", state: WorktreeQueueCommitPending, sequence: 2, active: 0, disposition: int(WorktreeQueueDispositionCommitted)},
		{id: "removed", state: WorktreeQueueFailed, sequence: 3, active: 0, disposition: int(WorktreeQueueDispositionRemoved)},
	} {
		_, err = db.Exec(`INSERT INTO gct_worktree_suggestion_queue_entries(id,repository_id,job_id,snapshot_digest,paths_json,mode,request_key,sequence,state,created_at,updated_at,active,disposition) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, row.id, "repo", "job-"+row.id, "digest", `["file.go"]`, 1, "key-"+row.id, row.sequence, row.state, "2026-11-12T00:00:00Z", "2026-11-12T00:00:00Z", row.active, row.disposition)
		if err != nil {
			t.Fatal(err)
		}
	}
	queue, err := store.ListWorktreeQueue(ctx, "repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(queue.Entries) != 1 || queue.Entries[0].ID != "live" || queue.Entries[0].State != WorktreeQueueStaged {
		t.Fatalf("migration changed active membership: %#v", queue.Entries)
	}
	var count int
	if err = db.QueryRow(`SELECT COUNT(*) FROM gct_worktree_suggestion_queue_entries`).Scan(&count); err != nil || count != 3 {
		t.Fatalf("migration lost retained queue history: count=%d err=%v", count, err)
	}
	for id, want := range map[string]WorktreeQueueState{"live": WorktreeQueueStaged, "committed": WorktreeQueueCommitted, "removed": WorktreeQueueRemoved} {
		var got WorktreeQueueState
		if err = db.QueryRow(`SELECT state FROM gct_worktree_suggestion_queue_entries WHERE id=?`, id).Scan(&got); err != nil || got != want {
			t.Errorf("migrated row %q state=%d, want %d (err=%v)", id, got, want, err)
		}
	}
	columns, err := db.Query(`PRAGMA table_info(gct_worktree_suggestion_queue_entries)`)
	if err != nil {
		t.Fatal(err)
	}
	defer columns.Close()
	for columns.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err = columns.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		if name == "active" || name == "disposition" {
			t.Errorf("legacy membership column %q remains after migration", name)
		}
	}
	if err = columns.Err(); err != nil {
		t.Fatal(err)
	}
	if err = store.ensureWorktreeQueue(ctx); err != nil {
		t.Fatalf("second migration pass is not idempotent: %v", err)
	}
}

func TestWorktreeQueueMigrationFailsClosedOnAmbiguousMembership(t *testing.T) {
	ctx := context.Background()
	store, db := queueStore(t)
	_, err := db.Exec(`CREATE TABLE gct_worktree_suggestion_queue_entries (
		id TEXT PRIMARY KEY, repository_id TEXT NOT NULL, job_id TEXT NOT NULL, snapshot_digest TEXT NOT NULL,
		paths_json TEXT NOT NULL, mode INTEGER NOT NULL, request_key TEXT NOT NULL, sequence INTEGER NOT NULL,
		state INTEGER NOT NULL, failure_reason TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
		active INTEGER NOT NULL DEFAULT 1, disposition INTEGER NOT NULL DEFAULT 0)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO gct_worktree_suggestion_queue_entries(id,repository_id,job_id,snapshot_digest,paths_json,mode,request_key,sequence,state,created_at,updated_at,active,disposition) VALUES('ambiguous','repo','job','digest','[]',1,'key',1,1,'now','now',0,0)`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.ListWorktreeQueue(ctx, "repo"); err == nil {
		t.Fatal("migration accepted ambiguous inactive membership")
	}
	var active, disposition int
	if err = db.QueryRow(`SELECT active,disposition FROM gct_worktree_suggestion_queue_entries WHERE id='ambiguous'`).Scan(&active, &disposition); err != nil || active != 0 || disposition != 0 {
		t.Fatalf("failed migration modified ambiguous retained data: active=%d disposition=%d err=%v", active, disposition, err)
	}
}

func TestWorktreeQueuePersistenceOrderingReplayAndHeadConflicts(t *testing.T) {
	ctx := context.Background()
	store, db := queueStore(t)
	first := WorktreeQueueEntry{ID: "entry-1", RepositoryID: "repo", JobID: "job-1", SnapshotDigest: "sha256:one", Paths: []string{"a.go"}, Mode: 1, RequestKey: "request-1"}
	created, rev, err := store.EnqueueWorktreeQueue(ctx, first)
	if err != nil || rev != 1 || created.Sequence != 1 {
		t.Fatalf("enqueue first: %#v rev=%d err=%v", created, rev, err)
	}
	replay, rev, err := store.EnqueueWorktreeQueue(ctx, WorktreeQueueEntry{ID: "different-id", RepositoryID: "repo", JobID: "job-1", SnapshotDigest: "sha256:one", Paths: []string{"a.go"}, Mode: 1, RequestKey: "request-1"})
	if err != nil || rev != 1 || replay.ID != first.ID {
		t.Fatalf("enqueue replay changed identity: %#v rev=%d err=%v", replay, rev, err)
	}
	second, rev, err := store.EnqueueWorktreeQueue(ctx, WorktreeQueueEntry{ID: "entry-2", RepositoryID: "repo", JobID: "job-2", SnapshotDigest: "sha256:two", Paths: []string{"b.go"}, Mode: 2, RequestKey: "request-2"})
	if err != nil || rev != 2 || second.Sequence != 2 {
		t.Fatalf("enqueue second: %#v rev=%d err=%v", second, rev, err)
	}
	if _, _, err = store.UpdateWorktreeQueueEntry(ctx, "repo", second.ID, second.ID, 2, 2, ""); !errors.Is(err, ErrQueueConflict) {
		t.Fatalf("later entry advanced ahead of head: %v", err)
	}
	q, _, err := store.UpdateWorktreeQueueEntry(ctx, "repo", first.ID, first.ID, 2, 2, "")
	if err != nil || q.Revision != 3 || len(q.Entries) != 2 || q.Entries[0].State != 2 {
		t.Fatalf("head stage update failed: %#v err=%v", q, err)
	}
	q, _, err = store.UpdateWorktreeQueueEntry(ctx, "repo", first.ID, first.ID, 3, 1, "")
	if err != nil || q.Revision != 4 || q.Entries[0].State != 1 {
		t.Fatalf("staged head did not return to queued after unstaging: %#v err=%v", q, err)
	}
	if _, _, err = store.RetireWorktreeQueueHead(ctx, "repo", first.ID, 4, 1); !errors.Is(err, ErrQueueConflict) {
		t.Fatalf("unstaged head was allowed to retire as committed: %v", err)
	}
	q, _, err = store.UpdateWorktreeQueueEntry(ctx, "repo", first.ID, first.ID, 4, 2, "")
	if err != nil || q.Revision != 5 {
		t.Fatalf("restaged head update failed: %#v err=%v", q, err)
	}
	q, _, err = store.UpdateWorktreeQueueEntry(ctx, "repo", first.ID, first.ID, 5, 5, "")
	if err != nil || q.Revision != 6 {
		t.Fatalf("head did not enter explicit commit-pending state: %#v err=%v", q, err)
	}
	q, _, err = store.RetireWorktreeQueueHead(ctx, "repo", first.ID, 6, 1)
	if err != nil || q.Revision != 7 || len(q.Entries) != 1 || q.Entries[0].ID != second.ID {
		t.Fatalf("head retirement failed: %#v err=%v", q, err)
	}
	if _, _, err = store.RetireWorktreeQueueHead(ctx, "repo", first.ID, 7, 2); !errors.Is(err, ErrQueueConflict) {
		t.Fatalf("stale head advanced queue: %v", err)
	}
	reopened, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	q, err = reopened.ListWorktreeQueue(ctx, "repo")
	if err != nil || q.Revision != 7 || len(q.Entries) != 1 || q.Entries[0].ID != second.ID {
		t.Fatalf("queue did not persist: %#v err=%v", q, err)
	}
	if _, _, err = reopened.EnqueueWorktreeQueue(ctx, first); err != nil {
		t.Fatal(err)
	}
	q, err = reopened.ListWorktreeQueue(ctx, "repo")
	if err != nil || len(q.Entries) != 1 || q.Entries[0].ID != second.ID || q.Revision != 7 {
		t.Fatalf("retired request replay re-entered queue: %#v err=%v", q, err)
	}
}

func TestWorktreeQueueReplacementPreservesOrderAndRetriesByPayload(t *testing.T) {
	ctx := context.Background()
	store, _ := queueStore(t)
	_, _, err := store.EnqueueWorktreeQueue(ctx, WorktreeQueueEntry{ID: "head", RepositoryID: "repo", JobID: "old-head", SnapshotDigest: "sha256:old", Paths: []string{"a"}, Mode: 1, RequestKey: "old-key"})
	if err != nil {
		t.Fatal(err)
	}
	later, rev, err := store.EnqueueWorktreeQueue(ctx, WorktreeQueueEntry{ID: "later", RepositoryID: "repo", JobID: "later-job", SnapshotDigest: "sha256:later", Paths: []string{"b"}, Mode: 1, RequestKey: "later-key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.ReplaceWorktreeQueueEntry(ctx, "repo", later.ID, rev, WorktreeQueueEntry{RepositoryID: "repo", JobID: "duplicate-job", SnapshotDigest: "sha256:duplicate", Paths: []string{"a"}, Mode: 1, RequestKey: "duplicate-key"}); !errors.Is(err, ErrQueuePathConflict) {
		t.Fatalf("replacement overlapped another active selection: %v", err)
	}
	replacement := WorktreeQueueEntry{RepositoryID: "repo", JobID: "fresh-job", SnapshotDigest: "sha256:fresh", Paths: []string{"b", "c"}, Mode: 2, RequestKey: "fresh-key"}
	q, entry, err := store.ReplaceWorktreeQueueEntry(ctx, "repo", later.ID, rev, replacement)
	if err != nil || q.Revision != 3 || entry.Sequence != 2 || entry.ID != later.ID {
		t.Fatalf("replace changed ordering: %#v %#v err=%v", q, entry, err)
	}
	q, entry, err = store.ReplaceWorktreeQueueEntry(ctx, "repo", later.ID, rev, replacement)
	if err != nil || q.Revision != 3 || entry.JobID != "fresh-job" {
		t.Fatalf("replacement retry failed: %#v %#v err=%v", q, entry, err)
	}
	if _, _, err = store.ReplaceWorktreeQueueEntry(ctx, "repo", later.ID, rev-1, WorktreeQueueEntry{RepositoryID: "repo", JobID: "other", SnapshotDigest: "sha256:x", Paths: []string{"x"}, Mode: 1, RequestKey: "other-key"}); !errors.Is(err, ErrQueueConflict) {
		t.Fatalf("stale replacement revision accepted: %v", err)
	}
}

func TestWorktreeQueueRejectsPathOwnershipConflictsTransactionally(t *testing.T) {
	ctx := context.Background()
	store, _ := queueStore(t)
	makeAdmission := func(id, digest, key string, paths []string) (Job, WorktreeQueueEntry) {
		job := Job{ID: id, RepositoryID: "repo", IdempotencyKey: key, InputDigest: digest}
		entry := WorktreeQueueEntry{ID: "entry-" + id, RepositoryID: "repo", JobID: id, SnapshotDigest: digest, Paths: paths, Mode: 1, RequestKey: key}
		return job, entry
	}
	firstJob, firstEntry := makeAdmission("job-1", "digest-1", "key-1", []string{"shared.go"})
	if _, _, _, err := store.CreateWorktreeJobAndEnqueue(ctx, firstJob, firstEntry); err != nil {
		t.Fatal(err)
	}
	secondJob, secondEntry := makeAdmission("job-2", "digest-2", "key-2", []string{"other.go", "shared.go"})
	if _, _, _, err := store.CreateWorktreeJobAndEnqueue(ctx, secondJob, secondEntry); !errors.Is(err, ErrQueuePathConflict) {
		t.Fatalf("concurrent path ownership conflict was accepted: %v", err)
	}
	if _, err := store.GetByIdempotency(ctx, secondJob.IdempotencyKey); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("rejected queue admission left a durable job behind: %v", err)
	}
	queue, err := store.ListWorktreeQueue(ctx, "repo")
	if err != nil || queue.Revision != 1 || len(queue.Entries) != 1 || queue.Entries[0].JobID != firstJob.ID {
		t.Fatalf("rejected admission changed the queue: %#v, %v", queue, err)
	}
}

func TestWorktreeQueueCommitPendingProtectsHeadUntilSettlement(t *testing.T) {
	ctx := context.Background()
	store, _ := queueStore(t)
	head, rev, err := store.EnqueueWorktreeQueue(ctx, WorktreeQueueEntry{ID: "head", RepositoryID: "repo", JobID: "job", SnapshotDigest: "digest", Paths: []string{"a.go"}, Mode: 1, RequestKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.UpdateWorktreeQueueEntry(ctx, "repo", head.ID, head.ID, rev, 5, ""); err != nil {
		t.Fatalf("staged head could not be reserved for commit: %v", err)
	}
	if _, _, err = store.ReplaceWorktreeQueueEntry(ctx, "repo", head.ID, rev+1, WorktreeQueueEntry{RepositoryID: "repo", JobID: "replacement", SnapshotDigest: "replacement-digest", Paths: []string{"a.go"}, Mode: 1, RequestKey: "replacement-key"}); !errors.Is(err, ErrQueueConflict) {
		t.Fatalf("commit-pending head was replaced: %v", err)
	}
	if _, _, err = store.RemoveWorktreeQueueEntry(ctx, "repo", head.ID, rev+1); !errors.Is(err, ErrQueueConflict) {
		t.Fatalf("commit-pending head was removed: %v", err)
	}
	if _, _, err = store.UpdateWorktreeQueueEntry(ctx, "repo", head.ID, head.ID, rev+1, 1, ""); err != nil {
		t.Fatalf("failed commit could not return head to queued state: %v", err)
	}
	queue, err := store.ListWorktreeQueue(ctx, "repo")
	if err != nil || queue.Revision != rev+2 || queue.Entries[0].State != 1 {
		t.Fatalf("failed commit settlement was not durable: %#v, %v", queue, err)
	}
	if _, _, err = store.UpdateWorktreeQueueEntry(ctx, "repo", head.ID, head.ID, queue.Revision, 2, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.UpdateWorktreeQueueEntry(ctx, "repo", head.ID, head.ID, queue.Revision+1, 5, ""); err != nil {
		t.Fatal(err)
	}
	settled, retired, err := store.RetireWorktreeQueueHead(ctx, "repo", head.ID, queue.Revision+2, WorktreeQueueDispositionCommitted)
	if err != nil || len(settled.Entries) != 0 || retired.State != WorktreeQueueCommitted {
		t.Fatalf("successful commit did not settle queue head: %#v retired=%#v err=%v", settled, retired, err)
	}
}

func TestWorktreeQueueRemovalRetainsHistoryAndReleasesPaths(t *testing.T) {
	ctx := context.Background()
	store, db := queueStore(t)
	entry := WorktreeQueueEntry{ID: "removed-entry", RepositoryID: "repo", JobID: "removed-job", SnapshotDigest: "digest", Paths: []string{"reusable.go"}, Mode: 1, RequestKey: "removed-key"}
	if _, _, err := store.EnqueueWorktreeQueue(ctx, entry); err != nil {
		t.Fatal(err)
	}
	queue, removed, err := store.RemoveWorktreeQueueEntry(ctx, "repo", entry.ID, 1)
	if err != nil || len(queue.Entries) != 0 || removed.State != WorktreeQueueRemoved {
		t.Fatalf("remove did not transition to terminal state: queue=%#v removed=%#v err=%v", queue, removed, err)
	}
	var state WorktreeQueueState
	if err = db.QueryRow(`SELECT state FROM gct_worktree_suggestion_queue_entries WHERE id=?`, entry.ID).Scan(&state); err != nil || state != WorktreeQueueRemoved {
		t.Fatalf("removed history was not retained with its terminal state: state=%d err=%v", state, err)
	}
	entry.ID = "replacement-entry"
	entry.JobID = "replacement-job"
	entry.RequestKey = "replacement-key"
	if _, _, err = store.EnqueueWorktreeQueue(ctx, entry); err != nil {
		t.Fatalf("removal did not release path ownership: %v", err)
	}
	queue, err = store.ListWorktreeQueue(ctx, "repo")
	if err != nil || len(queue.Entries) != 1 || queue.Entries[0].ID != entry.ID {
		t.Fatalf("replacement path was not admitted: queue=%#v err=%v", queue, err)
	}
}

func TestWorktreeQueueMutationsReportMissingEntryAsConflict(t *testing.T) {
	ctx := context.Background()
	store, _ := queueStore(t)
	entry := WorktreeQueueEntry{ID: "present-entry", RepositoryID: "repo", JobID: "job", SnapshotDigest: "digest", Paths: []string{"file.go"}, Mode: 1, RequestKey: "key"}
	if _, _, err := store.EnqueueWorktreeQueue(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.RemoveWorktreeQueueEntry(ctx, "repo", "missing-entry", 1); !errors.Is(err, ErrQueueConflict) {
		t.Fatalf("missing queue entry escaped as a storage error: %v", err)
	}
	if _, _, err := store.UpdateWorktreeQueueEntry(ctx, "repo", "missing-entry", "missing-entry", 1, WorktreeQueueStaged, ""); !errors.Is(err, ErrQueueConflict) {
		t.Fatalf("missing queue entry update escaped as a storage error: %v", err)
	}
}

func TestWorktreeQueuePersistsFailureReasonChangesWithRevision(t *testing.T) {
	ctx := context.Background()
	store, _ := queueStore(t)
	entry, rev, err := store.EnqueueWorktreeQueue(ctx, WorktreeQueueEntry{ID: "head", RepositoryID: "repo", JobID: "job", SnapshotDigest: "digest", Paths: []string{"a.go"}, Mode: 1, RequestKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.UpdateWorktreeQueueEntry(ctx, "repo", entry.ID, entry.ID, rev, 4, "first stale reason"); err != nil {
		t.Fatal(err)
	}
	queue, _, err := store.UpdateWorktreeQueueEntry(ctx, "repo", entry.ID, entry.ID, rev+1, 4, "refreshed stale reason")
	if err != nil || queue.Revision != rev+2 || queue.Entries[0].FailureReason != "refreshed stale reason" {
		t.Fatalf("stale reason change was not durably versioned: %#v, %v", queue, err)
	}
}
