package pricing

import (
	"fmt"
	"math"
	"strings"

	"agent-manager/internal/domain"
)

// SubscriptionAllocationBasis is deliberately caller-supplied. There is no
// universally correct way to amortize a subscription fee.
type SubscriptionAllocationBasis string

const (
	AllocationByRun       SubscriptionAllocationBasis = "by_run"
	AllocationByToken     SubscriptionAllocationBasis = "by_token"
	AllocationByWallClock SubscriptionAllocationBasis = "by_wall_clock"
)

// AllocateSubscriptionFee allocates a period fee without mutating the
// recorded period or any run row. The denominator is explicit and must be
// supplied by the caller for the selected basis.
func AllocateSubscriptionFee(period domain.SubscriptionPeriod, basis SubscriptionAllocationBasis, runs int64, tokens int64, wallClockSeconds float64) (int64, error) {
	if strings.TrimSpace(string(basis)) == "" {
		return 0, fmt.Errorf("subscription allocation basis is required")
	}
	if period.AmountMicroUSD < 0 {
		return 0, fmt.Errorf("subscription amount cannot be negative")
	}
	var numerator, denominator float64
	switch basis {
	case AllocationByRun:
		numerator, denominator = 1, float64(runs)
	case AllocationByToken:
		numerator, denominator = float64(tokens), float64(period.QuotaTokens)
	case AllocationByWallClock:
		numerator, denominator = wallClockSeconds, period.EndsAt.Sub(period.StartsAt).Seconds()
	default:
		return 0, fmt.Errorf("unsupported subscription allocation basis %q", basis)
	}
	if denominator <= 0 || numerator < 0 {
		return 0, fmt.Errorf("subscription allocation denominator must be positive")
	}
	amount := float64(period.AmountMicroUSD) * numerator / denominator
	if amount > float64(math.MaxInt64) {
		return 0, fmt.Errorf("subscription allocation overflows micro-usd")
	}
	return int64(math.Round(amount)), nil
}
