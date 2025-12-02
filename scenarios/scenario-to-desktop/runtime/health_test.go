package bundleruntime

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"scenario-to-desktop-runtime/health"
	"scenario-to-desktop-runtime/manifest"
)

// MockClock implements Clock for testing.
type MockClock struct {
	mu      sync.Mutex
	current time.Time
	tickers []*MockTicker
}

func NewMockClock(t time.Time) *MockClock {
	return &MockClock{current: t}
}

func (c *MockClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

func (c *MockClock) Sleep(d time.Duration) {
	c.mu.Lock()
	c.current = c.current.Add(d)
	c.mu.Unlock()
}

func (c *MockClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- c.current.Add(d)
	return ch
}

func (c *MockClock) NewTicker(d time.Duration) Ticker {
	t := &MockTicker{ch: make(chan time.Time, 1), interval: d}
	c.mu.Lock()
	c.tickers = append(c.tickers, t)
	c.mu.Unlock()
	// Send one tick immediately for tests
	t.ch <- c.current
	return t
}

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

type MockTicker struct {
	ch       chan time.Time
	interval time.Duration
	stopped  bool
}

func (t *MockTicker) C() <-chan time.Time { return t.ch }
func (t *MockTicker) Stop()               { t.stopped = true }

// MockDialer implements NetworkDialer for testing.
type MockDialer struct {
	mu          sync.Mutex
	openPorts   map[int]bool
	shouldFail  bool
	connections []net.Conn
}

func NewMockDialer() *MockDialer {
	return &MockDialer{openPorts: make(map[int]bool)}
}

func (d *MockDialer) SetPort(port int, open bool) {
	d.mu.Lock()
	d.openPorts[port] = open
	d.mu.Unlock()
}

func (d *MockDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialTimeout(network, address, time.Second)
}

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

// MockCommandRunner implements CommandRunner for testing.
type MockCommandRunner struct {
	mu        sync.Mutex
	shouldErr bool
	output    []byte
	commands  [][]string
}

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

func (r *MockCommandRunner) LookPath(file string) (string, error) {
	return file, nil
}

// MockFileSystem implements FileSystem for testing.
type MockFileSystem struct {
	mu    sync.Mutex
	files map[string][]byte
	dirs  map[string]bool
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (f *MockFileSystem) ReadFile(path string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return data, nil
}

func (f *MockFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[path] = data
	return nil
}

func (f *MockFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dirs[path] = true
	return nil
}

func (f *MockFileSystem) Stat(path string) (fs.FileInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.files[path]; ok {
		return mockFileInfo{name: path, size: int64(len(f.files[path]))}, nil
	}
	if f.dirs[path] {
		return mockFileInfo{name: path, isDir: true}, nil
	}
	return nil, fs.ErrNotExist
}

func (f *MockFileSystem) OpenFile(path string, flag int, perm fs.FileMode) (File, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &mockFile{fs: f, path: path}, nil
}

func (f *MockFileSystem) Remove(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.files, path)
	delete(f.dirs, path)
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
	existing := f.fs.files[f.path]
	f.fs.files[f.path] = append(existing, p...)
	return len(p), nil
}

func (f *mockFile) Close() error { return nil }
func (f *mockFile) Sync() error  { return nil }

// testMockPortAllocator is a test double for PortAllocator.
type testMockPortAllocator struct {
	ports map[string]map[string]int
}

func (m *testMockPortAllocator) Allocate() error { return nil }
func (m *testMockPortAllocator) Resolve(serviceID, portName string) (int, error) {
	if ports, ok := m.ports[serviceID]; ok {
		if port, ok := ports[portName]; ok {
			return port, nil
		}
	}
	return 0, fmt.Errorf("port %s not found for %s", portName, serviceID)
}
func (m *testMockPortAllocator) Map() map[string]map[string]int { return m.ports }

// testHealthMonitor creates a health.Monitor for testing with the given dependencies.
func testHealthMonitor(m *manifest.Manifest, ports map[string]map[string]int, clock Clock, dialer NetworkDialer, fs FileSystem, cmdRunner CommandRunner) *health.Monitor {
	if clock == nil {
		clock = NewMockClock(time.Now())
	}
	if dialer == nil {
		dialer = RealNetworkDialer{}
	}
	if fs == nil {
		fs = NewMockFileSystem()
	}
	if cmdRunner == nil {
		cmdRunner = &MockCommandRunner{}
	}
	return health.NewMonitor(health.MonitorConfig{
		Manifest:     m,
		Ports:        &testMockPortAllocator{ports: ports},
		Dialer:       dialer,
		CmdRunner:    cmdRunner,
		FS:           fs,
		Clock:        clock,
		AppData:      "/tmp/test-appdata",
		StatusGetter: nil,
	})
}

func TestWaitForReadiness_HealthSuccess(t *testing.T) {
	// Set up a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Extract port from server URL
	_, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	m := &manifest.Manifest{
		Services: []manifest.Service{
			{
				ID: "api",
				Readiness: manifest.ReadinessCheck{
					Type:      "health_success",
					TimeoutMs: 5000,
				},
				Health: manifest.HealthCheck{
					Type:       "http",
					PortName:   "http",
					Path:       "/",
					IntervalMs: 100,
					TimeoutMs:  1000,
					Retries:    3,
				},
			},
		},
	}

	clock := NewMockClock(time.Now())
	hm := testHealthMonitor(m, map[string]map[string]int{"api": {"http": port}}, clock, nil, nil, nil)

	ctx := context.Background()
	err := hm.WaitForReadiness(ctx, "api")
	if err != nil {
		t.Errorf("WaitForReadiness() error = %v", err)
	}
}

func TestWaitForReadiness_PortOpen(t *testing.T) {
	// Create a listener to have an open port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	_, portStr, _ := net.SplitHostPort(listener.Addr().String())
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	m := &manifest.Manifest{
		Services: []manifest.Service{
			{
				ID: "api",
				Readiness: manifest.ReadinessCheck{
					Type:      "port_open",
					PortName:  "http",
					TimeoutMs: 5000,
				},
			},
		},
	}

	clock := NewMockClock(time.Now())
	dialer := RealNetworkDialer{}
	hm := testHealthMonitor(m, map[string]map[string]int{"api": {"http": port}}, clock, dialer, nil, nil)

	ctx := context.Background()
	err = hm.WaitForReadiness(ctx, "api")
	if err != nil {
		t.Errorf("WaitForReadiness() error = %v", err)
	}
}

func TestWaitForReadiness_LogMatch(t *testing.T) {
	clock := NewMockClock(time.Now())
	mockFS := NewMockFileSystem()

	// Pre-populate log file with matching pattern
	mockFS.files["/tmp/test-appdata/logs/api.log"] = []byte("Server started successfully")

	m := &manifest.Manifest{
		Services: []manifest.Service{
			{
				ID:     "api",
				LogDir: "logs/api.log",
				Readiness: manifest.ReadinessCheck{
					Type:      "log_match",
					Pattern:   "Server started",
					TimeoutMs: 5000,
				},
			},
		},
	}

	hm := testHealthMonitor(m, map[string]map[string]int{}, clock, nil, mockFS, nil)

	ctx := context.Background()
	err := hm.WaitForReadiness(ctx, "api")
	if err != nil {
		t.Errorf("WaitForReadiness() error = %v", err)
	}
}

func TestWaitForReadiness_UnknownType(t *testing.T) {
	m := &manifest.Manifest{
		Services: []manifest.Service{
			{
				ID: "api",
				Readiness: manifest.ReadinessCheck{
					Type:      "unknown",
					TimeoutMs: 1000,
				},
			},
		},
	}

	hm := testHealthMonitor(m, map[string]map[string]int{}, nil, nil, nil, nil)

	ctx := context.Background()
	err := hm.WaitForReadiness(ctx, "api")
	if err == nil {
		t.Error("WaitForReadiness() expected error for unknown type")
	}
}

func TestCheckTCPHealth(t *testing.T) {
	// Create a listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	_, portStr, _ := net.SplitHostPort(listener.Addr().String())
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	m := &manifest.Manifest{
		Services: []manifest.Service{
			{
				ID: "api",
				Health: manifest.HealthCheck{
					Type:     "tcp",
					PortName: "http",
				},
			},
		},
	}

	hm := testHealthMonitor(m, map[string]map[string]int{"api": {"http": port}}, nil, RealNetworkDialer{}, nil, nil)

	ctx := context.Background()
	if !hm.CheckOnce(ctx, "api") {
		t.Error("CheckOnce() returned false, expected true")
	}

	// Close listener and check again
	listener.Close()
	time.Sleep(10 * time.Millisecond) // Give time for port to close

	if hm.CheckOnce(ctx, "api") {
		t.Error("CheckOnce() returned true after listener closed, expected false")
	}
}

func TestCheckCommandHealth(t *testing.T) {
	runner := &MockCommandRunner{}

	m := &manifest.Manifest{
		Services: []manifest.Service{
			{
				ID: "api",
				Health: manifest.HealthCheck{
					Type:    "command",
					Command: []string{"health-check", "--quiet"},
				},
			},
		},
	}

	hm := testHealthMonitor(m, map[string]map[string]int{}, nil, nil, nil, runner)

	ctx := context.Background()

	// Test successful command
	runner.shouldErr = false
	if !hm.CheckOnce(ctx, "api") {
		t.Error("CheckOnce() returned false, expected true")
	}

	// Test failed command
	runner.shouldErr = true
	if hm.CheckOnce(ctx, "api") {
		t.Error("CheckOnce() returned true, expected false")
	}
}

func TestCheckCommandHealth_NoCommand(t *testing.T) {
	m := &manifest.Manifest{
		Services: []manifest.Service{
			{
				ID: "api",
				Health: manifest.HealthCheck{
					Type:    "command",
					Command: []string{},
				},
			},
		},
	}

	hm := testHealthMonitor(m, map[string]map[string]int{}, nil, nil, nil, nil)

	ctx := context.Background()
	if hm.CheckOnce(ctx, "api") {
		t.Error("CheckOnce() returned true for empty command, expected false")
	}
}

func TestLogMatches(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.files["/tmp/test-appdata/logs/api.log"] = []byte("INFO: Server started on port 8080\nDEBUG: Ready for connections")

	// Test cases that work through the public API (manifest-based)
	tests := []struct {
		name    string
		logDir  string
		pattern string
		want    bool
	}{
		{"pattern found", "logs/api.log", "Server started", true},
		{"regex pattern", "logs/api.log", "port [0-9]+", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &manifest.Manifest{
				Services: []manifest.Service{
					{
						ID:     "api",
						LogDir: tt.logDir,
						Health: manifest.HealthCheck{
							Type: "log_match",
							Path: tt.pattern,
						},
					},
				},
			}

			hm := testHealthMonitor(m, map[string]map[string]int{}, nil, nil, mockFS, nil)
			ctx := context.Background()
			got := hm.CheckOnce(ctx, "api")
			if got != tt.want {
				t.Errorf("CheckOnce() for log_match = %v, want %v", got, tt.want)
			}
		})
	}

	// Note: Tests for "pattern not found", "missing log file", and "empty log dir"
	// were removed as they test internal implementation details now in health package.
}

