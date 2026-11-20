package runannotations

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const minimalPack = `{
  "pack": {
    "name": "test",
    "version": "1",
    "runtime": {"mode": "passive", "default_role": "decide.accurate"},
    "labels": [
      {
        "id": "run.type",
        "group": "run",
        "description": "kind",
        "scope": "run",
        "method": "deterministic",
        "rule": "workload_kind == 'workflow_node' ? 'workflow' : workload_kind",
        "evidence": [{"source": "workload", "field": "kind"}],
        "sampling": {"rate": 1.0, "stratum": "candidate"},
        "gate": {"precision": 0.9, "recall": 0.9, "min_gold": 10},
        "search": {"facet": "label:run.type", "min_probability": 0.6},
        "friction": {"enabled": false, "cost_unit": "run", "importance": "low"},
        "compatibility": {"change": "breaking", "compatible_with": []}
      }
    ]
  }
}`

func writePack(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pack.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func packPathForTest(t *testing.T, labels string) string {
	t.Helper()
	return writePack(t, packWithLabels(labels))
}

func TestDefaultPackLoadsAndCompiles(t *testing.T) {
	path := filepath.Join("..", "..", "..", ".vrooli", "annotations", "default.json")
	pack, err := LoadCompiled(path)
	if err != nil {
		t.Fatalf("load default pack: %v", err)
	}
	if pack.Pack.Name != "default" || pack.Pack.Version == "" {
		t.Fatalf("identity = %s %s", pack.Pack.Name, pack.Pack.Version)
	}
	if pack.Pack.Runtime.ModelAffinity == "" {
		t.Fatal("default pack declares no scheduler model affinity")
	}
	if got := len(pack.Pack.Labels); got != 12 {
		t.Fatalf("labels = %d, want 12", got)
	}
	seen := map[string]bool{}
	for _, label := range pack.Pack.Labels {
		hash := pack.LabelHashes[label.ID]
		if len(hash) != 64 {
			t.Fatalf("%s hash = %q", label.ID, hash)
		}
		if seen[hash] {
			t.Fatalf("duplicate hash %s", hash)
		}
		seen[hash] = true
		if label.Gate.MinGold < 1 {
			t.Fatalf("%s has no gate", label.ID)
		}
		if strings.TrimSpace(label.Search.Facet) == "" {
			t.Fatalf("%s has no search facet", label.ID)
		}
	}
	if len(pack.PackHash) != 64 {
		t.Fatalf("pack hash = %q", pack.PackHash)
	}
}

func TestLabelDefinitionHashIgnoresReportingFields(t *testing.T) {
	pack, err := ParsePack([]byte(minimalPack))
	if err != nil {
		t.Fatal(err)
	}
	base := pack.Labels[0]
	changed := base
	changed.Description = "different prose"
	changed.Gate.Precision = 0.1
	changed.Search.MinProbability = floatPointer(0.1)
	changed.Friction.Enabled = true
	changed.Friction.CostUnit = "token"
	changed.Sampling.Rate = floatPointer(0.2)
	changed.Compatibility.CompatibleWith = []string{strings.Repeat("a", 64)}
	changed.Optional = true
	first, err := LabelDefinitionHash(base)
	if err != nil {
		t.Fatal(err)
	}
	second, err := LabelDefinitionHash(changed)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("reporting fields changed the hash: %s != %s", first, second)
	}
}

func TestLabelDefinitionHashChangesWithSemantics(t *testing.T) {
	pack, err := ParsePack([]byte(minimalPack))
	if err != nil {
		t.Fatal(err)
	}
	base := pack.Labels[0]
	first, _ := LabelDefinitionHash(base)

	reworded := base
	reworded.Rule = "workload_kind == 'workflow_node' ? 'workflow' : 'adhoc'"
	second, _ := LabelDefinitionHash(reworded)
	if first == second {
		t.Fatal("a changed rule kept the same definition hash")
	}

	// Whitespace normalization must not count as a semantic change.
	spaced := base
	spaced.Rule = "workload_kind == 'workflow_node'   ?   'workflow' : workload_kind"
	third, _ := LabelDefinitionHash(spaced)
	if first != third {
		t.Fatal("cosmetic whitespace changed the definition hash")
	}
}

