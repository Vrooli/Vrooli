package main

import (
	"context"
	"errors"
	"github.com/vrooli/cli-core/cliutil"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

func TestBaselineStartRequestHasDedicatedAdmissionCaller(t *testing.T) {
	request := baselineStartRequest("scenario-to-desktop", "", 0)
	if got := request.Header().Get(cliutil.HeaderCaller); got != baselineAdmissionCaller {
		t.Fatalf("X-Vrooli-Caller = %q, want %q", got, baselineAdmissionCaller)
	}
	if got := request.Msg.GetCaptureProfile(); got != "baseline" {
		t.Fatalf("capture profile = %q, want baseline", got)
	}
}

func TestBaselineStartRequestCarriesCollectionReservation(t *testing.T) {
	request := baselineStartRequest("calendar", "gct:collection:7:agi:baseline", 12)
	if got := request.Msg.GetCollectionReservationId(); got != "gct:collection:7:agi:baseline" {
		t.Fatalf("reservation id = %q", got)
	}
	if got := request.Msg.GetCollectionReservationMemberCount(); got != 12 {
		t.Fatalf("reservation member count = %d, want 12", got)
	}
}

func TestCachedScenarioURLResolverCollapsesConcurrentDiscovery(t *testing.T) {
	var calls atomic.Int32
	resolver := &cachedScenarioURLResolver{
		ttl: time.Minute,
		resolve: func(context.Context) (string, error) {
			calls.Add(1)
			time.Sleep(10 * time.Millisecond)
			return "http://localhost:15421/", nil
		},
	}

	const workers = 8
	urls := make([]string, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			urls[i], errs[i] = resolver.Resolve(context.Background())
		}(i)
	}
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("discovery calls = %d, want one shared resolution", got)
	}
	for i := range urls {
		if errs[i] != nil || urls[i] != "http://localhost:15421" {
			t.Fatalf("worker %d got url=%q err=%v", i, urls[i], errs[i])
		}
	}
}

func TestStartBaselineRunWithRetryRetriesResourceExhausted(t *testing.T) {
	original := baselineRunRetryDelays
	baselineRunRetryDelays = []time.Duration{time.Millisecond}
	t.Cleanup(func() { baselineRunRetryDelays = original })
	attempts := 0
	response, err := startBaselineRunWithRetry(context.Background(), func() (*connect.Response[runspb.StartRunResponse], error) {
		attempts++
		if attempts == 1 {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("preview full"))
		}
		return connect.NewResponse(&runspb.StartRunResponse{RunId: "run-1"}), nil
	})
	if err != nil {
		t.Fatalf("retry start: %v", err)
	}
	if response.Msg.GetRunId() != "run-1" || attempts != 2 {
		t.Fatalf("response=%+v attempts=%d, want run-1 after two attempts", response.Msg, attempts)
	}
}

func TestStartBaselineRunWithRetryDoesNotRetryOtherErrors(t *testing.T) {
	original := baselineRunRetryDelays
	baselineRunRetryDelays = []time.Duration{time.Millisecond}
	t.Cleanup(func() { baselineRunRetryDelays = original })
	attempts := 0
	_, err := startBaselineRunWithRetry(context.Background(), func() (*connect.Response[runspb.StartRunResponse], error) {
		attempts++
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("bad request"))
	})
	if err == nil || attempts != 1 {
		t.Fatalf("err=%v attempts=%d, want one failed attempt", err, attempts)
	}
}

func TestWaitForBaselineTerminalReattachesAfterLiveSnapshot(t *testing.T) {
	attempts := 0
	response, err := waitForBaselineTerminal(context.Background(), func() (*connect.Response[runspb.WaitRunResponse], error) {
		attempts++
		if attempts == 1 {
			return connect.NewResponse(&runspb.WaitRunResponse{
				Status: &runspb.RunLiveStatus{Status: "in_progress"},
			}), nil
		}
		return connect.NewResponse(&runspb.WaitRunResponse{
			Status:                        &runspb.RunLiveStatus{Status: "passed"},
			TerminalRun:                   &runspb.RunInfo{RunId: "run-1", Status: "passed"},
			TerminalSnapshotSchemaVersion: 1,
		}), nil
	})
	if err != nil {
		t.Fatalf("wait for terminal: %v", err)
	}
	if attempts != 2 || response.Msg.GetTerminalSnapshotSchemaVersion() != 1 {
		t.Fatalf("attempts=%d response=%+v, want reattached canonical terminal response", attempts, response.Msg)
	}
}

