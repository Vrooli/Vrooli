package main

import (
	"strings"
	"testing"
)

func TestBuildVPSDeployPlanIncludesCaddyAndScenarioStart(t *testing.T) {
	// [REQ:STC-P0-005] Start resources + scenario and verify HTTPS
	// [REQ:STC-P0-011] Caddy + Let's Encrypt configured and verified
	manifest := CloudManifest{
		Version: "1.0.0",
		Target: ManifestTarget{
			Type: "vps",
			VPS: &ManifestVPS{
				Host:    "203.0.113.10",
				Port:    22,
				User:    "root",
				Workdir: "/root/Vrooli",
			},
		},
		Scenario: ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite", "vrooli-autoheal"},
			Resources: []string{"postgres"},
		},
		Bundle: ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:  ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
		Edge:   ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true}},
	}

	plan, err := BuildVPSDeployPlan(manifest)
	if err != nil {
		t.Fatalf("BuildVPSDeployPlan: %v", err)
	}

	var hasCaddy, hasTargetStart, hasExportedPorts, hasHTTPS bool
	for _, step := range plan {
		if step.ID == "caddy_install" || step.ID == "caddy_config" {
			hasCaddy = true
		}
		if step.ID == "scenario_start_target" {
			// Check that ports are properly exported (not just set inline)
			if strings.Contains(step.Command, "export API_PORT=3001 UI_PORT=3000 WS_PORT=3002") {
				hasExportedPorts = true
			}
			if strings.Contains(step.Command, "vrooli scenario start") {
				hasTargetStart = true
			}
		}
		if step.ID == "verify_https" && strings.Contains(step.Command, "https://example.com/health") {
			hasHTTPS = true
		}
	}
	if !hasCaddy {
		t.Fatalf("expected caddy steps in plan")
	}
	if !hasTargetStart {
		t.Fatalf("expected scenario_start_target step with vrooli scenario start")
	}
	if !hasExportedPorts {
		t.Fatalf("expected port environment variables to be exported (not just inline)")
	}
	if !hasHTTPS {
		t.Fatalf("expected verify_https step")
	}
}

func TestBuildPortEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		ports    ManifestPorts
		expected string
	}{
		{
			name:     "empty ports",
			ports:    ManifestPorts{},
			expected: "",
		},
		{
			name:     "single port",
			ports:    ManifestPorts{"api": 8080},
			expected: "export API_PORT=8080 &&",
		},
		{
			name:     "multiple ports sorted",
			ports:    ManifestPorts{"ui": 3000, "api": 8080, "ws": 9000},
			expected: "export API_PORT=8080 UI_PORT=3000 WS_PORT=9000 &&",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPortEnvVars(tt.ports)
			if result != tt.expected {
				t.Errorf("buildPortEnvVars(%v) = %q, want %q", tt.ports, result, tt.expected)
			}
		})
	}
}
