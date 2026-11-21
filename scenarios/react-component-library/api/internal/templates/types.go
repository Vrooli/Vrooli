// Package templates persists the design studio's template shelf.
//
// A template is an authoring draft: the same Template shape the studio renders
// (ui/src/workbench/templates/spec.ts), kept server-side so a shelf entry is
// durable across browsers instead of living only in localStorage. It is not a
// published library asset; `local` on the client still means "authoring draft".
//
// palette and spec are opaque JSON: the server never re-models the TemplateNode
// tree. The spec is stored inside spec_json alongside the two authored copy
// fields the shelf renders (slotHint, idea) because the table's columns are
// fixed and those strings belong to the same authored document.
package templates

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	// ErrNotFound reports an id that has no row.
	ErrNotFound = errors.New("template not found")
	// ErrConflict reports a create whose id already exists.
	ErrConflict = errors.New("template already exists")
	// ErrInvalid reports a payload that cannot be persisted. The wrapped
	// message names the offending field.
	ErrInvalid = errors.New("invalid template")
)

const (
	// MaxSpecBytes bounds the spec JSON (1 MiB) before it is stored.
	MaxSpecBytes = 1 << 20
	// MaxPaletteBytes bounds the palette JSON (8 KiB).
	MaxPaletteBytes = 8 << 10

	MaxNameLength     = 120
	MaxTaglineLength  = 240
	MaxCategoryLength = 80

	MinRadius = 0
	MaxRadius = 48
)

// idPattern is the client's stable slug shape: lower-case, starts alphanumeric,
// then letters, digits, dot, underscore and hyphen.
var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)

