package facts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestDiscoverDeclaredComponentSurfaces(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".vrooli", "service.json"), `{"components": {
		"api": {"role":"api", "build":{"dir":"backend"}},
		"ui": {"role":"ui", "build":{"dir":"ui"}},
		"playwright-driver": {"role":"sidecar", "build":{"dir":"playwright-driver"}},
		"queue": {"role":"worker", "run":{"cwd":"queue"}},
		"absent": {"role":"sidecar", "build":{"dir":"absent"}}
	}}`)
	for _, dir := range []string{"backend", "ui", "playwright-driver", "queue"} {
		writeFile(t, filepath.Join(root, dir, "package.json"), `{"devDependencies":{"jest":"29"}}`)
	}
	surfaces := discoverSurfaces(&factsv1.TargetContext{RootPath: root, ScenarioAware: true})
	seen := map[string]bool{}
	for _, surface := range surfaces {
		require.False(t, seen[surface.GetId()], "component must have one surface: %s", surface.GetId())
		seen[surface.GetId()] = true
	}
	for _, tc := range []struct {
		id, dir string
		kind    factsv1.SurfaceKind
		status  factsv1.SurfaceStatus
	}{
		{"api", "backend", factsv1.SurfaceKind_SURFACE_KIND_API, factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN},
		{"ui", "ui", factsv1.SurfaceKind_SURFACE_KIND_UI, factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN},
		{"playwright-driver", "playwright-driver", factsv1.SurfaceKind_SURFACE_KIND_SIDECAR, factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN},
		{"queue", "queue", factsv1.SurfaceKind_SURFACE_KIND_WORKER, factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN},
		{"absent", "absent", factsv1.SurfaceKind_SURFACE_KIND_SIDECAR, factsv1.SurfaceStatus_SURFACE_STATUS_MISSING},
	} {
		t.Run(tc.id, func(t *testing.T) {
			surface := findSurface(surfaces, tc.id)
			require.NotNil(t, surface, "declared components must not disappear from owner validation")
			require.Equal(t, filepath.Join(root, tc.dir), surface.GetPath())
			require.Equal(t, tc.kind, surface.GetKind())
			require.Equal(t, tc.status, surface.GetStatus())
		})
	}
}

func TestDeclaredComponentRootsStayInsideTarget(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(outside, "package.json"), `{}`)
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "linked")))
	for _, dir := range []string{"../outside", outside, "linked"} {
		t.Run(dir, func(t *testing.T) {
			manifest, err := json.Marshal(map[string]any{"components": map[string]any{
				"driver": map[string]any{"role": "sidecar", "build": map[string]string{"dir": dir}},
			}})
			require.NoError(t, err)
			writeFile(t, filepath.Join(root, ".vrooli", "service.json"), string(manifest))
			surfaces := discoverSurfaces(&factsv1.TargetContext{RootPath: root, ScenarioAware: true})
			surface := findSurface(surfaces, "driver")
			require.NotNil(t, surface, "unsafe declaration must remain explicit evidence")
			require.Equal(t, factsv1.SurfaceStatus_SURFACE_STATUS_UNSUPPORTED, surface.GetStatus())
			require.Empty(t, surface.GetPath(), "downstream validation must not execute an outside root")
		})
	}
}

func TestDiscoverNestedParseUnitsEmitsPolyglotDependencyEvidence(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{
		"web/bun.lock":           `{"lockfileVersion":1}`,
		"tools/requirements.txt": "requests==2.32.0\n",
		"native/Cargo.toml":      "[package]\nname = \"demo\"\nversion = \"0.1.0\"\n",
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	units := discoverNestedParseUnits(root)
	seen := map[string]bool{}
	for _, unit := range units {
		seen[unit.GetLanguage()] = true
	}
	for _, language := range []string{"node", "python", "rust"} {
		if !seen[language] {
			t.Errorf("language %q not discovered: %+v", language, units)
		}
	}
	for _, unit := range units {
		if unit.GetToolchain() == nil {
			t.Errorf("%s parse unit has no neutral toolchain observation", unit.GetId())
		}
	}
}

func TestParseUnitPublishesNeutralToolchainObservation(t *testing.T) {
	root := t.TempDir()
	writeFileFixture(t, filepath.Join(root, "package.json"), `{"packageManager":"pnpm@9.0.0","scripts":{"test":"vitest run"},"devDependencies":{"vitest":"1.0.0"}}`)
	writeFileFixture(t, filepath.Join(root, "pnpm-lock.yaml"), "lockfileVersion: '9.0'\n")
	units := discoverNestedParseUnits(root)
	var node *factsv1.ParseUnit
	for _, unit := range units {
		if unit.GetLanguage() == "node" {
			node = unit
			break
		}
	}
	if node == nil || node.GetToolchain() == nil {
		t.Fatalf("node parse unit has no toolchain observation: %+v", units)
	}
	observation := node.GetToolchain()
	if observation.GetEcosystem() != "node" || observation.GetPackageManager() != "pnpm@9.0.0" {
		t.Fatalf("toolchain observation = %+v", observation)
	}
	if len(observation.GetLockfilePaths()) != 1 || len(observation.GetRunnerIndicators()) != 2 {
		t.Fatalf("toolchain paths/indicators = %+v", observation)
	}
}
