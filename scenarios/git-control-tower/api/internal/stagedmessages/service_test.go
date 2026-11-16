// [REQ:GCT-OT-P1-002]
package stagedmessages

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"

	"git-control-tower/internal/reviewjobs"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

const legacyProviderInputBytes = 196608

func TestNewIDReturnsEntropyFailureWithoutPanicking(t *testing.T) {
	id, err := newID("message-", strings.NewReader("short"))
	if err == nil || id != "" {
		t.Fatalf("short entropy read returned an identifier: %q, %v", id, err)
	}
	if !strings.Contains(err.Error(), "generate message- identifier") {
		t.Fatalf("entropy failure lost its operation context: %v", err)
	}
}

func TestQueuedSelectionCanReplaceOrphanedSuggestionJob(t *testing.T) {
	if err := validateQueuedJobReplacement(reviewjobs.Job{}, sql.ErrNoRows); err != nil {
		t.Fatalf("orphaned job blocked recovery from immutable selection: %v", err)
	}
	if err := validateQueuedJobReplacement(reviewjobs.Job{State: reviewjobs.Running}, nil); !errors.Is(err, ErrConflict) {
		t.Fatalf("active job was allowed to be replaced: %v", err)
	}
	if err := validateQueuedJobReplacement(reviewjobs.Job{}, errors.New("database unavailable")); err == nil || err.Error() != "database unavailable" {
		t.Fatalf("storage failure was treated as an orphan: %v", err)
	}
}

func TestCollectUsesInjectedIdentityAndClock(t *testing.T) {
	s, _, _, _ := serviceFixture(t)
	fixedTime := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.FixedZone("fixture", -5*60*60))
	s.generateID = func(prefix string) (string, error) { return prefix + "fixed", nil }
	s.now = func() time.Time { return fixedTime }

	collection, err := s.Collect(context.Background(), "repo", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if collection.GetId() != "collection-fixed" || collection.GetCollectedAt() != fixedTime.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("collection ignored its injected identity or clock: %+v", collection)
	}
}

