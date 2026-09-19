package validation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"unit-health/internal/discovery"
	"unit-health/internal/evidence"
	"unit-health/internal/executor"
)

type countingRunner struct{ calls atomic.Int32 }

type delayedRunner struct {
	calls atomic.Int32
	delay time.Duration
}

type blockingRunner struct {
	calls   atomic.Int32
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func (r *blockingRunner) Run(_ context.Context, command executor.Command) executor.Result {
	r.calls.Add(1)
	r.once.Do(func() { close(r.started) })
	<-r.release
	return executor.Result{WorkspaceID: command.WorkspaceID, Name: command.Name, Status: executor.StatusPassed, DurationMS: 42, CPUTimeMS: 7}
}

func (r *countingRunner) Run(_ context.Context, command executor.Command) executor.Result {
	r.calls.Add(1)
	return executor.Result{WorkspaceID: command.WorkspaceID, Name: command.Name, Status: executor.StatusPassed, DurationMS: 42, CPUTimeMS: 7}
}

func (r *delayedRunner) Run(_ context.Context, command executor.Command) executor.Result {
	r.calls.Add(1)
	time.Sleep(r.delay)
	return executor.Result{WorkspaceID: command.WorkspaceID, Name: command.Name, Status: executor.StatusPassed, DurationMS: r.delay.Milliseconds(), CPUTimeMS: 1}
}

func TestValidateWarmCacheMeetsTwentyFoldRunnerAvoidanceTarget(t *testing.T) {
	root := t.TempDir()
	inv := goSurfaceInventoryAt(t, root)
	store, err := evidence.NewStore(t.TempDir(), 1<<20, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	runner := &delayedRunner{delay: 100 * time.Millisecond}
	service := newService(fakeDiscoverer{inv: inv}, loadSpec(t))
	service.EvidenceStore = store
	service.Executor = runner
	request := Request{Scenario: "demo", IncludeExecution: true, UseCache: true}

	start := time.Now()
	if _, err := service.Validate(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	cold := time.Since(start)
	start = time.Now()
	response, err := service.Validate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	warm := time.Since(start)
	if !response.CacheHit || runner.calls.Load() != 1 {
		t.Fatalf("warm response hit=%v runner calls=%d", response.CacheHit, runner.calls.Load())
	}
	if warm <= 0 || cold < 20*warm {
		t.Fatalf("warm cache speedup was below 20x: cold=%s warm=%s", cold, warm)
	}
}

func TestEvidenceKeyIncludesEffectiveWorkspaceRunnerProfiles(t *testing.T) {
	root := t.TempDir()
	service := &Service{}
	bounded := []Workspace{{ID: "ui", RootPath: root, RunnerProfile: "bounded-batches"}}
	serial := []Workspace{{ID: "ui", RootPath: root, RunnerProfile: "serial-isolation"}}
	boundedKey, err := service.evidenceKey(root, "scenario", bounded)
	if err != nil {
		t.Fatal(err)
	}
	serialKey, err := service.evidenceKey(root, "scenario", serial)
	if err != nil {
		t.Fatal(err)
	}
	if boundedKey.Digest == serialKey.Digest {
		t.Fatal("effective runner profile change reused the same evidence key")
	}
}

func TestEvidenceKeySeparatesIdenticalWorkspaceContentsByScope(t *testing.T) {
	root := t.TempDir()
	service := &Service{}
	first, err := service.evidenceKey(root, "scenario", []Workspace{{ID: "api", RootPath: root}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.evidenceKey(root, "scenario", []Workspace{{ID: "worker", RootPath: root}})
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest == second.Digest {
		t.Fatal("workspace scope change reused the same evidence key")
	}
}

func TestEvidenceKeySeparatesFastTestAndCoverageEvidence(t *testing.T) {
	root := t.TempDir()
	service := &Service{}
	workspaces := []Workspace{{
		ID: "api", RootPath: root, TestCommand: "go test ./...", CoverageCommand: "go test -coverprofile=coverage.out ./...",
	}}
	coverageKey, err := service.evidenceKeyForMode(root, "scenario", workspaces, false)
	if err != nil {
		t.Fatal(err)
	}
	fastKey, err := service.evidenceKeyForMode(root, "scenario", workspaces, true)
	if err != nil {
		t.Fatal(err)
	}
	if coverageKey.Digest == fastKey.Digest {
		t.Fatal("fast-test and coverage evidence reused the same key")
	}
}

func TestEvidenceKeyIncludesHostHermeticCapabilities(t *testing.T) {
	root := t.TempDir()
	key, err := (&Service{}).evidenceKey(root, "scenario", []Workspace{{ID: "api", RootPath: root}})
	if err != nil {
		t.Fatal(err)
	}
	canonical := string(key.Canonical)
	for _, dimension := range []string{"hermetic_network_deny", "hermetic_declared_net", "hermetic_workspace_ro", "hermetic_child_leak", "hermetic_open_handles", "hermetic_order_independent", "hermetic_environment"} {
		if !strings.Contains(canonical, dimension) {
			t.Fatalf("canonical key omitted host capability dimension %q: %s", dimension, canonical)
		}
	}
}

func TestFastTestPlanDoesNotRunCoverageCommandOrRetainCoverageArtifacts(t *testing.T) {
	workspaces := []Workspace{{
		ID: "ui", Language: "typescript", TestCommand: "pnpm test", TestExecutable: "/tools/pnpm",
		TestArgs: []string{"run", "test"}, CoverageCommand: "pnpm test:coverage", CoverageExecutable: "/tools/pnpm",
		CoverageArgs: []string{"run", "test:coverage"}, TestArtifacts: []Artifact{{Kind: "lcov", Reference: "coverage/lcov.info"}},
	}}
	plan := buildExecutionPlanForMode(workspaces, true)
	if len(plan.Commands) != 1 || plan.Commands[0].Command != "pnpm test" || plan.Commands[0].Executable != "/tools/pnpm" {
		t.Fatalf("fast-test plan=%+v", plan)
	}
	if len(plan.Commands[0].Args) != 2 || plan.Commands[0].Args[1] != "test" || len(plan.Commands[0].Artifacts) != 0 {
		t.Fatalf("fast-test command retained coverage projection: %+v", plan.Commands[0])
	}
}

func TestValidateFastTestOnlyExecutesTestEvidenceWithoutCoverage(t *testing.T) {
	root := t.TempDir()
	service := newService(fakeDiscoverer{inv: goSurfaceInventoryAt(t, root)}, loadSpec(t))
	service.Executor = &countingRunner{}
	response, err := service.Validate(context.Background(), Request{Scenario: "demo", IncludeExecution: true, FastTestOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Plan.Commands) != 1 || strings.Contains(response.Plan.Commands[0].Command, "coverprofile") {
		t.Fatalf("fast-test execution plan=%+v", response.Plan)
	}
	if len(response.Coverage) != 0 {
		t.Fatalf("fast-test response unexpectedly contains coverage=%+v", response.Coverage)
	}
}

func TestValidateCacheHitStartsNoRunnerChildren(t *testing.T) {
	root := t.TempDir()
	inv := goSurfaceInventoryAt(t, root)
	store, err := evidence.NewStore(t.TempDir(), 1<<20, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	runner := &countingRunner{}
	service := newService(fakeDiscoverer{inv: inv}, loadSpec(t))
	service.EvidenceStore = store
	service.Executor = runner
	request := Request{Scenario: "demo", IncludeExecution: true, UseCache: true}
	first, err := service.Validate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.CacheHit || runner.calls.Load() != 1 {
		t.Fatalf("first response hit=%v calls=%d", first.CacheHit, runner.calls.Load())
	}
	second, err := service.Validate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !second.CacheHit {
		t.Fatal("second response was not an exact cache hit")
	}
	if runner.calls.Load() != 1 {
		t.Fatalf("cache hit started runner children: calls=%d", runner.calls.Load())
	}
	if second.CacheSavedWallTimeMS != 42 || second.CacheSavedCPUTimeMS != 7 || second.CacheRetainedBytes <= 0 {
		t.Fatalf("cache observability = wall=%d cpu=%d bytes=%d, want saved timing and retained bytes", second.CacheSavedWallTimeMS, second.CacheSavedCPUTimeMS, second.CacheRetainedBytes)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "go.mod"), []byte("module changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	third, err := service.Validate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if third.CacheHit || runner.calls.Load() != 2 {
		t.Fatalf("source change reused evidence: hit=%v calls=%d", third.CacheHit, runner.calls.Load())
	}
	if len(third.CacheInvalidatedDimensions) != 1 || third.CacheInvalidatedDimensions[0] != "exact_key_miss" {
		t.Fatalf("cache miss dimensions=%v, want exact_key_miss", third.CacheInvalidatedDimensions)
	}
}

func TestValidateCoalescesConcurrentExactCacheMisses(t *testing.T) {
	root := t.TempDir()
	inv := goSurfaceInventoryAt(t, root)
	store, err := evidence.NewStore(t.TempDir(), 1<<20, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	runner := &blockingRunner{started: make(chan struct{}), release: make(chan struct{})}
	service := newService(fakeDiscoverer{inv: inv}, loadSpec(t))
	service.EvidenceStore = store
	service.Executor = runner
	request := Request{Scenario: "demo", IncludeExecution: true, UseCache: true}
	firstDone := make(chan error, 1)
	go func() {
		_, validateErr := service.Validate(context.Background(), request)
		firstDone <- validateErr
	}()
	<-runner.started
	secondDone := make(chan error, 1)
	go func() {
		_, validateErr := service.Validate(context.Background(), request)
		secondDone <- validateErr
	}()
	close(runner.release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if err := <-secondDone; err != nil {
		t.Fatal(err)
	}
	if calls := runner.calls.Load(); calls != 1 {
		t.Fatalf("concurrent exact cache miss started %d runner children, want 1", calls)
	}
}

func goSurfaceInventoryAt(t *testing.T, root string) discovery.Inventory {
	t.Helper()
	writeFile(t, filepath.Join(root, "api", "go.mod"), "module x\n")
	return discovery.Inventory{Scenario: "demo", TargetKind: "scenario", RootPath: root, Surfaces: []discovery.Surface{{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api"), Status: "known"}}}
}
