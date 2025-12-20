package engine

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
)

// PlaywrightEngine talks to a local Playwright driver (Node) over HTTP/Unix
// sockets. The driver is responsible for translating CompiledInstructions into
// Playwright actions and returning contract StepOutcome payloads.
type PlaywrightEngine struct {
	sessions *session.Manager
	log      *logrus.Logger
}

// NewPlaywrightEngine constructs an engine using environment configuration.
// It requires PLAYWRIGHT_DRIVER_URL unless allowDefault is true, in which case
// it falls back to localhost for developer setups.
func NewPlaywrightEngine(log *logrus.Logger) (*PlaywrightEngine, error) {
	mgr, err := session.NewManager(session.WithLogger(log))
	if err != nil {
		return nil, err
	}
	return &PlaywrightEngine{sessions: mgr, log: log}, nil
}

// NewPlaywrightEngineWithDefault allows the localhost fallback when explicit
// configuration is absent (useful for local development and tests).
func NewPlaywrightEngineWithDefault(log *logrus.Logger) (*PlaywrightEngine, error) {
	// NewManager already allows default, so this is the same as NewPlaywrightEngine
	return NewPlaywrightEngine(log)
}

// NewPlaywrightEngineWithHTTPClient constructs an engine with a custom HTTP client.
// This is primarily used for testing to inject mock HTTP responses.
func NewPlaywrightEngineWithHTTPClient(driverURL string, httpClient HTTPDoer, log *logrus.Logger) (*PlaywrightEngine, error) {
	if httpClient == nil {
		return nil, errors.New("httpClient is required")
	}
	client, err := driver.NewClientWithURL(driverURL, driver.WithHTTPClient(httpClient), driver.WithLogger(log))
	if err != nil {
		return nil, err
	}
	mgr := session.NewManagerWithClient(client, session.WithLogger(log))
	return &PlaywrightEngine{sessions: mgr, log: log}, nil
}

// Name returns the engine identifier.
func (e *PlaywrightEngine) Name() string { return "playwright" }

// Capabilities returns a conservative capability descriptor for the local
// Playwright driver. Update when driver gains richer support (HAR/video, etc.).
func (e *PlaywrightEngine) Capabilities(ctx context.Context) (contracts.EngineCapabilities, error) {
	if e == nil {
		return contracts.EngineCapabilities{}, errors.New("engine not configured")
	}
	if err := e.sessions.Client().Health(ctx); err != nil {
		return contracts.EngineCapabilities{}, err
	}
	return contracts.EngineCapabilities{
		SchemaVersion:         contracts.CapabilitiesSchemaVersion,
		Engine:                e.Name(),
		Version:               "v1",
		RequiresDocker:        false,
		RequiresXvfb:          false,
		MaxConcurrentSessions: driver.ResolveMaxConcurrentSessions(),
		AllowsParallelTabs:    true,
		SupportsHAR:           true,
		SupportsVideo:         true,
		SupportsIframes:       true,
		SupportsFileUploads:   true,
		SupportsDownloads:     true,
		SupportsTracing:       true,
		MaxViewportWidth:      1920,
		MaxViewportHeight:     1080,
	}, nil
}

// StartSession asks the driver to create a new browser/context/page tuple.
func (e *PlaywrightEngine) StartSession(ctx context.Context, spec SessionSpec) (EngineSession, error) {
	// Convert engine.SessionSpec to session.Spec
	sessionSpec := session.Spec{
		ExecutionID:    spec.ExecutionID,
		WorkflowID:     spec.WorkflowID,
		Mode:           session.ModeExecution,
		ViewportWidth:  spec.ViewportWidth,
		ViewportHeight: spec.ViewportHeight,
		ReuseMode:      string(spec.ReuseMode),
		BaseURL:        spec.BaseURL,
		Labels:         spec.Labels,
		Capabilities: session.CapabilityRequirement{
			NeedsParallelTabs: spec.Capabilities.NeedsParallelTabs,
			NeedsIframes:      spec.Capabilities.NeedsIframes,
			NeedsFileUploads:  spec.Capabilities.NeedsFileUploads,
			NeedsDownloads:    spec.Capabilities.NeedsDownloads,
			NeedsHAR:          spec.Capabilities.NeedsHAR,
			NeedsVideo:        spec.Capabilities.NeedsVideo,
			NeedsTracing:      spec.Capabilities.NeedsTracing,
			MinViewportWidth:  spec.Capabilities.MinViewportWidth,
			MinViewportHeight: spec.Capabilities.MinViewportHeight,
		},
	}

	// Add frame streaming config if enabled (for live execution preview)
	if spec.FrameStreaming != nil {
		sessionSpec.FrameStreaming = &session.FrameStreamingConfig{
			CallbackURL: spec.FrameStreaming.CallbackURL,
			Quality:     spec.FrameStreaming.Quality,
			FPS:         spec.FrameStreaming.FPS,
			Scale:       spec.FrameStreaming.Scale,
		}
	}

	logger := e.log
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	logger.WithFields(logrus.Fields{
		"execution_id":    spec.ExecutionID,
		"viewport_width":  spec.ViewportWidth,
		"viewport_height": spec.ViewportHeight,
	}).Info("Starting playwright session with viewport")

	// Create session via Manager - the returned *session.Session satisfies EngineSession
	sess, err := e.sessions.Create(ctx, sessionSpec)
	if err != nil {
		return nil, err
	}

	return sess, nil
}

