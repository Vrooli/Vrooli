package fetch_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"web-search/internal/research/fetch"

	"github.com/stretchr/testify/require"
)

// TestLiveSmokeHTTPFetcher is deliberately NON-hermetic: it drives the
// production fetch stack against a real public URL.
//
// Lesson behind this test (2026-06-10 live assessment): the previous fetch
// substrate posted to a browserless route that the deployed service did not
// serve, so every production L2 fetch 404'd — yet every seam-injected test
// stayed green, because the seam tests never touched the real transport. A
// substrate-level contract can only be pinned by a non-hermetic probe.
//
// Guards: skipped in -short mode and when the network is unreachable, so
// hermetic CI stays deterministic while the scenario's integration phase
// gets the real signal.
func TestLiveSmokeHTTPFetcher(t *testing.T) {
	if testing.Short() {
		t.Skip("live smoke skipped in -short mode")
	}
	if os.Getenv("WEB_SEARCH_SKIP_LIVE_SMOKE") != "" {
		t.Skip("live smoke skipped via WEB_SEARCH_SKIP_LIVE_SMOKE")
	}
	// Offline guard: resolve the target host before committing to the test.
	if _, err := net.DefaultResolver.LookupHost(context.Background(), "example.com"); err != nil {
		t.Skipf("network unreachable, skipping live smoke: %v", err)
	}

	f := fetch.NewHTTPFetcher(20*time.Second, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	text, err := f.Fetch(ctx, "https://example.com/")
	require.NoError(t, err, "production HTTP fetch leg failed against a live stable URL")
	require.NotEmpty(t, text, "live fetch must extract non-empty readable text")
}
