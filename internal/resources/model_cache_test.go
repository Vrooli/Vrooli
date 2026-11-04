package resources

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func modelCacheFixture(t *testing.T) (*modelCacheStore, *bool, *time.Time) {
	t.Helper()
	root := t.TempDir()
	models := filepath.Join(root, "data", "models")
	if err := os.MkdirAll(models, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(models, "weights.bin"), []byte("owned weights"), 0o600); err != nil {
		t.Fatal(err)
	}
	now, running := time.Unix(100*86400, 0), false
	manifest := ResourceManifest{Name: "test-model-cache", ManagedService: &resourcedeployment.ManagedService{
		DataArtifacts: []resourcedeployment.ManagedServiceDataArtifact{{Name: "weights", Path: "models/weights.bin"}}},
		Storage: &ResourceStorage{Entries: map[string]ResourceStorageEntry{"models": {Kind: "dir", Class: "data", Subpath: "models",
			Budget:  &manifestpkg.ResourceStorageBudget{MaxAge: "30d"},
			Reclaim: &manifestpkg.ResourceStorageReclaim{Pruner: "custom", Command: modelCacheOwnerCommand + " test-model-cache"}}}}}
	return &modelCacheStore{manifest: manifest, entryName: "models", root: models, state: filepath.Join(root, "state", "usage.json"), age: 30 * 24 * time.Hour,
		now: func() time.Time { return now }, running: func() (bool, error) { return running, nil }}, &running, &now
}

func TestModelCacheRequiresKnownConfirmedIdleLifetime(t *testing.T) {
	store, running, now := modelCacheFixture(t)
	preview, err := store.preview()
	if err != nil || preview.Eligible {
		t.Fatalf("unknown history must be kept: %+v %v", preview, err)
	}
	*running = true
	if err := store.note("started"); err != nil {
		t.Fatal(err)
	}
	*running = false
	*now = now.Add(40 * 24 * time.Hour)
	preview, err = store.preview()
	if err != nil || preview.Eligible || !strings.Contains(preview.Reason, "confirmed stop") {
		t.Fatalf("abnormal loss must not invent idle time: %+v %v", preview, err)
	}
	if err := store.note("stopped"); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(31 * 24 * time.Hour)
	preview, err = store.preview()
	if err != nil || !preview.Eligible {
		t.Fatalf("confirmed idle bundle must become eligible: %+v %v", preview, err)
	}
}

func TestModelCacheApplyProtectsLivePinnedAndChangedInputs(t *testing.T) {
	for _, change := range []string{"live", "pin", "bytes", "model-revision", "unexpected-file"} {
		t.Run(change, func(t *testing.T) {
			store, running, now := modelCacheFixture(t)
			if err := store.note("stopped"); err != nil {
				t.Fatal(err)
			}
			*now = now.Add(31 * 24 * time.Hour)
			preview, err := store.preview()
			if err != nil || !preview.Eligible {
				t.Fatalf("fixture not eligible: %v %+v", err, preview)
			}
			switch change {
			case "live":
				*running = true
			case "pin":
				usage, _ := store.usage()
				usage.Pinned = true
				if err := store.save(usage); err != nil {
					t.Fatal(err)
				}
			case "bytes":
				if err := os.WriteFile(filepath.Join(store.root, "weights.bin"), []byte("new bytes"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "model-revision":
				store.manifest.ManagedService.DataArtifacts[0].Name = "replacement"
			case "unexpected-file":
				if err := os.WriteFile(filepath.Join(store.root, "notes.txt"), []byte("operator source"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if result, err := store.apply(preview); err == nil || result.Removed {
				t.Fatalf("changed/protected cache was removed: %+v %v", result, err)
			}
			if _, err := os.Stat(filepath.Join(store.root, "weights.bin")); err != nil {
				t.Fatal("weights disappeared")
			}
		})
	}
}

func TestModelCacheReacquisitionStartsFreshWindowAndPreservesPin(t *testing.T) {
	store, _, now := modelCacheFixture(t)
	if err := store.note("stopped"); err != nil {
		t.Fatal(err)
	}
	usage, _ := store.usage()
	usage.Pinned = true
	if err := store.save(usage); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(40 * 24 * time.Hour)
	if err := store.note("installed"); err != nil {
		t.Fatal(err)
	}
	preview, err := store.preview()
	if err != nil || preview.Eligible || !preview.Pinned || !preview.LastActivity.Equal(*now) {
		t.Fatalf("reacquisition lost protection: %+v %v", preview, err)
	}
}

func TestModelCacheDeletesOnlyItsOwnedBundleAndRejectsSymlinks(t *testing.T) {
	store, _, now := modelCacheFixture(t)
	outside := filepath.Join(filepath.Dir(store.root), "operator.wav")
	if err := os.WriteFile(outside, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.note("stopped"); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(31 * 24 * time.Hour)
	if err := os.Symlink(outside, filepath.Join(store.root, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.preview(); err == nil {
		t.Fatal("symlink tree accepted")
	}
	os.Remove(filepath.Join(store.root, "escape"))
	preview, err := store.preview()
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.apply(preview)
	if err != nil || !result.Removed {
		t.Fatalf("eligible bundle not removed: %+v %v", result, err)
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "preserve" {
		t.Fatal("operator export was changed")
	}
}

func TestModelCacheCorruptUsageNeverResetsPinsOrAllowsDeletion(t *testing.T) {
	store, _, _ := modelCacheFixture(t)
	os.MkdirAll(filepath.Dir(store.state), 0o700)
	os.WriteFile(store.state, []byte(`{"pinned":`), 0o600)
	if err := store.note("installed"); err == nil {
		t.Fatal("corrupt usage was silently replaced")
	}
	if _, err := store.preview(); err == nil {
		t.Fatal("corrupt usage allowed a deletion preview")
	}
}

func TestModelCacheGeneratedSupportFilesAreExplicitAndDoNotAdmitOperatorFiles(t *testing.T) {
	store, _, now := modelCacheFixture(t)
	entry := store.manifest.Storage.Entries["models"]
	entry.Reclaim.GeneratedFiles = []string{"generated.py"}
	if _, _, err := modelCacheEntry(store.manifest, "models"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"generated.py", "generated.py.sha256"} {
		if err := os.WriteFile(filepath.Join(store.root, path), []byte("derived"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.note("stopped"); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(31 * 24 * time.Hour)
	preview, err := store.preview()
	if err != nil || !preview.Eligible {
		t.Fatalf("declared support files protected: %+v %v", preview, err)
	}
	if err := os.WriteFile(filepath.Join(store.root, "operator-notes.txt"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	preview, err = store.preview()
	if err != nil || preview.Eligible {
		t.Fatalf("unknown operator data admitted: %+v %v", preview, err)
	}
	entry.Reclaim.GeneratedFiles = []string{"../operator-notes.txt"}
	if _, _, err := modelCacheEntry(store.manifest, "models"); err == nil {
		t.Fatal("generated-file traversal accepted")
	}
}
