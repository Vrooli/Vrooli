package renderjobs

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"device-studio/internal/jobs"
	"device-studio/internal/rendering"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

type retryFinalizer struct {
	calls int
	after func()
}

type unavailableBlobStore struct{}

func (unavailableBlobStore) Put(context.Context, string, io.Reader, string) error {
	return errors.New("blob store unavailable")
}

func (unavailableBlobStore) Get(context.Context, string) (io.ReadCloser, string, error) {
	return nil, "", errors.New("blob store unavailable")
}

func (unavailableBlobStore) Delete(context.Context, string) error {
	return errors.New("blob store unavailable")
}

func (f *retryFinalizer) prepare(context.Context, *Coordinator, routePin, jobs.Job, jobs.Attempt) (preparedAttempt, error) {
	return preparedAttempt{}, nil
}

func (f *retryFinalizer) validateCompletion(context.Context, *Coordinator, routePin, jobs.Job, jobs.Attempt, persistedIntent, string, string) (jobs.CompletionEvidence, error) {
	return jobs.CompletionEvidence{}, nil
}

func (f *retryFinalizer) finalizeTerminal(context.Context, *Coordinator, jobs.SubjectFinalization) error {
	f.calls++
	if f.after != nil {
		f.after()
	}
	if f.calls == 1 {
		return errors.New("temporary projection store failure")
	}
	return nil
}

type preparedAdapter struct{ operation preparedOperation }

func (a preparedAdapter) prepare(context.Context, *Coordinator, routePin, jobs.Job, jobs.Attempt) (preparedOperation, error) {
	return a.operation, nil
}

func (preparedAdapter) validateCompletion(context.Context, *Coordinator, routePin, jobs.Job, jobs.Attempt, persistedIntent, string, string) (jobs.CompletionEvidence, error) {
	return jobs.CompletionEvidence{}, nil
}

func (preparedAdapter) finalizeTerminal(context.Context, *Coordinator, jobs.SubjectFinalization) error {
	return nil
}

func TestTerminalSubjectProjectionRetriesBeforeMarkerRemoval(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "subject.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(jobs.Schema()); err != nil {
		t.Fatal(err)
	}
	svc := jobs.NewService(jobs.NewSQLiteRepository(db), func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) })
	in := jobs.Input{ProjectID: "p", Bindings: []jobs.ArtifactBinding{{Role: "source", ArtifactID: "m", RevisionID: "r", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}, ToolProfileDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", RendererVersion: "inspect-v1", OutputProfile: "receipt-v1", SubjectKind: jobs.SubjectSourceInspection, SubjectID: "m"}
	job, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "inspection", Input: in})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.RequestCancel(context.Background(), job.ID, job.Version); err != nil {
		t.Fatal(err)
	}
	calls := &retryFinalizer{}
	roots := filerouting.New(storage.Paths{})
	dataRoot := filepath.Join(t.TempDir(), "data")
	if err := roots.InstallTestRoots(storage.Paths{DataDir: dataRoot}, "subject-lease", time.Minute); err != nil {
		t.Fatal(err)
	}
	ctx := database.WithTestMode(context.Background())
	route := routePin{Root: dataRoot, TestMode: true}
	c := &Coordinator{jobs: svc, roots: roots, adapters: map[jobs.SubjectKind]subjectAdapter{jobs.SubjectSourceInspection: calls}}
	terminal, err := svc.Get(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.finalizeTerminal(ctx, route, terminal); err == nil {
		t.Fatal("first finalization should fail for retry test")
	}
	if _, err = svc.PendingSubjectFinalization(context.Background(), job.ID); err != nil {
		t.Fatalf("failed projection consumed durable marker: %v", err)
	}
	if err = c.finalizeTerminal(ctx, route, terminal); err != nil {
		t.Fatal(err)
	}
	if calls.calls != 2 {
		t.Fatalf("finalizer calls=%d, want 2", calls.calls)
	}
	if _, err = svc.PendingSubjectFinalization(context.Background(), job.ID); !errors.Is(err, jobs.ErrNotFound) {
		t.Fatalf("successful retry did not clear marker: %v", err)
	}
}

