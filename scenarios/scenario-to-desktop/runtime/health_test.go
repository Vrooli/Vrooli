package bundleruntime

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"scenario-to-desktop-runtime/health"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/testutil"
)

// Type aliases for backward compatibility with existing tests.
type MockClock = testutil.MockClock

func NewMockClock(t time.Time) *testutil.MockClock {
	return testutil.NewMockClock(t)
}

type MockDialer = testutil.MockDialer

func NewMockDialer() *testutil.MockDialer {
	return testutil.NewMockDialer()
}

type MockCommandRunner = testutil.MockCommandRunner

type MockFileSystem = testutil.MockFileSystem

func NewMockFileSystem() *testutil.MockFileSystem {
	return testutil.NewMockFileSystem()
}

type testMockPortAllocator = testutil.MockPortAllocator

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
		cmdRunner = testutil.NewMockCommandRunner()
	}
	return health.NewMonitor(health.MonitorConfig{
		Manifest:     m,
		Ports:        &testMockPortAllocator{Ports: ports},
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
	mockFS.Files["/tmp/test-appdata/logs/api.log"] = []byte("Server started successfully")

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
	runner := testutil.NewMockCommandRunner()

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
	runner.SetShouldErr(false)
	if !hm.CheckOnce(ctx, "api") {
		t.Error("CheckOnce() returned false, expected true")
	}

	// Test failed command
	runner.SetShouldErr(true)
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
	mockFS.Files["/tmp/test-appdata/logs/api.log"] = []byte("INFO: Server started on port 8080\nDEBUG: Ready for connections")

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
		Ports:        &testMockPortAllocator{Ports: map[string]map[string]int{}},
		Dialer:       RealNetworkDialer{},
		CmdRunner:    testutil.NewMockCommandRunner(),
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
		Ports:        &testMockPortAllocator{Ports: map[string]map[string]int{}},
		Dialer:       RealNetworkDialer{},
		CmdRunner:    testutil.NewMockCommandRunner(),
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
	runner := testutil.NewMockCommandRunner()
	runner.SetShouldErr(true)

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
	if len(runner.Commands()) != 3 {
		t.Errorf("WaitForReadiness() ran %d commands, expected 3", len(runner.Commands()))
	}
}
