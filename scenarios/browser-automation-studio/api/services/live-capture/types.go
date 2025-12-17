// Package livecapture provides live recording session management and related types.
package livecapture

import "github.com/vrooli/browser-automation-studio/automation/contracts"

// RecordedAction represents an action captured during recording.
// This mirrors the JSON structure sent from playwright-driver.
// This is the canonical Go type for recorded actions - use this type throughout
// the codebase instead of defining local copies.
type RecordedAction struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"sessionId"`
	SequenceNum int                    `json:"sequenceNum"`
	Timestamp   string                 `json:"timestamp"`
	DurationMs  int                    `json:"durationMs,omitempty"`
	ActionType  string                 `json:"actionType"`
	Confidence  float64                `json:"confidence"`
	Selector    *SelectorSet           `json:"selector,omitempty"`
	ElementMeta *ElementMeta           `json:"elementMeta,omitempty"`
	BoundingBox *contracts.BoundingBox `json:"boundingBox,omitempty"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	URL         string                 `json:"url"`
	FrameID     string                 `json:"frameId,omitempty"`
	CursorPos   *contracts.Point       `json:"cursorPos,omitempty"`
}

// SelectorSet contains multiple selector strategies for resilience.
type SelectorSet struct {
	Primary    string              `json:"primary"`
	Candidates []SelectorCandidate `json:"candidates"`
}

// SelectorCandidate is a single selector with metadata.
type SelectorCandidate struct {
	Type        string  `json:"type"`
	Value       string  `json:"value"`
	Confidence  float64 `json:"confidence"`
	Specificity int     `json:"specificity"`
}

// ElementMeta captures information about the target element.
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
