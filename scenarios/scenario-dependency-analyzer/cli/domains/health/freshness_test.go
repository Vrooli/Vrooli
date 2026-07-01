package health

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestImpactedSurfacesFansOutThroughTransitiveInRepoModules(t *testing.T) {
	root := filepath.Clean("/repo")
	modules := map[string]goModule{
		"github.com/vrooli/root": {
			dir:      filepath.Join(root),
			module:   "github.com/vrooli/root",
			requires: map[string]struct{}{},
		},
		"github.com/vrooli/shared": {
			dir:      filepath.Join(root, "packages", "shared"),
			module:   "github.com/vrooli/shared",
			requires: map[string]struct{}{},
		},
		"github.com/vrooli/mid": {
			dir:      filepath.Join(root, "packages", "mid"),
			module:   "github.com/vrooli/mid",
			requires: map[string]struct{}{"github.com/vrooli/shared": {}},
		},
		"github.com/vrooli/consumer": {
			dir:      filepath.Join(root, "scenarios", "consumer", "api"),
			module:   "github.com/vrooli/consumer",
			requires: map[string]struct{}{"github.com/vrooli/mid": {}},
		},
		"github.com/vrooli/unrelated": {
			dir:      filepath.Join(root, "scenarios", "unrelated", "api"),
			module:   "github.com/vrooli/unrelated",
			requires: map[string]struct{}{},
		},
	}
	surfaces := []goSurface{
		{
			scenario: "consumer",
			surface:  "api",
			goMod:    filepath.Join(root, "scenarios", "consumer", "api", "go.mod"),
			module:   "github.com/vrooli/consumer",
		},
		{
			scenario: "unrelated",
			surface:  "api",
			goMod:    filepath.Join(root, "scenarios", "unrelated", "api", "go.mod"),
			module:   "github.com/vrooli/unrelated",
		},
	}

	got := impactedSurfaces(root, surfaces, modules, []string{"packages/shared/shared.go"})

	consumerGoMod := filepath.Join(root, "scenarios", "consumer", "api", "go.mod")
	if !reflect.DeepEqual(got[consumerGoMod], []string{"packages/shared/shared.go"}) {
		t.Fatalf("consumer impacted paths = %#v, want shared package change", got[consumerGoMod])
	}
	unrelatedGoMod := filepath.Join(root, "scenarios", "unrelated", "api", "go.mod")
	if _, ok := got[unrelatedGoMod]; ok {
		t.Fatalf("unrelated surface was impacted: %#v", got[unrelatedGoMod])
	}
}

func TestImpactedSurfacesRootModuleChangesFanOutToAllSurfaces(t *testing.T) {
	root := filepath.Clean("/repo")
	surfaces := []goSurface{
		{scenario: "api", surface: "api", goMod: filepath.Join(root, "scenarios", "api", "api", "go.mod"), module: "github.com/vrooli/api"},
		{scenario: "cli", surface: "cli", goMod: filepath.Join(root, "scenarios", "cli", "cli", "go.mod"), module: "github.com/vrooli/cli"},
	}

	got := impactedSurfaces(root, surfaces, nil, []string{"go.sum"})

	for _, surface := range surfaces {
		if !reflect.DeepEqual(got[surface.goMod], []string{"go.mod", "go.sum"}) {
			t.Fatalf("%s impacted paths = %#v, want root go.mod/go.sum fanout", surface.goMod, got[surface.goMod])
		}
	}
}

func TestImpactedSurfacesIncludesTouchedSurfaceModuleMetadata(t *testing.T) {
	root := filepath.Clean("/repo")
	modules := map[string]goModule{
		"github.com/vrooli/demo-api": {
			dir:      filepath.Join(root, "scenarios", "demo", "api"),
			module:   "github.com/vrooli/demo-api",
			requires: map[string]struct{}{},
		},
		"github.com/vrooli/demo-cli": {
			dir:      filepath.Join(root, "scenarios", "demo", "cli"),
			module:   "github.com/vrooli/demo-cli",
			requires: map[string]struct{}{},
		},
	}
	surfaces := []goSurface{
		{
			scenario: "demo",
			surface:  "api",
			goMod:    filepath.Join(root, "scenarios", "demo", "api", "go.mod"),
			module:   "github.com/vrooli/demo-api",
		},
		{
			scenario: "demo",
			surface:  "cli",
			goMod:    filepath.Join(root, "scenarios", "demo", "cli", "go.mod"),
			module:   "github.com/vrooli/demo-cli",
		},
	}

	got := impactedSurfaces(root, surfaces, modules, []string{
		"scenarios/demo/api/go.mod",
		"scenarios/demo/cli/main.go",
	})

	apiGoMod := filepath.Join(root, "scenarios", "demo", "api", "go.mod")
	if !reflect.DeepEqual(got[apiGoMod], []string{"scenarios/demo/api/go.mod"}) {
		t.Fatalf("api impacted paths = %#v, want own go.mod change", got[apiGoMod])
	}
	cliGoMod := filepath.Join(root, "scenarios", "demo", "cli", "go.mod")
	if _, ok := got[cliGoMod]; ok {
		t.Fatalf("cli source-only change should not trigger dependency freshness: %#v", got[cliGoMod])
	}
}

