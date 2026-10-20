package renderjobs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"device-studio/internal/jobs"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	_ "modernc.org/sqlite"
)

type integrationShot struct{ resolved shots.ResolvedShot }

func (s integrationShot) ResolveForRender(_ context.Context, id string, revision int64) (shots.ResolvedShot, error) {
	if id != s.resolved.Shot.ID || revision != s.resolved.Revision.Revision {
		return shots.ResolvedShot{}, errors.New("unexpected shot revision")
	}
	return s.resolved, nil
}

type integrationMedia struct {
	values map[string]projects.SourceMedia
	data   map[string][]byte
}

func (m *integrationMedia) Import(context.Context, string, string, int64, io.Reader) (projects.SourceMedia, error) {
	return projects.SourceMedia{}, errors.New("unused")
}
func (m *integrationMedia) Get(_ context.Context, projectID, id string) (projects.SourceMedia, error) {
	v := m.values[id]
	if v.ProjectID != projectID {
		return projects.SourceMedia{}, errors.New("wrong project")
	}
	return v, nil
}
func (m *integrationMedia) Open(_ context.Context, projectID, id string) (projects.SourceMedia, io.ReadCloser, error) {
	v, err := m.Get(context.Background(), projectID, id)
	if err != nil {
		return v, nil, err
	}
	return v, io.NopCloser(bytes.NewReader(m.data[id])), nil
}
func (m *integrationMedia) List(context.Context, string, int32, string) ([]projects.SourceMedia, string, error) {
	return nil, "", nil
}

type integrationOps struct {
	mu           sync.Mutex
	starts       int
	lookups      int
	stagedWorker []byte
	intent       platform.OperationIntent
	ref          platform.OperationRef
	startErr     error
	terminal     bool
	releaseCount int
}

func (o *integrationOps) NewIntent(owner, attempt string, spec platform.OperationSpec) (platform.OperationIntent, error) {
	return platform.NewOperationIntent(owner, attempt, spec)
}
func (o *integrationOps) Start(ctx context.Context, intent platform.OperationIntent, spec platform.OperationSpec) (*platform.OwnedOperation, error) {
	o.mu.Lock()
	o.starts++
	o.intent = intent
	o.ref = platform.OperationRef{Intent: intent, Unit: "device-studio-test.service", NativeInvocationID: "native-invocation"}
	startErr := o.startErr
	o.mu.Unlock()
	data, err := os.ReadFile(spec.Path)
	if err != nil {
		return nil, err
	}
	o.mu.Lock()
	o.stagedWorker = append([]byte(nil), data...)
	o.mu.Unlock()
	if startErr != nil {
		return nil, startErr
	}
	if err := rendering.RunManifest(ctx, filepath.Join(spec.Dir, rendering.ManifestName)); err != nil {
		return nil, err
	}
	o.mu.Lock()
	o.terminal = true
	o.mu.Unlock()
	return nil, nil
}
func (o *integrationOps) Lookup(_ context.Context, intent platform.OperationIntent) (platform.OperationRef, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.lookups++
	if intent != o.intent || !o.terminal {
		return platform.OperationRef{}, platform.ErrOperationNotFound
	}
	return o.ref, nil
}
func (o *integrationOps) Inspect(_ context.Context, ref platform.OperationRef) (platform.OperationObservation, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return platform.OperationObservation{Ref: ref, State: platform.OperationStoppedOutcomeKnown, Result: "success", ExecMainCode: 1, ExecMainStatus: 0, ExitKnown: true}, nil
}
func (o *integrationOps) Stop(_ context.Context, ref platform.OperationRef) (platform.OperationStopReceipt, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return platform.OperationStopReceipt{Ref: ref, NativeQuiescent: true}, nil
}
func (o *integrationOps) Release(_ context.Context, ref platform.OperationRef) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if ref != o.ref {
		return platform.ErrOperationIdentity
	}
	o.releaseCount++
	return nil
}

type cancelAfterClaim struct{ jobs.Service }

func (s cancelAfterClaim) Start(ctx context.Context, id string, claim jobs.ExecutionClaimInput, version int64) (jobs.Job, error) {
	j, err := s.Service.Start(ctx, id, claim, version)
	if err != nil {
		return j, err
	}
	return s.Service.RequestCancel(ctx, id, j.Version)
}

type cancelBeforeComplete struct{ jobs.Service }

func (s cancelBeforeComplete) Complete(ctx context.Context, id, attempt string, evidence jobs.CompletionEvidence, version int64) (jobs.Job, error) {
	if _, err := s.Service.RequestCancel(ctx, id, version); err != nil {
		return jobs.Job{}, err
	}
	return s.Service.Complete(ctx, id, attempt, evidence, version)
}

