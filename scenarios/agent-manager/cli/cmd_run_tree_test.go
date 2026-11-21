package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	clitest "agent-manager/cli/internal/testutil"

	"github.com/vrooli/cli-core/cliutil"
)

const nestedRunID = "3f2b8099-9b33-41e0-9af6-ea72a29988fe"

func appForServer(t *testing.T, url string) *App {
	t.Helper()
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: url}
	}, nil)
	return &App{services: NewServices(api)}
}

// fakeWaitClock advances on sleep, so retry and timeout paths run instantly.
func fakeWaitClock(t *testing.T) {
	t.Helper()
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	saved := waitClock
	waitClock.now = func() time.Time { return now }
	waitClock.sleep = func(d time.Duration) { now = now.Add(d) }
	t.Cleanup(func() { waitClock = saved })
}

func TestRunWaitReportsTheSettledRunAndBlocksServerSide(t *testing.T) {
	server := clitest.NewRecordingServer(t, `{"run":{"id":"`+nestedRunID+`","status":"RUN_STATUS_COMPLETE"},"outcome":"terminal"}`)
	app := appForServer(t, server.URL())
	output := captureStdout(t, func() error { return app.runWait([]string{nestedRunID, "--timeout", "2m"}) })
	if !strings.Contains(output, "complete (outcome terminal)") {
		t.Fatalf("run wait output = %q", output)
	}
	requests := server.Requests()
	if len(requests) != 1 || requests[0].Path != "/agent_manager.v1.AgentManagerService/WaitRun" || !strings.Contains(requests[0].Body, `"timeout_seconds":120`) {
		t.Fatalf("run wait must make one bounded server-side wait, got %+v", requests)
	}
}

func TestRunWaitExitsFiveWhenTheTimeoutElapsesWithTheRunLive(t *testing.T) {
	fakeWaitClock(t)
	server := clitest.NewRecordingServer(t, `{"run":{"id":"`+nestedRunID+`","status":"RUN_STATUS_RUNNING"},"outcome":"timed_out"}`)
	app := appForServer(t, server.URL())
	var err error
	captureStdout(t, func() error { err = app.runWait([]string{nestedRunID, "--timeout", "3s"}); return nil })
	if exitCode(err) != exitWaitTimedOut {
		t.Fatalf("a timed-out wait must exit %d, got %v", exitWaitTimedOut, err)
	}
	if calls := len(server.Requests()); calls > 3 {
		t.Fatalf("an early timed_out answer must not become a hot loop: %d calls in a 3s wait", calls)
	}
}

func TestRunWaitExitsSixAtOnceWhenAgentManagerIsUnreachable(t *testing.T) {
	fakeWaitClock(t)
	dead := httptest.NewServer(http.NotFoundHandler())
	url := dead.URL
	dead.Close()
	app := appForServer(t, url)
	for _, args := range [][]string{{nestedRunID}, {"--children", nestedRunID}} {
		var err error
		captureStdout(t, func() error { err = app.runWait(append(args, "--json")); return nil })
		if exitCode(err) != exitAgentManagerUnreachable {
			t.Fatalf("run wait %v against a dead Agent Manager must exit %d, got %v", args, exitAgentManagerUnreachable, err)
		}
	}
}

func TestRunWaitRidesOutARestartAfterReachingAgentManager(t *testing.T) {
	fakeWaitClock(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"wake_reason":"timer","active":[{"run_id":"` + nestedRunID + `","status":"running"}],"observed_at":"2026-11-06T12:00:00Z"}`))
		case 2, 3:
			http.Error(w, `{"message":"restarting"}`, http.StatusServiceUnavailable)
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"wake_reason":"child_run_ended","ended":[{"run_id":"` + nestedRunID + `","status":"complete"}],"observed_at":"2026-11-06T12:05:00Z"}`))
		}
	}))
	t.Cleanup(server.Close)
	app := appForServer(t, server.URL)
	output := captureStdout(t, func() error {
		return app.runWait([]string{"--children", nestedRunID, "--timeout", "0", "--since", "2026-11-06T11:00:00Z"})
	})
	if calls.Load() != 4 || !strings.Contains(output, "child_run_ended") || !strings.Contains(output, "--since 2026-11-06T12:05:00Z") {
		t.Fatalf("calls=%d output=%q; want retries through the restart and the next --since", calls.Load(), output)
	}
}

