package tts

import (
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/clock"
	"audio-tools/internal/logx"
)

func TestNewCacheWithClock(t *testing.T) {
	c := NewCacheWithClock(1024, clock.System{})
	require.NotNil(t, c)
	c2 := NewCacheWithClock(1024, nil)
	require.NotNil(t, c2)
}

func TestSetConfigLogger(t *testing.T) {
	prev := SetConfigLogger(logx.Std{})
	t.Cleanup(func() { SetConfigLogger(prev) })
	require.NotNil(t, prev)
}
