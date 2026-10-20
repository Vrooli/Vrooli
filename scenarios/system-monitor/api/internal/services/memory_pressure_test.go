package services

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/infrastructure"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

// fixedPressureCollector stands in for the OS-backed pressure collector.
type fixedPressureCollector struct {
	collectors.BaseCollector
	level string
}

func (c *fixedPressureCollector) Collect(context.Context) (*collectors.MetricData, error) {
	reading := collectors.MemoryPressureReading{
		Level: c.level, LevelSource: "test", LevelReason: "fixture", Platform: "linux",
		Signals: map[string]collectors.PressureSignal{
			collectors.SignalPSIFullAvg60: {Value: 19.77, Status: collectors.SignalMeasured, Provenance: "fixture", Units: "percent"},
		},
		ObservedAt: time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
	}
	return &collectors.MetricData{CollectorName: "pressure", Timestamp: reading.ObservedAt, Values: map[string]interface{}{"memory_pressure": reading}}, nil
}

func newPressureService(t *testing.T, level string, sampler procsampler.Sampler) *MonitorService {
	t.Helper()
	repo := memory.NewRepository()
	collector := &fixedPressureCollector{BaseCollector: collectors.NewBaseCollector("pressure", time.Second), level: level}
	svc := NewMonitorService(&config.Config{}, repo, infrastructure.NewStaticProvider(),
		WithMonitorClock(NewStubClock(time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC))),
		WithCollectors(collector),
		WithProcessSampling(repo, sampler, procsampler.NewAttributor(nil)))
	svc.collectMetrics()
	return svc
}

func TestGetMemoryPressure_AttributesContributorsWhenElevated(t *testing.T) {
	sampler := &fakeSampler{samples: []procsampler.ProcessSample{
		{PID: 1, Comm: "system-monitor-api", FaultStatus: procsampler.FaultMeasured, MajorFaultsPerSecond: 2, SwapStatus: "measured"},
		{PID: 2, PPID: 1, Comm: "node", FaultStatus: procsampler.FaultMeasured, MajorFaultsPerSecond: 5000, SwapStatus: "measured", SwapKB: 4_000_000, FaultCounter: "linux /proc/<pid>/stat majflt"},
		{PID: 3, Comm: "sshd", FaultStatus: procsampler.FaultMeasured, SwapStatus: "measured"},
	}}
	svc := newPressureService(t, collectors.MemoryPressureCritical, sampler)

	got, err := svc.GetMemoryPressure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Level != collectors.MemoryPressureCritical || got.Signals[collectors.SignalPSIFullAvg60].Value != 19.77 {
		t.Fatalf("reading = %+v", got)
	}
	if got.ContributorsStatus != "measured" || len(got.Contributors) != 2 {
		t.Fatalf("contributors = %s %+v, want the two paging processes", got.ContributorsStatus, got.Contributors)
	}
	top := got.Contributors[0]
	if top.PID != 2 || top.Owner != "system-monitor" || top.FaultCounter == "" {
		t.Fatalf("top contributor = %+v, want pid 2 attributed to its system-monitor-api parent", top)
	}
}

func TestGetMemoryPressure_DoesNotSampleProcessesWhenNormal(t *testing.T) {
	sampler := &fakeSampler{}
	svc := newPressureService(t, collectors.MemoryPressureNormal, sampler)
	callsAfterCycle := sampler.calls

	got, err := svc.GetMemoryPressure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.ContributorsStatus != "not_requested" || len(got.Contributors) != 0 {
		t.Fatalf("contributors at normal pressure = %s %+v", got.ContributorsStatus, got.Contributors)
	}
	if sampler.calls != callsAfterCycle {
		t.Fatal("a normal reading walked the process table")
	}
}

func TestGetMemoryPressure_UnsupportedSamplerIsReportedNotEmpty(t *testing.T) {
	svc := newPressureService(t, collectors.MemoryPressureElevated, &fakeSampler{err: procsampler.ErrUnsupported})
	got, err := svc.GetMemoryPressure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.ContributorsStatus != "unsupported" || got.ContributorsReason == "" {
		t.Fatalf("contributors = %s (%q), want unsupported with a reason", got.ContributorsStatus, got.ContributorsReason)
	}
}

func TestGetMemoryPressure_BeforeFirstSampleIsUnknown(t *testing.T) {
	repo := memory.NewRepository()
	svc := NewMonitorService(&config.Config{}, repo, infrastructure.NewStaticProvider(),
		WithCollectors(&fixedPressureCollector{BaseCollector: collectors.NewBaseCollector("pressure", time.Second), level: collectors.MemoryPressureCritical}))
	got, err := svc.GetMemoryPressure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Level != collectors.MemoryPressureUnknown {
		t.Fatalf("level before sampling = %s, want unknown", got.Level)
	}
}
