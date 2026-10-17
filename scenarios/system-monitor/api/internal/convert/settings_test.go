package convert

import (
	"testing"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
)

// TestSettingsProtoRoundTrip_PreservesDiskBands asserts every disk-escalation
// setting survives a trip through the proto surface.
//
// This matters because settings are written through the API as protobuf. A
// field present in Settings but absent from the converter would be silently
// zeroed on every update, and the sanitiser would then restore the default —
// so an operator's configured bands would appear to save and quietly not
// apply. That is the same shape of defect as the incident itself: a
// configuration surface that does not reach the code reading it.
func TestSettingsProtoRoundTrip_PreservesDiskBands(t *testing.T) {
	want := services.Settings{
		Active:                        true,
		MetricCollectionInterval:      20,
		AnomalyDetectionInterval:      30,
		ThresholdCheckInterval:        20,
		CooldownPeriodSeconds:         300,
		CPUThreshold:                  85,
		MemoryThreshold:               90,
		DiskThreshold:                 72,
		DiskHighPercent:               84,
		DiskCriticalPercent:           93,
		DiskEscalationCooldownSeconds: 900,
		DiskEscalationDebounceTicks:   3,
		DiskFastFillJumpPercent:       7,
		MetricsRetentionDays:          30,
		RetentionCheckIntervalSeconds: 3600,
		RetentionRunOnStartup:         true,
		CompactAfterRetention:         false,
	}

	got := ProtoToSettings(SettingsToProto(&want))
	if got == nil {
		t.Fatal("ProtoToSettings returned nil")
	}
	if *got != want {
		t.Errorf("proto round-trip lost settings:\n got=%+v\nwant=%+v", *got, want)
	}
}

// TestSettingsProtoRoundTrip_NilSafe asserts the converters tolerate nil rather
// than panicking inside a request handler.
func TestSettingsProtoRoundTrip_NilSafe(t *testing.T) {
	if SettingsToProto(nil) != nil {
		t.Error("SettingsToProto(nil) should return nil")
	}
	if ProtoToSettings(nil) != nil {
		t.Error("ProtoToSettings(nil) should return nil")
	}
}
