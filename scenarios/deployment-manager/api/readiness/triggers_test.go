package readiness

import (
	"testing"
	"time"
)

func TestTriggerInputDetectsChangesAndElapsedApproval(t *testing.T) {
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if !(TriggerInput{Kind: TriggerPriceChange, PreviousValue: "10", CurrentValue: "12"}).Fired(now) {
		t.Fatal("expected price change trigger")
	}
	if !(TriggerInput{Kind: TriggerElapsedApproval, PreviousValue: "same", CurrentValue: "same", LastApprovedAt: now.Add(-25 * time.Hour), ApprovalLifetime: 24 * time.Hour}).Fired(now) {
		t.Fatal("expected elapsed approval trigger")
	}
}

func TestWaiverRequiresReasonActorAndCommit(t *testing.T) {
	if err := (Waiver{Actor: "operator", Commit: "abc", At: time.Now()}).Validate(); err == nil {
		t.Fatal("expected reasonless waiver refusal")
	}
	if err := (Waiver{Reason: "incident", Actor: "operator", Commit: "abc", At: time.Now()}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAcceptanceExpiresWhenTriggerFires(t *testing.T) {
	acceptance := Acceptance{Reason: "temporary", Actor: "operator", Commit: "abc", ExpiresOnTrigger: true}
	if !acceptance.Expired(true) || acceptance.Expired(false) {
		t.Fatalf("acceptance expiry = %v", acceptance)
	}
}
