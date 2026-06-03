package restores_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
		Executor:    restoresmocks.NewSyncExecutor(),
	})
	return svc, eng, capturers
}

// TestRestore_PerSourceKind verifies that for each of the six source kinds,
// RestoreTarget calls Engine.SnapshotRestore for the right repo+snapshot into a
// temporary artifact directory AND calls the matching capturer's Restore with
// that artifact path and the caller-chosen final location. Status must be
// restored.
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

			capt := capturers[kind]
			if len(capt.Restores) == 0 {
				t.Errorf("capturer %s Restore was not called", kind)
				return
			}
			restore := capt.Restores[0]
			if restore.ArtifactPath == "" {
				t.Fatal("capturer restore artifact path is empty")
			}
			if restore.ArtifactPath == restore.Target {
				t.Fatalf("capturer restore artifact path must differ from target: %q", restore.ArtifactPath)
			}
			if restore.Target != location {
				t.Fatalf("capturer restore target = %q, want %q", restore.Target, location)
			}
			if restore.Locator != "locator-"+string(kind) {
				t.Fatalf("capturer restore locator = %q", restore.Locator)
			}
			wantCall := "SnapshotRestore(nightly,snap-abc," + restore.ArtifactPath + ")"
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
		})
	}
}

func TestRestoreTarget_RestoresSnapshotArtifactToFinalLocation(t *testing.T) {
	ctx := context.Background()
	scratchRoot := t.TempDir()
	restoreRoot := t.TempDir()
	finalLocation := filepath.Join(restoreRoot, "final")

	artifactSeed := t.TempDir()
	mustWriteFile(t, filepath.Join(artifactSeed, "root.txt"), "alpha\n")
	mustWriteFile(t, filepath.Join(artifactSeed, "nested", "child.txt"), "beta\n")

	var snapshotTarget string
	eng := &mocks.FakeKopiaEngine{
		SnapshotRestoreFn: func(_ context.Context, repo, snapshotID, target string) error {
			if repo != "nightly" {
				t.Fatalf("repo = %q, want nightly", repo)
			}
			if snapshotID != "snap-abc" {
				t.Fatalf("snapshot id = %q, want snap-abc", snapshotID)
			}
			snapshotTarget = target
			return copyTree(artifactSeed, target)
		},
	}

	fsCapturer := &sourcesmocks.FakeCapturer{SourceKind: sources.KindFilesystem}
	fsCapturer.RestoreFn = func(ctx context.Context, spec sources.RestoreSpec) error {
		if spec.ArtifactPath == spec.Target {
			t.Fatalf("artifact path and final target must differ: %q", spec.ArtifactPath)
		}
		return copyTree(spec.ArtifactPath, spec.Target)
	}

	svc := restores.NewService(restores.Deps{
		Repo: restores.NewSQLiteRepository(newRestoresDB(t), mocks.NewFakeClock(time.Time{})),
		Targets: &restoresmocks.FakeTargetLookup{
			Targets: map[string]restores.TargetForRestore{
				"tgt-fs": {ID: "tgt-fs", Kind: sources.KindFilesystem, Locator: "source-locator"},
			},
		},
		Destinations: &restoresmocks.FakeDestinationLookup{
			Destinations: map[string]restores.DestinationForRestore{
				"dst-1": {ID: "dst-1", Name: "nightly"},
			},
		},
		Engine:      eng,
		Sources:     sources.NewRegistry(fsCapturer),
		Clock:       mocks.NewFakeClock(time.Time{}),
		ScratchRoot: scratchRoot,
		Executor:    restoresmocks.NewSyncExecutor(),
	})

	rec, err := svc.RestoreTarget(ctx, "tgt-fs", "dst-1", "snap-abc", finalLocation)
	if err != nil {
		t.Fatalf("RestoreTarget: %v", err)
	}
	if rec.Status != restores.RestoreRestored {
		t.Fatalf("status = %s, want restored", rec.Status)
	}
	if snapshotTarget == "" {
		t.Fatal("snapshot restore target was not recorded")
	}
	if !strings.HasPrefix(snapshotTarget, scratchRoot) {
		t.Fatalf("snapshot restore target %q is not under scratch root %q", snapshotTarget, scratchRoot)
	}
	if snapshotTarget == finalLocation {
		t.Fatalf("snapshot restore target must not be final location %q", finalLocation)
	}
	if _, err := os.Stat(snapshotTarget); !os.IsNotExist(err) {
		t.Fatalf("temporary artifact dir should be cleaned up; stat err = %v", err)
	}
	assertFile(t, filepath.Join(finalLocation, "root.txt"), "alpha\n")
	assertFile(t, filepath.Join(finalLocation, "nested", "child.txt"), "beta\n")
}

// TestRestoreTarget_RefusesNonEmptyTarget proves the fail-closed safety
// contract: restoring into an existing non-empty directory is rejected with
// ErrInvalidRestore{location} and the engine is never touched.
func TestRestoreTarget_RefusesNonEmptyTarget(t *testing.T) {
	ctx := context.Background()
	nonEmpty := t.TempDir()
	mustWriteFile(t, filepath.Join(nonEmpty, "existing.txt"), "do not clobber\n")

	eng := &mocks.FakeKopiaEngine{}
	svc := restores.NewService(restores.Deps{
		Repo: restores.NewSQLiteRepository(newRestoresDB(t), mocks.NewFakeClock(time.Time{})),
		Targets: &restoresmocks.FakeTargetLookup{
			Targets: map[string]restores.TargetForRestore{
				"tgt-fs": {ID: "tgt-fs", Kind: sources.KindFilesystem, Locator: "source-locator"},
			},
		},
		Destinations: &restoresmocks.FakeDestinationLookup{
			Destinations: map[string]restores.DestinationForRestore{"dst-1": {ID: "dst-1", Name: "nightly"}},
		},
		Engine:      eng,
		Sources:     sources.NewRegistry(&sourcesmocks.FakeCapturer{SourceKind: sources.KindFilesystem}),
		Clock:       mocks.NewFakeClock(time.Time{}),
		ScratchRoot: t.TempDir(),
	})

	_, err := svc.RestoreTarget(ctx, "tgt-fs", "dst-1", "snap-abc", nonEmpty)
	var invalid restores.ErrInvalidRestore
	if !errors.As(err, &invalid) || invalid.Field != "location" {
		t.Fatalf("expected ErrInvalidRestore{location}, got %v", err)
	}
	for _, c := range eng.Calls {
		if strings.HasPrefix(c, "SnapshotRestore") {
			t.Fatalf("engine must not be touched when target is non-empty; calls = %v", eng.Calls)
		}
	}
	// The pre-existing file is untouched.
	assertFile(t, filepath.Join(nonEmpty, "existing.txt"), "do not clobber\n")
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o640)
	})
}

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o640); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, string(got), want)
	}
}
