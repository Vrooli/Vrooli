package livedesktop

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/procmetrics"
	"scenario-to-desktop-api/screenrecording"
	"scenario-to-desktop-api/shared/packaging"
	"scenario-to-desktop-api/target"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/cli-core/cliutil"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

// Service orchestrates live desktop session lifecycle.
type Service struct {
	store               Store
	backend             PlatformBackend
	logger              *slog.Logger
	vrooliRoot          string
	artifactStagingRoot string
	recorder            screenrecording.Recorder
	videoInspector      func(context.Context, string) (screenrecording.MediaInspection, error)
	captures            *captures.Service
	windowController    *procmetrics.XdotoolDetector
	dataDir             string
	rendererURLResolver func(context.Context, string) (string, error)
}

// NewService creates a new live desktop service.
func NewService(store Store, backend PlatformBackend, logger *slog.Logger, vrooliRoot string) *Service {
	return &Service{
		store:               store,
		backend:             backend,
		logger:              logger,
		vrooliRoot:          vrooliRoot,
		dataDir:             filepath.Join(vrooliRoot, "scenarios", "scenario-to-desktop", "data", "livedesktop"),
		rendererURLResolver: resolveScenarioRendererURL,
	}
}

func resolveScenarioRendererURL(ctx context.Context, scenarioName string) (string, error) {
	_ = ctx
	portText := cliutil.DetectPortFromVrooli(scenarioName, "UI_PORT")()
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 {
		if err == nil {
			err = fmt.Errorf("port detector returned %q", portText)
		}
		return "", fmt.Errorf("resolve UI_PORT for %q: %w", scenarioName, err)
	}
	return fmt.Sprintf("http://127.0.0.1:%d", port), nil
}

// WithRendererURLResolver overrides the live UI-port resolver used by the
// validation launch seam. Tests can keep this boundary deterministic without
// weakening the production target identity contract.
func (s *Service) WithRendererURLResolver(resolver func(context.Context, string) (string, error)) {
	s.rendererURLResolver = resolver
}

// WithRecorder sets the screen recorder on the service.
func (s *Service) WithRecorder(r screenrecording.Recorder) {
	s.recorder = r
}

// WithVideoInspector overrides the producer-side recording integrity check.
// Production uses screenrecording.InspectVideo; the seam keeps action tests
// deterministic without weakening the runtime check.
func (s *Service) WithVideoInspector(inspector func(context.Context, string) (screenrecording.MediaInspection, error)) {
	s.videoInspector = inspector
}

// WithCaptures sets the captures service for persistent capture storage.
func (s *Service) WithCaptures(svc *captures.Service) {
	s.captures = svc
}

// WithWindowController wires the shared xdotool seam used by both metrics and
// interactive window actions. Keeping one detector preserves its availability
// cache and gives callers one honest degraded decision.
func (s *Service) WithWindowController(detector *procmetrics.XdotoolDetector) {
	s.windowController = detector
}

// WithDataDir overrides the data directory for screenshots and recordings.
func (s *Service) WithDataDir(dir string) {
	s.dataDir = dir
}

