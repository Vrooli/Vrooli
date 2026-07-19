package archtest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBacklogWorkshopPilotUsesWorkflowBoundaryOnly(t *testing.T) {
	root := swarmScenarioRoot(t)
	for _, relative := range []string{
		"api/internal/backlog/research.go",
		"api/internal/backlog/workshop_workflow.go",
		"api/internal/agentmanager/workflow.go",
	} {
		data, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, forbidden := range []string{".CreateRun(", ".ContinueRun(", "SpawnBacklog(", "SpawnResearch("} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s bypasses workflow boundary with %s", relative, forbidden)
			}
		}
	}
}

func TestBacklogWorkshopUsesGenericInvocationSeam(t *testing.T) {
	root := swarmScenarioRoot(t)
	backlogSource, err := os.ReadFile(filepath.Join(root, "api/internal/backlog/workshop_workflow.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(backlogSource)
	for _, required := range []string{".StartWorkflow(", ".CollectWorkflow(", "agentmanager.Invocation{"} {
		if !strings.Contains(text, required) {
			t.Fatalf("backlog workshop adapter missing generic workflow seam %q", required)
		}
	}
	agentManagerSource, err := os.ReadFile(filepath.Join(root, "api/internal/agentmanager/workflow.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"BacklogWorkshop", "StartWorkshopRound", "CollectWorkshopRound"} {
		if strings.Contains(string(agentManagerSource), forbidden) {
			t.Fatalf("agent-manager retains a backlog-specific workflow adapter %q", forbidden)
		}
	}
}

func TestBacklogWorkshopDefinitionStaysStructurallySmall(t *testing.T) {
	path := filepath.Join(swarmScenarioRoot(t), ".vrooli", "agent-manager", "backlog-workshop-round.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var definition map[string]any
	if err := json.Unmarshal(data, &definition); err != nil {
		t.Fatal(err)
	}
	nodes, _ := definition["nodes"].([]any)
	if len(nodes) != 2 {
		t.Fatalf("node count = %d, want one fresh run plus end", len(nodes))
	}
	encoded := string(data)
	for _, forbidden := range []string{"targetKind", "domainAction", "callback", "classifier", "continue", "shell", "command"} {
		if strings.Contains(encoded, `"`+forbidden+`"`) {
			t.Fatalf("workflow contains forbidden framework field %q", forbidden)
		}
	}
	if !strings.Contains(encoded, `"profileKey": "swarm-manager/deep-work"`) || !strings.Contains(encoded, `"oneOf"`) {
		t.Fatal("workflow does not declare its node-local profile and discriminated result")
	}
}

func swarmScenarioRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
