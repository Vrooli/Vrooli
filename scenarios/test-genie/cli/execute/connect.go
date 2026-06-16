package execute

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	runs_v1connect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// isStdoutTTY reports whether stdout is an interactive terminal. Heartbeat
// keep-alives are kept for a TTY and suppressed otherwise (mirror of
// report/color.go's detection).
func isStdoutTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

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

// autoBackgroundOnUnknownETA reports whether a run with an UNKNOWN ETA should be
// auto-backgrounded (treated as potentially long) rather than followed inline.
// Default on, so an unestimatable first run never blocks an agent's tool past
// its timeout. TEST_GENIE_AUTOBACKGROUND_ON_UNKNOWN_ETA=0/false disables it
// (unknown-ETA runs then follow inline). It is moot when auto-backgrounding is
// disabled entirely (TEST_GENIE_AUTOBACKGROUND_SECONDS=0).
func autoBackgroundOnUnknownETA() bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("TEST_GENIE_AUTOBACKGROUND_ON_UNKNOWN_ETA")))
	return raw != "0" && raw != "false"
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
// run that is still executing. It names `runs wait --json` — the quiet,
// single-return verb that blocks server-side and returns ONCE with the verdict +
// exit code (and, on --timeout, a backoff hint) — rather than `runs follow`
// (a continuous, heartbeating stream). A backgrounded stream re-wakes an agent on
// every heartbeat and produces "still waiting…" spam; a single quiet wait does
// not. Use followCommand for the human/TUI live-watch verb.
func reattachCommand(scenario, runID string) string {
	return "test-genie runs wait --json " + scenario + " " + runID
}

// followCommand is the human/TUI live-watch verb: a continuous event stream with
// progress lines and heartbeats. Prefer reattachCommand for agents and scripts.
func followCommand(scenario, runID string) string {
	return "test-genie runs follow " + scenario + " " + runID
}

// humanDuration renders a second count as a compact human string (e.g. "2m30s").
func humanDuration(seconds int) string {
	if seconds <= 0 {
		return "unknown"
	}
	return (time.Duration(seconds) * time.Second).String()
}
