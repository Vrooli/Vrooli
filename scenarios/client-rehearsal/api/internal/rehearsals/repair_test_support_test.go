package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
)

func requireRepairFixtureHash(t *testing.T, label, raw, want string) {
	t.Helper()
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(raw))); got != want {
		t.Fatalf("%s SHA-256 changed: %s", label, got)
	}
}

// runPublicWorkflowRepairPair owns the common two-run repair lifecycle. The
// caller keeps the workflow-specific failure/effect and corrected-operation
// expectations beside the case that defines them.
func runPublicWorkflowRepairPair(t *testing.T, service Service, seeded CreateInput, publicWorkflow, correctedTitle string) (Rehearsal, Rehearsal) {
	t.Helper()
	failed, err := service.CreateAndRun(context.Background(), seeded)
	if err != nil {
		t.Fatalf("run seeded defect %q: %v", seeded.Title, err)
	}
	if failed.State != StateFailed {
		t.Fatalf("seeded workflow %q state = %s, want %s", seeded.Title, failed.State, StateFailed)
	}

	correctedInput := seeded
	correctedInput.Title = correctedTitle
	correctedInput.WorkflowJSON = publicWorkflow
	correctedInput.RepairOfID = failed.ID
	corrected, err := service.CreateAndRun(context.Background(), correctedInput)
	if err != nil {
		t.Fatalf("run corrected workflow %q: %v", correctedTitle, err)
	}
	if corrected.State != StateNeedsReview || !traceHasOperation(corrected.Trace, "repair lineage") {
		t.Fatalf("corrected workflow %q lacks review state and repair lineage: %#v", correctedTitle, corrected)
	}
	return failed, corrected
}

// requireWorkflowTriggerID preserves the case's independently selected trigger
// kind/name while failing if a fixture drifts to zero or multiple matches.
func requireWorkflowTriggerID(t *testing.T, workflow Workflow, kind TriggerKind, name string) string {
	t.Helper()
	var matches []string
	for _, node := range workflow.Nodes {
		if node.Trigger == kind && (name == "" || node.Name == name) {
			matches = append(matches, node.ID)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("workflow has %d matching %s trigger(s) named %q, want exactly one", len(matches), kind, name)
	}
	return matches[0]
}
