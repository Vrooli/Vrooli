package physicalruntime

import (
	"context"
	"database/sql"
	"errors"
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
	for _, id := range []string{"generic-wedge-smart-display-candidate-v1", "innerscene-digital-frame-tabletop-candidate-v2", "innerscene-digital-frame-wall-candidate-v3", "generic-vehicle-center-stack-candidate-v1"} {
		if _, err := owners.Starter.devices.GetHead(testCtx, id); !errors.Is(err, devices.ErrRevisionNotFound) {
			t.Fatalf("normal bootstrap published opt-in screen family %q", id)
		}
	}
	states := make([]ScreenFamilyStarterState, 0, 5)
	for _, candidate := range []struct {
		initialize func(context.Context) (ScreenFamilyStarterState, error)
		asset      string
	}{
		{owners.InitializeReviewedSmartDisplayCandidate, smartDisplayAssetFile},
		{owners.InitializeReviewedDigitalFrameCandidate, "assets/digital-frame/innerscene-frame-tabletop-v2.glb"},
		{owners.InitializeReviewedDigitalFrameWallCandidate, "assets/digital-frame/innerscene-frame-wall-v3.glb"},
		{func(ctx context.Context) (ScreenFamilyStarterState, error) {
			return owners.InitializeReviewedVehicleCenterStackProfile(ctx, "carplay_simulated")
		}, vehicleStackAssetFile},
		{func(ctx context.Context) (ScreenFamilyStarterState, error) {
			return owners.InitializeReviewedVehicleCenterStackProfile(ctx, "android_auto_simulated")
		}, vehicleStackAssetFile},
	} {
		state, err := candidate.initialize(testCtx)
		if err != nil {
			t.Fatal(err)
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
		binding, err := rendering.ResolvePhysicalScreenGeometry(asset, state.Device.Candidate, state.Preset.Setup.DisplayId)
		if err != nil || binding.UVMin != [2]float64{} || binding.UVMax != [2]float64{1, 1} {
			t.Fatalf("screen family preset does not resolve one unit-UV display: mapping=%+v err=%v", binding, err)
		}
		states = append(states, state)
	}
	car, android := states[3], states[4]
	if car.Preset.PresetID == android.Preset.PresetID || car.Preset.Setup.Device.Sha256 != android.Preset.Setup.Device.Sha256 || car.Device.SHA256 != android.Device.SHA256 {
		t.Fatal("simulated center-stack profiles must have distinct presets over identical candidate geometry")
	}
	if _, err := owners.InitializeReviewedVehicleCenterStackProfile(testCtx, "manufacturer-compatible"); err == nil {
		t.Fatal("unsupported vehicle profile was accepted")
	}
}
