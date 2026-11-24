package hub_test

import (
	"context"
	"testing"
	"time"

	hub "notification-hub/internal/hub"
	"notification-hub/internal/modules"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
)

func TestWorkerDispatchKeepsPinnedTestPoolForAsyncReceipt(t *testing.T) {
	db := hub.OpenTestDB(t, "primary.db")
	require.NoError(t, database.EnsureSchemas(context.Background(), db, modules.AllSchemas()...))
	service := hub.New(db, nil, nil)
	t.Cleanup(service.Close)
	require.NoError(t, db.InstallTestPool(context.Background(), hub.FileDSN(t, "worker-lease.db"), "worker-context-lease", 0))
	ctx, err := db.PinTestPool(database.WithTestMode(context.Background()), "worker-context-lease")
	require.NoError(t, err)
	require.NoError(t, database.EnsureSchemas(ctx, db, modules.AllSchemas()...))
	service.SetPushSender(&endpointFanoutSender{})
	require.NoError(t, service.RegisterPushSubscription(ctx, hub.TestRecipientSubject, hub.PushSubscription{
		ID: "worker-context-device", Endpoint: "https://push.example.test/worker-context", P256DH: "client-key", Auth: "auth-secret", Origin: "https://app.example.test",
	}))

	n, err := service.Send(ctx, hub.SendInput{
		RequestedBy: hub.TestRecipientSubject, Title: "Pinned lease", Body: "Receipt must stay with its lease",
		Urgency: "normal", SensitivityLabel: "public", IdempotencyKey: "worker-context-lease-send",
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		got, receipts, getErr := service.Get(ctx, n.ID)
		return getErr == nil && got.State == "delivered" && len(receipts) == 1
	}, asyncWait, 10*time.Millisecond, "async worker must process and receipt the notification in the pinned test pool")
	_, _, err = service.Get(context.Background(), n.ID)
	require.ErrorIs(t, err, hub.ErrNotFound, "the primary database must not receive the leased notification")
	require.Zero(t, db.LeaseStats().PrimaryDuringTestModeRequests)
}
