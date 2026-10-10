// This file carries a parent run's effort reference to its children (DL-2).
package orchestration

import (
	"context"

	"agent-manager/internal/domain"

	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
)

// effortWorkerRelationship is the relationship an inherited effort reference
// carries on a child run.
const effortWorkerRelationship = "worker"

// inheritEffortReferences returns refs plus, for each public, active,
// verified effort reference the parent run holds, the same reference as
// "worker" when the child does not already name that effort. Lineage is
// effort-level: any later check-in of the effort sees the child through the
// reference, while ParentRunID stays the immutable provenance link.
func inheritEffortReferences(parent *domain.Run, refs []*eventpb.WorkReference) []*eventpb.WorkReference {
	if parent == nil {
		return refs
	}
	named := map[string]bool{}
	for _, ref := range refs {
		if ref.GetKind() == "effort" {
			named[ref.GetId()] = true
		}
	}
	for _, ref := range parent.WorkReferences {
		if !inheritableEffortReference(ref) || named[ref.GetId()] {
			continue
		}
		child := proto.Clone(ref).(*eventpb.WorkReference)
		child.Relationship = effortWorkerRelationship
		refs = append(refs, child)
		named[ref.GetId()] = true
	}
	return refs
}

func inheritableEffortReference(ref *eventpb.WorkReference) bool {
	return ref.GetKind() == "effort" && ref.GetId() != "" && ref.GetVerified() &&
		ref.GetVisibility() == eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC &&
		ref.GetState() == eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE
}

// inheritParentEffortReferences applies inheritEffortReferences to a create
// request with a parent. An unreadable parent leaves the request unchanged;
// dependent-delegation admission already refuses that case.
func (o *Orchestrator) inheritParentEffortReferences(ctx context.Context, req *CreateRunRequest) {
	if req.ParentRunID == nil || o.runs == nil {
		return
	}
	parent, err := o.runs.Get(ctx, *req.ParentRunID)
	if err != nil {
		return
	}
	req.WorkReferences = inheritEffortReferences(parent, req.WorkReferences)
}
