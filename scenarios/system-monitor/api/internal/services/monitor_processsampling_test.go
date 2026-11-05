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
		{PID: 3, PPID: 99, Comm: "sshd", CPUPct: 5, RSSKB: 50}, // unknown, outside the global top-N union
	}}
	clk := &StubClock{current: time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)}

	svc := newSamplingService(t, sampler, repo, clk)
	svc.sampleProcesses(context.Background(), clk.Now())

	entries, err := svc.GetProcessTimeline(context.Background(), time.Minute, "", 10)
	if err != nil {
		t.Fatalf("timeline: %v", err)
	}
	// The global top-N=2 CPU/RSS union contains the two scenario processes.
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

func TestMonitorService_PersistsConfiguredOwnerOutsideGlobalTopN(t *testing.T) {
	repo := memory.NewRepository()
	sampler := &fakeSampler{samples: []procsampler.ProcessSample{
		{PID: 1, PPID: 1, Comm: "web-console-api", CPUPct: 90, RSSKB: 100},
		{PID: 2, PPID: 1, Comm: "device-control-api", CPUPct: 0.1, RSSKB: 10},
		{PID: 3, PPID: 1, Comm: "system-monitor-api", CPUPct: 80, RSSKB: 1000},
	}}
	clk := &StubClock{current: time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)}

	svc := newSamplingService(t, sampler, repo, clk)
	svc.procSampleOwners = []string{"device-control"}
	svc.sampleProcesses(context.Background(), clk.Now())

	entries, err := svc.GetProcessTimeline(context.Background(), time.Minute, "device-control", 10)
	if err != nil {
		t.Fatalf("timeline: %v", err)
	}
	if len(entries) != 1 || entries[0].Comm != "device-control-api" {
		t.Fatalf("device-control entries = %#v, want its low-ranked API process", entries)
	}
}

func TestMonitorService_ProcessTimelineInRangeUsesExplicitHalfOpenBounds(t *testing.T) {
	repo := memory.NewRepository()
	start := time.Date(2026, 11, 3, 4, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Second)
	if err := repo.SaveProcessSamples(context.Background(), []repository.ProcessSample{
		{Timestamp: start.Add(-time.Second), PID: 1, Owner: "device-control", Comm: "api", CPUPct: 90, RSSKB: 100},
		{Timestamp: start.Add(10 * time.Second), PID: 1, Owner: "device-control", Comm: "api", CPUPct: 10, RSSKB: 300},
		{Timestamp: end, PID: 1, Owner: "device-control", Comm: "api", CPUPct: 90, RSSKB: 900},
	}); err != nil {
		t.Fatalf("save samples: %v", err)
	}
	svc := newSamplingService(t, nil, repo, &StubClock{current: end.Add(time.Hour)})
	entries, err := svc.GetProcessTimelineInRange(context.Background(), start, end, "device-control", 10, "cpu")
	if err != nil {
		t.Fatalf("query explicit interval: %v", err)
	}
	if len(entries) != 1 || entries[0].SampleCount != 1 || entries[0].CPUPct != 10 || entries[0].RSSKB != 300 {
		t.Fatalf("entries = %#v, want only the in-range sample", entries)
	}
}

func TestMonitorService_ProcessTimelineInRangeRejectsUnboundedInterval(t *testing.T) {
	svc := newSamplingService(t, nil, memory.NewRepository(), &StubClock{current: time.Now()})
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	if _, err := svc.GetProcessTimelineInRange(context.Background(), start, start.Add(24*time.Hour+time.Second), "", 10, "cpu"); err == nil {
		t.Fatal("expected interval over 24 hours to be rejected")
	}
}

