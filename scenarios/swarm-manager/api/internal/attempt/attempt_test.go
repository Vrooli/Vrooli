package attempt

import (
	"testing"

	"swarm-manager/internal/transitionrun"
)

func TestProjectJoinsLifecycleWithoutPersistingIt(t *testing.T) {
	value := Attempt{SubjectKind: "backlog-item", SubjectRef: "execute/item", TransitionKey: "work.review", RoundNum: 1, Status: "complete"}
	got := Project(value, transitionrun.Correlation{ExecutionID: "exec-1", ApplyState: transitionrun.ApplyStateComplete, Outcome: "accepted", TerminalCode: "accepted"})
	if got.ExecutionID != "exec-1" || got.ApplyState != transitionrun.ApplyStateComplete || got.Outcome != "accepted" {
		t.Fatalf("projection = %#v", got)
	}
}
