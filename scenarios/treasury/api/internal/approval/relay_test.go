package approval_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"treasury/internal/approval"
)

func TestNotificationRelayForwardsOnlyTheVerifiedProducerToken(t *testing.T) {
	var received string
	var receivedTestMode string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		received = request.Header.Get("X-Agent-Identity-Token")
		receivedTestMode = request.Header.Get("X-Vrooli-Test-Mode")
		writer.Header().Set("Content-Type", "application/proto")
	}))
	defer server.Close()

	relay, err := approval.NewNotificationRelay(server.URL, server.Client())
	require.NoError(t, err)
	ctx := approval.WithProducerToken(context.Background(), "verified-run-token")
	err = relay.Relay(ctx, approval.Request{
		ID: "approval-1", AmountMinor: 110, Currency: "USD", Counterparty: "research-provider",
		CreatedAt: time.Now(),
	})
	require.NoError(t, err)
	require.Equal(t, "verified-run-token", received)
	require.Empty(t, receivedTestMode, "production relay must not enter a test database lease")

	leased, err := approval.NewLeasedTestNotificationRelay(server.URL, server.Client())
	require.NoError(t, err)
	require.NoError(t, leased.Relay(ctx, approval.Request{ID: "approval-test", AmountMinor: 110, Currency: "USD", Counterparty: "research-provider"}))
	require.Equal(t, "verified-run-token", received)
	require.Equal(t, "1", receivedTestMode)

	err = relay.Relay(context.Background(), approval.Request{ID: "approval-2", AmountMinor: 1, Currency: "USD", Counterparty: "vendor"})
	require.ErrorContains(t, err, "verified Notification Hub producer token is required")
	require.Equal(t, "verified-run-token", received, "missing context must not make an anonymous second send")
}
