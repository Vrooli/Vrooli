package livedesktop

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scenario-to-desktop-api/procmetrics"
	"scenario-to-desktop-api/screenrecording"
)

// mockDisplayManager implements screenrecording.DisplayManager for testing.
type mockDisplayManager struct {
	display *screenrecording.ManagedDisplay
	err     error
}

func (m *mockDisplayManager) CreateDisplay(w, h int) (string, func(), error) {
	if m.err != nil {
		return "", nil, m.err
	}
	return m.display.DisplayID, func() {}, nil
}

func (m *mockDisplayManager) CreateManagedDisplay(w, h int) (*screenrecording.ManagedDisplay, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.display, nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// mockVNCStart returns a VNCStartFunc that succeeds with the given ports.
func mockVNCStart(vncPort, wsPort int) VNCStartFunc {
	return func(display string) (int, int, *exec.Cmd, *exec.Cmd, error) {
		return vncPort, wsPort, nil, nil, nil
	}
}

// mockVNCStartError returns a VNCStartFunc that always fails.
func mockVNCStartError(msg string) VNCStartFunc {
	return func(display string) (int, int, *exec.Cmd, *exec.Cmd, error) {
		return 0, 0, nil, nil, fmt.Errorf("%s", msg)
	}
}

// mockVNCStop is a no-op VNCStopFunc.
func mockVNCStop(session *Session) {}

func newTestService(store Store, dm screenrecording.DisplayManager, startFn VNCStartFunc) *Service {
	svc := NewService(store, dm, newTestLogger(), "")
	svc.startVNC = startFn
	svc.stopVNC = mockVNCStop
	return svc
}

func TestStartSession_Success(t *testing.T) {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{
		display: &screenrecording.ManagedDisplay{DisplayID: ":99"},
	}
	svc := newTestService(store, dm, mockVNCStart(5900, 6080))

	session, err := svc.StartSession(context.Background(), SessionConfig{
		Width:        1280,
		Height:       720,
		ScenarioName: "test",
	})
	require.NoError(t, err)
	assert.Equal(t, StateRunning, session.State)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "test", session.ScenarioName)
	assert.Equal(t, 1280, session.Width)
	assert.Equal(t, 720, session.Height)
	assert.Equal(t, 5900, session.VNCPort)
	assert.Equal(t, 6080, session.WSPort)
}

func TestStartSession_DisplayFails(t *testing.T) {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{err: fmt.Errorf("display creation failed")}
	svc := newTestService(store, dm, mockVNCStart(5900, 6080))

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.Error(t, err)
	require.NotNil(t, session)
	assert.Equal(t, StateError, session.State)
	assert.Contains(t, session.Error, "display creation failed")
}

func TestStartSession_VNCFails(t *testing.T) {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{
		display: &screenrecording.ManagedDisplay{DisplayID: ":99"},
	}
	svc := newTestService(store, dm, mockVNCStartError("vnc failed"))

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.Error(t, err)
	require.NotNil(t, session)
	assert.Equal(t, StateError, session.State)
	assert.Contains(t, session.Error, "VNC start failed")

	// Verify the display was stopped (ManagedDisplay.Stop sets stopped=true)
	assert.False(t, dm.display.IsRunning(), "display should have been stopped after VNC failure")
}

func TestStopSession_Success(t *testing.T) {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{
		display: &screenrecording.ManagedDisplay{DisplayID: ":99"},
	}
	svc := newTestService(store, dm, mockVNCStart(5900, 6080))

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	err = svc.StopSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, StateStopped, session.State)
}

func TestStopSession_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	svc := newTestService(store, &mockDisplayManager{}, mockVNCStart(5900, 6080))

	err := svc.StopSession("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestHeartbeat_UpdatesTimestamp(t *testing.T) {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{
		display: &screenrecording.ManagedDisplay{DisplayID: ":99"},
	}
	svc := newTestService(store, dm, mockVNCStart(5900, 6080))

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	oldHeartbeat := session.LastHeartbeat
	time.Sleep(10 * time.Millisecond)

	err = svc.Heartbeat(session.ID)
	require.NoError(t, err)
	assert.True(t, session.LastHeartbeat.After(oldHeartbeat))
}

