// Package domain provides core domain types for browser-automation-studio.
// DOC: docs/architecture/recording.md#recording-action
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ActionSource represents the provenance of a recorded action.
// DOC: docs/architecture/recording.md#action-source
type ActionSource string

const (
	// ActionSourceAuto indicates the action was captured automatically by the driver.
	ActionSourceAuto ActionSource = "auto"
	// ActionSourceManual indicates the action was manually entered by a user.
	ActionSourceManual ActionSource = "manual"
	// ActionSourceAISuggested indicates the action was suggested by an AI assistant.
	ActionSourceAISuggested ActionSource = "ai_suggested"
)

// RecordingAction represents a user action captured during a recording session.
// This is the domain entity for persisted actions, separate from the driver's
// wire format (driver.RecordedAction) and the timeline view (RecordedActionEntry).
//
// DOC: docs/architecture/recording.md#recording-action
type RecordingAction struct {
	// ID uniquely identifies this action within the system.
	ID uuid.UUID `json:"id"`

	// SessionID links this action to its recording session.
	SessionID string `json:"sessionId"`

	// PageID identifies which browser page/tab this action occurred on.
	PageID uuid.UUID `json:"pageId"`

	// SequenceNum orders actions within a session (1-indexed).
	SequenceNum int `json:"sequenceNum"`

	// ActionType identifies the type of action (click, type, navigate, etc.).
	ActionType string `json:"actionType"`

	// Timestamp is when the action occurred.
	Timestamp time.Time `json:"timestamp"`

	// DurationMs is how long the action took to complete.
	DurationMs int `json:"durationMs,omitempty"`

	// Selector contains the selector strategies used to locate the target element.
	Selector *SelectorSet `json:"selector,omitempty"`

	// ElementMeta captures metadata about the target element at action time.
	ElementMeta *ElementMeta `json:"elementMeta,omitempty"`

	// BoundingBox captures the position and dimensions of the target element.
	BoundingBox *BoundingBox `json:"boundingBox,omitempty"`

	// Payload contains action-specific data (e.g., text for typing, key for keypress).
	Payload map[string]interface{} `json:"payload,omitempty"`

	// URL is the page URL at the time of the action.
	URL string `json:"url,omitempty"`

	// PageTitle is the page title at the time of the action.
	PageTitle string `json:"pageTitle,omitempty"`

	// Confidence indicates selector reliability (1.0 = high confidence).
	Confidence float64 `json:"confidence"`

	// Source indicates how this action was captured.
	Source ActionSource `json:"source"`

	// CreatedAt is when this record was persisted to the database.
	CreatedAt time.Time `json:"createdAt"`
}

// SelectorSet contains multiple selector strategies for resilient element targeting.
// The primary selector is tried first, with candidates providing fallbacks.
type SelectorSet struct {
	Primary    string              `json:"primary"`
	Candidates []SelectorCandidate `json:"candidates,omitempty"`
}

// SelectorCandidate represents a single selector strategy with metadata.
type SelectorCandidate struct {
	Type        string  `json:"type"`        // e.g., "css", "xpath", "text", "data-testid"
	Value       string  `json:"value"`       // The actual selector string
	Confidence  float64 `json:"confidence"`  // Selector reliability score
	Specificity int     `json:"specificity"` // CSS specificity score
}

// ElementMeta captures information about the target element at action time.
// This metadata aids debugging and selector improvement.
type ElementMeta struct {
	TagName    string            `json:"tagName"`
	ID         string            `json:"id,omitempty"`
	ClassName  string            `json:"className,omitempty"`
	InnerText  string            `json:"innerText,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	IsVisible  bool              `json:"isVisible"`
	IsEnabled  bool              `json:"isEnabled"`
	Role       string            `json:"role,omitempty"`
	AriaLabel  string            `json:"ariaLabel,omitempty"`
}

// BoundingBox captures the position and dimensions of an element in viewport coordinates.
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
