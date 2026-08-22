package manifest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/capacity"
	manifest "github.com/vrooli/vrooli/internal/resources/manifest"
)

// Feature: the fleet migration was lossless
//
//	As the operator of a migrated fleet
//	I want proof that the authored acceleration block says exactly what the
//	three surfaces it replaced said
//	So that a silent value change cannot hide inside a mechanical migration.
//
// The pre-migration surfaces are recorded in testdata, and this test compares
// them field by field against what each manifest declares now.
//
// preMigrationSurfaces is the recorded shape of the three declarations the
// acceleration block replaced. It is declared here rather than imported: the Go
// types for those surfaces are gone, and a guard against a migration must not
// depend on the thing it migrated away from.
type preMigrationSurfaces struct {
	GPU *struct {
		Probe          string            `json:"probe"`
		ComposeOverlay string            `json:"compose_overlay"`
		EnvOverrides   map[string]string `json:"env_overrides"`
	} `json:"gpu"`
	RequirementsGPU *struct {
		MinCUDACompute string `json:"min_cuda_compute"`
	} `json:"requirements_gpu"`
	Capacity *capacity.ResourceClaimSpec `json:"capacity"`
}

func loadPreMigrationSurfaces(t *testing.T) map[string]preMigrationSurfaces {
	t.Helper()
	path := filepath.Join("testdata", "migration", "pre_migration_surfaces.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var out map[string]preMigrationSurfaces
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return out
}

// Scenario: every migrated manifest says what its legacy surfaces said.
func TestMigrationParityAgainstTheNormalisedLegacyForm(t *testing.T) {
	root := findRepoRoot(t)
	before := loadPreMigrationSurfaces(t)

	// Two resources are deliberate exceptions, and the test names them rather
	// than skipping them silently.
	deliberateChanges := map[string]string{
		// reranker declared NO accelerator surface at all while its acquisition
		// predicate selected a CUDA image at compute >= 8.9. Its new block is a
		// first declaration, not a migration.
		"reranker": "declared no accelerator surface before; the new block is a first declaration",
		// sherpa-onnx declared an empty requirements.gpu but ships no GPU
		// artifact and no overlay. It now declares nothing, honestly.
		"sherpa-onnx": "declared an empty requirements.gpu but does no accelerated work; it now declares none",
	}

	for name, surfaces := range before {
		t.Run(name, func(t *testing.T) {
			// Given the manifest as it stands after the migration
			after, err := manifest.Load(filepath.Join(root, "resources", name, "resource.json"))
			if err != nil {
				t.Fatalf("load migrated %s: %v", name, err)
			}

			// Then no legacy surface survives anywhere in the fleet. The parsed
			// struct has no fields for them, so the raw JSON is what says so.
			raw, err := os.ReadFile(filepath.Join(root, "resources", name, "resource.json"))
			if err != nil {
				t.Fatalf("read %s manifest: %v", name, err)
			}
			var keys map[string]json.RawMessage
			if err := json.Unmarshal(raw, &keys); err != nil {
				t.Fatalf("parse %s manifest: %v", name, err)
			}
			for _, legacy := range []string{"gpu", "capacity"} {
				if value, present := keys[legacy]; present && strings.TrimSpace(string(value)) != "null" {
					t.Fatalf("%s still declares the legacy %q block", name, legacy)
				}
			}
			var requirements map[string]json.RawMessage
			if json.Unmarshal(keys["requirements"], &requirements) == nil {
				if value, present := requirements["gpu"]; present && strings.TrimSpace(string(value)) != "null" {
					t.Fatalf("%s still declares requirements.gpu", name)
				}
			}

			if reason, deliberate := deliberateChanges[name]; deliberate {
				t.Logf("%s: deliberate change, not a migration — %s", name, reason)
				return
			}

			// Then the authored block carries what the legacy surfaces said.
			authored := after.Acceleration
			if authored == nil {
				t.Fatalf("%s declares no acceleration block after migration", name)
			}
			// Every legacy resource preferred its accelerator and fell back to
			// the CPU, so every migrated one still leads with cuda and ends at
			// the cpu floor. Backends added between them are later
			// cross-platform work, not migration drift, and are named rather
			// than silently allowed.
			if len(authored.Backends) < 2 {
				t.Fatalf("%s backends = %v, want at least [cuda cpu]", name, authored.Backends)
			}
			if authored.Backends[0] != manifest.BackendCUDA {
				t.Fatalf("%s leads with %q, want cuda as the legacy surfaces did", name, authored.Backends[0])
			}
			if authored.Backends[len(authored.Backends)-1] != manifest.BackendCPU {
				t.Fatalf("%s ends at %q, want the cpu floor the legacy surfaces implied", name, authored.Backends[len(authored.Backends)-1])
			}
			if added := authored.Backends[1 : len(authored.Backends)-1]; len(added) > 0 {
				t.Logf("%s: declares %v in addition to the migrated cuda/cpu pair", name, added)
			}
			if authored.EffectiveRequire() != manifest.RequirePreferred {
				t.Fatalf("%s require = %q, want %q; an empty requirements.gpu never meant mandatory", name, authored.EffectiveRequire(), manifest.RequirePreferred)
			}
			cuda, ok := authored.Config(manifest.BackendCUDA)
			if !ok {
				t.Fatalf("%s declares no cuda config after migration", name)
			}
			// requirements.gpu.min_cuda_compute -> acceleration.cuda.min_compute
			wantCompute := ""
			if surfaces.RequirementsGPU != nil {
				wantCompute = surfaces.RequirementsGPU.MinCUDACompute
			}
			if cuda.MinCompute != wantCompute {
				t.Fatalf("%s cuda.min_compute = %q, legacy requirements.gpu said %q", name, cuda.MinCompute, wantCompute)
			}
			// gpu.compose_overlay -> acceleration.cuda.compose_overlay
			// gpu.env_overrides   -> acceleration.cuda.env
			wantOverlay := ""
			wantEnv := map[string]string{}
			if surfaces.GPU != nil {
				wantOverlay = surfaces.GPU.ComposeOverlay
				wantEnv = surfaces.GPU.EnvOverrides
			}
			if cuda.ComposeOverlay != wantOverlay {
				t.Fatalf("%s cuda.compose_overlay = %q, legacy gpu block said %q", name, cuda.ComposeOverlay, wantOverlay)
			}
			if len(cuda.Env) != len(wantEnv) {
				t.Fatalf("%s cuda.env = %v, legacy gpu.env_overrides said %v", name, cuda.Env, wantEnv)
			}
			for key, value := range wantEnv {
				if cuda.Env[key] != value {
					t.Fatalf("%s cuda.env[%q] = %q, legacy said %q", name, key, cuda.Env[key], value)
				}
			}

			// And the claim carries the same reservation, byte for byte
			assertClaimParity(t, name, authored.Claim, surfaces.Capacity)
		})
	}
}

// assertClaimParity compares the reservation field by field. Phase 11 changed
// two claims deliberately, so the comparison names what it allows to differ.
func assertClaimParity(t *testing.T, name string, got, want *capacity.ResourceClaimSpec) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Fatalf("%s declares a claim after migration but had none before", name)
		}
		return
	}
	if got == nil {
		t.Fatalf("%s declared a claim before migration and declares none now", name)
	}
	if got.ResourceKind != want.ResourceKind {
		t.Fatalf("%s claim.resource_kind = %q, legacy said %q", name, got.ResourceKind, want.ResourceKind)
	}
	if got.PreferredBytes != want.PreferredBytes {
		t.Fatalf("%s claim.preferred_bytes = %d, legacy said %d", name, got.PreferredBytes, want.PreferredBytes)
	}
	if got.Priority != want.Priority {
		t.Fatalf("%s claim.priority = %q, legacy said %q", name, got.Priority, want.Priority)
	}
	// Phase 11 gave kokoro and speaker-verification the degrade ladder and the
	// yield flag they lacked, and moved kokoro's floor to the ladder's last
	// rung. Those two are recorded changes, not migration drift.
	if want.Profile == nil {
		if got.Profile == nil {
			t.Fatalf("%s still declares a VRAM claim with no degrade ladder", name)
		}
		t.Logf("%s: gained a degrade ladder and yield_when_idle in the verb phase; floor moved %d -> %d", name, want.FloorBytes, got.FloorBytes)
		return
	}
	if got.FloorBytes != want.FloorBytes {
		t.Fatalf("%s claim.floor_bytes = %d, legacy said %d", name, got.FloorBytes, want.FloorBytes)
	}
	if got.YieldWhenIdle != want.YieldWhenIdle {
		t.Fatalf("%s claim.yield_when_idle = %v, legacy said %v", name, got.YieldWhenIdle, want.YieldWhenIdle)
	}
	if len(got.Profile.Steps) != len(want.Profile.Steps) {
		t.Fatalf("%s claim ladder has %d steps, legacy had %d", name, len(got.Profile.Steps), len(want.Profile.Steps))
	}
	for i, step := range want.Profile.Steps {
		if got.Profile.Steps[i] != step {
			t.Fatalf("%s claim ladder step %d = %+v, legacy said %+v", name, i, got.Profile.Steps[i], step)
		}
	}
}
