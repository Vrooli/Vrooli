package livedesktop

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/screenrecording"
	"scenario-to-desktop-api/shared/packaging"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service orchestrates live desktop session lifecycle.
type Service struct {
	store      Store
	backend    PlatformBackend
	logger     *slog.Logger
	vrooliRoot string
	recorder   screenrecording.Recorder
	captures   *captures.Service
	dataDir    string
}

// NewService creates a new live desktop service.
func NewService(store Store, backend PlatformBackend, logger *slog.Logger, vrooliRoot string) *Service {
	return &Service{
		store:      store,
		backend:    backend,
		logger:     logger,
		vrooliRoot: vrooliRoot,
		dataDir:    filepath.Join(vrooliRoot, "scenarios", "scenario-to-desktop", "data", "livedesktop"),
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

// StartSession creates a new live desktop session with remote access.
func (s *Service) StartSession(ctx context.Context, cfg SessionConfig) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("starting desktop session: %w", err)
	}
	if cfg.Width == 0 {
		cfg.Width = 1280
	}
	if cfg.Height == 0 {
		cfg.Height = 720
	}

	// Validate platform against available backend
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
		CreatedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		stopCh:        make(chan struct{}),
	}

	if err := s.store.Create(session); err != nil {
		return nil, fmt.Errorf("storing session: %w", err)
	}

	// Create display via platform backend
	display, err := s.backend.CreateDisplay(cfg.Width, cfg.Height)
	if err != nil {
		session.SetError(fmt.Sprintf("display creation failed: %v", err))
		_ = s.store.Update(session)
		return session, fmt.Errorf("creating display: %w", err)
	}
	session.Display = display

	// Start remote access via platform backend
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
	if err := ctx.Err(); err != nil {
		s.backend.StopRemoteAccess(handle)
		display.Stop()
		session.SetError("session creation cancelled")
		_ = s.store.Update(session)
		return session, fmt.Errorf("starting desktop session: %w", err)
	}

	session.SetState(StateRunning)
	_ = s.store.Update(session)

	s.logger.Info("live desktop session started",
		"session_id", session.ID,
		"display", display.DisplayID(),
		"platform", s.backend.PlatformID(),
		"vnc_port", info.Port,
		"ws_port", info.WSPort,
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
	session.signalStop()

	// Kill launched app process
	s.killAppProcess(session)

	// Stop remote access
	if session.RemoteAccess != nil {
		s.backend.StopRemoteAccess(session.RemoteAccess)
	}

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

	// Collect launch options from session state
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

	// Launch through platform backend
	proc, err := s.backend.LaunchApp(context.Background(), session.Display, appPath, opts)
	if err != nil {
		return err
	}

	session.mu.Lock()
	session.AppProcess = proc
	session.AppRunning = true
	session.mu.Unlock()

	// Start process monitor if backend supports it
	monitorFactory := s.backend.NewMonitorFactory()
	if monitorFactory != nil {
		monitor := monitorFactory.NewMonitor()
		if err := monitor.Start(context.Background(), proc.PID(), session.Display.DisplayID(), session.Width, session.Height); err != nil {
			s.logger.Warn("failed to start process monitor", "error", err)
		} else {
			session.SetMonitor(monitor)
		}
	}

	// Reap the process in the background so it doesn't become a zombie. Its
	// lifecycle belongs to the desktop session, not to the request that launched
	// the app.
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			if !proc.IsRunning() {
				if m := session.GetMonitor(); m != nil {
					m.Stop()
				}
				session.SetAppRunning(false)
				return
			}
			select {
			case <-session.Done():
				return
			case <-ticker.C:
			}
		}
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
	// Stop the monitor before killing the process so it can compute final metrics.
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
