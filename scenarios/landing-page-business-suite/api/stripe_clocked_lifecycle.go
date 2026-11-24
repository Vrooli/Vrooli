package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/monetization"
)

// replayPY1TestClockSubscriptionLifecycle advances isolated Stripe test-clock
// subscriptions and accepts outcomes only after signed webhook processing,
// LPBS durable receipts, and the renewal's accepted Money Ledger posting.
func (s *Server) replayPY1TestClockSubscriptionLifecycle(ctx context.Context, runID string, listener *stripeTestListener, writer commerce.ServiceOrderLedgerWriter, target commerce.PY1TestLedgerTarget, report *monetization.StripeOutcomeReport) error {
	if s == nil || s.stripeService == nil || listener == nil || writer == nil || report == nil {
		return errors.New("PY1 subscription lifecycle dependencies are incomplete")
	}
	priceID := s.lowestConfiguredSubscriptionTestPrice()
	if priceID == "" {
		return errors.New("LPBS has no configured subscription test price for the clock lifecycle")
	}
	frozenAt := time.Now().UTC().Unix() - 30
	clock, err := s.stripeService.createPY1TestClock(ctx, runID, frozenAt)
	if err != nil || clock == nil {
		return errors.New("could not create isolated Stripe test clock")
	}
	subscriptions := make(map[string]*stripeTestClockSubscription, 3)
	for _, outcome := range []string{"renewal", "past_due", "cancel"} {
		subscription, createErr := s.stripeService.createPY1TestClockSubscription(ctx, runID, clock.ID, priceID, outcome)
		if createErr != nil || subscription == nil || subscription.CurrentPeriodEnd <= frozenAt {
			return errors.New("could not create an active clock-bound test subscription")
		}
		subscriptions[outcome] = subscription
	}
	declinePaymentMethod, err := s.stripeService.createPY1DecliningRenewalPaymentMethod(ctx, runID, subscriptions["past_due"].CustomerID)
	if err != nil {
		return errors.New("could not prepare the isolated test card for failed renewal")
	}
	if err := s.stripeService.setPY1TestClockDefaultPaymentMethod(ctx, runID, subscriptions["past_due"].SubscriptionID, declinePaymentMethod, "past_due"); err != nil {
		return errors.New("could not set the failed-renewal test payment method")
	}
	if _, err := s.stripeService.schedulePY1TestClockCancellation(ctx, runID, subscriptions["cancel"].SubscriptionID); err != nil {
		return errors.New("could not schedule test subscription cancellation")
	}
	advanceTo := subscriptions["renewal"].CurrentPeriodEnd
	for _, outcome := range []string{"past_due", "cancel"} {
		if subscriptions[outcome].CurrentPeriodEnd > advanceTo {
			advanceTo = subscriptions[outcome].CurrentPeriodEnd
		}
	}
	clock, err = s.stripeService.advancePY1TestClock(ctx, runID, clock.ID, advanceTo+30)
	if err != nil {
		return errors.New("could not advance the Stripe test clock to the subscription boundary")
	}
	clock, err = s.stripeService.waitForPY1TestClock(ctx, runID, clock.ID, time.Second)
	if err != nil || clock == nil || clock.Status != "ready" {
		return errors.New("Stripe test clock did not finish processing lifecycle events")
	}
	webhooks := s.stripeService.webhookService()
	if err := webhooks.DispatchPendingTestLifecycleReceipts(ctx, writer, target, runID); err != nil {
		return errors.New("Money Ledger lifecycle outbox retry failed after test clock completion")
	}

	needed := map[string]bool{"renewal": false, "past_due": false, "cancel": false}
	if receipt, err := webhooks.LoadTestLifecycleRenewalReceipt(ctx, runID, "renewal"); err != nil {
		return errors.New("could not check for a previously retained renewal receipt")
	} else if receipt != nil && receipt.MoneyLedgerStatus == "posted" {
		needed["renewal"] = true
		setPY1LifecycleOutcomeObservation(report, receipt)
	}
	var pastDueInvoice *commerce.StripeTestLifecycleReceipt
	var pastDueStatus *commerce.StripeTestLifecycleReceipt
	if receipt, err := webhooks.LoadTestLifecycleReceiptByOutcomeAndEventType(ctx, runID, "past_due", "invoice.payment_failed"); err != nil {
		return errors.New("could not check for a previously retained failed-renewal receipt")
	} else {
		pastDueInvoice = receipt
	}
	if receipt, err := webhooks.LoadTestLifecycleReceiptByOutcomeEventTypeAndStatus(ctx, runID, "past_due", "customer.subscription.updated", "past_due"); err != nil {
		return errors.New("could not check for a previously retained past-due status receipt")
	} else if receipt != nil && pastDueInvoice != nil && receipt.SubscriptionID == pastDueInvoice.SubscriptionID && pastDueInvoice.PaymentStatus == "failed" {
		needed["past_due"] = true
		setPY1PastDueOutcomeObservation(report, receipt, pastDueInvoice)
	}
	if receipt, err := webhooks.LoadTestLifecycleReceiptByOutcomeAndEventType(ctx, runID, "cancel", "customer.subscription.deleted"); err != nil {
		return errors.New("could not check for a previously retained cancellation receipt")
	} else if receipt != nil && receipt.SubscriptionStatus == "canceled" {
		needed["cancel"] = true
		setPY1LifecycleOutcomeObservation(report, receipt)
	}
	acceptedTypes := map[string]bool{
		"invoice.paid": true, "invoice.payment_failed": true,
		"customer.subscription.updated": true, "customer.subscription.deleted": true,
	}
	for !needed["renewal"] || !needed["past_due"] || !needed["cancel"] {
		status, eventID, eventType, waitErr := listener.waitForEventDelivery(ctx, acceptedTypes)
		if waitErr != nil || status != "passed" || eventID == "" {
			return errors.New("signed Stripe lifecycle webhook delivery was not observed")
		}
		receipt, receiptErr := webhooks.LoadTestLifecycleReceiptByEvent(ctx, eventID)
		if receiptErr != nil {
			return errors.New("LPBS could not load the lifecycle receipt by Stripe event ID")
		}
		if receipt == nil || receipt.RunID != runID {
			continue
		}
		if receipt.EventType != eventType {
			return errors.New("LPBS lifecycle receipt event type differs from delivered Stripe event")
		}
		if receipt.EventType == "invoice.paid" && receipt.MoneyLedgerStatus == "pending" {
			if _, err := webhooks.DispatchTestLifecycleEventReceipt(ctx, writer, target, receipt.EventID); err != nil {
				return errors.New("Money Ledger did not accept a paid test subscription invoice")
			}
		}
		matches := false
		switch {
		case receipt.Outcome == "renewal" && eventType == "invoice.paid" && receipt.BillingReason == "subscription_cycle" && receipt.PaymentStatus == "paid":
			needed["renewal"] = true
			matches = true
		case receipt.Outcome == "past_due" && eventType == "invoice.payment_failed" && receipt.PaymentStatus == "failed":
			pastDueInvoice = receipt
			if pastDueStatus != nil && pastDueStatus.SubscriptionID == receipt.SubscriptionID {
				needed["past_due"] = true
				setPY1PastDueOutcomeObservation(report, pastDueStatus, receipt)
			}
		case receipt.Outcome == "past_due" && eventType == "customer.subscription.updated" && receipt.SubscriptionStatus == "past_due":
			pastDueStatus = receipt
			if pastDueInvoice != nil && pastDueInvoice.SubscriptionID == receipt.SubscriptionID {
				needed["past_due"] = true
				setPY1PastDueOutcomeObservation(report, receipt, pastDueInvoice)
			}
		case receipt.Outcome == "cancel" && eventType == "customer.subscription.deleted" && receipt.SubscriptionStatus == "canceled":
			needed["cancel"] = true
			matches = true
		}
		if matches {
			setPY1LifecycleOutcomeObservation(report, receipt)
		}
	}
	if err := webhooks.DispatchPendingTestLifecycleReceipts(ctx, writer, target, runID); err != nil {
		return errors.New("Money Ledger lifecycle outbox has an unaccepted paid invoice")
	}

	renewal, err := webhooks.DispatchTestLifecycleReceipt(ctx, writer, target, runID, "renewal")
	if err != nil || renewal == nil || renewal.MoneyLedgerStatus != "posted" {
		return errors.New("paid test renewal has no accepted Money Ledger posting receipt")
	}
	setPY1LifecycleOutcomeObservation(report, renewal)
	return nil
}

