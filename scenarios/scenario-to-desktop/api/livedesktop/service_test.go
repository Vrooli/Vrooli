package livedesktop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"scenario-to-desktop-api/procmetrics"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testProcess struct {
	cmd *exec.Cmd
}

func (p *testProcess) PID() int {
	if p != nil && p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

func (p *testProcess) IsRunning() bool {
	return p != nil && p.cmd != nil && p.cmd.ProcessState == nil && p.cmd.Process != nil
}

// --- Mock PlatformBackend ---

type mockPlatformBackend struct {
	mu             sync.Mutex
	display        *mockDisplay
	displayErr     error
	remoteInfo     RemoteAccessInfo
	remoteErr      error
	launchErr      error
	screenshotErr  error
	clipboardVal   string
	clipboardErr   error
	resizeErr      error
	monitorFactory procmetrics.MonitorFactory
	killCalled     bool
	lastLaunchOpts LaunchOptions
}

func newMockBackend() *mockPlatformBackend {
	return &mockPlatformBackend{
		display: &mockDisplay{id: ":99", w: 1280, h: 720, running: true},
		remoteInfo: RemoteAccessInfo{
			Protocol: "vnc",
			Port:     5900,
			WSPort:   6080,
		},
	}
}

func (b *mockPlatformBackend) PlatformID() string { return "linux-mock" }

func (b *mockPlatformBackend) CreateDisplay(w, h int) (PlatformDisplay, error) {
	if b.displayErr != nil {
		return nil, b.displayErr
	}
	b.display.w = w
	b.display.h = h
	return b.display, nil
}

func (b *mockPlatformBackend) StartRemoteAccess(display PlatformDisplay) (RemoteAccessInfo, RemoteAccessHandle, error) {
	if b.remoteErr != nil {
		return RemoteAccessInfo{}, nil, b.remoteErr
	}
	return b.remoteInfo, "mock-handle", nil
}

func (b *mockPlatformBackend) StopRemoteAccess(handle RemoteAccessHandle) {}

func (b *mockPlatformBackend) LaunchApp(ctx context.Context, display PlatformDisplay, appPath string, opts LaunchOptions) (PlatformProcess, error) {
	b.mu.Lock()
	b.lastLaunchOpts = opts
	b.mu.Unlock()
	if b.launchErr != nil {
		return nil, b.launchErr
	}
	// Launch a real sleep process so we can test kill
	cmd := exec.Command("sleep", "3600")
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &testProcess{cmd: cmd}, nil
}

func (b *mockPlatformBackend) KillApp(proc PlatformProcess) {
	b.mu.Lock()
	b.killCalled = true
	b.mu.Unlock()
	lp, ok := proc.(*testProcess)
	if ok && lp != nil && lp.cmd != nil && lp.cmd.Process != nil {
		_ = lp.cmd.Process.Kill()
		_ = lp.cmd.Wait()
	}
}

func (b *mockPlatformBackend) CaptureScreenshot(ctx context.Context, display PlatformDisplay, outputPath string) error {
	return b.screenshotErr
}

func (b *mockPlatformBackend) ReadClipboard(ctx context.Context, display PlatformDisplay) (string, error) {
	return b.clipboardVal, b.clipboardErr
}

func (b *mockPlatformBackend) WriteClipboard(ctx context.Context, display PlatformDisplay, content string) error {
	b.mu.Lock()
	b.clipboardVal = content
	b.mu.Unlock()
	return b.clipboardErr
}

func (b *mockPlatformBackend) ResizeDisplay(ctx context.Context, display PlatformDisplay, width, height int) error {
	return b.resizeErr
}

func (b *mockPlatformBackend) NewMonitorFactory() procmetrics.MonitorFactory {
	return b.monitorFactory
}

// --- Mock Display ---

type mockDisplay struct {
	id      string
	w, h    int
	running bool
}

func (d *mockDisplay) DisplayID() string { return d.id }
func (d *mockDisplay) Width() int        { return d.w }
func (d *mockDisplay) Height() int       { return d.h }
func (d *mockDisplay) IsRunning() bool   { return d.running }
func (d *mockDisplay) Stop()             { d.running = false }

// --- Helpers ---

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestService(store Store, backend PlatformBackend) *Service {
	return NewService(store, backend, newTestLogger(), "")
}

func TestFindArtifactByDigestSearchesStaging(t *testing.T) {
	root := t.TempDir()
	stagingRoot := filepath.Join(root, "cache", "staging")
	artifact := filepath.Join(stagingRoot, "demo", "pipeline-1", "platforms", "electron", "dist-electron", "Demo.AppImage")
	require.NoError(t, os.MkdirAll(filepath.Dir(artifact), 0o755))
	payload := []byte("exact validation artifact")
	require.NoError(t, os.WriteFile(artifact, payload, 0o755))
	digest := sha256.Sum256(payload)

	service := NewService(NewInMemoryStore(), newMockBackend(), newTestLogger(), root)
	service.WithArtifactStagingRoot(stagingRoot)
	resolved, err := service.FindArtifactByDigest("demo", "sha256:"+hex.EncodeToString(digest[:]))

	require.NoError(t, err)
	assert.Equal(t, artifact, resolved)

	_, err = service.FindArtifactByDigest("demo", "sha256:"+strings.Repeat("0", 64))
	assert.Error(t, err)
}

// --- Tests ---

func TestStartSession_Success(t *testing.T) {
	store := NewInMemoryStore()
	backend := newMockBackend()
	svc := newTestService(store, backend)

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
	assert.Equal(t, "linux", session.Platform)
}

func TestStartSession_RejectsCancelledContext(t *testing.T) {
	store := NewInMemoryStore()
	svc := newTestService(store, newMockBackend())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	session, err := svc.StartSession(ctx, SessionConfig{ScenarioName: "test"})

	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, session)
	assert.Empty(t, store.List())
}

