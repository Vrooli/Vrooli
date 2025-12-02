package bundleruntime

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-runtime/telemetry"
)

func TestRecordTelemetry(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")

	fixedTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	mockClock := NewMockClock(fixedTime)
	mockFS := NewMockFileSystem()

	s := &Supervisor{
		telemetryPath: telemetryPath,
		clock:         mockClock,
		fs:            mockFS,
		telemetry:     telemetry.NewFileRecorder(telemetryPath, mockClock, mockFS),
	}

	details := map[string]interface{}{
		"service_id": "api",
		"message":    "test event",
	}

	if err := s.recordTelemetry("test_event", details); err != nil {
		t.Fatalf("recordTelemetry() error = %v", err)
	}

	// Verify file was written
	data, err := mockFS.ReadFile(telemetryPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Parse JSONL record
	var rec telemetry.Record
	if err := json.Unmarshal(data[:len(data)-1], &rec); err != nil { // -1 to remove trailing newline
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if rec.Event != "test_event" {
		t.Errorf("rec.Event = %q, want %q", rec.Event, "test_event")
	}
	if rec.ServiceID != "api" {
		t.Errorf("rec.ServiceID = %q, want %q", rec.ServiceID, "api")
	}
	if !rec.Timestamp.Equal(fixedTime) {
		t.Errorf("rec.Timestamp = %v, want %v", rec.Timestamp, fixedTime)
	}
}

func TestRecordTelemetry_MultipleRecords(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")

	mockClock := NewMockClock(time.Now())
	mockFS := NewMockFileSystem()

	s := &Supervisor{
		telemetryPath: telemetryPath,
		clock:         mockClock,
		fs:            mockFS,
		telemetry:     telemetry.NewFileRecorder(telemetryPath, mockClock, mockFS),
	}

	// Record multiple events
	events := []string{"event1", "event2", "event3"}
	for _, event := range events {
		if err := s.recordTelemetry(event, nil); err != nil {
			t.Fatalf("recordTelemetry(%q) error = %v", event, err)
		}
	}

	// Verify all records were appended
	data, _ := mockFS.ReadFile(telemetryPath)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var rec telemetry.Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("Unmarshal line %d error = %v", i, err)
		}
		if rec.Event != events[i] {
			t.Errorf("line %d Event = %q, want %q", i, rec.Event, events[i])
		}
	}
}

func TestRecordTelemetry_NoServiceID(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")

	mockClock := NewMockClock(time.Now())
	mockFS := NewMockFileSystem()

	s := &Supervisor{
		telemetryPath: telemetryPath,
		clock:         mockClock,
		fs:            mockFS,
		telemetry:     telemetry.NewFileRecorder(telemetryPath, mockClock, mockFS),
	}

	details := map[string]interface{}{
		"key": "value",
	}

	if err := s.recordTelemetry("generic_event", details); err != nil {
		t.Fatalf("recordTelemetry() error = %v", err)
	}

	data, _ := mockFS.ReadFile(telemetryPath)
	var rec telemetry.Record
	json.Unmarshal(data[:len(data)-1], &rec)

	if rec.ServiceID != "" {
		t.Errorf("rec.ServiceID = %q, want empty", rec.ServiceID)
	}
}

func TestRecordTelemetry_NilDetails(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")

	mockClock := NewMockClock(time.Now())
	mockFS := NewMockFileSystem()

	s := &Supervisor{
		telemetryPath: telemetryPath,
		clock:         mockClock,
		fs:            mockFS,
		telemetry:     telemetry.NewFileRecorder(telemetryPath, mockClock, mockFS),
	}

	if err := s.recordTelemetry("event_no_details", nil); err != nil {
		t.Fatalf("recordTelemetry() error = %v", err)
	}

	data, _ := mockFS.ReadFile(telemetryPath)
	var rec telemetry.Record
	if err := json.Unmarshal(data[:len(data)-1], &rec); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if rec.Event != "event_no_details" {
		t.Errorf("rec.Event = %q, want %q", rec.Event, "event_no_details")
	}
}

func TestRecordTelemetry_CreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "subdir", "nested", "telemetry.jsonl")

	mockClock := NewMockClock(time.Now())
	realFS := RealFileSystem{}

	s := &Supervisor{
		telemetryPath: telemetryPath,
		clock:         mockClock,
		fs:            realFS,
		telemetry:     telemetry.NewFileRecorder(telemetryPath, mockClock, realFS),
	}

	if err := s.recordTelemetry("test_event", nil); err != nil {
		t.Fatalf("recordTelemetry() error = %v", err)
	}

	// Verify file exists
	if _, err := s.fs.Stat(telemetryPath); err != nil {
		t.Errorf("telemetry file not created: %v", err)
	}
}

func TestRecordTelemetry_TimestampIsUTC(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")

	// Use a non-UTC timezone
	localTime := time.Date(2025, 6, 15, 14, 30, 0, 0, time.FixedZone("EST", -5*3600))
	mockClock := NewMockClock(localTime)
	mockFS := NewMockFileSystem()

	s := &Supervisor{
		telemetryPath: telemetryPath,
		clock:         mockClock,
		fs:            mockFS,
		telemetry:     telemetry.NewFileRecorder(telemetryPath, mockClock, mockFS),
	}

	if err := s.recordTelemetry("test_event", nil); err != nil {
		t.Fatalf("recordTelemetry() error = %v", err)
	}

	data, _ := mockFS.ReadFile(telemetryPath)
	var rec telemetry.Record
	json.Unmarshal(data[:len(data)-1], &rec)

	// Timestamp should be in UTC
	if rec.Timestamp.Location().String() != "UTC" {
		t.Errorf("Timestamp location = %q, want UTC", rec.Timestamp.Location().String())
	}
}

func TestTelemetryRecord_JSON(t *testing.T) {
	rec := telemetry.Record{
		Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Event:     "service_start",
		ServiceID: "api",
		Details: map[string]interface{}{
			"port":    8080,
			"version": "1.0.0",
		},
	}

	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Verify JSON structure
	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"event":"service_start"`) {
		t.Error("JSON missing event field")
	}
	if !strings.Contains(jsonStr, `"service_id":"api"`) {
		t.Error("JSON missing service_id field")
	}
	if !strings.Contains(jsonStr, `"ts":`) {
		t.Error("JSON missing ts field")
	}
}

func TestRecordTelemetry_WithRealFileSystem(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")

	realClock := RealClock{}
	realFS := RealFileSystem{}

	s := &Supervisor{
		telemetryPath: telemetryPath,
		clock:         realClock,
		fs:            realFS,
		telemetry:     telemetry.NewFileRecorder(telemetryPath, realClock, realFS),
	}

	// Record an event
	err := s.recordTelemetry("runtime_start", map[string]interface{}{
		"service_id": "supervisor",
		"version":    "1.0.0",
	})
	if err != nil {
		t.Fatalf("recordTelemetry() error = %v", err)
	}

	// Append another event
	err = s.recordTelemetry("service_start", map[string]interface{}{
		"service_id": "api",
	})
	if err != nil {
		t.Fatalf("recordTelemetry() error = %v (second call)", err)
	}

	// Read and verify
	data, err := s.fs.ReadFile(telemetryPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}
