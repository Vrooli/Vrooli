package sessions

import (
	"context"

	"github.com/vrooli/api-core/database"

	"web-console/internal/backend"
	"web-console/internal/policy"
)

// Sessions created under a routed test lease are shaped differently from an
// operator's sessions, because a test that leaks one must not leave a durable
// artifact behind.
//
// Two properties do the work:
//
//   - Standard backend, never persistent. A persistent session is a tmux pane
//     on the shared web-console tmux server, which survives API restarts by
//     design and is re-adopted by startup recovery. A leaked one is therefore
//     permanent. A standard session is a plain PTY owned by this process: it
//     cannot outlive the API, cannot be re-adopted, and never touches the
//     operator's tmux server at all.
//
//   - A short expiry, never `never`. The expiration sweeper reaps it even if
//     the case that created it crashed before teardown, so a leak is bounded
//     in minutes rather than forever.
//
// Together these make a leaked test session self-healing rather than clutter
// the operator has to clean up by hand.
const (
	// testSessionTTL is the forced expiry for test-lease sessions. Long enough
	// for the slowest BAS terminal case (the reconnect/replay case sleeps and
	// reloads the workspace), short enough that a leak is transient.
	testSessionTTL = "10m"

	// testSessionOwner marks provenance so leaked test sessions are
	// identifiable and bulk-removable without guessing from timestamps.
	testSessionOwner = "vrooli-test-lease"

	// testSessionLabel is the human-facing label shown in the sessions list.
	testSessionLabel = "test lease (auto-expires)"
)

// isTestLease reports whether this request is running under an installed
// routed test lease. It is true only for requests carrying X-Vrooli-Test-Mode
// that reached a server with TestModeMiddleware wired, so an operator's own
// traffic can never take this path.
func isTestLease(ctx context.Context) bool {
	return database.IsTestMode(ctx)
}

// applyTestLeaseShape rewrites a create request so the resulting session is
// disposable. Returns the backend and policy to use, plus the provenance to
// record. Non-test requests are returned unchanged.
func applyTestLeaseShape(ctx context.Context, bid backend.ID, pol *policy.Policy) (backend.ID, *policy.Policy) {
	if !isTestLease(ctx) {
		return bid, pol
	}
	forced := policy.Policy{Mode: policy.Custom, Duration: testSessionTTL}
	return backend.Standard, &forced
}
