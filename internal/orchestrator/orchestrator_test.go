package orchestrator

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/vroolierr"
	testprocess "github.com/vrooli/vrooli/packages/testkit-go/processfixture"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=1 | LAST: 2026-04-11

func TestListAndStatusReflectRuntimeRecords(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest("alpha", testscenario.WithDisplayName("Alpha"), testscenario.WithDescription("running"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	testscenario.WriteScenarioService(t, root, "beta", testscenario.ScenarioServiceManifest("beta", testscenario.WithDisplayName("Beta"), testscenario.WithDescription("stopped"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	testprocess.WriteScenarioProcessRecord(t, home, "alpha", "start-api", process.Record{PID: os.Getpid(), PGID: os.Getpid(), Scenario: "alpha", Step: "start-api", Port: 18080, StartedAt: time.Now().Add(-2 * time.Minute).UTC(), Status: "running"})

	service := New(root, home, io.Discard, io.Discard)

	items, err := service.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(items))
	}

	alpha, exists, err := service.Status("alpha")
	if err != nil {
		t.Fatalf("Status(alpha): %v", err)
	}
	if !exists {
		t.Fatal("expected alpha to exist")
	}
	if alpha.Status != "running" || alpha.Processes != 1 {
		t.Fatalf("alpha status = %+v", alpha)
	}
	if alpha.Ports["API_PORT"] != 18080 {
		t.Fatalf("alpha ports = %#v", alpha.Ports)
	}

	running, err := service.Running()
	if err != nil {
		t.Fatalf("Running: %v", err)
	}
	if len(running) != 1 || running[0].Name != "alpha" {
		t.Fatalf("running = %#v", running)
	}
}

func TestDetailReturnsTypedNotFoundError(t *testing.T) {
	service := New(t.TempDir(), t.TempDir(), io.Discard, io.Discard)

	_, err := service.Detail("missing")
	if err == nil {
		t.Fatal("expected missing scenario error")
	}
	if got := vroolierr.Code(err, ""); got != "scenario_not_found" {
		t.Fatalf("error code = %q, want scenario_not_found", got)
	}
}

func TestResolvePortFallsBackFromUIToAPI(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest("alpha", testscenario.WithDisplayName("Alpha"), testscenario.WithDescription("running"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	testprocess.WriteScenarioProcessRecord(t, home, "alpha", "start-api", process.Record{PID: os.Getpid(), PGID: os.Getpid(), Scenario: "alpha", Step: "start-api", Port: 18080, StartedAt: time.Now().Add(-2 * time.Minute).UTC(), Status: "running"})

	service := New(root, home, io.Discard, io.Discard)
	resolved, err := service.ResolvePort("alpha", "UI_PORT")
	if err != nil {
		t.Fatalf("ResolvePort: %v", err)
	}
	if resolved.Name != "API_PORT" || resolved.Port != 18080 || resolved.URL != "http://localhost:18080" {
		t.Fatalf("resolved = %+v", resolved)
	}
}

func TestResolvePortUsesRegistryInPreferModeWithoutProcessRecords(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv(scenarioruntime.ModeEnv, scenarioruntime.ModePrefer)
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest("alpha", testscenario.WithDisplayName("Alpha"), testscenario.WithDescription("registry"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	writeRegistryRuntime(t, home, "alpha", scenarioruntime.StatusRunning, scenarioruntime.ClaimStatusBound, "api", "API_PORT", 18081)

	service := New(root, home, io.Discard, io.Discard)
	resolved, err := service.ResolvePort("alpha", "API_PORT")
	if err != nil {
		t.Fatalf("ResolvePort: %v", err)
	}
	if resolved.Name != "API_PORT" || resolved.Port != 18081 {
		t.Fatalf("resolved = %+v, want API_PORT 18081", resolved)
	}

	status, exists, err := service.Status("alpha")
	if err != nil {
		t.Fatalf("Status(alpha): %v", err)
	}
	if !exists {
		t.Fatal("expected alpha to exist")
	}
	if status.Status != "running" || status.Ports["API_PORT"] != 18081 {
		t.Fatalf("status = %+v, want registry running with API_PORT 18081", status)
	}
}

func TestStrictRegistryModeDoesNotUseLegacyProcessRecords(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv(scenarioruntime.ModeEnv, scenarioruntime.ModeStrict)
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest("alpha", testscenario.WithDisplayName("Alpha"), testscenario.WithDescription("legacy"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	testprocess.WriteScenarioProcessRecord(t, home, "alpha", "start-api", process.Record{PID: os.Getpid(), PGID: os.Getpid(), Scenario: "alpha", Step: "start-api", Port: 18080, StartedAt: time.Now().Add(-2 * time.Minute).UTC(), Status: "running"})

	service := New(root, home, io.Discard, io.Discard)
	status, exists, err := service.Status("alpha")
	if err != nil {
		t.Fatalf("Status(alpha): %v", err)
	}
	if !exists {
		t.Fatal("expected alpha to exist")
	}
	if status.Status == "running" || len(status.Ports) != 0 {
		t.Fatalf("status = %+v, want stopped/empty from strict registry mode", status)
	}

	_, err = service.ResolvePort("alpha", "API_PORT")
	if got := vroolierr.Code(err, ""); got != "scenario_not_running" {
		t.Fatalf("ResolvePort error code = %q, want scenario_not_running; err=%v", got, err)
	}
}

func TestStrictRegistryModeResolvesPortWithoutPIDVisibility(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv(scenarioruntime.ModeEnv, scenarioruntime.ModeStrict)
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest("alpha", testscenario.WithDisplayName("Alpha"), testscenario.WithDescription("registry"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	testprocess.WriteScenarioProcessRecord(t, home, "alpha", "start-api", process.Record{
		PID:       999999999,
		PGID:      999999999,
		Scenario:  "alpha",
		Step:      "start-api",
		Port:      18080,
		StartedAt: time.Now().Add(-2 * time.Minute).UTC(),
		Status:    "running",
	})
	writeRegistryRuntime(t, home, "alpha", scenarioruntime.StatusRunning, scenarioruntime.ClaimStatusBound, "api", "API_PORT", 18081)

	service := New(root, home, io.Discard, io.Discard)
	resolved, err := service.ResolvePort("alpha", "API_PORT")
	if err != nil {
		t.Fatalf("ResolvePort: %v", err)
	}
	if resolved.Port != 18081 || resolved.Name != "API_PORT" {
		t.Fatalf("resolved = %+v, want registry API_PORT 18081", resolved)
	}

	status, exists, err := service.Status("alpha")
	if err != nil {
		t.Fatalf("Status(alpha): %v", err)
	}
	if !exists {
		t.Fatal("expected alpha to exist")
	}
	if status.Status != "running" || status.Ports["API_PORT"] != 18081 {
		t.Fatalf("status = %+v, want registry running with API_PORT 18081", status)
	}
}

func TestStrictRegistryModeDoesNotExposeReservedRegistryClaimsAsPorts(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv(scenarioruntime.ModeEnv, scenarioruntime.ModeStrict)
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest("alpha", testscenario.WithDisplayName("Alpha"), testscenario.WithDescription("registry"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	writeRegistryRuntime(t, home, "alpha", scenarioruntime.StatusRunning, scenarioruntime.ClaimStatusReserved, "api", "API_PORT", 18081)

	service := New(root, home, io.Discard, io.Discard)
	status, exists, err := service.Status("alpha")
	if err != nil {
		t.Fatalf("Status(alpha): %v", err)
	}
	if !exists {
		t.Fatal("expected alpha to exist")
	}
	if status.Status != "running" {
		t.Fatalf("status.Status = %q, want running instance without exposed ports", status.Status)
	}
	if len(status.Ports) != 0 {
		t.Fatalf("status.Ports = %#v, want no ports from reserved claim", status.Ports)
	}
	_, err = service.ResolvePort("alpha", "API_PORT")
	if got := vroolierr.Code(err, ""); got != "scenario_port_not_found" {
		t.Fatalf("ResolvePort error code = %q, want scenario_port_not_found; err=%v", got, err)
	}
}

func TestPreferRegistryModeFallsBackToLegacyWhenRegistryMissing(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv(scenarioruntime.ModeEnv, scenarioruntime.ModePrefer)
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest("alpha", testscenario.WithDisplayName("Alpha"), testscenario.WithDescription("legacy"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	testprocess.WriteScenarioProcessRecord(t, home, "alpha", "start-api", process.Record{PID: os.Getpid(), PGID: os.Getpid(), Scenario: "alpha", Step: "start-api", Port: 18080, StartedAt: time.Now().Add(-2 * time.Minute).UTC(), Status: "running"})

	service := New(root, home, io.Discard, io.Discard)
	resolved, err := service.ResolvePort("alpha", "API_PORT")
	if err != nil {
		t.Fatalf("ResolvePort: %v", err)
	}
	if resolved.Port != 18080 {
		t.Fatalf("resolved.Port = %d, want legacy fallback 18080", resolved.Port)
	}
}

func TestPreferRegistryModeFallsBackWhenRegistryInstanceIsFromPreviousBoot(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv(scenarioruntime.ModeEnv, scenarioruntime.ModePrefer)
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest("alpha", testscenario.WithDisplayName("Alpha"), testscenario.WithDescription("legacy"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	testprocess.WriteScenarioProcessRecord(t, home, "alpha", "start-api", process.Record{PID: os.Getpid(), PGID: os.Getpid(), Scenario: "alpha", Step: "start-api", Port: 18080, StartedAt: time.Now().Add(-2 * time.Minute).UTC(), Status: "running"})
	writeRegistryRuntimeWithBoot(t, home, "alpha", "previous-boot", scenarioruntime.StatusRunning, scenarioruntime.ClaimStatusBound, "api", "API_PORT", 18081)

	service := New(root, home, io.Discard, io.Discard)
	service.hostSession = func(context.Context, string) (hostsession.Snapshot, error) {
		return hostsession.Snapshot{BootID: "current-boot", SessionID: "current-boot"}, nil
	}
	resolved, err := service.ResolvePort("alpha", "API_PORT")
	if err != nil {
		t.Fatalf("ResolvePort: %v", err)
	}
	if resolved.Port != 18080 {
		t.Fatalf("resolved.Port = %d, want legacy fallback 18080", resolved.Port)
	}
}

func TestStrictRegistryModeFailsClosedForPreviousBootClaim(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv(scenarioruntime.ModeEnv, scenarioruntime.ModeStrict)
	testscenario.WriteScenarioService(t, root, "workspace-sandbox", testscenario.ScenarioServiceManifest("workspace-sandbox", testscenario.WithDisplayName("Workspace Sandbox"), testscenario.WithDescription("registry"), testscenario.WithPorts(map[string]scenario.Port{
		"api": {EnvVar: "API_PORT", Range: "28080-28090"},
		"ws":  {EnvVar: "WS_PORT", Range: "28830-28840"},
	})))
	deadPID := 999999999
	writeRegistryRuntimeWithBoot(t, home, "workspace-sandbox", "previous-boot", scenarioruntime.StatusRunning, scenarioruntime.ClaimStatusBound, "ws", "WS_PORT", 28836)
	addRegistryProcessRef(t, home, "inst-workspace-sandbox", deadPID, "previous-boot")

	service := New(root, home, io.Discard, io.Discard)
	service.hostSession = func(context.Context, string) (hostsession.Snapshot, error) {
		return hostsession.Snapshot{BootID: "current-boot", SessionID: "current-boot"}, nil
	}
	status, exists, err := service.Status("workspace-sandbox")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !exists {
		t.Fatal("expected scenario to exist")
	}
	if status.Status == "running" || len(status.Ports) != 0 {
		t.Fatalf("status = %+v, want fail-closed stopped/empty", status)
	}
	_, err = service.ResolvePort("workspace-sandbox", "WS_PORT")
	if got := vroolierr.Code(err, ""); got != "scenario_not_running" {
		t.Fatalf("ResolvePort error code = %q, want scenario_not_running; err=%v", got, err)
	}
}

func TestRegistryAllowlistScopesPreferModeReads(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv(scenarioruntime.ModeEnv, scenarioruntime.ModePrefer)
	t.Setenv(scenarioruntime.AllowlistEnv, "alpha")
	for _, name := range []string{"alpha", "beta"} {
		testscenario.WriteScenarioService(t, root, name, testscenario.ScenarioServiceManifest(name, testscenario.WithDisplayName(name), testscenario.WithDescription("registry"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	}
	writeRegistryRuntime(t, home, "alpha", scenarioruntime.StatusRunning, scenarioruntime.ClaimStatusBound, "api", "API_PORT", 18081)
	writeRegistryRuntime(t, home, "beta", scenarioruntime.StatusRunning, scenarioruntime.ClaimStatusBound, "api", "API_PORT", 18082)

	service := New(root, home, io.Discard, io.Discard)
	alpha, exists, err := service.Status("alpha")
	if err != nil {
		t.Fatalf("Status(alpha): %v", err)
	}
	if !exists || alpha.Status != "running" || alpha.Ports["API_PORT"] != 18081 {
		t.Fatalf("alpha status = %+v, want allowlisted registry runtime", alpha)
	}
	beta, exists, err := service.Status("beta")
	if err != nil {
		t.Fatalf("Status(beta): %v", err)
	}
	if !exists || beta.Status == "running" || len(beta.Ports) != 0 {
		t.Fatalf("beta status = %+v, want non-allowlisted legacy/off behavior", beta)
	}
}

func TestRegistryAllowlistScopesStrictModeReads(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv(scenarioruntime.ModeEnv, scenarioruntime.ModeStrict)
	t.Setenv(scenarioruntime.AllowlistEnv, "alpha")
	for _, name := range []string{"alpha", "beta"} {
		testscenario.WriteScenarioService(t, root, name, testscenario.ScenarioServiceManifest(name, testscenario.WithDisplayName(name), testscenario.WithDescription("mixed"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	}
	writeRegistryRuntime(t, home, "alpha", scenarioruntime.StatusRunning, scenarioruntime.ClaimStatusBound, "api", "API_PORT", 18081)
	testprocess.WriteScenarioProcessRecord(t, home, "beta", "start-api", process.Record{PID: os.Getpid(), PGID: os.Getpid(), Scenario: "beta", Step: "start-api", Port: 18082, StartedAt: time.Now().Add(-2 * time.Minute).UTC(), Status: "running"})

	service := New(root, home, io.Discard, io.Discard)
	alpha, exists, err := service.Status("alpha")
	if err != nil {
		t.Fatalf("Status(alpha): %v", err)
	}
	if !exists || alpha.Status != "running" || alpha.Ports["API_PORT"] != 18081 {
		t.Fatalf("alpha status = %+v, want allowlisted strict registry runtime", alpha)
	}
	beta, exists, err := service.Status("beta")
	if err != nil {
		t.Fatalf("Status(beta): %v", err)
	}
	if !exists || beta.Status != "running" || beta.Ports["API_PORT"] != 18082 {
		t.Fatalf("beta status = %+v, want non-allowlisted legacy runtime", beta)
	}
}

func TestLatestRuntimeInstanceDoesNotDependOnStoreOrdering(t *testing.T) {
	base := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	instances := []scenarioruntime.Instance{
		{InstanceID: "inst-old", Scenario: "alpha", Generation: 1, UpdatedAt: base.Add(2 * time.Minute)},
		{InstanceID: "inst-new", Scenario: "alpha", Generation: 2, UpdatedAt: base},
		{InstanceID: "inst-same-generation-newer", Scenario: "alpha", Generation: 2, UpdatedAt: base.Add(time.Minute)},
	}

	got := latestRuntimeInstance(instances)
	if got.InstanceID != "inst-same-generation-newer" {
		t.Fatalf("latestRuntimeInstance = %q, want inst-same-generation-newer", got.InstanceID)
	}
}

func TestResolvePortRejectsStoppedScenario(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest("alpha", testscenario.WithDisplayName("Alpha"), testscenario.WithDescription("stopped"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))

	service := New(root, home, io.Discard, io.Discard)
	_, err := service.ResolvePort("alpha", "API_PORT")
	if err == nil {
		t.Fatal("expected stopped scenario error")
	}
	if got := vroolierr.Code(err, ""); got != "scenario_not_running" {
		t.Fatalf("error code = %q, want scenario_not_running", got)
	}
}

func TestInventoryIgnoresRuntimeRecordsForUnknownScenarios(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest("alpha", testscenario.WithDisplayName("Alpha"), testscenario.WithDescription("running"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	testprocess.WriteScenarioProcessRecord(t, home, "alpha", "start-api", process.Record{PID: os.Getpid(), PGID: os.Getpid(), Scenario: "alpha", Step: "start-api", Port: 18080, StartedAt: time.Now().Add(-2 * time.Minute).UTC(), Status: "running"})
	testprocess.WriteScenarioProcessRecord(t, home, "ghost", "start-api", process.Record{PID: os.Getpid(), PGID: os.Getpid(), Scenario: "ghost", Step: "start-api", Port: 28080, StartedAt: time.Now().Add(-2 * time.Minute).UTC(), Status: "running"})

	service := New(root, home, io.Discard, io.Discard)
	inventory, err := service.Inventory()
	if err != nil {
		t.Fatalf("Inventory: %v", err)
	}
	if len(inventory) != 1 || inventory[0].Scenario.Slug != "alpha" {
		t.Fatalf("inventory = %#v", inventory)
	}
}

func TestDetailRejectsBrokenProcessMetadata(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest("alpha", testscenario.WithDisplayName("Alpha"), testscenario.WithDescription("running"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	brokenPath := filepath.Join(home, ".vrooli", "processes", "scenarios", "alpha", "broken.json")
	if err := os.MkdirAll(filepath.Dir(brokenPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(brokenPath), err)
	}
	if err := os.WriteFile(brokenPath, []byte("{broken\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", brokenPath, err)
	}

	service := New(root, home, io.Discard, io.Discard)
	if _, err := service.Detail("alpha"); err == nil {
		t.Fatal("expected invalid process metadata to fail detail loading")
	}
}

func TestStartDetailedUsesInjectedRunnerFactory(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest("alpha", testscenario.WithDisplayName("Alpha"), testscenario.WithDescription("running"), testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}})))
	testprocess.WriteScenarioProcessRecord(t, home, "alpha", "start-api", process.Record{PID: os.Getpid(), PGID: os.Getpid(), Scenario: "alpha", Step: "start-api", Port: 18080, StartedAt: time.Now().Add(-2 * time.Minute).UTC(), Status: "running"})

	service := New(root, home, io.Discard, io.Discard)
	service.newRunner = func(root, home string, stdout, stderr io.Writer, logger ...*slog.Logger) (lifecycleRunner, error) {
		return fakeLifecycleRunner{
			startFn: func(name string, opts lifecycle.StartOptions) (lifecycle.Result, error) {
				return lifecycle.Result{
					Scenario: scenarioFixture(name, filepath.Join(root, "scenarios", name)),
					AllocatedPorts: map[string]int{
						"API_PORT": 18080,
					},
					Health: "healthy",
				}, nil
			},
		}, nil
	}

	result, err := service.StartDetailed("alpha", lifecycle.StartOptions{})
	if err != nil {
		t.Fatalf("StartDetailed: %v", err)
	}
	if result.View.Name != "alpha" {
		t.Fatalf("view name = %q", result.View.Name)
	}
}

type fakeLifecycleRunner struct {
	startFn   func(name string, opts lifecycle.StartOptions) (lifecycle.Result, error)
	restartFn func(name string, opts lifecycle.StartOptions) (lifecycle.Result, error)
	stopFn    func(name string, opts lifecycle.StopOptions) error
}

func (f fakeLifecycleRunner) Start(name string, opts lifecycle.StartOptions) (lifecycle.Result, error) {
	if f.startFn == nil {
		return lifecycle.Result{}, nil
	}
	return f.startFn(name, opts)
}

func (f fakeLifecycleRunner) Restart(name string, opts lifecycle.StartOptions) (lifecycle.Result, error) {
	if f.restartFn == nil {
		return lifecycle.Result{}, nil
	}
	return f.restartFn(name, opts)
}

func (f fakeLifecycleRunner) Stop(name string, opts lifecycle.StopOptions) error {
	if f.stopFn == nil {
		return nil
	}
	return f.stopFn(name, opts)
}

func scenarioFixture(name, path string) scenario.Scenario {
	return scenario.Scenario{
		Slug: name,
		Path: path,
		Manifest: scenario.ServiceManifest{
			Service: scenario.ServiceMetadata{
				Name: name,
			},
			Ports: map[string]scenario.Port{
				"api": {EnvVar: "API_PORT"},
			},
		},
	}
}

func writeRegistryRuntime(t *testing.T, home, scenarioName, instanceStatus, claimStatus, portName, envVar string, port int) {
	t.Helper()
	host, err := hostsession.DefaultProvider{}.Current(context.Background(), home)
	if err != nil {
		t.Fatalf("host session: %v", err)
	}
	writeRegistryRuntimeWithBoot(t, home, scenarioName, host.BootID, instanceStatus, claimStatus, portName, envVar, port)
}

func writeRegistryRuntimeWithBoot(t *testing.T, home, scenarioName, bootID, instanceStatus, claimStatus, portName, envVar string, port int) {
	t.Helper()
	ctx := context.Background()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()

	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID: "inst-" + scenarioName,
		Scenario:   scenarioName,
		Status:     scenarioruntime.StatusStarting,
		Phase:      "develop",
		OwnerKind:  scenarioruntime.OwnerKindLifecycle,
		StartedAt:  time.Now().Add(-time.Minute).UTC(),
		WorkingDir: filepath.Join(home, "scenarios", scenarioName),
		HostBootID: bootID,
	}, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	if instanceStatus != scenarioruntime.StatusStarting {
		instance, err = store.UpdateInstanceStatus(ctx, instance.InstanceID, instance.Generation, instanceStatus, "develop")
		if err != nil {
			t.Fatalf("UpdateInstanceStatus: %v", err)
		}
	}
	claim, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-" + scenarioName + "-" + portName,
		InstanceID: instance.InstanceID,
		Scenario:   scenarioName,
		PortName:   portName,
		EnvVar:     envVar,
		Port:       port,
		BindHost:   "127.0.0.1",
		Status:     scenarioruntime.ClaimStatusReserved,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}
	if claimStatus == scenarioruntime.ClaimStatusBound {
		if _, err := store.BindPortClaim(ctx, claim.ClaimID); err != nil {
			t.Fatalf("BindPortClaim: %v", err)
		}
	}
}

func addRegistryProcessRef(t *testing.T, home, instanceID string, pid int, bootID string) {
	t.Helper()
	ctx := context.Background()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()
	if _, err := store.AddProcessRef(ctx, scenarioruntime.ProcessRef{
		RefID:      "proc-" + instanceID,
		InstanceID: instanceID,
		PID:        &pid,
		Step:       "start-api",
		Status:     "running",
		StartedAt:  time.Now().Add(-time.Minute).UTC(),
		HostBootID: bootID,
	}); err != nil {
		t.Fatalf("AddProcessRef: %v", err)
	}
}
