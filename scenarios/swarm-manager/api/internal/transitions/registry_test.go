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
}

func TestDeclaredRegistryCoversEveryTargetTransition(t *testing.T) {
	registry, err := LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatalf("LoadDir declared registry: %v", err)
	}
	if got, want := len(registry.Definitions()), 23; got != want {
		t.Fatalf("registered transition count = %d, want %d", got, want)
	}
	for _, key := range []string{
		"capture.classify", "backlog.refine", "plan.author", "plan.repair", "plan.execute",
		"work.review", "review.evidence_request", "work.control", "initiative.discover", "initiative.execute", "scenario.spec_sync",
	} {
		if _, ok := registry.Get(key); !ok {
			t.Errorf("registry is missing %q", key)
		}
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
