// Package testutil provides mock implementations for testing the bundle runtime.
package testutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"sync"
	"time"

	"scenario-to-desktop-runtime/infra"
)

// =============================================================================
// Clock Mocks
// =============================================================================

// MockClock implements infra.Clock for testing.
type MockClock struct {
	mu      sync.Mutex
	current time.Time
	tickers []*MockTicker
}

// NewMockClock creates a new MockClock initialized to the given time.
func NewMockClock(t time.Time) *MockClock {
	return &MockClock{current: t}
}

// Now returns the mock's current time.
func (c *MockClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

// Sleep advances the mock time by the given duration.
func (c *MockClock) Sleep(d time.Duration) {
	c.mu.Lock()
	c.current = c.current.Add(d)
	c.mu.Unlock()
}

// After returns a channel that receives the mock time after the duration.
func (c *MockClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- c.current.Add(d)
	return ch
}

// NewTicker creates a mock ticker.
func (c *MockClock) NewTicker(d time.Duration) infra.Ticker {
	t := &MockTicker{ch: make(chan time.Time, 1), interval: d}
	c.mu.Lock()
	c.tickers = append(c.tickers, t)
	c.mu.Unlock()
	// Send one tick immediately for tests
	t.ch <- c.current
	return t
}

// Advance moves the mock time forward and notifies all tickers.
func (c *MockClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.current = c.current.Add(d)
	for _, t := range c.tickers {
		select {
		case t.ch <- c.current:
		default:
		}
	}
	c.mu.Unlock()
}

// MockTicker implements infra.Ticker for testing.
type MockTicker struct {
	ch       chan time.Time
	interval time.Duration
	stopped  bool
}

// C returns the ticker channel.
func (t *MockTicker) C() <-chan time.Time { return t.ch }

// Stop marks the ticker as stopped.
func (t *MockTicker) Stop() { t.stopped = true }

// Ensure MockClock implements infra.Clock.
var _ infra.Clock = (*MockClock)(nil)

// =============================================================================
// Network Mocks
// =============================================================================

// MockDialer implements infra.NetworkDialer for testing.
type MockDialer struct {
	mu         sync.Mutex
	openPorts  map[int]bool
	shouldFail bool
}

// NewMockDialer creates a new MockDialer.
func NewMockDialer() *MockDialer {
	return &MockDialer{openPorts: make(map[int]bool)}
}

// SetPort configures whether a port appears open or closed.
func (d *MockDialer) SetPort(port int, open bool) {
	d.mu.Lock()
	d.openPorts[port] = open
	d.mu.Unlock()
}

// SetShouldFail configures the dialer to fail all connections.
func (d *MockDialer) SetShouldFail(fail bool) {
	d.mu.Lock()
	d.shouldFail = fail
	d.mu.Unlock()
}

// Dial implements net.Dial behavior.
func (d *MockDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialTimeout(network, address, time.Second)
}

// DialTimeout implements net.DialTimeout behavior.
func (d *MockDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.shouldFail {
		return nil, errors.New("dial failed")
	}
	// Parse port from address
	_, portStr, _ := net.SplitHostPort(address)
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil {
		if d.openPorts[port] {
			return &mockConn{}, nil
		}
	}
	return nil, errors.New("connection refused")
}

// Listen is not implemented for the mock.
func (d *MockDialer) Listen(network, address string) (net.Listener, error) {
	return nil, errors.New("not implemented")
}

type mockConn struct{}

func (c *mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (c *mockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (c *mockConn) Close() error                       { return nil }
func (c *mockConn) LocalAddr() net.Addr                { return nil }
func (c *mockConn) RemoteAddr() net.Addr               { return nil }
func (c *mockConn) SetDeadline(t time.Time) error      { return nil }
func (c *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *mockConn) SetWriteDeadline(t time.Time) error { return nil }

// Ensure MockDialer implements infra.NetworkDialer.
var _ infra.NetworkDialer = (*MockDialer)(nil)

// =============================================================================
// Command Mocks
// =============================================================================

// MockCommandRunner implements infra.CommandRunner for testing.
type MockCommandRunner struct {
	mu        sync.Mutex
	shouldErr bool
	output    []byte
	commands  [][]string
}

// NewMockCommandRunner creates a new MockCommandRunner.
func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{}
}

// SetOutput configures the output returned by Output calls.
func (r *MockCommandRunner) SetOutput(output []byte) {
	r.mu.Lock()
	r.output = output
	r.mu.Unlock()
}

// SetShouldErr configures whether commands should fail.
func (r *MockCommandRunner) SetShouldErr(shouldErr bool) {
	r.mu.Lock()
	r.shouldErr = shouldErr
	r.mu.Unlock()
}

// Commands returns all commands that were executed.
func (r *MockCommandRunner) Commands() [][]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.commands
}

// Run executes a command and returns an error if configured to fail.
func (r *MockCommandRunner) Run(ctx context.Context, name string, args []string) error {
	r.mu.Lock()
	r.commands = append(r.commands, append([]string{name}, args...))
	shouldErr := r.shouldErr
	r.mu.Unlock()
	if shouldErr {
		return errors.New("command failed")
	}
	return nil
}

// Output executes a command and returns the configured output.
func (r *MockCommandRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	r.mu.Lock()
	r.commands = append(r.commands, append([]string{name}, args...))
	output := r.output
	shouldErr := r.shouldErr
	r.mu.Unlock()
	if shouldErr {
		return nil, errors.New("command failed")
	}
	return output, nil
}

// LookPath returns the file path as-is.
func (r *MockCommandRunner) LookPath(file string) (string, error) {
	return file, nil
}

// Ensure MockCommandRunner implements infra.CommandRunner.
var _ infra.CommandRunner = (*MockCommandRunner)(nil)

// =============================================================================
// FileSystem Mocks
// =============================================================================

// MockFileSystem implements infra.FileSystem for testing.
type MockFileSystem struct {
	mu    sync.Mutex
	Files map[string][]byte
	Dirs  map[string]bool
}

// NewMockFileSystem creates a new MockFileSystem.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files: make(map[string][]byte),
		Dirs:  make(map[string]bool),
	}
}

