package physicalruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

func TestReviewedWatchCandidateIsOptInReplayableAndSourceBound(t *testing.T) {
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
	const lease = "watch-candidate-replay-lease"
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
	if _, err := owners.Starter.devices.GetHead(testCtx, watchDeviceID); !errors.Is(err, devices.ErrRevisionNotFound) {
		t.Fatal("normal starter initialization published the opt-in watch candidate")
	}
	first, err := owners.InitializeReviewedWatchCandidate(testCtx)
	if err != nil {
		t.Fatalf("initialize explicit watch candidate: %v", err)
	}
	second, err := owners.InitializeReviewedWatchCandidate(testCtx)
	if err != nil {
		t.Fatalf("replay explicit watch candidate: %v", err)
	}
	if first.Bundle.ManifestSHA256 != second.Bundle.ManifestSHA256 || first.Device.SHA256 != second.Device.SHA256 || first.Scene.SHA256 != second.Scene.SHA256 || first.Motion.SHA256 != second.Motion.SHA256 || first.Preset.SHA256 != second.Preset.SHA256 {
		t.Fatal("watch candidate replay changed immutable bundle or owner revisions")
	}
	definition := first.Device.Candidate.Definition
	if first.Device.State != devices.DeviceRevisionCandidate || definition.Group != devices.GroupWatch || definition.Kind != "generic_rectangular_smartwatch_style" || definition.IdentityStatus != devices.IdentityGeneric || definition.Variant != "unknown; source hierarchy scale preserved" || definition.Qualification != "unqualified" {
		t.Fatalf("watch candidate identity or qualification overclaims source: %+v", definition)
	}
	if len(definition.ModelCandidates) != 1 || definition.ModelCandidates[0].SourcePath != "" || definition.ModelCandidates[0].SHA256 != watchAssetSHA256 || definition.ModelCandidates[0].License != "CC BY 4.0" || definition.ModelCandidates[0].Author != "SDC PERFORMANCE™; base model attribution to bharathsrk13" || definition.ModelCandidates[0].RightsNotes == "" {
		t.Fatalf("watch candidate lost creator attribution or rights limits: %+v", definition.ModelCandidates)
	}
	glb, err := starterAssets.ReadFile(watchAssetFile)
	if err != nil {
		t.Fatal(err)
	}
	mapping, err := rendering.ResolvePhysicalScreenGeometry(glb, first.Device.Candidate, watchDisplayID)
	if err != nil || mapping.NodeIndex != 27 || mapping.MeshIndex != 14 || mapping.PrimitiveIndex != 0 || mapping.MaterialIndex != 11 || mapping.UVMin != [2]float64{} || mapping.UVMax != [2]float64{1, 1} {
		t.Fatalf("watch screen selector is not the exact normalized source display: mapping=%+v err=%v", mapping, err)
	}
	if first.Scene.SceneID != watchSceneID || first.Scene.State != "candidate" || first.Placement.DeviceSupportID != watchSupportID || first.Placement.SceneSupportID != laptopTableSupportID || first.Placement.SceneSupportWorld.Translation.Y != .75 {
		t.Fatalf("watch scene/table pose drifted: scene=%+v placement=%+v", first.Scene, first.Placement)
	}
	if len(first.Scene.Candidate.CameraFrames) != 1 {
		t.Fatal("watch scene must retain its single fitted candidate camera")
	}
	camera := first.Scene.Candidate.CameraFrames[0]
	displayCenter := physical.Vec3{X: watchDisplayCenter.X, Y: .75 + watchDisplayCenter.Y, Z: watchDisplayCenter.Z}
	wantFocus, err := watchCameraAxisDistance(camera.Frame, displayCenter)
	if err != nil || camera.Focus == nil || camera.Focus.Mode != scenes.FocusDistance || camera.Focus.DistanceM == nil || math.Abs(*camera.Focus.DistanceM-wantFocus) > 1e-9 {
		t.Fatalf("candidate camera must focus at the source-hash-pinned display center: focus=%+v want=%0.12g err=%v", camera.Focus, wantFocus, err)
	}
	for _, part := range first.Scene.Candidate.Parts {
		if part.NodeName == "TeaCup" || part.NodeName == "TeaCupHandle" || part.NodeName == "TeaSurface" {
			t.Fatalf("watch candidate room unexpectedly includes tea prop %q", part.NodeName)
		}
	}
	wantSetup := &shotsv1.PhysicalShotSetup{
		Device:    &shotsv1.ComponentRevisionRef{Id: first.Device.DeviceID, Revision: first.Device.Revision, Sha256: first.Device.SHA256},
		Scene:     &shotsv1.ComponentRevisionRef{Id: first.Scene.SceneID, Revision: first.Scene.Revision, Sha256: first.Scene.SHA256},
		Motion:    &shotsv1.ComponentRevisionRef{Id: first.Motion.MotionID, Revision: first.Motion.Revision, Sha256: first.Motion.SHA256},
		DisplayId: first.Placement.DisplayID, DeviceSupportId: first.Placement.DeviceSupportID, SceneSupportId: first.Placement.SceneSupportID, CameraId: first.Placement.CameraID,
		SceneSupportWorld: starterTRS(first.Placement.SceneSupportWorld), SupportPivotM: starterVec3(first.Placement.SupportPivotM), DeviceRootRest: starterTRS(first.Placement.DeviceRootRest),
		StillProfiles: []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1},
	}
	if first.Preset.PresetID != watchPresetID || !proto.Equal(first.Preset.Setup, wantSetup) {
		t.Fatalf("watch preset is not bound to candidate revisions and still output: %+v", first.Preset.Setup)
	}
	placement := ReviewedIPadPlacement{DisplayID: first.Placement.DisplayID, DeviceSupportID: first.Placement.DeviceSupportID, SceneSupportID: first.Placement.SceneSupportID, SelectedCameraID: first.Placement.CameraID, SceneSupportWorld: first.Placement.SceneSupportWorld, SupportPivotM: first.Placement.SupportPivotM, DeviceRootRest: first.Placement.DeviceRootRest}
	state := IPadStarterState{Device: first.Device, Scene: first.Scene, Motion: first.Motion, Placement: placement}
	project, err := projectService.Create(testCtx, projects.CreateInput{Name: "Portable generic watch render-closure review"})
	if err != nil {
		t.Fatal(err)
	}
	chart := watchNativeCalibrationChart(t)
	screen, err := mediaService.Import(testCtx, project.ID, "watch-render-closure-chart.png", int64(len(chart)), bytes.NewReader(chart))
	if err != nil {
		t.Fatal(err)
	}
	shotService := shots.NewService(shots.NewSQLiteRepository(db), projectService, mediaService, nil, nil)
	shots.SetPhysicalResolver(shotService, owners.Shots)
	snapshot := ipadStillSnapshot(state, screen.ID, placement, shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_OPTIX)
	snapshot.PhysicalBinding.ScreenWidthPixels, snapshot.PhysicalBinding.ScreenHeightPixels = 629, 537
	shot, revision, err := shotService.Create(testCtx, project.ID, "watch-portable-closure", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	profileBytes, profile, err := rendering.ResolvePhysicalBlenderProfile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	workerPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	blobs, err := routedblobstore.New(roots, "renderjobs")
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := renderjobs.New(renderjobs.Dependencies{
		Shots: shotService, Media: mediaService, Jobs: jobs.NewService(jobs.NewSQLiteRepository(db), time.Now), Blobs: blobs, Roots: roots,
		Exe: func() (string, error) { return workerPath, nil }, PhysicalArtifacts: owners.Library,
		PhysicalProfile: func(context.Context) ([]byte, *cliv1.BlenderToolProfile, error) { return profileBytes, profile, nil }, PhysicalAdapter: rendering.PhysicalAdapterBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	job, created, err := coordinator.Submit(testCtx, renderjobs.SubmitRequest{ProjectID: project.ID, ShotID: shot.ID, Revision: revision.Revision, IdempotencyKey: "watch-portable-closure-only", OutputProfile: renderjobs.OutputProfilePNG})
	if err != nil || !created || job.State != jobs.Queued || len(job.Attempts) != 1 || job.Attempts[0].State != jobs.AttemptQueued || job.Attempts[0].Execution != nil || job.Attempts[0].StartedAt != nil {
		t.Fatalf("portable watch render closure must save a queued attempt without worker dispatch: created=%v state=%s attempts=%+v err=%v", created, job.State, job.Attempts, err)
	}
}

func TestWatchSourceMeasurementsCorrectAABBPlacementAndRun2Focus(t *testing.T) {
	data, err := starterAssets.ReadFile("assets/watch/asset-manifest.json")
	if err != nil || sha256Hex(data) != watchManifestSHA {
		t.Fatalf("read pinned watch geometry manifest: err=%v sha256=%s", err, sha256Hex(data))
	}
	var manifest struct {
		Source struct {
			SHA string `json:"sha256"`
		} `json:"source"`
		Geometry struct {
			Exact struct {
				Vertices uint64    `json:"transformed_position_vertex_count"`
				MinimumY float64   `json:"minimum_y_m"`
				ScreenN  uint64    `json:"display_center_vertex_count"`
				Center   []float64 `json:"display_center_m"`
			} `json:"exact_source_position_measurements"`
		} `json:"geometry"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	facts := manifest.Geometry.Exact
	if manifest.Source.SHA != watchSourceSHA256 || facts.Vertices != 161468 || facts.MinimumY != watchSupportMinimumY || facts.ScreenN != 86 || len(facts.Center) != 3 || math.Abs(facts.Center[0]-watchDisplayCenter.X) > 1e-12 || math.Abs(facts.Center[1]-watchDisplayCenter.Y) > 1e-12 || math.Abs(facts.Center[2]-watchDisplayCenter.Z) > 1e-12 {
		t.Fatalf("manifest geometry facts differ from pinned source measurements: %+v", facts)
	}
	glb, err := starterAssets.ReadFile(watchAssetFile)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := rendering.InspectPhysicalGLB(glb)
	if err != nil {
		t.Fatal(err)
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil || math.Abs(-bounds.min.Y-.007084853342457564) > 1e-9 || math.Abs(inventory.Nodes[28].Frame.Scale.Y-.6430670822618203) > 1e-9 {
		t.Fatalf("run2 conservative AABB/root scale measurements changed: bounds=%+v err=%v", bounds, err)
	}
	placement := watchPlacement()
	if placement.DeviceRootRest.Translation.Y != 0 || math.Abs(facts.MinimumY+placement.SceneSupportWorld.Translation.Y-.75) > 1e-9 {
		t.Fatalf("pinned source vertex minimum must meet the saved table plane: facts=%+v placement=%+v", facts, placement)
	}
	run2Camera := physical.TRS{
		Translation: physical.Vec3{X: .10808689186144864, Y: .8365183054057627, Z: .1828086931352243},
		Rotation:    physical.QuatXYZW{X: -.13391219193216977, Y: .26336950432949446, Z: .03694414237501352, W: .9546408536121491},
		Scale:       physical.Vec3{X: 1, Y: 1, Z: 1},
	}
	oldOffset := -bounds.min.Y
	oldRootDepth, err := watchCameraAxisDistance(run2Camera, physical.Vec3{Y: .75 + oldOffset})
	if err != nil {
		t.Fatal(err)
	}
	oldScreenDepth, err := watchCameraAxisDistance(run2Camera, physical.Vec3{X: watchDisplayCenter.X, Y: .75 + oldOffset + watchDisplayCenter.Y, Z: watchDisplayCenter.Z})
	if err != nil || math.Abs(oldRootDepth-oldScreenDepth-.0219844478297902) > 1e-6 {
		t.Fatalf("retained run2 pose must reproduce the 22 mm wrapper-focus error: root=%0.12g screen=%0.12g err=%v", oldRootDepth, oldScreenDepth, err)
	}
}

func TestWatchCatalogTruthAndAssetIndexPinCandidate(t *testing.T) {
	var series9 struct {
		Kind     string `json:"kind"`
		Displays []struct {
			Topology string `json:"topology"`
		} `json:"displays"`
	}
	series9Path := filepath.Join("..", "..", "..", "assets", "devices", "apple-watch-series-9-45mm.json")
	data, err := os.ReadFile(series9Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &series9); err != nil {
		t.Fatal(err)
	}
	if series9.Kind != "rectangular_smartwatch" || len(series9.Displays) != 1 || series9.Displays[0].Topology != "planar" {
		t.Fatalf("Series 9 rectangular planar display must not be mapped to a round-watch kind: %+v", series9)
	}
	var index struct {
		Entries []struct {
			CatalogKey string `json:"catalog_key"`
			Path       string `json:"path"`
			SHA256     string `json:"sha256"`
			Status     string `json:"status"`
		} `json:"entries"`
		PackageManifests []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"package_manifests"`
	}
	indexPath := filepath.Join("assets", "asset-index.json")
	b, err := starterAssets.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &index); err != nil {
		t.Fatal(err)
	}
	wantHash, err := hex.DecodeString(watchAssetSHA256)
	if err != nil || len(wantHash) != sha256.Size {
		t.Fatal("invalid pinned watch asset hash")
	}
	entryFound, manifestFound := false, false
	for _, entry := range index.Entries {
		if entry.CatalogKey == watchAssetID && entry.Path == "watch/series7-generic-neutral-screen.glb" && entry.SHA256 == watchAssetSHA256 && entry.Status == "candidate_only_unqualified" {
			entryFound = true
		}
	}
	for _, manifest := range index.PackageManifests {
		if manifest.Path == "watch/asset-manifest.json" && manifest.SHA256 == watchManifestSHA {
			manifestFound = true
		}
	}
	if !entryFound || !manifestFound {
		t.Fatalf("asset index does not pin watch artifact and package manifest: entry=%v manifest=%v", entryFound, manifestFound)
	}
}
