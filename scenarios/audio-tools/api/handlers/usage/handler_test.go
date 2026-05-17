package usage_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	usageH "audio-tools/handlers/usage"
	localdb "audio-tools/internal/database"
	"audio-tools/internal/store"
	"audio-tools/internal/testutil/db"

	usagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage"
	usageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage/usage_v1connect"
)

func newServer(t *testing.T) (usageconnect.UsageServiceClient, *store.UsageStore) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	us := store.NewUsageStore(d)
	mod := usageH.Module(usageH.Deps{Store: us})
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return usageconnect.NewUsageServiceClient(http.DefaultClient, srv.URL), us
}

func TestUsageHandler_ListRecent_MapsRowsToProto(t *testing.T) {
	c, us := newServer(t)
	ctx := context.Background()
	now := time.Now().UTC()
	require.NoError(t, us.Insert(ctx, store.UsageRow{
		OperationID:  "op-1",
		EmittedAt:    now.Add(-1 * time.Minute),
		Capability:   "stt",
		Operation:    "transcribe",
		ProviderTier: "local",
		ProviderID:   "whisper",
		ModelID:      "base",
		LatencyMs:    123,
	}))

	res, err := c.ListRecent(ctx, connect.NewRequest(&usagev1.ListRecentRequest{Limit: 10}))
	require.NoError(t, err)
	rows := res.Msg.GetRows()
	require.Len(t, rows, 1)
	require.Equal(t, "op-1", rows[0].GetOperationId())
	require.Equal(t, "stt", rows[0].GetCapability())
	require.Equal(t, "whisper", rows[0].GetProviderId())
	require.Equal(t, 123.0, rows[0].GetLatencyMs())
}

func TestUsageHandler_GetSummary_DistributionAndTotals(t *testing.T) {
	c, us := newServer(t)
	ctx := context.Background()
	now := time.Now().UTC()
	rows := []store.UsageRow{
		{OperationID: "a", EmittedAt: now.Add(-time.Minute), Capability: "stt", Operation: "transcribe", ProviderTier: "local", ProviderID: "whisper", LatencyMs: 100, CreditsCharged: 0},
		{OperationID: "b", EmittedAt: now.Add(-time.Minute), Capability: "stt", Operation: "transcribe", ProviderTier: "vrooli", ProviderID: "lpbs", LatencyMs: 50, CreditsCharged: 2},
		{OperationID: "c", EmittedAt: now.Add(-time.Minute), Capability: "tts", Operation: "synthesize", ProviderTier: "local", ProviderID: "kokoro", LatencyMs: 200, CreditsCharged: 0, FallbackReason: "vrooli_unavailable"},
	}
	for _, r := range rows {
		require.NoError(t, us.Insert(ctx, r))
	}

	res, err := c.GetSummary(ctx, connect.NewRequest(&usagev1.GetSummaryRequest{}))
	require.NoError(t, err)
	sum := res.Msg.GetSummary()
	require.EqualValues(t, 3, sum.GetOperationsTotal())
	require.EqualValues(t, 2, sum.GetCreditsTotal())
	require.NotEmpty(t, sum.GetDistribution())
	require.NotEmpty(t, sum.GetFallbackReasons())
}

func TestUsageHandler_NoStoreReturnsFailedPrecondition(t *testing.T) {
	mod := usageH.Module(usageH.Deps{})
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	c := usageconnect.NewUsageServiceClient(http.DefaultClient, srv.URL)

	_, err := c.ListRecent(context.Background(), connect.NewRequest(&usagev1.ListRecentRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}
