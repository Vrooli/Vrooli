package session

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

// Viewport is the driver's canonical width/height value used for defaults.
type Viewport = driver.Viewport

// Manager handles session lifecycle with tracking and cleanup.
type Manager struct {
	client   *driver.Client
	sessions map[string]*Session
	mu       sync.RWMutex
	log      *logrus.Logger

	// Defaults
	defaultViewport Viewport
	apiHost         string
	apiPort         string

	executionArtifactsRoot string
}

// Option configures a Manager.
type Option func(*Manager)

// WithLogger sets a custom logger.
func WithLogger(log *logrus.Logger) Option {
	return func(m *Manager) {
		if log != nil {
			m.log = log
		}
	}
}

// WithDefaultViewport sets default viewport dimensions.
func WithDefaultViewport(width, height int) Option {
	return func(m *Manager) {
		m.defaultViewport = Viewport{Width: width, Height: height}
	}
}

// WithAPIEndpoint sets the API host and port for frame callback URLs.
func WithAPIEndpoint(host, port string) Option {
	return func(m *Manager) {
		m.apiHost = host
		m.apiPort = port
	}
}

// WithExecutionArtifactsRoot sets the base directory for execution-level artifacts.
// This path is used to construct stable per-execution artifact directories.
func WithExecutionArtifactsRoot(root string) Option {
	return func(m *Manager) {
		m.executionArtifactsRoot = strings.TrimSpace(root)
	}
}