func integrationCoordinator(t *testing.T, ops *integrationOps, cancelAtClaim bool) (*Coordinator, context.Context, string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "jobs.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(jobs.Schema()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	service := jobs.NewService(jobs.NewSQLiteRepository(db), time.Now)
	if cancelAtClaim {
		service = cancelAfterClaim{service}
	}
	primary, testRoot := t.TempDir(), t.TempDir()
	roots := filerouting.New(storage.Paths{DataDir: primary})
	if err := roots.InstallTestRoots(storage.Paths{DataDir: testRoot}, "integration-lease", time.Hour); err != nil {
		t.Fatal(err)
	}
	ctx := database.WithTestMode(context.Background())
	workerPath := filepath.Join(t.TempDir(), "device-studio-api")
	if err := os.WriteFile(workerPath, []byte("worker-v1"), 0o700); err != nil {
		t.Fatal(err)
	}
	media := newIntegrationMedia(t)
	resolved := integrationResolved(media)
	c, err := New(Dependencies{Shots: integrationShot{resolved}, Media: media, Jobs: service, Blobs: blobstore.NewMemoryBlobStore(), Roots: roots, Ops: ops, Exe: func() (string, error) { return workerPath, nil }})
	if err != nil {
		t.Fatal(err)
	}
	return c, ctx, workerPath
}

func newIntegrationMedia(t *testing.T) *integrationMedia {
	t.Helper()
	m := &integrationMedia{values: map[string]projects.SourceMedia{}, data: map[string][]byte{}}
	for id, gray := range map[string]bool{"plate": false, "screen": false, "active": true} {
		var img image.Image
		if gray {
			g := image.NewGray(image.Rect(0, 0, 4, 4))
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					g.SetGray(x, y, color.Gray{Y: 255})
				}
			}
			img = g
		} else {
			rgba := image.NewRGBA(image.Rect(0, 0, 4, 4))
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					rgba.Set(x, y, color.RGBA{R: uint8(20 + x*30), G: uint8(30 + y*25), B: 100, A: 255})
				}
			}
			img = rgba
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatal(err)
		}
		data := buf.Bytes()
		sum := sha256.Sum256(data)
		digest := hex.EncodeToString(sum[:])
		model := "rgba"
		if gray {
			model = "grayscale"
		}
		m.data[id] = data
		m.values[id] = projects.SourceMedia{ID: id, ProjectID: "project", SHA256: digest, SizeBytes: int64(len(data)), MediaType: "image/png", Width: 4, Height: 4, ColorModel: model, ColorProfileStatus: "srgb"}
	}
	return m
}

func integrationResolved(m *integrationMedia) shots.ResolvedShot {
	media := func(id string) *shotsv1.ResolvedSourceMedia {
		v := m.values[id]
		return &shotsv1.ResolvedSourceMedia{Id: id, Sha256: v.SHA256, SizeBytes: v.SizeBytes, MediaType: v.MediaType, WidthPixels: v.Width, HeightPixels: v.Height}
	}
	snapshot := &shotsv1.ShotSnapshot{Scene: &shotsv1.PhotographicScene{PlateMediaId: "plate", ActiveMaskMediaId: "active", DisplayWidthPixels: 4, DisplayHeightPixels: 4, CoordinateConvention: shotsv1.CoordinateConvention_COORDINATE_CONVENTION_PLATE_RASTER_EDGE_PIXELS_V1, PlateQuad: &shotsv1.PlateQuad{TopLeft: &shotsv1.ShotPoint{X: 0, Y: 0}, TopRight: &shotsv1.ShotPoint{X: 4, Y: 0}, BottomRight: &shotsv1.ShotPoint{X: 4, Y: 4}, BottomLeft: &shotsv1.ShotPoint{X: 0, Y: 4}}}, Presentation: &shotsv1.ScreenPresentation{SourceMediaId: "screen", CaptureArea: shotsv1.CaptureArea_CAPTURE_AREA_APP_ONLY, Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN, FitMode: shotsv1.ScreenFitMode_SCREEN_FIT_MODE_CONTAIN}, MotionMode: shotsv1.MotionMode_MOTION_MODE_FIXED}
	return shots.ResolvedShot{ProjectID: "project", Shot: shots.Shot{ID: "shot", ProjectID: "project"}, Revision: shots.Revision{ShotID: "shot", Revision: 1, Snapshot: snapshot, RecipeSHA256: digestBytes([]byte("recipe")), CompositorProfile: rendering.CompositorProfile, RenderEligible: true, Plate: media("plate"), ActiveMask: media("active"), Screen: media("screen")}}
}

func submitIntegrationJob(t *testing.T, c *Coordinator, ctx context.Context) jobs.Job {
	t.Helper()
	j, created, err := c.Submit(ctx, SubmitRequest{ProjectID: "project", ShotID: "shot", Revision: 1, IdempotencyKey: "integration"})
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected a new job")
	}
	return j
}