func TestPackValidationRejectsAmbiguousDefinitions(t *testing.T) {
	cases := map[string]string{
		"duplicate label": `[` + labelJSON("a") + `,` + labelJSON("a") + `]`,
		"bad cel": `[{
			"id":"a","group":"g","scope":"run","method":"deterministic","rule":"this is not cel",
			"evidence":[],"gate":{"precision":0.5,"recall":0.5,"min_gold":1},
			"search":{"facet":"f"},"compatibility":{"change":"breaking"}}]`,
		"unknown evidence": `[{
			"id":"a","group":"g","scope":"run","method":"deterministic","rule":"true",
			"evidence":[{"source":"transcript","field":"text"}],"gate":{"precision":0.5,"recall":0.5,"min_gold":1},
			"search":{"facet":"f"},"compatibility":{"change":"breaking"}}]`,
		"gate without gold": `[{
			"id":"a","group":"g","scope":"run","method":"deterministic","rule":"true",
			"evidence":[],"gate":{"precision":0.5,"recall":0.5,"min_gold":0},
			"search":{"facet":"f"},"compatibility":{"change":"breaking"}}]`,
		"prior without noul": `[{
			"id":"a","group":"g","scope":"run","method":"decision","prior_rule":"duration_min > 1",
			"question":{"type":"choice","instructions":"x","options":[{"label":"a"},{"label":"b"}]},
			"evidence":[],"gate":{"precision":0.5,"recall":0.5,"min_gold":1},
			"search":{"facet":"f"},"compatibility":{"change":"breaking"}}]`,
		"choice question one option": `[{
			"id":"a","group":"g","scope":"run","method":"decision",
			"question":{"type":"choice","instructions":"x","options":[{"label":"a"}]},
			"evidence":[],"gate":{"precision":0.5,"recall":0.5,"min_gold":1},
			"search":{"facet":"f"},"compatibility":{"change":"breaking"}}]`,
	}
	for name, labels := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParsePack([]byte(packWithLabels(labels))); err == nil {
				t.Fatalf("pack was accepted: %s", labels)
			}
		})
	}
}

func packWithLabels(labels string) string {
	return `{"pack":{"name":"test","version":"1","labels":` + labels + `}}`
}

func TestParsePackAcceptsBareObject(t *testing.T) {
	bare := `{"name":"t","version":"1","labels":[` + labelJSON("a") + `]}`
	if _, err := ParsePack([]byte(bare)); err != nil {
		t.Fatalf("bare pack: %v", err)
	}
}

func TestCompileRejectsInvalidCompatibilityHash(t *testing.T) {
	body := strings.Replace(minimalPack, `"compatible_with": []`, `"compatible_with": ["not-a-hash"]`, 1)
	if _, err := ParsePack([]byte(body)); err == nil {
		t.Fatal("invalid compatibility hash was accepted")
	}
}

func labelJSON(id string) string {
	return `{"id":"` + id + `","group":"g","scope":"run","method":"deterministic","rule":"true","evidence":[],` +
		`"gate":{"precision":0.5,"recall":0.5,"min_gold":1},"search":{"facet":"f"},"compatibility":{"change":"breaking"}}`
}

func floatPointer(value float64) *float64 { return &value }

func TestGateMeasurementMustNameItsDefinitionRevision(t *testing.T) {
	plain, err := ParsePack([]byte(minimalPack))
	if err != nil {
		t.Fatal(err)
	}
	hash, err := LabelDefinitionHash(plain.Labels[0])
	if err != nil {
		t.Fatal(err)
	}
	withMeasurement := func(measuredHash string) string {
		measured := fmt.Sprintf(`"gate": {"precision": 0.9, "recall": 0.9, "min_gold": 10, "measured": {"precision": 0.95, "recall": 0.95, "gold": 50, "definition_hash": %q}}`, measuredHash)
		return strings.Replace(minimalPack, `"gate": {"precision": 0.9, "recall": 0.9, "min_gold": 10}`, measured, 1)
	}
	for _, tc := range []struct {
		name       string
		measured   string
		wantProven bool
		wantReason string
	}{
		{name: "matching revision", measured: hash, wantProven: true},
		{name: "stale revision", measured: "old-definition", wantReason: "different definition"},
		{name: "unnamed revision", measured: "", wantReason: "different definition"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			compiled, err := ParsePack([]byte(withMeasurement(tc.measured)))
			if err != nil {
				t.Fatal(err)
			}
			pack, err := compiled.Compile()
			if err != nil {
				t.Fatal(err)
			}
			label, ok := pack.Label("run.type")
			if !ok {
				t.Fatal("label missing")
			}
			if got := label.label.Gate.Proven(); got != tc.wantProven {
				t.Fatalf("proven = %v, want %v (state %s)", got, tc.wantProven, label.label.Gate.State())
			}
			if tc.wantReason != "" {
				status := gateStatus(label.label.Gate)
				if status.State != "unmeasured" || !strings.Contains(status.Reason, tc.wantReason) {
					t.Fatalf("gate status = %+v, want the stale-measurement reason", status)
				}
			}
		})
	}
}

func TestRequireChangesTheDefinitionAndIsValidated(t *testing.T) {
	pack, err := ParsePack([]byte(minimalPack))
	if err != nil {
		t.Fatal(err)
	}
	base := pack.Labels[0]
	first, _ := LabelDefinitionHash(base)

	guarded := base
	guarded.Require = "agent_text_count > 0"
	second, _ := LabelDefinitionHash(guarded)
	if first == second {
		t.Fatal("a new require predicate kept the same definition hash")
	}
	guarded.Require = "agent_text_count   >   0"
	third, _ := LabelDefinitionHash(guarded)
	if second != third {
		t.Fatal("cosmetic whitespace in require changed the definition hash")
	}
	invalid := []byte(`[{
	  "id":"a","group":"g","scope":"run","method":"decision","require":"this is not cel",
	  "question":{"type":"noul","instructions":"x"},
	  "evidence":[],"gate":{"precision":0.5,"recall":0.5,"min_gold":1},
	  "search":{"facet":"f"},"compatibility":{"change":"breaking"}}]`)
	if _, err := ParsePack(invalid); err == nil {
		t.Fatal("an invalid require expression was accepted")
	}
}
