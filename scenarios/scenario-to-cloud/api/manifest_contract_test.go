package main

import (
	"testing"
)

// testManifestBase creates a valid base manifest for testing.
// The manifest is valid but uses a synthetic scenario ID that won't exist on disk,
// so warnings about missing service.json are expected and acceptable.
func testManifestBase(scenarioID string, scenarios []string) CloudManifest {
	m := CloudManifest{
		Version:  "1.0.0",
		Target:   ManifestTarget{Type: "vps", VPS: &ManifestVPS{Host: "10.0.0.1"}},
		Scenario: ManifestScenario{ID: scenarioID},
		Dependencies: ManifestDependencies{
			Scenarios: scenarios,
		},
		Bundle: ManifestBundle{
			IncludePackages: true,
			IncludeAutoheal: true,
			Scenarios:       append(scenarios, "vrooli-autoheal"),
		},
		Ports: ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
		Edge:  ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true}},
	}
	m.Dependencies.Analyzer.Tool = "scenario-dependency-analyzer"
	return m
}

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
			"ui":  3000,
			"api": 3001,
			"ws":  3002,
		},
		Edge: ManifestEdge{
			Domain: "example.com",
			Caddy:  ManifestCaddy{Enabled: true, Email: "ops@example.com"},
		},
	}
	manifest.Dependencies.Analyzer.Tool = "scenario-dependency-analyzer"

	normalized, issues := ValidateAndNormalizeManifest(manifest)
	// Warnings are acceptable (e.g., when service.json can't be read from filesystem during tests).
	// Only fail on blocking errors that would prevent deployment.
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			t.Fatalf("expected no errors, got: %+v", issues)
		}
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

func TestValidateAndNormalizeManifest_DefaultsAreApplied(t *testing.T) {
	// [REQ:STC-P0-001] cloud manifest export contract - verify all normalization defaults
	t.Run("default VPS port is 22", func(t *testing.T) {
		manifest := testManifestBase("test", []string{"test"})
		normalized, issues := ValidateAndNormalizeManifest(manifest)
		for _, issue := range issues {
			if issue.Severity == SeverityError {
				t.Fatalf("unexpected error: %+v", issue)
			}
		}
		if normalized.Target.VPS.Port != 22 {
			t.Errorf("expected port 22, got %d", normalized.Target.VPS.Port)
		}
	})

	t.Run("default VPS user is root", func(t *testing.T) {
		manifest := testManifestBase("test", []string{"test"})
		normalized, issues := ValidateAndNormalizeManifest(manifest)
		for _, issue := range issues {
			if issue.Severity == SeverityError {
				t.Fatalf("unexpected error: %+v", issue)
			}
		}
		if normalized.Target.VPS.User != "root" {
			t.Errorf("expected user root, got %q", normalized.Target.VPS.User)
		}
	})

	t.Run("default workdir is /root/Vrooli", func(t *testing.T) {
		manifest := testManifestBase("test", []string{"test"})
		normalized, issues := ValidateAndNormalizeManifest(manifest)
		for _, issue := range issues {
			if issue.Severity == SeverityError {
				t.Fatalf("unexpected error: %+v", issue)
			}
		}
		if normalized.Target.VPS.Workdir != "/root/Vrooli" {
			t.Errorf("expected workdir /root/Vrooli, got %q", normalized.Target.VPS.Workdir)
		}
	})

	t.Run("default ports are applied when missing", func(t *testing.T) {
		manifest := testManifestBase("test", []string{"test"})
		manifest.Ports = nil // Deliberately nil
		normalized, issues := ValidateAndNormalizeManifest(manifest)
		for _, issue := range issues {
			if issue.Severity == SeverityError {
				t.Fatalf("unexpected error: %+v", issue)
			}
		}
		if normalized.Ports["ui"] != 3000 {
			t.Errorf("expected ui port 3000, got %d", normalized.Ports["ui"])
		}
		if normalized.Ports["api"] != 3001 {
			t.Errorf("expected api port 3001, got %d", normalized.Ports["api"])
		}
		if normalized.Ports["ws"] != 3002 {
			t.Errorf("expected ws port 3002, got %d", normalized.Ports["ws"])
		}
	})

	t.Run("bundle.scenarios populated from dependencies if empty", func(t *testing.T) {
		manifest := testManifestBase("test", []string{"test", "dep-a"})
		manifest.Bundle.Scenarios = []string{} // Empty, should be populated from dependencies
		normalized, issues := ValidateAndNormalizeManifest(manifest)
		for _, issue := range issues {
			if issue.Severity == SeverityError {
				t.Fatalf("unexpected error: %+v", issue)
			}
		}
		// Should include both from dependencies plus vrooli-autoheal
		if len(normalized.Bundle.Scenarios) < 2 {
			t.Errorf("expected at least 2 scenarios in bundle, got %v", normalized.Bundle.Scenarios)
		}
	})

	t.Run("default DNS policy is required", func(t *testing.T) {
		manifest := testManifestBase("test", []string{"test"})
		manifest.Edge.DNSPolicy = ""
		normalized, issues := ValidateAndNormalizeManifest(manifest)
		for _, issue := range issues {
			if issue.Severity == SeverityError {
				t.Fatalf("unexpected error: %+v", issue)
			}
		}
		if normalized.Edge.DNSPolicy != DNSPolicyRequired {
			t.Errorf("expected dns_policy required, got %q", normalized.Edge.DNSPolicy)
		}
	})
}

func TestValidateAndNormalizeManifest_StableOutput(t *testing.T) {
	// [REQ:STC-P0-001] manifest normalization is deterministic
	manifest := testManifestBase("test", []string{"z-scenario", "a-scenario", "test"})
	manifest.Dependencies.Resources = []string{"redis", "postgres"}
	manifest.Bundle.Scenarios = []string{"z-scenario", "a-scenario", "test"}
	manifest.Bundle.Resources = []string{"redis", "postgres"}

	norm1, _ := ValidateAndNormalizeManifest(manifest)
	norm2, _ := ValidateAndNormalizeManifest(manifest)

	// Dependencies should be sorted
	if norm1.Dependencies.Scenarios[0] != norm2.Dependencies.Scenarios[0] {
		t.Error("scenarios should be stable sorted")
	}
	if norm1.Dependencies.Resources[0] != norm2.Dependencies.Resources[0] {
		t.Error("resources should be stable sorted")
	}

	// Bundle scenarios should be sorted
	if norm1.Bundle.Scenarios[0] != norm2.Bundle.Scenarios[0] {
		t.Error("bundle scenarios should be stable sorted")
	}
}
