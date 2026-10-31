package physicalruntime

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"path/filepath"
	"strings"
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
	_ "modernc.org/sqlite"
)

func TestScreenFamilyCandidatesAreOptInBoundAndProfileSpecific(t *testing.T) {
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
	const lease = "screen-family-candidate-test"
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
		t.Fatal(err)
	}
	for _, id := range []string{"generic-wedge-smart-display-candidate-v2-screen-vflip", "generic-smart-display-candidate-v3", "generic-smart-display-candidate-v4", "generic-smart-display-candidate-v5", "generic-smart-display-candidate-v6", "generic-smart-display-candidate-v7", "innerscene-digital-frame-tabletop-candidate-v2", "innerscene-digital-frame-wall-candidate-v3", "innerscene-digital-frame-tabletop-candidate-v4-glassless", "innerscene-digital-frame-wall-v4-glassless", "innerscene-digital-frame-wall-v5-mounted", "generic-vehicle-center-stack-candidate-v2-screen-vflip", "generic-round-watch-candidate-v2", "generic-round-watch-candidate-v3", "generic-round-watch-body-candidate-v4", "generic-television-candidate-v3", "grab3d-generic-dashboard-candidate-v1", "grab3d-generic-dashboard-studio-candidate-v1"} {
		if _, err := owners.Starter.devices.GetHead(testCtx, id); !errors.Is(err, devices.ErrRevisionNotFound) {
			t.Fatalf("normal bootstrap published opt-in screen family %q", id)
		}
	}
	if _, err := owners.Starter.scenes.GetHead(testCtx, "grab3d-generic-dashboard-studio-wall-v2"); !errors.Is(err, scenes.ErrRevisionNotFound) {
		t.Fatalf("normal bootstrap published opt-in vehicle studio scene v2: %v", err)
	}
	states := make([]ScreenFamilyStarterState, 0, 16)
	for _, candidate := range []struct {
		initialize   func(context.Context) (ScreenFamilyStarterState, error)
		asset        string
		node         string
		flipV        bool
		glassOmitted bool
	}{
		{owners.InitializeReviewedSmartDisplayCandidate, smartDisplayAssetFile, "MainDisplay", true, false},
		{owners.InitializeReviewedSmartDisplayV3Candidate, smartDisplayV3AssetFile, "MainDisplayActivePlane", false, false},
		{owners.InitializeReviewedDigitalFrameCandidate, "assets/digital-frame/innerscene-frame-tabletop-v2.glb", "Unprinted frame insert", false, false},
		{owners.InitializeReviewedDigitalFrameWallCandidate, "assets/digital-frame/innerscene-frame-wall-v3.glb", "Unprinted frame insert", false, false},
		{func(ctx context.Context) (ScreenFamilyStarterState, error) {
			return owners.InitializeReviewedVehicleCenterStackProfile(ctx, "carplay_simulated")
		}, vehicleStackAssetFile, "CenterDisplay", true, false},
		{func(ctx context.Context) (ScreenFamilyStarterState, error) {
			return owners.InitializeReviewedVehicleCenterStackProfile(ctx, "android_auto_simulated")
		}, vehicleStackAssetFile, "CenterDisplay", true, false},
		{owners.InitializeReviewedDigitalFrameV4Candidate, "assets/digital-frame/innerscene-frame-tabletop-v4-glassless.glb", "Unprinted frame insert", false, true},
		{owners.InitializeReviewedDigitalFrameWallV4Candidate, "assets/digital-frame/innerscene-frame-wall-v4-glassless.glb", "Unprinted frame insert", false, true},
		{owners.InitializeReviewedSmartDisplayV4Candidate, smartDisplayV4AssetFile, "MainDisplayActivePlane", false, false},
		{owners.InitializeReviewedRoundWatchV2Candidate, roundWatchV2AssetFile, "One clean circular replaceable display", false, false},
		{owners.InitializeReviewedTelevisionV3Candidate, televisionV3AssetFile, "display", false, false},
		{func(ctx context.Context) (ScreenFamilyStarterState, error) {
			return owners.InitializeReviewedVehicleDashboardCC0Profile(ctx, "carplay_simulated")
		}, vehicleStackV3AssetFile, "vehicle-display-assembly", false, false},
		{func(ctx context.Context) (ScreenFamilyStarterState, error) {
			return owners.InitializeReviewedVehicleDashboardCC0Profile(ctx, "android_auto_simulated")
		}, vehicleStackV3AssetFile, "vehicle-display-assembly", false, false},
		{owners.InitializeReviewedDigitalFrameWallV5Candidate, digitalFrameWallV5AssetFile, "Unprinted frame insert", false, true},
		{owners.InitializeReviewedSmartDisplayV5Candidate, smartDisplayV5AssetFile, "MainDisplayActivePlane", false, false},
		{owners.InitializeReviewedSmartDisplayV6Candidate, smartDisplayV6AssetFile, "MainDisplayActivePlane", false, false},
		{owners.InitializeReviewedSmartDisplayV7Candidate, smartDisplayV7AssetFile, "MainDisplayActivePlane", false, false},
		{owners.InitializeReviewedRoundWatchV3Candidate, roundWatchV3AssetFile, "One clean circular replaceable display", false, false},
		{func(ctx context.Context) (ScreenFamilyStarterState, error) {
			return owners.InitializeReviewedVehicleDashboardStudioProfile(ctx, "carplay_simulated")
		}, vehicleStackV3AssetFile, "vehicle-display-assembly", false, false},
		{func(ctx context.Context) (ScreenFamilyStarterState, error) {
			return owners.InitializeReviewedVehicleDashboardStudioProfile(ctx, "android_auto_simulated")
		}, vehicleStackV3AssetFile, "vehicle-display-assembly", false, false},
		{func(ctx context.Context) (ScreenFamilyStarterState, error) {
			return owners.InitializeReviewedVehicleDashboardStudioV2Profile(ctx, "carplay_simulated")
		}, vehicleStackV3AssetFile, "vehicle-display-assembly", false, false},
		{func(ctx context.Context) (ScreenFamilyStarterState, error) {
			return owners.InitializeReviewedVehicleDashboardStudioV2Profile(ctx, "android_auto_simulated")
		}, vehicleStackV3AssetFile, "vehicle-display-assembly", false, false},
		{owners.InitializeReviewedRoundWatchBodyV4Candidate, roundWatchBodyV4AssetFile, "One clean circular replaceable display", false, false},
	} {
		state, err := candidate.initialize(testCtx)
		if err != nil {
			t.Fatalf("initialize screen-family candidate %s: %v", candidate.asset, err)
		}
		definition := state.Device.Candidate.Definition
		if state.Device.State != devices.DeviceRevisionCandidate || definition.Qualification != "unqualified" || definition.IdentityStatus != devices.IdentityConcept || len(definition.ModelCandidates) != 1 || definition.ModelCandidates[0].SourcePath != "" {
			t.Fatalf("screen-family candidate crossed its unqualified boundary: %+v", definition)
		}
		if state.Preset.Setup.Device.Id != state.Device.DeviceID || state.Preset.Setup.Device.Sha256 != state.Device.SHA256 || state.Preset.Setup.Scene.Id != state.Scene.SceneID || state.Preset.Setup.Scene.Sha256 != state.Scene.SHA256 || state.Preset.Setup.Motion.Id != state.Motion.MotionID || len(state.Preset.Setup.StillProfiles) != 1 {
			t.Fatalf("screen-family still preset is not pinned to exact owner revisions: %+v", state.Preset.Setup)
		}
		asset, err := starterAssets.ReadFile(candidate.asset)
		if err != nil {
			t.Fatalf("read candidate geometry %q: %v", candidate.asset, err)
		}
		if candidate.glassOmitted {
			inventory, err := rendering.InspectPhysicalGLB(asset)
			if err != nil {
				t.Fatalf("inspect visible-screen frame: %v", err)
			}
			for _, node := range inventory.Nodes {
				if node.Name == "Clear protective picture glazing" {
					t.Fatal("frame v4 left protective glazing in its active hierarchy")
				}
			}
			if len(inventory.Meshes) <= 7 || len(inventory.Materials) <= 5 || inventory.Materials[0].Name != "Clear glass" {
				t.Fatal("frame v4 did not retain source glazing mesh/material records")
			}
		}
		if candidate.flipV {
			asset, err = flipScreenUVV(asset, candidate.node, 0)
			if err != nil {
				t.Fatalf("derive candidate display view: %v", err)
			}
		}
		binding, err := rendering.ResolvePhysicalScreenGeometry(asset, state.Device.Candidate, state.Preset.Setup.DisplayId)
		if err != nil || binding.UVMin != [2]float64{} || binding.UVMax != [2]float64{1, 1} {
			t.Fatalf("screen family preset does not resolve one unit-UV display: mapping=%+v err=%v", binding, err)
		}
		states = append(states, state)
	}
	v3 := states[1]
	if v3.Device.DeviceID != "generic-smart-display-candidate-v3" || v3.Device.Candidate.Geometry.SHA256 != smartDisplayV3AssetSHA || v3.Bundle.Manifest.Artifacts[0].SizeBytes != 219856 {
		t.Fatalf("v3 source identity did not reach the candidate owner unchanged: device=%s ref=%+v", v3.Device.DeviceID, v3.Device.Candidate.Geometry)
	}
	v4 := states[8]
	if v4.Device.DeviceID != "generic-smart-display-candidate-v4" || v4.Device.Candidate.Geometry.SHA256 != smartDisplayV4AssetSHA || v4.Device.Candidate.Definition.Qualification != "unqualified" {
		t.Fatalf("v4 smart-display candidate did not preserve its versioned, unqualified asset: %+v", v4.Device)
	}
	dimensions := v3.Device.Candidate.Definition.Dimensions["source_geometry_envelope"]
	if dimensions.Width.Value == nil || dimensions.Height.Value == nil || dimensions.Depth.Value == nil || *dimensions.Width.Value != 177.4 || *dimensions.Height.Value != 120.4 || *dimensions.Depth.Value != 69.5 || dimensions.Width.Uncertainty.Status != "unknown" {
		t.Fatalf("v3 authored envelope/uncertainty was not preserved: %+v", dimensions)
	}
	if v3.Device.Candidate.Definition.Displays[0].ActiveArea.Status != "unknown" || v3.Device.State != devices.DeviceRevisionCandidate {
		t.Fatalf("v3 estimated screen geometry was mislabeled qualified: %+v", v3.Device.Candidate.Definition.Displays[0])
	}
	v6 := states[15]
	if v6.Device.DeviceID != "generic-smart-display-candidate-v6" || v6.Device.Candidate.Geometry.SHA256 != smartDisplayV6AssetSHA || v6.Device.Candidate.Definition.Qualification != "unqualified" || v6.Preset.Setup.DisplayId != "main_display" {
		t.Fatalf("v6 did not preserve candidate-only screen binding: %+v", v6.Device)
	}
	v7 := states[16]
	if v7.Device.DeviceID != "generic-smart-display-candidate-v7" || v7.Device.Candidate.Geometry.SHA256 != smartDisplayV7AssetSHA || v7.Device.Candidate.Definition.Qualification != "unqualified" || v7.Preset.Setup.DisplayId != "main_display" {
		t.Fatalf("v7 did not preserve candidate-only screen binding: %+v", v7.Device)
	}
	qualifiedV7 := importScreenFamilyVisualReviewForTest(t, testCtx, owners, v7)
	if qualifiedV7.Revision.State != devices.DeviceRevisionQualified || qualifiedV7.Revision.Revision != v7.Device.Revision+1 || qualifiedV7.PresetRevision != v7.Preset.Revision+1 {
		t.Fatalf("portable visual review did not append a scoped device child and refreshed preset: result=%+v", qualifiedV7)
	}
	qualifiedPreset, err := owners.InitializePhysicalStarter(testCtx, "generic-smart-display-concept-v7")
	if err != nil {
		t.Fatalf("reopen imported qualified smart-display v7 starter: %v", err)
	}
	qualifiedState, err := owners.InitializeReviewedSmartDisplayV7Candidate(testCtx)
	if err != nil || qualifiedState.Device.State != devices.DeviceRevisionQualified || qualifiedState.Device.Revision != qualifiedV7.Revision.Revision || qualifiedState.Preset.Revision != qualifiedV7.PresetRevision || qualifiedPreset.Revision != qualifiedV7.PresetRevision || qualifiedState.Preset.SHA256 != qualifiedV7.PresetSHA256 {
		t.Fatalf("initializer did not return the exact imported qualification child and refreshed preset: state=%+v preset=%+v err=%v", qualifiedState.Device, qualifiedState.Preset, err)
	}
	candidateOnly, err := owners.PhysicalStarterCandidateOnly(testCtx, qualifiedPreset)
	if err != nil || candidateOnly {
		t.Fatalf("qualified smart-display preset did not report its exact device status: candidateOnly=%v err=%v", candidateOnly, err)
	}
	changedFactory := v7.Device.Candidate
	changedFactory.Definition.Description += " unexpected changed factory content"
	if _, err := owners.Starter.saveScreenFamilyCandidateIfUnchanged(testCtx, changedFactory, smartDisplayV7Spec()); !errors.Is(err, devices.ErrRevisionConflict) {
		t.Fatalf("initializer accepted a qualified head whose factory candidate differs: %v", err)
	}
	car, android := states[4], states[5]
	if car.Preset.PresetID == android.Preset.PresetID || car.Preset.Setup.Device.Sha256 != android.Preset.Setup.Device.Sha256 || car.Device.SHA256 != android.Device.SHA256 {
		t.Fatal("simulated center-stack profiles must have distinct presets over identical candidate geometry")
	}
	watch := states[9]
	if watch.Device.DeviceID != "generic-round-watch-candidate-v2" || watch.Device.Candidate.Definition.Displays[0].ActiveArea.Shape != "circle" || watch.Device.Candidate.Definition.Displays[0].ActiveArea.Status != "candidate_only" || watch.Preset.PresetID != "generic-round-watch-still-v2" {
		t.Fatalf("round-watch v2 is not a candidate-only circular display: %+v", watch.Device.Candidate.Definition.Displays)
	}
	watchV3 := states[17]
	if watchV3.Device.DeviceID != "generic-round-watch-candidate-v3" || watchV3.Device.Candidate.Geometry.SHA256 != roundWatchV3AssetSHA || watchV3.Device.Candidate.Definition.Qualification != "unqualified" || watchV3.Preset.PresetID != "generic-round-watch-still-v3" {
		t.Fatalf("round-watch v3 should retain a new candidate-only revision and exact screen preset: %+v", watchV3.Device)
	}
	vehicleCar, vehicleAndroid := states[11], states[12]
	if vehicleCar.Preset.PresetID == vehicleAndroid.Preset.PresetID || vehicleCar.Device.SHA256 != vehicleAndroid.Device.SHA256 || vehicleCar.Device.Candidate.Geometry.SHA256 != vehicleStackV3AssetSHA {
		t.Fatal("CC0 dashboard simulated profiles must use new profile-specific presets over the exact pinned display host")
	}
	studioCar, studioAndroid := states[18], states[19]
	for _, replay := range []struct {
		profile string
		want    ScreenFamilyStarterState
	}{{"carplay_simulated", studioCar}, {"android_auto_simulated", studioAndroid}} {
		got, err := owners.InitializeReviewedVehicleDashboardStudioProfile(testCtx, replay.profile)
		if err != nil {
			t.Fatalf("replay neutral vehicle studio %s after both profiles exist: %v", replay.profile, err)
		}
		if got.Device.SHA256 != replay.want.Device.SHA256 || got.Scene.SHA256 != replay.want.Scene.SHA256 || got.Preset.SHA256 != replay.want.Preset.SHA256 {
			t.Fatalf("vehicle studio %s replay changed immutable heads: device=%s/%s scene=%s/%s preset=%s/%s", replay.profile, got.Device.SHA256, replay.want.Device.SHA256, got.Scene.SHA256, replay.want.Scene.SHA256, got.Preset.SHA256, replay.want.Preset.SHA256)
		}
	}
	studioCarV2, studioAndroidV2 := states[20], states[21]
	if studioCarV2.Scene.SceneID != "grab3d-generic-dashboard-studio-wall-v2" || studioAndroidV2.Scene.SceneID != studioCarV2.Scene.SceneID || studioCarV2.Preset.PresetID == studioAndroidV2.Preset.PresetID || studioCarV2.Scene.SHA256 == studioCar.Scene.SHA256 {
		t.Fatalf("vehicle studio v2 must have a versioned scene and profile-specific presets: car=%+v android=%+v", studioCarV2.Scene, studioAndroidV2.Preset)
	}
	if studioCarV2.Device.DeviceID != studioCar.Device.DeviceID || studioCarV2.Device.Candidate.Definition.Qualification != "unqualified" {
		t.Fatalf("vehicle studio context v2 must retain the same unqualified device candidate: %+v", studioCarV2.Device)
	}
	wallGLB, err := starterAssets.ReadFile(vehicleStudioSceneV2File)
	if err != nil {
		t.Fatal(err)
	}
	wallInventory, err := rendering.InspectPhysicalGLB(wallGLB)
	if err != nil {
		t.Fatalf("inspect compact v2 studio geometry: %v", err)
	}
	if len(wallInventory.Nodes) != 1 || wallInventory.Nodes[0].Index != 0 || wallInventory.Nodes[0].Name != "RoomBackWall" {
		t.Fatalf("v2 vehicle studio GLB must expose only its selected reachable wall node, got %+v", wallInventory.Nodes)
	}
	geometryBinding := ""
	for _, artifact := range studioCarV2.Scene.Candidate.Artifacts {
		if artifact.Ref.Role == library.RoleGeometryGLB {
			if artifact.Ref.SHA256 != vehicleStudioSceneV2SHA {
				t.Fatalf("v2 vehicle studio scene resolved unexpected geometry bytes: %+v", artifact.Ref)
			}
			geometryBinding = artifact.ID
		}
	}
	if geometryBinding == "" || len(studioCarV2.Scene.Candidate.Parts) != 1 || studioCarV2.Scene.Candidate.Parts[0].ArtifactBindingID != geometryBinding || studioCarV2.Scene.Candidate.Parts[0].NodeIndex != 0 || studioCarV2.Scene.Candidate.Parts[0].NodeName != "RoomBackWall" {
		t.Fatalf("v2 vehicle studio scene does not bind the selected reachable wall node: artifacts=%+v parts=%+v", studioCarV2.Scene.Candidate.Artifacts, studioCarV2.Scene.Candidate.Parts)
	}
	for _, replay := range []struct {
		profile string
		want    ScreenFamilyStarterState
	}{{"carplay_simulated", studioCarV2}, {"android_auto_simulated", studioAndroidV2}} {
		got, err := owners.InitializeReviewedVehicleDashboardStudioV2Profile(testCtx, replay.profile)
		if err != nil {
			t.Fatalf("replay neutral vehicle studio v2 %s after both profiles exist: %v", replay.profile, err)
		}
		if got.Device.SHA256 != replay.want.Device.SHA256 || got.Scene.SHA256 != replay.want.Scene.SHA256 || got.Preset.SHA256 != replay.want.Preset.SHA256 {
			t.Fatalf("vehicle studio v2 %s replay changed immutable heads", replay.profile)
		}
	}
	bodyV4 := states[22]
	if bodyV4.Device.DeviceID != "generic-round-watch-body-candidate-v4" || bodyV4.Device.Candidate.Geometry.SHA256 != roundWatchBodyV4AssetSHA || bodyV4.Preset.PresetID != "generic-round-watch-body-floating-still-v4" || bodyV4.Device.Candidate.Definition.Displays[0].ActiveArea.Shape != "circle" {
		t.Fatalf("floating watch-body v4 did not retain a versioned circular display candidate: %+v", bodyV4.Device)
	}
	if bodyV4.Preset.Setup.DeviceRootRest.GetTranslationM().GetY() < .08 {
		t.Fatalf("floating watch-body preset must preserve its visible air gap above the study surface: %+v", bodyV4.Preset.Setup.DeviceRootRest)
	}
	bodyGLB, err := starterAssets.ReadFile(roundWatchBodyV4AssetFile)
	if err != nil {
		t.Fatal(err)
	}
	bodyInventory, err := rendering.InspectPhysicalGLB(bodyGLB)
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range bodyInventory.Nodes {
		if strings.Contains(strings.ToLower(node.Name), "strap") || strings.Contains(strings.ToLower(node.Name), "lug") || strings.Contains(strings.ToLower(node.Name), "buckle") {
			t.Fatalf("floating watch-body v4 retained an active wearable attachment node %q", node.Name)
		}
	}
	for _, required := range []string{"Rounded monocoque case with soft bevel", "One clean circular replaceable display", "Recessed digital crown"} {
		found := false
		for _, node := range bodyInventory.Nodes {
			found = found || node.Name == required
		}
		if !found {
			t.Fatalf("floating watch-body v4 lost its required authored body detail %q", required)
		}
	}
	studioSceneAssetFound := false
	for _, artifact := range studioCar.Scene.Candidate.Artifacts {
		if artifact.Ref.Role == library.RoleGeometryGLB && artifact.Ref.SHA256 == vehicleStudioSceneSHA {
			studioSceneAssetFound = true
		}
	}
	if studioCar.Device.DeviceID != "grab3d-generic-dashboard-studio-candidate-v1" || studioCar.Device.Candidate.Definition.Qualification != "unqualified" || studioCar.Scene.SceneID != "grab3d-generic-dashboard-studio-wall-v1" || studioCar.Preset.PresetID == studioAndroid.Preset.PresetID || !studioSceneAssetFound {
		t.Fatalf("neutral vehicle studio candidates lost distinct profiles or unqualified scope: car=%+v android=%+v", studioCar.Preset, studioAndroid.Preset)
	}
	tv := states[10]
	if tv.Device.DeviceID != "generic-television-candidate-v3" || tv.Device.Candidate.Geometry.SHA256 != televisionV3AssetSHA || tv.Device.Candidate.Definition.Qualification != "unqualified" {
		t.Fatalf("thin-bezel TV v3 lost its immutable candidate-only identity: %+v", tv.Device)
	}
	wallV5 := states[13]
	if wallV5.Device.DeviceID != "innerscene-digital-frame-wall-candidate-v5-mounted" || wallV5.Device.Candidate.Geometry.SHA256 != digitalFrameWallV5AssetSHA || wallV5.Preset.PresetID != "innerscene-digital-frame-wall-still-v5-mounted" {
		t.Fatalf("wall-mounted v5 did not preserve its new versioned identity: device=%+v preset=%s", wallV5.Device.Candidate.Geometry, wallV5.Preset.PresetID)
	}
	smartV5 := states[14]
	if smartV5.Device.DeviceID != "generic-smart-display-candidate-v5" || smartV5.Device.Candidate.Geometry.SHA256 != smartDisplayV5AssetSHA || smartV5.Preset.PresetID != "generic-smart-display-tabletop-still-v5" {
		t.Fatalf("smart-display v5 did not preserve its distinct candidate-only identity: %+v", smartV5.Device.Candidate.Geometry)
	}
	smartV5GLB, err := starterAssets.ReadFile(smartDisplayV5AssetFile)
	if err != nil {
		t.Fatal(err)
	}
	if binding, err := rendering.ResolvePhysicalScreenGeometry(smartV5GLB, smartV5.Device.Candidate, "main_display"); err != nil || binding.UVMin != [2]float64{} || binding.UVMax != [2]float64{1, 1} {
		t.Fatalf("smart-display v5 changed the existing unit-UV screen binding: %+v err=%v", binding, err)
	}
	smartV5Inventory, err := rendering.InspectPhysicalGLB(smartV5GLB)
	if err != nil || len(smartV5Inventory.Meshes) <= 6 || len(smartV5Inventory.Meshes[4].Primitives) != 1 || smartV5Inventory.Meshes[4].Primitives[0].VertexCount != 4608 || len(smartV5Inventory.Meshes[6].Primitives) != 1 {
		t.Fatalf("smart-display v5 did not increase rounded shell tessellation while retaining the explicit screen mesh: %+v err=%v", smartV5Inventory.Meshes, err)
	}
	if wallV5.Preset.Setup.DeviceRootRest.TranslationM.Z < 0.0164 || wallV5.Preset.Setup.DeviceRootRest.TranslationM.Z > 0.0166 || wallV5.Preset.Setup.SceneSupportId != familyWallV5SupportID {
		t.Fatalf("wall-mounted v5 did not align its complete cleat rear bound to the wall clearance: %+v", wallV5.Preset.Setup)
	}
	wallV5GLB, err := starterAssets.ReadFile(digitalFrameWallV5AssetFile)
	if err != nil {
		t.Fatal(err)
	}
	wallV5Inventory, err := rendering.InspectPhysicalGLB(wallV5GLB)
	if err != nil {
		t.Fatal(err)
	}
	var cleatZ, screenZ *float64
	for _, node := range wallV5Inventory.Nodes {
		switch node.Name {
		case "WallMountCleat":
			z := node.Frame.Translation.Z
			cleatZ = &z
		case "Unprinted frame insert":
			z := node.Frame.Translation.Z
			screenZ = &z
		}
	}
	wallV5Bounds, err := physicalGLBWorldBounds(wallV5Inventory)
	if err != nil || cleatZ == nil || screenZ == nil || math.Abs(*cleatZ+0.013) > 1e-6 || math.Abs(*screenZ+0.007) > 1e-6 || math.Abs(wallV5Bounds.min.Z+0.017) > 1e-6 {
		t.Fatalf("wall v5 must move only the mount pad and keep the screen seated: cleat=%v screen=%v bounds=%+v err=%v", cleatZ, screenZ, wallV5Bounds, err)
	}
	if _, err := owners.InitializeReviewedVehicleCenterStackProfile(testCtx, "manufacturer-compatible"); err == nil {
		t.Fatal("unsupported vehicle profile was accepted")
	}
}

