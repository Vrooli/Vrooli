package metrics

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/repo-contract-go/repocontracttest"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestStopWallClockMonotonic(t *testing.T) {
	m := Start()
	time.Sleep(5 * time.Millisecond)
	out := m.Stop()
	if out == nil {
		t.Fatal("Stop returned nil")
	}
	if out.GetWallClockMs() <= 0 {
		t.Fatalf("wall_clock_ms = %d, want > 0", out.GetWallClockMs())
	}
	if out.GetStartedAt() == nil || out.GetCompletedAt() == nil {
		t.Fatal("started_at/completed_at must be stamped")
	}
	if !out.GetCompletedAt().AsTime().After(out.GetStartedAt().AsTime()) {
		t.Fatal("completed_at must be after started_at")
	}
}

func TestFlatAndNestedStages(t *testing.T) {
	m := Start()
	a := m.Stage("discover")
	time.Sleep(3 * time.Millisecond)
	a.End()

	b := m.Stage("compile")
	b.Gauge("descriptors", 12)
	child := b.Stage("parse")
	time.Sleep(3 * time.Millisecond)
	child.End()
	b.End()
	out := m.Stop()

	if len(out.GetStages()) != 2 {
		t.Fatalf("want 2 top-level stages, got %d", len(out.GetStages()))
	}
	if out.GetStages()[0].GetName() != "discover" || out.GetStages()[1].GetName() != "compile" {
		t.Fatalf("stages out of order: %q, %q", out.GetStages()[0].GetName(), out.GetStages()[1].GetName())
	}
	for _, st := range out.GetStages() {
		if st.GetDurationMs() < 0 {
			t.Fatalf("stage %q negative duration", st.GetName())
		}
	}
	compile := out.GetStages()[1]
	if got := compile.GetGauges()["descriptors"]; got != 12 {
		t.Fatalf("per-stage gauge = %v, want 12", got)
	}
	if len(compile.GetChildren()) != 1 || compile.GetChildren()[0].GetName() != "parse" {
		t.Fatalf("nested child missing: %+v", compile.GetChildren())
	}
	if compile.GetResources() == nil {
		t.Fatal("per-stage resources should populate")
	}
}

func TestWholeOpGaugeRoundTrips(t *testing.T) {
	m := Start()
	m.Gauge("usd_cost", 0.42)
	out := m.Stop()
	if got := out.GetGauges()["usd_cost"]; got != 0.42 {
		t.Fatalf("whole-op gauge = %v, want 0.42", got)
	}
}

func TestBaselineEnvironmentAlwaysReliable(t *testing.T) {
	m := Start()
	out := m.Stop()
	env := out.GetEnvironment()
	if env == nil {
		t.Fatal("environment must always be present")
	}
	if env.GetOs() != runtime.GOOS {
		t.Fatalf("os = %q, want %q", env.GetOs(), runtime.GOOS)
	}
	if env.GetArch() != runtime.GOARCH {
		t.Fatalf("arch = %q, want %q", env.GetArch(), runtime.GOARCH)
	}
	if env.GetNumCpu() != int32(runtime.NumCPU()) {
		t.Fatalf("num_cpu = %d, want %d", env.GetNumCpu(), runtime.NumCPU())
	}
	if env.GetRuntimeVersion() == "" {
		t.Fatal("runtime_version should be set from stdlib")
	}
}

func TestWithEnvironmentOverridesButBackfills(t *testing.T) {
	m := Start(WithEnvironment(&commonv1.CaptureEnvironment{
		TotalMemBytes: 16777216000,
		Gpus:          []*commonv1.GpuInfo{{Index: 0, Name: "NVIDIA RTX 4090", Vendor: "nvidia"}},
	}))
	out := m.Stop()
	env := out.GetEnvironment()
	if env.GetTotalMemBytes() != 16777216000 {
		t.Fatalf("total_mem_bytes = %d, want 16777216000", env.GetTotalMemBytes())
	}
	if len(env.GetGpus()) != 1 {
		t.Fatalf("present GPUs not preserved: %+v", env.GetGpus())
	}
	// os/arch/num_cpu backfilled even though caller left them zero.
	if env.GetOs() != runtime.GOOS || env.GetArch() != runtime.GOARCH || env.GetNumCpu() == 0 {
		t.Fatalf("baseline env not backfilled: %+v", env)
	}
}

