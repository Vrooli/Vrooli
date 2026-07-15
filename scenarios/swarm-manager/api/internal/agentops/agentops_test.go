package agentops

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"swarm-manager/internal/operatingmode"
)

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

// TestTargetVocabularyMatchesOperatingMode pins the single target vocabulary:
// the contracts package and the operating-mode engine must agree byte-for-byte
// on the three kinds, so a rename in one is a compile/test failure that forces
// the other.
func TestTargetVocabularyMatchesOperatingMode(t *testing.T) {
	want := []operatingmode.TargetKind{
		operatingmode.TargetBacklogItem,
		operatingmode.TargetInitiative,
		operatingmode.TargetPlanExecution,
		operatingmode.TargetScenario,
	}
	if len(want) != len(AllTargetKinds) {
		t.Fatalf("target kind count drift: agentops=%d operatingmode=%d", len(AllTargetKinds), len(want))
	}
	for i := range want {
		if string(want[i]) != string(AllTargetKinds[i]) {
			t.Fatalf("target vocabulary drift at %d: agentops=%q operatingmode=%q", i, AllTargetKinds[i], want[i])
		}
	}
}

// TestSeedOperationContractsValidate proves every enumerated operation contract
// is schema-valid, target-runnable, and semantically consistent.
func TestSeedOperationContractsValidate(t *testing.T) {
	seeds := SeedOperationContracts()
	if len(seeds) != len(AllOperationIDs) {
		t.Fatalf("seed count %d != operation vocabulary %d", len(seeds), len(AllOperationIDs))
	}
	for _, oc := range seeds {
		if err := ValidateOperationContract(mustJSON(t, oc)); err != nil {
			t.Fatalf("operation %q invalid: %v", oc.ID, err)
		}
		if len(CompatibleTargets(oc.TargetRequirements.Capabilities)) == 0 {
			t.Fatalf("operation %q has no compatible target", oc.ID)
		}
	}
}

// TestReviewRoundSharedAcrossBacklogItemAndInitiative pins the shared-contract
// design: a review operation whose requirements are met by both backlog-item
// and initiative is compatible with both.
func TestReviewRoundSharedAcrossBacklogItemAndInitiative(t *testing.T) {
	var review OperationContract
	for _, oc := range SeedOperationContracts() {
		if oc.ID == OpReviewRound {
			review = oc
		}
	}
	compat := CompatibleTargets(review.TargetRequirements.Capabilities)
	has := func(k TargetKind) bool {
		for _, c := range compat {
			if c == k {
				return true
			}
		}
		return false
	}
	if !has(TargetBacklogItem) || !has(TargetInitiative) {
		t.Fatalf("review-round must be shareable across backlog-item and initiative, got %v", compat)
	}
}

// TestOperationTargetCompatibility exercises compatible AND incompatible pairs.
func TestOperationTargetCompatibility(t *testing.T) {
	// initiative-review requires member items + acceptance criteria: initiative
	// yes, backlog-item no.
	initReview := []CapabilityID{CapMemberItems, CapAcceptanceCriteria, CapReviewArtifacts}
	if err := CheckOperationTargetCompatibility(OpInitiativeReview, initReview, TargetInitiative); err != nil {
		t.Fatalf("initiative-review should run on initiative: %v", err)
	}
	err := CheckOperationTargetCompatibility(OpInitiativeReview, initReview, TargetBacklogItem)
	if err == nil || !strings.Contains(err.Error(), "missing capabilities") {
		t.Fatalf("initiative-review must be incompatible with backlog-item, got %v", err)
	}
	// plan-execution cannot run a spec-document operation.
	if err := CheckOperationTargetCompatibility(OpWorkshopRound, []CapabilityID{CapSpecDocument}, TargetPlanExecution); err == nil {
		t.Fatalf("workshop-round must be incompatible with plan-execution")
	}
	// unknown target kind fails closed.
	if err := CheckOperationTargetCompatibility(OpReviewRound, []CapabilityID{CapReviewArtifacts}, TargetKind("galaxy")); err == nil {
		t.Fatalf("unknown target kind must fail closed")
	}
}

