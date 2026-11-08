package lifecycle

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/packagegov"
	"github.com/vrooli/vrooli/internal/scenario"
)

// installCompletenessFixture builds a repo with one governed file: package
// and a UI whose node_modules copy matches the package's build outputs.
func installCompletenessFixture(t *testing.T) (repoRoot, uiDir, packageRoot string) {
	t.Helper()
	repoRoot = t.TempDir()
	packageRoot = filepath.Join(repoRoot, "packages", "demo")
	uiDir = filepath.Join(repoRoot, "scenarios", "alpha", "ui")
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(packageRoot, ".vrooli", "package.json"), `{"$schema":"schemas/package.schema.json","version":"1.0.0","package":{"name":"demo","display_name":"@vrooli/demo","description":"Fixture package.","kind":"js_runtime","language":"javascript","module_identifiers":["@vrooli/demo"],
		"adoption":{"scenario_adoptable":true,"allowed_consumers":["scenario_ui"],"adoption_modes":["file_dependency"]},
		"lifecycle":{"build":[{"name":"build","run":["node","build.mjs"],"outputs":["dist/**"]}]},
		"refresh":{"strategy":"scenario_setup","restart_running_consumers":false},"docs":["packages/demo/README.md"]}}`)
	write(filepath.Join(packageRoot, "package.json"), `{"name":"@vrooli/demo","version":"1.0.0","files":["dist"]}`)
	write(filepath.Join(packageRoot, "dist", "server", "index.js"), "export * from './notifications.js';\n")
	write(filepath.Join(uiDir, "package.json"), `{"name":"alpha-ui","dependencies":{"@vrooli/demo":"file:../../../packages/demo"}}`)
	write(filepath.Join(uiDir, "node_modules", "@vrooli", "demo", "package.json"), `{"name":"@vrooli/demo","version":"1.0.0"}`)
	write(filepath.Join(uiDir, "node_modules", "@vrooli", "demo", "dist", "server", "index.js"), "export * from './notifications.js';\n")
	return repoRoot, uiDir, packageRoot
}

func TestIncompleteSharedPackageInstallDetectsFileAddedAfterInstall(t *testing.T) {
	repoRoot, uiDir, packageRoot := installCompletenessFixture(t)
	if gap, incomplete := incompleteSharedPackageInstall(repoRoot, uiDir); incomplete {
		t.Fatalf("a complete install must not read as incomplete: %s", gap)
	}

	// The package rebuild adds a module the installed copy never received.
	if err := os.WriteFile(filepath.Join(packageRoot, "dist", "server", "notifications.js"), []byte("export const x = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gap, incomplete := incompleteSharedPackageInstall(repoRoot, uiDir)
	if !incomplete || !strings.Contains(gap, "@vrooli/demo") || !strings.Contains(gap, "dist/server/notifications.js") {
		t.Fatalf("missing installed output must be named, got incomplete=%t gap=%q", incomplete, gap)
	}

	// The scenario-level setup verdict must turn stale on the same evidence,
	// so a restart reinstalls instead of reporting the scenario fresh.
	item := scenario.Scenario{Slug: "alpha", Path: filepath.Dir(uiDir)}
	component := scenario.Component{Role: "ui", Build: scenario.ComponentBuild{Kind: "pnpm_vite", Dir: "ui"}}
	if gap, incomplete := componentInstallGap(item, component, repoRoot); !incomplete || gap == "" {
		t.Fatal("component install gap must report the incomplete install")
	}
}

func TestIncompleteSharedPackageInstallIgnoresUnshippedOutputDirectories(t *testing.T) {
	repoRoot, uiDir, packageRoot := installCompletenessFixture(t)
	// An output directory the installed copy does not ship is not a gap.
	if err := os.MkdirAll(filepath.Join(packageRoot, "dist", "dev-only"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "dist", "dev-only", "fixture.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if gap, incomplete := incompleteSharedPackageInstall(repoRoot, uiDir); incomplete {
		t.Fatalf("unshipped output directories must not force a reinstall: %s", gap)
	}
}

// hardlinkInstalledCopy replaces the fixture's installed copy with a hard link
// to the package output, the shape a pnpm file: install produces.
func hardlinkInstalledCopy(t *testing.T, uiDir, packageRoot string) string {
	t.Helper()
	installed := filepath.Join(uiDir, "node_modules", "@vrooli", "demo", "dist", "server", "index.js")
	if err := os.Remove(installed); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(packageRoot, "dist", "server", "index.js"), installed); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	return installed
}

func rebuildPackage(t *testing.T, packageRoot, script string) {
	t.Helper()
	build := packagegov.CommandSpec{Name: "build", Run: []string{"sh", "-c", script}}
	if err := packagegov.RunCommandsWithOptions(packageRoot, []packagegov.CommandSpec{build}, io.Discard, io.Discard, packagegov.CommandOptions{Stdin: strings.NewReader("")}); err != nil {
		t.Fatalf("governed rebuild: %v", err)
	}
}

func TestGovernedRebuildKeepsHardlinkedConsumerOnOldSnapshotAndMarksItStale(t *testing.T) {
	repoRoot, uiDir, packageRoot := installCompletenessFixture(t)
	installed := hardlinkInstalledCopy(t, uiDir, packageRoot)
	if gap, incomplete := incompleteSharedPackageInstall(repoRoot, uiDir); incomplete {
		t.Fatalf("a hard-linked install must read complete: %s", gap)
	}

	// The rebuild rewrites index.js in place and adds the module it imports.
	rebuildPackage(t, packageRoot, "printf \"export * from './notifications.js';\\nexport * from './added.js';\\n\" > dist/server/index.js && printf 'export const a = 1;\\n' > dist/server/added.js")

	raw, err := os.ReadFile(installed)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "export * from './notifications.js';\n" {
		t.Fatalf("consumer copy = %q, want the old consistent snapshot", raw)
	}
	if gap, incomplete := incompleteSharedPackageInstall(repoRoot, uiDir); !incomplete || !strings.Contains(gap, "@vrooli/demo") {
		t.Fatalf("detached consumer must read stale, got incomplete=%t gap=%q", incomplete, gap)
	}
}

func TestGovernedContentOnlyRebuildMarksDetachedConsumerStale(t *testing.T) {
	repoRoot, uiDir, packageRoot := installCompletenessFixture(t)
	hardlinkInstalledCopy(t, uiDir, packageRoot)

	// Same size, different bytes, no new file: only a content comparison sees it.
	rebuildPackage(t, packageRoot, "printf \"export * from './notificationz.js';\\n\" > dist/server/index.js")

	gap, incomplete := incompleteSharedPackageInstall(repoRoot, uiDir)
	if !incomplete || !strings.Contains(gap, "dist/server/index.js") {
		t.Fatalf("content-changed output must read stale, got incomplete=%t gap=%q", incomplete, gap)
	}
}
