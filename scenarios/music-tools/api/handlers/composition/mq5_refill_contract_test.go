package composition

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	testdb "github.com/vrooli/api-core/databasetest"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/composition"
	refillconnect "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/composition/composition_v1connect"
	core "music-tools/internal/composition"
	"music-tools/internal/styles"
)

// This API-only contract verifies that Music Library events become durable,
// idempotent candidate requests. The request remains held and creates no
// composition job, so this test cannot trigger generation before MQ1.
func TestCandidateRefillConnectContractIsHeldAndCancellable(t *testing.T) {
	db := testdb.NewSQLite(t)
	_, err := db.Exec(core.Schema())
	require.NoError(t, err)
	pool, err := core.NewPoolWithStore(db)
	require.NoError(t, err)
	state := &ModuleState{Pool: pool, Styles: styles.NewStore()}
	router := mux.NewRouter()
	Module(state).Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := refillconnect.NewCandidateInventoryServiceClient(server.Client(), server.URL)

	request := &pb.CandidateRefillRequest{
		RequestId: "c7c584da7ed24820b22825f1f119c346", ReservationId: "c7c584da7ed24820b22825f1f119c346",
		Sha256: "a6ab5d3585ec7fd9bf29e3aa973509a3513de8f39d8feeaa774d94986f4f167c",
		Purpose: "launch-video", Project: "mq5-contract", Revision: "cut-1",
	}
	first, err := client.RequestRefill(context.Background(), connect.NewRequest(request))
	require.NoError(t, err)
	require.Equal(t, "held", first.Msg.State)
	require.Equal(t, "launch-instrumental", first.Msg.StyleId, "omitted style resolves through the configured default")

	retry, err := client.RequestRefill(context.Background(), connect.NewRequest(request))
	require.NoError(t, err)
	require.Equal(t, first.Msg.RequestId, retry.Msg.RequestId)
	require.Equal(t, first.Msg.State, retry.Msg.State)

	conflict := *request
	conflict.Sha256 = strings.Repeat("b", 64)
	_, err = client.RequestRefill(context.Background(), connect.NewRequest(&conflict))
	require.Error(t, err, "one durable event identity cannot be rebound to another track")

	cancelled, err := client.CancelRefill(context.Background(), connect.NewRequest(&pb.CancelCandidateRefillRequest{RequestId: request.RequestId}))
	require.NoError(t, err)
	require.Equal(t, "cancelled", cancelled.Msg.State)
	cancelRetry, err := client.CancelRefill(context.Background(), connect.NewRequest(&pb.CancelCandidateRefillRequest{RequestId: request.RequestId}))
	require.NoError(t, err)
	require.Equal(t, "cancelled", cancelRetry.Msg.State)
	available, reserved, target, _ := pool.Status("launch-instrumental")
	require.Zero(t, available)
	require.Zero(t, reserved)
	require.Zero(t, target, "recording and cancelling a refill event must not start or imply generation")
}
