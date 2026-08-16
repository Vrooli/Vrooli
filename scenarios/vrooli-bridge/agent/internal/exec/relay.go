package exec

import (
	"context"
	"fmt"
	"strings"
	"time"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

const (
	DefaultRelayMaxResponseBytes uint64 = 1 << 20
	MaxRelayResponseBytes        uint64 = 8 << 20
	RelayResponseLimitReason            = "relay response exceeds byte limit"
)

// RelayExecution is the bounded result of one short-lived relay command. The
// channel layer turns it into typed response frames; this package only owns
// safe argv construction and direct process execution.
type RelayExecution struct {
	ExitCode      int
	TotalBytes    uint64
	Cancelled     bool
	LimitExceeded bool
	Reason        string
}

// ExecuteRelay runs a RelayRequest without a shell. Output is delivered to
// onData in bounded chunks. The callback may cancel ctx (the channel layer
// does this when the aggregate response limit is reached or reporting fails),
// which causes CommandRunner's context-aware process termination path to run.
func (r *Runner) ExecuteRelay(ctx context.Context, request *channelv1.RelayRequest, maxBytes uint64, onData func([]byte)) RelayExecution {
	if request == nil {
		return RelayExecution{ExitCode: rejectExitCode, Reason: "relay request is nil"}
	}
	if maxBytes == 0 {
		maxBytes = DefaultRelayMaxResponseBytes
	}
	if maxBytes > MaxRelayResponseBytes {
		maxBytes = MaxRelayResponseBytes
	}
	job := &channelv1.JobPush{
		Scenario: request.GetScenario(), Verb: request.GetCommand(),
		Args:           append([]string(nil), request.GetArgs()...),
		TimeoutSeconds: request.GetTimeoutSeconds(),
	}
	argv, err := BuildArgv(r.bin, job)
	if err != nil {
		return RelayExecution{ExitCode: rejectExitCode, Reason: err.Error()}
	}

	runCtx, stop := context.WithCancel(ctx)
	defer stop()
	if request.GetTimeoutSeconds() > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(runCtx, time.Duration(request.GetTimeoutSeconds())*time.Second)
		defer cancel()
	}
	var total uint64
	limitExceeded := false
	exitCode, runErr := r.command.Run(runCtx, argv, r.workDir, func(chunk string) {
		if limitExceeded || onData == nil {
			return
		}
		bytes := []byte(chunk)
		remaining := maxBytes - total
		if uint64(len(bytes)) > remaining {
			bytes = bytes[:remaining]
			limitExceeded = true
		}
		if len(bytes) > 0 {
			total += uint64(len(bytes))
			onData(append([]byte(nil), bytes...))
		}
		if uint64(len(chunk)) > remaining {
			limitExceeded = true
			stop()
		}
	})

	if limitExceeded {
		return RelayExecution{ExitCode: rejectExitCode, TotalBytes: total, LimitExceeded: true, Reason: RelayResponseLimitReason}
	}
	if runCtx.Err() != nil {
		return RelayExecution{ExitCode: exitCode, TotalBytes: total, Cancelled: true, Reason: runCtx.Err().Error()}
	}
	if runErr != nil && exitCode == 0 {
		exitCode = startFailureExitCode
	}
	result := RelayExecution{ExitCode: exitCode, TotalBytes: total}
	if runErr != nil {
		result.Reason = runErr.Error()
	}
	if result.ExitCode == 0 {
		return result
	}
	if strings.TrimSpace(result.Reason) == "" {
		result.Reason = fmt.Sprintf("relay command exited with code %d", result.ExitCode)
	}
	return result
}