// TestTargetCapabilityDescriptorsValidate proves the registry descriptors are
// schema-valid and every capability is in the closed vocabulary.
func TestTargetCapabilityDescriptorsValidate(t *testing.T) {
	for _, d := range TargetCapabilities() {
		if err := ValidateTargetCapabilityDescriptor(mustJSON(t, d)); err != nil {
			t.Fatalf("descriptor %q invalid: %v", d.TargetKind, err)
		}
	}
	// An unknown capability is rejected.
	bad := TargetCapabilityDescriptor{Kind: "agentops-target-capability", TargetKind: TargetInitiative, Provides: []CapabilityID{"provides-teleportation"}}
	if err := ValidateTargetCapabilityDescriptor(mustJSON(t, bad)); err == nil {
		t.Fatalf("unknown capability must be rejected")
	}
}

// --- Binding precedence + the four fail-closed states ---

type stubChecker struct {
	missingRevision bool
	incompatible    bool
}

func (s stubChecker) RevisionExists(mode, revision string) bool { return !s.missingRevision }
func (s stubChecker) ModeCompatible(mode string, op OperationID, target TargetKind) bool {
	return !s.incompatible
}

func binding(layer BindingLayer, ownerKind, ownerID, mode, rev string) OperationBinding {
	b := OperationBinding{Kind: "agentops-operation-binding", Operation: OpReviewRound, Layer: layer, Mode: mode, ModeRevision: rev}
	if layer != LayerSystemDefault {
		b.Owner = &BindingOwner{Kind: ownerKind, ID: ownerID}
	}
	return b
}

func TestBindingPrecedenceDeterministic(t *testing.T) {
	scope := ResolutionScope{InvocationID: "inv-1", ItemRef: "fix/x", InitiativeName: "init-a", Target: TargetInitiative}
	all := []OperationBinding{
		binding(LayerSystemDefault, "", "", "review-default", "r0"),
		binding(LayerInitiativeOverride, "initiative", "init-a", "review-init", "r1"),
		binding(LayerBacklogItemOverride, "backlog-item", "fix/x", "review-item", "r2"),
		binding(LayerAuthorizedInvocation, "invocation", "inv-1", "review-inv", "r3"),
	}
	cases := []struct {
		name       string
		candidates []OperationBinding
		wantMode   string
		wantLayer  BindingLayer
	}{
		{"authorized invocation wins", all, "review-inv", LayerAuthorizedInvocation},
		{"item over initiative", all[:3], "review-item", LayerBacklogItemOverride},
		{"initiative over default", all[:2], "review-init", LayerInitiativeOverride},
		{"default alone", all[:1], "review-default", LayerSystemDefault},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveBinding(OpReviewRound, tc.candidates, scope, stubChecker{})
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if got.Mode != tc.wantMode || got.Layer != tc.wantLayer {
				t.Fatalf("got mode=%q layer=%q, want %q/%q", got.Mode, got.Layer, tc.wantMode, tc.wantLayer)
			}
		})
	}
}

func TestBindingAbsenceIsTypedError(t *testing.T) {
	scope := ResolutionScope{Target: TargetInitiative}
	_, err := ResolveBinding(OpReviewRound, nil, scope, stubChecker{})
	if !errors.Is(err, ErrNoBinding) {
		t.Fatalf("absence must be ErrNoBinding, got %v", err)
	}
}

func TestBindingInvalidOverrideFailsClosedNoFallback(t *testing.T) {
	scope := ResolutionScope{ItemRef: "fix/x", InitiativeName: "init-a", Target: TargetInitiative}
	// A disabled highest-precedence (item) override must NOT fall through to the
	// initiative override below it.
	item := binding(LayerBacklogItemOverride, "backlog-item", "fix/x", "review-item", "r2")
	item.Disabled = true
	candidates := []OperationBinding{
		binding(LayerInitiativeOverride, "initiative", "init-a", "review-init", "r1"),
		item,
	}
	_, err := ResolveBinding(OpReviewRound, candidates, scope, stubChecker{})
	if !errors.Is(err, ErrInvalidOverride) {
		t.Fatalf("disabled winning override must be ErrInvalidOverride (no fallback), got %v", err)
	}
	// Two bindings at the same layer/scope is ambiguous → fail closed.
	dup := []OperationBinding{
		binding(LayerInitiativeOverride, "initiative", "init-a", "a", "r1"),
		binding(LayerInitiativeOverride, "initiative", "init-a", "b", "r2"),
	}
	if _, err := ResolveBinding(OpReviewRound, dup, scope, stubChecker{}); !errors.Is(err, ErrInvalidOverride) {
		t.Fatalf("ambiguous same-layer bindings must fail closed, got %v", err)
	}
}

