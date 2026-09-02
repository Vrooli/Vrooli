package transitions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

type declaredWorkflow struct {
	SchemaVersion string `json:"schemaVersion"`
	Owner         string `json:"owner"`
	Key           string `json:"key"`
	Nodes         []struct {
		ID       string      `json:"id"`
		Kind     string      `json:"kind"`
		Run      *nodePrompt `json:"run"`
		Continue *nodePrompt `json:"continue"`
	} `json:"nodes"`
}

type nodePrompt struct {
	PromptTemplate *string `json:"promptTemplate"`
	PromptRef      *struct {
		SkillID string `json:"skillId"`
	} `json:"promptRef"`
}

func TestDeclaredRegistryCoversEveryTargetTransition(t *testing.T) {
	registry, err := LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatalf("LoadDir declared registry: %v", err)
	}
	if got, want := len(registry.Definitions()), 20; got != want {
		t.Fatalf("registered transition count = %d, want %d", got, want)
	}
	for _, key := range []string{
		"capture.classify", "plan.workshop.review", "plan.workshop.reconcile", "plan.author", "plan.repair", "plan.execute",
		"work.review", "review.evidence_request", "goal.discover", "goal.plan", "milestone.review", "scenario.spec_sync",
		"follow_up.dispatch", "goal.close_out",
		"session.meta_orchestration", "session.swarm_operations", "session.workflow_authoring",
	} {
		if _, ok := registry.Get(key); !ok {
			t.Errorf("registry is missing %q", key)
		}
	}
}

func TestPhasedPlanSliceVerifiesTheCanonicalAuthoredProjection(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "prompt-manager", "store", "skills", "packs", "core", "swarm-manager-workflow-phased-plan-slice", "SKILL.md")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read phased-plan slice contract: %v", err)
	}
	body := string(contents)
	for _, marker := range []string{
		"plan-manager plans get <plan_reference> --json",
		"jq -cS",
		"del(.status,.content_hash,.updated_at,.work_posture",
		"delete `status`, `baseline_scope`, and `last_validation` from every phase",
		"freshen_status: baseline_required",
		"exact `capture_argv`",
		"Plan Manager reports `final_dod_required`",
		"status says `execution_complete`",
		"required terminal Definition-of-Done validation result is absent",
		"plan-manager exec complete <plan_execution_id>",
		"Plan Manager execution is already complete",
		"Set `approvalRequired: true`",
		"operator approval wait",
	} {
		if !strings.Contains(body, marker) {
			t.Errorf("phased-plan slice contract is missing signed-rendering marker %q", marker)
		}
	}
}

