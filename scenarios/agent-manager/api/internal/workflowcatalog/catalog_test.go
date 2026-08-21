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
			{ID: "start", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "Review {{.input}}", Bindings: []domain.WorkflowInputBinding{{Name: "input", Source: domain.WorkflowBindingInput, Limit: 1, MaxBytes: 4096, RenderAs: "json", MissingPolicy: "error"}}}},
			{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
		},
		Edges: []domain.WorkflowEdge{{From: "start", To: "done"}}, Budgets: validBudgets(),
	}
}

func validBudgets() domain.WorkflowBudgets {
	return domain.WorkflowBudgets{WallTimeSeconds: 60, MaxTurns: 4, MaxTokens: 1000, MaxChargeMicroUSD: 1, MaxNodeAttempts: 3, MaxChildren: 2, MaxConcurrency: 2, MaxRecursion: 2, MaxRetries: 2, MaxWaitSeconds: 30}
}

// singleRunSugar is the shorthand form: one run node, no entryNode, no edges,
// no end node.
func singleRunSugar() domain.WorkflowDefinition {
	schema := json.RawMessage(`{"type":"object","additionalProperties":false}`)
	out := json.RawMessage(`{"type":"object","properties":{"result":{"type":"object","additionalProperties":true}},"required":["result"],"additionalProperties":false}`)
	return domain.WorkflowDefinition{
		SchemaVersion: domain.WorkflowSchemaVersionV1, Owner: "example", Key: "example/single", Version: "1.0.0",
		InputSchema: schema, OutputSchema: out,
		Nodes: []domain.WorkflowNode{
			{ID: "work", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "Do {{.input}}", Bindings: []domain.WorkflowInputBinding{{Name: "input", Source: domain.WorkflowBindingInput, Limit: 1, MaxBytes: 4096, RenderAs: "json", MissingPolicy: "error"}}}},
		},
		Budgets: validBudgets(),
	}
}

// singleRunExplicit is the fully-spelled equivalent of singleRunSugar: the same
// run node plus the entryNode, implied end, and edge the sugar synthesizes.
func singleRunExplicit() domain.WorkflowDefinition {
	d := singleRunSugar()
	d.EntryNode = "work"
	d.Nodes = append(d.Nodes, domain.WorkflowNode{
		ID: "end", Kind: domain.WorkflowNodeEnd,
		End: &domain.WorkflowEndNode{Status: "succeeded", Bindings: []domain.WorkflowInputBinding{{
			Name: "result", Source: domain.WorkflowBindingStructured, Selector: "$.value", Order: "desc", Limit: 1, MaxBytes: 16384, RenderAs: "json", MissingPolicy: "error",
		}}},
	})
	d.Edges = []domain.WorkflowEdge{{From: "work", To: "end"}}
	return d
}

