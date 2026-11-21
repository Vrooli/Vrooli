package services

import (
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// diskIOHistoryObservation stores the measured provider rates in the same
// cycle timeline as the other monitoring signals. It intentionally omits
// legacy scalar-only providers so placeholder values cannot become history.
func diskIOHistoryObservation(data *models.InfrastructureMonitorData, fallbackAt time.Time) *collectors.MetricData {
	if data == nil {
		return nil
	}
	values := map[string]interface{}{}
	writeRateState(values, "read_rate", data.StorageIO.ReadRate, fallbackAt)
	writeRateState(values, "write_rate", data.StorageIO.WriteRate, fallbackAt)
	if len(values) == 0 {
		return nil
	}
	at := data.Timestamp
	if at.IsZero() {
		at = fallbackAt
	}
	return &collectors.MetricData{
		CollectorName: "disk_io",
		Timestamp:     at,
		Type:          "disk_io",
		Values:        values,
		Tags:          map[string]string{"source": "system-monitor/infrastructure/storage_io"},
	}
}

func failedDiskIOHistoryObservation(err error, at time.Time) *collectors.MetricData {
	reason := "disk I/O history could not be sampled"
	if err != nil {
		reason = err.Error()
	}
	values := map[string]interface{}{}
	for _, key := range []string{"read_rate", "write_rate"} {
		values[key+"_status"] = "failed"
		values[key+"_reason"] = reason
		values[key+"_provenance"] = "system-monitor/infrastructure/storage_io"
		values[key+"_units"] = "MB/s"
		values[key+"_observed_at"] = at.UTC().Format(time.RFC3339Nano)
	}
	return &collectors.MetricData{
		CollectorName: "disk_io",
		Timestamp:     at,
		Type:          "disk_io",
		Values:        values,
		Tags:          map[string]string{"source": "system-monitor/infrastructure/storage_io"},
	}
}

func writeRateState(values map[string]interface{}, key string, state models.MetricState, fallbackAt time.Time) {
	if state.Status == "" {
		return
	}
	values[key+"_status"] = state.Status
	if state.Reason != "" {
		values[key+"_reason"] = state.Reason
	}
	if state.Provenance != "" {
		values[key+"_provenance"] = state.Provenance
	}
	units := state.Units
	if units == "" {
		units = "MB/s"
	}
	values[key+"_units"] = units
	at := state.ObservedAt
	if at.IsZero() {
		at = fallbackAt
	}
	values[key+"_observed_at"] = at.UTC().Format(time.RFC3339Nano)
	if state.Status == "measured" {
		values[key] = state.Value
	}
}
