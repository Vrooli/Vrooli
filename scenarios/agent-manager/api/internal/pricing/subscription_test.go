package pricing

import (
	"testing"
	"time"

	"agent-manager/internal/domain"
)

func TestAllocateSubscriptionFeeRequiresExplicitBasis(t *testing.T) {
	period := domain.SubscriptionPeriod{AmountMicroUSD: 1_000_000, QuotaTokens: 100, StartsAt: time.Unix(0, 0), EndsAt: time.Unix(3600, 0)}
	if _, err := AllocateSubscriptionFee(period, "", 1, 10, 60); err == nil {
		t.Fatal("missing allocation basis accepted")
	}
	amount, err := AllocateSubscriptionFee(period, AllocationByToken, 1, 25, 0)
	if err != nil || amount != 250_000 {
		t.Fatalf("amount=%d err=%v", amount, err)
	}
}
