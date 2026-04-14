package setup

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostreq"
	hostreqspec "github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/resources"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testkitvrooli "github.com/vrooli/vrooli/packages/testkit-go/vrooli"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=3 | LAST: 2026-04-11

func TestApplyEnvironmentSetsDefaultsAndRestoresState(t *testing.T) {
	t.Setenv("TARGET", "")
	t.Setenv("LOCATION", "")
	root := t.TempDir()
	restore, err := applyEnvironment(root, filepath.Join(root, ".vrooli", "service.json"), Options{
		Environment: "production",
		Resources:   "none",
		Scenarios:   "scenario-a,scenario-b",
		Yes:         "yes",
		SudoMode:    "skip",
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("applyEnvironment: %v", err)
	}

	if got := os.Getenv("APP_ROOT"); got != root {
		t.Fatalf("APP_ROOT = %q", got)
	}
	if got := os.Getenv("TARGET"); got != defaultTarget {
		t.Fatalf("TARGET = %q", got)
	}
	if got := os.Getenv("LOCATION"); got != defaultLocation {
		t.Fatalf("LOCATION = %q", got)
	}
	if got := os.Getenv("ENVIRONMENT"); got != "production" {
		t.Fatalf("ENVIRONMENT = %q", got)
	}
	if got := os.Getenv("RESOURCES"); got != "none" {
		t.Fatalf("RESOURCES = %q", got)
	}
	if got := os.Getenv("SCENARIOS"); got != "scenario-a,scenario-b" {
		t.Fatalf("SCENARIOS = %q", got)
	}
	if got := os.Getenv("YES"); got != "yes" {
		t.Fatalf("YES = %q", got)
	}
	if got := os.Getenv("SUDO_MODE"); got != "skip" {
		t.Fatalf("SUDO_MODE = %q", got)
	}
	if got := os.Getenv("DRY_RUN"); got != "true" {
		t.Fatalf("DRY_RUN = %q", got)
	}
	if got := os.Getenv("SERVICE_JSON_PATH"); got != filepath.Join(root, ".vrooli", "service.json") {
		t.Fatalf("SERVICE_JSON_PATH = %q", got)
	}

	restore()

	if got := os.Getenv("APP_ROOT"); got != "" {
		t.Fatalf("APP_ROOT after restore = %q", got)
	}
	if got := os.Getenv("TARGET"); got != "" {
		t.Fatalf("TARGET after restore = %q", got)
	}
	if got := os.Getenv("SERVICE_JSON_PATH"); got != "" {
		t.Fatalf("SERVICE_JSON_PATH after restore = %q", got)
	}
	if got := os.Getenv("SCENARIOS"); got != "" {
		t.Fatalf("SCENARIOS after restore = %q", got)
	}
}

func TestMarkCompleteWritesSetupMarker(t *testing.T) {
	root := t.TempDir()

	if err := markComplete(root); err != nil {
		t.Fatalf("markComplete: %v", err)
	}

	setupMarker := filepath.Join(root, "data", ".setup-complete")
	payload := testkitgo.ReadJSONFile(t, setupMarker)
	if payload["setup_version"] != "2.0.0" {
		t.Fatalf("setup_version = %v", payload["setup_version"])
	}
	if _, err := os.Stat(filepath.Join(root, "data", ".resources-populated")); !os.IsNotExist(err) {
		t.Fatalf("expected no resources marker, got %v", err)
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
		"docs/GETTING_STARTED.md",
		"docs/devops/README.md",
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
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	resolveHostRequirementsFn = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	inspectRequirementsFn = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}

	runtimeCalls := 0
	ensureRequirementsFn = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
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
	markCompleteFn = func(root string) error {
		markCompleteCalled = true
		return nil
	}

	if err := RunSetupWithOptions(root, home, Options{Resources: "none"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}
	if runtimeCalls != 1 {
		t.Fatalf("ensureRequirements calls = %d, want 1", runtimeCalls)
	}
	if !markCompleteCalled {
		t.Fatal("expected markCompleteFn to be called")
	}
	if _, err := os.Stat(filepath.Join(root, "data")); err != nil {
		t.Fatalf("expected data dir: %v", err)
	}
}

func TestRunSetupDryRunUsesApplyPlanningAndSkipsMutations(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	currentHostFn = func() vrooliruntime.Host {
		return vrooliruntime.Host{OS: "linux", PackageManager: "apt-get", SupportsSetup: true, SupportsDevelop: true}
	}
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	resolveHostRequirementsFn = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{
			Tools: []hostreq.ResolvedRequirement{
				{Name: "tmux", Kind: hostreq.KindTool, Required: true},
			},
		}, nil
	}
	inspectRequirementsFn = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return reportFromResolution(environment, resolution, false), nil
	}

	ensureCalls := 0
	ensureRequirementsFn = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
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
	markCompleteFn = func(root string) error {
		markCompleteCalled = true
		return nil
	}

	resourceInstallCalls := 0
	resourcesController = func(root, home string) resourceRunner {
		return resourceRunnerFunc(func(name string, args []string, stdout, stderr io.Writer) error {
			resourceInstallCalls++
			return nil
		})
	}

	stdout := &strings.Builder{}
	if err := RunSetupWithOptions(root, home, Options{DryRun: true, Resources: "redis", Scenarios: "alpha"}, stdout, io.Discard); err != nil {
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

func TestRunSetupPassesScenarioSelectionToResolver(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }

	var captured hostreq.ResolveOptions
	resolveHostRequirementsFn = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		captured = opts
		return hostreq.Resolution{}, nil
	}
	inspectRequirementsFn = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}
	ensureRequirementsFn = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}
	markCompleteFn = func(root string) error { return nil }

	if err := RunSetupWithOptions(root, home, Options{Scenarios: "alpha,beta", Resources: "none", DryRun: true}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}
	if captured.Scenarios != "alpha,beta" {
		t.Fatalf("captured.Scenarios = %q", captured.Scenarios)
	}
}

