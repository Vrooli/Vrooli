package physicalruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"device-studio/internal/devices"
	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/presentations"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/renderjobs"
	"device-studio/internal/routedblobstore"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/encoding/protojson"
	_ "modernc.org/sqlite"
)

func TestReviewedStarterInitializesOnlyOnMatchingPinnedTestRoutes(t *testing.T) {
	ctx := context.Background()
	primaryData := filepath.Join(t.TempDir(), "primary-data")
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	providers := []database.SchemaProvider{
		database.SchemaProviderFunc(library.Schema), database.SchemaProviderFunc(devices.Schema),
		database.SchemaProviderFunc(scenes.Schema), database.SchemaProviderFunc(motions.Schema), database.SchemaProviderFunc(projects.Schema), database.SchemaProviderFunc(shots.Schema), database.SchemaProviderFunc(jobs.Schema),
	}
	if err := database.EnsureSchemas(ctx, db.Primary(), providers...); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(initCtx context.Context, pool *sql.DB) error {
		return database.EnsureSchemas(initCtx, pool, providers...)
	})
	roots := filerouting.New(storage.Paths{DataDir: primaryData})
	testData := filepath.Join(t.TempDir(), "leased-data")
	if err := db.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), "starter-lease", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := roots.InstallTestRoots(storage.Paths{DataDir: testData}, "starter-lease", time.Minute); err != nil {
		t.Fatal(err)
	}
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
	testCtx := database.WithTestMode(ctx)
	persistedMonitorV1 := seedPersistedMonitorV1State(t, owners, testCtx)
	if _, err := owners.Devices.GetHead(testCtx, monitorV2DeviceID); !errors.Is(err, devices.ErrRevisionNotFound) {
		t.Fatalf("screen-mapped v2 candidate must be absent before migration-free v1 startup, got err=%v", err)
	}
	first, err := owners.InitializeReviewedStarter(testCtx)
	if err != nil {
		t.Fatalf("initialize reviewed phone and laptop starters on leased routes: %v", err)
	}
	firstLaptop, err := owners.InitializeReviewedLaptopStarter(testCtx)
	if err != nil {
		t.Fatalf("read initialized laptop candidate on leased routes: %v", err)
	}
	second, err := owners.InitializeReviewedStarter(testCtx)
	if err != nil {
		t.Fatalf("exact replay: %v", err)
	}
	secondLaptop, err := owners.InitializeReviewedLaptopStarter(testCtx)
	if err != nil {
		t.Fatalf("laptop replay: %v", err)
	}
	monitorDeviceAfterReplay, err := owners.Devices.GetHead(testCtx, monitorDeviceID)
	if err != nil {
		t.Fatal(err)
	}
	monitorSceneAfterReplay, err := owners.Scenes.GetHead(testCtx, monitorSceneID)
	if err != nil {
		t.Fatal(err)
	}
	monitorPresetAfterReplay, err := owners.Presets.Head(testCtx, monitorPresetID)
	if err != nil {
		t.Fatal(err)
	}
	monitorBundleAfterReplay, err := owners.LibraryCatalog.GetRevision(testCtx, monitorBundleID, persistedMonitorV1.Bundle.Manifest.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if monitorDeviceAfterReplay.Revision != persistedMonitorV1.Device.Revision || monitorDeviceAfterReplay.SHA256 != persistedMonitorV1.Device.SHA256 ||
		monitorSceneAfterReplay.Revision != persistedMonitorV1.Scene.Revision || monitorSceneAfterReplay.SHA256 != persistedMonitorV1.Scene.SHA256 ||
		monitorPresetAfterReplay.Revision != persistedMonitorV1.Preset.Revision || monitorPresetAfterReplay.SHA256 != persistedMonitorV1.Preset.SHA256 ||
		monitorBundleAfterReplay.ManifestSHA256 != persistedMonitorV1.Bundle.ManifestSHA256 || monitorBundleAfterReplay.ReviewSHA256 != persistedMonitorV1.Bundle.ReviewSHA256 {
		t.Fatalf("new initializer changed persisted v1 monitor heads: before=%+v after_device=%+v after_scene=%+v after_preset=%+v after_bundle=%+v", persistedMonitorV1, monitorDeviceAfterReplay, monitorSceneAfterReplay, monitorPresetAfterReplay, monitorBundleAfterReplay)
	}
	if first.Phone.SHA256 != second.Phone.SHA256 || first.Scene.SHA256 != second.Scene.SHA256 || first.Motion.SHA256 != second.Motion.SHA256 || first.PhoneBundle.ManifestSHA256 != second.PhoneBundle.ManifestSHA256 || first.SceneBundle.ManifestSHA256 != second.SceneBundle.ManifestSHA256 {
		t.Fatal("exact starter replay changed immutable bundle or candidate revisions")
	}
	if firstLaptop.Device.Candidate.Geometry.SHA256 != laptopGeometrySHA256 || len(firstLaptop.Scene.Candidate.Artifacts) != 1 || firstLaptop.Scene.Candidate.Artifacts[0].Ref.SHA256 != laptopRoomSHA256 || firstLaptop.Bundle.ManifestSHA256 == "" ||
		secondLaptop.Device.SHA256 != firstLaptop.Device.SHA256 || secondLaptop.Scene.SHA256 != firstLaptop.Scene.SHA256 ||
		secondLaptop.Bundle.ManifestSHA256 != firstLaptop.Bundle.ManifestSHA256 {
		t.Fatalf("laptop candidate did not initialize and replay on the same pinned route: first device=%s scene=%s bundle=%s; replay device=%s scene=%s bundle=%s", firstLaptop.Device.SHA256, firstLaptop.Scene.SHA256, firstLaptop.Bundle.ManifestSHA256, secondLaptop.Device.SHA256, secondLaptop.Scene.SHA256, secondLaptop.Bundle.ManifestSHA256)
	}
	if first.Preset.PresetID != "phone-study-hero" || first.Preset.Revision != 1 || first.Preset.SHA256 == "" || second.Preset.PresetID != first.Preset.PresetID || second.Preset.Revision != first.Preset.Revision || second.Preset.SHA256 != first.Preset.SHA256 {
		t.Fatalf("starter setup preset did not converge on one immutable revision: first=%+v replay=%+v", first.Preset, second.Preset)
	}
	if first.Preset.Setup.GetDevice().GetSha256() != first.Phone.SHA256 || first.Preset.Setup.GetScene().GetSha256() != first.Scene.SHA256 || first.Preset.Setup.GetMotion().GetSha256() != first.Motion.SHA256 || first.Preset.Setup.GetDisplayId() != first.PhoneStudyPlacement.DisplayID || first.Preset.Setup.GetCameraId() != first.PhoneStudyPlacement.SelectedCameraID {
		t.Fatalf("starter preset did not pin exact initialized revisions and placement: %+v", first.Preset.Setup)
	}
	monitorPreset, err := owners.Presets.Head(testCtx, monitorPresetID)
	if err != nil {
		t.Fatalf("monitor setup must be selectable through the existing preset owner: %v", err)
	}
	monitorReplay, err := owners.Presets.Head(testCtx, monitorPresetID)
	if err != nil || monitorPreset.Revision != 1 || monitorPreset.SHA256 == "" || monitorReplay.SHA256 != monitorPreset.SHA256 {
		t.Fatalf("monitor setup must converge on one immutable preset revision: first=%+v replay=%+v err=%v", monitorPreset, monitorReplay, err)
	}
	monitorDevice, err := owners.Devices.GetRevision(testCtx, monitorPreset.Setup.GetDevice().GetId(), monitorPreset.Setup.GetDevice().GetRevision())
	if err != nil {
		t.Fatal(err)
	}
	monitorScene, err := owners.Scenes.GetRevision(testCtx, monitorPreset.Setup.GetScene().GetId(), monitorPreset.Setup.GetScene().GetRevision())
	if err != nil {
		t.Fatal(err)
	}
	if monitorDevice.SHA256 != "297eccc8ba18069a7d71d125d9bb2fb83ca2445d011f88e244ded31d1b190b6f" || monitorScene.SHA256 != "f1d46273366fe6f20199ab30f2488953ad7b1adc4c5a3541d8af31b343858a0e" || monitorPreset.SHA256 != "69a0b2d35257776f2ade440f22a600f31ab08a18fa28c8da6a9ffab842862647" || monitorDevice.Candidate.Geometry.SHA256 != monitorAssetSHA256 || monitorDevice.Candidate.Definition.IdentityStatus != devices.IdentityGeneric || monitorDevice.Candidate.Definition.Qualification != "unqualified" || monitorDevice.Candidate.Definition.Displays[0].ActiveArea.Status != "unknown" {
		t.Fatalf("monitor candidate must pin the source-preserving generic asset without inventing active area or qualification: %+v", monitorDevice.Candidate.Definition)
	}
	monitorGLB, err := starterAssets.ReadFile(monitorAssetFile)
	if err != nil {
		t.Fatal(err)
	}
	monitorMapping, err := rendering.ResolvePhysicalScreenGeometry(monitorGLB, monitorDevice.Candidate, monitorDisplayID)
	if err != nil || monitorMapping.NodeIndex != 1 || monitorMapping.PrimitiveIndex != 0 || monitorMapping.UVMin != [2]float64{0, 0} || monitorMapping.UVMax != [2]float64{1, 1} {
		t.Fatalf("published monitor display must resolve through its exact normalized screen UV binding: mapping=%+v err=%v", monitorMapping, err)
	}
	if monitorPreset.Setup.GetDevice().GetSha256() != monitorDevice.SHA256 || monitorPreset.Setup.GetScene().GetSha256() != monitorScene.SHA256 || monitorPreset.Setup.GetMotion().GetSha256() != first.Motion.SHA256 || monitorPreset.Setup.GetCameraId() != monitorCameraID || monitorPreset.Setup.GetSceneSupportWorld().GetTranslationM().GetY() != .75 || monitorPreset.Setup.GetSupportPivotM().GetY() != 0 {
		t.Fatalf("monitor preset must pin exact device/scene/static-motion revisions and source-feet support pose: %+v", monitorPreset.Setup)
	}
	monitorPose := monitorPlacement()
	placedMonitor := ReviewedIPadPlacement{SceneSupportWorld: monitorPose.SceneSupportWorld, SupportPivotM: monitorPose.SupportPivotM, DeviceRootRest: monitorPose.DeviceRootRest}
	monitorVertices, err := ipadGLBPlacedNodeVertices(monitorGLB, placedMonitor)
	if err != nil {
		t.Fatalf("measure retained monitor foot vertices at the table pose: %v", err)
	}
	for _, foot := range []string{"Split display foot", "Split display foot.001"} {
		vertices := monitorVertices[foot]
		if len(vertices) == 0 {
			t.Fatalf("monitor GLB has no retained POSITION vertices for support foot %q", foot)
		}
		minimumY := math.Inf(1)
		for _, vertex := range vertices {
			minimumY = math.Min(minimumY, vertex.Y)
		}
		if math.Abs(minimumY-.75) > 1e-6 {
			t.Fatalf("monitor support foot %q reaches Y=%.9fm at the table pose, want 0.750000m within float32 tolerance", foot, minimumY)
		}
	}
	for _, part := range monitorScene.Candidate.Parts {
		if part.NodeName == "TeaCup" || part.NodeName == "TeaCupHandle" || part.NodeName == "TeaSurface" {
			t.Fatalf("monitor scene must reuse the cup-free table source: %+v", part)
		}
	}
	if monitorScene.SceneID != monitorSceneID || len(monitorScene.Candidate.CameraFrames) != 1 || monitorScene.Candidate.CameraFrames[0].ID != monitorCameraID {
		t.Fatalf("monitor preset must pin its own camera-fitted no-cup scene: %+v", monitorScene)
	}
	monitorV2Preset, err := owners.Presets.Head(testCtx, monitorV2PresetID)
	if err != nil {
		t.Fatalf("screen-mapped monitor candidate must be selectable through the existing preset owner: %v", err)
	}
	monitorV2Device, err := owners.Devices.GetRevision(testCtx, monitorV2Preset.Setup.GetDevice().GetId(), monitorV2Preset.Setup.GetDevice().GetRevision())
	if err != nil {
		t.Fatal(err)
	}
	monitorV2Scene, err := owners.Scenes.GetRevision(testCtx, monitorV2Preset.Setup.GetScene().GetId(), monitorV2Preset.Setup.GetScene().GetRevision())
	if err != nil {
		t.Fatal(err)
	}
	monitorV2GLB, err := starterAssets.ReadFile(monitorV2AssetFile)
	if err != nil || sha256Hex(monitorV2GLB) != monitorV2AssetSHA256 {
		t.Fatalf("screen-mapped monitor asset differs from its v2 manifest pin: %v", err)
	}
	monitorV2Inventory, err := rendering.InspectPhysicalGLB(monitorV2GLB)
	if err != nil {
		t.Fatal(err)
	}
	monitorV2Mapping, err := rendering.ResolvePhysicalScreenGeometry(monitorV2GLB, monitorV2Device.Candidate, monitorDisplayID)
	if err != nil || monitorV2Device.DeviceID != monitorV2DeviceID || monitorV2Preset.Setup.GetDevice().GetId() != monitorV2DeviceID || monitorV2Preset.Setup.GetScene().GetId() != monitorV2SceneID || monitorV2Preset.Setup.GetCameraId() != monitorV2CameraID || monitorV2Mapping.NodeIndex != 1 || monitorV2Mapping.MeshIndex != 1 || monitorV2Mapping.PrimitiveIndex != 0 || monitorV2Mapping.MaterialIndex != 3 || monitorV2Mapping.UVMin != [2]float64{0, 0} || monitorV2Mapping.UVMax != [2]float64{1, 1} {
		t.Fatalf("v2 preset must pin the split planar display primitive and its exact unit UVs: mapping=%+v err=%v", monitorV2Mapping, err)
	}
	if monitorV2Scene.SceneID != monitorV2SceneID || monitorV2Preset.Setup.GetScene().GetRevision() != monitorV2Scene.Revision || len(monitorV2Inventory.Meshes[1].Primitives) != 2 || monitorV2Inventory.Meshes[1].Primitives[0].TriangleCount != 2 || monitorV2Inventory.Meshes[1].Primitives[0].Material != 3 || monitorV2Inventory.Meshes[1].Primitives[1].TriangleCount != 298 || monitorV2Inventory.Meshes[1].Primitives[1].Material != 1 || monitorV2Inventory.Materials[1].Name != "Dark inactive display glass" || monitorV2Inventory.Materials[1].HasBaseMap != monitorV2Inventory.Materials[3].HasBaseMap || monitorV2Inventory.Materials[1].HasNormal != monitorV2Inventory.Materials[3].HasNormal || monitorV2Inventory.Materials[1].NormalScale != monitorV2Inventory.Materials[3].NormalScale {
		t.Fatalf("v2 must preserve the original 298-triangle neutral bevel material and clone only its screen slot: %+v", monitorV2Inventory.Meshes[1].Primitives)
	}
	if monitorV2Device.Candidate.DisplayBindings[0].ScreenPrimitiveIndex == nil || *monitorV2Device.Candidate.DisplayBindings[0].ScreenPrimitiveIndex != 0 || monitorV2Device.Candidate.DisplayBindings[0].ScreenMaterialIndex == nil || *monitorV2Device.Candidate.DisplayBindings[0].ScreenMaterialIndex != 3 || monitorV2Preset.Setup.GetSceneSupportWorld().GetTranslationM().GetY() != .75 {
		t.Fatalf("v2 candidate must pin primitive 0/material 3 and the reviewed table pose: %+v", monitorV2Device.Candidate.DisplayBindings)
	}
	ipad, err := owners.InitializeReviewedIPadStarter(testCtx)
	if err != nil {
		t.Fatalf("initialize DatSketch iPad candidate through Library and device owners: %v", err)
	}
	ipadReplay, err := owners.InitializeReviewedIPadStarter(testCtx)
	if err != nil {
		t.Fatalf("replay DatSketch iPad candidate: %v", err)
	}
	if ipad.Bundle.ManifestSHA256 == "" || ipad.Bundle.ReviewSHA256 == "" || ipad.Bundle.Review.ID != "device-studio-e3-starter-local-render-review-v1-ipad-uv-normalized" || len(ipad.Bundle.Manifest.Artifacts) != 1 || ipad.Bundle.Manifest.Artifacts[0].SHA256 != ipadDerivativeSHA256 {
		t.Fatalf("iPad bytes were not published through the reviewed Library owner: %+v", ipad.Bundle)
	}
	ipadGLB, err := starterAssets.ReadFile("assets/ipad/ipad-mini-screen-uv-normalized.glb")
	if err != nil {
		t.Fatal(err)
	}
	screenMapping, err := rendering.ResolvePhysicalScreenGeometry(ipadGLB, ipad.Device.Candidate, "main_display")
	if err != nil {
		t.Fatalf("real saved iPad Library candidate must pass screen geometry preflight: %v", err)
	}
	if screenMapping.UVMin != [2]float64{0, 0} || screenMapping.UVMax != [2]float64{1, 1} || screenMapping.NodeIndex != 14 || screenMapping.PrimitiveIndex != 0 {
		t.Fatalf("normalized display mapping differs from reviewed candidate: %+v", screenMapping)
	}
	oldNeutral, err := starterAssets.ReadFile("assets/ipad/ipad-mini-neutral-screen.glb")
	if err != nil {
		t.Fatal(err)
	}
	oldCandidate := ipad.Device.Candidate
	oldCandidate.Geometry.SHA256 = ipadNeutralDerivativeSHA256
	oldCandidate.Geometry.SizeBytes = int64(len(oldNeutral))
	if _, err := rendering.ResolvePhysicalScreenGeometry(oldNeutral, oldCandidate, "main_display"); err == nil {
		t.Fatal("pre-normalization source UV overscan unexpectedly passed the full-rectangle screen contract")
	}
	if ipad.Device.State != devices.DeviceRevisionCandidate || ipad.Device.DeviceID != ipadCandidateID || ipad.Device.Candidate.Definition.Qualification != "unqualified" || ipad.Device.Candidate.Definition.IdentityStatus != devices.IdentityGeneric {
		t.Fatalf("iPad was promoted or misidentified beyond candidate status: %+v", ipad.Device)
	}
	if ipad.Device.SHA256 != ipadReplay.Device.SHA256 || ipad.Bundle.ManifestSHA256 != ipadReplay.Bundle.ManifestSHA256 || len(ipad.Device.Candidate.Parts) != 18 || ipadReplay.Device.Revision != ipad.Device.Revision {
		t.Fatalf("iPad initializer is not idempotent or omitted source semantic nodes: first=%+v replay=%+v", ipad.Device, ipadReplay.Device)
	}
	if ipad.Scene.SceneID != ipadSceneID || ipad.Scene.SHA256 == first.Scene.SHA256 || ipad.Scene.SHA256 != ipadReplay.Scene.SHA256 || first.Scene.SceneID != "phone-study-environment" {
		t.Fatalf("iPad fit must save/replay its own scene without changing the phone scene: ipad=%+v replay=%+v phone=%+v", ipad.Scene, ipadReplay.Scene, first.Scene)
	}
	if len(ipad.Scene.Candidate.CameraFrames) != 1 || ipad.Scene.Candidate.CameraFrames[0].ID != ipadCameraID || ipad.Scene.Candidate.CameraFrames[0].SensorFit != scenes.SensorFitHorizontal {
		t.Fatalf("iPad scene camera is not the separate horizontal-fit camera: %+v", ipad.Scene.Candidate.CameraFrames)
	}
	placedVertices, err := ipadGLBPlacedVertices(ipadGLB, ipad.Placement)
	if err != nil {
		t.Fatalf("transform actual iPad mesh vertices for camera-fit assertion: %v", err)
	}
	assertIPadCameraContainsVertices(t, placedVertices, ipad.Scene.Candidate.CameraFrames[0], 16.0/9.0, .10)
	rootPart, screenPart := false, false
	for _, part := range ipad.Device.Candidate.Parts {
		rootPart = rootPart || part.ID == ipadPartID(17) && part.NodeIndex == 17 && part.Role == "device_root"
		screenPart = screenPart || part.ID == ipadPartID(14) && part.NodeIndex == 14 && part.Role == "display_surface"
	}
	if !rootPart || !screenPart || ipad.Device.Candidate.DisplayBindings[0].PartID != ipadPartID(14) {
		t.Fatalf("iPad semantic hierarchy does not bind its measured screen and root: %+v", ipad.Device.Candidate.Parts)
	}
	if ipad.Placement.ProfileID != ReviewedIPadPlacementProfileID || ipad.Placement.DeviceRootRest.Translation.Y != .002631086703401778 || ipad.Placement.SceneSupportWorld.Translation.Y != .75 {
		t.Fatalf("iPad candidate placement lost its measured support-plane pose: %+v", ipad.Placement)
	}
	if ipad.Device.Candidate.Definition.Model != "iPad mini 6 — DatSketch model" || ipad.Device.Candidate.Definition.ModelCandidates[0].License != "CC BY 4.0" || ipad.Device.Candidate.Definition.ModelCandidates[0].Author != "DatSketch" || ipad.Device.Candidate.Definition.ModelCandidates[0].SourcePath != "" {
		t.Fatalf("iPad creator attribution or source identity is missing: %+v", ipad.Device.Candidate.Definition.ModelCandidates)
	}
	ipadV3, err := owners.InitializeReviewedIPadEnvironmentV3(testCtx)
	if err != nil {
		t.Fatalf("initialize separate iPad environment v3 scene: %v", err)
	}
	ipadV3Replay, err := owners.InitializeReviewedIPadEnvironmentV3(testCtx)
	if err != nil {
		t.Fatalf("replay iPad environment v3 scene: %v", err)
	}
	geometryBound := false
	for _, binding := range ipadV3.Scene.Candidate.Artifacts {
		if binding.Ref.ArtifactID == ipadEnvironmentV3ArtifactID && binding.Ref.SHA256 == ipadEnvironmentV3SHA256 {
			geometryBound = true
		}
	}
	if ipadV3.Device.DeviceID != ipadOrientedCandidateID || ipadV3.Device.SHA256 != ipadV3Replay.Device.SHA256 || ipadV3.Device.Candidate.Geometry.SHA256 != ipadOrientedGLBSHA256 ||
		ipadV3.Scene.SceneID != ipadEnvironmentV3SceneID || ipadV3.Scene.SHA256 != ipadV3Replay.Scene.SHA256 ||
		ipadV3.SceneBundle.Manifest.BundleID != ipadEnvironmentV3BundleID || ipadV3.SceneBundle.ManifestSHA256 == "" || ipadV3.SceneBundle.ReviewSHA256 == "" ||
		ipadV3.SceneBundle.ManifestSHA256 != ipadV3Replay.SceneBundle.ManifestSHA256 || ipadV3.SceneBundle.Review.ID != "device-studio-e3-starter-local-render-review-v1-ipad-environment-v3" ||
		ipadV3.Placement.SelectedCameraID != ipadEnvironmentV3CameraID || len(ipadV3.Scene.Candidate.Artifacts) != 5 || !geometryBound {
		t.Fatalf("iPad environment v3 was not published and replayed as a separate scene candidate: first=%+v replay=%+v", ipadV3, ipadV3Replay)
	}
	if orientedHead, err := owners.Scenes.GetHead(testCtx, ipadOrientedSceneID); err != nil || orientedHead.SHA256 == ipadV3.Scene.SHA256 || orientedHead.SceneID != ipadOrientedSceneID {
		t.Fatalf("adding iPad environment v3 changed the prior oriented scene: head=%+v err=%v", orientedHead, err)
	}
	t.Logf("iPad portable candidate revision=%d sha256=%s", ipad.Device.Revision, ipad.Device.SHA256)
	if first.Phone.State != devices.DeviceRevisionCandidate || first.Scene.State != "candidate" || first.Motion.State != "candidate" || first.Phone.Candidate.Definition.Qualification != "unqualified" {
		t.Fatalf("starter promoted a candidate into qualification: phone=%q scene=%q motion=%q", first.Phone.State, first.Scene.State, first.Motion.State)
	}
	placement := first.PhoneStudyPlacement
	if placement.ProfileID != ReviewedPhoneStudyPlacementProfileID || placement.ProfileVersion != 1 || placement.DisplayID != "main_display" || placement.DeviceSupportID != "device-root-support" || placement.SceneSupportID != "table-top-flat" || placement.SelectedCameraID != "phone-study-camera" {
		t.Fatalf("reviewed starter placement IDs/profile changed: %+v", placement)
	}
	if first.Phone.DeviceID != "authored-phone-neutral" || first.Scene.SceneID != "phone-study-environment" || first.Motion.MotionID != "phone-still-identity" {
		t.Fatal("reviewed placement is not returned with the exact starter candidate references")
	}
	if placement.SceneSupportWorld.Translation != (physical.Vec3{Y: .75}) || placement.SceneSupportWorld.Rotation != (physical.QuatXYZW{W: 1}) || placement.SceneSupportWorld.Scale != (physical.Vec3{X: 1, Y: 1, Z: 1}) || placement.SupportPivotM != (physical.Vec3{Y: .0071073174476624}) {
		t.Fatalf("reviewed phone support transform changed: %+v", placement)
	}
	wantRotation := physical.QuatXYZW{X: -.6885449545442038, Y: -.08161248882759843, Z: -.07844978823902271, W: .7163036213536392}
	if placement.DeviceRootRest.Translation != (physical.Vec3{}) || placement.DeviceRootRest.Rotation != wantRotation || placement.DeviceRootRest.Scale != (physical.Vec3{X: 1, Y: 1, Z: 1}) {
		t.Fatalf("reviewed phone root pose changed from the saved API3 numeric fixture: %+v", placement.DeviceRootRest)
	}
	dimensions := first.Phone.Candidate.Definition.Dimensions["portrait_unrotated"]
	if dimensions.Width.Value == nil || dimensions.Height.Value == nil || dimensions.Depth.Value == nil || dimensions.Width.Method != "derived" || dimensions.Height.Method != "derived" || dimensions.Depth.Method != "derived" {
		t.Fatalf("phone candidate omitted geometry-derived envelope facts: %+v", dimensions)
	}
	if math.Abs(first.Phone.Candidate.SourceToCanonical.Translation.Y)+math.Abs(first.Phone.Candidate.SourceToCanonical.Translation.Z) < 0.001 {
		t.Fatalf("candidate omitted geometry-derived source recentering: %+v", first.Phone.Candidate.SourceToCanonical.Translation)
	}
	testPool, err := db.PoolForContext(testCtx)
	if err != nil {
		t.Fatal(err)
	}
	var leasedBundles, primaryBundles int
	if err := testPool.QueryRowContext(testCtx, `SELECT count(*) FROM library_bundle_revisions`).Scan(&leasedBundles); err != nil {
		t.Fatal(err)
	}
	if err := db.Primary().QueryRowContext(ctx, `SELECT count(*) FROM library_bundle_revisions`).Scan(&primaryBundles); err != nil {
		t.Fatal(err)
	}
	if leasedBundles != 8 || primaryBundles != 0 {
		t.Fatalf("starter route isolation failed: leased bundles=%d primary bundles=%d", leasedBundles, primaryBundles)
	}
	if stats := roots.LeaseStats(); stats.TestRootWrites == 0 || stats.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("starter file writes escaped the pinned test roots: %+v", stats)
	}

	// Exercise the public shot service against the exact saved starter revisions
	// and the same pinned test route used by the bundle/candidate owners.
	project, err := projectService.Create(testCtx, projects.CreateInput{Name: "Physical shot integration"})
	if err != nil {
		t.Fatal(err)
	}
	timingAPI := os.Getenv("DEVICE_STUDIO_E3_TIMING_API_BIN")
	screenBytes := starterTestScreenPNG(t)
	if timingAPI != "" {
		screenBytes = starterApprovedTimingScreenPNG(t)
	}
	screen, err := mediaService.Import(testCtx, project.ID, "screen.png", int64(len(screenBytes)), bytes.NewReader(screenBytes))
	if err != nil {
		t.Fatalf("import project-owned screen raster: %v", err)
	}
	videoResolution := starterResolveRecordingLayout(t, testCtx, project.ID, first, owners, mediaService)
	if videoResolution.RecordingLayout == nil || videoResolution.PresentationProfile != presentations.SimulatedCompositorProfile || videoResolution.RecordingLayout.ProfileID == videoResolution.PresentationProfile {
		t.Fatalf("owned resolver did not retain a distinct verified recording layout: %+v", videoResolution)
	}
	shotService := shots.NewService(shots.NewSQLiteRepository(db), projectService, mediaService, nil, nil)
	shots.SetPhysicalResolver(shotService, owners.Shots)
	shotSnapshot := starterPhysicalShotSnapshot(first, screen.ID)
	if timingAPI != "" {
		config, err := png.DecodeConfig(bytes.NewReader(screenBytes))
		if err != nil {
			t.Fatalf("read timing screen raster dimensions: %v", err)
		}
		shotSnapshot.GetPhysicalBinding().ScreenWidthPixels = uint32(config.Width)
		shotSnapshot.GetPhysicalBinding().ScreenHeightPixels = uint32(config.Height)
	}
	createdShot, createdRevision, err := shotService.Create(testCtx, project.ID, "Starter phone still", shotSnapshot)
	if err != nil {
		t.Fatalf("create physical shot from exact candidate revisions: %v", err)
	}
	if createdRevision.Revision != 1 || !createdRevision.RenderEligible || createdRevision.Physical == nil || createdRevision.Physical.DeviceSHA256 != first.Phone.SHA256 || createdRevision.Physical.SceneSHA256 != first.Scene.SHA256 || createdRevision.Physical.MotionSHA256 != first.Motion.SHA256 {
		t.Fatalf("created shot did not retain exact saved candidate revisions: %+v", createdRevision)
	}
	updatedShot, updatedRevision, err := shotService.Save(testCtx, createdShot.ID, 1, "Starter phone still v2", shotSnapshot)
	if err != nil {
		t.Fatalf("save immutable physical shot revision: %v", err)
	}
	if updatedShot.HeadRevision != 2 || updatedRevision.Revision != 2 {
		t.Fatalf("physical shot save did not advance one revision: shot=%+v revision=%d", updatedShot, updatedRevision.Revision)
	}
	reopenedShot, reopenedRevision, err := shotService.Get(testCtx, createdShot.ID)
	if err != nil {
		t.Fatalf("reopen saved physical shot: %v", err)
	}
	if reopenedShot.HeadRevision != 2 || reopenedRevision.Physical != nil || reopenedRevision.RecipeSHA256 != updatedRevision.RecipeSHA256 || reopenedRevision.SceneSHA256 != first.Scene.SHA256 || reopenedRevision.Snapshot.GetPhysicalBinding().GetDevice().GetSha256() != first.Phone.SHA256 || reopenedRevision.Snapshot.GetPhysicalBinding().GetScene().GetSha256() != first.Scene.SHA256 || reopenedRevision.Snapshot.GetPhysicalBinding().GetMotion().GetSha256() != first.Motion.SHA256 {
		t.Fatalf("reopened shot changed its immutable physical references: shot=%+v revision=%+v", reopenedShot, reopenedRevision)
	}
	compiled, err := shotService.Compile(testCtx, createdShot.ID, 2)
	if err != nil {
		t.Fatalf("recompile reopened physical revision: %v", err)
	}
	if compiled.Physical == nil || compiled.RecipeSHA256 != updatedRevision.RecipeSHA256 || compiled.Physical.DeviceSHA256 != first.Phone.SHA256 || compiled.Physical.SceneSHA256 != first.Scene.SHA256 || compiled.Physical.MotionSHA256 != first.Motion.SHA256 {
		t.Fatalf("compiled shot changed its immutable physical closure: %+v", compiled)
	}

	// Progress the saved revision through the same coordinator and routed job
	// owner used by the public render endpoint. The fake operation owner checks
	// the staged worker contract and deliberately does not launch Blender.
	workerPath := filepath.Join(t.TempDir(), "device-studio-api")
	if err := os.WriteFile(workerPath, []byte("test-worker-executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	physicalProfile := starterTestBlenderProfile()
	profileBytes, err := protojson.Marshal(physicalProfile)
	if err != nil {
		t.Fatal(err)
	}
	ops := &starterPhysicalOperationOwner{}
	renderBlobs, err := routedblobstore.New(roots, "renderjobs")
	if err != nil {
		t.Fatal(err)
	}
	timing := &physicalSubmitTiming{}
	var shotResolver renderjobs.ShotResolver = shotService
	var jobService jobs.Service = jobs.NewService(jobs.NewSQLiteRepository(db), time.Now)
	var blobStore blobstore.BlobStore = renderBlobs
	var artifactResolver renderjobs.PhysicalArtifactResolver = owners.Library
	profileResolver := func(context.Context) ([]byte, *cliv1.BlenderToolProfile, error) {
		return profileBytes, physicalProfile, nil
	}
	if timingAPI != "" {
		if !filepath.IsAbs(timingAPI) {
			t.Fatalf("timing API binary path must be absolute: %q", timingAPI)
		}
		if _, err := os.Stat(timingAPI); err != nil {
			t.Fatalf("inspect timing API binary: %v", err)
		}
		shotResolver = timedPhysicalShotResolver{ShotResolver: shotResolver, timing: timing}
		jobService = timedPhysicalJobs{Service: jobService, timing: timing}
		blobStore = timedPhysicalBlobStore{BlobStore: blobStore, timing: timing}
		artifactResolver = timedPhysicalArtifacts{PhysicalArtifactResolver: artifactResolver, timing: timing}
		profileResolver = func(ctx context.Context) ([]byte, *cliv1.BlenderToolProfile, error) {
			started := time.Now()
			data, profile, err := rendering.ResolvePhysicalBlenderProfile(ctx)
			timing.record("blender_profile_resolve", time.Since(started), 0)
			return data, profile, err
		}
	}
	coordinator, err := renderjobs.New(renderjobs.Dependencies{
		Shots: shotResolver, Media: mediaService, Jobs: jobService,
		Blobs: blobStore, Roots: roots, Ops: ops,
		Exe: func() (string, error) {
			if timingAPI != "" {
				return timingAPI, nil
			}
			return workerPath, nil
		},
		PhysicalArtifacts: artifactResolver, PhysicalProfile: profileResolver,
		PhysicalAdapter: rendering.PhysicalAdapterBytes,
	})
	if err != nil {
		t.Fatalf("construct physical render coordinator: %v", err)
	}
	if timingAPI != "" {
		for _, key := range []string{"physical-preview-cold", "physical-preview-warm"} {
			started := time.Now()
			job, created, submitErr := coordinator.Submit(testCtx, renderjobs.SubmitRequest{
				ProjectID: project.ID, ShotID: createdShot.ID, Revision: 2, IdempotencyKey: key, OutputProfile: renderjobs.OutputProfilePhysicalPreviewPNG,
			})
			if submitErr != nil || !created || job.ID == "" {
				t.Fatalf("timed physical preview submit %q failed: created=%v err=%v", key, created, submitErr)
			}
			timing.record(key, time.Since(started), 0)
		}
		apiBytes, err := readTimingAPIExecutable(timingAPI)
		if err != nil {
			t.Fatal(err)
		}
		apiHash := sha256.Sum256(apiBytes)
		timingJSON, err := json.Marshal(struct {
			APIPath string               `json:"api_path"`
			APISize int                  `json:"api_size_bytes"`
			APISHA  string               `json:"api_sha256"`
			Source  string               `json:"screen_source_sha256"`
			Width   int                  `json:"screen_source_width"`
			Height  int                  `json:"screen_source_height"`
			Spans   []physicalSubmitSpan `json:"spans"`
		}{timingAPI, len(apiBytes), hex.EncodeToString(apiHash[:]), "c4d846f0b79b839a6c7e5ef1aba0ca97cda24f96a14741eeb5410bb14ed19eb8", 1206, 2622, timing.snapshot()})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("PHYSICAL_PREVIEW_SUBMIT_TIMING %s", timingJSON)
		return
	}
	job, created, err := coordinator.Submit(testCtx, renderjobs.SubmitRequest{
		ProjectID: project.ID, ShotID: createdShot.ID, Revision: 2, IdempotencyKey: "starter-physical-still-v1", OutputProfile: renderjobs.OutputProfilePNG,
	})
	if err != nil || !created {
		t.Fatalf("submit saved physical revision: created=%v err=%v", created, err)
	}
	advanced, err := coordinator.Advance(testCtx, job.ID)
	if err != nil {
		t.Fatalf("advance saved physical revision to the operation owner: %v", err)
	}
	if advanced.State != jobs.Running || ops.starts != 1 || ops.manifest.JobID != job.ID || ops.manifest.AttemptID == "" || ops.manifest.Device.CanonicalSHA != first.Phone.SHA256 || ops.manifest.Scene.CanonicalSHA != first.Scene.SHA256 || ops.manifest.Motion.CanonicalSHA != first.Motion.SHA256 {
		t.Fatalf("physical operation did not receive the exact immutable saved closure: state=%s starts=%d manifest=%+v", advanced.State, ops.starts, ops.manifest)
	}

	// Submit the actual Library-published iPad candidate through the same job
	// closure builder with a stub worker/profile. Submit builds and validates the
	// portable manifest but does not advance or launch the native operation.
	ipadScreenBytes := ipadReviewScreenPNG(t)
	ipadScreen, err := mediaService.Import(testCtx, project.ID, "ipad-screen.png", int64(len(ipadScreenBytes)), bytes.NewReader(ipadScreenBytes))
	if err != nil {
		t.Fatalf("import iPad review screen: %v", err)
	}
	ipadShot, ipadRevision, err := shotService.Create(testCtx, project.ID, "iPad portable candidate", ipadStillSnapshot(ipad, ipadScreen.ID, ipad.Placement, shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_CPU))
	if err != nil {
		t.Fatalf("create shot from the real saved iPad candidate: %v", err)
	}
	if !ipadRevision.RenderEligible {
		t.Fatalf("saved iPad candidate is not render eligible: %+v", ipadRevision.Issues)
	}
	ipadJob, ipadCreated, err := coordinator.Submit(testCtx, renderjobs.SubmitRequest{
		ProjectID: project.ID, ShotID: ipadShot.ID, Revision: ipadRevision.Revision, IdempotencyKey: "starter-ipad-portable-closure-v1", OutputProfile: renderjobs.OutputProfilePNG,
	})
	if err != nil || !ipadCreated {
		t.Fatalf("submit real iPad candidate through physical closure preflight: created=%v err=%v", ipadCreated, err)
	}
	var ipadClosure bool
	for _, binding := range ipadJob.Input.Bindings {
		ipadClosure = ipadClosure || binding.Role == "physical_closure"
	}
	if !ipadClosure {
		t.Fatal("submitted iPad job omitted the portable physical closure")
	}
	if ops.starts != 1 {
		t.Fatalf("iPad Submit unexpectedly advanced the native operation owner: starts=%d", ops.starts)
	}

	tamperedJob, created, err := coordinator.Submit(testCtx, renderjobs.SubmitRequest{
		ProjectID: project.ID, ShotID: createdShot.ID, Revision: 2, IdempotencyKey: "starter-physical-still-tamper-v1", OutputProfile: renderjobs.OutputProfilePNG,
	})
	if err != nil || !created {
		t.Fatalf("submit tamper-check physical job: created=%v err=%v", created, err)
	}
	var closureKey string
	for _, binding := range tamperedJob.Input.Bindings {
		if binding.Role == "physical_closure" {
			closureKey = binding.ArtifactID
		}
	}
	if closureKey == "" {
		t.Fatal("physical job omitted its closure binding")
	}
	closureReader, _, err := renderBlobs.Get(testCtx, closureKey)
	if err != nil {
		t.Fatal(err)
	}
	closureBytes, readErr := io.ReadAll(closureReader)
	closeErr := closureReader.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		t.Fatal(err)
	}
	if len(closureBytes) == 0 {
		t.Fatal("stored physical closure is empty")
	}
	closureBytes[len(closureBytes)-1] ^= 1
	if err := renderBlobs.Put(testCtx, closureKey, bytes.NewReader(closureBytes), "application/json"); err != nil {
		t.Fatalf("tamper with stored closure fixture: %v", err)
	}
	if _, err := coordinator.Advance(testCtx, tamperedJob.ID); err == nil {
		t.Fatal("coordinator dispatched a job whose physical closure bytes changed after submission")
	}
	if ops.starts != 1 {
		t.Fatalf("tampered physical closure reached native operation owner: starts=%d", ops.starts)
	}
}

