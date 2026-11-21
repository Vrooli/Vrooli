package workflowcatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"agent-manager/internal/domain"
)

func TestBountyReplaySolverAndScorerContractsAreBlindAndRoleSeparated(t *testing.T) {
	root := filepath.Join("..", "..", "..", ".vrooli", "agent-manager")
	loadJSON := func(name string) map[string]any {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := json.Unmarshal(data, &value); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		return value
	}
	loadWorkflow := func(name string) domain.WorkflowDefinition {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		result, err := Parse(data, nil)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if result.Digest == "" || domain.HasBlockingDiagnostic(result.Diagnostics) {
			t.Fatalf("workflow %s is not admissible: %+v", name, result.Diagnostics)
		}
		return result.Definition
	}

	solverWorkflow := loadWorkflow("bounty-solver-workflow.json")
	scorerWorkflow := loadWorkflow("bounty-scorer-workflow.json")
	solverProfile := loadJSON("bounty-solver.json")
	scorerProfile := loadJSON("bounty-scorer.json")

	solverRole := solverWorkflow.Nodes[0].Run.ProfileKey
	scorerRole := scorerWorkflow.Nodes[0].Run.ProfileKey
	if solverRole != "agent-manager/bounty-solver" || scorerRole != "agent-manager/bounty-scorer" || solverRole == scorerRole {
		t.Fatalf("expected distinct solver and scorer profiles, got %q and %q", solverRole, scorerRole)
	}
	if solverProfile["roleRef"] == scorerProfile["roleRef"] {
		t.Fatalf("solver and scorer must resolve to distinct Agent Manager roles: %v", solverProfile["roleRef"])
	}
	for name, profile := range map[string]map[string]any{"solver": solverProfile, "scorer": scorerProfile} {
		if profile["toolRestrictionPolicy"] != "enforced" || profile["networkAccess"] != "NETWORK_ACCESS_NONE" {
			t.Errorf("%s profile must enforce its tool and network boundary", name)
		}
	}
	if tools, ok := scorerProfile["allowedTools"].([]any); !ok || len(tools) != 0 {
		t.Fatalf("scorer must have no tools, got %#v", scorerProfile["allowedTools"])
	}
	if cfg, ok := solverProfile["sandboxConfig"].(map[string]any); !ok || cfg["manualReview"] != false || cfg["autoApply"] != true {
		t.Fatalf("solver changes must be applied only to the disposable candidate checkout without a human gate: %#v", solverProfile["sandboxConfig"])
	}

	assertInputKeys := func(name string, definition domain.WorkflowDefinition, want ...string) {
		t.Helper()
		var schema map[string]any
		if err := json.Unmarshal(definition.InputSchema, &schema); err != nil {
			t.Fatal(err)
		}
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s input has no properties", name)
		}
		for _, forbidden := range []string{"answer", "reference", "winningPatch", "referencePatch", "sourceArchive"} {
			if _, exists := props[forbidden]; exists {
				t.Errorf("%s input exposes forbidden answer material field %q", name, forbidden)
			}
		}
		if len(props) != len(want) {
			t.Errorf("%s input keys changed: got %d properties, expected %d", name, len(props), len(want))
		}
		for _, key := range want {
			if _, exists := props[key]; !exists {
				t.Errorf("%s input is missing required contract field %q", name, key)
			}
		}
	}
	assertInputKeys("solver", solverWorkflow, "caseId", "task", "checkout")
	assertInputKeys("scorer", scorerWorkflow, "caseId", "solverRunId", "frozenEvaluatorDigest", "preimageEvaluatorDigest", "candidateEvaluatorDigest", "referenceEvaluatorDigest", "preimageReceiptDigest", "candidateReceiptDigest", "referenceReceiptDigest", "preimagePassed", "candidatePassed", "referencePassed")

	if solverWorkflow.Nodes[0].Run.ProjectRootTemplate != "{{.checkout}}" || solverWorkflow.Nodes[0].Run.ScopePathTemplate != "." {
		t.Fatalf("solver must run in the caller-selected disposable checkout: %+v", solverWorkflow.Nodes[0].Run)
	}
	if scorerWorkflow.Nodes[0].Run.ProjectRootTemplate != "" || scorerWorkflow.Nodes[0].Run.ScopePathTemplate != "" {
		t.Fatalf("scorer must not receive a source checkout: %+v", scorerWorkflow.Nodes[0].Run)
	}

	assertPromptBindings := func(name, skillID string, run *domain.WorkflowRunNode) {
		t.Helper()
		if run.PromptRef == nil || run.PromptRef.SkillID != skillID {
			t.Fatalf("%s must reference prompt contract %q, got %+v", name, skillID, run.PromptRef)
		}
		skillPath := filepath.Join("..", "..", "..", "..", "prompt-manager", "store", "skills", "packs", "core", skillID, "SKILL.md")
		content, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatal(err)
		}
		used := map[string]bool{}
		for _, match := range regexp.MustCompile(`\{\{\.([a-z][a-z0-9_]*)\}\}`).FindAllStringSubmatch(string(content), -1) {
			used[match[1]] = true
		}
		bound := map[string]bool{}
		for _, binding := range run.Bindings {
			bound[binding.Name] = true
			if !used[binding.Name] {
				t.Errorf("%s declares unused binding %q", name, binding.Name)
			}
		}
		for variable := range used {
			if !bound[variable] {
				t.Errorf("%s skill uses unbound template variable %q", name, variable)
			}
		}
	}
	assertPromptBindings("solver", "bounty-replay-solver", solverWorkflow.Nodes[0].Run)
	assertPromptBindings("scorer", "bounty-replay-scorer", scorerWorkflow.Nodes[0].Run)
	if strings.Contains(string(scorerWorkflow.InputSchema), `"answer"`) || strings.Contains(string(scorerWorkflow.InputSchema), `"referencePatch"`) {
		t.Fatal("scorer contract exposed answer material")
	}
}
