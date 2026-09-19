package goals

import (
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals"
	g "personal-planner/internal/goals"
)

func toProto(x g.Goal) *v.Goal {
	return &v.Goal{Id: x.ID, Title: x.Title, Purpose: x.Purpose, Status: x.Status, ProgressMethod: x.ProgressMethod, ProgressBasisPoints: x.ProgressBasisPoints, TargetBasisPoints: x.TargetBasisPoints, Revision: x.Revision}
}
