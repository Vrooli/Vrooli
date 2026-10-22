package goals

import (
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals"
	g "personal-planner/internal/goals"
)

func toProto(x g.Goal) *v.Goal {
	goal := &v.Goal{Id: x.ID, Title: x.Title, Purpose: x.Purpose, Status: x.Status, ProgressMethod: x.ProgressMethod, TargetBasisPoints: x.TargetBasisPoints, Revision: x.Revision, TargetDate: x.TargetDate}
	if x.Measure != nil {
		goal.Measure = &v.GoalMeasure{Unit: x.Measure.Unit, TargetAmountMinor: x.Measure.TargetAmountMinor, CurrencyCode: x.Measure.CurrencyCode, Basis: x.Measure.Basis, Recurrence: x.Measure.Recurrence, Period: x.Measure.Period}
		goal.ProgressState = "unknown"
	} else {
		goal.ProgressBasisPoints = &x.ProgressBasisPoints
	}
	return goal
}
