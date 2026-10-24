package lifecycle

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/packagegov"
	packagefixture "github.com/vrooli/vrooli/internal/packagegov/packagegovtest"
	"github.com/vrooli/vrooli/internal/scenario"
)

// basShapedRepo mirrors browser-automation-studio's build shape: the api
// replaces the whole repository module and a separately versioned package;
// the ui has pnpm file: dependencies (one governed package that requires
// another, one generated Proto output) and Vite/tsconfig aliases into the
// Proto output.
func basShapedRepo(t *testing.T) (repo string, item scenario.Scenario) {
	t.Helper()
	repo = t.TempDir()
	w := func(rel, content string) {
		testkitgo.WriteFile(t, filepath.Join(repo, filepath.FromSlash(rel)), content)
	}
	w("go.mod", "module example.com/repo\n\ngo 1.22\n")
	w("go.sum", "")
	w("internal/tuning/tuning.go", "package tuning\n\nconst Limit = 1\n")
	w("internal/tuning/deep/deep.go", "package deep\n\nimport \"example.com/repo/internal/tuning\"\n\nconst V = tuning.Limit\n")
	w("internal/unused/unused.go", "package unused\n")
	// A repository-module package outside internal/ (packages/monetization-go
	// in the real repository) that reaches internal packages.
	w("packages/money/money.go", "package money\n\nimport \"example.com/repo/internal/tuning/deep\"\n\nconst Cents = deep.V\n")
	w("packages/apicore/go.mod", "module example.com/apicore\n\ngo 1.22\n")
	w("packages/apicore/server/server.go", "package server\n\nconst Port = 1\n")
	w("packages/proto/gen/typescript/demo/v1/demo.ts", "export const demo = 1;\n")
	w("packages/proto/gen/.vrooli-proto-artifact.json", `{"artifact_id":"art-1"}`)
	packagefixture.WritePackageManifest(t, repo, "widgets", packagefixture.PackageManifest("widgets",
		packagefixture.WithPackageKind(packagegov.KindJSRuntime),
		packagefixture.WithPackageModuleIdentifiers("@vrooli/widgets"),
		packagefixture.WithPackageLifecycle(packagegov.LifecyclePolicy{
			Requires: []string{"icons"},
			Build:    []packagegov.CommandSpec{{Name: "build", Run: []string{"pnpm", "build"}, Outputs: []string{"dist/**"}}},
		}),
	))
	w("packages/widgets/package.json", `{"name":"@vrooli/widgets"}`)
	w("packages/widgets/dist/index.js", "export {};\n")
	packagefixture.WritePackageManifest(t, repo, "icons", packagefixture.PackageManifest("icons",
		packagefixture.WithPackageKind(packagegov.KindJSRuntime),
		packagefixture.WithPackageModuleIdentifiers("@vrooli/icons"),
	))
	w("packages/icons/package.json", `{"name":"@vrooli/icons"}`)

	dir := filepath.Join(repo, "scenarios", "demo")
	w("scenarios/demo/api/go.mod", strings.Join([]string{
		"module example.com/demo",
		"",
		"go 1.22",
		"",
		"require (",
		"\texample.com/repo v0.0.0",
		"\texample.com/apicore v0.0.0",
		")",
		"",
		"replace example.com/repo => ../../..",
		"",
		"replace example.com/apicore => ../../../packages/apicore",
		"",
	}, "\n"))
	w("scenarios/demo/api/main.go", "package main\n\nimport (\n\t\"example.com/apicore/server\"\n\t\"example.com/repo/packages/money\"\n)\n\nfunc main() { _ = server.Port + money.Cents }\n")
	w("scenarios/demo/ui/package.json", `{"dependencies":{"@vrooli/widgets":"file:../../../packages/widgets","@vrooli/proto-types":"file:../../../packages/proto/gen/typescript"}}`)
	w("scenarios/demo/ui/vite.config.ts", "export default { resolve: { alias: { '@vrooli/generated-proto': path.resolve(__dirname, '../../../packages/proto/gen/typescript'), '@local': './src' } } };\n")
	w("scenarios/demo/ui/tsconfig.json", `{"compilerOptions":{"paths":{"@vrooli/proto-types/*":["../../../packages/proto/gen/typescript/*"]}}}`)
	item = scenario.Scenario{
		Slug: "demo",
		Path: dir,
		Manifest: scenario.ServiceManifest{Components: map[string]scenario.Component{
			"api": {Role: "api", Build: scenario.ComponentBuild{Kind: "go_module", Dir: "api"}},
			"ui":  {Role: "ui", Build: scenario.ComponentBuild{Kind: "pnpm_vite", Dir: "ui"}},
		}},
	}
	return repo, item
}

