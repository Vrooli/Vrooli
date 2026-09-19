package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyBacklogWorkshopImplementationsAreDeleted(t *testing.T) {
	root := swarmScenarioRoot(t)
	for _, name := range []string{"research.go", "workshop_save.go", "workshop_workflow.go", "clarification.go", "clarification_state.go", "clarification_workflow.go"} {
		if _, err := os.Stat(filepath.Join(root, "api", "internal", "backlog", name)); !os.IsNotExist(err) {
			t.Fatalf("legacy implementation %s remains", name)
		}
	}
}

func TestPlanAuthorUsesSharedTransitionRunner(t *testing.T) {
	path := filepath.Join("..", "backlog", "plan_author.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, forbidden := range []string{"StartWorkflow(", "CollectWorkflow(", "planAuthorPending"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("plan author must delegate workflow mechanics to transitionrunner, found %q", forbidden)
		}
	}
	for _, required := range []string{"transitionRunner.Start(", "transitionRunner.Apply("} {
		if !strings.Contains(source, required) {
			t.Fatalf("plan author must use shared transition runner %q", required)
		}
	}
}

func TestPlanRepairUsesSharedTransitionRunnerAndNoLocalLedger(t *testing.T) {
	root := swarmScenarioRoot(t)
	path := filepath.Join(root, "api", "internal", "backlog", "plan_repair.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, forbidden := range []string{"StartWorkflow(", "CollectWorkflow(", "planRepair.Start(", "planRepair.Collect("} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("plan repair must delegate workflow mechanics to transitionrunner, found %q", forbidden)
		}
	}
	for _, required := range []string{"transitionRunner.StartPrepared(", "transitionRunner.Apply("} {
		if !strings.Contains(source, required) {
			t.Fatalf("plan repair must use shared transition runner %q", required)
		}
	}
	for _, name := range []string{"service.go", "store.go"} {
		if _, err := os.Stat(filepath.Join(root, "api", "internal", "planrepair", name)); !os.IsNotExist(err) {
			t.Fatalf("legacy plan-repair correlation implementation %s remains", name)
		}
	}
}