func TestLaunchApp_DisplayNotRunning(t *testing.T) {
	store := NewInMemoryStore()
	// Create a display that is already stopped
	display := &screenrecording.ManagedDisplay{DisplayID: ":99"}
	display.Stop()
	dm := &mockDisplayManager{display: display}
	svc := newTestService(store, dm, mockVNCStart(5900, 6080))

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	err = svc.LaunchApp(session.ID, "/usr/bin/xterm")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestStopSession_KillsAppProcess(t *testing.T) {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{
		display: &screenrecording.ManagedDisplay{DisplayID: ":99"},
	}
	svc := newTestService(store, dm, mockVNCStart(5900, 6080))

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	// Simulate an app process by setting AppCmd to a long-running command
	cmd := exec.Command("sleep", "3600")
	require.NoError(t, cmd.Start())
	session.AppCmd = cmd

	// Stop session should kill the app process
	err = svc.StopSession(session.ID)
	require.NoError(t, err)

	// The app process should have been killed (Wait returns immediately)
	err = cmd.Wait()
	assert.Error(t, err, "process should have been killed")
}

func TestLaunchApp_KillsPreviousApp(t *testing.T) {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{
		display: &screenrecording.ManagedDisplay{DisplayID: ":99"},
	}
	svc := newTestService(store, dm, mockVNCStart(5900, 6080))

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	// Simulate a running app
	oldCmd := exec.Command("sleep", "3600")
	require.NoError(t, oldCmd.Start())
	session.mu.Lock()
	session.AppCmd = oldCmd
	session.mu.Unlock()

	// Launch a new app (will fail since /nonexistent doesn't exist, but the old app should be killed first)
	_ = svc.LaunchApp(session.ID, "/nonexistent-binary")

	// The old app process should have been killed
	err = oldCmd.Wait()
	assert.Error(t, err, "old app process should have been killed before new launch")
}

func TestListSessions(t *testing.T) {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{
		display: &screenrecording.ManagedDisplay{DisplayID: ":99"},
	}
	svc := newTestService(store, dm, mockVNCStart(5900, 6080))

	// Start two sessions (each needs a fresh display mock since the first gets consumed)
	_, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "s1"})
	require.NoError(t, err)

	dm.display = &screenrecording.ManagedDisplay{DisplayID: ":100"}
	_, err = svc.StartSession(context.Background(), SessionConfig{ScenarioName: "s2"})
	require.NoError(t, err)

	sessions := svc.ListSessions()
	assert.Len(t, sessions, 2)
}

// --- Process Monitor Tests ---

// mockMonitor records Start/Stop calls and returns a canned report.
type mockMonitor struct {
	mu        sync.Mutex
	started   bool
	stopped   bool
	startPID  int
	startDisp string
	startErr  error
	report    *procmetrics.Report
	doneCh    chan struct{}
}

func newMockMonitor(report *procmetrics.Report) *mockMonitor {
	return &mockMonitor{
		report: report,
		doneCh: make(chan struct{}),
	}
}

func (m *mockMonitor) Start(_ context.Context, pid int, display string, _, _ int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
	m.startPID = pid
	m.startDisp = display
	if m.startErr != nil {
		return m.startErr
	}
	return nil
}

func (m *mockMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.stopped {
		m.stopped = true
		close(m.doneCh)
	}
}

func (m *mockMonitor) Report() *procmetrics.Report {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.report
}

func (m *mockMonitor) Done() <-chan struct{} {
	return m.doneCh
}

// mockMonitorFactory creates mockMonitors.
type mockMonitorFactory struct {
	monitor *mockMonitor
}

func (f *mockMonitorFactory) NewMonitor() procmetrics.Monitor {
	return f.monitor
}

func TestSessionView_IncludesMetrics(t *testing.T) {
	splashDur := int64(800)
	readyDur := int64(1500)
	now := time.Now()
	splashAt := now.Add(-1500 * time.Millisecond)
	report := &procmetrics.Report{
		Startup: procmetrics.StartupTiming{
			LaunchAt:         now.Add(-2000 * time.Millisecond),
			SplashVisibleAt:  &splashAt,
			SplashDurationMs: &splashDur,
			ReadyAt:          &now,
			ReadyMs:          &readyDur,
		},
		Samples: []procmetrics.Sample{
			{Timestamp: now, CPUPercent: 25.5, RSSBytes: 150 * 1024 * 1024, PeakBytes: 200 * 1024 * 1024, Threads: 8},
		},
	}
	monitor := newMockMonitor(report)

	session := &Session{
		ID:           "test-1",
		ScenarioName: "myapp",
		State:        StateRunning,
		AppRunning:   true,
		Monitor:      monitor,
	}

	view := session.View()
	require.NotNil(t, view.Metrics)
	assert.True(t, view.Metrics.SplashDetected)
	require.NotNil(t, view.Metrics.SplashDurationMs)
	assert.Equal(t, int64(800), *view.Metrics.SplashDurationMs)
	assert.True(t, view.Metrics.ReadyDetected)
	require.NotNil(t, view.Metrics.ReadyDurationMs)
	assert.Equal(t, int64(1500), *view.Metrics.ReadyDurationMs)
	require.NotNil(t, view.Metrics.CurrentCPU)
	assert.InDelta(t, 25.5, *view.Metrics.CurrentCPU, 0.01)
	require.NotNil(t, view.Metrics.CurrentRSSMB)
	assert.InDelta(t, 150.0, *view.Metrics.CurrentRSSMB, 0.01)
	assert.Equal(t, 1, view.Metrics.SampleCount)
}

