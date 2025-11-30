package main

import (
	"database/sql"
	"strings"
	"time"
)

// AccountService exposes subscription, credits, and entitlement helpers.
type AccountService struct {
	db          *sql.DB
	planService *PlanService
	bundleKey   string
}

// SubscriptionInfo describes the user's plan/subscription status.
type SubscriptionInfo struct {
	Status         string    `json:"status"`
	SubscriptionID string    `json:"subscription_id,omitempty"`
	CustomerEmail  string    `json:"customer_email,omitempty"`
	PlanTier       string    `json:"plan_tier,omitempty"`
	PriceID        string    `json:"price_id,omitempty"`
	BundleKey      string    `json:"bundle_key,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CreditInfo captures wallet balances plus included credits.
type CreditInfo struct {
	CustomerEmail            string  `json:"customer_email"`
	BalanceCredits           int64   `json:"balance_credits"`
	BonusCredits             int64   `json:"bonus_credits"`
	DisplayCreditsLabel      string  `json:"display_credits_label"`
	DisplayCreditsMultiplier float64 `json:"display_credits_multiplier"`
}

// EntitlementPayload is used by bundled apps to unlock features.
type EntitlementPayload struct {
	Status       string            `json:"status"`
	PlanTier     string            `json:"plan_tier,omitempty"`
	PriceID      string            `json:"price_id,omitempty"`
	Features     []string          `json:"features,omitempty"`
	Credits      *CreditInfo       `json:"credits,omitempty"`
	Subscription *SubscriptionInfo `json:"subscription,omitempty"`
}

func NewAccountService(db *sql.DB, planService *PlanService) *AccountService {
	return &AccountService{
		db:          db,
		planService: planService,
		bundleKey:   planService.BundleKey(),
	}
}

func (s *AccountService) GetSubscription(userIdentity string) (*SubscriptionInfo, error) {
	if strings.TrimSpace(userIdentity) == "" {
		return &SubscriptionInfo{Status: "inactive"}, nil
	}

	query := `
		SELECT subscription_id, status, customer_email, plan_tier, price_id, bundle_key, updated_at
		FROM subscriptions
		WHERE (customer_email = $1 OR customer_id = $1)
		ORDER BY updated_at DESC
		LIMIT 1
	`

	row := s.db.QueryRow(query, userIdentity)
	var info SubscriptionInfo
	if err := row.Scan(
		&info.SubscriptionID,
		&info.Status,
		&info.CustomerEmail,
		&info.PlanTier,
		&info.PriceID,
		&info.BundleKey,
		&info.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return &SubscriptionInfo{Status: "inactive"}, nil
		}
		return nil, err
	}

	if info.PlanTier == "" && info.PriceID != "" {
		if plan, err := s.planService.GetPlanByPriceID(info.PriceID); err == nil {
			info.PlanTier = plan.PlanTier
		}
	}

	return &info, nil
}

func (s *AccountService) GetCredits(userIdentity string) (*CreditInfo, error) {
	if strings.TrimSpace(userIdentity) == "" {
		return &CreditInfo{
			CustomerEmail:            "",
			BalanceCredits:           0,
			BonusCredits:             0,
			DisplayCreditsLabel:      "credits",
			DisplayCreditsMultiplier: 1.0,
		}, nil
	}

	query := `
		SELECT customer_email, balance_credits, bonus_credits, updated_at
		FROM credit_wallets
		WHERE customer_email = $1
		LIMIT 1
	`

	row := s.db.QueryRow(query, userIdentity)
	var info CreditInfo
	var updatedAt time.Time
	if err := row.Scan(
		&info.CustomerEmail,
		&info.BalanceCredits,
		&info.BonusCredits,
		&updatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			// No wallet yet; return zero values
			info.CustomerEmail = userIdentity
		} else {
			return nil, err
		}
	}

	pOverview, err := s.planService.GetPricingOverview()
	if err == nil {
		info.DisplayCreditsLabel = pOverview.Bundle.DisplayCreditsLabel
		info.DisplayCreditsMultiplier = pOverview.Bundle.DisplayCreditsMultiplier
	} else {
		info.DisplayCreditsLabel = "credits"
		info.DisplayCreditsMultiplier = 1.0
	}

	return &info, nil
}

func (s *AccountService) GetEntitlements(userIdentity string) (*EntitlementPayload, error) {
	subscription, err := s.GetSubscription(userIdentity)
	if err != nil {
		return nil, err
	}

	credits, err := s.GetCredits(userIdentity)
	if err != nil {
		return nil, err
	}

	payload := &EntitlementPayload{
		Status:       subscription.Status,
		PlanTier:     subscription.PlanTier,
		PriceID:      subscription.PriceID,
		Credits:      credits,
		Subscription: subscription,
	}

	if subscription.PriceID != "" {
		if plan, err := s.planService.GetPlanByPriceID(subscription.PriceID); err == nil {
			if features, ok := plan.Metadata["features"]; ok {
				if arr, ok := features.([]interface{}); ok {
					for _, feature := range arr {
						if str, ok := feature.(string); ok {
							payload.Features = append(payload.Features, str)
						}
					}
				}
			}
		}
	}

	return payload, nil
}
