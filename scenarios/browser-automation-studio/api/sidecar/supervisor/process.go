package supervisor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// Process abstracts process operations for testing.
// The production implementation wraps exec.Cmd.
type Process interface {
	// Start starts the process.
	Start() error

	// Stop stops the process gracefully.
	// It sends SIGTERM and waits up to gracePeriod for exit.
	// If the process doesn't exit, it sends SIGKILL.
	Stop(gracePeriod time.Duration) error

	// Wait blocks until the process exits and returns any error.
	// Returns nil if the process exits with code 0.
	Wait() error

	// Pid returns the process ID, or 0 if not running.
	Pid() int

	// Running returns true if the process is currently running.
	Running() bool

	// ExitChan returns a channel that's closed when the process exits.
	// This allows callers to be notified of process exit without blocking on Wait.
	ExitChan() <-chan struct{}
}

// NodeProcess is a Process implementation that runs a Node.js script.
// It mirrors the service.json pattern: "cd playwright-driver && node dist/server.js"
type NodeProcess struct {
	nodePath  string
	driverDir string
	script    string
	port      int
	log       *logrus.Logger

	cmd      *exec.Cmd
	exitChan chan struct{}
	exitErr  error
	mu       sync.Mutex
	running  bool
}

// NewNodeProcess creates a new NodeProcess.
//
// Parameters:
//   - nodePath: path to the node binary (e.g., "node" or "/usr/bin/node")
//   - driverDir: directory containing the playwright-driver (e.g., "playwright-driver")
//   - script: script to run within driverDir (e.g., "dist/server.js")
//   - port: port for the driver to listen on
//   - log: logger for process events
func NewNodeProcess(nodePath, driverDir, script string, port int, log *logrus.Logger) *NodeProcess {
	return &NodeProcess{
		nodePath:  nodePath,
		driverDir: driverDir,
		script:    script,
		port:      port,
		log:       log,
	}
}

// Start starts the Node.js process.
func (p *NodeProcess) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("process already running")
	}

	// Resolve the working directory relative to the API binary location
	// The API binary is at: <scenario>/api/browser-automation-studio-api
	// playwright-driver is at: <scenario>/playwright-driver
	workDir, err := p.resolveDriverDir()
	if err != nil {
		return fmt.Errorf("failed to resolve driver directory: %w", err)
	}

	// Verify the script exists
	scriptPath := filepath.Join(workDir, p.script)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("driver script not found: %s", scriptPath)
	}

	// Create the command
	p.cmd = exec.Command(p.nodePath, p.script)
	p.cmd.Dir = workDir

	// Set environment variables
	p.cmd.Env = append(os.Environ(),
		fmt.Sprintf("PLAYWRIGHT_DRIVER_PORT=%d", p.port),
	)

	// Capture stdout/stderr for logging
	p.cmd.Stdout = &logWriter{log: p.log, level: logrus.InfoLevel, prefix: "[playwright-driver]"}
	p.cmd.Stderr = &logWriter{log: p.log, level: logrus.WarnLevel, prefix: "[playwright-driver]"}

	// Start the process
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	p.running = true
	p.exitChan = make(chan struct{})

	// Monitor process exit in background
	go p.waitForExit()

	p.log.WithFields(logrus.Fields{
		"pid":  p.cmd.Process.Pid,
		"port": p.port,
		"dir":  workDir,
	}).Info("Started playwright-driver process")

	return nil
}

// waitForExit waits for the process to exit and updates state.
func (p *NodeProcess) waitForExit() {
	err := p.cmd.Wait()

	p.mu.Lock()
	p.running = false
	p.exitErr = err
	close(p.exitChan)
	p.mu.Unlock()

	if err != nil {
		p.log.WithError(err).Warn("playwright-driver process exited with error")
	} else {
		p.log.Info("playwright-driver process exited normally")
	}
}

