package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRootForTest locates the working tree that owns the companion registry.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 12; i++ {
		_, packagesErr := os.Stat(filepath.Join(dir, companionPackagesRoot))
		_, registryErr := os.Stat(filepath.Join(dir, ".vrooli", "test-companions.json"))
		if packagesErr == nil && registryErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skip("companion registry is not reachable from this working tree")
	return ""
}

// TestCompanionRegistryTracksPackageExports is the anti-drift gate. The
// companion rules can only detect duplicates of symbols the registry lists, so
// an unlisted export is an undetectable duplicate, and nothing else in the tree
// would report that.
func TestCompanionRegistryTracksPackageExports(t *testing.T) {
	root := repoRootForTest(t)
	registryPath := filepath.Join(root, ".vrooli", "test-companions.json")

	current, ok := loadCompanionRegistry(root)
	if !ok {
		t.Fatalf("could not load %s; it must be valid JSON at schema %s", registryPath, companionRegistrySchemaVersion)
	}

	generated, err := generateCompanionRegistry(root, current.Seams)
	if err != nil {
		t.Fatalf("generate companion registry: %v", err)
	}
	want, err := marshalCompanionRegistry(generated)
	if err != nil {
		t.Fatalf("marshal companion registry: %v", err)
	}
	got, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("read %s: %v", registryPath, err)
	}
	if string(got) == string(want) {
		return
	}

	if os.Getenv(companionRegistryUpdateEnv) == "1" {
		if writeErr := os.WriteFile(registryPath, want, 0o644); writeErr != nil {
			t.Fatalf("write %s: %v", registryPath, writeErr)
		}
		// Fail after writing on purpose: a run that regenerates the registry has
		// not verified it, and a passing green here would let an unreviewed
		// rewrite land as though it had been checked.
		t.Fatalf("regenerated %s; review the diff and re-run without %s", registryPath, companionRegistryUpdateEnv)
	}

	lines := diffCompanionRegistries(generated, current)
	t.Fatalf("%s no longer matches the exported companion surface:\n  %s\nRegenerate with %s=1 go test ./internal/validation/",
		registryPath, strings.Join(lines, "\n  "), companionRegistryUpdateEnv)
}

// TestCompanionRegistryCoversEveryConventionalCompanion guards the specific
// failure that motivated generation: connectxtest is named as canonical in the
// shared-package testing convention, yet the hand-maintained registry never
// listed it, so every local reimplementation of it was invisible.
func TestCompanionRegistryCoversEveryConventionalCompanion(t *testing.T) {
	root := repoRootForTest(t)
	exports, err := discoverCompanionExports(root)
	if err != nil {
		t.Fatalf("discover companions: %v", err)
	}
	found := map[string]bool{}
	for _, export := range exports {
		found[export.ImportPath] = true
	}
	for _, importPath := range []string{
		"github.com/vrooli/api-core/apihttptest",
		"github.com/vrooli/api-core/connectxtest",
		"github.com/vrooli/api-core/databasetest",
		"github.com/vrooli/api-core/scheduletest",
		"github.com/vrooli/api-core/servertest",
		"github.com/vrooli/cli-core/cliapptest",
	} {
		if !found[importPath] {
			t.Errorf("companion %s was not discovered; discovery found %v", importPath, sortedImportPaths(exports))
		}
	}
}

// TestCompanionDiscoveryIgnoresNonModuleDirectories proves discovery reads each
// module's own go.mod. A directory with no module of its own must not inherit
// the repository module path and invent companions beneath it.
func TestCompanionDiscoveryIgnoresNonModuleDirectories(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, companionPackagesRoot, "ts-only", "src", "index.ts"), "export const x = 1;\n")
	writeFile(t, filepath.Join(root, companionPackagesRoot, "go-mod", "go.mod"), "module example.com/gomod\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, companionPackagesRoot, "go-mod", "widget", "widget.go"), "package widget\n")
	writeFile(t, filepath.Join(root, companionPackagesRoot, "go-mod", "widgettest", "widget.go"), `package widgettest

func Helper() {}
`)

	exports, err := discoverCompanionExports(root)
	if err != nil {
		t.Fatalf("discover companions: %v", err)
	}
	if len(exports) != 1 {
		t.Fatalf("discovered %v, want exactly the widgettest companion", sortedImportPaths(exports))
	}
	if exports[0].ImportPath != "example.com/gomod/widgettest" {
		t.Errorf("import path = %q, want example.com/gomod/widgettest", exports[0].ImportPath)
	}
	if exports[0].Owner != "go-mod/widget" {
		t.Errorf("owner = %q, want go-mod/widget", exports[0].Owner)
	}
}

// TestCompanionDiscoverySkipsExternalTestPackages keeps `foo_test`, which is
// the external test package for foo, from being read as a companion for `foo_`.
func TestCompanionDiscoverySkipsExternalTestPackages(t *testing.T) {
	if _, ok := companionOwnerPackage("widget_test"); ok {
		t.Error("widget_test must not be treated as a companion package")
	}
	if _, ok := companionOwnerPackage("test"); ok {
		t.Error("test must not be treated as a companion package")
	}
	owned, ok := companionOwnerPackage("widgettest")
	if !ok || owned != "widget" {
		t.Errorf("companionOwnerPackage(widgettest) = %q,%v; want widget,true", owned, ok)
	}
}

func sortedImportPaths(exports []companionExport) []string {
	paths := make([]string, 0, len(exports))
	for _, export := range exports {
		paths = append(paths, export.ImportPath)
	}
	return paths
}
