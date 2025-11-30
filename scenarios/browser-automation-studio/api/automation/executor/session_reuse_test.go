package executor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/events"
)

// sessionTracker tracks session lifecycle operations for testing
type sessionTracker struct {
	mu sync.Mutex

	StartCalls int
	ResetCalls int
	CloseCalls int
	RunCalls   int

	// Track which sessions were created
	Sessions []string
}

func (t *sessionTracker) recordStart(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.StartCalls++
	t.Sessions = append(t.Sessions, id)
}

func (t *sessionTracker) recordReset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ResetCalls++
}

func (t *sessionTracker) recordClose() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.CloseCalls++
}

func (t *sessionTracker) recordRun() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.RunCalls++
}

// trackingEngine wraps an engine to track session operations
type trackingEngine struct {
	tracker *sessionTracker
}

func (e *trackingEngine) Name() string { return "tracking-engine" }

func (e *trackingEngine) Capabilities(context.Context) (contracts.EngineCapabilities, error) {
	return contracts.EngineCapabilities{
		SchemaVersion:         contracts.CapabilitiesSchemaVersion,
		Engine:                e.Name(),
		MaxConcurrentSessions: 10,
	}, nil
}

func (e *trackingEngine) StartSession(_ context.Context, spec engine.SessionSpec) (engine.EngineSession, error) {
	sessionID := uuid.New().String()
	e.tracker.recordStart(sessionID)
	return &trackingSession{
		id:      sessionID,
		tracker: e.tracker,
	}, nil
}

// trackingSession tracks session method calls
type trackingSession struct {
	id      string
	tracker *sessionTracker
}

func (s *trackingSession) Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	s.tracker.recordRun()

	// Handle entry probe
	if instruction.Index == -1 || instruction.NodeID == entryProbeNodeID {
		now := time.Now().UTC()
		return contracts.StepOutcome{
			SchemaVersion:  contracts.StepOutcomeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			StepIndex:      instruction.Index,
			NodeID:         instruction.NodeID,
			StepType:       instruction.Type,
			Success:        true,
			StartedAt:      now,
			CompletedAt:    &now,
		}, nil
	}

	now := time.Now().UTC()
	return contracts.StepOutcome{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    uuid.Nil, // Will be set by executor
		StepIndex:      instruction.Index,
		NodeID:         instruction.NodeID,
		StepType:       instruction.Type,
		Success:        true,
		StartedAt:      now,
		CompletedAt:    &now,
	}, nil
}

func (s *trackingSession) Reset(context.Context) error {
	s.tracker.recordReset()
	return nil
}

func (s *trackingSession) Close(context.Context) error {
	s.tracker.recordClose()
	return nil
}

func (s *trackingSession) ID() string {
	return s.id
}

// TestSessionReuseModeReuse verifies that "reuse" mode keeps the same session
// without calling Reset or Close between steps (only cleanup at end)
func TestSessionReuseModeReuse(t *testing.T) {
	tracker := &sessionTracker{}
	eng := &trackingEngine{tracker: tracker}

	plan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "step-1", Type: "navigate", Params: map[string]any{"url": "https://example.com"}},
			{Index: 1, NodeID: "step-2", Type: "click", Params: map[string]any{"selector": "#btn"}},
			{Index: 2, NodeID: "step-3", Type: "wait", Params: map[string]any{"selector": "#result"}},
		},
		CreatedAt: time.Now().UTC(),
	}

	exec := NewSimpleExecutor(nil)
	memSink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	req := Request{
		Plan:          plan,
		EngineName:    eng.Name(),
		EngineFactory: engine.NewStaticFactory(eng),
		Recorder:      &stubRecorder{},
		EventSink:     memSink,
		ReuseMode:     engine.ReuseModeReuse, // Explicit reuse mode
	}

	err := exec.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify session behavior for "reuse" mode:
	// - Should create exactly 1 session
	// - Should NOT call Reset between steps
	// - Should call Close once at the end (defer cleanup)
	if tracker.StartCalls != 1 {
		t.Errorf("ReuseModeReuse: expected 1 StartSession call, got %d", tracker.StartCalls)
	}
	if tracker.ResetCalls != 0 {
		t.Errorf("ReuseModeReuse: expected 0 Reset calls, got %d", tracker.ResetCalls)
	}
	// 1 Close from defer cleanup at end of Execute()
	if tracker.CloseCalls != 1 {
		t.Errorf("ReuseModeReuse: expected 1 Close call (cleanup), got %d", tracker.CloseCalls)
	}
	// Should have run all 3 instructions (entry probe might be skipped for navigate-first)
	if tracker.RunCalls < 3 {
		t.Errorf("ReuseModeReuse: expected at least 3 Run calls, got %d", tracker.RunCalls)
	}
}

