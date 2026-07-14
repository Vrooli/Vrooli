package crosscompile

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// moduleRoot walks up from this test file to the api module root (the
// directory containing go.mod).
func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("crosscompile: cannot resolve caller path")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("crosscompile: go.mod not found above test file")
		}
		dir = parent
	}
}

// TestCrossCompile is the permanent OS-seam gate. The api module MUST build
// CGO-free for the non-Linux targets workspace-sandbox stays portable to.
// A Linux-only syscall used outside a //go:build-tagged file breaks one of
// these builds and fails this test, keeping the fsmount/namespace/process
// seams honest across platforms. This mirrors `make cross-compile`.
func TestCrossCompile(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Fatalf("crosscompile: go toolchain not found on PATH: %v", err)
	}
	root := moduleRoot(t)

	targets := []struct{ goos, goarch string }{
		{"darwin", "arm64"},
		{"windows", "amd64"},
	}
	for _, tgt := range targets {
		tgt := tgt
		t.Run(tgt.goos+"/"+tgt.goarch, func(t *testing.T) {
			cmd := exec.Command("go", "build", "./...")
			cmd.Dir = root
			cmd.Env = append(os.Environ(),
				"GOOS="+tgt.goos,
				"GOARCH="+tgt.goarch,
				"CGO_ENABLED=0",
				"GOWORK=off",
			)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("cross-compile %s/%s failed: %v\n%s", tgt.goos, tgt.goarch, err, out)
			}
		})
	}
}
