package engine

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// PlaywrightEngine talks to a local Playwright driver (Node) over HTTP/Unix
// sockets. The driver is responsible for translating CompiledInstructions into
// Playwright actions and returning contract StepOutcome payloads.
type PlaywrightEngine struct {
	driverURL  string
	httpClient HTTPDoer
	log        *logrus.Logger
}

// NewPlaywrightEngine constructs an engine using environment configuration.
// It requires PLAYWRIGHT_DRIVER_URL unless allowDefault is true, in which case
// it falls back to localhost for developer setups.
func NewPlaywrightEngine(log *logrus.Logger) (*PlaywrightEngine, error) {
	return newPlaywrightEngine(resolvePlaywrightDriverURL(log, false), log)
}

// NewPlaywrightEngineWithDefault allows the localhost fallback when explicit
// configuration is absent (useful for local development and tests).
func NewPlaywrightEngineWithDefault(log *logrus.Logger) (*PlaywrightEngine, error) {
	return newPlaywrightEngine(resolvePlaywrightDriverURL(log, true), log)
}

func newPlaywrightEngine(driverURL string, log *logrus.Logger) (*PlaywrightEngine, error) {
	if strings.TrimSpace(driverURL) == "" {
		return nil, fmt.Errorf("PLAYWRIGHT_DRIVER_URL is required")
	}
	// 5 minute timeout to allow for slow playwright operations (screenshots, network waiting, etc.)
	client := &http.Client{Timeout: 5 * time.Minute}
	return &PlaywrightEngine{
		driverURL:  strings.TrimRight(driverURL, "/"),
		httpClient: client,
		log:        log,
	}, nil
}

// NewPlaywrightEngineWithHTTPClient constructs an engine with a custom HTTP client.
// This is primarily used for testing to inject mock HTTP responses.
func NewPlaywrightEngineWithHTTPClient(driverURL string, httpClient HTTPDoer, log *logrus.Logger) (*PlaywrightEngine, error) {
	if strings.TrimSpace(driverURL) == "" {
		return nil, fmt.Errorf("PLAYWRIGHT_DRIVER_URL is required")
	}
	if httpClient == nil {
		return nil, fmt.Errorf("httpClient is required")
	}
	return &PlaywrightEngine{
		driverURL:  strings.TrimRight(driverURL, "/"),
		httpClient: httpClient,
		log:        log,
	}, nil
}

// Name returns the engine identifier.
func (e *PlaywrightEngine) Name() string { return "playwright" }

// Capabilities returns a conservative capability descriptor for the local
// Playwright driver. Update when driver gains richer support (HAR/video, etc.).
func (e *PlaywrightEngine) Capabilities(_ context.Context) (contracts.EngineCapabilities, error) {
	if err := e.health(context.Background()); err != nil {
		return contracts.EngineCapabilities{}, err
	}
	return contracts.EngineCapabilities{
		SchemaVersion:         contracts.CapabilitiesSchemaVersion,
		Engine:                e.Name(),
		Version:               "v1",
		RequiresDocker:        false,
		RequiresXvfb:          false,
		MaxConcurrentSessions: 2,
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

// health ensures the driver is reachable.
func (e *PlaywrightEngine) health(ctx context.Context) error {
	if e == nil || e.httpClient == nil {
		return &PlaywrightDriverError{
			Op:      "health",
			URL:     "",
			Message: "playwright engine not configured",
			Hint:    "ensure NewPlaywrightEngine() was called successfully",
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.driverURL+"/health", http.NoBody)
	if err != nil {
		return &PlaywrightDriverError{
			Op:      "health",
			URL:     e.driverURL,
			Message: "failed to create health check request",
			Cause:   err,
		}
	}
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return &PlaywrightDriverError{
			Op:      "health",
			URL:     e.driverURL,
			Message: "playwright driver is not responding",
			Cause:   err,
			Hint:    "ensure playwright-driver is running (check 'make start' or lifecycle status)",
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return &PlaywrightDriverError{
			Op:      "health",
			URL:     e.driverURL,
			Message: fmt.Sprintf("playwright driver returned unhealthy status %d: %s", resp.StatusCode, strings.TrimSpace(string(body))),
			Hint:    "check playwright-driver logs for errors",
		}
	}
	return nil
}

// PlaywrightDriverError provides structured error information for driver issues.
// This helps users troubleshoot common configuration and connectivity problems.
type PlaywrightDriverError struct {
	Op      string // operation that failed (health, start_session, run, etc.)
	URL     string // driver URL that was contacted
	Message string // human-readable error message
	Cause   error  // underlying error if any
	Hint    string // troubleshooting suggestion
}

func (e *PlaywrightDriverError) Error() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("playwright driver %s failed", e.Op))
	if e.URL != "" {
		parts = append(parts, fmt.Sprintf("at %s", e.URL))
	}
	parts = append(parts, fmt.Sprintf(": %s", e.Message))
	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf(" (%v)", e.Cause))
	}
	if e.Hint != "" {
		parts = append(parts, fmt.Sprintf(" [hint: %s]", e.Hint))
	}
	return strings.Join(parts, "")
}

