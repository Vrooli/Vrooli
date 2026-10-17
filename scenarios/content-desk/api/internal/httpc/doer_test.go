package httpc_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"content-desk/internal/httpc"
	"content-desk/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

// fetchBody is a tiny inline reference caller — represents the shape
// any production code would take when consuming the Doer seam. The
// test exercises both the production satisfier (*http.Client) and the
// test fake (mocks.FakeDoer) through this same function so the
// substitution contract is proven end-to-end.
func fetchBody(d httpc.Doer, url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := d.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// TestDoer_ProductionPath wires *http.Client directly against an
// httptest.Server. Proves the compile-time assertion in doer.go isn't
// a lie: callers can pass a real client wherever Doer is required.
func TestDoer_ProductionPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "hello from production")
	}))
	t.Cleanup(srv.Close)

	got, err := fetchBody(http.DefaultClient, srv.URL)
	require.NoError(t, err)
	require.Equal(t, "hello from production", got)
}

// TestDoer_TestPath substitutes the fake. Same caller, different seam
// implementation — the substitution is the load-bearing contract; the
// test asserts both that the body round-trips and that the fake
// recorded the inbound request for after-the-fact assertions.
func TestDoer_TestPath(t *testing.T) {
	fake := &mocks.FakeDoer{}
	fake.AddResponse(http.StatusOK, []byte("hello from fake"))

	got, err := fetchBody(fake, "https://example.invalid/path")
	require.NoError(t, err)
	require.Equal(t, "hello from fake", got)

	require.Equal(t, int64(1), fake.Calls.Load())
	require.Len(t, fake.Requests, 1)
	require.Equal(t, http.MethodGet, fake.Requests[0].Method)
	require.True(t, strings.HasSuffix(fake.Requests[0].URL.Path, "/path"))
}
