package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"swarm-manager/internal/transitions"
)

// workflowDeclaration is the slice of an Agent Manager workflow file this test
// cares about: what the declaration says its input must contain.
type workflowDeclaration struct {
	Key         string `json:"key"`
	InputSchema struct {
		Required []string `json:"required"`
	} `json:"inputSchema"`
}

func loadWorkflowDeclarations(t *testing.T) map[string]workflowDeclaration {
	t.Helper()
	dir := filepath.Join("..", ".vrooli", "agent-manager")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	declarations := map[string]workflowDeclaration{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var declaration workflowDeclaration
		if err := json.Unmarshal(raw, &declaration); err != nil {
			t.Fatalf("decode %s: %v", entry.Name(), err)
		}
		if declaration.Key != "" {
			declarations[declaration.Key] = declaration
		}
	}
	return declarations
}

// TestEveryWorkflowTransitionHasARegisteredInputBuilder is the composition-time
// half of the dispatch gate. VerifyDispatchTable only checks apply actions, so
// a transition could boot with no way to build its input at all.
func TestEveryWorkflowTransitionHasARegisteredInputBuilder(t *testing.T) {
	server := newTestServer(t)
	if server.transitionRunner == nil {
		t.Fatal("test server has no transition runner")
	}
	var missing []string
	for _, definition := range server.transitionRegistry.Definitions() {
		if definition.Kind != transitions.KindWorkflow {
			continue
		}
		if !server.transitionRunner.HasInput(definition.Key) {
			missing = append(missing, definition.Key)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("workflow transitions with no registered input builder: %v", missing)
	}
}

// TestRegisteredBuildersCoverTheirDeclaredInputContract is the regression test
// for the placeholder-builder defect. Six builders returned only an `entity`
// key while their workflow declared `snapshot`, `plan`, `constraints` and more,
// so StartTransition could never produce a valid input. The check is structural
// — it compares what a builder is expected to produce against what the workflow
// declaration requires — because the runtime call needs a real subject and
// several subjects here are expensive to fixture.
func TestRegisteredBuildersCoverTheirDeclaredInputContract(t *testing.T) {
	server := newTestServer(t)
	declarations := loadWorkflowDeclarations(t)

	// producedTopLevelKeys records the top-level keys each registered builder
	// emits. Keep it in sync with the builder when you change its input shape;
	// the test fails when a declaration requires a key no builder produces.
	producedTopLevelKeys := map[string][]string{
		"capture.classify":        {"capture", "grounding"},
		"plan.author":             {"entity", "snapshot"},
		"plan.repair":             {"entity", "plan", "validation", "constraints"},
		"plan.execute":            {"projectRoot", "plan", "planExecutionId", "consumer", "constraints"},
		"work.review":             {"entity", "snapshot"},
		"review.evidence_request": {"entity", "snapshot"},
		"work.correct":            {"entity", "snapshot"},
		"work.follow_up":          {"entity", "snapshot"},
		"scenario.spec_sync":      {"entity", "snapshot"},
		"goal.discover":           {"entity", "snapshot", "supported_ops"},
		"goal.plan":               {"entity", "snapshot", "supported_ops"},
		"milestone.review":        {"entity", "snapshot", "supported_ops"},
		"plan.workshop.review":    {"subject", "snapshot"},
		"plan.workshop.reconcile": {"subject", "snapshot", "response", "accepted_proposals"},
	}

	for _, definition := range server.transitionRegistry.Definitions() {
		if definition.Kind != transitions.KindWorkflow || definition.Workflow == nil {
			continue
		}
		declaration, ok := declarations[definition.Workflow.Key]
		if !ok {
			t.Errorf("transition %q names workflow %q with no declaration file", definition.Key, definition.Workflow.Key)
			continue
		}
		produced, ok := producedTopLevelKeys[definition.Key]
		if !ok {
			t.Errorf("transition %q has no recorded builder output shape; add one so its input contract stays checked", definition.Key)
			continue
		}
		emitted := map[string]struct{}{}
		for _, key := range produced {
			emitted[key] = struct{}{}
		}
		for _, required := range declaration.InputSchema.Required {
			if _, ok := emitted[required]; !ok {
				t.Errorf("transition %q builds an input without %q, which workflow %q declares as required", definition.Key, required, definition.Workflow.Key)
			}
		}
	}
}