type starterRecordingMediaStub struct {
	projects.SourceMediaService
	media projects.SourceMedia
}

func seedPersistedMonitorV1State(t *testing.T, owners Owners, ctx context.Context) MonitorStarterState {
	t.Helper()
	pinned, err := owners.Starter.pinRoute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	phone, err := owners.Starter.initializePinned(pinned)
	if err != nil {
		t.Fatal(err)
	}
	laptop, err := owners.Starter.initializeLaptopCandidatePinned(pinned)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := owners.Starter.publishMonitorBundle(pinned, monitorV1Seed())
	if err != nil {
		t.Fatalf("seed persisted v1 monitor bundle: %v", err)
	}
	if bundle.ManifestSHA256 != "86b14e4c43665991b8760c8341c9bdaaaf494f8e76c8e6b5c69580914869f250" || bundle.ReviewSHA256 != "46bf98c3537b100852604e1c3bdfc34d79929012faf10ac4120d6b2ace629f5a" {
		t.Fatalf("v1 monitor bundle generation differs from persisted revision: manifest=%s review=%s", bundle.ManifestSHA256, bundle.ReviewSHA256)
	}
	deviceJSON, err := os.ReadFile("testdata/monitor-v1-candidate-revision-1.json")
	if err != nil {
		t.Fatal(err)
	}
	var deviceCandidate devices.DeviceCandidate
	if err := json.Unmarshal(deviceJSON, &deviceCandidate); err != nil {
		t.Fatal(err)
	}
	device, err := owners.Devices.SaveCandidate(pinned, deviceCandidate, 0)
	if err != nil || device.SHA256 != "297eccc8ba18069a7d71d125d9bb2fb83ca2445d011f88e244ded31d1b190b6f" {
		t.Fatalf("seed exact persisted v1 device revision: sha=%s err=%v", device.SHA256, err)
	}
	sceneJSON, err := os.ReadFile("testdata/monitor-v1-scene-revision-1.json")
	if err != nil {
		t.Fatal(err)
	}
	var sceneCandidate scenes.SceneCandidate
	if err := json.Unmarshal(sceneJSON, &sceneCandidate); err != nil {
		t.Fatal(err)
	}
	scene, err := owners.Scenes.SaveCandidate(pinned, sceneCandidate, 0)
	if err != nil || scene.SHA256 != "f1d46273366fe6f20199ab30f2488953ad7b1adc4c5a3541d8af31b343858a0e" {
		t.Fatalf("seed exact persisted v1 scene revision: sha=%s err=%v", scene.SHA256, err)
	}
	presetJSON, err := os.ReadFile("testdata/monitor-v1-preset-revision-1.json")
	if err != nil {
		t.Fatal(err)
	}
	var presetSnapshot struct {
		PresetID string          `json:"preset_id"`
		Name     string          `json:"name"`
		Setup    json.RawMessage `json:"setup"`
	}
	if err := json.Unmarshal(presetJSON, &presetSnapshot); err != nil {
		t.Fatal(err)
	}
	setup := &shotsv1.PhysicalShotSetup{}
	if err := protojson.Unmarshal(presetSnapshot.Setup, setup); err != nil {
		t.Fatal(err)
	}
	preset, err := owners.Presets.Save(pinned, presetSnapshot.PresetID, 0, presetSnapshot.Name, setup)
	if err != nil || preset.SHA256 != "69a0b2d35257776f2ade440f22a600f31ab08a18fa28c8da6a9ffab842862647" {
		t.Fatalf("seed exact persisted v1 preset revision: sha=%s err=%v", preset.SHA256, err)
	}
	state, err := owners.Starter.initializeMonitorCandidatePinned(pinned, laptop.FramedScene, phone.Motion)
	if err != nil {
		t.Fatalf("reconcile old persisted v1 monitor state: %v", err)
	}
	if state.Device.SHA256 != device.SHA256 || state.Scene.SHA256 != scene.SHA256 || state.Preset.SHA256 != preset.SHA256 || state.Bundle.ManifestSHA256 != bundle.ManifestSHA256 {
		t.Fatalf("persisted v1 monitor state did not remain exact: device=%s scene=%s preset=%s bundle=%s", state.Device.SHA256, state.Scene.SHA256, state.Preset.SHA256, state.Bundle.ManifestSHA256)
	}
	return state
}

