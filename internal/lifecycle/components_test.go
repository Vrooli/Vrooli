package lifecycle

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
)

func TestBuilderRegistryDeclaresImplementedAndReservedKinds(t *testing.T) {
	registry := BuilderRegistry()
	want := []string{"cargo", "go_module", "node_bundle", "pnpm_vite", "python_uv", "reuse"}
	if got := RegisteredBuilderKinds(); !reflect.DeepEqual(got, want) {
		t.Fatalf("RegisteredBuilderKinds() = %v, want %v", got, want)
	}
	for _, kind := range []string{"go_module", "pnpm_vite", "node_bundle", "reuse"} {
		if registry[kind].Reserved {
			t.Fatalf("%s unexpectedly reserved", kind)
		}
	}
	for _, kind := range []string{"python_uv", "cargo"} {
		if !registry[kind].Reserved {
			t.Fatalf("%s must remain reserved until it has an adopter", kind)
		}
	}
	if registry["go_module"].Environment["GOWORK"] != "off" {
		t.Fatalf("go_module must preserve the executor's GOWORK=off build environment")
	}
}

func TestResolveComponentArgvResolvesBinaryReuseAndExtension(t *testing.T) {
	components := map[string]scenario.Component{
		"api": {
			Build: scenario.ComponentBuild{Kind: "go_module", Dir: "api"},
		},
		"registry": {
			Build: scenario.ComponentBuild{Reuse: "api"},
		},
	}
	root := filepath.Join("workspace", "scenario-to-mcp")
	got, err := resolveComponentArgvForOS([]string{"{{bin.registry}}", "helper{{ext}}"}, root, "scenario-to-mcp", components, "windows")
	if err != nil {
		t.Fatalf("resolveComponentArgvForOS: %v", err)
	}
	want := []string{
		filepath.Join(root, "api", "scenario-to-mcp-api.exe"),
		"helper.exe",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv = %v, want %v", got, want)
	}
}

func TestResolveComponentArgvRejectsReuseCycle(t *testing.T) {
	components := map[string]scenario.Component{
		"api":      {Build: scenario.ComponentBuild{Reuse: "registry"}},
		"registry": {Build: scenario.ComponentBuild{Reuse: "api"}},
	}
	if _, err := resolveComponentArgvForOS([]string{"{{bin.api}}"}, "/scenario", "demo", components, "linux"); err == nil {
		t.Fatal("expected build.reuse cycle to fail")
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
		},
	}

	steps := declaredPhaseSteps(manifest, "develop", nil)
	if len(steps) != 2 {
		t.Fatalf("develop step count = %d, want 2", len(steps))
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
