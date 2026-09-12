package sidecar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBackoffSchedule(t *testing.T) {
	want := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1600 * time.Millisecond,
		5 * time.Second,
		5 * time.Second,
	}
	for i, d := range want {
		require.Equalf(t, d, backoffSchedule(i), "backoff[%d]", i)
	}
	// Negative index is clamped to 0.
	require.Equal(t, 100*time.Millisecond, backoffSchedule(-3))
}

func TestRestartLedgerBudget(t *testing.T) {
	l := restartLedger{}
	now := time.Now()

	for i := 0; i < restartBudgetCount-1; i++ {
		l.record(now.Add(time.Duration(i) * time.Second))
	}
	require.False(t, l.exhausted())

	l.record(now.Add(5 * time.Second))
	require.True(t, l.exhausted())
	require.Equal(t, restartBudgetCount, l.count())
}

func TestRestartLedgerSlidingWindow(t *testing.T) {
	l := restartLedger{}
	base := time.Now()

	// Five restarts long ago — outside the window.
	for i := 0; i < 5; i++ {
		l.record(base.Add(time.Duration(i) * time.Second))
	}
	require.True(t, l.exhausted())

	// One more restart well past the window expires the old ones.
	l.record(base.Add(restartBudgetWindow + 10*time.Second))
	require.False(t, l.exhausted(), "old entries should age out of the rolling window")
	require.Equal(t, 1, l.count())
}
