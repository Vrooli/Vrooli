package queue

import (
	"os/exec"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/settings"
)

// Execution represents a running task execution.
type Execution struct {
	TaskID    string
	AgentTag  string
	RunID     string    // agent-manager run ID
	Cmd       *exec.Cmd // Legacy fallback, nil when using agent-manager
	Started   time.Time
	TimeoutAt time.Time
	TimedOut  bool
}

// IsTimedOut returns true if the execution has exceeded its timeout.
func (e *Execution) IsTimedOut() bool {
	if e == nil {
		return false
	}
	if e.TimedOut {
		return true
	}
	if !e.TimeoutAt.IsZero() && time.Now().After(e.TimeoutAt) {
		return true
	}
	return false
}

// PID returns the process ID if available.
func (e *Execution) PID() int {
	if e == nil || e.Cmd == nil || e.Cmd.Process == nil {
		return 0
	}
	return e.Cmd.Process.Pid
}

// SlotSnapshot captures concurrency accounting for scheduling decisions.
type SlotSnapshot struct {
	Slots     int
	Running   int
	Available int
}

// ConcurrencyManager handles slot accounting and execution tracking.
type ConcurrencyManager struct {
	mu         sync.RWMutex
	executions map[string]*Execution
}

// NewConcurrencyManager creates a new ConcurrencyManager.
func NewConcurrencyManager() *ConcurrencyManager {
	return &ConcurrencyManager{
		executions: make(map[string]*Execution),
	}
}

// AcquireSlot reserves an execution slot for a task.
// Returns the execution record.
func (m *ConcurrencyManager) AcquireSlot(taskID, agentTag string, timeout time.Duration) *Execution {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var timeoutAt time.Time
	if timeout > 0 {
		timeoutAt = now.Add(timeout)
	}

	exec := &Execution{
		TaskID:    taskID,
		AgentTag:  agentTag,
		Started:   now,
		TimeoutAt: timeoutAt,
	}
	m.executions[taskID] = exec
	return exec
}

// UpdateRunID sets the agent-manager run ID for an execution.
func (m *ConcurrencyManager) UpdateRunID(taskID, runID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if exec, ok := m.executions[taskID]; ok {
		exec.RunID = runID
	}
}

// ReleaseSlot removes an execution from tracking.
func (m *ConcurrencyManager) ReleaseSlot(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.executions, taskID)
}

// GetExecution returns the execution for a task, if tracked.
func (m *ConcurrencyManager) GetExecution(taskID string) *Execution {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.executions[taskID]
}

// GetRunID returns the run ID for a task, empty if not found.
func (m *ConcurrencyManager) GetRunID(taskID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if exec, ok := m.executions[taskID]; ok {
		return exec.RunID
	}
	return ""
}

// IsRunning returns true if a task is currently tracked.
func (m *ConcurrencyManager) IsRunning(taskID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.executions[taskID]
	return ok
}

// GetActiveCount returns the number of currently tracked executions.
func (m *ConcurrencyManager) GetActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.executions)
}

// GetRunningTaskIDs returns the IDs of all currently tracked tasks.
func (m *ConcurrencyManager) GetRunningTaskIDs() map[string]struct{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]struct{}, len(m.executions))
	for taskID := range m.executions {
		result[taskID] = struct{}{}
	}
	return result
}

// GetTimedOutTasks returns executions that have exceeded their timeout.
func (m *ConcurrencyManager) GetTimedOutTasks() []*Execution {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var timedOut []*Execution
	for _, exec := range m.executions {
		if exec.IsTimedOut() {
			timedOut = append(timedOut, exec)
		}
	}
	return timedOut
}

// MarkTimedOut marks an execution as timed out.
func (m *ConcurrencyManager) MarkTimedOut(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if exec, ok := m.executions[taskID]; ok {
		exec.TimedOut = true
	}
}

// ComputeSlotSnapshot calculates available slots considering internal and external executions.
func (m *ConcurrencyManager) ComputeSlotSnapshot(externalActive map[string]struct{}) SlotSnapshot {
	m.mu.RLock()
	internalRunning := len(m.executions)
	internalIDs := make(map[string]struct{}, internalRunning)
	for taskID := range m.executions {
		internalIDs[taskID] = struct{}{}
	}
	m.mu.RUnlock()

	// Count external tasks not already tracked internally
	running := internalRunning
	for taskID := range externalActive {
		if _, tracked := internalIDs[taskID]; !tracked {
			running++
		}
	}

	slots := settings.GetSettings().Slots
	if slots <= 0 {
		slots = 1
	}
	available := slots - running
	if available < 0 {
		available = 0
	}

	return SlotSnapshot{
		Slots:     slots,
		Running:   running,
		Available: available,
	}
}
