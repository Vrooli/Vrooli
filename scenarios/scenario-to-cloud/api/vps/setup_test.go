package vps

import (
	"scenario-to-cloud/domain"
	"strings"
	"testing"
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
	if !strings.Contains(cmd, "xargs -0 -r tar -cf") || !strings.Contains(cmd, "tar -xf") {
		t.Fatalf("expected tar backup/restore workflow in cleanup command: %s", cmd)
	}
}

func TestBuildScenarioCleanupCommand_LegacyFallbackPreservesDetectedMutablePaths(t *testing.T) {
	manifest := domain.CloudManifest{
		Target: domain.ManifestTarget{
			VPS: &domain.ManifestVPS{
				Workdir: "/root/Vrooli",
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
	if !strings.Contains(cmd, `PRESERVE_REL_LIST="$MUTABLE_REL_LIST"`) {
		t.Fatalf("expected legacy fallback to preserve detected mutable paths: %s", cmd)
	}
	if strings.Contains(cmd, "cleanup blocked: mutable path") {
		t.Fatalf("did not expect strict mutable-path blocker when preserve_paths is empty: %s", cmd)
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
