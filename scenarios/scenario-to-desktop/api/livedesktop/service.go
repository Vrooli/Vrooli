package livedesktop

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"

	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/procmetrics"
	"scenario-to-desktop-api/screenrecording"
	"scenario-to-desktop-api/shared/packaging"
)

// VNCStartFunc is the signature for starting a VNC session.
type VNCStartFunc func(display string) (vncPort, wsPort int, x11vncCmd, websockifyCmd *exec.Cmd, err error)

// VNCStopFunc is the signature for stopping VNC processes.
type VNCStopFunc func(session *Session)

// Service orchestrates live desktop session lifecycle.
type Service struct {
	store          Store
	displayMgr     screenrecording.DisplayManager
	logger         *slog.Logger
	vrooliRoot     string
	startVNC       VNCStartFunc
	stopVNC        VNCStopFunc
	recorder       screenrecording.Recorder
	captures       *captures.Service
	shell          ShellFunc
	dataDir        string
	monitorFactory procmetrics.MonitorFactory
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
		shell:      defaultShell,
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

// WithMonitor sets the process monitor factory for tracking app startup time and resource usage.
func (s *Service) WithMonitor(factory procmetrics.MonitorFactory) {
	s.monitorFactory = factory
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

	// Build command and environment based on session control state
	session.mu.Lock()
	networkMode := session.NetworkMode
	bandwidthKbps := session.BandwidthKbps
	darkMode := session.DarkMode
	locale := session.Locale
	envVars := make(map[string]string)
	for k, v := range session.EnvVars {
		envVars[k] = v
	}
	session.mu.Unlock()

	// Build environment
	env := os.Environ()
	env = append(env, fmt.Sprintf("DISPLAY=%s", session.Display.DisplayID))

	// Apply custom env vars
	for k, v := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Apply dark mode
	if darkMode {
		env = append(env, "GTK_THEME=Adwaita:dark")
	}

	// Apply locale
	if locale != "" {
		env = append(env, fmt.Sprintf("LANG=%s", locale))
		env = append(env, fmt.Sprintf("LC_ALL=%s", locale))
	}

	// Build command args based on network mode
	var cmdName string
	var cmdArgs []string

	switch networkMode {
	case "offline":
		cmdName = "unshare"
		cmdArgs = []string{"--net", appPath}
	case "slow":
		// Use unshare + tc for bandwidth limiting
		tcCmd := fmt.Sprintf("tc qdisc add dev lo root tbf rate %dkbit burst 32kbit latency 400ms && exec %s",
			bandwidthKbps, appPath)
		cmdName = "unshare"
		cmdArgs = []string{"--net", "sh", "-c", tcCmd}
	default:
		cmdName = appPath
		cmdArgs = nil
	}

	// Append dark mode flag for Electron apps
	if darkMode {
		if cmdName == appPath {
			cmdArgs = append(cmdArgs, "--force-dark-mode")
		} else {
			// For unshare-wrapped commands, the app path is in args
			cmdArgs = append(cmdArgs, "--force-dark-mode")
		}
	}

	cmd := exec.CommandContext(context.Background(), cmdName, cmdArgs...)
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launching app: %w", err)
	}

	session.mu.Lock()
	session.AppCmd = cmd
	session.AppRunning = true
	session.mu.Unlock()

	// Start process monitor if factory is configured.
	if s.monitorFactory != nil {
		monitor := s.monitorFactory.NewMonitor()
		if err := monitor.Start(context.Background(), cmd.Process.Pid, session.Display.DisplayID, session.Width, session.Height); err != nil {
			s.logger.Warn("failed to start process monitor", "error", err)
		} else {
			session.SetMonitor(monitor)
		}
	}

	// Reap the process in the background so it doesn't become a zombie
	go func() {
		_ = cmd.Wait()
		if m := session.GetMonitor(); m != nil {
			m.Stop()
		}
		session.SetAppRunning(false)
	}()

	s.logger.Info("app launched on desktop", "session_id", sessionID, "app", appPath,
		"network_mode", networkMode, "dark_mode", darkMode, "locale", locale)
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
	cmd := session.AppCmd
	session.AppCmd = nil
	session.AppRunning = false
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
