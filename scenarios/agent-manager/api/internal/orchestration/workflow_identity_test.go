package orchestration

import "testing"

func TestWorkflowIdentityMeta_UsesOnlyWorkflowLineageValues(t *testing.T) {
	meta := workflowIdentityMeta(map[string]string{
		workflowExecutionEnv: " execution-1 ",
		workflowNodeEnv:      " node-a ",
		workflowAttemptEnv:   " attempt-1 ",
		"VROOLI_UNRELATED":   "ignored",
	})
	want := map[string]string{"workflowExecutionId": "execution-1", "workflowNodeId": "node-a", "workflowAttemptId": "attempt-1"}
	if len(meta) != len(want) {
		t.Fatalf("meta = %#v, want %#v", meta, want)
	}
	for key, value := range want {
		if meta[key] != value {
			t.Fatalf("meta[%q] = %q, want %q", key, meta[key], value)
		}
	}
}
