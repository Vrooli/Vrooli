package convert

import (
	"testing"
	"time"

	metricspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

func TestMetricsResponseToProtoPreservesMeasuredZeroAndUnavailableState(t *testing.T) {
	response := MetricsResponseToProto(&models.MetricsResponse{
		CPUState:  models.MetricState{Status: "measured", Value: 0},
		GPUState:  models.MetricState{Status: "unsupported", Reason: "no GPU present"},
		Timestamp: time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC),
	})

	if measured, ok := response.GetCpu().GetState().(*metricspb.MetricValue_Measured); !ok || measured.Measured != 0 {
		t.Fatalf("CPU state = %T %#v, want measured zero", response.GetCpu().GetState(), response.GetCpu().GetState())
	}
	if unsupported, ok := response.GetGpu().GetState().(*metricspb.MetricValue_UnsupportedReason); !ok || unsupported.UnsupportedReason != "no GPU present" {
		t.Fatalf("GPU state = %T %#v, want unsupported reason", response.GetGpu().GetState(), response.GetGpu().GetState())
	}
}

func TestKernelMemoryProtoPreservesStatesAcrossCurrentDetailAndTimeline(t *testing.T) {
	when := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	kernel := &models.KernelMemoryMetrics{
		SlabBytes:          models.MetricState{Status: "measured", Value: 0, Units: "bytes", ObservedAt: when},
		ReclaimableBytes:   models.MetricState{Status: "unsupported", Reason: "missing kernel field"},
		UnreclaimableBytes: models.MetricState{Status: "failed", Reason: "snapshot failure"},
		HostTotalBytes:     models.MetricState{Status: "not_yet_sampled", Reason: "old sample"},
	}
	got := []*metricspb.KernelMemoryMetrics{
		MetricsResponseToProto(&models.MetricsResponse{Timestamp: when, KernelMemory: kernel}).GetKernelMemory(),
		DetailedMetricsToProto(&models.DetailedMetrics{Timestamp: when, MemoryDetails: models.MemoryMetrics{KernelMemory: kernel}}).GetMemoryDetails().GetKernelMemory(),
		MetricsTimelineResponseToProto(&models.MetricsTimelineResponse{Samples: []models.MetricTimelineSample{{Timestamp: when, KernelMemory: kernel}}}).GetSamples()[0].GetKernelMemory(),
	}
	for i, value := range got {
		if _, ok := value.GetSlabBytes().GetState().(*metricspb.MetricValue_Measured); !ok {
			t.Fatalf("projection%d lost measured zero: %v", i, value)
		}
		if value.GetReclaimableBytes().GetUnsupportedReason() != "missing kernel field" || value.GetUnreclaimableBytes().GetFailedError() != "snapshot failure" || value.GetHostTotalBytes().GetNotYetSampledReason() != "old sample" {
			t.Fatalf("projection%d lost unavailable states: %v", i, value)
		}
	}
}

func TestTimelineResponsePreservesExactWindow(t *testing.T) {
	start := time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	got := MetricsTimelineResponseToProto(&models.MetricsTimelineResponse{WindowStart: start, WindowEnd: end, WindowSeconds: 3600})
	if !got.GetWindowStart().AsTime().Equal(start) || !got.GetWindowEnd().AsTime().Equal(end) {
		t.Fatalf("window lost: %v – %v", got.GetWindowStart(), got.GetWindowEnd())
	}
	if empty := MetricsTimelineResponseToProto(&models.MetricsTimelineResponse{}); empty.GetWindowStart() != nil || empty.GetWindowEnd() != nil {
		t.Fatal("missing bounds must remain absent")
	}
}

func TestTimelineGPUDistinguishesAbsentObservationFromMeasuredZero(t *testing.T) {
	when := time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	zero := 0.0
	got := MetricsTimelineResponseToProto(&models.MetricsTimelineResponse{Samples: []models.MetricTimelineSample{
		{Timestamp: when},
		{Timestamp: when, GPUUsage: &zero, GPUState: models.MetricState{Status: "measured", Value: 0, ObservedAt: when}},
		{Timestamp: when, GPUState: models.MetricState{Status: "failed", Reason: "probe failed"}},
	}}).GetSamples()
	if got[0].Gpu != nil {
		t.Fatalf("absent GPU observation became %v", got[0].Gpu)
	}
	if _, ok := got[1].GetGpu().GetState().(*metricspb.MetricValue_Measured); !ok || got[1].GetGpu().GetMeasured() != 0 {
		t.Fatalf("measured zero lost: %v", got[1].Gpu)
	}
	if got[2].GetGpu().GetFailedError() != "probe failed" {
		t.Fatalf("failure lost: %v", got[2].Gpu)
	}
}

func TestStorageIORatesPreserveMeasuredZeroAndUnsupportedLegacy(t *testing.T) {
	at := time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	got := storageIOInfoToProto(models.StorageIOInfo{
		ReadRate:  models.MetricState{Status: "measured", Value: 0, ObservedAt: at, Units: "MB/s"},
		WriteRate: models.MetricState{Status: "failed", Reason: "disk counters unavailable"},
	})
	if value, ok := got.GetReadRate().GetState().(*metricspb.MetricValue_Measured); !ok || value.Measured != 0 || !got.GetReadRate().GetObservedAt().AsTime().Equal(at) {
		t.Fatalf("measured zero lost: %v", got.GetReadRate())
	}
	if got.GetWriteRate().GetFailedError() != "disk counters unavailable" {
		t.Fatalf("failure lost: %v", got.GetWriteRate())
	}
	legacy := storageIOInfoToProto(models.StorageIOInfo{ReadMBPerSec: 15, WriteMBPerSec: 8})
	if legacy.GetReadRate().GetUnsupportedReason() == "" || legacy.GetWriteRate().GetUnsupportedReason() == "" {
		t.Fatalf("legacy scalars were treated as measured: %v", legacy)
	}
}
