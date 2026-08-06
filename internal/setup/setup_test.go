package setup

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/dockerhost"
	"github.com/vrooli/vrooli/internal/hostreq"
	hostreqspec "github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/projectstate"
	"github.com/vrooli/vrooli/internal/resources"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=4 | LAST: 2026-04-16

func TestMarkCompleteWritesSetupMarker(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := markComplete(home, root); err != nil {
		t.Fatalf("markComplete: %v", err)
	}

	locator, err := projectstate.NewLocator(home, root)
	if err != nil {
		t.Fatalf("NewLocator: %v", err)
	}
	setupMarker := locator.SetupCompletePath()
	payload := testkitgo.ReadJSONFile(t, setupMarker)
	if payload["setup_version"] != "2.0.0" {
		t.Fatalf("setup_version = %v", payload["setup_version"])
	}
	if payload["project_key"] != locator.ProjectKey() {
		t.Fatalf("project_key = %v, want %q", payload["project_key"], locator.ProjectKey())
	}
	if _, err := os.Stat(locator.ResourcesPopulatedPath()); !os.IsNotExist(err) {
		t.Fatalf("expected no resources marker, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".vrooli", "state")); !os.IsNotExist(err) {
		t.Fatalf("expected no repo-local setup state, got %v", err)
	}
}

func TestEnsureBootstrapPackageManagerInstallsHomebrewOnFreshDarwin(t *testing.T) {
	brewReady := false
	lookPath := func(name string) (string, error) {
		switch name {
		case "curl":
			return "/usr/bin/curl", nil
		case "brew":
			if brewReady {
				return "/opt/homebrew/bin/brew", nil
			}
		}
		return "", exec.ErrNotFound
	}
	var calls []shell.Spec
	run := func(spec shell.Spec) error {
		calls = append(calls, spec)
		switch spec.Name {
		case "curl":
			for index, arg := range spec.Args {
				if arg == "-o" && index+1 < len(spec.Args) {
					return os.WriteFile(spec.Args[index+1], []byte("#!/bin/bash\n"), 0o700)
				}
			}
			t.Fatal("curl did not receive an output path")
		case "/bin/bash":
			brewReady = true
			if !strings.Contains(strings.Join(spec.Env, "\n"), "NONINTERACTIVE=1") {
				t.Fatal("Homebrew installer was not made non-interactive")
			}
		}
		return nil
	}
	err := ensureBootstrapPackageManager(
		vrooliruntime.Host{OS: "darwin"}, t.TempDir(), vrooliruntime.EnsureOptions{}, lookPath, run,
	)
	if err != nil {
		t.Fatalf("ensureBootstrapPackageManager: %v", err)
	}
	if len(calls) != 2 || calls[0].Name != "curl" || calls[1].Name != "/bin/bash" {
		t.Fatalf("calls = %#v, want curl then /bin/bash", calls)
	}
}

func TestEnsureBootstrapPackageManagerLeavesLinuxPackageManagerAlone(t *testing.T) {
	calls := 0
	err := ensureBootstrapPackageManager(
		vrooliruntime.Host{OS: "linux", PackageManager: "apt-get"}, t.TempDir(), vrooliruntime.EnsureOptions{},
		func(string) (string, error) { calls++; return "", exec.ErrNotFound },
		func(shell.Spec) error { calls++; return nil },
	)
	if err != nil {
		t.Fatalf("ensureBootstrapPackageManager: %v", err)
	}
	if calls != 0 {
		t.Fatalf("linux package-manager bootstrap made %d calls, want none", calls)
	}
}

func TestRepoMaintainsCanonicalInstallContract(t *testing.T) {
	repoRoot := testkitgo.ProjectRoot(t)

	makefileData, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	makefileContents := string(makefileData)
	if !strings.Contains(makefileContents, "install: build") {
		t.Fatalf("Makefile missing install target contract")
	}
	if !strings.Contains(makefileContents, "INSTALL_DIR = $(HOME)/.vrooli/bin") &&
		!strings.Contains(makefileContents, "INSTALL_DIR := $(HOME)/.vrooli/bin") {
		t.Fatalf("Makefile missing canonical install dir")
	}
}

func TestRunSetupUsesNativeRuntimeAndMarksComplete(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	svc.deps.currentHost = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	svc.deps.loadProject = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	svc.deps.resolveHostRequirements = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	svc.deps.inspectRequirements = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}

	sequence := []string{}
	svc.deps.ensureBootstrapTools = func(gotHome string, opts vrooliruntime.EnsureOptions) error {
		sequence = append(sequence, "bootstrap")
		if gotHome != home {
			t.Fatalf("bootstrap home = %q, want %q", gotHome, home)
		}
		return nil
	}
	runtimeCalls := 0
	svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		sequence = append(sequence, "requirements")
		runtimeCalls++
		if opts.DryRun {
			t.Fatal("expected non-dry-run setup to execute real install/apply path")
		}
		if !opts.AutoInstall {
			t.Fatal("expected AutoInstall to remain enabled during setup execution")
		}
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}
	markCompleteCalled := false
	svc.deps.markComplete = func(gotHome, gotRoot string) error {
		markCompleteCalled = true
		if gotHome != home || gotRoot != root {
			t.Fatalf("markComplete inputs = (%q, %q), want (%q, %q)", gotHome, gotRoot, home, root)
		}
		return nil
	}
	schemaSyncCalled := false
	svc.deps.syncResourceSchema = func(root string) error {
		schemaSyncCalled = true
		return nil
	}
	manager := &stubCLIInstallManager{}
	svc.deps.newCLIInstallManager = func(root, home string) (cliInstallManager, error) {
		manager.root = root
		manager.home = home
		return manager, nil
	}

	if err := svc.RunSetupWithOptions(root, home, Options{Resources: "none"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}
	if runtimeCalls != 1 {
		t.Fatalf("ensureRequirements calls = %d, want 1", runtimeCalls)
	}
	if got := strings.Join(sequence, ","); got != "bootstrap,requirements" {
		t.Fatalf("setup order = %q, want bootstrap,requirements", got)
	}
	if !markCompleteCalled {
		t.Fatal("expected markCompleteFn to be called")
	}
	if !schemaSyncCalled {
		t.Fatal("expected syncResourceSchemaFn to be called")
	}
	if manager.installEnabledResourceCalls != 0 {
		t.Fatalf("InstallEnabledResourceCLIs calls = %d, want 0 for resources=none", manager.installEnabledResourceCalls)
	}
	if manager.installAllScenarioCalls != 0 {
		t.Fatalf("InstallAllScenarioCLIs calls = %d, want 0 for scenarios=none", manager.installAllScenarioCalls)
	}
	if manager.root != root || manager.home != home {
		t.Fatalf("manager inputs = (%q, %q), want (%q, %q)", manager.root, manager.home, root, home)
	}
	if _, err := os.Stat(filepath.Join(root, "data")); err != nil {
		t.Fatalf("expected data dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".vrooli", "logs")); !os.IsNotExist(err) {
		t.Fatalf("expected no repo-local logs dir, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".vrooli", "state")); !os.IsNotExist(err) {
		t.Fatalf("expected no repo-local state dir, got %v", err)
	}
	locator, err := projectstate.NewLocator(home, root)
	if err != nil {
		t.Fatalf("NewLocator: %v", err)
	}
	if _, err := os.Stat(locator.SetupStateDir()); err != nil {
		t.Fatalf("expected user-home setup state dir: %v", err)
	}
}

func TestInstallSelectedCLIsHonorsExplicitSelections(t *testing.T) {
	manager := &stubCLIInstallManager{}
	if err := installSelectedCLIs(manager, " postgres,redis ", "alpha, beta"); err != nil {
		t.Fatalf("installSelectedCLIs: %v", err)
	}
	if got := strings.Join(manager.installedResources, ","); got != "postgres,redis" {
		t.Fatalf("installed resources = %q", got)
	}
	if got := strings.Join(manager.installedScenarios, ","); got != "alpha,beta" {
		t.Fatalf("installed scenarios = %q", got)
	}
	if manager.installEnabledResourceCalls != 0 || manager.installAllScenarioCalls != 0 {
		t.Fatalf("bulk installers unexpectedly called: %#v", manager)
	}
}

func TestInstallSelectedCLIsPreservesEnabledAndAllSelectors(t *testing.T) {
	manager := &stubCLIInstallManager{}
	if err := installSelectedCLIs(manager, "enabled", "all"); err != nil {
		t.Fatalf("installSelectedCLIs: %v", err)
	}
	if manager.installEnabledResourceCalls != 1 || manager.installAllScenarioCalls != 1 {
		t.Fatalf("bulk installer calls = resources:%d scenarios:%d", manager.installEnabledResourceCalls, manager.installAllScenarioCalls)
	}
}

func TestBootstrapAwareRequirementsRequiresGitAndGoButNotDocker(t *testing.T) {
	resolution := bootstrapAwareRequirements(hostreq.Resolution{Tools: []hostreq.ResolvedRequirement{
		{Name: "docker", Kind: hostreq.KindTool, Required: true},
		{Name: "tmux", Kind: hostreq.KindTool, Required: true},
		{Name: "go", Kind: hostreq.KindTool, Required: false, Environments: []string{"development"}},
	}})
	if got := len(resolution.Tools); got != 3 {
		t.Fatalf("tool count = %d, want git/go/tmux", got)
	}
	if resolution.Tools[0].Name != "git" || resolution.Tools[1].Name != "go" {
		t.Fatalf("bootstrap tools are not first: %#v", resolution.Tools)
	}
	for _, name := range []string{"git", "go"} {
		item := findResolvedRequirement(resolution.Tools, name)
		if item == nil || !item.Required {
			t.Fatalf("%s is not required: %#v", name, item)
		}
		if got := strings.Join(item.Environments, ","); got != "development,production,minimal" {
			t.Fatalf("%s environments = %q", name, got)
		}
	}
	if findResolvedRequirement(resolution.Tools, "docker") != nil {
		t.Fatal("Docker remained a global bootstrap requirement")
	}
}

func TestBootstrapAwareRequirementsOrdersRasdaemonBeforeMcelog(t *testing.T) {
	resolution := bootstrapAwareRequirements(hostreq.Resolution{Tools: []hostreq.ResolvedRequirement{
		{Name: "mcelog", Kind: hostreq.KindTool, Required: true},
		{Name: "rasdaemon", Kind: hostreq.KindTool, Required: true},
	}})

	rasdaemonIndex := -1
	mcelogIndex := -1
	for i, requirement := range resolution.Tools {
		switch requirement.Name {
		case "rasdaemon":
			rasdaemonIndex = i
		case "mcelog":
			mcelogIndex = i
		}
	}
	if rasdaemonIndex == -1 || mcelogIndex == -1 {
		t.Fatalf("requirements missing from ordered result: %#v", resolution.Tools)
	}
	if rasdaemonIndex >= mcelogIndex {
		t.Fatalf("rasdaemon index %d must precede mcelog index %d: %#v", rasdaemonIndex, mcelogIndex, resolution.Tools)
	}
}

func TestRecoverHostToolPATHFindsOffPathGo(t *testing.T) {
	home := t.TempDir()
	goBin := filepath.Join(home, "go", "bin")
	if err := os.MkdirAll(goBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goBin, "go"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	recoverHostToolPATH(home)
	resolved, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("go remains off PATH: %v", err)
	}
	if resolved == "" {
		t.Fatal("go resolved to an empty path")
	}
	if !strings.Contains(os.Getenv("PATH"), goBin) {
		t.Fatalf("recovered PATH does not include %q: %s", goBin, os.Getenv("PATH"))
	}
}

func TestRunSetupTriggersOnboardingAfterSuccessfulSetup(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	svc.deps.currentHost = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	svc.deps.loadProject = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	svc.deps.resolveHostRequirements = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	svc.deps.inspectRequirements = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}
	svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}

	markCompleteCalled := false
	svc.deps.markComplete = func(gotHome, gotRoot string) error {
		markCompleteCalled = true
		if gotHome != home || gotRoot != root {
			t.Fatalf("markComplete inputs = (%q, %q), want (%q, %q)", gotHome, gotRoot, home, root)
		}
		return nil
	}

	onboardingCalls := 0
	svc.deps.openOnboardingURL = func(url string) error {
		onboardingCalls++
		return nil
	}
	svc.deps.osExecutable = func() (string, error) { return "/bin/true", nil }
	svc.deps.onboardingPortCommandRunner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("41234\n"), nil
	}
	writeOnboardingScenarioFixture(t, root)

	if err := svc.RunSetupWithOptions(root, home, Options{Resources: "none"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}
	if !markCompleteCalled {
		t.Fatal("expected markCompleteFn to be called")
	}
	if onboardingCalls != 1 {
		t.Fatalf("onboarding calls = %d, want 1", onboardingCalls)
	}
}