func TestRunSetupPrintsPlanAndDryRunResult(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	currentHostFn = func() vrooliruntime.Host {
		return vrooliruntime.Host{OS: "linux", PackageManager: "apt-get", SupportsSetup: true, SupportsDevelop: true}
	}
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	resolveHostRequirementsFn = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	inspectRequirementsFn = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
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
	ensureRequirementsFn = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
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
	markCompleteFn = func(root string) error { return nil }

	stdout := &strings.Builder{}
	if err := RunSetupWithOptions(root, home, Options{Resources: "none", Scenarios: "alpha", DryRun: true}, stdout, io.Discard); err != nil {
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
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectManifest := testkitvrooli.ProjectServiceManifest(
		testkitvrooli.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{"redis": {Enabled: false}},
		}),
	)
	projectManifest.HostTools = []hostreqspec.Declaration{
		{Name: "git", Required: true, Reason: "repo operations"},
		{Name: "docker", Required: true, Reason: "container runtime"},
	}
	projectScenario := writeProjectFixtureWithServiceManifest(t, root, projectManifest)
	testkitvrooli.WriteScenarioService(t, root, "alpha", testkitvrooli.ScenarioServiceManifest(
		"alpha",
		testkitvrooli.WithLifecycle(scenario.Lifecycle{}),
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
	testkitvrooli.WriteResourceManifest(t, root, "redis", testkitvrooli.ResourceManifest(
		"redis",
		testkitvrooli.WithResourceDriver("external-cli"),
		testkitvrooli.WithResourceBinary("redis-server"),
		testkitvrooli.WithResourcePlatforms(manifestpkg.ResourcePlatforms{
			Linux:   "supported",
			MacOS:   "partial",
			Windows: "unsupported",
		}),
		testkitvrooli.WithResourceHostTools(
			hostreqspec.Declaration{Name: "sqlite", Required: false, Reason: "resource cache introspection", Manual: true},
		),
	))

	currentHostFn = func() vrooliruntime.Host {
		return vrooliruntime.Host{OS: "linux", PackageManager: "apt-get", SupportsSetup: true, SupportsDevelop: true}
	}
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }

	var plannedResolution hostreq.Resolution
	inspectRequirementsFn = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		plannedResolution = resolution
		return reportFromResolution(environment, resolution, false), nil
	}
	ensureRequirementsFn = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return reportFromResolution(opts.Environment, resolution, true), nil
	}

	stdout := &strings.Builder{}
	if err := RunSetupWithOptions(root, home, Options{Resources: "redis", Scenarios: "alpha", DryRun: true}, stdout, io.Discard); err != nil {
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

func TestRunSetupExportsLegacyEnvironmentContractToResourceInstall(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	resolveHostRequirementsFn = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	inspectRequirementsFn = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}
	ensureRequirementsFn = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}
	markCompleteFn = func(root string) error { return nil }

	type installCall struct {
		name string
		args []string
		env  map[string]string
	}
	var installs []installCall
	resourcesController = func(root, home string) resourceRunner {
		return resourceRunnerFunc(func(name string, args []string, stdout, stderr io.Writer) error {
			installs = append(installs, installCall{
				name: name,
				args: append([]string(nil), args...),
				env: map[string]string{
					"APP_ROOT":           os.Getenv("APP_ROOT"),
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

	err := RunSetupWithOptions(root, home, Options{
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
		if call.env["APP_ROOT"] != root {
			t.Fatalf("resource %s APP_ROOT = %q", call.name, call.env["APP_ROOT"])
		}
		if call.env["SERVICE_JSON_PATH"] != filepath.Join(root, ".vrooli", "service.json") {
			t.Fatalf("resource %s SERVICE_JSON_PATH = %q", call.name, call.env["SERVICE_JSON_PATH"])
		}
		if call.env["ENVIRONMENT"] != "minimal" {
			t.Fatalf("resource %s ENVIRONMENT = %q", call.name, call.env["ENVIRONMENT"])
		}
		if call.env["RESOURCES"] != "redis,postgres" {
			t.Fatalf("resource %s RESOURCES = %q", call.name, call.env["RESOURCES"])
		}
		if call.env["SCENARIOS"] != "scenario-a,scenario-b" {
			t.Fatalf("resource %s SCENARIOS = %q", call.name, call.env["SCENARIOS"])
		}
		if call.env["YES"] != "yes" {
			t.Fatalf("resource %s YES = %q", call.name, call.env["YES"])
		}
		if call.env["SUDO_MODE"] != "skip" || call.env["SUDO_MODE_EXPLICIT"] != "skip" {
			t.Fatalf("resource %s sudo env = %#v", call.name, call.env)
		}
		if call.env["TARGET"] != defaultTarget {
			t.Fatalf("resource %s TARGET = %q", call.name, call.env["TARGET"])
		}
		if call.env["LOCATION"] != defaultLocation {
			t.Fatalf("resource %s LOCATION = %q", call.name, call.env["LOCATION"])
		}
		if call.env["DRY_RUN"] != "" {
			t.Fatalf("resource %s DRY_RUN = %q", call.name, call.env["DRY_RUN"])
		}
	}
}

func TestRunSetupDryRunSkipsResourceInstallEvenWhenResourcesSelected(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	resolveHostRequirementsFn = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	inspectRequirementsFn = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}
	ensureRequirementsFn = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}

	resourceInstallCalls := 0
	resourcesController = func(root, home string) resourceRunner {
		return resourceRunnerFunc(func(name string, args []string, stdout, stderr io.Writer) error {
			resourceInstallCalls++
			return nil
		})
	}

	if err := RunSetupWithOptions(root, home, Options{Resources: "redis,postgres", DryRun: true}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions: %v", err)
	}
	if resourceInstallCalls != 0 {
		t.Fatalf("resource install calls = %d, want 0", resourceInstallCalls)
	}
}

func TestRunDevelopRunsSetupWhenNeededAndStartsNativeServices(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)
	writePortRegistryFixture(t, root)
	writeExecutableFile(t, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "#!/usr/bin/env bash\nexit 0\n")
	t.Setenv("VROOLI_API_PORT", "18096")
	t.Setenv("VROOLI_API_PORT", "18095")

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	resolveHostRequirementsFn = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	inspectRequirementsFn = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}
	ensureRequirementsFn = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}

	setupCalls := 0
	markCompleteFn = func(root string) error {
		setupCalls++
		return os.WriteFile(filepath.Join(root, "data", ".setup-complete"), []byte("ok\n"), 0o644)
	}

	apiStarted := false
	startProjectAPIFn = func(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error {
		apiStarted = true
		if spec.Command == "" || spec.Port != 18095 {
			t.Fatalf("spec = %+v", spec)
		}
		return nil
	}
	healthCalls := 0
	healthCheckFn = func(port int, timeout time.Duration) error {
		healthCalls++
		if port != 18095 {
			t.Fatalf("port = %d", port)
		}
		return nil
	}
	orchestratorStarted := false
	startOrchestratorFn = func(root, home string, stdout, stderr io.Writer) error {
		orchestratorStarted = true
		return nil
	}

	stdout := &strings.Builder{}
	if err := RunDevelopWithOptions(root, home, Options{}, stdout, io.Discard); err != nil {
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
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)
	writePortRegistryFixture(t, root)
	writeExecutableFile(t, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "#!/usr/bin/env bash\nexit 0\n")
	t.Setenv("VROOLI_API_PORT", "18095")

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	resolveHostRequirementsFn = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	inspectRequirementsFn = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}
	ensureRequirementsFn = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}
	markCompleteFn = func(root string) error {
		return os.WriteFile(filepath.Join(root, "data", ".setup-complete"), []byte("ok\n"), 0o644)
	}
	loadDotEnvFn = func(path string) (map[string]string, error) {
		return map[string]string{
			"VROOLI_API_PORT": "18095",
			"FROM_DOT_ENV":    "present",
		}, nil
	}

	var capturedSpec apiLaunchSpec
	startProjectAPIFn = func(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error {
		capturedSpec = spec
		return nil
	}
	healthCheckFn = func(port int, timeout time.Duration) error { return nil }
	startOrchestratorFn = func(root, home string, stdout, stderr io.Writer) error { return nil }

	err := RunDevelopWithOptions(root, home, Options{
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
	if env["APP_ROOT"] != root {
		t.Fatalf("APP_ROOT = %q", env["APP_ROOT"])
	}
	if env["SERVICE_JSON_PATH"] != filepath.Join(root, ".vrooli", "service.json") {
		t.Fatalf("SERVICE_JSON_PATH = %q", env["SERVICE_JSON_PATH"])
	}
	if env["ENVIRONMENT"] != "production" {
		t.Fatalf("ENVIRONMENT = %q", env["ENVIRONMENT"])
	}
	if env["RESOURCES"] != "enabled" {
		t.Fatalf("RESOURCES = %q", env["RESOURCES"])
	}
	if env["YES"] != "yes" {
		t.Fatalf("YES = %q", env["YES"])
	}
	if env["SUDO_MODE"] != "skip" || env["SUDO_MODE_EXPLICIT"] != "skip" {
		t.Fatalf("sudo env = %#v", env)
	}
	if env["TARGET"] != defaultTarget {
		t.Fatalf("TARGET = %q", env["TARGET"])
	}
	if env["LOCATION"] != defaultLocation {
		t.Fatalf("LOCATION = %q", env["LOCATION"])
	}
	if env["DRY_RUN"] != "true" {
		t.Fatalf("DRY_RUN = %q", env["DRY_RUN"])
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
	restore := stubSetupDeps(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)
	writePortRegistryFixture(t, root)
	writeExecutableFile(t, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "#!/usr/bin/env bash\nexit 0\n")
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "data", ".setup-complete"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write setup marker: %v", err)
	}

	currentHostFn = func() vrooliruntime.Host { return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true} }
	loadProjectFn = func(root string) (scenario.Scenario, error) { return projectScenario, nil }
	resolveHostRequirementsFn = func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	inspectRequirementsFn = func(environment string, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: environment}, nil
	}
	ensureRequirementsFn = func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{Environment: opts.Environment}, nil
	}

	setupCalls := 0
	markCompleteFn = func(root string) error {
		setupCalls++
		return nil
	}
	startProjectAPIFn = func(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error { return nil }
	healthCheckFn = func(port int, timeout time.Duration) error { return nil }
	startOrchestratorFn = func(root, home string, stdout, stderr io.Writer) error { return nil }

	if err := RunDevelopWithOptions(root, home, Options{}, io.Discard, io.Discard); err != nil {
		t.Fatalf("RunDevelopWithOptions: %v", err)
	}
	if setupCalls != 0 {
		t.Fatalf("setup calls = %d, want 0", setupCalls)
	}
}

func TestRunSetupRejectsUnsupportedHost(t *testing.T) {
	restore := stubSetupDeps(t)
	defer restore()

	currentHostFn = func() vrooliruntime.Host {
		return vrooliruntime.Host{OS: "darwin", SupportsSetup: false, Notes: []string{"unsupported in test"}}
	}

	err := RunSetupWithOptions(t.TempDir(), t.TempDir(), Options{}, io.Discard, io.Discard)
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

func stubSetupDeps(t *testing.T) func() {
	t.Helper()
	originalCurrentHostFn := currentHostFn
	originalLoadProjectFn := loadProjectFn
	originalMarkCompleteFn := markCompleteFn
	originalResolveHostRequirementsFn := resolveHostRequirementsFn
	originalInspectRequirementsFn := inspectRequirementsFn
	originalEnsureRequirementsFn := ensureRequirementsFn
	originalStartProjectAPIFn := startProjectAPIFn
	originalStartOrchestratorFn := startOrchestratorFn
	originalHealthCheckFn := healthCheckFn
	originalLoadDotEnvFn := loadDotEnvFn
	originalResourcesController := resourcesController
	return func() {
		currentHostFn = originalCurrentHostFn
		loadProjectFn = originalLoadProjectFn
		markCompleteFn = originalMarkCompleteFn
		resolveHostRequirementsFn = originalResolveHostRequirementsFn
		inspectRequirementsFn = originalInspectRequirementsFn
		ensureRequirementsFn = originalEnsureRequirementsFn
		startProjectAPIFn = originalStartProjectAPIFn
		startOrchestratorFn = originalStartOrchestratorFn
		healthCheckFn = originalHealthCheckFn
		loadDotEnvFn = originalLoadDotEnvFn
		resourcesController = originalResourcesController
	}
}

func writeProjectFixture(t *testing.T, root string) scenario.Scenario {
	t.Helper()
	manifest := testkitvrooli.ProjectServiceManifest(
		testkitvrooli.WithPorts(map[string]scenario.Port{
			"api": {EnvVar: "VROOLI_API_PORT", Port: intPtr(8092)},
		}),
		testkitvrooli.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{"redis": {Enabled: false}},
		}),
	)
	testkitvrooli.WriteProjectService(t, root, manifest)
	return scenario.Scenario{
		Slug:        "project-alpha",
		Path:        root,
		ServicePath: scenario.ProjectServicePath(root),
		Manifest:    manifest,
	}
}

