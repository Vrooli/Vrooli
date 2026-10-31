package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrReceiptUnsupported = errors.New("receipt source is unsupported")
	ErrReceiptInvalid     = errors.New("receipt could not be verified")
	ErrReceiptBound       = errors.New("receipt is bound to another account")
	ErrReceiptReplay      = errors.New("receipt has already been registered")
)

// Receipt is the platform-neutral input accepted by the purchase rail seam.
// The token is verified by the selected platform validator before persistence.
type Receipt struct {
	Source       string
	Token        string
	UserIdentity string
}

// NormalizedSubscription is the only shape the commerce core consumes. Once
// normalized, entitlement resolution is deliberately unaware of the source.
type NormalizedSubscription struct {
	SubscriptionID       string
	ExternalSubscription string
	UserIdentity         string
	Status               string
	PlanTier             string
	PriceID              string
	BundleKey            string
}

// ReceiptValidator turns one verified platform receipt into a subscription.
// Implementations must reject unverifiable or already-bound receipts.
type ReceiptValidator interface {
	Validate(context.Context, Receipt) (NormalizedSubscription, error)
}

// ReceiptValidators routes sources to independent platform seams.
type ReceiptValidators map[string]ReceiptValidator

// Validate selects a source validator without exposing source branches to
// account, wallet, lease, or gate code.
func (v ReceiptValidators) Validate(ctx context.Context, receipt Receipt) (NormalizedSubscription, error) {
	source := strings.ToLower(strings.TrimSpace(receipt.Source))
	validator, ok := v[source]
	if !ok {
		return NormalizedSubscription{}, fmt.Errorf("%w: %s", ErrReceiptUnsupported, source)
	}
	return validator.Validate(ctx, receipt)
}

// VerifiedReceiptFunc adapts a platform SDK verifier to ReceiptValidator.
type VerifiedReceiptFunc func(context.Context, Receipt) (NormalizedSubscription, error)

// Validate implements ReceiptValidator.
func (f VerifiedReceiptFunc) Validate(ctx context.Context, receipt Receipt) (NormalizedSubscription, error) {
	if f == nil {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	return f(ctx, receipt)
}

// AppleReceiptValidator is the StoreKit seam. The verifier is where Apple
// signed-transaction/JWS and root-certificate validation is injected.
type AppleReceiptValidator struct{ Verify VerifiedReceiptFunc }

// Validate verifies an Apple transaction and normalizes it.
func (v AppleReceiptValidator) Validate(ctx context.Context, receipt Receipt) (NormalizedSubscription, error) {
	if strings.ToLower(strings.TrimSpace(receipt.Source)) != "apple" {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	return v.Verify.Validate(ctx, receipt)
}

// GoogleReceiptValidator is the Play Billing seam. The verifier is where the
// Play Developer API purchase-token validation is injected.
type GoogleReceiptValidator struct{ Verify VerifiedReceiptFunc }

// Validate verifies a Google purchase token and normalizes it.
func (v GoogleReceiptValidator) Validate(ctx context.Context, receipt Receipt) (NormalizedSubscription, error) {
	if strings.ToLower(strings.TrimSpace(receipt.Source)) != "google" {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	return v.Verify.Validate(ctx, receipt)
}

// StripeReceiptValidator adapts the existing webhook-derived subscription
// record to the same normalized shape as mobile receipts.
type StripeReceiptValidator struct{ Verify VerifiedReceiptFunc }

// Validate verifies a Stripe event-derived subscription.
func (v StripeReceiptValidator) Validate(ctx context.Context, receipt Receipt) (NormalizedSubscription, error) {
	if strings.ToLower(strings.TrimSpace(receipt.Source)) != "stripe" {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	return v.Verify.Validate(ctx, receipt)
}

// RegisterReceipt persists a normalized receipt result. The source/external
// pair is unique, so replaying a receipt is rejected rather than refreshing
// an existing row or silently creating a second subscription.
func (s *Service) RegisterReceipt(ctx context.Context, validators ReceiptValidators, receipt Receipt) (*EntitlementPayload, error) {
	user := s.normalizeEmail(receipt.UserIdentity)
	if user == "" || strings.TrimSpace(receipt.Token) == "" {
		return nil, ErrReceiptInvalid
	}
	normalized, err := validators.Validate(ctx, receipt)
	if err != nil {
		return nil, err
	}
	if s.normalizeEmail(normalized.UserIdentity) != user {
		return nil, ErrReceiptBound
	}
	if normalized.SubscriptionID == "" || normalized.ExternalSubscription == "" {
		return nil, ErrReceiptInvalid
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO subscriptions (subscription_id, customer_email, status, source, external_subscription_id, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		ON CONFLICT DO NOTHING
	`, normalized.SubscriptionID, user, normalized.Status, strings.ToLower(receipt.Source), normalized.ExternalSubscription, normalized.PlanTier, normalized.PriceID, normalized.BundleKey)
	if err != nil {
		return nil, fmt.Errorf("persist validated receipt: %w", err)
	}
	if rows, rowsErr := result.RowsAffected(); rowsErr == nil && rows == 0 {
		return nil, ErrReceiptReplay
	}
	s.cacheMutex.Lock()
	delete(s.cache, user)
	s.cacheMutex.Unlock()
	return s.GetEntitlementsContext(ctx, user)
}
