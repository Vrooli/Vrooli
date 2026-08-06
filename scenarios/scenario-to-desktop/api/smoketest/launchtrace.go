package smoketest

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// LaunchTraceSchemaVersion identifies the persisted launch trace contract.
const LaunchTraceSchemaVersion = "launch-trace.v1"

type LaunchRunKind string

const (
	LaunchRunProtocol LaunchRunKind = "protocol"
	LaunchRunDemo     LaunchRunKind = "demo"
)

// LaunchEventName is deliberately closed. New events require a contract
// change so downstream timing comparisons cannot silently lose a phase.
type LaunchEventName string

const (
	EventRecorderStarted       LaunchEventName = "recorder_started"
	EventProtocolStarted       LaunchEventName = "protocol_started"
	EventProtocolCompleted     LaunchEventName = "protocol_completed"
	EventDemoSpawn             LaunchEventName = "demo_spawn"
	EventElectronReady         LaunchEventName = "electron_ready"
	EventSplashCreated         LaunchEventName = "splash_created"
	EventSplashLoadCompleted   LaunchEventName = "splash_load_completed"
	EventSplashReadyToShow     LaunchEventName = "splash_ready_to_show"
	EventSplashShown           LaunchEventName = "splash_shown"
	EventSplashFirstPaint      LaunchEventName = "splash_first_paint"
	EventRuntimeSpawned        LaunchEventName = "runtime_spawned"
	EventRuntimeTokenAvailable LaunchEventName = "runtime_token_available"
	EventRuntimeHealthReady    LaunchEventName = "runtime_health_ready"
	EventRuntimeReady          LaunchEventName = "runtime_ready"
	EventPortDiscovered        LaunchEventName = "port_discovered"
	EventServerReady           LaunchEventName = "server_ready"
	EventMainWindowCreated     LaunchEventName = "main_window_created"
	EventMainWindowLoad        LaunchEventName = "main_window_load_completed"
	EventMainWindowShown       LaunchEventName = "main_window_shown"
	EventAppReady              LaunchEventName = "app_ready"
	EventJourneyStarted        LaunchEventName = "journey_started"
	EventRecordingEnded        LaunchEventName = "recording_ended"
)

var (
	credentialKeyPattern   = regexp.MustCompile(`(?i)(token|secret|password|credential|authorization|api[_-]?key|private[_-]?key)`)
	credentialValuePattern = regexp.MustCompile(`(?i)(bearer\s+[A-Za-z0-9._-]+|-----begin [^-]+ key-----|[A-Za-z0-9+/]{32,}={0,2})`)
)

// LaunchEvent is an ordered event with both a durable wall-clock timestamp
// and a process-relative monotonic timestamp for timing decisions.
type LaunchEvent struct {
	Name        LaunchEventName   `json:"name"`
	Component   string            `json:"component"`
	Role        string            `json:"role"`
	MonotonicNs int64             `json:"monotonic_ns"`
	WallTime    time.Time         `json:"wall_time"`
	Details     map[string]string `json:"details,omitempty"`
}

// LaunchTrace is written once by the producer. Protocol and demo traces use
// different run IDs even when they belong to one smoke-test orchestration.
type LaunchTrace struct {
	SchemaVersion string        `json:"schema_version"`
	RunID         string        `json:"run_id"`
	RunKind       LaunchRunKind `json:"run_kind"`
	StartedAt     time.Time     `json:"started_at"`
	CompletedAt   time.Time     `json:"completed_at,omitempty"`
	Events        []LaunchEvent `json:"events"`
}

