// Package playwright provides a Driver implementation that wraps the Playwright-based
// browser automation backend. It bridges the unified Driver interface to the existing
// HTTP client that communicates with playwright-driver.
//
// DOC: docs/architecture/driver-interface.md#playwright-driver
package playwright

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
)

// Driver implements the driver.Driver interface using the Playwright backend.
// It wraps the existing driver.Client and provides session management with
// recording callback injection.
type Driver struct {
	client   *driver.Client
	log      *logrus.Logger
	sessions map[string]*session
	mu       sync.RWMutex
}

// Option configures a Driver.
type Option func(*Driver)

// WithLogger sets a custom logger.
func WithLogger(log *logrus.Logger) Option {
	return func(d *Driver) {
		d.log = log
	}
}

// WithClient sets a custom client (for testing).
func WithClient(client *driver.Client) Option {
	return func(d *Driver) {
		d.client = client
	}
}

// NewDriver creates a new Playwright driver.
func NewDriver(opts ...Option) (*Driver, error) {
	d := &Driver{
		log:      logrus.StandardLogger(),
		sessions: make(map[string]*session),
	}

	for _, opt := range opts {
		opt(d)
	}

	// Create client if not provided
	if d.client == nil {
		client, err := driver.NewClient(driver.WithLogger(d.log))
		if err != nil {
			return nil, fmt.Errorf("create playwright client: %w", err)
		}
		d.client = client
	}

	return d, nil
}

// CreateSession creates a new browser session.
func (d *Driver) CreateSession(ctx context.Context, spec driver.SessionSpec) (driver.Session, error) {
	// Build request from spec
	req := &driver.CreateSessionRequest{
		ExecutionID: spec.ExecutionID.String(),
		WorkflowID:  spec.ExecutionID.String(), // Use execution ID as workflow ID for recording
		Viewport: driver.Viewport{
			Width:  spec.Viewport.Width,
			Height: spec.Viewport.Height,
		},
		ReuseMode:      "fresh", // Recording sessions always use fresh mode
		Labels:         spec.Labels,
		BrowserProfile: spec.BrowserProfile,
	}

	// Add storage state if provided
	if len(spec.StorageState) > 0 {
		req.StorageState = spec.StorageState
	}

	// Create session via client
	resp, err := d.client.CreateSession(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Create session wrapper
	sess := &session{
		id:        resp.SessionID,
		client:    d.client,
		log:       d.log,
		recording: spec.Recording,
	}

	// Store session
	d.mu.Lock()
	d.sessions[sess.id] = sess
	d.mu.Unlock()

	d.log.WithFields(logrus.Fields{
		"session_id":    sess.id,
		"execution_id":  spec.ExecutionID,
		"has_recording": spec.Recording != nil,
	}).Debug("Created playwright session")

	return sess, nil
}

// CloseSession closes a browser session by ID.
func (d *Driver) CloseSession(ctx context.Context, sessionID string) error {
	d.mu.Lock()
	sess, ok := d.sessions[sessionID]
	if ok {
		delete(d.sessions, sessionID)
	}
	d.mu.Unlock()

	if !ok {
		return driver.ErrSessionNotFound
	}

	return sess.Close(ctx)
}

// Health checks if the Playwright driver backend is available.
func (d *Driver) Health(ctx context.Context) error {
	return d.client.Health(ctx)
}

// Type returns the driver type identifier.
func (d *Driver) Type() driver.DriverType {
	return driver.DriverTypePlaywright
}

// Client returns the underlying client for direct access (migration helper).
func (d *Driver) Client() *driver.Client {
	return d.client
}

// session implements driver.Session for Playwright.
type session struct {
	id        string
	client    *driver.Client
	log       *logrus.Logger
	recording *driver.RecordingConfig
	mu        sync.RWMutex
	closed    bool
}

// ID returns the session ID.
func (s *session) ID() string {
	return s.id
}

// Navigate navigates to a URL.
func (s *session) Navigate(ctx context.Context, url string, opts *driver.NavigateOptions) (*driver.NavigateResult, error) {
	if s.isClosed() {
		return nil, driver.ErrSessionClosed
	}

	req := &driver.NavigateRequest{URL: url}
	if opts != nil {
		req.WaitUntil = opts.WaitUntil
		if opts.Timeout > 0 {
			req.TimeoutMs = int(opts.Timeout.Milliseconds())
		}
	}

	resp, err := s.client.Navigate(ctx, s.id, req)
	if err != nil {
		return nil, err
	}

	// Report action to recording callback if configured
	if s.recording != nil && s.recording.ActionCallback != nil {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			ActionType: "navigate",
			URL:        resp.URL,
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Confidence: 1.0,
			PageTitle:  resp.Title,
		}
		// Use nil page ID for now - caller should track active page
		s.recording.ActionCallback(action, uuid.Nil)
	}

	return &driver.NavigateResult{
		URL:        resp.URL,
		Title:      resp.Title,
		StatusCode: resp.StatusCode,
	}, nil
}