func TestTerminalProjectionRequiresLiveMatchingRoute(t *testing.T) {
	ctx, svc, roots, route, terminal := terminalProjectionFixture(t, "route-lease")
	adapter := &retryFinalizer{}
	c := &Coordinator{jobs: svc, roots: roots, adapters: map[jobs.SubjectKind]subjectAdapter{jobs.SubjectSourceInspection: adapter}}

	wrong := route
	wrong.Root = filepath.Join(t.TempDir(), "unrelated")
	if err := c.finalizeTerminal(ctx, wrong, terminal); err == nil {
		t.Fatal("projection proceeded for a different route")
	}
	if adapter.calls != 0 {
		t.Fatalf("wrong-route projection calls=%d", adapter.calls)
	}
	if _, err := svc.PendingSubjectFinalization(context.Background(), terminal.ID); err != nil {
		t.Fatalf("wrong-route check consumed pending marker: %v", err)
	}

	if err := roots.ClearTestRoots("route-lease"); err != nil {
		t.Fatal(err)
	}
	if err := c.finalizeTerminal(ctx, route, terminal); err == nil {
		t.Fatal("projection proceeded after routed roots expired/cleared")
	}
	if adapter.calls != 0 {
		t.Fatalf("expired-route projection calls=%d", adapter.calls)
	}
	if _, err := svc.PendingSubjectFinalization(context.Background(), terminal.ID); err != nil {
		t.Fatalf("expired-route check consumed pending marker: %v", err)
	}
}

func TestRouteLossAfterProjectionKeepsFinalizationRetryable(t *testing.T) {
	ctx, svc, roots, route, terminal := terminalProjectionFixture(t, "projection-route-loss")
	adapter := &retryFinalizer{after: func() {
		if err := roots.ClearTestRoots("projection-route-loss"); err != nil {
			t.Errorf("clear route during projection: %v", err)
		}
	}}
	c := &Coordinator{jobs: svc, roots: roots, adapters: map[jobs.SubjectKind]subjectAdapter{jobs.SubjectSourceInspection: adapter}}
	if err := c.finalizeTerminal(ctx, route, terminal); err == nil {
		t.Fatal("route loss between projection and acknowledgement was ignored")
	}
	if adapter.calls != 1 {
		t.Fatalf("projection calls=%d, want one before route loss", adapter.calls)
	}
	if _, err := svc.PendingSubjectFinalization(context.Background(), terminal.ID); err != nil {
		t.Fatalf("route loss acknowledged finalization marker: %v", err)
	}
}

func terminalProjectionFixture(t *testing.T, leaseID string) (context.Context, jobs.Service, *filerouting.RoutedRoots, routePin, jobs.Job) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "terminal.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(jobs.Schema()); err != nil {
		t.Fatal(err)
	}
	svc := jobs.NewService(jobs.NewSQLiteRepository(db), func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) })
	in := jobs.Input{ProjectID: "p", Bindings: []jobs.ArtifactBinding{{Role: "source", ArtifactID: "m", RevisionID: "r", SHA256: strings.Repeat("a", 64)}}, ToolProfileDigest: strings.Repeat("b", 64), RendererVersion: "inspect-v1", OutputProfile: "receipt-v1", SubjectKind: jobs.SubjectSourceInspection, SubjectID: "m"}
	job, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: leaseID, Input: in})
	if err != nil {
		t.Fatal(err)
	}
	terminal, err := svc.RequestCancel(context.Background(), job.ID, job.Version)
	if err != nil || terminal.State != jobs.Cancelled {
		t.Fatalf("queue cancel: state=%q err=%v", terminal.State, err)
	}
	root := filepath.Join(t.TempDir(), "data")
	roots := filerouting.New(storage.Paths{})
	if err := roots.InstallTestRoots(storage.Paths{DataDir: root}, leaseID, time.Minute); err != nil {
		t.Fatal(err)
	}
	return database.WithTestMode(context.Background()), svc, roots, routePin{Root: root, TestMode: true}, terminal
}