func (e *PlaywrightDriverError) Unwrap() error {
	return e.Cause
}

// StartSession asks the driver to create a new browser/context/page tuple.
func (e *PlaywrightEngine) StartSession(ctx context.Context, spec SessionSpec) (EngineSession, error) {
	req := startSessionRequest{
		ExecutionID: spec.ExecutionID,
		WorkflowID:  spec.WorkflowID,
		Viewport: viewport{
			Width:  spec.ViewportWidth,
			Height: spec.ViewportHeight,
		},
		ReuseMode: string(spec.ReuseMode),
		BaseURL:   spec.BaseURL,
		Labels:    spec.Labels,
		RequiredCapabilities: capabilityRequest{
			Tabs:      spec.Capabilities.NeedsParallelTabs,
			Iframes:   spec.Capabilities.NeedsIframes,
			Uploads:   spec.Capabilities.NeedsFileUploads,
			Downloads: spec.Capabilities.NeedsDownloads,
			HAR:       spec.Capabilities.NeedsHAR,
			Video:     spec.Capabilities.NeedsVideo,
			ViewportW: spec.Capabilities.MinViewportWidth,
			ViewportH: spec.Capabilities.MinViewportHeight,
			Tracing:   spec.Capabilities.NeedsTracing,
		},
	}

	e.log.WithFields(logrus.Fields{
		"execution_id":    spec.ExecutionID,
		"viewport_width":  spec.ViewportWidth,
		"viewport_height": spec.ViewportHeight,
	}).Info("Starting playwright session with viewport")

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return nil, fmt.Errorf("encode start session request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, e.driverURL+"/session/start", &buf)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, &PlaywrightDriverError{
			Op:      "start_session",
			URL:     e.driverURL,
			Message: "failed to connect to playwright driver",
			Cause:   err,
			Hint:    "verify playwright-driver is running and PLAYWRIGHT_DRIVER_URL is correct",
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		bodyStr := strings.TrimSpace(string(body))
		hint := "check playwright-driver logs for details"
		if strings.Contains(bodyStr, "Maximum concurrent sessions") {
			hint = "too many concurrent sessions - wait for other executions to complete or increase session limit"
		} else if strings.Contains(bodyStr, "browser") && strings.Contains(bodyStr, "launch") {
			hint = "browser failed to launch - check chromium installation and system resources"
		}
		return nil, &PlaywrightDriverError{
			Op:      "start_session",
			URL:     e.driverURL,
			Message: fmt.Sprintf("driver returned status %d: %s", resp.StatusCode, bodyStr),
			Hint:    hint,
		}
	}
	var parsed startSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, &PlaywrightDriverError{
			Op:      "start_session",
			URL:     e.driverURL,
			Message: "invalid response from driver (JSON decode failed)",
			Cause:   err,
			Hint:    "driver may have returned HTML error page instead of JSON",
		}
	}
	if strings.TrimSpace(parsed.SessionID) == "" {
		return nil, &PlaywrightDriverError{
			Op:      "start_session",
			URL:     e.driverURL,
			Message: "driver returned empty session_id",
			Hint:    "driver may be overloaded or encountered an internal error",
		}
	}
	return &playwrightSession{
		sessionID: parsed.SessionID,
		engine:    e,
	}, nil
}

