package manifest_test

import (
	"path/filepath"
	"testing"

	manifest "github.com/vrooli/vrooli/internal/resources/manifest"
)

// Feature: a compose resource's accelerator overlay survives the migration
//
//	As kokoro
//	I want my GPU compose overlay and pinned image to reach the driver
//	So that declaring an accelerator still changes how I am started.

// Scenario: kokoro's overlay and image pin live on its cuda backend config.
func TestComposeResourceCarriesItsOverlayOnTheCUDABackend(t *testing.T) {
	cases := []struct {
		resource    string
		overlay     string
		imageEnvKey string
	}{
		{resource: "kokoro", overlay: "docker/docker-compose.gpu.yml", imageEnvKey: "KOKORO_IMAGE"},
	}

	for _, tc := range cases {
		t.Run(tc.resource, func(t *testing.T) {
			// Given the resource's manifest as it stands in the repository
			path := filepath.Join(findRepoRoot(t), "resources", tc.resource, "resource.json")
			m, err := manifest.Load(path)
			if err != nil {
				t.Fatalf("load %s: %v", path, err)
			}

			// When its accelerator declaration is read
			declaration := m.EffectiveAcceleration()
			if declaration == nil {
				t.Fatal("the resource declares no acceleration block")
			}
			cuda, ok := declaration.Config(manifest.BackendCUDA)
			if !ok {
				t.Fatal("the resource declares no cuda backend config")
			}

			// Then the overlay the compose driver layers on is there
			if cuda.ComposeOverlay != tc.overlay {
				t.Fatalf("cuda.compose_overlay = %q, want %q", cuda.ComposeOverlay, tc.overlay)
			}
			// And the pinned image env survives, so the accelerated variant is
			// still selected by digest rather than by tag
			if _, present := cuda.Env[tc.imageEnvKey]; !present {
				t.Fatalf("cuda.env = %v, want it to carry %s", cuda.Env, tc.imageEnvKey)
			}
			// And the resource still falls back to the CPU rather than failing
			if declaration.EffectiveRequire() != manifest.RequirePreferred {
				t.Fatalf("require = %q, want %q", declaration.EffectiveRequire(), manifest.RequirePreferred)
			}
			if _, ok := declaration.Config(manifest.BackendCPU); !ok {
				t.Fatal("the resource declares no cpu floor")
			}
		})
	}
}
