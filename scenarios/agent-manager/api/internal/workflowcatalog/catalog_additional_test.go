package workflowcatalog

import (
	"bytes"
	"encoding/json"
	"testing"

	"agent-manager/internal/domain"
)

func TestValidateExperimentEvaluatorAndParallelDiagnostics(t *testing.T) {
	t.Run("armed prompt requires evaluator", func(t *testing.T) {
		d := validDefinition()
		d.Nodes[0].Run.PromptTemplate = ""
		d.Nodes[0].Run.PromptRef = &domain.WorkflowPromptRef{ExperimentID: "experiment", SkillID: "skill"}
		result, err := Validate(d, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !hasDiagnostic(result.Diagnostics, "experiment_evaluator_required") {
			t.Fatalf("diagnostics = %#v", result.Diagnostics)
		}
	})

	t.Run("parallel branch validates members and join", func(t *testing.T) {
		d := validDefinition()
		d.Nodes[0].Branch = &domain.WorkflowBranchNode{Parallel: true}
		result, err := Validate(d, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !hasDiagnostic(result.Diagnostics, "parallel_members") {
			t.Fatalf("diagnostics = %#v", result.Diagnostics)
		}
	})
}

func TestNormalizeSchemaAndParseBoundaryDiagnostics(t *testing.T) {
	diagnostics := []domain.WorkflowDiagnostic{}
	add := func(code, path, message string) {
		diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: code, Path: path, Message: message})
	}
	if got := normalizeSchema(nil, "inputSchema", add); got != nil || !hasDiagnostic(diagnostics, "schema_required") {
		t.Fatalf("empty schema result=%s diagnostics=%#v", got, diagnostics)
	}
	diagnostics = nil
	tooLarge := json.RawMessage(bytes.Repeat([]byte("x"), MaxSchemaBytes+1))
	_ = normalizeSchema(tooLarge, "inputSchema", add)
	if !hasDiagnostic(diagnostics, "schema_size") {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}

	for _, raw := range [][]byte{nil, []byte(`{} {}`)} {
		if _, err := Parse(raw, nil); err == nil {
			t.Fatalf("Parse(%q) succeeded, want boundary error", raw)
		}
	}
	result, err := Parse([]byte(`{}`), nil)
	if err != nil || !hasDiagnostic(result.Diagnostics, "schema_version") {
		t.Fatalf("Parse empty definition result=%#v err=%v", result, err)
	}
}

func hasDiagnostic(diagnostics []domain.WorkflowDiagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