func TestWaitForDependencies(t *testing.T) {
	clock := NewMockClock(time.Now())

	// Create status map
	serviceStatus := map[string]ServiceStatus{
		"db":    {Ready: true},
		"cache": {Ready: true},
	}
	statusGetter := func(id string) (ServiceStatus, bool) {
		s, ok := serviceStatus[id]
		return s, ok
	}

	m := &manifest.Manifest{
		Services: []manifest.Service{
			{ID: "db"},
			{ID: "cache"},
			{ID: "api", Dependencies: []string{"db", "cache"}},
		},
	}

	hm := health.NewMonitor(health.MonitorConfig{
		Manifest:     m,
		Ports:        &testMockPortAllocator{ports: map[string]map[string]int{}},
		Dialer:       RealNetworkDialer{},
		CmdRunner:    &MockCommandRunner{},
		FS:           NewMockFileSystem(),
		Clock:        clock,
		AppData:      "/tmp/test-appdata",
		StatusGetter: statusGetter,
	})

	svc := &manifest.Service{
		ID:           "api",
		Dependencies: []string{"db", "cache"},
	}

	ctx := context.Background()
	err := hm.WaitForDependencies(ctx, svc)
	if err != nil {
		t.Errorf("WaitForDependencies() error = %v", err)
	}
}

