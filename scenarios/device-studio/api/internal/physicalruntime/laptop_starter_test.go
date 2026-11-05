package physicalruntime

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"device-studio/internal/composition"
	"device-studio/internal/devices"
	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/renderjobs"
	"device-studio/internal/routedblobstore"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

func TestReviewedLaptopCandidateAndPhysicalClosure(t *testing.T) {
	ctx := context.Background()
	providers := []database.SchemaProvider{
		database.SchemaProviderFunc(library.Schema), database.SchemaProviderFunc(devices.Schema),
		database.SchemaProviderFunc(scenes.Schema), database.SchemaProviderFunc(motions.Schema),
		database.SchemaProviderFunc(projects.Schema), database.SchemaProviderFunc(shots.Schema), database.SchemaProviderFunc(jobs.Schema),
	}
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.EnsureSchemas(ctx, db.Primary(), providers...); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(initCtx context.Context, pool *sql.DB) error {
		return database.EnsureSchemas(initCtx, pool, providers...)
	})
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(t.TempDir(), "primary-data")})
	if err := db.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), "laptop-candidate-lease", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := roots.InstallTestRoots(storage.Paths{DataDir: filepath.Join(t.TempDir(), "leased-data")}, "laptop-candidate-lease", time.Minute); err != nil {
		t.Fatal(err)
	}
	testCtx := database.WithTestMode(ctx)
	projectRepo := projects.NewSQLiteRepository(db, nil)
	projectService := projects.NewService(projectRepo, nil)
	mediaStore, err := routedblobstore.New(roots, "media")
	if err != nil {
		t.Fatal(err)
	}
	mediaService := projects.NewSourceMediaService(projects.SourceMediaDeps{Projects: projectService, Repository: projectRepo.(projects.SourceMediaRepository), Store: mediaStore})
	owners, err := New(db, roots, mediaService)
	if err != nil {
		t.Fatal(err)
	}
	laptop, err := owners.InitializeReviewedLaptopStarter(testCtx)
	if err != nil {
		t.Fatalf("initialize the attributed derive16 laptop candidate on the paired test route: %v", err)
	}
	replay, err := owners.InitializeReviewedLaptopStarter(testCtx)
	if err != nil {
		t.Fatalf("replay exact laptop candidate revisions: %v", err)
	}
	if laptop.Device.SHA256 != replay.Device.SHA256 || laptop.Scene.SHA256 != replay.Scene.SHA256 || laptop.FramedScene.SHA256 != replay.FramedScene.SHA256 || laptop.Motion.SHA256 != replay.Motion.SHA256 || laptop.Bundle.ManifestSHA256 != replay.Bundle.ManifestSHA256 {
		t.Fatal("laptop initializer replay changed its immutable bundle or candidate revisions")
	}
	if laptop.Device.State != devices.DeviceRevisionCandidate || laptop.Device.Candidate.Definition.Qualification != "unqualified" || laptop.Device.Candidate.Definition.ModelCandidates[0].SourcePath != "" || laptop.Device.Candidate.Definition.ModelCandidates[0].License != "CC BY 4.0" {
		t.Fatalf("laptop identity, source rights, or candidate-only status changed: %+v", laptop.Device)
	}
	laptopGLB, err := starterAssets.ReadFile("assets/laptop/laptop-neutral-rig.glb")
	if err != nil {
		t.Fatal(err)
	}
	screenMap, err := rendering.ResolvePhysicalScreenGeometry(laptopGLB, laptop.Device.Candidate, laptopDisplayID)
	if err != nil || screenMap.NodeIndex != 5 || screenMap.UVMin != [2]float64{0, 0} || screenMap.UVMax != [2]float64{1, 1} {
		t.Fatalf("derived screen UV profile did not pass exact geometry preflight: mapping=%+v err=%v", screenMap, err)
	}
	uvWidthMM, uvHeightMM, err := rendering.ResolvePhysicalDisplayGeometryMetrics(laptopGLB, laptop.Device.Candidate, laptopDisplayID)
	if err != nil {
		t.Fatalf("resolve UV-calibrated active screen dimensions: %v", err)
	}
	t.Logf("derive16 UV-calibrated screen dimensions: %.3f × %.3f mm (ratio %.6f); old AABB metric %.3f × %.3f mm", uvWidthMM, uvHeightMM, uvWidthMM/uvHeightMM, laptop.Device.Candidate.Definition.Displays[0].CoordinateMetric.WidthMM, laptop.Device.Candidate.Definition.Displays[0].CoordinateMetric.HeightMM)
	uvCandidate := laptop.Device.Candidate
	uvCandidate.Definition.Displays[0].CoordinateMetric.HeightMM = uvHeightMM
	uvCorners, err := rendering.ResolvePhysicalDisplayCorners(laptopGLB, uvCandidate, laptopDisplayID)
	if err != nil {
		t.Fatalf("resolve active screen rectangle: %v", err)
	}
	t.Logf("derive16 active display corners: %+v", uvCorners)
	targetHeight := (.42 * 1920 / 1080) / (uvWidthMM / uvHeightMM)
	_, endpointResidual, err := rendering.CompilePhysicalCameraEndpoint(rendering.PhysicalCameraEndpoint{DisplayCorners: uvCorners, RasterWidth: 1920, RasterHeight: 1080, Target: rendering.NormalizedRect{X: .48, Y: .25, Width: .42, Height: targetHeight}, FocalLengthMM: 50, SensorWidthMM: 36, SensorHeightMM: 24})
	if err != nil {
		t.Fatalf("compile fitted laptop endpoint: %v", err)
	}
	t.Logf("derive16 fitted endpoint residual: %.6f px", endpointResidual)
	if endpointResidual > 2 {
		t.Fatalf("fitted endpoint residual %.6f px exceeds the two-pixel handoff target", endpointResidual)
	}
	if laptop.Scene.SceneID != laptopSceneID || laptop.Scene.State != "candidate" || len(laptop.Scene.Candidate.Parts) != 11 {
		t.Fatalf("cup-free room candidate is not separately identified and unqualified: %+v", laptop.Scene)
	}
	if laptop.FramedScene.SceneID != laptopFramedSceneID || laptop.FramedScene.State != "candidate" || laptop.FramedPlacement.CameraID != laptopFramedCameraID || laptop.FramedScene.SHA256 == laptop.Scene.SHA256 {
		t.Fatalf("framed laptop scene must be a separate candidate with its own camera: scene=%+v placement=%+v", laptop.FramedScene, laptop.FramedPlacement)
	}
	oriented, err := owners.InitializeReviewedLaptopScreenUpright(testCtx)
	if err != nil {
		t.Fatalf("initialize display-only orientation candidate: %v", err)
	}
	orientedReplay, err := owners.InitializeReviewedLaptopScreenUpright(testCtx)
	if err != nil {
		t.Fatalf("replay display-only orientation candidate: %v", err)
	}
	if laptop.Device.SHA256 != replay.Device.SHA256 || oriented.Device.DeviceID != laptopOrientedDeviceID || oriented.Device.SHA256 != orientedReplay.Device.SHA256 || oriented.FramedScene.SHA256 != orientedReplay.FramedScene.SHA256 || oriented.Preset.SHA256 != orientedReplay.Preset.SHA256 {
		t.Fatal("oriented candidate replay changed the original candidate or its own immutable references")
	}
	dispatchedPreset, err := owners.InitializePhysicalStarter(testCtx, "laptop-screen-upright-180")
	if err != nil || dispatchedPreset.PresetID != oriented.Preset.PresetID || dispatchedPreset.Revision != oriented.Preset.Revision || dispatchedPreset.SHA256 != oriented.Preset.SHA256 {
		t.Fatalf("public starter dispatch must resolve the exact oriented preset: preset=%+v err=%v", dispatchedPreset, err)
	}
	stillMotion, err := owners.Starter.saveStillMotion(testCtx)
	if err != nil {
		t.Fatalf("resolve exact shared identity still motion: %v", err)
	}
	if oriented.Device.Candidate.Definition.Qualification != "unqualified" || oriented.Device.Candidate.Definition.ModelCandidates[0].SourcePath != "" || oriented.Preset.Setup.Device.Id != oriented.Device.DeviceID || oriented.Preset.Setup.Scene.Id != oriented.FramedScene.SceneID || oriented.Preset.Setup.Motion.Id != stillMotion.MotionID || oriented.Preset.Setup.Motion.Sha256 != stillMotion.SHA256 || oriented.Motion.MotionID != laptopMotionID || oriented.Preset.PresetID != "laptop-screen-upright-180-review-still-v2" || oriented.Preset.Setup.CameraId != laptopOrientedCameraID {
		t.Fatal("oriented still preset must bind the exact candidate/scene and identity motion while retaining the separate hinge motion")
	}
	orientedGLB, err := starterAssets.ReadFile(laptopOrientedAssetFile)
	if err != nil {
		t.Fatal(err)
	}
	orientedScreen, err := rendering.ResolvePhysicalScreenGeometry(orientedGLB, oriented.Device.Candidate, laptopDisplayID)
	if err != nil || orientedScreen.NodeIndex != 5 || orientedScreen.PrimitiveIndex != 0 || orientedScreen.UVMin != [2]float64{0, 0} || orientedScreen.UVMax != [2]float64{1, 1} {
		t.Fatalf("oriented screen binding did not pass the exact screen preflight: %+v err=%v", orientedScreen, err)
	}
	vertical, err := owners.InitializeReviewedLaptopScreenVFlip(testCtx)
	if err != nil {
		t.Fatalf("initialize vertical-only laptop screen candidate: %v", err)
	}
	verticalReplay, err := owners.InitializeReviewedLaptopScreenVFlip(testCtx)
	if err != nil || vertical.Device.SHA256 != verticalReplay.Device.SHA256 || vertical.FramedScene.SHA256 != verticalReplay.FramedScene.SHA256 || vertical.Preset.SHA256 != verticalReplay.Preset.SHA256 {
		t.Fatalf("vertical-only candidate replay changed its immutable refs: err=%v", err)
	}
	if vertical.Device.DeviceID == oriented.Device.DeviceID || vertical.Preset.PresetID != "laptop-screen-vflip-review-still-v1" || vertical.Device.Candidate.Definition.Qualification != "unqualified" || vertical.Device.Candidate.Geometry.SHA256 != "1d833e3d08d0722952ecd9b6616cf933af11fca5716cfe4bc67aa110850272bd" {
		t.Fatal("vertical-only screen view must use a distinct, pinned, unqualified candidate revision")
	}
	verticalPreset, err := owners.InitializePhysicalStarter(testCtx, "laptop-screen-vflip")
	if err != nil || verticalPreset.PresetID != vertical.Preset.PresetID || verticalPreset.SHA256 != vertical.Preset.SHA256 {
		t.Fatalf("public selector did not return the exact vertical-flip preset: preset=%+v err=%v", verticalPreset, err)
	}
	verticalGLB, err := starterAssets.ReadFile("assets/laptop/laptop-neutral-rig-screen-vflip-v1.glb")
	if err != nil {
		t.Fatal(err)
	}
	verticalScreen, err := rendering.ResolvePhysicalScreenGeometry(verticalGLB, vertical.Device.Candidate, laptopDisplayID)
	if err != nil || verticalScreen.NodeIndex != 5 || verticalScreen.PrimitiveIndex != 0 || verticalScreen.UVMin != [2]float64{0, 0} || verticalScreen.UVMax != [2]float64{1, 1} {
		t.Fatalf("vertical-flip screen binding did not pass exact geometry preflight: %+v err=%v", verticalScreen, err)
	}
	textureBindings := map[string]bool{}
	for _, artifact := range laptop.FramedScene.Candidate.Artifacts {
		if artifact.Ref.Role == library.RoleTextureImage {
			textureBindings[artifact.ID] = artifact.Ref.SHA256 != ""
		}
	}
	for _, textureID := range []string{"travertine-color", "travertine-normal-gl", "travertine-roughness", "travertine-height"} {
		if !textureBindings[textureID] {
			t.Fatalf("framed laptop scene omits pinned shared table texture %q", textureID)
		}
	}
	var texturedTable bool
	for _, material := range laptop.FramedScene.Candidate.Materials {
		if len(material.Inputs) == 4 {
			texturedTable = material.Grade != nil && material.Grade.MixFactor == .55
		}
	}
	if !texturedTable {
		t.Fatal("framed laptop scene did not use the shared approved table material recipe")
	}
	deviceInventory, err := rendering.InspectPhysicalGLB(laptopGLB)
	if err != nil {
		t.Fatal(err)
	}
	sweep, err := physicalGLBJointSweepBounds(deviceInventory, laptop.Device.Candidate, laptop.Motion.Candidate, laptopJointID)
	if err != nil {
		t.Fatal(err)
	}
	maxFrameX, maxFrameY := assertLaptopSweepFitsCamera(t, sweep, laptop.Device.Candidate.SourceToCanonical, laptop.FramedPlacement, laptop.FramedScene.Candidate.CameraFrames[0])
	t.Logf("framed candidate camera preflight: swept_bounds=%+v frame_half_extents=%.6f,%.6f camera_translation=%+v", sweep, maxFrameX, maxFrameY, laptop.FramedScene.Candidate.CameraFrames[0].Frame.Translation)
	for _, part := range laptop.Scene.Candidate.Parts {
		if part.NodeName == "TeaCup" || part.NodeName == "TeaCupHandle" || part.NodeName == "TeaSurface" {
			t.Fatalf("cup/tea prop entered the laptop scene candidate: %+v", part)
		}
	}
	if laptop.Motion.MotionID != laptopMotionID || laptop.Motion.Candidate.DurationTicks != 48 || laptop.Motion.Candidate.Timebase != (composition.Rational{Num: 24, Den: 1}) {
		t.Fatalf("laptop hinge candidate timing changed: %+v", laptop.Motion.Candidate)
	}
	for _, tick := range []int64{0, 24, 47} {
		pose, err := motions.SampleAt(laptop.Motion.Candidate, composition.Rational{Num: tick, Den: 1})
		if err != nil || len(pose.Tracks) != 1 || pose.Tracks[0].AngleRad == nil {
			t.Fatalf("sample candidate lid at tick %d: pose=%+v err=%v", tick, pose, err)
		}
		want := map[int64]float64{0: 0, 24: laptopClosedAngleRad / 2, 47: laptopClosedAngleRad}[tick]
		if got := *pose.Tracks[0].AngleRad; math.Abs(got-want) > 1e-12 {
			t.Fatalf("candidate pose tick %d = %g, want %g", tick, got, want)
		}
	}
	if laptop.Placement.DeviceSupportID != "laptop-root-support" || laptop.Placement.SceneSupportID != laptopTableSupportID || laptop.Placement.CameraID != laptopCameraID || laptop.Placement.SceneSupportWorld.Translation.Y != .75 {
		t.Fatalf("candidate placement is not pinned to the laptop support and separate camera: %+v", laptop.Placement)
	}

	// Exercise the actual shot resolver and render-job closure builder. The
	// coordinator Submit path validates and persists the immutable v5 worker
	// manifest but does not advance or launch the native operation owner.
	project, err := projectService.Create(testCtx, projects.CreateInput{Name: "Laptop candidate review"})
	if err != nil {
		t.Fatal(err)
	}
	screenBytes := laptopTestScreenPNG(t)
	screen, err := mediaService.Import(testCtx, project.ID, "laptop-screen.png", int64(len(screenBytes)), bytes.NewReader(screenBytes))
	if err != nil {
		t.Fatal(err)
	}
	shotService := shots.NewService(shots.NewSQLiteRepository(db), projectService, mediaService, nil, nil)
	shots.SetPhysicalResolver(shotService, owners.Shots)
	placement := laptop.Placement
	shotSnapshot := &shotsv1.ShotSnapshot{
		PhysicalBinding: &shotsv1.PhysicalShotBinding{
			Device:    &shotsv1.ComponentRevisionRef{Id: laptop.Device.DeviceID, Revision: laptop.Device.Revision, Sha256: laptop.Device.SHA256},
			Scene:     &shotsv1.ComponentRevisionRef{Id: laptop.Scene.SceneID, Revision: laptop.Scene.Revision, Sha256: laptop.Scene.SHA256},
			Motion:    &shotsv1.ComponentRevisionRef{Id: laptop.Motion.MotionID, Revision: laptop.Motion.Revision, Sha256: laptop.Motion.SHA256},
			DisplayId: placement.DisplayID, DeviceSupportId: placement.DeviceSupportID, SceneSupportId: placement.SceneSupportID, CameraId: placement.CameraID,
			SceneSupportWorld: laptopTRSProto(placement.SceneSupportWorld), SupportPivotM: laptopVecProto(placement.SupportPivotM), DeviceRootRest: laptopTRSProto(placement.DeviceRootRest),
			ScreenWidthPixels: 1644, ScreenHeightPixels: 1000, RendererProfileId: rendering.PhysicalRendererProfile, RendererProfileVersion: 1,
			Backend:      shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_CPU,
			MotionOutput: &shotsv1.PhysicalMotionOutputBinding{ProfileId: rendering.PhysicalMotionOutput, ProfileVersion: 1, Width: 1920, Height: 1080, OutputRateNumerator: 24, OutputRateDenominator: 1, FrameCount: 48},
		},
		Presentation: &shotsv1.ScreenPresentation{SourceMediaId: screen.ID, CaptureArea: shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED, Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN, FitMode: shotsv1.ScreenFitMode_SCREEN_FIT_MODE_CONTAIN},
		MotionMode:   shotsv1.MotionMode_MOTION_MODE_FIXED,
	}
	shot, revision, err := shotService.Create(testCtx, project.ID, "Laptop hinge candidate", shotSnapshot)
	if err != nil {
		t.Fatalf("save physical laptop shot from candidate revisions: %v", err)
	}
	if !revision.RenderEligible || revision.Physical == nil || revision.Physical.DeviceSHA256 != laptop.Device.SHA256 || revision.Physical.SceneSHA256 != laptop.Scene.SHA256 || revision.Physical.MotionSHA256 != laptop.Motion.SHA256 {
		t.Fatalf("saved laptop shot did not pin the exact candidate revisions: %+v", revision)
	}
	workerPath := filepath.Join(t.TempDir(), "device-studio-api")
	if err := os.WriteFile(workerPath, []byte("test-worker-executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	profile := starterTestBlenderProfile()
	profileBytes, err := protojson.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	ops := &starterPhysicalOperationOwner{}
	renderBlobs, err := routedblobstore.New(roots, "renderjobs")
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := renderjobs.New(renderjobs.Dependencies{
		Shots: shotService, Media: mediaService, Jobs: jobs.NewService(jobs.NewSQLiteRepository(db), time.Now), Blobs: renderBlobs,
		Roots: roots, Ops: ops, Exe: func() (string, error) { return workerPath, nil }, PhysicalArtifacts: owners.Library,
		PhysicalProfile: func(context.Context) ([]byte, *cliv1.BlenderToolProfile, error) { return profileBytes, profile, nil },
		PhysicalAdapter: rendering.PhysicalAdapterBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	job, created, err := coordinator.Submit(testCtx, renderjobs.SubmitRequest{ProjectID: project.ID, ShotID: shot.ID, Revision: revision.Revision, IdempotencyKey: "laptop-candidate-closure-v1", OutputProfile: renderjobs.OutputProfileMP4})
	if err != nil || !created {
		t.Fatalf("submit actual laptop candidate through closure builder: created=%t err=%v", created, err)
	}
	if ops.starts != 0 {
		t.Fatalf("closure integration test launched a native worker before Advance: starts=%d", ops.starts)
	}
	var closureKey string
	for _, binding := range job.Input.Bindings {
		if binding.Role == "physical_closure" {
			closureKey = binding.ArtifactID
		}
	}
	if closureKey == "" {
		t.Fatal("laptop motion job omitted its physical closure")
	}
	reader, _, err := renderBlobs.Get(testCtx, closureKey)
	if err != nil {
		t.Fatal(err)
	}
	closureBytes, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		t.Fatal(err)
	}
	var closure struct {
		Manifest rendering.PhysicalRenderManifest `json:"manifest"`
	}
	if err := json.Unmarshal(closureBytes, &closure); err != nil {
		t.Fatalf("decode persisted candidate closure: %v", err)
	}
	if closure.Manifest.Version != rendering.PhysicalArticulatedManifestVersion || closure.Manifest.Output.ArticulationProfile != rendering.PhysicalArticulationProfile || closure.Manifest.Device.CanonicalSHA != laptop.Device.SHA256 || closure.Manifest.Scene.CanonicalSHA != laptop.Scene.SHA256 || closure.Manifest.Motion.CanonicalSHA != laptop.Motion.SHA256 {
		t.Fatalf("closure lost the immutable laptop candidate/articulation bindings: %+v", closure.Manifest)
	}
	if closure.Manifest.Output.Kind != "motion_mp4" || closure.Manifest.Output.FrameCount != 48 || len(closure.Manifest.Output.Samples) != 48 {
		t.Fatalf("closure did not carry the full reviewed 48-frame hinge schedule: kind=%q frames=%d samples=%d", closure.Manifest.Output.Kind, closure.Manifest.Output.FrameCount, len(closure.Manifest.Output.Samples))
	}
	stillBinding := &shotsv1.PhysicalShotBinding{
		Device:    proto.Clone(vertical.Preset.Setup.Device).(*shotsv1.ComponentRevisionRef),
		Scene:     proto.Clone(vertical.Preset.Setup.Scene).(*shotsv1.ComponentRevisionRef),
		Motion:    proto.Clone(vertical.Preset.Setup.Motion).(*shotsv1.ComponentRevisionRef),
		DisplayId: vertical.Preset.Setup.DisplayId, DeviceSupportId: vertical.Preset.Setup.DeviceSupportId,
		SceneSupportId: vertical.Preset.Setup.SceneSupportId, CameraId: vertical.Preset.Setup.CameraId,
		SceneSupportWorld: proto.Clone(vertical.Preset.Setup.SceneSupportWorld).(*physicalv1.TRS), SupportPivotM: proto.Clone(vertical.Preset.Setup.SupportPivotM).(*physicalv1.Vec3),
		DeviceRootRest: proto.Clone(vertical.Preset.Setup.DeviceRootRest).(*physicalv1.TRS), ScreenWidthPixels: 1644, ScreenHeightPixels: 1000,
		RendererProfileId: rendering.PhysicalRendererProfile, RendererProfileVersion: 1,
		OutputProfile:    shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1,
		MotionSampleTick: &shotsv1.RationalTime{Numerator: 0, Denominator: 1}, Backend: shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_CPU,
	}
	stillSnapshot := &shotsv1.ShotSnapshot{PhysicalBinding: stillBinding, Presentation: &shotsv1.ScreenPresentation{SourceMediaId: screen.ID,
		CaptureArea: shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED, Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN,
		FitMode: shotsv1.ScreenFitMode_SCREEN_FIT_MODE_CONTAIN}, MotionMode: shotsv1.MotionMode_MOTION_MODE_FIXED}
	stillShot, stillRevision, err := shotService.Create(testCtx, project.ID, "Laptop upright display still", stillSnapshot)
	if err != nil || !stillRevision.RenderEligible {
		t.Fatalf("create render-eligible still from exact identity-motion preset: shot=%+v revision=%+v err=%v", stillShot, stillRevision, err)
	}
	stillJob, stillCreated, err := coordinator.Submit(testCtx, renderjobs.SubmitRequest{ProjectID: project.ID, ShotID: stillShot.ID, Revision: stillRevision.Revision,
		IdempotencyKey: "laptop-upright-identity-still-closure-v2", OutputProfile: renderjobs.OutputProfilePNG})
	if err != nil || !stillCreated {
		t.Fatalf("compile real still-render closure from upright candidate: created=%t err=%v", stillCreated, err)
	}
	var stillClosureKey string
	for _, binding := range stillJob.Input.Bindings {
		if binding.Role == "physical_closure" {
			stillClosureKey = binding.ArtifactID
		}
	}
	if stillClosureKey == "" {
		t.Fatal("upright still render omitted its physical closure")
	}
	stillReader, _, err := renderBlobs.Get(testCtx, stillClosureKey)
	if err != nil {
		t.Fatal(err)
	}
	stillBytes, readErr := io.ReadAll(stillReader)
	if err := errors.Join(readErr, stillReader.Close()); err != nil {
		t.Fatal(err)
	}
	var stillClosure struct {
		Manifest rendering.PhysicalRenderManifest `json:"manifest"`
	}
	if err := json.Unmarshal(stillBytes, &stillClosure); err != nil {
		t.Fatalf("decode upright still closure: %v", err)
	}
	if stillClosure.Manifest.Output.Kind != "still_png" || stillClosure.Manifest.Motion.CanonicalSHA != stillMotion.SHA256 {
		t.Fatalf("upright still closure did not pin still-compatible identity motion: output=%+v motion=%+v", stillClosure.Manifest.Output, stillClosure.Manifest.Motion)
	}
	for ordinal, want := range map[int]float64{0: 0, 24: laptopClosedAngleRad / 2, 47: laptopClosedAngleRad} {
		got := closure.Manifest.Output.Samples[ordinal].JointAngles[laptopJointID]
		if math.Abs(got-want) > 1e-12 {
			t.Fatalf("closure sample %d angle=%g want=%g", ordinal, got, want)
		}
	}
}

func assertLaptopSweepFitsCamera(t *testing.T, bounds physicalBounds, source physical.TRS, placement LaptopCandidatePlacement, camera scenes.SceneCameraFrame) (float64, float64) {
	t.Helper()
	scene, err := placement.SceneSupportWorld.Matrix()
	if err != nil {
		t.Fatal(err)
	}
	pivot, err := (physical.TRS{Translation: placement.SupportPivotM, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}).Matrix()
	if err != nil {
		t.Fatal(err)
	}
	root, err := placement.DeviceRootRest.Matrix()
	if err != nil {
		t.Fatal(err)
	}
	sourceMatrix, err := source.Matrix()
	if err != nil {
		t.Fatal(err)
	}
	world, err := physical.Compose(scene, pivot)
	if err == nil {
		world, err = physical.Compose(world, root)
	}
	if err == nil {
		world, err = physical.Compose(world, sourceMatrix)
	}
	if err != nil {
		t.Fatal(err)
	}
	cm, err := camera.Frame.Matrix()
	if err != nil {
		t.Fatal(err)
	}
	origin, err := physical.TransformPoint(cm, physical.Vec3{})
	if err != nil {
		t.Fatal(err)
	}
	axis := func(v physical.Vec3) physical.Vec3 {
		p, err := physical.TransformPoint(cm, v)
		if err != nil {
			t.Fatal(err)
		}
		return normalize(subVec(p, origin))
	}
	right, up, forward := axis(physical.Vec3{X: 1}), axis(physical.Vec3{Y: 1}), axis(physical.Vec3{Z: -1})
	maxX, maxY := 0.0, 0.0
	for mask := 0; mask < 8; mask++ {
		p := physical.Vec3{X: bounds.min.X, Y: bounds.min.Y, Z: bounds.min.Z}
		if mask&1 != 0 {
			p.X = bounds.max.X
		}
		if mask&2 != 0 {
			p.Y = bounds.max.Y
		}
		if mask&4 != 0 {
			p.Z = bounds.max.Z
		}
		p, err = physical.TransformPoint(world, p)
		if err != nil {
			t.Fatal(err)
		}
		d := subVec(p, origin)
		depth := dotVec(d, forward)
		if depth <= 0 {
			t.Fatalf("swept geometry corner is behind fitted camera: depth=%g", depth)
		}
		maxX = math.Max(maxX, math.Abs(dotVec(d, right)*(*camera.FocalLengthMM / *camera.SensorWidthMM)/depth))
		maxY = math.Max(maxY, math.Abs(dotVec(d, up)*(*camera.FocalLengthMM/(*camera.SensorWidthMM/(16.0/9.0)))/depth))
	}
	if maxX > .400001 || maxY > .400001 {
		t.Fatalf("continuous hinge sweep exceeds ten-percent camera margin: normalized half extents x=%.6f y=%.6f", maxX, maxY)
	}
	return maxX, maxY
}

func laptopVecProto(value physical.Vec3) *physicalv1.Vec3 {
	return &physicalv1.Vec3{X: value.X, Y: value.Y, Z: value.Z}
}

func laptopTRSProto(value physical.TRS) *physicalv1.TRS {
	return &physicalv1.TRS{TranslationM: laptopVecProto(value.Translation), RotationXyzw: &physicalv1.QuatXYZW{X: value.Rotation.X, Y: value.Rotation.Y, Z: value.Rotation.Z, W: value.Rotation.W}, Scale: laptopVecProto(value.Scale)}
}

func laptopTestScreenPNG(t *testing.T) []byte {
	t.Helper()
	const width, height = 1644, 1000
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 46, G: 126, B: 166, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}
