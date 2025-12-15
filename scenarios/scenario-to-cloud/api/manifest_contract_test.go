package main

import "testing"

func TestValidateAndNormalizeManifest_MinimalValidVPS(t *testing.T) {
	// [REQ:STC-P0-001] cloud manifest export contract (consumer-side validation)
	manifest := CloudManifest{
		Version: "1.0.0",
		Target: ManifestTarget{
			Type: "vps",
			VPS: &ManifestVPS{
				Host: "203.0.113.10",
			},
		},
		Scenario: ManifestScenario{
			ID: "landing-page-business-suite",
		},
		Dependencies: ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite"},
			Resources: []string{},
		},
		Bundle: ManifestBundle{
			IncludePackages: true,
			IncludeAutoheal: true,
			Scenarios:       []string{"landing-page-business-suite", "vrooli-autoheal"},
		},
		Ports: ManifestPorts{
			UI:  3000,
			API: 3001,
			WS:  3002,
		},
		Edge: ManifestEdge{
			Domain: "example.com",
			Caddy:  ManifestCaddy{Enabled: true, Email: "ops@example.com"},
		},
	}
	manifest.Dependencies.Analyzer.Tool = "scenario-dependency-analyzer"

	normalized, issues := ValidateAndNormalizeManifest(manifest)
	if len(issues) > 0 {
		t.Fatalf("expected no issues, got: %+v", issues)
	}
	if normalized.Target.VPS == nil || normalized.Target.VPS.User != "root" {
		t.Fatalf("expected default vps user root, got: %+v", normalized.Target.VPS)
	}
	if normalized.Target.VPS.Port != 22 {
		t.Fatalf("expected default vps port 22, got: %d", normalized.Target.VPS.Port)
	}
	if normalized.Target.VPS.Workdir != "/root/Vrooli" {
		t.Fatalf("expected default workdir /root/Vrooli, got: %q", normalized.Target.VPS.Workdir)
	}
}
