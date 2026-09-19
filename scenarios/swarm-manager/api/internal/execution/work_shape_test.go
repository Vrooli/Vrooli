package execution

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

// [REQ:SWM-P0-002] Retained development authority must not enter plan execution
// through direct queue/start APIs, force, or execution-record input builders.
func TestPlanWorkGuardProtectsQueueStartAndTransition(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "shape", map[string]any{"name": "shape", "title": "Shape", "status": "backlog"})
	mustWriteDeliverableFile(t, root, "idea", "shape")
	s := NewService(ServiceConfig{DataRoot: root, StorePath: filepath.Join(root, "runs.json"), PlanRenderer: testPlanRenderer(), AgentService: &stubAgentService{}, TransitionRegistry: testTransitionRegistry(t)})
	workflow := &stubPhasedPlanWorkflow{}
	s.SetPhasedPlanWorkflow(workflow)
	denied := errors.New("contract-development cannot use plan execution")
	s.SetPlanWorkGuard(func(_ context.Context, ref string) error {
		if ref != "idea/shape" {
			t.Fatalf("wrong owner reference: %s", ref)
		}
		return denied
	})
	req := CreateRequest{BacklogKind: "idea", BacklogName: "shape", Mode: ModeManual, Force: true}
	if _, err := s.QueueBacklog(t.Context(), req); !errors.Is(err, denied) {
		t.Fatalf("force bypassed guard: %v", err)
	}
	records, err := s.store.Load()
	if err != nil || len(records) != 0 {
		t.Fatalf("denied queue mutated records: %+v %v", records, err)
	}
	if got := mustLoadBacklogItem(t, filepath.Join(root, "ideas", "shape", "spec.json"))["status"]; got != "backlog" {
		t.Fatalf("denied queue changed status: %v", got)
	}

	// Simulate an older pending record, then consult current owner state again.
	s.SetPlanWorkGuard(nil)
	record, err := s.QueueBacklog(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	s.SetPlanWorkGuard(func(context.Context, string) error { return denied })
	if _, err := s.Start(t.Context(), record.ExecutionID); !errors.Is(err, denied) {
		t.Fatalf("pending start bypassed guard: %v", err)
	}
	for _, key := range []string{"plan.execute", "work.correct", "work.follow_up", "scenario.spec_sync"} {
		if _, err := s.transitionRunner.BuildInput(t.Context(), key, record.ExecutionID); !errors.Is(err, denied) {
			t.Fatalf("%s bypassed guard: %v", key, err)
		}
	}
	if workflow.startCalls != 0 {
		t.Fatalf("guard dispatched %d workflows", workflow.startCalls)
	}
}
