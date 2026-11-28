package workflowcatalog

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"github.com/vrooli/vrooli/packages/proto/contracts"
)

func TestValidateRetiredExperimentsAndParallelDiagnostics(t *testing.T) {
	t.Run("armed prompt is refused as retired", func(t *testing.T) {
		d := validDefinition()
		d.Nodes[0].Run.PromptTemplate = ""
		d.Nodes[0].Run.PromptRef = &domain.WorkflowPromptRef{ExperimentID: "experiment", SkillID: "skill"}
		result, err := Validate(d, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !hasDiagnostic(result.Diagnostics, "skill_experiments_retired") {
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

func TestWorkflowUntilUsesSharedCompletionConditionByteLimit(t *testing.T) {
	d := validDefinition()
	d.Nodes[0].Run.Until = strings.Repeat("x", contracts.AgentManagerMaxGoalCompletionBytes)
	result, err := Validate(d, nil)
	if err != nil || hasDiagnostic(result.Diagnostics, "until_length") {
		t.Fatalf("until at the shared limit rejected: %+v err=%v", result.Diagnostics, err)
	}

	d.Nodes[0].Run.Until += "x"
	result, err = Validate(d, nil)
	if err != nil || !hasDiagnostic(result.Diagnostics, "until_length") {
		t.Fatalf("until above the shared limit should report until_length: %+v err=%v", result.Diagnostics, err)
	}

	encoded, err := json.Marshal(validDefinitionWithUntil(strings.Repeat("x", contracts.AgentManagerMaxGoalCompletionBytes)))
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) > MaxDefinitionBytes {
		t.Fatalf("aggregate workflow cap %d does not accommodate max until (%d-byte workflow)", MaxDefinitionBytes, len(encoded))
	}
	if _, err := Parse(encoded, nil); err != nil {
		t.Fatalf("workflow parser rejected max-size until: %v", err)
	}
}

func TestContinuationValidationReusesSourceAnalysisWithoutLosingPerConsumerChecks(t *testing.T) {
	d := validDefinition()
	d.EntryNode = "choose"
	d.Nodes = []domain.WorkflowNode{
		{ID: "choose", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{}},
		{ID: "source", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "source"}},
		{ID: "first", Kind: domain.WorkflowNodeContinue, Continue: &domain.WorkflowContinueNode{ConversationFromNode: "source", PromptTemplate: "first"}},
		{ID: "second", Kind: domain.WorkflowNodeContinue, Continue: &domain.WorkflowContinueNode{ConversationFromNode: "source", PromptTemplate: "second"}},
		{ID: "bypassed", Kind: domain.WorkflowNodeContinue, Continue: &domain.WorkflowContinueNode{ConversationFromNode: "source", PromptTemplate: "bypassed"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	d.Edges = []domain.WorkflowEdge{
		{From: "choose", To: "source", Condition: "true"},
		{From: "choose", To: "bypassed", Condition: "false"},
		{From: "source", To: "first"},
		{From: "first", To: "second"},
		{From: "second", To: "bypassed"},
		{From: "bypassed", To: "done"},
	}

	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasDiagnosticPath(result.Diagnostics, "continuation_order", "nodes.bypassed.continue.conversationFromNode") {
		t.Fatalf("continuation reachable without its declared source was accepted: %+v", result.Diagnostics)
	}
	if hasDiagnosticPath(result.Diagnostics, "continuation_order", "nodes.first.continue.conversationFromNode") ||
		hasDiagnosticPath(result.Diagnostics, "continuation_order", "nodes.second.continue.conversationFromNode") {
		t.Fatalf("valid continuations sharing one source were rejected: %+v", result.Diagnostics)
	}
}

func hasDiagnosticPath(diagnostics []domain.WorkflowDiagnostic, code, path string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code && diagnostic.Path == path {
			return true
		}
	}
	return false
}

func validDefinitionWithUntil(until string) domain.WorkflowDefinition {
	d := validDefinition()
	d.Nodes[0].Run.Until = until
	return d
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
