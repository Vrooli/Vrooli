package config

import (
	"os"
	"path/filepath"
	"testing"

	"workspace-sandbox/internal/types"
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

	store := NewFileProfileStoreAtPath(filepath.Join(tmpDir, "profiles.json"))

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

	t.Run("vrooli-aware profile delegates $HOME visibility to the home overlay", func(t *testing.T) {
		// 2026-04-28 home-overlay refactor: $HOME-relative state is no
		// longer surfaced through ad-hoc per-subpath ReadOnlyBinds.
		// Instead, driver.Mount creates a per-sandbox fuse-overlayfs
		// over the host $HOME and bwrap binds the merged dir at the
		// host $HOME path inside the namespace. The profile must NOT
		// re-introduce per-subpath binds (they shadow the overlay) and
		// must point HOME at the host home so $HOME-relative lookups
		// resolve to the overlay's merged dir.
		profile, err := store.Get("vrooli-aware")
		if err != nil {
			t.Fatalf("Get(vrooli-aware): %v", err)
		}
		for _, k := range []string{
			"$HOME/.local/bin",
			"$HOME/.local/share",
			"$HOME/.config/vrooli",
			"$HOME/.claude",
		} {
			if _, ok := profile.ReadOnlyBinds[k]; ok {
				t.Errorf("vrooli-aware profile must not bind %s — the home overlay covers it; per-subpath binds shadow the overlay and reintroduce drift", k)
			}
		}
		if got := profile.Environment["HOME"]; got != "$HOME" {
			t.Errorf("vrooli-aware profile HOME = %q; want %q so $HOME-relative lookups resolve to the home overlay merged dir, not /tmp", got, "$HOME")
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
	store1 := NewFileProfileStoreAtPath(filepath.Join(tmpDir, "profiles.json"))
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
	store2 := NewFileProfileStoreAtPath(filepath.Join(tmpDir, "profiles.json"))
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
	filePath := filepath.Join(tmpDir, "profiles.json")
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

	store := NewFileProfileStoreAtPath(filepath.Join(tmpDir, "profiles.json"))

	// Save a profile
	if err := store.Save(IsolationProfile{ID: "reload-test", Name: "Before Reload"}); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Modify the file directly (simulating external modification)
	filePath := filepath.Join(tmpDir, "profiles.json")
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

// TestDefaultProfiles_HomeOverlayRequirement pins the requirement value
// each builtin profile carries. Catches accidental flips of the
// vrooli-aware profile from required → optional/not_needed (which
// would silently allow exec on a broken overlay).
func TestDefaultProfiles_HomeOverlayRequirement(t *testing.T) {
	want := map[string]types.HomeOverlayRequirement{
		"full":         types.HomeOverlayNotNeeded,
		"vrooli-aware": types.HomeOverlayRequired,
	}
	for _, p := range DefaultProfiles() {
		expected, ok := want[p.ID]
		if !ok {
			continue
		}
		if p.HomeOverlayRequirement != expected {
			t.Errorf("profile %q HomeOverlayRequirement = %q, want %q",
				p.ID, p.HomeOverlayRequirement, expected)
		}
	}
}

// TestFileProfileStore_RejectsInvalidHomeOverlayRequirement asserts the
// loader-time validator: a profiles.json containing an unknown
// requirement value fails the load instead of being silently coerced.
func TestFileProfileStore_RejectsInvalidHomeOverlayRequirement(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.json")
	bad := `[{"id":"bad","name":"Bad","homeOverlayRequirement":"nonsense"}]`
	if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	store := NewFileProfileStoreAtPath(path)
	_, err := store.List()
	if err == nil {
		t.Fatal("expected error for invalid homeOverlayRequirement, got nil")
	}
}

// TestFileProfileStore_EmptyHomeOverlayRequirementDefaults asserts that
// a profiles.json that omits the field is treated as "not_needed".
// This keeps custom profiles authored before this field existed
// loadable without manual rewriting.
func TestFileProfileStore_EmptyHomeOverlayRequirementDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.json")
	good := `[{"id":"unset","name":"Unset","networkAccess":"full"}]`
	if err := os.WriteFile(path, []byte(good), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	store := NewFileProfileStoreAtPath(path)
	got, err := store.Get("unset")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.HomeOverlayRequirement != types.HomeOverlayNotNeeded {
		t.Errorf("HomeOverlayRequirement = %q, want %q",
			got.HomeOverlayRequirement, types.HomeOverlayNotNeeded)
	}
}

// TestFileProfileStore_SaveRejectsInvalidHomeOverlayRequirement covers
// the runtime API: programmatic Save with an unknown value fails.
func TestFileProfileStore_SaveRejectsInvalidHomeOverlayRequirement(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileProfileStoreAtPath(filepath.Join(tmpDir, "profiles.json"))
	err := store.Save(IsolationProfile{
		ID:                     "bad-save",
		Name:                   "Bad Save",
		HomeOverlayRequirement: "wat",
	})
	if err == nil {
		t.Fatal("expected error from Save with invalid requirement")
	}
}

func TestFileProfileStoreUsesCanonicalConfigPath(t *testing.T) {
	home := t.TempDir()
	scenarioDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	store, err := NewFileProfileStore(scenarioDir)
	if err != nil {
		t.Fatalf("NewFileProfileStore() error = %v", err)
	}

	canonicalPath := filepath.Join(home, ".config", "vrooli", "workspace-sandbox", "profiles.json")
	if store.path != canonicalPath {
		t.Fatalf("store.path = %q, want %q", store.path, canonicalPath)
	}
}