// NewManager creates a unified session manager.
func NewManager(opts ...Option) (*Manager, error) {
	client, err := driver.NewClient()
	if err != nil {
		return nil, fmt.Errorf("create driver client: %w", err)
	}

	m := &Manager{
		client:                 client,
		sessions:               make(map[string]*Session),
		log:                    logrus.StandardLogger(),
		defaultViewport:        Viewport{Width: 1280, Height: 720},
		apiHost:                resolveAPIHost(),
		apiPort:                resolveAPIPort(),
		executionArtifactsRoot: "",
	}

	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

// NewManagerWithClient creates a manager with a custom driver client (for testing).
func NewManagerWithClient(client *driver.Client, opts ...Option) *Manager {
	m := &Manager{
		client:                 client,
		sessions:               make(map[string]*Session),
		log:                    logrus.StandardLogger(),
		defaultViewport:        Viewport{Width: 1280, Height: 720},
		apiHost:                resolveAPIHost(),
		apiPort:                resolveAPIPort(),
		executionArtifactsRoot: "",
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Create creates a new session with the given specification.
func (m *Manager) Create(ctx context.Context, spec Spec) (*Session, error) {
	spec.PermissionProfile = clonePermissionProfile(spec.PermissionProfile)
	if spec.AudioMode != sessionpermission.AudioModeNone {
		if spec.PermissionProfile == nil || spec.PermissionProfile.AudioMode != spec.AudioMode || spec.AppTarget != nil || spec.Mode == ModeRecording {
			return nil, errors.New("audio session requires a matching Agent Manager audio mode on a managed execution session")
		}
		if strings.TrimSpace(spec.FakeMicrophoneWav) == "" || !filepath.IsAbs(spec.FakeMicrophoneWav) {
			return nil, errors.New("audio session requires an owner-resolved absolute fixture path")
		}
		if spec.AudioPlaybackPauseMS < 0 || spec.AudioPlaybackPauseMS > 1000 || spec.AudioPlaybackStartDelayMS < 0 || spec.AudioPlaybackStartDelayMS > 5000 {
			return nil, errors.New("audio playback timing is outside the BAS limit")
		}
		if spec.AudioMode == sessionpermission.AudioModePipeWire {
			if !spec.AudioDeviceEvidence || !spec.AudioPlaybackDeferStart {
				return nil, errors.New("PipeWire qualification requires device evidence and deferred fixture playback")
			}
		} else if spec.AudioMode == sessionpermission.AudioModeFakeMicrophone {
			if spec.AudioDeviceEvidence || spec.AudioPlaybackPauseMS != 0 || spec.AudioPlaybackStartDelayMS != 0 || spec.AudioPlaybackDeferStart {
				return nil, errors.New("fake-microphone mode cannot request host-device evidence or PipeWire playback controls")
			}
		} else {
			return nil, errors.New("unsupported Agent Manager audio mode")
		}
	} else if spec.AudioDeviceEvidence || spec.AudioPlaybackPauseMS != 0 || spec.AudioPlaybackStartDelayMS != 0 || spec.AudioPlaybackDeferStart {
		return nil, errors.New("audio controls require an Agent Manager audio mode")
	}
	permissionScoped := spec.Authorizer != nil || spec.OpenAuthorizer != nil || spec.OpenRequest != nil || spec.CallerResolver != nil || spec.PermissionProfile != nil
	var expectedCallerID string
	if permissionScoped {
		if spec.Authorizer == nil || spec.OpenAuthorizer == nil || spec.OpenRequest == nil || spec.CallerResolver == nil || spec.PermissionProfile == nil {
			return nil, errors.New("create session: permission authorizer requires pre-open admission")
		}
		if err := spec.PermissionProfile.Validate(); err != nil {
			return nil, fmt.Errorf("create session: invalid browser permission profile: %w", err)
		}
		bound, ok := spec.Authorizer.(sessionpermission.ProfileBoundAuthorizer)
		if !ok || !reflect.DeepEqual(bound.ProfileSnapshot(), *spec.PermissionProfile) {
			return nil, errors.New("create session: driver permission profile must match the broker authorizer")
		}
		if strings.TrimSpace(spec.PermissionProfile.ID) == "" {
			return nil, errors.New("create session: permission profile id is required")
		}
		callerID, err := spec.CallerResolver.BrowserCallerID(ctx)
		if err != nil || callerID == "" {
			return nil, errors.New("create session: trusted browser caller identity unavailable")
		}
		expectedCallerID = callerID
		openRequest := *spec.OpenRequest
		openRequest.OwnerID = spec.ExecutionID.String()
		openRequest.CallerID = callerID
		openRequest.ProfileID = spec.PermissionProfile.ID
		if err := openRequest.Validate(*spec.PermissionProfile); err != nil {
			return nil, fmt.Errorf("create session: invalid browser admission request: %w", err)
		}
		if err := spec.OpenAuthorizer.AuthorizeOpen(ctx, openRequest); err != nil {
			return nil, fmt.Errorf("create session: browser admission denied: %w", err)
		}
	}
	spec = m.applyDefaults(spec)
	req := m.buildRequest(spec)

	m.log.WithFields(logrus.Fields{
		"execution_id":  spec.ExecutionID,
		"mode":          spec.Mode.String(),
		"viewport":      fmt.Sprintf("%dx%d", spec.ViewportWidth, spec.ViewportHeight),
		"has_recording": spec.Recording != nil,
	}).Debug("Creating session")

	resp, err := m.client.CreateSession(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	session, err := m.adoptCreated(resp, spec.ExecutionID.String(), spec.Mode, spec.Recording, spec.AppTarget != nil, spec.BrowserBackend)
	if err != nil {
		return nil, err
	}
	session.mu.Lock()
	if session.authorizer != nil && !reflect.DeepEqual(session.permissionProfile, spec.PermissionProfile) {
		session.mu.Unlock()
		return nil, errors.New("create session: existing lease has a different permission profile")
	}
	if session.expectedCallerID != "" && expectedCallerID != "" && session.expectedCallerID != expectedCallerID {
		session.mu.Unlock()
		return nil, errors.New("create session: existing lease belongs to a different trusted caller")
	}
	if session.authorizer == nil {
		session.authorizer = spec.Authorizer
		session.permissionProfile = clonePermissionProfile(spec.PermissionProfile)
	}
	if session.expectedCallerID == "" {
		session.expectedCallerID = expectedCallerID
	}
	session.callerResolver = spec.CallerResolver
	session.mu.Unlock()
	return session, nil
}

func clonePermissionProfile(profile *sessionpermission.Profile) *sessionpermission.Profile {
	if profile == nil {
		return nil
	}
	copy := *profile
	copy.URLAllow = append([]string(nil), profile.URLAllow...)
	copy.URLDeny = append([]string(nil), profile.URLDeny...)
	copy.CredentialIDs = append([]string(nil), profile.CredentialIDs...)
	if profile.ExtensionPermissions != nil {
		copy.ExtensionPermissions = make(map[string][]string, len(profile.ExtensionPermissions))
		for id, permissions := range profile.ExtensionPermissions {
			copy.ExtensionPermissions[id] = append([]string(nil), permissions...)
		}
	}
	return &copy
}

// CreateForDrill admits a session through the broker using the drill's
// one-shot scoped token. The returned session carries the lease, so even a
// successful fault drill is closed through the normal owner-checked path.
func (m *Manager) CreateForDrill(ctx context.Context, req *driver.CreateSessionRequest, token string) (*Session, error) {
	resp, err := m.client.CreateSessionForDrill(ctx, req, token)
	if err != nil {
		return nil, err
	}
	return m.adoptCreated(resp, req.ExecutionID, ModeExecution, nil, false, req.Options.BrowserBackend)
}

var errReturnedSessionLeaseCollision = errors.New("returned browser session id is already owned by another lease")

// adoptCreated closes only the exact newly returned lease when a session-ID
// collision prevents registration. The bounded detached close still runs if
// the request was canceled after the driver created its session.
func (m *Manager) adoptCreated(resp *driver.CreateSessionResponse, executionID string, mode Mode, recording *RecordingCallbacks, appTarget bool, browserBackend string) (*Session, error) {
	session, err := m.adopt(resp, executionID, mode, recording, appTarget, browserBackend)
	if err == nil || !errors.Is(err, errReturnedSessionLeaseCollision) {
		return session, err
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, cleanupErr := m.client.CloseSessionWithLease(cleanupCtx, resp.SessionID, executionID, resp.LeaseID); cleanupErr != nil {
		return nil, errors.Join(err, fmt.Errorf("close unadopted browser session lease: %w", cleanupErr))
	}
	return nil, err
}

// ListObservedSessions provides recovery metadata without refreshing driver
// activity. Session-route access remains inside the broker package.
func (m *Manager) ListObservedSessions(ctx context.Context) ([]driver.ObservedSession, error) {
	return m.client.ListObservedSessions(ctx)
}

// ForceCloseSession performs the driver's authenticated recovery close.
func (m *Manager) ForceCloseSession(ctx context.Context, sessionID, expectedOwnerExecutionID, expectedLeaseID string) error {
	return m.client.ForceCloseSession(ctx, sessionID, expectedOwnerExecutionID, expectedLeaseID)
}

func (m *Manager) adopt(resp *driver.CreateSessionResponse, executionID string, mode Mode, recording *RecordingCallbacks, appTarget bool, browserBackend string) (*Session, error) {

	m.mu.Lock()
	defer m.mu.Unlock()
	if existing := m.sessions[resp.SessionID]; existing != nil && existing.executionID == executionID && existing.leaseID == resp.LeaseID {
		existing.mu.Lock()
		defer existing.mu.Unlock()
		if existing.browserBackend != browserBackend {
			return nil, fmt.Errorf("create session: returned lease belongs to browser backend %q, not %q", existing.browserBackend, browserBackend)
		}
		if existing.terminal != nil {
			return nil, fmt.Errorf("create session: returned lease is terminating")
		}
		if resp.LastInstructionSequence > existing.lastInstructionSequence {
			existing.lastInstructionSequence = resp.LastInstructionSequence
		}
		return existing, nil
	}
	if existing := m.sessions[resp.SessionID]; existing != nil {
		return nil, fmt.Errorf("%w: session %q is already indexed to execution %q lease %q", errReturnedSessionLeaseCollision, resp.SessionID, existing.executionID, existing.leaseID)
	}

	session := &Session{
		id:                      resp.SessionID,
		initialDriverPageID:     resp.ActivePageID,
		activeDriverPageID:      resp.ActivePageID,
		lastInstructionSequence: resp.LastInstructionSequence,
		executionID:             executionID,
		leaseID:                 resp.LeaseID,
		browserBackend:          browserBackend,
		mode:                    mode,
		appTarget:               appTarget,
		client:                  m.client,
		actualViewport:          resp.ActualViewport,
		identityEvidence:        resp.IdentityEvidence,
		audioDeviceEvidence:     resp.AudioDeviceEvidence,
		recording:               recording,
	}
	session.onTerminal = func() {
		m.forget(session.id, session)
	}

	m.sessions[session.id] = session

	m.log.WithFields(logrus.Fields{
		"session_id":   session.id,
		"execution_id": executionID,
	}).Info("Session created")

	return session, nil
}

// applyDefaults fills in default values for missing spec fields.
func (m *Manager) applyDefaults(spec Spec) Spec {
	if spec.ViewportWidth <= 0 {
		spec.ViewportWidth = m.defaultViewport.Width
	}
	if spec.ViewportHeight <= 0 {
		spec.ViewportHeight = m.defaultViewport.Height
	}
	if spec.ReuseMode == "" {
		spec.ReuseMode = "reuse"
	}
	if spec.Labels == nil {
		spec.Labels = make(map[string]string)
	}
	spec.Labels["mode"] = spec.Mode.String()

	// Apply frame streaming defaults
	if spec.FrameStreaming != nil {
		if spec.FrameStreaming.Quality <= 0 {
			spec.FrameStreaming.Quality = 55
		}
		if spec.FrameStreaming.FPS <= 0 {
			spec.FrameStreaming.FPS = 6
		}
		if spec.FrameStreaming.Scale == "" {
			spec.FrameStreaming.Scale = "css"
		}
	}

	return spec
}

// buildRequest converts a Spec to a driver.CreateSessionRequest.
func (m *Manager) buildRequest(spec Spec) *driver.CreateSessionRequest {
	req := &driver.CreateSessionRequest{
		ExecutionID: spec.ExecutionID.String(),
		WorkflowID:  spec.WorkflowID.String(),
		Options: driver.SessionOptions{
			Viewport: driver.Viewport{
				Width:  spec.ViewportWidth,
				Height: spec.ViewportHeight,
			},
			ReuseMode:             spec.ReuseMode,
			BrowserBackend:        spec.BrowserBackend,
			FrameScale:            "css",
			Labels:                spec.Labels,
			SessionProfileVersion: spec.SessionProfileVersion,
			BrowserProfile:        spec.BrowserProfile,
			AppTarget:             spec.AppTarget,
			ValidationContext:     spec.ValidationContext,
			PermissionProfile:     spec.PermissionProfile,
		},
	}
	if spec.FrameStreaming != nil && spec.FrameStreaming.Scale != "" {
		req.Options.FrameScale = spec.FrameStreaming.Scale
	}

	// Frame streaming (all modes support live preview)
	if spec.FrameStreaming != nil {
		streamURL := spec.FrameStreaming.URL
		if streamURL == "" {
			streamURL = m.buildFrameStreamURL()
		}
		req.Options.FrameStreaming = &driver.FrameStreamingConfig{
			URL:     streamURL,
			Quality: spec.FrameStreaming.Quality,
			FPS:     spec.FrameStreaming.FPS,
		}
	}

	// Storage state for authenticated sessions (all modes support this)
	if len(spec.StorageState) > 0 {
		req.Options.StorageState = spec.StorageState
	}

	// Execution-specific config
	if spec.Mode == ModeExecution || spec.Mode == ModeHybrid {
		req.Options.BaseURL = spec.BaseURL

		if !spec.Capabilities.IsEmpty() {
			req.Options.RequiredCapabilities = &driver.CapabilityRequest{
				Tabs:          spec.Capabilities.NeedsParallelTabs,
				Iframes:       spec.Capabilities.NeedsIframes,
				Uploads:       spec.Capabilities.NeedsFileUploads,
				Downloads:     spec.Capabilities.NeedsDownloads,
				HAR:           spec.Capabilities.NeedsHAR,
				Video:         spec.Capabilities.NeedsVideo,
				Tracing:       spec.Capabilities.NeedsTracing,
				PerfTrace:     spec.Capabilities.NeedsPerfTrace,
				Accessibility: spec.Capabilities.NeedsAccessibility,
			}
		}
		if paths := m.buildArtifactPaths(spec, req.Options.RequiredCapabilities); paths != nil {
			req.Options.ArtifactPaths = paths
		}
		if spec.FakeMicrophoneWav != "" {
			req.Options.FakeMedia = &driver.FakeMediaConfig{MicrophoneWav: spec.FakeMicrophoneWav}
		}
		if spec.AudioMode != sessionpermission.AudioModeNone {
			req.Options.Permissions = []string{"microphone"}
			req.Options.AudioDeviceEvidence = spec.AudioDeviceEvidence
			req.Options.AudioPlaybackPauseMS = spec.AudioPlaybackPauseMS
			req.Options.AudioPlaybackStartDelayMS = spec.AudioPlaybackStartDelayMS
			req.Options.AudioPlaybackDeferStart = spec.AudioPlaybackDeferStart
		}
	}

	return req
}

func (m *Manager) buildArtifactPaths(spec Spec, caps *driver.CapabilityRequest) *driver.ArtifactPaths {
	root := strings.TrimSpace(m.executionArtifactsRoot)
	if root == "" || caps == nil {
		return nil
	}

	execID := spec.ExecutionID.String()
	artifactRoot := filepath.Join(root, execID, "artifacts")
	paths := &driver.ArtifactPaths{
		Root: artifactRoot,
	}

	if caps.Video {
		paths.VideoDir = filepath.Join(artifactRoot, "videos")
	}
	if caps.HAR {
		paths.HARPath = filepath.Join(artifactRoot, "har", fmt.Sprintf("execution-%s.har", execID))
	}
	if caps.Tracing {
		paths.TracePath = filepath.Join(artifactRoot, "traces", fmt.Sprintf("execution-%s.zip", execID))
	}
	if caps.PerfTrace {
		paths.PerfDir = filepath.Join(artifactRoot, "performance")
	}
	if caps.Accessibility {
		paths.AccessibilityDir = filepath.Join(artifactRoot, "accessibility")
	}

	if strings.TrimSpace(paths.VideoDir) == "" && strings.TrimSpace(paths.HARPath) == "" &&
		strings.TrimSpace(paths.TracePath) == "" && strings.TrimSpace(paths.PerfDir) == "" &&
		strings.TrimSpace(paths.AccessibilityDir) == "" {
		return nil
	}

	return paths
}

// buildFrameStreamURL configures the one driver-to-hub frame transport.
func (m *Manager) buildFrameStreamURL() string {
	return fmt.Sprintf("ws://%s:%s/ws/frames", m.apiHost, m.apiPort)
}

// Get returns a session by ID.
func (m *Manager) Get(sessionID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	return s, ok
}

// Close closes a session by ID.
func (m *Manager) Close(ctx context.Context, sessionID string) error {
	m.mu.RLock()
	session, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	return session.Close(ctx)
}

// CloseAll closes all active sessions.
func (m *Manager) CloseAll(ctx context.Context) {
	m.mu.RLock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.mu.RUnlock()

	for _, s := range sessions {
		if err := s.Close(ctx); err != nil {
			m.log.WithError(err).WithField("session_id", s.id).Warn("Failed to close session")
		}
	}
}

// forget removes a terminal session while protecting against a future
// replacement that happens to reuse the same driver-generated id.
func (m *Manager) forget(sessionID string, expected *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if current, ok := m.sessions[sessionID]; ok && current == expected {
		delete(m.sessions, sessionID)
	}
}

// ActiveCount returns the number of active sessions.
func (m *Manager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// Health checks driver availability without exposing the client to callers.
func (m *Manager) Health(ctx context.Context) error { return m.client.Health(ctx) }

// CircuitBreakerState reports driver health telemetry without exposing routes.
func (m *Manager) CircuitBreakerState() string { return m.client.CircuitBreakerState() }

// SetAdministrativeSecret configures the one-shot recovery credential on the
// broker-owned transport without exposing that transport to production callers.
func (m *Manager) SetAdministrativeSecret(secret string) { m.client.SetAdministrativeSecret(secret) }

// RouteSessionRequest resolves an API-owned session before forwarding a raw
// response-preserving driver endpoint through its session owner.
func (m *Manager) RouteSessionRequest(ctx context.Context, sessionID, method, suffix string, body []byte, headers http.Header) (*http.Response, error) {
	owned, ok := m.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return owned.RouteRequest(ctx, method, suffix, body, headers)
}

// ManagedExtensionRequest forwards a Broker-authorized extension operation
// only through the exact API-owned managed session owner/lease/profile.
func (m *Manager) ManagedExtensionRequest(ctx context.Context, sessionID, ownerID, leaseID, profileID, permissionRef, callerID, method, suffix string, body []byte) (*http.Response, error) {
	owned, ok := m.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return owned.ManagedExtensionRequest(ctx, ownerID, leaseID, profileID, permissionRef, callerID, method, suffix, body)
}

func resolveAPIHost() string {
	if host := os.Getenv("API_HOST"); host != "" {
		return host
	}
	return "127.0.0.1"
}

func resolveAPIPort() string {
	if port := os.Getenv("API_PORT"); port != "" {
		return port
	}
	return "8080"
}
