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
