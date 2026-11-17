package buildinfo

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The build-cache helper serves every go command once activated, so a stale
// helper must be detected. Its fingerprint must cover every repository package
// it links; the narrow target list is only correct while that holds.
func TestGoCacheProgramFingerprintCoversEveryLinkedRepositoryPackage(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	list := exec.Command(goBin, "list", "-deps", "-f", "{{if not .Standard}}{{.Dir}}{{end}}", "./cmd/vrooli-gocacheprog")
	list.Dir = root
	output, err := list.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	targets := FingerprintTargetsForExecutable("vrooli-gocacheprog")
	for _, dir := range strings.Fields(string(output)) {
		rel, err := filepath.Rel(root, dir)
		if err != nil || strings.HasPrefix(rel, "..") {
			t.Errorf("helper links %s from outside the repository; its fingerprint cannot cover it", dir)
			continue
		}
		covered := false
		for _, target := range targets {
			if rel == target || strings.HasPrefix(rel, target+string(filepath.Separator)) {
				covered = true
			}
		}
		if !covered {
			t.Errorf("helper links %s, which its fingerprint targets %v do not cover", rel, targets)
		}
	}
}
