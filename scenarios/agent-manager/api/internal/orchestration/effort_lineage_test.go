package orchestration

import (
	"testing"

	"agent-manager/internal/domain"

	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

func TestChildRunInheritsParentEffortReferenceAsWorker(t *testing.T) {
	effort := func(id, relationship string, verified bool) *eventpb.WorkReference {
		return &eventpb.WorkReference{Kind: "effort", Id: id, Revision: "9", Relationship: relationship, Verified: verified, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}
	}
	parent := &domain.Run{WorkReferences: []*eventpb.WorkReference{
		effort("effort:goal", "orchestrator", true),
		effort("effort:unverified", "orchestrator", false),
		effort("effort:declared", "orchestrator", true),
		{Kind: "scenario-instance", Id: "bas@shadow", Relationship: "targets"},
	}}
	declared := effort("effort:declared", "reviewer", true)
	refs := inheritEffortReferences(parent, []*eventpb.WorkReference{declared})
	if len(refs) != 2 || refs[0] != declared {
		t.Fatalf("want the declared reference kept and one inherited reference, got %v", refs)
	}
	got := refs[1]
	if got.GetId() != "effort:goal" || got.GetRelationship() != "worker" || got.GetRevision() != "9" || !got.GetVerified() {
		t.Fatalf("inherited reference must be the parent's effort as worker, got %v", got)
	}
	if parent.WorkReferences[0].GetRelationship() != "orchestrator" {
		t.Fatal("inheritance mutated the parent's reference")
	}
	if refs := inheritEffortReferences(nil, nil); refs != nil {
		t.Fatalf("a run without a parent inherits nothing, got %v", refs)
	}
}
