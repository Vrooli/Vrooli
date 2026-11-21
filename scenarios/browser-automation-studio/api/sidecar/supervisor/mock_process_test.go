package supervisor

import (
	"sync"
	"sync/atomic"
	"time"
)

// MockProcess is a Process implementation for testing.
type MockProcess struct {
	StartErr      error
	StopErr       error
	WaitErr       error
	StartCalled   atomic.Int64
	StopCalled    atomic.Int64
	LastGraceMS   atomic.Int64
	SimulateCrash chan struct{} // Close to simulate a crash

	mu       sync.Mutex
	running  bool
	pid      int
	exitChan chan struct{}
}

// NewMockProcess creates a new MockProcess.
func NewMockProcess() *MockProcess {
	return &MockProcess{
		SimulateCrash: make(chan struct{}),
		pid:           12345,
	}
}

// Start implements Process.
func (m *MockProcess) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StartCalled.Add(1)
	if m.StartErr != nil {
		return m.StartErr
	}

	m.running = true
	m.exitChan = make(chan struct{})

	// Capture the current crash channel - this ensures each Start() invocation
	// listens to the crash channel that exists at that point in time.
	// Tests can reassign SimulateCrash between restarts and each restart
	// will pick up the new channel.
	crashChan := m.SimulateCrash
	exitChan := m.exitChan

	// Monitor for simulated crash
	go func() {
		select {
		case <-crashChan:
			m.mu.Lock()
			if m.running {
				m.running = false
				close(m.exitChan)
			}
			m.mu.Unlock()
		case <-exitChan:
			// Normal exit
		}
	}()

	return nil
}

// Stop implements Process.
func (m *MockProcess) Stop(gracePeriod time.Duration) error {
	m.LastGraceMS.Store(gracePeriod.Milliseconds())
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StopCalled.Add(1)
	if m.StopErr != nil {
		return m.StopErr
	}

	if m.running {
		m.running = false
		close(m.exitChan)
	}

	return nil
}

// Wait implements Process.
func (m *MockProcess) Wait() error {
	m.mu.Lock()
	exitChan := m.exitChan
	m.mu.Unlock()

	if exitChan != nil {
		<-exitChan
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	return m.WaitErr
}

// Pid implements Process.
func (m *MockProcess) Pid() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return 0
	}
	return m.pid
}

// Running implements Process.
func (m *MockProcess) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// ExitChan implements Process.
func (m *MockProcess) ExitChan() <-chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.exitChan
}

// TriggerCrash simulates an unexpected process crash.
// This is the preferred way to trigger crashes in tests as it properly
// closes the exit channel that the supervisor monitors.
func (m *MockProcess) TriggerCrash() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.running = false
		close(m.exitChan)
	}
}

// compile-time check
var _ Process = (*MockProcess)(nil)
