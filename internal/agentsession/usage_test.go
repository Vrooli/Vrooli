package agentsession

import (
	"errors"
	"testing"

	platformgo "github.com/vrooli/platform-go"
)

func TestMeasureUsageSumsMembersAndNeverReportsUnknownAsZero(t *testing.T) {
	host := &fakeHost{memory: map[int][2]int64{1: {100, 10}, 2: {200, 20}}}

	all := MeasureUsage(host, []int{1, 2}, nil)
	if all.MemoryStatus != UsageMeasured || all.MemoryBytes == nil || *all.MemoryBytes != 300 || *all.SwapBytes != 30 || all.Processes != 2 {
		t.Fatalf("measured = %+v", all)
	}

	partial := MeasureUsage(host, []int{1, 3}, nil)
	if partial.MemoryStatus != UsagePartial || partial.SwapStatus != UsagePartial || *partial.MemoryBytes != 100 || partial.Detail == "" {
		t.Fatalf("partial = %+v, want a lower bound with a reason", partial)
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
