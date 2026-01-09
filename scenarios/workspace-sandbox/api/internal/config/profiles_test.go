package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultProfiles(t *testing.T) {
	profiles := DefaultProfiles()

	if len(profiles) != 2 {
		t.Errorf("expected 2 default profiles, got %d", len(profiles))
	}

	// Check full profile
	var fullProfile *IsolationProfile
	for i := range profiles {
		if profiles[i].ID == "full" {
			fullProfile = &profiles[i]
			break
		}
	}
	if fullProfile == nil {
		t.Fatal("expected 'full' profile to exist")
	}
	if !fullProfile.Builtin {
		t.Error("expected 'full' profile to be builtin")
	}
	if fullProfile.NetworkAccess != "none" {
		t.Errorf("expected 'full' network access to be 'none', got %s", fullProfile.NetworkAccess)
	}

	// Check vrooli-aware profile
	var vrooliProfile *IsolationProfile
	for i := range profiles {
		if profiles[i].ID == "vrooli-aware" {
			vrooliProfile = &profiles[i]
			break
		}
	}
	if vrooliProfile == nil {
		t.Fatal("expected 'vrooli-aware' profile to exist")
	}
	if !vrooliProfile.Builtin {
		t.Error("expected 'vrooli-aware' profile to be builtin")
	}
	if vrooliProfile.NetworkAccess != "localhost" {
		t.Errorf("expected 'vrooli-aware' network access to be 'localhost', got %s", vrooliProfile.NetworkAccess)
	}
}

func TestFileProfileStore(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "profiles-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewFileProfileStore(tmpDir)

	t.Run("List returns builtin profiles when no custom profiles exist", func(t *testing.T) {
		profiles, err := store.List()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(profiles) != 2 {
			t.Errorf("expected 2 profiles (builtins), got %d", len(profiles))
		}
	})

	t.Run("Get returns builtin profile", func(t *testing.T) {
		profile, err := store.Get("full")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if profile.ID != "full" {
			t.Errorf("expected ID 'full', got %s", profile.ID)
		}
		if !profile.Builtin {
			t.Error("expected profile to be builtin")
		}
	})

	t.Run("Get returns error for non-existent profile", func(t *testing.T) {
		_, err := store.Get("non-existent")
		if err == nil {
			t.Error("expected error for non-existent profile")
		}
	})

	t.Run("Save creates custom profile", func(t *testing.T) {
		custom := IsolationProfile{
			ID:            "custom-test",
			Name:          "Custom Test",
			Description:   "A test profile",
			NetworkAccess: "full",
			ReadOnlyBinds: map[string]string{"/usr": "/usr"},
		}
		err := store.Save(custom)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify it's in the list
		profiles, err := store.List()
		if err != nil {
			t.Fatalf("unexpected error listing: %v", err)
		}
		if len(profiles) != 3 {
			t.Errorf("expected 3 profiles (2 builtin + 1 custom), got %d", len(profiles))
		}

		// Verify we can get it
		retrieved, err := store.Get("custom-test")
		if err != nil {
			t.Fatalf("unexpected error getting: %v", err)
		}
		if retrieved.Name != "Custom Test" {
			t.Errorf("expected Name 'Custom Test', got %s", retrieved.Name)
		}
	})

	t.Run("Save updates existing custom profile", func(t *testing.T) {
		updated := IsolationProfile{
			ID:            "custom-test",
			Name:          "Updated Test",
			Description:   "Updated description",
			NetworkAccess: "localhost",
		}
		err := store.Save(updated)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		retrieved, err := store.Get("custom-test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if retrieved.Name != "Updated Test" {
			t.Errorf("expected Name 'Updated Test', got %s", retrieved.Name)
		}
	})

	t.Run("Save returns error for builtin profile", func(t *testing.T) {
		err := store.Save(IsolationProfile{ID: "full", Name: "Modified Full"})
		if err == nil {
			t.Error("expected error when modifying builtin profile")
		}
	})

	t.Run("Delete removes custom profile", func(t *testing.T) {
		err := store.Delete("custom-test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = store.Get("custom-test")
		if err == nil {
			t.Error("expected error getting deleted profile")
		}
	})

	t.Run("Delete returns error for builtin profile", func(t *testing.T) {
		err := store.Delete("full")
		if err == nil {
			t.Error("expected error when deleting builtin profile")
		}
	})

	t.Run("Delete returns error for non-existent profile", func(t *testing.T) {
		err := store.Delete("non-existent")
		if err == nil {
			t.Error("expected error when deleting non-existent profile")
		}
	})
}

func TestFileProfileStorePersistence(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "profiles-persist-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create and save a profile
	store1 := NewFileProfileStore(tmpDir)
	custom := IsolationProfile{
		ID:            "persist-test",
		Name:          "Persistence Test",
		Description:   "Tests persistence",
		NetworkAccess: "none",
		Hostname:      "test-sandbox",
	}
	if err := store1.Save(custom); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Create new store instance to verify file was written
	store2 := NewFileProfileStore(tmpDir)
	retrieved, err := store2.Get("persist-test")
	if err != nil {
		t.Fatalf("failed to get after reload: %v", err)
	}
	if retrieved.Name != "Persistence Test" {
		t.Errorf("expected Name 'Persistence Test', got %s", retrieved.Name)
	}
	if retrieved.Hostname != "test-sandbox" {
		t.Errorf("expected Hostname 'test-sandbox', got %s", retrieved.Hostname)
	}

	// Verify file exists
	filePath := filepath.Join(tmpDir, ".vrooli", "workspace-sandbox-profiles.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("expected profiles file to exist")
	}
}

func TestFileProfileStoreReload(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "profiles-reload-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewFileProfileStore(tmpDir)

	// Save a profile
	if err := store.Save(IsolationProfile{ID: "reload-test", Name: "Before Reload"}); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Modify the file directly (simulating external modification)
	filePath := filepath.Join(tmpDir, ".vrooli", "workspace-sandbox-profiles.json")
	newContent := `[{"id":"reload-test","name":"After External Edit","description":"","builtin":false,"networkAccess":"","readOnlyBinds":null,"readWriteBinds":null,"environment":null,"hostname":""}]`
	if err := os.WriteFile(filePath, []byte(newContent), 0o644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Without reload, should still see old cached value
	cached, _ := store.Get("reload-test")
	if cached.Name != "Before Reload" {
		t.Errorf("expected cached Name 'Before Reload', got %s", cached.Name)
	}

	// After reload, should see new value
	if err := store.Reload(); err != nil {
		t.Fatalf("failed to reload: %v", err)
	}
	reloaded, _ := store.Get("reload-test")
	if reloaded.Name != "After External Edit" {
		t.Errorf("expected reloaded Name 'After External Edit', got %s", reloaded.Name)
	}
}