// Click clicks on an element.
func (s *session) Click(ctx context.Context, selector string, opts *driver.ClickOptions) error {
	if s.isClosed() {
		return driver.ErrSessionClosed
	}

	// Build instruction payload for click
	instruction := map[string]interface{}{
		"type": "click",
		"params": map[string]interface{}{
			"selector": selector,
		},
	}

	if opts != nil {
		params := instruction["params"].(map[string]interface{})
		if opts.Button != "" {
			params["button"] = opts.Button
		}
		if opts.ClickCount > 0 {
			params["clickCount"] = opts.ClickCount
		}
		if opts.Delay > 0 {
			params["delay"] = opts.Delay
		}
		if opts.Force {
			params["force"] = opts.Force
		}
		if opts.Position != nil {
			params["position"] = map[string]float64{
				"x": opts.Position.X,
				"y": opts.Position.Y,
			}
		}
	}

	// Run via instructions API
	if err := s.client.RunInstructions(ctx, s.id, []map[string]interface{}{instruction}); err != nil {
		return err
	}

	// Report action to recording callback if configured
	if s.recording != nil && s.recording.ActionCallback != nil {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			ActionType: "click",
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Confidence: 1.0,
			Selector: &driver.SelectorSet{
				Primary: selector,
			},
		}
		s.recording.ActionCallback(action, uuid.Nil)
	}

	return nil
}

// Type types text into an element.
func (s *session) Type(ctx context.Context, selector string, text string, opts *driver.TypeOptions) error {
	if s.isClosed() {
		return driver.ErrSessionClosed
	}

	// Build instruction payload for type
	instruction := map[string]interface{}{
		"type": "type",
		"params": map[string]interface{}{
			"selector": selector,
			"text":     text,
		},
	}

	if opts != nil {
		params := instruction["params"].(map[string]interface{})
		if opts.Delay > 0 {
			params["delay"] = opts.Delay
		}
		if opts.Clear {
			params["clear"] = opts.Clear
		}
	}

	if err := s.client.RunInstructions(ctx, s.id, []map[string]interface{}{instruction}); err != nil {
		return err
	}

	// Report action to recording callback if configured
	if s.recording != nil && s.recording.ActionCallback != nil {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			ActionType: "type",
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Confidence: 1.0,
			Selector: &driver.SelectorSet{
				Primary: selector,
			},
			Payload: map[string]interface{}{
				"text": text,
			},
		}
		s.recording.ActionCallback(action, uuid.Nil)
	}

	return nil
}

// Hover hovers over an element.
func (s *session) Hover(ctx context.Context, selector string, opts *driver.HoverOptions) error {
	if s.isClosed() {
		return driver.ErrSessionClosed
	}

	instruction := map[string]interface{}{
		"type": "hover",
		"params": map[string]interface{}{
			"selector": selector,
		},
	}

	if opts != nil {
		params := instruction["params"].(map[string]interface{})
		if opts.Force {
			params["force"] = opts.Force
		}
		if opts.Position != nil {
			params["position"] = map[string]float64{
				"x": opts.Position.X,
				"y": opts.Position.Y,
			}
		}
	}

	if err := s.client.RunInstructions(ctx, s.id, []map[string]interface{}{instruction}); err != nil {
		return err
	}

	if s.recording != nil && s.recording.ActionCallback != nil {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			ActionType: "hover",
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Confidence: 1.0,
			Selector: &driver.SelectorSet{
				Primary: selector,
			},
		}
		s.recording.ActionCallback(action, uuid.Nil)
	}

	return nil
}