func (s starterRecordingMediaStub) Get(ctx context.Context, projectID, mediaID string) (projects.SourceMedia, error) {
	if projectID == s.media.ProjectID && mediaID == s.media.ID {
		return s.media, nil
	}
	return s.SourceMediaService.Get(ctx, projectID, mediaID)
}

func starterResolveRecordingLayout(t *testing.T, ctx context.Context, projectID string, state StarterState, owners Owners, mediaOwner projects.SourceMediaService) *shots.PhysicalResolution {
	t.Helper()
	start, end, stream := int64(0), int64(120), int64(0)
	sourceSHA, clockSHA := strings.Repeat("a", 64), strings.Repeat("b", 64)
	receipt := &projects.InspectionReceipt{
		Version: 1, ID: "accepted-receipt", SHA256: strings.Repeat("c", 64), SourceSHA256: sourceSHA, SourceSizeBytes: 1024,
		JobID: "accepted-job", AttemptID: "accepted-attempt", Classification: "accepted", VideoStreamIndex: &stream,
		TimeBaseNumerator: 1, TimeBaseDenominator: 30, StreamStartPTS: &start, SourceEndPTS: &end, LastFrameDurationTicks: 2,
		VideoWidth: 1206, VideoHeight: 2400, ClockSHA256: clockSHA, ClockArtifactRef: "accepted-clock", ClockArtifactSizeBytes: 16,
	}
	video := projects.SourceMedia{
		ID: "accepted-video", ProjectID: projectID, SHA256: sourceSHA, SizeBytes: 1024, Kind: projects.MediaKindVideo,
		Readiness: projects.ReadinessReady, InspectionReceipt: receipt,
	}
	mediaStub := starterRecordingMediaStub{SourceMediaService: mediaOwner, media: video}
	resolver, err := shots.NewPhysicalResolver(owners.Devices, owners.Scenes, owners.Motions, owners.Library, mediaStub)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := starterPhysicalShotSnapshot(state, video.ID)
	snapshot.PhysicalBinding.OutputProfile = shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_UNSPECIFIED
	snapshot.PhysicalBinding.MotionSampleTick = nil
	snapshot.PhysicalBinding.MotionOutput = &shotsv1.PhysicalMotionOutputBinding{
		ProfileId: rendering.PhysicalMotionOutput, ProfileVersion: 1, Width: 1080, Height: 1920,
		OutputRateNumerator: 24, OutputRateDenominator: 1, FrameCount: 24,
	}
	snapshot.Presentation = &shotsv1.ScreenPresentation{
		SourceMediaId: video.ID, CaptureArea: shotsv1.CaptureArea_CAPTURE_AREA_APP_ONLY,
		Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM,
		PhoneSystemChrome: &shotsv1.PhoneSystemChrome{
			TopFill: &shotsv1.PresentationRGB{Red: 24, Green: 26, Blue: 29}, BottomFill: &shotsv1.PresentationRGB{Red: 24, Green: 26, Blue: 29},
			StatusInk: &shotsv1.PresentationRGB{Red: 245, Green: 245, Blue: 245}, HomeInk: &shotsv1.PresentationRGB{Red: 245, Green: 245, Blue: 245},
		},
	}
	timing := &shotsv1.VideoTiming{
		SourceMediaId: video.ID, SourceSha256: sourceSHA, SourceSizeBytes: video.SizeBytes,
		InspectionJobId: receipt.JobID, InspectionAttemptId: receipt.AttemptID, InspectionReceiptId: receipt.ID, InspectionReceiptSha256: receipt.SHA256,
		ClockArtifactId: receipt.ClockArtifactRef, ClockSha256: receipt.ClockSHA256, ClockSizeBytes: receipt.ClockArtifactSizeBytes,
		TimeBaseNum: 1, TimeBaseDen: 30, StreamStartPts: &start, SourceEndPts: end, LastFrameDurationTicks: receipt.LastFrameDurationTicks,
		TrimStartPts: 0, TrimEndPts: 30, PlaybackRateNum: 1, PlaybackRateDen: 1, OutputRateNum: 24, OutputRateDen: 1,
		EofPolicy: shotsv1.VideoEOFPolicy_VIDEO_EOF_POLICY_REJECT, VideoCompositionProfileId: shots.VideoProfileFixed, VideoCompositionProfileVersion: 1,
	}
	resolved, err := resolver.Resolve(ctx, projectID, snapshot.PhysicalBinding, snapshot.Presentation, timing, presentations.SemanticVersionV3)
	if err != nil {
		t.Fatalf("owned resolver rejected accepted recording fixture: %v", err)
	}
	return resolved
}

