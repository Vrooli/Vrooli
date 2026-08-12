package control

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type bridgeHTTPClientFunc func(*http.Request) (*http.Response, error)

func (f bridgeHTTPClientFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func TestOwnerSessionHTTPClientAddsBearerAndRetriesOnceAfterUnauthorized(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), "owner-token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("owner-session-token\n"), 0o600))
	t.Setenv("VROOLI_BRIDGE_TOKEN_FILE", tokenPath)

	requests := 0
	client := &ownerSessionHTTPClient{base: bridgeHTTPClientFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		require.Equal(t, "Bearer owner-session-token", req.Header.Get("Authorization"))
		status := http.StatusUnauthorized
		if requests == 2 {
			status = http.StatusOK
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewReader(nil)), Header: make(http.Header)}, nil
	})}
	req, err := http.NewRequest(http.MethodPost, "http://bridge.test/ListAttachedDevices", bytes.NewBufferString("{}"))
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 2, requests)
}

func TestOwnerTokenFileRejectsGroupReadableCredentials(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), "owner-token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("owner-session-token"), 0o640))
	t.Setenv("VROOLI_BRIDGE_TOKEN_FILE", tokenPath)

	_, err := (&ownerSessionHTTPClient{}).ownerToken(t.Context())
	require.ErrorContains(t, err, "must be owner-only")
}
