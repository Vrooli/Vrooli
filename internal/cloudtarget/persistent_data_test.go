package cloudtarget

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type envRecordingRunner struct {
	recordingRunner
	env [][]string
}

func (r *envRecordingRunner) RunEnv(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	r.env = append(r.env, env)
	return r.Run(ctx, name, args...)
}

func stageFixture(t *testing.T, f fixture, op string, fence uint64) {
	t.Helper()
	if _, err := f.store.Stage(context.Background(), StageRequest{Effect: f.effect(op, "stage", fence), Verify: f.verify()}); err != nil {
		t.Fatal(err)
	}
}

func mustReadLink(t *testing.T, path string) string {
	t.Helper()
	target, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("readlink %s: %v", path, err)
	}
	return target
}

// [REQ:STC-P0-028] Legacy conversion: the first activation adopts declared
// legacy directories from the in-place workdir by rename into the persistent
// root and links the release tree to them; a heuristic directory nobody
// mapped is reported and left untouched, never deleted.
func TestActivateAdoptsLegacyDataAndRefusesToDeleteUnmapped(t *testing.T) {
	f := newFixture(t, append(defaultBundleEntries(), bundleEntry{name: "scenarios/" + testScenario + "/uploads/.keep", body: ""}), nil)
	stageFixture(t, f, "op-1", 1)
	legacyRoot := t.TempDir()
	scenarioDir := filepath.Join(legacyRoot, "scenarios", testScenario)
	for _, rel := range []string{"uploads", "data/records", "cache"} {
		if err := os.MkdirAll(filepath.Join(scenarioDir, rel), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "uploads", "invoice-001.txt"), []byte("seeded upload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cache", "hot"), []byte("cache"), 0o644); err != nil {
		t.Fatal(err)
	}
	bindings, err := ParseDataBindingFlags([]string{"uploads=" + testScenario + "/uploads"})
	if err != nil {
		t.Fatal(err)
	}
	carry, err := ParseLegacyCarryFlags([]string{"scenarios/" + testScenario + "/data/records"})
	if err != nil {
		t.Fatal(err)
	}
	activator := &recordingActivator{}
	result, err := f.store.Activate(context.Background(), ActivateRequest{Effect: f.effect("op-1", "activate", 1), Release: f.manifest.ReleaseDigest, Strategy: StrategyMaintenance, Activator: activator, DataBindings: bindings, LegacyCarry: carry, LegacyRoot: legacyRoot})
	if err != nil || result.Receipt.Outcome != OutcomeSucceeded {
		t.Fatalf("activate = %+v err=%v", result.Receipt, err)
	}
	persistent, _ := f.store.PersistentDataDir(testDeployment)
	if got, err := os.ReadFile(filepath.Join(persistent, "uploads", "invoice-001.txt")); err != nil || string(got) != "seeded upload" {
		t.Fatalf("seeded upload not adopted into the persistent root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(scenarioDir, "uploads")); !os.IsNotExist(err) {
		t.Fatal("adoption must move, not copy: the legacy directory should be gone")
	}
	releaseDir, _ := f.store.ReleaseDir(testDeployment, f.manifest.ReleaseDigest)
	if link := mustReadLink(t, filepath.Join(releaseDir, "scenarios", testScenario, "uploads")); link != filepath.Join(persistent, "uploads") {
		t.Fatalf("uploads link = %s", link)
	}
	if link := mustReadLink(t, filepath.Join(releaseDir, "scenarios", testScenario, "data", "records")); link != filepath.Join(persistent, legacyCarryDirName, testScenario, "data", "records") {
		t.Fatalf("legacy carry link = %s", link)
	}
	if got, err := os.ReadFile(filepath.Join(scenarioDir, "cache", "hot")); err != nil || string(got) != "cache" {
		t.Fatalf("unmapped cache directory must survive untouched: %v", err)
	}
	unmapped, _ := result.Receipt.Details["legacy_unmapped"].([]string)
	if len(unmapped) != 1 || !strings.HasSuffix(unmapped[0], ":"+testScenario+"/cache") {
		t.Fatalf("legacy_unmapped = %v", result.Receipt.Details["legacy_unmapped"])
	}
	if len(activator.activations) != 1 {
		t.Fatalf("activations = %d", len(activator.activations))
	}
}

// [REQ:STC-P0-028] Seeded data survives a code update and a failed
// activation: the second release links the same persistent directory, and a
// candidate that never becomes healthy leaves the data and the prior pointer
// in place.
func TestPersistentDataSurvivesUpdateAndFailedActivation(t *testing.T) {
	first := newFixture(t, defaultBundleEntries(), nil)
	second := newFixture(t, append(defaultBundleEntries(), bundleEntry{name: "scenarios/" + testScenario + "/CHANGELOG", body: "v2"}), nil)
	second.store = first.store
	stageFixture(t, first, "op-1", 1)
	bindings, _ := ParseDataBindingFlags([]string{"uploads=" + testScenario + "/uploads"})
	ctx := context.Background()
	if _, err := first.store.Activate(ctx, ActivateRequest{Effect: first.effect("op-1", "activate", 1), Release: first.manifest.ReleaseDigest, Activator: &recordingActivator{}, DataBindings: bindings}); err != nil {
		t.Fatal(err)
	}
	persistent, _ := first.store.PersistentDataDir(testDeployment)
	if err := os.WriteFile(filepath.Join(persistent, "uploads", "photo-002.txt"), []byte("written while v1 ran"), 0o644); err != nil {
		t.Fatal(err)
	}
	stageFixture(t, second, "op-2", 2)
	failed := &recordingActivator{err: errStub("candidate never became healthy")}
	if _, err := second.store.Activate(ctx, ActivateRequest{Effect: second.effect("op-2", "activate", 2), Release: second.manifest.ReleaseDigest, Activator: failed, DataBindings: bindings}); mustCode(t, err) != CodeActivationFailed {
		t.Fatalf("expected activation_failed, got %v", err)
	}
	active, _ := first.store.ReadActive(testDeployment)
	if active == nil || active.ActiveRelease != first.manifest.ReleaseDigest {
		t.Fatalf("prior release must stay active after a failed candidate: %+v", active)
	}
	if got, err := os.ReadFile(filepath.Join(persistent, "uploads", "photo-002.txt")); err != nil || string(got) != "written while v1 ran" {
		t.Fatalf("data lost after failed activation: %v", err)
	}
	if _, err := second.store.Activate(ctx, ActivateRequest{Effect: second.effect("op-3", "activate", 3), Release: second.manifest.ReleaseDigest, Activator: &recordingActivator{}, DataBindings: bindings}); err != nil {
		t.Fatal(err)
	}
	secondDir, _ := second.store.ReleaseDir(testDeployment, second.manifest.ReleaseDigest)
	if link := mustReadLink(t, filepath.Join(secondDir, "scenarios", testScenario, "uploads")); link != filepath.Join(persistent, "uploads") {
		t.Fatalf("second release must bind the same persistent directory, got %s", link)
	}
	if got, err := os.ReadFile(filepath.Join(persistent, "uploads", "photo-002.txt")); err != nil || string(got) != "written while v1 ran" {
		t.Fatalf("data lost across the update: %v", err)
	}
	active, _ = first.store.ReadActive(testDeployment)
	if active.ActiveRelease != second.manifest.ReleaseDigest || active.PreviousRelease != first.manifest.ReleaseDigest {
		t.Fatalf("pointer after update = %+v", active)
	}
}

type errStub string

func (e errStub) Error() string { return string(e) }

// [REQ:STC-P0-028] Listener ports reach the lifecycle owner as the
// conventional environment, never as argv, and a restart of the recorded
// release re-runs the activator while keeping the predecessor.
func TestActivatorPinsPortsThroughEnvironmentAndRestartKeepsPredecessor(t *testing.T) {
	runner := &envRecordingRunner{}
	activator := ScenarioRestartActivator{Runner: runner, Executable: "vrooli"}
	if err := activator.Activate(context.Background(), Activation{ReleaseDir: "/rel", Scenarios: []string{testScenario}, Ports: map[string]int{"ui": 3000, "api": 3001}}); err != nil {
		t.Fatal(err)
	}
	if len(runner.env) != 1 || strings.Join(runner.env[0], " ") != "API_PORT=3001 UI_PORT=3000" {
		t.Fatalf("env = %v", runner.env)
	}
	for _, call := range runner.calls {
		for _, arg := range call {
			if strings.Contains(arg, "3000") {
				t.Fatalf("port must not travel as argv: %v", call)
			}
		}
	}
	plain := ScenarioRestartActivator{Runner: &recordingRunner{}}
	if err := plain.Activate(context.Background(), Activation{ReleaseDir: "/rel", Scenarios: []string{testScenario}, Ports: map[string]int{"ui": 3000}}); err == nil {
		t.Fatal("a runner without environment support must refuse pinned ports")
	}

	f := newFixture(t, defaultBundleEntries(), nil)
	stageFixture(t, f, "op-1", 1)
	ctx := context.Background()
	if _, err := f.store.Activate(ctx, ActivateRequest{Effect: f.effect("op-1", "activate", 1), Release: f.manifest.ReleaseDigest, Activator: &recordingActivator{}}); err != nil {
		t.Fatal(err)
	}
	unchanged, err := f.store.Activate(ctx, ActivateRequest{Effect: f.effect("op-2", "activate", 2), Release: f.manifest.ReleaseDigest, Activator: &recordingActivator{}})
	if err != nil || unchanged.Receipt.Outcome != OutcomeUnchanged {
		t.Fatalf("re-activating the active release without --restart must be unchanged: %+v %v", unchanged.Receipt, err)
	}
	restarter := &recordingActivator{}
	restarted, err := f.store.Activate(ctx, ActivateRequest{Effect: f.effect("op-3", "activate", 3), Release: f.manifest.ReleaseDigest, Activator: restarter, RestartIfActive: true, Ports: map[string]int{"ui": 3000}})
	if err != nil || restarted.Receipt.Outcome != OutcomeSucceeded || len(restarter.activations) != 1 || restarter.activations[0].Ports["ui"] != 3000 {
		t.Fatalf("restart = %+v err=%v activations=%+v", restarted.Receipt, err, restarter.activations)
	}
	active, _ := f.store.ReadActive(testDeployment)
	if active.ActiveRelease != f.manifest.ReleaseDigest || active.PreviousRelease != "" {
		t.Fatalf("restart must keep the pointer: %+v", active)
	}
}

func TestParseDataBindingFlagsRefusesEscapes(t *testing.T) {
	for _, bad := range []string{"uploads", "=x/y", "uploads=x", "uploads=x/../y", "uploads=/abs/path", "up loads=x/y"} {
		if _, err := ParseDataBindingFlags([]string{bad}); mustCode(t, err) != CodeInvalidArgument {
			t.Fatalf("%q: expected invalid_argument, got %v", bad, err)
		}
	}
	if _, err := ParseDataBindingFlags([]string{"a=x/y", "a=x/z"}); mustCode(t, err) != CodeInvalidArgument {
		t.Fatalf("duplicate id must be refused: %v", err)
	}
}