// Template is the wire and storage shape. Palette and Spec are raw JSON
// because the client owns those shapes; both are always JSON objects (possibly
// empty). SlotHint and Idea are the authored copy that travels with the spec.
type Template struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Tagline   string          `json:"tagline"`
	Category  string          `json:"category"`
	Palette   json.RawMessage `json:"palette"`
	Display   string          `json:"display"`
	Radius    int             `json:"radius"`
	Spec      json.RawMessage `json:"spec"`
	SlotHint  string          `json:"slotHint"`
	Idea      string          `json:"idea"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt string          `json:"updatedAt"`
}

// Input is a create or replace payload. Pointer fields distinguish "absent"
// (use the documented default) from an explicit zero value. ID is
// client-assigned on create and ignored on update (the path is authoritative).
type Input struct {
	ID       string
	Name     string
	Tagline  string
	Category string
	Palette  json.RawMessage
	Display  string
	Radius   *int
	Spec     json.RawMessage
	SlotHint string
	Idea     string
}

// Repository is the storage seam. Implementations own created_at/updated_at.
type Repository interface {
	List(ctx context.Context) ([]Template, error)
	Get(ctx context.Context, id string) (Template, error)
	Create(ctx context.Context, input Input) (Template, error)
	Update(ctx context.Context, id string, input Input) (Template, error)
	Delete(ctx context.Context, id string) error
}

// authoredSpec is the spec_json envelope: the TemplateNode tree plus the copy
// fields the shelf renders. Storing them together keeps the table's column
// contract while round-tripping a Template without loss.
type authoredSpec struct {
	Spec     json.RawMessage `json:"spec"`
	SlotHint string          `json:"slotHint,omitempty"`
	Idea     string          `json:"idea,omitempty"`
}

// normalized is a validated payload ready for the INSERT/UPDATE.
type normalized struct {
	name     string
	tagline  string
	category string
	palette  json.RawMessage
	display  string
	radius   int
	spec     json.RawMessage
	slotHint string
	idea     string
}

// validateID checks the stable slug contract. Empty is reported as missing.
func validateID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalid)
	}
	if !idPattern.MatchString(id) {
		return fmt.Errorf("%w: id must match ^[a-z0-9][a-z0-9._-]{0,127}$", ErrInvalid)
	}
	return nil
}

// normalize applies the documented defaults and limits. Every error names the
// field so the handler can return it verbatim under invalid_argument.
func normalize(input Input) (normalized, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return normalized{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if utf8.RuneCountInString(name) > MaxNameLength {
		return normalized{}, fmt.Errorf("%w: name must be at most %d characters", ErrInvalid, MaxNameLength)
	}

	tagline := strings.TrimSpace(input.Tagline)
	if utf8.RuneCountInString(tagline) > MaxTaglineLength {
		return normalized{}, fmt.Errorf("%w: tagline must be at most %d characters", ErrInvalid, MaxTaglineLength)
	}

	category := strings.TrimSpace(input.Category)
	if utf8.RuneCountInString(category) > MaxCategoryLength {
		return normalized{}, fmt.Errorf("%w: category must be at most %d characters", ErrInvalid, MaxCategoryLength)
	}

	display := strings.ToLower(strings.TrimSpace(input.Display))
	if display == "" {
		display = "sans"
	}
	if display != "sans" && display != "serif" {
		return normalized{}, fmt.Errorf(`%w: display must be "sans" or "serif"`, ErrInvalid)
	}

	radius := 14
	if input.Radius != nil {
		radius = *input.Radius
	}
	if radius < MinRadius || radius > MaxRadius {
		return normalized{}, fmt.Errorf("%w: radius must be between %d and %d", ErrInvalid, MinRadius, MaxRadius)
	}

	palette, err := normalizeJSONObject("palette", input.Palette, MaxPaletteBytes)
	if err != nil {
		return normalized{}, err
	}
	spec, err := normalizeJSONObject("spec", input.Spec, MaxSpecBytes)
	if err != nil {
		return normalized{}, err
	}

	return normalized{
		name:     name,
		tagline:  tagline,
		category: category,
		palette:  palette,
		display:  display,
		radius:   radius,
		spec:     spec,
		slotHint: strings.TrimSpace(input.SlotHint),
		idea:     strings.TrimSpace(input.Idea),
	}, nil
}

// normalizeJSONObject validates an opaque JSON object and returns its compact
// stored form. Absent/null means the empty object.
func normalizeJSONObject(field string, raw json.RawMessage, limit int) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage("{}"), nil
	}
	if len(trimmed) > limit {
		return nil, fmt.Errorf("%w: %s must be at most %d bytes", ErrInvalid, field, limit)
	}
	if trimmed[0] != '{' || !json.Valid(trimmed) {
		return nil, fmt.Errorf("%w: %s must be a JSON object", ErrInvalid, field)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, trimmed); err != nil {
		return nil, fmt.Errorf("%w: %s is not valid JSON", ErrInvalid, field)
	}
	return compact.Bytes(), nil
}

// specDocument renders the spec_json column value.
func (n normalized) specDocument() ([]byte, error) {
	document, err := json.Marshal(authoredSpec{
		Spec:     n.spec,
		SlotHint: n.slotHint,
		Idea:     n.idea,
	})
	if err != nil {
		return nil, fmt.Errorf("encode spec document: %w", err)
	}
	return document, nil
}

// decodeSpecDocument is the inverse of specDocument. A legacy/foreign row that
// stored a bare TemplateNode (no envelope) is still readable: the whole value
// is then treated as the spec and the copy fields are empty.
func decodeSpecDocument(raw string) (spec json.RawMessage, slotHint, idea string) {
	trimmed := bytes.TrimSpace([]byte(raw))
	if len(trimmed) == 0 {
		return json.RawMessage("{}"), "", ""
	}
	var document authoredSpec
	if err := json.Unmarshal(trimmed, &document); err == nil && document.Spec != nil {
		slotHint = document.SlotHint
		idea = document.Idea
		return validJSONObject(document.Spec), slotHint, idea
	}
	return validJSONObject(json.RawMessage(trimmed)), "", ""
}

func validJSONObject(raw json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '{' && json.Valid(trimmed) {
		return json.RawMessage(trimmed)
	}
	return json.RawMessage("{}")
}
