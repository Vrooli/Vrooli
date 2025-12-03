package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/testutil"
)

// testMonitor creates a Monitor configured for testing.
func testMonitor(m *manifest.Manifest, ports map[string]map[string]int, clock *testutil.MockClock, dialer infra.NetworkDialer, fs *testutil.MockFileSystem, cmdRunner *testutil.MockCommandRunner) *Monitor {
	var clockIface infra.Clock = clock
	if clock == nil {
		clockIface = testutil.NewMockClock(time.Now())
	}
	if dialer == nil {
		dialer = infra.RealNetworkDialer{}
	}
	var fsIface infra.FileSystem = fs
	if fs == nil {
		fsIface = testutil.NewMockFileSystem()
	}
	var cmdIface infra.CommandRunner = cmdRunner
	if cmdRunner == nil {
		cmdIface = testutil.NewMockCommandRunner()
	}

	return NewMonitor(MonitorConfig{
		Manifest:     m,
		Ports:        &testutil.MockPortAllocator{Ports: ports},
		Dialer:       dialer,
		CmdRunner:    cmdIface,
		FS:           fsIface,
		Clock:        clockIface,
		AppData:      "/tmp/test-appdata",
		StatusGetter: nil,
	})
}

func TestWaitForReadiness_HealthSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

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

	clock := testutil.NewMockClock(time.Now())
	hm := testMonitor(m, map[string]map[string]int{"api": {"http": port}}, clock, nil, nil, nil)

	ctx := context.Background()
	err := hm.WaitForReadiness(ctx, "api")
	if err != nil {
		t.Errorf("WaitForReadiness() error = %v", err)
	}
}

func TestWaitForReadiness_PortOpen(t *testing.T) {
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

	clock := testutil.NewMockClock(time.Now())
	dialer := infra.RealNetworkDialer{}
	hm := testMonitor(m, map[string]map[string]int{"api": {"http": port}}, clock, dialer, nil, nil)

	ctx := context.Background()
	err = hm.WaitForReadiness(ctx, "api")
	if err != nil {
		t.Errorf("WaitForReadiness() error = %v", err)
	}
}

func TestWaitForReadiness_LogMatch(t *testing.T) {
	clock := testutil.NewMockClock(time.Now())
	mockFS := testutil.NewMockFileSystem()
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

	hm := testMonitor(m, map[string]map[string]int{}, clock, nil, mockFS, nil)

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

	hm := testMonitor(m, map[string]map[string]int{}, nil, nil, nil, nil)

	ctx := context.Background()
	err := hm.WaitForReadiness(ctx, "api")
	if err == nil {
		t.Error("WaitForReadiness() expected error for unknown type")
	}
}

func TestCheckOnce_TCP(t *testing.T) {
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

	hm := testMonitor(m, map[string]map[string]int{"api": {"http": port}}, nil, infra.RealNetworkDialer{}, nil, nil)

	ctx := context.Background()
	if !hm.CheckOnce(ctx, "api") {
		t.Error("CheckOnce() returned false, expected true")
	}

	listener.Close()
	time.Sleep(10 * time.Millisecond)

	if hm.CheckOnce(ctx, "api") {
		t.Error("CheckOnce() returned true after listener closed, expected false")
	}
}

func TestCheckOnce_Command(t *testing.T) {
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

	hm := testMonitor(m, map[string]map[string]int{}, nil, nil, nil, runner)

	ctx := context.Background()

	runner.SetShouldErr(false)
	if !hm.CheckOnce(ctx, "api") {
		t.Error("CheckOnce() returned false, expected true")
	}

	runner.SetShouldErr(true)
	if hm.CheckOnce(ctx, "api") {
		t.Error("CheckOnce() returned true, expected false")
	}
}

func TestCheckOnce_CommandEmpty(t *testing.T) {
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

	hm := testMonitor(m, map[string]map[string]int{}, nil, nil, nil, nil)

	ctx := context.Background()
	if hm.CheckOnce(ctx, "api") {
		t.Error("CheckOnce() returned true for empty command, expected false")
	}
}

