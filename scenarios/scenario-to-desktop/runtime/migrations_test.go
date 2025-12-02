package bundleruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"scenario-to-desktop-runtime/manifest"
)

func TestLoadMigrations(t *testing.T) {
	tmp := t.TempDir()

	t.Run("returns empty state for missing file", func(t *testing.T) {
		s := &Supervisor{
			migrationsPath: filepath.Join(tmp, "nonexistent", "migrations.json"),
		}

		state, err := s.loadMigrations()
		if err != nil {
			t.Fatalf("loadMigrations() error = %v", err)
		}
		if state.Applied == nil {
			t.Error("loadMigrations() returned nil Applied map")
		}
		if len(state.Applied) != 0 {
			t.Errorf("loadMigrations() Applied = %v, want empty", state.Applied)
		}
	})

	t.Run("loads existing state", func(t *testing.T) {
		migPath := filepath.Join(tmp, "migrations.json")
		state := migrationsState{
			AppVersion: "1.0.0",
			Applied: map[string][]string{
				"api": {"v1", "v2"},
			},
		}
		data, _ := json.Marshal(state)
		if err := os.WriteFile(migPath, data, 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		s := &Supervisor{migrationsPath: migPath}
		got, err := s.loadMigrations()
		if err != nil {
			t.Fatalf("loadMigrations() error = %v", err)
		}
		if got.AppVersion != "1.0.0" {
			t.Errorf("AppVersion = %q, want %q", got.AppVersion, "1.0.0")
		}
		if len(got.Applied["api"]) != 2 {
			t.Errorf("Applied[api] = %v, want 2 items", got.Applied["api"])
		}
	})
}

func TestPersistMigrations(t *testing.T) {
	tmp := t.TempDir()
	migPath := filepath.Join(tmp, "subdir", "migrations.json")

	s := &Supervisor{migrationsPath: migPath}
	state := migrationsState{
		AppVersion: "2.0.0",
		Applied: map[string][]string{
			"api":    {"v1", "v2", "v3"},
			"worker": {"v1"},
		},
	}

	if err := s.persistMigrations(state); err != nil {
		t.Fatalf("persistMigrations() error = %v", err)
	}

	// Verify file was created with correct content
	data, err := os.ReadFile(migPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got migrationsState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.AppVersion != "2.0.0" {
		t.Errorf("persisted AppVersion = %q, want %q", got.AppVersion, "2.0.0")
	}
	if len(got.Applied["api"]) != 3 {
		t.Errorf("persisted Applied[api] = %v, want 3 items", got.Applied["api"])
	}
}

func TestInstallPhase(t *testing.T) {
	tests := []struct {
		name           string
		savedVersion   string
		currentVersion string
		want           string
	}{
		{"first install", "", "1.0.0", "first_install"},
		{"upgrade", "1.0.0", "2.0.0", "upgrade"},
		{"current version", "1.0.0", "1.0.0", "current"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Supervisor{
				opts: Options{
					Manifest: &manifest.Manifest{
						App: manifest.App{Version: tt.currentVersion},
					},
				},
				migrations: migrationsState{AppVersion: tt.savedVersion},
			}

			got := s.installPhase()
			if got != tt.want {
				t.Errorf("installPhase() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShouldRunMigration(t *testing.T) {
	tests := []struct {
		name   string
		runOn  string
		phase  string
		expect bool
	}{
		{"always on first_install", "always", "first_install", true},
		{"always on upgrade", "always", "upgrade", true},
		{"always on current", "always", "current", true},
		{"first_install on first_install", "first_install", "first_install", true},
		{"first_install on upgrade", "first_install", "upgrade", false},
		{"first_install on current", "first_install", "current", false},
		{"upgrade on first_install", "upgrade", "first_install", false},
		{"upgrade on upgrade", "upgrade", "upgrade", true},
		{"upgrade on current", "upgrade", "current", false},
		{"empty defaults to always", "", "upgrade", true},
		{"unknown run_on", "invalid", "upgrade", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := manifest.Migration{RunOn: tt.runOn}
			got := shouldRunMigration(m, tt.phase)
			if got != tt.expect {
				t.Errorf("shouldRunMigration() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestBuildAppliedSet(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
		check    string
		exists   bool
	}{
		{"empty list", []string{}, "v1", false},
		{"version exists", []string{"v1", "v2", "v3"}, "v2", true},
		{"version missing", []string{"v1", "v2", "v3"}, "v4", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := buildAppliedSet(tt.versions)
			if set[tt.check] != tt.exists {
				t.Errorf("buildAppliedSet()[%q] = %v, want %v", tt.check, set[tt.check], tt.exists)
			}
		})
	}
}

func TestEnsureAppVersionRecorded(t *testing.T) {
	tmp := t.TempDir()
	migPath := filepath.Join(tmp, "migrations.json")

	s := &Supervisor{
		opts: Options{
			Manifest: &manifest.Manifest{
				App: manifest.App{Version: "3.0.0"},
			},
		},
		migrationsPath: migPath,
		migrations:     migrationsState{Applied: map[string][]string{}},
	}

	if err := s.ensureAppVersionRecorded(); err != nil {
		t.Fatalf("ensureAppVersionRecorded() error = %v", err)
	}

	if s.migrations.AppVersion != "3.0.0" {
		t.Errorf("migrations.AppVersion = %q, want %q", s.migrations.AppVersion, "3.0.0")
	}

	// Verify persisted
	data, err := os.ReadFile(migPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var persisted migrationsState
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if persisted.AppVersion != "3.0.0" {
		t.Errorf("persisted AppVersion = %q, want %q", persisted.AppVersion, "3.0.0")
	}
}
