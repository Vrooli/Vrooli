package routing_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"

	"ai-gateway/internal/caller"
	"ai-gateway/internal/routing"
)

// ExecuteRoute's scenario is what the caller declares; the caller column is
// what the gateway observed, so a route event says who really sent it.
func TestRouteEvidenceStoresTheObservedCaller(t *testing.T) {
	repo := routing.NewSQLRepository(newSchemaDB(t))
	ctx := caller.WithLabel(context.Background(), "scenario:source-ledger-start-api/source-ledger-api")
	require.NoError(t, repo.Create(ctx, &routingv1.RouteEvidence{EventId: "rt-known", RequestId: "req", Scenario: "source-ledger", Status: "succeeded", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}))

	got, err := repo.Get(context.Background(), "rt-known")
	require.NoError(t, err)
	require.Equal(t, "scenario:source-ledger-start-api/source-ledger-api", got.GetCaller())
	require.Equal(t, "source-ledger", got.GetScenario())
}

// Writing the event never waits for identification: a row written while it is
// still running gets its caller as soon as it finishes.
func TestRouteEvidenceWrittenBeforeIdentificationGetsItsCallerAfterwards(t *testing.T) {
	repo := routing.NewSQLRepository(newSchemaDB(t))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	client, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	defer client.Close()
	server, err := listener.Accept()
	require.NoError(t, err)
	defer server.Close()
	ctx := context.WithValue(context.Background(), http.LocalAddrContextKey, server.LocalAddr())
	ctx = caller.Observe(ctx, server.RemoteAddr().String())

	require.NoError(t, repo.Create(ctx, &routingv1.RouteEvidence{EventId: "rt-pending", RequestId: "req", Status: "succeeded", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}))

	require.Eventually(t, func() bool {
		got, err := repo.Get(context.Background(), "rt-pending")
		// The test process holds both ends, and identification never names the
		// server itself, so the settled caller is unknown, but never empty.
		return err == nil && got.GetCaller() == caller.Unknown
	}, 5*time.Second, 10*time.Millisecond)
}

func TestRouteEvidenceWithoutAnObservationKeepsAnEmptyCaller(t *testing.T) {
	repo := routing.NewSQLRepository(newSchemaDB(t))
	require.NoError(t, repo.Create(context.Background(), &routingv1.RouteEvidence{EventId: "rt-none", RequestId: "req", Status: "succeeded", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}))
	got, err := repo.Get(context.Background(), "rt-none")
	require.NoError(t, err)
	require.Empty(t, got.GetCaller())
}