func TestBoundedWaitGivesUpOnAPersistentOutage(t *testing.T) {
	fakeWaitClock(t)
	calls := 0
	_, err := boundedWait(0, func(time.Duration) (bool, error) {
		calls++
		if calls == 1 {
			return false, nil
		}
		return false, errors.New("dial tcp: connection refused")
	})
	if !agentManagerUnreachable(err) || calls < 3 {
		t.Fatalf("a persistent outage must end as unreachable after retrying: calls=%d err=%v", calls, err)
	}
	if agentManagerUnreachable(&cliutil.APIError{StatusCode: http.StatusNotFound}) {
		t.Fatal("a 404 is an answer, not an outage")
	}
}

func TestRunTreePrintsDepthAndOrphans(t *testing.T) {
	server := clitest.NewRecordingServer(t, `{"root":{"run_id":"`+nestedRunID+`","depth":0,"status":"RUN_STATUS_CANCELLED","runner":"codex","model":"gpt-5","children":[{"run_id":"child","depth":1,"status":"RUN_STATUS_RUNNING","goal_delivery":"native_verified","elapsed_seconds":90,"weighted_tokens":1200,"weighted_tokens_known":true,"orphaned":true,"owner_expires_at":"`+time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)+`"}]},"node_count":2,"max_child_depth":3}`)
	app := appForServer(t, server.URL())
	output := captureStdout(t, func() error { return app.runTree([]string{nestedRunID}) })
	for _, want := range []string{nestedRunID + "  d0", "codex/gpt-5", "  child  d1", "goal=native_verified", "elapsed=1m30s", "weighted=1200", "ORPHANED", "AUTHORITY EXPIRED", "2 run(s); max child depth 3"} {
		if !strings.Contains(output, want) {
			t.Fatalf("run tree output lacks %q:\n%s", want, output)
		}
	}
}

func TestRunListFiltersByParent(t *testing.T) {
	services, recorder := newContractServices(t)
	app := &App{services: services}
	if err := app.runList([]string{"--parent-run-id", nestedRunID, "--json"}); err != nil {
		t.Fatal(err)
	}
	if requests := recorder.Requests(); len(requests) != 1 || !strings.Contains(requests[0].Query, "parentRunId="+nestedRunID) {
		t.Fatalf("run list request = %+v, want the parent filter", requests)
	}
	if err := app.runList([]string{"--parent-run-id", "not-a-uuid"}); err == nil {
		t.Fatal("a malformed parent ID must be refused before the request")
	}
}

func TestRunTreePrintsEachRunsStepBackTriggers(t *testing.T) {
	server := clitest.NewRecordingServer(t, `{"root":{"run_id":"`+nestedRunID+`","depth":0,"status":"RUN_STATUS_RUNNING","runner":"codex","step_back":[{"id":"short-workers","mechanism":"M8","detail":"5 of 6 settled descendants ran under 6 min","remedy":"delegate by outcome"}],"children":[{"run_id":"child","depth":1,"status":"RUN_STATUS_RUNNING","step_back":[{"id":"objective-lost","mechanism":"M1","detail":"1 turn(s) kept working","remedy":"end the turn after each unit"}]}]},"node_count":2,"max_child_depth":3}`)
	app := appForServer(t, server.URL())
	output := captureStdout(t, func() error { return app.runTree([]string{nestedRunID}) })
	for _, want := range []string{
		"    STEP-BACK short-workers (M8): 5 of 6 settled descendants ran under 6 min; re-orient: delegate by outcome",
		"      STEP-BACK objective-lost (M1): 1 turn(s) kept working; re-orient: end the turn after each unit",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("run tree output lacks %q:\n%s", want, output)
		}
	}
}
