package deployment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/config"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

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
			Bindings: []scenariomodel.Binding{{
				EnvVar: "HELPER_URL", Form: "http_base_url", Port: "api", WhenUnavailable: "fail",
			}},
		},
	}}}
	root := []types.BundleSkeletonService{{ID: "consumer", Env: map[string]string{}}}
	services := embedPeerServices(root, cfg, []types.DeploymentDependencyNode{{Name: "helper", Type: "scenario", Path: peerRoot}})
	if len(services) != 2 || services[1].ID != "helper--api" {
		t.Fatalf("embedded services = %#v", services)
	}
	if got := services[0].Env["HELPER_URL"]; got != "http://127.0.0.1:${helper--api.api}" {
		t.Fatalf("embedded binding = %q", got)
	}
	if !reflect.DeepEqual(services[0].Dependencies, []string{"helper--api"}) {
		t.Fatalf("embedded dependency = %#v", services[0].Dependencies)
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
				if len(skeleton.Peers) != 1 || skeleton.Peers[0].BundlePolicy != "discover" {
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
}
