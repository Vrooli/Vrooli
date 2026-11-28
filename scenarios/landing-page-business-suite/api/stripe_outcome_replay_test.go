package main

import (
	"testing"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/monetization"
)

func TestStripeOutcomeEnvironmentFailsClosedOutsideExplicitTestTargets(t *testing.T) {
	for _, env := range []string{"development", "local", "test", "staging"} {
		if !stripeOutcomeEnvironmentAllowed(env) {
			t.Errorf("%q should be allowed for test-mode Stripe outcomes", env)
		}
	}
	for _, env := range []string{"", "production", "prod", "qa", "unknown"} {
		if stripeOutcomeEnvironmentAllowed(env) {
			t.Errorf("%q must not be allowed for test-mode Stripe outcomes", env)
		}
	}
}

func TestStripeOutcomeLocalStandInOnlyMarksLocalTargets(t *testing.T) {
	for _, env := range []string{"development", "local", "test"} {
		if !stripeOutcomeLocalStandIn(env) {
			t.Errorf("%q should be labeled as a local stand-in", env)
		}
	}
	for _, env := range []string{"staging", "production", "unknown"} {
		if stripeOutcomeLocalStandIn(env) {
			t.Errorf("%q should not be labeled as a local stand-in", env)
		}
	}
}

func TestStripeOutcomeBehaviorsRequireEveryReceipt(t *testing.T) {
	if stripeOutcomeBehaviorsPassed(nil) {
		t.Fatal("nil report must not pass")
	}
	report := &monetization.StripeOutcomeReport{Outcomes: []monetization.OutcomeObservation{{Name: "real_webhook_delivery", Status: "passed"}}}
	if stripeOutcomeBehaviorsPassed(report) {
		t.Fatal("a webhook receipt must not substitute for the required payment outcomes")
	}
	for _, name := range stripeOutcomeNames {
		report.Outcomes = append(report.Outcomes, monetization.OutcomeObservation{Name: name, Status: "passed"})
	}
	if !stripeOutcomeBehaviorsPassed(report) {
		t.Fatal("all individually passed outcomes should pass the behavior gate")
	}
}

func TestPastDueOutcomeRetainsBothSignedEventsAndBothFacts(t *testing.T) {
	report := &monetization.StripeOutcomeReport{Outcomes: []monetization.OutcomeObservation{{Name: "past_due", Status: "unknown"}}}
	failed := &commerce.StripeTestLifecycleReceipt{
		EventID: "evt_invoice_failed", RunID: "py1-run", Outcome: "past_due", EventType: "invoice.payment_failed",
		ObjectID: "in_failed", SubscriptionID: "sub_past_due", PaymentStatus: "failed", MoneyLedgerStatus: "not_applicable",
	}
	status := &commerce.StripeTestLifecycleReceipt{
		EventID: "evt_subscription_past_due", RunID: "py1-run", Outcome: "past_due", EventType: "customer.subscription.updated",
		ObjectID: "sub_past_due", SubscriptionID: "sub_past_due", SubscriptionStatus: "past_due", MoneyLedgerStatus: "not_applicable",
	}
	setPY1PastDueOutcomeObservation(report, status, failed)
	item := report.Outcomes[0]
	if item.Status != "passed" || item.StripeEventID != status.EventID || item.StripeEventType != status.EventType || item.SubscriptionStatus != "past_due" || item.PaymentStatus != "failed" || len(item.RelatedStripeEventIDs) != 1 || item.RelatedStripeEventIDs[0] != failed.EventID {
		t.Fatalf("past_due outcome = %+v", item)
	}
	if item.Detail == "" {
		t.Fatal("past_due outcome should describe both causal events")
	}
}

func TestPastDueOutcomeRefusesUncorrelatedEvents(t *testing.T) {
	report := &monetization.StripeOutcomeReport{Outcomes: []monetization.OutcomeObservation{{Name: "past_due", Status: "unknown"}}}
	failed := &commerce.StripeTestLifecycleReceipt{EventID: "evt_failed", RunID: "run-a", Outcome: "past_due", SubscriptionID: "sub-a", PaymentStatus: "failed"}
	status := &commerce.StripeTestLifecycleReceipt{EventID: "evt_status", RunID: "run-b", Outcome: "past_due", SubscriptionID: "sub-a", SubscriptionStatus: "past_due"}
	setPY1PastDueOutcomeObservation(report, status, failed)
	if report.Outcomes[0].Status != "unknown" || report.Outcomes[0].StripeEventID != "" {
		t.Fatalf("uncorrelated events must not produce a pass: %+v", report.Outcomes[0])
	}
}
