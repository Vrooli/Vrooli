package metering

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// [REQ:SWBD-P1-014]
func TestLPBSGatewayUsesAuthenticatedReservationLifecycle(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		require.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/api/v1/usage/reservations":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "ai_credits", body["limit_key"])
			_, _ = w.Write([]byte(`{"reservation_id":"r-1"}`))
		default:
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}
	}))
	defer server.Close()

	g := LPBSGateway{BaseURL: server.URL, Token: "access-token", LimitKey: "ai_credits", Client: server.Client()}
	id, err := g.Reserve(context.Background(), "business_suite", 10)
	require.NoError(t, err)
	require.Equal(t, "r-1", id)
	require.NoError(t, g.Finalize(context.Background(), id, 8))
	require.NoError(t, g.Release(context.Background(), id))
	require.Equal(t, []string{
		"/api/v1/usage/reservations",
		"/api/v1/usage/reservations/r-1/finalize",
		"/api/v1/usage/reservations/r-1/release",
	}, paths)
}
