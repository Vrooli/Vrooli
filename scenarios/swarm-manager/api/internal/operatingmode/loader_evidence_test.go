package operatingmode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/evidence"
)

func TestLoadModeDefinitionCompilesEvidenceRequirements(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(modesDir, string(ModeHolisticLoop), ModeFileName))
	if err != nil {
		t.Fatalf("read fixture mode: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("decode fixture mode: %v", err)
	}
	phaseGraph := document["phase_graph"].(map[string]any)
	phases := phaseGraph["phases"].([]any)
	first := phases[0].(map[string]any)
	first["evidence_requirements"] = []any{map[string]any{
		"subject_kind": "plan", "action": "created", "producer": "plan-manager",
		"min_confidence": "authoritative", "min_count": 2,
		"match_fields": map[string]any{"plan_type": "implementation"},
	}}
	withEvidence, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode mode: %v", err)
	}
	def, err := LoadModeDefinition(withEvidence)
	if err != nil {
		t.Fatalf("LoadModeDefinition: %v", err)
	}
	phase, err := def.PhaseDefinition(Phase(first["id"].(string)))
	if err != nil {
		t.Fatalf("load evidence phase: %v", err)
	}
	if len(phase.EvidenceRequirements) != 1 {
		t.Fatalf("evidence requirements = %+v, want one", phase.EvidenceRequirements)
	}
	requirement := phase.EvidenceRequirements[0]
	if requirement.SubjectKind != "plan" || requirement.Action != "created" || requirement.ProducerID != "plan-manager" || requirement.MinConfidence != evidence.ConfidenceAuthoritative || requirement.MinCount != 2 || requirement.MatchFields["plan_type"] != "implementation" {
		t.Fatalf("compiled evidence requirement = %+v", requirement)
	}

	first["evidence_requirements"] = []any{map[string]any{
		"subject_kind": "plan", "action": "created", "min_confidence": "untrusted",
	}}
	invalid, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode invalid mode: %v", err)
	}
	if _, err := LoadModeDefinition(invalid); err == nil {
		t.Fatal("LoadModeDefinition accepted an invalid evidence confidence")
	}
}