func TestSingleNodeSugarCanonicalizesToExplicitDigest(t *testing.T) {
	sugar, err := Validate(singleRunSugar(), nil)
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := Validate(singleRunExplicit(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if domain.HasBlockingDiagnostic(sugar.Diagnostics) {
		t.Fatalf("sugar diagnostics: %+v", sugar.Diagnostics)
	}
	if domain.HasBlockingDiagnostic(explicit.Diagnostics) {
		t.Fatalf("explicit diagnostics: %+v", explicit.Diagnostics)
	}
	if sugar.Digest == "" {
		t.Fatal("sugar earned no digest")
	}
	if sugar.Digest != explicit.Digest {
		t.Fatalf("sugar and explicit digests differ:\n sugar=%s\n  expl=%s\n sugarJSON=%s\n explJSON=%s", sugar.Digest, explicit.Digest, sugar.Canonical, explicit.Canonical)
	}
}

func TestSingleNodeSugarRoundTripsThroughParse(t *testing.T) {
	data, err := json.Marshal(singleRunSugar())
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(data, nil)
	if err != nil {
		t.Fatalf("parse sugar: %v", err)
	}
	if domain.HasBlockingDiagnostic(parsed.Diagnostics) {
		t.Fatalf("parse diagnostics: %+v", parsed.Diagnostics)
	}
	if parsed.Definition.EntryNode != "work" || len(parsed.Definition.Nodes) != 2 || len(parsed.Definition.Edges) != 1 {
		t.Fatalf("sugar did not expand: entry=%q nodes=%d edges=%d", parsed.Definition.EntryNode, len(parsed.Definition.Nodes), len(parsed.Definition.Edges))
	}
	if parsed.Definition.Nodes[1].Kind != domain.WorkflowNodeEnd || parsed.Definition.Nodes[1].End.Status != "succeeded" {
		t.Fatalf("implied end missing: %+v", parsed.Definition.Nodes[1])
	}
}

func TestValidateAcceptsPromptRefAlternative(t *testing.T) {
	d := validDefinition()
	d.Nodes[0].Run.PromptTemplate = ""
	d.Nodes[0].Run.PromptRef = &domain.WorkflowPromptRef{SkillID: "example-skill"}
	d.Nodes[0].Run.Bindings = nil
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if domain.HasBlockingDiagnostic(result.Diagnostics) {
		t.Fatalf("promptRef alternative rejected: %+v", result.Diagnostics)
	}
}

func TestValidateTriggerPolicy(t *testing.T) {
	tests := []struct {
		name   string
		policy domain.WorkflowTriggerPolicy
		want   string
	}{
		{name: "defaults", policy: domain.WorkflowTriggerPolicy{}},
		{name: "allow depth", policy: domain.WorkflowTriggerPolicy{Initiators: []domain.WorkflowInitiator{domain.WorkflowInitiatorAgent}, SelfTrigger: domain.WorkflowSelfTriggerPolicy{Mode: domain.WorkflowSelfTriggerAllow, MaxDepth: 2}}},
		{name: "unknown initiator", policy: domain.WorkflowTriggerPolicy{Initiators: []domain.WorkflowInitiator{"robot"}}, want: "trigger_initiator"},
		{name: "deny cannot specify depth", policy: domain.WorkflowTriggerPolicy{SelfTrigger: domain.WorkflowSelfTriggerPolicy{Mode: domain.WorkflowSelfTriggerDeny, MaxDepth: 1}}, want: "trigger_self"},
		{name: "allow requires positive depth", policy: domain.WorkflowTriggerPolicy{SelfTrigger: domain.WorkflowSelfTriggerPolicy{Mode: domain.WorkflowSelfTriggerAllow}}, want: "trigger_self"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := validDefinition()
			d.Trigger = tt.policy
			result, err := Validate(d, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == "" && domain.HasBlockingDiagnostic(result.Diagnostics) {
				t.Fatalf("valid trigger rejected: %+v", result.Diagnostics)
			}
			if tt.want != "" && !hasCode(result.Diagnostics, tt.want) {
				t.Fatalf("missing %s diagnostic: %+v", tt.want, result.Diagnostics)
			}
		})
	}
}

func TestValidateScopePathTemplate(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		want  string
	}{
		{name: "declared binding", scope: "scenarios/{{.input.scenario}}"},
		{name: "unknown binding", scope: "scenarios/{{.missing}}", want: "scope_path_unbound"},
		{name: "malformed template", scope: "scenarios/{{.input", want: "scope_path_template"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := validDefinition()
			d.Nodes[0].Run.ScopePathTemplate = tt.scope
			result, err := Validate(d, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == "" && domain.HasBlockingDiagnostic(result.Diagnostics) {
				t.Fatalf("valid scope template rejected: %+v", result.Diagnostics)
			}
			if tt.want != "" && !hasCode(result.Diagnostics, tt.want) {
				t.Fatalf("missing %s diagnostic: %+v", tt.want, result.Diagnostics)
			}
		})
	}
}

func TestValidateRejectsBothPromptTemplateAndRef(t *testing.T) {
	d := validDefinition()
	d.Nodes[0].Run.PromptRef = &domain.WorkflowPromptRef{SkillID: "example-skill"}
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(result.Diagnostics, "prompt") {
		t.Fatalf("expected mutual-exclusion diagnostic: %+v", result.Diagnostics)
	}
}

func TestValidateWarnsForInlinePromptWithoutBlockingReconcile(t *testing.T) {
	result, err := Validate(validDefinition(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if domain.HasBlockingDiagnostic(result.Diagnostics) || result.Digest == "" {
		t.Fatalf("inline prompt must remain reconcilable: %+v", result.Diagnostics)
	}
	if !hasCode(result.Diagnostics, "inline_prompt") {
		t.Fatalf("expected inline prompt maturity warning: %+v", result.Diagnostics)
	}

	d := validDefinition()
	d.Nodes[0].Run.PromptTemplate = "resolved prompt"
	d.Nodes[0].Run.PromptProvenance = &domain.WorkflowPromptSource{SkillID: "workflow-skill", ContentHash: "sha256:pin"}
	result, err = Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if hasCode(result.Diagnostics, "inline_prompt") {
		t.Fatalf("resolved promptRef was treated as inline: %+v", result.Diagnostics)
	}
}

func TestValidateRejectsNeitherPrompt(t *testing.T) {
	d := validDefinition()
	d.Nodes[0].Run.PromptTemplate = ""
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(result.Diagnostics, "prompt") {
		t.Fatalf("expected required-prompt diagnostic: %+v", result.Diagnostics)
	}
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
	if domain.HasBlockingDiagnostic(first.Diagnostics) {
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
	d.Nodes = append(d.Nodes[:1], domain.WorkflowNode{ID: "again", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{}}, d.Nodes[1])
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
	if domain.HasBlockingDiagnostic(result.Diagnostics) {
		t.Fatalf("mixed-profile workflow with explicit forward continuation rejected: %+v", result.Diagnostics)
	}
}

func TestValidateRejectsContinuationSourceThatDoesNotDominate(t *testing.T) {
	d := validDefinition()
	d.Nodes = []domain.WorkflowNode{
		{ID: "choose", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{}},
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

func TestValidateAcceptsDistinctAuthoredTerminalOutcomes(t *testing.T) {
	for _, status := range []string{"succeeded", "blocked", "abstained", "budget_exhausted", "failed"} {
		t.Run(status, func(t *testing.T) {
			d := validDefinition()
			d.Nodes[1].End.Status = status
			result, err := Validate(d, nil)
			if err != nil {
				t.Fatal(err)
			}
			if hasCode(result.Diagnostics, "end") {
				t.Fatalf("terminal outcome %q rejected: %+v", status, result.Diagnostics)
			}
		})
	}
}

func TestValidateRejectsInvalidAgentNodeLimits(t *testing.T) {
	d := validDefinition()
	d.Nodes[0].Run.MaxTurns = -1
	d.Nodes[0].Run.TimeoutSeconds = MaxWallTimeSeconds + 1
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(result.Diagnostics, "node_limit") {
		t.Fatalf("invalid node limits accepted: %+v", result.Diagnostics)
	}
}

func TestValidateWaitTimeoutContracts(t *testing.T) {
	d := validDefinition()
	d.Nodes = []domain.WorkflowNode{
		{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", TimeoutSeconds: 0, OnTimeout: "missing"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "blocked"}},
	}
	d.EntryNode = "approval"
	d.Edges = []domain.WorkflowEdge{{From: "approval", To: "done"}}
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(result.Diagnostics, "wait_timeout") || !hasCode(result.Diagnostics, "wait_timeout_target") {
		t.Fatalf("invalid wait timeout contract accepted: %+v", result.Diagnostics)
	}
	d.Nodes[0].Wait.TimeoutSeconds = 10
	d.Nodes[0].Wait.OnTimeout = "done"
	result, err = Validate(d, nil)
	if err != nil || domain.HasBlockingDiagnostic(result.Diagnostics) {
		t.Fatalf("bounded timeout route rejected: result=%+v err=%v", result, err)
	}
}

func TestValidateBindingPresentationContracts(t *testing.T) {
	d := validDefinition()
	b := &d.Nodes[0].Run.Bindings[0]
	b.RenderAs, b.WrapTag, b.Lang, b.Overflow = "xml", "not valid", "go", "clip"
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"binding_wrap_tag", "binding_lang", "binding_overflow"} {
		if !hasCode(result.Diagnostics, code) {
			t.Fatalf("missing %s in %+v", code, result.Diagnostics)
		}
	}
	b.WrapTag, b.Lang, b.Overflow = "context", "", "truncate"
	result, err = Validate(d, nil)
	if err != nil || domain.HasBlockingDiagnostic(result.Diagnostics) {
		t.Fatalf("valid xml binding rejected: result=%+v err=%v", result, err)
	}
}

func diagnosticFor(diagnostics []domain.WorkflowDiagnostic, code string) (domain.WorkflowDiagnostic, bool) {
	for _, d := range diagnostics {
		if d.Code == code {
			return d, true
		}
	}
	return domain.WorkflowDiagnostic{}, false
}

func TestValidateRejectsMalformedEdgeCondition(t *testing.T) {
	d := validDefinition()
	d.Nodes = []domain.WorkflowNode{
		{ID: "gate", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	d.EntryNode = "gate"
	d.Edges = []domain.WorkflowEdge{{From: "gate", To: "done", Condition: "iteration <"}}
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	diag, ok := diagnosticFor(result.Diagnostics, "edge_condition")
	if !ok {
		t.Fatalf("syntactically broken edge condition accepted: %+v", result.Diagnostics)
	}
	if diag.Path != "edges[0].condition" {
		t.Fatalf("edge_condition diagnostic missing edge path: %q", diag.Path)
	}
	if result.Digest != "" {
		t.Fatal("a definition with a broken edge condition must not earn a digest")
	}
}

func TestValidateRejectsNonBoolEdgeCondition(t *testing.T) {
	d := validDefinition()
	d.Nodes = []domain.WorkflowNode{
		{ID: "gate", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	d.EntryNode = "gate"
	d.Edges = []domain.WorkflowEdge{{From: "gate", To: "done", Condition: "iteration + 1"}}
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(result.Diagnostics, "edge_condition") {
		t.Fatalf("non-bool edge condition accepted: %+v", result.Diagnostics)
	}
}

func TestValidateRejectsUnboundPromptPlaceholder(t *testing.T) {
	d := validDefinition()
	d.Nodes[0].Run.PromptTemplate = "Review {{.input}} against {{.missing}}"
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	diag, ok := diagnosticFor(result.Diagnostics, "prompt_unbound")
	if !ok {
		t.Fatalf("unbound placeholder accepted: %+v", result.Diagnostics)
	}
	if !diag.IsError() {
		t.Fatal("an unbound placeholder must be a blocking error")
	}
	if result.Digest != "" {
		t.Fatal("a definition with an unbound placeholder must not earn a digest")
	}
}

func TestValidateWarnsOnUnusedBinding(t *testing.T) {
	d := validDefinition()
	d.Nodes[0].Run.Bindings = append(d.Nodes[0].Run.Bindings, domain.WorkflowInputBinding{Name: "extra", Source: domain.WorkflowBindingInput, Selector: "$.extra", Limit: 1, MaxBytes: 4096, RenderAs: "json", MissingPolicy: "omit"})
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	diag, ok := diagnosticFor(result.Diagnostics, "prompt_unused_binding")
	if !ok {
		t.Fatalf("unused binding did not warn: %+v", result.Diagnostics)
	}
	if diag.IsError() {
		t.Fatal("an unused binding must be a non-blocking warning")
	}
	// A warning-only definition still registers: the digest is computed.
	if result.Digest == "" {
		t.Fatalf("a warning-only definition must still earn a digest: %+v", result.Diagnostics)
	}
}

func TestValidateRejectsMalformedPromptTemplate(t *testing.T) {
	d := validDefinition()
	// {{input}} with no dot is a call to an undefined function, exactly the
	// class of latent error the runtime renderer would hit.
	d.Nodes[0].Run.PromptTemplate = "Review {{input}}"
	result, err := Validate(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(result.Diagnostics, "prompt_template") {
		t.Fatalf("unparseable prompt template accepted: %+v", result.Diagnostics)
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

func TestValidateChargeBudgetRejectsHistoricalMicroUSDUnitError(t *testing.T) {
	var diagnostics []domain.WorkflowDiagnostic
	validateChargeBudget(domain.WorkflowBudgets{MaxTokens: 7_200_000, MaxChargeMicroUSD: 30}, func(code, path, message string) {
		diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: code, Path: path, Message: message})
	})
	if !hasCode(diagnostics, "charge_budget_unit") {
		t.Fatalf("implausible charge budget accepted: %+v", diagnostics)
	}
}

func TestValidateChargeBudgetAcceptsMeasurementScale(t *testing.T) {
	var diagnostics []domain.WorkflowDiagnostic
	validateChargeBudget(domain.WorkflowBudgets{MaxTokens: 18_000_000, MaxChargeMicroUSD: 18_000_000}, func(code, path, message string) {
		diagnostics = append(diagnostics, domain.WorkflowDiagnostic{Code: code, Path: path, Message: message})
	})
	if len(diagnostics) != 0 {
		t.Fatalf("measurement-scale charge budget rejected: %+v", diagnostics)
	}
}
