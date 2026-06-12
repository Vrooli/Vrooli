package autosteer

import "testing"

func TestRunCostKnown(t *testing.T) {
	if (RunCost{}).Known() {
		t.Fatal("zero RunCost must be unknown")
	}
	if !(RunCost{TotalTokens: 1}).Known() {
		t.Fatal("positive total must be known")
	}
	if (RunCost{TotalTokens: -5}).Known() {
		t.Fatal("negative total must be unknown")
	}
}

func TestOrchestratorRunCostStash(t *testing.T) { // [REQ:EM-P1-001]
	o := &ExecutionOrchestrator{pendingCost: make(map[string]RunCost)}

	// Unknown costs are not stashed.
	o.RecordRunCost("task-a", RunCost{TotalTokens: 0})
	if got := o.takeRunCost("task-a"); got.Known() {
		t.Fatalf("unknown cost should not be stashed, got %+v", got)
	}

	// A known cost round-trips once and is then consumed.
	o.RecordRunCost("task-a", RunCost{TotalTokens: 500})
	if got := o.takeRunCost("task-a"); got.TotalTokens != 500 {
		t.Fatalf("expected 500 tokens, got %d", got.TotalTokens)
	}
	if got := o.takeRunCost("task-a"); got.Known() {
		t.Fatal("cost must be consumed exactly once")
	}

	// Costs are keyed per task.
	o.RecordRunCost("task-b", RunCost{TotalTokens: 10})
	if got := o.takeRunCost("task-a"); got.Known() {
		t.Fatal("task-a must not see task-b's cost")
	}
	if got := o.takeRunCost("task-b"); got.TotalTokens != 10 {
		t.Fatalf("expected task-b cost 10, got %d", got.TotalTokens)
	}
}