func TestPhasedPlanSliceReviewRunResultProjectsUnderCanonicalResult(t *testing.T) {
	path := filepath.Join("..", "..", "..", ".vrooli", "agent-manager", "phased-plan-slice-review.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read phased-plan slice review workflow: %v", err)
	}
	var workflow struct {
		OutputSchema map[string]any `json:"outputSchema"`
		Nodes        []struct {
			Run *struct {
				ResultSpec struct {
					Schema map[string]any `json:"schema"`
				} `json:"resultSpec"`
			} `json:"run"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(contents, &workflow); err != nil {
		t.Fatalf("decode phased-plan slice review workflow: %v", err)
	}
	if len(workflow.Nodes) != 1 || workflow.Nodes[0].Run == nil {
		t.Fatalf("review workflow must contain one terminal run node: %#v", workflow.Nodes)
	}
	properties, ok := workflow.OutputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("workflow output properties missing: %#v", workflow.OutputSchema)
	}
	projected, ok := properties["result"].(map[string]any)
	if !ok {
		t.Fatalf("workflow output must project the terminal node value under canonical result: %#v", workflow.OutputSchema)
	}
	if !reflect.DeepEqual(projected, workflow.Nodes[0].Run.ResultSpec.Schema) {
		t.Fatalf("review output property must equal the bare terminal run result schema\nprojected: %#v\nresult: %#v", projected, workflow.Nodes[0].Run.ResultSpec.Schema)
	}
	parentPath := filepath.Join("..", "..", "..", ".vrooli", "agent-manager", "phased-plan-drain.json")
	parentContents, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatalf("read phased-plan parent workflow: %v", err)
	}
	if strings.Contains(string(parentContents), "output.review") {
		t.Fatal("parent workflow still selects the non-canonical review output key")
	}
	for _, marker := range []string{"$.output.result.note", "output.result.accepted"} {
		if !strings.Contains(string(parentContents), marker) {
			t.Errorf("parent workflow is missing canonical child output selector %q", marker)
		}
	}
}

func TestPhasedPlanSliceReviewDoesNotMakeApprovalCircular(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "prompt-manager", "store", "skills", "packs", "core", "swarm-manager-workflow-phased-plan-slice-review", "SKILL.md")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read phased-plan slice review contract: %v", err)
	}
	body := string(contents)
	for _, marker := range []string{
		"stops for operator approval",
		"future workflow obligations",
		"not evidence this pre-approval slice can already possess",
		"parent workflow success and Swarm consumer application happen only after this review accepts",
		"never require those future effects as evidence from the slice",
	} {
		if !strings.Contains(body, marker) {
			t.Errorf("review contract is missing non-circular approval marker %q", marker)
		}
	}
}

func TestIsSessionKindUsesRegistryDeclarations(t *testing.T) {
	registry, err := LoadFS(fstest.MapFS{"registry/session.json": {Data: []byte(`[
{"schemaVersion":"swarm-transition/v1","key":"session.review","subject":"review","kind":"session","session":{"briefRef":"brief/review","skillId":"review-skill","profileKey":"swarm-manager/review"},"inputContract":"session/v1","terminalOutcomes":["complete"],"applyAction":"review_session"}
]`)}}, "registry")
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	if !registry.IsSessionKind("review") {
		t.Fatal("registry-declared session kind was not recognized")
	}
	if registry.IsSessionKind("missing") {
		t.Fatal("undeclared session kind was recognized")
	}
}

func TestValidateRejectsMalformedHumanGate(t *testing.T) {
	definition := Definition{
		SchemaVersion:    SchemaVersion,
		Key:              "work.review",
		Subject:          "backlog-item",
		Kind:             KindDeterministic,
		InputContract:    "review/v1",
		TerminalOutcomes: []string{"accepted"},
		ApplyAction:      "apply_review",
		HumanGates:       []HumanGate{{ID: "operator-review", Decides: "Accept review", DefaultMode: GateModeManual, Threshold: 1.1, MinSample: 10}},
	}
	if err := Validate(definition); err == nil {
		t.Fatal("Validate accepted a human gate with an out-of-range threshold")
	}
}

func TestLoadFSRetainsHumanGateDeclaration(t *testing.T) {
	fsys := fstest.MapFS{"registry/gate.json": {Data: []byte(`[
{"schemaVersion":"swarm-transition/v1","key":"work.review","subject":"backlog-item","kind":"deterministic","inputContract":"review/v1","terminalOutcomes":["accepted"],"applyAction":"apply_review","humanGates":[{"id":"review-acceptance","decides":"Accept the review outcome","defaultMode":"manual","threshold":0.9,"minSample":20}]}
]`)}}
	registry, err := LoadFS(fsys, "registry")
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	definition, ok := registry.Get("work.review")
	if !ok || len(definition.HumanGates) != 1 || definition.HumanGates[0].ID != "review-acceptance" {
		t.Fatalf("human gate declaration = %#v", definition.HumanGates)
	}
}

func TestValidateRejectsHumanWaitWithoutGate(t *testing.T) {
	definition := Definition{
		SchemaVersion: SchemaVersion, Key: "plan.execute", Subject: "plan", Kind: KindWorkflow,
		Workflow: &Locator{Owner: "swarm-manager", Key: "execute"}, InputContract: "plan/v1",
		TerminalOutcomes: []string{"completed"}, ApplyAction: "apply_plan", HumanWait: true,
	}
	if err := Validate(definition); err == nil {
		t.Fatal("Validate accepted a human wait without a declared gate")
	}
}

func TestEffectiveGateModeUsesLiveOverride(t *testing.T) {
	definition := Definition{HumanGates: []HumanGate{{ID: "review", DefaultMode: GateModeManual}}}
	if got, ok := EffectiveGateMode(definition, map[string]string{"review": "auto"}, "review"); !ok || got != GateModeAuto {
		t.Fatalf("effective mode = %q, found=%v", got, ok)
	}
	if got, ok := EffectiveGateMode(definition, map[string]string{"review": "invalid"}, "review"); !ok || got != GateModeManual {
		t.Fatalf("invalid override changed mode = %q, found=%v", got, ok)
	}
}

func TestTransitionCatalogDocumentMatchesRegistryProjection(t *testing.T) {
	registry, err := LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatalf("LoadDir declared registry: %v", err)
	}
	path := filepath.Join("..", "..", "..", "docs", "reference", "transition-catalog.md")
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(actual), RenderCatalogMarkdown(registry); got != want {
		t.Fatalf("%s differs from the declared registry; regenerate it with:\n\tgo run ./cmd/gen-transition-catalog", path)
	}
}

func TestResolveWorkflowSelectsDeclaredLocator(t *testing.T) {
	registry, err := LoadFS(fstest.MapFS{"registry/work.json": {Data: []byte(`[
{"schemaVersion":"swarm-transition/v1","key":"work.follow_up","subject":"backlog-item","kind":"workflow","workflow":{"owner":"swarm-manager","key":"swarm-manager/custom-follow-up"},"inputContract":"work-follow-up-input/v1","terminalOutcomes":["completed"],"applyAction":"apply_follow_up"}
]`)}}, "registry")
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	workflow, err := registry.ResolveWorkflow("work.follow_up")
	if err != nil || workflow.Owner != "swarm-manager" || workflow.Key != "swarm-manager/custom-follow-up" {
		t.Fatalf("ResolveWorkflow = %#v, %v", workflow, err)
	}
	if _, err := registry.ResolveWorkflow("proposal.apply"); err == nil {
		t.Fatal("ResolveWorkflow accepted an undeclared transition")
	}
}

func TestEveryRegisteredWorkflowHasAMatchingDeclaration(t *testing.T) {
	registry, err := LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatalf("LoadDir declared registry: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join("..", "..", "..", ".vrooli", "agent-manager"))
	if err != nil {
		t.Fatalf("ReadDir workflow declarations: %v", err)
	}
	declared := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		contents, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "agent-manager", entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var workflow declaredWorkflow
		if err := json.Unmarshal(contents, &workflow); err != nil {
			t.Fatalf("decode %s: %v", entry.Name(), err)
		}
		if strings.TrimSpace(workflow.Owner) != "" && strings.TrimSpace(workflow.Key) != "" {
			declared[workflow.Owner+"/"+strings.TrimPrefix(workflow.Key, workflow.Owner+"/")] = struct{}{}
		}
	}
	for _, definition := range registry.Definitions() {
		if definition.Kind != KindWorkflow {
			continue
		}
		if _, ok := declared[definition.Workflow.Owner+"/"+strings.TrimPrefix(definition.Workflow.Key, definition.Workflow.Owner+"/")]; !ok {
			t.Errorf("workflow transition %q references undeclared workflow %s", definition.Key, definition.Workflow.Key)
		}
	}
}

func TestEveryDeclaredWorkflowIsReachableFromATransition(t *testing.T) {
	registry, err := LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	root := os.DirFS(filepath.Join("..", "..", "..", ".vrooli"))
	if err := ValidateWorkflowReachability(registry, root, "agent-manager", nil); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowReachabilityRequiresAReasonForUnboundDeclarations(t *testing.T) {
	registry, err := LoadFS(fstest.MapFS{"registry/root.json": {Data: []byte(`{
"schemaVersion":"swarm-transition/v1","key":"plan.execute","subject":"backlog-item","kind":"workflow",
"workflow":{"owner":"swarm-manager","key":"swarm-manager/root"},"inputContract":"plan/v1",
"terminalOutcomes":["completed"],"applyAction":"apply"}`)}}, "registry")
	if err != nil {
		t.Fatal(err)
	}
	workflows := fstest.MapFS{
		"workflows/root.json":   {Data: []byte(`{"schemaVersion":"agent-workflow/v1","key":"swarm-manager/root","nodes":[]}`)},
		"workflows/orphan.json": {Data: []byte(`{"schemaVersion":"agent-workflow/v1","key":"swarm-manager/orphan","nodes":[]}`)},
	}
	if err := ValidateWorkflowReachability(registry, workflows, "workflows", nil); err == nil || !strings.Contains(err.Error(), "swarm-manager/orphan") {
		t.Fatalf("unbound declaration error = %v", err)
	}
	if err := ValidateWorkflowReachability(registry, workflows, "workflows", map[string]string{"swarm-manager/orphan": "reserved for an operator-only experiment"}); err != nil {
		t.Fatalf("recorded unbound reason was rejected: %v", err)
	}
}

// [REQ:SWM-P0-014] governed workflows use canon-conformant contract-skill prompt refs
func TestRegisteredWorkflowNodesUseGovernedPromptReferences(t *testing.T) {
	registry, err := LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatalf("LoadDir declared registry: %v", err)
	}
	registered := map[string]struct{}{}
	for _, definition := range registry.Definitions() {
		if definition.Kind == KindWorkflow {
			registered[definition.Workflow.Key] = struct{}{}
		}
	}
	declarationDir := filepath.Join("..", "..", "..", ".vrooli", "agent-manager")
	entries, err := os.ReadDir(declarationDir)
	if err != nil {
		t.Fatalf("ReadDir workflow declarations: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(declarationDir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var workflow declaredWorkflow
		if err := json.Unmarshal(contents, &workflow); err != nil {
			t.Fatalf("decode %s: %v", entry.Name(), err)
		}
		if _, active := registered[workflow.Key]; !active {
			continue
		}
		for _, node := range workflow.Nodes {
			var prompt *nodePrompt
			switch node.Kind {
			case "run":
				prompt = node.Run
			case "continue":
				prompt = node.Continue
			default:
				continue
			}
			if prompt == nil {
				t.Errorf("workflow %q node %q has no prompt-bearing configuration", workflow.Key, node.ID)
				continue
			}
			if prompt.PromptTemplate != nil {
				t.Errorf("workflow %q node %q contains inline promptTemplate; use promptRef", workflow.Key, node.ID)
			}
			if prompt.PromptRef == nil || strings.TrimSpace(prompt.PromptRef.SkillID) == "" {
				t.Errorf("workflow %q node %q has no governed promptRef skillId", workflow.Key, node.ID)
			}
		}
	}
}

func TestValidateRejectsWorkflowMechanismFields(t *testing.T) {
	// Unknown JSON fields must be rejected by the decoder used for persisted
	// definitions, so a prompt, retry, or branch cannot be smuggled into Swarm.
	fsys := fstest.MapFS{"registry/invalid.json": {Data: []byte(`{
"schemaVersion":"swarm-transition/v1","key":"plan.author","subject":"backlog-item","kind":"workflow",
"workflow":{"owner":"swarm-manager","key":"swarm-manager/plan-author"},"inputContract":"plan-author-input/v1",
"terminalOutcomes":["ready"],"applyAction":"bind_validated_plan_ref","prompt":"forbidden"}`)}}
	if _, err := LoadFS(fsys, "registry"); err == nil {
		t.Fatal("LoadFS accepted an unknown workflow-mechanism field")
	}
}

func TestValidateKinds(t *testing.T) {
	base := Definition{SchemaVersion: SchemaVersion, Key: "proposal.apply", Subject: "proposal", InputContract: "proposal-apply-input/v1", TerminalOutcomes: []string{"applied"}, ApplyAction: "apply_proposal"}
	base.Kind = KindDeterministic
	if err := Validate(base); err != nil {
		t.Fatalf("Validate deterministic: %v", err)
	}
	base.Kind, base.Workflow = KindWorkflow, &Locator{Owner: "swarm-manager", Key: "swarm-manager/plan-author"}
	if err := Validate(base); err != nil {
		t.Fatalf("Validate workflow: %v", err)
	}
	base.Kind = KindSession
	if err := Validate(base); err == nil {
		t.Fatal("Validate accepted workflow locator for a session")
	}
}

func TestVerifyApplyActionsRequiresEverySelectedDispatcher(t *testing.T) {
	registry, err := LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	err = VerifyApplyActions(registry, map[string]struct{}{"apply_proposal": {}}, KindDeterministic)
	if err == nil || !strings.Contains(err.Error(), "dispatch_follow_up") || !strings.Contains(err.Error(), "mark_goal_achieved") {
		t.Fatalf("VerifyApplyActions error = %v", err)
	}
	if err := VerifyApplyActions(registry, map[string]struct{}{"apply_proposal": {}, "dispatch_follow_up": {}, "mark_goal_achieved": {}}, KindDeterministic); err != nil {
		t.Fatal(err)
	}
}
