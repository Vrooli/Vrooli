package blenderrenderer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func TestRepairPythonCacheDryRunAndQuarantinePreservePinnedTree(t *testing.T) {
	root := filepath.Join(t.TempDir(), "opt", ToolName)
	artifact := fixtureArtifact(t, root)
	cache := filepath.Join(root, ArchiveRoot, "4.5", "scripts", "modules", "__pycache__")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "module.cpython-312.pyc"), []byte("cache"), 0o644); err != nil {
		t.Fatal(err)
	}
	quarantineRoot := filepath.Join(t.TempDir(), "state", "tool-repair", ToolName)

	dry, err := RepairPythonCache(context.Background(), root, quarantineRoot, artifact, true)
	if err != nil || !dry.DryRun || dry.Repaired || len(dry.CacheDirectories) != 1 {
		t.Fatalf("dry-run receipt=%+v err=%v", dry, err)
	}
	if _, err := os.Stat(filepath.Join(cache, "module.cpython-312.pyc")); err != nil {
		t.Fatalf("dry-run moved cache: %v", err)
	}
	if _, err := os.Stat(quarantineRoot); !os.IsNotExist(err) {
		t.Fatalf("dry-run created quarantine root: %v", err)
	}

	got, err := RepairPythonCache(context.Background(), root, quarantineRoot, artifact, false)
	if err != nil || !got.Repaired || got.QuarantineID == "" {
		t.Fatalf("repair receipt=%+v err=%v", got, err)
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("cache directory remains in managed tree: %v", err)
	}
	quarantined := filepath.Join(quarantineRoot, got.QuarantineID, filepath.FromSlash(got.CacheDirectories[0]), "module.cpython-312.pyc")
	if data, err := os.ReadFile(quarantined); err != nil || string(data) != "cache" {
		t.Fatalf("quarantined cache=%q err=%v", data, err)
	}
	if _, err := os.Stat(filepath.Join(quarantineRoot, got.QuarantineID, "repair-receipt.json")); err != nil {
		t.Fatalf("repair receipt not retained: %v", err)
	}
	tree, err := binaryfetch.TreeDigest(root)
	if err != nil || tree != artifact.TreeSHA256 {
		t.Fatalf("remaining managed tree digest=%q err=%v; want pinned %q", tree, err, artifact.TreeSHA256)
	}
}