func TestDispatchUsesPinnedExecutableAndNeverRedispatchesAmbiguousStart(t *testing.T) {
	ops := &integrationOps{startErr: errors.New("injected ambiguous start")}
	c, ctx, workerPath := integrationCoordinator(t, ops, false)
	j := submitIntegrationJob(t, c, ctx)
	if err := os.WriteFile(workerPath, []byte("worker-v2"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Advance(ctx, j.ID); err == nil {
		t.Fatal("expected ambiguous native start")
	}
	ops.mu.Lock()
	starts := ops.starts
	staged := string(ops.stagedWorker)
	ops.mu.Unlock()
	if starts != 1 || staged != "worker-v1" {
		t.Fatalf("dispatch count/staged executable=%d/%q", starts, staged)
	}
	if _, err := c.Advance(ctx, j.ID); !errors.Is(err, ErrUnknownOutcome) {
		t.Fatalf("ambiguous operation should require reconciliation, got %v", err)
	}
	ops.mu.Lock()
	defer ops.mu.Unlock()
	if ops.starts != 1 {
		t.Fatalf("ambiguous operation redispatched %d times", ops.starts)
	}
}

func TestTestRoutedJobCannotFallBackToPrimaryStorageRoute(t *testing.T) {
	ops := &integrationOps{startErr: errors.New("must not dispatch")}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	j := submitIntegrationJob(t, c, ctx)
	if _, err := c.Advance(context.Background(), j.ID); !errors.Is(err, ErrRouteMismatch) {
		t.Fatalf("lost test route must fail closed, got %v", err)
	}
	ops.mu.Lock()
	starts := ops.starts
	ops.mu.Unlock()
	if starts != 0 {
		t.Fatalf("wrong route dispatched %d native operations", starts)
	}
	if got := c.roots.LeaseStats().PrimaryWritesDuringTestMode; got != 0 {
		t.Fatalf("writes escaped to primary during test route: %d", got)
	}
}

func TestCancellationDuringStagingWinsBeforeNativeStart(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, true)
	j := submitIntegrationJob(t, c, ctx)
	got, err := c.Advance(ctx, j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != jobs.Cancelled || got.Quiescence == nil || !got.Quiescence.NoOwnedExecution {
		t.Fatalf("staging cancellation did not persist predispatch proof: %#v", got)
	}
	ops.mu.Lock()
	defer ops.mu.Unlock()
	if ops.starts != 0 {
		t.Fatalf("native start raced a persisted cancellation: %d", ops.starts)
	}
}

func TestConcurrentAdvanceHasOneWorkspaceAndOneNativeDispatchWinner(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	j := submitIntegrationJob(t, c, ctx)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := c.Advance(ctx, j.ID); results <- err }()
	}
	wg.Wait()
	close(results)
	// Exactly one dispatch succeeds; the CAS loser cannot alter the winner's
	// staged workspace or dispatch a second native operation.
	var successes int
	for err := range results {
		if err == nil {
			successes++
		}
	}
	ops.mu.Lock()
	starts := ops.starts
	ops.mu.Unlock()
	if successes == 0 || successes > 2 || starts != 1 {
		t.Fatalf("dispatch successes/native starts = %d/%d", successes, starts)
	}
	completed, err := c.Reconcile(ctx, j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != jobs.Succeeded {
		t.Fatalf("winner did not reconcile: %s", completed.State)
	}
}

func TestCancellationWinningCompletionRaceRetainsUnitUntilQuiescence(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	j := submitIntegrationJob(t, c, ctx)
	if _, err := c.Advance(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	c.jobs = cancelBeforeComplete{c.jobs}
	if _, err := c.Reconcile(ctx, j.ID); err == nil {
		t.Fatal("completion should lose the serialized cancellation race")
	}
	requested, err := c.jobs.Get(ctx, j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if requested.State != jobs.CancelRequested {
		t.Fatalf("cancel did not win: %s", requested.State)
	}
	ops.mu.Lock()
	releasedBeforeProof := ops.releaseCount
	ops.mu.Unlock()
	if releasedBeforeProof != 0 {
		t.Fatal("native unit was released before cancellation quiescence was persisted")
	}
	cancelled, err := c.Cancel(ctx, j.ID, requested.Version)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.State != jobs.Cancelled || cancelled.Quiescence == nil || !cancelled.Quiescence.NoOwnedExecution {
		t.Fatalf("cancellation proof missing: %#v", cancelled)
	}
	ops.mu.Lock()
	defer ops.mu.Unlock()
	if ops.releaseCount != 1 {
		t.Fatalf("exact unit releases after proof = %d", ops.releaseCount)
	}
}

func TestSuccessPublishesVerifiedPNGAndTamperFailsAfterNativeQuiescence(t *testing.T) {
	for _, tamper := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "tampered-output"}[tamper], func(t *testing.T) {
			ops := &integrationOps{}
			c, ctx, _ := integrationCoordinator(t, ops, false)
			j := submitIntegrationJob(t, c, ctx)
			if _, err := c.Advance(ctx, j.ID); err != nil {
				t.Fatal(err)
			}
			workspace, _ := attemptWorkspace(func() string { r, _ := c.route(ctx); return r.Root }(), j.ID, j.Attempts[0].ID)
			if tamper {
				if err := os.WriteFile(filepath.Join(workspace, rendering.OutputName), []byte("changed"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := c.Reconcile(ctx, j.ID)
			if tamper {
				if err == nil || got.State != jobs.Failed {
					t.Fatalf("tampered artifact should fail only after native terminal proof; state=%s err=%v", got.State, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.State != jobs.Succeeded || len(got.Attempts[0].Completion.Outputs) != 1 {
				t.Fatalf("verified output not completed: %#v", got)
			}
		})
	}
}
