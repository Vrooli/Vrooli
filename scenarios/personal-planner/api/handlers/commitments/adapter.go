package commitments

import (
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/commitments"
	c "personal-planner/internal/commitments"
)

func toProto(x c.Commitment) *v.Commitment {
	return &v.Commitment{Id: x.ID, Result: x.Result, DefinitionOfDone: x.DefinitionOfDone, PromisedBoundary: x.PromisedBoundary, Timezone: x.Timezone, Beneficiary: x.Beneficiary, Assumptions: x.Assumptions, ScopeExclusions: x.ScopeExclusions, State: x.State, Risk: x.Risk, AcknowledgmentStatus: x.AcknowledgmentStatus, CreatedAtUnixSeconds: x.CreatedAt, UpdatedAtUnixSeconds: x.UpdatedAt, Revision: x.Revision}
}
func revisionToProto(x c.Revision) *v.CommitmentRevision {
	return &v.CommitmentRevision{Id: x.ID, CommitmentId: x.CommitmentID, PromisedBoundary: x.PromisedBoundary, Assumptions: x.Assumptions, ScopeExclusions: x.ScopeExclusions, Reason: x.Reason, AcknowledgmentStatus: x.AcknowledgmentStatus, CreatedAtUnixSeconds: x.CreatedAt, Revision: x.Revision}
}
