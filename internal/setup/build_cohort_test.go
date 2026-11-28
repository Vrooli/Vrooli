package setup

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
)

func gitFixture(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.invalid",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.invalid",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func writeSource(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A build compiles the snapshot, not the live checkout. An agent editing the
// shared worktree mid-build cannot change what the artifact contains, so the
// artifact and its fingerprint always describe one moment.
func TestBuildFromSnapshotIgnoresConcurrentEdits(t *testing.T) {
	root := t.TempDir()
	gitFixture(t, root, "init", "--quiet", "--initial-branch=main")
	writeSource(t, root, "go.mod", "module example.invalid/fixture\n\ngo 1.25\n")
	writeSource(t, root, "internal/main.go", "package internal\n\nconst Value = 1\n")
	gitFixture(t, root, "add", "-A")
	gitFixture(t, root, "commit", "--quiet", "-m", "fixture")

	var compiled string
	var compiledFingerprint string
	err := buildFromSnapshot(context.Background(), root, []string{"internal", "go.mod"}, func(buildRoot string) error {
		data, err := os.ReadFile(filepath.Join(buildRoot, "internal", "main.go"))
		if err != nil {
			return err
		}
		compiled = string(data)
		compiledFingerprint, err = buildinfo.ComputeSourceFingerprintForPaths(buildRoot, "internal", "go.mod")
		if err != nil {
			return err
		}
		// A concurrent agent edits the shared checkout while the build runs.
		return os.WriteFile(filepath.Join(root, "internal", "main.go"), []byte("package internal\n\nconst Value = 2\n"), 0o644)
	})
	if err != nil {
		t.Fatalf("build from snapshot: %v", err)
	}
	if !strings.Contains(compiled, "Value = 1") {
		t.Fatalf("the build saw a concurrent edit instead of the snapshot: %q", compiled)
	}
	if compiledFingerprint == "" {
		t.Fatal("the build has no fingerprint")
	}

	// The live checkout has moved on, so the artifact is stale relative to it,
	// but the artifact itself is coherent.
	liveFingerprint, err := buildinfo.ComputeSourceFingerprintForPaths(root, "internal", "go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if liveFingerprint == compiledFingerprint {
		t.Fatal("the checkout fingerprint did not move after a real edit")
	}
}
