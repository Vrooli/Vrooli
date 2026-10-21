package physicalruntime

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"path/filepath"
	"testing"
	"time"

	"device-studio/internal/devices"
	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/routedblobstore"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

func TestReviewedFoldableCandidateIsOptInReplayableAndDualRigid(t *testing.T) {
	ctx := context.Background()
	providers := []database.SchemaProvider{database.SchemaProviderFunc(library.Schema), database.SchemaProviderFunc(devices.Schema), database.SchemaProviderFunc(scenes.Schema), database.SchemaProviderFunc(motions.Schema), database.SchemaProviderFunc(projects.Schema), database.SchemaProviderFunc(shots.Schema), database.SchemaProviderFunc(jobs.Schema)}
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
	const lease = "foldable-candidate-replay-lease"
	if err := db.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), lease, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := roots.InstallTestRoots(storage.Paths{DataDir: filepath.Join(t.TempDir(), "leased-data")}, lease, time.Minute); err != nil {
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
	if _, err := owners.InitializeReviewedStarter(testCtx); err != nil {
		t.Fatalf("initialize ordinary starter: %v", err)
	}
	if _, err := owners.Starter.devices.GetHead(testCtx, foldDeviceID); !errors.Is(err, devices.ErrRevisionNotFound) {
		t.Fatal("ordinary starter initialization published the opt-in foldable candidate")
	}
	first, err := owners.InitializeReviewedFoldableCandidate(testCtx)
	if err != nil {
		t.Fatalf("initialize foldable concept: %v", err)
	}
	second, err := owners.InitializeReviewedFoldableCandidate(testCtx)
	if err != nil {
		t.Fatalf("replay foldable concept: %v", err)
	}
	if first.Bundle.ManifestSHA256 != second.Bundle.ManifestSHA256 || first.Device.SHA256 != second.Device.SHA256 || first.Scene.SHA256 != second.Scene.SHA256 || first.Motion.SHA256 != second.Motion.SHA256 || first.Preset.SHA256 != second.Preset.SHA256 {
		t.Fatal("foldable candidate replay changed immutable owner revisions")
	}
	definition := first.Device.Candidate.Definition
	if first.Device.State != devices.DeviceRevisionCandidate || definition.Group != devices.GroupFoldable || definition.IdentityStatus != devices.IdentityConcept || definition.Qualification != "unqualified" || len(definition.Displays) != 3 || len(definition.Joints) != 1 || len(definition.SupportedPoses) != 2 || len(definition.SupportedMotion) != 1 {
		t.Fatalf("foldable candidate scope or topology is wrong: %+v", definition)
	}
	if definition.Joints[0].Status != "candidate_only" || *definition.Joints[0].LowerRadians != 0 || math.Abs(*definition.Joints[0].UpperRadians-math.Pi) > 1e-12 {
		t.Fatalf("concept hinge limits are not explicit: %+v", definition.Joints[0])
	}
	poses := map[string]float64{}
	for _, pose := range definition.SupportedPoses {
		poses[pose.Name] = pose.JointState[foldJointID]
	}
	if poses["open-flat"] != 0 || math.Abs(poses["folded-closed"]-math.Pi) > 1e-12 || len(poses) != 2 {
		t.Fatalf("supported static poses do not match the authored hinge interval: %v", poses)
	}
	if len(first.Device.Candidate.Parts) != 8 || len(first.Device.Candidate.DisplayBindings) != 3 {
		t.Fatalf("candidate lost dual-rigid parts or screen bindings: parts=%d displays=%d", len(first.Device.Candidate.Parts), len(first.Device.Candidate.DisplayBindings))
	}
	glb, err := starterAssets.ReadFile(foldAssetFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, display := range definition.Displays {
		mapping, err := rendering.ResolvePhysicalScreenGeometry(glb, first.Device.Candidate, display.ID)
		if err != nil || mapping.UVMin != [2]float64{} || mapping.UVMax != [2]float64{1, 1} {
			t.Fatalf("display %s lacks its exact unit-UV surface: mapping=%+v err=%v", display.ID, mapping, err)
		}
	}
	if first.Scene.SceneID != foldSceneID || first.Placement.SceneSupportID != laptopTableSupportID || first.Placement.DeviceRootRest.Translation.Y != foldSupportLiftM || first.Placement.DeviceRootRest.Rotation.X >= 0 || first.Preset.PresetID != foldPresetID || first.Preset.Setup.DisplayId != "inner-left" || first.Device.DeviceID != "generic-dual-rigid-book-fold-concept-v2" || first.Bundle.Manifest.BundleID != "device-studio-starter-generic-book-fold-concept-v2" {
		t.Fatalf("foldable scene or capture recipe is not bound to the candidate: scene=%s placement=%+v preset=%+v", first.Scene.SceneID, first.Placement, first.Preset)
	}
	var camera *scenes.SceneCameraFrame
	for i := range first.Scene.Candidate.CameraFrames {
		if first.Scene.Candidate.CameraFrames[i].ID == foldCameraID {
			camera = &first.Scene.Candidate.CameraFrames[i]
		}
	}
	if camera == nil || camera.Focus == nil || camera.Focus.Mode != scenes.FocusDistance || camera.Focus.DistanceM == nil || !positiveFinite(*camera.Focus.DistanceM) {
		t.Fatalf("fold still must focus on the selected inner display, not the full device envelope: %+v", camera)
	}
	inventory, err := rendering.InspectPhysicalGLB(glb)
	if err != nil {
		t.Fatal(err)
	}
	var leftDisplay rendering.PhysicalGLBNode
	for _, node := range inventory.Nodes {
		if node.Name == "LeftInnerDisplay" {
			leftDisplay = node
		}
	}
	displayBounds, err := physicalGLBNodeMeshWorldBounds(inventory, leftDisplay.Index)
	if err != nil {
		t.Fatal(err)
	}
	displayPoints, err := monitorPlacedBounds(displayBounds, first.Device.Candidate.SourceToCanonical, first.Placement)
	if err != nil {
		t.Fatal(err)
	}
	wantFocus, err := watchCameraAxisDistance(camera.Frame, meanPhysicalPoints(displayPoints))
	if err != nil || math.Abs(*camera.Focus.DistanceM-wantFocus) > 1e-9 {
		t.Fatalf("fold focus does not target selected display center: got=%0.12g want=%0.12g err=%v", *camera.Focus.DistanceM, wantFocus, err)
	}
	if first.Motion.MotionID != foldMotionID || len(first.Motion.Candidate.Tracks) != 1 || first.Motion.Candidate.Tracks[0].Target.JointID != foldJointID {
		t.Fatalf("fold motion is not bound to the authored joint: %+v", first.Motion.Candidate)
	}
	if definition.ModelCandidates[0].SourcePath != "" || definition.ModelCandidates[0].Author != "Vrooli" || definition.ModelCandidates[0].License != "AGPL-3.0-only" {
		t.Fatalf("candidate provenance claims an external source: %+v", definition.ModelCandidates[0])
	}
}
