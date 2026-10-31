package paymentsettings_test

import (
	"context"
	"landing-page-react-vite-api/internal/paymentsettings"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpsertAndPartialUpdate(t *testing.T) {
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, paymentsettings.Schema)
	_, err := db.Exec(`DELETE FROM payment_settings`)
	require.NoError(t, err)

	svc := paymentsettings.NewService(db)
	ctx := context.Background()

	record, err := svc.SaveStripeSettings(ctx, paymentsettings.Input{
		PublishableKey: ptr("pk_live_123"),
		SecretKey:      ptr("sk_live_123"),
		WebhookSecret:  ptr("whsec_live_456"),
		DashboardURL:   ptr("https://dashboard.stripe.com/test"),
	})
	require.NoError(t, err)
	require.Equal(t, "pk_live_123", record.GetPublishableKey())

	reloaded, err := svc.GetStripeSettings(ctx)
	require.NoError(t, err)
	require.NotNil(t, reloaded)
	require.Equal(t, "sk_live_123", reloaded.GetSecretKey())

	// Partial update: only dashboard_url changes; keys are preserved.
	_, err = svc.SaveStripeSettings(ctx, paymentsettings.Input{DashboardURL: ptr("https://dashboard.stripe.com/alt")})
	require.NoError(t, err)

	final, err := svc.GetStripeSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, "https://dashboard.stripe.com/alt", final.GetDashboardUrl())
	require.Equal(t, "pk_live_123", final.GetPublishableKey())
}

func ptr(s string) *string { return &s }
