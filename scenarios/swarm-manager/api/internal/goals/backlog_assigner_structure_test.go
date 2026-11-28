package goals

import (
	"strings"
	"testing"

	"swarm-manager/internal/backlog"
)

// The batch seam carries no acceptance criteria, so any milestone it wrote
// would be permanently unverifiable. It also used to invent a goal named after
// the milestone when the goal was absent, which is what produced the mirrored
// goal/milestone pairs in the store. Both writes are refused.
func TestBacklogMilestoneAssignerRefusesToWriteGoalStructure(t *testing.T) {
	svc := newTestService(t, nil)
	assigner := NewBacklogMilestoneAssigner(svc)

	if err := assigner.Create(backlog.MilestoneSpec{Name: "brand-new-goal/some-milestone", Title: "Some milestone"}); err == nil {
		t.Fatal("assigner created goal structure")
	} else if !strings.Contains(err.Error(), "goal API") {
		t.Fatalf("error should route the caller to the goal API, got: %v", err)
	}
	if _, err := svc.Get("brand-new-goal"); err == nil {
		t.Fatal("assigner invented a goal for an unknown reference")
	}
	if err := assigner.Update(backlog.MilestoneSpec{Name: "brand-new-goal/some-milestone", Title: "Renamed"}); err == nil {
		t.Fatal("assigner updated goal structure")
	}
}