type starterPhysicalOperationOwner struct {
	starts   int
	manifest rendering.PhysicalRenderManifest
}

func (*starterPhysicalOperationOwner) NewIntent(owner, attempt string, spec platform.OperationSpec) (platform.OperationIntent, error) {
	return platform.NewOperationIntent(owner, attempt, spec)
}

func (o *starterPhysicalOperationOwner) Start(_ context.Context, _ platform.OperationIntent, spec platform.OperationSpec) (*platform.OwnedOperation, error) {
	o.starts++
	if len(spec.Args) != 3 || spec.Args[0] != "physical-render-worker" || spec.Args[1] != "--manifest" {
		return nil, errors.New("unexpected operation argv")
	}
	data, err := os.ReadFile(spec.Args[2])
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &o.manifest); err != nil {
		return nil, err
	}
	if err := rendering.ValidatePhysicalRenderManifest(o.manifest); err != nil {
		return nil, fmt.Errorf("staged physical manifest is invalid: %w", err)
	}
	return nil, nil
}

func (*starterPhysicalOperationOwner) Lookup(context.Context, platform.OperationIntent) (platform.OperationRef, error) {
	return platform.OperationRef{}, platform.ErrOperationNotFound
}
func (*starterPhysicalOperationOwner) Inspect(context.Context, platform.OperationRef) (platform.OperationObservation, error) {
	return platform.OperationObservation{}, platform.ErrOperationNotFound
}
func (*starterPhysicalOperationOwner) Stop(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error) {
	return platform.OperationStopReceipt{}, platform.ErrOperationNotFound
}
func (*starterPhysicalOperationOwner) Release(context.Context, platform.OperationRef) error {
	return nil
}

