package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/baselinefloor"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

// TestEffectiveSourceDirRealFloorIsolatesContent is the deterministic encoding of
// the Baseline Modes shadow-mode live spike (plan §9/§10). Where TestEffectiveSourceDir
// asserts the decision returns the right path STRING against a fake resolver, this
// drives the real floor end-to-end: it captures a real restore point with
// baselinefloor.Capture, edits the working tree out from under it, and reads the
// BYTES at whichever directory effectiveSourceDir resolves to. It proves the spike's
// core guarantee as a permanent test — while a shadow engagement is open, the live
// (serving) instance is pinned to the frozen OLD copy even after the working tree
// changes to NEW; the @shadow instance follows the NEW working tree; and once the
// engagement closes (promote/abandon), live follows the working tree again.
//
// It lives in package lifecycle (not lifecycle_test) so it can call the unexported
// effectiveSourceDir; baselinefloor does not import lifecycle, so there is no cycle.
// The floor→fact mapping the production resolver performs is covered separately by
// cmd/vrooli/floor_engagement_test.go; here the EngagementInfo is fed from a
// REAL captured restore-point path so the decision routes to real frozen content,
// not just a string.
func TestEffectiveSourceDirRealFloorIsolatesContent(t *testing.T) {
	const marker = "marker.txt"

	// 1. A working tree holding OLD code.
	workingTree := t.TempDir()
	writeFile(t, filepath.Join(workingTree, marker), "OLD")

	// 2. Capture a real restore point (the "baseline start" capture-before-merge step).
	store := baselinefloor.NewStore(t.TempDir())
	restorePoint := store.RestorePointPath("demo", "spike")
	if _, err := baselinefloor.Capture(workingTree, restorePoint, nil); err != nil {
		t.Fatalf("capture restore point: %v", err)
	}

	// 3. The candidate merges into the working tree: OLD -> NEW. The frozen copy
	//    must NOT see this.
	writeFile(t, filepath.Join(workingTree, marker), "NEW")

	item := func(variant string) scenario.Scenario {
		return scenario.Scenario{Slug: "demo", Path: workingTree, Variant: variant}
	}

	// 4. While the shadow engagement is open, the resolver reports the split with
	//    the real captured copy path.
	engaged := &fakeEngagementResolver{
		engaged: true,
		info: EngagementInfo{
			RestorePointDir: restorePoint,
			Slug:            "spike",
			Mode:            "shadow",
		},
	}
	r := &Runner{Engagements: engaged}

	// Live (empty variant) must resolve to the frozen copy and read OLD, even
	// though the working tree now holds NEW — this is the isolation guarantee.
	liveDir, err := r.effectiveSourceDir(item(""))
	if err != nil {
		t.Fatalf("live effectiveSourceDir while engaged: %v", err)
	}
	if got := readFile(t, filepath.Join(liveDir, marker)); got != "OLD" {
		t.Errorf("live serves %q from %q, want OLD (frozen copy)", got, liveDir)
	}

	// Shadow (@shadow) must resolve to the working tree and read NEW — the
	// candidate runs from the working tree the agent edited.
	shadowDir, err := r.effectiveSourceDir(item("shadow"))
	if err != nil {
		t.Fatalf("shadow effectiveSourceDir while engaged: %v", err)
	}
	if got := readFile(t, filepath.Join(shadowDir, marker)); got != "NEW" {
		t.Errorf("shadow serves %q from %q, want NEW (working tree)", got, shadowDir)
	}

	// 5. Engagement closes (promote/abandon both end the split). Live now follows
	//    the working tree again — read NEW.
	closed := &fakeEngagementResolver{engaged: false}
	r = &Runner{Engagements: closed}
	postDir, err := r.effectiveSourceDir(item(""))
	if err != nil {
		t.Fatalf("live effectiveSourceDir after close: %v", err)
	}
	if got := readFile(t, filepath.Join(postDir, marker)); got != "NEW" {
		t.Errorf("after engagement close live serves %q from %q, want NEW (working tree)", got, postDir)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	return string(data)
}

// TestEngagedLiveBuildsAndRunsFromFrozenCopy proves the layout decision reaches
// the build and the process, not only the registry's WorkingDir. While a shadow
// engagement is open, live's setup must write its artifact into the frozen copy
// and leave the working tree (which holds the candidate) untouched.
func TestEngagedLiveBuildsAndRunsFromFrozenCopy(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")
	workingTree := filepath.Join(root, "scenarios", "alpha")

	store := baselinefloor.NewStore(t.TempDir())
	restorePoint := store.RestorePointPath("alpha", "spike")
	if _, err := baselinefloor.Capture(workingTree, restorePoint, nil); err != nil {
		t.Fatalf("capture restore point: %v", err)
	}

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	runner.Engagements = &fakeEngagementResolver{
		engaged: true,
		info:    EngagementInfo{RestorePointDir: restorePoint, Slug: "spike", Mode: "shadow"},
	}

	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("Start(alpha live) while engaged: %v", err)
	}
	cleanupRunner(t, runner, "alpha", StopOptions{})

	frozenArtifact := filepath.Join(restorePoint, "api", "mock-api")
	if _, err := os.Stat(frozenArtifact); err != nil {
		t.Fatalf("engaged live did not build into the frozen copy: %v", err)
	}
	workingArtifact := filepath.Join(workingTree, "api", "mock-api")
	if _, err := os.Stat(workingArtifact); err == nil {
		t.Fatal("engaged live wrote its build artifact into the working tree the candidate owns")
	}
}