func (t *LaunchTrace) Validate() error {
	if t == nil {
		return fmt.Errorf("launch trace is nil")
	}
	if t.SchemaVersion != LaunchTraceSchemaVersion {
		return fmt.Errorf("unsupported launch trace schema %q", t.SchemaVersion)
	}
	if strings.TrimSpace(t.RunID) == "" {
		return fmt.Errorf("launch trace run_id is required")
	}
	if t.RunKind != LaunchRunProtocol && t.RunKind != LaunchRunDemo {
		return fmt.Errorf("unsupported launch trace run_kind %q", t.RunKind)
	}
	if t.StartedAt.IsZero() {
		return fmt.Errorf("launch trace started_at is required")
	}
	if !t.CompletedAt.IsZero() && t.CompletedAt.Before(t.StartedAt) {
		return fmt.Errorf("launch trace completed_at precedes started_at")
	}
	var previous int64
	for i, event := range t.Events {
		if !knownLaunchEvent(event.Name) {
			return fmt.Errorf("event %d has unknown name %q", i, event.Name)
		}
		if strings.TrimSpace(event.Component) == "" || strings.TrimSpace(event.Role) == "" {
			return fmt.Errorf("event %d requires component and role", i)
		}
		if event.MonotonicNs < 0 || (i > 0 && event.MonotonicNs < previous) {
			return fmt.Errorf("event %d violates monotonic ordering", i)
		}
		if event.WallTime.IsZero() {
			return fmt.Errorf("event %d wall_time is required", i)
		}
		if err := validateLaunchDetails(event.Details); err != nil {
			return fmt.Errorf("event %d: %w", i, err)
		}
		previous = event.MonotonicNs
	}
	for _, required := range requiredEvents(t.RunKind) {
		if !t.Has(required) {
			return fmt.Errorf("required event %q is missing", required)
		}
	}
	return nil
}

func (t LaunchTrace) Has(name LaunchEventName) bool {
	for _, event := range t.Events {
		if event.Name == name {
			return true
		}
	}
	return false
}

func (t LaunchTrace) Event(name LaunchEventName) (LaunchEvent, bool) {
	for _, event := range t.Events {
		if event.Name == name {
			return event, true
		}
	}
	return LaunchEvent{}, false
}

func (t LaunchTrace) Segment(start, end LaunchEventName) (time.Duration, error) {
	a, ok := t.Event(start)
	if !ok {
		return 0, fmt.Errorf("segment start event %q is missing", start)
	}
	b, ok := t.Event(end)
	if !ok {
		return 0, fmt.Errorf("segment end event %q is missing", end)
	}
	if b.MonotonicNs < a.MonotonicNs {
		return 0, fmt.Errorf("segment %q to %q is reversed", start, end)
	}
	return time.Duration(b.MonotonicNs-a.MonotonicNs) * time.Nanosecond, nil
}

func requiredEvents(kind LaunchRunKind) []LaunchEventName {
	if kind == LaunchRunProtocol {
		return []LaunchEventName{EventRecorderStarted, EventProtocolStarted, EventProtocolCompleted}
	}
	return []LaunchEventName{
		EventRecorderStarted, EventDemoSpawn, EventElectronReady,
		EventSplashCreated, EventSplashLoadCompleted, EventSplashFirstPaint,
		EventMainWindowCreated, EventMainWindowLoad, EventMainWindowShown, EventAppReady,
	}
}

func knownLaunchEvent(name LaunchEventName) bool {
	for _, candidate := range []LaunchEventName{
		EventRecorderStarted, EventProtocolStarted, EventProtocolCompleted, EventDemoSpawn,
		EventElectronReady, EventSplashCreated, EventSplashLoadCompleted, EventSplashReadyToShow,
		EventSplashShown, EventSplashFirstPaint, EventRuntimeSpawned, EventRuntimeTokenAvailable,
		EventRuntimeHealthReady, EventRuntimeReady, EventPortDiscovered, EventServerReady,
		EventMainWindowCreated, EventMainWindowLoad, EventMainWindowShown, EventAppReady,
		EventJourneyStarted, EventRecordingEnded,
	} {
		if name == candidate {
			return true
		}
	}
	return false
}

func validateLaunchDetails(details map[string]string) error {
	for key, value := range details {
		if credentialKeyPattern.MatchString(key) || credentialValuePattern.MatchString(value) {
			return fmt.Errorf("credential-shaped detail %q is not allowed", key)
		}
	}
	return nil
}

func (t LaunchTrace) MarshalValidated() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(t, "", "  ")
}