func TestSubjectAdapterDispatchIsExplicitAndLegacyDefaultsToStill(t *testing.T) {
	video := &retryFinalizer{}
	c := &Coordinator{adapters: map[jobs.SubjectKind]subjectAdapter{jobs.SubjectStillRender: stillSubjectAdapter{}, jobs.SubjectVideoRender: video}}
	if _, err := c.adapterFor(jobs.Job{}); err != nil {
		t.Fatalf("legacy job did not retain still adapter: %v", err)
	}
	if adapter, err := c.adapterFor(jobs.Job{Input: jobs.Input{SubjectKind: jobs.SubjectVideoRender}}); err != nil || adapter != video {
		t.Fatalf("typed subject selected wrong adapter: adapter=%T err=%v", adapter, err)
	}
	if _, err := c.adapterFor(jobs.Job{Input: jobs.Input{SubjectKind: jobs.SubjectKind("unknown.v9")}}); err == nil {
		t.Fatal("unknown kind fell through to still")
	}
}

func TestPreparedBlobInputStreamsToWorkspaceWithBindingAndDigestChecks(t *testing.T) {
	dataRoot := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dataRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	roots := filerouting.New(storage.Paths{})
	if err := roots.InstallTestRoots(storage.Paths{DataDir: dataRoot}, "stage-lease", time.Minute); err != nil {
		t.Fatal(err)
	}
	ctx := database.WithTestMode(context.Background())
	route := routePin{Root: dataRoot, TestMode: true}
	blobs := blobstore.NewMemoryBlobStore()
	payload := strings.Repeat("video-frame-schedule\n", 1024)
	key, sha := "jobs/subject/input", digestBytes([]byte(payload))
	if err := blobs.Put(ctx, key, strings.NewReader(payload), "application/octet-stream"); err != nil {
		t.Fatal(err)
	}
	c := &Coordinator{roots: roots, blobs: blobs}
	workspace, err := attemptWorkspace(dataRoot, "job-safe", "attempt-safe")
	if err != nil {
		t.Fatal(err)
	}
	if err = c.ensureDirectories(ctx, route, filepath.Dir(workspace)); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	job := jobs.Job{Input: jobs.Input{Bindings: []jobs.ArtifactBinding{{ArtifactID: key, SHA256: sha}}}}
	if err = c.stagePreparedFile(ctx, route, workspace, job, stagedFile{Name: "schedule.json", BlobKey: key, SHA256: sha, Size: int64(len(payload))}, maxWorkerInputBytes); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(workspace, "schedule.json"))
	if err != nil || string(got) != payload {
		t.Fatalf("streamed input mismatch: size=%d err=%v", len(got), err)
	}
	if err = c.stagePreparedFile(ctx, route, workspace, job, stagedFile{Name: "wrong.json", BlobKey: key, SHA256: strings.Repeat("0", 64), Size: int64(len(payload))}, maxWorkerInputBytes); err == nil {
		t.Fatal("unbound blob digest was accepted")
	}
}

func TestPreparedBoundBlobRejectsCorruptTruncatedAndOversizedBytes(t *testing.T) {
	dataRoot := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dataRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	roots := filerouting.New(storage.Paths{})
	if err := roots.InstallTestRoots(storage.Paths{DataDir: dataRoot}, "corrupt-stage-lease", time.Minute); err != nil {
		t.Fatal(err)
	}
	ctx := database.WithTestMode(context.Background())
	route := routePin{Root: dataRoot, TestMode: true}
	blobs := blobstore.NewMemoryBlobStore()
	c := &Coordinator{roots: roots, blobs: blobs}
	workspace, err := attemptWorkspace(dataRoot, "job-corrupt", "attempt-corrupt")
	if err != nil {
		t.Fatal(err)
	}
	if err = c.ensureDirectories(ctx, route, filepath.Dir(workspace)); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	want := []byte("the exact immutable bytes expected by this subject")
	wantSHA := digestBytes(want)
	for name, actual := range map[string][]byte{
		"corrupt":   []byte("the exact immutable bytes expected by this subjecX"),
		"truncated": want[:len(want)-1],
		"oversized": append(append([]byte(nil), want...), '!'),
	} {
		t.Run(name, func(t *testing.T) {
			key := "jobs/subject/" + name
			if err := blobs.Put(ctx, key, strings.NewReader(string(actual)), "application/octet-stream"); err != nil {
				t.Fatal(err)
			}
			job := jobs.Job{Input: jobs.Input{Bindings: []jobs.ArtifactBinding{{ArtifactID: key, SHA256: wantSHA}}}}
			err := c.stagePreparedFile(ctx, route, workspace, job, stagedFile{Name: name + ".bin", BlobKey: key, SHA256: wantSHA, Size: int64(len(want))}, maxWorkerInputBytes)
			if err == nil {
				t.Fatalf("accepted %s bound blob bytes", name)
			}
		})
	}
}

