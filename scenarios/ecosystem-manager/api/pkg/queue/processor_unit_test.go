package queue

import (
	"testing"
	"time"

	"github.com/ecosystem-manager/api/pkg/agentmanager"
	"github.com/ecosystem-manager/api/pkg/recycler"
	"github.com/ecosystem-manager/api/pkg/settings"
)

// MockRecycler implements recycler.RecyclerAPI for testing
type MockRecycler struct {
	EnqueuedTaskIDs []string
	WakeFunc        func()
	Started         bool
	Stopped         bool
	WakeCalls       int
}

// Compile-time assertion that MockRecycler implements recycler.RecyclerAPI
var _ recycler.RecyclerAPI = (*MockRecycler)(nil)

func (m *MockRecycler) Enqueue(taskID string) {
	m.EnqueuedTaskIDs = append(m.EnqueuedTaskIDs, taskID)
}

func (m *MockRecycler) SetWakeFunc(fn func()) {
	m.WakeFunc = fn
}

func (m *MockRecycler) Start() {
	m.Started = true
}

func (m *MockRecycler) Stop() {
	m.Stopped = true
}

func (m *MockRecycler) Wake() {
	m.WakeCalls++
}

func (m *MockRecycler) OnSettingsUpdated(previous, next settings.Settings) {}

func (m *MockRecycler) Stats() recycler.Stats {
	return recycler.Stats{}
}

// MockExecutionRegistry implements ExecutionRegistryAPI for testing
type MockExecutionRegistry struct {
	registrations map[string]*taskExecution
	runIDs        map[string]string
}

// Compile-time assertion that MockExecutionRegistry implements ExecutionRegistryAPI
var _ ExecutionRegistryAPI = (*MockExecutionRegistry)(nil)

func NewMockExecutionRegistry() *MockExecutionRegistry {
	return &MockExecutionRegistry{
		registrations: make(map[string]*taskExecution),
		runIDs:        make(map[string]string),
	}
}

func (m *MockExecutionRegistry) ReserveExecution(taskID, agentID string, startedAt time.Time) {
	m.registrations[taskID] = &taskExecution{taskID: taskID, agentTag: agentID, started: startedAt}
}

func (m *MockExecutionRegistry) RegisterRunID(taskID, runID string) {
	m.runIDs[taskID] = runID
}

func (m *MockExecutionRegistry) UnregisterExecution(taskID string) {
	delete(m.registrations, taskID)
	delete(m.runIDs, taskID)
}

func (m *MockExecutionRegistry) GetExecution(taskID string) (*taskExecution, bool) {
	exec, ok := m.registrations[taskID]
	return exec, ok
}

func (m *MockExecutionRegistry) IsTaskRunning(taskID string) bool {
	_, ok := m.registrations[taskID]
	return ok
}

func (m *MockExecutionRegistry) ListRunningTaskIDs() []string {
	ids := make([]string, 0, len(m.registrations))
	for id := range m.registrations {
		ids = append(ids, id)
	}
	return ids
}

func (m *MockExecutionRegistry) ListRunningTaskIDsAsMap() map[string]struct{} {
	result := make(map[string]struct{}, len(m.registrations))
	for id := range m.registrations {
		result[id] = struct{}{}
	}
	return result
}

func (m *MockExecutionRegistry) GetRunIDForTask(taskID string) string {
	return m.runIDs[taskID]
}

func (m *MockExecutionRegistry) Count() int {
	return len(m.registrations)
}

func (m *MockExecutionRegistry) GetAllExecutions() []taskExecutionSnapshot {
	snapshots := make([]taskExecutionSnapshot, 0, len(m.registrations))
	for taskID, exec := range m.registrations {
		snapshots = append(snapshots, taskExecutionSnapshot{
			TaskID:   taskID,
			AgentTag: exec.agentTag,
			Started:  exec.started,
		})
	}
	return snapshots
}

func (m *MockExecutionRegistry) GetTimedOutExecutions() []taskExecutionSnapshot {
	return nil
}

func (m *MockExecutionRegistry) MarkTimedOut(taskID string) {}

func (m *MockExecutionRegistry) Clear() int {
	count := len(m.registrations)
	m.registrations = make(map[string]*taskExecution)
	m.runIDs = make(map[string]string)
	return count
}

