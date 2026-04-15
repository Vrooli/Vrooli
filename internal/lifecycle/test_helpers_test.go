package lifecycle

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

func writeLifecycleFixture(t *testing.T, root, name string) {
	t.Helper()
	writeLifecycleFixtureManifest(t, root, lifecycleFixtureManifest(name))
}

func writeLifecycleFixtureManifest(t *testing.T, root string, manifest scenario.ServiceManifest) {
	t.Helper()

	if strings.TrimSpace(manifest.Service.Name) == "" {
		t.Fatalf("fixture manifest is missing service name")
	}

	testresource.WritePortRegistry(t, root, nil)
	testscenario.WriteScenarioService(t, root, manifest.Service.Name, manifest)
}

func lifecycleFixtureManifest(name string) scenario.ServiceManifest {
	return scenario.ServiceManifest{
		Version: "1.0.0",
		Service: scenario.ServiceMetadata{
			Name:        name,
			DisplayName: "Lifecycle " + name,
			Description: "Lifecycle validation fixture",
			Version:     "0.1.0",
		},
		Ports: map[string]scenario.Port{
			"api": {
				EnvVar: "API_PORT",
				Range:  "22000-22010",
			},
		},
		Lifecycle: scenario.Lifecycle{
			Version: "2.0.0",
			Health: &scenario.HealthConfig{
				Checks: []scenario.HealthCheck{
					{
						Name:     "api",
						Type:     "http",
						Target:   "http://127.0.0.1:${API_PORT}/health",
						Critical: true,
						Timeout:  1000,
					},
				},
				StartupGracePeriod: 250,
				Timeout:            5000,
				Interval:           250,
			},
			Setup: scenario.Phase{
				Condition: &scenario.Condition{
					Checks: []scenario.ConditionCheck{
						{Type: "binaries", Targets: []string{"api/mock-api"}},
					},
				},
				Steps: []scenario.PhaseStep{
					{
						Name: "build-api",
						Run:  "mkdir -p api public && printf 'package main\\n' > api/handler.go && printf '#!/usr/bin/env bash\\npython3 -m http.server \"$API_PORT\" --bind 127.0.0.1 --directory ../public\\n' > api/mock-api && chmod +x api/mock-api && printf 'ok\\n' > public/health",
					},
				},
			},
			Develop: scenario.Phase{
				Steps: []scenario.PhaseStep{
					{
						Name:       "start-api",
						Run:        "cd api && ./mock-api",
						Background: true,
						Condition:  &scenario.Condition{FileExists: "api/mock-api"},
					},
				},
			},
		},
	}
}