func TestRunSetupDryRunUsesApplyPlanningAndSkipsMutations(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	svc.deps.currentHost = func() vrooliruntime.Host {
		return vrooliruntime.Host{OS: "linux", PackageManager: "apt-get", SupportsSetup: true, SupportsDevelop: true}
	}
	svc.deps.loadProject = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	svc.deps.resolveHostRequirements = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{
			Tools: []hostreq.ResolvedRequirement{
				{Name: "tmux", Kind: hostreq.KindTool, Required: true},
			},
		}, nil
	}
	svc.deps.inspectRequirements = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return reportFromResolution(environment, resolution, false), nil
	}

	ensureCalls := 0
	svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		ensureCalls++
		if !opts.DryRun {
			t.Fatal("expected dry-run setup to preserve DryRun=true")
		}
		if !opts.AutoInstall {
			t.Fatal("expected dry-run setup to use runtime apply planning")
		}
		return reportFromResolution(opts.Environment, resolution, true), nil
	}

	markCompleteCalled := false
	svc.deps.markComplete = func(string, string) error {
		markCompleteCalled = true
		return nil
	}

	resourceInstallCalls := 0
	svc.deps.resourceController = func(root, home string) resourceRunner {
		return resourceRunnerFunc(func(name string, args []string, stdout, stderr io.Writer) error {
			resourceInstallCalls++
			return nil
		})
	}

	stdout := &strings.Builder{}
	if err := svc.RunSetupWithOptions(root, home, Options{DryRun: true, Resources: "redis", Scenarios: "alpha"}, stdout, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}
	if ensureCalls != 1 {
		t.Fatalf("ensureRequirements calls = %d, want 1", ensureCalls)
	}
	if markCompleteCalled {
		t.Fatal("did not expect markCompleteFn during dry-run")
	}
	if resourceInstallCalls != 0 {
		t.Fatalf("resource install calls = %d, want 0", resourceInstallCalls)
	}
	if _, err := os.Stat(filepath.Join(root, "data")); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run to avoid creating data dir, err=%v", err)
	}
	if !strings.Contains(stdout.String(), "Needs operator input") {
		t.Fatalf("stdout missing pending group:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "tmux") {
		t.Fatalf("stdout missing tmux line:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Dry-run mode skips git configuration, resource installation, and setup completion markers") {
		t.Fatalf("stdout missing dry-run skip note:\n%s", stdout.String())
	}
}

func TestRunSetupDryRunSkipsOnboarding(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	svc.deps.currentHost = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	svc.deps.loadProject = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	svc.deps.resolveHostRequirements = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	svc.deps.inspectRequirements = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}
	svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}

	onboardingCalls := 0
	svc.deps.openOnboardingURL = func(url string) error {
		onboardingCalls++
		return nil
	}
	svc.deps.osExecutable = func() (string, error) { return "/bin/true", nil }
	svc.deps.onboardingPortCommandRunner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("41234\n"), nil
	}
	writeOnboardingScenarioFixture(t, root)

	if err := svc.RunSetupWithOptions(root, home, Options{DryRun: true, Resources: "none"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}
	if onboardingCalls != 0 {
		t.Fatalf("onboarding calls = %d, want 0", onboardingCalls)
	}
}

