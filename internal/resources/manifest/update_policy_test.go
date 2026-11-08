package manifest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	manifest "github.com/vrooli/vrooli/internal/resources/manifest"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func updatePolicyManifest(t *testing.T) manifest.ResourceManifest {
	t.Helper()
	m, err := manifest.Load(filepath.Join(findRepoRoot(t), "resources", "cloudflared", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	m.ManagedService.UpdatePolicy = &resourcedeployment.ServiceUpdatePolicy{Strategy: "rolling-stateless", CandidatePort: "http"}
	return m
}

func TestRollingUpdateManifestRequiresAnIsolatedOwnedReadinessRoute(t *testing.T) {
	if err := manifest.Validate(updatePolicyManifest(t)); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*manifest.ResourceManifest){
		"undeclared port": func(m *manifest.ResourceManifest) { m.ManagedService.UpdatePolicy.CandidatePort = "other" },
		"additional port": func(m *manifest.ResourceManifest) {
			m.Ports = append(m.Ports, manifest.ResourcePort{Name: "admin", Host: 9999})
		},
		"companion": func(m *manifest.ResourceManifest) {
			m.Companions = []manifest.ResourceCompanion{{Name: "helper", Command: "helper"}}
		},
		"runtime environment":    func(m *manifest.ResourceManifest) { m.RuntimeEnvCommand = &manifest.ResourceRuntimeEnvCommand{} },
		"exclusive group":        func(m *manifest.ResourceManifest) { m.Lifecycle.ExclusiveGroup = "database" },
		"remote readiness":       func(m *manifest.ResourceManifest) { m.HealthChecks[0].Target = "https://example.com/ready" },
		"fixed readiness port":   func(m *manifest.ResourceManifest) { m.HealthChecks[0].Target = "http://127.0.0.1:20242/ready" },
		"wrong readiness status": func(m *manifest.ResourceManifest) { m.HealthChecks[0].ExpectedStatus = []int{500} },
		"no remapped launch":     func(m *manifest.ResourceManifest) { m.ManagedService.Arguments = []string{"tunnel", "run"} },
		"platform bypass": func(m *manifest.ResourceManifest) {
			m.ManagedService.ArgumentsByPlatform = map[string][]string{"windows": {"tunnel", "run"}}
		},
		"port environment override": func(m *manifest.ResourceManifest) {
			m.ManagedService.Environment = map[string]string{"RESOURCE_PORT_HTTP": "20242"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := updatePolicyManifest(t)
			change(&m)
			if err := manifest.Validate(m); err == nil {
				t.Fatal("unsafe candidate readiness contract accepted")
			}
		})
	}
}

func TestRollingUpdateManifestUsesCanonicalPortNames(t *testing.T) {
	m := updatePolicyManifest(t)
	m.ManagedService.UpdatePolicy.CandidatePort = "http-admin"
	m.Ports[0].Name = "http-admin"
	m.ManagedService.Arguments = []string{"--metrics=127.0.0.1:${RESOURCE_PORT_HTTP_ADMIN}"}
	for i := range m.HealthChecks {
		m.HealthChecks[i].Target = "http://127.0.0.1:${RESOURCE_PORT_HTTP_ADMIN}/ready"
	}
	if err := manifest.Validate(m); err != nil {
		t.Fatalf("valid remapping/readiness rejected: %v", err)
	}
}

func TestResourceSchemaClosesTheUpdatePolicyContract(t *testing.T) {
	schema := compileResourceSchema(t)
	data, err := os.ReadFile(filepath.Join(findRepoRoot(t), "resources", "cloudflared", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		policy map[string]any
		valid  bool
	}{
		{"eligible", map[string]any{"strategy": "rolling-stateless", "candidate_port": "http"}, true},
		{"missing strategy", map[string]any{"candidate_port": "http"}, false},
		{"unknown strategy", map[string]any{"strategy": "replace", "candidate_port": "http"}, false},
		{"invalid port", map[string]any{"strategy": "rolling-stateless", "candidate_port": "../http"}, false},
		{"unknown permission", map[string]any{"strategy": "rolling-stateless", "candidate_port": "http", "allow_force": true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var m map[string]any
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatal(err)
			}
			m["managed_service"].(map[string]any)["update_policy"] = tc.policy
			err := schema.Validate(m)
			if (err == nil) != tc.valid {
				t.Fatalf("schema valid=%v want=%v err=%v", err == nil, tc.valid, err)
			}
		})
	}
}
