package baselinefloor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// servingTreeFixture builds a repo with a shared Go package, a candidate
// working-tree copy of the target scenario, a sibling scenario, and a frozen
// restore point of the target captured outside the repo (as the floor does,
// under the cache). The target's go.mod uses the relative replace directives
// real scenarios use.
func servingTreeFixture(t *testing.T) (repo, restorePoint, base string) {
	t.Helper()
	repo = t.TempDir()
	writeFile(t, repo, "go.mod", "module example.com/root\n\ngo 1.22\n", 0o644)
	writeFile(t, repo, ".git/HEAD", "ref: refs/heads/main\n", 0o644)
	writeFile(t, repo, "packages/shared/go.mod", "module example.com/shared\n\ngo 1.22\n", 0o644)
	writeFile(t, repo, "packages/shared/shared.go", "package shared\n\nfunc Value() string { return \"shared\" }\n", 0o644)
	writeFile(t, repo, "packages/other/README.md", "other\n", 0o644)
	writeFile(t, repo, "internal/a/a.go", "package a\n", 0o644)
	writeFile(t, repo, "internal/b/b.go", "package b\n", 0o644)
	writeFile(t, repo, "scenarios/sibling/runtime/go.mod", "module example.com/sibling\n\ngo 1.22\n", 0o644)
	writeFile(t, repo, "scenarios/demo/api/main.go", "package main\n\nfunc main() { panic(\"candidate\") }\n", 0o644)
	writeFile(t, repo, "scenarios/demo/.vrooli/service.json", `{"service":{"name":"demo"}}`, 0o644)

	cache := t.TempDir()
	restorePoint = filepath.Join(cache, "demo", "baseline-goal", "restore-point")
	writeFile(t, restorePoint, "api/go.mod", strings.Join([]string{
		"module example.com/demo",
		"",
		"go 1.22",
		"",
		"require example.com/shared v0.0.0",
		"",
		"replace example.com/shared => ../../../packages/shared",
		"",
	}, "\n"), 0o644)
	writeFile(t, restorePoint, "api/main.go", "package main\n\nimport \"example.com/shared\"\n\nfunc main() { println(\"frozen\", shared.Value()) }\n", 0o644)
	base = filepath.Join(cache, "demo", "baseline-goal", "serving")
	return repo, restorePoint, base
}

func goBuild(t *testing.T, dir string) {
	t.Helper()
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain unavailable")
	}
	cmd := exec.Command(goBin, "build", "-buildvcs=false", "-o", os.DevNull, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOPROXY=off", "GOTOOLCHAIN=local")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build in %s failed: %v\n%s", dir, err, out)
	}
}

// TestMaterializeServingResolvesRelativeSharedPackages is the regression
// for the engaged live build that failed with "replacement directory
// ../../../packages/api-core does not exist": the frozen copy must be built
// from a location whose relative references reach the repository.
func TestMaterializeServingResolvesRelativeSharedPackages(t *testing.T) {
	repo, restorePoint, base := servingTreeFixture(t)

	dir, err := MaterializeServing(base, ServingSpec{Scenario: "demo", RepoRoot: repo, RestorePoint: restorePoint, Generation: "gen-1"})
	if err != nil {
		t.Fatalf("MaterializeServing: %v", err)
	}
	tree := filepath.Join(base, ServingGenerationKey("gen-1"))
	if want := ServingScenarioDir(tree, "demo"); dir != want {
		t.Fatalf("scenario dir = %q, want %q", dir, want)
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("scenario dir must be a real directory: info=%v err=%v", info, err)
	}
	if got := readFile(t, filepath.Join(dir, "api", "main.go")); !strings.Contains(got, "frozen") {
		t.Fatalf("serving copy holds %q, want the frozen restore-point source", got)
	}
	apiDir := filepath.Join(dir, "api")
	for rel, wantReal := range map[string]string{
		"../../../packages/shared/go.mod": filepath.Join(repo, "packages/shared/go.mod"),
		"../../sibling/runtime/go.mod":    filepath.Join(repo, "scenarios/sibling/runtime/go.mod"),
		"../../../go.mod":                 filepath.Join(repo, "go.mod"),
	} {
		resolved, err := filepath.EvalSymlinks(filepath.Join(apiDir, rel))
		if err != nil {
			t.Fatalf("relative reference %s does not resolve: %v", rel, err)
		}
		wantResolved, _ := filepath.EvalSymlinks(wantReal)
		if resolved != wantResolved {
			t.Errorf("relative reference %s resolves to %q, want %q", rel, resolved, wantResolved)
		}
	}
	if _, err := os.Lstat(filepath.Join(tree, ".git")); !os.IsNotExist(err) {
		t.Errorf(".git must not be linked into the serving tree (err=%v)", err)
	}
	writeFile(t, apiDir, "built-binary", "artifact", 0o755)
	if _, err := os.Stat(filepath.Join(restorePoint, "api", "built-binary")); !os.IsNotExist(err) {
		t.Errorf("serving build output leaked into the restore point (err=%v)", err)
	}
	goBuild(t, apiDir)
}