func TestWaitForBaselineTerminalReattachesAfterUnexpectedEOF(t *testing.T) {
	original := baselineRunRetryDelays
	baselineRunRetryDelays = []time.Duration{time.Millisecond}
	t.Cleanup(func() { baselineRunRetryDelays = original })
	attempts := 0
	response, err := waitForBaselineTerminal(context.Background(), func() (*connect.Response[runspb.WaitRunResponse], error) {
		attempts++
		if attempts == 1 {
			return nil, io.ErrUnexpectedEOF
		}
		return connect.NewResponse(&runspb.WaitRunResponse{
			Status:                        &runspb.RunLiveStatus{Status: "failed"},
			TerminalRun:                   &runspb.RunInfo{RunId: "run-1", Status: "failed"},
			TerminalSnapshotSchemaVersion: 1,
		}), nil
	})
	if err != nil {
		t.Fatalf("wait after EOF: %v", err)
	}
	if attempts != 2 || response.Msg.GetTerminalSnapshotSchemaVersion() != 1 {
		t.Fatalf("attempts=%d response=%+v, want reattached terminal response", attempts, response.Msg)
	}
}

func TestWaitForCanonicalBaselineTerminalRetriesSnapshotPublicationRace(t *testing.T) {
	original := baselineRunRetryDelays
	baselineRunRetryDelays = []time.Duration{time.Millisecond}
	t.Cleanup(func() { baselineRunRetryDelays = original })

	attempts := 0
	response, err := waitForCanonicalBaselineTerminal(context.Background(), func() (*connect.Response[runspb.WaitRunResponse], error) {
		attempts++
		if attempts == 1 {
			return connect.NewResponse(&runspb.WaitRunResponse{
				Status:      &runspb.RunLiveStatus{Status: "failed"},
				TerminalRun: &runspb.RunInfo{RunId: "run-1", Status: "failed"},
			}), nil
		}
		return connect.NewResponse(&runspb.WaitRunResponse{
			Status:                        &runspb.RunLiveStatus{Status: "failed"},
			TerminalRun:                   &runspb.RunInfo{RunId: "run-1", Status: "failed"},
			TerminalSnapshotSchemaVersion: 1,
		}), nil
	})
	if err != nil {
		t.Fatalf("wait for canonical terminal evidence: %v", err)
	}
	if attempts != 2 || response.Msg.GetTerminalSnapshotSchemaVersion() != 1 {
		t.Fatalf("attempts=%d response=%+v, want one retry before canonical evidence", attempts, response.Msg)
	}
}

func TestWaitForCanonicalBaselineTerminalRejectsPersistentDegradedEvidence(t *testing.T) {
	original := baselineRunRetryDelays
	baselineRunRetryDelays = []time.Duration{time.Millisecond}
	t.Cleanup(func() { baselineRunRetryDelays = original })

	_, err := waitForCanonicalBaselineTerminal(context.Background(), func() (*connect.Response[runspb.WaitRunResponse], error) {
		return connect.NewResponse(&runspb.WaitRunResponse{
			Status:                        &runspb.RunLiveStatus{Status: "failed"},
			TerminalRun:                   &runspb.RunInfo{RunId: "run-1", Status: "failed"},
			TerminalSnapshotSchemaVersion: 1,
			DegradedReasons:               []string{"source tree changed during run"},
		}), nil
	})
	if err == nil || !strings.Contains(err.Error(), "source tree changed during run") {
		t.Fatalf("err=%v, want persistent degraded evidence rejection", err)
	}
}