func starterTestBlenderProfile() *cliv1.BlenderToolProfile {
	return &cliv1.BlenderToolProfile{
		SchemaVersion: 1, ProfileId: rendering.PhysicalBlenderProfileID,
		State:            cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT,
		ArtifactIdentity: rendering.PhysicalBlenderArtifact, ExpectedTreeSha256: rendering.PhysicalBlenderTreeSHA,
		Os: "linux", Architecture: "amd64",
		Blender: &cliv1.RendererToolBinary{
			Path:   filepath.Join("/home/test/.vrooli/tools/blender", rendering.PhysicalBlenderArchive, "blender"),
			Sha256: rendering.PhysicalBlenderBinarySHA, SizeBytes: 300_000_000,
			Version: rendering.PhysicalBlenderVersion, ArchiveMemberPath: rendering.PhysicalBlenderArchive + "/blender",
		},
	}
}

func starterTestScreenPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 444, 965))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8((x * 17) % 255), G: uint8((y * 11) % 255), B: 92, A: 255})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func starterApprovedTimingScreenPNG(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "docs", "internal", "goal", "evidence", "E3", "physical-preview", "lease10", "leased-root", "data", "renderjobs", "renderjobs", "inputs", "c4d846f0b79b839a6c7e5ef1aba0ca97cda24f96a14741eeb5410bb14ed19eb8", "physical-screen.png")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pinned approved timing screen: %v", err)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != "c4d846f0b79b839a6c7e5ef1aba0ca97cda24f96a14741eeb5410bb14ed19eb8" {
		t.Fatal("timing screen bytes do not match the approved immutable source")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width != 1206 || config.Height != 2622 {
		t.Fatalf("timing screen raster is not the pinned 1206x2622 PNG: config=%+v err=%v", config, err)
	}
	return data
}

