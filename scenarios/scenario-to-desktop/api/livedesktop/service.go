package livedesktop

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"

	"scenario-to-desktop-api/screenrecording"
	"scenario-to-desktop-api/shared/packaging"
)

// VNCStartFunc is the signature for starting a VNC session.
type VNCStartFunc func(display string) (vncPort, wsPort int, x11vncCmd, websockifyCmd *exec.Cmd, err error)

// VNCStopFunc is the signature for stopping VNC processes.
type VNCStopFunc func(session *Session)

// Service orchestrates live desktop session lifecycle.
type Service struct {
	store      Store
	displayMgr screenrecording.DisplayManager
	logger     *slog.Logger
	vrooliRoot string
	startVNC   VNCStartFunc
	stopVNC    VNCStopFunc
}

// NewService creates a new live desktop service.
func NewService(store Store, displayMgr screenrecording.DisplayManager, logger *slog.Logger, vrooliRoot string) *Service {
	return &Service{
		store:      store,
		displayMgr: displayMgr,
		logger:     logger,
		vrooliRoot: vrooliRoot,
		startVNC:   startVNCSession,
		stopVNC:    stopVNCProcesses,
	}
}

// StartSession creates a new live desktop session with VNC access.
func (s *Service) StartSession(ctx context.Context, cfg SessionConfig) (*Session, error) {
	if cfg.Width == 0 {
		cfg.Width = 1280
	}
	if cfg.Height == 0 {
		cfg.Height = 720
	}

	session := &Session{
		ID:            uuid.New().String(),
		ScenarioName:  cfg.ScenarioName,
		State:         StateCreating,
		Width:         cfg.Width,
		Height:        cfg.Height,
		CreatedAt:     time.Now(),
		LastHeartbeat: time.Now(),
	}

	if err := s.store.Create(session); err != nil {
		return nil, fmt.Errorf("storing session: %w", err)
	}

	// Create managed display
	display, err := s.displayMgr.CreateManagedDisplay(cfg.Width, cfg.Height)
	if err != nil {
		session.SetError(fmt.Sprintf("display creation failed: %v", err))
		_ = s.store.Update(session)
		return session, fmt.Errorf("creating display: %w", err)
	}
	session.Display = display

	// Start VNC toolchain
	vncPort, wsPort, x11vncCmd, websockifyCmd, err := s.startVNC(display.DisplayID)
	if err != nil {
		display.Stop()
		session.SetError(fmt.Sprintf("VNC start failed: %v", err))
		_ = s.store.Update(session)
		return session, fmt.Errorf("starting VNC: %w", err)
	}
	session.VNCPort = vncPort
	session.WSPort = wsPort
	session.X11VNCCmd = x11vncCmd
	session.WebsockifyCmd = websockifyCmd

	session.SetState(StateRunning)
	_ = s.store.Update(session)

	s.logger.Info("live desktop session started",
		"session_id", session.ID,
		"display", display.DisplayID,
		"vnc_port", vncPort,
		"ws_port", wsPort,
	)

	return session, nil
}

// StopSession tears down a live desktop session.
func (s *Service) StopSession(sessionID string) error {
	session, err := s.store.Get(sessionID)
	if err != nil {
		return err
	}

	session.SetState(StateStopping)
	_ = s.store.Update(session)

	// Kill launched app process
	s.killAppProcess(session)

	// Stop VNC processes
	s.stopVNC(session)

	// Stop display
	if session.Display != nil {
		session.Display.Stop()
	}

	session.SetState(StateStopped)
	_ = s.store.Update(session)

	s.logger.Info("live desktop session stopped", "session_id", sessionID)
	return nil
}

// Heartbeat updates the last heartbeat for a session.
func (s *Service) Heartbeat(sessionID string) error {
	session, err := s.store.Get(sessionID)
	if err != nil {
		return err
	}
	session.Touch()
	return nil
}

// LaunchApp starts an application on the session's display.
// If appPath is empty, it auto-discovers the latest build artifact for the session's scenario.
// If an app is already running, it is killed before launching the new one.
func (s *Service) LaunchApp(sessionID, appPath string) error {
	session, err := s.store.Get(sessionID)
	if err != nil {
		return err
	}
	if session.Display == nil || !session.Display.IsRunning() {
		return fmt.Errorf("session display is not running")
	}

	// Auto-discover artifact if no explicit path provided
	if appPath == "" {
		discovered, err := s.findArtifact(session.ScenarioName)
		if err != nil {
			return fmt.Errorf("auto-discover artifact: %w", err)
		}
		appPath = discovered
	}

	// Kill any previously launched app process to prevent pileup
	s.killAppProcess(session)

	cmd := exec.CommandContext(context.Background(), appPath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("DISPLAY=%s", session.Display.DisplayID))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launching app: %w", err)
	}

	session.mu.Lock()
	session.AppCmd = cmd
	session.mu.Unlock()

	// Reap the process in the background so it doesn't become a zombie
	go func() {
		_ = cmd.Wait()
	}()

	s.logger.Info("app launched on desktop", "session_id", sessionID, "app", appPath)
	return nil
}

// killAppProcess kills and cleans up the app process for a session.
func (s *Service) killAppProcess(session *Session) {
	session.mu.Lock()
	cmd := session.AppCmd
	session.AppCmd = nil
	session.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

// FindArtifact finds the latest build artifact for a scenario.
func (s *Service) FindArtifact(scenarioName string) (string, error) {
	return s.findArtifact(scenarioName)
}

func (s *Service) findArtifact(scenarioName string) (string, error) {
	if s.vrooliRoot == "" {
		return "", fmt.Errorf("vrooliRoot not configured")
	}

	platform := currentPlatform()
	distPath := filepath.Join(s.vrooliRoot, "scenarios", scenarioName, "platforms", "electron", "dist-electron")
	artifact, err := packaging.FindBuiltPackage(distPath, platform)
	if err != nil {
		return "", fmt.Errorf("no %s artifact found for scenario %q: %w", platform, scenarioName, err)
	}
	return artifact, nil
}

func currentPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return "mac"
	case "windows":
		return "win"
	default:
		return "linux"
	}
}

// GetSession returns a session by ID.
func (s *Service) GetSession(id string) (*Session, error) {
	return s.store.Get(id)
}

// ListSessions returns all sessions.
func (s *Service) ListSessions() []*Session {
	return s.store.List()
}
