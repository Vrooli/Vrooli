package strategy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateWindowThresholds(t *testing.T) {
	tests := []struct {
		name                    string
		sampleRate              int
		advanceMs               int
		windowMs                int
		maxWindowMs             int
		wantAdvance             int
		wantMinWindow           int
		wantMaxWindow           int
		wantForceRetryStepBytes int
	}{
		{
			name:       "configured advance and capped first window",
			sampleRate: 16000, advanceMs: 1000, windowMs: 2000, maxWindowMs: 10000,
			wantAdvance: 32000, wantMinWindow: 48000, wantMaxWindow: 320000, wantForceRetryStepBytes: 32000,
		},
		{
			name:       "first window below cap",
			sampleRate: 16000, advanceMs: 500, windowMs: 1000, maxWindowMs: 10000,
			wantAdvance: 16000, wantMinWindow: 32000, wantMaxWindow: 320000, wantForceRetryStepBytes: 16000,
		},
		{
			name:       "fallback advance and capped first window",
			sampleRate: 16000, advanceMs: 0, windowMs: 4000, maxWindowMs: 5000,
			wantAdvance: 64000, wantMinWindow: 48000, wantMaxWindow: 160000, wantForceRetryStepBytes: 64000,
		},
		{
			name:       "retry fallback when derived advance is nonpositive",
			sampleRate: 16000, advanceMs: -1, windowMs: 0, maxWindowMs: 0,
			wantAdvance: 0, wantMinWindow: 0, wantMaxWindow: 0, wantForceRetryStepBytes: 3200,
		},
		{
			name:       "minimum retry step when sample rate is nonpositive",
			sampleRate: 0, advanceMs: 0, windowMs: 0, maxWindowMs: 0,
			wantAdvance: 0, wantMinWindow: 0, wantMaxWindow: 0, wantForceRetryStepBytes: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateWindowThresholds(tt.sampleRate, tt.advanceMs, tt.windowMs, tt.maxWindowMs)
			require.Equal(t, windowThresholds{
				advanceBytes: tt.wantAdvance, minWindowBytes: tt.wantMinWindow,
				maxWindowBytes: tt.wantMaxWindow, forceRetryStepBytes: tt.wantForceRetryStepBytes,
			}, got)
		})
	}
}
