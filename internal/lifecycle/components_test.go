//nolint:goconst // test data deliberately reuses stable environment fixtures.
package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/packagegov"
	packagefixture "github.com/vrooli/vrooli/internal/packagegov/packagegovtest"
	"github.com/vrooli/vrooli/internal/scenario"
)

func testBuildScenario(t *testing.T) scenario.Scenario {
	t.Helper()
	root := t.TempDir()
	return scenario.Scenario{
		Slug: "ledger-build-test",
		Path: root,
		Manifest: scenario.ServiceManifest{Components: map[string]scenario.Component{
			"api": {Role: "api", Build: scenario.ComponentBuild{Kind: "test_builder", Dir: "."}},
			"ui":  {Role: "ui", Build: scenario.ComponentBuild{Kind: "test_builder", Dir: "."}},
		}},
	}
}

func installTestBuilder(t *testing.T) {
	t.Helper()
	previous, existed := builderRegistry["test_builder"]
	builderRegistry["test_builder"] = BuilderSpec{
		Kind:          "test_builder",
		DefaultOutput: "{dir}/{component}.artifact",
		Build:         []string{"cp", "/etc/hosts", "{output}"},
	}
	t.Cleanup(func() {
		if existed {
			builderRegistry["test_builder"] = previous
		} else {
			delete(builderRegistry, "test_builder")
		}
	})
}

