package lifecycle

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/scenario"
)

// freshnessTestScene builds a repo with one scenario whose api/ module has a
// repo-root replace and a binary that is newer than its sources. Returns the
// repo root, the scenario path, the binary path, and the main.go path.
func freshnessTestScene(t *testing.T) (repoRoot, appPath, binPath, srcPath string) {
	t.Helper()
	repoRoot = t.TempDir()
	appPath = filepath.Join(repoRoot, "scenarios", "alpha")
	apiDir := filepath.Join(appPath, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	old := time.Now().Add(-2 * time.Hour)
	srcPath = filepath.Join(apiDir, "main.go")
	if err := os.WriteFile(srcPath, []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module alpha\n\nreplace github.com/vrooli/vrooli => ../../..\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.Chtimes(srcPath, old, old); err != nil {
		t.Fatalf("chtimes src: %v", err)
	}
	binPath = filepath.Join(apiDir, "alpha-api")
	if err := os.WriteFile(binPath, []byte("\x7fELFbinary"), 0o755); err != nil {
		t.Fatalf("write bin: %v", err)
	}
	newer := time.Now()
	if err := os.Chtimes(binPath, newer, newer); err != nil {
		t.Fatalf("chtimes bin: %v", err)
	}
	return repoRoot, appPath, binPath, srcPath
}

func freshnessTestItem(appPath string) scenario.Scenario {
	return scenario.Scenario{
		Slug: "alpha",
		Path: appPath,
		Manifest: scenario.ServiceManifest{
			Components: map[string]scenario.Component{
				"api": {
					Build: scenario.ComponentBuild{Kind: "go_module", Dir: "api", Output: "api/alpha-api"},
					Run:   scenario.ComponentRun{Argv: []string{"{{bin.api}}"}},
				},
			},
		},
	}
}

func freshnessBinaryCheck() scenario.ConditionCheck {
	return scenario.ConditionCheck{Type: "binaries", Targets: []string{"api/alpha-api"}}
}

func TestBinariesFreshnessBootstrapStampsManifest(t *testing.T) {
	repoRoot, appPath, binPath, _ := freshnessTestScene(t)
	r := &Runner{Root: repoRoot}
	item := freshnessTestItem(appPath)

	stale, _, err := r.binariesFreshness(item, freshnessBinaryCheck())
	if err != nil {
		t.Fatalf("binariesFreshness: %v", err)
	}
	if !stale {
		t.Fatal("a runnable artifact without a manifest must be stale")
	}
	artifacts, err := componentFreshnessArtifacts(appPath, repoRoot, item.Manifest.Components["api"], defaultHostProbeDeps())
	if err != nil || len(artifacts) != 1 {
		t.Fatalf("freshness artifacts: %v", err)
	}
	if err := r.stampArtifactFreshness(artifacts[0]); err != nil {
		t.Fatalf("stamp manifest: %v", err)
	}
	// The explicit build stamp should make the next check manifest-authoritative.
	manifestPath := cliutil.FreshnessManifestPath(binPath)
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected manifest stamped at %s: %v", manifestPath, err)
	}
}

func TestBinariesFreshnessManifestAuthoritative(t *testing.T) {
	repoRoot, appPath, _, srcPath := freshnessTestScene(t)
	r := &Runner{Root: repoRoot}
	item := freshnessTestItem(appPath)
	check := freshnessBinaryCheck()

	artifacts, err := componentFreshnessArtifacts(appPath, repoRoot, item.Manifest.Components["api"], defaultHostProbeDeps())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.stampArtifactFreshness(artifacts[0]); err != nil {
		t.Fatal(err)
	}

	// Editing the real input content must now read stale via the manifest.
	if err := os.WriteFile(srcPath, []byte("package main\nfunc main(){ _ = 1 }\n"), 0o644); err != nil {
		t.Fatalf("edit src: %v", err)
	}
	stale, reason, err := r.binariesFreshness(item, check)
	if err != nil {
		t.Fatalf("binariesFreshness: %v", err)
	}
	if !stale {
		t.Fatal("content edit to a real input must be stale")
	}
	if reason == "" {
		t.Fatal("stale reason should be populated")
	}
}

func TestBinariesFreshnessUnrelatedScenarioEditStaysFresh(t *testing.T) {
	repoRoot, appPath, _, _ := freshnessTestScene(t)
	r := &Runner{Root: repoRoot}
	item := freshnessTestItem(appPath)
	check := freshnessBinaryCheck()

	artifacts, err := componentFreshnessArtifacts(appPath, repoRoot, item.Manifest.Components["api"], defaultHostProbeDeps())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.stampArtifactFreshness(artifacts[0]); err != nil {
		t.Fatal(err)
	}

	// Headline regression: editing an UNRELATED scenario anywhere under the
	// repo root (reachable only via the repo-root replace, which is excluded)
	// must NOT mark alpha stale.
	betaDir := filepath.Join(repoRoot, "scenarios", "beta", "api")
	if err := os.MkdirAll(betaDir, 0o755); err != nil {
		t.Fatalf("mkdir beta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(betaDir, "main.go"), []byte("package main // beta edit"), 0o644); err != nil {
		t.Fatalf("write beta: %v", err)
	}
	stale, reason, err := r.binariesFreshness(item, check)
	if err != nil {
		t.Fatalf("binariesFreshness: %v", err)
	}
	if stale {
		t.Fatalf("unrelated scenario edit must not mark alpha stale, got reason %q", reason)
	}
}

func TestBinariesFreshnessToolchainChangeIsStale(t *testing.T) {
	repoRoot, appPath, binPath, _ := freshnessTestScene(t)
	r := &Runner{Root: repoRoot}
	item := freshnessTestItem(appPath)
	check := freshnessBinaryCheck()

	artifacts, err := componentFreshnessArtifacts(appPath, repoRoot, item.Manifest.Components["api"], defaultHostProbeDeps())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.stampArtifactFreshness(artifacts[0]); err != nil {
		t.Fatal(err)
	}

	// Simulate a toolchain upgrade by rewriting the recorded manifest's keyed
	// toolchain input. Identical source, different toolchain -> stale.
	manifestPath := cliutil.FreshnessManifestPath(binPath)
	m, ok, err := cliutil.ReadFreshnessManifest(manifestPath)
	if err != nil || !ok {
		t.Fatalf("read manifest: ok=%v err=%v", ok, err)
	}
	if m.KeyInputs == nil {
		m.KeyInputs = map[string]string{}
	}
	m.KeyInputs["toolchain"] = "go version go0.0.0 test/test"
	if err := cliutil.WriteFreshnessManifest(manifestPath, m); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	stale, reason, err := r.binariesFreshness(item, check)
	if err != nil {
		t.Fatalf("binariesFreshness: %v", err)
	}
	if !stale {
		t.Fatalf("toolchain change must mark stale, reason=%q", reason)
	}
}
