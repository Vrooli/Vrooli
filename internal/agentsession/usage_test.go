package agentsession

import (
	"errors"
	"testing"

	platformgo "github.com/vrooli/platform-go"
)

func TestMeasureScopeUsagePrefersCgroupChargedBytes(t *testing.T) {
	host := &fakeHost{
		memory: map[int][2]int64{1: {100, 10}},
		occupancy: map[string]platformgo.Occupancy{
			"vrooli-agent-one.scope": {MemoryBytes: 900, SwapBytes: 250},
		},
	}
	got := MeasureScopeUsage(host, "vrooli-agent-one.scope", []int{1}, nil)
	if got.MemoryStatus != UsageMeasured || *got.MemoryBytes != 900 || got.MemorySource != UsageSourceCgroupV2 {
		t.Fatalf("cgroup memory was not preferred: %+v", got)
	}
	if got.SwapStatus != UsageMeasured || *got.SwapBytes != 250 || got.SwapSource != UsageSourceCgroupV2 {
		t.Fatalf("cgroup swap was not preferred: %+v", got)
	}
}

func TestMeasureScopeUsageFallsBackPerFieldWithoutCallingUnknownZero(t *testing.T) {
	host := &fakeHost{
		memory: map[int][2]int64{1: {100, 10}},
		occupancy: map[string]platformgo.Occupancy{
			"vrooli-agent-one.scope": {MemoryBytes: platformgo.Unknown, SwapBytes: 0},
		},
	}
	got := MeasureScopeUsage(host, "vrooli-agent-one.scope", []int{1}, nil)
	if got.MemoryBytes == nil || *got.MemoryBytes != 100 || got.MemorySource != UsageSourceProcSum {
		t.Fatalf("unavailable cgroup memory did not use labeled process fallback: %+v", got)
	}
	if got.SwapBytes == nil || *got.SwapBytes != 0 || got.SwapSource != UsageSourceCgroupV2 {
		t.Fatalf("measured cgroup zero swap was not preserved: %+v", got)
	}
}

func TestMeasureScopeUsageRetainsMeasuredCgroupWhenMemberReadFails(t *testing.T) {
	host := &fakeHost{
		occupancy: map[string]platformgo.Occupancy{
			"vrooli-agent-one.scope": {MemoryBytes: 900, SwapBytes: 250},
		},
	}
	got := MeasureScopeUsage(host, "vrooli-agent-one.scope", nil, errors.New("cgroup.procs unreadable"))
	if got.MemoryBytes == nil || *got.MemoryBytes != 900 || got.MemorySource != UsageSourceCgroupV2 ||
		got.SwapBytes == nil || *got.SwapBytes != 250 || got.SwapSource != UsageSourceCgroupV2 {
		t.Fatalf("cgroup usage should remain measurable when PID listing fails: %+v", got)
	}
}

func TestMeasureUsageSumsMembersAndNeverReportsUnknownAsZero(t *testing.T) {
	host := &fakeHost{memory: map[int][2]int64{1: {100, 10}, 2: {200, 20}}}

	all := MeasureUsage(host, []int{1, 2}, nil)
	if all.MemoryStatus != UsageMeasured || all.MemoryBytes == nil || *all.MemoryBytes != 300 || *all.SwapBytes != 30 || all.Processes != 2 {
		t.Fatalf("measured = %+v", all)
	}

	partial := MeasureUsage(host, []int{1, 3}, nil)
	if partial.MemoryStatus != UsagePartial || partial.SwapStatus != UsagePartial || *partial.MemoryBytes != 100 || partial.Detail == "" {
		t.Fatalf("partial = %+v, want an incomplete process sum with a reason", partial)
	}

	for name, got := range map[string]ScopeUsage{
		"no members":       MeasureUsage(host, nil, nil),
		"none readable":    MeasureUsage(host, []int{7, 8}, nil),
		"scope unreadable": MeasureUsage(host, nil, errors.New("cgroup gone")),
		"no host":          MeasureUsage(nil, []int{1}, nil),
	} {
		if got.MemoryBytes != nil || got.SwapBytes != nil || got.MemoryStatus != UsageUnavailable || got.Detail == "" {
			t.Errorf("%s = %+v, want null figures, status unavailable and a reason", name, got)
		}
	}

	unsupported := MeasureUsage(host, nil, platformgo.ErrUnsupported)
	if unsupported.MemoryBytes != nil || unsupported.MemoryStatus != UsageUnsupported || unsupported.SwapStatus != UsageUnsupported {
		t.Fatalf("unsupported = %+v", unsupported)
	}
}