// ReadFile reads a file from the mock filesystem.
func (f *MockFileSystem) ReadFile(path string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.Files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return data, nil
}

// WriteFile writes data to a file in the mock filesystem.
func (f *MockFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Files[path] = data
	return nil
}

// MkdirAll creates a directory in the mock filesystem.
func (f *MockFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Dirs[path] = true
	return nil
}

// Stat returns file info for a path.
func (f *MockFileSystem) Stat(path string) (fs.FileInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if data, ok := f.Files[path]; ok {
		return mockFileInfo{name: path, size: int64(len(data))}, nil
	}
	if f.Dirs[path] {
		return mockFileInfo{name: path, isDir: true}, nil
	}
	return nil, fs.ErrNotExist
}

// OpenFile opens a file for writing.
func (f *MockFileSystem) OpenFile(path string, flag int, perm fs.FileMode) (infra.File, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &mockFile{fs: f, path: path}, nil
}

// Remove removes a file or directory.
func (f *MockFileSystem) Remove(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.Files, path)
	delete(f.Dirs, path)
	return nil
}

type mockFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() fs.FileMode  { return 0o644 }
func (m mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() interface{}   { return nil }

type mockFile struct {
	fs   *MockFileSystem
	path string
}

func (f *mockFile) Write(p []byte) (int, error) {
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()
	existing := f.fs.Files[f.path]
	f.fs.Files[f.path] = append(existing, p...)
	return len(p), nil
}

func (f *mockFile) Close() error { return nil }
func (f *mockFile) Sync() error  { return nil }

// Ensure MockFileSystem implements infra.FileSystem.
var _ infra.FileSystem = (*MockFileSystem)(nil)

// =============================================================================
// Process Mocks
// =============================================================================

// MockProcess implements a mock process for testing.
type MockProcess struct {
	pid      int
	waitErr  error
	signaled bool
	killed   bool
	waitCh   chan struct{}
}

// NewMockProcess creates a new MockProcess with the given PID.
func NewMockProcess(pid int) *MockProcess {
	return &MockProcess{
		pid:    pid,
		waitCh: make(chan struct{}),
	}
}

// Pid returns the mock process ID.
func (p *MockProcess) Pid() int { return p.pid }

// Wait blocks until the process exits.
func (p *MockProcess) Wait() error {
	<-p.waitCh
	return p.waitErr
}

// Signal records that a signal was sent.
func (p *MockProcess) Signal(sig os.Signal) error {
	p.signaled = true
	return nil
}

// Kill records that kill was called and closes the wait channel.
func (p *MockProcess) Kill() error {
	p.killed = true
	// Only close channel once - use select to avoid panic
	select {
	case <-p.waitCh:
		// Already closed
	default:
		close(p.waitCh)
	}
	return nil
}

// Signaled returns whether Signal was called.
func (p *MockProcess) Signaled() bool { return p.signaled }

// Killed returns whether Kill was called.
func (p *MockProcess) Killed() bool { return p.killed }

// Exit simulates the process exiting.
func (p *MockProcess) Exit(err error) {
	p.waitErr = err
	select {
	case <-p.waitCh:
	default:
		close(p.waitCh)
	}
}

// MockProcessRunner implements process starting for testing.
type MockProcessRunner struct {
	mu           sync.Mutex
	processes    []*MockProcess
	shouldFail   bool
	startedCmds  []string
	currentIndex int
}

// NewMockProcessRunner creates a new MockProcessRunner.
func NewMockProcessRunner() *MockProcessRunner {
	return &MockProcessRunner{}
}

// SetProcesses configures the processes to return on Start calls.
func (r *MockProcessRunner) SetProcesses(procs []*MockProcess) {
	r.mu.Lock()
	r.processes = procs
	r.currentIndex = 0
	r.mu.Unlock()
}

// SetShouldFail configures whether Start should fail.
func (r *MockProcessRunner) SetShouldFail(fail bool) {
	r.mu.Lock()
	r.shouldFail = fail
	r.mu.Unlock()
}

// StartedCmds returns all commands that were started.
func (r *MockProcessRunner) StartedCmds() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.startedCmds
}

