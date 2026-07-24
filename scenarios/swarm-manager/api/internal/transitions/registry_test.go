package transitions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

type declaredWorkflow struct {
	Owner string `json:"owner"`
	Key   string `json:"key"`
	Nodes []struct {
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
	if got, want := len(registry.Definitions()), 18; got != want {
		t.Fatalf("registered transition count = %d, want %d", got, want)
	}
	for _, key := range []string{
		"capture.classify", "plan.workshop.review", "plan.workshop.reconcile", "plan.author", "plan.repair", "plan.execute",
		"work.review", "review.evidence_request", "goal.discover", "goal.plan", "milestone.review", "scenario.spec_sync",
		"session.meta_orchestration", "session.swarm_operations", "session.workflow_authoring",
	} {
		if _, ok := registry.Get(key); !ok {
			t.Errorf("registry is missing %q", key)
		}
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
