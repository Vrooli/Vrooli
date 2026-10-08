package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Recurrence struct {
	ID          string
	WorkspaceID string
	Timezone    string
	LocalHour   int
	LocalMinute int
	Enabled     bool
}

type ScheduledDraft struct {
	Job        Job
	LocalDate  string
	ReviewOnly bool
}

// PrepareRecurringDraft creates a deduplicable queued proposal for the local
// calendar day. It intentionally contains no accepted plan, consumption,
// supplement, purchase, or notification side effect.
func PrepareRecurringDraft(recurrence Recurrence, now time.Time) (ScheduledDraft, error) {
	if recurrence.ID == "" || recurrence.WorkspaceID == "" {
		return ScheduledDraft{}, fmt.Errorf("recurrence requires id and workspace")
	}
	if !recurrence.Enabled {
		return ScheduledDraft{}, fmt.Errorf("recurrence is disabled")
	}
	if recurrence.LocalHour < 0 || recurrence.LocalHour > 23 || recurrence.LocalMinute < 0 || recurrence.LocalMinute > 59 {
		return ScheduledDraft{}, fmt.Errorf("recurrence local time is invalid")
	}
	location, err := time.LoadLocation(recurrence.Timezone)
	if err != nil {
		return ScheduledDraft{}, fmt.Errorf("recurrence timezone: %w", err)
	}
	local := now.In(location)
	localDate := local.Format("2006-01-02")
	dedup := strings.Join([]string{recurrence.ID, localDate}, ":")
	job, err := New(uuid.NewString(), recurrence.WorkspaceID, "recurring_plan_draft", dedup)
	if err != nil {
		return ScheduledDraft{}, err
	}
	job.RequestHash = fmt.Sprintf("%s|%s|%02d:%02d|%s", recurrence.ID, recurrence.WorkspaceID, recurrence.LocalHour, recurrence.LocalMinute, recurrence.Timezone)
	return ScheduledDraft{Job: job, LocalDate: localDate, ReviewOnly: true}, nil
}

func EnqueueRecurringDraft(ctx context.Context, repo Repository, recurrence Recurrence, now time.Time) (Job, error) {
	draft, err := PrepareRecurringDraft(recurrence, now)
	if err != nil {
		return Job{}, err
	}
	return repo.Create(ctx, draft.Job)
}
