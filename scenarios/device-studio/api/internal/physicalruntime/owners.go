// Package physicalruntime composes the candidate and rights owners used by
// saved physical shots. It contains no persistence of its own.
package physicalruntime

import (
	"context"
	"errors"

	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/projects"
	"device-studio/internal/routedblobstore"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
)

type Owners struct {
	Devices        *devices.DeviceRevisionService
	Scenes         *scenes.RevisionService
	Motions        *motions.RevisionService
	Shots          shots.PhysicalResolver
	Library        physical.LibraryArtifactResolver
	LibraryCatalog library.Service
	Starter        *starterInitializer
	Presets        *shots.PhysicalPresetService
	Qualification  *QualificationService
}

func New(db *database.RoutedDB, roots *filerouting.RoutedRoots, media projects.SourceMediaService) (Owners, error) {
	if db == nil || roots == nil || media == nil {
		return Owners{}, errors.New("physical runtime requires routed database, roots, and project media owner")
	}
	libraryStore, err := routedblobstore.New(roots, "library")
	if err != nil {
		return Owners{}, err
	}
	libraryOwner, err := library.NewService(library.NewSQLiteRepository(db), libraryStore, nil, nil)
	if err != nil {
		return Owners{}, err
	}
	artifactResolver, err := physical.NewLibraryArtifactResolver(libraryOwner)
	if err != nil {
		return Owners{}, err
	}
	deviceOwner, err := devices.NewDeviceRevisionService(devices.NewSQLiteDeviceRevisionRepository(db), nil)
	if err != nil {
		return Owners{}, err
	}
	sceneRepo, err := scenes.NewSQLiteRevisionRepository(db)
	if err != nil {
		return Owners{}, err
	}
	sceneOwner, err := scenes.NewRevisionService(sceneRepo, nil)
	if err != nil {
		return Owners{}, err
	}
	motionRepo, err := motions.NewSQLiteRevisionRepository(db)
	if err != nil {
		return Owners{}, err
	}
	motionOwner, err := motions.NewRevisionService(motionRepo, nil)
	if err != nil {
		return Owners{}, err
	}
	shotResolver, err := shots.NewPhysicalResolver(deviceOwner, sceneOwner, motionOwner, artifactResolver, media)
	if err != nil {
		return Owners{}, err
	}
	presetRepo, err := shots.NewSQLitePhysicalPresetRepository(db)
	if err != nil {
		return Owners{}, err
	}
	setupValidator, ok := shotResolver.(shots.PhysicalPresetValidator)
	if !ok {
		return Owners{}, errors.New("physical shot resolver does not provide setup validation")
	}
	presetService, err := shots.NewPhysicalPresetService(presetRepo, setupValidator, nil, nil)
	if err != nil {
		return Owners{}, err
	}
	starter, err := newStarterInitializer(db, roots, &libraryOwner, deviceOwner, sceneOwner, motionOwner, presetService, starterAssets)
	if err != nil {
		return Owners{}, err
	}
	return Owners{Devices: deviceOwner, Scenes: sceneOwner, Motions: motionOwner, Shots: shotResolver, Library: artifactResolver, LibraryCatalog: libraryOwner, Starter: starter, Presets: presetService}, nil
}

// InitializeReviewedStarter seeds the fixed phone starter and candidate-only
// laptop on one pinned database and file route, returning the phone response.
func (o Owners) InitializeReviewedStarter(ctx context.Context) (StarterState, error) {
	if o.Starter == nil {
		return StarterState{}, errors.New("reviewed starter initializer is unavailable")
	}
	pinned, err := o.Starter.pinRoute(ctx)
	if err != nil {
		return StarterState{}, err
	}
	phone, err := o.Starter.initializePinned(pinned)
	if err != nil {
		return StarterState{}, err
	}
	laptop, err := o.Starter.initializeLaptopCandidatePinned(pinned)
	if err != nil {
		return StarterState{}, err
	}
	if _, err := o.Starter.initializeMonitorCandidatePinned(pinned, laptop.FramedScene, phone.Motion); err != nil {
		return StarterState{}, err
	}
	if _, err := o.Starter.initializeMonitorScreenMappedCandidatePinned(pinned, laptop.FramedScene, phone.Motion); err != nil {
		return StarterState{}, err
	}
	return phone, nil
}

// InitializePhysicalStarter opts into one existing candidate initializer and
// returns its immutable setup preset. Empty starter IDs are the legacy phone.
func (o Owners) InitializePhysicalStarter(ctx context.Context, starterID string) (shots.PhysicalPresetRevision, error) {
	switch starterID {
	case "", "phone-default":
		state, err := o.InitializeReviewedStarter(ctx)
		return state.Preset, err
	case "laptop-screen-upright-180":
		state, err := o.InitializeReviewedLaptopScreenUpright(ctx)
		return state.Preset, err
	case "laptop-screen-vflip":
		state, err := o.InitializeReviewedLaptopScreenVFlip(ctx)
		return state.Preset, err
	case "ipad-screen-uv-oriented-180":
		state, err := o.InitializeReviewedIPadOrientedStarter(ctx)
		return state.Preset, err
	case "monitor-screen-mapped-v2":
		if o.Starter == nil {
			return shots.PhysicalPresetRevision{}, errors.New("monitor candidate initializer is unavailable")
		}
		pinned, err := o.Starter.pinRoute(ctx)
		if err != nil {
			return shots.PhysicalPresetRevision{}, err
		}
		phone, err := o.Starter.initializePinned(pinned)
		if err != nil {
			return shots.PhysicalPresetRevision{}, err
		}
		laptop, err := o.Starter.initializeLaptopCandidatePinned(pinned)
		if err != nil {
			return shots.PhysicalPresetRevision{}, err
		}
		state, err := o.Starter.initializeMonitorScreenMappedCandidatePinned(pinned, laptop.FramedScene, phone.Motion)
		return state.Preset, err
	case "generic-round-watch":
		state, err := o.InitializeReviewedWatchCandidate(ctx)
		return state.Preset, err
	case "generic-round-smartwatch-source":
		state, err := o.InitializeReviewedRoundWatchCandidate(ctx)
		return state.Preset, err
	case "generic-round-watch-v2":
		state, err := o.InitializeReviewedRoundWatchV2Candidate(ctx)
		return state.Preset, err
	case "generic-round-watch-v3":
		state, err := o.InitializeReviewedRoundWatchV3Candidate(ctx)
		return state.Preset, err
	case "generic-round-watch-body-v4-floating":
		state, err := o.InitializeReviewedRoundWatchBodyV4Candidate(ctx)
		return state.Preset, err
	case "generic-television-concept":
		state, err := o.InitializeReviewedTelevisionCandidate(ctx)
		return state.Preset, err
	case "generic-television-concept-v3":
		state, err := o.InitializeReviewedTelevisionV3Candidate(ctx)
		return state.Preset, err
	case "generic-book-fold":
		state, err := o.InitializeReviewedFoldableCandidate(ctx)
		return state.Preset, err
	case "generic-book-fold-v3-left-screen-vflip":
		state, err := o.InitializeReviewedFoldableUVV3(ctx)
		return state.Preset, err
	case "generic-book-fold-v3-left-screen-vflip-static":
		state, err := o.InitializeReviewedFoldableUVV3Static(ctx)
		return state.Preset, err
	case "generic-smart-display-concept":
		state, err := o.InitializeReviewedSmartDisplayCandidate(ctx)
		return state.Preset, err
	case "generic-smart-display-concept-v3":
		state, err := o.InitializeReviewedSmartDisplayV3Candidate(ctx)
		return state.Preset, err
	case "generic-smart-display-concept-v4":
		state, err := o.InitializeReviewedSmartDisplayV4Candidate(ctx)
		return state.Preset, err
	case "generic-smart-display-concept-v5":
		state, err := o.InitializeReviewedSmartDisplayV5Candidate(ctx)
		return state.Preset, err
	case "generic-smart-display-concept-v6":
		state, err := o.InitializeReviewedSmartDisplayV6Candidate(ctx)
		return state.Preset, err
	case "generic-smart-display-concept-v7":
		state, err := o.InitializeReviewedSmartDisplayV7Candidate(ctx)
		return state.Preset, err
	case reviewedSmartDisplayV7StarterID:
		return o.installReviewedSmartDisplayV7(ctx)
	case "generic-round-watch-body-v4-floating-reviewed", "generic-television-concept-v3-reviewed", "generic-digital-frame-concept-v4-glassless-reviewed", "generic-digital-frame-wall-v5-mounted-reviewed", "generic-book-fold-v3-left-screen-vflip-static-reviewed", "laptop-screen-vflip-reviewed":
		return o.installEmbeddedReviewedStarter(ctx, starterID)
	case "generic-digital-frame-concept":
		state, err := o.InitializeReviewedDigitalFrameCandidate(ctx)
		return state.Preset, err
	case "generic-digital-frame-wall":
		state, err := o.InitializeReviewedDigitalFrameWallCandidate(ctx)
		return state.Preset, err
	case "generic-digital-frame-concept-v4-glassless":
		state, err := o.InitializeReviewedDigitalFrameV4Candidate(ctx)
		return state.Preset, err
	case "generic-digital-frame-wall-v4-glassless":
		state, err := o.InitializeReviewedDigitalFrameWallV4Candidate(ctx)
		return state.Preset, err
	case "generic-digital-frame-wall-v5-mounted":
		state, err := o.InitializeReviewedDigitalFrameWallV5Candidate(ctx)
		return state.Preset, err
	case "generic-vehicle-center-stack-concept":
		state, err := o.InitializeReviewedVehicleCenterStackCandidate(ctx)
		return state.Preset, err
	case "generic-vehicle-center-stack-carplay-sim":
		state, err := o.InitializeReviewedVehicleCenterStackProfile(ctx, "carplay_simulated")
		return state.Preset, err
	case "generic-vehicle-center-stack-android-auto-sim":
		state, err := o.InitializeReviewedVehicleCenterStackProfile(ctx, "android_auto_simulated")
		return state.Preset, err
	case "generic-vehicle-dashboard-carplay-sim":
		state, err := o.InitializeReviewedVehicleDashboardCC0Profile(ctx, "carplay_simulated")
		return state.Preset, err
	case "generic-vehicle-dashboard-android-auto-sim":
		state, err := o.InitializeReviewedVehicleDashboardCC0Profile(ctx, "android_auto_simulated")
		return state.Preset, err
	case "generic-vehicle-dashboard-carplay-studio-v1":
		state, err := o.InitializeReviewedVehicleDashboardStudioProfile(ctx, "carplay_simulated")
		return state.Preset, err
	case "generic-vehicle-dashboard-android-auto-studio-v1":
		state, err := o.InitializeReviewedVehicleDashboardStudioProfile(ctx, "android_auto_simulated")
		return state.Preset, err
	case "generic-vehicle-dashboard-carplay-studio-v2":
		state, err := o.InitializeReviewedVehicleDashboardStudioV2Profile(ctx, "carplay_simulated")
		return state.Preset, err
	case "generic-vehicle-dashboard-android-auto-studio-v2":
		state, err := o.InitializeReviewedVehicleDashboardStudioV2Profile(ctx, "android_auto_simulated")
		return state.Preset, err
	default:
		return shots.PhysicalPresetRevision{}, ErrUnknownStarterID
	}
}

// PhysicalStarterCandidateOnly reports the status of the exact device revision
// pinned by a successfully initialized physical preset.
func (o Owners) PhysicalStarterCandidateOnly(ctx context.Context, preset shots.PhysicalPresetRevision) (bool, error) {
	if o.Devices == nil || preset.Setup == nil || preset.Setup.Device == nil || preset.Setup.Device.Id == "" || preset.Setup.Device.Revision == 0 {
		return true, errors.New("physical starter preset does not pin an inspectable device revision")
	}
	device, err := o.Devices.GetRevision(ctx, preset.Setup.Device.Id, preset.Setup.Device.Revision)
	if err != nil {
		return true, err
	}
	return device.State != devices.DeviceRevisionQualified || device.Candidate.Definition.Qualification != "qualified", nil
}

var ErrUnknownStarterID = errors.New("unknown physical starter ID")
