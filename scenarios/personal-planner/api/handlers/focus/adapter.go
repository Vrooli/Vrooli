package focus

import (
	"time"

	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/focus"
	"personal-planner/internal/focus"
)

func domainToProto(session focus.Session) *focusv1.FocusSession {
	return &focusv1.FocusSession{
		Id: session.ID, WorkItemId: session.WorkItemID, Title: session.Title,
		Mode: session.Mode, State: session.State, StartedAtUnixSeconds: session.StartedAt.Unix(),
		ActiveStartedAtUnixSeconds: activeStartedUnix(session.ActiveStartedAt),
		EndedAtUnixSeconds:         session.EndedAt.Unix(), ActiveSeconds: session.ActiveSeconds,
		WallSeconds: session.WallSeconds, Revision: session.Revision,
	}
}

func activeStartedUnix(value *time.Time) int64 {
	if value == nil {
		return 0
	}
	return value.Unix()
}

func actualToProto(actual focus.Actual) *focusv1.Actual {
	return &focusv1.Actual{Id: actual.ID, WorkItemId: actual.WorkItemID, Title: actual.Title, LocalDate: actual.LocalDate, ReportedMinutes: actual.ReportedMinutes, Certainty: actual.Certainty, Note: actual.Note, CreatedAtUnixSeconds: actual.CreatedAt.Unix(), Revision: actual.Revision}
}
