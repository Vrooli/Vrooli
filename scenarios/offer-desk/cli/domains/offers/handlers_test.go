package offers

import (
	"strings"
	"testing"

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
