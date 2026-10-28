package health

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
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

	modules, err := discoverGoModules(root)
	if err != nil {
		t.Fatalf("discoverGoModules: %v", err)
	}
	surfaces, err := discoverGoSurfacesFromModules(root, modules)
	if err != nil {
		t.Fatalf("discoverGoSurfacesFromModules: %v", err)
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

func TestFilterGoSurfacesByModuleAcceptsRepositoryRelativeDirectory(t *testing.T) {
	root := filepath.Clean("/repo")
	surfaces := []goSurface{
		{goMod: filepath.Join(root, "packages", "delivery-ramp-go", "go.mod"), module: "github.com/vrooli/delivery-ramp-go"},
		{goMod: filepath.Join(root, "packages", "other", "go.mod"), module: "github.com/vrooli/other"},
	}

	got := filterGoSurfacesByModule(root, surfaces, "packages/delivery-ramp-go")
	if len(got) != 1 || got[0].module != "github.com/vrooli/delivery-ramp-go" {
		t.Fatalf("filtered surfaces = %#v, want delivery-ramp-go only", got)
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

// unsortedGoSum is the real shape of the defect: every hash the module needs is
// present, but two lines sit after the last entry instead of in sorted place.
// `go mod tidy` rewrites go.sum only when the hash SET changes, so it leaves
// this untouched forever while `tidy -diff` keeps reporting a diff.
const unsortedGoSum = `connectrpc.com/connect v1.19.2 h1:aaa=
connectrpc.com/connect v1.19.2/go.mod h1:bbb=
gopkg.in/yaml.v3 v3.0.1 h1:ccc=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:ddd=
github.com/creack/pty/v2 v2.0.1 h1:eee=
github.com/creack/pty/v2 v2.0.1/go.mod h1:fff=
`

func newGoSumSurface(t *testing.T, contents string) (string, goSurface) {
	t.Helper()
	root := t.TempDir()
	surfaceDir := filepath.Join(root, "scenarios", "demo", "api")
	if err := os.MkdirAll(surfaceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(surfaceDir, "go.sum"), []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return root, goSurface{
		scenario: "demo",
		surface:  "api",
		goMod:    filepath.Join(surfaceDir, "go.mod"),
	}
}

func TestCheckGoFreshnessApplyForcesGoSumRewriteWhenTidyCannotConverge(t *testing.T) {
	root, surface := newGoSumSurface(t, unsortedGoSum)
	sorted := strings.Join(sortedGoSumLines([]byte(unsortedGoSum)), "\n") + "\n"

	calls := stubRunGo(t, []stubGoCall{
		{args: []string{"mod", "tidy", "-diff"}, diff: "diff --git a/go.sum b/go.sum\n"},
		// The real defect: tidy succeeds and writes nothing.
		{args: []string{"mod", "tidy"}, clean: true},
		{args: []string{"mod", "tidy", "-diff"}, diff: "diff --git a/go.sum b/go.sum\n"},
		// After go.sum is removed, tidy regenerates it sorted.
		{args: []string{"mod", "tidy"}, clean: true, do: func(dir string) {
			if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte(sorted), 0o644); err != nil {
				t.Fatalf("regenerate go.sum: %v", err)
			}
		}},
		{args: []string{"mod", "tidy", "-diff"}, clean: true},
	})

	got := checkGoFreshness(context.Background(), root, surface, nil, freshnessRequest{apply: true})

	if got.Status != "clean" {
		t.Fatalf("status = %q, want clean after the forced go.sum rewrite; report=%+v", got.Status, got)
	}
	if len(got.DiffPaths) != 0 {
		t.Fatalf("diff paths = %#v, want cleared after the forced rewrite", got.DiffPaths)
	}
	if *calls != 5 {
		t.Fatalf("go calls = %d, want detect/apply/verify/rewrite/verify", *calls)
	}
	after, err := os.ReadFile(filepath.Join(filepath.Dir(surface.goMod), "go.sum"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(after) != sorted {
		t.Fatalf("go.sum = %q, want the sorted rewrite", string(after))
	}
}

func TestCheckGoFreshnessApplyKeepsStaleWhenRewriteStillDiffs(t *testing.T) {
	root, surface := newGoSumSurface(t, unsortedGoSum)

	stubRunGo(t, []stubGoCall{
		{args: []string{"mod", "tidy", "-diff"}, diff: "diff --git a/go.sum b/go.sum\n"},
		{args: []string{"mod", "tidy"}, clean: true},
		{args: []string{"mod", "tidy", "-diff"}, diff: "diff --git a/go.sum b/go.sum\n"},
		{args: []string{"mod", "tidy"}, clean: true, do: func(dir string) {
			if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte(unsortedGoSum), 0o644); err != nil {
				t.Fatalf("regenerate go.sum: %v", err)
			}
		}},
		{args: []string{"mod", "tidy", "-diff"}, diff: "diff --git a/go.mod b/go.mod\n"},
	})

	got := checkGoFreshness(context.Background(), root, surface, nil, freshnessRequest{apply: true})

	if got.Status != "stale" {
		t.Fatalf("status = %q, want stale when the rewrite still diffs; report=%+v", got.Status, got)
	}
	if !reflect.DeepEqual(got.DiffPaths, []string{"scenarios/demo/api/go.mod"}) {
		t.Fatalf("diff paths = %#v, want verified post-rewrite diff paths", got.DiffPaths)
	}
}

// The repair reorders; it must never change what the module depends on.
func TestForceGoSumRewriteRestoresOriginalWhenHashSetChanges(t *testing.T) {
	_, surface := newGoSumSurface(t, unsortedGoSum)
	surfaceDir := filepath.Dir(surface.goMod)

	stubRunGo(t, []stubGoCall{
		{args: []string{"mod", "tidy"}, clean: true, do: func(dir string) {
			dropped := "connectrpc.com/connect v1.19.2 h1:aaa=\n"
			if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte(dropped), 0o644); err != nil {
				t.Fatalf("regenerate go.sum: %v", err)
			}
		}},
	})

	err := forceGoSumRewrite(context.Background(), surfaceDir)
	if err == nil {
		t.Fatal("forceGoSumRewrite returned nil, want an error when the hash set changes")
	}
	after, readErr := os.ReadFile(filepath.Join(surfaceDir, "go.sum"))
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}
	if string(after) != unsortedGoSum {
		t.Fatalf("go.sum = %q, want the original restored byte-for-byte", string(after))
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

func TestCheckGoFreshnessColdModuleRemainsUnqualified(t *testing.T) {
	root := t.TempDir()
	surface := goSurface{scenario: "demo", surface: "api", goMod: filepath.Join(root, "scenarios/demo/api/go.mod")}
	writeFreshnessFile(t, surface.goMod, "module demo\ngo 1.25.0\n")
	previousOffline, previousNetwork := runGoCommand, runGoNetwork
	offlineCalls, networkCalls := 0, 0
	runGoCommand = func(context.Context, string, ...string) (string, bool, error) {
		offlineCalls++
		return "", false, errors.New("go: module lookup disabled by GOPROXY=off")
	}
	runGoNetwork = func(context.Context, string, ...string) (string, bool, error) { networkCalls++; return "", true, nil }
	t.Cleanup(func() { runGoCommand, runGoNetwork = previousOffline, previousNetwork })

	got := checkGoFreshness(context.Background(), root, surface, nil, freshnessRequest{noCache: true})
	if got.Status != "needs_download" || offlineCalls != 1 || networkCalls != 1 {
		t.Fatalf("report=%+v offline=%d network=%d", got, offlineCalls, networkCalls)
	}
	report, err := checkFreshness(context.Background(), root, freshnessRequest{noCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Clean || report.Summary.NeedsDownload != 1 {
		t.Fatalf("cold cache cannot qualify module freshness: %+v", report)
	}
}

func TestCheckGoFreshnessDoesNotRetryKnownInRepoModule(t *testing.T) {
	root := t.TempDir()
	writeFreshnessFile(t, filepath.Join(root, "packages/envkit-go/go.mod"), "module github.com/vrooli/envkit-go\ngo 1.25.0\n")
	surface := goSurface{scenario: "demo", surface: "api", goMod: filepath.Join(root, "scenarios/demo/api/go.mod")}
	writeFreshnessFile(t, surface.goMod, "module demo\ngo 1.25.0\n")
	previous := runGoCommand
	calls := 0
	runGoCommand = func(context.Context, string, ...string) (string, bool, error) {
		calls++
		return "", false, errors.New("go: module lookup disabled by GOPROXY=off github.com/vrooli/envkit-go")
	}
	t.Cleanup(func() { runGoCommand = previous })
	got := checkGoFreshness(context.Background(), root, surface, nil, freshnessRequest{noCache: true})
	if got.Status != "error" || calls != 1 {
		t.Fatalf("report=%+v calls=%d", got, calls)
	}
}

func TestFreshnessCacheInvalidatesChangedLocalDependencyClosure(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")
	writeLocalFreshnessFixture(t, root)
	initFreshnessGitRepo(t, root)

	initial := checkFreshnessModule(t, root, "scenarios/consumer/api")
	if !initial.Clean || reportSurface(t, initial, "scenarios/consumer/api").CacheHit {
		t.Fatalf("initial consumer report = %+v, want an uncached clean check", initial)
	}
	unchanged := checkFreshnessModule(t, root, "scenarios/consumer/api")
	if item := reportSurface(t, unchanged, "scenarios/consumer/api"); item.Status != "clean" || !item.CacheHit {
		t.Fatalf("unchanged consumer = %+v, want clean cache hit", item)
	}
	unrelated := checkFreshnessModule(t, root, "scenarios/unrelated/api")
	if !unrelated.Clean || reportSurface(t, unrelated, "scenarios/unrelated/api").CacheHit {
		t.Fatalf("initial unrelated report = %+v, want an uncached clean check", unrelated)
	}

	writeFreshnessFile(t, filepath.Join(root, "packages/shared/go.mod"), "module example.test/shared\n\ngo 1.23\n\nrequire example.test/dep v0.0.0\n\nreplace example.test/dep => ../dep\n")
	writeFreshnessFile(t, filepath.Join(root, "packages/shared/shared.go"), "package shared\n\nimport \"example.test/dep\"\n\nconst Value = dep.Value\n")

	touched, err := checkFreshness(context.Background(), root, freshnessRequest{touched: true, concurrency: 1})
	if err != nil {
		t.Fatalf("touched check: %v", err)
	}
	consumer := reportSurface(t, touched, "scenarios/consumer/api")
	if consumer.Status != "stale" || consumer.CacheHit {
		t.Fatalf("touched consumer = %+v, want uncached stale after upstream local module change", consumer)
	}
	if !strings.Contains(strings.Join(consumer.DiffPaths, "\n"), "scenarios/consumer/api/go.mod") {
		t.Fatalf("touched consumer diff paths = %#v, want consumer go.mod", consumer.DiffPaths)
	}
	if _, included := findSurface(touched.Surfaces, "scenarios/unrelated/api"); included {
		t.Fatalf("unrelated surface was included in touched report: %+v", touched.Surfaces)
	}

	exact := checkFreshnessModule(t, root, "scenarios/consumer/api")
	if item := reportSurface(t, exact, "scenarios/consumer/api"); item.Status != "stale" || !item.CacheHit {
		t.Fatalf("exact consumer = %+v, want cached stale result under changed closure key", item)
	}
	unrelatedAfter := checkFreshnessModule(t, root, "scenarios/unrelated/api")
	if item := reportSurface(t, unrelatedAfter, "scenarios/unrelated/api"); item.Status != "clean" || !item.CacheHit {
		t.Fatalf("unrelated consumer = %+v, want unaffected clean cache hit", item)
	}
}

func TestFreshnessCacheInvalidatesChangedSurfaceMetadata(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")
	writeFreshnessFile(t, filepath.Join(root, "go.mod"), "module example.test/root\n\ngo 1.23\n")
	writeFreshnessFile(t, filepath.Join(root, "root.go"), "package root\n")
	writeFreshnessFile(t, filepath.Join(root, "packages/unused/go.mod"), "module example.test/unused\n\ngo 1.23\n")
	writeFreshnessFile(t, filepath.Join(root, "packages/unused/unused.go"), "package unused\n")

	writeFreshnessFile(t, filepath.Join(root, "go.mod"), "module example.test/root\n\ngo 1.23\n\n// clean cache-index seed\n")
	first := checkFreshnessModule(t, root, ".")
	if !first.Clean || reportSurface(t, first, "root").Status != "clean" {
		t.Fatalf("initial root report = %+v, want clean", first)
	}

	writeFreshnessFile(t, filepath.Join(root, "go.mod"), "module example.test/root\n\ngo 1.23\n\nrequire example.test/unused v0.0.0\n\nreplace example.test/unused => ./packages/unused\n")
	second := checkFreshnessModule(t, root, ".")
	rootResult := reportSurface(t, second, "root")
	if rootResult.Status != "stale" || rootResult.CacheHit {
		t.Fatalf("changed root module = %+v, want uncached stale result after same-path metadata change", rootResult)
	}
}

func TestFreshnessDependencyFingerprintStopsAtNestedModules(t *testing.T) {
	root := t.TempDir()
	surfaceDir := filepath.Join(root, "scenarios/consumer/api")
	dependencyDir := filepath.Join(root, "packages/shared")
	writeFreshnessFile(t, filepath.Join(surfaceDir, "go.mod"), "module example.test/consumer\n\ngo 1.23\n")
	writeFreshnessFile(t, filepath.Join(surfaceDir, "consumer.go"), "package consumer\n")
	writeFreshnessFile(t, filepath.Join(dependencyDir, "go.mod"), "module example.test/shared\n\ngo 1.23\n")
	writeFreshnessFile(t, filepath.Join(dependencyDir, "shared.go"), "package shared\n")
	nestedDir := filepath.Join(dependencyDir, "fixtures/nested")
	writeFreshnessFile(t, filepath.Join(nestedDir, "go.mod"), "module example.test/nested\n\ngo 1.23\n")
	writeFreshnessFile(t, filepath.Join(nestedDir, "nested.go"), "package nested\n")

	modules := map[string]goModule{
		"example.test/consumer": {dir: surfaceDir, module: "example.test/consumer", requires: map[string]struct{}{"example.test/shared": {}}},
		"example.test/shared":   {dir: dependencyDir, module: "example.test/shared", requires: map[string]struct{}{}},
	}
	surface := goSurface{goMod: filepath.Join(surfaceDir, "go.mod"), module: "example.test/consumer"}
	key := func() string {
		t.Helper()
		surfaces := []goSurface{surface}
		attachFreshnessCacheFingerprints(surfaces, modules)
		value, ok := freshnessCacheKey(surfaces[0])
		if !ok {
			t.Fatal("freshnessCacheKey unexpectedly refused complete module inputs")
		}
		return value
	}

	initial := key()
	writeFreshnessFile(t, filepath.Join(nestedDir, "nested.go"), "package nested\n\nconst Changed = true\n")
	if nestedChanged := key(); nestedChanged != initial {
		t.Fatalf("nested module source changed cache key: before=%s after=%s", initial, nestedChanged)
	}
	writeFreshnessFile(t, filepath.Join(dependencyDir, "shared.go"), "package shared\n\nconst Changed = true\n")
	if parentChanged := key(); parentChanged == initial {
		t.Fatalf("owning module source change did not change cache key: %s", parentChanged)
	}
}

func TestFreshnessCacheDoesNotPublishWhenSurfaceChangesDuringCheck(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")
	moduleDir := filepath.Join(root, "scenarios/demo/api")
	sourcePath := filepath.Join(moduleDir, "main.go")
	writeFreshnessFile(t, filepath.Join(moduleDir, "go.mod"), "module example.test/demo\n\ngo 1.23\n")
	writeFreshnessFile(t, sourcePath, "package demo\n\nconst Value = 1\n")
	preCheckKey := freshnessCacheKeyForModule(t, root, "scenarios/demo/api")

	previous := runGoCommand
	calls := 0
	runGoCommand = func(_ context.Context, _ string, args ...string) (string, bool, error) {
		calls++
		if !reflect.DeepEqual(args, []string{"mod", "tidy", "-diff"}) {
			t.Fatalf("go args = %#v, want tidy diff", args)
		}
		if calls == 1 {
			if err := os.WriteFile(sourcePath, []byte("package demo\n\nconst Value = 2\n"), 0o644); err != nil {
				t.Fatalf("mutate surface during check: %v", err)
			}
			return "", true, nil
		}
		return "diff --git a/go.mod b/go.mod\n", false, nil
	}
	t.Cleanup(func() { runGoCommand = previous })

	first := checkFreshnessModule(t, root, "scenarios/demo/api")
	if item := reportSurface(t, first, "scenarios/demo/api"); item.Status != "clean" || item.CacheHit {
		t.Fatalf("first check = %+v, want uncached clean observation", item)
	}
	if _, err := os.Stat(filepath.Join(freshnessCacheDir(root), preCheckKey+".json")); !os.IsNotExist(err) {
		t.Fatalf("pre-check identity cache entry stat error = %v, want no entry for changed inputs", err)
	}

	second := checkFreshnessModule(t, root, "scenarios/demo/api")
	if item := reportSurface(t, second, "scenarios/demo/api"); item.Status != "stale" || item.CacheHit {
		t.Fatalf("second check = %+v, want a fresh stale result after the surface changed", item)
	}
	if calls != 2 {
		t.Fatalf("go calls = %d, want both checks to execute", calls)
	}
}

func TestFreshnessCacheDoesNotPublishWhenDependencyChangesDuringCheck(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")
	writeLocalFreshnessFixture(t, root)
	preCheckKey := freshnessCacheKeyForModule(t, root, "scenarios/consumer/api")

	previous := runGoCommand
	calls := 0
	runGoCommand = func(_ context.Context, _ string, args ...string) (string, bool, error) {
		calls++
		if !reflect.DeepEqual(args, []string{"mod", "tidy", "-diff"}) {
			t.Fatalf("go args = %#v, want tidy diff", args)
		}
		if calls == 1 {
			writeFreshnessFile(t, filepath.Join(root, "packages/shared/shared.go"), "package shared\n\nconst Value = \"changed during check\"\n")
			return "", true, nil
		}
		return "diff --git a/go.mod b/go.mod\n", false, nil
	}
	t.Cleanup(func() { runGoCommand = previous })

	first := checkFreshnessModule(t, root, "scenarios/consumer/api")
	if item := reportSurface(t, first, "scenarios/consumer/api"); item.Status != "clean" || item.CacheHit {
		t.Fatalf("first check = %+v, want uncached clean observation", item)
	}
	if _, err := os.Stat(filepath.Join(freshnessCacheDir(root), preCheckKey+".json")); !os.IsNotExist(err) {
		t.Fatalf("captured dependency identity cache entry stat error = %v, want no entry for changed inputs", err)
	}

	second := checkFreshnessModule(t, root, "scenarios/consumer/api")
	if item := reportSurface(t, second, "scenarios/consumer/api"); item.Status != "stale" || item.CacheHit {
		t.Fatalf("second check = %+v, want fresh stale result after dependency change", item)
	}
	if calls != 2 {
		t.Fatalf("go calls = %d, want both checks to execute", calls)
	}
}

func freshnessCacheKeyForModule(t *testing.T, root, module string) string {
	t.Helper()
	modules, err := discoverGoModules(root)
	if err != nil {
		t.Fatalf("discover modules: %v", err)
	}
	surfaces, err := discoverGoSurfacesFromModules(root, modules)
	if err != nil {
		t.Fatalf("discover surfaces: %v", err)
	}
	surfaces = filterGoSurfacesByModule(root, surfaces, module)
	if len(surfaces) != 1 {
		t.Fatalf("selected surfaces = %d, want 1 for %s", len(surfaces), module)
	}
	attachFreshnessCacheFingerprints(surfaces, modules)
	key, ok := freshnessCacheKey(surfaces[0])
	if !ok {
		t.Fatalf("cache identity unavailable for %s", module)
	}
	return key
}

func checkFreshnessModule(t *testing.T, root, module string) freshnessReport {
	t.Helper()
	report, err := checkFreshness(context.Background(), root, freshnessRequest{module: module, concurrency: 1})
	if err != nil {
		t.Fatalf("check module %s: %v", module, err)
	}
	return report
}

func reportSurface(t *testing.T, report freshnessReport, moduleDir string) freshnessSurfaceReport {
	t.Helper()
	for _, item := range report.Surfaces {
		modulePath := filepath.ToSlash(filepath.Dir(item.GoModPath))
		if modulePath == moduleDir || (moduleDir == "root" && modulePath == ".") {
			return item
		}
	}
	t.Fatalf("surface %q absent from report: %+v", moduleDir, report.Surfaces)
	return freshnessSurfaceReport{}
}

func findSurface(surfaces []freshnessSurfaceReport, moduleDir string) (freshnessSurfaceReport, bool) {
	for _, item := range surfaces {
		modulePath := filepath.ToSlash(filepath.Dir(item.GoModPath))
		if modulePath == moduleDir || (moduleDir == "root" && modulePath == ".") {
			return item, true
		}
	}
	return freshnessSurfaceReport{}, false
}

func writeLocalFreshnessFixture(t *testing.T, root string) {
	t.Helper()
	writeFreshnessFile(t, filepath.Join(root, "packages/shared/go.mod"), "module example.test/shared\n\ngo 1.23\n")
	writeFreshnessFile(t, filepath.Join(root, "packages/shared/shared.go"), "package shared\n\nconst Value = \"shared\"\n")
	writeFreshnessFile(t, filepath.Join(root, "packages/dep/go.mod"), "module example.test/dep\n\ngo 1.23\n")
	writeFreshnessFile(t, filepath.Join(root, "packages/dep/dep.go"), "package dep\n\nconst Value = \"dep\"\n")
	writeFreshnessFile(t, filepath.Join(root, "scenarios/consumer/api/go.mod"), "module example.test/consumer\n\ngo 1.23\n\nrequire example.test/shared v0.0.0\n\nreplace example.test/shared => ../../../packages/shared\nreplace example.test/dep => ../../../packages/dep\n")
	writeFreshnessFile(t, filepath.Join(root, "scenarios/consumer/api/main.go"), "package consumer\n\nimport \"example.test/shared\"\n\nvar Value = shared.Value\n")
	writeFreshnessFile(t, filepath.Join(root, "scenarios/unrelated/api/go.mod"), "module example.test/unrelated\n\ngo 1.23\n")
	writeFreshnessFile(t, filepath.Join(root, "scenarios/unrelated/api/main.go"), "package unrelated\n\nconst Value = \"unrelated\"\n")
}

func initFreshnessGitRepo(t *testing.T, root string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	run("init", "-q")
	run("config", "user.email", "fixture@example.test")
	run("config", "user.name", "Freshness Fixture")
	run("add", ".")
	run("commit", "-qm", "fixture baseline")
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

func TestCheckFreshnessModuleApplyActionPreservesAndQuotesSelector(t *testing.T) {
	root := t.TempDir()
	selector := "scenarios/demo api's"
	moduleDir := filepath.Join(root, filepath.FromSlash(selector))
	writeFreshnessFile(t, filepath.Join(moduleDir, "go.mod"), "module example.com/demo\n")
	stubRunGo(t, []stubGoCall{
		{args: []string{"mod", "tidy", "-diff"}, diff: "diff --git a/go.mod b/go.mod\n"},
	})

	got, err := checkFreshness(context.Background(), root, freshnessRequest{module: selector, concurrency: 1})
	if err != nil {
		t.Fatalf("checkFreshness: %v", err)
	}
	if got.Summary.Stale != 1 || len(got.NextActions) != 1 {
		t.Fatalf("report = %+v, want one stale selected module and one apply action", got)
	}
	want := "scenario-dependency-analyzer freshness --module 'scenarios/demo api'\\''s' --apply"
	if got.NextActions[0].Command != want {
		t.Fatalf("next action command = %q, want %q", got.NextActions[0].Command, want)
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
	// do runs before the call returns, with the surface directory the real
	// command would have run in. Lets a stubbed `go mod tidy` write go.sum the
	// way the real one would.
	do func(dir string)
}

func stubRunGo(t *testing.T, calls []stubGoCall) *int {
	t.Helper()
	count := 0
	previous := runGoCommand
	runGoCommand = func(_ context.Context, dir string, args ...string) (string, bool, error) {
		if count >= len(calls) {
			t.Fatalf("unexpected go call %d with args %#v", count+1, args)
		}
		call := calls[count]
		count++
		if !reflect.DeepEqual(args, call.args) {
			t.Fatalf("go call %d args = %#v, want %#v", count, args, call.args)
		}
		if call.do != nil {
			call.do(dir)
		}
		return call.diff, call.clean, call.err
	}
	t.Cleanup(func() { runGoCommand = previous })
	return &count
}

func TestDiscoverGoModulesPreservesRootOwnerAgainstRetainedCopy(t *testing.T) {
	root := t.TempDir()
	writeFreshnessFile(t, filepath.Join(root, "go.mod"), "module example.com/root\n")
	writeFreshnessFile(t, filepath.Join(root, "scenarios", "demo", "evidence", "go.mod"), "module example.com/root\n")
	modules, err := discoverGoModules(root)
	if err != nil {
		t.Fatal(err)
	}
	if modules["example.com/root"].dir != root {
		t.Fatalf("retained copy shadowed root owner: %+v", modules)
	}
}

func TestFreshnessUnknownModuleCannotReportCleanZeroChecks(t *testing.T) {
	root := t.TempDir()
	writeFreshnessFile(t, filepath.Join(root, "go.mod"), "module example.com/root\n")
	_, err := checkFreshness(context.Background(), root, freshnessRequest{module: "missing/go.mod"})
	if err == nil || !strings.Contains(err.Error(), "no Go module matches") {
		t.Fatalf("unknown selector must fail, got %v", err)
	}
}

func TestFreshnessAmbiguousNonRootModuleCannotReportClean(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"real", "retained"} {
		writeFreshnessFile(t, filepath.Join(root, name, "go.mod"), "module example.com/duplicate\n")
	}
	_, err := checkFreshness(context.Background(), root, freshnessRequest{module: "example.com/duplicate"})
	if err == nil || !strings.Contains(err.Error(), "multiple source directories") {
		t.Fatalf("ambiguous module must fail, got %v", err)
	}
}

func TestModuleDotSelectsOnlyRoot(t *testing.T) {
	root := t.TempDir()
	surfaces := []goSurface{
		{goMod: filepath.Join(root, "go.mod"), module: "example.com/root"},
		{goMod: filepath.Join(root, "other", "go.mod"), module: "example.com/other"},
	}
	got := filterGoSurfacesByModule(root, surfaces, ".")
	if len(got) != 1 || got[0].module != "example.com/root" {
		t.Fatalf("dot selector expanded beyond root: %+v", got)
	}
	if got := filterGoSurfacesByModule(root, surfaces, " "); len(got) != 0 {
		t.Fatalf("whitespace selector selected modules: %+v", got)
	}
}

func TestExplicitEmptyModuleRefusesBeforeApply(t *testing.T) {
	for _, selector := range []string{"", " "} {
		t.Run(selector, func(t *testing.T) {
			calls := stubRunGo(t, nil)
			err := runFreshness([]string{"--module", selector, "--apply"})
			if err == nil || !strings.Contains(err.Error(), "nonempty") {
				t.Fatalf("empty selector must refuse: %v", err)
			}
			if *calls != 0 {
				t.Fatal("empty selector executed Go")
			}
		})
	}
}

func TestExplicitModulePathDisambiguatesRetainedCopies(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"resources/go.mod", "scenarios/demo/evidence/go.mod"} {
		writeFreshnessFile(t, filepath.Join(root, path), "module example.com/resources\n")
	}
	stubRunGo(t, []stubGoCall{{args: []string{"mod", "tidy", "-diff"}, clean: true}})
	got, err := checkFreshness(context.Background(), root, freshnessRequest{module: "resources/go.mod", concurrency: 1, noCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary.Checked != 1 || len(got.Surfaces) != 1 || got.Surfaces[0].GoModPath != "resources/go.mod" {
		t.Fatalf("explicit owner selection changed: %+v", got)
	}
}

func TestExplicitModuleSymlinkCannotEscapeRoot(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	writeFreshnessFile(t, filepath.Join(external, "go.mod"), "module example.com/external\n")
	if err := os.Symlink(external, filepath.Join(root, "outside")); err != nil {
		t.Fatal(err)
	}
	stubRunGo(t, nil)
	_, err := checkFreshness(context.Background(), root, freshnessRequest{module: "outside", apply: true})
	if err == nil || !strings.Contains(err.Error(), "escapes repository") {
		t.Fatalf("escaped owner must refuse: %v", err)
	}
}

func TestKilledModuleRetryCannotReportClean(t *testing.T) {
	root := t.TempDir()
	writeFreshnessFile(t, filepath.Join(root, "go.mod"), "module example.com/root\n")
	previousOffline, previousNetwork := runGoCommand, runGoNetwork
	runGoCommand = func(context.Context, string, ...string) (string, bool, error) {
		return "", false, errors.New("go: module lookup disabled by GOPROXY=off")
	}
	runGoNetwork = func(context.Context, string, ...string) (string, bool, error) {
		return "", false, errors.New("signal: killed: go: finding module for package example.com/dep")
	}
	t.Cleanup(func() { runGoCommand, runGoNetwork = previousOffline, previousNetwork })
	got, err := checkFreshness(context.Background(), root, freshnessRequest{module: "go.mod", noCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.Clean || got.Summary.NeedsDownload != 1 || !strings.Contains(got.Surfaces[0].Error, "killed") {
		t.Fatalf("killed resolution cannot qualify freshness: %+v", got)
	}
}
