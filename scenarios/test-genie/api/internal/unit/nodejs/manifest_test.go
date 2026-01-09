package nodejs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest_Valid(t *testing.T) {
	dir := t.TempDir()
	content := `{"scripts":{"test":"vitest","build":"vite build"},"packageManager":"pnpm@8.0.0"}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	manifest, err := LoadManifest(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if manifest == nil {
		t.Fatal("LoadManifest() returned nil")
	}
	if manifest.Scripts["test"] != "vitest" {
		t.Errorf("Scripts[test] = %q, want %q", manifest.Scripts["test"], "vitest")
	}
	if manifest.PackageManager != "pnpm@8.0.0" {
		t.Errorf("PackageManager = %q, want %q", manifest.PackageManager, "pnpm@8.0.0")
	}
}

func TestLoadManifest_NotFound(t *testing.T) {
	dir := t.TempDir()

	manifest, err := LoadManifest(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatalf("LoadManifest() error = %v, want nil for missing file", err)
	}
	if manifest != nil {
		t.Error("LoadManifest() returned non-nil for missing file")
	}
}

func TestLoadManifest_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{invalid`), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	_, err := LoadManifest(filepath.Join(dir, "package.json"))
	if err == nil {
		t.Error("LoadManifest() error = nil, want error for invalid JSON")
	}
}

func TestLoadManifest_NoScripts(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	manifest, err := LoadManifest(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if manifest.Scripts == nil {
		t.Error("Scripts should be initialized to empty map")
	}
}

func TestLoadManifest_DefaultTestScript(t *testing.T) {
	dir := t.TempDir()
	content := `{"scripts":{"test":"echo \"Error: no test specified\" && exit 1"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	manifest, err := LoadManifest(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if manifest.Scripts["test"] != "" {
		t.Errorf("Scripts[test] = %q, want empty string for default npm test script", manifest.Scripts["test"])
	}
}

func TestManifest_HasTestScript(t *testing.T) {
	tests := []struct {
		name     string
		manifest *Manifest
		want     bool
	}{
		{
			name:     "nil manifest",
			manifest: nil,
			want:     false,
		},
		{
			name:     "empty scripts",
			manifest: &Manifest{Scripts: map[string]string{}},
			want:     false,
		},
		{
			name:     "empty test script",
			manifest: &Manifest{Scripts: map[string]string{"test": ""}},
			want:     false,
		},
		{
			name:     "valid test script",
			manifest: &Manifest{Scripts: map[string]string{"test": "vitest"}},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.manifest.HasTestScript(); got != tt.want {
				t.Errorf("HasTestScript() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectPackageManager_FromField(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  string
	}{
		{"pnpm with version", "pnpm@8.0.0", "pnpm"},
		{"yarn with version", "yarn@3.5.0", "yarn"},
		{"npm with version", "npm@9.0.0", "npm"},
		{"pnpm uppercase", "PNPM@8.0.0", "pnpm"},
		{"empty field", "", "npm"},
		{"unknown manager", "bun@1.0.0", "npm"},
	}

	dir := t.TempDir()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := &Manifest{PackageManager: tt.field}
			if got := DetectPackageManager(manifest, dir); got != tt.want {
				t.Errorf("DetectPackageManager() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectPackageManager_FromLockfile(t *testing.T) {
	tests := []struct {
		name     string
		lockfile string
		want     string
	}{
		{"pnpm-lock.yaml", "pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn.lock", "yarn"},
		{"no lockfile", "", "npm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.lockfile != "" {
				if err := os.WriteFile(filepath.Join(dir, tt.lockfile), []byte(""), 0o644); err != nil {
					t.Fatalf("failed to create lockfile: %v", err)
				}
			}

			manifest := &Manifest{} // No packageManager field
			if got := DetectPackageManager(manifest, dir); got != tt.want {
				t.Errorf("DetectPackageManager() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectPackageManager_FieldTakesPriority(t *testing.T) {
	dir := t.TempDir()
	// Create yarn.lock but specify pnpm in packageManager
	if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0o644); err != nil {
		t.Fatalf("failed to create yarn.lock: %v", err)
	}

	manifest := &Manifest{PackageManager: "pnpm@8.0.0"}
	if got := DetectPackageManager(manifest, dir); got != "pnpm" {
		t.Errorf("DetectPackageManager() = %q, want %q (packageManager field should take priority)", got, "pnpm")
	}
}

func TestParsePackageManagerField(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"pnpm@8.0.0", "pnpm"},
		{"pnpm", "pnpm"},
		{"PNPM@8.0.0", "pnpm"},
		{"yarn@3.5.0", "yarn"},
		{"yarn", "yarn"},
		{"npm@9.0.0", "npm"},
		{"npm", "npm"},
		{"bun@1.0.0", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parsePackageManagerField(tt.input); got != tt.want {
				t.Errorf("parsePackageManagerField(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
