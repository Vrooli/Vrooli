package scenario

import (
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=1 | LAST: 2026-04-11

func TestDescribeRuntimeUsesSharedPortInferenceAndHealth(t *testing.T) {
	startedAt := time.Date(2026, time.April, 11, 12, 0, 0, 0, time.UTC)
	manifest := ServiceManifest{
		Ports: map[string]Port{
			"api": {EnvVar: "API_PORT"},
			"ui":  {EnvVar: "UI_PORT"},
		},
		Lifecycle: Lifecycle{
			Health: &HealthConfig{
				Checks: []HealthCheck{
					{Name: "api", Type: "http", Target: "http://127.0.0.1:${API_PORT}/health", Critical: true},
				},
			},
		},
	}
	runtime := process.ScenarioRuntime{
		Name:         "alpha",
		ProcessCount: 2,
		Runtime:      "5m",
		StartedAt:    &startedAt,
		Records: []process.Record{
			{PID: 123, Step: "start-api", Port: 18080, StartedAt: startedAt},
			{PID: 124, Step: "launch-ui", Port: 38080, StartedAt: startedAt},
		},
	}

	details := DescribeRuntime(manifest, runtime)
	if details.Status != "running" {
		t.Fatalf("details.Status = %q, want running", details.Status)
	}
	if details.Processes != 2 {
		t.Fatalf("details.Processes = %d, want 2", details.Processes)
	}
	if details.Runtime != "5m" {
		t.Fatalf("details.Runtime = %q", details.Runtime)
	}
	if details.StartedAt == nil || !details.StartedAt.Equal(startedAt) {
		t.Fatalf("details.StartedAt = %#v", details.StartedAt)
	}
	if details.Ports["API_PORT"] != 18080 || details.Ports["UI_PORT"] != 38080 {
		t.Fatalf("details.Ports = %#v", details.Ports)
	}
	if got := len(details.PortBindings); got != 2 {
		t.Fatalf("len(details.PortBindings) = %d, want 2", got)
	}
	if details.PortBindings[0].Key != "API_PORT" || details.PortBindings[0].Step != "start-api" {
		t.Fatalf("details.PortBindings[0] = %#v", details.PortBindings[0])
	}
	if details.Health != "unhealthy" {
		t.Fatalf("details.Health = %q, want unhealthy", details.Health)
	}
}

func TestInferPortEnvVarNormalizesHistoricalStepNames(t *testing.T) {
	manifest := ServiceManifest{
		Ports: map[string]Port{
			"api":       {EnvVar: "API_PORT"},
			"ui":        {EnvVar: "UI_PORT"},
			"websocket": {EnvVar: "WS_PORT"},
		},
	}

	cases := map[string]string{
		"start-api":    "API_PORT",
		"run-ui":       "UI_PORT",
		"launch-vite":  "UI_PORT",
		"serve-ws":     "WS_PORT",
		"socket-proxy": "WS_PORT",
	}

	for step, want := range cases {
		if got := InferPortEnvVar(manifest, step); got != want {
			t.Fatalf("InferPortEnvVar(%q) = %q, want %q", step, got, want)
		}
	}
}

func TestRuntimeEndpointsPreferManifestOrderAndDescriptions(t *testing.T) {
	manifest := ServiceManifest{
		Ports: map[string]Port{
			"ui":  {EnvVar: "UI_PORT", Description: "Frontend"},
			"api": {EnvVar: "API_PORT", Description: "Backend"},
		},
	}

	endpoints := RuntimeEndpoints(manifest, map[string]int{
		"UI_PORT":    38080,
		"API_PORT":   18080,
		"ADMIN_PORT": 19090,
	})
	if len(endpoints) != 3 {
		t.Fatalf("len(endpoints) = %d, want 3", len(endpoints))
	}
	if endpoints[0].Key != "API_PORT" || endpoints[0].URL != "http://localhost:18080" || endpoints[0].Description != "Backend" {
		t.Fatalf("endpoints[0] = %#v", endpoints[0])
	}
	if endpoints[1].Key != "UI_PORT" || endpoints[1].URL != "http://localhost:38080" || endpoints[1].Description != "Frontend" {
		t.Fatalf("endpoints[1] = %#v", endpoints[1])
	}
	if endpoints[2].Key != "ADMIN_PORT" || endpoints[2].Name != "ADMIN_PORT" || endpoints[2].URL != "http://localhost:19090" {
		t.Fatalf("endpoints[2] = %#v", endpoints[2])
	}
}
