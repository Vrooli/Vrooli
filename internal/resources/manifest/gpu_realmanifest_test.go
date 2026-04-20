package manifest_test

import (
	"path/filepath"
	"testing"

	manifest "github.com/vrooli/vrooli/internal/resources/manifest"
)

// Ensures the whisper and kokoro manifests we just edited still parse and
// validate after adding the gpu block, and that the gpu block is wired
// through with the expected probe.
func TestRealManifestsWithGPUBlockLoad(t *testing.T) {
	repoRoot := findRepoRoot(t)
	cases := []struct {
		name        string
		overlay     string
		imageEnvKey string
	}{
		{"whisper", "docker/docker-compose.gpu.yml", "WHISPER_IMAGE"},
		{"kokoro", "docker/docker-compose.gpu.yml", "KOKORO_IMAGE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(repoRoot, "resources", tc.name, "resource.json")
			m, err := manifest.Load(path)
			if err != nil {
				t.Fatalf("load %s: %v", path, err)
			}
			if m.GPU == nil {
				t.Fatal("expected gpu block to be populated")
			}
			if m.GPU.Probe != "nvidia" {
				t.Fatalf("expected nvidia probe, got %q", m.GPU.Probe)
			}
			if m.GPU.ComposeOverlay != tc.overlay {
				t.Fatalf("expected overlay %q, got %q", tc.overlay, m.GPU.ComposeOverlay)
			}
			if _, ok := m.GPU.EnvOverrides[tc.imageEnvKey]; !ok {
				t.Fatalf("expected %s in env_overrides, got %v", tc.imageEnvKey, m.GPU.EnvOverrides)
			}
		})
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := filepath.Glob(filepath.Join(dir, "go.mod")); err == nil {
			matches, _ := filepath.Glob(filepath.Join(dir, "go.mod"))
			if len(matches) > 0 {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not locate repo root (go.mod)")
	return ""
}
