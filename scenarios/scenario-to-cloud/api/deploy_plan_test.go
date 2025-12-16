package main

import (
	"strings"
	"testing"
)

func TestBuildVPSDeployPlanIncludesCaddyAndScenarioStart(t *testing.T) {
	// [REQ:STC-P0-005] Start resources + scenario and verify HTTPS
	// [REQ:STC-P0-011] Caddy + Let’s Encrypt configured and verified
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
		Ports:  ManifestPorts{UI: 3000, API: 3001, WS: 3002},
		Edge:   ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true}},
	}

	plan, err := BuildVPSDeployPlan(manifest)
	if err != nil {
		t.Fatalf("BuildVPSDeployPlan: %v", err)
	}

	var hasCaddy, hasTargetStart, hasHTTPS bool
	for _, step := range plan {
		if step.ID == "caddy_install" || step.ID == "caddy_config" {
			hasCaddy = true
		}
		if step.ID == "scenario_start_target" && strings.Contains(step.Command, "UI_PORT=3000") {
			hasTargetStart = true
		}
		if step.ID == "verify_https" && strings.Contains(step.Command, "https://example.com/health") {
			hasHTTPS = true
		}
	}
	if !hasCaddy || !hasTargetStart || !hasHTTPS {
		t.Fatalf("expected caddy/target/https steps, got: %+v", plan)
	}
}

