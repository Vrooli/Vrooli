package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	runtimeManifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/config"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

func writeBundleGraphOwner(t *testing.T, dir, name string, deps map[string]scenariomodel.Dependency) {
	t.Helper()
	cfg := scenariomodel.ServiceManifest{
		Service:      scenariomodel.ServiceMetadata{Name: name, Version: "1.0.0"},
		Dependencies: scenariomodel.Dependencies{Scenarios: deps},
		Components: map[string]scenariomodel.Component{
			"api": {Role: "api", Build: scenariomodel.ComponentBuild{Kind: "go_module", Dir: "api"}},
			"ui":  {Role: "ui", Build: scenariomodel.ComponentBuild{Kind: "pnpm_vite", Dir: "ui"}},
		},
	}
	path := filepath.Join(dir, name, ".vrooli")
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "service.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "desktop-features.json"), []byte(`{"work":{"service":"api","port":"api"}}`), 0600); err != nil {
		t.Fatal(err)
	}
}

// A discovered branch can encounter an owner before the branch that embeds it.
// The latter must retain that owner's children, UI request and feature exports.
func TestFreshBundleManifestPreservesSharedOwnerSemantics(t *testing.T) {
	for _, cycle := range []bool{false, true} {
		t.Run(fmt.Sprintf("cycle=%t", cycle), func(t *testing.T) {
			dir := t.TempDir()
			writeBundleGraphOwner(t, dir, "root", map[string]scenariomodel.Dependency{
				"a-discovered": {BundlePolicy: "discover"},
				"z-embedded":   {BundlePolicy: "embed", Required: true},
			})
			writeBundleGraphOwner(t, dir, "a-discovered", map[string]scenariomodel.Dependency{"shared": {BundlePolicy: "discover", VersionRange: ">=1.0.0"}})
			writeBundleGraphOwner(t, dir, "z-embedded", map[string]scenariomodel.Dependency{"shared": {BundlePolicy: "embed", Required: true, IncludeUI: true, VersionRange: ">=1.0.0"}})
			writeBundleGraphOwner(t, dir, "shared", map[string]scenariomodel.Dependency{"leaf": {BundlePolicy: "embed", Required: true}})
			var back map[string]scenariomodel.Dependency
			if cycle {
				back = map[string]scenariomodel.Dependency{"root": {BundlePolicy: "discover"}}
			}
			writeBundleGraphOwner(t, dir, "leaf", back)
			path := filepath.Join(dir, "root")
			cfg, err := config.LoadServiceConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			nodes := BuildDependencyNodeList(dir, "root", cfg, map[string]struct{}{"root": {}})
			old := BuildBundleManifest("root", path, time.Time{}, nodes, cfg)
			fresh, err := BuildFreshBundleManifest(context.Background(), "root", path, dir, cfg)
			if err != nil {
				t.Fatal(err)
			}
			fresh.GeneratedAt = time.Time{}
			if !jsonEqual(old, fresh) {
				a, _ := json.Marshal(old)
				b, _ := json.Marshal(fresh)
				t.Fatalf("manifest semantics changed\nexpanded=%s\nbounded=%s", a, b)
			}
			services := map[string]bool{}
			for _, service := range fresh.Skeleton.Services {
				services[service.ID] = true
			}
			for _, id := range []string{"shared--api", "shared--ui", "leaf--api"} {
				if !services[id] {
					t.Fatalf("required transitive/shared service %q lost", id)
				}
			}
			if services["a-discovered--api"] {
				t.Fatal("discovered-only service was bundled")
			}
		})
	}
}

