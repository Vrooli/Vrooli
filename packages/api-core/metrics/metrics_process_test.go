//go:build unix

package metrics

import (
	"os"
	"os/exec"
	"strconv"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestCollectorReportsObservedChildPeak(t *testing.T) {
	m := Start()
	cmd := exec.Command(os.Args[0], "-test.run=TestPeakRSSIncludesChildProcesses")
	cmd.Env = append(os.Environ(), "METRICS_CHILD_ALLOC_MIB="+strconv.Itoa(256))
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	m.ObserveProcess(cmd.ProcessState)
	out := m.Stop()
	if out.GetResources().GetMemory() == commonv1.Reliability_RELIABILITY_UNAVAILABLE {
		t.Fatal("observed child memory must be available")
	}
	if out.GetResources().GetPeakRssBytes() < 128*1024*1024 {
		t.Fatalf("peak_rss_bytes = %d, want at least 128 MiB", out.GetResources().GetPeakRssBytes())
	}
}

func TestCollectorWithoutChildReportsUnavailableMemoryButCPU(t *testing.T) {
	m := Start()
	for i := 0; i < 100000; i++ {
		_ = i * i
	}
	out := m.Stop()
	if out.GetResources().GetMemory() != commonv1.Reliability_RELIABILITY_UNAVAILABLE {
		t.Fatalf("memory reliability = %v, want UNAVAILABLE", out.GetResources().GetMemory())
	}
	if out.GetResources().GetPeakRssBytes() != 0 {
		t.Fatalf("peak_rss_bytes = %d, want 0", out.GetResources().GetPeakRssBytes())
	}
	if out.GetResources().GetCpu() == commonv1.Reliability_RELIABILITY_UNAVAILABLE {
		t.Fatal("CPU should remain measurable without a child")
	}
}

func TestStageObserveProcessRollsUpToCollector(t *testing.T) {
	m := Start()
	s := m.Stage("scan")
	cmd := exec.Command(os.Args[0], "-test.run=TestPeakRSSIncludesChildProcesses")
	cmd.Env = append(os.Environ(), "METRICS_CHILD_ALLOC_MIB="+strconv.Itoa(32))
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	s.ObserveProcess(cmd.ProcessState)
	s.End()
	out := m.Stop()
	if out.GetResources().GetPeakRssBytes() == 0 || out.GetStages()[0].GetResources().GetPeakRssBytes() == 0 {
		t.Fatal("stage child peak must be reported at both stage and operation level")
	}
}

func TestConcurrentCollectorsKeepTheirObservedChildPeaks(t *testing.T) {
	a, b := Start(), Start()
	run := func(m *Collector, mib int) int64 {
		cmd := exec.Command(os.Args[0], "-test.run=TestPeakRSSIncludesChildProcesses")
		cmd.Env = append(os.Environ(), "METRICS_CHILD_ALLOC_MIB="+strconv.Itoa(mib))
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		m.ObserveProcess(cmd.ProcessState)
		return m.Stop().GetResources().GetPeakRssBytes()
	}
	aPeak, bPeak := run(a, 32), run(b, 64)
	if aPeak == 0 || bPeak == 0 || aPeak == bPeak {
		t.Fatalf("collectors did not retain their own child peaks: %d, %d", aPeak, bPeak)
	}
}

func TestResourceUsageMeasurementScopeDefaultsAndIsStamped(t *testing.T) {
	var decoded commonv1.ResourceUsage
	if got := decoded.GetMeasurementScope(); got != commonv1.MeasurementScope_MEASUREMENT_SCOPE_UNSPECIFIED {
		t.Fatalf("absent measurement scope = %v, want UNSPECIFIED", got)
	}
	if got := Start().Stop().GetResources().GetMeasurementScope(); got != commonv1.MeasurementScope_MEASUREMENT_SCOPE_OPERATION {
		t.Fatalf("new collector measurement scope = %v, want OPERATION", got)
	}
}