func TestCheckOnce_LogMatch(t *testing.T) {
	mockFS := testutil.NewMockFileSystem()
	mockFS.Files["/tmp/test-appdata/logs/api.log"] = []byte("INFO: Server started on port 8080\nDEBUG: Ready for connections")

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

			hm := testMonitor(m, map[string]map[string]int{}, nil, nil, mockFS, nil)
			ctx := context.Background()
			got := hm.CheckOnce(ctx, "api")
			if got != tt.want {
				t.Errorf("CheckOnce() for log_match = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWaitForDependencies(t *testing.T) {
	clock := testutil.NewMockClock(time.Now())

	serviceStatus := map[string]Status{
		"db":    {Ready: true},
		"cache": {Ready: true},
	}
	statusGetter := func(id string) (Status, bool) {
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

	hm := NewMonitor(MonitorConfig{
		Manifest:     m,
		Ports:        &testutil.MockPortAllocator{Ports: map[string]map[string]int{}},
		Dialer:       infra.RealNetworkDialer{},
		CmdRunner:    testutil.NewMockCommandRunner(),
		FS:           testutil.NewMockFileSystem(),
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
	clock := testutil.NewMockClock(time.Now())

	serviceStatus := map[string]Status{}
	statusGetter := func(id string) (Status, bool) {
		s, ok := serviceStatus[id]
		return s, ok
	}

	m := &manifest.Manifest{
		Services: []manifest.Service{
			{ID: "api", Dependencies: []string{"nonexistent"}},
		},
	}

	hm := NewMonitor(MonitorConfig{
		Manifest:     m,
		Ports:        &testutil.MockPortAllocator{Ports: map[string]map[string]int{}},
		Dialer:       infra.RealNetworkDialer{},
		CmdRunner:    testutil.NewMockCommandRunner(),
		FS:           testutil.NewMockFileSystem(),
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

	hm := testMonitor(m, map[string]map[string]int{}, nil, nil, nil, nil)

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

func TestWaitForDependencies_MissingStatusGetter(t *testing.T) {
	m := &manifest.Manifest{
		Services: []manifest.Service{
			{ID: "api", Dependencies: []string{"db"}},
		},
	}

	hm := NewMonitor(MonitorConfig{
		Manifest: m,
		Ports:    &testutil.MockPortAllocator{},
		Dialer:   infra.RealNetworkDialer{},
		CmdRunner: func() infra.CommandRunner {
			return testutil.NewMockCommandRunner()
		}(),
		FS:    testutil.NewMockFileSystem(),
		Clock: &advancingClock{now: time.Now(), advance: 10 * time.Second},
		// StatusGetter intentionally omitted
	})

	err := hm.WaitForDependencies(context.Background(), &m.Services[0])
	if err == nil || !strings.Contains(err.Error(), "status getter not configured") {
		t.Fatalf("WaitForDependencies() error = %v, want status getter error", err)
	}
}

func TestWaitForDependencies_TimesOutWhenDependencyStuck(t *testing.T) {
	m := &manifest.Manifest{
		Services: []manifest.Service{
			{ID: "api", Dependencies: []string{"db"}},
		},
	}

	statuses := map[string]Status{
		"db": {Ready: false, Message: "initializing"},
	}

	clock := &advancingClock{now: time.Now(), advance: 5 * time.Minute}
	hm := NewMonitor(MonitorConfig{
		Manifest: m,
		Ports:    &testutil.MockPortAllocator{},
		Dialer:   infra.RealNetworkDialer{},
		CmdRunner: func() infra.CommandRunner {
			return testutil.NewMockCommandRunner()
		}(),
		FS:    testutil.NewMockFileSystem(),
		Clock: clock,
		StatusGetter: func(id string) (Status, bool) {
			st, ok := statuses[id]
			return st, ok
		},
	})

	err := hm.WaitForDependencies(context.Background(), &m.Services[0])
	if err == nil {
		t.Fatal("WaitForDependencies() expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "dependencies not ready") {
		t.Fatalf("WaitForDependencies() error = %v, want dependencies not ready message", err)
	}
}

type advancingClock struct {
	now     time.Time
	advance time.Duration
}

func (c *advancingClock) Now() time.Time {
	return c.now
}

func (c *advancingClock) Sleep(d time.Duration) {
	c.now = c.now.Add(d)
}

func (c *advancingClock) After(d time.Duration) <-chan time.Time {
	c.now = c.now.Add(c.advance)
	ch := make(chan time.Time, 1)
	ch <- c.now
	return ch
}

func (c *advancingClock) NewTicker(d time.Duration) infra.Ticker {
	return &noopTicker{ch: make(chan time.Time)}
}

type noopTicker struct {
	ch chan time.Time
}

func (n *noopTicker) C() <-chan time.Time {
	return n.ch
}

func (n *noopTicker) Stop() {}

func TestWaitForReadiness_Retries(t *testing.T) {
	clock := testutil.NewMockClock(time.Now())
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
					TimeoutMs: 3000,
				},
			},
		},
	}

	hm := testMonitor(m, map[string]map[string]int{}, clock, nil, nil, runner)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := hm.WaitForReadiness(ctx, "api")
	if err == nil {
		t.Error("WaitForReadiness() expected error after retries exhausted")
	}

	if len(runner.Commands()) != 3 {
		t.Errorf("WaitForReadiness() ran %d commands, expected 3", len(runner.Commands()))
	}
}
