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

func TestTimelineDiskPreservesMeasuredValueAndFailure(t *testing.T) {
	when := time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	got := MetricsTimelineResponseToProto(&models.MetricsTimelineResponse{Samples: []models.MetricTimelineSample{
		{Timestamp: when},
		{Timestamp: when, DiskUsage: 63.5, DiskUsageState: models.MetricState{Status: "measured", Value: 63.5, ObservedAt: when, Units: "percent"}},
		{Timestamp: when, DiskUsageState: models.MetricState{Status: "failed", Reason: "statfs unavailable"}},
	}}).GetSamples()
	if got[0].GetDiskUsage() != nil {
		t.Fatalf("absent disk observation became %v", got[0].GetDiskUsage())
	}
	if _, ok := got[1].GetDiskUsage().GetState().(*metricspb.MetricValue_Measured); !ok || got[1].GetDiskUsage().GetMeasured() != 63.5 {
		t.Fatalf("measured disk usage lost: %v", got[1].GetDiskUsage())
	}
	if got[2].GetDiskUsage().GetFailedError() != "statfs unavailable" {
		t.Fatalf("disk failure lost: %v", got[2].GetDiskUsage())
	}
}

func TestTimelineDiskIOPreservesMeasuredZeroAndTypedGaps(t *testing.T) {
	when := time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	got := MetricsTimelineResponseToProto(&models.MetricsTimelineResponse{Samples: []models.MetricTimelineSample{
		{Timestamp: when},
		{Timestamp: when, DiskReadRateState: models.MetricState{Status: "measured", Value: 0, ObservedAt: when, Units: "MB/s"}},
		{Timestamp: when, DiskWriteRateState: models.MetricState{Status: "failed", Reason: "disk counters reset", ObservedAt: when, Units: "MB/s"}},
		{Timestamp: when, DiskReadRateState: models.MetricState{Status: "unsupported", Reason: "physical disk counters unavailable", ObservedAt: when, Units: "MB/s"}},
	}}).GetSamples()
	if got[0].GetDiskReadRate() != nil || got[0].GetDiskWriteRate() != nil {
		t.Fatalf("missing rates became observations: read=%v write=%v", got[0].GetDiskReadRate(), got[0].GetDiskWriteRate())
	}
	if measured, ok := got[1].GetDiskReadRate().GetState().(*metricspb.MetricValue_Measured); !ok || measured.Measured != 0 {
		t.Fatalf("measured zero lost: %v", got[1].GetDiskReadRate())
	}
	if got[2].GetDiskWriteRate().GetFailedError() != "disk counters reset" {
		t.Fatalf("failed interval lost: %v", got[2].GetDiskWriteRate())
	}
	if got[3].GetDiskReadRate().GetUnsupportedReason() != "physical disk counters unavailable" {
		t.Fatalf("unsupported platform lost: %v", got[3].GetDiskReadRate())
	}
}

func TestProcessMonitorProjectionKeepsMissingHealthDistinctFromMeasuredZero(t *testing.T) {
	missing := ProcessMonitorDataToProto(&models.ProcessMonitorData{})
	if missing.GetProcessHealth() != nil {
		t.Fatalf("missing health was fabricated: %v", missing.GetProcessHealth())
	}

	measured := ProcessMonitorDataToProto(&models.ProcessMonitorData{
		ProcessHealth: &models.ProcessHealthInfo{
			ZombieCount: models.MetricState{Status: "measured", Value: 0, Units: "count"},
		},
	})
	count := measured.GetProcessHealth().GetZombieCount()
	if value, ok := count.GetState().(*metricspb.MetricValue_Measured); !ok || value.Measured != 0 {
		t.Fatalf("measured zero was lost: %v", count)
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
	if legacy.GetReadMbPerSec() != 0 || legacy.GetWriteMbPerSec() != 0 {
		t.Fatalf("unsupported legacy throughput scalars were retained: %v", legacy)
	}
}

func TestStorageIOUnsupportedScalarsStayUnavailable(t *testing.T) {
	got := storageIOInfoToProto(models.StorageIOInfo{
		QueueDepthState: models.MetricState{Status: "unsupported", Reason: "Disk queue depth is not collected", Units: "requests"},
		IOWaitState:     models.MetricState{Status: "unsupported", Reason: "Disk I/O wait is not collected", Units: "%"},
	})
	if got.GetDiskQueueDepthState().GetUnsupportedReason() != "Disk queue depth is not collected" {
		t.Fatalf("queue depth state = %v", got.GetDiskQueueDepthState())
	}
	if got.GetIoWaitState().GetUnsupportedReason() != "Disk I/O wait is not collected" {
		t.Fatalf("I/O wait state = %v", got.GetIoWaitState())
	}
}

func TestStorageIOLegacyScalarsAreNotProjectedAsMeasurements(t *testing.T) {
	got := storageIOInfoToProto(models.StorageIOInfo{DiskQueueDepth: 0.2, IOWaitPercent: 2.5})
	if got.GetDiskQueueDepthState().GetUnsupportedReason() == "" {
		t.Fatalf("legacy queue-depth scalar became a measurement: %v", got.GetDiskQueueDepthState())
	}
	if got.GetIoWaitState().GetUnsupportedReason() == "" {
		t.Fatalf("legacy I/O-wait scalar became a measurement: %v", got.GetIoWaitState())
	}
	if got.GetDiskQueueDepth() != 0 || got.GetIoWaitPercent() != 0 {
		t.Fatalf("unsupported compatibility fields retained placeholder values: %v", got)
	}
}