// frozenFloorResolver is the production resolver's shape (cmd/vrooli
// floorEngagementResolver) over a test store: a shadow engagement whose
// capture froze the shared sources, served from a generation tree.
type frozenFloorResolver struct {
	store    *baselinefloor.Store
	slug     string
	frozen   []string
	captured string
}

func (r *frozenFloorResolver) Engagement(scenarioName string) (EngagementInfo, bool, error) {
	return EngagementInfo{
		RestorePointDir:     r.store.RestorePointPath(scenarioName, r.slug),
		Mode:                "shadow",
		Slug:                r.slug,
		Generation:          r.captured,
		CaptureID:           r.captured,
		SharedSourcesFrozen: true,
	}, true, nil
}

func (r *frozenFloorResolver) MaterializeServingSource(scenarioName string, info EngagementInfo, repoRoot string) (string, error) {
	return baselinefloor.MaterializeServing(r.store.ServingBasePath(scenarioName, info.Slug), baselinefloor.ServingSpec{
		Scenario:      scenarioName,
		RepoRoot:      repoRoot,
		RestorePoint:  info.RestorePointDir,
		Generation:    info.Generation,
		SharedSources: r.store.SharedSourcesPath(scenarioName, info.Slug),
		Frozen:        r.frozen,
	})
}

// TestEngagedLiveBuildsFromFrozenSharedSources is the DL-7 acceptance at the
// lifecycle level (O9):
//
//	Given an engaged shadow and a change to a shared package in the working tree
//	When live restarts during the engagement
//	Then live builds from its frozen shared packages and reports healthy
func TestEngagedLiveBuildsFromFrozenSharedSources(t *testing.T) {
	if runtime.GOOS != "linux" {
		testkitgo.SkipPlatform(t, "lifecycle process management currently targets linux")
	}
	root, home := t.TempDir(), t.TempDir()
	writeLifecycleFixture(t, root, "alpha")
	workingTree := filepath.Join(root, "scenarios", "alpha")
	testkitgo.WriteFile(t, filepath.Join(root, "packages", "shared", "go.mod"), "module example.com/shared\n\ngo 1.22\n")
	testkitgo.WriteFile(t, filepath.Join(root, "packages", "shared", "shared.go"), "package shared\n\nfunc Status() string { return \"healthy\" }\n")
	testkitgo.WriteFile(t, filepath.Join(workingTree, "api", "go.mod"), "module fixture\n\ngo 1.22\n\nrequire example.com/shared v0.0.0\n\nreplace example.com/shared => ../../../packages/shared\n")
	mainGo := readFile(t, filepath.Join(workingTree, "api", "main.go"))
	mainGo = strings.Replace(mainGo, "import (\n", "import (\n \"example.com/shared\"\n", 1)
	mainGo = strings.Replace(mainGo, `{\"status\":\"healthy\"`, `{\"status\":\"`+"\"+shared.Status()+\""+`\"`, 1)
	writeFile(t, filepath.Join(workingTree, "api", "main.go"), mainGo)

	// Capture: restore point plus the frozen shared sources.
	item, err := scenario.Load(root, "alpha", scenario.SandboxEnv{})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := SharedSourceRoots(context.Background(), root, item)
	if err != nil {
		t.Fatalf("SharedSourceRoots: %v", err)
	}
	store := baselinefloor.NewStore(t.TempDir())
	if _, err := baselinefloor.Capture(workingTree, store.RestorePointPath("alpha", "goal"), nil); err != nil {
		t.Fatalf("capture: %v", err)
	}
	floorRoots := make([]baselinefloor.SharedRoot, 0, len(roots))
	frozen := make([]string, 0, len(roots))
	for _, root := range roots {
		floorRoots = append(floorRoots, baselinefloor.SharedRoot{Path: root.Path, Reason: root.Reason})
		frozen = append(frozen, root.Path)
	}
	if !slices.Contains(frozen, "packages/shared") {
		t.Fatalf("shared package not in the freeze: %v", frozen)
	}
	if _, _, err := baselinefloor.CaptureSharedSources(root, store.SharedSourcesPath("alpha", "goal"), floorRoots, false); err != nil {
		t.Fatalf("freeze: %v", err)
	}

	// The candidate breaks the shared package in the working tree.
	testkitgo.WriteFile(t, filepath.Join(root, "packages", "shared", "shared.go"), "package shared\n\nfunc Renamed() string { return \"moved\" }\n")

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	runner.Engagements = &frozenFloorResolver{store: store, slug: "goal", frozen: frozen, captured: "capture-1"}
	result, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("engaged live start against a broken working-tree shared package: %v", err)
	}
	cleanupRunner(t, runner, "alpha", StopOptions{})
	if result.Health != "healthy" {
		t.Fatalf("health = %q", result.Health)
	}
	if _, err := runner.Restart("alpha", StartOptions{ForceSetup: true}); err != nil {
		t.Fatalf("forced rebuild of engaged live: %v", err)
	}
	logPath, err := process.ScenarioLifecycleLogPath(home, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if log := readFile(t, logPath); !strings.Contains(log, "serving-tree event=frozen-shared") || !strings.Contains(log, `capture_id="capture-1"`) {
		t.Fatalf("frozen build did not skip repository regeneration or name its capture:\n%s", log)
	}
}
