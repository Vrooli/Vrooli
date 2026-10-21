package physicalruntime

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	devicesH "device-studio/handlers/devices"
	libraryH "device-studio/handlers/library"
	motionsH "device-studio/handlers/motions"
	scenesH "device-studio/handlers/scenes"
	shotsH "device-studio/handlers/shots"
	systemdb "device-studio/internal/database"
	devicesDomain "device-studio/internal/devices"
	jobsDomain "device-studio/internal/jobs"
	libraryDomain "device-studio/internal/library"
	motionsDomain "device-studio/internal/motions"
	runtimeowners "device-studio/internal/physicalruntime"
	"device-studio/internal/projects"
	"device-studio/internal/routedblobstore"
	scenesDomain "device-studio/internal/scenes"
	shotsDomain "device-studio/internal/shots"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices/devices_v1connect"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library/library_v1connect"
	motionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/motions"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/motions/motions_v1connect"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	physicalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical/physical_v1connect"
	scenesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes/scenes_v1connect"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots/shots_v1connect"
	_ "modernc.org/sqlite"
)

// This crosses the public Connect boundaries from the candidate-only starter
// initializer through preset and component readback; direct stores alone miss
// schema/domain fields omitted by ProtoJSON projection.
func TestPublicFamilyStarterAndPinnedRevisionRoundTrips(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	providers := []database.SchemaProvider{
		database.SchemaProviderFunc(systemdb.SystemSchema), database.SchemaProviderFunc(projects.Schema),
		database.SchemaProviderFunc(libraryDomain.Schema), database.SchemaProviderFunc(devicesDomain.Schema),
		database.SchemaProviderFunc(scenesDomain.Schema), database.SchemaProviderFunc(motionsDomain.Schema),
		database.SchemaProviderFunc(shotsDomain.Schema), database.SchemaProviderFunc(jobsDomain.Schema),
	}
	if err := database.EnsureSchemas(ctx, db.Primary(), providers...); err != nil {
		t.Fatal(err)
	}
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(t.TempDir(), "data")})
	repo := projects.NewSQLiteRepository(db, nil)
	mediaStore, err := routedblobstore.New(roots, "media")
	if err != nil {
		t.Fatal(err)
	}
	media := projects.NewSourceMediaService(projects.SourceMediaDeps{Projects: projects.NewService(repo, nil), Repository: repo.(projects.SourceMediaRepository), Store: mediaStore})
	owners, err := runtimeowners.New(db, roots, media)
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	for _, mount := range []func(*mux.Router){
		Module(owners).Mount, devicesH.ModuleWithService(owners.Devices).Mount, scenesH.ModuleWithService(owners.Scenes).Mount,
		motionsH.ModuleWithService(owners.Motions).Mount, libraryH.ModuleWithService(owners.LibraryCatalog).Mount,
		shotsH.ModuleWithPhysicalOwners(db, roots, nil, owners.Shots, owners.Presets, nil).Mount,
	} {
		mount(router)
	}
	server := httptest.NewServer(router)
	defer server.Close()
	transport := server.Client()
	starter := physicalconnect.NewPhysicalStarterServiceClient(transport, server.URL)
	devices := devices_v1connect.NewDevicesServiceClient(transport, server.URL)
	scenes := scenes_v1connect.NewScenesServiceClient(transport, server.URL)
	motions := motions_v1connect.NewMotionsServiceClient(transport, server.URL)
	library := library_v1connect.NewLibraryCatalogServiceClient(transport, server.URL)
	shots := shots_v1connect.NewShotsServiceClient(transport, server.URL)
	families := []string{"generic-round-watch", "generic-round-smartwatch-source", "generic-round-watch-v2", "generic-television-concept", "generic-television-concept-v3", "generic-smart-display-concept", "generic-smart-display-concept-v3", "generic-smart-display-concept-v4", "generic-smart-display-concept-v5", "generic-smart-display-concept-v6", "generic-digital-frame-concept", "generic-digital-frame-wall", "generic-digital-frame-concept-v4-glassless", "generic-digital-frame-wall-v4-glassless", "generic-digital-frame-wall-v5-mounted", "generic-vehicle-center-stack-concept", "generic-vehicle-center-stack-carplay-sim", "generic-vehicle-center-stack-android-auto-sim", "generic-vehicle-dashboard-carplay-sim", "generic-vehicle-dashboard-android-auto-sim", "generic-book-fold", "generic-book-fold-v3-left-screen-vflip", "generic-book-fold-v3-left-screen-vflip-static"}
	for _, id := range families {
		t.Run(id, func(t *testing.T) {
			initialized, err := starter.InitializeReviewedStarter(ctx, connect.NewRequest(&physicalv1.InitializeReviewedStarterRequest{StarterId: id}))
			if err != nil {
				t.Fatalf("public initializer: %v", err)
			}
			if !initialized.Msg.GetCandidateOnly() || initialized.Msg.GetSelectedStarterId() != id || initialized.Msg.GetInitializedPreset() == nil {
				t.Fatalf("initializer did not return the requested candidate preset: %+v", initialized.Msg)
			}
			presetRef := initialized.Msg.GetInitializedPreset()
			presetResponse, err := shots.GetPhysicalShotPreset(ctx, connect.NewRequest(&shotsv1.GetPhysicalShotPresetRequest{PresetId: presetRef.GetId(), Revision: presetRef.GetRevision()}))
			if err != nil {
				t.Fatalf("public preset read: %v", err)
			}
			preset := presetResponse.Msg.GetRevision()
			if preset.GetPresetId() != presetRef.GetId() || preset.GetRevision() != presetRef.GetRevision() || preset.GetSha256() != presetRef.GetSha256() || !preset.GetCandidateOnly() {
				t.Fatalf("preset projection changed pinned identity or candidate-only status: %+v", preset)
			}
			setup := preset.GetSetup()
			if setup == nil {
				t.Fatal("public preset omitted its setup")
			}
			deviceRef, sceneRef, motionRef := setup.GetDevice(), setup.GetScene(), setup.GetMotion()
			if deviceRef == nil || sceneRef == nil || motionRef == nil {
				t.Fatalf("preset setup omitted a pinned component: %+v", setup)
			}
			deviceResponse, err := devices.GetDeviceRevision(ctx, connect.NewRequest(&devicesv1.GetDeviceRevisionRequest{DeviceId: deviceRef.GetId(), Revision: deviceRef.GetRevision()}))
			if err != nil {
				t.Fatalf("public device revision read: %v", err)
			}
			device := deviceResponse.Msg.GetRevision()
			if device.GetSha256() != deviceRef.GetSha256() || device.GetDeviceId() != deviceRef.GetId() || device.GetRevision() != deviceRef.GetRevision() {
				t.Fatalf("device revision does not match preset reference: %+v", device)
			}
			geometry := device.GetCandidate().GetGeometry()
			if geometry == nil {
				t.Fatal("public device candidate omitted its Library geometry reference")
			}
			bundleResponse, err := library.GetBundleRevision(ctx, connect.NewRequest(&libraryv1.GetBundleRevisionRequest{BundleId: geometry.GetBundleId(), Revision: geometry.GetBundleRevision()}))
			if err != nil {
				t.Fatalf("public Library bundle read: %v", err)
			}
			bundle := bundleResponse.Msg.GetRevision()
			if bundle.GetManifestSha256() != geometry.GetManifestSha256() || bundle.GetReviewSha256() != geometry.GetReviewSha256() {
				t.Fatalf("public Library metadata differs from device artifact pin: %+v", bundle)
			}
			foundGeometry := false
			for _, artifact := range bundle.GetManifest().GetArtifacts() {
				if artifact.GetId() == geometry.GetArtifactId() && artifact.GetSha256() == geometry.GetSha256() {
					foundGeometry = true
				}
			}
			if !foundGeometry {
				t.Fatalf("Library revision omitted pinned geometry artifact %q", geometry.GetArtifactId())
			}
			if id == "generic-television-concept" {
				bindings := device.GetCandidate().GetDisplayBindings()
				if len(bindings) == 0 || bindings[0].ScreenPrimitiveIndex == nil || bindings[0].ScreenMaterialIndex == nil {
					t.Fatalf("TV public projection lost optional screen primitive/material presence: %+v", bindings)
				}
			}
			if id == "generic-television-concept-v3" {
				bindings := device.GetCandidate().GetDisplayBindings()
				if len(bindings) != 1 || bindings[0].GetScreenPrimitiveIndex() != 1 || bindings[0].ScreenPrimitiveIndex == nil || bindings[0].ScreenMaterialIndex == nil {
					t.Fatalf("TV v3 public projection lost its distinct screen plane selector: %+v", bindings)
				}
			}
			if id == "generic-round-watch-v2" {
				definition := device.GetCandidate().GetDefinition()
				bindings := device.GetCandidate().GetDisplayBindings()
				if len(bindings) != 1 || bindings[0].ScreenPrimitiveIndex == nil || bindings[0].GetScreenPrimitiveIndex() != 0 || definition.GetDisplays()[0].GetActiveArea().GetShape() != "circle" || definition.GetDisplays()[0].GetActiveArea().GetStatus() != "candidate_only" {
					t.Fatalf("round-watch v2 public projection lost circular display geometry: bindings=%+v display=%+v", bindings, definition.GetDisplays())
				}
			}
			if id == "generic-smart-display-concept-v3" || id == "generic-smart-display-concept-v4" || id == "generic-smart-display-concept-v5" || id == "generic-smart-display-concept-v6" {
				bindings := device.GetCandidate().GetDisplayBindings()
				if len(bindings) != 1 || bindings[0].GetDisplayId() != "main_display" || bindings[0].ScreenPrimitiveIndex == nil || bindings[0].GetScreenPrimitiveIndex() != 0 || bindings[0].ScreenMaterialIndex == nil || bindings[0].GetScreenMaterialIndex() != 2 {
					t.Fatalf("smart-display public projection lost exact screen node selectors: %+v", bindings)
				}
				if device.GetState() != "candidate" || device.GetCandidate().GetDefinition().GetIdentityStatus() != "concept" {
					t.Fatalf("smart-display public state lost candidate/concept status: %+v", device)
				}
			}
			sceneResponse, err := scenes.GetSceneRevision(ctx, connect.NewRequest(&scenesv1.GetSceneRevisionRequest{SceneId: sceneRef.GetId(), Revision: sceneRef.GetRevision()}))
			if err != nil {
				t.Fatalf("public scene revision read: %v", err)
			}
			if sceneResponse.Msg.GetRevision().GetSceneId() != sceneRef.GetId() || sceneResponse.Msg.GetRevision().GetRevision() != sceneRef.GetRevision() || sceneResponse.Msg.GetRevision().GetSha256() != sceneRef.GetSha256() {
				t.Fatalf("scene revision digest differs from preset reference: %+v", sceneResponse.Msg.GetRevision())
			}
			motionResponse, err := motions.GetMotionRevision(ctx, connect.NewRequest(&motionsv1.GetMotionRevisionRequest{MotionId: motionRef.GetId(), Revision: motionRef.GetRevision()}))
			if err != nil {
				t.Fatalf("public motion revision read: %v", err)
			}
			if motionResponse.Msg.GetRevision().GetMotionId() != motionRef.GetId() || motionResponse.Msg.GetRevision().GetRevision() != motionRef.GetRevision() || motionResponse.Msg.GetRevision().GetSha256() != motionRef.GetSha256() {
				t.Fatalf("motion revision digest differs from preset reference: %+v", motionResponse.Msg.GetRevision())
			}
			if id == "generic-vehicle-center-stack-carplay-sim" || id == "generic-vehicle-center-stack-android-auto-sim" || id == "generic-vehicle-dashboard-carplay-sim" || id == "generic-vehicle-dashboard-android-auto-sim" {
				replayed, err := starter.InitializeReviewedStarter(ctx, connect.NewRequest(&physicalv1.InitializeReviewedStarterRequest{StarterId: id}))
				if err != nil {
					t.Fatalf("public initializer replay: %v", err)
				}
				replayedPreset := replayed.Msg.GetInitializedPreset()
				if replayedPreset.GetId() != presetRef.GetId() || replayedPreset.GetRevision() != presetRef.GetRevision() || replayedPreset.GetSha256() != presetRef.GetSha256() {
					t.Fatalf("vehicle profile replay changed its immutable preset reference: %+v", replayedPreset)
				}
			}
		})
	}
}