type physicalSubmitSpan struct {
	Name         string  `json:"name"`
	Milliseconds float64 `json:"milliseconds"`
	Bytes        int64   `json:"bytes,omitempty"`
}

type physicalSubmitTiming struct {
	mu    sync.Mutex
	spans []physicalSubmitSpan
}

func (t *physicalSubmitTiming) record(name string, elapsed time.Duration, size int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spans = append(t.spans, physicalSubmitSpan{Name: name, Milliseconds: float64(elapsed) / float64(time.Millisecond), Bytes: size})
}

func (t *physicalSubmitTiming) snapshot() []physicalSubmitSpan {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]physicalSubmitSpan(nil), t.spans...)
}

type timedPhysicalShotResolver struct {
	renderjobs.ShotResolver
	timing *physicalSubmitTiming
}

func (r timedPhysicalShotResolver) ResolveForRender(ctx context.Context, id string, revision int64) (shots.ResolvedShot, error) {
	started := time.Now()
	resolved, err := r.ShotResolver.ResolveForRender(ctx, id, revision)
	r.timing.record("shots_resolve_for_render", time.Since(started), 0)
	return resolved, err
}

type timedPhysicalJobs struct {
	jobs.Service
	timing *physicalSubmitTiming
}

func (j timedPhysicalJobs) Submit(ctx context.Context, input jobs.SubmitInput) (jobs.Job, bool, error) {
	started := time.Now()
	job, created, err := j.Service.Submit(ctx, input)
	j.timing.record("jobs_submit", time.Since(started), 0)
	return job, created, err
}

