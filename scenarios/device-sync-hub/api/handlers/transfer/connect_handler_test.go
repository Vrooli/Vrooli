package transfer_test

import (
	"context"
	"testing"
	"time"

	handlertransfer "device-sync-hub/handlers/transfer"
	"device-sync-hub/internal/deviceauth"
	"device-sync-hub/internal/devices"
	"device-sync-hub/internal/testutil/db"
	internaltransfer "device-sync-hub/internal/transfer"

	"github.com/vrooli/api-core/scheduletest"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/blobstore"
	apidb "github.com/vrooli/api-core/database"

	localdb "device-sync-hub/internal/database"

	transferv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer"
)

// connectAPI is the subset of the (unexported) Connect handler the tests drive.
// Naming it as an interface lets an external test package hold the handler
// without naming its unexported concrete type (same approach as the devices
// handler test).
type connectAPI interface {
	CreateTextItem(context.Context, *connect.Request[transferv1.CreateTextItemRequest]) (*connect.Response[transferv1.CreateTextItemResponse], error)
	ListItems(context.Context, *connect.Request[transferv1.ListItemsRequest]) (*connect.Response[transferv1.ListItemsResponse], error)
	GetItem(context.Context, *connect.Request[transferv1.GetItemRequest]) (*connect.Response[transferv1.GetItemResponse], error)
	DeleteItem(context.Context, *connect.Request[transferv1.DeleteItemRequest]) (*connect.Response[transferv1.DeleteItemResponse], error)
}

func newConnect(t *testing.T) connectAPI {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internaltransfer.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC))
	svc := internaltransfer.NewService(internaltransfer.Config{
		Repo:  internaltransfer.NewSQLiteRepository(d, clk),
		Blobs: blobstore.NewMemoryBlobStore(),
		Clock: clk,
	})
	return handlertransfer.NewConnectHandler(handlertransfer.Deps{Service: svc})
}

func deviceCtx(ownerID, deviceID string) context.Context {
	return deviceauth.WithDevice(context.Background(), devices.Device{
		ID: deviceID, OwnerID: ownerID, TrustState: devices.TrustTrusted,
	})
}

func TestCreateText_RequiresDeviceToken(t *testing.T) {
	h := newConnect(t)
	_, err := h.CreateTextItem(context.Background(), connect.NewRequest(&transferv1.CreateTextItemRequest{Text: "hi"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestCreateListGetDeleteRoundTrip(t *testing.T) {
	h := newConnect(t)
	ctx := deviceCtx("owner-1", "dev-a")

	created, err := h.CreateTextItem(ctx, connect.NewRequest(&transferv1.CreateTextItemRequest{
		Text: "hello world", Name: "greeting", Retention: transferv1.Retention_RETENTION_PINNED,
	}))
	require.NoError(t, err)
	id := created.Msg.Item.Id
	require.NotEmpty(t, id)
	assert.Equal(t, transferv1.ItemKind_ITEM_KIND_TEXT, created.Msg.Item.Kind)
	assert.Equal(t, "hello world", created.Msg.Item.Text)

	list, err := h.ListItems(ctx, connect.NewRequest(&transferv1.ListItemsRequest{}))
	require.NoError(t, err)
	require.Len(t, list.Msg.Items, 1)

	got, err := h.GetItem(ctx, connect.NewRequest(&transferv1.GetItemRequest{Id: id}))
	require.NoError(t, err)
	assert.Equal(t, "greeting", got.Msg.Item.Name)

	del, err := h.DeleteItem(ctx, connect.NewRequest(&transferv1.DeleteItemRequest{Id: id}))
	require.NoError(t, err)
	assert.Equal(t, id, del.Msg.Id)

	_, err = h.GetItem(ctx, connect.NewRequest(&transferv1.GetItemRequest{Id: id}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestDirectedItemVisibilityAcrossDevices(t *testing.T) {
	h := newConnect(t)

	// dev-a sends a broadcast and a directed-to-dev-b item.
	_, err := h.CreateTextItem(deviceCtx("o", "dev-a"), connect.NewRequest(&transferv1.CreateTextItemRequest{Text: "broadcast", Retention: transferv1.Retention_RETENTION_PINNED}))
	require.NoError(t, err)
	_, err = h.CreateTextItem(deviceCtx("o", "dev-a"), connect.NewRequest(&transferv1.CreateTextItemRequest{Text: "for-b", Retention: transferv1.Retention_RETENTION_PINNED, TargetDeviceId: "dev-b"}))
	require.NoError(t, err)

	// dev-c (third device) sees only the broadcast.
	listC, err := h.ListItems(deviceCtx("o", "dev-c"), connect.NewRequest(&transferv1.ListItemsRequest{}))
	require.NoError(t, err)
	require.Len(t, listC.Msg.Items, 1)
	assert.Equal(t, "broadcast", listC.Msg.Items[0].Text)

	// dev-b sees both.
	listB, err := h.ListItems(deviceCtx("o", "dev-b"), connect.NewRequest(&transferv1.ListItemsRequest{}))
	require.NoError(t, err)
	assert.Len(t, listB.Msg.Items, 2)
}
