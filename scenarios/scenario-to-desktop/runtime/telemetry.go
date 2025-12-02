package bundleruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// telemetryRecord represents a single telemetry event in JSONL format.
// Events include: runtime_start, runtime_shutdown, runtime_error, service_start,
// service_ready, service_not_ready, service_exit, gpu_status, secrets_missing,
// secrets_updated, migration_start, migration_applied, migration_failed,
// asset_missing, asset_size_exceeded, asset_size_warning, asset_size_suspicious,
// playwright_chromium_fallback, gpu_required_missing, gpu_fallback_cpu,
// gpu_optional_unavailable.
type telemetryRecord struct {
	Timestamp time.Time              `json:"ts"`
	Event     string                 `json:"event"`
	ServiceID string                 `json:"service_id,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// recordTelemetry appends a telemetry event to the JSONL file.
// The details map can include a "service_id" key for service-specific events.
func (s *Supervisor) recordTelemetry(event string, details map[string]interface{}) error {
	rec := telemetryRecord{
		Timestamp: time.Now().UTC(),
		Event:     event,
		Details:   details,
	}

	// Extract service_id from details if present for cleaner JSON structure.
	if details != nil {
		if sid, ok := details["service_id"].(string); ok {
			rec.ServiceID = sid
		}
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(s.telemetryPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.telemetryPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}