func TestRunSetupPassesScenarioSelectionToResolver(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	svc.deps.currentHost = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	svc.deps.loadProject = func(root string) (scenario.Scenario, error) { return projectScenario, nil }

	var captured hostreq.ResolveOptions
	svc.deps.resolveHostRequirements = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		captured = opts
		return hostreq.Resolution{}, nil
	}
	svc.deps.inspectRequirements = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}
	svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}
	svc.deps.markComplete = func(string, string) error { return nil }

	if err := svc.RunSetupWithOptions(root, home, Options{Scenarios: "alpha,beta", Resources: "none", DryRun: true}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}
	if captured.Scenarios != "alpha,beta" {
		t.Fatalf("captured.Scenarios = %q", captured.Scenarios)
	}
}

func TestRunSetupDryRunPrintsSingleGroupedResult(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	svc.deps.currentHost = func() vrooliruntime.Host {
		return vrooliruntime.Host{OS: "linux", PackageManager: "apt-get", SupportsSetup: true, SupportsDevelop: true}
	}
	svc.deps.loadProject = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	svc.deps.resolveHostRequirements = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	svc.deps.inspectRequirements = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{
			Environment: environment,
			Host:        vrooliruntime.Host{OS: "linux", PackageManager: "apt-get"},
			Tools: []vrooliruntime.ToolStatus{
				{
					Name:           "git",
					Kind:           hostreq.KindTool,
					Required:       true,
					ExecutionState: vrooliruntime.ExecutionAlreadyPresent,
					Reasons:        []string{"repo operations"},
					Provenance: []hostreq.Provenance{
						{Kind: "root", Name: "vrooli", Source: ".vrooli/service.json"},
					},
				},
				{
					Name:           "tmux",
					Kind:           hostreq.KindTool,
					Required:       true,
					ExecutionState: vrooliruntime.ExecutionPending,
					Reasons:        []string{"scenario shell tooling"},
					Provenance: []hostreq.Provenance{
						{Kind: "scenario", Name: "alpha", Source: "scenarios/alpha/.vrooli/service.json"},
					},
				},
			},
			Safeguards: []vrooliruntime.SafeguardStatus{
				{
					Name:           "remote_session_protection",
					Kind:           hostreq.KindSafeguard,
					Required:       false,
					ExecutionState: vrooliruntime.ExecutionNotApplicable,
					Notes:          []string{"host does not expose sysctl hooks"},
					Provenance: []hostreq.Provenance{
						{Kind: "root", Name: "vrooli", Source: ".vrooli/service.json"},
					},
				},
			},
		}, nil
	}
	svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{
			Environment: opts.Environment,
			Host:        vrooliruntime.Host{OS: "linux", PackageManager: "apt-get"},
			Tools: []vrooliruntime.ToolStatus{
				{
					Name:           "git",
					Kind:           hostreq.KindTool,
					Required:       true,
					ExecutionState: vrooliruntime.ExecutionAlreadyPresent,
					Reasons:        []string{"repo operations"},
					Provenance: []hostreq.Provenance{
						{Kind: "root", Name: "vrooli", Source: ".vrooli/service.json"},
					},
				},
				{
					Name:           "tmux",
					Kind:           hostreq.KindTool,
					Required:       true,
					ExecutionState: vrooliruntime.ExecutionWouldInstall,
					Notes:          []string{"dry-run: would run apt-get install -y tmux"},
					Reasons:        []string{"scenario shell tooling"},
					Provenance: []hostreq.Provenance{
						{Kind: "scenario", Name: "alpha", Source: "scenarios/alpha/.vrooli/service.json"},
					},
				},
			},
			Safeguards: []vrooliruntime.SafeguardStatus{
				{
					Name:           "remote_session_protection",
					Kind:           hostreq.KindSafeguard,
					Required:       false,
					ExecutionState: vrooliruntime.ExecutionNotApplicable,
					Notes:          []string{"host does not expose sysctl hooks"},
					Provenance: []hostreq.Provenance{
						{Kind: "root", Name: "vrooli", Source: ".vrooli/service.json"},
					},
				},
			},
		}, nil
	}
	svc.deps.markComplete = func(string, string) error { return nil }

	stdout := &strings.Builder{}
	if err := svc.RunSetupWithOptions(root, home, Options{Resources: "none", Scenarios: "alpha", DryRun: true}, stdout, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}

	output := stdout.String()
	// We now render exactly one block (dry-run result), grouped.
	if strings.Contains(output, "Host requirements plan") {
		t.Fatalf("expected no plan block on dry-run; got:\n%s", output)
	}
	if strings.Count(output, "Host requirements") != 1 {
		t.Fatalf("expected one Host requirements heading, got %d:\n%s",
			strings.Count(output, "Host requirements"), output)
	}
	for _, expected := range []string{
		"Host requirements dry-run result",
		"Already present (1): git",
		"Needs operator input",
		"tmux",
		"Not applicable (1):",
		"remote_session_protection",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("stdout missing %q:\n%s", expected, output)
		}
	}
}

