package offers

import (
	"strings"
	"testing"
	"time"

	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
)

func TestBoardReportIncludesFinancialPosture(t *testing.T) {
	report := boardReport(nil, &offerspb.BoardResponse{
		Position:          &ledgerpb.PositionResponse{RevenueMinor: 1200, BurnMinor: 900, CashMinor: 5000, Currency: "USD", RunwayMonths: 5.5},
		DefaultAliveGap:   "default-alive threshold met with 1.25 buffer",
		PostureSource:     "money-ledger.position: Operating",
		PostureAgeSeconds: 12,
		Goals:             []*ledgerpb.GoalVerdict{{Goal: &ledgerpb.Goal{Name: "default-alive"}, Met: true, SustainedPeriods: 3, RequiredPeriods: 3}},
	})
	joined := strings.Join(report.Summary, "\n")
	for _, expected := range []string{
		"revenue=1200 burn=900 cash=5000 USD runway=5.50 months",
		"Default-alive gap: default-alive threshold met with 1.25 buffer",
		"Posture source: money-ledger.position: Operating (age=12s)",
		"Goal default-alive: met=true sustained=3/3",
	} {
		if !strings.Contains(joined, expected) {
			t.Errorf("board summary missing %q: %s", expected, joined)
		}
	}
}

func TestResolveFactObservedAtUsesInjectedClockAndHonorsExplicitTime(t *testing.T) {
	now := time.Date(2026, 11, 24, 1, 2, 3, 0, time.FixedZone("test", 2*60*60))
	observed, err := resolveFactObservedAt("", func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if !observed.Equal(now.UTC()) || observed.Location() != time.UTC {
		t.Fatalf("default observed_at = %s, want UTC %s", observed, now.UTC())
	}

	explicit := "2026-11-23T23:59:00Z"
	observed, err = resolveFactObservedAt(explicit, func() time.Time {
		t.Fatal("explicit timestamps must not consult the clock")
		return time.Time{}
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := observed.Format(time.RFC3339); got != explicit {
		t.Fatalf("explicit observed_at = %s, want %s", got, explicit)
	}
}

func TestResolveFactObservedAtRejectsMissingClockAndInvalidExplicitTime(t *testing.T) {
	if _, err := resolveFactObservedAt("", nil); err == nil || !strings.Contains(err.Error(), "clock is not configured") {
		t.Fatalf("missing clock error = %v", err)
	}
	if _, err := resolveFactObservedAt("not-a-time", func() time.Time { return time.Now() }); err == nil || !strings.Contains(err.Error(), "must be RFC3339") {
		t.Fatalf("invalid timestamp error = %v", err)
	}
}