func TestDispatchRejectsAdapterSubjectModeMismatchBeforeClaim(t *testing.T) {
	adapter := preparedAdapter{operation: preparedOperation{Mode: operationStillRender, ManifestBytes: []byte(`{"subject":"video"}`)}}
	c := &Coordinator{adapters: map[jobs.SubjectKind]subjectAdapter{jobs.SubjectVideoRender: adapter}}
	job := jobs.Job{ID: "video-job", Input: jobs.Input{
		SubjectKind: jobs.SubjectVideoRender,
		Bindings:    []jobs.ArtifactBinding{{Role: "worker_executable", ArtifactID: "worker", SHA256: strings.Repeat("a", 64)}},
	}, Attempts: []jobs.Attempt{{ID: "attempt", Number: 1, State: jobs.AttemptQueued}}}
	_, err := c.dispatch(context.Background(), routePin{Root: t.TempDir(), TestMode: true}, job)
	if err == nil || !strings.Contains(err.Error(), "mismatched operation mode") {
		t.Fatalf("mismatched subject mode was not rejected by dispatch: %v", err)
	}
}

func TestSubjectOperationModesAreAllowlisted(t *testing.T) {
	ownerHome := filepath.Join(t.TempDir(), "operation-mode-home")
	cliPath := filepath.Join(ownerHome, ".vrooli", "bin", "vrooli")
	if err := os.MkdirAll(filepath.Dir(cliPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cliPath, []byte("vrooli"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", ownerHome)
	for _, tc := range []struct {
		mode    operationMode
		command string
		wall    time.Duration
	}{{operationStillRender, "render-worker", operationWallTime}, {operationSourceInspect, "inspect-worker", 5 * time.Minute}, {operationVideoRender, "video-render-worker", 5 * time.Minute}, {operationPhysicalRender, "physical-render-worker", time.Duration(rendering.PhysicalStillCPUWallTimeSeconds) * time.Second}, {operationPhysicalPreview, "physical-render-worker", time.Duration(rendering.PhysicalPreviewWallTimeSeconds) * time.Second}, {operationPhysicalMotion, "physical-render-worker", time.Duration(rendering.PhysicalMotionWallTimeSeconds) * time.Second}, {operationPhysicalMotionV2, "physical-render-worker", time.Duration(rendering.PhysicalMotionV2WallTimeSeconds) * time.Second}, {operationPhysicalRecording, "physical-render-worker", time.Duration(rendering.PhysicalRecordingWallTimeSeconds) * time.Second}} {
		spec, err := operationSpecFor(tc.mode, "/worker", "/manifest", "/workspace")
		if err != nil || len(spec.Args) != 3 || spec.Args[0] != tc.command {
			t.Fatalf("mode %q: spec=%+v err=%v", tc.mode, spec, err)
		}
		if tc.mode == operationVideoRender && spec.Limits.WallTime != 5*time.Minute {
			t.Fatalf("video wall limit = %s, want 5m", spec.Limits.WallTime)
		}
		if isPhysicalOperation(tc.mode) && (spec.Limits.MemoryMaxBytes != int64(rendering.PhysicalOperationMemoryMaxBytes) || spec.Limits.TasksMax != int(rendering.PhysicalOperationTasksMax) || spec.Limits.CPUWeight != int(rendering.PhysicalOperationCPUWeight)) {
			t.Fatalf("physical operation limits = %+v, do not match closed profile", spec.Limits)
		}
		if spec.Limits.WallTime != tc.wall {
			t.Fatalf("mode %q wall limit=%s, want %s", tc.mode, spec.Limits.WallTime, tc.wall)
		}
	}
	if _, err := operationSpecFor(operationMode("user-command"), "/worker", "/manifest", "/workspace"); err == nil {
		t.Fatal("arbitrary worker command was accepted")
	}
}

func TestPhysicalJobModeIsBoundBySubjectAndImmutableOutputProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		want    operationMode
		invalid bool
	}{
		{"still", OutputProfilePNG, operationPhysicalRender, false},
		{"preview", OutputProfilePhysicalPreviewPNG, operationPhysicalPreview, false},
		{"motion-v1", OutputProfilePhysicalMotionMP4, operationPhysicalMotion, false},
		{"motion-v2", OutputProfilePhysicalMotionMP4V2, operationPhysicalMotionV2, false},
		{"recording-v1", OutputProfilePhysicalRecordingMP4, operationPhysicalRecording, false},
		{"unknown", "user-command", "", true},
	}
	if physicalOutputKind(OutputProfilePhysicalPreviewPNG) != "physical_preview_png" || physicalOutputKind(OutputProfilePNG) != "physical_still_png" {
		t.Fatal("physical preview and final still output kinds are not distinct")
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := operationModeForJob(jobs.Input{SubjectKind: jobs.SubjectPhysicalRender, OutputProfile: tt.profile})
			if tt.invalid {
				if err == nil {
					t.Fatalf("unsupported physical output profile mapped to %q", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("mode=%q err=%v, want %q", got, err, tt.want)
			}
		})
	}
	for _, profile := range []string{OutputProfilePhysicalMotionMP4, OutputProfilePhysicalMotionMP4V2, OutputProfilePhysicalRecordingMP4} {
		if _, err := operationModeForJob(jobs.Input{SubjectKind: jobs.SubjectStillRender, OutputProfile: profile}); err == nil {
			t.Fatalf("physical motion profile %q was accepted for a legacy still subject", profile)
		}
	}
}

func TestPhysicalMotionGenericRequestReplaysEitherPinnedProfile(t *testing.T) {
	for _, profile := range []string{OutputProfilePhysicalMotionMP4, OutputProfilePhysicalMotionMP4V2, OutputProfilePhysicalRecordingMP4} {
		input := jobs.Input{SubjectKind: jobs.SubjectPhysicalRender, OutputProfile: profile}
		if !idempotentOutputProfileMatches(input, OutputProfileMP4) {
			t.Errorf("generic MP4 request did not match pinned physical profile %q", profile)
		}
	}
	if idempotentOutputProfileMatches(jobs.Input{SubjectKind: jobs.SubjectVideoRender, OutputProfile: OutputProfilePhysicalMotionMP4V2}, OutputProfileMP4) {
		t.Fatal("generic MP4 request matched a physical-only profile on another subject")
	}
	if idempotentOutputProfileMatches(jobs.Input{SubjectKind: jobs.SubjectPhysicalRender, OutputProfile: OutputProfilePhysicalMotionMP4}, OutputProfilePNG) {
		t.Fatal("PNG request matched an immutable physical motion job")
	}
}

func TestPhysicalMotionSubmitUsesGenericMP4Format(t *testing.T) {
	request := SubmitRequest{ProjectID: "project", ShotID: "shot", Revision: 2, IdempotencyKey: "motion-format", OutputProfile: OutputProfileMP4}
	if err := validateSubmitRequest(request); err != nil {
		t.Fatalf("generic MP4 output request rejected: %v", err)
	}
	request.OutputProfile = OutputProfilePhysicalMotionMP4
	if err := validateSubmitRequest(request); err == nil {
		t.Fatal("internal v1 render profile was accepted as a caller-selected output format")
	}
}

func TestPhysicalWorkerUsesRuntimeOwnerHomeAndPrivateScratch(t *testing.T) {
	ownerHome := filepath.Join(t.TempDir(), "owner-home")
	workspace := filepath.Join(t.TempDir(), "attempt")
	cliPath := filepath.Join(ownerHome, ".vrooli", "bin", "vrooli")
	if err := os.MkdirAll(filepath.Dir(cliPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cliPath, []byte("vrooli"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", ownerHome)
	spec, err := operationSpecFor(operationPhysicalRender, "/worker", filepath.Join(workspace, rendering.PhysicalManifestName), workspace)
	if err != nil {
		t.Fatal(err)
	}
	wantHome := "HOME=" + ownerHome
	privateTMP := "TMPDIR=" + filepath.Join(workspace, "tmp")
	privateConfig := "XDG_CONFIG_HOME=" + filepath.Join(workspace, "config")
	if !containsEnv(spec.Env, wantHome) || !containsEnv(spec.Env, "VROOLI_BIN="+cliPath) || !containsEnv(spec.Env, privateTMP) || !containsEnv(spec.Env, privateConfig) {
		t.Fatalf("physical worker environment must use owner home for governed CLI lookup and private scratch: %v", spec.Env)
	}
}

func containsEnv(env []string, want string) bool {
	for _, value := range env {
		if value == want {
			return true
		}
	}
	return false
}

func TestKnownTerminalProjectionRetriesIndependentlyOfNativeRelease(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "release-independent.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(jobs.Schema()); err != nil {
		t.Fatal(err)
	}
	svc := jobs.NewService(jobs.NewSQLiteRepository(db), func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) })
	in := jobs.Input{ProjectID: "p", Bindings: []jobs.ArtifactBinding{{Role: "source", ArtifactID: "m", RevisionID: "r", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}, ToolProfileDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", RendererVersion: "inspect-v1", OutputProfile: "receipt-v1", SubjectKind: jobs.SubjectSourceInspection, SubjectID: "m"}
	job, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "release-independent", Input: in})
	if err != nil {
		t.Fatal(err)
	}
	a := job.Attempts[0]
	running, err := svc.Start(context.Background(), job.ID, jobs.ExecutionClaimInput{AttemptID: a.ID, AttemptNumber: a.Number, OwnerID: "test-owner", ScopeID: "test-scope", ArtifactID: "claim", SHA256: "1212121212121212121212121212121212121212121212121212121212121212"}, job.Version)
	if err != nil {
		t.Fatal(err)
	}
	terminal, err := svc.Fail(context.Background(), job.ID, a.ID, "known rejection", running.Version)
	if err != nil {
		t.Fatal(err)
	}
	dataRoot := filepath.Join(t.TempDir(), "data")
	if err = os.MkdirAll(dataRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	roots := filerouting.New(storage.Paths{})
	if err = roots.InstallTestRoots(storage.Paths{DataDir: dataRoot}, "release-lease", time.Minute); err != nil {
		t.Fatal(err)
	}
	ctx := database.WithTestMode(context.Background())
	route := routePin{Root: dataRoot, TestMode: true}
	adapter := &retryFinalizer{}
	c := &Coordinator{jobs: svc, roots: roots, blobs: unavailableBlobStore{}, adapters: map[jobs.SubjectKind]subjectAdapter{jobs.SubjectSourceInspection: adapter}}
	if _, err = c.releaseTerminal(ctx, route, terminal); err == nil {
		t.Fatal("release error should remain visible")
	}
	if adapter.calls != 1 {
		t.Fatalf("projection was blocked by release failure; calls=%d", adapter.calls)
	}
	if _, err = svc.PendingSubjectFinalization(context.Background(), job.ID); err != nil {
		t.Fatalf("retryable projection marker lost after independent release failure: %v", err)
	}
	if _, err = c.releaseTerminal(ctx, route, terminal); err == nil {
		t.Fatal("unresolved release should continue to be reported")
	}
	if adapter.calls != 2 {
		t.Fatalf("terminal projection retry count=%d", adapter.calls)
	}
	if _, err = svc.PendingSubjectFinalization(context.Background(), job.ID); !errors.Is(err, jobs.ErrNotFound) {
		t.Fatalf("successful projection retry did not clear marker: %v", err)
	}
}