func TestRunSetupDryRunResolvesRootScenarioAndResourceDeclarations(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectManifest := testscenario.ProjectServiceManifest(
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{"redis": {Enabled: false}},
		}),
	)
	projectManifest.HostTools = []hostreqspec.Declaration{
		{Name: "git", Required: true, Reason: "repo operations"},
		{Name: "docker", Required: true, Reason: "container runtime"},
	}
	projectScenario := writeProjectFixtureWithServiceManifest(t, root, projectManifest)
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithLifecycle(scenario.Lifecycle{}),
	))
	alphaPath := scenario.ServicePath(root, "alpha")
	alphaManifest, err := scenario.ReadService(alphaPath)
	if err != nil {
		t.Fatalf("ReadService(%s): %v", alphaPath, err)
	}
	alphaManifest.HostTools = []hostreqspec.Declaration{
		{Name: "tmux", Required: true, Reason: "scenario shell tooling"},
	}
	alphaManifest.HostSafeguards = []hostreqspec.Declaration{
		{Name: "remote_session_protection", Required: true, Reason: "protect remote sessions"},
	}
	testkitgo.WriteJSON(t, alphaPath, alphaManifest)
	testresource.WriteResourceManifest(t, root, "redis", testresource.ResourceManifest(
		"redis",
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceBinary("redis-server"),
		testresource.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "partial",
			Windows: "unsupported",
		}),
		testresource.WithResourceHostTools(
			hostreqspec.Declaration{Name: "sqlite", Required: false, Reason: "resource cache introspection", Manual: true},
		),
	))

	svc.deps.currentHost = func() vrooliruntime.Host {
		return vrooliruntime.Host{OS: "linux", PackageManager: "apt-get", SupportsSetup: true, SupportsDevelop: true}
	}
	svc.deps.loadProject = func(root string) (scenario.Scenario, error) { return projectScenario, nil }

	var plannedResolution hostreq.Resolution
	svc.deps.inspectRequirements = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return reportFromResolution(environment, resolution, false), nil
	}
	svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		plannedResolution = resolution
		return reportFromResolution(opts.Environment, resolution, true), nil
	}

	stdout := &strings.Builder{}
	if err := svc.RunSetupWithOptions(root, home, Options{Resources: "redis", Scenarios: "alpha", DryRun: true}, stdout, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}

	if len(plannedResolution.Tools) != 4 {
		t.Fatalf("tool count = %d, want 4", len(plannedResolution.Tools))
	}
	if findResolvedRequirement(plannedResolution.Tools, "git") == nil {
		t.Fatal("expected root git declaration")
	}
	if findResolvedRequirement(plannedResolution.Tools, "docker") != nil {
		t.Fatal("did not expect Docker in global setup requirements")
	}
	if findResolvedRequirement(plannedResolution.Tools, "go") == nil {
		t.Fatal("expected bootstrap Go declaration")
	}
	if findResolvedRequirement(plannedResolution.Tools, "tmux") == nil {
		t.Fatal("expected scenario tmux declaration")
	}
	if findResolvedRequirement(plannedResolution.Tools, "sqlite") == nil {
		t.Fatal("expected resource sqlite declaration")
	}
	if findResolvedRequirement(plannedResolution.Safeguards, "remote_session_protection") == nil {
		t.Fatal("expected scenario safeguard declaration")
	}

	output := stdout.String()
	for _, expected := range []string{
		"Already present",
		"git",
		"go",
		"tmux",
		"remote_session_protection",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("stdout missing %q:\n%s", expected, output)
		}
	}
}

