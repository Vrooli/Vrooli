package deployment

import (
	"testing"
	"time"
)

func TestChannelEmitter_Creation(t *testing.T) {
	t.Parallel()

	emitter := NewChannelEmitter(10)
	if emitter == nil {
		t.Fatal("NewChannelEmitter() returned nil")
	}
	if emitter.closed {
		t.Error("NewChannelEmitter() should not be closed")
	}
	if emitter.ch == nil {
		t.Error("NewChannelEmitter() channel should not be nil")
	}
}

func TestChannelEmitter_Emit(t *testing.T) {
	t.Parallel()

	emitter := NewChannelEmitter(10)

	event := Event{
		Type:      "step_started",
		Step:      "upload",
		StepTitle: "Uploading bundle",
		Progress:  25.0,
		Message:   "Test message",
	}

	emitter.Emit(event)

	select {
	case received := <-emitter.Channel():
		if received.Type != "step_started" {
			t.Errorf("Type = %q, want %q", received.Type, "step_started")
		}
		if received.Step != "upload" {
			t.Errorf("Step = %q, want %q", received.Step, "upload")
		}
		if received.StepTitle != "Uploading bundle" {
			t.Errorf("StepTitle = %q, want %q", received.StepTitle, "Uploading bundle")
		}
		if received.Progress != 25.0 {
			t.Errorf("Progress = %f, want 25.0", received.Progress)
		}
		if received.Timestamp == "" {
			t.Error("Timestamp should be auto-filled")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Emit() did not send event to channel")
	}
}

func TestChannelEmitter_PreservesExistingTimestamp(t *testing.T) {
	t.Parallel()

	emitter := NewChannelEmitter(10)
	customTimestamp := "2025-01-15T12:00:00Z"

	event := Event{
		Type:      "step_completed",
		Step:      "upload",
		Timestamp: customTimestamp,
	}

	emitter.Emit(event)

	select {
	case received := <-emitter.Channel():
		if received.Timestamp != customTimestamp {
			t.Errorf("Timestamp = %q, want %q (should preserve existing)", received.Timestamp, customTimestamp)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Emit() did not send event to channel")
	}
}

func TestChannelEmitter_Close(t *testing.T) {
	t.Parallel()

	emitter := NewChannelEmitter(10)

	emitter.Close()

	if !emitter.closed {
		t.Error("Close() should set closed to true")
	}

	// Verify channel is closed
	select {
	case _, ok := <-emitter.Channel():
		if ok {
			t.Error("Channel should be closed after Close()")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Channel should return immediately when closed")
	}
}

func TestChannelEmitter_CloseIdempotent(t *testing.T) {
	t.Parallel()

	emitter := NewChannelEmitter(10)

	// Multiple close calls should not panic
	emitter.Close()
	emitter.Close()
	emitter.Close()
}

func TestChannelEmitter_EmitAfterClose(t *testing.T) {
	t.Parallel()

	emitter := NewChannelEmitter(10)
	emitter.Close()

	// Should not panic when emitting after close
	event := Event{
		Type: "step_started",
		Step: "upload",
	}
	emitter.Emit(event) // Should silently ignore
}

func TestNoOpEmitter(t *testing.T) {
	t.Parallel()

	var emitter NoOpEmitter

	// These should not panic
	emitter.Emit(Event{Type: "test"})
	emitter.Close()
	emitter.Emit(Event{Type: "test"})
}

func TestAllSteps(t *testing.T) {
	t.Parallel()

	steps := AllSteps()

	if len(steps) == 0 {
		t.Fatal("AllSteps() returned empty slice")
	}

	// Check first two steps are bundle_build and preflight
	if steps[0].ID != "bundle_build" {
		t.Errorf("First step ID = %q, want %q", steps[0].ID, "bundle_build")
	}
	if steps[1].ID != "preflight" {
		t.Errorf("Second step ID = %q, want %q", steps[1].ID, "preflight")
	}

	// Verify each step has non-empty ID and Title
	for i, step := range steps {
		if step.ID == "" {
			t.Errorf("Step %d has empty ID", i)
		}
		if step.Title == "" {
			t.Errorf("Step %d (%s) has empty Title", i, step.ID)
		}
	}
}

func TestSetupSteps(t *testing.T) {
	t.Parallel()

	if len(SetupSteps) == 0 {
		t.Fatal("SetupSteps is empty")
	}

	// Check expected setup step IDs
	expectedIDs := []string{"mkdir", "bootstrap", "upload", "extract", "setup", "autoheal", "verify_setup"}
	for i, expected := range expectedIDs {
		if i >= len(SetupSteps) {
			t.Errorf("Missing expected step: %s", expected)
			continue
		}
		if SetupSteps[i].ID != expected {
			t.Errorf("SetupSteps[%d].ID = %q, want %q", i, SetupSteps[i].ID, expected)
		}
	}
}

func TestDeploySteps(t *testing.T) {
	t.Parallel()

	if len(DeploySteps) == 0 {
		t.Fatal("DeploySteps is empty")
	}

	// Verify all steps have positive weights
	for _, step := range DeploySteps {
		if step.Weight <= 0 {
			t.Errorf("DeployStep %s has non-positive weight: %f", step.ID, step.Weight)
		}
	}
}
