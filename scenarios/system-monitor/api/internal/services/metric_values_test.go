package services

import (
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
)

func TestMetricStateExplicitFailureWinsOverLegacyValue(t *testing.T) {
	state := metricState(&collectors.MetricData{
		CollectorName: "cpu",
		Timestamp:     time.Unix(100, 0),
		Values: map[string]interface{}{
			"usage_percent": 0.0,
			"status":        "failed",
			"reason":        "not_yet_sampled",
		},
	}, "usage_percent", "CPU has not been sampled yet")

	if state.Status != "failed" || state.Value != 0 || state.Reason != "not_yet_sampled" {
		t.Fatalf("metricState() = %#v, want failed state with reason and no measured value", state)
	}
}

func TestDiskMetricStatePreservesNestedFailure(t *testing.T) {
	state := diskMetricState(&collectors.MetricData{
		CollectorName: "disk",
		Values: map[string]interface{}{
			"usage": map[string]interface{}{
				"percent": 0.0,
				"status":  "failed",
				"reason":  "statfs unavailable",
			},
		},
	})

	if state.Status != "failed" || state.Value != 0 || state.Reason != "statfs unavailable" {
		t.Fatalf("diskMetricState() = %#v, want nested failed state", state)
	}
}

func TestMetricStateUsesPerSignalCPUEnvelope(t *testing.T) {
	state := metricState(&collectors.MetricData{
		CollectorName: "cpu",
		Values: map[string]interface{}{
			"status":                      "measured",
			"quota_throttling_status":     "unsupported",
			"quota_throttling_reason":     "no cgroup CPU limit applies",
			"quota_throttling_provenance": "cgroup v2 cpu.max",
		},
	}, "quota_throttling", "quota unavailable")
	if state.Status != "unsupported" || state.Reason != "no cgroup CPU limit applies" || state.Value != 0 {
		t.Fatalf("metricState() = %#v, want per-signal unsupported state", state)
	}
}