func TestRunSetupDryRunSkipsResourceInstallEvenWhenResourcesSelected(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	svc.deps.currentHost = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	svc.deps.loadProject = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	svc.deps.resolveHostRequirements = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	svc.deps.inspectRequirements = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}
	svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}

	resourceInstallCalls := 0
	svc.deps.resourceController = func(root, home string) resourceRunner {
		return resourceRunnerFunc(func(name string, args []string, stdout, stderr io.Writer) error {
			resourceInstallCalls++
			return nil
		})
	}

	if err := svc.RunSetupWithOptions(root, home, Options{Resources: "redis,postgres", DryRun: true}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}
	if resourceInstallCalls != 0 {
		t.Fatalf("resource install calls = %d, want 0", resourceInstallCalls)
	}
}

func TestDockerIsDemandedOnlyBySelectedContainerResources(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WriteResourceManifest(t, root, "native", testresource.ResourceManifest(
		"native",
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceBinary("native-server"),
	))
	testresource.WriteResourceManifest(t, root, "containerized", testresource.ResourceManifest(
		"containerized",
		testresource.WithResourceDriver("docker-service"),
		testresource.WithResourceComposeFile("docker-compose.yml"),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{Image: "fixture:1.0.0"}),
	))
	manifest, err := resources.NewController(root, home).ResourceManifest("containerized")
	if err != nil {
		t.Fatalf("load container fixture: %v", err)
	}
	if manifest.Driver != "docker-service" {
		t.Fatalf("container fixture driver = %q", manifest.Driver)
	}

	originalInspect := inspectDockerHealthFn
	t.Cleanup(func() { inspectDockerHealthFn = originalInspect })
	inspections := 0
	inspectDockerHealthFn = func() dockerhost.Health {
		inspections++
		return dockerhost.Health{Detail: "daemon unavailable"}
	}
	if err := preflightDockerResources(root, home, nil); err != nil {
		t.Fatalf("empty selection: %v", err)
	}
	if err := preflightDockerResources(root, home, []string{"native"}); err != nil {
		t.Fatalf("native resource: %v", err)
	}
	if inspections != 0 {
		t.Fatalf("Docker inspected %d times without a container resource", inspections)
	}
	err = preflightDockerResources(root, home, []string{"containerized"})
	if err == nil || !strings.Contains(err.Error(), "selected resources require Docker (containerized)") {
		t.Fatalf("container resource error = %v", err)
	}
	if inspections != 1 {
		t.Fatalf("Docker inspections = %d, want 1", inspections)
	}
}

