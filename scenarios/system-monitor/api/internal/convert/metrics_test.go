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