// Stop stops the process gracefully.
func (p *NodeProcess) Stop(gracePeriod time.Duration) error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	cmd := p.cmd
	exitChan := p.exitChan
	p.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pid := cmd.Process.Pid
	p.log.WithField("pid", pid).Info("Stopping playwright-driver process")

	// Send SIGTERM
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// Process may have already exited
		if !isProcessExited(err) {
			p.log.WithError(err).Warn("Failed to send SIGTERM")
		}
	}

	// Wait for graceful exit or timeout
	select {
	case <-exitChan:
		p.log.WithField("pid", pid).Info("Process exited gracefully")
		return nil
	case <-time.After(gracePeriod):
		p.log.WithField("pid", pid).Warn("Grace period expired, sending SIGKILL")
	}

	// Send SIGKILL
	if err := cmd.Process.Kill(); err != nil {
		if !isProcessExited(err) {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Wait for the kill to take effect
	<-exitChan
	p.log.WithField("pid", pid).Info("Process killed")

	return nil
}

// Wait blocks until the process exits.
func (p *NodeProcess) Wait() error {
	p.mu.Lock()
	exitChan := p.exitChan
	p.mu.Unlock()

	if exitChan == nil {
		return fmt.Errorf("process not started")
	}

	<-exitChan

	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitErr
}

// Pid returns the process ID.
func (p *NodeProcess) Pid() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

// Running returns true if the process is running.
func (p *NodeProcess) Running() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// ExitChan returns a channel that's closed when the process exits.
func (p *NodeProcess) ExitChan() <-chan struct{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitChan
}

// isProcessExited returns true if the error indicates the process has already exited.
func isProcessExited(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "os: process already finished" ||
		err.Error() == "process already finished"
}

// resolveDriverDir resolves the driver directory path.
// If the path is absolute, it's used as-is.
// If relative, it's resolved relative to the API binary's parent directory
// (the scenario root), falling back to the current working directory.
func (p *NodeProcess) resolveDriverDir() (string, error) {
	// If already absolute, use as-is
	if filepath.IsAbs(p.driverDir) {
		return p.driverDir, nil
	}

	// Try to resolve relative to the API binary location
	// The API binary is typically at: <scenario>/api/browser-automation-studio-api
	// So we go up one level to get to <scenario>/
	if execPath, err := os.Executable(); err == nil {
		// Get the directory containing the executable (api/)
		execDir := filepath.Dir(execPath)
		// Go up one level to the scenario root
		scenarioRoot := filepath.Dir(execDir)
		// Construct the full path
		candidatePath := filepath.Join(scenarioRoot, p.driverDir)
		if absPath, err := filepath.Abs(candidatePath); err == nil {
			// Check if this path exists
			if _, err := os.Stat(absPath); err == nil {
				p.log.WithField("path", absPath).Debug("Resolved driver directory relative to executable")
				return absPath, nil
			}
		}
	}

	// Fall back to resolving relative to current working directory
	absPath, err := filepath.Abs(p.driverDir)
	if err != nil {
		return "", err
	}

	p.log.WithField("path", absPath).Debug("Resolved driver directory relative to cwd")
	return absPath, nil
}

// logWriter is an io.Writer that logs each line to logrus.
type logWriter struct {
	log    *logrus.Logger
	level  logrus.Level
	prefix string
	buf    []byte
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)

	// Process complete lines
	for {
		idx := -1
		for i, b := range w.buf {
			if b == '\n' {
				idx = i
				break
			}
		}
		if idx == -1 {
			break
		}

		line := string(w.buf[:idx])
		w.buf = w.buf[idx+1:]

		if len(line) > 0 {
			w.log.Log(w.level, w.prefix+" "+line)
		}
	}

	return len(p), nil
}

// MockProcess is a Process implementation for testing.
type MockProcess struct {
	StartErr      error
	StopErr       error
	WaitErr       error
	StartCalled   int
	StopCalled    int
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

	m.StartCalled++
	if m.StartErr != nil {
		return m.StartErr
	}

	m.running = true
	m.exitChan = make(chan struct{})

	// Monitor for simulated crash
	go func() {
		select {
		case <-m.SimulateCrash:
			m.mu.Lock()
			m.running = false
			close(m.exitChan)
			m.mu.Unlock()
		case <-m.exitChan:
			// Normal exit
		}
	}()

	return nil
}

// Stop implements Process.
func (m *MockProcess) Stop(gracePeriod time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StopCalled++
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

// compile-time checks
var (
	_ Process = (*NodeProcess)(nil)
	_ Process = (*MockProcess)(nil)
)
