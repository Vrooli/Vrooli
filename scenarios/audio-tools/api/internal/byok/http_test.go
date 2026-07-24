package byok

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/testutil/mocks"
)

func TestDoAudioRequestReturnsBodyLatencyAndProviderErrors(t *testing.T) {
	clock := mocks.NewFakeClock(time.Unix(100, 0))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("upstream unavailable"))
			return
		}
		_, _ = w.Write([]byte("audio"))
	}))
	t.Cleanup(server.Close)

	request, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	body, latency, err := DoAudioRequest(context.Background(), http.DefaultClient, clock, "test-audio", request)
	require.NoError(t, err)
	require.Equal(t, []byte("audio"), body)
	require.Zero(t, latency)

	request, err = http.NewRequest(http.MethodGet, server.URL+"/error", nil)
	require.NoError(t, err)
	_, _, err = DoAudioRequest(context.Background(), http.DefaultClient, clock, "test-audio", request)
	require.EqualError(t, err, "test-audio: HTTP 502: upstream unavailable")
}
