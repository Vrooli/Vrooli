package review

import (
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/review"
	r "personal-planner/internal/review"
)

func toProto(x r.DailySummary) *v.DailySummary {
	return &v.DailySummary{LocalDate: x.LocalDate, PlannedMinutes: x.PlannedMinutes, RecordedActiveMinutes: x.RecordedActiveMinutes, FocusSessionCount: x.FocusSessionCount, ActiveGoalCount: x.ActiveGoalCount, UnrecordedMinutes: x.UnrecordedMinutes, CoverageNote: x.CoverageNote}
}

func weeklyToProto(x r.WeeklySummary) *v.WeeklySummary {
	out := &v.WeeklySummary{WeekStartLocalDate: x.WeekStartLocalDate, PlannedMinutes: x.PlannedMinutes, RecordedActiveMinutes: x.RecordedActiveMinutes, FocusSessionCount: x.FocusSessionCount, ActiveGoalCount: x.ActiveGoalCount, CoverageNote: x.CoverageNote}
	for _, day := range x.Days {
		out.Days = append(out.Days, toProto(day))
	}
	return out
}

func reflectionToProto(x r.ReviewReflection) *v.ReviewReflection {
	return &v.ReviewReflection{LocalDate: x.LocalDate, Text: x.Text, UpdatedAt: x.UpdatedAt}
}
