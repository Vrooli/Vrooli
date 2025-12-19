package session

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
)

// Viewport defines default browser dimensions.
type Viewport struct {
	Width  int
	Height int
}

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
}

// Option configures a Manager.
type Option func(*Manager)

// WithLogger sets a custom logger.
func WithLogger(log *logrus.Logger) Option {
	return func(m *Manager) { m.log = log }
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

// NewManager creates a unified session manager.
func NewManager(opts ...Option) (*Manager, error) {
	client, err := driver.NewClient()
	if err != nil {
		return nil, fmt.Errorf("create driver client: %w", err)
	}

	m := &Manager{
		client:          client,
		sessions:        make(map[string]*Session),
		log:             logrus.StandardLogger(),
		defaultViewport: Viewport{Width: 1280, Height: 720},
		apiHost:         resolveAPIHost(),
		apiPort:         resolveAPIPort(),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

// NewManagerWithClient creates a manager with a custom driver client (for testing).
func NewManagerWithClient(client *driver.Client, opts ...Option) *Manager {
	m := &Manager{
		client:          client,
		sessions:        make(map[string]*Session),
		log:             logrus.StandardLogger(),
		defaultViewport: Viewport{Width: 1280, Height: 720},
		apiHost:         resolveAPIHost(),
		apiPort:         resolveAPIPort(),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Create creates a new session with the given specification.
func (m *Manager) Create(ctx context.Context, spec Spec) (*Session, error) {
	spec = m.applyDefaults(spec)
	req := m.buildRequest(spec)

	m.log.WithFields(logrus.Fields{
		"execution_id": spec.ExecutionID,
		"mode":         spec.Mode.String(),
		"viewport":     fmt.Sprintf("%dx%d", spec.ViewportWidth, spec.ViewportHeight),
	}).Debug("Creating session")

	resp, err := m.client.CreateSession(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	session := &Session{
		id:     resp.SessionID,
		mode:   spec.Mode,
		client: m.client,
	}

	m.mu.Lock()
	m.sessions[session.id] = session
	m.mu.Unlock()

	m.log.WithFields(logrus.Fields{
		"session_id":   session.id,
		"execution_id": spec.ExecutionID,
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
		Viewport: driver.Viewport{
			Width:  spec.ViewportWidth,
			Height: spec.ViewportHeight,
		},
		ReuseMode: spec.ReuseMode,
		Labels:    spec.Labels,
	}

	// Frame streaming (all modes support live preview)
	if spec.FrameStreaming != nil {
		callbackURL := spec.FrameStreaming.CallbackURL
		if callbackURL == "" {
			callbackURL = m.buildFrameCallbackURL(spec)
		}
		req.FrameStreaming = &driver.FrameStreamingConfig{
			CallbackURL: callbackURL,
			Quality:     spec.FrameStreaming.Quality,
			FPS:         spec.FrameStreaming.FPS,
			Scale:       spec.FrameStreaming.Scale,
		}
	}

	// Recording-specific config
	if spec.Mode == ModeRecording || spec.Mode == ModeHybrid {
		if len(spec.StorageState) > 0 {
			req.StorageState = spec.StorageState
		}
	}

	// Execution-specific config
	if spec.Mode == ModeExecution || spec.Mode == ModeHybrid {
		req.BaseURL = spec.BaseURL

		if !spec.Capabilities.IsEmpty() {
			req.RequiredCapabilities = &driver.CapabilityRequest{
				Tabs:      spec.Capabilities.NeedsParallelTabs,
				Iframes:   spec.Capabilities.NeedsIframes,
				Uploads:   spec.Capabilities.NeedsFileUploads,
				Downloads: spec.Capabilities.NeedsDownloads,
				HAR:       spec.Capabilities.NeedsHAR,
				Video:     spec.Capabilities.NeedsVideo,
				Tracing:   spec.Capabilities.NeedsTracing,
			}
		}
	}

	return req
}

// buildFrameCallbackURL constructs the frame callback URL based on mode.
func (m *Manager) buildFrameCallbackURL(spec Spec) string {
	if spec.Mode == ModeRecording {
		return fmt.Sprintf(
			"http://%s:%s/api/v1/recordings/live/%s/frame",
			m.apiHost, m.apiPort, spec.ExecutionID,
		)
	}
	return fmt.Sprintf(
		"http://%s:%s/api/v1/executions/%s/frames",
		m.apiHost, m.apiPort, spec.ExecutionID,
	)
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
	m.mu.Lock()
	session, ok := m.sessions[sessionID]
	if ok {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()

	if !ok {
		return nil
	}

	return session.Close(ctx)
}

// CloseAll closes all active sessions.
func (m *Manager) CloseAll(ctx context.Context) {
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()

	for _, s := range sessions {
		if err := s.Close(ctx); err != nil {
			m.log.WithError(err).WithField("session_id", s.id).Warn("Failed to close session")
		}
	}
}

// ActiveCount returns the number of active sessions.
func (m *Manager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// Client returns the underlying driver client.
func (m *Manager) Client() *driver.Client {
	return m.client
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
