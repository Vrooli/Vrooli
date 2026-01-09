// Package telemetry provides telemetry event recording for the bundle runtime.
package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"scenario-to-desktop-runtime/infra"
)

// Record represents a single telemetry event in JSONL format.
// Events include: runtime_start, runtime_shutdown, runtime_error, service_start,
// service_ready, service_not_ready, service_exit, gpu_status, secrets_missing,
// secrets_updated, migration_start, migration_applied, migration_failed,
// asset_missing, asset_size_exceeded, asset_size_warning, asset_size_suspicious,
// playwright_chromium_fallback, gpu_required_missing, gpu_fallback_cpu,
// gpu_optional_unavailable.
type Record struct {
	Timestamp time.Time              `json:"ts"`
	Event     string                 `json:"event"`
	ServiceID string                 `json:"service_id,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// Recorder abstracts telemetry recording for testing.
type Recorder interface {
	// Record records a telemetry event.
	Record(event string, details map[string]interface{}) error
}

// FileRecorder records telemetry events to a JSONL file.
type FileRecorder struct {
	Path  string
	Clock infra.Clock
	FS    infra.FileSystem
}

// NewFileRecorder creates a new FileRecorder.
func NewFileRecorder(path string, clock infra.Clock, fs infra.FileSystem) *FileRecorder {
	return &FileRecorder{
		Path:  path,
		Clock: clock,
		FS:    fs,
	}
}

// Record appends a telemetry event to the JSONL file.
// The details map can include a "service_id" key for service-specific events.
func (r *FileRecorder) Record(event string, details map[string]interface{}) error {
	rec := Record{
		Timestamp: r.Clock.Now().UTC(),
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

	if err := r.FS.MkdirAll(filepath.Dir(r.Path), 0o755); err != nil {
		return err
	}
	f, err := r.FS.OpenFile(r.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// Ensure FileRecorder implements Recorder.
var _ Recorder = (*FileRecorder)(nil)

// NopRecorder is a no-op implementation of Recorder for testing.
type NopRecorder struct{}

// Record does nothing and returns nil.
func (NopRecorder) Record(event string, details map[string]interface{}) error {
	return nil
}

// Ensure NopRecorder implements Recorder.
var _ Recorder = NopRecorder{}
