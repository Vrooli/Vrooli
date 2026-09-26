package manifest

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	capacitypkg "github.com/vrooli/vrooli/internal/capacity"
)

type footprintPolicy struct {
	Roles map[string]struct {
		Model string `json:"model"`
	} `json:"roles"`
	Models map[string]struct {
		VRAMGBEstimate *float64 `json:"vram_gb_estimate"`
		ResidentBytes  *int64   `json:"resident_bytes"`
	} `json:"models"`
}

// validateMaterializedPolicySteps makes the policy the authority while keeping
// the claim profile self-contained for the capacity broker. Manifest steps are
// a checked materialized view, never an independent model list.
func validateMaterializedPolicySteps(manifestPath string, resourceManifest ResourceManifest) error {
	if resourceManifest.Acceleration == nil || resourceManifest.Acceleration.Claim == nil || resourceManifest.Acceleration.Claim.Profile == nil {
		return nil
	}
	profile := resourceManifest.Acceleration.Claim.Profile
	source := strings.TrimSpace(profile.StepsSource)
	if source == "" {
		return nil
	}
	if filepath.Base(source) != source {
		return fmt.Errorf("acceleration.claim.profile.steps_source must name a sibling file")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(manifestPath), source))
	if err != nil {
		return fmt.Errorf("read acceleration.claim.profile.steps_source %s: %w", source, err)
	}
	var policy footprintPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return fmt.Errorf("parse acceleration.claim.profile.steps_source %s: %w", source, err)
	}
	derived, err := derivePolicySteps(policy)
	if err != nil {
		return fmt.Errorf("derive acceleration.claim.profile.steps from %s: %w", source, err)
	}
	actual := profile.Steps
	// A policy owns model-footprint rungs. A terminal CPU rung is an execution
	// mode with zero device memory, not a model, so it remains manifest-owned.
	if len(actual) > 0 && actual[len(actual)-1].Label == "cpu" && actual[len(actual)-1].AmountBytes == 0 {
		actual = actual[:len(actual)-1]
	}
	actualByLabel := make(map[string]int64, len(actual))
	for _, step := range actual {
		actualByLabel[step.Label] = step.AmountBytes
	}
	derivedByLabel := make(map[string]int64, len(derived))
	for _, step := range derived {
		derivedByLabel[step.Label] = step.AmountBytes
	}
	for _, step := range actual {
		if _, servesRole := derivedByLabel[step.Label]; !servesRole {
			return fmt.Errorf("acceleration.claim.profile rung %q serves no role in %s", step.Label, source)
		}
	}
	for _, step := range derived {
		amount, present := actualByLabel[step.Label]
		if !present {
			return fmt.Errorf("policy-served model %q is not an acceleration.claim.profile rung", step.Label)
		}
		if amount != step.AmountBytes {
			return fmt.Errorf("acceleration.claim.profile rung %q has %d bytes; %s declares %d", step.Label, amount, source, step.AmountBytes)
		}
	}
	if len(actual) != len(derived) {
		return fmt.Errorf("acceleration.claim.profile has %d steps; %s derives %d", len(actual), source, len(derived))
	}
	for index := range derived {
		if actual[index] != derived[index] {
			return fmt.Errorf("acceleration.claim.profile steps are not the footprint-descending materialization of %s", source)
		}
	}
	claim := resourceManifest.Acceleration.Claim
	if claim.PreferredBytes != derived[0].AmountBytes {
		return fmt.Errorf("acceleration.claim.default_preferred_bytes must equal the first policy-derived rung (%d)", derived[0].AmountBytes)
	}
	wantFloor := derived[len(derived)-1].AmountBytes
	if len(profile.Steps) > len(actual) {
		wantFloor = 0
	}
	if claim.FloorBytes != wantFloor {
		return fmt.Errorf("acceleration.claim.floor_bytes must equal the terminal policy or cpu rung (%d)", wantFloor)
	}
	return nil
}

func derivePolicySteps(policy footprintPolicy) ([]capacitypkg.DegradeStep, error) {
	models := make(map[string]struct{}, len(policy.Roles))
	for role, binding := range policy.Roles {
		model := strings.TrimSpace(binding.Model)
		if model == "" {
			return nil, fmt.Errorf("role %q has no model", role)
		}
		models[model] = struct{}{}
	}
	steps := make([]capacitypkg.DegradeStep, 0, len(models))
	for model := range models {
		definition, ok := policy.Models[model]
		if !ok {
			return nil, fmt.Errorf("policy-served model %q has no models entry", model)
		}
		var amount int64
		switch {
		case definition.ResidentBytes != nil && *definition.ResidentBytes > 0:
			amount = *definition.ResidentBytes
		case definition.VRAMGBEstimate != nil && *definition.VRAMGBEstimate > 0:
			amount = int64(math.Round(*definition.VRAMGBEstimate * 1024 * 1024 * 1024))
		default:
			return nil, fmt.Errorf("policy-served model %q has no positive resident footprint", model)
		}
		steps = append(steps, capacitypkg.DegradeStep{Label: model, AmountBytes: amount})
	}
	sort.Slice(steps, func(i, j int) bool {
		if steps[i].AmountBytes == steps[j].AmountBytes {
			return steps[i].Label < steps[j].Label
		}
		return steps[i].AmountBytes > steps[j].AmountBytes
	})
	if len(steps) == 0 {
		return nil, fmt.Errorf("policy defines no served models")
	}
	return steps, nil
}