func TestMonitorService_ProcessSamplesInRangeReportsTruncationAndExactBounds(t *testing.T) {
	repo := memory.NewRepository()
	start := time.Date(2026, 11, 3, 4, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Second)
	if err := repo.SaveProcessSamples(context.Background(), []repository.ProcessSample{
		{Timestamp: start, PID: 1, Owner: "device-control", Comm: "api", CPUPct: 1, RSSKB: 100},
		{Timestamp: start.Add(5 * time.Second), PID: 2, Owner: "device-control", Comm: "node", CPUPct: 2, RSSKB: 200},
		{Timestamp: start.Add(10 * time.Second), PID: 3, Owner: "other", Comm: "other"},
		{Timestamp: end, PID: 4, Owner: "device-control", Comm: "outside"},
	}); err != nil {
		t.Fatalf("save samples: %v", err)
	}
	svc := newSamplingService(t, nil, repo, &StubClock{current: end.Add(time.Hour)})
	rows, truncated, err := svc.GetProcessSamplesInRange(context.Background(), start, end, "device-control", 1)
	if err != nil {
		t.Fatalf("query process samples: %v", err)
	}
	if len(rows) != 1 || rows[0].PID != 1 || !truncated {
		t.Fatalf("rows=%#v truncated=%v, want first in-range sample and explicit truncation", rows, truncated)
	}
}

func TestMonitorService_ProcessSamplesInRangeRejectsMissingOwnerAndLongWindow(t *testing.T) {
	svc := newSamplingService(t, nil, memory.NewRepository(), &StubClock{current: time.Now()})
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name  string
		end   time.Time
		owner string
		limit int
	}{
		{name: "missing owner", end: start.Add(time.Minute), limit: 10},
		{name: "long range", end: start.Add(24*time.Hour + time.Second), owner: "device-control", limit: 10},
		{name: "oversize limit", end: start.Add(time.Minute), owner: "device-control", limit: MaxProcessSampleQueryLimit + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := svc.GetProcessSamplesInRange(context.Background(), start, tc.end, tc.owner, tc.limit); err == nil {
				t.Fatal("expected invalid query to be rejected")
			}
		})
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

func TestSelectRankSamplesRetainsConfiguredOwnerOutsideGlobalRanks(t *testing.T) {
	samples := []procsampler.ProcessSample{
		{PID: 1, Owner: "web-console", CPUPct: 90, RSSKB: 100},
		{PID: 2, Owner: "system-monitor", CPUPct: 80, RSSKB: 1000},
		{PID: 3, Owner: "device-control", CPUPct: 0.2, RSSKB: 20},
		{PID: 4, Owner: "device-control", CPUPct: 0.1, RSSKB: 30},
	}

	selected, dropped := selectRankSamples(samples, 1, nil, []string{"device-control"})
	selectedPIDs := make(map[int]struct{}, len(selected))
	for _, sample := range selected {
		selectedPIDs[sample.PID] = struct{}{}
	}
	for _, pid := range []int{1, 2, 3, 4} {
		if _, ok := selectedPIDs[pid]; !ok {
			t.Errorf("PID %d was dropped from selected process samples: %#v", pid, selected)
		}
	}
	if dropped != 0 {
		t.Errorf("dropped = %d, want 0 because the configured owner leaders cover all remaining processes", dropped)
	}
}

func TestSelectRankSamplesKeepsLeadersFromPagingAndGPU(t *testing.T) {
	samples := []procsampler.ProcessSample{
		{PID: 1, CPUPct: 99, RSSKB: 10},
		{PID: 2, CPUPct: 1, RSSKB: 900},
		{PID: 3, CPUPct: 2, RSSKB: 20, MajorFaultsPerSecond: 30},
		{PID: 4, CPUPct: 3, RSSKB: 30},
		{PID: 5, CPUPct: 4, RSSKB: 40},
	}
	selected, dropped := selectRankSamples(samples, 1, map[int]float64{4: 800}, nil)
	if dropped != 1 || len(selected) != 4 {
		t.Fatalf("selected=%#v dropped=%d, want four independent leaders retained", selected, dropped)
	}
	for i, pid := range []int{1, 2, 3, 4} {
		if selected[i].PID != pid {
			t.Errorf("selected[%d].PID = %d, want %d in source order", i, selected[i].PID, pid)
		}
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
