package integrations

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fixedConsentURL string

func (u fixedConsentURL) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return string(u), nil
}
func TestNativeConsentReader_GivenLifecycleSource_WhenReading_ThenOnlyExactReadAndNoRedirect(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "/api/v1/authorizations/id/status", r.URL.Path)
		require.Empty(t, r.Header.Get("Authorization"))
		fmt.Fprint(w, `{"authorization_id":"id","state":"pending","review_expires_at":"2030-01-01T00:00:00Z"}`)
	}))
	defer server.Close()
	reader := AgentManagerConsentReader{Resolver: fixedConsentURL(server.URL)}
	status, err := reader.ReadConsent(context.Background(), "id")
	require.NoError(t, err)
	require.Equal(t, "pending", status.State)
	require.Equal(t, 1, hits)
	reader.Resolver = fixedConsentURL("https://remote.example.test")
	_, err = reader.ReadConsent(context.Background(), "id")
	require.Error(t, err)
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, server.URL+"/api/v1/authorizations/id/status", http.StatusFound)
	}))
	defer redirect.Close()
	reader.Resolver = fixedConsentURL(redirect.URL)
	_, err = reader.ReadConsent(context.Background(), "id")
	require.Error(t, err)
	require.Equal(t, 1, hits)
}