// TestMaterializeServingOverlaysFrozenRoots pins the overlay: a frozen root
// is a real copy from the snapshot, each of its ancestors is a real
// directory, and its siblings stay repository links. The frozen build keeps
// compiling after the working tree breaks the shared package (O9).
func TestMaterializeServingOverlaysFrozenRoots(t *testing.T) {
	repo, restorePoint, base := servingTreeFixture(t)
	snapshot := filepath.Join(filepath.Dir(base), "shared-sources")
	roots := []SharedRoot{
		{Path: "packages/shared", Reason: "go:api"},
		{Path: "internal/a", Reason: "go:api"},
		{Path: "go.mod", Reason: "go:module ."},
	}
	if _, _, err := CaptureSharedSources(repo, snapshot, roots, false); err != nil {
		t.Fatalf("CaptureSharedSources: %v", err)
	}
	// The candidate breaks the shared package after the capture.
	writeFile(t, repo, "packages/shared/shared.go", "package shared\n\nfunc Renamed() string { return \"moved\" }\n", 0o644)

	dir, err := MaterializeServing(base, ServingSpec{
		Scenario: "demo", RepoRoot: repo, RestorePoint: restorePoint, Generation: "gen-1",
		SharedSources: snapshot, Frozen: []string{"packages/shared", "internal/a", "go.mod"},
	})
	if err != nil {
		t.Fatalf("MaterializeServing: %v", err)
	}
	tree := filepath.Join(base, ServingGenerationKey("gen-1"))
	isLink := func(rel string) bool {
		info, err := os.Lstat(filepath.Join(tree, rel))
		if err != nil {
			t.Fatalf("lstat %s: %v", rel, err)
		}
		return info.Mode()&os.ModeSymlink != 0
	}
	for rel, wantLink := range map[string]bool{
		"packages":        false,
		"packages/shared": false,
		"packages/other":  true,
		"internal":        false,
		"internal/a":      false,
		"internal/b":      true,
		"go.mod":          false,
		"scenarios":       false,
		"scenarios/demo":  false,
	} {
		if got := isLink(rel); got != wantLink {
			t.Errorf("%s symlink = %t, want %t", rel, got, wantLink)
		}
	}
	goBuild(t, filepath.Join(dir, "api"))
}