func TestRunDevelopRunsSetupWhenNeededAndStartsNativeServices(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)
	testresource.WritePortRegistry(t, root, nil)
	testkitgo.WriteExecutable(t, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "#!/usr/bin/env bash\nexit 0\n")
	t.Setenv("VROOLI_API_PORT", "18096")
	t.Setenv("VROOLI_API_PORT", "18095")

	svc.deps.currentHost = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	svc.deps.loadProject = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	svc.deps.resolveHostRequirements = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	svc.deps.inspectRequirements = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}
	svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}

	setupCalls := 0
	svc.deps.markComplete = func(gotHome, gotRoot string) error {
		setupCalls++
		return writeSetupCompleteMarker(t, gotHome, gotRoot)
	}

	apiStarted := false
	svc.deps.startProjectAPI = func(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error {
		apiStarted = true
		if spec.Command == "" || spec.Port != 18095 {
			t.Fatalf("spec = %+v", spec)
		}
		return nil
	}
	healthCalls := 0
	svc.deps.healthCheck = func(port int, timeout time.Duration) error {
		healthCalls++
		if port != 18095 {
			t.Fatalf("port = %d", port)
		}
		return nil
	}
	orchestratorStarted := false
	svc.deps.startOrchestrator = func(root, home string, stdout, stderr io.Writer) error {
		orchestratorStarted = true
		return nil
	}

	stdout := &strings.Builder{}
	if err := svc.RunDevelopWithOptions(root, home, Options{}, stdout, io.Discard); err != nil {
		t.Fatalf("RunDevelopWithOptions: %v", err)
	}
	if setupCalls != 1 {
		t.Fatalf("setup calls = %d, want 1", setupCalls)
	}
	if !apiStarted {
		t.Fatal("expected project API to start")
	}
	if healthCalls != 1 {
		t.Fatalf("health calls = %d, want 1", healthCalls)
	}
	if !orchestratorStarted {
		t.Fatal("expected orchestrator startup")
	}
	if !strings.Contains(stdout.String(), "Running setup before develop") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestDevelopHealthTimeoutAllowsColdGoRun(t *testing.T) {
	if got := developHealthTimeout(apiLaunchSpec{Command: "go", Args: []string{"run", "./cmd/vrooli-api"}}); got != 2*time.Minute {
		t.Fatalf("cold go-run timeout = %s, want 2m", got)
	}
	if got := developHealthTimeout(apiLaunchSpec{Command: "/tmp/vrooli-api"}); got != 30*time.Second {
		t.Fatalf("prebuilt timeout = %s, want 30s", got)
	}
}

func TestRunDevelopSkipsSetupWhenMarkerExists(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)
	testresource.WritePortRegistry(t, root, nil)
	testkitgo.WriteExecutable(t, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "#!/usr/bin/env bash\nexit 0\n")
	if err := writeSetupCompleteMarker(t, home, root); err != nil {
		t.Fatalf("write setup marker: %v", err)
	}

	svc.deps.currentHost = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	svc.deps.loadProject = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	svc.deps.resolveHostRequirements = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	svc.deps.inspectRequirements = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}
	svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}

	setupCalls := 0
	svc.deps.markComplete = func(string, string) error {
		setupCalls++
		return nil
	}
	svc.deps.startProjectAPI = func(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error { return nil }
	svc.deps.healthCheck = func(port int, timeout time.Duration) error { return nil }
	svc.deps.startOrchestrator = func(root, home string, stdout, stderr io.Writer) error { return nil }

	if err := svc.RunDevelopWithOptions(root, home, Options{}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunDevelopWithOptions: %v", err)
	}
	if setupCalls != 0 {
		t.Fatalf("setup calls = %d, want 0", setupCalls)
	}
}

