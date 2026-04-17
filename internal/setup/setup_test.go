package setup

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostreq"
	hostreqspec "github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/projectstate"
	"github.com/vrooli/vrooli/internal/resources"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=4 | LAST: 2026-04-16

func TestMarkCompleteWritesSetupMarker(t *testing.T) {
	root := t.TempDir()

	if err := markComplete(root); err != nil {
		t.Fatalf("markComplete: %v", err)
	}

	setupMarker := projectstate.SetupCompletePath(root)
	payload := testkitgo.ReadJSONFile(t, setupMarker)
	if payload["setup_version"] != "2.0.0" {
		t.Fatalf("setup_version = %v", payload["setup_version"])
	}
	if _, err := os.Stat(projectstate.ResourcesPopulatedPath(root)); !os.IsNotExist(err) {
		t.Fatalf("expected no resources marker, got %v", err)
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

func TestRepoRemovesLegacyHostSetupSurfaces(t *testing.T) {
	repoRoot := testkitgo.ProjectRoot(t)

	for _, rel := range []string{
		"scripts/lib/setup.sh",
		"scripts/lib/setup-conditions/binaries-check.sh",
		"scripts/lib/setup-conditions/cli-check.sh",
		"scripts/lib/setup-conditions/data-check.sh",
		"scripts/lib/setup-conditions/dependencies-check.sh",
		"scripts/lib/setup-conditions/directories-check.sh",
		"scripts/lib/setup-conditions/files-check.sh",
		"scripts/lib/setup-conditions/resources-check.sh",
		"scripts/lib/setup-conditions/ui-bundle-check.sh",
		"scripts/lib/deps/ajv.sh",
		"scripts/lib/deps/ast-grep.sh",
		"scripts/lib/deps/bats.sh",
		"scripts/lib/deps/js-yaml.sh",
		"scripts/lib/deps/lychee.sh",
		"scripts/lib/deps/shellcheck.sh",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, rel)); !os.IsNotExist(err) {
			t.Fatalf("expected legacy host setup surface %s to be deleted, stat err=%v", rel, err)
		}
	}

	for _, rel := range []string{
		"docs/QUICKSTART.md",
		"docs/operations/README.md",
		"internal/lifecycle/setup.go",
		"scenarios/scenario-to-cloud/api/bundling_rules_test.go",
		"scripts/README.md",
	} {
		data, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", rel, err)
		}
		text := string(data)
		for _, forbidden := range []string{
			"scripts/lib/setup.sh",
			"./scripts/setup.sh",
			"scripts/lib/setup-conditions",
			"setup.sh<br/>Called by all main scripts",
			"runs `setup.sh`",
		} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s still contains forbidden legacy setup reference %q", rel, forbidden)
			}
		}
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

	runtimeCalls := 0
	svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
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
	svc.deps.markComplete = func(root string) error {
		markCompleteCalled = true
		return nil
	}
	schemaSyncCalled := false
	svc.deps.syncResourceSchema = func(root string) error {
		schemaSyncCalled = true
		return nil
	}
	manager := &stubCLIInstallManager{}
	svc.deps.newCLIInstallManager = func(root, home string) cliInstallManager {
		manager.root = root
		manager.home = home
		return manager
	}

	if err := svc.RunSetupWithOptions(root, home, Options{Resources: "none"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}
	if runtimeCalls != 1 {
		t.Fatalf("ensureRequirements calls = %d, want 1", runtimeCalls)
	}
	if !markCompleteCalled {
		t.Fatal("expected markCompleteFn to be called")
	}
	if !schemaSyncCalled {
		t.Fatal("expected syncResourceSchemaFn to be called")
	}
	if manager.installEnabledResourceCalls != 1 {
		t.Fatalf("InstallEnabledResourceCLIs calls = %d, want 1", manager.installEnabledResourceCalls)
	}
	if manager.installAllScenarioCalls != 1 {
		t.Fatalf("InstallAllScenarioCLIs calls = %d, want 1", manager.installAllScenarioCalls)
	}
	if manager.root != root || manager.home != home {
		t.Fatalf("manager inputs = (%q, %q), want (%q, %q)", manager.root, manager.home, root, home)
	}
	if _, err := os.Stat(filepath.Join(root, "data")); err != nil {
		t.Fatalf("expected data dir: %v", err)
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
	svc.deps.markComplete = func(root string) error {
		markCompleteCalled = true
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
	svc.deps.markComplete = func(root string) error {
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
	if !strings.Contains(stdout.String(), "tmux [required] would_install") {
		t.Fatalf("stdout missing would_install result:\n%s", stdout.String())
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
	svc.deps.markComplete = func(root string) error { return nil }

	if err := svc.RunSetupWithOptions(root, home, Options{Scenarios: "alpha,beta", Resources: "none", DryRun: true}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}
	if captured.Scenarios != "alpha,beta" {
		t.Fatalf("captured.Scenarios = %q", captured.Scenarios)
	}
}

func TestRunSetupPrintsPlanAndDryRunResult(t *testing.T) {
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
	svc.deps.markComplete = func(root string) error { return nil }

	stdout := &strings.Builder{}
	if err := svc.RunSetupWithOptions(root, home, Options{Resources: "none", Scenarios: "alpha", DryRun: true}, stdout, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}

	output := stdout.String()
	for _, expected := range []string{
		"Host requirements plan",
		"environment=development resources=none scenarios=alpha dry_run=true",
		"git [required] already_present",
		"tmux [required] planned_install",
		"remote_session_protection [optional] not_applicable",
		"Host requirements dry-run result",
		"tmux [required] would_install",
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
		plannedResolution = resolution
		return reportFromResolution(environment, resolution, false), nil
	}
	svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
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
	if findResolvedRequirement(plannedResolution.Tools, "docker") == nil {
		t.Fatal("expected root docker declaration")
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
		"git [required] already_present",
		"docker [required] planned_install",
		"tmux [required] planned_install",
		"sqlite [optional] manual_action_required",
		"remote_session_protection [required] planned_apply",
		"docker [required] would_install",
		"tmux [required] would_install",
		"remote_session_protection [required] would_apply",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("stdout missing %q:\n%s", expected, output)
		}
	}
}

func TestRunSetupDoesNotLeakLegacyEnvironmentContractToResourceInstall(t *testing.T) {
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
	svc.deps.markComplete = func(root string) error { return nil }

	type installCall struct {
		name string
		args []string
		env  map[string]string
	}
	var installs []installCall
	svc.deps.resourceController = func(root, home string) resourceRunner {
		return resourceRunnerFunc(func(name string, args []string, stdout, stderr io.Writer) error {
			installs = append(installs, installCall{
				name: name,
				args: append([]string(nil), args...),
				env: map[string]string{
					"SERVICE_JSON_PATH":  os.Getenv("SERVICE_JSON_PATH"),
					"ENVIRONMENT":        os.Getenv("ENVIRONMENT"),
					"RESOURCES":          os.Getenv("RESOURCES"),
					"SCENARIOS":          os.Getenv("SCENARIOS"),
					"YES":                os.Getenv("YES"),
					"SUDO_MODE":          os.Getenv("SUDO_MODE"),
					"SUDO_MODE_EXPLICIT": os.Getenv("SUDO_MODE_EXPLICIT"),
					"TARGET":             os.Getenv("TARGET"),
					"LOCATION":           os.Getenv("LOCATION"),
					"DRY_RUN":            os.Getenv("DRY_RUN"),
				},
			})
			return nil
		})
	}

	err := svc.RunSetupWithOptions(root, home, Options{
		Environment: "minimal",
		Resources:   "redis,postgres",
		Scenarios:   "scenario-a,scenario-b",
		Yes:         "yes",
		SudoMode:    "skip",
	}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}
	if len(installs) != 2 {
		t.Fatalf("install calls = %d, want 2", len(installs))
	}
	for _, call := range installs {
		if got := strings.Join(call.args, "|"); got != "install" {
			t.Fatalf("resource %s args = %q", call.name, got)
		}
		for _, key := range []string{"SERVICE_JSON_PATH", "ENVIRONMENT", "RESOURCES", "SCENARIOS", "YES", "SUDO_MODE", "SUDO_MODE_EXPLICIT", "TARGET", "LOCATION", "DRY_RUN"} {
			if call.env[key] != "" {
				t.Fatalf("resource %s leaked legacy env %s = %q", call.name, key, call.env[key])
			}
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
	svc.deps.markComplete = func(root string) error {
		setupCalls++
		return writeSetupCompleteMarker(t, root)
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

func TestRunDevelopExportsLegacyEnvironmentContractToAPILaunch(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)
	testresource.WritePortRegistry(t, root, nil)
	testkitgo.WriteExecutable(t, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "#!/usr/bin/env bash\nexit 0\n")
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
	svc.deps.markComplete = func(root string) error {
		return writeSetupCompleteMarker(t, root)
	}
	svc.deps.loadDotEnv = func(path string) (map[string]string, error) {
		return map[string]string{
			"VROOLI_API_PORT": "18095",
			"FROM_DOT_ENV":    "present",
		}, nil
	}

	var capturedSpec apiLaunchSpec
	svc.deps.startProjectAPI = func(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error {
		capturedSpec = spec
		return nil
	}
	svc.deps.healthCheck = func(port int, timeout time.Duration) error { return nil }
	svc.deps.startOrchestrator = func(root, home string, stdout, stderr io.Writer) error { return nil }

	err := svc.RunDevelopWithOptions(root, home, Options{
		Environment: "production",
		Resources:   "enabled",
		Yes:         "yes",
		SudoMode:    "skip",
		DryRun:      true,
	}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("RunDevelopWithOptions: %v", err)
	}
	if capturedSpec.Command == "" {
		t.Fatal("expected API launch spec to be populated")
	}
	if capturedSpec.LogFile != filepath.Join(home, ".vrooli", "logs", "vrooli-api.log") {
		t.Fatalf("LogFile = %q", capturedSpec.LogFile)
	}
	env := envMapFromList(capturedSpec.Env)
	for _, key := range []string{"SERVICE_JSON_PATH", "ENVIRONMENT", "RESOURCES", "YES", "SUDO_MODE", "SUDO_MODE_EXPLICIT", "TARGET", "LOCATION", "DRY_RUN"} {
		if env[key] != "" {
			t.Fatalf("%s = %q, want empty legacy env surface", key, env[key])
		}
	}
	if env["FROM_DOT_ENV"] != "present" {
		t.Fatalf("FROM_DOT_ENV = %q", env["FROM_DOT_ENV"])
	}
	if env["VROOLI_API_PORT"] != "18095" {
		t.Fatalf("VROOLI_API_PORT = %q", env["VROOLI_API_PORT"])
	}
	if capturedSpec.Port != 18095 {
		t.Fatalf("capturedSpec.Port = %d", capturedSpec.Port)
	}
}

func TestRunDevelopSkipsSetupWhenMarkerExists(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)
	testresource.WritePortRegistry(t, root, nil)
	testkitgo.WriteExecutable(t, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "#!/usr/bin/env bash\nexit 0\n")
	if err := writeSetupCompleteMarker(t, root); err != nil {
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
	svc.deps.markComplete = func(root string) error {
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

func TestRunDevelopTriggersOnboardingFallback(t *testing.T) {
	svc := stubSetupDeps(t)

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)
	testresource.WritePortRegistry(t, root, nil)
	testkitgo.WriteExecutable(t, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "#!/usr/bin/env bash\nexit 0\n")
	if err := writeSetupCompleteMarker(t, root); err != nil {
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

type resourceRunnerFunc func(name string, args []string, stdout, stderr io.Writer) error

func (fn resourceRunnerFunc) Run(name string, args []string, stdout, stderr io.Writer) error {
	return fn(name, args, stdout, stderr)
}

func envMapFromList(env []string) map[string]string {
	values := make(map[string]string, len(env))
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	return values
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
	deps.newCLIInstallManager = func(root, home string) cliInstallManager { return &stubCLIInstallManager{} }
	return newSetupService(deps)
}

type stubCLIInstallManager struct {
	root                        string
	home                        string
	installAllScenarioCalls     int
	installEnabledResourceCalls int
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
