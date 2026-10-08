package performance

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCollectorWindowDoesNotIncludeEvictedCounts(t *testing.T) {
	c := NewCollector("window", 30, 2)
	for i := 0; i < 4; i++ {
		c.Record(&FrameTimings{Skipped: true})
	}
	for i := 0; i < 2; i++ {
		c.Record(&FrameTimings{FrameBytes: 1000, DriverCaptureMs: 10, DriverTotalMs: 15})
	}
	stats := c.GetAggregated()
	require.Equal(t, 2, stats.FrameCount)
	require.Zero(t, stats.SkippedCount)
	require.Equal(t, 1000, stats.AvgFrameBytes)
	require.Equal(t, 6, c.GetFrameCount())
}

func TestCollectorPositiveCapacityAndLifetimeCadence(t *testing.T) {
	for _, capacity := range []int{-1, 0, 1, 2} {
		t.Run(fmt.Sprint(capacity), func(t *testing.T) {
			require.NotPanics(t, func() {
				c := NewCollector("bounded", 30, capacity)
				for i := 1; i <= 60; i++ {
					c.Record(&FrameTimings{SequenceNum: i})
				}
				require.Equal(t, 60, c.GetFrameCount())
				require.True(t, c.ShouldBroadcast())
				require.Equal(t, max(1, capacity), c.GetAggregated().FrameCount)
				recent := c.GetRecentFrames(1)
				require.Equal(t, 60, recent[0].SequenceNum)
				recent[0].SequenceNum = -1
				require.Equal(t, 60, c.GetRecentFrames(1)[0].SequenceNum)
			})
		})
	}
}

func TestCollectorDoesNotInferNetworkFromProcessing(t *testing.T) {
	c := NewCollector("processing", 30, 2)
	c.Record(&FrameTimings{DriverCaptureMs: 1, APITotalMs: 300})
	stats := c.GetAggregated()
	require.NotEqual(t, BottleneckNetwork, stats.PrimaryBottleneck)
	require.Contains(t, stats.BottleneckDescription, "Processing")
}

func TestCollectorObservationIntervalAndReset(t *testing.T) {
	now := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	c := NewCollector("clock", 30, 2)
	c.now = func() time.Time { return now }
	c.lastRecorded = now
	require.Zero(t, c.GetAggregated().FrameCount)
	for i := 0; i < 4; i++ {
		now = now.Add(time.Second)
		c.Record(&FrameTimings{Skipped: true})
	}
	for i := 0; i < 2; i++ {
		now = now.Add(time.Second)
		c.Record(&FrameTimings{FrameBytes: 1000, Timestamp: now.Add(-24 * time.Hour)})
	}
	stats := c.GetAggregated()
	require.Equal(t, int64(2000), stats.WindowDurationMs)
	require.Equal(t, now.Add(-2*time.Second), stats.WindowStartTime)
	require.Equal(t, float64(1), stats.ActualFps)
	require.Equal(t, int64(1000), stats.BandwidthBytesPerSec)
	now = now.Add(2 * time.Second)
	require.Equal(t, float64(.5), c.GetAggregated().ActualFps)
	require.Equal(t, int64(500), c.GetAggregated().BandwidthBytesPerSec)
	c.Reset()
	require.Empty(t, c.GetRecentFrames(0))
	require.Zero(t, c.GetFrameCount())
	require.False(t, c.ShouldBroadcast())
	require.Zero(t, c.GetAggregated().WindowDurationMs)
	now = now.Add(time.Second)
	c.Record(&FrameTimings{FrameBytes: 500})
	require.Equal(t, float64(1), c.GetAggregated().ActualFps)
	require.Equal(t, int64(500), c.GetAggregated().BandwidthBytesPerSec)
}

func TestCollectorPercentilesUseTheirObservedCohort(t *testing.T) {
	c := NewCollector("cohort", 30, 3)
	c.Record(&FrameTimings{Skipped: true, DriverCaptureMs: 100, DriverTotalMs: 1000})
	c.Record(&FrameTimings{FrameBytes: 1000, DriverCaptureMs: 10, DriverTotalMs: 15, APITotalMs: 5})
	c.Record(&FrameTimings{FrameBytes: 3000, DriverCaptureMs: 20, DriverTotalMs: 25, APITotalMs: 5})
	stats := c.GetAggregated()
	require.Equal(t, 3, stats.FrameCount)
	require.Equal(t, 1, stats.SkippedCount)
	require.Equal(t, 2000, stats.AvgFrameBytes)
	require.Equal(t, float64(20), stats.CaptureP50Ms)
	require.Equal(t, float64(25), stats.E2EP50Ms)
	require.Equal(t, float64(29), stats.E2EP90Ms)
	require.Equal(t, float64(29.9), stats.E2EP99Ms)
	require.Equal(t, float64(30), stats.E2EMaxMs)
}

func TestCollectorConcurrentReadWrite(t *testing.T) {
	c := NewCollector("concurrent", 30, 32)
	var wg sync.WaitGroup
	for writer := 0; writer < 4; writer++ {
		wg.Go(func() {
			for i := 0; i < 250; i++ {
				c.Record(&FrameTimings{FrameBytes: 1000})
				c.GetAggregated()
				c.GetRecentFrames(3)
			}
		})
	}
	wg.Wait()
	require.Equal(t, 1000, c.GetFrameCount())
	require.Equal(t, 32, c.GetAggregated().FrameCount)
	require.Equal(t, 1000, c.GetAggregated().AvgFrameBytes)
}

func BenchmarkCollectorRecord(b *testing.B) {
	c := NewCollector("benchmark", 30, 300)
	frame := FrameTimings{FrameBytes: 1000, DriverCaptureMs: 10}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Record(&frame)
	}
}
