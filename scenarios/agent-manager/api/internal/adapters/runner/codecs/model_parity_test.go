package codecs

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"agent-manager/internal/modelregistry"
)

// TestModelParity_CodecSubsetOfRegistry is the model drift gate required by the
// grok-runner plan (Phase 3). It asserts every codec's compiled
// SupportedModels (its curated default catalog) is a SUBSET of the operator
// catalog in config/model-registry.json for the same runner.
//
// Direction matters: codecs/*.go SupportedModels are the fallback defaults used
// when the JSON is absent; model-registry.json is the live source of truth
// (D2). Requiring codec ⊆ registry catches the failure mode where a hand-edit
// to one drifts from the other — e.g. the registry routes a runner through a
// new provider but the codec still advertises the old model ids.
//
// Runtime-discovered Ollama models (ollama/* — pulled per-host, not catalogued
// in static JSON) are excluded; ForTest codecs leave the Ollama lister nil so
// none surface here anyway.
func TestModelParity_CodecSubsetOfRegistry(t *testing.T) {
	// Parse the registry directly rather than via modelregistry.Load: this
	// gate asserts runner-key + model parity, not preset-chain validity, so it
	// must not couple to (or be masked by) unrelated preset validation.
	raw, err := os.ReadFile(modelregistry.ResolvePath())
	if err != nil {
		t.Fatalf("read model registry: %v", err)
	}
	var reg modelregistry.Registry
	if err := json.Unmarshal(raw, &reg); err != nil {
		t.Fatalf("parse model registry: %v", err)
	}

	codecs := []Codec{
		NewClaudeForTest(),
		NewCodexForTest(),
		NewOpenCodeForTest(),
	}

	for _, codec := range codecs {
		runnerKey := string(codec.Type())
		t.Run(runnerKey, func(t *testing.T) {
			runnerReg, ok := reg.Runners[runnerKey]
			if !ok {
				t.Fatalf("runner %q has codec but no model-registry.json entry", runnerKey)
			}
			registryModels := make(map[string]struct{}, len(runnerReg.Models))
			for _, m := range runnerReg.Models {
				registryModels[m.ID] = struct{}{}
			}

			for _, model := range codec.Capabilities().SupportedModels {
				if strings.HasPrefix(model, ollamaModelPrefix) {
					continue // runtime-discovered, not catalogued in JSON
				}
				if _, ok := registryModels[model]; !ok {
					t.Errorf("codec %q advertises model %q absent from model-registry.json — "+
						"reconcile codec SupportedModels with config/model-registry.json (registry is the source of truth)",
						runnerKey, model)
				}
			}
		})
	}
}
