package bundle

import (
	"os"
	"path/filepath"
	"testing"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

func TestCollectPlatforms(t *testing.T) {
	tests := []struct {
		name     string
		manifest bundlemanifest.Manifest
		expected []string
	}{
		{
			name: "single service single platform",
			manifest: bundlemanifest.Manifest{
				Services: []bundlemanifest.Service{
					{ID: "svc1", Binaries: map[string]bundlemanifest.Binary{"linux": {Path: "/path/to/linux"}}},
				},
			},
			expected: []string{"linux"},
		},
		{
			name: "multiple services multiple platforms",
			manifest: bundlemanifest.Manifest{
				Services: []bundlemanifest.Service{
					{ID: "svc1", Binaries: map[string]bundlemanifest.Binary{"linux": {Path: "/path"}, "darwin": {Path: "/path"}}},
					{ID: "svc2", Binaries: map[string]bundlemanifest.Binary{"linux": {Path: "/path"}, "windows": {Path: "/path"}}},
				},
			},
			expected: []string{"darwin", "linux", "windows"},
		},
		{
			name: "empty services",
			manifest: bundlemanifest.Manifest{
				Services: []bundlemanifest.Service{},
			},
			expected: nil,
		},
		{
			name: "service with no binaries",
			manifest: bundlemanifest.Manifest{
				Services: []bundlemanifest.Service{
					{ID: "svc1", Binaries: map[string]bundlemanifest.Binary{}},
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectPlatforms(tt.manifest)
			if len(result) != len(tt.expected) {
				t.Errorf("collectPlatforms() = %v, want %v", result, tt.expected)
				return
			}
			for i, p := range result {
				if i < len(tt.expected) && p != tt.expected[i] {
					t.Errorf("collectPlatforms()[%d] = %q, want %q", i, p, tt.expected[i])
				}
			}
		})
	}
}

func TestBundleExtraExists(t *testing.T) {
	tests := []struct {
		name     string
		entries  []interface{}
		expected bool
	}{
		{
			name:     "empty entries",
			entries:  []interface{}{},
			expected: false,
		},
		{
			name:     "nil entries",
			entries:  nil,
			expected: false,
		},
		{
			name: "bundle from exists",
			entries: []interface{}{
				map[string]interface{}{"from": "bundle", "to": "output"},
			},
			expected: true,
		},
		{
			name: "bundle to exists",
			entries: []interface{}{
				map[string]interface{}{"from": "source", "to": "bundle"},
			},
			expected: true,
		},
		{
			name: "no bundle entry",
			entries: []interface{}{
				map[string]interface{}{"from": "source", "to": "dest"},
			},
			expected: false,
		},
		{
			name: "non-map entry",
			entries: []interface{}{
				"string entry",
				map[string]interface{}{"from": "bundle", "to": "dest"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bundleExtraExists(tt.entries)
			if result != tt.expected {
				t.Errorf("bundleExtraExists() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEnsureBundleExtraResources(t *testing.T) {
	t.Run("package.json not found", func(t *testing.T) {
		err := ensureBundleExtraResources("/nonexistent/path")
		if err == nil {
			t.Error("expected error for nonexistent path")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgPath := filepath.Join(tmpDir, "package.json")
		if err := os.WriteFile(pkgPath, []byte("not valid json"), 0o644); err != nil {
			t.Fatalf("failed to write package.json: %v", err)
		}

		err := ensureBundleExtraResources(tmpDir)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("adds bundle extra resources", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgPath := filepath.Join(tmpDir, "package.json")
		initial := []byte(`{"name": "test"}`)
		if err := os.WriteFile(pkgPath, initial, 0o644); err != nil {
			t.Fatalf("failed to write package.json: %v", err)
		}

		err := ensureBundleExtraResources(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the file was updated
		data, err := os.ReadFile(pkgPath)
		if err != nil {
			t.Fatalf("failed to read updated package.json: %v", err)
		}
		// Should contain "bundle" in the updated file
		if len(data) <= len(initial) {
			t.Error("expected package.json to be updated with extra resources")
		}
	})

	t.Run("preserves existing bundle entry", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgPath := filepath.Join(tmpDir, "package.json")
		initial := []byte(`{"name":"test","build":{"extraResources":[{"from":"bundle","to":"bundle"}]}}`)
		if err := os.WriteFile(pkgPath, initial, 0o644); err != nil {
			t.Fatalf("failed to write package.json: %v", err)
		}

		err := ensureBundleExtraResources(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestResolveManifestPath(t *testing.T) {
	fileOps := &defaultFileOperations{}

	t.Run("valid path", func(t *testing.T) {
		root := "/tmp/test"
		result, err := resolveManifestPath(fileOps, root, "subdir/file.json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(root, "subdir/file.json")
		if result != expected {
			t.Errorf("resolveManifestPath() = %q, want %q", result, expected)
		}
	})

	t.Run("path traversal attempt", func(t *testing.T) {
		root := "/tmp/test"
		_, err := resolveManifestPath(fileOps, root, "../etc/passwd")
		if err == nil {
			t.Error("expected error for path traversal")
		}
	})
}

func TestResolveBundlePath(t *testing.T) {
	fileOps := &defaultFileOperations{}
	tmpDir := t.TempDir()

	t.Run("valid path creates directory", func(t *testing.T) {
		result, err := resolveBundlePath(fileOps, tmpDir, "subdir/file.bin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(tmpDir, "subdir/file.bin")
		if result != expected {
			t.Errorf("resolveBundlePath() = %q, want %q", result, expected)
		}

		// Verify parent directory was created
		parentDir := filepath.Dir(result)
		if _, err := os.Stat(parentDir); os.IsNotExist(err) {
			t.Error("expected parent directory to be created")
		}
	})

	t.Run("path traversal attempt", func(t *testing.T) {
		_, err := resolveBundlePath(fileOps, tmpDir, "../escape/file.bin")
		if err == nil {
			t.Error("expected error for path traversal")
		}
	})
}
