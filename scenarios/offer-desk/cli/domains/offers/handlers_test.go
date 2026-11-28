package offers

import (
	"strings"
	"testing"
	"time"

	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func TestAuditReportMarksBoundedHistoryAndPreservesDecisionFields(t *testing.T) {
	report := auditReport(nil, &offerspb.ListAuditResponse{
		Records: []*offerspb.CatalogAuditEntry{{
			NodeId: "offer-1", Actor: "operator", PriorStatus: offerspb.Status_IDEA,
			NextStatus: offerspb.Status_ACTIVE, Reason: "approved after review",
			CreatedAt: timestamppb.New(time.Date(2026, 11, 24, 8, 0, 0, 0, time.UTC)),
		}},
		Truncated: true,
	})
	if len(report.Summary) != 2 || !strings.Contains(report.Summary[1], "truncated") {
		t.Fatalf("truncation summary = %#v", report.Summary)
	}
	if len(report.Results) != 1 {
		t.Fatalf("got %d rows, want 1", len(report.Results))
	}
	for _, expected := range []string{"offer-1", "operator", "IDEA→ACTIVE", "approved after review", "2026-11-24T08:00:00Z"} {
		if !strings.Contains(report.Results[0], expected) {
			t.Errorf("audit row missing %q: %s", expected, report.Results[0])
		}
	}
}

func TestGateReadReportsExposeFreshnessAndTriggerVerdictsBoundary(t *testing.T) {
	facts := factsReport(nil, &offerspb.ListFactsResponse{
		EvaluatedAt: timestamppb.New(time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)),
		Facts: []*offerspb.FactRead{{
			Fact:      &offerspb.Fact{Name: "activation_rate", Value: 0.7, ObservedAt: timestamppb.New(time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)), StaleAfterDays: 1},
			Freshness: offerspb.FactFreshness_FACT_FRESHNESS_STALE,
		}},
	})
	if !strings.Contains(strings.Join(facts.Results, "\n"), "activation_rate=0.7 freshness=stale") || !strings.Contains(strings.Join(facts.Results, "\n"), "stale_after_days=1") || !strings.Contains(strings.Join(facts.Summary, "\n"), "freshness evaluated at 2026-11-24T12:00:00Z") {
		t.Fatalf("fact read omitted value, evaluated freshness, or observation time: %#v", facts)
	}
	triggers := triggersReport(nil, &offerspb.ListTriggersResponse{Triggers: []*offerspb.Trigger{{
		Id: "t1", NodeId: "n1", Composition: offerspb.TriggerComposition_ALL,
		Clauses: []*offerspb.TriggerClause{{FactName: "activation_rate", Operator: ">=", Threshold: 0.5}},
	}}})
	if !strings.Contains(triggers.Results[0], "activation_rate >= 0.5") || !strings.Contains(strings.Join(triggers.Summary, " "), "run gates-evaluate") {
		t.Fatalf("trigger read omitted condition or evaluation boundary: %#v", triggers)
	}
}