func installTimedTestBuilder(t *testing.T, reservation int64, failAPI bool) {
	t.Helper()
	script := filepath.Join(t.TempDir(), "builder.sh")
	contents := "#!/bin/sh\n"
	if failAPI {
		contents += "case \"$1\" in *api.artifact) exit 17;; esac\n"
	}
	contents += "sleep 0.30\nprintf ok > \"$1\"\n"
	if err := os.WriteFile(script, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
	previous, existed := builderRegistry["test_builder"]
	builderRegistry["test_builder"] = BuilderSpec{
		Kind:                   "test_builder",
		DefaultOutput:          "{dir}/{component}.artifact",
		MemoryReservationBytes: reservation,
		Build:                  []string{script, "{output}"},
	}
	t.Cleanup(func() {
		if existed {
			builderRegistry["test_builder"] = previous
		} else {
			delete(builderRegistry, "test_builder")
		}
	})
}

func runBuildLedgerTest(t *testing.T, item scenario.Scenario, force bool, stale ...string) int {
	t.Helper()
	verdicts := make(map[string]artifactVerdict, len(stale))
	for _, name := range stale {
		verdicts[name] = artifactVerdict{Target: name, Stale: true}
	}
	runner := &Runner{Root: item.Path, Home: t.TempDir()}
	count, err := runner.buildDeclaredComponents(context.Background(), item, nil, io.Discard, force, verdicts)
	if err != nil {
		t.Fatalf("buildDeclaredComponents: %v", err)
	}
	return count
}

func TestBuildDeclaredComponents_SkipsFreshComponents(t *testing.T) {
	installTestBuilder(t)
	if got := runBuildLedgerTest(t, testBuildScenario(t), false); got != 0 {
		t.Fatalf("built fresh components = %d, want 0", got)
	}
}

func TestBuildDeclaredComponents_RebuildsStaleUIOnly(t *testing.T) {
	installTestBuilder(t)
	if got := runBuildLedgerTest(t, testBuildScenario(t), false, "ui"); got != 1 {
		t.Fatalf("built components = %d, want 1", got)
	}
}

func TestBuildDeclaredComponents_RebuildsBothWhenBothStale(t *testing.T) {
	installTestBuilder(t)
	if got := runBuildLedgerTest(t, testBuildScenario(t), false, "api", "ui"); got != 2 {
		t.Fatalf("built components = %d, want 2", got)
	}
}

func TestBuildDeclaredComponents_ForceRebuildsAll(t *testing.T) {
	installTestBuilder(t)
	if got := runBuildLedgerTest(t, testBuildScenario(t), true); got != 2 {
		t.Fatalf("built components = %d, want 2", got)
	}
}

func TestBuildDeclaredComponents_IndependentComponentsOverlap(t *testing.T) {
	installTimedTestBuilder(t, 1, false)
	start := time.Now()
	if got := runBuildLedgerTest(t, testBuildScenario(t), false, "api", "ui"); got != 2 {
		t.Fatalf("built components = %d, want 2", got)
	}
	if elapsed := time.Since(start); elapsed >= 550*time.Millisecond {
		t.Fatalf("independent builds took %s; expected overlap", elapsed)
	}
}

func TestBuildDeclaredComponents_BudgetOfOneSerialises(t *testing.T) {
	installTimedTestBuilder(t, 1<<62, false)
	start := time.Now()
	if got := runBuildLedgerTest(t, testBuildScenario(t), false, "api", "ui"); got != 2 {
		t.Fatalf("built components = %d, want 2", got)
	}
	if elapsed := time.Since(start); elapsed < 550*time.Millisecond {
		t.Fatalf("budget-one builds took %s; expected serialization", elapsed)
	}
}

func TestBuildDeclaredComponents_FailureCancelsSiblings(t *testing.T) {
	installTimedTestBuilder(t, 1, true)
	_, err := (&Runner{Root: t.TempDir(), Home: t.TempDir()}).buildDeclaredComponents(context.Background(), testBuildScenario(t), nil, io.Discard, false, map[string]artifactVerdict{"api": {Stale: true}, "ui": {Stale: true}})
	if err == nil || !strings.Contains(err.Error(), "component api") {
		t.Fatalf("error = %v, want component api failure", err)
	}
}

func TestBuilderRegistryDeclaresImplementedAndReservedKinds(t *testing.T) {
	registry := BuilderRegistry()
	want := []string{"cargo", "go_module", "node_bundle", "pnpm_vite", "python_uv"}
	if got := RegisteredBuilderKinds(); !reflect.DeepEqual(got, want) {
		t.Fatalf("RegisteredBuilderKinds() = %v, want %v", got, want)
	}
	for _, kind := range []string{"go_module", "pnpm_vite", "node_bundle"} {
		if registry[kind].Reserved {
			t.Fatalf("%s unexpectedly reserved", kind)
		}
	}
	if registry["python_uv"].Reserved {
		t.Fatal("python_uv must be executable once registered")
	}
	if !registry["cargo"].Reserved {
		t.Fatal("cargo must remain reserved until it has an adopter")
	}
	if registry["go_module"].Environment["GOWORK"] != "off" {
		t.Fatalf("go_module must preserve the executor's GOWORK=off build environment")
	}
}

func TestBuilderRegistryMatchesServiceSchema(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".vrooli", "schemas", "service.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	definitions := document["definitions"].(map[string]any)
	componentBuild := definitions["componentBuild"].(map[string]any)
	enum := componentBuild["properties"].(map[string]any)["kind"].(map[string]any)["enum"].([]any)
	got := make([]string, len(enum))
	for i, value := range enum {
		got[i] = value.(string)
	}
	// "reuse" remains in the schema for backward-compatible diagnostics, but
	// is intentionally not an executable registry row.
	got = slices.DeleteFunc(got, func(kind string) bool { return kind == "reuse" })
	if want := RegisteredBuilderKinds(); !reflect.DeepEqual(got, want) {
		t.Fatalf("service schema builder kinds = %v, registry = %v", got, want)
	}
}

func TestBuilderDigestKeysAreEmitted(t *testing.T) {
	deps := hostProbeDeps{
		getenv: func(name string) string {
			if name == "NODE_ENV" {
				return "development"
			}
			if name == BuildModeEnvVar {
				return "profile"
			}
			return ""
		},
		lookPath:      exec.LookPath,
		nodeVersion:   func() string { return "22" },
		pythonVersion: func() string { return "3.11" },
		uvVersion:     func() string { return "0.8" },
		goEnv: func(keys ...string) map[string]string {
			values := make(map[string]string, len(keys))
			for _, key := range keys {
				values[key] = "set"
			}
			return values
		},
	}
	deps.cache = &hostProbeCache{}
	for kind, spec := range BuilderRegistry() {
		if spec.Reserved {
			continue
		}
		keys := builderFreshnessKeys(spec, deps)
		for _, declared := range spec.DigestKeys {
			key := strings.ToLower(declared)
			if _, ok := keys[key]; !ok {
				t.Errorf("builder %s declares digest key %q but resolver emitted %v", kind, declared, keys)
			}
		}
	}
}

func TestInstallInputDigestGateSkipsUnchangedInputs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{"go.mod": "module example\n", "go.sum": "sum-v1\n"} {
		if err := os.WriteFile(filepath.Join(root, "api", name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	component := scenario.Component{Build: scenario.ComponentBuild{Kind: "go_module", Dir: "api"}}
	spec := BuilderSpec{Kind: "go_module", Install: []string{"go", "mod", "download"}, InstallInputs: []string{"go.mod", "go.sum"}}
	needed, _, err := installNeeded(root, root, component, spec)
	if err != nil || !needed {
		t.Fatalf("first installNeeded = %t, %v; want true", needed, err)
	}
	if err := recordInstallDigest(root, root, component, spec); err != nil {
		t.Fatal(err)
	}
	needed, _, err = installNeeded(root, root, component, spec)
	if err != nil || needed {
		t.Fatalf("unchanged installNeeded = %t, %v; want false", needed, err)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "go.sum"), []byte("sum-v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	needed, _, err = installNeeded(root, root, component, spec)
	if err != nil || !needed {
		t.Fatalf("changed installNeeded = %t, %v; want true", needed, err)
	}
}

func TestInstallInputDigestIncludesGovernedSharedPackageOutputs(t *testing.T) {
	repoRoot := t.TempDir()
	for _, name := range []string{"common.schema.json", "package.schema.json"} {
		data, err := os.ReadFile(filepath.Join("..", "..", ".vrooli", "schemas", name))
		if err != nil {
			t.Fatal(err)
		}
		schemaDir := filepath.Join(repoRoot, ".vrooli", "schemas")
		if err := os.MkdirAll(schemaDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(schemaDir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	packagefixture.WritePackageManifest(t, repoRoot, "shared", packagefixture.PackageManifest(
		"shared",
		packagefixture.WithPackageModuleIdentifiers("@local/shared"),
		packagefixture.WithPackageGenerateCommands(packagegov.CommandSpec{
			Name:    "build",
			Run:     []string{"echo", "build"},
			Outputs: []string{"dist/index.js"},
		}),
	))
	sharedOutput := filepath.Join(repoRoot, "packages", "shared", "dist", "index.js")
	if err := os.MkdirAll(filepath.Dir(sharedOutput), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sharedOutput, []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	scenarioRoot := filepath.Join(repoRoot, "scenarios", "demo")
	uiRoot := filepath.Join(scenarioRoot, "ui")
	if err := os.MkdirAll(uiRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	packagefixture.WriteNodePackageManifest(t, filepath.Join(uiRoot, "package.json"), packagefixture.NodePackageManifest{
		Name: "demo-ui",
		Dependencies: map[string]string{
			"@local/shared": "file:../../../packages/shared",
		},
	})

	component := scenario.Component{Build: scenario.ComponentBuild{Kind: "pnpm_vite", Dir: "ui"}}
	spec := BuilderSpec{
		Kind:                     "pnpm_vite",
		Install:                  []string{"pnpm", "install"},
		InstallInputs:            []string{"package.json"},
		FollowsWorkspaceFileDeps: true,
	}
	needed, _, err := installNeeded(repoRoot, scenarioRoot, component, spec)
	if err != nil || !needed {
		t.Fatalf("first installNeeded = %t, %v; want true", needed, err)
	}
	if err := recordInstallDigest(repoRoot, scenarioRoot, component, spec); err != nil {
		t.Fatal(err)
	}
	needed, _, err = installNeeded(repoRoot, scenarioRoot, component, spec)
	if err != nil || needed {
		t.Fatalf("unchanged installNeeded = %t, %v; want false", needed, err)
	}
	if err := os.WriteFile(sharedOutput, []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	needed, _, err = installNeeded(repoRoot, scenarioRoot, component, spec)
	if err != nil || !needed {
		t.Fatalf("shared output change installNeeded = %t, %v; want true", needed, err)
	}
}

func TestResolveComponentArgvRejectsDeprecatedBinaryReuse(t *testing.T) {
	components := map[string]scenario.Component{
		"api": {
			Build: scenario.ComponentBuild{Kind: "go_module", Dir: "api"},
		},
		"registry": {
			Build: scenario.ComponentBuild{Reuse: "api"},
		},
	}
	root := filepath.Join("workspace", "scenario-to-mcp")
	if _, err := resolveComponentArgvForOS([]string{"{{bin.registry}}", "helper{{ext}}"}, root, "scenario-to-mcp", components, "windows"); err == nil || !strings.Contains(err.Error(), "declare the artifact once") {
		t.Fatalf("resolveComponentArgvForOS error = %v, want deprecated reuse guidance", err)
	}
}

func TestResolveComponentArgvRejectsDeprecatedReuse(t *testing.T) {
	components := map[string]scenario.Component{
		"api":      {Build: scenario.ComponentBuild{Reuse: "registry"}},
		"registry": {Build: scenario.ComponentBuild{Reuse: "api"}},
	}
	if _, err := resolveComponentArgvForOS([]string{"{{bin.api}}"}, "/scenario", "demo", components, "linux"); err == nil || !strings.Contains(err.Error(), "deprecated build.reuse") {
		t.Fatalf("expected deprecated build.reuse rejection, got %v", err)
	}
}

func TestComponentBuildTargetsIncludesDeclaredSecondaryArtifacts(t *testing.T) {
	component := scenario.Component{Build: scenario.ComponentBuild{
		Kind:  "go_module",
		Dir:   "api",
		Entry: ".",
		Outputs: []scenario.ComponentBuildOutput{
			{Entry: "./cmd/workspace-sandbox-launcher", Output: "api/workspace-sandbox-launcher"},
		},
	}}
	targets, err := componentBuildTargets("api", "/repo/scenarios/workspace-sandbox", "workspace-sandbox", component, builderRegistry["go_module"], "darwin")
	if err != nil {
		t.Fatalf("componentBuildTargets: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("target count = %d, want 2", len(targets))
	}
	if got, want := targets[0].Entry, "."; got != want {
		t.Fatalf("primary entry = %q, want %q", got, want)
	}
	if got, want := targets[0].Output, "/repo/scenarios/workspace-sandbox/api/workspace-sandbox-api"; got != want {
		t.Fatalf("primary output = %q, want %q", got, want)
	}
	if got, want := targets[1].Entry, "./cmd/workspace-sandbox-launcher"; got != want {
		t.Fatalf("secondary entry = %q, want %q", got, want)
	}
	if got, want := targets[1].Output, "/repo/scenarios/workspace-sandbox/api/workspace-sandbox-launcher"; got != want {
		t.Fatalf("secondary output = %q, want %q", got, want)
	}
}

func TestGoModuleFreshnessIncludesDeclaredSecondaryArtifacts(t *testing.T) {
	root := t.TempDir()
	component := scenario.Component{Build: scenario.ComponentBuild{
		Kind: "go_module",
		Dir:  "api",
		Outputs: []scenario.ComponentBuildOutput{
			{Entry: "./cmd/helper", Output: "api/helper"},
		},
	}}
	artifacts, err := componentFreshnessArtifacts(root, root, component, defaultHostProbeDeps())
	if err != nil {
		t.Fatalf("componentFreshnessArtifacts: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("freshness artifact count = %d, want 2", len(artifacts))
	}
	if got, want := artifacts[1].Target, "api/helper"; got != want {
		t.Fatalf("secondary freshness target = %q, want %q", got, want)
	}
}

func TestDeclaredCommandForComponentOwnsArgvEnvironmentAndPort(t *testing.T) {
	root := t.TempDir()
	item := scenario.Scenario{
		Slug: "alpha",
		Path: root,
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}},
			Components: map[string]scenario.Component{
				"api": {
					Build: scenario.ComponentBuild{Kind: "go_module", Dir: "api"},
					Run: scenario.ComponentRun{
						Argv: []string{"{{bin.api}}", "--port", "${API_PORT}"},
						CWD:  "api",
						Env:  map[string]string{"SERVICE_URL": "http://127.0.0.1:${API_PORT}"},
						Port: "api",
					},
				},
			},
		},
	}
	command, direct, err := declaredCommandForStep(item, scenario.PhaseStep{Name: "start-api"}, []string{"API_PORT=18080", "PATH=/bin"})
	if err != nil {
		t.Fatalf("declaredCommandForStep: %v", err)
	}
	if !direct {
		t.Fatal("component command did not select direct execution")
	}
	wantArgv := []string{filepath.Join(root, "api", "alpha-api"), "--port", "18080"}
	if !reflect.DeepEqual(command.Argv, wantArgv) {
		t.Fatalf("argv = %v, want %v", command.Argv, wantArgv)
	}
	if command.Dir != filepath.Join(root, "api") || command.Port != 18080 || command.PortKey != "API_PORT" {
		t.Fatalf("command = %#v", command)
	}
	if got := envValue(command.Env, "SERVICE_URL"); got != "http://127.0.0.1:18080" {
		t.Fatalf("SERVICE_URL = %q", got)
	}
}

func TestDeclaredDevelopPhaseRetainsComponentArgv(t *testing.T) {
	manifest := scenario.ServiceManifest{
		Components: map[string]scenario.Component{
			"api": {
				Role: "api",
				Run:  scenario.ComponentRun{Argv: []string{"{{bin.api}}"}},
			},
			"ui": {
				Role: "ui",
				Run:  scenario.ComponentRun{Argv: []string{"node", "server.js"}},
			},
			"sidecar": {
				Role: "sidecar",
				Run:  scenario.ComponentRun{Argv: []string{"node", "sidecar.js"}, SupervisedBy: "api"},
			},
		},
	}

	steps := declaredPhaseSteps(manifest, "develop", nil)
	if len(steps) != 2 {
		t.Fatalf("develop step count = %d, want 2 unsupervised components", len(steps))
	}
	for _, step := range steps {
		if len(step.Exec) == 0 {
			t.Fatalf("component step %q lost its declared argv", step.Name)
		}
		if !step.Background {
			t.Fatalf("component step %q must run in the background", step.Name)
		}
	}
	if got, want := steps[0].Exec, []string{"{{bin.api}}"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("api argv = %v, want %v", got, want)
	}
	if got, want := steps[1].Exec, []string{"node", "server.js"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ui argv = %v, want %v", got, want)
	}
}

func TestDeclaredCommandForTypedProvisioningStepRejectsShellPlaceholder(t *testing.T) {
	item := scenario.Scenario{Slug: "alpha", Path: t.TempDir()}
	step := scenario.PhaseStep{Name: "prepare", Exec: []string{"tool", "${resource.ollama.enabled}"}}
	if _, _, err := declaredCommandForStep(item, step, []string{"PATH=/bin"}); err == nil {
		t.Fatal("expected non-contract placeholder to fail")
	}
}

func TestPNPMViteRegistryOwnsAgentInboxFreshnessContract(t *testing.T) {
	t.Setenv("NODE_ENV", "development")
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	appRoot := filepath.Join(repoRoot, "scenarios", "agent-inbox")
	deps := defaultHostProbeDeps()
	component := scenario.Component{Build: scenario.ComponentBuild{Kind: "pnpm_vite", Dir: "ui"}}

	derived, err := componentFreshnessArtifacts(appRoot, repoRoot, component, deps)
	if err != nil {
		t.Fatalf("componentFreshnessArtifacts: %v", err)
	}
	if len(derived) != 1 {
		t.Fatalf("derived artifact count = %d, want 1", len(derived))
	}
	artifact := derived[0]
	if artifact.Target != "ui/dist/index.html" || artifact.CheckType != "pnpm_vite" {
		t.Fatalf("registry freshness = %#v", artifact)
	}
	if artifact.Spec.SourceRoot != filepath.Join(appRoot, "ui") || artifact.KeyInputs["node_env"] == "" {
		t.Fatalf("registry source/key inputs = %#v / %#v", artifact.Spec, artifact.KeyInputs)
	}
}

func TestComponentReadinessSupportsPortOpen(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	manifest := scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}}
	component := scenario.Component{Run: scenario.ComponentRun{
		Port:      "api",
		Readiness: &scenario.ComponentReadiness{Type: "port_open", TimeoutMS: 1000},
	}}
	if err := checkComponentReadiness(manifest, component, map[string]string{"API_PORT": fmt.Sprint(port)}); err != nil {
		t.Fatalf("checkComponentReadiness: %v", err)
	}
}

func TestCreateComponentRuntimeDirectories(t *testing.T) {
	root := t.TempDir()
	component := scenario.Component{Run: scenario.ComponentRun{DataDirs: []string{"data/cache"}, LogDir: "logs"}}
	if err := createComponentRuntimeDirectories(root, component); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{"data/cache", "logs"} {
		if info, err := os.Stat(filepath.Join(root, relative)); err != nil || !info.IsDir() {
			t.Fatalf("runtime directory %s: info=%v err=%v", relative, info, err)
		}
	}
}

// TestBuildArgvSelectsProfileChannel: VROOLI_BUILD_MODE=profile selects the
// declared profile channel argv for the pnpm builders, and every argv stays
// shell-free so the channel is reachable where package scripts run through
// cmd.exe rather than a POSIX shell.
func TestBuildArgvSelectsProfileChannel(t *testing.T) {
	registry := BuilderRegistry()
	for _, kind := range []string{"pnpm_vite", "node_bundle"} {
		spec := registry[kind]
		if got, want := spec.BuildArgv(""), []string{"pnpm", "run", "build"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("%s default channel = %v, want %v", kind, got, want)
		}
		if got, want := spec.BuildArgv("profile"), []string{"pnpm", "run", "build:profile"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("%s profile channel = %v, want %v", kind, got, want)
		}
	}
	// A builder with no declared profile channel keeps its default argv.
	goSpec := registry["go_module"]
	if got, want := goSpec.BuildArgv("profile"), goSpec.Build; !reflect.DeepEqual(got, want) {
		t.Fatalf("go_module profile channel = %v, want the default %v", got, want)
	}
}

func TestGoModuleBuildArgvUsesRepositoryFlagPolicy(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	contract := `{"build.go_flags":{"develop":["-trimpath"],"distribution":["-trimpath","-buildvcs=false"],"scenario":["-trimpath","-buildvcs=false"]}}`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := goModuleBuildArgvForRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"go", "build", "-trimpath", "-buildvcs=false", "-o", "{output}", "{entry}"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("policy-aware Go argv = %v, want %v", got, want)
	}
}

// TestBuildArgvIsShellFree: no builder channel may carry a shell metacharacter.
// A conditional inside a package script is exactly what made the perf-build
// channel unreachable on Windows; the selection belongs in Go.
func TestBuildArgvIsShellFree(t *testing.T) {
	shellTokens := []string{"$(", "`", "&&", "||", ";", "|", ">", "<", "[ "}
	for kind, spec := range BuilderRegistry() {
		for _, mode := range []string{"", "profile"} {
			for _, arg := range append(spec.BuildArgv(mode), spec.Install...) {
				for _, token := range shellTokens {
					if strings.Contains(arg, token) {
						t.Errorf("%s (mode %q) argv %q contains shell token %q", kind, mode, arg, token)
					}
				}
			}
		}
	}
}

// TestNormalizeBuildMode: only "profile" is a channel; anything else is the
// default build, so an operator typo never selects a package script that no
// scenario is required to declare.
func TestNormalizeBuildMode(t *testing.T) {
	for raw, want := range map[string]string{
		"profile": "profile", " profile ": "profile", "PROFILE": "profile",
		"": "", "prof": "", "production": "", "true": "",
	} {
		if got := NormalizeBuildMode(raw); got != want {
			t.Errorf("NormalizeBuildMode(%q) = %q, want %q", raw, got, want)
		}
	}
}

// TestBuildModeForEnvPrefersOverride: a lifecycle override pins the channel;
// otherwise the process environment decides.
func TestBuildModeForEnvPrefersOverride(t *testing.T) {
	getenv := func(string) string { return "profile" }
	if got := BuildModeForEnv(map[string]string{BuildModeEnvVar: ""}, getenv); got != "" {
		t.Fatalf("explicit empty override = %q, want the default channel", got)
	}
	if got := BuildModeForEnv(nil, getenv); got != "profile" {
		t.Fatalf("process env = %q, want profile", got)
	}
	if got := BuildModeForEnv(nil, nil); got != "" {
		t.Fatalf("no env source = %q, want the default channel", got)
	}
}
