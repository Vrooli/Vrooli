package account_test

import (
	"context"
	"landing-page-react-vite-api/internal/account"
	"landing-page-react-vite-api/internal/plan"
	"landing-page-react-vite-api/internal/stripe"
	"landing-page-react-vite-api/internal/testutil/billingfix"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	accountH "landing-page-react-vite-api/handlers/account"
)

func TestAccountHandlerIdentityHeader(t *testing.T) {
	t.Setenv("BUNDLE_KEY", "acct_bundle")
	t.Setenv("BUNDLE_ENVIRONMENT", "production")

	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, plan.Schema, stripe.Schema)
	for _, table := range []string{"subscriptions", "credit_wallets", "bundle_prices", "bundle_products"} {
		_, err := db.Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}

	productID := billingfix.UpsertBundleProduct(t, db, "acct_bundle", "Account Bundle", "prod_acct", "production", 1_000_000, 0.01, "credits")
	billingfix.InsertBundlePrice(t, db, productID, "price_acct", "Account Plan", "studio", "month", "usd",
		9999, false, "none", 0, 0, "k", 1_000_000, 0, 1, 10, "none", "subscription",
		map[string]interface{}{"features": []string{"Gate"}})

	const email = "user@example.com"
	_, err := db.Exec(`INSERT INTO subscriptions (subscription_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,NOW(),NOW())`, "sub_acct", email, "active", "studio", "price_acct", "acct_bundle")
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO credit_wallets (customer_email, balance_credits, bonus_credits, updated_at) VALUES ($1,$2,$3,NOW())`, email, 1_000_000, 0)
	require.NoError(t, err)

	h := accountH.NewConnectHandler(accountH.Deps{Service: account.NewService(db, plan.NewService(db))})
	ctx := context.Background()

	subReq := connect.NewRequest(&landingv1.GetMySubscriptionRequest{})
	subReq.Header().Set("X-User-Email", email)
	sub, err := h.GetMySubscription(ctx, subReq)
	require.NoError(t, err)
	require.Equal(t, landingv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE, sub.Msg.Status.State)

	credReq := connect.NewRequest(&landingv1.GetMyCreditsRequest{})
	credReq.Header().Set("X-User-Email", email)
	cred, err := h.GetMyCredits(ctx, credReq)
	require.NoError(t, err)
	require.Equal(t, int64(1_000_000), cred.Msg.Balance.BalanceCredits)

	entReq := connect.NewRequest(&landingv1.GetEntitlementsRequest{})
	entReq.Header().Set("X-User-Email", email)
	ent, err := h.GetEntitlements(ctx, entReq)
	require.NoError(t, err)
	require.Equal(t, "active", ent.Msg.Status)
	require.Equal(t, "studio", ent.Msg.PlanTier)
	require.NotEmpty(t, ent.Msg.Features)
}
