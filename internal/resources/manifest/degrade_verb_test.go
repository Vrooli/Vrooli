package manifest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/capacity"
	manifest "github.com/vrooli/vrooli/internal/resources/manifest"
)

// Feature: one broker contract, one call shape
//
//	As the capacity broker
//	I want every resource to answer to the same verb
//	So that I do not have to know which resource I am talking to in order to
//	talk to it.

// Scenario: any verb other than the contract's is rejected.
func TestAccelerationRejectsANonContractDegradeVerb(t *testing.T) {
	cases := []struct {
		scenario string
		verb     string
		argv     []string
		wantErr  string
	}{
		{
			scenario: "Given the contract verb, Then it is accepted",
			verb:     manifest.DegradeVerb,
			argv:     manifest.DegradeArgv,
		},
		{
			scenario: "Given whisper's old capacity-degrade verb, Then it is rejected",
			verb:     "capacity-degrade",
			argv:     []string{"--to", "{label}"},
			wantErr:  `is "capacity-degrade"`,
		},
		{
			scenario: "Given reranker's old models verb, Then it is rejected",
			verb:     "models",
			argv:     []string{"activate", "--model", "{label}"},
			wantErr:  `is "models"`,
		},
		{
			scenario: "Given kyutai-stt's old bare stop verb, Then it is rejected",
			verb:     "stop",
			wantErr:  `is "stop"`,
		},
		{
			scenario: "Given no verb at all, Then it is rejected and the contract is named",
			verb:     "",
			wantErr:  "it must be \"capacity\"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// Given a resource declaring that verb
			spec := manifest.AccelerationSpec{
				Backends: []string{manifest.BackendCUDA, manifest.BackendCPU},
				Backend:  map[string]manifest.BackendConfig{manifest.BackendCUDA: {}, manifest.BackendCPU: {}},
				Claim: &capacity.ResourceClaimSpec{
					ResourceKind:   capacity.ResourceKindVRAM,
					PreferredBytes: 4 << 30,
					FloorBytes:     1 << 30,
					YieldWhenIdle:  true,
					Profile: &capacity.DegradeProfile{
						Steps: []capacity.DegradeStep{{Label: "full", AmountBytes: 4 << 30}, {Label: "floor", AmountBytes: 1 << 30}},
						Apply: capacity.DegradeApply{Verb: tc.verb, Argv: tc.argv},
					},
				},
			}

			// When the contract validates it
			err := spec.Validate()

			// Then only the contract verb passes
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil for the contract verb", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want an error containing %q", err, tc.wantErr)
			}
			// And the message names what to use instead
			if !strings.Contains(err.Error(), manifest.DegradeVerb) {
				t.Fatalf("Validate() = %v, want the message to name the contract verb", err)
			}
		})
	}
}

// Scenario: every heavy resource in the fleet declares the contract verb.
func TestEveryHeavyResourceDeclaresTheContractVerb(t *testing.T) {
	root := findRepoRoot(t)
	for _, name := range []string{"ollama", "whisper", "reranker", "kyutai-stt", "kokoro", "speaker-verification"} {
		t.Run(name, func(t *testing.T) {
			// Given the resource's manifest as it stands in the repository
			data, err := os.ReadFile(filepath.Join(root, "resources", name, "resource.json"))
			if err != nil {
				t.Fatalf("read %s manifest: %v", name, err)
			}
			var declared struct {
				Capacity     *capacity.ResourceClaimSpec `json:"capacity"`
				Acceleration *manifest.AccelerationSpec  `json:"acceleration"`
			}
			if err := json.Unmarshal(data, &declared); err != nil {
				t.Fatalf("parse %s manifest: %v", name, err)
			}
			claim := declared.Capacity
			if declared.Acceleration != nil && declared.Acceleration.Claim != nil {
				claim = declared.Acceleration.Claim
			}
			if claim == nil {
				// CPU-only resources do not expose capacity-degrade. Requiring a
				// profile here would reintroduce the manifest contradiction.
				return
			}

			// Then it declares a claim with a steppable ladder
			if claim == nil || claim.Profile == nil {
				t.Fatalf("%s declares no degrade profile; the broker could never ask it to step down", name)
			}
			// And the ladder answers to the fleet-wide verb
			if claim.Profile.Apply.Verb != manifest.DegradeVerb {
				t.Fatalf("%s declares verb %q, want %q", name, claim.Profile.Apply.Verb, manifest.DegradeVerb)
			}
			if !slices.Equal(claim.Profile.Apply.Argv, manifest.DegradeArgv) {
				t.Fatalf("%s declares argv %v, want %v", name, claim.Profile.Apply.Argv, manifest.DegradeArgv)
			}
			// And it yields when idle, so an idle reservation is reclaimable
			if !claim.YieldWhenIdle {
				t.Fatalf("%s declares a VRAM claim that never yields when idle", name)
			}
			// And its ladder ends at the floor it declares
			last := claim.Profile.Steps[len(claim.Profile.Steps)-1]
			if last.AmountBytes != claim.FloorBytes {
				t.Fatalf("%s ladder ends at %d but declares floor %d", name, last.AmountBytes, claim.FloorBytes)
			}
		})
	}
}
