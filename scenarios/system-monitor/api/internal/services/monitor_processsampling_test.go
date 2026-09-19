package services

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/infrastructure"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

// fakeSampler returns a fixed set of samples (or an error) and counts calls.
type fakeSampler struct {
	samples []procsampler.ProcessSample
	err     error
	calls   int
}

func (f *fakeSampler) Sample(context.Context) ([]procsampler.ProcessSample, error) {
	f.calls++
	return f.samples, f.err
}

func newSamplingService(t *testing.T, sampler procsampler.Sampler, repo repository.Repository, clk Clock) *MonitorService {
	t.Helper()
	cfg := &config.Config{Monitoring: config.MonitoringConfig{
		MetricsInterval:    10 * time.Second,
		ProcSampleInterval: 20 * time.Second,
		ProcSampleTopN:     2,
	}}
	return NewMonitorService(cfg, repo, infrastructure.NewStaticProvider(),
		WithMonitorClock(clk),
		WithProcessSampling(repo, sampler, procsampler.NewAttributor(nil)),
	)
}

func TestMonitorService_SamplesAttributesAndPersists(t *testing.T) {
	repo := memory.NewRepository()
	sampler := &fakeSampler{samples: []procsampler.ProcessSample{
		{PID: 1, PPID: 1, Comm: "security-health-api", CPUPct: 90, RSSKB: 100},
		{PID: 2, PPID: 1, Comm: "osv-scanner", Cwd: "/home/u/Vrooli/scenarios/security-health", CPUPct: 50, RSSKB: 200},
		{PID: 3, PPID: 99, Comm: "sshd", CPUPct: 5, RSSKB: 50}, // unknown, capped out (top-N=2)
	}}
	clk := &StubClock{current: time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)}

	svc := newSamplingService(t, sampler, repo, clk)
	svc.sampleProcesses(context.Background(), clk.Now())

	entries, err := svc.GetProcessTimeline(context.Background(), time.Minute, "", 10)
	if err != nil {
		t.Fatalf("timeline: %v", err)
	}
	// top-N=2 means only the two highest-CPU samples persisted.
	if len(entries) != 2 {
		t.Fatalf("persisted %d entries, want 2 (top-N cap)", len(entries))
	}
	// Both attribute to security-health (api binary + cwd match).
	for _, e := range entries {
		if e.Owner != "security-health" {
			t.Errorf("entry %q owner = %q, want security-health", e.Comm, e.Owner)
		}
	}
}

func TestSelectDualRankSamplesKeepsMemoryLeaderOutsideCPUTopN(t *testing.T) {
	selected, dropped := selectDualRankSamples([]procsampler.ProcessSample{
		{PID: 1, CPUPct: 90, RSSKB: 10},
		{PID: 2, CPUPct: 80, RSSKB: 20},
		{PID: 3, CPUPct: 1, RSSKB: 900},
	}, 2)
	if dropped != 0 || len(selected) != 3 {
		t.Fatalf("selected=%#v dropped=%d, want bounded CPU/RSS union of all three", selected, dropped)
	}
	if selected[2].PID != 3 {
		t.Fatalf("RSS leader was not retained: %#v", selected)
	}
}

func TestMonitorService_ProcessSampleIntervalGate(t *testing.T) {
	repo := memory.NewRepository()
	sampler := &fakeSampler{}
	clk := &StubClock{current: time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)}
	svc := newSamplingService(t, sampler, repo, clk)

	now := clk.Now()
	if !svc.shouldSampleProcesses(now) {
		t.Fatal("first sample should be allowed")
	}
	svc.sampleProcesses(context.Background(), now)

	// 10s later (< 20s interval): gated off.
	if svc.shouldSampleProcesses(now.Add(10 * time.Second)) {
		t.Fatal("sampling within the interval should be gated off")
	}
	// 25s later: allowed again.
	if !svc.shouldSampleProcesses(now.Add(25 * time.Second)) {
		t.Fatal("sampling after the interval should be allowed")
	}
}

func TestMonitorService_SamplerErrorIsNonFatal(t *testing.T) {
	repo := memory.NewRepository()
	sampler := &fakeSampler{err: procsampler.ErrUnsupported}
	clk := &StubClock{current: time.Now()}
	svc := newSamplingService(t, sampler, repo, clk)

	// Should not panic and should persist nothing.
	svc.sampleProcesses(context.Background(), clk.Now())
	entries, _ := svc.GetProcessTimeline(context.Background(), time.Minute, "", 10)
	if len(entries) != 0 {
		t.Fatalf("unsupported sampler persisted %d entries, want 0", len(entries))
	}
}
