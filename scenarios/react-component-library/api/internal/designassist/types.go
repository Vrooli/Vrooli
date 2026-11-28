// Package designassist turns a natural-language request into a validated,
// inert patch for one design document.
//
// It does not talk to a model. `designinference` owns dispatch and the durable
// idempotency record; this package owns what the studio is allowed to apply:
// the bounded context, the patch schema, the instruction, and the semantic
// validation against the document the caller loaded. A proposal is evidence,
// not an edit: the studio applies it as one undoable command and can discard
// it without touching storage.
package designassist

import "errors"

// ErrNotConfigured reports a handler without an inference runner.
var ErrNotConfigured = errors.New("design assistance is not configured")

// ErrInvalid reports a request that cannot produce a proposal.
var ErrInvalid = errors.New("invalid assistance request")

// ScopeKind names what the person selected before asking for a patch.
type ScopeKind string

const (
	ScopeDesign ScopeKind = "design"
	ScopeFrame  ScopeKind = "frame"
	ScopeLayer  ScopeKind = "layer"
)

// Scope is the part of the document the proposal may touch.
type Scope struct {
	Kind    ScopeKind `json:"kind"`
	FrameID string    `json:"frameId,omitempty"`
	NodeID  string    `json:"nodeId,omitempty"`
}

// Request is the validated HTTP body.
type Request struct {
	Prompt         string `json:"prompt"`
	Scope          Scope  `json:"scope"`
	Revision       int    `json:"revision"`
	IdempotencyKey string `json:"idempotencyKey"`
}

// Operation is one patch step. Fields are flat because the model tolerates a
// flat shape far better than a discriminated union; semantic validation names
// the missing field for its kind.
type Operation struct {
	Kind     string   `json:"kind"`
	NodeID   string   `json:"nodeId,omitempty"`
	ParentID string   `json:"parentId,omitempty"`
	Key      string   `json:"key,omitempty"`
	Value    any      `json:"value,omitempty"`
	Text     *string  `json:"text,omitempty"`
	Node     *NewNode `json:"node,omitempty"`
	Index    *int     `json:"index,omitempty"`
}

// NewNode is a layer the model wants to add. It never carries an id: the
// client assigns one, so the model cannot collide with an existing node.
type NewNode struct {
	Type     string         `json:"type"`
	Props    map[string]any `json:"props"`
	Children []NewNode      `json:"children,omitempty"`
}

// Proposal is the model's answer after schema and semantic validation.
type Proposal struct {
	Summary    string      `json:"summary"`
	Operations []Operation `json:"operations"`
}

// OperationKinds is the closed set the studio can apply.
var OperationKinds = []string{
	"setText",
	"setProp",
	"addLayer",
	"removeLayer",
	"moveLayer",
	"setTheme",
}

// ContainerTypes may own children. Kept in sync with the client's
// `CONTAINER_TYPES`; the client still re-validates against the live tree.
var ContainerTypes = []string{"page", "stack", "row", "grid", "card", "surface"}

// AddableTypes may be created by a proposal. Structural and asset-bearing
// layers are excluded on purpose: pages and slots are authored here, images
// come through the asset flow, and a model must not invent a fetchable src.
var AddableTypes = []string{
	"stack",
	"row",
	"grid",
	"surface",
	"text",
	"button",
	"input",
	"card",
	"stat",
	"avatar",
	"chip",
	"meter",
	"divider",
	"spacer",
	"swatch",
}

// PaletteKeys are the only theme entries a proposal may change.
var PaletteKeys = []string{
	"bg",
	"surface",
	"surfaceAlt",
	"ink",
	"muted",
	"line",
	"accent",
	"accentInk",
	"accentSoft",
	"secondary",
}
