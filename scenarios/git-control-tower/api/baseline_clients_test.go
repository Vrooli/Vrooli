package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

func TestBaselineStartRequestHasDedicatedAdmissionCaller(t *testing.T) {
	request := baselineStartRequest("scenario-to-desktop")
	if got := request.Header().Get("X-Vrooli-Caller"); got != baselineAdmissionCaller {
		t.Fatalf("X-Vrooli-Caller = %q, want %q", got, baselineAdmissionCaller)
	}
	if got := request.Msg.GetCaptureProfile(); got != "baseline" {
		t.Fatalf("capture profile = %q, want baseline", got)
	}
}

func TestStartBaselineRunWithRetryRetriesResourceExhausted(t *testing.T) {
	original := baselineAdmissionRetryDelays
	baselineAdmissionRetryDelays = []time.Duration{time.Millisecond}
	t.Cleanup(func() { baselineAdmissionRetryDelays = original })
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
	original := baselineAdmissionRetryDelays
	baselineAdmissionRetryDelays = []time.Duration{time.Millisecond}
	t.Cleanup(func() { baselineAdmissionRetryDelays = original })
	attempts := 0
	_, err := startBaselineRunWithRetry(context.Background(), func() (*connect.Response[runspb.StartRunResponse], error) {
		attempts++
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("bad request"))
	})
	if err == nil || attempts != 1 {
		t.Fatalf("err=%v attempts=%d, want one failed attempt", err, attempts)
	}
}