// TestProcessorWithMockedDependencies demonstrates that the Processor can be
// instantiated with all mocked dependencies for unit testing.
func TestProcessorWithMockedDependencies(t *testing.T) {
	// Create mock dependencies
	mockRegistry := NewMockExecutionRegistry()
	mockAgentSvc := agentmanager.NewMockAgentService()
	mockRecycler := &MockRecycler{}

	// Create broadcast channel
	broadcast := make(chan any, 10)
	defer close(broadcast)

	// Create processor with all mocked dependencies
	processor := NewProcessor(ProcessorDeps{
		Storage:   nil, // Would normally be real storage
		Assembler: nil, // Would normally be real assembler
		Broadcast: broadcast,
		Registry:  mockRegistry,
		AgentSvc:  mockAgentSvc,
		Recycler:  mockRecycler,
	})

	// Verify processor was created
	if processor == nil {
		t.Fatal("Expected processor to be created")
	}

	// Verify mock dependencies were injected by checking behavior
	// Test registry - reserve an execution and verify it's tracked
	processor.registry.ReserveExecution("test-task", "test-agent", time.Now())
	if !mockRegistry.IsTaskRunning("test-task") {
		t.Error("Expected mock registry to be injected - task not tracked")
	}

	// Test agent service - verify IsAvailable returns mock value
	if !processor.agentSvc.IsAvailable(nil) {
		t.Error("Expected mock agent service to be injected - IsAvailable returned false")
	}
	if mockAgentSvc.Calls.IsAvailable != 1 {
		t.Error("Expected mock agent service to track calls")
	}

	// Test recycler - enqueue and verify it was captured
	processor.recycler.Enqueue("another-task")
	if len(mockRecycler.EnqueuedTaskIDs) != 1 || mockRecycler.EnqueuedTaskIDs[0] != "another-task" {
		t.Error("Expected mock recycler to be injected - enqueue not captured")
	}
}

// TestProcessorRegistryInteraction tests that the processor correctly interacts
// with the execution registry via the interface.
func TestProcessorRegistryInteraction(t *testing.T) {
	mockRegistry := NewMockExecutionRegistry()
	broadcast := make(chan any, 10)
	defer close(broadcast)

	processor := NewProcessor(ProcessorDeps{
		Broadcast: broadcast,
		Registry:  mockRegistry,
	})

	// Test that we can reserve an execution through the processor's registry
	taskID := "test-task-123"
	agentTag := "test-agent"

	processor.registry.ReserveExecution(taskID, agentTag, time.Now())

	// Verify the registration was recorded
	if !mockRegistry.IsTaskRunning(taskID) {
		t.Errorf("Expected task %s to be registered", taskID)
	}

	// Test count
	if mockRegistry.Count() != 1 {
		t.Errorf("Expected 1 registered task, got %d", mockRegistry.Count())
	}

	// Test unregister
	processor.registry.UnregisterExecution(taskID)
	if mockRegistry.IsTaskRunning(taskID) {
		t.Errorf("Expected task %s to be unregistered", taskID)
	}
}

// TestProcessorAgentServiceInteraction tests that the processor correctly
// interacts with the agent service via the interface.
func TestProcessorAgentServiceInteraction(t *testing.T) {
	mockAgentSvc := agentmanager.NewMockAgentService()
	broadcast := make(chan any, 10)
	defer close(broadcast)

	processor := NewProcessor(ProcessorDeps{
		Broadcast: broadcast,
		AgentSvc:  mockAgentSvc,
	})

	// Verify agent service is accessible and mock values work
	if !processor.agentSvc.IsAvailable(nil) {
		t.Error("Expected mock agent service to report as available")
	}
	if mockAgentSvc.Calls.IsAvailable != 1 {
		t.Errorf("Expected 1 IsAvailable call, got %d", mockAgentSvc.Calls.IsAvailable)
	}
}

// TestNewProcessorDefaultsCreatesRegistry verifies that NewProcessor creates
// a default registry when none is provided.
func TestNewProcessorDefaultsCreatesRegistry(t *testing.T) {
	broadcast := make(chan any, 10)
	defer close(broadcast)

	processor := NewProcessor(ProcessorDeps{
		Broadcast: broadcast,
		// No Registry provided - should create default
	})

	if processor.registry == nil {
		t.Error("Expected default registry to be created")
	}

	// Verify it's a real ExecutionRegistry (implements the interface)
	_, ok := processor.registry.(*ExecutionRegistry)
	if !ok {
		t.Error("Expected default registry to be *ExecutionRegistry")
	}
}

// TestProcessorStopDoesNotPanicWithoutBackground verifies that Stop()
// works correctly even when background workers weren't started.
func TestProcessorStopDoesNotPanicWithoutBackground(t *testing.T) {
	broadcast := make(chan any, 10)
	defer close(broadcast)

	processor := NewProcessor(ProcessorDeps{
		Broadcast: broadcast,
	})

	// This should not panic - processor was created without starting workers
	processor.Stop()
}
