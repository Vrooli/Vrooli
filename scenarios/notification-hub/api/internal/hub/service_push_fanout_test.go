package hub_test

import (
	"context"
	"testing"

	hub "notification-hub/internal/hub"

	"github.com/stretchr/testify/require"
)

type endpointFanoutSender struct {
	endpoints []string
	bodies    []string
	gone      map[string]bool
}

func (s *endpointFanoutSender) Send(_ context.Context, subscription hub.PushSubscription, message hub.PushMessage) (string, error) {
	s.endpoints = append(s.endpoints, subscription.Endpoint)
	s.bodies = append(s.bodies, message.Body)
	if s.gone[subscription.Endpoint] {
		return "", retiredEndpointError{}
	}
	return "provider:" + subscription.Endpoint, nil
}

type retiredEndpointError struct{}

func (retiredEndpointError) Error() string { return "push endpoint is retired" }
func (retiredEndpointError) Gone() bool    { return true }

func TestProcess_ReEnabledSameOriginEndpointsFanOutWithRecipientAndSensitivityRules(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	const origin = "https://app.example.test"
	firstEndpoint := registerSubscription(t, service, "alice")
	secondEndpoint := registerSubscription(t, service, "alice")
	bobEndpoint := registerSubscription(t, service, "bob")

	device, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: origin, MachineID: "local"})
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE channel_addresses SET approved_labels='["public"]' WHERE device_id=? AND channel='web_push'`, device.ID)
	require.NoError(t, err)
	emailDevice, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: "office email", MachineID: "local"})
	require.NoError(t, err)
	_, err = service.UpsertChannelAddress(ctx, "alice", hub.ChannelAddress{
		DeviceID: emailDevice.ID, Channel: "email", Address: "alice@example.test",
		ApprovedLabels: []string{"sensitive"},
	})
	require.NoError(t, err)

	push := &endpointFanoutSender{gone: map[string]bool{}}
	email := &recordingEmail{}
	service.SetPushSender(push)
	service.SetEmailSender(email)
	id := insertPending(t, db, "alice", "sensitive incident detail", "sensitive")
	require.NoError(t, service.Process(ctx, id))

	require.ElementsMatch(t, []string{firstEndpoint, secondEndpoint}, push.endpoints,
		"each same-origin browser endpoint registered for the configured recipient must be targeted")
	require.NotContains(t, push.endpoints, bobEndpoint, "another recipient's endpoint must never receive Alice's notification")
	require.Equal(t, []string{"Notification available in notification-hub", "Notification available in notification-hub"}, push.bodies,
		"the push device allows only public content, so every browser receives the existing pointer-only projection")
	require.Equal(t, []string{"alice@example.test"}, email.addresses, "non-push channel addresses remain one target each")
	require.Equal(t, []string{"sensitive incident detail"}, email.bodies, "the explicitly approved email channel retains sensitive content")
}

func TestProcess_RetiredBrowserEndpointDoesNotSuppressAnotherBrowser(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	retiredEndpoint := registerSubscription(t, service, "alice")
	activeEndpoint := registerSubscription(t, service, "alice")
	push := &endpointFanoutSender{gone: map[string]bool{retiredEndpoint: true}}
	service.SetPushSender(push)
	id := insertPending(t, db, "alice", "hello", "public")

	require.NoError(t, service.Process(ctx, id))
	require.ElementsMatch(t, []string{retiredEndpoint, activeEndpoint}, push.endpoints,
		"a 404/410 for one endpoint is local to that browser; other registered endpoints must still be tried")
	notification, receipts, err := service.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "delivered", notification.State)
	require.Len(t, receipts, 1)
	var retired, active int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM push_subscriptions WHERE endpoint=?`, retiredEndpoint).Scan(&retired))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM push_subscriptions WHERE endpoint=?`, activeEndpoint).Scan(&active))
	require.Zero(t, retired, "only the endpoint rejected as gone is removed")
	require.Equal(t, 1, active, "the successful browser registration remains stored")
}

func TestProcess_FinalRetiredEndpointDoesNotReuseMirroredAddress(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	endpoint := registerSubscription(t, service, "alice")
	push := &endpointFanoutSender{gone: map[string]bool{endpoint: true}}
	service.SetPushSender(push)
	firstID := insertPending(t, db, "alice", "first", "public")
	require.NoError(t, service.Process(ctx, firstID))

	var subscriptions, mirroredAddresses int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM push_subscriptions WHERE endpoint=?`, endpoint).Scan(&subscriptions))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM channel_addresses WHERE channel='web_push' AND address=?`, endpoint).Scan(&mirroredAddresses))
	require.Zero(t, subscriptions)
	require.Equal(t, 1, mirroredAddresses, "the legacy mirrored address may remain, but must not become a target without a registered subscription")
	status, err := service.ChannelsStatus(ctx, "alice", "")
	require.NoError(t, err)
	require.Len(t, status, 1)
	require.Equal(t, "not_configured", status[0].Disposition, "status must not report a retired mirrored endpoint as ready")

	secondID := insertPending(t, db, "alice", "second", "public")
	require.NoError(t, service.Process(ctx, secondID))
	second, _, err := service.Get(ctx, secondID)
	require.NoError(t, err)
	require.Equal(t, "unroutable", second.State, "a future notice must not retry the removed endpoint through its mirror")
	require.Equal(t, []string{endpoint}, push.endpoints, "the removed endpoint is never submitted again")
}