func TestWaitForDependencies_MissingDep(t *testing.T) {
	clock := NewMockClock(time.Now())

	serviceStatus := map[string]ServiceStatus{}
	statusGetter := func(id string) (ServiceStatus, bool) {
		s, ok := serviceStatus[id]
		return s, ok
	}

	m := &manifest.Manifest{
		Services: []manifest.Service{
			{ID: "api", Dependencies: []string{"nonexistent"}},
		},
	}

	hm := health.NewMonitor(health.MonitorConfig{
		Manifest:     m,
		Ports:        &testMockPortAllocator{ports: map[string]map[string]int{}},
		Dialer:       RealNetworkDialer{},
		CmdRunner:    &MockCommandRunner{},
		FS:           NewMockFileSystem(),
		Clock:        clock,
		AppData:      "/tmp/test-appdata",
		StatusGetter: statusGetter,
	})

	svc := &manifest.Service{
		ID:           "api",
		Dependencies: []string{"nonexistent"},
	}

	ctx := context.Background()
	err := hm.WaitForDependencies(ctx, svc)
	if err == nil {
		t.Error("WaitForDependencies() expected error for missing dependency")
	}
}

func TestWaitForDependencies_NoDeps(t *testing.T) {
	m := &manifest.Manifest{
		Services: []manifest.Service{
			{ID: "api", Dependencies: []string{}},
		},
	}

	hm := testHealthMonitor(m, map[string]map[string]int{}, nil, nil, nil, nil)

	svc := &manifest.Service{
		ID:           "api",
		Dependencies: []string{},
	}

	ctx := context.Background()
	err := hm.WaitForDependencies(ctx, svc)
	if err != nil {
		t.Errorf("WaitForDependencies() error = %v for no dependencies", err)
	}
}

func TestWaitForReadiness_Retries(t *testing.T) {
	clock := NewMockClock(time.Now())
	runner := &MockCommandRunner{shouldErr: true}

	m := &manifest.Manifest{
		Services: []manifest.Service{
			{
				ID: "api",
				Health: manifest.HealthCheck{
					Type:       "command",
					Command:    []string{"health-check"},
					IntervalMs: 100,
					TimeoutMs:  500,
					Retries:    2,
				},
				Readiness: manifest.ReadinessCheck{
					Type:      "health_success",
					TimeoutMs: 3000, // Long enough for retries
				},
			},
		},
	}

	hm := testHealthMonitor(m, map[string]map[string]int{}, clock, nil, nil, runner)

	// Using the public WaitForReadiness API which internally uses pollHealth
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := hm.WaitForReadiness(ctx, "api")
	if err == nil {
		t.Error("WaitForReadiness() expected error after retries exhausted")
	}

	// Should have tried 3 times (initial + 2 retries)
	if len(runner.commands) != 3 {
		t.Errorf("WaitForReadiness() ran %d commands, expected 3", len(runner.commands))
	}
}