func TestSharedSourceRootsResolvesBASShapedScenario(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain unavailable")
	}
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOPROXY", "off")
	repo, item := basShapedRepo(t)
	roots, err := SharedSourceRoots(context.Background(), repo, item)
	if err != nil {
		t.Fatalf("SharedSourceRoots: %v", err)
	}
	got := map[string]string{}
	for _, root := range roots {
		got[root.Path] = root.Reason
	}
	for _, want := range []string{
		"go.mod",                        // the replaced repository module
		"go.sum",                        //
		"internal/tuning",               // repository-module packages, cut to two levels
		"packages/apicore/go.mod",       // separately versioned module
		"packages/apicore/server",       //
		"packages/money",                // repository-module package outside internal/
		"packages/widgets",              // file: dependency
		"packages/icons",                // its governed requires
		"packages/proto/gen/typescript", // file: + alias + tsconfig paths, cut by nesting
		"packages/proto/gen/.vrooli-proto-artifact.json", // frozen Proto artifact identity
	} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing root %s (got %v)", want, got)
		}
	}
	for path := range got {
		switch {
		case strings.HasPrefix(path, "scenarios/demo"):
			t.Errorf("the scenario itself is in the restore point, not a shared root: %s", path)
		case path == "internal/unused", path == "packages/proto", path == "packages/proto/gen":
			t.Errorf("a path outside the build closure was frozen: %s", path)
		case strings.HasPrefix(path, "internal/tuning/"), strings.HasPrefix(path, "packages/proto/gen/typescript/"):
			t.Errorf("nested root not compacted: %s", path)
		}
	}
}

// A file: dependency whose declared build outputs are missing cannot be
// frozen: the serving build would read a package nothing rebuilds.
func TestSharedSourceRootsRequiresBuiltOutputs(t *testing.T) {
	repo, item := basShapedRepo(t)
	delete(item.Manifest.Components, "api")
	if err := os.Rename(filepath.Join(repo, "packages/widgets/dist"), filepath.Join(repo, "packages/widgets/unbuilt")); err != nil {
		t.Fatal(err)
	}
	if _, err := SharedSourceRoots(context.Background(), repo, item); err == nil || !strings.Contains(err.Error(), "dist") {
		t.Fatalf("missing outputs: %v", err)
	}
}

// An unresolvable Go closure fails instead of freezing a partial set.
func TestSharedSourceRootsFailsOnBrokenGoClosure(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain unavailable")
	}
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOPROXY", "off")
	repo, item := basShapedRepo(t)
	testkitgo.WriteFile(t, filepath.Join(repo, "scenarios/demo/api/main.go"), "package main\n\nimport \"example.com/missing/pkg\"\n\nfunc main() { pkg.X() }\n")
	if _, err := SharedSourceRoots(context.Background(), repo, item); err == nil || !strings.Contains(err.Error(), "go list") {
		t.Fatalf("broken closure: %v", err)
	}
}

// Declared build outputs and data directories are excluded from a restore
// point; source is not.
func TestCaptureExcludePatternsCoverOutputsAndDataDirs(t *testing.T) {
	item := scenario.Scenario{Slug: "demo", Path: "/repo/scenarios/demo", Manifest: scenario.ServiceManifest{Components: map[string]scenario.Component{
		"api": {Build: scenario.ComponentBuild{Kind: "go_module", Dir: "api"}, Run: scenario.ComponentRun{DataDirs: []string{"data/executions"}, LogDir: "logs"}},
		"ui":  {Build: scenario.ComponentBuild{Kind: "pnpm_vite", Dir: "ui"}},
	}}}
	got := strings.Join(CaptureExcludePatterns(item), ",")
	for _, want := range []string{"/api/demo-api", "/ui/dist/", "/data/executions/", "/logs/"} {
		if !strings.Contains(got, want) {
			t.Errorf("patterns %q lack %q", got, want)
		}
	}
}
