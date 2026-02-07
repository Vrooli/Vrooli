package vps

import (
	"strings"
	"testing"

	"scenario-to-cloud/domain"
)

func TestBuildScenarioCleanupCommand_UsesPreservePaths(t *testing.T) {
	manifest := domain.CloudManifest{
		Target: domain.ManifestTarget{
			VPS: &domain.ManifestVPS{
				Workdir: "/root/Vrooli",
				PreservePaths: []string{
					"scenarios/landing-page-business-suite/api/uploads",
				},
			},
		},
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Bundle: domain.ManifestBundle{
			Scenarios: []string{"landing-page-business-suite"},
		},
	}

	cmd, err := buildScenarioCleanupCommand(manifest)
	if err != nil {
		t.Fatalf("buildScenarioCleanupCommand: %v", err)
	}
	if !strings.Contains(cmd, "cleanup blocked: mutable path") {
		t.Fatalf("expected mutable-path safety gate in cleanup command: %s", cmd)
	}
	if !strings.Contains(cmd, "api/uploads") {
		t.Fatalf("expected preserve path in cleanup command: %s", cmd)
	}
	if !strings.Contains(cmd, "tar -cf") || !strings.Contains(cmd, "tar -xf") {
		t.Fatalf("expected tar backup/restore workflow in cleanup command: %s", cmd)
	}
}

func TestBuildScenarioCleanupCommand_RejectsInvalidPreservePath(t *testing.T) {
	manifest := domain.CloudManifest{
		Target: domain.ManifestTarget{
			VPS: &domain.ManifestVPS{
				Workdir:       "/root/Vrooli",
				PreservePaths: []string{"../etc/passwd"},
			},
		},
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Bundle: domain.ManifestBundle{
			Scenarios: []string{"landing-page-business-suite"},
		},
	}

	if _, err := buildScenarioCleanupCommand(manifest); err == nil {
		t.Fatal("expected error for invalid preserve path")
	}
}
