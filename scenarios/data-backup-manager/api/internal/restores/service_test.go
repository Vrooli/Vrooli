package restores_test

import (
	"context"
	"testing"
	"time"

	"data-backup-manager/internal/restores"
	restoresmocks "data-backup-manager/internal/restores/mocks"
	"data-backup-manager/internal/sources"
	sourcesmocks "data-backup-manager/internal/sources/mocks"
	"data-backup-manager/internal/testutil/mocks"
)

// buildRestoreService wires the restore service against a real sqlite repo plus
// fakes for every other seam. It registers a FakeCapturer for each of the six
// source kinds and returns the service, the engine mock, and the capturer map.
func buildRestoreService(t *testing.T) (restores.Service, *mocks.FakeKopiaEngine, map[sources.SourceKind]*sourcesmocks.FakeCapturer) {
	t.Helper()

	allKinds := []sources.SourceKind{
		sources.KindFilesystem,
		sources.KindSQLite,
		sources.KindPostgres,
		sources.KindRedis,
		sources.KindQdrant,
		sources.KindObjectStorage,
	}

	capturers := make(map[sources.SourceKind]*sourcesmocks.FakeCapturer, len(allKinds))
	capturerSlice := make([]sources.Capturer, 0, len(allKinds))
	for _, k := range allKinds {
		c := &sourcesmocks.FakeCapturer{SourceKind: k}
		capturers[k] = c
		capturerSlice = append(capturerSlice, c)
	}
	registry := sources.NewRegistry(capturerSlice...)

	eng := &mocks.FakeKopiaEngine{}
	clk := mocks.NewFakeClock(time.Time{})

	targets := map[string]restores.TargetForRestore{}
	for _, k := range allKinds {
		targets["tgt-"+string(k)] = restores.TargetForRestore{
			ID:      "tgt-" + string(k),
			Kind:    k,
			Locator: "locator-" + string(k),
		}
	}

	svc := restores.NewService(restores.Deps{
		Repo: restores.NewSQLiteRepository(newRestoresDB(t), clk),
		Targets: &restoresmocks.FakeTargetLookup{
			Targets: targets,
		},
		Destinations: &restoresmocks.FakeDestinationLookup{
			Destinations: map[string]restores.DestinationForRestore{
				"dst-1": {ID: "dst-1", Name: "nightly"},
			},
		},
		Engine:      eng,
		Sources:     registry,
		Clock:       clk,
		ScratchRoot: t.TempDir(),
	})
	return svc, eng, capturers
}

// TestRestore_PerSourceKind verifies that for each of the six source kinds,
// RestoreTarget calls Engine.SnapshotRestore for the right repo+snapshot+location
// AND calls the matching capturer's Restore. Status must be restored.
func TestRestore_PerSourceKind(t *testing.T) {
	allKinds := []sources.SourceKind{
		sources.KindFilesystem,
		sources.KindSQLite,
		sources.KindPostgres,
		sources.KindRedis,
		sources.KindQdrant,
		sources.KindObjectStorage,
	}

	for _, kind := range allKinds {
		kind := kind
		t.Run(string(kind), func(t *testing.T) {
			ctx := context.Background()
			svc, eng, capturers := buildRestoreService(t)

			targetID := "tgt-" + string(kind)
			const destID = "dst-1"
			const snapID = "snap-abc"
			const location = "/restore/dest"

			rec, err := svc.RestoreTarget(ctx, targetID, destID, snapID, location)
			if err != nil {
				t.Fatalf("RestoreTarget(%s): %v", kind, err)
			}
			if rec.Status != restores.RestoreRestored {
				t.Errorf("status = %s, want restored", rec.Status)
			}

			// Engine.SnapshotRestore must have been called with the right repo, snap, location.
			wantCall := "SnapshotRestore(nightly,snap-abc,/restore/dest)"
			found := false
			for _, c := range eng.Calls {
				if c == wantCall {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected engine call %q; calls: %v", wantCall, eng.Calls)
			}

			// The matching capturer's Restore must have been called.
			capt := capturers[kind]
			if len(capt.Restores) == 0 {
				t.Errorf("capturer %s Restore was not called", kind)
			}
		})
	}
}
