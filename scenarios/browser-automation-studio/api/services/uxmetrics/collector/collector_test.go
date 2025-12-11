package collector

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics/repository"
)

// mockEventSink is a test double for automation/events.Sink
type mockEventSink struct {
	published []autocontracts.EventEnvelope
	limits    autocontracts.EventBufferLimits
}

func (m *mockEventSink) Publish(ctx context.Context, event autocontracts.EventEnvelope) error {
	m.published = append(m.published, event)
	return nil
}

func (m *mockEventSink) Limits() autocontracts.EventBufferLimits {
	return m.limits
}

func TestCollector_Publish_DelegatesToUnderlyingSink(t *testing.T) {
	mockSink := &mockEventSink{
		limits: autocontracts.DefaultEventBufferLimits,
	}
	repo := repository.NewMockRepository()
	collector := NewCollector(mockSink, repo)

	event := autocontracts.EventEnvelope{
		Kind:        autocontracts.EventKindStepStarted,
		ExecutionID: uuid.New(),
		Timestamp:   time.Now(),
	}

	err := collector.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mockSink.published) != 1 {
		t.Fatalf("expected 1 event published to delegate, got %d", len(mockSink.published))
	}
}

func TestCollector_Limits_DelegatesToUnderlyingSink(t *testing.T) {
	expected := autocontracts.EventBufferLimits{PerExecution: 100, PerAttempt: 25}
	mockSink := &mockEventSink{limits: expected}
	repo := repository.NewMockRepository()
	collector := NewCollector(mockSink, repo)

	limits := collector.Limits()
	if limits.PerExecution != expected.PerExecution {
		t.Errorf("expected PerExecution %d, got %d", expected.PerExecution, limits.PerExecution)
	}
	if limits.PerAttempt != expected.PerAttempt {
		t.Errorf("expected PerAttempt %d, got %d", expected.PerAttempt, limits.PerAttempt)
	}
}

func TestCollector_OnStepCompleted_SavesInteractionTrace(t *testing.T) {
	mockSink := &mockEventSink{}
	repo := repository.NewMockRepository()
	collector := NewCollector(mockSink, repo)

	execID := uuid.New()
	now := time.Now()

	outcome := &autocontracts.StepOutcome{
		StepIndex:  0,
		NodeID:     "click-1",
		StepType:   "click",
		Success:    true,
		StartedAt:  now,
		DurationMs: 500,
	}

	event := autocontracts.EventEnvelope{
		Kind:        autocontracts.EventKindStepCompleted,
		ExecutionID: execID,
		Timestamp:   now,
		Payload:     outcome,
	}

	err := collector.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that an interaction trace was saved
	if len(repo.SavedTraces) != 1 {
		t.Fatalf("expected 1 trace saved, got %d", len(repo.SavedTraces))
	}

	trace := repo.SavedTraces[0]
	if trace.ExecutionID != execID {
		t.Errorf("expected execution ID %s, got %s", execID, trace.ExecutionID)
	}
	if trace.StepIndex != 0 {
		t.Errorf("expected step index 0, got %d", trace.StepIndex)
	}
	if trace.ActionType != contracts.ActionClick {
		t.Errorf("expected action type 'click', got '%s'", trace.ActionType)
	}
	if !trace.Success {
		t.Error("expected success=true")
	}
}

func TestCollector_OnStepCompleted_SavesCursorPath(t *testing.T) {
	mockSink := &mockEventSink{}
	repo := repository.NewMockRepository()
	collector := NewCollector(mockSink, repo)

	execID := uuid.New()
	now := time.Now()

	// Outcome with cursor trail
	outcome := &autocontracts.StepOutcome{
		StepIndex:  0,
		NodeID:     "click-1",
		StepType:   "click",
		Success:    true,
		StartedAt:  now,
		CompletedAt: func() *time.Time { t := now.Add(500 * time.Millisecond); return &t }(),
		DurationMs: 500,
		CursorTrail: []autocontracts.CursorPosition{
			{Point: autocontracts.Point{X: 0, Y: 0}, RecordedAt: now},
			{Point: autocontracts.Point{X: 100, Y: 0}, RecordedAt: now.Add(100 * time.Millisecond)},
			{Point: autocontracts.Point{X: 100, Y: 100}, RecordedAt: now.Add(200 * time.Millisecond)},
		},
	}

	event := autocontracts.EventEnvelope{
		Kind:        autocontracts.EventKindStepCompleted,
		ExecutionID: execID,
		Timestamp:   now,
		Payload:     outcome,
	}

	err := collector.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that a cursor path was saved
	if len(repo.SavedCursorPaths) != 1 {
		t.Fatalf("expected 1 cursor path saved, got %d", len(repo.SavedCursorPaths))
	}

	// The cursor path should have computed metrics
	for _, path := range repo.SavedCursorPaths {
		if path.StepIndex != 0 {
			t.Errorf("expected step index 0, got %d", path.StepIndex)
		}
		if len(path.Points) != 3 {
			t.Errorf("expected 3 points, got %d", len(path.Points))
		}
		if path.TotalDistancePx <= 0 {
			t.Error("expected positive total distance")
		}
	}
}

