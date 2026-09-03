package account

import (
	"context"
	"fmt"
	"strings"
	"time"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"landing-page-business-suite-api/internal/commerce"
)

// CommerceSource is the narrow commerce seam consumed by AccountService
// transport. Keeping this adapter beside the handler prevents the API root
// from becoming a second account domain.
type CommerceSource interface {
	GetSubscriptionContext(context.Context, string) (*shared.SubscriptionStatus, error)
	GetCreditsContext(context.Context, string) (*commerce.CreditsEnvelope, error)
	GetEntitlementsContext(context.Context, string) (*commerce.EntitlementPayload, error)
}

type commerceReader struct{ service CommerceSource }

// NewCommerceReader projects commerce DTOs into AccountService transport DTOs.
func NewCommerceReader(service CommerceSource) Reader {
	return commerceReader{service: service}
}

func (r commerceReader) GetSubscriptionContext(ctx context.Context, user string) (*shared.SubscriptionStatus, error) {
	return r.service.GetSubscriptionContext(ctx, user)
}

func (r commerceReader) GetCreditsContext(ctx context.Context, user string) (*Credits, error) {
	credits, err := r.service.GetCreditsContext(ctx, user)
	if err != nil {
		return nil, err
	}
	if credits == nil {
		return nil, fmt.Errorf("credits source returned no payload")
	}
	return &Credits{Balance: credits.Balance, DisplayCreditsLabel: credits.DisplayCreditsLabel, DisplayCreditsMultiplier: credits.DisplayCreditsMultiplier}, nil
}

func (r commerceReader) GetEntitlementsContext(ctx context.Context, user string) (*Entitlements, error) {
	entitlements, err := r.service.GetEntitlementsContext(ctx, user)
	if err != nil {
		return nil, err
	}
	if entitlements == nil {
		return nil, fmt.Errorf("entitlements source returned no payload")
	}
	return &Entitlements{Status: entitlements.Status, PlanTier: entitlements.PlanTier, PriceID: entitlements.PriceID, Features: entitlements.Features, BillingCycleStart: entitlements.BillingCycleStart, Credits: entitlements.Credits, Subscription: entitlements.Subscription}, nil
}

func (r commerceReader) GetCommercialContext(ctx context.Context, user, placement, capabilityID string) (*CommercialContext, error) {
	// Eligibility is evaluated here, on the commercial owner. Integrations is
	// intentionally allow-listed and capability-scoped; arbitrary placements
	// never receive promotional content.
	entitlements, err := r.service.GetEntitlementsContext(ctx, user)
	if err != nil {
		return nil, err
	}
	if entitlements == nil {
		return nil, fmt.Errorf("entitlements source returned no payload")
	}
	now := time.Now().UTC()
	result := &CommercialContext{
		SubscriptionStatus: entitlements.Status,
		PlanTier:           entitlements.PlanTier,
		EntitlementIDs:     append([]string(nil), entitlements.Features...),
		EvaluatedAt:        now.Format(time.RFC3339),
		GeneratedAt:        now.Format(time.RFC3339),
		StaleAfter:         now.Add(5 * time.Minute).Format(time.RFC3339),
		Source:             "landing-page-business-suite",
	}
	if entitlements.Credits != nil {
		result.CreditBalance = entitlements.Credits.GetBalanceCredits()
	}
	// A commercial message must be relevant to a requested capability and the
	// supported placement. Never emit a generic settings advertisement.
	if placement == "integrations" && capabilityID != "" && (strings.TrimSpace(entitlements.PlanTier) == "" || strings.EqualFold(entitlements.PlanTier, "free")) {
		result.Content = []*lpbsv1.CommercialContent{{
			ContentId:      "integrations-capability-account-" + capabilityID,
			Placement:      placement,
			Title:          "Connect an account",
			Description:    "Connect a provider account to enable this capability.",
			Priority:       "contextual",
			Eligible:       true,
			CtaLabel:       "Manage account",
			CtaDestination: "/account",
			ExpiresAt:      now.Add(5 * time.Minute).Format(time.RFC3339),
			Dismissible:    true,
		}}
	}
	return result, nil
}

var (
	_ Reader         = commerceReader{}
	_ CommerceSource = (*commerce.Service)(nil)
)
