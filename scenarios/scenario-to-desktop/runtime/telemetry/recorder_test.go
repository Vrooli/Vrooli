package telemetry

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/testutil"
)

func TestFileRecorder_Record(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")

	fixedTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	mockClock := testutil.NewMockClock(fixedTime)
	mockFS := testutil.NewMockFileSystem()

	recorder := NewFileRecorder(telemetryPath, mockClock, mockFS)

	details := map[string]interface{}{
		"service_id": "api",
		"message":    "test event",
	}

	if err := recorder.Record("test_event", details); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	data, err := mockFS.ReadFile(telemetryPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var rec Record
	if err := json.Unmarshal(data[:len(data)-1], &rec); err != nil {
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

func TestFileRecorder_MultipleRecords(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")

	mockClock := testutil.NewMockClock(time.Now())
	mockFS := testutil.NewMockFileSystem()

	recorder := NewFileRecorder(telemetryPath, mockClock, mockFS)

	events := []string{"event1", "event2", "event3"}
	for _, event := range events {
		if err := recorder.Record(event, nil); err != nil {
			t.Fatalf("Record(%q) error = %v", event, err)
		}
	}

	data, _ := mockFS.ReadFile(telemetryPath)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var rec Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("Unmarshal line %d error = %v", i, err)
		}
		if rec.Event != events[i] {
			t.Errorf("line %d Event = %q, want %q", i, rec.Event, events[i])
		}
	}
}

func TestFileRecorder_NoServiceID(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")

	mockClock := testutil.NewMockClock(time.Now())
	mockFS := testutil.NewMockFileSystem()

	recorder := NewFileRecorder(telemetryPath, mockClock, mockFS)

	details := map[string]interface{}{
		"key": "value",
	}

	if err := recorder.Record("generic_event", details); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	data, _ := mockFS.ReadFile(telemetryPath)
	var rec Record
	json.Unmarshal(data[:len(data)-1], &rec)

	if rec.ServiceID != "" {
		t.Errorf("rec.ServiceID = %q, want empty", rec.ServiceID)
	}
}

func TestFileRecorder_NilDetails(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")

	mockClock := testutil.NewMockClock(time.Now())
	mockFS := testutil.NewMockFileSystem()

	recorder := NewFileRecorder(telemetryPath, mockClock, mockFS)

	if err := recorder.Record("event_no_details", nil); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	data, _ := mockFS.ReadFile(telemetryPath)
	var rec Record
	if err := json.Unmarshal(data[:len(data)-1], &rec); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if rec.Event != "event_no_details" {
		t.Errorf("rec.Event = %q, want %q", rec.Event, "event_no_details")
	}
}

func TestFileRecorder_CreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "subdir", "nested", "telemetry.jsonl")

	mockClock := testutil.NewMockClock(time.Now())
	realFS := infra.RealFileSystem{}

	recorder := NewFileRecorder(telemetryPath, mockClock, realFS)

	if err := recorder.Record("test_event", nil); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	if _, err := realFS.Stat(telemetryPath); err != nil {
		t.Errorf("telemetry file not created: %v", err)
	}
}

func TestFileRecorder_TimestampIsUTC(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")

	// Use a non-UTC timezone
	localTime := time.Date(2025, 6, 15, 14, 30, 0, 0, time.FixedZone("EST", -5*3600))
	mockClock := testutil.NewMockClock(localTime)
	mockFS := testutil.NewMockFileSystem()

	recorder := NewFileRecorder(telemetryPath, mockClock, mockFS)

	if err := recorder.Record("test_event", nil); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	data, _ := mockFS.ReadFile(telemetryPath)
	var rec Record
	json.Unmarshal(data[:len(data)-1], &rec)

	if rec.Timestamp.Location().String() != "UTC" {
		t.Errorf("Timestamp location = %q, want UTC", rec.Timestamp.Location().String())
	}
}

func TestRecord_JSON(t *testing.T) {
	rec := Record{
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

func TestFileRecorder_WithRealFileSystem(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")

	realClock := infra.RealClock{}
	realFS := infra.RealFileSystem{}

	recorder := NewFileRecorder(telemetryPath, realClock, realFS)

	err := recorder.Record("runtime_start", map[string]interface{}{
		"service_id": "supervisor",
		"version":    "1.0.0",
	})
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	err = recorder.Record("service_start", map[string]interface{}{
		"service_id": "api",
	})
	if err != nil {
		t.Fatalf("Record() error = %v (second call)", err)
	}

	data, err := realFS.ReadFile(telemetryPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

func TestNopRecorder(t *testing.T) {
	recorder := NopRecorder{}
	err := recorder.Record("test_event", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Errorf("NopRecorder.Record() error = %v, want nil", err)
	}
}