// Each capture generation gets its own tree; an existing tree is never
// rewritten, because a running instance keeps the absolute paths it
// resolved at startup.
func TestMaterializeServingSeparatesGenerations(t *testing.T) {
	repo, restorePoint, base := servingTreeFixture(t)
	first, err := MaterializeServing(base, ServingSpec{Scenario: "demo", RepoRoot: repo, RestorePoint: restorePoint, Generation: "gen-1"})
	if err != nil {
		t.Fatalf("first materialize: %v", err)
	}
	writeFile(t, first, "api/node_modules/marker", "installed", 0o644)
	if again, err := MaterializeServing(base, ServingSpec{Scenario: "demo", RepoRoot: repo, RestorePoint: restorePoint, Generation: "gen-1"}); err != nil || again != first {
		t.Fatalf("same generation: %q, %v", again, err)
	}
	if got := readFile(t, filepath.Join(first, "api/node_modules/marker")); got != "installed" {
		t.Fatalf("same-generation materialize discarded build state")
	}

	writeFile(t, restorePoint, "api/main.go", "package main\n\nfunc main() { println(\"recaptured\") }\n", 0o644)
	second, err := MaterializeServing(base, ServingSpec{Scenario: "demo", RepoRoot: repo, RestorePoint: restorePoint, Generation: "gen-2"})
	if err != nil {
		t.Fatalf("new-generation materialize: %v", err)
	}
	if second == first {
		t.Fatal("a new generation must materialize a new tree")
	}
	if got := readFile(t, filepath.Join(second, "api/main.go")); !strings.Contains(got, "recaptured") {
		t.Fatalf("new generation holds stale source %q", got)
	}
	if got := readFile(t, filepath.Join(first, "api/main.go")); !strings.Contains(got, "frozen") {
		t.Fatalf("the running generation's tree was rewritten: %q", got)
	}
}

// A restore point captured before the scenario's .vrooli configuration was
// kept gets it from the working tree, without runtime state.
func TestMaterializeServingFillsMissingConfigFromWorkingTree(t *testing.T) {
	repo, restorePoint, base := servingTreeFixture(t)
	writeFile(t, repo, "scenarios/demo/.vrooli/secrets.json", `{"token":"x"}`, 0o600)
	dir, err := MaterializeServing(base, ServingSpec{
		Scenario: "demo", RepoRoot: repo, RestorePoint: restorePoint, Generation: "gen-1",
		ConfigFallback: filepath.Join(repo, "scenarios", "demo"),
	})
	if err != nil {
		t.Fatalf("MaterializeServing: %v", err)
	}
	if got := readFile(t, filepath.Join(dir, ".vrooli", "service.json")); !strings.Contains(got, "demo") {
		t.Fatalf("service.json = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, ".vrooli", "secrets.json")); !os.IsNotExist(err) {
		t.Fatalf("secrets copied into the serving tree (err=%v)", err)
	}
}

func TestReapServingTreesKeepsInUseAndCurrent(t *testing.T) {
	store := NewStore(t.TempDir())
	base := store.ServingBasePath("demo", "goal")
	for _, name := range []string{"g-current", "g-running", "g-old"} {
		writeFile(t, filepath.Join(base, name), "scenarios/demo/marker", name, 0o644)
	}
	writeFile(t, store.ServingTreePath("demo", "goal"), "scenarios/demo/marker", "legacy", 0o644)
	reaped, err := store.ReapServingTrees("demo", "goal", "g-current", func(tree string) bool {
		return filepath.Base(tree) == "g-running"
	})
	if err != nil {
		t.Fatalf("ReapServingTrees: %v", err)
	}
	if len(reaped) != 2 {
		t.Fatalf("reaped %v, want g-old and the legacy tree", reaped)
	}
	for name, want := range map[string]bool{"g-current": true, "g-running": true, "g-old": false} {
		_, err := os.Stat(filepath.Join(base, name))
		if (err == nil) != want {
			t.Errorf("%s exists=%t, want %t", name, err == nil, want)
		}
	}
}

func TestMaterializeServingRejectsInvalidInput(t *testing.T) {
	repo, restorePoint, base := servingTreeFixture(t)
	for _, name := range []string{"", "..", "a/b"} {
		if _, err := MaterializeServing(base, ServingSpec{Scenario: name, RepoRoot: repo, RestorePoint: restorePoint, Generation: "g"}); err == nil {
			t.Errorf("scenario %q: expected error", name)
		}
	}
	for _, frozen := range []string{"scenarios/demo/api", "../outside", ".git"} {
		if _, err := MaterializeServing(base, ServingSpec{Scenario: "demo", RepoRoot: repo, RestorePoint: restorePoint, Generation: "g", SharedSources: t.TempDir(), Frozen: []string{frozen}}); err == nil {
			t.Errorf("frozen root %q: expected error", frozen)
		}
	}
}