func TestSessionView_NilMonitor(t *testing.T) {
	session := &Session{
		ID:    "test-1",
		State: StateRunning,
	}
	view := session.View()
	assert.Nil(t, view.Metrics)
}

func TestSetGetMonitor(t *testing.T) {
	session := &Session{}
	assert.Nil(t, session.GetMonitor())

	m := newMockMonitor(nil)
	session.SetMonitor(m)
	assert.Equal(t, m, session.GetMonitor())

	session.SetMonitor(nil)
	assert.Nil(t, session.GetMonitor())
}

func TestStopSession_StopsMonitor(t *testing.T) {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{
		display: &screenrecording.ManagedDisplay{DisplayID: ":99"},
	}
	svc := newTestService(store, dm, mockVNCStart(5900, 6080))

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	monitor := newMockMonitor(nil)
	session.SetMonitor(monitor)

	err = svc.StopSession(session.ID)
	require.NoError(t, err)

	monitor.mu.Lock()
	assert.True(t, monitor.stopped, "monitor should be stopped when session is stopped")
	monitor.mu.Unlock()
}

func TestKillApp_StopsMonitor(t *testing.T) {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{
		display: &screenrecording.ManagedDisplay{DisplayID: ":99"},
	}
	svc := newTestService(store, dm, mockVNCStart(5900, 6080))

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	// Set up a running app with a monitor
	cmd := exec.Command("sleep", "3600")
	require.NoError(t, cmd.Start())
	session.mu.Lock()
	session.AppCmd = cmd
	session.AppRunning = true
	session.mu.Unlock()

	monitor := newMockMonitor(nil)
	session.SetMonitor(monitor)

	// Kill the app (simulates relaunch scenario)
	svc.killAppProcess(session)

	monitor.mu.Lock()
	assert.True(t, monitor.stopped, "monitor should be stopped when app is killed")
	monitor.mu.Unlock()
	assert.Nil(t, session.GetMonitor(), "monitor should be nil after kill")
}

func TestSessionView_MonitorStartedNoSamplesYet(t *testing.T) {
	// Simulates the state right after app launch: monitor is running but
	// no resource samples collected and window not yet detected.
	monitor := newMockMonitor(&procmetrics.Report{
		Startup: procmetrics.StartupTiming{
			LaunchAt: time.Now(),
		},
	})

	session := &Session{
		ID:         "test-1",
		State:      StateRunning,
		AppRunning: true,
		Monitor:    monitor,
	}

	view := session.View()
	require.NotNil(t, view.Metrics, "metrics should be non-nil even with no samples")
	assert.False(t, view.Metrics.SplashDetected)
	assert.Nil(t, view.Metrics.SplashDurationMs)
	assert.False(t, view.Metrics.ReadyDetected)
	assert.Nil(t, view.Metrics.ReadyDurationMs)
	assert.Nil(t, view.Metrics.CurrentCPU, "no CPU data before first sample")
	assert.Nil(t, view.Metrics.CurrentRSSMB, "no memory data before first sample")
	assert.Equal(t, 0, view.Metrics.SampleCount)
}

func TestBuildMetricsView_NilReport(t *testing.T) {
	assert.Nil(t, buildMetricsView(nil))
}

func TestBuildMetricsView_EmptyReport(t *testing.T) {
	mv := buildMetricsView(&procmetrics.Report{})
	require.NotNil(t, mv)
	assert.False(t, mv.SplashDetected)
	assert.Nil(t, mv.SplashDurationMs)
	assert.False(t, mv.ReadyDetected)
	assert.Nil(t, mv.ReadyDurationMs)
	assert.Equal(t, 0, mv.SampleCount)
}

func TestLaunchApp_NoMonitorFactory_NoError(t *testing.T) {
	store := NewInMemoryStore()
	dm := &mockDisplayManager{
		display: &screenrecording.ManagedDisplay{DisplayID: ":99"},
	}
	svc := newTestService(store, dm, mockVNCStart(5900, 6080))
	// No monitor factory set — should work fine

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	// Launch a real binary (sleep) to test the path without monitor
	err = svc.LaunchApp(session.ID, "/bin/sleep")
	require.NoError(t, err)
	assert.Nil(t, session.GetMonitor())

	// Clean up
	svc.killAppProcess(session)
}
