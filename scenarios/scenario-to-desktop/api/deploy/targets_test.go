package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

// mustSave is a test helper that saves a deploy target and fails the test on error.
func mustSave(t *testing.T, repo *TargetRepository, key string, target *DeployTarget) {
	t.Helper()
	if err := repo.Save(key, target); err != nil {
		t.Fatalf("Save(%q): %v", key, err)
	}
}

// assertListLen is a test helper that lists targets and asserts the expected count.
func assertListLen(t *testing.T, repo *TargetRepository, wantLen int, context string) {
	t.Helper()
	targets, err := repo.List()
	if err != nil {
		t.Fatalf("List (%s): %v", context, err)
	}
	if len(targets) != wantLen {
		t.Errorf("List (%s): expected %d targets, got %d", context, wantLen, len(targets))
	}
}

// assertTargetField is a test helper that gets a target and checks a field value.
func assertTargetField(t *testing.T, repo *TargetRepository, key, fieldName, want string, getter func(*DeployTarget) string) {
	t.Helper()
	target, err := repo.Get(key)
	if err != nil {
		t.Fatalf("Get(%q): %v", key, err)
	}
	if got := getter(target); got != want {
		t.Errorf("target %q %s = %q, want %q", key, fieldName, got, want)
	}
}

func TestTargetRepositoryCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("create .vrooli dir: %v", err)
	}

	repo := NewTargetRepository(tmpDir)

	t.Run("list empty", func(t *testing.T) {
		assertListLen(t, repo, 0, "empty")
	})

	t.Run("save and get", func(t *testing.T) {
		mustSave(t, repo, "prod", &DeployTarget{
			Label:         "Production",
			ScenarioName:  "landing-page-business-suite",
			RemoteProfile: "prod-server",
		})
		assertTargetField(t, repo, "prod", "Label", "Production", func(d *DeployTarget) string { return d.Label })
		assertTargetField(t, repo, "prod", "ScenarioName", "landing-page-business-suite", func(d *DeployTarget) string { return d.ScenarioName })
		assertTargetField(t, repo, "prod", "RemoteProfile", "prod-server", func(d *DeployTarget) string { return d.RemoteProfile })
	})

	t.Run("save another and list both", func(t *testing.T) {
		mustSave(t, repo, "staging", &DeployTarget{
			Label:         "Staging",
			ScenarioName:  "landing-page-business-suite",
			RemoteProfile: "staging-server",
		})
		assertListLen(t, repo, 2, "after second save")
	})

	t.Run("update existing", func(t *testing.T) {
		mustSave(t, repo, "prod", &DeployTarget{
			Label:         "Production (updated)",
			ScenarioName:  "landing-page-business-suite",
			RemoteProfile: "prod-server-v2",
		})
		assertTargetField(t, repo, "prod", "RemoteProfile", "prod-server-v2", func(d *DeployTarget) string { return d.RemoteProfile })
	})

	t.Run("delete", func(t *testing.T) {
		if err := repo.Delete("staging"); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		assertListLen(t, repo, 1, "after delete")
	})

	t.Run("get not found", func(t *testing.T) {
		_, err := repo.Get("nonexistent")
		if err == nil {
			t.Fatal("expected error for missing target")
		}
	})

	t.Run("delete not found", func(t *testing.T) {
		err := repo.Delete("nonexistent")
		if err == nil {
			t.Fatal("expected error for deleting missing target")
		}
	})
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