// WaitForSelector waits for an element to appear.
func (s *session) WaitForSelector(ctx context.Context, selector string, opts *driver.WaitOptions) error {
	if s.isClosed() {
		return driver.ErrSessionClosed
	}

	instruction := map[string]interface{}{
		"type": "wait",
		"params": map[string]interface{}{
			"selector": selector,
		},
	}

	if opts != nil {
		params := instruction["params"].(map[string]interface{})
		if opts.State != "" {
			params["state"] = opts.State
		}
		if opts.Timeout > 0 {
			params["timeout"] = int(opts.Timeout.Milliseconds())
		}
	}

	return s.client.RunInstructions(ctx, s.id, []map[string]interface{}{instruction})
}

// Screenshot captures a screenshot.
func (s *session) Screenshot(ctx context.Context, opts *driver.ScreenshotOptions) (*driver.ScreenshotResult, error) {
	if s.isClosed() {
		return nil, driver.ErrSessionClosed
	}

	req := &driver.CaptureScreenshotRequest{
		Format:  "jpeg",
		Quality: 85,
	}

	if opts != nil {
		if opts.Type != "" {
			req.Format = opts.Type
		}
		if opts.Quality > 0 {
			req.Quality = opts.Quality
		}
	}

	resp, err := s.client.CaptureScreenshot(ctx, s.id, req)
	if err != nil {
		return nil, err
	}

	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("decode screenshot: %w", err)
	}

	return &driver.ScreenshotResult{
		Data:      data,
		MediaType: resp.MediaType,
		Width:     resp.Width,
		Height:    resp.Height,
	}, nil
}

// Evaluate evaluates JavaScript.
func (s *session) Evaluate(ctx context.Context, expression string, opts *driver.EvaluateOptions) (interface{}, error) {
	if s.isClosed() {
		return nil, driver.ErrSessionClosed
	}

	instruction := map[string]interface{}{
		"type": "evaluate",
		"params": map[string]interface{}{
			"expression": expression,
		},
	}

	if opts != nil && len(opts.Args) > 0 {
		params := instruction["params"].(map[string]interface{})
		params["args"] = opts.Args
	}

	// For evaluate, we need the result - RunInstructions doesn't return results
	// This is a limitation that would need the driver API to be extended
	// For now, just run it without capturing result
	if err := s.client.RunInstructions(ctx, s.id, []map[string]interface{}{instruction}); err != nil {
		return nil, err
	}

	return nil, nil
}

// GetURL returns the current page URL.
func (s *session) GetURL(ctx context.Context) (string, error) {
	if s.isClosed() {
		return "", driver.ErrSessionClosed
	}

	resp, err := s.client.GetNavigationState(ctx, s.id)
	if err != nil {
		return "", err
	}

	return resp.URL, nil
}

// GetTitle returns the current page title.
func (s *session) GetTitle(ctx context.Context) (string, error) {
	if s.isClosed() {
		return "", driver.ErrSessionClosed
	}

	resp, err := s.client.GetNavigationState(ctx, s.id)
	if err != nil {
		return "", err
	}

	return resp.Title, nil
}

// Pages returns the page tracker (not implemented for basic Playwright driver).
func (s *session) Pages() driver.PageTracker {
	// Page tracking is managed at a higher level in the session manager
	return nil
}

// Close closes the session.
func (s *session) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	_, err := s.client.CloseSession(ctx, s.id)
	return err
}

func (s *session) isClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// Compile-time interface compliance
var (
	_ driver.Driver  = (*Driver)(nil)
	_ driver.Session = (*session)(nil)
)