func TestStartSession_ExplicitLinuxPlatform(t *testing.T) {
	store := NewInMemoryStore()
	backend := newMockBackend()
	svc := newTestService(store, backend)

	session, err := svc.StartSession(context.Background(), SessionConfig{
		ScenarioName: "test",
		Platform:     "linux",
	})
	require.NoError(t, err)
	assert.Equal(t, StateRunning, session.State)
	assert.Equal(t, "linux", session.Platform)
}

func TestStartSession_EmptyPlatformDefaultsToLinux(t *testing.T) {
	store := NewInMemoryStore()
	backend := newMockBackend()
	svc := newTestService(store, backend)

	session, err := svc.StartSession(context.Background(), SessionConfig{
		ScenarioName: "test",
		Platform:     "",
	})
	require.NoError(t, err)
	assert.Equal(t, "linux", session.Platform)
}

func TestStartSession_UnsupportedPlatform(t *testing.T) {
	store := NewInMemoryStore()
	backend := newMockBackend()
	svc := newTestService(store, backend)

	_, err := svc.StartSession(context.Background(), SessionConfig{
		ScenarioName: "test",
		Platform:     "windows",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported platform")
	assert.Contains(t, err.Error(), "windows")
}

func TestStartSession_InvalidPlatform(t *testing.T) {
	store := NewInMemoryStore()
	backend := newMockBackend()
	svc := newTestService(store, backend)

	_, err := svc.StartSession(context.Background(), SessionConfig{
		ScenarioName: "test",
		Platform:     "invalid",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported platform")
}

func TestStartSession_DisplayFails(t *testing.T) {
	store := NewInMemoryStore()
	backend := newMockBackend()
	backend.displayErr = fmt.Errorf("display creation failed")
	svc := newTestService(store, backend)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.Error(t, err)
	require.NotNil(t, session)
	assert.Equal(t, StateError, session.State)
	assert.Contains(t, session.Error, "display creation failed")
}

func TestStartSession_RemoteAccessFails(t *testing.T) {
	store := NewInMemoryStore()
	backend := newMockBackend()
	backend.remoteErr = fmt.Errorf("vnc failed")
	svc := newTestService(store, backend)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.Error(t, err)
	require.NotNil(t, session)
	assert.Equal(t, StateError, session.State)
	assert.Contains(t, session.Error, "remote access start failed")

	// Verify the display was stopped
	assert.False(t, backend.display.IsRunning(), "display should have been stopped after remote access failure")
}

func TestStopSession_Success(t *testing.T) {
	store := NewInMemoryStore()
	backend := newMockBackend()
	svc := newTestService(store, backend)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	err = svc.StopSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, StateStopped, session.State)
}

func TestStopSession_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	svc := newTestService(store, newMockBackend())

	err := svc.StopSession("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestHeartbeat_UpdatesTimestamp(t *testing.T) {
	store := NewInMemoryStore()
	backend := newMockBackend()
	svc := newTestService(store, backend)

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
	backend := newMockBackend()
	svc := newTestService(store, backend)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	// Stop the display
	backend.display.running = false

	err = svc.LaunchApp(session.ID, "/usr/bin/xterm")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestStopSession_KillsAppProcess(t *testing.T) {
	store := NewInMemoryStore()
	backend := newMockBackend()
	svc := newTestService(store, backend)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	// Simulate an app process
	cmd := exec.Command("sleep", "3600")
	require.NoError(t, cmd.Start())
	session.mu.Lock()
	session.AppProcess = &testProcess{cmd: cmd}
	session.AppRunning = true
	session.mu.Unlock()

	// Stop session should kill the app process
	err = svc.StopSession(session.ID)
	require.NoError(t, err)

	backend.mu.Lock()
	assert.True(t, backend.killCalled, "backend.KillApp should have been called")
	backend.mu.Unlock()
}

func TestLaunchApp_KillsPreviousApp(t *testing.T) {
	store := NewInMemoryStore()
	backend := newMockBackend()
	svc := newTestService(store, backend)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	// Simulate a running app
	oldCmd := exec.Command("sleep", "3600")
	require.NoError(t, oldCmd.Start())
	session.mu.Lock()
	session.AppProcess = &testProcess{cmd: oldCmd}
	session.AppRunning = true
	session.mu.Unlock()

	// Launch a new app — the old app should be killed first
	err = svc.LaunchApp(session.ID, "/bin/sleep")
	require.NoError(t, err)

	// The old app process should have been killed
	err = oldCmd.Wait()
	assert.Error(t, err, "old app process should have been killed before new launch")

	// Clean up
	svc.killAppProcess(session)
}

func TestListSessions(t *testing.T) {
	store := NewInMemoryStore()
	backend := newMockBackend()
	svc := newTestService(store, backend)

	_, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "s1"})
	require.NoError(t, err)

	_, err = svc.StartSession(context.Background(), SessionConfig{ScenarioName: "s2"})
	require.NoError(t, err)

	sessions := svc.ListSessions()
	assert.Len(t, sessions, 2)
}

// --- Process Monitor Tests ---

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
	return m.startErr
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
	backend := newMockBackend()
	svc := newTestService(store, backend)

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
	backend := newMockBackend()
	svc := newTestService(store, backend)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	// Set up a running app with a monitor
	cmd := exec.Command("sleep", "3600")
	require.NoError(t, cmd.Start())
	session.mu.Lock()
	session.AppProcess = &testProcess{cmd: cmd}
	session.AppRunning = true
	session.mu.Unlock()

	monitor := newMockMonitor(nil)
	session.SetMonitor(monitor)

	svc.killAppProcess(session)

	monitor.mu.Lock()
	assert.True(t, monitor.stopped, "monitor should be stopped when app is killed")
	monitor.mu.Unlock()
	assert.Nil(t, session.GetMonitor(), "monitor should be nil after kill")
}

func TestSessionView_MonitorStartedNoSamplesYet(t *testing.T) {
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
	backend := newMockBackend()
	svc := newTestService(store, backend)

	session, err := svc.StartSession(context.Background(), SessionConfig{ScenarioName: "test"})
	require.NoError(t, err)

	err = svc.LaunchApp(session.ID, "/bin/sleep")
	require.NoError(t, err)
	assert.Nil(t, session.GetMonitor())

	svc.killAppProcess(session)
}
