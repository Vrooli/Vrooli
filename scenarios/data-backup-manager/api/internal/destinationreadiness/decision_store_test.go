package destinationreadiness

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/operatordecision"
)

func TestFileDecisionStorePersistsAndCountsPendingAsk(t *testing.T) {
	store := &FileDecisionStore{Root: filepath.Join(t.TempDir(), "asks")}
	now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	ask := &operatordecision.PendingAsk{
		ID: "ask-1", Subject: "recovery-1", State: operatordecision.PendingStatePending,
		RaisedAt: now, ExpiresAt: now.Add(time.Hour), EventID: "event-1", Published: true,
		Extra: map[string]string{"recovery_id": "recovery-1", "plan_id": "plan-1"},
	}
	if err := store.Put(context.Background(), ask); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := store.Get(context.Background(), ask.ID)
	if err != nil || got == nil {
		t.Fatalf("Get = (%+v, %v)", got, err)
	}
	if !got.Published || got.Extra["plan_id"] != "plan-1" {
		t.Fatalf("ask fields did not round-trip: %+v", got)
	}
	count, err := store.CountOutstandingForSubject(context.Background(), "recovery-1", now)
	if err != nil || count != 1 {
		t.Fatalf("CountOutstandingForSubject = (%d, %v), want (1, nil)", count, err)
	}
	got.State = operatordecision.PendingStateRejected
	if err := store.Put(context.Background(), got); err != nil {
		t.Fatalf("Put resolved: %v", err)
	}
	count, err = store.CountOutstandingForSubject(context.Background(), "recovery-1", now)
	if err != nil || count != 0 {
		t.Fatalf("resolved ask count = (%d, %v), want (0, nil)", count, err)
	}
}
