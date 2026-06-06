package baselinefloor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureRestore_RoundTrip(t *testing.T) {
	workTree := t.TempDir()
	writeFile(t, workTree, "main.go", "package main", 0o644)
	writeFile(t, workTree, "api/handler.go", "package api", 0o644)
	// Build artifacts that must be excluded from the restore point.
	writeFile(t, workTree, "node_modules/x/i.js", "junk", 0o644)
	writeFile(t, workTree, "ui/dist/b.js", "junk", 0o644)

	rp := filepath.Join(t.TempDir(), "restore-point")
	stats, err := Capture(workTree, rp, nil)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if stats.ReflinkFiles+stats.DeepCopyFiles != 2 {
		t.Errorf("captured files = %d, want 2 (excludes applied) %+v", stats.ReflinkFiles+stats.DeepCopyFiles, stats)
	}
	if _, err := os.Stat(filepath.Join(rp, "node_modules")); !os.IsNotExist(err) {
		t.Errorf("node_modules leaked into restore point")
	}

	// Restore into a fresh dir; source files come back, excluded ones do not.
	dest := filepath.Join(t.TempDir(), "restored")
	if _, err := Restore(rp, dest, nil); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got := readFile(t, filepath.Join(dest, "main.go")); got != "package main" {
		t.Errorf("restored main.go = %q", got)
	}
	if got := readFile(t, filepath.Join(dest, "api/handler.go")); got != "package api" {
		t.Errorf("restored handler.go = %q", got)
	}
}

func TestRestore_IsOverlayNotMirror(t *testing.T) {
	// Capture a tree, then simulate abandoned work that added a new file to the
	// working tree. Restoring overlays the restore point but leaves the new file
	// in place ("park dirty work", not a destructive mirror).
	workTree := t.TempDir()
	writeFile(t, workTree, "keep.go", "original", 0o644)

	rp := filepath.Join(t.TempDir(), "rp")
	if _, err := Capture(workTree, rp, nil); err != nil {
		t.Fatalf("Capture: %v", err)
	}

	// Diverge the working tree after the capture.
	writeFile(t, workTree, "keep.go", "edited-after-capture", 0o644)
	writeFile(t, workTree, "new_work.go", "added-after-capture", 0o644)

	if _, err := Restore(rp, workTree, nil); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	// Captured file is rolled back...
	if got := readFile(t, filepath.Join(workTree, "keep.go")); got != "original" {
		t.Errorf("keep.go = %q, want original (rolled back)", got)
	}
	// ...but the newly added file survives (overlay, not mirror).
	if got := readFile(t, filepath.Join(workTree, "new_work.go")); got != "added-after-capture" {
		t.Errorf("new_work.go = %q, want preserved", got)
	}
}

func TestCapture_ErrorsOnMissingSource(t *testing.T) {
	if _, err := Capture(filepath.Join(t.TempDir(), "nope"), t.TempDir(), nil); err == nil {
		t.Fatal("expected error capturing a missing source")
	}
}

func TestResolveCopyOptions_DefaultsAndExcludeFill(t *testing.T) {
	def := resolveCopyOptions(nil)
	if !def.Reflink {
		t.Error("nil opts should enable reflink")
	}
	if _, ok := def.Exclude["node_modules"]; !ok {
		t.Error("nil opts should apply default excludes")
	}
	// A supplied options value with a nil Exclude map gets the defaults filled.
	filled := resolveCopyOptions(&CopyOptions{Reflink: false})
	if _, ok := filled.Exclude["node_modules"]; !ok {
		t.Error("nil Exclude should be filled with defaults")
	}
	if filled.Reflink {
		t.Error("explicit Reflink:false should be preserved")
	}
}