func importScreenFamilyVisualReviewForTest(t *testing.T, ctx context.Context, owners Owners, state ScreenFamilyStarterState) devices.QualificationReviewResult {
	t.Helper()
	service, err := NewQualificationService(owners, qualificationRenderStub{})
	if err != nil {
		t.Fatal(err)
	}
	bundles, artifacts, err := service.exportQualificationSourceClosure(ctx, state.Device.Candidate.Geometry, state.Scene.Candidate.Artifacts)
	if err != nil {
		t.Fatalf("export exact test source closure: %v", err)
	}
	output := []byte("fixture-only native still bytes")
	outputSHA := digestHex(output)
	setup := state.Preset.Setup
	deviceRest, err := qualificationTRS(setup.DeviceRootRest)
	if err != nil {
		t.Fatal(err)
	}
	sceneWorld, err := qualificationTRS(setup.SceneSupportWorld)
	if err != nil {
		t.Fatal(err)
	}
	if setup.SupportPivotM == nil {
		t.Fatal("test starter has no support pivot")
	}
	review := devices.QualificationReview{
		Scope: "visual_setup", Result: "passed", Reviewer: "test-only explicit reviewer",
		Evidence:          []devices.Evidence{{Source: "test://qualification-package/native-still", Kind: "physical_still_png"}},
		CandidateRevision: state.Device.Revision, CandidateSHA256: state.Device.SHA256,
		SourceBundleID: state.Device.Candidate.Geometry.BundleID, SourceBundleRevision: state.Device.Candidate.Geometry.BundleRevision,
		SourceManifestSHA256: state.Device.Candidate.Geometry.ManifestSHA256, SourceReviewSHA256: state.Device.Candidate.Geometry.ReviewSHA256,
		SourceArtifactID: state.Device.Candidate.Geometry.ArtifactID, SourceArtifactSHA256: state.Device.Candidate.Geometry.SHA256,
		DimensionStates: []string{"source_geometry_envelope"}, DisplayIDs: []string{setup.DisplayId}, AllowedMotionIDs: []string{"static"}, SupportIDs: []string{setup.DeviceSupportId},
		PresetID: state.Preset.PresetID, PresetRevision: state.Preset.Revision, PresetSHA256: state.Preset.SHA256,
		SceneID: state.Scene.SceneID, SceneRevision: state.Scene.Revision, SceneSHA256: state.Scene.SHA256,
		MotionID: state.Motion.MotionID, MotionRevision: state.Motion.Revision, MotionSHA256: state.Motion.SHA256,
		NativeRenderJobID: "fixture-only-job", NativeOutputSHA256: outputSHA, NativeOutputKind: "physical_still_png",
		RightsClaimsAcknowledged: true, ReviewedAt: time.Date(2026, 10, 21, 15, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}
	for _, bundle := range bundles {
		if bundle.Manifest.BundleID == state.Device.Candidate.Geometry.BundleID && bundle.Manifest.Revision == state.Device.Candidate.Geometry.BundleRevision {
			review.SourceAllowedUses = libraryUses(bundle.Review.AllowedUses)
		}
	}
	if len(review.SourceAllowedUses) == 0 {
		t.Fatal("test package source bundle has no approved render use")
	}
	proof := renderjobs.QualificationRenderEvidence{
		DeviceID: state.Device.DeviceID, DeviceRevision: state.Device.Revision, DeviceSHA256: state.Device.SHA256,
		SceneID: state.Scene.SceneID, SceneRevision: state.Scene.Revision, SceneSHA256: state.Scene.SHA256,
		MotionID: state.Motion.MotionID, MotionRevision: state.Motion.Revision, MotionSHA256: state.Motion.SHA256,
		DisplayID: setup.DisplayId, DeviceSupportID: setup.DeviceSupportId, SceneSupportID: setup.SceneSupportId, CameraID: setup.CameraId,
		OutputSHA256: outputSHA, OutputKind: review.NativeOutputKind, DeviceRootRest: deviceRest, SceneSupportWorld: sceneWorld,
		SupportPivot: physical.Vec3{X: setup.SupportPivotM.X, Y: setup.SupportPivotM.Y, Z: setup.SupportPivotM.Z},
	}
	pkg := qualificationPackage{
		Version: 2, Origin: qualificationPackageOrigin{ProjectID: "test-project", JobID: review.NativeRenderJobID, Reviewed: review.ReviewedAt},
		Candidate: state.Device, SourceBundles: bundles, SourceArtifacts: artifacts, Scene: state.Scene, Motion: state.Motion,
		Preset: state.Preset, Review: review, Render: proof, Output: output,
	}
	data, err := sealQualificationPackage(pkg)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ImportReviewPackage(ctx, data, state.Device.Revision)
	if err != nil {
		t.Fatalf("import exact scoped test package: %v", err)
	}
	return result
}
