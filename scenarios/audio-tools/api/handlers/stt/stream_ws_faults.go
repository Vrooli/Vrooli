package stt

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// The fault seam is deliberately double-gated. The target must have a live
// server-owned DB-and-file isolation lease, and the individual WebSocket
// upgrade must carry an explicit test-mode signal. Browser qualification uses
// query parameters because browser JavaScript cannot attach custom headers to
// a WebSocket handshake. Ordinary deployments cannot trigger a qualification
// fault because they never install the development-only isolation lease.
const (
	streamTestModeHeader  = "X-Vrooli-Test-Mode"
	streamTestFaultHeader = "X-Audio-Tools-STT-Fault"
	streamTestModeQuery   = "test_mode"
	streamTestFaultQuery  = "test_fault"
)

type streamTestFault struct {
	providerBusy                bool
	closeAfterChunks            int
	closeAfterChunksRecoverable int
	closeAfterCommits           int
	pauseAfterChunks            int
	pauseReadsFor               time.Duration
	delayProcessedAckFor        time.Duration
	suppressProcessedAck        bool
}

func (f streamTestFault) enabled() bool {
	return f.providerBusy || f.closeAfterChunks > 0 || f.closeAfterChunksRecoverable > 0 || f.closeAfterCommits > 0 || f.pauseAfterChunks > 0 || f.delayProcessedAckFor > 0 || f.suppressProcessedAck
}

// streamTestFaultFromRequest accepts only deterministic faults with a bounded
// trigger. It is intentionally not a general fault language: new failure
// classes should be added with an explicit product-path assertion first.
func streamTestFaultFromRequest(r *http.Request, isolationActive bool) (streamTestFault, error) {
	if !isolationActive {
		return streamTestFault{}, nil
	}
	testMode := strings.TrimSpace(r.Header.Get(streamTestModeHeader)) == "1" || r.URL.Query().Get(streamTestModeQuery) == "1"
	if !testMode {
		return streamTestFault{}, nil
	}

	raw := strings.TrimSpace(r.Header.Get(streamTestFaultHeader))
	if raw == "" {
		raw = strings.TrimSpace(r.URL.Query().Get(streamTestFaultQuery))
	}
	if raw == "" {
		return streamTestFault{}, nil
	}
	if raw == "provider_busy" {
		return streamTestFault{providerBusy: true}, nil
	}
	const closeAfterPrefix = "close_after_chunk:"
	if strings.HasPrefix(raw, closeAfterPrefix) {
		count, err := strconv.Atoi(strings.TrimPrefix(raw, closeAfterPrefix))
		if err != nil || count <= 0 || count > 10_000 {
			return streamTestFault{}, fmt.Errorf("%s must be a positive chunk count no greater than 10000", streamTestFaultHeader)
		}
		return streamTestFault{closeAfterChunks: count}, nil
	}
	const recoverableCloseAfterChunkPrefix = "close_after_chunk_recoverable:"
	if strings.HasPrefix(raw, recoverableCloseAfterChunkPrefix) {
		count, err := strconv.Atoi(strings.TrimPrefix(raw, recoverableCloseAfterChunkPrefix))
		if err != nil || count <= 0 || count > 10_000 {
			return streamTestFault{}, fmt.Errorf("%s must be a positive chunk count no greater than 10000", streamTestFaultHeader)
		}
		return streamTestFault{closeAfterChunksRecoverable: count}, nil
	}
	const closeAfterCommitPrefix = "close_after_commit:"
	if strings.HasPrefix(raw, closeAfterCommitPrefix) {
		count, err := strconv.Atoi(strings.TrimPrefix(raw, closeAfterCommitPrefix))
		if err != nil || count <= 0 || count > 10_000 {
			return streamTestFault{}, fmt.Errorf("%s must be a positive commit count no greater than 10000", streamTestFaultHeader)
		}
		return streamTestFault{closeAfterCommits: count}, nil
	}
	const pauseReadsPrefix = "pause_reads_after_chunk:"
	if strings.HasPrefix(raw, pauseReadsPrefix) {
		countAndDelay := strings.Split(strings.TrimPrefix(raw, pauseReadsPrefix), ":")
		if len(countAndDelay) != 2 {
			return streamTestFault{}, fmt.Errorf("%s pause_reads_after_chunk requires <chunk>:<milliseconds>", streamTestFaultHeader)
		}
		count, countErr := strconv.Atoi(countAndDelay[0])
		delayMS, delayErr := strconv.Atoi(countAndDelay[1])
		if countErr != nil || count <= 0 || count > 10_000 || delayErr != nil || delayMS <= 0 || delayMS > 10_000 {
			return streamTestFault{}, fmt.Errorf("%s pause_reads_after_chunk requires a chunk count and milliseconds in 1..10000", streamTestFaultHeader)
		}
		return streamTestFault{pauseAfterChunks: count, pauseReadsFor: time.Duration(delayMS) * time.Millisecond}, nil
	}
	const delayAckPrefix = "delay_processed_ack_ms:"
	if strings.HasPrefix(raw, delayAckPrefix) {
		delayMS, err := strconv.Atoi(strings.TrimPrefix(raw, delayAckPrefix))
		if err != nil || delayMS <= 0 || delayMS > 10_000 {
			return streamTestFault{}, fmt.Errorf("%s delay_processed_ack_ms must be in 1..10000", streamTestFaultHeader)
		}
		return streamTestFault{delayProcessedAckFor: time.Duration(delayMS) * time.Millisecond}, nil
	}
	if raw == "suppress_processed_ack" {
		return streamTestFault{suppressProcessedAck: true}, nil
	}
	return streamTestFault{}, fmt.Errorf("unsupported %s value %q", streamTestFaultHeader, raw)
}

func streamTestFaultsAuthorized(d Deps) bool {
	return d.TestIsolationActive != nil && d.TestIsolationActive()
}

// waitStreamTestFaultDelay is cancellable so a qualification delay never turns
// a server shutdown into a leaked or hanging session.
func waitStreamTestFaultDelay(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