type timedPhysicalBlobStore struct {
	blobstore.BlobStore
	timing *physicalSubmitTiming
}

func (b timedPhysicalBlobStore) Put(ctx context.Context, key string, reader io.Reader, mime string) error {
	started := time.Now()
	counted := &countedPhysicalSubmitReader{Reader: reader}
	err := b.BlobStore.Put(ctx, key, counted, mime)
	b.timing.record("blob_put", time.Since(started), counted.bytes)
	return err
}

func (b timedPhysicalBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	started := time.Now()
	reader, mime, err := b.BlobStore.Get(ctx, key)
	b.timing.record("blob_get_open", time.Since(started), 0)
	if err != nil {
		return nil, mime, err
	}
	return &timedPhysicalSubmitReader{ReadCloser: reader, timing: b.timing, name: "blob_get_read"}, mime, nil
}

type timedPhysicalArtifacts struct {
	renderjobs.PhysicalArtifactResolver
	timing *physicalSubmitTiming
}

func (r timedPhysicalArtifacts) Resolve(ctx context.Context, ref physical.ArtifactRef, use library.Use) (library.ResolvedArtifact, error) {
	started := time.Now()
	resolved, err := r.PhysicalArtifactResolver.Resolve(ctx, ref, use)
	r.timing.record("physical_artifact_resolve", time.Since(started), ref.SizeBytes)
	if err == nil && resolved.Reader != nil {
		resolved.Reader = &timedPhysicalSubmitReader{ReadCloser: resolved.Reader, timing: r.timing, name: "physical_artifact_read", expected: ref.SizeBytes}
	}
	return resolved, err
}

