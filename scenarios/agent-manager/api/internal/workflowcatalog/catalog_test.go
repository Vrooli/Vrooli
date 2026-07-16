package workflowcatalog

import (
	"encoding/json"
	"strings"
	"testing"

	"agent-manager/internal/domain"
)

func validDefinition() domain.WorkflowDefinition {
	schema := json.RawMessage(`{"type":"object","additionalProperties":false}`)
	return domain.WorkflowDefinition{
		SchemaVersion: domain.WorkflowSchemaVersionV1, Owner: "example", Key: "example/review", Version: "1.2.3",
		InputSchema: schema, OutputSchema: schema, EntryNode: "start",
		Nodes: []domain.WorkflowNode{
			{ID: "start", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "Review {{input}}", Bindings: []domain.WorkflowInputBinding{{Name: "input", Source: domain.WorkflowBindingInput, Limit: 1, MaxBytes: 4096, RenderAs: "json", MissingPolicy: "error"}}}},
			{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
		},
		Edges: []domain.WorkflowEdge{{From: "start", To: "done"}}, Budgets: validBudgets(),
	}
}

func validBudgets() domain.WorkflowBudgets {
	return domain.WorkflowBudgets{WallTimeSeconds: 60, MaxTurns: 4, MaxTokens: 1000, MaxCostUSD: 1, MaxNodeAttempts: 3, MaxChildren: 2, MaxConcurrency: 2, MaxRecursion: 2, MaxRetries: 2, MaxWaitSeconds: 30}
}

func TestWorkflowBudgetSafetyCeilings(t *testing.T) {
	definition := validDefinition()
	definition.Budgets.MaxConcurrency = MaxConcurrency + 1
	result, err := Validate(definition, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) == 0 {
		t.Fatal("workflow above operator concurrency ceiling unexpectedly valid")
	}
	if !hasCode(result.Diagnostics, "budget_ceiling") {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
}

func TestValidateCanonicalDigestIsStable(t *testing.T) {
	d := validDefinition()
	first, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", first.Diagnostics)
	}
	if first.Digest == "" || first.Digest != second.Digest || string(first.Canonical) != string(second.Canonical) {
		t.Fatalf("canonical result is not stable: %#v %#v", first, second)
	}
}

func TestParseRejectsUnknownRuntimeAndCallbackFields(t *testing.T) {
	data, _ := json.Marshal(validDefinition())
	for _, field := range []string{`"callback":"https://example.test"`, `"runId":"mutable"`, `"exec":"rm -rf /"`} {
		mutated := strings.Replace(string(data), `"description"`, field+`,"description"`, 1)
		if mutated == string(data) {
			mutated = strings.Replace(string(data), `"schemaVersion"`, field+`,"schemaVersion"`, 1)
		}
		if _, err := Parse([]byte(mutated), nil); err == nil {
			t.Fatalf("unknown field %s was accepted", field)
		}
	}
}

func TestValidateRequiresExplicitBoundedCycleEdges(t *testing.T) {
	d := validDefinition()
	d.Nodes = append(d.Nodes[:1], domain.WorkflowNode{ID: "again", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Expression: "counter < 2"}}, d.Nodes[1])
	d.Edges = []domain.WorkflowEdge{{From: "start", To: "again"}, {From: "again", To: "start"}, {From: "again", To: "done"}}
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(result.Diagnostics, "cycle_budget") {
		t.Fatalf("unbounded cycle accepted: %+v", result.Diagnostics)
	}
	d.Edges[0].MaxTraversals, d.Edges[1].MaxTraversals = 2, 2
	result, err = Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if hasCode(result.Diagnostics, "cycle_budget") {
		t.Fatalf("bounded cycle rejected: %+v", result.Diagnostics)
	}
}

func TestValidateContinuationRequiresExplicitAncestorRun(t *testing.T) {
	d := validDefinition()
	d.Nodes = []domain.WorkflowNode{{ID: "start", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "first"}}, {ID: "followup", Kind: domain.WorkflowNodeContinue, Continue: &domain.WorkflowContinueNode{ConversationFromNode: "missing", PromptTemplate: "second"}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	d.Edges = []domain.WorkflowEdge{{From: "start", To: "followup"}, {From: "followup", To: "done"}}
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(result.Diagnostics, "continuation_source") {
		t.Fatalf("implicit continuation accepted: %+v", result.Diagnostics)
	}
	d.Nodes[1].Continue.ConversationFromNode = "start"
	result, err = Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if hasCode(result.Diagnostics, "continuation_source") || hasCode(result.Diagnostics, "continuation_order") {
		t.Fatalf("explicit continuation rejected: %+v", result.Diagnostics)
	}
}

func TestValidateMixedProfilesAndForwardContinuation(t *testing.T) {
	d := validDefinition()
	d.Nodes = []domain.WorkflowNode{
		{ID: "plan", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{ProfileKey: "planner", PromptTemplate: "plan"}},
		{ID: "implement", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{ProfileKey: "builder", PromptTemplate: "implement"}},
		{ID: "revise", Kind: domain.WorkflowNodeContinue, Continue: &domain.WorkflowContinueNode{ConversationFromNode: "plan", PromptTemplate: "revise using implementation"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	d.EntryNode = "plan"
	d.Edges = []domain.WorkflowEdge{{From: "plan", To: "implement"}, {From: "implement", To: "revise"}, {From: "revise", To: "done"}}
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("mixed-profile workflow with explicit forward continuation rejected: %+v", result.Diagnostics)
	}
}

func TestValidateRejectsContinuationSourceThatDoesNotDominate(t *testing.T) {
	d := validDefinition()
	d.Nodes = []domain.WorkflowNode{
		{ID: "choose", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Expression: "true"}},
		{ID: "optional", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{ProfileKey: "planner", PromptTemplate: "optional"}},
		{ID: "revise", Kind: domain.WorkflowNodeContinue, Continue: &domain.WorkflowContinueNode{ConversationFromNode: "optional", PromptTemplate: "revise"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	d.EntryNode = "choose"
	d.Edges = []domain.WorkflowEdge{{From: "choose", To: "optional", Condition: "true"}, {From: "choose", To: "revise", Condition: "false"}, {From: "optional", To: "revise"}, {From: "revise", To: "done"}}
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(result.Diagnostics, "continuation_order") {
		t.Fatalf("non-dominating continuation source accepted: %+v", result.Diagnostics)
	}
}

func TestValidateRejectsTranscriptBindingAndUnboundedProjection(t *testing.T) {
	d := validDefinition()
	d.Nodes[0].Run.Bindings[0].Source = domain.WorkflowBindingSource("transcript")
	d.Nodes[0].Run.Bindings[0].MaxBytes = 0
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(result.Diagnostics, "binding_source") || !hasCode(result.Diagnostics, "binding_size") {
		t.Fatalf("unsafe binding accepted: %+v", result.Diagnostics)
	}
}

func hasCode(diagnostics []domain.WorkflowDiagnostic, code string) bool {
	for _, d := range diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}