// PlaywrightDriverError provides structured error information for driver issues.
// This is an alias for driver.Error for backward compatibility.
type PlaywrightDriverError = driver.Error

// resolveMaxConcurrentSessions aligns the engine capability with the driver pool limit.
// Falls back to the driver default (10) when no override is provided, and clamps to a safe range.
// Deprecated: Use driver.ResolveMaxConcurrentSessions() instead.
func resolveMaxConcurrentSessions() int {
	return driver.ResolveMaxConcurrentSessions()
}

// resolvePlaywrightDriverURL resolves the driver URL from environment.
// Deprecated: Use driver.NewClient() which handles URL resolution internally.
func resolvePlaywrightDriverURL(log *logrus.Logger, allowDefault bool) string {
	raw := strings.TrimSpace(os.Getenv("PLAYWRIGHT_DRIVER_URL"))
	if raw != "" {
		return raw
	}
	if allowDefault {
		if log != nil {
			log.WithField("default_url", driver.DefaultDriverURL).Warn("PLAYWRIGHT_DRIVER_URL not set; defaulting to local driver")
		}
		return driver.DefaultDriverURL
	}
	return ""
}

// Compile-time interface check.
var _ AutomationEngine = (*PlaywrightEngine)(nil)

// Legacy type definitions kept for backward compatibility with tests.

type startSessionRequest struct {
	ExecutionID uuid.UUID         `json:"execution_id"`
	WorkflowID  uuid.UUID         `json:"workflow_id"`
	Viewport    viewport          `json:"viewport"`
	ReuseMode   string            `json:"reuse_mode"`
	BaseURL     string            `json:"base_url,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`

	RequiredCapabilities capabilityRequest `json:"required_capabilities"`
}

type capabilityRequest struct {
	Tabs       bool `json:"tabs,omitempty"`
	Iframes    bool `json:"iframes,omitempty"`
	Uploads    bool `json:"uploads,omitempty"`
	Downloads  bool `json:"downloads,omitempty"`
	HAR        bool `json:"har,omitempty"`
	Video      bool `json:"video,omitempty"`
	Tracing    bool `json:"tracing,omitempty"`
	ViewportW  int  `json:"viewport_width,omitempty"`
	ViewportH  int  `json:"viewport_height,omitempty"`
	MaxTimeout int  `json:"max_timeout_ms,omitempty"`
}

type viewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type startSessionResponse struct {
	SessionID string `json:"session_id"`
}

type runRequest struct {
	Instruction contracts.CompiledInstruction `json:"instruction"`
}

type driverOutcome struct {
	contracts.StepOutcome

	ScreenshotBase64    string `json:"screenshot_base64,omitempty"`
	ScreenshotMediaType string `json:"screenshot_media_type,omitempty"`
	ScreenshotWidth     int    `json:"screenshot_width,omitempty"`
	ScreenshotHeight    int    `json:"screenshot_height,omitempty"`

	DOMHTML    string `json:"dom_html,omitempty"`
	DOMPreview string `json:"dom_preview,omitempty"`

	VideoPath string `json:"video_path,omitempty"`
	TracePath string `json:"trace_path,omitempty"`
}

// ResolveMaxConcurrentSessions returns the configured max concurrent sessions.
// This is exported for use by other packages.
func ResolveMaxConcurrentSessions() int {
	const (
		driverDefaultMaxSessions = 10
		maxSessionsCeiling       = 100
		minSessionsFloor         = 1
	)

	envVal := strings.TrimSpace(os.Getenv("MAX_SESSIONS"))
	if envVal != "" {
		if parsed, err := strconv.Atoi(envVal); err == nil && parsed >= minSessionsFloor && parsed <= maxSessionsCeiling {
			return parsed
		}
	}

	return driverDefaultMaxSessions
}
