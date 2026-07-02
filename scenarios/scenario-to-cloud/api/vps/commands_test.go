package vps

import (
	"strings"
	"testing"

	"scenario-to-cloud/domain"
)

func TestBuildStopAllCommand(t *testing.T) {
	t.Parallel()

	manifest := domain.CloudManifest{
		Scenario: domain.ManifestScenario{ID: "my-app"},
		Dependencies: domain.ManifestDependencies{
			Scenarios: []string{"my-app", "helper-app"},
			Resources: []string{"postgres", "redis"},
		},
	}

	cmd := BuildStopAllCommand("/opt/vrooli", manifest)

	// Should contain stop commands for the scenario, dependencies, and resources
	if !strings.Contains(cmd, "scenario stop") {
		t.Error("expected scenario stop command")
	}
	if !strings.Contains(cmd, "'my-app'") {
		t.Error("expected my-app in command")
	}
	if !strings.Contains(cmd, "'helper-app'") {
		t.Error("expected helper-app in command")
	}
	if !strings.Contains(cmd, "resource stop") {
		t.Error("expected resource stop command")
	}
	if !strings.Contains(cmd, "'postgres'") {
		t.Error("expected postgres in command")
	}
	if !strings.Contains(cmd, "'redis'") {
		t.Error("expected redis in command")
	}
}

func TestBuildStopAllCommand_NoDeps(t *testing.T) {
	t.Parallel()

	manifest := domain.CloudManifest{
		Scenario: domain.ManifestScenario{ID: "solo-app"},
	}

	cmd := BuildStopAllCommand("/opt/vrooli", manifest)

	// Should still have the target scenario stop
	if !strings.Contains(cmd, "'solo-app'") {
		t.Error("expected solo-app in command")
	}
}

func TestBuildDockerPruneCommand(t *testing.T) {
	t.Parallel()

	cmd := BuildDockerPruneCommand()

	if !strings.Contains(cmd, "docker stop") {
		t.Error("expected docker stop")
	}
	if !strings.Contains(cmd, "docker rm") {
		t.Error("expected docker rm")
	}
	if !strings.Contains(cmd, "docker volume rm") {
		t.Error("expected docker volume rm")
	}
	if !strings.Contains(cmd, "docker system prune") {
		t.Error("expected docker system prune")
	}
}

func TestBuildCleanupCommand(t *testing.T) {
	t.Parallel()

	manifest := domain.CloudManifest{
		Scenario: domain.ManifestScenario{ID: "my-app"},
	}

	tests := []struct {
		level    int
		contains []string
	}{
		{1, []string{"rm -rf", "builds"}},
		{2, []string{"scenario stop", "rm -rf", "builds"}},
		{3, []string{"scenario stop", "docker"}},
		{4, []string{"scenario stop", "rm -rf"}},
		{5, []string{"scenario stop", "rm -rf", "docker", "apt-get autoremove"}},
	}

	for _, tt := range tests {
		cmd, desc := BuildCleanupCommand("/opt/vrooli", manifest, tt.level)
		if desc == "" {
			t.Errorf("level %d: empty description", tt.level)
		}
		for _, substr := range tt.contains {
			if !strings.Contains(cmd, substr) {
				t.Errorf("level %d: command should contain %q, got: %s", tt.level, substr, cmd)
			}
		}
	}
}

func TestBuildCleanupCommand_InvalidLevel(t *testing.T) {
	t.Parallel()

	manifest := domain.CloudManifest{}
	cmd, _ := BuildCleanupCommand("/opt/vrooli", manifest, 99)

	if !strings.Contains(cmd, "Invalid") {
		t.Errorf("invalid level should produce 'Invalid' message, got: %s", cmd)
	}
}
