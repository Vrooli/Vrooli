package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/scenario"
)

// phase0BinScene builds a real-disk repo with one scenario whose api/ module has
// a repo-root replace, returning the repo root, scenario path, api dir, the
// binary path, and the main.go path. Callers set mtimes per test. These tests
// exercise the live bootstrap mtime path (binariesFreshness with no recorded
// manifest), which is what the deleted binariesNeedSetupWithDeps used to cover.
func phase0BinScene(t *testing.T) (repoRoot, appPath, apiDir, binPath, srcPath string) {
	t.Helper()
	repoRoot = t.TempDir()
	appPath = filepath.Join(repoRoot, "scenarios", "alpha")
	apiDir = filepath.Join(appPath, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	srcPath = filepath.Join(apiDir, "main.go")
	if err := os.WriteFile(srcPath, []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	// Repo-root replace: the input adapter must never let an edit reachable only
	// through it mark this scenario stale.
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module alpha\n\nreplace github.com/vrooli/vrooli => ../../..\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	binPath = filepath.Join(apiDir, "alpha-api")
	if err := os.WriteFile(binPath, []byte("\x7fELFbinary"), 0o755); err != nil {
		t.Fatalf("write bin: %v", err)
	}
	return repoRoot, appPath, apiDir, binPath, srcPath
}

func chtimes(t *testing.T, path string, ts time.Time) {
	t.Helper()
	if err := os.Chtimes(path, ts, ts); err != nil {
		t.Fatalf("chtimes %s: %v", path, err)
	}
}

func TestBinariesFreshnessMissingBinary(t *testing.T) {
	repoRoot, appPath, _, binPath, _ := phase0BinScene(t)
	if err := os.Remove(binPath); err != nil {
		t.Fatalf("remove bin: %v", err)
	}
	r := &Runner{Root: repoRoot}
	item := freshnessTestItem(appPath)
	stale, reason, err := r.binariesFreshness(item, freshnessBinaryCheck())
	if err != nil {
		t.Fatalf("binariesFreshness: %v", err)
	}
	if !stale {
		t.Fatal("expected stale when binary missing")
	}
	if !strings.Contains(reason, "Missing artifact") || !strings.Contains(reason, "api/alpha-api") {
		t.Fatalf("reason should name the missing binary, got %q", reason)
	}
}

func TestBinariesFreshnessSourceNewerNamesFile(t *testing.T) {
	repoRoot, appPath, apiDir, binPath, srcPath := phase0BinScene(t)
	old := time.Now().Add(-2 * time.Hour)
	chtimes(t, binPath, old)
	chtimes(t, filepath.Join(apiDir, "go.mod"), old)
	chtimes(t, srcPath, time.Now()) // source newer than binary
	r := &Runner{Root: repoRoot}
	item := freshnessTestItem(appPath)
	stale, reason, err := r.binariesFreshness(item, freshnessBinaryCheck())
	if err != nil {
		t.Fatalf("binariesFreshness: %v", err)
	}
	if !stale {
		t.Fatal("expected stale when source newer than binary")
	}
	if !strings.Contains(reason, "source newer") || !strings.Contains(reason, "main.go") {
		t.Fatalf("reason should name the offending file, got %q", reason)
	}
}

func TestBinariesFreshnessIgnoresTestFiles(t *testing.T) {
	repoRoot, appPath, apiDir, binPath, srcPath := phase0BinScene(t)
	old := time.Now().Add(-2 * time.Hour)
	chtimes(t, srcPath, old)
	chtimes(t, filepath.Join(apiDir, "go.mod"), old)
	chtimes(t, binPath, old)
	// Only a newer _test.go file: never changes the binary, must not be stale.
	testFile := filepath.Join(apiDir, "main_test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	chtimes(t, testFile, time.Now())
	r := &Runner{Root: repoRoot}
	item := freshnessTestItem(appPath)
	stale, reason, err := r.binariesFreshness(item, freshnessBinaryCheck())
	if err != nil {
		t.Fatalf("binariesFreshness: %v", err)
	}
	if stale {
		t.Fatalf("editing a _test.go must not mark binary stale, got reason %q", reason)
	}
}

func TestBinariesFreshnessFresh(t *testing.T) {
	repoRoot, appPath, _, binPath, srcPath := phase0BinScene(t)
	chtimes(t, srcPath, time.Now().Add(-2*time.Hour))
	chtimes(t, binPath, time.Now())
	r := &Runner{Root: repoRoot}
	item := freshnessTestItem(appPath)
	stale, _, err := r.binariesFreshness(item, freshnessBinaryCheck())
	if err != nil {
		t.Fatalf("binariesFreshness: %v", err)
	}
	if stale {
		t.Fatal("binary newer than all sources must be fresh")
	}
}

func TestForceSetupFor(t *testing.T) {
	cases := []struct {
		name string
		opts StartOptions
		slug string
		want bool
	}{
		{"no force", StartOptions{}, "alpha", false},
		{"force all", StartOptions{ForceSetup: true}, "alpha", true},
		{"force specific match", StartOptions{ForceSetup: true, ForceSetupScenario: "alpha"}, "alpha", true},
		{"force specific other", StartOptions{ForceSetup: true, ForceSetupScenario: "beta"}, "alpha", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := forceSetupFor(tc.opts, tc.slug); got != tc.want {
				t.Fatalf("forceSetupFor=%v want %v", got, tc.want)
			}
		})
	}
}

func TestDependencyRestartReason(t *testing.T) {
	cases := []struct {
		name        string
		running     bool
		healthy     bool
		stale       bool
		reasons     []string
		wantContain string
	}{
		{"not running", false, false, false, nil, "not running"},
		{"unhealthy", true, false, false, nil, "unhealthy"},
		{"stale only", true, true, true, []string{"Source newer than binary api: main.go"}, "setup needed: Source newer"},
		{"stale no reasons", true, true, true, nil, "setup needed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := dependencyRestartReason(tc.running, tc.healthy, tc.stale, tc.reasons)
			if !strings.Contains(got, tc.wantContain) {
				t.Fatalf("reason %q does not contain %q", got, tc.wantContain)
			}
		})
	}
}

// binariesFreshness keeps the focused fingerprint tests at the artifact seam;
// production setup authority comes exclusively from components[].build.
func (r *Runner) binariesFreshness(item scenario.Scenario, check scenario.ConditionCheck) (bool, string, error) {
	deps := defaultHostProbeDeps()
	artifacts, err := binariesFreshnessArtifacts(item.Path, r.Root, check, deps)
	if err != nil {
		return false, "", err
	}
	for _, artifact := range artifacts {
		verdict, err := r.evaluateArtifactFreshness(artifact, deps)
		if err != nil {
			return false, "", err
		}
		if verdict.Stale {
			return true, verdict.HumanReason, nil
		}
	}
	return false, "", nil
}