func setPY1LifecycleOutcomeObservation(report *monetization.StripeOutcomeReport, receipt *commerce.StripeTestLifecycleReceipt) {
	setPY1LifecycleOutcomeObservationWithRelated(report, receipt)
}

func setPY1PastDueOutcomeObservation(report *monetization.StripeOutcomeReport, statusEvent, failedInvoice *commerce.StripeTestLifecycleReceipt) {
	if statusEvent == nil || failedInvoice == nil || statusEvent.SubscriptionID != failedInvoice.SubscriptionID || statusEvent.RunID != failedInvoice.RunID || statusEvent.Outcome != "past_due" || failedInvoice.Outcome != "past_due" || statusEvent.SubscriptionStatus != "past_due" || failedInvoice.PaymentStatus != "failed" {
		return
	}
	setPY1LifecycleOutcomeObservationWithRelated(report, statusEvent, failedInvoice.EventID)
	if report == nil {
		return
	}
	for i := range report.Outcomes {
		item := &report.Outcomes[i]
		if item.Name != "past_due" {
			continue
		}
		item.PaymentStatus = failedInvoice.PaymentStatus
		item.BillingReason = failedInvoice.BillingReason
		item.Detail = fmt.Sprintf("Stripe failed-renewal event %s and subscription-state event %s were persisted for subscription %s; status %s, payment %s, Money Ledger state %s.", failedInvoice.EventID, statusEvent.EventID, statusEvent.SubscriptionID, statusEvent.SubscriptionStatus, failedInvoice.PaymentStatus, statusEvent.MoneyLedgerStatus)
		return
	}
}

func setPY1LifecycleOutcomeObservationWithRelated(report *monetization.StripeOutcomeReport, receipt *commerce.StripeTestLifecycleReceipt, related ...string) {
	if report == nil || receipt == nil {
		return
	}
	for i := range report.Outcomes {
		item := &report.Outcomes[i]
		if item.Name != receipt.Outcome {
			continue
		}
		item.StripeEventID = receipt.EventID
		for _, eventID := range related {
			if eventID == "" || eventID == receipt.EventID {
				continue
			}
			found := false
			for _, existing := range item.RelatedStripeEventIDs {
				if existing == eventID {
					found = true
					break
				}
			}
			if !found {
				item.RelatedStripeEventIDs = append(item.RelatedStripeEventIDs, eventID)
			}
		}
		item.StripeEventType = receipt.EventType
		item.StripeObjectID = receipt.ObjectID
		item.SubscriptionID = receipt.SubscriptionID
		item.SubscriptionStatus = receipt.SubscriptionStatus
		item.PaymentStatus = receipt.PaymentStatus
		item.BillingReason = receipt.BillingReason
		item.LifecycleReceiptStatus = "persisted"
		item.PostgresReceiptStatus = "processed"
		item.MoneyLedgerStatus = receipt.MoneyLedgerStatus
		item.MoneyLedgerExternalID = receipt.MoneyLedgerExternalID
		item.MoneyLedgerPostingID = receipt.MoneyLedgerPostingID
		item.MoneyLedgerReceiptID = receipt.MoneyLedgerReceiptID
		item.AmountMinor = receipt.AmountMinor
		item.Currency = receipt.Currency
		if receipt.MoneyLedgerStatus == "not_applicable" || receipt.MoneyLedgerStatus == "posted" {
			item.Status = "passed"
		}
		item.Detail = fmt.Sprintf("Stripe lifecycle event %s persisted for subscription %s with status %s; Money Ledger state %s.", receipt.EventID, receipt.SubscriptionID, receipt.SubscriptionStatus, receipt.MoneyLedgerStatus)
		return
	}
}
