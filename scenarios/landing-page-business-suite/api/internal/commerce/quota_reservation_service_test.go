package commerce

import (
	"context"
	"testing"
)

func TestQuotaReservationTransitionMatrix(t *testing.T) {
	for _, state := range []string{"pending", "finalized", "released", "expired", "unknown"} {
		for event, next := range map[string]string{"finalize": "finalized", "release": "released", "expire": "expired"} {
			got, err := quotaTransition(state, event)
			if state == "pending" {
				if err != nil || got != next {
					t.Fatalf("%s + %s = %s,%v", state, event, got, err)
				}
			} else if err == nil {
				t.Fatalf("terminal/unknown %s accepted %s", state, event)
			}
		}
	}
}

func TestQuotaOwnerRejectsCreditKindBeforePersistence(t *testing.T) {
	owner := NewQuotaReservationService(nil, fixedAccountLimits{value: 100}, "sqlite", nil)
	if _, err := owner.Reserve(context.Background(), "member@example.test", "pro", "ai_credits", 1, 0); err == nil {
		t.Fatal("credit units must never enter the non-credit quota owner")
	}
}