func TestCollector_OnCursorUpdate(t *testing.T) {
	mockSink := &mockEventSink{}
	repo := repository.NewMockRepository()
	collector := NewCollector(mockSink, repo)

	execID := uuid.New()
	now := time.Now()

	err := collector.OnCursorUpdate(context.Background(), execID, 0, contracts.TimedPoint{
		X:         50,
		Y:         75,
		Timestamp: now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cursor updates are buffered, so nothing saved yet
	// They would be flushed on step completion
}

func TestCollector_FlushExecution(t *testing.T) {
	mockSink := &mockEventSink{}
	repo := repository.NewMockRepository()
	collector := NewCollector(mockSink, repo)

	execID := uuid.New()

	// Add some cursor updates (these will be buffered internally)
	now := time.Now()
	_ = collector.OnCursorUpdate(context.Background(), execID, 0, contracts.TimedPoint{X: 0, Y: 0, Timestamp: now})
	_ = collector.OnCursorUpdate(context.Background(), execID, 0, contracts.TimedPoint{X: 50, Y: 50, Timestamp: now.Add(time.Second)})

	// Flush should clean up the buffers
	err := collector.FlushExecution(context.Background(), execID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollector_OnExecutionCompleted_FlushesData(t *testing.T) {
	mockSink := &mockEventSink{}
	repo := repository.NewMockRepository()
	collector := NewCollector(mockSink, repo)

	execID := uuid.New()

	// Simulate execution completed event
	event := autocontracts.EventEnvelope{
		Kind:        autocontracts.EventKindExecutionCompleted,
		ExecutionID: execID,
		Timestamp:   time.Now(),
	}

	err := collector.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Execution completed should be delegated
	if len(mockSink.published) != 1 {
		t.Errorf("expected 1 event delegated, got %d", len(mockSink.published))
	}
}

func TestBuildCursorPath(t *testing.T) {
	mockSink := &mockEventSink{}
	repo := repository.NewMockRepository()
	collector := NewCollector(mockSink, repo)

	start := time.Now()
	end := start.Add(500 * time.Millisecond)

	trail := []autocontracts.CursorPosition{
		{Point: autocontracts.Point{X: 0, Y: 0}, RecordedAt: start},
		{Point: autocontracts.Point{X: 100, Y: 0}, RecordedAt: start.Add(200 * time.Millisecond)},
		{Point: autocontracts.Point{X: 100, Y: 100}, RecordedAt: end},
	}

	path := collector.buildCursorPath(0, trail, start, &end)

	if path.StepIndex != 0 {
		t.Errorf("expected step index 0, got %d", path.StepIndex)
	}
	if len(path.Points) != 3 {
		t.Errorf("expected 3 points, got %d", len(path.Points))
	}

	// Total distance = 100 (horizontal) + 100 (vertical) = 200
	expectedDist := 200.0
	if path.TotalDistancePx < expectedDist-1 || path.TotalDistancePx > expectedDist+1 {
		t.Errorf("expected total distance ~%f, got %f", expectedDist, path.TotalDistancePx)
	}

	// Direct distance = sqrt(100^2 + 100^2) = ~141.4
	if path.DirectDistance < 140 || path.DirectDistance > 143 {
		t.Errorf("expected direct distance ~141.4, got %f", path.DirectDistance)
	}

	// Directness should be ~0.707 (direct/total)
	if path.Directness < 0.7 || path.Directness > 0.72 {
		t.Errorf("expected directness ~0.707, got %f", path.Directness)
	}
}

func TestMapStepTypeToAction(t *testing.T) {
	tests := []struct {
		input    string
		expected contracts.ActionType
	}{
		{"click", contracts.ActionClick},
		{"type", contracts.ActionType_},
		{"input", contracts.ActionType_},
		{"fill", contracts.ActionType_},
		{"scroll", contracts.ActionScroll},
		{"navigate", contracts.ActionNavigation},
		{"goto", contracts.ActionNavigation},
		{"wait", contracts.ActionWait},
		{"hover", contracts.ActionHover},
		{"drag", contracts.ActionDrag},
		{"dragdrop", contracts.ActionDrag},
		{"unknown", contracts.ActionClick}, // default
	}

	for _, tc := range tests {
		result := mapStepTypeToAction(tc.input)
		if result != tc.expected {
			t.Errorf("mapStepTypeToAction(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}