func TestDiscoverGoSurfacesIncludesNonAPIModules(t *testing.T) {
	root := t.TempDir()
	writeFreshnessFile(t, filepath.Join(root, "scenarios", "demo", "api", "go.mod"), "module example.com/demo-api\n")
	writeFreshnessFile(t, filepath.Join(root, "scenarios", "demo", "cli", "go.mod"), "module example.com/demo-cli\n")
	writeFreshnessFile(t, filepath.Join(root, "scenarios", "demo", "ui", "package.json"), "{}\n")

	surfaces, err := discoverGoSurfaces(root)
	if err != nil {
		t.Fatalf("discoverGoSurfaces: %v", err)
	}
	got := map[string]string{}
	for _, surface := range surfaces {
		got[surface.surface] = surface.module
	}
	if !reflect.DeepEqual(got, map[string]string{
		"api": "example.com/demo-api",
		"cli": "example.com/demo-cli",
	}) {
		t.Fatalf("surfaces = %#v, want api and cli Go modules", got)
	}
}

func TestCheckGoFreshnessApplyReportsCleanAfterVerifiedTidy(t *testing.T) {
	root := filepath.Clean("/repo")
	surface := goSurface{
		scenario: "demo",
		surface:  "api",
		goMod:    filepath.Join(root, "scenarios", "demo", "api", "go.mod"),
	}
	calls := stubRunGo(t, []stubGoCall{
		{args: []string{"mod", "tidy", "-diff"}, diff: "diff --git a/go.mod b/go.mod\n"},
		{args: []string{"mod", "tidy"}, clean: true},
		{args: []string{"mod", "tidy", "-diff"}, clean: true},
	})

	got := checkGoFreshness(context.Background(), root, surface, nil, freshnessRequest{apply: true})

	if got.Status != "clean" {
		t.Fatalf("status = %q, want clean after verified apply; report=%+v", got.Status, got)
	}
	if len(got.DiffPaths) != 0 {
		t.Fatalf("diff paths = %#v, want cleared after verified apply", got.DiffPaths)
	}
	if *calls != 3 {
		t.Fatalf("go calls = %d, want detect/apply/verify", *calls)
	}
}

func TestCheckGoFreshnessApplyKeepsStaleWhenPostApplyVerificationStillDiffs(t *testing.T) {
	root := filepath.Clean("/repo")
	surface := goSurface{
		scenario: "demo",
		surface:  "api",
		goMod:    filepath.Join(root, "scenarios", "demo", "api", "go.mod"),
	}
	stubRunGo(t, []stubGoCall{
		{args: []string{"mod", "tidy", "-diff"}, diff: "diff --git a/go.sum b/go.sum\n"},
		{args: []string{"mod", "tidy"}, clean: true},
		{args: []string{"mod", "tidy", "-diff"}, diff: "diff --git a/go.mod b/go.mod\n"},
	})

	got := checkGoFreshness(context.Background(), root, surface, nil, freshnessRequest{apply: true})

	if got.Status != "stale" {
		t.Fatalf("status = %q, want stale when post-apply verification still diffs; report=%+v", got.Status, got)
	}
	if !reflect.DeepEqual(got.DiffPaths, []string{"scenarios/demo/api/go.mod"}) {
		t.Fatalf("diff paths = %#v, want verified post-apply diff paths", got.DiffPaths)
	}
}

func TestReportTouchedPathsKeepsOnlyDependencyRelevantChanges(t *testing.T) {
	got := reportTouchedPaths(map[string][]string{
		"/repo/scenarios/demo/api/go.mod": {
			"packages/shared/shared.go",
			"packages/shared/shared.go",
			"scenarios/demo/api/go.mod",
		},
		"/repo/scenarios/other/api/go.mod": {
			"packages/other/other.go",
		},
	})

	want := []string{
		"packages/other/other.go",
		"packages/shared/shared.go",
		"scenarios/demo/api/go.mod",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("report touched paths = %#v, want compact dependency-relevant paths", got)
	}
}

func TestLimitStringsRetainsCountSentinel(t *testing.T) {
	got := limitStrings([]string{"a", "b", "c"}, 2)
	want := []string{"a", "b", "... 1 more"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("limited strings = %#v, want sentinel", got)
	}
}