type countedPhysicalSubmitReader struct {
	io.Reader
	bytes int64
}

func (r *countedPhysicalSubmitReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.bytes += int64(n)
	return n, err
}

type timedPhysicalSubmitReader struct {
	io.ReadCloser
	timing   *physicalSubmitTiming
	name     string
	expected int64
	started  time.Time
	bytes    int64
	once     sync.Once
}

func (r *timedPhysicalSubmitReader) Read(p []byte) (int, error) {
	if r.started.IsZero() {
		r.started = time.Now()
	}
	n, err := r.ReadCloser.Read(p)
	r.bytes += int64(n)
	return n, err
}

func (r *timedPhysicalSubmitReader) Close() error {
	err := r.ReadCloser.Close()
	r.once.Do(func() {
		elapsed := time.Duration(0)
		if !r.started.IsZero() {
			elapsed = time.Since(r.started)
		}
		if r.expected > 0 && r.bytes != r.expected {
			r.name += "_partial"
		}
		r.timing.record(r.name, elapsed, r.bytes)
	})
	return err
}

func readTimingAPIExecutable(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 256<<20 {
		return nil, errors.New("timing API binary is not a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, (256<<20)+1))
	if err != nil || int64(len(data)) != info.Size() {
		return nil, errors.New("timing API binary changed or could not be read completely")
	}
	return data, nil
}

func starterPhysicalShotSnapshot(state StarterState, sourceMediaID string) *shotsv1.ShotSnapshot {
	placement := state.PhoneStudyPlacement
	return &shotsv1.ShotSnapshot{
		PhysicalBinding: &shotsv1.PhysicalShotBinding{
			Device:    &shotsv1.ComponentRevisionRef{Id: state.Phone.DeviceID, Revision: state.Phone.Revision, Sha256: state.Phone.SHA256},
			Scene:     &shotsv1.ComponentRevisionRef{Id: state.Scene.SceneID, Revision: state.Scene.Revision, Sha256: state.Scene.SHA256},
			Motion:    &shotsv1.ComponentRevisionRef{Id: state.Motion.MotionID, Revision: state.Motion.Revision, Sha256: state.Motion.SHA256},
			DisplayId: placement.DisplayID, DeviceSupportId: placement.DeviceSupportID, SceneSupportId: placement.SceneSupportID, CameraId: placement.SelectedCameraID,
			SceneSupportWorld: &physicalv1.TRS{TranslationM: &physicalv1.Vec3{Y: placement.SceneSupportWorld.Translation.Y}, RotationXyzw: &physicalv1.QuatXYZW{X: placement.SceneSupportWorld.Rotation.X, Y: placement.SceneSupportWorld.Rotation.Y, Z: placement.SceneSupportWorld.Rotation.Z, W: placement.SceneSupportWorld.Rotation.W}, Scale: &physicalv1.Vec3{X: placement.SceneSupportWorld.Scale.X, Y: placement.SceneSupportWorld.Scale.Y, Z: placement.SceneSupportWorld.Scale.Z}},
			SupportPivotM:     &physicalv1.Vec3{X: placement.SupportPivotM.X, Y: placement.SupportPivotM.Y, Z: placement.SupportPivotM.Z},
			DeviceRootRest:    &physicalv1.TRS{TranslationM: &physicalv1.Vec3{X: placement.DeviceRootRest.Translation.X, Y: placement.DeviceRootRest.Translation.Y, Z: placement.DeviceRootRest.Translation.Z}, RotationXyzw: &physicalv1.QuatXYZW{X: placement.DeviceRootRest.Rotation.X, Y: placement.DeviceRootRest.Rotation.Y, Z: placement.DeviceRootRest.Rotation.Z, W: placement.DeviceRootRest.Rotation.W}, Scale: &physicalv1.Vec3{X: placement.DeviceRootRest.Scale.X, Y: placement.DeviceRootRest.Scale.Y, Z: placement.DeviceRootRest.Scale.Z}},
			ScreenWidthPixels: 444, ScreenHeightPixels: 965, RendererProfileId: "device-studio-physical-cycles-v1", RendererProfileVersion: 1,
			OutputProfile:    shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1,
			MotionSampleTick: &shotsv1.RationalTime{Numerator: 0, Denominator: 1},
			Backend:          shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_OPTIX,
		},
		Presentation: &shotsv1.ScreenPresentation{SourceMediaId: sourceMediaID, CaptureArea: shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED, Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN, FitMode: shotsv1.ScreenFitMode_SCREEN_FIT_MODE_CONTAIN},
		MotionMode:   shotsv1.MotionMode_MOTION_MODE_FIXED,
	}
}

func TestReviewedStarterRefusesMismatchedDatabaseAndFileLease(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(t.TempDir(), "primary-data")})
	if err := db.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), "database-lease", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := roots.InstallTestRoots(storage.Paths{DataDir: filepath.Join(t.TempDir(), "leased-data")}, "different-file-lease", time.Minute); err != nil {
		t.Fatal(err)
	}
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
	if _, err := owners.InitializeReviewedStarter(database.WithTestMode(ctx)); err == nil {
		t.Fatal("starter initialized across mismatched database and file leases")
	}
	if err := roots.ClearTestRoots("different-file-lease"); err != nil {
		t.Fatal(err)
	}
	if _, err := owners.InitializeReviewedStarter(database.WithTestMode(ctx)); err == nil {
		t.Fatal("starter fell back to primary files when the test file lease was missing")
	}
	if stats := roots.LeaseStats(); stats.TestRootWrites != 0 || stats.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("mismatched or missing route attempted file writes: %+v", stats)
	}
}