// TestSessionReuseModeClean verifies that "clean" mode calls Reset after each step
func TestSessionReuseModeClean(t *testing.T) {
	tracker := &sessionTracker{}
	eng := &trackingEngine{tracker: tracker}

	plan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "step-1", Type: "navigate", Params: map[string]any{"url": "https://example.com"}},
			{Index: 1, NodeID: "step-2", Type: "click", Params: map[string]any{"selector": "#btn"}},
			{Index: 2, NodeID: "step-3", Type: "wait", Params: map[string]any{"selector": "#result"}},
		},
		CreatedAt: time.Now().UTC(),
	}

	exec := NewSimpleExecutor(nil)
	memSink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	req := Request{
		Plan:          plan,
		EngineName:    eng.Name(),
		EngineFactory: engine.NewStaticFactory(eng),
		Recorder:      &stubRecorder{},
		EventSink:     memSink,
		ReuseMode:     engine.ReuseModeClean, // Clean mode - resets state between steps
	}

	err := exec.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify session behavior for "clean" mode:
	// - Should create exactly 1 session
	// - Should call Reset AFTER each step (N times for N steps)
	// - Should call Close once at the end (defer cleanup)
	if tracker.StartCalls != 1 {
		t.Errorf("ReuseModeClean: expected 1 StartSession call, got %d", tracker.StartCalls)
	}
	// Reset is called AFTER each step (including the last one)
	expectedResets := 3 // For 3 steps, reset called after each step
	if tracker.ResetCalls != expectedResets {
		t.Errorf("ReuseModeClean: expected %d Reset calls, got %d", expectedResets, tracker.ResetCalls)
	}
	// 1 Close from defer cleanup at end
	if tracker.CloseCalls != 1 {
		t.Errorf("ReuseModeClean: expected 1 Close call (cleanup), got %d", tracker.CloseCalls)
	}
}

// TestSessionReuseModeFresh verifies that "fresh" mode creates a new session for each step
func TestSessionReuseModeFresh(t *testing.T) {
	tracker := &sessionTracker{}
	eng := &trackingEngine{tracker: tracker}

	plan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "step-1", Type: "navigate", Params: map[string]any{"url": "https://example.com"}},
			{Index: 1, NodeID: "step-2", Type: "click", Params: map[string]any{"selector": "#btn"}},
			{Index: 2, NodeID: "step-3", Type: "wait", Params: map[string]any{"selector": "#result"}},
		},
		CreatedAt: time.Now().UTC(),
	}

	exec := NewSimpleExecutor(nil)
	memSink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	req := Request{
		Plan:          plan,
		EngineName:    eng.Name(),
		EngineFactory: engine.NewStaticFactory(eng),
		Recorder:      &stubRecorder{},
		EventSink:     memSink,
		ReuseMode:     engine.ReuseModeFresh, // Fresh mode - new session each step
	}

	err := exec.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify session behavior for "fresh" mode:
	// - Should create a new session for each step (N sessions for N steps)
	// - Should call Close after each step (N times - session is nil after last so no defer)
	// - Should NOT call Reset
	if tracker.StartCalls != 3 {
		t.Errorf("ReuseModeFresh: expected 3 StartSession calls (one per step), got %d", tracker.StartCalls)
	}
	if tracker.ResetCalls != 0 {
		t.Errorf("ReuseModeFresh: expected 0 Reset calls, got %d", tracker.ResetCalls)
	}
	// Close is called after each step; session becomes nil so defer doesn't add another
	expectedCloses := 3 // For 3 steps, close called after each step
	if tracker.CloseCalls != expectedCloses {
		t.Errorf("ReuseModeFresh: expected %d Close calls, got %d", expectedCloses, tracker.CloseCalls)
	}

	// Verify unique sessions were created
	if len(tracker.Sessions) != 3 {
		t.Errorf("ReuseModeFresh: expected 3 unique sessions, got %d", len(tracker.Sessions))
	}
}