func TestCheckGoSurfacesRunsConcurrentlyAndPreservesOrder(t *testing.T) {
	root := filepath.Clean("/repo")
	surfaces := []goSurface{
		{scenario: "one", surface: "api", goMod: filepath.Join(root, "scenarios", "one", "api", "go.mod")},
		{scenario: "two", surface: "api", goMod: filepath.Join(root, "scenarios", "two", "api", "go.mod")},
		{scenario: "three", surface: "api", goMod: filepath.Join(root, "scenarios", "three", "api", "go.mod")},
	}

	var mu sync.Mutex
	active := 0
	maxActive := 0
	previous := runGoCommand
	runGoCommand = func(_ context.Context, _ string, _ ...string) (string, bool, error) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		active--
		mu.Unlock()
		return "", true, nil
	}
	t.Cleanup(func() { runGoCommand = previous })

	got := checkGoSurfaces(context.Background(), root, surfaces, nil, freshnessRequest{concurrency: 2})

	if len(got) != len(surfaces) {
		t.Fatalf("reports = %d, want %d", len(got), len(surfaces))
	}
	if got[0].Scenario != "one" || got[1].Scenario != "two" || got[2].Scenario != "three" {
		t.Fatalf("reports not in surface order: %#v", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if maxActive != 2 {
		t.Fatalf("max active checks = %d, want concurrency limit 2", maxActive)
	}
}

func TestCheckFreshnessAddsReconcileActionForErroredSurfaces(t *testing.T) {
	root := t.TempDir()
	writeFreshnessFile(t, filepath.Join(root, "scenarios", "demo", "api", "go.mod"), "module example.com/demo\n")
	stubRunGo(t, []stubGoCall{
		{args: []string{"mod", "tidy", "-diff"}, err: errors.New("reading github.com/vrooli/cli-core/go.mod at revision v0.0.0: ERROR: Repository not found.")},
	})

	got, err := checkFreshness(context.Background(), root, freshnessRequest{concurrency: 1})
	if err != nil {
		t.Fatalf("checkFreshness: %v", err)
	}

	if got.Summary.Errors != 1 {
		t.Fatalf("errors = %d, want 1; report=%+v", got.Summary.Errors, got)
	}
	if len(got.NextActions) != 1 {
		t.Fatalf("next actions = %#v, want reconcile action", got.NextActions)
	}
	if got.NextActions[0].Command != "scenario-dependency-analyzer deps reconcile --all --json" {
		t.Fatalf("next action command = %q, want deps reconcile preview", got.NextActions[0].Command)
	}
	if got.NextActions[0].Fixability != "guided" {
		t.Fatalf("next action fixability = %q, want guided", got.NextActions[0].Fixability)
	}
}

func TestCheckFreshnessDoesNotSuggestReconcileForUnknownErrors(t *testing.T) {
	root := t.TempDir()
	writeFreshnessFile(t, filepath.Join(root, "scenarios", "demo", "api", "go.mod"), "module example.com/demo\n")
	stubRunGo(t, []stubGoCall{
		{args: []string{"mod", "tidy", "-diff"}, err: errors.New("permission denied")},
	})

	got, err := checkFreshness(context.Background(), root, freshnessRequest{concurrency: 1})
	if err != nil {
		t.Fatalf("checkFreshness: %v", err)
	}
	if len(got.NextActions) != 0 {
		t.Fatalf("next actions = %#v, want no local-replace reconcile action for unknown errors", got.NextActions)
	}
}

func TestFreshnessRetrievalHintsIncludesEveryAction(t *testing.T) {
	report := freshnessReport{
		Mode: "touched",
		NextActions: []freshnessAction{
			{Command: "scenario-dependency-analyzer freshness --touched --apply"},
			{Command: "scenario-dependency-analyzer deps reconcile --all --json"},
		},
	}

	got := freshnessRetrievalHints(report)
	want := []string{
		"scenario-dependency-analyzer freshness --touched --apply",
		"scenario-dependency-analyzer deps reconcile --all --json",
		"scenario-dependency-analyzer freshness --touched --json",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("retrieval hints = %#v, want %#v", got, want)
	}
}

func writeFreshnessFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

type stubGoCall struct {
	args  []string
	diff  string
	clean bool
	err   error
}

func stubRunGo(t *testing.T, calls []stubGoCall) *int {
	t.Helper()
	count := 0
	previous := runGoCommand
	runGoCommand = func(_ context.Context, _ string, args ...string) (string, bool, error) {
		if count >= len(calls) {
			t.Fatalf("unexpected go call %d with args %#v", count+1, args)
		}
		call := calls[count]
		count++
		if !reflect.DeepEqual(args, call.args) {
			t.Fatalf("go call %d args = %#v, want %#v", count, args, call.args)
		}
		return call.diff, call.clean, call.err
	}
	t.Cleanup(func() { runGoCommand = previous })
	return &count
}
