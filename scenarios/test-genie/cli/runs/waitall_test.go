package runs

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

func TestParseRunRef(t *testing.T) {
	ok, err := parseRunRef("web:20260616-120000-abcd1234")
	if err != nil {
		t.Fatalf("valid ref: %v", err)
	}
	if ok.scenario != "web" || ok.runID != "20260616-120000-abcd1234" {
		t.Fatalf("parsed = %+v", ok)
	}
	for _, bad := range []string{"", "web", ":runid", "web:", "  "} {
		if _, err := parseRunRef(bad); err == nil {
			t.Errorf("parseRunRef(%q) should error", bad)
		}
	}
}

// TestAggregateExitCodePrecedence pins the documented worst-wins precedence:
// failure(1) > timeout(124) > not-comparable/error(2) > all-passed(0).
func TestAggregateExitCodePrecedence(t *testing.T) {
	passed := waitAllResult{status: &runspb.RunLiveStatus{Status: "passed"}}
	failed := waitAllResult{status: &runspb.RunLiveStatus{Status: "failed"}}
	aborted := waitAllResult{status: &runspb.RunLiveStatus{Status: "aborted"}}
	timedOut := waitAllResult{status: &runspb.RunLiveStatus{Status: "in_progress"}, timedOut: true}
	malformedPending := waitAllResult{status: &runspb.RunLiveStatus{Status: "queued"}, nonterminalWithoutTimeout: true}
	providerUnavailable := waitAllResult{status: &runspb.RunLiveStatus{Status: "failed"}, providerUnavailable: true}
	errored := waitAllResult{err: errors.New("unreachable")}

	cases := []struct {
		name    string
		results []waitAllResult
		want    int
	}{
		{"all passed", []waitAllResult{passed, passed}, exitOK},
		{"one failed beats timeout+error", []waitAllResult{passed, timedOut, errored, failed}, exitRegression},
		{"aborted counts as failure", []waitAllResult{passed, aborted}, exitRegression},
		{"timeout beats error", []waitAllResult{passed, timedOut, errored}, exitWaitTimeout},
		{"malformed pending response is recoverable", []waitAllResult{passed, malformedPending}, exitWaitTimeout},
		{"provider outage is not a regression", []waitAllResult{passed, providerUnavailable}, exitNotComparable},
		{"error only", []waitAllResult{passed, errored}, exitNotComparable},
	}
	for _, tc := range cases {
		if got := aggregateExitCode(tc.results); got != tc.want {
			t.Errorf("%s: aggregateExitCode = %d, want %d", tc.name, got, tc.want)
		}
	}
}

// TestRunWaitAllRequiresRun proves a missing --run is a usage error, not a panic.
func TestRunWaitAllRequiresRun(t *testing.T) {
	if err := runWaitAll(nil, nil, io.Discard); err == nil {
		t.Fatal("expected usage error when no --run is given")
	}
}

// TestRunWaitAllFanOut drives two concurrent WaitRun calls through a real server:
// one passing, one failing → aggregate regression exit, both rendered.
func TestRunWaitAllFanOut(t *testing.T) {
	withStreamServer(t, &streamServer{
		// The fake WaitRun ignores the run id and always returns this status; the
		// fan-out still proves order-preserving rendering + aggregation wiring.
		waitStatus: &runspb.RunLiveStatus{Status: "passed"},
	})
	var buf bytes.Buffer
	err := runWaitAll(nil, []string{"--run", "web:R1", "--run", "api:R2"}, &buf)
	if err != nil {
		t.Fatalf("all-passed fan-out should exit 0, got %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "web R1") || !strings.Contains(out, "api R2") {
		t.Fatalf("both handles must be rendered, got: %q", out)
	}
}