func TestQueueSelectionCapturesWithoutStartingAndFinalizesExactStagedContent(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	dir, git := fixtureGit(t)
	s.snapshots = GitSnapshotter{}
	writeFixture(t, dir, "file.txt", "captured staged content\n")
	git("", "add", "file.txt")
	for i := 0; i < 80; i++ {
		path := fmt.Sprintf("unrelated-%03d.bin", i)
		if err := os.WriteFile(filepath.Join(dir, path), bytes.Repeat([]byte{'x'}, 128<<10), 0o600); err != nil {
			t.Fatal(err)
		}
		git("", "add", "--", path)
	}
	queued, err := s.QueueWorktreeSelection(context.Background(), "repo", dir, &v1.QueueWorktreeSelectionRequest{SelectionId: "selection-1", Source: v1.WorktreeSelectionSource_WORKTREE_SELECTION_SOURCE_HEAD_TO_INDEX, Paths: []string{"file.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	if queued.State != v1.QueuedWorktreeSelectionState_QUEUED_WORKTREE_SELECTION_STATE_AWAITING_UNSTAGE || provider.starts != 0 || provider.searches != 0 {
		t.Fatalf("admission dispatched work or lost awaiting state: response=%+v starts=%d searches=%d", queued, provider.starts, provider.searches)
	}
	queue, err := s.ListWorktreeQueue(context.Background(), "repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(queue.Entries) != 1 || queue.Entries[0].JobId != "" || queue.Entries[0].SelectionId != "selection-1" || queue.Entries[0].SelectionSource != v1.WorktreeSelectionSource_WORKTREE_SELECTION_SOURCE_HEAD_TO_INDEX {
		t.Fatalf("queue list did not expose captured selection: %+v", queue)
	}
	git("", "reset", "HEAD", "--", "file.txt")
	finalized, err := s.FinalizeQueuedSelection(context.Background(), "repo", dir, &v1.FinalizeQueuedSelectionRequest{SelectionId: "selection-1", ExpectedSnapshotDigest: queued.SnapshotDigest})
	if err != nil {
		t.Fatal(err)
	}
	if finalized.State != v1.QueuedWorktreeSelectionState_QUEUED_WORKTREE_SELECTION_STATE_QUEUED {
		t.Fatalf("finalized state = %v", finalized.State)
	}
	writeFixture(t, dir, "file.txt", "newer different content\n")
	if _, err = s.StartQueuedSelection(context.Background(), "repo", dir, &v1.StartQueuedSelectionRequest{SelectionId: "selection-1", SnapshotDigest: queued.SnapshotDigest, RequestKey: "start-selection-1", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK}); !errors.Is(err, ErrConflict) {
		t.Fatalf("start accepted content changed since queue capture: %v", err)
	}
	writeFixture(t, dir, "file.txt", "captured staged content\n")
	git("", "add", "file.txt")
	storedQueue, err := store.ListWorktreeQueue(context.Background(), "repo")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.UpdateWorktreeQueueEntry(context.Background(), "repo", "selection-1", "selection-1", storedQueue.Revision, reviewjobs.WorktreeQueueStaged, ""); err != nil {
		t.Fatal(err)
	}
	job, err := s.StartQueuedSelection(context.Background(), "repo", dir, &v1.StartQueuedSelectionRequest{SelectionId: "selection-1", SnapshotDigest: queued.SnapshotDigest, RequestKey: "start-selection-1", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK})
	if err != nil {
		t.Fatal(err)
	}
	if job.GetId() == "" || provider.starts != 1 {
		t.Fatalf("explicit start did not bind a job: job=%+v starts=%d", job, provider.starts)
	}
	replayed, err := s.StartQueuedSelection(context.Background(), "repo", dir, &v1.StartQueuedSelectionRequest{SelectionId: "selection-1", SnapshotDigest: queued.SnapshotDigest, RequestKey: "start-selection-1", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK})
	if err != nil || replayed.GetId() != job.GetId() {
		t.Fatalf("same request did not reconnect to its job: replay=%+v err=%v", replayed, err)
	}
	if _, err = s.StartQueuedSelection(context.Background(), "repo", dir, &v1.StartQueuedSelectionRequest{SelectionId: "selection-1", SnapshotDigest: queued.SnapshotDigest, RequestKey: "start-selection-2", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP}); !errors.Is(err, ErrConflict) {
		t.Fatalf("second request replaced an already-owned suggestion job: %v", err)
	}
	queue, err = s.ListWorktreeQueue(context.Background(), "repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(queue.Entries) != 1 || queue.Entries[0].JobId != job.GetId() || queue.Entries[0].Mode != v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK {
		t.Fatalf("rejected second start changed queue ownership: %+v", queue)
	}
	if err = store.Transition(context.Background(), job.GetId(), []reviewjobs.State{reviewjobs.Queued, reviewjobs.Running}, reviewjobs.Failed, "", "fixture failure"); err != nil {
		t.Fatal(err)
	}
	retry, err := s.StartQueuedSelection(context.Background(), "repo", dir, &v1.StartQueuedSelectionRequest{SelectionId: "selection-1", SnapshotDigest: queued.SnapshotDigest, RequestKey: "start-selection-3", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK})
	if err != nil || retry.GetId() == job.GetId() {
		t.Fatalf("explicit retry did not replace a terminal failed job: retry=%+v err=%v", retry, err)
	}
}

func TestFailedQueueEntryCanRetryWhenItsSuggestionJobWasDeleted(t *testing.T) {
	s, store, _, provider, db := serviceFixtureWithDB(t)
	dir, git := fixtureGit(t)
	s.snapshots = GitSnapshotter{}
	writeFixture(t, dir, "file.txt", "captured staged content\n")
	git("", "add", "file.txt")
	queued, err := s.QueueWorktreeSelection(context.Background(), "repo", dir, &v1.QueueWorktreeSelectionRequest{
		SelectionId: "orphaned-selection", Source: v1.WorktreeSelectionSource_WORKTREE_SELECTION_SOURCE_HEAD_TO_INDEX, Paths: []string{"file.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	git("", "reset", "HEAD", "--", "file.txt")
	if _, err = s.FinalizeQueuedSelection(context.Background(), "repo", dir, &v1.FinalizeQueuedSelectionRequest{SelectionId: "orphaned-selection", ExpectedSnapshotDigest: queued.SnapshotDigest}); err != nil {
		t.Fatal(err)
	}
	git("", "add", "file.txt")
	first, err := s.StartQueuedSelection(context.Background(), "repo", dir, &v1.StartQueuedSelectionRequest{
		SelectionId: "orphaned-selection", SnapshotDigest: queued.SnapshotDigest, RequestKey: "initial-orphaned-attempt", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK,
	})
	if err != nil {
		t.Fatal(err)
	}
	if provider.starts != 1 {
		t.Fatalf("initial suggestion starts=%d, want 1", provider.starts)
	}
	queue, err := store.ListWorktreeQueue(context.Background(), "repo")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.UpdateWorktreeQueueEntry(context.Background(), "repo", "orphaned-selection", "orphaned-selection", queue.Revision, reviewjobs.WorktreeQueueFailed, "job lookup returned not_found"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.StartQueuedSelection(context.Background(), "repo", dir, &v1.StartQueuedSelectionRequest{
		SelectionId: "orphaned-selection", SnapshotDigest: queued.SnapshotDigest, RequestKey: "must-not-replace-active-job", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK,
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("failed queue state allowed replacement of a still-active job: %v", err)
	}
	// Model retention/repair deleting the referenced job while its failed
	// queue entry and immutable selection payload remain durable.
	if _, err = db.ExecContext(context.Background(), `DELETE FROM gct_review_jobs WHERE id=?`, first.GetId()); err != nil {
		t.Fatal(err)
	}
	retry, err := s.StartQueuedSelection(context.Background(), "repo", dir, &v1.StartQueuedSelectionRequest{
		SelectionId: "orphaned-selection", SnapshotDigest: queued.SnapshotDigest, RequestKey: "fresh-orphaned-retry", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK,
	})
	if err != nil {
		t.Fatalf("fresh-key retry from failed queue entry: %v", err)
	}
	if retry.GetId() == "" || retry.GetId() == first.GetId() || provider.starts != 2 {
		t.Fatalf("retry did not create a replacement job: first=%q retry=%q starts=%d", first.GetId(), retry.GetId(), provider.starts)
	}
	queue, err = store.ListWorktreeQueue(context.Background(), "repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(queue.Entries) != 1 || queue.Entries[0].JobID != retry.GetId() || queue.Entries[0].State != reviewjobs.WorktreeQueueQueued || queue.Entries[0].RequestKey != "fresh-orphaned-retry" {
		t.Fatalf("retry did not restore queue ownership to the new job: %+v", queue)
	}
}

type fakeSnapshot struct {
	mu       sync.Mutex
	identity string
	patch    string
}

func (f *fakeSnapshot) Identity(context.Context, string, string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.identity, nil
}

func (f *fakeSnapshot) Capture(ctx context.Context, repo, dir string) (Captured, error) {
	identity, _ := f.Identity(ctx, repo, dir)
	return Captured{Snapshot: &v1.StagedMessageSnapshot{Digest: identity, RepositoryId: repo, Files: []*v1.StagedMessageFile{{Path: "file", EditCount: 1, Patch: f.patch}}}}, nil
}
func (f *fakeSnapshot) drift() { f.mu.Lock(); f.identity = "sha256:new"; f.mu.Unlock() }

type fakeProvider struct {
	contexts                                          []*v1.StagedMessageContext
	omissions                                         []*v1.Omission
	searchErr                                         error
	lastInput                                         DraftInput
	mu                                                sync.Mutex
	starts, searches, gets, cancels, cancelByKeyCalls int
	status                                            string
	phase                                             string
	error                                             string
	errorCode                                         string
	retryable                                         bool
	executionStarted                                  bool
	executionStartedKnown                             bool
	tokensUsed                                        int64
	tokensKnown                                       bool
	chargeMicroUSD                                    int64
	chargeKnown                                       bool
	readiness                                         ProviderReadiness
	getErr                                            error
	cancelByKeyErr                                    error
	cancelByKeyResult                                 DispatchCancellation
	startEntered                                      chan struct{}
	startRelease                                      chan struct{}
	startErrors                                       map[int]error
	startKeys                                         []string
	cancelStatus                                      string
	result                                            *v1.StagedMessageSuggestion
	entered, release                                  chan struct{}
}

func (f *fakeProvider) Search(context.Context, *v1.StagedMessageSnapshot) ([]*v1.StagedMessageContext, []*v1.Omission, error) {
	f.mu.Lock()
	f.searches++
	f.mu.Unlock()
	return f.contexts, f.omissions, f.searchErr
}

func (f *fakeProvider) Readiness(context.Context) ProviderReadiness { return f.readiness }

func (f *fakeProvider) Start(_ context.Context, input DraftInput, key string) (Execution, error) {
	f.mu.Lock()
	f.starts++
	call := f.starts
	f.startKeys = append(f.startKeys, key)
	f.lastInput = input
	execution := Execution{ID: "owner-execution", Status: f.status, Phase: f.phase, Result: f.result, Error: f.error, ErrorCode: f.errorCode, Retryable: f.retryable, ExecutionStarted: f.executionStarted, ExecutionStartedKnown: f.executionStartedKnown, TokensUsed: f.tokensUsed, TokensKnown: f.tokensKnown, ChargeMicroUSD: f.chargeMicroUSD, ChargeKnown: f.chargeKnown}
	entered, release := f.startEntered, f.startRelease
	var startErr error
	if f.startErrors != nil {
		startErr = f.startErrors[call]
	}
	f.mu.Unlock()
	if entered != nil && call == 1 {
		close(entered)
	}
	if release != nil && call == 1 {
		<-release
	}
	if startErr != nil {
		return Execution{}, startErr
	}
	return execution, nil
}

func (f *fakeProvider) Get(context.Context, string, DraftInput) (Execution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gets++
	if f.getErr != nil {
		return Execution{}, f.getErr
	}
	return Execution{ID: "owner-execution", Status: f.status, Phase: f.phase, Result: f.result, Error: f.error, ErrorCode: f.errorCode, Retryable: f.retryable, ExecutionStarted: f.executionStarted, ExecutionStartedKnown: f.executionStartedKnown, TokensUsed: f.tokensUsed, TokensKnown: f.tokensKnown, ChargeMicroUSD: f.chargeMicroUSD, ChargeKnown: f.chargeKnown}, nil
}

func (f *fakeProvider) CancelByKey(context.Context, string, DraftInput) (DispatchCancellation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancelByKeyCalls++
	return f.cancelByKeyResult, f.cancelByKeyErr
}

func (f *fakeProvider) Wait(ctx context.Context, id string, input DraftInput, _ time.Duration) (Execution, error) {
	if f.entered != nil {
		close(f.entered)
		select {
		case <-f.release:
		case <-ctx.Done():
			return Execution{}, ctx.Err()
		}
	}
	return f.Get(ctx, id, input)
}

func (f *fakeProvider) Cancel(context.Context, string, DraftInput) (Execution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancels++
	status := f.cancelStatus
	if status == "" {
		status = f.status
	}
	return Execution{ID: "owner-execution", Status: status, Result: f.result}, nil
}

func (f *fakeProvider) complete(status string, result *v1.StagedMessageSuggestion) {
	f.mu.Lock()
	f.status = status
	f.result = result
	f.mu.Unlock()
}

func serviceFixture(t *testing.T) (*Service, *reviewjobs.Store, *fakeSnapshot, *fakeProvider) {
	s, store, snap, provider, _ := serviceFixtureWithDB(t)
	return s, store, snap, provider
}

func TestDraftInputFormatsAdaptWithoutRewritingDurableVersion(t *testing.T) {
	for _, tc := range []struct {
		name         string
		version      int
		deepEvidence bool
		synthesis    bool
		legacyDraft  bool
	}{
		{name: "oldest admitted format", version: 2, legacyDraft: true},
		{name: "deep evidence compatibility format", version: 3, deepEvidence: true},
		{name: "canonical current format", version: 4, deepEvidence: true, synthesis: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := DraftInput{FormatVersion: tc.version, OperatorNote: "preserve intent"}
			raw, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			durable, err := decodeDraftInput(raw)
			if err != nil {
				t.Fatal(err)
			}
			semantics := durable.semantics()
			if semantics.deepEvidence != tc.deepEvidence || semantics.synthesis != tc.synthesis || semantics.legacyDraft != tc.legacyDraft {
				t.Fatalf("format %d adapted to %+v", tc.version, semantics)
			}
			if durable.FormatVersion != tc.version || durable.OperatorNote != input.OperatorNote {
				t.Fatalf("adapter rewrote durable input: %+v", durable)
			}
		})
	}
}

func TestDecodeDraftInputRejectsUndocumentedFormats(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "missing version", raw: `{"snapshot":{},"contexts":[],"operator_note":""}`},
		{name: "zero version", raw: `{"format_version":0}`},
		{name: "negative version", raw: `{"format_version":-1}`},
		{name: "older undocumented version", raw: `{"format_version":1}`},
		{name: "future version", raw: `{"format_version":5}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := decodeDraftInput([]byte(tc.raw)); err == nil {
				t.Fatalf("accepted unsupported durable input: %s", tc.raw)
			}
		})
	}
}

func createQueuedDraft(t *testing.T, store *reviewjobs.Store, key string) reviewjobs.Job {
	t.Helper()
	snapshot := &v1.StagedMessageSnapshot{Digest: "sha256:original", RepositoryId: "repo", Files: []*v1.StagedMessageFile{{Path: "file", EditCount: 1}}}
	raw, err := json.Marshal(DraftInput{FormatVersion: 2, Snapshot: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	job, _, err := store.Create(t.Context(), reviewjobs.Job{ID: "message-" + key, IdempotencyKey: key, InputDigest: digest(raw), InputJSON: string(raw), Operation: operation, RepositoryID: "repo", State: reviewjobs.Queued})
	if err != nil {
		t.Fatal(err)
	}
	return job
}

func timePointer(value time.Time) *time.Time { return &value }

func serviceFixtureWithDB(t *testing.T) (*Service, *reviewjobs.Store, *fakeSnapshot, *fakeProvider, *sql.DB) {
	t.Helper()
	db, e := sql.Open("sqlite", ":memory:")
	if e != nil {
		t.Fatal(e)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	store, e := reviewjobs.New(db)
	if e != nil {
		t.Fatal(e)
	}
	snap := &fakeSnapshot{identity: "sha256:original"}
	provider := &fakeProvider{status: "running"}
	return New(store, snap, nil, provider), store, snap, provider, db
}

func TestJobGetIsObservationalAndAdvanceDispatches(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	snapshot := &v1.StagedMessageSnapshot{Digest: "sha256:original", RepositoryId: "repo", Files: []*v1.StagedMessageFile{{Path: "file", EditCount: 1}}}
	input, err := json.Marshal(DraftInput{FormatVersion: 2, Snapshot: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	job, _, err := store.Create(t.Context(), reviewjobs.Job{ID: "message-observe", IdempotencyKey: "observe", InputDigest: digest(input), InputJSON: string(input), Operation: operation, RepositoryID: "repo", State: reviewjobs.Queued})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Job(t.Context(), "repo", "fixture", job.ID, "", "get", 0)
	if err != nil || got.Status != string(reviewjobs.Queued) || provider.starts != 0 {
		t.Fatalf("GET changed execution state: response=%+v starts=%d err=%v", got, provider.starts, err)
	}
	got, err = s.Job(t.Context(), "repo", "fixture", job.ID, "", "advance", 0)
	if err != nil || got.Status != string(reviewjobs.Running) || provider.starts != 1 {
		t.Fatalf("explicit advance did not dispatch once: response=%+v starts=%d err=%v", got, provider.starts, err)
	}
}

func TestJobStatusMarksStaleOnlyOnOwnerNotFound(t *testing.T) {
	for _, tc := range []struct {
		name  string
		err   error
		state reviewjobs.State
		code  string
	}{
		{name: "owner confirms missing execution", err: connect.NewError(connect.CodeNotFound, errors.New("missing")), state: reviewjobs.Stale, code: "execution_missing"},
		{name: "transport outage preserves last confirmed state", err: connect.NewError(connect.CodeUnavailable, errors.New("offline")), state: reviewjobs.Running, code: "execution_status_unavailable"},
		{name: "authentication required to read saved execution", err: connect.NewError(connect.CodeUnauthenticated, errors.New("credentials unavailable")), state: reviewjobs.Running, code: "authentication_required"},
		{name: "policy denies reading saved execution", err: connect.NewError(connect.CodePermissionDenied, errors.New("policy detail")), state: reviewjobs.Running, code: "policy_denied"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, store, _, provider := serviceFixture(t)
			provider.getErr = tc.err
			snapshot := &v1.StagedMessageSnapshot{Digest: "sha256:original", RepositoryId: "repo", Files: []*v1.StagedMessageFile{{Path: "file", EditCount: 1}}}
			raw, err := json.Marshal(DraftInput{FormatVersion: 2, Snapshot: snapshot})
			if err != nil {
				t.Fatal(err)
			}
			job, _, err := store.Create(t.Context(), reviewjobs.Job{ID: "status-evidence", IdempotencyKey: "status-evidence", InputDigest: digest(raw), InputJSON: string(raw), Operation: operation, RepositoryID: "repo", ExecutionID: "owner-execution", State: reviewjobs.Running})
			if err != nil {
				t.Fatal(err)
			}
			got, err := s.Job(t.Context(), "repo", "fixture", job.ID, "", "advance", 0)
			if err != nil || got.Status != string(tc.state) || got.ErrorCode != tc.code {
				t.Fatalf("status = %+v, err=%v; want state=%q code=%q", got, err, tc.state, tc.code)
			}
			if tc.code == "authentication_required" || tc.code == "policy_denied" {
				if !got.Retryable || got.ExecutionStartedKnown || got.TokensKnown || got.ChargeKnown || strings.Contains(got.Error, "credentials unavailable") || strings.Contains(got.Error, "policy detail") {
					t.Fatalf("authorization failure lost recovery guidance, leaked private detail, or invented execution certainty: %+v", got)
				}
				stored, readErr := store.Get(t.Context(), job.ID)
				if readErr != nil || stored.ErrorCode != tc.code || stored.State != reviewjobs.Running {
					t.Fatalf("authorization failure was not durably classified without terminating the saved run: %+v err=%v", stored, readErr)
				}
			}
		})
	}
}

func TestClassifyDispatchErrorsByRecoveryAction(t *testing.T) {
	for _, tc := range []struct {
		name      string
		code      connect.Code
		first     bool
		err       error
		wantCode  string
		wantRetry bool
	}{
		{name: "invalid request", code: connect.CodeInvalidArgument, first: true, wantCode: "invalid_request"},
		{name: "authentication", code: connect.CodeUnauthenticated, first: true, wantCode: "authentication_required", wantRetry: true},
		{name: "policy denial", code: connect.CodePermissionDenied, first: true, wantCode: "policy_denied"},
		{name: "provider precondition", code: connect.CodeFailedPrecondition, first: true, wantCode: "provider_precondition_failed", wantRetry: true},
		{name: "unsupported operation", code: connect.CodeUnimplemented, first: true, wantCode: "unsupported_operation"},
		{name: "unavailable after attempt", code: connect.CodeUnavailable, first: true, wantCode: "dispatch_unconfirmed"},
		{name: "status unavailable after acceptance", code: connect.CodeUnavailable, first: false, wantCode: "execution_status_unavailable", wantRetry: true},
		{name: "authentication lost after acceptance", code: connect.CodeUnauthenticated, first: false, wantCode: "authentication_required", wantRetry: true},
		{name: "policy denies status after acceptance", code: connect.CodePermissionDenied, first: false, wantCode: "policy_denied", wantRetry: true},
		{name: "evidence unavailable after execution", code: connect.CodeUnavailable, first: false, err: EvidenceReadError{Cause: errors.New("private evidence detail")}, wantCode: "evidence_unavailable", wantRetry: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.err
			if err == nil {
				err = connect.NewError(tc.code, errors.New("private transport detail"))
			}
			code, message, retryable := classifyDispatchError(err, tc.first)
			if code != tc.wantCode || retryable != tc.wantRetry || message == "" || strings.Contains(message, "private transport detail") {
				t.Fatalf("classification = (%q, %q, %t), want code=%q retryable=%t and safe explanation", code, message, retryable, tc.wantCode, tc.wantRetry)
			}
		})
	}
}

func TestBudgetFailurePersistsOwnerWorkAndChargeReceipts(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	provider.status = "failed"
	provider.error = "Drafting stopped because the workflow token budget was exhausted."
	provider.errorCode = "budget_exhausted"
	provider.retryable = false
	provider.executionStarted, provider.executionStartedKnown = true, true
	provider.tokensUsed, provider.tokensKnown = 1200, true
	provider.chargeMicroUSD, provider.chargeKnown = 25000, true

	_, got := admission(t, s, "budget-exhausted")
	stored, err := store.Get(t.Context(), got.Id)
	if err != nil || got.Status != string(reviewjobs.Failed) || got.ErrorCode != "budget_exhausted" || got.Retryable || !got.ExecutionStartedKnown || !got.ExecutionStarted || !got.TokensKnown || got.TokensUsed != 1200 || !got.ChargeKnown || got.ChargeMicroUsd != 25000 {
		t.Fatalf("projected budget receipt = %+v err=%v", got, err)
	}
	if stored.State != reviewjobs.Failed || stored.ErrorCode != "budget_exhausted" || stored.Retryable || !stored.ExecutionStartedKnown || !stored.ExecutionStarted || !stored.TokensKnown || stored.TokensUsed != 1200 || !stored.ChargeKnown || stored.ChargeMicroUSD != 25000 {
		t.Fatalf("persisted budget receipt = %+v", stored)
	}
}

func TestProviderUnavailableBeforeDispatchPersistsKnownNoWork(t *testing.T) {
	s, store, _, _ := serviceFixture(t)
	collection, err := s.Collect(t.Context(), "repo", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	input := DraftInput{FormatVersion: 2, Snapshot: collection.Snapshot, Contexts: collection.Contexts}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	job, _, err := store.Create(t.Context(), reviewjobs.Job{
		ID: "provider-unavailable", IdempotencyKey: key("repo", "provider-unavailable"),
		InputDigest: digest(raw), InputJSON: string(raw), Operation: operation, RepositoryID: "repo", State: reviewjobs.Queued,
	})
	if err != nil {
		t.Fatal(err)
	}
	s.provider = nil

	got, err := s.Job(t.Context(), "repo", "fixture", job.ID, "", "advance", 0)
	stored, readErr := store.Get(t.Context(), job.ID)
	if err != nil || readErr != nil || got.Status != string(reviewjobs.Failed) || got.ErrorCode != "provider_unavailable" || !got.Retryable || !got.ExecutionStartedKnown || got.ExecutionStarted || !got.TokensKnown || got.TokensUsed != 0 || !got.ChargeKnown || got.ChargeMicroUsd != 0 {
		t.Fatalf("provider-unavailable projection = %+v err=%v readErr=%v", got, err, readErr)
	}
	if stored.State != reviewjobs.Failed || stored.ErrorCode != "provider_unavailable" || !stored.Retryable || !stored.ExecutionStartedKnown || stored.ExecutionStarted || !stored.TokensKnown || stored.TokensUsed != 0 || !stored.ChargeKnown || stored.ChargeMicroUSD != 0 {
		t.Fatalf("provider-unavailable receipt = %+v", stored)
	}
}

func TestEvidenceReadFailureKeepsJobRecoverableAndClearsOnReconnect(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	_, admitted := admission(t, s, "evidence-read-recovery")
	provider.getErr = EvidenceReadError{Cause: errors.New("private evidence-store detail")}
	uncertain, err := s.Job(t.Context(), "repo", "fixture", admitted.Id, "", "advance", 0)
	if err != nil || uncertain.Status != string(reviewjobs.Running) || uncertain.Id != admitted.Id || uncertain.ExecutionId != admitted.ExecutionId || uncertain.ErrorCode != "evidence_unavailable" || !uncertain.Retryable {
		t.Fatalf("evidence read failure lost its recoverable owner identity: job=%+v err=%v", uncertain, err)
	}
	if !strings.Contains(uncertain.Error, "reconnect to retrieve the same result") || strings.Contains(uncertain.Error, "private evidence-store detail") || uncertain.ExecutionStartedKnown || uncertain.TokensKnown || uncertain.ChargeKnown {
		t.Fatalf("evidence failure explanation or certainty is not truthful: %+v", uncertain)
	}
	stored, err := store.Get(t.Context(), admitted.Id)
	if err != nil || stored.State != reviewjobs.Running || stored.ErrorCode != "evidence_unavailable" || stored.ExecutionID != admitted.ExecutionId {
		t.Fatalf("evidence failure was not durably classified: stored=%+v err=%v", stored, err)
	}

	provider.getErr = nil
	provider.status = "succeeded"
	provider.result = validResult()
	recovered, err := s.Job(t.Context(), "repo", "fixture", admitted.Id, "", "advance", 0)
	if err != nil || recovered.Status != string(reviewjobs.Succeeded) || recovered.Id != admitted.Id || recovered.Error != "" || recovered.ErrorCode != "" || recovered.Result == nil {
		t.Fatalf("reconnect did not recover and clear the transient evidence failure: job=%+v err=%v", recovered, err)
	}
}

type evidenceUnavailableAtStartProvider struct{ *fakeProvider }

func (p *evidenceUnavailableAtStartProvider) Start(_ context.Context, input DraftInput, _ string) (Execution, error) {
	p.fakeProvider.mu.Lock()
	p.fakeProvider.starts++
	p.fakeProvider.lastInput = input
	p.fakeProvider.mu.Unlock()
	return Execution{ID: "owner-execution", Status: "succeeded", Phase: "draft", ExecutionStarted: true, ExecutionStartedKnown: true}, EvidenceReadError{Cause: errors.New("private result read detail")}
}

func TestEvidenceReadFailureAtStartPersistsHandleAndWorkReceipt(t *testing.T) {
	s, store, _, baseProvider := serviceFixture(t)
	provider := &evidenceUnavailableAtStartProvider{fakeProvider: baseProvider}
	provider.status = "succeeded"
	provider.result = validResult()
	s.provider = provider

	_, admitted := admission(t, s, "evidence-unavailable-at-start")
	if admitted.Status != string(reviewjobs.Running) || admitted.ExecutionId != "owner-execution" || admitted.ErrorCode != "evidence_unavailable" || !admitted.Retryable || !admitted.ExecutionStartedKnown || !admitted.ExecutionStarted || admitted.TokensKnown || admitted.ChargeKnown {
		t.Fatalf("initial result read failure lost known execution/work evidence: %+v", admitted)
	}
	stored, err := store.Get(t.Context(), admitted.Id)
	if err != nil || stored.State != reviewjobs.Running || stored.ExecutionID != "owner-execution" || stored.ExecutionStartedKnown != true || !stored.ExecutionStarted {
		t.Fatalf("initial result read failure was not durably attached: %+v err=%v", stored, err)
	}

	recovered, err := s.Job(t.Context(), "repo", "fixture", admitted.Id, "", "advance", 0)
	if err != nil || recovered.Status != string(reviewjobs.Succeeded) || recovered.ErrorCode != "" || recovered.Result == nil || provider.starts != 1 || provider.gets != 1 {
		t.Fatalf("same-handle reconnect did not recover without redispatch: job=%+v starts=%d gets=%d err=%v", recovered, provider.starts, provider.gets, err)
	}
}

func TestMaintenanceRefusalPersistsAwaitingProviderAndRequiresExplicitNewJobRetry(t *testing.T) {
	s, _, _, provider := serviceFixture(t)
	provider.status = "failed"
	provider.error = "Agent execution is paused for Workspace Sandbox maintenance; no agent work started."
	provider.errorCode = "provider_admission_maintenance"
	provider.retryable = true
	provider.executionStartedKnown = true
	provider.executionStarted = false
	provider.tokensKnown = true
	provider.chargeKnown = false
	provider.readiness = ProviderReadiness{Provider: "workspace-sandbox", State: "closed", Known: true, ObservedAt: timePointer(time.Now()), Message: "Workspace Sandbox is in lifecycle maintenance."}

	collection, paused := admission(t, s, "maintenance-pause")
	if paused.Status != string(reviewjobs.AwaitingProvider) || paused.ErrorCode != "provider_admission_maintenance" || !paused.Retryable || !paused.ExecutionStartedKnown || paused.ExecutionStarted || !paused.TokensKnown || paused.TokensUsed != 0 || !paused.ChargeKnown || paused.ChargeMicroUsd != 0 {
		t.Fatalf("maintenance refusal did not persist as a known zero-work provider wait: %+v", paused)
	}
	observed, err := s.Job(t.Context(), "repo", "fixture", paused.Id, "", "get", 0)
	if err != nil || observed.Status != string(reviewjobs.AwaitingProvider) || provider.starts != 1 || provider.gets != 0 {
		t.Fatalf("provider wait was not observational: job=%+v starts=%d gets=%d err=%v", observed, provider.starts, provider.gets, err)
	}
	if _, err = s.Retry(t.Context(), "repo", "fixture", &v1.RetryStagedMessageRequest{JobId: paused.Id, SnapshotDigest: paused.SnapshotDigest, RequestKey: "retry-while-closed", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK}); !errors.Is(err, ErrProviderNotReady) || provider.starts != 1 {
		t.Fatalf("closed provider accepted an automatic or explicit retry: starts=%d err=%v", provider.starts, err)
	}

	provider.readiness = ProviderReadiness{Provider: "workspace-sandbox", State: "open", Known: true, ObservedAt: timePointer(time.Now()), Message: "Workspace Sandbox is ready for agent runs."}
	provider.status = "running"
	provider.error, provider.errorCode, provider.retryable = "", "", false
	provider.executionStartedKnown, provider.tokensKnown = false, false
	newJob, err := s.Retry(t.Context(), "repo", "fixture", &v1.RetryStagedMessageRequest{JobId: paused.Id, SnapshotDigest: collection.Snapshot.Digest, RequestKey: "retry-after-ready", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK})
	if err != nil || newJob.Id == paused.Id || newJob.Status != string(reviewjobs.Running) || provider.starts != 2 || provider.lastInput.Snapshot.Digest != collection.Snapshot.Digest {
		t.Fatalf("ready provider did not create a new job from the exact saved selection: old=%+v new=%+v starts=%d err=%v", paused, newJob, provider.starts, err)
	}
}

func TestMaintenanceCodeWithStartedReceiptDoesNotClaimProviderWaitOrZeroCharge(t *testing.T) {
	s, _, _, provider := serviceFixture(t)
	provider.status = "failed"
	provider.error = "maintenance reported after execution started"
	provider.errorCode = "provider_admission_maintenance"
	provider.retryable = true
	provider.executionStartedKnown = true
	provider.executionStarted = true
	provider.tokensKnown = true
	provider.tokensUsed = 40
	provider.chargeKnown = true
	provider.chargeMicroUSD = 725_000

	_, got := admission(t, s, "maintenance-after-start")
	if got.Status != string(reviewjobs.Failed) || !got.ExecutionStartedKnown || !got.ExecutionStarted || !got.TokensKnown || got.TokensUsed != 40 || !got.ChargeKnown || got.ChargeMicroUsd != 725_000 {
		t.Fatalf("contradictory post-start maintenance code was treated as a proven prelaunch refusal: %+v", got)
	}
}

func TestLegacyMaintenanceFailureProjectsAwaitingProviderWithoutMutatingOrCharging(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	snapshot := &v1.StagedMessageSnapshot{Digest: "sha256:original", RepositoryId: "repo", Files: []*v1.StagedMessageFile{{Path: "file", EditCount: 1}}}
	raw, err := json.Marshal(DraftInput{FormatVersion: 2, Snapshot: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	created, _, err := store.Create(t.Context(), reviewjobs.Job{
		ID: "legacy-maintenance", IdempotencyKey: "legacy-maintenance", InputDigest: digest(raw), InputJSON: string(raw),
		Operation: operation, RepositoryID: "repo", ExecutionID: "workflow-run",
		State: reviewjobs.Failed, ErrorCode: "provider_admission_maintenance", Retryable: true,
		ExecutionStartedKnown: true, ExecutionStarted: false, TokensKnown: true, TokensUsed: 0,
		ChargeKnown: false, Error: "Agent execution is paused for Workspace Sandbox maintenance; no agent work started.",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.Job(t.Context(), "repo", "fixture", created.ID, "", "get", 0)
	if err != nil || got.Status != string(reviewjobs.AwaitingProvider) || got.ErrorCode != "provider_admission_maintenance" || !got.ExecutionStartedKnown || got.ExecutionStarted || !got.TokensKnown || got.TokensUsed != 0 || got.ChargeKnown || got.ChargeMicroUsd != 0 {
		t.Fatalf("legacy maintenance proof was not projected as a charge-unknown provider wait: job=%+v err=%v", got, err)
	}
	stored, err := store.Get(t.Context(), created.ID)
	if err != nil || stored.State != reviewjobs.Failed || stored.ChargeKnown || provider.gets != 0 || provider.starts != 0 {
		t.Fatalf("GET mutated the durable legacy row or reconciled with the provider: stored=%+v gets=%d err=%v", stored, provider.gets, err)
	}

	provider.readiness = ProviderReadiness{Provider: "workspace-sandbox", State: "open", Known: true, ObservedAt: timePointer(time.Now()), Message: "Workspace Sandbox is ready for agent runs."}
	retried, err := s.Retry(t.Context(), "repo", "fixture", &v1.RetryStagedMessageRequest{JobId: created.ID, SnapshotDigest: snapshot.Digest, RequestKey: "legacy-explicit-retry", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK})
	if err != nil || retried.Id == created.ID || retried.Status != string(reviewjobs.Running) || provider.starts != 1 || provider.lastInput.Snapshot.Digest != snapshot.Digest {
		t.Fatalf("explicit open-readiness retry did not create a new job from the saved input: previous=%+v retry=%+v starts=%d err=%v", created, retried, provider.starts, err)
	}
	stored, err = store.Get(t.Context(), created.ID)
	if err != nil || stored.State != reviewjobs.Failed || stored.ChargeKnown {
		t.Fatalf("explicit retry rewrote the legacy record or invented charge certainty: stored=%+v err=%v", stored, err)
	}
}

func TestMaintenanceRetryRequiresKnownOpenProviderReadiness(t *testing.T) {
	for _, tc := range []struct {
		name      string
		readiness ProviderReadiness
		wantRetry bool
	}{
		{name: "unknown", readiness: ProviderReadiness{Provider: "workspace-sandbox", State: "unknown", Known: false, Message: "Workspace Sandbox readiness could not be confirmed."}},
		{name: "closed", readiness: ProviderReadiness{Provider: "workspace-sandbox", State: "closed", Known: true, ObservedAt: timePointer(time.Now()), Message: "Workspace Sandbox is in lifecycle maintenance."}},
		{name: "open", readiness: ProviderReadiness{Provider: "workspace-sandbox", State: "open", Known: true, ObservedAt: timePointer(time.Now()), Message: "Workspace Sandbox is ready for agent runs."}, wantRetry: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, store, _, provider := serviceFixture(t)
			provider.readiness = tc.readiness
			snapshot := &v1.StagedMessageSnapshot{Digest: "sha256:original", RepositoryId: "repo", Files: []*v1.StagedMessageFile{{Path: "file", EditCount: 1}}}
			input, err := json.Marshal(DraftInput{FormatVersion: 2, Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK, Snapshot: snapshot})
			if err != nil {
				t.Fatal(err)
			}
			previous, _, err := store.Create(t.Context(), reviewjobs.Job{ID: "maintenance-paused", IdempotencyKey: "previous", InputDigest: digest(input), InputJSON: string(input), Operation: operation, RepositoryID: "repo", State: reviewjobs.AwaitingProvider, ErrorCode: "provider_admission_maintenance"})
			if err != nil {
				t.Fatal(err)
			}
			got, err := s.Retry(t.Context(), "repo", "fixture", &v1.RetryStagedMessageRequest{JobId: previous.ID, SnapshotDigest: snapshot.Digest, RequestKey: "retry-" + tc.name, Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK})
			if tc.wantRetry {
				if err != nil || got.Status != string(reviewjobs.Running) || provider.starts != 1 {
					t.Fatalf("ready provider did not allow one retry: job=%+v starts=%d err=%v", got, provider.starts, err)
				}
				return
			}
			if !errors.Is(err, ErrProviderNotReady) || provider.starts != 0 {
				t.Fatalf("unready provider allowed retry: job=%+v starts=%d err=%v", got, provider.starts, err)
			}
		})
	}
}

func admission(t *testing.T, s *Service, key string) (*v1.StagedMessageCollection, *v1.StagedMessageJob) {
	t.Helper()
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	j, e := s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: key})
	if e != nil {
		t.Fatal(e)
	}
	return c, j
}

func validResult() *v1.StagedMessageSuggestion {
	return &v1.StagedMessageSuggestion{Title: "Update selected behavior", Coverage: &v1.StagedMessageCoverage{CoveredPaths: []string{"file"}, MatchedEdits: 1, TotalEdits: 1}}
}

func TestCollectionNeverStartsAgentWhenNoReusableMessage(t *testing.T) {
	s, _, _, provider := serviceFixture(t)
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	if c.Status != "no_suggestions" || len(c.Suggestions) != 0 || provider.starts != 0 || provider.searches != 1 {
		t.Fatalf("collection %#v starts=%d", c, provider.starts)
	}
}

func TestWorktreeSelectionCreatesIndependentJobDuringStagedJob(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	dir, git := fixtureGit(t)
	stagedCollection, err := s.Collect(context.Background(), "repo", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	staged, err := s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: stagedCollection.Id, SnapshotDigest: stagedCollection.Snapshot.Digest, RequestKey: "staged-running"})
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, dir, "file.txt", "index value\n")
	git("", "add", "file.txt")
	writeFixture(t, dir, "file.txt", "unstaged value\n")
	writeFixture(t, dir, "new.txt", "untracked value\n")
	worktree, err := s.StartWorktree(context.Background(), "repo", dir, &v1.StartWorktreeStagedMessageRequest{Paths: []string{"file.txt", "new.txt"}, RequestKey: "selected-worktree", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP})
	if err != nil {
		t.Fatal(err)
	}
	if worktree.Id == "" || worktree.Id == staged.Id {
		t.Fatalf("selection did not receive an independent durable job: staged=%q worktree=%q", staged.Id, worktree.Id)
	}
	if !worktree.Applicable {
		t.Fatalf("unchanged unstaged selection reported inapplicable: %s", worktree.ApplicabilityReason)
	}
	searches := provider.searches
	retry, err := s.StartWorktree(context.Background(), "repo", dir, &v1.StartWorktreeStagedMessageRequest{Paths: []string{"new.txt", "file.txt"}, RequestKey: "selected-worktree", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP})
	if err != nil || retry.Id != worktree.Id || provider.searches != searches {
		t.Fatalf("lost-ack retry did not reconnect to immutable worktree job: retry=%#v searches=%d/%d err=%v", retry, provider.searches, searches, err)
	}
	queue, err := s.ListWorktreeQueue(context.Background(), "repo")
	if err != nil || len(queue.Entries) != 1 || queue.Entries[0].JobId != worktree.Id || queue.Entries[0].RequestKey != "selected-worktree" || queue.Entries[0].Sequence != 1 || len(queue.Entries[0].Paths) != 2 {
		t.Fatalf("worktree admission did not atomically link durable queue membership: %#v err=%v", queue, err)
	}
	job, err := store.Get(context.Background(), worktree.Id)
	if err != nil {
		t.Fatal(err)
	}
	var input DraftInput
	if input, err = decodeDraftInput([]byte(job.InputJSON)); err != nil {
		t.Fatal(err)
	}
	if !input.WorktreeCapture || input.Snapshot.Digest == stagedCollection.Snapshot.Digest || string(input.CapturedWorktree["file.txt"].Before) != "index value\n" || string(input.CapturedWorktree["file.txt"].After) != "unstaged value\n" || string(input.CapturedWorktree["new.txt"].After) != "untracked value\n" {
		t.Fatalf("durable worktree job did not bind its selected bytes: %#v", input)
	}
	if provider.starts < 2 {
		t.Fatalf("independent staged and worktree jobs were not both dispatched: %d", provider.starts)
	}
	page, err := s.ReadEvidence(context.Background(), "repo", dir, &v1.ReadStagedMessageEvidenceRequest{JobId: worktree.Id, SnapshotDigest: input.Snapshot.Digest, MaxBytes: 16384}, "")
	if err != nil || len(page.Files) != 2 || !page.Files[0].CompletePatch || !page.Files[1].CompletePatch {
		t.Fatalf("deep evidence did not use its durable captured identity: page=%#v err=%v", page, err)
	}
	git("", "add", "file.txt", "new.txt")
	stagedExact, err := s.Job(context.Background(), "repo", dir, worktree.Id, "", "advance", 0)
	if err != nil || !stagedExact.Applicable {
		t.Fatalf("exactly staged selection lost applicability: result=%#v err=%v", stagedExact, err)
	}
	writeFixture(t, dir, "unrelated.txt", "unrelated commit\n")
	git("", "add", "unrelated.txt")
	git("", "commit", "--only", "-m", "unrelated HEAD movement", "--", "unrelated.txt")
	afterHeadMove, err := s.Job(context.Background(), "repo", dir, worktree.Id, "", "advance", 0)
	if err != nil || !afterHeadMove.Applicable {
		t.Fatalf("unrelated HEAD movement staled selected worktree paths: result=%#v err=%v", afterHeadMove, err)
	}
	writeFixture(t, dir, "file.txt", "stale replacement\n")
	stale, err := s.Job(context.Background(), "repo", dir, worktree.Id, "", "advance", 0)
	if err != nil || stale.Applicable {
		t.Fatalf("changed selected content retained applicability: result=%#v err=%v", stale, err)
	}
	replayed, err := s.StartWorktree(context.Background(), "repo", dir, &v1.StartWorktreeStagedMessageRequest{Paths: []string{"new.txt", "file.txt"}, RequestKey: "selected-worktree", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP})
	if err != nil || replayed.Id != worktree.Id || replayed.Applicable {
		t.Fatalf("lost-ack replay did not reconnect to the original now-stale job: replay=%#v err=%v", replayed, err)
	}
	if _, err := s.StartWorktree(context.Background(), "repo", dir, &v1.StartWorktreeStagedMessageRequest{Paths: []string{"other.txt"}, RequestKey: "selected-worktree", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP}); !errors.Is(err, ErrConflict) {
		t.Fatalf("request key reuse for different paths was accepted: %v", err)
	}
}

func TestWorktreeAdmissionPathsPersistProviderContextLimits(t *testing.T) {
	for _, path := range []string{"worktree", "queued-selection"} {
		t.Run(path, func(t *testing.T) {
			s, store, _, provider := serviceFixture(t)
			dir, git := fixtureGit(t)
			s.snapshots = GitSnapshotter{}
			writeFixture(t, dir, "file.txt", "captured content\n")
			git("", "add", "file.txt")
			writeFixture(t, dir, "file.txt", "changed worktree content\n")
			provider.omissions = []*v1.Omission{{Path: "private/file.txt", Reason: "context source unavailable"}}
			provider.searchErr = errors.New("provider timeout api_key=supersecret " + strings.Repeat("transcript", 80))
			var job *v1.StagedMessageJob
			var err error
			if path == "worktree" {
				job, err = s.StartWorktree(context.Background(), "repo", dir, &v1.StartWorktreeStagedMessageRequest{Paths: []string{"file.txt"}, RequestKey: "omissions-worktree"})
			} else {
				queued, queueErr := s.QueueWorktreeSelection(context.Background(), "repo", dir, &v1.QueueWorktreeSelectionRequest{SelectionId: "omissions-selection", Source: v1.WorktreeSelectionSource_WORKTREE_SELECTION_SOURCE_INDEX_TO_WORKTREE, Paths: []string{"file.txt"}})
				if queueErr != nil {
					t.Fatal(queueErr)
				}
				job, err = s.StartQueuedSelection(context.Background(), "repo", dir, &v1.StartQueuedSelectionRequest{SelectionId: queued.SelectionId, SnapshotDigest: queued.SnapshotDigest, RequestKey: "omissions-queued", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK})
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(job.Omissions) != 2 || job.Omissions[0].GetPath() != "private/file.txt" || !strings.Contains(job.Omissions[1].GetReason(), "provider timeout") || strings.Contains(job.Omissions[1].GetReason(), "supersecret") || len(job.Omissions[1].GetReason()) > maxProviderReasonBytes {
				t.Fatalf("returned omissions were lost or unsafe: %+v", job.Omissions)
			}
			stored, err := store.Get(context.Background(), job.Id)
			if err != nil {
				t.Fatal(err)
			}
			var input DraftInput
			if input, err = decodeDraftInput([]byte(stored.InputJSON)); err != nil {
				t.Fatal(err)
			}
			if len(input.Omissions) != 2 || input.Omissions[0].GetPath() != "private/file.txt" || strings.Contains(input.Omissions[1].GetReason(), "supersecret") {
				t.Fatalf("immutable input did not preserve safe omissions: %+v", input.Omissions)
			}
		})
	}
}

func TestRetryDraftUsesSavedInputWithoutBrowserAdmission(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	collection, failed := admission(t, s, "retry-source")
	if err := store.Transition(context.Background(), failed.GetId(), []reviewjobs.State{reviewjobs.Queued, reviewjobs.Running}, reviewjobs.Failed, failed.GetExecutionId(), "invalid structured result"); err != nil {
		t.Fatal(err)
	}
	request := &v1.RetryStagedMessageRequest{JobId: failed.GetId(), SnapshotDigest: collection.GetSnapshot().GetDigest(), RequestKey: "retry-saved-input", Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK}
	retried, err := s.Retry(context.Background(), "repo", "fixture", request)
	if err != nil || retried.GetId() == failed.GetId() || provider.starts != 2 {
		t.Fatalf("retry did not create a new job from the saved input: job=%+v starts=%d err=%v", retried, provider.starts, err)
	}
	replay, err := s.Retry(context.Background(), "repo", "fixture", request)
	if err != nil || replay.GetId() != retried.GetId() || provider.starts != 2 {
		t.Fatalf("retry idempotency did not reconnect to the saved retry: replay=%+v starts=%d err=%v", replay, provider.starts, err)
	}
}

func TestWorktreeRefreshReplacesDurableQueueEntryAtomicallyAndReplays(t *testing.T) {
	s, _, _, _ := serviceFixture(t)
	dir, _ := fixtureGit(t)
	writeFixture(t, dir, "first.txt", "first selection\n")
	first, err := s.StartWorktree(context.Background(), "repo", dir, &v1.StartWorktreeStagedMessageRequest{Paths: []string{"first.txt"}, RequestKey: "first-selection"})
	if err != nil {
		t.Fatal(err)
	}
	queue, err := s.ListWorktreeQueue(context.Background(), "repo")
	if err != nil || len(queue.Entries) != 1 {
		t.Fatalf("initial durable queue absent: %#v %v", queue, err)
	}
	entry := queue.Entries[0]
	writeFixture(t, dir, "second.txt", "refreshed selection\n")
	req := &v1.StartWorktreeStagedMessageRequest{Paths: []string{"second.txt"}, RequestKey: "refreshed-selection", ReplaceQueueEntryId: entry.Id, ExpectedQueueRevision: queue.Revision}
	refreshed, err := s.StartWorktree(context.Background(), "repo", dir, req)
	if err != nil || refreshed.Id == first.Id {
		t.Fatalf("refresh failed to create a distinct job: %#v %v", refreshed, err)
	}
	queue, err = s.ListWorktreeQueue(context.Background(), "repo")
	if err != nil || len(queue.Entries) != 1 || queue.Revision != 2 || queue.Entries[0].Id != entry.Id || queue.Entries[0].JobId != refreshed.Id || queue.Entries[0].Sequence != entry.Sequence || queue.Entries[0].RequestKey != req.RequestKey {
		t.Fatalf("refresh did not replace entry in place: %#v %v", queue, err)
	}
	replayed, err := s.StartWorktree(context.Background(), "repo", dir, req)
	if err != nil || replayed.Id != refreshed.Id {
		t.Fatalf("lost-ack refresh did not replay: %#v %v", replayed, err)
	}
	after, err := s.ListWorktreeQueue(context.Background(), "repo")
	if err != nil || after.Revision != queue.Revision || len(after.Entries) != 1 || after.Entries[0].Id != entry.Id {
		t.Fatalf("refresh retry appended or changed queue: %#v %v", after, err)
	}
	oldReplay, err := s.StartWorktree(context.Background(), "repo", dir, &v1.StartWorktreeStagedMessageRequest{Paths: []string{"first.txt"}, RequestKey: "first-selection"})
	if err != nil || oldReplay.Id != first.Id {
		t.Fatalf("original request replay lost its job identity: %#v %v", oldReplay, err)
	}
	afterOldReplay, err := s.ListWorktreeQueue(context.Background(), "repo")
	if err != nil || afterOldReplay.Revision != 2 || len(afterOldReplay.Entries) != 1 || afterOldReplay.Entries[0].JobId != refreshed.Id {
		t.Fatalf("old request replay duplicated a replaced queue entry: %#v %v", afterOldReplay, err)
	}
}

func TestCollectionBoundsUnicodeContextWithoutSplittingCharacters(t *testing.T) {
	s, _, _, provider := serviceFixture(t)
	title := "A" + strings.Repeat("界", 100)
	body := strings.Repeat("界", 3000)
	provider.contexts = []*v1.StagedMessageContext{{Id: "unicode-source", Revision: "1", Title: title, Body: body}}
	c, err := s.Collect(context.Background(), "repo", "fixture")
	if err != nil {
		t.Fatalf("valid Unicode source must remain collectable: %v", err)
	}
	if len(c.Contexts) != 1 {
		t.Fatal("bounded Unicode context was dropped")
	}
	source := c.Contexts[0]
	if !utf8.ValidString(source.Title) || !utf8.ValidString(source.Body) || source.Title != title[:238] || source.Body != body[:8190] || len(source.Title) > 240 || len(source.Body) > 8192 {
		t.Fatalf("source clipping split characters or lost complete prefix: title bytes=%d body bytes=%d", len(source.Title), len(source.Body))
	}
	if _, err := protojson.Marshal(c); err != nil {
		t.Fatalf("bounded collection must encode as valid protobuf JSON: %v", err)
	}
	if provider.starts != 0 {
		t.Fatal("Unicode context collection started inference")
	}
}

func TestDurableAdmissionIsIdempotentAndReattachesByRequestKey(t *testing.T) {
	s, store, snap, p := serviceFixture(t)
	p.phase = "draft-message"
	c, j := admission(t, s, "request")
	if j.Phase != "draft-message" {
		t.Fatalf("owner stage was not persisted in the job: %+v", j)
	}
	same, e := s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "request"})
	if e != nil || same.Id != j.Id || p.starts != 1 {
		t.Fatalf("duplicate %#v err=%v starts=%d", same, e, p.starts)
	}
	reopened := New(store, snap, nil, p)
	got, e := reopened.Job(context.Background(), "repo", "fixture", "", "request", "advance", 0)
	if e != nil || got.Id != j.Id || got.Phase != "draft-message" || p.starts != 1 {
		t.Fatalf("reattach %#v err=%v starts=%d", got, e, p.starts)
	}
	_, e = reopened.Job(context.Background(), "other", "fixture", j.Id, "", "advance", 0)
	if !errors.Is(e, ErrNotFound) {
		t.Fatalf("cross repo %v", e)
	}
	_, e = s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "request", OperatorNote: "different"})
	if !errors.Is(e, ErrConflict) {
		t.Fatalf("key conflict %v", e)
	}
}

func TestStagedDriftRetainsResultAndDoesNotRestart(t *testing.T) {
	s, _, snap, p := serviceFixture(t)
	_, j := admission(t, s, "drift")
	p.complete("succeeded", validResult())
	snap.drift()
	got, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "advance", 0)
	if e != nil || got.Result == nil || got.Applicable || got.Status != "succeeded" || p.starts != 1 {
		t.Fatalf("drift %#v err=%v starts=%d", got, e, p.starts)
	}
}

func TestCancellationRemainsPendingUntilOwnerConfirmationAndRejectsLateResult(t *testing.T) {
	s, _, _, p := serviceFixture(t)
	_, j := admission(t, s, "cancel")
	got, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "cancel", 0)
	if e != nil || got.Status != "cancelling" || !got.CancelRequested {
		t.Fatalf("cancel %#v %v", got, e)
	}
	p.complete("succeeded", validResult())
	got, e = s.Job(context.Background(), "repo", "fixture", j.Id, "", "advance", 0)
	if e != nil || got.Result != nil || got.Status != "cancelled" {
		t.Fatalf("late result %#v %v", got, e)
	}
	p.complete("cancelled", nil)
	got, e = s.Job(context.Background(), "repo", "fixture", j.Id, "", "advance", 0)
	if e != nil || got.Status != "cancelled" || got.Result != nil {
		t.Fatalf("owner cancel %#v %v", got, e)
	}
}

func TestCancelWinsAgainstConcurrentWaitCompletion(t *testing.T) {
	s, _, _, p := serviceFixture(t)
	_, j := admission(t, s, "race")
	p.entered = make(chan struct{})
	p.release = make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "wait", time.Second)
		done <- e
	}()
	<-p.entered
	if _, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "cancel", 0); e != nil {
		t.Fatal(e)
	}
	p.complete("succeeded", validResult())
	close(p.release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	got, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "advance", 0)
	if e != nil || got.Result != nil || got.Status != "cancelled" {
		t.Fatalf("raced result %#v %v", got, e)
	}
}

func TestCancelBeforeOwnerStartDoesNotDispatch(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	job := createQueuedDraft(t, store, "cancel-before-start")
	var cancelErr error
	s.beforeDispatch = func(ctx context.Context, _ reviewjobs.Job) {
		_, cancelErr = s.Job(ctx, "repo", "fixture", job.ID, "", "cancel", 0)
	}

	got, err := s.Job(t.Context(), "repo", "fixture", job.ID, "", "advance", 0)
	stored, readErr := store.Get(t.Context(), job.ID)
	if err != nil || cancelErr != nil || readErr != nil || got.Status != string(reviewjobs.Cancelled) || stored.State != reviewjobs.Cancelled || provider.starts != 0 {
		t.Fatalf("cancel-before-start dispatched or stayed unresolved: projected=%+v stored=%+v starts=%d err=%v cancelErr=%v readErr=%v", got, stored, provider.starts, err, cancelErr, readErr)
	}
}

func TestCancelAfterAttemptMarkerUsesOwnerFenceWithoutStarting(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	queued := createQueuedDraft(t, store, "cancel-after-attempt-marker")
	dispatching, changed, err := store.ClaimAdvisoryDispatch(t.Context(), queued)
	if err != nil || !changed {
		t.Fatalf("claim dispatch: changed=%t err=%v", changed, err)
	}
	marked, changed, err := store.MarkAdvisoryDispatchAttempt(t.Context(), dispatching)
	if err != nil || !changed {
		t.Fatalf("mark dispatch attempt: changed=%t err=%v", changed, err)
	}
	if err := store.RequestAdvisoryCancel(t.Context(), queued.ID, "repo"); err != nil {
		t.Fatal(err)
	}
	provider.cancelByKeyResult = DispatchCancellation{Fenced: true}

	got, err := s.Job(t.Context(), "repo", "fixture", queued.ID, "", "advance", 0)
	stored, readErr := store.Get(t.Context(), queued.ID)
	if err != nil || readErr != nil || got.Status != string(reviewjobs.Cancelled) || stored.State != reviewjobs.Cancelled || stored.ErrorCode != "cancelled_before_start" || provider.starts != 0 || provider.cancelByKeyCalls != 1 {
		t.Fatalf("owner-fenced cancellation miss did not complete safely: projected=%+v stored=%+v starts=%d fences=%d err=%v readErr=%v marked=%+v", got, stored, provider.starts, provider.cancelByKeyCalls, err, readErr, marked)
	}
	if !stored.ExecutionStartedKnown || stored.ExecutionStarted || !stored.TokensKnown || stored.TokensUsed != 0 || !stored.ChargeKnown || stored.ChargeMicroUSD != 0 {
		t.Fatalf("owner-fenced miss did not preserve proven no-work receipt: %+v", stored)
	}
}

func TestCancelFenceWinsAgainstBlockedStartBeforeOwnerAdmission(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	provider.startEntered = make(chan struct{})
	provider.startRelease = make(chan struct{})
	provider.cancelByKeyResult = DispatchCancellation{Fenced: true}
	provider.startErrors = map[int]error{1: connect.NewError(connect.CodeFailedPrecondition, errors.New("request key fenced by cancellation"))}
	queued := createQueuedDraft(t, store, "cancel-fence-blocked-start")
	done := make(chan error, 1)
	go func() {
		_, err := s.Job(context.Background(), "repo", "fixture", queued.ID, "", "advance", 0)
		done <- err
	}()
	select {
	case <-provider.startEntered:
	case <-time.After(3 * time.Second):
		t.Fatal("Start did not reach the deterministic owner barrier")
	}
	cancelled, err := s.Job(t.Context(), "repo", "fixture", queued.ID, "", "cancel", 0)
	stored, readErr := store.Get(t.Context(), queued.ID)
	if err != nil || readErr != nil || cancelled.Status != string(reviewjobs.Cancelled) || stored.State != reviewjobs.Cancelled || !stored.ExecutionStartedKnown || stored.ExecutionStarted || !stored.TokensKnown || stored.TokensUsed != 0 || !stored.ChargeKnown || stored.ChargeMicroUSD != 0 {
		t.Fatalf("atomic fence did not establish a safe pre-start cancellation: job=%+v stored=%+v err=%v readErr=%v", cancelled, stored, err, readErr)
	}
	close(provider.startRelease)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	stored, readErr = store.Get(t.Context(), queued.ID)
	if readErr != nil || provider.starts != 1 || provider.cancelByKeyCalls != 1 || stored.State != reviewjobs.Cancelled || stored.ExecutionID != "" || stored.ResultJSON != "" {
		t.Fatalf("late blocked Start crossed the cancellation fence: stored=%+v starts=%d fences=%d readErr=%v", stored, provider.starts, provider.cancelByKeyCalls, readErr)
	}
}

func TestCancelRacingAcceptedStartAdoptsAndCancelsOwner(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	provider.startEntered = make(chan struct{})
	provider.startRelease = make(chan struct{})
	provider.status = "cancelling"
	provider.cancelByKeyResult = DispatchCancellation{Fenced: true, Execution: &Execution{ID: "owner-execution", Status: "cancelling"}}
	provider.cancelStatus = "cancelled"
	queued := createQueuedDraft(t, store, "cancel-racing-start")
	done := make(chan struct {
		job *v1.StagedMessageJob
		err error
	}, 1)
	go func() {
		job, err := s.Job(context.Background(), "repo", "fixture", queued.ID, "", "advance", 0)
		done <- struct {
			job *v1.StagedMessageJob
			err error
		}{job: job, err: err}
	}()
	select {
	case <-provider.startEntered:
	case <-time.After(3 * time.Second):
		t.Fatal("Start did not reach the deterministic race barrier")
	}
	waiting, err := s.Job(t.Context(), "repo", "fixture", queued.ID, "", "cancel", 0)
	if err != nil || waiting.Status != string(reviewjobs.Cancelling) || !waiting.CancelRequested || provider.cancels != 0 || provider.cancelByKeyCalls != 1 {
		t.Fatalf("atomic owner cancellation did not cancel the already admitted handle: job=%+v cancels=%d fences=%d err=%v", waiting, provider.cancels, provider.cancelByKeyCalls, err)
	}
	close(provider.startRelease)
	result := <-done
	stored, readErr := store.Get(t.Context(), queued.ID)
	if result.err != nil || readErr != nil || result.job.Status != string(reviewjobs.Cancelling) || stored.State != reviewjobs.Cancelling || stored.ResultJSON != "" || stored.ExecutionID != "owner-execution" || provider.starts != 1 || provider.cancels != 0 {
		t.Fatalf("accepted Start was not adopted and cancelled: result=%+v stored=%+v starts=%d cancels=%d err=%v readErr=%v", result.job, stored, provider.starts, provider.cancels, result.err, readErr)
	}
	provider.mu.Lock()
	provider.status = "cancelled"
	provider.mu.Unlock()
	settled, err := s.Settle(t.Context(), fixtureDirectory)
	stored, readErr = store.Get(t.Context(), queued.ID)
	if err != nil || readErr != nil || settled != 1 || stored.State != reviewjobs.Cancelled || provider.cancels != 1 {
		t.Fatalf("adopted terminal receipt did not settle: settled=%d stored=%+v cancels=%d err=%v readErr=%v", settled, stored, provider.cancels, err, readErr)
	}
}

func TestCancelledAmbiguousDispatchAdoptsHandleAndSettleFinishesCancellation(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	provider.startErrors = map[int]error{1: connect.NewError(connect.CodeUnavailable, errors.New("lost Start acknowledgement"))}
	provider.status = "cancelling"
	provider.cancelByKeyResult = DispatchCancellation{Fenced: true, Execution: &Execution{ID: "owner-execution", Status: "cancelling"}}
	provider.cancelStatus = "running"
	queued := createQueuedDraft(t, store, "cancel-ambiguous-start")
	uncertain, err := s.Job(t.Context(), "repo", "fixture", queued.ID, "", "advance", 0)
	if err != nil || uncertain.Status != string(reviewjobs.Dispatching) || uncertain.ErrorCode != "dispatch_unconfirmed" || provider.starts != 1 {
		t.Fatalf("ambiguous owner acknowledgement was not preserved: job=%+v starts=%d err=%v", uncertain, provider.starts, err)
	}

	awaiting, err := s.Job(t.Context(), "repo", "fixture", queued.ID, "", "cancel", 0)
	stored, readErr := store.Get(t.Context(), queued.ID)
	if err != nil || readErr != nil || awaiting.Status != string(reviewjobs.Cancelling) || !awaiting.CancelRequested || awaiting.ExecutionId != "owner-execution" || stored.ExecutionID != "owner-execution" || provider.starts != 1 || provider.cancelByKeyCalls != 1 || provider.cancels != 0 {
		t.Fatalf("cancel did not fence, adopt and cancel the same owner job: job=%+v stored=%+v starts=%d fences=%d cancels=%d err=%v readErr=%v", awaiting, stored, provider.starts, provider.cancelByKeyCalls, provider.cancels, err, readErr)
	}

	provider.mu.Lock()
	provider.cancelStatus = "cancelled"
	provider.mu.Unlock()
	settled, err := s.Settle(t.Context(), fixtureDirectory)
	stored, readErr = store.Get(t.Context(), queued.ID)
	if err != nil || readErr != nil || settled != 1 || stored.State != reviewjobs.Cancelled || stored.ResultJSON != "" || provider.cancels != 1 {
		t.Fatalf("settle did not continue cancellation through terminal owner confirmation: settled=%d stored=%+v cancels=%d err=%v readErr=%v", settled, stored, provider.cancels, err, readErr)
	}
}

func TestCancelFencedStartAdoptsTerminalOwnerReceiptWithoutRecancel(t *testing.T) {
	for _, tc := range []struct {
		name, ownerStatus string
		want              reviewjobs.State
	}{
		{name: "succeeded before cancellation", ownerStatus: "succeeded", want: reviewjobs.Cancelled},
		{name: "failed before cancellation", ownerStatus: "failed", want: reviewjobs.Failed},
		{name: "cancelled before cancellation receipt", ownerStatus: "cancelled", want: reviewjobs.Cancelled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, store, _, provider := serviceFixture(t)
			provider.startEntered = make(chan struct{})
			provider.startRelease = make(chan struct{})
			provider.status = tc.ownerStatus
			provider.error = "terminal owner error"
			provider.errorCode = "child_runner_failed"
			provider.executionStarted, provider.executionStartedKnown = true, true
			provider.tokensUsed, provider.tokensKnown = 9, true
			provider.chargeKnown = false
			provider.result = &v1.StagedMessageSuggestion{Title: "terminal success", Body: "discard if cancellation was requested"}
			provider.cancelByKeyResult = DispatchCancellation{Fenced: true, Execution: &Execution{ID: "owner-execution", Status: tc.ownerStatus}}
			queued := createQueuedDraft(t, store, "cancel-terminal-"+tc.ownerStatus)
			type result struct {
				job *v1.StagedMessageJob
				err error
			}
			done := make(chan result, 1)
			go func() {
				job, err := s.Job(context.Background(), "repo", "fixture", queued.ID, "", "advance", 0)
				done <- result{job: job, err: err}
			}()
			select {
			case <-provider.startEntered:
			case <-time.After(3 * time.Second):
				t.Fatal("Start did not reach the deterministic terminal-race barrier")
			}

			got, err := s.Job(t.Context(), "repo", "fixture", queued.ID, "", "cancel", 0)
			close(provider.startRelease)
			lateStart := <-done
			stored, readErr := store.Get(t.Context(), queued.ID)
			if err != nil || readErr != nil || got.Status != string(tc.want) || stored.State != tc.want || stored.ExecutionID != "owner-execution" {
				t.Fatalf("terminal owner receipt was not adopted: job=%+v stored=%+v err=%v readErr=%v", got, stored, err, readErr)
			}
			stored, readErr = store.Get(t.Context(), queued.ID)
			if lateStart.err != nil || lateStart.job.Status != string(tc.want) || readErr != nil || stored.State != tc.want {
				t.Fatalf("late Start overwrote the terminal owner receipt: result=%+v stored=%+v readErr=%v", lateStart, stored, readErr)
			}
			if provider.starts != 1 || provider.cancelByKeyCalls != 1 || provider.cancels != 0 || provider.gets != 1 {
				t.Fatalf("terminal receipt replayed dispatch/cancel: starts=%d fences=%d cancels=%d gets=%d", provider.starts, provider.cancelByKeyCalls, provider.cancels, provider.gets)
			}
			if stored.ResultJSON != "" || !stored.ExecutionStartedKnown || !stored.ExecutionStarted || !stored.TokensKnown || stored.TokensUsed != 9 || stored.ChargeKnown {
				t.Fatalf("terminal receipt lost cancellation or accounting evidence: %+v", stored)
			}
		})
	}
}

func TestMalformedStructuredResultFailsWithoutTranscriptFallback(t *testing.T) {
	s, _, _, p := serviceFixture(t)
	_, j := admission(t, s, "invalid")
	p.complete("succeeded", &v1.StagedMessageSuggestion{Title: "fake\nbody"})
	got, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "advance", 0)
	if e != nil || got.Status != "failed" || got.ErrorCode != "invalid_structured_output" || !got.Retryable || got.Result != nil {
		t.Fatalf("invalid %#v %v", got, e)
	}
}

func TestStaleCollectionCannotStartAgent(t *testing.T) {
	s, _, snap, p := serviceFixture(t)
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	snap.drift()
	_, e = s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "stale"})
	if !errors.Is(e, ErrConflict) || p.starts != 0 {
		t.Fatalf("stale admitted %v starts=%d", e, p.starts)
	}
}

type lostAckProvider struct {
	fakeProvider
	muAck   sync.Mutex
	keys    []string
	ackLost bool
}

func (p *lostAckProvider) Start(ctx context.Context, input DraftInput, key string) (Execution, error) {
	p.muAck.Lock()
	defer p.muAck.Unlock()
	p.keys = append(p.keys, key)
	if !p.ackLost {
		p.ackLost = true
		return Execution{}, errors.New("connection lost after owner accepted request")
	}
	return Execution{ID: "same-owner-execution", Status: "running"}, nil
}

func TestLostAdmissionAcknowledgementReconcilesSameDurableKey(t *testing.T) {
	s, store, snap, _ := serviceFixture(t)
	provider := &lostAckProvider{}
	s.provider = provider
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	first, e := s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "lost-ack"})
	if e != nil || first.Status == "failed" || first.ExecutionId != "" {
		t.Fatalf("lost ack %#v %v", first, e)
	}
	reopened := New(store, snap, nil, provider)
	got, e := reopened.Job(context.Background(), "repo", "fixture", "", "lost-ack", "advance", 0)
	if e != nil || got.ExecutionId != "same-owner-execution" || len(provider.keys) != 2 || provider.keys[0] != provider.keys[1] {
		t.Fatalf("reattach %#v err=%v keys=%v", got, e, provider.keys)
	}
}

type uncertainAdmissionProvider struct {
	fakeProvider
	deny     bool
	keys     []string
	accepted map[string]bool
}

func (p *uncertainAdmissionProvider) Start(_ context.Context, _ DraftInput, requestKey string) (Execution, error) {
	p.keys = append(p.keys, requestKey)
	if p.deny {
		return Execution{}, connect.NewError(connect.CodeUnauthenticated, errors.New("caller credential expired"))
	}
	if !p.accepted[requestKey] {
		p.accepted[requestKey] = true
		return Execution{}, errors.New("connection lost after owner admission")
	}
	return Execution{ID: "original-owner-execution", Status: "running"}, nil
}

func TestLostAdmissionAcknowledgementSurvivesLaterAuthenticationFailure(t *testing.T) {
	s, store, snapshot, _ := serviceFixture(t)
	p := &uncertainAdmissionProvider{accepted: map[string]bool{}}
	s.provider = p
	_, first := admission(t, s, "uncertain-admission")
	if first.Status != "dispatching" || first.ExecutionId != "" || first.ErrorCode != "dispatch_unconfirmed" || strings.Contains(first.Error, "connection lost") {
		t.Fatalf("initial uncertainty was lost: %v", first)
	}
	p.deny = true
	reopened := New(store, snapshot, nil, p)
	denied, err := reopened.Job(t.Context(), "repo", "fixture", "", "uncertain-admission", "advance", 0)
	if err != nil || denied.Status != "dispatching" || denied.Id != first.Id || denied.ExecutionId != "" || denied.Error == "" {
		t.Fatalf("later auth failure invented definitive rejection: %v %v", denied, err)
	}
	// Provider absence after another restart is also not evidence that the
	// original owner admission never happened.
	unavailable, err := New(store, snapshot, nil, nil).Job(t.Context(), "repo", "fixture", "", "uncertain-admission", "advance", 0)
	if err != nil || unavailable.Status != "dispatching" || unavailable.Id != first.Id || unavailable.ErrorCode != "dispatch_unconfirmed" || unavailable.ExecutionStartedKnown || unavailable.TokensKnown || unavailable.ChargeKnown {
		t.Fatalf("provider outage discarded uncertain admission or invented no-work certainty: %v %v", unavailable, err)
	}
	p.deny = false
	restored, err := New(store, snapshot, nil, p).Job(t.Context(), "repo", "fixture", "", "uncertain-admission", "advance", 0)
	if err != nil || restored.Status != "running" || restored.ExecutionId != "original-owner-execution" || restored.Id != first.Id || len(p.accepted) != 1 || len(p.keys) != 3 {
		t.Fatalf("credential restore did not recover single original execution: %v %v keys=%v", restored, err, p.keys)
	}
	for _, requestKey := range p.keys {
		if requestKey != key("repo", "uncertain-admission") {
			t.Fatal("reconciliation changed original owner key")
		}
	}
}

func TestFirstDefinitiveAdmissionRejectionRemainsTerminal(t *testing.T) {
	s, store, snapshot, _ := serviceFixture(t)
	p := &uncertainAdmissionProvider{deny: true, accepted: map[string]bool{}}
	s.provider = p
	_, job := admission(t, s, "first-auth-rejection")
	if job.Status != "failed" || job.ExecutionId != "" || job.ErrorCode != "authentication_required" || strings.Contains(job.Error, "credential expired") || len(p.accepted) != 0 || len(p.keys) != 1 {
		t.Fatalf("first definitive refusal not preserved: %v", job)
	}
	if got, err := New(store, snapshot, nil, p).Job(t.Context(), "repo", "fixture", "", "first-auth-rejection", "advance", 0); err != nil || got.Status != "failed" || len(p.keys) != 1 {
		t.Fatalf("retry contacted owner after definitive refusal: %v %v", got, err)
	}
}

type interleavedAdmissionProvider struct {
	fakeProvider
	admissionMu                sync.Mutex
	keys                       []string
	accepted                   map[string]bool
	firstEntered, firstRelease chan struct{}
	retryEntered, retryRelease chan struct{}
}

func (p *interleavedAdmissionProvider) Start(ctx context.Context, _ DraftInput, requestKey string) (Execution, error) {
	p.admissionMu.Lock()
	p.keys = append(p.keys, requestKey)
	call := len(p.keys)
	if call == 2 {
		p.accepted[requestKey] = true
	}
	p.admissionMu.Unlock()
	if call == 1 {
		close(p.firstEntered)
		select {
		case <-p.firstRelease:
		case <-ctx.Done():
			return Execution{}, ctx.Err()
		}
		return Execution{}, connect.NewError(connect.CodeUnauthenticated, errors.New("first caller credential expired"))
	}
	if call == 2 {
		close(p.retryEntered)
		select {
		case <-p.retryRelease:
		case <-ctx.Done():
			return Execution{}, ctx.Err()
		}
		return Execution{}, errors.New("retry connection lost after owner accepted request")
	}
	return Execution{ID: "original-interleaved-execution", Status: "running"}, nil
}

func TestConcurrentRetryAdmissionCannotBeTerminatedByOlderAuthenticationRefusal(t *testing.T) {
	s, store, snapshot, _ := serviceFixture(t)
	p := &interleavedAdmissionProvider{accepted: map[string]bool{}, firstEntered: make(chan struct{}), firstRelease: make(chan struct{}), retryEntered: make(chan struct{}), retryRelease: make(chan struct{})}
	s.provider = p
	c, err := s.Collect(t.Context(), "repo", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	type response struct {
		job *v1.StagedMessageJob
		err error
	}
	first := make(chan response, 1)
	go func() {
		job, err := s.Start(t.Context(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "interleaved"})
		first <- response{job, err}
	}()
	select {
	case <-p.firstEntered:
	case <-time.After(3 * time.Second):
		t.Fatal("first admission did not reach owner")
	}
	retries := make(chan response, 1)
	go func() {
		job, err := New(store, snapshot, nil, p).Job(t.Context(), "repo", "fixture", "", "interleaved", "advance", 0)
		retries <- response{job, err}
	}()
	select {
	case <-p.retryEntered:
	case <-time.After(3 * time.Second):
		close(p.firstRelease)
		close(p.retryRelease)
		t.Fatal("retry did not reach owner")
	}
	close(p.firstRelease)
	var firstReply response
	select {
	case firstReply = <-first:
	case <-time.After(3 * time.Second):
		close(p.retryRelease)
		t.Fatal("first admission failed to finish")
	}
	close(p.retryRelease)
	var retryReply response
	select {
	case retryReply = <-retries:
	case <-time.After(3 * time.Second):
		t.Fatal("retry failed to finish")
	}
	if firstReply.err != nil || firstReply.job.Status != "dispatching" {
		t.Fatalf("older refusal erased accepted retry: %v %v", firstReply.job, firstReply.err)
	}
	retry := retryReply.job
	if retryReply.err != nil || retry.Status != "dispatching" || retry.Id != firstReply.job.Id || retry.ExecutionId != "" {
		t.Fatalf("accepted retry uncertainty lost: %v %v", retry, retryReply.err)
	}
	restored, err := New(store, snapshot, nil, p).Job(t.Context(), "repo", "fixture", "", "interleaved", "advance", 0)
	if err != nil || restored.Id != retry.Id || restored.ExecutionId != "original-interleaved-execution" || restored.Status != "running" {
		t.Fatalf("did not recover original execution: %v %v", restored, err)
	}
	p.admissionMu.Lock()
	defer p.admissionMu.Unlock()
	if len(p.accepted) != 1 || len(p.keys) != 3 {
		t.Fatalf("unexpected owner admissions: accepted=%v calls=%d", p.accepted, len(p.keys))
	}
	for _, requestKey := range p.keys {
		if requestKey != key("repo", "interleaved") {
			t.Fatal("retry replaced original owner identity")
		}
	}
}

func TestResultCoverageRejectsMissingDuplicatedAndInventedScope(t *testing.T) {
	input := DraftInput{Snapshot: &v1.StagedMessageSnapshot{Files: []*v1.StagedMessageFile{{Path: "file", EditCount: 1}}}}
	for _, test := range []struct {
		name   string
		change func(*v1.StagedMessageSuggestion)
	}{
		{"missing file", func(r *v1.StagedMessageSuggestion) { r.Coverage.CoveredPaths = nil }},
		{"duplicate file", func(r *v1.StagedMessageSuggestion) {
			r.Coverage.UncoveredPaths = []string{"file"}
			r.Coverage.Partial = true
		}},
		{"invented path", func(r *v1.StagedMessageSuggestion) { r.Coverage.CoveredPaths = []string{"other"} }},
		{"invalid count", func(r *v1.StagedMessageSuggestion) { r.Coverage.MatchedEdits = 2 }},
		{"invalid trailer", func(r *v1.StagedMessageSuggestion) {
			r.Trailers = []*v1.StagedMessageTrailer{{Key: "bad:key", Value: "value"}}
		}},
		{"inferred attribution", func(r *v1.StagedMessageSuggestion) {
			r.Trailers = []*v1.StagedMessageTrailer{{Key: "Vrooli-Run", Value: "invented"}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := validResult()
			test.change(r)
			if e := validateResult(input, r); e == nil {
				t.Fatal("invalid structured result accepted")
			}
		})
	}
}

type concurrentOwner struct {
	fakeProvider
	admissionMu sync.Mutex
	keys        map[string]bool
	entered     chan struct{}
	release     chan struct{}
}

func (p *concurrentOwner) Start(ctx context.Context, _ DraftInput, key string) (Execution, error) {
	p.admissionMu.Lock()
	p.keys[key] = true
	p.admissionMu.Unlock()
	p.entered <- struct{}{}
	select {
	case <-p.release:
		return Execution{ID: "one-owner-execution", Status: "running"}, nil
	case <-ctx.Done():
		return Execution{}, ctx.Err()
	}
}

func TestConcurrentAdmissionRetainsOneJobAndOneOwnerIdempotencyIdentity(t *testing.T) {
	s, _, _, _ := serviceFixture(t)
	p := &concurrentOwner{keys: map[string]bool{}, entered: make(chan struct{}, 4), release: make(chan struct{})}
	s.provider = p
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	type reply struct {
		job *v1.StagedMessageJob
		err error
	}
	replies := make(chan reply, 4)
	call := func() {
		j, e := s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "concurrent"})
		replies <- reply{j, e}
	}
	go call()
	select {
	case <-p.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("first admission did not reach owner")
	}
	for i := 0; i < 3; i++ {
		go call()
	}
	// Competing row claims may elide redundant owner calls. The contract is
	// one durable job and owner identity, not one HTTP call per observer.
	close(p.release)
	want := ""
	for i := 0; i < 4; i++ {
		r := <-replies
		if r.err != nil {
			t.Fatal(r.err)
		}
		if want == "" {
			want = r.job.Id
		}
		if r.job.Id != want {
			t.Fatal("duplicate durable jobs")
		}
	}
	p.admissionMu.Lock()
	defer p.admissionMu.Unlock()
	if len(p.keys) != 1 {
		t.Fatalf("created new owner identities: %v", p.keys)
	}
}

func TestWaitOnLostAdmissionDoesNotSpinOnFastTransportFailure(t *testing.T) {
	s, _, _, _ := serviceFixture(t)
	p := &alwaysUnavailable{}
	s.provider = p
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	j, e := s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "outage"})
	if e != nil {
		t.Fatal(e)
	}
	start := time.Now()
	got, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "wait", 25*time.Millisecond)
	if e != nil || got.Status != "dispatching" || got.Error == "" || time.Since(start) < 20*time.Millisecond {
		t.Fatalf("unpaced/false-terminal wait %#v %v elapsed=%v", got, e, time.Since(start))
	}
}

type alwaysUnavailable struct{ fakeProvider }

func (*alwaysUnavailable) Start(context.Context, DraftInput, string) (Execution, error) {
	return Execution{}, errors.New("owner transport unavailable")
}

func TestDraftInputRetainsCanonicalEvidenceAboveProviderPromptBound(t *testing.T) {
	s, _, snapshot, p := serviceFixture(t)
	snapshot.patch = strings.Repeat("x", 100000)
	var selected []string
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("context-%d", i)
		selected = append(selected, id)
		p.contexts = append(p.contexts, &v1.StagedMessageContext{Id: id, Revision: "immutable", Body: strings.Repeat("y", 8100)})
	}
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, SelectedContextIds: selected, RequestKey: "bounded"})
	if e != nil {
		t.Fatal(e)
	}
	raw, err := json.Marshal(p.lastInput)
	if err != nil || len(raw) <= legacyProviderInputBytes || len(raw) > MaxDraftInputBytes || p.lastInput.FormatVersion != 2 || len(p.lastInput.Snapshot.Files[0].Patch) != 100000 || len(c.Snapshot.Files[0].Patch) != 100000 {
		t.Fatal("canonical durable input was constrained by the separate provider packaging bound", err)
	}
}

func TestLegacyAdmissionRetryUsesOriginalDurableInputAndRejectsChangedIntent(t *testing.T) {
	s, store, snapshot, provider := serviceFixture(t)
	snapshot.patch = strings.Repeat("x", 100000)
	provider.contexts = []*v1.StagedMessageContext{{Id: "source@1", Revision: "1", Body: "retained context"}}
	c, err := s.Collect(context.Background(), "repo", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	oldSnapshot := proto.Clone(c.Snapshot).(*v1.StagedMessageSnapshot)
	oldSnapshot.Files[0].Patch = ""
	oldSnapshot.Files[0].OmissionReason = "patch omitted by immutable draft input byte bound"
	old := DraftInput{FormatVersion: 2, Snapshot: oldSnapshot, Contexts: provider.contexts, OperatorNote: "same intent"}
	raw, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	job, _, err := store.Create(context.Background(), reviewjobs.Job{ID: "legacy-job", IdempotencyKey: key("repo", "legacy-key"), InputDigest: digest(raw), InputJSON: string(raw), ResultRef: c.Id, Operation: operation, RepositoryID: "repo", ExecutionID: "legacy-owner", State: reviewjobs.Running})
	if err != nil {
		t.Fatal(err)
	}
	req := &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, SelectedContextIds: []string{"source@1"}, OperatorNote: "same intent", RequestKey: "legacy-key"}
	got, err := s.Start(context.Background(), "repo", "fixture", req)
	if err != nil || got.Id != job.ID || provider.starts != 0 || got.Snapshot.Files[0].Patch != "" {
		t.Fatalf("legacy retry did not reattach original input: %v %v starts=%d", got, err, provider.starts)
	}
	retained, err := store.Get(context.Background(), job.ID)
	if err != nil || retained.InputJSON != string(raw) {
		t.Fatal("legacy immutable input was repackaged", err)
	}
	for _, change := range []string{"note", "sources", "collection", "subject"} {
		t.Run(change, func(t *testing.T) {
			altered := proto.Clone(req).(*v1.StartStagedMessageRequest)
			switch change {
			case "note":
				altered.OperatorNote = "different intent"
			case "sources":
				altered.SelectedContextIds = nil
			case "collection":
				other, err := s.Collect(context.Background(), "repo", "fixture")
				if err != nil {
					t.Fatal(err)
				}
				altered.CollectionId = other.Id
			case "subject":
				altered.SnapshotDigest = "different subject"
			}
			if _, err := s.Start(context.Background(), "repo", "fixture", altered); !errors.Is(err, ErrConflict) {
				t.Fatalf("changed %s must conflict: %v", change, err)
			}
		})
	}
}

type retainedFixture []RetainedMessage

func (r retainedFixture) Messages(context.Context, string, *v1.StagedMessageSnapshot) ([]RetainedMessage, []*v1.Omission, error) {
	return r, nil, nil
}

type immutableSnapshotFixture struct{ captured Captured }

func (f immutableSnapshotFixture) Capture(context.Context, string, string) (Captured, error) {
	return f.captured, nil
}

func (f immutableSnapshotFixture) Identity(context.Context, string, string) (string, error) {
	return f.captured.Snapshot.Digest, nil
}

func TestCollectionPersists4096LongPathsAndBudgetsExplanatoryDuplicates(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	snapshot := &v1.StagedMessageSnapshot{RepositoryId: "repo", Digest: "sha256:complete-large-manifest"}
	files := make([]RetainedFile, 0, MaxManifestFiles)
	for i := 0; i < MaxManifestFiles; i++ {
		// Roughly1KiB paths fit ordinary filesystem component lengths; the full
		// selection identity remains much larger than the provider prompt budget.
		path := fmt.Sprintf("scenarios/scope-%02d/%sfile-%04d.go", i%32, strings.Repeat("ordinary-directory/", 55), i)
		snapshot.Files = append(snapshot.Files, &v1.StagedMessageFile{Path: path, OldPath: path, OldBlobId: "old-blob", NewBlobId: "new-blob", OldMode: "100644", NewMode: "100644", OmissionReason: "representative detail omitted"})
		files = append(files, RetainedFile{Path: path, BaseBlobID: "old-blob", BlobID: "new-blob", OldMode: "100644", NewMode: "100644", Captured: true})
	}
	var messages retainedFixture
	for i := 0; i < 100; i++ {
		messages = append(messages, RetainedMessage{ID: fmt.Sprintf("proposal-%03d", i), Revision: "1", Title: "Retained scoped update", Body: strings.Repeat("context ", 1000), Files: files})
	}
	s.snapshots = immutableSnapshotFixture{Captured{Snapshot: snapshot}}
	s.messages = messages
	c, err := s.Collect(context.Background(), "repo", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := protojson.Marshal(c)
	if err != nil || len(encoded) <= 2<<20 || len(encoded) > MaxCollectionBytes || len(c.Snapshot.Files) != MaxManifestFiles || len(c.Suggestions) != 1 || len(c.Contexts) != 100 {
		t.Fatalf("large collection not bounded: bytes=%d files=%d suggestions=%d contexts=%d err=%v", len(encoded), len(c.Snapshot.Files), len(c.Suggestions), len(c.Contexts), err)
	}
	if len(c.Suggestions[0].Coverage.CoveredPaths) != MaxManifestFiles || c.Suggestions[0].Coverage.Partial {
		t.Fatal("complete deterministic coverage was truncated")
	}
	if provider.starts != 0 {
		t.Fatal("large collection started an agent")
	}
	job, err := s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: snapshot.Digest, SelectedContextIds: []string{c.Contexts[0].Id}, RequestKey: "large-selection"})
	if err != nil {
		t.Fatal(err)
	}
	retained, err := store.Get(context.Background(), job.Id)
	if err != nil || len(retained.InputJSON) <= legacyProviderInputBytes || len(provider.lastInput.Snapshot.Files) != MaxManifestFiles {
		t.Fatalf("complete durable admission lost: %v input bytes=%d", err, len(retained.InputJSON))
	}
	reattached, err := New(store, s.snapshots, nil, provider).Job(context.Background(), "repo", "fixture", "", "large-selection", "advance", 0)
	if err != nil || reattached.Id != job.Id || len(reattached.Snapshot.Files) != MaxManifestFiles || reattached.Snapshot.Digest != snapshot.Digest {
		t.Fatal("restart lost canonical manifest", err)
	}
	t.Logf("collection JSON=%d bytes; durable job input=%d bytes; manifest=%d; contexts=%d; complete suggestions=%d", len(encoded), len(retained.InputJSON), len(c.Snapshot.Files), len(c.Contexts), len(c.Suggestions))
}

type expiredReadProvider struct{ fakeProvider }

func (*expiredReadProvider) Get(context.Context, string, DraftInput) (Execution, error) {
	return Execution{}, connect.NewError(connect.CodeUnauthenticated, errors.New("owner read credential expired"))
}

func TestOwnerReadAuthenticationFailureDoesNotInventTerminalExecution(t *testing.T) {
	s, store, snapshot, _ := serviceFixture(t)
	_, j := admission(t, s, "auth-expiry")
	s.provider = &expiredReadProvider{}
	got, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "advance", 0)
	if e != nil || got.Status != "running" || got.ExecutionId == "" || got.Error == "" {
		t.Fatalf("read failure changed authoritative phase %#v %v", got, e)
	}
	reopened := New(store, snapshot, nil, nil)
	got, e = reopened.Job(context.Background(), "repo", "fixture", j.Id, "", "advance", 0)
	if e != nil || got.Status != "running" || got.ExecutionId == "" {
		t.Fatalf("provider outage invented termination %#v %v", got, e)
	}
}
