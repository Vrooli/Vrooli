package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// IntroOfferService owns the durable eligibility and redemption state for
// introductory offers. Provider metadata synchronization is deliberately
// best-effort after the local transaction commits.
type IntroOfferService struct {
	store     StripeStore
	requester StripeRequester
	logf      func(string, map[string]interface{})
}

func NewIntroOfferService(store StripeStore, requester StripeRequester, logf func(string, map[string]interface{})) *IntroOfferService {
	return &IntroOfferService{store: store, requester: requester, logf: logf}
}

func (s *IntroOfferService) Eligible(ctx context.Context, email string) (bool, error) {
	email = normalizeEmail(email)
	if email == "" {
		return false, nil
	}
	if s.store == nil {
		return false, errors.New("intro offer store unavailable")
	}
	var used sql.NullBool
	err := s.store.QueryRowContext(ctx, `SELECT has_used_intro FROM users WHERE email = $1`, email).Scan(&used)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("check intro eligibility: %w", err)
	}
	return !used.Valid || !used.Bool, nil
}

func (s *IntroOfferService) MarkUsed(ctx context.Context, email, customerID, couponID, planTier, subscriptionID string) error {
	email = normalizeEmail(email)
	if email == "" {
		return errors.New("email required to mark intro used")
	}
	if s.store == nil {
		return errors.New("intro offer store unavailable")
	}
	tx, err := s.store.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `INSERT INTO users (email, has_used_intro, stripe_customer_id) VALUES ($1, TRUE, $2) ON CONFLICT (email) DO UPDATE SET has_used_intro = TRUE, stripe_customer_id = COALESCE(NULLIF($2, ''), users.stripe_customer_id), updated_at = NOW()`, email, customerID); err != nil {
		return fmt.Errorf("update user intro flag: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO intro_coupon_usage (email, stripe_customer_id, coupon_id, plan_tier, subscription_id) VALUES ($1, $2, $3, $4, $5)`, email, customerID, couponID, planTier, subscriptionID); err != nil {
		return fmt.Errorf("insert intro coupon usage: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	if customerID != "" && s.requester != nil {
		metadataCtx := context.WithoutCancel(ctx)
		go s.syncCustomerMetadata(metadataCtx, customerID, couponID)
	}
	s.log("intro_coupon_marked_used", map[string]interface{}{"level": "info", "email": email, "customer_id": customerID, "coupon_id": couponID, "plan_tier": planTier, "subscription_id": subscriptionID})
	return nil
}

func (s *IntroOfferService) syncCustomerMetadata(ctx context.Context, customerID, couponID string) {
	values := url.Values{"metadata[has_used_intro]": {"true"}, "metadata[intro_coupon_id]": {couponID}}
	_, err := s.requester.Request(ctx, http.MethodPost, "/v1/customers/"+url.PathEscape(customerID), strings.NewReader(values.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		s.log("stripe_customer_metadata_update_failed", map[string]interface{}{"customer_id": customerID, "error": err.Error()})
	}
}

func (s *IntroOfferService) log(event string, fields map[string]interface{}) {
	if s.logf != nil {
		s.logf(event, fields)
	}
}
