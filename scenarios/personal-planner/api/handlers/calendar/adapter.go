package calendar

import (
	"time"

	calendar "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/calendar"
	"google.golang.org/protobuf/types/known/timestamppb"
	d "personal-planner/internal/calendar"
)

func toProto(a d.Allocation) *calendar.Allocation {
	return &calendar.Allocation{Id: a.ID, WorkItemId: a.WorkItemID, Title: a.Title, SourceLabel: a.SourceLabel, LocalDate: a.LocalDate, StartMinutes: int32(a.StartMinutes), DurationMinutes: int32(a.DurationMinutes), State: a.State, CreatedAt: timestamp(a.CreatedAt), CarriedFromId: a.CarriedFromID}
}
func timestamp(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }
