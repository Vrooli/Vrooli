package aimoneyfixtures

import (
	"context"
	"errors"
	"sync"
)

const (
	StripeAsyncPaymentSucceeded = "checkout.session.async_payment_succeeded"
	StripeAsyncPaymentFailed    = "checkout.session.async_payment_failed"
	StripeDisputeCreated        = "charge.dispute.created"
	StripeDisputeClosed         = "charge.dispute.closed"
	StripeRefunded              = "charge.refunded"
	StripeEarlyFraudWarning     = "radar.early_fraud_warning.created"
)

type StripeEvent struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	ObjectID string `json:"object_id"`
	Livemode bool   `json:"livemode"`
}

// FakeStripe is a deterministic event stream. It does not sign webhooks or
// emulate Stripe's transport/retry behavior; those need real/mock Stripe tests.
type FakeStripe struct {
	mu     sync.Mutex
	events []StripeEvent
	next   int
}

func (f *FakeStripe) Publish(event StripeEvent) error {
	if event.ID == "" || event.Type == "" {
		return errors.New("stripe event id and type are required")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, existing := range f.events {
		if existing.ID == event.ID {
			return nil
		}
	}
	f.events = append(f.events, event)
	return nil
}

func (f *FakeStripe) Next(ctx context.Context) (StripeEvent, error) {
	select {
	case <-ctx.Done():
		return StripeEvent{}, ctx.Err()
	default:
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.next >= len(f.events) {
		return StripeEvent{}, errors.New("no stripe event available")
	}
	event := f.events[f.next]
	f.next++
	return event, nil
}

func (f *FakeStripe) Events() []StripeEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]StripeEvent, len(f.events))
	copy(out, f.events)
	return out
}