type playwrightSession struct {
	sessionID string
	engine    *PlaywrightEngine
}

func (s *playwrightSession) Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	req := runRequest{Instruction: instruction}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return contracts.StepOutcome{}, fmt.Errorf("encode run request: %w", err)
	}

	url := fmt.Sprintf("%s/session/%s/run", s.engine.driverURL, url.PathEscape(s.sessionID))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return contracts.StepOutcome{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.engine.httpClient.Do(httpReq)
	if err != nil {
		return contracts.StepOutcome{}, fmt.Errorf("run instruction: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return contracts.StepOutcome{}, fmt.Errorf("run instruction failed: %s", strings.TrimSpace(string(body)))
	}
	return decodeDriverOutcome(resp.Body)
}

func (s *playwrightSession) Reset(ctx context.Context) error {
	url := fmt.Sprintf("%s/session/%s/reset", s.engine.driverURL, url.PathEscape(s.sessionID))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := s.engine.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("reset failed: %s", strings.TrimSpace(string(body)))
	}
	return nil
}

func (s *playwrightSession) Close(ctx context.Context) error {
	url := fmt.Sprintf("%s/session/%s/close", s.engine.driverURL, url.PathEscape(s.sessionID))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := s.engine.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("close failed: %s", strings.TrimSpace(string(body)))
	}
	return nil
}

// decodeDriverOutcome converts the driver response into a StepOutcome,
// decoding inline base64 screenshot/DOM payloads.
func decodeDriverOutcome(r io.Reader) (contracts.StepOutcome, error) {
	var resp driverOutcome
	if err := json.NewDecoder(r).Decode(&resp); err != nil {
		return contracts.StepOutcome{}, fmt.Errorf("decode driver outcome: %w", err)
	}

	out := resp.StepOutcome
	out.SchemaVersion = contracts.StepOutcomeSchemaVersion
	out.PayloadVersion = contracts.PayloadVersion
	if resp.ScreenshotBase64 != "" {
		data, err := base64.StdEncoding.DecodeString(resp.ScreenshotBase64)
		if err != nil {
			return contracts.StepOutcome{}, fmt.Errorf("decode screenshot: %w", err)
		}
		out.Screenshot = &contracts.Screenshot{
			Data:        data,
			MediaType:   resp.ScreenshotMediaType,
			CaptureTime: time.Now().UTC(),
			Width:       resp.ScreenshotWidth,
			Height:      resp.ScreenshotHeight,
		}
	}

	if resp.DOMHTML != "" {
		out.DOMSnapshot = &contracts.DOMSnapshot{
			HTML:        resp.DOMHTML,
			Preview:     resp.DOMPreview,
			CollectedAt: time.Now().UTC(),
		}
	}

	if resp.VideoPath != "" || resp.TracePath != "" {
		if out.Notes == nil {
			out.Notes = map[string]string{}
		}
		if resp.VideoPath != "" {
			out.Notes["video_path"] = resp.VideoPath
		}
		if resp.TracePath != "" {
			out.Notes["trace_path"] = resp.TracePath
		}
	}

	return out, nil
}

func resolvePlaywrightDriverURL(log *logrus.Logger, allowDefault bool) string {
	raw := strings.TrimSpace(os.Getenv("PLAYWRIGHT_DRIVER_URL"))
	if raw != "" {
		return raw
	}
	if allowDefault {
		if log != nil {
			log.WithField("default_url", "http://127.0.0.1:39400").Warn("PLAYWRIGHT_DRIVER_URL not set; defaulting to local driver")
		}
		return "http://127.0.0.1:39400"
	}
	return ""
}

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