func TestRunDevelopSkipsOrchestratorWhenScenariosAreNone(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)
	testresource.WritePortRegistry(t, root, nil)
	testkitgo.WriteExecutable(t, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "#!/usr/bin/env bash\nexit 0\n")
	if err := writeSetupCompleteMarker(t, home, root); err != nil {
		t.Fatalf("write setup marker: %v", err)
	}

	svc.deps.currentHost = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	svc.deps.loadProject = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	svc.deps.startProjectAPI = func(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error { return nil }
	svc.deps.healthCheck = func(port int, timeout time.Duration) error { return nil }
	orchestratorCalls := 0
	svc.deps.startOrchestrator = func(root, home string, stdout, stderr io.Writer) error {
		orchestratorCalls++
		return nil
	}

	stdout := &strings.Builder{}
	if err := svc.RunDevelopWithOptions(root, home, Options{Scenarios: "none"}, stdout, io.Discard); err != nil {
		t.Fatalf("RunDevelopWithOptions: %v", err)
	}
	if orchestratorCalls != 0 {
		t.Fatalf("orchestrator calls = %d, want 0", orchestratorCalls)
	}
	if !strings.Contains(stdout.String(), "orchestrator skipped") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunDevelopTriggersOnboardingFallback(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)
	testresource.WritePortRegistry(t, root, nil)
	testkitgo.WriteExecutable(t, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "#!/usr/bin/env bash\nexit 0\n")
	if err := writeSetupCompleteMarker(t, home, root); err != nil {
		t.Fatalf("write setup marker: %v", err)
	}

	svc.deps.currentHost = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	svc.deps.loadProject = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	svc.deps.resolveHostRequirements = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	svc.deps.inspectRequirements = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}
	svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}
	svc.deps.startProjectAPI = func(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error { return nil }
	svc.deps.healthCheck = func(port int, timeout time.Duration) error { return nil }
	svc.deps.startOrchestrator = func(root, home string, stdout, stderr io.Writer) error { return nil }

	onboardingCalls := 0
	svc.deps.openOnboardingURL = func(url string) error {
		onboardingCalls++
		return nil
	}
	svc.deps.osExecutable = func() (string, error) { return "/bin/true", nil }
	svc.deps.onboardingPortCommandRunner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("41234\n"), nil
	}
	writeOnboardingScenarioFixture(t, root)

	if err := svc.RunDevelopWithOptions(root, home, Options{}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunDevelopWithOptions: %v", err)
	}
	if onboardingCalls != 1 {
		t.Fatalf("onboarding calls = %d, want 1", onboardingCalls)
	}
}

