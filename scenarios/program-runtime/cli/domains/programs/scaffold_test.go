package programs

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildScaffoldRejectsBadInput(t *testing.T) {
	cases := []struct{ name, purpose, binding string }{
		{"NotKebab.Program", "a purpose that is long enough to pass", ""},
		{"noscenariodot", "a purpose that is long enough to pass", ""},
		{"scenario.program", "too short", ""},
		{"scenario.program", "a purpose that is long enough to pass validation here", "bad-binding"},
	}
	for _, tc := range cases {
		if _, err := buildScaffold(tc.name, tc.purpose, tc.binding); err == nil {
			t.Fatalf("buildScaffold(%q,%q,%q) = nil error, want rejection", tc.name, tc.purpose, tc.binding)
		}
	}
}

func TestBuildScaffoldEmitsSchemaShapedContract(t *testing.T) {
	required := []string{"name", "version", "purpose", "inputs", "assumptions", "invariants", "bindings", "outputs", "budget", "fixtures"}

	for _, binding := range []string{"", "vrooli/scenario/status"} {
		res, err := buildScaffold("program-runtime.self-test", "Probe one scenario and report a bounded status signal for the author.", binding)
		if err != nil {
			t.Fatalf("buildScaffold with binding %q: %v", binding, err)
		}
		var contract map[string]any
		if err := json.Unmarshal([]byte(res.ContractJSON), &contract); err != nil {
			t.Fatalf("contract is not valid JSON: %v", err)
		}
		for _, key := range required {
			if _, ok := contract[key]; !ok {
				t.Fatalf("binding %q: contract missing required key %q", binding, key)
			}
		}
		// errors.classes must be non-empty (schema minItems:1) — the bug the
		// validate step caught during authoring.
		classes := contract["outputs"].(map[string]any)["errors"].(map[string]any)["classes"].([]any)
		if len(classes) == 0 {
			t.Fatalf("binding %q: errors.classes is empty; schema requires at least one", binding)
		}
		if !strings.Contains(res.ProgramPy, "program.run(STATES") {
			t.Fatalf("binding %q: program skeleton missing the program.run state machine", binding)
		}
		if binding != "" && len(contract["bindings"].([]any)) != 1 {
			t.Fatalf("binding %q: expected exactly one declared binding", binding)
		}
	}
}

func TestSchemaRelPathDependsOnOwner(t *testing.T) {
	if got := schemaRelPath("program-runtime"); got != "../../schemas/program-contract.schema.json" {
		t.Fatalf("own-scenario schema path = %q", got)
	}
	if got := schemaRelPath("device-control"); got != "../../../program-runtime/schemas/program-contract.schema.json" {
		t.Fatalf("cross-scenario schema path = %q", got)
	}
}
