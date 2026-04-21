package livedesktop

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"vrooli-emulator-api/captures"
	"vrooli-emulator-api/screenrecording"

	"github.com/google/uuid"
)

// Service orchestrates session lifecycle on a virtual display.
type Service struct {
	store    Store
	backend  PlatformBackend
	logger   *slog.Logger
	recorder screenrecording.Recorder
	captures *captures.Service
	dataDir  string
}

// NewService creates a new session service.
func NewService(store Store, backend PlatformBackend, logger *slog.Logger) *Service {
	return &Service{
		store:   store,
		backend: backend,
		logger:  logger,
	}
}

// WithRecorder sets the screen recorder on the service.
func (s *Service) WithRecorder(r screenrecording.Recorder) {
	s.recorder = r
}

// WithCaptures sets the captures service for persistent capture storage.
func (s *Service) WithCaptures(svc *captures.Service) {
	s.captures = svc
}

// WithDataDir overrides the data directory for screenshots and recordings.
func (s *Service) WithDataDir(dir string) {
	s.dataDir = dir
}

// StartSession creates a new session. When cfg.Headless is true, only a
// virtual display is allocated and remote-access plumbing is skipped.
func (s *Service) StartSession(ctx context.Context, cfg SessionConfig) (*Session, error) {
	if cfg.Width == 0 {
		cfg.Width = 1280
	}
	if cfg.Height == 0 {
		cfg.Height = 720
	}

	requestedPlatform := cfg.Platform
	if requestedPlatform == "" {
		requestedPlatform = "linux"
	}
	backendBase := strings.SplitN(s.backend.PlatformID(), "-", 2)[0]
	if requestedPlatform != backendBase {
		return nil, fmt.Errorf("unsupported platform: %q (only %q is currently available)", requestedPlatform, backendBase)
	}

	session := &Session{
		ID:            uuid.New().String(),
		ScenarioName:  cfg.ScenarioName,
		State:         StateCreating,
		Width:         cfg.Width,
		Height:        cfg.Height,
		Platform:      requestedPlatform,
		Headless:      cfg.Headless,
		CreatedAt:     time.Now(),
		LastHeartbeat: time.Now(),
	}

	if err := s.store.Create(session); err != nil {
		return nil, fmt.Errorf("storing session: %w", err)
	}

	display, err := s.backend.CreateDisplay(cfg.Width, cfg.Height)
	if err != nil {
		session.SetError(fmt.Sprintf("display creation failed: %v", err))
		_ = s.store.Update(session)
		return session, fmt.Errorf("creating display: %w", err)
	}
	session.Display = display

	if cfg.Headless {
		session.SetState(StateRunning)
		_ = s.store.Update(session)
		s.logger.Info("headless session started",
			"session_id", session.ID,
			"display", display.DisplayID(),
			"platform", s.backend.PlatformID(),
		)
		return session, nil
	}

	info, handle, err := s.backend.StartRemoteAccess(display)
	if err != nil {
		display.Stop()
		session.SetError(fmt.Sprintf("remote access start failed: %v", err))
		_ = s.store.Update(session)
		return session, fmt.Errorf("starting remote access: %w", err)
	}
	session.RemoteAccess = handle
	session.RemoteInfo = info
	session.VNCPort = info.Port
	session.WSPort = info.WSPort

	session.SetState(StateRunning)
	_ = s.store.Update(session)

	s.logger.Info("session started",
		"session_id", session.ID,
		"display", display.DisplayID(),
		"platform", s.backend.PlatformID(),
		"vnc_port", info.Port,
		"ws_port", info.WSPort,
	)

	return session, nil
}

// StopSession tears down a session.
func (s *Service) StopSession(sessionID string) error {
	session, err := s.store.Get(sessionID)
	if err != nil {
		return err
	}

	session.SetState(StateStopping)
	_ = s.store.Update(session)

	s.killAppProcess(session)

	if session.RemoteAccess != nil {
		s.backend.StopRemoteAccess(session.RemoteAccess)
	}

	if session.Display != nil {
		session.Display.Stop()
	}

	session.SetState(StateStopped)
	_ = s.store.Update(session)

	s.logger.Info("session stopped", "session_id", sessionID)
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

// LaunchApp starts an application on the session's display. The caller must
// resolve the executable path; the service does no auto-discovery.
// If an app is already running, it is killed before launching the new one.
func (s *Service) LaunchApp(sessionID, appPath string) error {
	session, err := s.store.Get(sessionID)
	if err != nil {
		return err
	}
	if appPath == "" {
		return fmt.Errorf("app_path is required")
	}
	if session.Display == nil || !session.Display.IsRunning() {
		return fmt.Errorf("session display is not running")
	}

	s.killAppProcess(session)

	session.mu.Lock()
	opts := LaunchOptions{
		NetworkMode:   session.NetworkMode,
		BandwidthKbps: session.BandwidthKbps,
		DarkMode:      session.DarkMode,
		Locale:        session.Locale,
		EnvVars:       make(map[string]string),
	}
	for k, v := range session.EnvVars {
		opts.EnvVars[k] = v
	}
	session.mu.Unlock()

	proc, err := s.backend.LaunchApp(context.Background(), session.Display, appPath, opts)
	if err != nil {
		return err
	}

	session.mu.Lock()
	session.AppProcess = proc
	session.AppRunning = true
	session.mu.Unlock()

	monitorFactory := s.backend.NewMonitorFactory()
	if monitorFactory != nil {
		monitor := monitorFactory.NewMonitor()
		if err := monitor.Start(context.Background(), proc.PID(), session.Display.DisplayID(), session.Width, session.Height); err != nil {
			s.logger.Warn("failed to start process monitor", "error", err)
		} else {
			session.SetMonitor(monitor)
		}
	}

	go func() {
		for proc.IsRunning() {
			time.Sleep(500 * time.Millisecond)
		}
		if m := session.GetMonitor(); m != nil {
			m.Stop()
		}
		session.SetAppRunning(false)
	}()

	s.logger.Info("app launched on desktop", "session_id", sessionID, "app", appPath,
		"network_mode", opts.NetworkMode, "dark_mode", opts.DarkMode, "locale", opts.Locale)
	return nil
}

// ExecuteAction dispatches a control action against a session.
func (s *Service) ExecuteAction(ctx context.Context, sessionID, action string, params json.RawMessage) (*ActionResult, error) {
	session, err := s.store.Get(sessionID)
	if err != nil {
		return nil, err
	}

	executor, err := lookupAction(action)
	if err != nil {
		return nil, err
	}

	result, err := executor.Execute(ctx, session, s, params)
	if err != nil {
		return nil, err
	}

	_ = s.store.Update(session)
	return result, nil
}

// killAppProcess kills and cleans up the app process for a session.
func (s *Service) killAppProcess(session *Session) {
	if m := session.GetMonitor(); m != nil {
		m.Stop()
		session.SetMonitor(nil)
	}

	session.mu.Lock()
	proc := session.AppProcess
	session.AppProcess = nil
	session.AppRunning = false
	session.mu.Unlock()

	if proc != nil {
		s.backend.KillApp(proc)
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
