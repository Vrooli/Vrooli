package bundles

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchSkeletonBundleBoundsUnresponsiveAnalyzer(t *testing.T) {
	requestCanceled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		close(requestCanceled)
	}))
	defer server.Close()
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_URL", server.URL)

	previousTimeout := skeletonFetchTimeout
	skeletonFetchTimeout = 20 * time.Millisecond
	t.Cleanup(func() { skeletonFetchTimeout = previousTimeout })

	_, err := FetchSkeletonBundle(context.Background(), "secrets-manager")
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected analyzer deadline error, got %v", err)
	}
	select {
	case <-requestCanceled:
	case <-time.After(time.Second):
		t.Fatal("analyzer request was not canceled after the deadline")
	}
}
