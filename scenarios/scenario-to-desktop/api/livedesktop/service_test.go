package livedesktop

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
