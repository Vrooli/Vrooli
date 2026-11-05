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
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const monitorNativeReviewLeaseID = "monitor-screenmap-v2-native-review"

func TestMonitorNativeCleanupLeasePinsInstalledRoutes(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(root, "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(root, "primary-data")})
	if err = db.InstallTestPool(ctx, filepath.Join(root, "leased.db"), monitorNativeReviewLeaseID, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err = roots.InstallTestRoots(storage.Paths{DataDir: filepath.Join(root, "leased-data")}, monitorNativeReviewLeaseID, time.Minute); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := roots.ClearTestRoots(monitorNativeReviewLeaseID); err != nil {
			t.Error(err)
		}
		if err := db.ClearTestPool(monitorNativeReviewLeaseID); err != nil {
			t.Error(err)
		}
	})
	pinned, err := db.PinTestPool(database.WithTestMode(ctx), monitorNativeReviewLeaseID)
	if err != nil {
		t.Fatalf("pin native cleanup database lease: %v", err)
	}
	pinned, err = roots.PinTestRoots(pinned, monitorNativeReviewLeaseID)
	if err != nil {
		t.Fatalf("pin native cleanup file lease: %v", err)
	}
	if _, err := db.PoolForContext(pinned); err != nil {
		t.Fatal(err)
	}
	if _, err := roots.Pick(pinned, storage.ClassData); err != nil {
		t.Fatal(err)
	}
}