func TestRunSetupRejectsUnsupportedHost(t *testing.T) {
	svc := stubSetupDeps(t)

	svc.deps.currentHost = func() vrooliruntime.Host {
		return vrooliruntime.Host{OS: "darwin", SupportsSetup: false, Notes: []string{"unsupported in test"}}
	}

	err := svc.RunSetupWithOptions(t.TempDir(), t.TempDir(), Options{}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected unsupported host error")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadDotEnvParsesCommonForms(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	if err := os.WriteFile(path, []byte("FOO=bar\nexport BAZ=\"two\"\n# comment\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	values, err := loadDotEnv(path)
	if err != nil {
		t.Fatalf("loadDotEnv: %v", err)
	}
	if values["FOO"] != "bar" || values["BAZ"] != "two" {
		t.Fatalf("values = %#v", values)
	}
}

func TestMaybeOpenOnboardingPersistsPromptedState(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	writeOnboardingScenarioFixture(t, root)

	svc.deps.osExecutable = func() (string, error) { return "/bin/true", nil }
	svc.deps.onboardingPortCommandRunner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("38123\n"), nil
	}
	opened := ""
	svc.deps.openOnboardingURL = func(url string) error {
		opened = url
		return nil
	}

	stdout := &strings.Builder{}
	if err := svc.maybeOpenOnboarding(root, home, stdout, io.Discard); err != nil {
		t.Fatalf("maybeOpenOnboarding: %v", err)
	}
	if opened != "http://127.0.0.1:38123" {
		t.Fatalf("opened URL = %q", opened)
	}
	if !strings.Contains(stdout.String(), "Opening Vrooli onboarding") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	configPath := filepath.Join(home, ".config", "vrooli", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var stored struct {
		Onboarding onboardingPreferences `json:"onboarding"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if stored.Onboarding.PromptedAt == "" {
		t.Fatal("expected prompted_at to be recorded")
	}
}

func TestMaybeOpenOnboardingRespectsEnvOptOut(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	writeOnboardingScenarioFixture(t, root)
	t.Setenv(onboardingSkipEnv, "1")

	opened := false
	svc.deps.openOnboardingURL = func(url string) error {
		opened = true
		return nil
	}

	if err := svc.maybeOpenOnboarding(root, home, io.Discard, io.Discard); err != nil {
		t.Fatalf("maybeOpenOnboarding: %v", err)
	}
	if opened {
		t.Fatal("expected opt-out env var to skip onboarding")
	}
}

func TestMaybeOpenOnboardingRespectsPersistentAutoOpenOptOut(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	writeOnboardingScenarioFixture(t, root)
	autoOpen := false
	configPath := filepath.Join(home, ".config", "vrooli", "config.json")
	doc, err := json.Marshal(map[string]any{
		"onboarding": onboardingPreferences{AutoOpen: &autoOpen},
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, doc, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	opened := false
	svc.deps.openOnboardingURL = func(url string) error {
		opened = true
		return nil
	}

	if err := svc.maybeOpenOnboarding(root, home, io.Discard, io.Discard); err != nil {
		t.Fatalf("maybeOpenOnboarding: %v", err)
	}
	if opened {
		t.Fatal("expected persistent auto_open=false to skip onboarding")
	}
}

func TestMaybeOpenOnboardingIgnoresLegacyConfigLocation(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	writeOnboardingScenarioFixture(t, root)

	autoOpen := false
	legacyConfigPath := filepath.Join(home, ".vrooli", "config.json")
	doc, err := json.Marshal(map[string]any{
		"onboarding": onboardingPreferences{AutoOpen: &autoOpen},
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(legacyConfigPath), 0o755); err != nil {
		t.Fatalf("mkdir legacy config dir: %v", err)
	}
	if err := os.WriteFile(legacyConfigPath, doc, 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	svc.deps.osExecutable = func() (string, error) { return "/bin/true", nil }
	svc.deps.onboardingPortCommandRunner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("38123\n"), nil
	}

	opened := ""
	svc.deps.openOnboardingURL = func(url string) error {
		opened = url
		return nil
	}

	if err := svc.maybeOpenOnboarding(root, home, io.Discard, io.Discard); err != nil {
		t.Fatalf("maybeOpenOnboarding: %v", err)
	}
	if opened != "http://127.0.0.1:38123" {
		t.Fatalf("opened URL = %q", opened)
	}
	if _, err := os.Stat(legacyConfigPath); err != nil {
		t.Fatalf("legacy config should remain untouched: %v", err)
	}
}

type resourceRunnerFunc func(name string, args []string, stdout, stderr io.Writer) error

func (fn resourceRunnerFunc) Run(name string, args []string, stdout, stderr io.Writer) error {
	return fn(name, args, stdout, stderr)
}

func reportFromResolution(environment string, resolution hostreq.Resolution, executed bool) vrooliruntime.Report {
	report := vrooliruntime.Report{
		Environment: environment,
		Host:        vrooliruntime.Host{OS: "linux", PackageManager: "apt-get"},
		Tools:       make([]vrooliruntime.ToolStatus, 0, len(resolution.Tools)),
		Safeguards:  make([]vrooliruntime.SafeguardStatus, 0, len(resolution.Safeguards)),
	}
	for _, requirement := range resolution.Tools {
		state := vrooliruntime.ExecutionPending
		notes := append([]string(nil), requirement.Notes...)
		if executed {
			if requirement.Manual {
				state = vrooliruntime.ExecutionManualActionRequired
			} else {
				state = vrooliruntime.ExecutionWouldInstall
				notes = append(notes, "dry-run: would run apt-get install -y "+requirement.Name)
			}
		}
		if requirement.Name == "git" {
			state = vrooliruntime.ExecutionAlreadyPresent
		}
		report.Tools = append(report.Tools, vrooliruntime.ToolStatus{
			Name:           requirement.Name,
			Kind:           requirement.Kind,
			Required:       requirement.Required,
			ExecutionState: state,
			Reasons:        append([]string(nil), requirement.Reasons...),
			Notes:          notes,
			Provenance:     append([]hostreq.Provenance(nil), requirement.Provenance...),
		})
	}
	for _, requirement := range resolution.Safeguards {
		state := vrooliruntime.ExecutionPending
		notes := append([]string(nil), requirement.Notes...)
		if requirement.Manual {
			state = vrooliruntime.ExecutionManualActionRequired
		} else if executed {
			state = vrooliruntime.ExecutionWouldApply
			notes = append(notes, "dry-run: would apply "+requirement.Name)
		}
		report.Safeguards = append(report.Safeguards, vrooliruntime.SafeguardStatus{
			Name:           requirement.Name,
			Kind:           requirement.Kind,
			Required:       requirement.Required,
			ExecutionState: state,
			Reasons:        append([]string(nil), requirement.Reasons...),
			Notes:          notes,
			Provenance:     append([]hostreq.Provenance(nil), requirement.Provenance...),
		})
	}
	return report
}

func stubSetupDeps(t *testing.T) *setupService {
	t.Helper()
	deps := defaultSetupDeps()
	deps.syncResourceSchema = func(root string) error { return nil }
	deps.ensureBootstrapTools = func(string, vrooliruntime.EnsureOptions) error { return nil }
	deps.newCLIInstallManager = func(root, home string) (cliInstallManager, error) { return &stubCLIInstallManager{}, nil }
	// These tests exercise setup orchestration with isolated temporary homes;
	// credential backend readiness is covered by the securestore tests and is
	// deliberately not allowed to probe the developer's real keyring here.
	deps.configureCredentialBackend = func(io.Writer, io.Writer) error { return nil }
	return newSetupService(deps)
}

type stubCLIInstallManager struct {
	root                        string
	home                        string
	installAllScenarioCalls     int
	installEnabledResourceCalls int
	installedScenarios          []string
	installedResources          []string
}

func (s *stubCLIInstallManager) InstallScenarioCLI(name string) error {
	s.installedScenarios = append(s.installedScenarios, name)
	return nil
}

func (s *stubCLIInstallManager) EnsureScenarioCLI(name string) error {
	return nil
}

func (s *stubCLIInstallManager) InstallResourceCLI(name string) error {
	s.installedResources = append(s.installedResources, name)
	return nil
}

func (s *stubCLIInstallManager) InstallAllScenarioCLIs() error {
	s.installAllScenarioCalls++
	return nil
}

func (s *stubCLIInstallManager) InstallEnabledResourceCLIs() error {
	s.installEnabledResourceCalls++
	return nil
}

func findResolvedRequirement(items []hostreq.ResolvedRequirement, name string) *hostreq.ResolvedRequirement {
	for i := range items {
		if items[i].Name == name {
			return &items[i]
		}
	}
	return nil
}

var _ resourceRunner = (*resources.Controller)(nil)

func intPtr(value int) *int {
	return &value
}
