package clock_test

import (
	"testing"
	"time"

	"scenario-completeness-scoring/internal/clock"

	"github.com/stretchr/testify/require"
)

func TestSystemNowReturnsCurrentTime(t *testing.T) {
	before := time.Now()
	got := clock.System{}.Now()
	after := time.Now()

	require.False(t, got.Before(before), "System.Now should not return a time before the call starts")
	require.False(t, got.After(after), "System.Now should not return a time after the call completes")
}

func TestSystemSatisfiesClock(t *testing.T) {
	var c clock.Clock = clock.System{}
	require.NotZero(t, c.Now())
}