func TestOptInRenderMonitorStarterSupportedHero(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E4_MONITOR_NATIVE") != "1" {
		t.Skip("set DEVICE_STUDIO_E4_MONITOR_NATIVE=1 to run the owned monitor still render")
	}
	apiBin := os.Getenv("DEVICE_STUDIO_E4_MONITOR_API_BIN")
	if filepath.Clean(apiBin) != apiBin || !filepath.IsAbs(apiBin) {
		t.Fatalf("API binary path must be absolute and clean: %q", apiBin)
	}
	workerInfo, err := os.Lstat(apiBin)
	if err != nil || !workerInfo.Mode().IsRegular() || workerInfo.Mode()&os.ModeSymlink != 0 || workerInfo.Size() <= 0 || workerInfo.Size() > 256<<20 {
		t.Fatalf("API binary is not a bounded regular file: %v", err)
	}
	worker, err := os.ReadFile(apiBin)
	if err != nil || len(worker) == 0 || len(worker) > 256<<20 {
		t.Fatalf("read bounded reviewed worker executable: %v", err)
	}
	evidence := filepath.Clean(os.Getenv("DEVICE_STUDIO_E4_MONITOR_EVIDENCE_DIR"))
	if !filepath.IsAbs(evidence) || !strings.Contains(filepath.ToSlash(evidence), "/innerscene-ultrawide-monitor/native-runs/") || filepath.Base(evidence) == "native-runs" {
		t.Fatalf("evidence path outside monitor native run tree: %q", evidence)
	}
	if err := os.Mkdir(evidence, 0o700); err != nil {
		t.Fatalf("create exclusive evidence directory: %v", err)
	}
	backendName := os.Getenv("DEVICE_STUDIO_E4_MONITOR_BACKEND")
	backend, err := ipadNativeBackend(backendName)
	if err != nil || backend != shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_OPTIX {
		t.Fatalf("admitted backend must explicitly be OPTIX: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 11*time.Minute)
	defer cancel()
	providers := []database.SchemaProvider{database.SchemaProviderFunc(library.Schema), database.SchemaProviderFunc(devices.Schema), database.SchemaProviderFunc(scenes.Schema), database.SchemaProviderFunc(motions.Schema), database.SchemaProviderFunc(projects.Schema), database.SchemaProviderFunc(shots.Schema), database.SchemaProviderFunc(jobs.Schema)}
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
	if err := db.InstallTestPool(ctx, filepath.Join(evidence, "leased.db"), monitorNativeReviewLeaseID, 25*time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := roots.InstallTestRoots(storage.Paths{DataDir: filepath.Join(evidence, "leased-data")}, monitorNativeReviewLeaseID, 25*time.Minute); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ds, rs := db.LeaseStats(), roots.LeaseStats()
		re, de := roots.ClearTestRoots(monitorNativeReviewLeaseID), db.ClearTestPool(monitorNativeReviewLeaseID)
		if err := writeIPadExclusive(filepath.Join(evidence, "routing-cleanup-receipt.json"), marshalIPadJSON(map[string]any{"database_test_requests": ds.TestPoolRequests, "database_primary_requests": ds.PrimaryDuringTestModeRequests, "file_test_writes": rs.TestRootWrites, "file_primary_writes": rs.PrimaryWritesDuringTestMode, "root_error": errorText(re), "database_error": errorText(de)})); err != nil {
			t.Errorf("retain routed cleanup receipt: %v", err)
		}
		if re != nil || de != nil {
			t.Errorf("clear routed stores: roots=%v db=%v", re, de)
		}
		if ds.PrimaryDuringTestModeRequests != 0 || rs.PrimaryWritesDuringTestMode != 0 {
			t.Errorf("test mode escaped isolated routes: primary database requests=%d primary file writes=%d", ds.PrimaryDuringTestModeRequests, rs.PrimaryWritesDuringTestMode)
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
	if _, err := owners.InitializeReviewedStarter(testCtx); err != nil {
		t.Fatalf("initialize reviewed candidates and monitor preset: %v", err)
	}
	preset, err := owners.Presets.Head(testCtx, monitorV2PresetID)
	if err != nil {
		t.Fatal(err)
	}
	setup := preset.Setup
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
	if device.DeviceID != setup.Device.Id || device.Revision != setup.Device.Revision || device.SHA256 != setup.Device.Sha256 || scene.SceneID != setup.Scene.Id || scene.Revision != setup.Scene.Revision || scene.SHA256 != setup.Scene.Sha256 || motion.MotionID != setup.Motion.Id || motion.Revision != setup.Motion.Revision || motion.SHA256 != setup.Motion.Sha256 || device.Candidate.Geometry.SHA256 != monitorV2AssetSHA256 || device.Candidate.Definition.Qualification != "unqualified" || device.Candidate.Definition.IdentityStatus != devices.IdentityGeneric || len(device.Candidate.Definition.Displays) == 0 || device.Candidate.Definition.Displays[0].ActiveArea.Status != "unknown" || setup.CameraId != monitorV2CameraID || scene.SceneID != monitorV2SceneID {
		t.Fatalf("monitor preset identity drift: device=%+v scene=%s camera=%s", device.Candidate.Definition, scene.SceneID, setup.CameraId)
	}
	state := IPadStarterState{Device: device, Scene: scene, Motion: motion}
	pl := monitorPlacementFor(monitorV2Seed())
	state.Placement = ReviewedIPadPlacement{DisplayID: pl.DisplayID, DeviceSupportID: pl.DeviceSupportID, SceneSupportID: pl.SceneSupportID, SelectedCameraID: pl.CameraID, SceneSupportWorld: pl.SceneSupportWorld, SupportPivotM: pl.SupportPivotM, DeviceRootRest: pl.DeviceRootRest}
	project, err := projectService.Create(testCtx, projects.CreateInput{Name: "Generic monitor candidate review"})
	if err != nil {
		t.Fatal(err)
	}
	chart := monitorNativeChart(t)
	chartSum := sha256.Sum256(chart)
	screen, err := mediaService.Import(testCtx, project.ID, "monitor-calibration-chart.png", int64(len(chart)), bytes.NewReader(chart))
	if err != nil {
		t.Fatal(err)
	}
	shotService := shots.NewService(shots.NewSQLiteRepository(db), projectService, mediaService, nil, nil)
	shots.SetPhysicalResolver(shotService, owners.Shots)
	profileBytes, profile, err := rendering.ResolvePhysicalBlenderProfile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if profile.GetBlender().GetPath() == "" {
		t.Fatal("managed Blender profile has no executable")
	}
	profileSum := sha256.Sum256(profileBytes)
	profileSHA := hex.EncodeToString(profileSum[:])
	blobs, err := routedblobstore.New(roots, "renderjobs")
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := renderjobs.New(renderjobs.Dependencies{Shots: shotService, Media: mediaService, Jobs: jobs.NewService(jobs.NewSQLiteRepository(db), time.Now), Blobs: blobs, Roots: roots, Exe: func() (string, error) { return apiBin, nil }, PhysicalArtifacts: owners.Library, PhysicalProfile: func(context.Context) ([]byte, *cliv1.BlenderToolProfile, error) { return profileBytes, profile, nil }, PhysicalAdapter: rendering.PhysicalAdapterBytes})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := ipadStillSnapshot(state, screen.ID, state.Placement, backend)
	snapshot.PhysicalBinding.ScreenWidthPixels = 2430
	snapshot.PhysicalBinding.ScreenHeightPixels = 1000
	binding := snapshot.PhysicalBinding
	if !proto.Equal(setup.Device, binding.Device) || !proto.Equal(setup.Scene, binding.Scene) || !proto.Equal(setup.Motion, binding.Motion) || setup.DisplayId != binding.DisplayId || setup.DeviceSupportId != binding.DeviceSupportId || setup.SceneSupportId != binding.SceneSupportId || setup.CameraId != binding.CameraId || !proto.Equal(setup.SceneSupportWorld, binding.SceneSupportWorld) || !proto.Equal(setup.SupportPivotM, binding.SupportPivotM) || !proto.Equal(setup.DeviceRootRest, binding.DeviceRootRest) {
		t.Fatal("still binding diverges from the saved monitor preset refs or support pose")
	}
	if binding.MotionSampleTick == nil || binding.MotionSampleTick.Numerator != 0 || binding.MotionSampleTick.Denominator != 1 {
		t.Fatalf("still does not use the preset identity motion at tick zero: %+v", binding.MotionSampleTick)
	}
	shot, rev, err := shotService.Create(testCtx, project.ID, "monitor-supported-candidate-tabletop", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !rev.RenderEligible {
		t.Fatalf("monitor candidate is not render eligible: %+v", rev.Issues)
	}
	job, created, err := coordinator.Submit(testCtx, renderjobs.SubmitRequest{ProjectID: project.ID, ShotID: shot.ID, Revision: 1, IdempotencyKey: "monitor-candidate-hero", OutputProfile: renderjobs.OutputProfilePNG})
	if err != nil || !created {
		t.Fatalf("submit native still: created=%v err=%v", created, err)
	}
	t.Cleanup(func() {
		cleanupIPadJob(t, coordinator, db, roots, evidence, project.ID, job.ID, profileSHA, monitorNativeReviewLeaseID)
	})
	workerSum := sha256.Sum256(worker)
	if err := writeIPadExclusive(filepath.Join(evidence, "native-job-submission.json"), marshalIPadJSON(map[string]any{"project_id": project.ID, "shot_id": shot.ID, "job": job, "preset_id": preset.PresetID, "preset_revision": preset.Revision, "preset_sha256": preset.SHA256, "device_sha256": device.SHA256, "scene_sha256": scene.SHA256, "motion_sha256": motion.SHA256, "chart_sha256": hex.EncodeToString(chartSum[:]), "worker_sha256": hex.EncodeToString(workerSum[:]), "profile_sha256": profileSHA, "backend": backendName, "status": "candidate_native_hero_only_unqualified"})); err != nil {
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
		t.Fatalf("monitor native still ended %s: %+v", job.State, job.Failure)
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
	outputSum := sha256.Sum256(output)
	if err := writeIPadExclusive(filepath.Join(evidence, "monitor-hero.png"), output); err != nil {
		t.Fatalf("retain native monitor PNG for visual review: %v", err)
	}
	if err := writeIPadExclusive(filepath.Join(evidence, "native-output-receipt.json"), marshalIPadJSON(map[string]any{"job": opened, "output_sha256": hex.EncodeToString(outputSum[:]), "output_bytes": len(output), "width": cfg.Width, "height": cfg.Height, "status": "candidate_native_hero_only_unqualified"})); err != nil {
		t.Fatal(err)
	}
}
func monitorNativeChart(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2430, 1000))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{R: 245, G: 246, B: 248, A: 255}), image.Point{}, draw.Src)
	for x := 0; x < 2430; x += 100 {
		for y := 0; y < 1000; y++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 190, G: 195, B: 202, A: 255})
		}
	}
	for i := 0; i < 8; i++ {
		c := color.NRGBA{R: uint8(25 * i), G: uint8(255 - 25*i), B: uint8((i * 37) % 256), A: 255}
		draw.Draw(img, image.Rect(70+i*180, 120, 220+i*180, 260), image.NewUniform(c), image.Point{}, draw.Src)
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
