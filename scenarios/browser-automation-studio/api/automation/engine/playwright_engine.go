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
	httpClient *http.Client
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
	client := &http.Client{Timeout: 2 * time.Minute}
	return &PlaywrightEngine{
		driverURL:  strings.TrimRight(driverURL, "/"),
		httpClient: client,
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
		return fmt.Errorf("playwright engine not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.driverURL+"/health", http.NoBody)
	if err != nil {
		return err
	}
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("playwright driver health check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("playwright driver unhealthy: %s", strings.TrimSpace(string(body)))
	}
	return nil
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
		return nil, fmt.Errorf("start session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("start session failed: %s", strings.TrimSpace(string(body)))
	}
	var parsed startSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode start session response: %w", err)
	}
	if strings.TrimSpace(parsed.SessionID) == "" {
		return nil, fmt.Errorf("start session failed: missing session_id")
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
