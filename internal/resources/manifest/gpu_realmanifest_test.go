package manifest_test

import (
	"path/filepath"
	"testing"

	manifest "github.com/vrooli/vrooli/internal/resources/manifest"
)

// Feature: a native managed resource selects an accelerator artifact
//
//	As kokoro
//	I want GPU facts to select a checksum-pinned native artifact
//	So that declaring an accelerator changes acquisition without Docker.

// Scenario: kokoro uses a composed native target instead of a compose overlay.
func TestNativeResourceUsesTargetSelectionInsteadOfComposeOverlay(t *testing.T) {
	cases := []struct {
		resource    string
		imageEnvKey string
	}{
		{resource: "kokoro", imageEnvKey: "KOKORO_IMAGE"},
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

			// Then no retired compose overlay is carried into the native path.
			if cuda.ComposeOverlay != "" {
				t.Fatalf("cuda.compose_overlay = %q, want empty native overlay", cuda.ComposeOverlay)
			}
			// And the retired image selector is absent; the acquisition target
			// carries the digest and the GPU predicate instead.
			if _, present := cuda.Env[tc.imageEnvKey]; present {
				t.Fatalf("cuda.env = %v, must not carry retired %s", cuda.Env, tc.imageEnvKey)
			}
			if len(m.ManagedService.Acquisition.Targets) == 0 || m.ManagedService.Acquisition.Targets[0].Kind != "" || m.ManagedService.Acquisition.Targets[0].Compose == nil {
				t.Fatal("expected a composed native acquisition target")
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
