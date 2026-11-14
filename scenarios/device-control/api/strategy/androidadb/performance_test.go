package androidadb

import (
	"device-control/strategy"
	"testing"
)

func TestPerformancePresentationFreshnessAndDroppedIntervals(t *testing.T) {
	data := "16666667\n0 10000000000 0\n0 10016666667 0\n0 10066666667 0\n0 10066666667 0\n0 9223372036854775807 0\n"
	var p strategy.AppPerformance
	parseSurfacePerformance(data, 10.1, &p)
	if p.SurfaceFrames != 3 || p.SurfaceFPS == nil || *p.SurfaceFPS < 29 || *p.SurfaceFPS > 31 || p.SlowIntervals != 1 || p.SurfaceP95MS == nil || *p.SurfaceP95MS < 49 {
		t.Fatalf("unexpected presentation metrics: %+v", p)
	}
	var stale strategy.AppPerformance
	parseSurfacePerformance(data, 20, &stale)
	if stale.SurfaceFPS != nil || stale.SurfaceReason != "presentation timestamps are stale" {
		t.Fatalf("stale data must not claim an active frame rate: %+v", stale)
	}
}

func TestPerformanceGraphicsCountersPreserveZero(t *testing.T) {
	var p strategy.AppPerformance
	parseGraphicsPerformance("Stats since: 123ns\nTotal frames rendered: 18\nJanky frames: 0 (0.0%)\nNumber Missed Vsync: 2\n", &p)
	if p.GraphicsSinceNS == nil || *p.GraphicsSinceNS != 123 || p.GraphicsFrames == nil || *p.GraphicsFrames != 18 || p.JankyFrames == nil || *p.JankyFrames != 0 || p.MissedVsync == nil || *p.MissedVsync != 2 {
		t.Fatalf("unexpected graphics counters: %+v", p)
	}
}