func TestBindingDeletedRevisionAndIncompatibleMode(t *testing.T) {
	scope := ResolutionScope{InitiativeName: "init-a", Target: TargetInitiative}
	candidates := []OperationBinding{binding(LayerInitiativeOverride, "initiative", "init-a", "review-init", "r1")}
	if _, err := ResolveBinding(OpReviewRound, candidates, scope, stubChecker{missingRevision: true}); !errors.Is(err, ErrDeletedRevision) {
		t.Fatalf("missing revision must be ErrDeletedRevision, got %v", err)
	}
	if _, err := ResolveBinding(OpReviewRound, candidates, scope, stubChecker{incompatible: true}); !errors.Is(err, ErrIncompatibleMode) {
		t.Fatalf("incompatible mode must be ErrIncompatibleMode, got %v", err)
	}
}

func TestBindingDocumentSchema(t *testing.T) {
	// A system-default binding with an owner is invalid.
	bad := OperationBinding{Kind: "agentops-operation-binding", Operation: OpReviewRound, Layer: LayerSystemDefault, Owner: &BindingOwner{Kind: "backlog-item", ID: "x"}, Mode: "m", ModeRevision: "r"}
	if err := ValidateBinding(mustJSON(t, bad)); err == nil {
		t.Fatalf("system-default binding must not carry an owner")
	}
	// A well-formed override binding validates.
	good := binding(LayerBacklogItemOverride, "backlog-item", "fix/x", "review-item", "r2")
	if err := ValidateBinding(mustJSON(t, good)); err != nil {
		t.Fatalf("valid binding rejected: %v", err)
	}
}

// --- Provenance completeness ---

func fullProvenance() ExecutionProvenance {
	return ExecutionProvenance{
		Kind: "agentops-execution-provenance", Operation: OpReviewRound, OperationVersion: "1.0.0",
		Binding: ProvenanceBinding{Layer: LayerInitiativeOverride, OwnerKind: "initiative", OwnerID: "init-a"},
		Mode:    "holistic-loop", ModeRevision: "r1",
		CompiledModeDigest:    "sha256:" + strings.Repeat("a", 64),
		PromptCatalogRevision: "pc-1", PromptCatalogDigest: "sha256:" + strings.Repeat("b", 64),
		Target:            ProvenanceTarget{Kind: TargetInitiative, ID: "init-a"},
		CallerInputDigest: "sha256:" + strings.Repeat("c", 64),
		PolicyRevision:    "pol-1", WorkflowInstanceID: "wf-1",
	}
}

func TestProvenanceCompleteness(t *testing.T) {
	if err := ValidateProvenance(mustJSON(t, fullProvenance())); err != nil {
		t.Fatalf("full provenance rejected: %v", err)
	}
	// Drop each required field in turn → schema rejects it (no partial
	// provenance can authorize a run).
	for _, field := range []string{"operation", "mode", "mode_revision", "compiled_mode_digest", "prompt_catalog_digest", "target", "caller_input_digest", "policy_revision", "workflow_instance_id", "binding"} {
		m := map[string]any{}
		if err := json.Unmarshal(mustJSON(t, fullProvenance()), &m); err != nil {
			t.Fatal(err)
		}
		delete(m, field)
		if err := ValidateProvenance(mustJSON(t, m)); err == nil {
			t.Fatalf("provenance missing %q must be rejected", field)
		}
	}
	// A malformed digest is rejected.
	p := fullProvenance()
	p.CompiledModeDigest = "not-a-digest"
	if err := ValidateProvenance(mustJSON(t, p)); err == nil {
		t.Fatalf("malformed digest must be rejected")
	}
	// system-default binding must be attributed to the system owner.
	p2 := fullProvenance()
	p2.Binding = ProvenanceBinding{Layer: LayerSystemDefault, OwnerKind: "initiative", OwnerID: "x"}
	if err := ValidateProvenance(mustJSON(t, p2)); err == nil {
		t.Fatalf("system-default provenance with non-system owner must be rejected")
	}
}