// TestSessionReuseModeFromMetadata verifies mode can be set via plan metadata
func TestSessionReuseModeFromMetadata(t *testing.T) {
	tracker := &sessionTracker{}
	eng := &trackingEngine{tracker: tracker}

	plan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "step-1", Type: "navigate", Params: map[string]any{"url": "https://example.com"}},
			{Index: 1, NodeID: "step-2", Type: "click", Params: map[string]any{"selector": "#btn"}},
		},
		Metadata: map[string]any{
			"sessionReuseMode": "clean", // Set via metadata
		},
		CreatedAt: time.Now().UTC(),
	}

	exec := NewSimpleExecutor(nil)
	memSink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	req := Request{
		Plan:          plan,
		EngineName:    eng.Name(),
		EngineFactory: engine.NewStaticFactory(eng),
		Recorder:      &stubRecorder{},
		EventSink:     memSink,
		// ReuseMode NOT set - should read from metadata
	}

	err := exec.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should behave like clean mode when set via metadata
	if tracker.StartCalls != 1 {
		t.Errorf("Metadata clean mode: expected 1 StartSession call, got %d", tracker.StartCalls)
	}
	// Reset called after each step (2 steps = 2 resets)
	if tracker.ResetCalls != 2 {
		t.Errorf("Metadata clean mode: expected 2 Reset calls, got %d", tracker.ResetCalls)
	}
	// 1 Close from defer cleanup
	if tracker.CloseCalls != 1 {
		t.Errorf("Metadata clean mode: expected 1 Close call, got %d", tracker.CloseCalls)
	}
}

// TestSessionReuseModeDefaultsToReuse verifies default behavior is "reuse"
func TestSessionReuseModeDefaultsToReuse(t *testing.T) {
	tracker := &sessionTracker{}
	eng := &trackingEngine{tracker: tracker}

	plan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "step-1", Type: "navigate", Params: map[string]any{"url": "https://example.com"}},
			{Index: 1, NodeID: "step-2", Type: "click", Params: map[string]any{"selector": "#btn"}},
		},
		CreatedAt: time.Now().UTC(),
	}

	exec := NewSimpleExecutor(nil)
	memSink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	req := Request{
		Plan:          plan,
		EngineName:    eng.Name(),
		EngineFactory: engine.NewStaticFactory(eng),
		Recorder:      &stubRecorder{},
		EventSink:     memSink,
		// No ReuseMode set, no metadata - should default to "reuse"
	}

	err := exec.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should default to reuse behavior: same session, no resets, cleanup at end
	if tracker.StartCalls != 1 {
		t.Errorf("Default mode: expected 1 StartSession call, got %d", tracker.StartCalls)
	}
	if tracker.ResetCalls != 0 {
		t.Errorf("Default mode: expected 0 Reset calls, got %d", tracker.ResetCalls)
	}
	// 1 Close from defer cleanup at end
	if tracker.CloseCalls != 1 {
		t.Errorf("Default mode: expected 1 Close call (cleanup), got %d", tracker.CloseCalls)
	}
}
