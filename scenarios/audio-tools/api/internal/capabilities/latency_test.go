package capabilities_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/capabilities"
)

type fixedChecker struct {
	delay  time.Duration
	status capabilities.Status
}

func (c fixedChecker) Check(context.Context) (capabilities.Status, string) {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.status, "checked"
}

func TestLatencyRecorderRecordsLastCheck(t *testing.T) {
	recorder := capabilities.NewLatencyRecorder()
	checker := recorder.Wrap("whisper-stt", fixedChecker{delay: 5 * time.Millisecond, status: capabilities.StatusAvailable})

	status, _ := checker.Check(context.Background())
	require.Equal(t, capabilities.StatusAvailable, status)

	last, ok := recorder.Last("whisper-stt")
	require.True(t, ok)
	require.GreaterOrEqual(t, last, 5*time.Millisecond)
	require.Less(t, last, time.Second)

	_, ok = recorder.Last("kokoro-tts")
	require.False(t, ok, "an unchecked provider has no measurement")
}

func TestLatencyRecorderWrapAllPreservesInput(t *testing.T) {
	recorder := capabilities.NewLatencyRecorder()
	input := map[string]capabilities.Checker{
		"a": fixedChecker{status: capabilities.StatusAvailable},
		"b": fixedChecker{status: capabilities.StatusUnavailable},
	}
	wrapped := recorder.WrapAll(input)
	require.Len(t, wrapped, 2)
	_, stillPlain := input["a"].(fixedChecker)
	require.True(t, stillPlain, "WrapAll must not mutate the caller's map")

	_, _ = wrapped["a"].Check(context.Background())
	_, ok := recorder.Last("a")
	require.True(t, ok)
}

func TestLatencyRecorderIsConcurrentSafe(t *testing.T) {
	recorder := capabilities.NewLatencyRecorder()
	wrapped := recorder.WrapAll(map[string]capabilities.Checker{
		"a": fixedChecker{status: capabilities.StatusAvailable},
		"b": fixedChecker{status: capabilities.StatusAvailable},
	})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		for id, checker := range wrapped {
			wg.Add(1)
			go func(id string, checker capabilities.Checker) {
				defer wg.Done()
				_, _ = checker.Check(context.Background())
				_, _ = recorder.Last(id)
			}(id, checker)
		}
	}
	wg.Wait()
}
