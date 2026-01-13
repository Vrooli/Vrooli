package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeManifestHash(t *testing.T) {
	// Create temp directory and file for testing
	tmpDir := t.TempDir()

	t.Run("empty path returns empty hash", func(t *testing.T) {
		hash, mtime, err := ComputeManifestHash("")
		if err != nil {
			t.Fatalf("ComputeManifestHash(\"\") error = %v", err)
		}
		if hash != "" {
			t.Errorf("hash = %q, want empty", hash)
		}
		if mtime != 0 {
			t.Errorf("mtime = %d, want 0", mtime)
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, _, err := ComputeManifestHash("/nonexistent/file.json")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("valid file returns hash", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "test-manifest.json")
		content := []byte(`{"name": "test", "version": "1.0.0"}`)
		if err := os.WriteFile(testFile, content, 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		hash, mtime, err := ComputeManifestHash(testFile)
		if err != nil {
			t.Fatalf("ComputeManifestHash() error = %v", err)
		}
		if hash == "" {
			t.Error("hash should not be empty")
		}
		if mtime == 0 {
			t.Error("mtime should not be 0")
		}
		// SHA256 produces 64-character hex string
		if len(hash) != 64 {
			t.Errorf("hash length = %d, want 64", len(hash))
		}
	})

	t.Run("same content produces same hash", func(t *testing.T) {
		content := []byte(`{"test": "data"}`)

		file1 := filepath.Join(tmpDir, "file1.json")
		file2 := filepath.Join(tmpDir, "file2.json")
		os.WriteFile(file1, content, 0644)
		os.WriteFile(file2, content, 0644)

		hash1, _, _ := ComputeManifestHash(file1)
		hash2, _, _ := ComputeManifestHash(file2)

		if hash1 != hash2 {
			t.Errorf("same content should produce same hash: %s != %s", hash1, hash2)
		}
	})

	t.Run("different content produces different hash", func(t *testing.T) {
		file1 := filepath.Join(tmpDir, "diff1.json")
		file2 := filepath.Join(tmpDir, "diff2.json")
		os.WriteFile(file1, []byte(`{"a": 1}`), 0644)
		os.WriteFile(file2, []byte(`{"a": 2}`), 0644)

		hash1, _, _ := ComputeManifestHash(file1)
		hash2, _, _ := ComputeManifestHash(file2)

		if hash1 == hash2 {
			t.Error("different content should produce different hash")
		}
	})
}

func TestComputeStateHash(t *testing.T) {
	t.Run("nil state returns empty", func(t *testing.T) {
		hash := ComputeStateHash(nil)
		if hash != "" {
			t.Errorf("ComputeStateHash(nil) = %q, want empty", hash)
		}
	})

	t.Run("state produces hash", func(t *testing.T) {
		state := &ScenarioState{
			ScenarioName:  "test-scenario",
			SchemaVersion: SchemaVersion,
			FormState: FormState{
				SelectedTemplate: "basic",
				Framework:        "electron",
			},
		}

		hash := ComputeStateHash(state)
		if hash == "" {
			t.Error("hash should not be empty")
		}
		if len(hash) != 64 {
			t.Errorf("hash length = %d, want 64", len(hash))
		}
	})

	t.Run("hash excludes Hash field", func(t *testing.T) {
		state1 := &ScenarioState{
			ScenarioName: "test",
			Hash:         "existing-hash",
		}
		state2 := &ScenarioState{
			ScenarioName: "test",
			Hash:         "different-hash",
		}

		hash1 := ComputeStateHash(state1)
		hash2 := ComputeStateHash(state2)

		if hash1 != hash2 {
			t.Error("Hash field should be excluded from computation")
		}
	})

	t.Run("different content produces different hash", func(t *testing.T) {
		state1 := &ScenarioState{ScenarioName: "scenario1"}
		state2 := &ScenarioState{ScenarioName: "scenario2"}

		hash1 := ComputeStateHash(state1)
		hash2 := ComputeStateHash(state2)

		if hash1 == hash2 {
			t.Error("different states should produce different hashes")
		}
	})
}

func TestComputeFingerprintHash(t *testing.T) {
	t.Run("nil fingerprint returns empty", func(t *testing.T) {
		hash := ComputeFingerprintHash(nil)
		if hash != "" {
			t.Errorf("ComputeFingerprintHash(nil) = %q, want empty", hash)
		}
	})

	t.Run("fingerprint produces short hash", func(t *testing.T) {
		fp := &InputFingerprint{
			ManifestPath: "/path/to/manifest.json",
			ManifestHash: "abc123",
		}

		hash := ComputeFingerprintHash(fp)
		if hash == "" {
			t.Error("hash should not be empty")
		}
		// Short hash is 16 characters (8 bytes hex-encoded)
		if len(hash) != 16 {
			t.Errorf("hash length = %d, want 16", len(hash))
		}
	})
}

func TestComputeFormStateHash(t *testing.T) {
	t.Run("nil form state returns empty", func(t *testing.T) {
		hash := ComputeFormStateHash(nil)
		if hash != "" {
			t.Errorf("ComputeFormStateHash(nil) = %q, want empty", hash)
		}
	})

	t.Run("form state produces hash", func(t *testing.T) {
		fs := &FormState{
			SelectedTemplate: "basic",
			Framework:        "electron",
			AppDisplayName:   "My App",
		}

		hash := ComputeFormStateHash(fs)
		if hash == "" {
			t.Error("hash should not be empty")
		}
		if len(hash) != 64 {
			t.Errorf("hash length = %d, want 64", len(hash))
		}
	})
}

func TestExtractBundleFingerprint(t *testing.T) {
	fs := &FormState{
		BundleManifestPath: "/path/to/manifest.json",
	}

	fp := ExtractBundleFingerprint(fs, "hash123", 1234567890)

	if fp.ManifestPath != fs.BundleManifestPath {
		t.Errorf("ManifestPath = %v, want %v", fp.ManifestPath, fs.BundleManifestPath)
	}
	if fp.ManifestHash != "hash123" {
		t.Errorf("ManifestHash = %v, want hash123", fp.ManifestHash)
	}
	if fp.ManifestMtime != 1234567890 {
		t.Errorf("ManifestMtime = %v, want 1234567890", fp.ManifestMtime)
	}
}

func TestExtractPreflightFingerprint(t *testing.T) {
	fs := &FormState{
		BundleManifestPath: "/path/to/manifest.json",
		PreflightSecrets: map[string]string{
			"SECRET_B": "value2",
			"SECRET_A": "value1",
		},
		PreflightStartServices: true,
	}

	fp := ExtractPreflightFingerprint(fs, "hash123")

	if fp.ManifestPath != fs.BundleManifestPath {
		t.Errorf("ManifestPath = %v, want %v", fp.ManifestPath, fs.BundleManifestPath)
	}
	if fp.ManifestHash != "hash123" {
		t.Errorf("ManifestHash = %v, want hash123", fp.ManifestHash)
	}
	// Secret keys should be sorted
	if len(fp.PreflightSecretKeys) != 2 {
		t.Fatalf("expected 2 secret keys, got %d", len(fp.PreflightSecretKeys))
	}
	if fp.PreflightSecretKeys[0] != "SECRET_A" || fp.PreflightSecretKeys[1] != "SECRET_B" {
		t.Errorf("secret keys should be sorted: %v", fp.PreflightSecretKeys)
	}
	if !fp.StartServices {
		t.Error("StartServices should be true")
	}
}

func TestExtractGenerateFingerprint(t *testing.T) {
	fs := &FormState{
		SelectedTemplate: "spa",
		Framework:        "electron",
		DeploymentMode:   "bundled",
		AppDisplayName:   "My App",
		AppDescription:   "Description",
		IconPath:         "/path/to/icon.png",
	}

	fp := ExtractGenerateFingerprint(fs)

	if fp.TemplateType != "spa" {
		t.Errorf("TemplateType = %v, want spa", fp.TemplateType)
	}
	if fp.Framework != "electron" {
		t.Errorf("Framework = %v, want electron", fp.Framework)
	}
	if fp.DeploymentMode != "bundled" {
		t.Errorf("DeploymentMode = %v, want bundled", fp.DeploymentMode)
	}
	if fp.AppDisplayName != "My App" {
		t.Errorf("AppDisplayName = %v, want My App", fp.AppDisplayName)
	}
}

func TestExtractBuildFingerprint(t *testing.T) {
	fs := &FormState{
		Platforms: PlatformSelection{
			Win:   true,
			Mac:   false,
			Linux: true,
		},
		SigningEnabledForBuild: true,
		LocationMode:           "proper",
	}

	fp := ExtractBuildFingerprint(fs, "signing-hash")

	// Platforms should be sorted
	if len(fp.Platforms) != 2 {
		t.Fatalf("expected 2 platforms, got %d", len(fp.Platforms))
	}
	if fp.Platforms[0] != "linux" || fp.Platforms[1] != "win" {
		t.Errorf("platforms should be sorted: %v", fp.Platforms)
	}
	if !fp.SigningEnabled {
		t.Error("SigningEnabled should be true")
	}
	if fp.SigningConfigHash != "signing-hash" {
		t.Errorf("SigningConfigHash = %v, want signing-hash", fp.SigningConfigHash)
	}
	if fp.OutputLocation != "proper" {
		t.Errorf("OutputLocation = %v, want proper", fp.OutputLocation)
	}
}

func TestExtractSmokeTestFingerprint(t *testing.T) {
	fp := ExtractSmokeTestFingerprint("linux")

	if fp.SmokeTestPlatform != "linux" {
		t.Errorf("SmokeTestPlatform = %v, want linux", fp.SmokeTestPlatform)
	}
}

func TestFingerprintsEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b *InputFingerprint
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil",
			a:    nil,
			b:    &InputFingerprint{ManifestPath: "/path"},
			want: false,
		},
		{
			name: "b nil",
			a:    &InputFingerprint{ManifestPath: "/path"},
			b:    nil,
			want: false,
		},
		{
			name: "equal fingerprints",
			a:    &InputFingerprint{ManifestPath: "/path", ManifestHash: "hash"},
			b:    &InputFingerprint{ManifestPath: "/path", ManifestHash: "hash"},
			want: true,
		},
		{
			name: "different fingerprints",
			a:    &InputFingerprint{ManifestPath: "/path1"},
			b:    &InputFingerprint{ManifestPath: "/path2"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FingerprintsEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("FingerprintsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManifestFingerprintChanged(t *testing.T) {
	tests := []struct {
		name    string
		stored  *InputFingerprint
		current *InputFingerprint
		want    bool
	}{
		{
			name:    "both nil",
			stored:  nil,
			current: nil,
			want:    false,
		},
		{
			name:    "stored nil, current not nil",
			stored:  nil,
			current: &InputFingerprint{ManifestPath: "/path"},
			want:    true,
		},
		{
			name:    "path changed",
			stored:  &InputFingerprint{ManifestPath: "/old/path"},
			current: &InputFingerprint{ManifestPath: "/new/path"},
			want:    true,
		},
		{
			name:    "hash changed",
			stored:  &InputFingerprint{ManifestPath: "/path", ManifestHash: "old"},
			current: &InputFingerprint{ManifestPath: "/path", ManifestHash: "new"},
			want:    true,
		},
		{
			name:    "same path and hash",
			stored:  &InputFingerprint{ManifestPath: "/path", ManifestHash: "same"},
			current: &InputFingerprint{ManifestPath: "/path", ManifestHash: "same"},
			want:    false,
		},
		{
			name:    "stored hash empty (no comparison)",
			stored:  &InputFingerprint{ManifestPath: "/path", ManifestHash: ""},
			current: &InputFingerprint{ManifestPath: "/path", ManifestHash: "new"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ManifestFingerprintChanged(tt.stored, tt.current)
			if got != tt.want {
				t.Errorf("ManifestFingerprintChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}
