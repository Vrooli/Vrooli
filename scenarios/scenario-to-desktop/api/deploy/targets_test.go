package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTargetRepositoryCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("create .vrooli dir: %v", err)
	}

	repo := NewTargetRepository(tmpDir)

	// List empty
	targets, err := repo.List()
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}

	// Save
	err = repo.Save("prod", &DeployTarget{
		Label:         "Production",
		ScenarioName:  "landing-page-business-suite",
		RemoteProfile: "prod-server",
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Get
	target, err := repo.Get("prod")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if target.Label != "Production" {
		t.Errorf("expected label 'Production', got %q", target.Label)
	}
	if target.ScenarioName != "landing-page-business-suite" {
		t.Errorf("unexpected scenario_name: %q", target.ScenarioName)
	}
	if target.RemoteProfile != "prod-server" {
		t.Errorf("unexpected remote_profile: %q", target.RemoteProfile)
	}

	// Save another
	err = repo.Save("staging", &DeployTarget{
		Label:         "Staging",
		ScenarioName:  "landing-page-business-suite",
		RemoteProfile: "staging-server",
	})
	if err != nil {
		t.Fatalf("Save staging: %v", err)
	}

	// List both
	targets, err = repo.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(targets))
	}

	// Update existing
	err = repo.Save("prod", &DeployTarget{
		Label:         "Production (updated)",
		ScenarioName:  "landing-page-business-suite",
		RemoteProfile: "prod-server-v2",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	target, err = repo.Get("prod")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if target.RemoteProfile != "prod-server-v2" {
		t.Errorf("expected updated profile, got %q", target.RemoteProfile)
	}

	// Delete
	err = repo.Delete("staging")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	targets, err = repo.List()
	if err != nil {
		t.Fatalf("List after delete: %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("expected 1 target after delete, got %d", len(targets))
	}

	// Get not found
	_, err = repo.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing target")
	}

	// Delete not found
	err = repo.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error for deleting missing target")
	}
}

func TestTargetRepositoryFileNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewTargetRepository(tmpDir)

	// List when file doesn't exist yet (should return empty)
	targets, err := repo.List()
	if err != nil {
		t.Fatalf("List no file: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestTargetRepositoryCreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	// .vrooli dir doesn't exist yet
	repo := NewTargetRepository(tmpDir)

	err := repo.Save("test", &DeployTarget{
		Label:         "Test",
		ScenarioName:  "some-scenario",
		RemoteProfile: "some-profile",
	})
	if err != nil {
		t.Fatalf("Save should create .vrooli dir: %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(filepath.Join(tmpDir, ".vrooli", "deploy-targets.json"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty file")
	}
}

func TestTargetRepositoryPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Save with one repo instance
	repo1 := NewTargetRepository(tmpDir)
	err := repo1.Save("prod", &DeployTarget{
		Label:         "Production",
		ScenarioName:  "lpbs",
		RemoteProfile: "prod",
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Read with a fresh instance
	repo2 := NewTargetRepository(tmpDir)
	target, err := repo2.Get("prod")
	if err != nil {
		t.Fatalf("Get from fresh repo: %v", err)
	}
	if target.Label != "Production" {
		t.Errorf("unexpected label: %q", target.Label)
	}
}