// --- Transition policy: proves data can never encode arbitrary execution ---

func validPolicy() TransitionPolicy {
	return TransitionPolicy{
		Kind: "agentops-transition-policy", ID: "backlog-default", Version: "1.0.0", DomainKind: "backlog-item",
		Transitions: []PolicyTransition{
			{FromState: "running", OnOutcome: "completed", Action: ActionOpenReview, ToState: "awaiting-decision"},
			{FromState: "awaiting-decision", OnOutcome: "accepted", Action: ActionCompleteItem, ToState: "terminal-complete"},
			{FromState: "running", OnOutcome: "blocked", Action: ActionEscalateNeedsAttention, ToState: "blocked", Params: map[string]any{"reason": "blocked"}},
		},
	}
}

func TestTransitionPolicyValid(t *testing.T) {
	if err := ValidateTransitionPolicy(mustJSON(t, validPolicy())); err != nil {
		t.Fatalf("valid policy rejected: %v", err)
	}
}

func TestTransitionPolicyRejectsUnregisteredAction(t *testing.T) {
	p := validPolicy()
	p.Transitions[0].Action = ActionName("run-shell-script")
	if err := ValidateTransitionPolicy(mustJSON(t, p)); err == nil {
		t.Fatalf("unregistered action must be rejected")
	}
}

// TestTransitionPolicyRejectsUnregisteredOperation proves an operation-specific
// rule can only pin a registered operation-contract id, so a policy can never
// route on an operation the catalog does not declare — the load-time twin of the
// operation-aware EvaluateTransition dispatch.
func TestTransitionPolicyRejectsUnregisteredOperation(t *testing.T) {
	p := validPolicy()
	p.Transitions[0].Operation = OperationID("not-an-operation")
	if err := ValidateTransitionPolicy(mustJSON(t, p)); err == nil {
		t.Fatalf("policy naming an unregistered operation must be rejected")
	}
	// A registered operation on the same rule is accepted.
	p.Transitions[0].Operation = OpWorkshopFinalize
	if err := ValidateTransitionPolicy(mustJSON(t, p)); err != nil {
		t.Fatalf("policy naming a registered operation must be accepted: %v", err)
	}
}

// TestTransitionPolicyCannotEncodeArbitraryExecution proves the security
// property: there is NO schema-accepted way to name a Go function, shell
// command, service, or file path, and params cannot carry a nested structure.
func TestTransitionPolicyCannotEncodeArbitraryExecution(t *testing.T) {
	// A free-form "command" / "handler" / "exec" field is rejected by
	// additionalProperties:false at the transition level.
	for _, smuggle := range []string{"command", "handler", "exec", "shell", "run", "path", "go_func"} {
		raw := []byte(`{
		  "kind":"agentops-transition-policy","id":"x","version":"1.0.0","domain_kind":"backlog-item",
		  "transitions":[{"from_state":"running","action":"open-review","to_state":"awaiting-decision","` + smuggle + `":"/bin/sh -c evil"}]
		}`)
		if err := ValidateTransitionPolicy(raw); err == nil {
			t.Fatalf("policy with smuggled %q field must be rejected", smuggle)
		}
	}
	// A nested params value (where a path/command could hide) is rejected.
	raw := []byte(`{
	  "kind":"agentops-transition-policy","id":"x","version":"1.0.0","domain_kind":"backlog-item",
	  "transitions":[{"from_state":"running","action":"open-review","to_state":"awaiting-decision","params":{"nested":{"cmd":"evil"}}}]
	}`)
	if err := ValidateTransitionPolicy(raw); err == nil {
		t.Fatalf("policy with nested params structure must be rejected")
	}
	// Every registered action is a plain name — no action name embeds a path or
	// shell metacharacter.
	for _, a := range AllActionNames {
		if strings.ContainsAny(string(a), "/\\ .:$&|;`") {
			t.Fatalf("action name %q must be a plain kebab identifier", a)
		}
	}
}

