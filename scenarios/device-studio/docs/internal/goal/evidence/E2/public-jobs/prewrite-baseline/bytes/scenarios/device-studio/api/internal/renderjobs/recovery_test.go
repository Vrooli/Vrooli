package renderjobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"device-studio/internal/jobs"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
	_ "modernc.org/sqlite"
)

func recoveryFixture(t *testing.T) (*Coordinator, context.Context, jobs.Service, jobs.Job, platform.OperationIntent, string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "jobs.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CGO_ENABLED", "0")
	if _, err := db.Exec(jobs.Schema()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	js := jobs.NewService(jobs.NewSQLiteRepository(db), time.Now)
	primary, testData := t.TempDir(), t.TempDir()
	roots := filerouting.New(storage.Paths{DataDir: primary})
	if err := roots.InstallTestRoots(storage.Paths{DataDir: testData}, "lease", time.Hour); err != nil {
		t.Fatal(err)
	}
	ctx := database.WithTestMode(context.Background())
	coord := &Coordinator{jobs: js, blobs: blobstore.NewMemoryBlobStore(), roots: roots, now: time.Now}
	coord.ops = absentOperations{}
	route, err := coord.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	in := jobs.Input{ProjectID: "p", ShotRevisionID: "shot:s:revision:1", ShotRevision: 1,
		Bindings: []jobs.ArtifactBinding{
			{Role: "shot_snapshot", ArtifactID: "shot:s:revision:1", RevisionID: "revision:1", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			{Role: "worker_executable", ArtifactID: workerBlobPrefix + "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd/device-studio-api", RevisionID: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", SHA256: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}, routeBinding(route)},
		SourceClockDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ToolProfileDigest: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", RendererVersion: RendererVersion, OutputProfile: OutputProfilePNG}
	j, _, err := js.Submit(ctx, jobs.SubmitInput{IdempotencyKey: "k", Input: in})
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := attemptWorkspace(route.Root, j.ID, j.Attempts[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	spec := operationSpec(filepath.Join(workspace, workerFileName), filepath.Join(workspace, "render-manifest.json"), workspace)
	intent, err := platform.NewOperationIntent(OwnerID, j.Attempts[0].ID, spec)
	if err != nil {
		t.Fatal(err)
	}
	envelope := persistedIntent{Version: 1, JobID: j.ID, AttemptID: j.Attempts[0].ID, InputFingerprint: j.InputFingerprint, Route: route, Intent: intent, Spec: spec, WorkerSHA256: bindingDigest(j.Input, "worker_executable")}
	intentBytes, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	key := intentBlobPrefix + j.ID + "/" + j.Attempts[0].ID + "/" + digestBytes(intentBytes) + ".json"
	if err := coord.putVerified(ctx, route, key, intentBytes, "application/json", digestBytes(intentBytes)); err != nil {
		t.Fatal(err)
	}
	running, err := js.Start(ctx, j.ID, jobs.ExecutionClaimInput{AttemptID: j.Attempts[0].ID, AttemptNumber: 1, OwnerID: OwnerID, ScopeID: intent.InvocationID, ArtifactID: key, SHA256: digestBytes(intentBytes)}, j.Version)
	if err != nil {
		t.Fatal(err)
	}
	return coord, ctx, js, running, intent, key
}

type absentOperations struct{ starts, lookups int }

func (o absentOperations) NewIntent(owner, attempt string, spec platform.OperationSpec) (platform.OperationIntent, error) {
	return platform.NewOperationIntent(owner, attempt, spec)
}
func (o absentOperations) Start(context.Context, platform.OperationIntent, platform.OperationSpec) (*platform.OwnedOperation, error) {
	return nil, nil
}
func (o absentOperations) Lookup(context.Context, platform.OperationIntent) (platform.OperationRef, error) {
	return platform.OperationRef{}, platform.ErrOperationNotFound
}
func (o absentOperations) Inspect(context.Context, platform.OperationRef) (platform.OperationObservation, error) {
	return platform.OperationObservation{}, platform.ErrOperationNotFound
}
func (o absentOperations) Stop(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error) {
	return platform.OperationStopReceipt{}, platform.ErrOperationNotFound
}
func (o absentOperations) Release(context.Context, platform.OperationRef) error { return nil }

var _ OperationOwner = absentOperations{}

func TestCancelBeforeNativeStartUsesFreshClaimAndPersistsProof(t *testing.T) {
	c, ctx, js, running, intent, key := recoveryFixture(t)
	route, _ := c.route(ctx)
	cancelled, err := js.RequestCancel(ctx, running.ID, running.Version)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := currentAttempt(cancelled)
	if !ok {
		t.Fatal("missing claimed attempt")
	}
	got, err := c.cancelBeforeNativeStart(ctx, route, cancelled, a, persistedIntent{Intent: intent}, key)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != jobs.Cancelled || got.Quiescence == nil || !got.Quiescence.NoOwnedExecution {
		t.Fatalf("predispatch cancellation was not durably confirmed: state=%s proof=%#v", got.State, got.Quiescence)
	}
}

func TestCancelRequestedRetryDoesNotRepeatTransition(t *testing.T) {
	c, ctx, js, running, _, _ := recoveryFixture(t)
	requested, err := js.RequestCancel(ctx, running.ID, running.Version)
	if err != nil {
		t.Fatal(err)
	}
	c.jobs = js
	// Reissuing the caller's current version must proceed to exact-owner
	// reconciliation; it must not issue RequestCancel a second time.
	if requested.State != jobs.CancelRequested {
		t.Fatalf("got %s", requested.State)
	}
	// The operation does not exist in this fixture, so the expected result is
	// unknown outcome rather than a false terminal cancellation.
	_, err = c.Cancel(ctx, requested.ID, requested.Version)
	if !errors.Is(err, ErrUnknownOutcome) {
		t.Fatalf("missing native operation must remain unknown after loading intent without workspace: %v", err)
	}
}

type retryReleaseOperations struct {
	ref   platform.OperationRef
	count int
}

func (o *retryReleaseOperations) NewIntent(owner, attempt string, spec platform.OperationSpec) (platform.OperationIntent, error) {
	return platform.NewOperationIntent(owner, attempt, spec)
}
func (o *retryReleaseOperations) Start(context.Context, platform.OperationIntent, platform.OperationSpec) (*platform.OwnedOperation, error) {
	return nil, nil
}
func (o *retryReleaseOperations) Lookup(context.Context, platform.OperationIntent) (platform.OperationRef, error) {
	return o.ref, nil
}
func (o *retryReleaseOperations) Inspect(context.Context, platform.OperationRef) (platform.OperationObservation, error) {
	return platform.OperationObservation{}, nil
}
func (o *retryReleaseOperations) Stop(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error) {
	return platform.OperationStopReceipt{}, nil
}
func (o *retryReleaseOperations) Release(_ context.Context, ref platform.OperationRef) error {
	o.count++
	if ref != o.ref {
		return platform.ErrOperationIdentity
	}
	if o.count == 1 {
		return errors.New("simulated crash boundary before release")
	}
	return nil
}

func TestTerminalJobRetriesExactUnitReleaseAfterRestart(t *testing.T) {
	c, ctx, js, running, intent, _ := recoveryFixture(t)
	a, _ := currentAttempt(running)
	failed, err := js.Fail(ctx, running.ID, a.ID, "test terminal evidence", running.Version)
	if err != nil {
		t.Fatal(err)
	}
	ref := platform.OperationRef{Intent: intent, Unit: "vrooli-device-studio-test.service", NativeInvocationID: "native-test-invocation"}
	ops := &retryReleaseOperations{ref: ref}
	c.ops = ops
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.releaseTerminal(ctx, route, failed); err == nil {
		t.Fatal("first release failure was hidden")
	}
	// A newly constructed coordinator has no in-memory release state. The
	// terminal ledger plus exact persisted intent is enough to retry cleanup.
	restarted := &Coordinator{jobs: js, blobs: c.blobs, roots: c.roots, ops: ops, now: time.Now}
	got, err := restarted.Reconcile(ctx, failed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != jobs.Failed || ops.count != 2 {
		t.Fatalf("terminal state/release retries = %s/%d", got.State, ops.count)
	}
}
