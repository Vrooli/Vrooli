package evidence

import (
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/proto"
)

// CloneRetainedSets returns independent copies of non-nil retained evidence
// sets. The returned slice is non-nil even when sets is nil, matching the
// request-boundary behavior used by Test Genie's validation flows.
func CloneRetainedSets(sets []*scenariovalidationv1.RetainedEvidenceSet) []*scenariovalidationv1.RetainedEvidenceSet {
	cloned := make([]*scenariovalidationv1.RetainedEvidenceSet, 0, len(sets))
	for _, set := range sets {
		if set != nil {
			cloned = append(cloned, proto.Clone(set).(*scenariovalidationv1.RetainedEvidenceSet))
		}
	}
	return cloned
}
