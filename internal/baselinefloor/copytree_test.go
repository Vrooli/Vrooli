package baselinefloor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

// writeFile is a test helper that writes content to dir/rel, creating parents.
func writeFile(t *testing.T, dir, rel, content string, mode os.FileMode) string {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
	return full
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestCopyTree_CopiesTreePreservingContentAndMode(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")

	writeFile(t, src, "main.go", "package main", 0o644)
	writeFile(t, src, "api/server.go", "package api", 0o600)
	writeFile(t, src, "ui/src/app.js", "console.log(1)", 0o644)

	stats, err := CopyTree(src, dst, CopyOptions{Exclude: defaultExcludes})
	if err != nil {
		t.Fatalf("CopyTree: %v", err)
	}

	if got := readFile(t, filepath.Join(dst, "main.go")); got != "package main" {
		t.Errorf("main.go content = %q", got)
	}
	if got := readFile(t, filepath.Join(dst, "api/server.go")); got != "package api" {
		t.Errorf("server.go content = %q", got)
	}
	// Mode is preserved.
	fi, err := os.Stat(filepath.Join(dst, "api/server.go"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("server.go mode = %v, want 0600", fi.Mode().Perm())
	}
	if stats.ReflinkFiles+stats.DeepCopyFiles != 3 {
		t.Errorf("copied files = %d, want 3 (%+v)", stats.ReflinkFiles+stats.DeepCopyFiles, stats)
	}
}

func TestCopyTree_SkipsExcludedDirsAndFiles(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")

	writeFile(t, src, "main.go", "keep", 0o644)
	writeFile(t, src, "node_modules/dep/index.js", "drop", 0o644)
	writeFile(t, src, "ui/dist/bundle.js", "drop", 0o644)
	writeFile(t, src, "coverage/lcov.info", "drop", 0o644)
	writeFile(t, src, ".git/config", "drop", 0o644)
	writeFile(t, src, ".vrooli/service.json", "drop", 0o644)
	writeFile(t, src, ".build-fingerprint.json", "drop", 0o644)
	writeFile(t, src, "api/generated/pb.go", "drop", 0o644)

	if _, err := CopyTree(src, dst, CopyOptions{Exclude: defaultExcludes}); err != nil {
		t.Fatalf("CopyTree: %v", err)
	}

	mustExist := []string{"main.go"}
	for _, p := range mustExist {
		if _, err := os.Stat(filepath.Join(dst, p)); err != nil {
			t.Errorf("expected %s to be copied: %v", p, err)
		}
	}
	mustNotExist := []string{
		"node_modules", "ui/dist", "coverage", ".git", ".vrooli",
		".build-fingerprint.json", "api/generated",
	}
	for _, p := range mustNotExist {
		if _, err := os.Stat(filepath.Join(dst, p)); !os.IsNotExist(err) {
			t.Errorf("expected %s to be excluded, but it exists (err=%v)", p, err)
		}
	}
}

func TestCopyTree_RecreatesSymlinksVerbatim(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "symlink semantics differ on Windows")
	}
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")

	writeFile(t, src, "real.txt", "target", 0o644)
	// A relative symlink to a sibling file.
	if err := os.Symlink("real.txt", filepath.Join(src, "link.txt")); err != nil {
		t.Fatal(err)
	}
	// A symlink pointing into an excluded tree must remain a link, not a copy.
	writeFile(t, src, "node_modules/x/y.js", "excluded", 0o644)
	if err := os.Symlink("node_modules/x/y.js", filepath.Join(src, "danger.link")); err != nil {
		t.Fatal(err)
	}

	stats, err := CopyTree(src, dst, CopyOptions{Exclude: defaultExcludes})
	if err != nil {
		t.Fatalf("CopyTree: %v", err)
	}
	if stats.Symlinks != 2 {
		t.Errorf("symlinks = %d, want 2", stats.Symlinks)
	}

	target, err := os.Readlink(filepath.Join(dst, "link.txt"))
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != "real.txt" {
		t.Errorf("link target = %q, want real.txt", target)
	}
	// The dangerous link is preserved as a (now-dangling) link, NOT dereferenced
	// into a copy of excluded content.
	dl := filepath.Join(dst, "danger.link")
	if fi, err := os.Lstat(dl); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("danger.link should be a symlink, got fi=%v err=%v", fi, err)
	}
}

func TestCopyTree_OverlayIsIdempotent(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")
	writeFile(t, src, "a.txt", "v1", 0o644)

	if _, err := CopyTree(src, dst, CopyOptions{}); err != nil {
		t.Fatal(err)
	}
	// Change source and re-copy; dst must reflect the new content (overwrite).
	writeFile(t, src, "a.txt", "v2-longer", 0o644)
	if _, err := CopyTree(src, dst, CopyOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dst, "a.txt")); got != "v2-longer" {
		t.Errorf("after re-copy a.txt = %q, want v2-longer", got)
	}
}

func TestCopyTree_ReflinkOptionProducesCorrectContent(t *testing.T) {
	// On ext4 (this host) the reflink falls back to a deep copy; on btrfs/xfs it
	// clones. Either way the content must be byte-identical and the file counted.
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")
	writeFile(t, src, "data.bin", "reflink-or-fallback", 0o644)

	stats, err := CopyTree(src, dst, CopyOptions{Reflink: true})
	if err != nil {
		t.Fatalf("CopyTree reflink: %v", err)
	}
	if got := readFile(t, filepath.Join(dst, "data.bin")); got != "reflink-or-fallback" {
		t.Errorf("content = %q", got)
	}
	if stats.ReflinkFiles+stats.DeepCopyFiles != 1 {
		t.Errorf("file count = %d, want 1 (%+v)", stats.ReflinkFiles+stats.DeepCopyFiles, stats)
	}
}

func TestCopyTree_ErrorsWhenSrcNotDir(t *testing.T) {
	src := writeFile(t, t.TempDir(), "f.txt", "x", 0o644)
	if _, err := CopyTree(src, t.TempDir(), CopyOptions{}); err == nil {
		t.Fatal("expected error copying a non-directory src")
	}
	if _, err := CopyTree(filepath.Join(t.TempDir(), "missing"), t.TempDir(), CopyOptions{}); err == nil {
		t.Fatal("expected error for missing src")
	}
}

func TestDefaultExcludes_ReturnsIndependentCopy(t *testing.T) {
	a := DefaultExcludes()
	a["custom-artifact"] = struct{}{}
	if _, leaked := defaultExcludes["custom-artifact"]; leaked {
		t.Fatal("DefaultExcludes mutation leaked into the package default")
	}
	if _, ok := DefaultExcludes()["node_modules"]; !ok {
		t.Fatal("DefaultExcludes missing node_modules")
	}
}
