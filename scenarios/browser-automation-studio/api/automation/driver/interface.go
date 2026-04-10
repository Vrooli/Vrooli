// Package driver provides browser automation driver interfaces and implementations.
// This file defines the unified Driver interface that abstracts different browser
// automation backends (Playwright, ClaudeCode CLI, etc.) for the recording system.
//
// The key architectural insight is that recording callbacks are injected at the
// driver level during session creation, ensuring all browser actions (manual,
// AI-initiated, or programmatic) flow through the same recording pipeline.
//
// DOC: docs/architecture/driver-interface.md
package driver

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

// DriverType identifies the browser automation backend.
type DriverType string

const (
	// DriverTypePlaywright uses the Playwright-based browser driver.
	DriverTypePlaywright DriverType = "playwright"
	// DriverTypeClaudeCode uses the Claude Code CLI for AI-driven navigation.
	DriverTypeClaudeCode DriverType = "claudecode"
)

// ErrNotImplemented is returned when a driver operation is not yet implemented.
var ErrNotImplemented = errors.New("driver operation not implemented")

// ErrSessionNotFound is returned when a session ID doesn't exist.
var ErrSessionNotFound = errors.New("session not found")

// ErrSessionClosed is returned when operating on a closed session.
var ErrSessionClosed = errors.New("session closed")

// Driver provides browser automation capabilities.
// Implementations manage session lifecycle and delegate browser operations
// to their respective backends (Playwright, Claude CLI, etc.).
//
// All drivers must ensure that recording callbacks (when configured) are
// invoked for every browser action, regardless of how the action was initiated.
type Driver interface {
	// CreateSession creates a new browser session with the given specification.
	// Recording callbacks in the spec are injected to capture all browser actions.
	CreateSession(ctx context.Context, spec SessionSpec) (Session, error)

	// CloseSession closes a browser session by ID.
	CloseSession(ctx context.Context, sessionID string) error

	// Health checks if the driver backend is available and healthy.
	Health(ctx context.Context) error

	// Type returns the driver type identifier.
	Type() DriverType
}

// Session represents an active browser session.
// It provides browser automation operations that work regardless of the
// underlying driver implementation.
type Session interface {
	// ID returns the unique session identifier.
	ID() string

	// Navigate navigates the active page to a URL.
	Navigate(ctx context.Context, url string, opts *NavigateOptions) (*NavigateResult, error)

	// Click clicks on an element matching the selector.
	Click(ctx context.Context, selector string, opts *ClickOptions) error

	// Type types text into an element matching the selector.
	Type(ctx context.Context, selector string, text string, opts *TypeOptions) error

	// Hover hovers over an element matching the selector.
	Hover(ctx context.Context, selector string, opts *HoverOptions) error

	// WaitForSelector waits for an element matching the selector to appear.
	WaitForSelector(ctx context.Context, selector string, opts *WaitOptions) error

	// Screenshot captures a screenshot of the current page.
	Screenshot(ctx context.Context, opts *ScreenshotOptions) (*ScreenshotResult, error)

	// Evaluate evaluates JavaScript in the page context.
	Evaluate(ctx context.Context, expression string, opts *EvaluateOptions) (interface{}, error)

	// GetURL returns the current page URL.
	GetURL(ctx context.Context) (string, error)

	// GetTitle returns the current page title.
	GetTitle(ctx context.Context) (string, error)

	// Pages returns the page tracker for multi-tab sessions.
	// May return nil if the driver doesn't support multi-tab tracking.
	Pages() PageTracker

	// Close closes the session and releases resources.
	Close(ctx context.Context) error
}

// SessionSpec configures a new browser session.
type SessionSpec struct {
	// ExecutionID uniquely identifies this execution/session for tracking.
	ExecutionID uuid.UUID

	// Viewport specifies the browser viewport dimensions.
	Viewport ViewportSpec

	// StorageState contains browser storage state (cookies, localStorage) as JSON.
	// Used to restore authentication state from previous sessions.
	StorageState json.RawMessage

	// Recording configures action recording callbacks.
	// When set, all browser actions will be reported through these callbacks.
	Recording *RecordingConfig

	// BrowserProfile configures anti-detection and human-like behavior settings.
	BrowserProfile *sessionprofilepersistence.BrowserProfile

	// Labels contains arbitrary metadata for the session.
	Labels map[string]string
}

// ViewportSpec defines browser viewport dimensions.
type ViewportSpec struct {
	Width  int
	Height int
}

