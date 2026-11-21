package jobs

import (
	"testing"
	"time"
)

func TestRecurringDraftUsesLocalPeriodAcrossDST(t *testing.T) {
	recurrence := Recurrence{ID: "weekly-1", WorkspaceID: "w1", Timezone: "America/New_York", LocalHour: 8, LocalMinute: 30, Enabled: true}
	before, err := PrepareRecurringDraft(recurrence, time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	after, err := PrepareRecurringDraft(recurrence, time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if before.LocalDate != "2026-03-08" || after.LocalDate != "2026-03-09" || before.Job.DedupKey == after.Job.DedupKey {
		t.Fatalf("before=%#v after=%#v", before, after)
	}
	if !before.ReviewOnly || !after.ReviewOnly || before.Job.Type != "recurring_plan_draft" {
		t.Fatalf("scheduled draft policy was not review-only: before=%#v after=%#v", before, after)
	}
}

func TestRecurringDraftRetryHasStableDedupKey(t *testing.T) {
	recurrence := Recurrence{ID: "daily-1", WorkspaceID: "w1", Timezone: "UTC", LocalHour: 7, LocalMinute: 0, Enabled: true}
	first, err := PrepareRecurringDraft(recurrence, time.Date(2026, 9, 18, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	retry, err := PrepareRecurringDraft(recurrence, time.Date(2026, 9, 18, 23, 59, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if first.Job.DedupKey != retry.Job.DedupKey || first.Job.RequestHash != retry.Job.RequestHash {
		t.Fatalf("retry was not deduplicable: first=%#v retry=%#v", first.Job, retry.Job)
	}
}
