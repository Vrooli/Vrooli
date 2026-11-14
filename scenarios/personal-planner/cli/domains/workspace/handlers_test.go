package workspace

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateProfileIntFlagRejectsNarrowingOverflow(t *testing.T) {
	// The parser used by updateCall must reject values that cannot be represented
	// by the proto int32 fields before any narrowing conversion occurs.
	_, err := parseInt32Flag("2147483648", "daily-capacity-minutes")
	require.Error(t, err)
	value, err := parseInt32Flag("1440", "daily-capacity-minutes")
	require.NoError(t, err)
	require.Equal(t, int32(1440), value)
}

func TestTestModeHTTPClientAddsRoutingHeaderWithoutMutatingRequest(t *testing.T) {
	t.Setenv(testModeEnv, "1")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "1", r.Header.Get("X-Vrooli-Test-Mode"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	request, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	response, err := (testModeHTTPClient{HTTPClient: server.Client()}).Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	require.Equal(t, http.StatusNoContent, response.StatusCode)
	require.Empty(t, request.Header.Get("X-Vrooli-Test-Mode"))
}