func writeProjectFixtureWithServiceManifest(t *testing.T, root string, manifest scenario.ServiceManifest) scenario.Scenario {
	t.Helper()
	testkitvrooli.WriteProjectService(t, root, manifest)
	servicePath := scenario.ProjectServicePath(root)
	if strings.TrimSpace(manifest.Service.Name) == "" {
		manifest.Service.Name = filepath.Base(root)
	}
	return scenario.Scenario{
		Slug:        manifest.Service.Name,
		Path:        root,
		ServicePath: servicePath,
		Manifest:    manifest,
	}
}

func writeProjectFixtureWithManifest(t *testing.T, root, manifest string) scenario.Scenario {
	t.Helper()
	servicePath := filepath.Join(root, ".vrooli", "service.json")
	testkitvrooli.WriteMalformedProjectService(t, root, manifest)
	parsed, err := scenario.ReadService(servicePath)
	if err != nil {
		t.Fatalf("ReadService: %v", err)
	}
	return scenario.Scenario{
		Slug:        parsed.Service.Name,
		Path:        root,
		ServicePath: servicePath,
		Manifest:    parsed,
	}
}

func writeSetupTestFile(t *testing.T, path, contents string) {
	t.Helper()
	testkitgo.WriteFile(t, path, contents)
}

func findResolvedRequirement(items []hostreq.ResolvedRequirement, name string) *hostreq.ResolvedRequirement {
	for i := range items {
		if items[i].Name == name {
			return &items[i]
		}
	}
	return nil
}

func writePortRegistryFixture(t *testing.T, root string) {
	t.Helper()
	testkitvrooli.WritePortRegistry(t, root, nil)
}

func writeExecutableFile(t *testing.T, path, contents string) {
	t.Helper()
	testkitgo.WriteExecutable(t, path, contents)
}

var _ resourceRunner = (*resources.Controller)(nil)

func intPtr(value int) *int {
	return &value
}