func TestRepairPythonCacheRefusesDriftAndUnsafeCacheContents(t *testing.T) {
	t.Run("unrelated drift", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "opt", ToolName)
		artifact := fixtureArtifact(t, root)
		cache := filepath.Join(root, ArchiveRoot, "__pycache__")
		if err := os.MkdirAll(cache, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cache, "module.pyc"), []byte("cache"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "unrelated"), []byte("drift"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := RepairPythonCache(context.Background(), root, filepath.Join(t.TempDir(), "state"), artifact, false)
		if err == nil || !strings.Contains(err.Error(), "does not match pinned tree") {
			t.Fatalf("drift error=%v", err)
		}
		if _, err := os.Stat(filepath.Join(cache, "module.pyc")); err != nil {
			t.Fatalf("drift refusal changed cache: %v", err)
		}
	})

	for _, tc := range []struct {
		name string
		make func(*testing.T, string)
	}{
		{name: "mixed source file", make: func(t *testing.T, cache string) {
			if err := os.WriteFile(filepath.Join(cache, "module.pyc"), []byte("cache"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(cache, "module.py"), []byte("source"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "nested directory", make: func(t *testing.T, cache string) {
			if err := os.Mkdir(filepath.Join(cache, "nested"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "symlink child", make: func(t *testing.T, cache string) {
			if err := os.Symlink("missing.pyc", filepath.Join(cache, "linked.pyc")); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "symlink cache directory", make: func(t *testing.T, cache string) {
			if err := os.Remove(cache); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(t.TempDir(), cache); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "opt", ToolName)
			artifact := fixtureArtifact(t, root)
			cache := filepath.Join(root, ArchiveRoot, "__pycache__")
			if err := os.MkdirAll(cache, 0o755); err != nil {
				t.Fatal(err)
			}
			tc.make(t, cache)
			_, err := RepairPythonCache(context.Background(), root, filepath.Join(t.TempDir(), "state"), artifact, true)
			if err == nil || !strings.Contains(err.Error(), "refus") {
				t.Fatalf("unsafe cache accepted: %v", err)
			}
		})
	}
}

func TestRepairPythonCacheRefusesQuarantineInsideManagedTree(t *testing.T) {
	root := filepath.Join(t.TempDir(), "opt", ToolName)
	artifact := fixtureArtifact(t, root)
	cache := filepath.Join(root, ArchiveRoot, "__pycache__")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "x.pyc"), []byte("cache"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := RepairPythonCache(context.Background(), root, filepath.Join(root, "quarantine"), artifact, true)
	if err == nil || !strings.Contains(err.Error(), "outside the managed tool root") {
		t.Fatalf("inside quarantine error=%v", err)
	}
}

func TestRepairPythonCacheRequiresCleanAbsoluteRootsWithoutSymlinkAncestors(t *testing.T) {
	root := filepath.Join(t.TempDir(), "opt", ToolName)
	artifact := fixtureArtifact(t, root)
	cache := filepath.Join(root, ArchiveRoot, "__pycache__")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "x.pyc"), []byte("cache"), 0o644); err != nil {
		t.Fatal(err)
	}

	relativeRoot, relErr := filepath.Rel("/", root)
	if relErr != nil {
		t.Fatal(relErr)
	}
	if _, err := RepairPythonCache(context.Background(), relativeRoot, filepath.Join(t.TempDir(), "state"), artifact, true); err == nil || !strings.Contains(err.Error(), "absolute and clean") {
		t.Fatalf("relative managed root was accepted: %v", err)
	}
	linkRoot := t.TempDir()
	link := filepath.Join(linkRoot, "managed-link")
	if err := os.Symlink(filepath.Dir(root), link); err != nil {
		t.Fatal(err)
	}
	if _, err := RepairPythonCache(context.Background(), filepath.Join(link, ToolName), filepath.Join(t.TempDir(), "state"), artifact, true); err == nil || !strings.Contains(err.Error(), "symlink component") {
		t.Fatalf("symlinked managed ancestor was accepted: %v", err)
	}

	quarantineBase := filepath.Join(t.TempDir(), "outside")
	if err := os.Mkdir(quarantineBase, 0o755); err != nil {
		t.Fatal(err)
	}
	quarantineLink := filepath.Join(t.TempDir(), "state-link")
	if err := os.Symlink(quarantineBase, quarantineLink); err != nil {
		t.Fatal(err)
	}
	if _, err := RepairPythonCache(context.Background(), root, filepath.Join(quarantineLink, "repair"), artifact, false); err == nil || !strings.Contains(err.Error(), "symlink component") {
		t.Fatalf("symlink quarantine ancestor was accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cache, "x.pyc")); err != nil {
		t.Fatalf("invalid-root refusal changed managed tree: %v", err)
	}
}

func TestRepairPythonCacheRejectsMalformedPinnedTreeDigest(t *testing.T) {
	root := filepath.Join(t.TempDir(), "opt", ToolName)
	artifact := fixtureArtifact(t, root)
	artifact.TreeSHA256 = "not-a-digest"
	if _, err := RepairPythonCache(context.Background(), root, filepath.Join(t.TempDir(), "state"), artifact, true); err == nil || !strings.Contains(err.Error(), "SHA-256 hex digest") {
		t.Fatalf("malformed tree pin was accepted: %v", err)
	}
}

func TestWriteReceiptAtomicSyncsAndRemovesUncommittedReceiptOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "receipt.json")
	if err := writeReceiptAtomic(path, []byte("durable\n")); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "durable\n" {
		t.Fatalf("receipt=%q err=%v", got, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("receipt mode=%o, want 600", info.Mode().Perm())
	}

	failedPath := filepath.Join(dir, "failed.json")
	call := 0
	err = writeReceiptAtomicWith(failedPath, []byte("not durable"), func(string) error {
		call++
		if call == 1 {
			return errors.New("injected directory sync failure")
		}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "sync failure") {
		t.Fatalf("sync failure err=%v", err)
	}
	if _, statErr := os.Lstat(failedPath); !os.IsNotExist(statErr) {
		t.Fatalf("uncommitted receipt remains: %v", statErr)
	}
	if _, statErr := os.Lstat(failedPath + ".tmp"); !os.IsNotExist(statErr) {
		t.Fatalf("temporary receipt remains: %v", statErr)
	}
}
