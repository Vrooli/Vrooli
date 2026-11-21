package approval

import (
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/operatordecision"

	"treasury/internal/budget"
)

func heldCharge() Request {
	return Request{
		ID: "cf1-agent-manager-auth-over-cap:approval", AuthorizationID: "cf1-agent-manager-auth-over-cap", BookID: "book-7", MandateID: "mandate-3",
		RequestingAgent: "agent:run-42", AmountMinor: 110, Currency: "USD", Counterparty: "0x70997970c51812dc3a010c7d01b50e0d17dc79c8",
		Status: StatusQueued, ExpiresAt: time.Date(2026, 11, 15, 14, 5, 0, 0, time.UTC),
	}
}

func TestHeldPaymentArrivesAsAPageTheOperatorCanAnswerCold(t *testing.T) {
	limits := &Limits{
		HeldBecause: "operator approval is required for this exact charge above the automatic transaction cap",
		Headroom:    budget.Headroom{PerTransactionCapMinor: 100, PeriodicCapMinor: 1000, PeriodUsedMinor: 250, PeriodRemainingMinor: 750, TotalCapMinor: 5000, TotalUsedMinor: 900, TotalRemainingMinor: 4100},
	}
	facts := approvalDecisionFacts(heldCharge(), limits)
	if err := facts.Validate(); err != nil {
		t.Fatal(err)
	}
	if problems := operatordecision.CoverProblems(facts); len(problems) > 0 {
		t.Errorf("a cold reader cannot parse the cover: %v", problems)
	}
	if facts.Question != "Approve one held payment of $1.10 to wallet 0x7099…79c8? It is over the automatic per-payment limit of $1.00." {
		t.Errorf("question = %q", facts.Question)
	}
	for name, text := range map[string]string{"question": facts.Question, "big picture": facts.BigPicture, "subject": facts.Subject} {
		for _, id := range []string{"cf1-", "book-7", "mandate-3", "run-42", "110 USD"} {
			if strings.Contains(text, id) {
				t.Errorf("%s %q carries %q; ids and raw minor units belong in technical details", name, text, id)
			}
		}
	}
	for _, want := range []string{
		"0x70997970c51812dc3a010c7d01b50e0d17dc79c8", "agent:run-42", "14:05 UTC", // full payee, who asked, deadline
		"Why it was held:", "Over by $0.10", "$2.50 used of $10.00", "Left after paying: $6.40", "$9.00 used of $50.00",
		"Only this payment:", `data-od-answer="approve"`, `data-od-answer="decline"`, "the hold lapses",
	} {
		if !strings.Contains(facts.Explainer, want) {
			t.Errorf("page is missing %q", want)
		}
	}
	if !strings.Contains(facts.TechnicalDetails, "cf1-agent-manager-auth-over-cap:approval") || facts.SubjectKey == "" || !operatordecision.SubjectKeyRule.MatchString(facts.SubjectKey) {
		t.Errorf("technical details or subject key missing: %q %q", facts.TechnicalDetails, facts.SubjectKey)
	}
}

func TestHeldPaymentPageStillAnswerableWithoutLimits(t *testing.T) {
	facts := approvalDecisionFacts(heldCharge(), nil)
	if err := facts.Validate(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(facts.Question, "limit") || strings.Contains(facts.Explainer, "Against the budget") {
		t.Errorf("without limits the decision must not invent them: %q", facts.Question)
	}
	if !strings.Contains(facts.Explainer, "$1.10") {
		t.Error("page must still show the amount")
	}
}

func TestMoneyRendersMinorUnits(t *testing.T) {
	for in, want := range map[int64]string{110: "$1.10", 5: "$0.05", 123456: "$1234.56", -250: "-$2.50"} {
		if got := money(in, "usd"); got != want {
			t.Errorf("money(%d) = %q, want %q", in, got, want)
		}
	}
	if got := money(110, "JPYX"); got != "1.10 JPYX" {
		t.Errorf("unknown currency = %q", got)
	}
}
