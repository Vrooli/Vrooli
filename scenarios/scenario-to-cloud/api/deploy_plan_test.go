package main

import (
	"strings"
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/vps"
)

func TestBuildDeployPlanIncludesCaddyAndScenarioStart(t *testing.T) {
	// [REQ:STC-P0-005] Start resources + scenario and verify HTTPS
	// [REQ:STC-P0-011] Caddy + Let's Encrypt configured and verified
	manifest := domain.CloudManifest{
		Version: "1.0.0",
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS: &domain.ManifestVPS{
				Host:    "203.0.113.10",
				Port:    22,
				User:    "root",
				Workdir: "/root/Vrooli",
			},
		},
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: domain.ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite", "vrooli-autoheal"},
			Resources: []string{"postgres"},
		},
		Bundle: domain.ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:  domain.ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
		Edge:   domain.ManifestEdge{Domain: "example.com", Caddy: domain.ManifestCaddy{Enabled: true}},
	}

	plan, err := vps.BuildDeployPlan(manifest)
	if err != nil {
		t.Fatalf("vps.BuildDeployPlan: %v", err)
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
		ports    domain.ManifestPorts
		expected string
	}{
		{
			name:     "empty ports",
			ports:    domain.ManifestPorts{},
			expected: "",
		},
		{
			name:     "single port",
			ports:    domain.ManifestPorts{"api": 8080},
			expected: "export API_PORT=8080 &&",
		},
		{
			name:     "multiple ports sorted",
			ports:    domain.ManifestPorts{"ui": 3000, "api": 8080, "ws": 9000},
			expected: "export API_PORT=8080 UI_PORT=3000 WS_PORT=9000 &&",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vps.BuildPortEnvVars(tt.ports)
			if result != tt.expected {
				t.Errorf("BuildPortEnvVars(%v) = %q, want %q", tt.ports, result, tt.expected)
			}
		})
	}
}

func TestBuildWaitForPortScriptUsesHomeEnv(t *testing.T) {
	script := vps.BuildWaitForPortScript("127.0.0.1", 35000, 10, "UI")
	if strings.Contains(script, "~/.vrooli") {
		t.Fatalf("expected wait script to avoid ~ expansion in logs path")
	}
	if !strings.Contains(script, "$HOME/.vrooli") {
		t.Fatalf("expected wait script to use $HOME for logs path")
	}
}