// Process is the interface for a running process.
type Process interface {
	Pid() int
	Wait() error
	Signal(sig os.Signal) error
	Kill() error
}

// Start launches a mock process.
func (r *MockProcessRunner) Start(ctx context.Context, cmd string, args []string, env []string, dir string, stdout, stderr io.Writer) (Process, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.startedCmds = append(r.startedCmds, cmd)
	if r.shouldFail {
		return nil, errors.New("start failed")
	}
	if r.currentIndex < len(r.processes) {
		proc := r.processes[r.currentIndex]
		r.currentIndex++
		return proc, nil
	}
	// Return a default process
	return NewMockProcess(12345), nil
}

// =============================================================================
// Port Allocator Mocks
// =============================================================================

// MockPortAllocator implements port allocation for testing.
type MockPortAllocator struct {
	Ports map[string]map[string]int
}

// NewMockPortAllocator creates a new MockPortAllocator.
func NewMockPortAllocator() *MockPortAllocator {
	return &MockPortAllocator{
		Ports: make(map[string]map[string]int),
	}
}

// SetPort configures a port for a service.
func (m *MockPortAllocator) SetPort(serviceID, portName string, port int) {
	if m.Ports[serviceID] == nil {
		m.Ports[serviceID] = make(map[string]int)
	}
	m.Ports[serviceID][portName] = port
}

// Allocate is a no-op for the mock.
func (m *MockPortAllocator) Allocate() error { return nil }

// Resolve returns the configured port.
func (m *MockPortAllocator) Resolve(serviceID, portName string) (int, error) {
	if ports, ok := m.Ports[serviceID]; ok {
		if port, ok := ports[portName]; ok {
			return port, nil
		}
	}
	return 0, fmt.Errorf("port %s not found for %s", portName, serviceID)
}

// Map returns a copy of all configured ports.
func (m *MockPortAllocator) Map() map[string]map[string]int {
	result := make(map[string]map[string]int)
	for svc, ports := range m.Ports {
		result[svc] = make(map[string]int)
		for name, port := range ports {
			result[svc][name] = port
		}
	}
	return result
}