func TestFreshBundleManifestBoundsRepeatedFanIn(t *testing.T) {
	dir := t.TempDir()
	const levels = 18
	for level := levels - 1; level >= 0; level-- {
		for _, side := range []string{"a", "b"} {
			deps := map[string]scenariomodel.Dependency{}
			if level+1 < levels {
				for _, next := range []string{"a", "b"} {
					deps[fmt.Sprintf("n%02d%s", level+1, next)] = scenariomodel.Dependency{BundlePolicy: "embed", Required: true}
				}
			}
			writeBundleGraphOwner(t, dir, fmt.Sprintf("n%02d%s", level, side), deps)
		}
	}
	writeBundleGraphOwner(t, dir, "root", map[string]scenariomodel.Dependency{"n00a": {BundlePolicy: "embed"}, "n00b": {BundlePolicy: "embed"}})
	path := filepath.Join(dir, "root")
	cfg, err := config.LoadServiceConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	graph := &bundleDependencyGraph{ctx: ctx, children: map[string][]types.DeploymentDependencyNode{}}
	nodes := BuildDependencyNodeListWithOptions(dir, "root", cfg, map[string]struct{}{"root": {}}, DependencyBuildOptions{bundleGraph: graph})
	count := 0
	var walk func([]types.DeploymentDependencyNode)
	walk = func(nodes []types.DeploymentDependencyNode) {
		for _, n := range nodes {
			count++
			walk(n.Children)
		}
	}
	walk(nodes)
	if count > 4*levels || len(graph.children) != 2*levels {
		t.Fatalf("fan-in expanded paths: nodes=%d owners=%d", count, len(graph.children))
	}
	fresh, err := BuildFreshBundleManifest(ctx, "root", path, dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(fresh.Skeleton.Services) != 2+2*levels {
		t.Fatalf("lost owner services: %d", len(fresh.Skeleton.Services))
	}
	raw, err := json.Marshal(fresh)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > 500000 {
		t.Fatalf("manifest grew with dependency paths: %d bytes", len(raw))
	}
	t.Logf("18-level fan-in: %d owners, %d retained edges, %d manifest bytes", len(graph.children), count, len(raw))
}

func TestBuildReportBoundsRepeatedFanIn(t *testing.T) {
	dir := t.TempDir()
	const levels = 8
	for level := levels - 1; level >= 0; level-- {
		for _, side := range []string{"a", "b"} {
			deps := map[string]scenariomodel.Dependency{}
			if level+1 < levels {
				for _, next := range []string{"a", "b"} {
					deps[fmt.Sprintf("n%02d%s", level+1, next)] = scenariomodel.Dependency{Required: true}
				}
			}
			writeBundleGraphOwner(t, dir, fmt.Sprintf("n%02d%s", level, side), deps)
		}
	}
	writeBundleGraphOwner(t, dir, "root", map[string]scenariomodel.Dependency{"n00a": {Required: true}, "n00b": {Required: true}})
	path := filepath.Join(dir, "root")
	cfg, err := config.LoadServiceConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	report := BuildReport("root", path, dir, cfg)
	count := 0
	var walk func([]types.DeploymentDependencyNode)
	walk = func(nodes []types.DeploymentDependencyNode) {
		for _, node := range nodes {
			count++
			walk(node.Children)
		}
	}
	walk(report.Dependencies)
	if count > 4*levels {
		t.Fatalf("report expanded shared dependency paths: %d nodes for %d owners", count, 2*levels)
	}
	if len(report.BundleManifest.Dependencies) != 2*levels {
		t.Fatalf("lost unique dependency owners: %d", len(report.BundleManifest.Dependencies))
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > 400000 {
		t.Fatalf("report output grew with dependency paths: %d bytes", len(raw))
	}
	t.Logf("%d owners, %d retained edges, %d report bytes", 2*levels, count, len(raw))
}

func TestFreshBundleManifestHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	manifest, err := BuildFreshBundleManifest(ctx, "root", "absent", "absent", &types.Manifest{})
	if !errors.Is(err, context.Canceled) || manifest != nil {
		t.Fatalf("cancelled query emitted manifest: %v %v", manifest, err)
	}
}

func TestFreshBundleManifestRevisitsCycleThroughEmbeddedBranch(t *testing.T) {
	dir := t.TempDir()
	writeBundleGraphOwner(t, dir, "root", map[string]scenariomodel.Dependency{"a": {BundlePolicy: "discover"}, "z": {BundlePolicy: "embed"}})
	writeBundleGraphOwner(t, dir, "a", map[string]scenariomodel.Dependency{"b": {BundlePolicy: "embed"}, "leaf": {BundlePolicy: "embed"}})
	writeBundleGraphOwner(t, dir, "b", map[string]scenariomodel.Dependency{"a": {BundlePolicy: "embed"}})
	writeBundleGraphOwner(t, dir, "z", map[string]scenariomodel.Dependency{"b": {BundlePolicy: "embed"}})
	writeBundleGraphOwner(t, dir, "leaf", nil)
	path := filepath.Join(dir, "root")
	cfg, err := config.LoadServiceConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	nodes := BuildDependencyNodeList(dir, "root", cfg, map[string]struct{}{"root": {}})
	old := BuildBundleManifest("root", path, time.Time{}, nodes, cfg)
	fresh, err := BuildFreshBundleManifest(context.Background(), "root", path, dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	fresh.GeneratedAt = time.Time{}
	if !jsonEqual(old, fresh) {
		t.Fatal("later embedded path through a previously encountered cycle lost manifest facts")
	}
}

func TestEmbedPeerServicesProjectsDeclaredPeer(t *testing.T) {
	peerRoot := t.TempDir()
	peerConfig := scenariomodel.ServiceManifest{
		Service: scenariomodel.ServiceMetadata{Name: "helper"},
		Ports:   map[string]scenariomodel.Port{"api": {EnvVar: "API_PORT", Range: "24000-24100"}},
		Components: map[string]scenariomodel.Component{"api": {
			Role:  "api",
			Build: scenariomodel.ComponentBuild{Kind: "go_module", Dir: "api"},
			Run:   scenariomodel.ComponentRun{Argv: []string{"{{bin.api}}"}, Port: "api"},
		}},
	}
	payload, err := json.Marshal(peerConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(peerRoot, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(peerRoot, ".vrooli", "service.json"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &types.Manifest{Dependencies: scenariomodel.Dependencies{Scenarios: map[string]scenariomodel.Dependency{
		"helper": {
			BundlePolicy: "embed",
		},
	}}}
	root := []types.BundleSkeletonService{{ID: "consumer", Env: map[string]string{}}}
	services := embedPeerServices(root, cfg, []types.DeploymentDependencyNode{{Name: "helper", Type: "scenario", Path: peerRoot}})
	if len(services) != 2 || services[1].ID != "helper--api" {
		t.Fatalf("embedded services = %#v", services)
	}
	if _, ok := services[0].Env["HELPER_URL"]; ok {
		t.Fatal("embedded peer binding was projected into the consumer environment")
	}
	if !reflect.DeepEqual(services[0].Dependencies, []string(nil)) {
		t.Fatalf("embedded dependency projection = %#v", services[0].Dependencies)
	}
}

func TestEmbeddedPeerProjectionRecursesAndHonorsIncludeUI(t *testing.T) {
	tmp := t.TempDir()
	scenarios := filepath.Join(tmp, "scenarios")
	rootPath := filepath.Join(scenarios, "desktop-app")
	peerPath := filepath.Join(scenarios, "helper")
	secondPath := filepath.Join(scenarios, "second-helper")
	nestedPath := filepath.Join(scenarios, "nested")
	writeServiceConfig := func(path, name string, dependencies map[string]scenariomodel.Dependency, components map[string]scenariomodel.Component) {
		t.Helper()
		manifest := scenariomodel.ServiceManifest{Service: scenariomodel.ServiceMetadata{Name: name, Version: "1.4.0"}, Dependencies: scenariomodel.Dependencies{Scenarios: dependencies}, Components: components}
		data, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		configPath := filepath.Join(path, ".vrooli", "service.json")
		if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(configPath, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	components := map[string]scenariomodel.Component{
		"api": {Role: "api", Build: scenariomodel.ComponentBuild{Kind: "go_module", Dir: "api"}},
		"ui":  {Role: "ui", Build: scenariomodel.ComponentBuild{Kind: "pnpm_vite", Dir: "ui"}},
	}
	writeServiceConfig(rootPath, "desktop-app", map[string]scenariomodel.Dependency{
		"helper":        {BundlePolicy: "embed", IncludeUI: false},
		"second-helper": {BundlePolicy: "embed", IncludeUI: false},
	}, nil)
	writeServiceConfig(peerPath, "helper", map[string]scenariomodel.Dependency{"nested": {BundlePolicy: "embed", IncludeUI: false}}, components)
	writeServiceConfig(secondPath, "second-helper", map[string]scenariomodel.Dependency{"nested": {BundlePolicy: "embed", IncludeUI: true}}, components)
	writeServiceConfig(nestedPath, "nested", nil, components)
	for _, path := range []string{filepath.Join(peerPath, "api"), filepath.Join(peerPath, "ui"), filepath.Join(secondPath, "api"), filepath.Join(secondPath, "ui"), filepath.Join(nestedPath, "api"), filepath.Join(nestedPath, "ui", "dist")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(nestedPath, "ui", "dist", "index.html"), []byte("<html></html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadServiceConfig(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	secondConfig, err := config.LoadServiceConfig(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("second-helper dependencies: %#v", secondConfig.Dependencies.Scenarios)
	nodes := []types.DeploymentDependencyNode{
		{Name: "helper", Type: "scenario", Path: peerPath, Children: []types.DeploymentDependencyNode{{Name: "nested", Type: "scenario", Path: nestedPath}}},
		{Name: "second-helper", Type: "scenario", Path: secondPath, Children: []types.DeploymentDependencyNode{{Name: "nested", Type: "scenario", Path: nestedPath}}},
	}
	services := embedPeerServices(buildSkeletonServices(cfg), cfg, nodes)
	ids := make(map[string]types.BundleSkeletonService)
	for _, service := range services {
		ids[service.ID] = service
	}
	if _, ok := ids["helper--api"]; !ok {
		t.Fatal("embedded peer API service missing")
	}
	if _, ok := ids["helper--ui"]; ok {
		t.Fatal("peer UI was included without include_ui")
	}
	if nested, ok := ids["nested--api"]; !ok || nested.Build == nil || nested.Build.SourceDir != "peers/nested/api" {
		t.Fatalf("transitive peer service = %#v, want source under peers/nested", nested)
	}
	if _, ok := ids["nested--ui"]; !ok {
		t.Fatalf("shared peer UI omitted although one included dependency edge requested it: %#v", ids)
	}
	peers := derivePeers(cfg, nodes)
	if len(peers) != 3 {
		t.Fatalf("recursive peer metadata = %#v", peers)
	}
	files := discoverBundleFiles("desktop-app", rootPath, cfg, nodes)
	if !bundleFileExists(files, "peers/helper/api") || !bundleFileExists(files, "peers/nested/api") || !bundleFileExists(files, "peers/nested/ui/dist/index.html") {
		t.Fatalf("recursive peer source files missing: %#v", files)
	}
}

func bundleFileExists(files []types.BundleFileEntry, path string) bool {
	for _, file := range files {
		if file.Path == path && file.Exists {
			return true
		}
	}
	return false
}

func TestProgramBindingPeerCarriesDeclaredVersionRange(t *testing.T) {
	cfg := &types.Manifest{Dependencies: scenariomodel.Dependencies{Scenarios: map[string]scenariomodel.Dependency{
		"vrooli-memory": {BundlePolicy: "discover", VersionRange: ">=1.2.0 <2.0.0"},
	}}}
	nodes := []types.DeploymentDependencyNode{{
		Name:   "vrooli-memory",
		Type:   "scenario",
		Source: "program-binding",
	}}
	peers := deriveProgramBindingPeers(cfg, nodes)
	if len(peers) != 1 || peers[0].VersionRange != ">=1.2.0 <2.0.0" {
		t.Fatalf("program-binding peer projection = %#v", peers)
	}
}

func TestPeerCarriesDeclaredVersionRange(t *testing.T) {
	cfg := &types.Manifest{Dependencies: scenariomodel.Dependencies{Scenarios: map[string]scenariomodel.Dependency{
		"landing-page-business-suite": {BundlePolicy: "discover", VersionRange: ">=1.0.0"},
	}}}
	peers := derivePeers(cfg)
	if len(peers) != 1 || peers[0].VersionRange != ">=1.0.0" {
		t.Fatalf("peer projection = %#v", peers)
	}
}

func TestRealScenarioComponentProjectionFidelity(t *testing.T) {
	repoRoot, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		serviceIDs []string
	}{
		{name: "hello-desktop", serviceIDs: []string{"api", "ui"}},
		{name: "browser-automation-studio", serviceIDs: []string{"api", "playwright-driver", "ui"}},
		{name: "scenario-to-mcp", serviceIDs: []string{"api", "registry", "ui"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scenarioPath := filepath.Join(repoRoot, "scenarios", test.name)
			if _, statErr := os.Stat(filepath.Join(scenarioPath, ".vrooli", "service.json")); os.IsNotExist(statErr) {
				t.Skipf("fixture scenario %q is not present in this worktree", test.name)
			}
			cfg, err := config.LoadServiceConfig(scenarioPath)
			if err != nil {
				t.Fatal(err)
			}
			exported := BuildBundleManifest(test.name, scenarioPath, time.Unix(0, 0).UTC(), nil, cfg)
			skeleton := exported.Skeleton
			if skeleton == nil {
				t.Fatal("bundle export omitted desktop skeleton")
			}
			if skeleton.IPC.Port != 0 {
				t.Fatalf("IPC allocator input = %d, want 0", skeleton.IPC.Port)
			}
			ids := make([]string, 0, len(skeleton.Services))
			for _, service := range skeleton.Services {
				ids = append(ids, service.ID)
			}
			if !reflect.DeepEqual(ids, test.serviceIDs) {
				t.Fatalf("service IDs = %#v, want %#v", ids, test.serviceIDs)
			}
			if test.name == "browser-automation-studio" {
				if len(skeleton.Peers) != 4 || skeleton.Peers[0].BundlePolicy != "discover" {
					t.Fatalf("peer projection = %#v", skeleton.Peers)
				}
				if got := skeleton.Services[0].Env["PLAYWRIGHT_DRIVER_URL"]; got != "http://127.0.0.1:${playwright-driver.playwright_driver}" {
					t.Fatalf("sidecar URL projection = %q", got)
				}
				if skeleton.Services[1].Type != "worker" {
					t.Fatalf("sidecar service type = %q", skeleton.Services[1].Type)
				}
			}
			payload, err := json.MarshalIndent(skeleton, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("projected bundle skeleton:\n%s", payload)
		})
	}
}

func TestBuildSkeletonServicesProjectsDeclaredComponents(t *testing.T) {
	cfg := &types.Manifest{
		Ports: map[string]scenariomodel.Port{
			"api": {EnvVar: "API_PORT", Range: "23100-23200"},
		},
		Components: map[string]scenariomodel.Component{
			"api": {
				Role:  "api",
				Build: scenariomodel.ComponentBuild{Kind: "go_module", Dir: "api", Entry: "."},
				Run: scenariomodel.ComponentRun{
					Argv:     []string{"{{bin.api}}", "serve"},
					CWD:      "api",
					Env:      map[string]string{"MODE": "desktop"},
					Port:     "api",
					DataDirs: []string{"data/api"},
					LogDir:   "logs/api",
					Readiness: &scenariomodel.ComponentReadiness{
						Type:      "http",
						Path:      "/health",
						TimeoutMS: 12000,
					},
				},
			},
			"ui": {
				Role:  "ui",
				Build: scenariomodel.ComponentBuild{Kind: "pnpm_vite", Dir: "ui", Entry: "src/main.tsx"},
				Run: scenariomodel.ComponentRun{
					Argv: []string{"node", "server.js"},
					CWD:  "ui/dist",
					DependsOn: []scenariomodel.ComponentDependency{{
						Component: "api",
						Wait:      "ready",
					}},
				},
			},
		},
		Lifecycle: scenariomodel.Lifecycle{Health: &scenariomodel.HealthConfig{Checks: []scenariomodel.HealthCheck{{
			Type:     "http",
			Target:   "http://127.0.0.1:${API_PORT}/healthz",
			Interval: 3000,
			Timeout:  1500,
		}}}},
	}

	services := buildSkeletonServices(cfg)
	if len(services) != 2 || services[0].ID != "api" || services[1].ID != "ui" {
		t.Fatalf("declared components were not projected deterministically: %#v", services)
	}
	api := services[0]
	if api.Type != "api-binary" || api.Build == nil || api.Build.Type != "go" {
		t.Fatalf("api projection = %#v", api)
	}
	if api.Build.OutputPattern != "bin/api/{{platform}}/api{{ext}}" {
		t.Fatalf("output pattern = %q", api.Build.OutputPattern)
	}
	if api.Binaries["linux-x64"].Path != "bin/api/linux-x64/api" || api.Binaries["win-x64"].Path != "bin/api/win-x64/api.exe" {
		t.Fatalf("platform binary paths = %#v", api.Binaries)
	}
	if got := api.Binaries["linux-x64"].Cwd; got != "" {
		t.Fatalf("desktop API cwd = %q, want bundle root", got)
	}
	if !reflect.DeepEqual(api.Binaries["linux-x64"].Args, []string{"serve"}) {
		t.Fatalf("binary args = %#v", api.Binaries["linux-x64"].Args)
	}
	if api.Env["API_PORT"] != "${api.api}" || api.Env["MODE"] != "desktop" {
		t.Fatalf("projected env = %#v", api.Env)
	}
	if api.Health.Type != "http" || api.Health.Path != "/healthz" || api.Readiness.Type != "health_success" {
		t.Fatalf("health/readiness = %#v / %#v", api.Health, api.Readiness)
	}
	if !reflect.DeepEqual(services[1].Dependencies, []string{"api"}) {
		t.Fatalf("ui dependencies = %#v", services[1].Dependencies)
	}
	if got := services[1].Binaries["linux-x64"].Path; got != "ui/dist/index.html" {
		t.Fatalf("ui entry point = %q, want ui/dist/index.html", got)
	}
	if args := services[1].Binaries["linux-x64"].Args; len(args) != 0 {
		t.Fatalf("ui entry point args = %#v, want none", args)
	}
}

func TestDesktopPortRangeTranslatesFixedTierOnePortsToRoleBands(t *testing.T) {
	fixed := 21233
	got := desktopPortRange(scenariomodel.Port{EnvVar: "UI_PORT", Port: &fixed})
	if got.Min != 20000 || got.Max != 24999 {
		t.Fatalf("fixed UI port range = %#v, want 20000-24999", got)
	}
	if got.Min == fixed && got.Max == fixed {
		t.Fatalf("fixed UI port remained a single-port request: %#v", got)
	}
}

func TestDesktopPortRangePrefersDeclaredRangeOverFixedPort(t *testing.T) {
	fixed := 18800
	got := desktopPortRange(scenariomodel.Port{EnvVar: "API_PORT", Port: &fixed, Range: "18800-18899"})
	if got.Min != 18800 || got.Max != 18899 {
		t.Fatalf("declared range = %#v, want 18800-18899", got)
	}
}

func TestDesktopPortRangeKeepsRangeOnlyAndDefaults(t *testing.T) {
	ranged := desktopPortRange(scenariomodel.Port{Range: "23100-23200"})
	if ranged.Min != 23100 || ranged.Max != 23200 {
		t.Fatalf("range-only result = %#v, want 23100-23200", ranged)
	}
	defaulted := desktopPortRange(scenariomodel.Port{})
	if defaulted.Min != 20000 || defaulted.Max != 24000 {
		t.Fatalf("default result = %#v, want 20000-24000", defaulted)
	}
}

func TestDesktopSkeletonPreservesPersonalLocalIdentity(t *testing.T) {
	cfg := &types.Manifest{Authentication: &scenariomodel.AuthenticationProfile{Version: 1, Profile: "hybrid", DefaultMode: "personal_local", HumanSignIn: "optional", Provider: "cloudflare-access", Resource: "web-console", Offline: true}}
	skeleton := buildDesktopBundleSkeleton("web-console", t.TempDir(), cfg, nil)
	auth := skeleton.Authentication
	if auth == nil || auth.Mode != "personal_local" || !auth.Offline || auth.Provider != "" || auth.RequiresAuthenticator || auth.HumanSignIn != "optional" {
		t.Fatalf("personal-local identity contract lost or became networked: %+v", auth)
	}
}

func TestOwnerDesktopFeaturesExportFromSource(t *testing.T) {
	for owner, feature := range map[string]string{"audio-tools": "audio.processing", "agent-manager": "usage.live"} {
		t.Run(owner, func(t *testing.T) {
			root := filepath.Join("..", "..", "..", "..", owner)
			cfg, err := config.LoadServiceConfig(root)
			if err != nil {
				t.Fatal(err)
			}
			skeleton := buildDesktopBundleSkeleton(owner, root, cfg, nil)
			declaration, ok := skeleton.Features[feature]
			if !ok || len(skeleton.Features) != 1 || declaration.OwnerVersion != "0.0.1" || declaration.Service != "api" || declaration.Port != "api" {
				t.Fatalf("source declaration lost: %#v", skeleton.Features)
			}
			encoded, err := json.Marshal(skeleton)
			if err != nil {
				t.Fatal(err)
			}
			var runtimeContract runtimeManifest.Manifest
			if err := json.Unmarshal(encoded, &runtimeContract); err != nil {
				t.Fatal(err)
			}
			if err := runtimeManifest.ValidateFeatureExports(runtimeContract.Features, runtimeContract.Services); err != nil {
				t.Fatal(err)
			}
			var result map[string]any
			if err := json.Unmarshal(encoded, &result); err != nil {
				t.Fatal(err)
			}
			if _, ok := result["features"]; !ok {
				t.Fatal("skeleton dropped owner features")
			}
		})
	}
}

func TestEmbeddedOwnerFeaturesExportFromSource(t *testing.T) {
	for owner, feature := range map[string]string{"audio-tools": "audio.processing", "agent-manager": "usage.live"} {
		for _, policy := range []string{"embed", "either", "discover"} {
			for _, programBinding := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/program=%t", owner, policy, programBinding), func(t *testing.T) {
					ownerPath := filepath.Join("..", "..", "..", "..", owner)
					cfg := &types.Manifest{Service: scenariomodel.ServiceMetadata{Name: "consumer", Version: "9.9.9"}, Dependencies: scenariomodel.Dependencies{Scenarios: map[string]scenariomodel.Dependency{owner: {BundlePolicy: policy, VersionRange: ">=0.0.1 <1"}}}}
					node := types.DeploymentDependencyNode{Name: owner, Type: "scenario", Path: ownerPath}
					if programBinding {
						node.Source = "program-binding"
					}
					skeleton := buildDesktopBundleSkeleton("consumer", t.TempDir(), cfg, []types.DeploymentDependencyNode{node})
					if len(skeleton.Features) != 0 {
						t.Fatalf("embedded owner became root feature: %#v", skeleton.Features)
					}
					peers := skeleton.Peers
					if programBinding {
						peers = skeleton.ProgramBindingPeers
						for _, peer := range skeleton.Peers {
							if peer.Scenario == owner {
								t.Fatal("program peer duplicated")
							}
						}
					}
					ownerPeers := make([]types.BundleSkeletonPeer, 0, 1)
					for _, peer := range peers {
						if peer.Scenario == owner {
							ownerPeers = append(ownerPeers, peer)
						}
					}
					if len(ownerPeers) != 1 {
						t.Fatalf("owner peer missing or duplicated: %#v", peers)
					}
					peers = ownerPeers
					if policy == "discover" {
						if len(peers[0].Features) != 0 {
							t.Fatal("discovered peer declared embedded features")
						}
						return
					}
					declaration, ok := peers[0].Features[feature]
					if !ok || declaration.OwnerVersion != "0.0.1" || declaration.Service != owner+"--api" || declaration.Port != "api" {
						t.Fatalf("wrong embedded owner declaration: %#v", peers[0])
					}
					encoded, err := json.Marshal(skeleton)
					if err != nil {
						t.Fatal(err)
					}
					var native runtimeManifest.Manifest
					if err := json.Unmarshal(encoded, &native); err != nil {
						t.Fatal(err)
					}
					if err := runtimeManifest.ValidatePeerFeatureExports(append(native.Peers, native.ProgramBindingPeers...), native.Services); err != nil {
						t.Fatal(err)
					}
					if len(native.Features) != 0 {
						t.Fatal("native manifest leaked owner feature into root")
					}
				})
			}
		}
	}
}
