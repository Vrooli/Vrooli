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
	"image/color"
	"image/draw"
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

const watchNativeReviewLeaseID = "series7-neutral-watch-native-review"

func TestOptInRenderWatchCandidateSupportedHero(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E4_WATCH_NATIVE") != "1" {
		t.Skip("set DEVICE_STUDIO_E4_WATCH_NATIVE=1 to run the owned watch still render")
	}
	apiBin := os.Getenv("DEVICE_STUDIO_E4_WATCH_API_BIN")
	if filepath.Clean(apiBin) != apiBin || !filepath.IsAbs(apiBin) {
		t.Fatalf("API binary path must be absolute and clean: %q", apiBin)
	}
	workerInfo, err := os.Lstat(apiBin)
	if err != nil || !workerInfo.Mode().IsRegular() || workerInfo.Mode()&os.ModeSymlink != 0 || workerInfo.Size() <= 0 || workerInfo.Size() > 256<<20 {
		t.Fatalf("API worker is not a bounded regular file: %v", err)
	}
	worker, err := os.ReadFile(apiBin)
	if err != nil || len(worker) == 0 || len(worker) > 256<<20 {
		t.Fatalf("read pinned API worker: %v", err)
	}
	evidence := filepath.Clean(os.Getenv("DEVICE_STUDIO_E4_WATCH_EVIDENCE_DIR"))
	if !filepath.IsAbs(evidence) || !strings.Contains(filepath.ToSlash(evidence), "/watch-series7-candidate-admission-20261021/native-runs/") || filepath.Base(evidence) == "native-runs" {
		t.Fatalf("evidence path is outside the admitted exclusive watch run tree: %q", evidence)
	}
	if err := os.Mkdir(evidence, 0o700); err != nil {
		t.Fatalf("create exclusive evidence directory: %v", err)
	}
	backend, err := ipadNativeBackend(os.Getenv("DEVICE_STUDIO_E4_WATCH_BACKEND"))
	if err != nil || backend != shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_OPTIX {
		t.Fatalf("watch hero requires the explicitly admitted OPTIX backend: %v", err)
	}
	const deadline = 11 * time.Minute
	const leaseTTL = 25 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	providers := []database.SchemaProvider{
		database.SchemaProviderFunc(library.Schema), database.SchemaProviderFunc(devices.Schema),
		database.SchemaProviderFunc(scenes.Schema), database.SchemaProviderFunc(motions.Schema),
		database.SchemaProviderFunc(projects.Schema), database.SchemaProviderFunc(shots.Schema), database.SchemaProviderFunc(jobs.Schema),
	}
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(evidence, "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.EnsureSchemas(ctx, db.Primary(), providers...); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(c context.Context, pool *sql.DB) error { return database.EnsureSchemas(c, pool, providers...) })
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(evidence, "primary-data")})
	if err := db.InstallTestPool(ctx, filepath.Join(evidence, "leased.db"), watchNativeReviewLeaseID, leaseTTL); err != nil {
		t.Fatal(err)
	}
	if err := roots.InstallTestRoots(storage.Paths{DataDir: filepath.Join(evidence, "leased-data")}, watchNativeReviewLeaseID, leaseTTL); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ds, rs := db.LeaseStats(), roots.LeaseStats()
		re, de := roots.ClearTestRoots(watchNativeReviewLeaseID), db.ClearTestPool(watchNativeReviewLeaseID)
		receipt := marshalIPadJSON(map[string]any{"lease_id": watchNativeReviewLeaseID, "database_test_requests": ds.TestPoolRequests, "database_primary_requests_during_test_mode": ds.PrimaryDuringTestModeRequests, "file_test_writes": rs.TestRootWrites, "file_primary_writes_during_test_mode": rs.PrimaryWritesDuringTestMode, "file_roots_clear_error": errorText(re), "database_clear_error": errorText(de)})
		if err := writeIPadExclusive(filepath.Join(evidence, "routing-cleanup-receipt.json"), receipt); err != nil {
			t.Errorf("retain route cleanup receipt: %v", err)
		}
		if re != nil || de != nil {
			t.Errorf("clear isolated watch routes: roots=%v db=%v", re, de)
		}
		if ds.PrimaryDuringTestModeRequests != 0 || rs.PrimaryWritesDuringTestMode != 0 {
			t.Errorf("routed test escaped: primary database requests=%d primary file writes=%d", ds.PrimaryDuringTestModeRequests, rs.PrimaryWritesDuringTestMode)
		}
	})
	testCtx := database.WithTestMode(ctx)
	repo := projects.NewSQLiteRepository(db, nil)
	projectService := projects.NewService(repo, nil)
	mediaStore, err := routedblobstore.New(roots, "media")
	if err != nil {
		t.Fatal(err)
	}
	mediaService := projects.NewSourceMediaService(projects.SourceMediaDeps{Projects: projectService, Repository: repo.(projects.SourceMediaRepository), Store: mediaStore})
	owners, err := New(db, roots, mediaService)
	if err != nil {
		t.Fatal(err)
	}
	state, err := owners.InitializeReviewedWatchCandidate(testCtx)
	if err != nil {
		t.Fatalf("initialize explicit watch candidate and preset: %v", err)
	}
	preset, err := owners.Presets.Head(testCtx, watchPresetID)
	if err != nil {
		t.Fatal(err)
	}
	setup := preset.Setup
	if setup == nil || len(setup.StillProfiles) != 1 || setup.StillProfiles[0] != shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1 {
		t.Fatal("watch preset does not pin its single 2560x1440 still profile")
	}
	device, err := owners.Devices.GetRevision(testCtx, setup.Device.Id, setup.Device.Revision)
	if err != nil {
		t.Fatal(err)
	}
	scene, err := owners.Scenes.GetRevision(testCtx, setup.Scene.Id, setup.Scene.Revision)
	if err != nil {
		t.Fatal(err)
	}
	motion, err := owners.Motions.GetRevision(testCtx, setup.Motion.Id, setup.Motion.Revision)
	if err != nil {
		t.Fatal(err)
	}
	asset, err := starterAssets.ReadFile(watchAssetFile)
	if err != nil {
		t.Fatal(err)
	}
	if setup.Device.Id != watchDeviceID || device.DeviceID != setup.Device.Id || device.Revision != setup.Device.Revision || device.SHA256 != setup.Device.Sha256 || setup.Device.Sha256 != state.Device.SHA256 || setup.Scene.Id != watchSceneID || scene.SceneID != setup.Scene.Id || scene.Revision != setup.Scene.Revision || scene.SHA256 != setup.Scene.Sha256 || setup.Scene.Sha256 != state.Scene.SHA256 || motion.MotionID != setup.Motion.Id || motion.Revision != setup.Motion.Revision || motion.SHA256 != setup.Motion.Sha256 || setup.Motion.Sha256 != state.Motion.SHA256 || state.Device.Candidate.Geometry.SHA256 != watchAssetSHA256 || sha256Hex(asset) != watchAssetSHA256 || setup.CameraId != watchCameraID || setup.DisplayId != watchDisplayID || device.Candidate.Definition.Group != devices.GroupWatch || device.Candidate.Definition.Kind != "generic_rectangular_smartwatch_style" || device.Candidate.Definition.IdentityStatus != devices.IdentityGeneric || device.Candidate.Definition.Variant != "unknown; source hierarchy scale preserved" || device.Candidate.Definition.Qualification != "unqualified" || len(device.Candidate.Definition.ModelCandidates) != 1 || device.Candidate.Definition.ModelCandidates[0].SourcePath != "" || device.Candidate.Definition.ModelCandidates[0].SHA256 != watchAssetSHA256 {
		t.Fatal("watch native harness is not bound to the exact pinned candidate, preset, profile, and derivative")
	}
	geometry, err := rendering.ResolvePhysicalScreenGeometry(asset, device.Candidate, watchDisplayID)
	if err != nil || geometry.NodeIndex != 27 || geometry.MeshIndex != 14 || geometry.PrimitiveIndex != 0 || geometry.MaterialIndex != 11 || geometry.UVMin != [2]float64{} || geometry.UVMax != [2]float64{1, 1} {
		t.Fatalf("watch derivative screen binding changed: geometry=%+v err=%v", geometry, err)
	}
	placement := ReviewedIPadPlacement{DisplayID: state.Placement.DisplayID, DeviceSupportID: state.Placement.DeviceSupportID, SceneSupportID: state.Placement.SceneSupportID, SelectedCameraID: state.Placement.CameraID, SceneSupportWorld: state.Placement.SceneSupportWorld, SupportPivotM: state.Placement.SupportPivotM, DeviceRootRest: state.Placement.DeviceRootRest}
	stateForRender := IPadStarterState{Device: device, Scene: scene, Motion: motion, Placement: placement}
	project, err := projectService.Create(testCtx, projects.CreateInput{Name: "Generic rectangular watch candidate review"})
	if err != nil {
		t.Fatal(err)
	}
	chart := watchNativeCalibrationChart(t)
	chartSum := sha256.Sum256(chart)
	screen, err := mediaService.Import(testCtx, project.ID, "asymmetric-watch-calibration.png", int64(len(chart)), bytes.NewReader(chart))
	if err != nil {
		t.Fatal(err)
	}
	shotService := shots.NewService(shots.NewSQLiteRepository(db), projectService, mediaService, nil, nil)
	shots.SetPhysicalResolver(shotService, owners.Shots)
	snapshot := ipadStillSnapshot(stateForRender, screen.ID, placement, backend)
	snapshot.PhysicalBinding.ScreenWidthPixels, snapshot.PhysicalBinding.ScreenHeightPixels = 629, 537
	binding := snapshot.PhysicalBinding
	if !proto.Equal(setup.Device, binding.Device) || !proto.Equal(setup.Scene, binding.Scene) || !proto.Equal(setup.Motion, binding.Motion) || setup.DisplayId != binding.DisplayId || setup.DeviceSupportId != binding.DeviceSupportId || setup.SceneSupportId != binding.SceneSupportId || setup.CameraId != binding.CameraId || !proto.Equal(setup.SceneSupportWorld, binding.SceneSupportWorld) || !proto.Equal(setup.SupportPivotM, binding.SupportPivotM) || !proto.Equal(setup.DeviceRootRest, binding.DeviceRootRest) || binding.MotionSampleTick == nil || binding.MotionSampleTick.Numerator != 0 || binding.MotionSampleTick.Denominator != 1 {
		t.Fatal("watch still binding differs from pinned preset refs, placement, or identity-motion tick")
	}
	shot, revision, err := shotService.Create(testCtx, project.ID, "watch-neutral-screen-map-review", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !revision.RenderEligible {
		t.Fatalf("watch candidate shot is not render eligible: %+v", revision.Issues)
	}
	profileBytes, profile, err := rendering.ResolvePhysicalBlenderProfile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if profile.GetBlender().GetPath() == "" {
		t.Fatal("managed Blender profile has no executable")
	}
	profileSum := sha256.Sum256(profileBytes)
	profileSHA := hex.EncodeToString(profileSum[:])
	workerSum := sha256.Sum256(worker)
	blobs, err := routedblobstore.New(roots, "renderjobs")
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := renderjobs.New(renderjobs.Dependencies{Shots: shotService, Media: mediaService, Jobs: jobs.NewService(jobs.NewSQLiteRepository(db), time.Now), Blobs: blobs, Roots: roots, Exe: func() (string, error) { return apiBin, nil }, PhysicalArtifacts: owners.Library, PhysicalProfile: func(context.Context) ([]byte, *cliv1.BlenderToolProfile, error) { return profileBytes, profile, nil }, PhysicalAdapter: rendering.PhysicalAdapterBytes})
	if err != nil {
		t.Fatal(err)
	}
	job, created, err := coordinator.Submit(testCtx, renderjobs.SubmitRequest{ProjectID: project.ID, ShotID: shot.ID, Revision: revision.Revision, IdempotencyKey: "series7-watch-neutral-native-hero", OutputProfile: renderjobs.OutputProfilePNG})
	if err != nil || !created {
		t.Fatalf("submit watch still: created=%v err=%v", created, err)
	}
	t.Cleanup(func() {
		cleanupIPadJob(t, coordinator, db, roots, evidence, project.ID, job.ID, profileSHA, watchNativeReviewLeaseID)
	})
	if err := writeIPadExclusive(filepath.Join(evidence, "native-job-submission.json"), marshalIPadJSON(map[string]any{"job": job, "project": project, "shot": shot, "shot_revision": revision, "preset_id": preset.PresetID, "preset_revision": preset.Revision, "preset_sha256": preset.SHA256, "bundle_manifest_sha256": state.Bundle.ManifestSHA256, "bundle_review_sha256": state.Bundle.ReviewSHA256, "source_sha256": watchSourceSHA256, "derivation_manifest_sha256": watchManifestSHA, "device_id": device.DeviceID, "device_revision": device.Revision, "device_sha256": device.SHA256, "scene_id": scene.SceneID, "scene_revision": scene.Revision, "scene_sha256": scene.SHA256, "motion_id": motion.MotionID, "motion_sha256": motion.SHA256, "derivative_sha256": watchAssetSHA256, "chart_sha256": hex.EncodeToString(chartSum[:]), "worker_sha256": hex.EncodeToString(workerSum[:]), "profile_sha256": profileSHA, "backend": "OPTIX", "output_profile": "2560x1440", "screen_pixel_aspect": "629:537", "source_display_metric_mm": []float64{25.071, 21.404}, "qualification": "unqualified_candidate_native_still_only"})); err != nil {
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
		t.Fatalf("watch native still ended %s: %+v", job.State, job.Failure)
	}
	opened, output, err := coordinator.OpenOutput(testCtx, project.ID, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(output))
	if err != nil || cfg.Width != 2560 || cfg.Height != 1440 {
		t.Fatalf("watch output config=%+v err=%v", cfg, err)
	}
	if _, err := png.Decode(bytes.NewReader(output)); err != nil {
		t.Fatal(err)
	}
	outputSum := sha256.Sum256(output)
	if err := writeIPadExclusive(filepath.Join(evidence, "watch-neutral-hero.png"), output); err != nil {
		t.Fatalf("retain native watch PNG for review: %v", err)
	}
	if err := writeIPadExclusive(filepath.Join(evidence, "native-output-receipt.json"), marshalIPadJSON(map[string]any{"job": opened, "output_sha256": hex.EncodeToString(outputSum[:]), "output_bytes": len(output), "width": cfg.Width, "height": cfg.Height, "status": "candidate_native_still_only_unqualified"})); err != nil {
		t.Fatal(err)
	}
}

func watchNativeCalibrationChart(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 629, 537))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{R: 15, G: 24, B: 43, A: 255}), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 0, 629, 72), image.NewUniform(color.NRGBA{R: 25, G: 190, B: 205, A: 255}), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(35, 110, 245, 260), image.NewUniform(color.NRGBA{R: 240, G: 172, B: 55, A: 255}), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(330, 290, 594, 494), image.NewUniform(color.NRGBA{R: 204, G: 55, B: 132, A: 255}), image.Point{}, draw.Src)
	for x := 275; x < 305; x++ {
		for y := 105; y < 495; y++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 225, G: 233, B: 242, A: 255})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