func TestNoGpuSamplerLeavesGpuUnavailable(t *testing.T) {
	m := Start()
	out := m.Stop()
	if got := out.GetResources().GetGpu(); got != commonv1.Reliability_RELIABILITY_UNAVAILABLE {
		t.Fatalf("gpu reliability = %v, want UNAVAILABLE", got)
	}
	if len(out.GetResources().GetGpus()) != 0 {
		t.Fatal("gpus should be empty without a sampler")
	}
}

func TestGpuSamplerPopulatesStageAndTopLevel(t *testing.T) {
	sampler := func(context.Context) []*commonv1.GpuUsage {
		return []*commonv1.GpuUsage{{
			Index: 0, Name: "Fake GPU", Vendor: "nvidia",
			UtilPercent: 11, MemUsedBytes: 734003200, MemTotalBytes: 25757220864,
			ProcessScoped: true,
		}}
	}
	m := Start(WithGpuSampler(sampler))
	st := m.Stage("infer")
	st.End()
	out := m.Stop()

	if out.GetResources().GetGpu() != commonv1.Reliability_RELIABILITY_BEST_EFFORT {
		t.Fatalf("top-level gpu reliability = %v, want BEST_EFFORT", out.GetResources().GetGpu())
	}
	if len(out.GetResources().GetGpus()) != 1 || out.GetResources().GetGpus()[0].GetName() != "Fake GPU" {
		t.Fatalf("top-level gpus not populated: %+v", out.GetResources().GetGpus())
	}
	stage := out.GetStages()[0]
	if stage.GetResources().GetGpu() != commonv1.Reliability_RELIABILITY_BEST_EFFORT {
		t.Fatalf("per-stage gpu reliability = %v, want BEST_EFFORT", stage.GetResources().GetGpu())
	}
	if len(stage.GetResources().GetGpus()) != 1 {
		t.Fatalf("per-stage gpus not populated: %+v", stage.GetResources().GetGpus())
	}
}

func TestConcurrentCollectorsBestEffort(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "rusage unavailable on this platform; reliability would be UNAVAILABLE")
	}
	a := Start()
	b := Start() // two collectors active simultaneously
	// Give each a stage so a per-stage reliability is computed under concurrency.
	a.Stage("x").End()
	b.Stage("y").End()
	var wg sync.WaitGroup
	wg.Add(2)
	var ao, bo *commonv1.ExecutionMetrics
	go func() { defer wg.Done(); ao = a.Stop() }()
	go func() { defer wg.Done(); bo = b.Stop() }()
	wg.Wait()

	if !sampleRusage().ok {
		repocontracttest.SkipPlatform(t, "rusage not sampleable on this platform")
	}
	if ao.GetResources().GetCpu() != commonv1.Reliability_RELIABILITY_BEST_EFFORT &&
		bo.GetResources().GetCpu() != commonv1.Reliability_RELIABILITY_BEST_EFFORT {
		t.Fatalf("expected at least one collector to report BEST_EFFORT cpu under concurrency; got %v / %v",
			ao.GetResources().GetCpu(), bo.GetResources().GetCpu())
	}
}

func TestStopIsIdempotent(t *testing.T) {
	m := Start()
	first := m.Stop()
	second := m.Stop()
	if first != second {
		t.Fatal("Stop should return the same result on repeated calls")
	}
}

func TestSingleFlightCpuReliabilityWhenSupported(t *testing.T) {
	m := Start()
	out := m.Stop()
	want := commonv1.Reliability_RELIABILITY_RELIABLE
	if !sampleRusage().ok {
		want = commonv1.Reliability_RELIABILITY_UNAVAILABLE
	}
	if got := out.GetResources().GetCpu(); got != want {
		t.Fatalf("single-flight cpu reliability = %v, want %v", got, want)
	}
}