// WithArtifactStagingRoot adds the canonical pipeline staging directory to
// exact artifact lookup. Validation cells bind to a digest, so a staged
// artifact must remain selectable even when the pipeline intentionally uses a
// temporary output location.
func (s *Service) WithArtifactStagingRoot(root string) {
	s.artifactStagingRoot = strings.TrimSpace(root)
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

// LaunchElectronValidation starts the explicit validation target path. Normal
// live-desktop launches remain unchanged and do not expose CDP. The target
// session owns the process, profile, port, renderer selection, and cleanup.
func (s *Service) LaunchElectronValidation(ctx context.Context, sessionID, appPath string, opts target.ElectronLaunchOptions, renderer target.RendererExpectation) (*domainv1.ElectronTarget, error) {
	session, err := s.store.Get(sessionID)
	if err != nil {
		return nil, err
	}
	if session.Display == nil || !session.Display.IsRunning() {
		return nil, fmt.Errorf("session display is not running")
	}
	if appPath == "" {
		appPath, err = s.findArtifact(session.ScenarioName)
		if err != nil {
			return nil, fmt.Errorf("auto-discover artifact: %w", err)
		}
	}
	if opts.ScenarioName == "" {
		opts.ScenarioName = session.ScenarioName
	}
	if strings.TrimSpace(opts.ScenarioName) == "" {
		return nil, fmt.Errorf("Electron validation scenario name is required")
	}
	if renderer.URLPrefix == "http://127.0.0.1:" {
		if s.rendererURLResolver == nil {
			return nil, fmt.Errorf("Electron validation renderer URL resolver is unavailable")
		}
		resolved, resolveErr := s.rendererURLResolver(ctx, opts.ScenarioName)
		if resolveErr != nil {
			return nil, fmt.Errorf("resolve Electron validation renderer: %w", resolveErr)
		}
		renderer.URLPrefix = strings.TrimSpace(resolved)
		if renderer.URLPrefix == "" {
			return nil, fmt.Errorf("resolve Electron validation renderer returned an empty URL")
		}
	}
	apiPort, err := discovery.ResolveScenarioPort(ctx, opts.ScenarioName, "API_PORT")
	if err != nil {
		return nil, fmt.Errorf("resolve Electron validation scenario API: %w", err)
	}
	s.killAppProcess(session)
	environment := make(map[string]string)
	environment["DISPLAY"] = session.Display.DisplayID()
	// The bundled app owns its renderer and local service process, while the
	// provider-owned routed lease is installed on the selected scenario API.
	// Give the generated Electron main process the target-owned endpoint so its
	// validation-only proxy can carry app-owned API requests to that leased
	// service without exposing the lease identifier.
	environment["VROOLI_VALIDATION_API_URL"] = fmt.Sprintf("http://127.0.0.1:%d", apiPort)
	if strings.TrimSpace(renderer.URLPrefix) != "" {
		environment["VROOLI_VALIDATION_RENDERER_URL"] = renderer.URLPrefix
	}
	session.mu.Lock()
	if session.DarkMode {
		environment["GTK_THEME"] = "Adwaita:dark"
	}
	if session.Locale != "" {
		environment["LANG"] = session.Locale
		environment["LC_ALL"] = session.Locale
	}
	for key, value := range session.EnvVars {
		if _, exists := environment[key]; !exists {
			environment[key] = value
		}
	}
	session.mu.Unlock()

	electronSession, err := target.StartElectronSession(ctx, target.ElectronSessionOptions{
		ArtifactPath: appPath,
		Launch:       opts,
		Renderer:     renderer,
		Environment:  environment,
	})
	if err != nil {
		return nil, err
	}
	session.mu.Lock()
	session.ElectronValidation = electronSession
	session.AppRunning = true
	session.mu.Unlock()
	_ = s.store.Update(session)
	s.logger.Info("Electron validation target launched", "session_id", sessionID, "pid", electronSession.PID(), "renderer_id", electronSession.Target().GetRendererId())
	return electronSession.Target(), nil
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
	electronSession := session.ElectronValidation
	session.ElectronValidation = nil
	session.AppRunning = false
	session.mu.Unlock()

	if proc != nil {
		s.backend.KillApp(proc)
	}
	if electronSession != nil {
		if err := electronSession.Close(context.Background()); err != nil {
			s.logger.Warn("failed to clean up Electron validation target", "session_id", session.ID, "error", err)
		}
	}
}

// FindArtifact finds the latest build artifact for a scenario.
func (s *Service) FindArtifact(scenarioName string) (string, error) {
	return s.findArtifact(scenarioName)
}

// FindArtifactByDigest resolves the exact artifact selected by a validation
// cell. It searches both the scenario's durable output and pipeline staging;
// temporary pipeline outputs are valid validation inputs until their evidence
// is handed off, and must not be replaced with an unrelated older build.
func (s *Service) FindArtifactByDigest(scenarioName, digest string) (string, error) {
	if s == nil || s.vrooliRoot == "" {
		return "", fmt.Errorf("vrooliRoot not configured")
	}
	want := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(digest)), "sha256:")
	if want == "" {
		return "", fmt.Errorf("artifact digest is required")
	}
	roots := []string{
		filepath.Join(s.vrooliRoot, "scenarios", scenarioName, "platforms", "electron", "dist-electron"),
	}
	if s.artifactStagingRoot != "" {
		roots = append(roots, filepath.Join(s.artifactStagingRoot, scenarioName))
	}
	var matches []string
	for _, root := range roots {
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return nil
				}
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil || !info.Mode().IsRegular() {
				return nil
			}
			actual, err := digestFile(path)
			if err != nil {
				return err
			}
			if actual == want {
				matches = append(matches, path)
			}
			return nil
		}); err != nil {
			return "", fmt.Errorf("search artifact root %q: %w", root, err)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no artifact for scenario %q matches digest sha256:%s", scenarioName, want)
	}
	return matches[0], nil
}

func digestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
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
