package validation

import (
	"context"
	"path/filepath"
	"testing"

	"unit-health/internal/discovery"
	"unit-health/internal/executor"
)

// fakeExecutorByName keys canned results by command Name so install and test
// steps of the same workspace can be driven independently.
type fakeExecutorByName struct {
	byName map[string]executor.Result
}

func (f fakeExecutorByName) Run(_ context.Context, cmd executor.Command) executor.Result {
	r, ok := f.byName[cmd.Name]
	if !ok {
		r = executor.Result{Status: executor.StatusPassed}
	}
	r.WorkspaceID = cmd.WorkspaceID
	r.Name = cmd.Name
	return r
}

func withInstallResolver(t *testing.T, bin string, ok bool) {
	t.Helper()
	orig := installBinaryResolver
	t.Cleanup(func() { installBinaryResolver = orig })
	installBinaryResolver = func([]string) (string, bool) { return bin, ok }
}

func TestNodeInstallCommand(t *testing.T) {
	cases := []struct {
		bin  string
		ok   bool
		want string
		got  bool
	}{
		{"pnpm", true, "pnpm install --frozen-lockfile --ignore-scripts", true},
		{"yarn", true, "yarn install --frozen-lockfile --ignore-scripts", true},
		{"npm", true, "npm ci --ignore-scripts", true},
		{"", false, "", false},
	}
	for _, c := range cases {
		withInstallResolver(t, c.bin, c.ok)
		got, ok := nodeInstallCommand("pnpm")
		if ok != c.got || got != c.want {
			t.Errorf("nodeInstallCommand(bin=%q) = %q,%v; want %q,%v", c.bin, got, ok, c.want, c.got)
		}
	}
}

func TestInstallCandidateOrderDedupAndPreference(t *testing.T) {
	got := installCandidateOrder("yarn")
	want := []string{"yarn", "pnpm", "npm"}
	if len(got) != len(want) {
		t.Fatalf("order = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v; want %v", got, want)
		}
	}
	// Unknown preferred is ignored; canonical chain still present.
	if got := installCandidateOrder("bun"); got[0] != "pnpm" {
		t.Errorf("unknown preferred not ignored: %v", got)
	}
}

func TestBuildExecutionPlanEmitsInstallBeforeTest(t *testing.T) {
	ws := []Workspace{{
		ID:              "ui",
		Language:        "typescript",
		RootPath:        "/x/ui",
		InstallCommand:  "pnpm install --frozen-lockfile --ignore-scripts",
		TestCommand:     "pnpm test",
		CoverageCommand: "pnpm test:coverage",
	}}
	plan := buildExecutionPlan(ws)
	if len(plan.Commands) != 2 {
		t.Fatalf("commands = %+v", plan.Commands)
	}
	if plan.Commands[0].Kind != kindInstall || plan.Commands[1].Kind != kindTest {
		t.Fatalf("kinds = %q,%q; want install,test", plan.Commands[0].Kind, plan.Commands[1].Kind)
	}
	if plan.Commands[1].Command != "pnpm test:coverage" {
		t.Errorf("test command = %q; want coverage command", plan.Commands[1].Command)
	}
}

// nodeInventory builds a discovery inventory with one ready Vitest UI surface.
func nodeInventory(t *testing.T) discovery.Inventory {
	t.Helper()
	root := t.TempDir()
	uiRoot := filepath.Join(root, "ui")
	writeFile(t, filepath.Join(uiRoot, "package.json"), `{
  "name": "ui",
  "scripts": {"test": "vitest run", "test:coverage": "vitest run --coverage"},
  "devDependencies": {"vitest": "^1.0.0"}
}`)
	return discovery.Inventory{
		Scenario: "demo", TargetKind: "scenario", RootPath: root,
		Surfaces: []discovery.Surface{{ID: "ui", Kind: "ui", Language: "typescript", RootPath: uiRoot, PackageManager: "pnpm", Status: "known"}},
	}
}

func TestExecuteInstallFailureSkipsTestAndFlagsDependency(t *testing.T) {
	withInstallResolver(t, "pnpm", true)
	spec := loadSpec(t)
	svc := newService(fakeDiscoverer{inv: nodeInventory(t)}, spec)
	svc.Executor = fakeExecutorByName{byName: map[string]executor.Result{
		"typescript install": {Status: executor.StatusFailed, FailureClass: executor.ClassMissingDependency, ExitCode: 1, Stderr: "ERR_PNPM_NO_LOCKFILE"},
	}}

	resp, err := svc.Validate(context.Background(), Request{Scenario: "demo", IncludeExecution: true})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !hasFinding(resp.Findings, codeTestDependencyMissing) {
		t.Errorf("expected %s finding, got %v", codeTestDependencyMissing, codes(resp.Findings))
	}
	if hasFinding(resp.Findings, codeTestExecutionFailure) {
		t.Errorf("test should be skipped, not run, on install failure: %v", codes(resp.Findings))
	}
	var sawSkip bool
	for _, cr := range resp.CommandResults {
		if cr.Name == "typescript test" && cr.Status == statusSkipped {
			sawSkip = true
		}
	}
	if !sawSkip {
		t.Errorf("expected the test command skipped; results=%+v", resp.CommandResults)
	}
}

func TestExecuteInstallSuccessRunsTest(t *testing.T) {
	withInstallResolver(t, "pnpm", true)
	spec := loadSpec(t)
	svc := newService(fakeDiscoverer{inv: nodeInventory(t)}, spec)
	svc.Executor = fakeExecutorByName{byName: map[string]executor.Result{
		"typescript install": {Status: executor.StatusPassed},
		"typescript test":    {Status: executor.StatusPassed},
	}}

	resp, err := svc.Validate(context.Background(), Request{Scenario: "demo", IncludeExecution: true})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if hasFinding(resp.Findings, codeTestDependencyMissing) {
		t.Errorf("unexpected dependency finding on clean install: %v", codes(resp.Findings))
	}
	var install, test bool
	for _, cr := range resp.CommandResults {
		switch cr.Name {
		case "typescript install":
			install = cr.Status == "passed"
		case "typescript test":
			test = cr.Status == "passed"
		}
	}
	if !install || !test {
		t.Errorf("expected both install and test to run and pass; results=%+v", resp.CommandResults)
	}
}