// RecordingConfig configures recording callbacks for a session.
// These callbacks are invoked by the driver for every browser action,
// enabling unified recording regardless of action source (manual, AI, programmatic).
type RecordingConfig struct {
	// ActionCallback is called when a browser action is performed.
	// The callback receives the action details and the page ID where it occurred.
	ActionCallback func(action *RecordedAction, pageID uuid.UUID)

	// PageEventCallback is called when a page lifecycle event occurs
	// (page created, navigated, closed).
	PageEventCallback func(event *PageEvent)

	// FrameCallback is called when a new frame is captured for live preview.
	FrameCallback func(frame *Frame)
}

// PageEvent represents a page lifecycle event.
type PageEvent struct {
	ID        uuid.UUID
	Type      PageEventType
	PageID    uuid.UUID
	URL       string
	Title     string
	OpenerID  *uuid.UUID
	Timestamp time.Time
}

// PageEventType identifies the type of page lifecycle event.
type PageEventType string

const (
	PageEventCreated   PageEventType = "page_created"
	PageEventNavigated PageEventType = "page_navigated"
	PageEventClosed    PageEventType = "page_closed"
)

// Frame represents a captured video frame for live preview.
type Frame struct {
	Data        []byte
	MediaType   string
	Width       int
	Height      int
	CapturedAt  time.Time
	ContentHash string
}

// PageTracker manages pages (tabs) within a session.
type PageTracker interface {
	// ListPages returns all pages in the session.
	ListPages() []PageInfo

	// GetActivePage returns the currently active page.
	GetActivePage() *PageInfo

	// SetActivePage switches the active page.
	SetActivePage(pageID uuid.UUID) error
}

// PageInfo contains information about a browser page.
type PageInfo struct {
	ID        uuid.UUID
	URL       string
	Title     string
	IsActive  bool
	CreatedAt time.Time
}

// NavigateOptions configures navigation behavior.
type NavigateOptions struct {
	// WaitUntil specifies when navigation is considered complete.
	// Values: "load", "domcontentloaded", "networkidle"
	WaitUntil string

	// Timeout is the maximum time to wait for navigation.
	Timeout time.Duration

	// Referer sets the referer header for the navigation.
	Referer string
}

// NavigateResult contains the result of a navigation.
type NavigateResult struct {
	URL        string
	Title      string
	StatusCode int
}

// ClickOptions configures click behavior.
type ClickOptions struct {
	// Button specifies which mouse button to use.
	// Values: "left", "right", "middle"
	Button string

	// ClickCount specifies the number of clicks (1 for single, 2 for double).
	ClickCount int

	// Delay specifies time between mousedown and mouseup in milliseconds.
	Delay int

	// Force skips actionability checks if true.
	Force bool

	// Timeout is the maximum time to wait for the element.
	Timeout time.Duration

	// Position specifies the point to click relative to the element's top-left corner.
	Position *Point
}

// Point represents a 2D coordinate.
type Point struct {
	X float64
	Y float64
}

// TypeOptions configures typing behavior.
type TypeOptions struct {
	// Delay specifies time between key presses in milliseconds.
	Delay int

	// Clear clears the input before typing if true.
	Clear bool

	// Timeout is the maximum time to wait for the element.
	Timeout time.Duration
}

// HoverOptions configures hover behavior.
type HoverOptions struct {
	// Force skips actionability checks if true.
	Force bool

	// Timeout is the maximum time to wait for the element.
	Timeout time.Duration

	// Position specifies the point to hover relative to the element's top-left corner.
	Position *Point
}

// WaitOptions configures wait behavior.
type WaitOptions struct {
	// State specifies the element state to wait for.
	// Values: "attached", "detached", "visible", "hidden"
	State string

	// Timeout is the maximum time to wait.
	Timeout time.Duration
}

// ScreenshotOptions configures screenshot capture.
type ScreenshotOptions struct {
	// FullPage captures the entire scrollable page if true.
	FullPage bool

	// Quality is the image quality (1-100) for JPEG format.
	Quality int

	// Type specifies the image format.
	// Values: "png", "jpeg"
	Type string
}

// ScreenshotResult contains the captured screenshot.
type ScreenshotResult struct {
	Data      []byte
	MediaType string
	Width     int
	Height    int
}

// EvaluateOptions configures JavaScript evaluation.
type EvaluateOptions struct {
	// Args are arguments to pass to the function.
	Args []interface{}
}

// ActionSource indicates how an action was initiated.
type ActionSource string

const (
	// ActionSourceManual indicates user-initiated action (mouse/keyboard).
	ActionSourceManual ActionSource = "manual"
	// ActionSourceAI indicates AI-suggested or AI-executed action.
	ActionSourceAI ActionSource = "ai"
	// ActionSourcePlayback indicates action from workflow playback.
	ActionSourcePlayback ActionSource = "playback"
	// ActionSourceAuto is the default for driver-captured actions.
	ActionSourceAuto ActionSource = "auto"
)
