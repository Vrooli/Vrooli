package main

import (
	"strings"
	"testing"
)

func TestManifestValidationErrorsAreActionable(t *testing.T) {
	// [REQ:STC-P0-009] preflight errors are actionable
	manifest := CloudManifest{
		Version: "1.0.0",
		Target: ManifestTarget{
			Type: "vps",
			VPS:  &ManifestVPS{},
		},
		Scenario: ManifestScenario{ID: "landing-page-business-suite"},
		Bundle:   ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:    ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
		Edge:     ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true}},
	}

	_, issues := ValidateAndNormalizeManifest(manifest)
	if len(issues) == 0 {
		t.Fatalf("expected issues")
	}

	for _, issue := range issues {
		if issue.Severity == SeverityError && issue.Hint == "" {
			t.Fatalf("expected error issues to include a hint, got: %+v", issue)
		}
	}
}

func TestValidationErrors_MissingRequiredFields(t *testing.T) {
	// [REQ:STC-P0-009] Preflight errors are actionable - test coverage for all required field validations
	tests := []struct {
		name         string
		manifest     CloudManifest
		expectPath   string
		expectMsgSub string
	}{
		{
			name:         "missing version",
			manifest:     CloudManifest{Target: ManifestTarget{Type: "vps"}},
			expectPath:   "version",
			expectMsgSub: "version is required",
		},
		{
			name:         "missing target type",
			manifest:     CloudManifest{Version: "1.0.0"},
			expectPath:   "target.type",
			expectMsgSub: "target type is required",
		},
		{
			name:         "unsupported target type",
			manifest:     CloudManifest{Version: "1.0.0", Target: ManifestTarget{Type: "kubernetes"}},
			expectPath:   "target.type",
			expectMsgSub: "unsupported target.type",
		},
		{
			name:         "missing vps host",
			manifest:     CloudManifest{Version: "1.0.0", Target: ManifestTarget{Type: "vps", VPS: &ManifestVPS{}}},
			expectPath:   "target.vps.host",
			expectMsgSub: "VPS host is required",
		},
		{
			name:         "missing scenario id",
			manifest:     CloudManifest{Version: "1.0.0", Target: ManifestTarget{Type: "vps", VPS: &ManifestVPS{Host: "10.0.0.1"}}},
			expectPath:   "scenario.id",
			expectMsgSub: "scenario id is required",
		},
		{
			name: "missing dependencies.scenarios",
			manifest: CloudManifest{
				Version:  "1.0.0",
				Target:   ManifestTarget{Type: "vps", VPS: &ManifestVPS{Host: "10.0.0.1"}},
				Scenario: ManifestScenario{ID: "test-app"},
			},
			expectPath:   "dependencies.scenarios",
			expectMsgSub: "dependencies.scenarios is required",
		},
		{
			name: "invalid port value",
			manifest: CloudManifest{
				Version:  "1.0.0",
				Target:   ManifestTarget{Type: "vps", VPS: &ManifestVPS{Host: "10.0.0.1"}},
				Scenario: ManifestScenario{ID: "test-app"},
				Dependencies: ManifestDependencies{
					Scenarios: []string{"test-app"},
				},
				Ports: ManifestPorts{"ui": 99999},
			},
			expectPath:   "ports.ui",
			expectMsgSub: "port must be between 1 and 65535",
		},
		{
			name: "duplicate ports",
			manifest: CloudManifest{
				Version:  "1.0.0",
				Target:   ManifestTarget{Type: "vps", VPS: &ManifestVPS{Host: "10.0.0.1"}},
				Scenario: ManifestScenario{ID: "test-app"},
				Dependencies: ManifestDependencies{
					Scenarios: []string{"test-app"},
				},
				Ports: ManifestPorts{"ui": 3000, "api": 3000},
			},
			expectPath:   "ports",
			expectMsgSub: "ports must be distinct",
		},
		{
			name: "invalid domain format",
			manifest: CloudManifest{
				Version:  "1.0.0",
				Target:   ManifestTarget{Type: "vps", VPS: &ManifestVPS{Host: "10.0.0.1"}},
				Scenario: ManifestScenario{ID: "test-app"},
				Dependencies: ManifestDependencies{
					Scenarios: []string{"test-app"},
				},
				Edge: ManifestEdge{Domain: "https://example.com"},
			},
			expectPath:   "edge.domain",
			expectMsgSub: "must be a hostname (no scheme, no path)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, issues := ValidateAndNormalizeManifest(tt.manifest)

			found := false
			for _, issue := range issues {
				if issue.Path == tt.expectPath && strings.Contains(issue.Message, tt.expectMsgSub) {
					found = true
					if issue.Severity == SeverityError && issue.Hint == "" {
						t.Errorf("error issue for %s should have a hint", tt.expectPath)
					}
					break
				}
			}
			if !found {
				t.Errorf("expected issue with path=%q containing %q, got issues: %+v", tt.expectPath, tt.expectMsgSub, issues)
			}
		})
	}
}

func TestValidationErrors_EdgeCasesHaveActionableHints(t *testing.T) {
	// [REQ:STC-P0-009] Preflight errors are actionable - hints provide guidance
	manifest := CloudManifest{
		Version: "1.0.0",
		Target: ManifestTarget{
			Type: "vps",
			VPS:  &ManifestVPS{Host: "10.0.0.1"},
		},
		Scenario: ManifestScenario{ID: "test-app"},
		Dependencies: ManifestDependencies{
			Scenarios: []string{"other-app"}, // Missing test-app itself
		},
		Bundle: ManifestBundle{
			IncludePackages: true,
			IncludeAutoheal: true,
			Scenarios:       []string{"other-app"}, // Missing test-app
		},
		Ports: ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
		Edge:  ManifestEdge{Domain: "example.com"},
	}

	_, issues := ValidateAndNormalizeManifest(manifest)

	// Should have errors for scenario not included in dependencies and bundle
	errorCount := 0
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			errorCount++
			if issue.Hint == "" {
				t.Errorf("error issue %q should have actionable hint, got empty", issue.Path)
			}
			// Verify hint is non-trivial (more than just repeating the message)
			if len(issue.Hint) < 10 {
				t.Errorf("hint for %q seems too short to be actionable: %q", issue.Path, issue.Hint)
			}
		}
	}

	if errorCount == 0 {
		t.Error("expected at least one error issue for invalid manifest")
	}
}