// --- Workflow instance ---

func TestWorkflowInstanceValid(t *testing.T) {
	w := WorkflowInstance{
		Kind: "agentops-workflow-instance", SchemaVersion: "1.0.0", InstanceID: "wf-1",
		Domain:   WorkflowDomain{Kind: "initiative", ID: "init-a"},
		Strategy: &WorkflowStrategyRef{Name: "parallel-items"}, State: WorkflowRunning,
		Operations: []OperationExecutionRecord{
			{Operation: OpExecutionRun, ExecutionID: "e1", IdempotencyKey: "k1", ProvenanceDigest: "sha256:" + strings.Repeat("a", 64), State: "completed", Outcome: "completed"},
		},
		LegalActions: []ActionName{ActionOpenReview}, Version: 3,
	}
	if err := ValidateWorkflowInstance(mustJSON(t, w)); err != nil {
		t.Fatalf("valid workflow rejected: %v", err)
	}
}

func TestWorkflowInstanceStrategyOnlyOnInitiative(t *testing.T) {
	w := WorkflowInstance{
		Kind: "agentops-workflow-instance", SchemaVersion: "1.0.0", InstanceID: "wf-1",
		Domain:   WorkflowDomain{Kind: "backlog-item", ID: "fix/x"},
		Strategy: &WorkflowStrategyRef{Name: "parallel-items"}, State: WorkflowOpen, Version: 0,
	}
	if err := ValidateWorkflowInstance(mustJSON(t, w)); err == nil {
		t.Fatalf("member-item strategy on a backlog-item domain must be rejected")
	}
}

func TestWorkflowInstanceRejectsDuplicateIdempotencyKey(t *testing.T) {
	w := WorkflowInstance{
		Kind: "agentops-workflow-instance", SchemaVersion: "1.0.0", InstanceID: "wf-1",
		Domain: WorkflowDomain{Kind: "initiative", ID: "init-a"}, State: WorkflowRunning, Version: 1,
		Operations: []OperationExecutionRecord{
			{Operation: OpExecutionRun, ExecutionID: "e1", IdempotencyKey: "dup", ProvenanceDigest: "sha256:" + strings.Repeat("a", 64), State: "completed"},
			{Operation: OpReviewRound, ExecutionID: "e2", IdempotencyKey: "dup", ProvenanceDigest: "sha256:" + strings.Repeat("b", 64), State: "running"},
		},
	}
	if err := ValidateWorkflowInstance(mustJSON(t, w)); err == nil {
		t.Fatalf("duplicate idempotency key must be rejected")
	}
}

// --- Member-item strategy ---

func TestMemberItemStrategyValid(t *testing.T) {
	if err := ValidateMemberItemStrategy(mustJSON(t, DefaultMemberItemStrategy())); err != nil {
		t.Fatalf("default strategy rejected: %v", err)
	}
	bad := DefaultMemberItemStrategy()
	bad.ItemOperation = "not-an-operation"
	if err := ValidateMemberItemStrategy(mustJSON(t, bad)); err == nil {
		t.Fatalf("strategy with unregistered item operation must be rejected")
	}
}

// TestEverySchemaCompiles guards the embedded schemas: each compiles under the
// Draft 2020 compiler so a malformed schema is a build-time-adjacent failure.
func TestEverySchemaCompiles(t *testing.T) {
	for _, id := range SchemaIDs() {
		if err := ValidateDocument(id, []byte(`{}`)); err == nil {
			// An empty object should fail required-field validation, proving the
			// schema compiled and is enforcing structure (not that it accepts {}).
			t.Fatalf("schema %s accepted an empty object; expected required-field enforcement", id)
		}
	}
}
