package procsampler

import (
	"fmt"
	"syscall"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/darwinsys"
)

func TestRankPagingContributors_OrdersByFaultRateThenSwap(t *testing.T) {
	samples := []ProcessSample{
		{PID: 1, Comm: "idle", FaultStatus: FaultMeasured, SwapStatus: "measured"},
		{PID: 2, Comm: "swapped-quiet", FaultStatus: FaultMeasured, SwapStatus: "measured", SwapKB: 9_000_000},
		{PID: 3, Comm: "thrasher", FaultStatus: FaultMeasured, MajorFaultsPerSecond: 4200, SwapStatus: "measured", SwapKB: 100},
		{PID: 4, Comm: "faulting", FaultStatus: FaultMeasured, MajorFaultsPerSecond: 800, SwapStatus: "measured"},
		// A first-sample counter must not rank on its raw value.
		{PID: 5, Comm: "unsampled", FaultStatus: FaultNotYetSampled, MajorFaultsPerSecond: 99999, SwapStatus: "unsupported", SwapKB: 99999999},
	}
	got := RankPagingContributors(samples, 3)
	want := []int{3, 4, 2}
	if len(got) != len(want) {
		t.Fatalf("contributors = %v, want pids %v", pids(got), want)
	}
	for i := range want {
		if got[i].PID != want[i] {
			t.Fatalf("contributors = %v, want pids %v", pids(got), want)
		}
	}
}

func TestRankPagingContributors_IdleHostHasNone(t *testing.T) {
	samples := []ProcessSample{{PID: 1, FaultStatus: FaultMeasured, SwapStatus: "measured"}, {PID: 2, FaultStatus: FaultUnsupported, SwapStatus: "unsupported"}}
	if got := RankPagingContributors(samples, 5); len(got) != 0 {
		t.Fatalf("idle host contributors = %v, want none", pids(got))
	}
}

func pids(samples []ProcessSample) []int {
	out := make([]int, 0, len(samples))
	for _, s := range samples {
		out = append(out, s.PID)
	}
	return out
}

func TestApplyDarwinRusage_PageInsBecomeFaultCounter(t *testing.T) {
	sample := ProcessSample{PID: 42}
	applyDarwinRusage(&sample, darwinsys.RusageInfoV0{PageIns: 1234, ResidentSize: 8 << 20}, nil)
	if sample.MajorFaults != 1234 || sample.FaultStatus != FaultNotYetSampled || sample.FaultCounter != "macos proc_pid_rusage ri_pageins" {
		t.Fatalf("sample = %+v", sample)
	}
	if sample.SwapStatus != "unsupported" {
		t.Fatalf("macOS swap status = %q, want unsupported (no per-process swap accounting)", sample.SwapStatus)
	}
	if sample.RSSKB != 8192 {
		t.Fatalf("RSSKB = %d, want 8192", sample.RSSKB)
	}
}

func TestApplyDarwinRusage_PermissionDeniedIsUnsupportedNotZero(t *testing.T) {
	sample := ProcessSample{PID: 1}
	applyDarwinRusage(&sample, darwinsys.RusageInfoV0{}, fmt.Errorf("proc_pid_rusage(1): %w", syscall.EPERM))
	if sample.FaultStatus != FaultUnsupported || sample.FaultReason == "" {
		t.Fatalf("sample = %+v, want unsupported with a reason", sample)
	}
	tracker := newCPUDeltaTracker()
	now := time.Unix(100, 0)
	first := []ProcessSample{sample}
	tracker.apply(first, now)
	second := []ProcessSample{sample}
	tracker.apply(second, now.Add(10*time.Second))
	if first[0].FaultStatus != FaultUnsupported || second[0].FaultStatus != FaultUnsupported || second[0].MajorFaultsPerSecond != 0 {
		t.Fatalf("unreadable fault counter became %q/%q (%v/s) across samples", first[0].FaultStatus, second[0].FaultStatus, second[0].MajorFaultsPerSecond)
	}
}

func TestApplyWindowsMemoryCounters_MapsProcessMemoryCounters(t *testing.T) {
	sample := ProcessSample{PID: 7, FaultStatus: FaultUnsupported, SwapStatus: "unsupported"}
	applyWindowsMemoryCounters(&sample, windowsProcessCounters{UserTime100ns: 25_000_000, KernelTime100ns: 5_000_000, WorkingSetSize: 512 << 20, PagefileUsage: 300 << 20, PageFaultCount: 9000})
	if sample.SwapKB != 300*1024 || sample.SwapStatus != "measured" {
		t.Fatalf("PagefileUsage mapping: %+v", sample)
	}
	if sample.RSSKB != 512*1024 || sample.MajorFaults != 9000 || sample.FaultStatus != FaultNotYetSampled {
		t.Fatalf("working set / fault mapping: %+v", sample)
	}
	// 2.5 s user + 0.5 s kernel in 1/100 s ticks.
	if sample.utime != 250 || sample.stime != 50 {
		t.Fatalf("ticks = %d/%d, want 250/50", sample.utime, sample.stime)
	}
}

func TestDeltaTracker_FaultRateMeasuredOnSecondSample(t *testing.T) {
	tracker := newCPUDeltaTracker()
	start := time.Unix(100, 0)
	first := []ProcessSample{{PID: 9, MajorFaults: 1000}}
	tracker.apply(first, start)
	if first[0].FaultStatus != FaultNotYetSampled {
		t.Fatalf("first sample fault status = %q, want not_yet_sampled", first[0].FaultStatus)
	}
	second := []ProcessSample{{PID: 9, MajorFaults: 6000}}
	tracker.apply(second, start.Add(10*time.Second))
	if second[0].FaultStatus != FaultMeasured || second[0].MajorFaultsPerSecond != 500 {
		t.Fatalf("second sample = %+v, want measured 500/s", second[0])
	}
}
