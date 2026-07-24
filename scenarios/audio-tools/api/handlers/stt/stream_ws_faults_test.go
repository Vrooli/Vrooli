package stt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func faultRequest(value string, testMode bool) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/voice/stream", nil)
	if testMode {
		r.Header.Set(streamTestModeHeader, "1")
	}
	r.Header.Set(streamTestFaultHeader, value)
	return r
}

func TestStreamTestFaultFromRequest_RequiresBothGates(t *testing.T) {
	for _, tc := range []struct {
		name          string
		isolationActive bool
		testMode      bool
	}{
		{name: "isolation inactive", isolationActive: false, testMode: true},
		{name: "request test mode absent", isolationActive: true, testMode: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fault, err := streamTestFaultFromRequest(faultRequest("provider_busy", tc.testMode), tc.isolationActive)
			require.NoError(t, err)
			require.False(t, fault.enabled())
		})
	}
}

func TestStreamTestFaultFromRequest_ParsesSupportedDeterministicFaults(t *testing.T) {
	busy, err := streamTestFaultFromRequest(faultRequest("provider_busy", true), true)
	require.NoError(t, err)
	require.True(t, busy.providerBusy)

	closeAfter, err := streamTestFaultFromRequest(faultRequest("close_after_chunk:3", true), true)
	require.NoError(t, err)
	require.Equal(t, 3, closeAfter.closeAfterChunks)

	closeAfterCommit, err := streamTestFaultFromRequest(faultRequest("close_after_commit:2", true), true)
	require.NoError(t, err)
	require.Equal(t, 2, closeAfterCommit.closeAfterCommits)

	pause, err := streamTestFaultFromRequest(faultRequest("pause_reads_after_chunk:2:50", true), true)
	require.NoError(t, err)
	require.Equal(t, 2, pause.pauseAfterChunks)
	require.Equal(t, 50*time.Millisecond, pause.pauseReadsFor)

	delayAck, err := streamTestFaultFromRequest(faultRequest("delay_processed_ack_ms:75", true), true)
	require.NoError(t, err)
	require.Equal(t, 75*time.Millisecond, delayAck.delayProcessedAckFor)

	suppressedAck, err := streamTestFaultFromRequest(faultRequest("suppress_processed_ack", true), true)
	require.NoError(t, err)
	require.True(t, suppressedAck.suppressProcessedAck)

	_, err = streamTestFaultFromRequest(faultRequest("close_after_chunk:0", true), true)
	require.Error(t, err)
	_, err = streamTestFaultFromRequest(faultRequest("close_after_commit:0", true), true)
	require.Error(t, err)
	_, err = streamTestFaultFromRequest(faultRequest("pause_reads_after_chunk:1", true), true)
	require.Error(t, err)
	_, err = streamTestFaultFromRequest(faultRequest("delay_processed_ack_ms:0", true), true)
	require.Error(t, err)
	_, err = streamTestFaultFromRequest(faultRequest("delay_forever", true), true)
	require.Error(t, err)
}

func TestStreamTestFaultFromRequest_AcceptsBrowserQualificationQuery(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/voice/stream?test_mode=1&test_fault=provider_busy", nil)
	fault, err := streamTestFaultFromRequest(r, true)
	require.NoError(t, err)
	require.True(t, fault.providerBusy)

	disabled, err := streamTestFaultFromRequest(r, false)
	require.NoError(t, err)
	require.False(t, disabled.enabled(), "query parameters are inert without the boot-only gate")
}

func TestWaitStreamTestFaultDelay_CancelsImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, waitStreamTestFaultDelay(ctx, time.Second), context.Canceled)
}
