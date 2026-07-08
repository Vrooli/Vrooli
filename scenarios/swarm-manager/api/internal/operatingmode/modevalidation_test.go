package operatingmode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// baseModeDoc reads the holistic-loop mode.json into a mutable map used as the
// starting point for malformed-input tests.
func baseModeDoc(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(modesDir, string(ModeHolisticLoop), ModeFileName))
	if err != nil {
		t.Fatalf("read base mode: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode base mode: %v", err)
	}
	return doc
}

func marshal(t *testing.T, doc map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal doc: %v", err)
	}
	return raw
}

func firstPhaseNamed(doc map[string]any, id string) map[string]any {
	graph := doc["phase_graph"].(map[string]any)
	for _, p := range graph["phases"].([]any) {
		phase := p.(map[string]any)
		if phase["id"] == id {
			return phase
		}
	}
	return nil
}

// TestSchemaValidationRejects covers structural violations caught by the
// embedded JSON Schema during load.
func TestSchemaValidationRejects(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(doc map[string]any)
	}{
		{"unknown top-level field", func(doc map[string]any) { doc["bogus"] = true }},
		{"missing label", func(doc map[string]any) { delete(doc, "label") }},
		{"wrong kind", func(doc map[string]any) { doc["kind"] = "not-a-mode" }},
		{"non-scenario profile key", func(doc map[string]any) {
			doc["profile"] = map[string]any{"default_profile_key": "other/thing"}
		}},
		{"empty best_for", func(doc map[string]any) { doc["best_for"] = []any{} }},
		{"unknown scope kind", func(doc map[string]any) { doc["scope"] = map[string]any{"kind": "galaxy"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := baseModeDoc(t)
			tc.mutate(doc)
			if _, err := LoadModeDefinition(marshal(t, doc)); err == nil {
				t.Fatalf("LoadModeDefinition succeeded, want schema rejection")
			}
		})
	}
}

// TestSemanticValidationRejects covers cross-reference violations that pass the
// schema but must be rejected by the loader/validator with actionable errors.
func TestSemanticValidationRejects(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(doc map[string]any)
	}{
		{"start phase not registered", func(doc map[string]any) {
			doc["phase_graph"].(map[string]any)["start_phase"] = "ghost"
		}},
		{"guard routes to unregistered phase", func(doc map[string]any) {
			execute := firstPhaseNamed(doc, "execute")
			execute["transitions"] = []any{
				map[string]any{"when": map[string]any{"op": "always"}, "to": []any{"nowhere"}},
			}
		}},
		{"guard references undeclared field", func(doc map[string]any) {
			execute := firstPhaseNamed(doc, "execute")
			execute["transitions"] = []any{
				map[string]any{"when": map[string]any{"op": "eq", "field": "undeclared_field", "value": true}, "to": []any{"review"}},
			}
		}},
		{"when_in_doubt references unknown mode", func(doc map[string]any) {
			doc["when_in_doubt_pick_instead"] = "ghost-mode"
		}},
		{"terminal phase not registered", func(doc map[string]any) {
			doc["phase_graph"].(map[string]any)["terminal"] = []any{"ghost"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := baseModeDoc(t)
			tc.mutate(doc)
			def, err := LoadModeDefinition(marshal(t, doc))
			if err != nil {
				return // rejected at load — acceptable
			}
			if err := ValidateLoadedModes(map[Mode]Definition{def.Mode: def}); err == nil {
				t.Fatalf("ValidateLoadedModes succeeded, want semantic rejection")
			}
		})
	}
}

// TestValidateLoadedModesAcceptsShippedSet confirms the real on-disk mode set
// passes full validation as a group.
func TestValidateLoadedModesAcceptsShippedSet(t *testing.T) {
	defs, err := LoadModesFromDir(modesDir)
	if err != nil {
		t.Fatalf("LoadModesFromDir: %v", err)
	}
	if err := ValidateLoadedModes(defs); err != nil {
		t.Fatalf("ValidateLoadedModes(shipped) = %v", err)
	}
}

// TestValidateLoadedModesRejectsDriftedExampleRun proves a mode-owned
// example-run whose declared expected_path no longer matches the real guards is
// a fatal, actionable load error — the guarantee that makes example-runs a
// trusted test-before-use.
func TestValidateLoadedModesRejectsDriftedExampleRun(t *testing.T) {
	defs, err := LoadModesFromDir(modesDir)
	if err != nil {
		t.Fatalf("LoadModesFromDir: %v", err)
	}
	def := defs[ModeHolisticLoop]
	// A fixture whose expected_path omits the loop-back the guard actually takes.
	def.ExampleRuns = append(def.ExampleRuns, ExampleRun{
		Kind: DocumentKindExampleRun,
		ID:   "drifted",
		Mode: string(ModeHolisticLoop),
		Steps: []ExampleRunStep{
			{Phase: "investigate"},
			{Phase: "plan"},
			{Phase: "execute", Output: map[string]any{"replan_needed": true}},
		},
		ExpectedPath: []string{"investigate", "plan", "execute", "reconcile"},
	})
	defs[ModeHolisticLoop] = def
	if err := ValidateLoadedModes(defs); err == nil {
		t.Fatal("ValidateLoadedModes accepted a drifted example-run, want rejection")
	}
}

// TestValidateLoadedModesRequiresHappyPathPreset proves a phase mode that ships
// example-runs must own the reserved happy-path preset (the simulator default).
func TestValidateLoadedModesRequiresHappyPathPreset(t *testing.T) {
	defs, err := LoadModesFromDir(modesDir)
	if err != nil {
		t.Fatalf("LoadModesFromDir: %v", err)
	}
	def := defs[ModePhasedPlanDrain]
	filtered := def.ExampleRuns[:0:0]
	for _, run := range def.ExampleRuns {
		if run.ID != happyPathPresetID {
			filtered = append(filtered, run)
		}
	}
	def.ExampleRuns = filtered
	defs[ModePhasedPlanDrain] = def
	if err := ValidateLoadedModes(defs); err == nil {
		t.Fatal("ValidateLoadedModes accepted example-runs without a happy-path preset, want rejection")
	}
}
