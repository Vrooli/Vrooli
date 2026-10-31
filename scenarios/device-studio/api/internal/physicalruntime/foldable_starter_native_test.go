package physicalruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"device-studio/internal/devices"
	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/renderjobs"
	"device-studio/internal/routedblobstore"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	"encoding/hex"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

const foldNativeLeaseID = "generic-book-fold-native-review"

func TestOptInRenderFoldableCandidateOpenHero(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E3_FOLD_NATIVE") != "1" {
		t.Skip("set DEVICE_STUDIO_E3_FOLD_NATIVE=1 for the root-owned foldable still render")
	}
	apiBin, evidence := os.Getenv("DEVICE_STUDIO_E3_FOLD_API_BIN"), filepath.Clean(os.Getenv("DEVICE_STUDIO_E3_FOLD_EVIDENCE_DIR"))
	info, err := os.Lstat(apiBin)
	if filepath.Clean(apiBin) != apiBin || !filepath.IsAbs(apiBin) || err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 256<<20 {
		t.Fatalf("API worker must be a bounded regular file: %v", err)
	}
	worker, err := os.ReadFile(apiBin)
	if err != nil || len(worker) == 0 || len(worker) > 256<<20 {
		t.Fatalf("read bounded API worker: %v", err)
	}
	if !filepath.IsAbs(evidence) || !strings.Contains(filepath.ToSlash(evidence), "/foldable-authored-candidate/native-runs/") || filepath.Base(evidence) == "native-runs" {
		t.Fatalf("evidence directory is outside the exclusive foldable run tree: %q", evidence)
	}
	if err := os.Mkdir(evidence, 0o700); err != nil {
		t.Fatalf("create exclusive evidence leaf: %v", err)
	}
	backend, err := ipadNativeBackend(os.Getenv("DEVICE_STUDIO_E3_FOLD_BACKEND"))
	if err != nil || backend != shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_OPTIX {
		t.Fatalf("fold hero requires OPTIX: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	providers := []database.SchemaProvider{database.SchemaProviderFunc(library.Schema), database.SchemaProviderFunc(devices.Schema), database.SchemaProviderFunc(scenes.Schema), database.SchemaProviderFunc(motions.Schema), database.SchemaProviderFunc(projects.Schema), database.SchemaProviderFunc(shots.Schema), database.SchemaProviderFunc(jobs.Schema)}
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(evidence, "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = database.EnsureSchemas(ctx, db.Primary(), providers...); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(c context.Context, pool *sql.DB) error { return database.EnsureSchemas(c, pool, providers...) })
	lease := 25 * time.Minute
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(evidence, "primary-data")})
	if err = db.InstallTestPool(ctx, filepath.Join(evidence, "leased.db"), foldNativeLeaseID, lease); err != nil {
		t.Fatal(err)
	}
	if err = roots.InstallTestRoots(storage.Paths{DataDir: filepath.Join(evidence, "leased-data")}, foldNativeLeaseID, lease); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ds, rs := db.LeaseStats(), roots.LeaseStats()
		re, de := roots.ClearTestRoots(foldNativeLeaseID), db.ClearTestPool(foldNativeLeaseID)
		if err := writeIPadExclusive(filepath.Join(evidence, "routing-cleanup-receipt.json"), marshalIPadJSON(map[string]any{"lease_id": foldNativeLeaseID, "database_test_requests": ds.TestPoolRequests, "database_primary_requests": ds.PrimaryDuringTestModeRequests, "file_test_writes": rs.TestRootWrites, "file_primary_writes": rs.PrimaryWritesDuringTestMode, "root_error": errorText(re), "database_error": errorText(de)})); err != nil {
			t.Error(err)
		}
		if re != nil || de != nil || ds.PrimaryDuringTestModeRequests != 0 || rs.PrimaryWritesDuringTestMode != 0 {
			t.Errorf("native review cleanup/isolation failed: roots=%v db=%v primary_db=%d primary_files=%d", re, de, ds.PrimaryDuringTestModeRequests, rs.PrimaryWritesDuringTestMode)
		}
	})
	testCtx := database.WithTestMode(ctx)
	repo := projects.NewSQLiteRepository(db, nil)
	projectService := projects.NewService(repo, nil)
	media, err := routedblobstore.New(roots, "media")
	if err != nil {
		t.Fatal(err)
	}
	mediaService := projects.NewSourceMediaService(projects.SourceMediaDeps{Projects: projectService, Repository: repo.(projects.SourceMediaRepository), Store: media})
	owners, err := New(db, roots, mediaService)
	if err != nil {
		t.Fatal(err)
	}
	state, err := owners.InitializeReviewedFoldableCandidate(testCtx)
	if err != nil {
		t.Fatalf("initialize opt-in fold candidate: %v", err)
	}
	preset, err := owners.Presets.Head(testCtx, foldPresetID)
	if err != nil {
		t.Fatal(err)
	}
	setup := preset.Setup
	stillMotion, err := owners.Motions.GetRevision(testCtx, setup.Motion.Id, setup.Motion.Revision)
	if err != nil {
		t.Fatalf("load exact still motion bound by preset: %v", err)
	}
	device, scene, foldMotion := state.Device, state.Scene, state.Motion
	if setup.Device.Sha256 != state.Device.SHA256 || setup.Scene.Sha256 != state.Scene.SHA256 || setup.Motion.Id != stillMotion.MotionID || setup.Motion.Sha256 != stillMotion.SHA256 || stillMotion.MotionID != "phone-still-identity" || device.Candidate.Geometry.SHA256 != foldAssetSHA || scene.SceneID != foldSceneID || foldMotion.MotionID != foldMotionID || setup.DisplayId != "inner-left" || len(device.Candidate.Definition.Displays) != 3 || device.Candidate.Definition.Qualification != "unqualified" || len(setup.StillProfiles) != 1 || setup.StillProfiles[0] != shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1 {
		t.Fatal("native harness does not bind the exact unqualified foldable candidate and inner display")
	}
	pl := state.Placement
	placement := ReviewedIPadPlacement{DisplayID: pl.DisplayID, DeviceSupportID: pl.DeviceSupportID, SceneSupportID: pl.SceneSupportID, SelectedCameraID: pl.CameraID, SceneSupportWorld: pl.SceneSupportWorld, SupportPivotM: pl.SupportPivotM, DeviceRootRest: pl.DeviceRootRest}
	stateForRender := IPadStarterState{Device: device, Scene: scene, Motion: stillMotion, Placement: placement}
	project, err := projectService.Create(testCtx, projects.CreateInput{Name: "Generic dual-rigid foldable candidate review"})
	if err != nil {
		t.Fatal(err)
	}
	chart := foldNativeChart(t)
	chartSum := sha256.Sum256(chart)
	screen, err := mediaService.Import(testCtx, project.ID, "foldable-asymmetric-calibration.png", int64(len(chart)), bytes.NewReader(chart))
	if err != nil {
		t.Fatal(err)
	}
	shotService := shots.NewService(shots.NewSQLiteRepository(db), projectService, mediaService, nil, nil)
	shots.SetPhysicalResolver(shotService, owners.Shots)
	snapshot := ipadStillSnapshot(stateForRender, screen.ID, placement, backend)
	snapshot.PhysicalBinding.ScreenWidthPixels, snapshot.PhysicalBinding.ScreenHeightPixels = 605, 1440
	if !proto.Equal(setup.Device, snapshot.PhysicalBinding.Device) || !proto.Equal(setup.Scene, snapshot.PhysicalBinding.Scene) || !proto.Equal(setup.Motion, snapshot.PhysicalBinding.Motion) || setup.DisplayId != snapshot.PhysicalBinding.DisplayId || setup.CameraId != snapshot.PhysicalBinding.CameraId || snapshot.PhysicalBinding.MotionSampleTick == nil || snapshot.PhysicalBinding.MotionSampleTick.Numerator != 0 {
		t.Fatal("still binding differs from saved preset or open/identity pose")
	}
	shot, revision, err := shotService.Create(testCtx, project.ID, "foldable-open-neutral-screen-hero", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	profileBytes, profile, err := rendering.ResolvePhysicalBlenderProfile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	profileHash, workerHash := sha256.Sum256(profileBytes), sha256.Sum256(worker)
	blobs, err := routedblobstore.New(roots, "renderjobs")
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := renderjobs.New(renderjobs.Dependencies{Shots: shotService, Media: mediaService, Jobs: jobs.NewService(jobs.NewSQLiteRepository(db), time.Now), Blobs: blobs, Roots: roots, Exe: func() (string, error) { return apiBin, nil }, PhysicalArtifacts: owners.Library, PhysicalProfile: func(context.Context) ([]byte, *cliv1.BlenderToolProfile, error) { return profileBytes, profile, nil }, PhysicalAdapter: rendering.PhysicalAdapterBytes})
	if err != nil {
		t.Fatal(err)
	}
	job, created, err := coordinator.Submit(testCtx, renderjobs.SubmitRequest{ProjectID: project.ID, ShotID: shot.ID, Revision: revision.Revision, IdempotencyKey: "foldable-open-native-hero", OutputProfile: renderjobs.OutputProfilePNG})
	if err != nil || !created {
		t.Fatalf("submit foldable hero: created=%v err=%v", created, err)
	}
	t.Cleanup(func() {
		cleanupIPadJob(t, coordinator, db, roots, evidence, project.ID, job.ID, hex.EncodeToString(profileHash[:]), foldNativeLeaseID)
	})
	if err := writeIPadExclusive(filepath.Join(evidence, "native-job-submission.json"), marshalIPadJSON(map[string]any{"job": job, "preset_id": preset.PresetID, "preset_sha256": preset.SHA256, "bundle_manifest_sha256": state.Bundle.ManifestSHA256, "bundle_review_sha256": state.Bundle.ReviewSHA256, "source_manifest_sha256": foldManifestSHA, "device_sha256": device.SHA256, "scene_sha256": scene.SHA256, "still_motion_sha256": stillMotion.SHA256, "fold_motion_sha256": foldMotion.SHA256, "asset_sha256": foldAssetSHA, "chart_sha256": hex.EncodeToString(chartSum[:]), "worker_sha256": hex.EncodeToString(workerHash[:]), "profile_sha256": hex.EncodeToString(profileHash[:]), "backend": "OPTIX", "output_profile": "2560x1440", "selected_display": setup.DisplayId, "qualification": "candidate_native_still_only_unqualified"})); err != nil {
		t.Fatal(err)
	}
	for job.State != jobs.Succeeded && job.State != jobs.Failed && job.State != jobs.Cancelled {
		job, err = coordinator.Advance(testCtx, job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if job.State != jobs.Succeeded && job.State != jobs.Failed && job.State != jobs.Cancelled {
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(200 * time.Millisecond):
			}
		}
	}
	if job.State != jobs.Succeeded {
		t.Fatalf("foldable hero ended %s: %+v", job.State, job.Failure)
	}
	opened, output, err := coordinator.OpenOutput(testCtx, project.ID, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(output))
	if err != nil || cfg.Width != 2560 || cfg.Height != 1440 {
		t.Fatalf("output config=%+v err=%v", cfg, err)
	}
	if _, err := png.Decode(bytes.NewReader(output)); err != nil {
		t.Fatal(err)
	}
	if err := writeIPadExclusive(filepath.Join(evidence, "foldable-open-hero.png"), output); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(output)
	if err := writeIPadExclusive(filepath.Join(evidence, "native-output-receipt.json"), marshalIPadJSON(map[string]any{"job": opened, "sha256": hex.EncodeToString(sum[:]), "bytes": len(output), "width": cfg.Width, "height": cfg.Height, "status": "candidate_native_still_only_unqualified"})); err != nil {
		t.Fatal(err)
	}
}

func foldNativeChart(t *testing.T) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 605, 1440))
	fill := func(y0, y1, x0, x1 int, shade byte) {
		band := bytes.Repeat([]byte{shade}, x1-x0)
		for y := y0; y < y1; y++ {
			row := y * img.Stride
			copy(img.Pix[row+x0:row+x1], band)
		}
	}
	fill(0, 1440, 0, 605, 20)
	fill(0, 125, 0, 605, 220)
	fill(260, 520, 55, 260, 110)
	fill(810, 1110, 350, 555, 180)
	fill(230, 1220, 295, 310, 250)
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
