package execute

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	runs_v1connect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// defaultAutoBackgroundSeconds is the ETA at/above which a non-`--wait` run is
// launched in the background (the CLI returns immediately with a re-attach
// command) instead of followed inline. Tunable lever.
const (
	defaultAutoBackgroundSeconds = 60
	minAutoBackgroundSeconds     = 10
)

// autoBackgroundThreshold resolves the auto-background ETA threshold.
// TEST_GENIE_AUTOBACKGROUND_SECONDS overrides it; a value of 0 disables
// auto-backgrounding entirely (every run follows inline); other values are
// clamped up to the floor so a tiny threshold can't background near-instant
// runs.
func autoBackgroundThreshold() (seconds int, enabled bool) {
	raw := strings.TrimSpace(os.Getenv("TEST_GENIE_AUTOBACKGROUND_SECONDS"))
	if raw == "" {
		return defaultAutoBackgroundSeconds, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return defaultAutoBackgroundSeconds, true
	}
	if n == 0 {
		return 0, false
	}
	if n < minAutoBackgroundSeconds {
		n = minAutoBackgroundSeconds
	}
	return n, true
}

// newRunsClient builds a Connect client for the durable run-lifecycle RPCs. It
// uses a no-timeout HTTP client because FollowRun is a long-lived server stream
// and WaitRun may block for the full suite duration; per-call deadlines are
// applied by the callers instead.
func newRunsClient(baseURL string) runs_v1connect.RunsServiceClient {
	httpClient := &http.Client{Timeout: 0}
	return runs_v1connect.NewRunsServiceClient(httpClient, strings.TrimRight(baseURL, "/"))
}

// reattachCommand is the self-contained command an agent copies to block on a
// run that is still executing.
func reattachCommand(scenario, runID string) string {
	return "test-genie runs wait " + scenario + " " + runID
}

// humanDuration renders a second count as a compact human string (e.g. "2m30s").
func humanDuration(seconds int) string {
	if seconds <= 0 {
		return "unknown"
	}
	return (time.Duration(seconds) * time.Second).String()
}
