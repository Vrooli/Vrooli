// Package designs persists the studio's durable design documents.
//
// A design is the object model from docs/studio/PLAN.md: identity plus an
// opaque frames array and theme. The frames carry the existing TemplateNode
// spec JSON and are never re-modelled here — this package only needs to count
// them and hand them back byte-for-byte.
package designs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrNotFound reports a design or asset id that has no row.
var ErrNotFound = errors.New("design not found")

// ErrConflict reports a save whose base revision is stale. The update is
// conditional in SQL, so the check cannot race another writer.
var ErrConflict = errors.New("design revision conflict")

// ErrInvalid reports a request document that cannot be persisted.
var ErrInvalid = errors.New("invalid design")

// Document is the full design as stored and served.
//
// Frames and Theme are raw JSON because the client owns those shapes. Frames
// is always a JSON array (possibly empty); Theme is always a JSON object.
// Scenario and Page are the optional design ↔ page binding; nil means
// unbound (stored as empty text).
type Document struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	UpdatedAt string          `json:"updatedAt"`
	Revision  int             `json:"revision"`
	Frames    json.RawMessage `json:"frames"`
	Theme     json.RawMessage `json:"theme"`
	Options   json.RawMessage `json:"options"`
	Scenario  *string         `json:"scenario"`
	Page      *string         `json:"page"`
}

// Summary is the list projection: enough to render a design row without
// loading every frame.
type Summary struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	UpdatedAt  string  `json:"updatedAt"`
	FrameCount int     `json:"frameCount"`
	Scenario   *string `json:"scenario"`
	Page       *string `json:"page"`
}

// ListFilter narrows the collection. Empty fields mean "no filter"; matching
// is exact on the stored binding text.
type ListFilter struct {
	Scenario string
	Page     string
}

// CreateInput is a validated create payload. Scenario and Page are optional;
// Options carries design-owned chrome (radius/display) and defaults to {}.
type CreateInput struct {
	Name     string
	Frames   json.RawMessage
	Theme    json.RawMessage
	Options  json.RawMessage
	Scenario *string
	Page     *string
}

// UpdateInput replaces a design document. When Revision is non-nil the update
// only lands if the stored revision still equals it, returning ErrConflict
// otherwise. A nil Revision keeps last-write-wins semantics. A nil Options
// means {} — a save replaces the whole document, exactly like Theme.
type UpdateInput struct {
	Name     string
	Frames   json.RawMessage
	Theme    json.RawMessage
	Options  json.RawMessage
	Scenario *string
	Page     *string
	Revision *int
}

// Asset is the metadata projection of one design-scoped upload. Data lives in
// the table but is only served by the byte route, never in list payloads.
type Asset struct {
	ID       string `json:"id"`
	DesignID string `json:"designId"`
	Name     string `json:"name"`
	Mime     string `json:"mime"`
	Bytes    int64  `json:"bytes"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Fidelity string `json:"fidelity"`
	AddedAt  string `json:"addedAt"`
	URL      string `json:"url"`
}

// AssetInput is a decoded upload. The handler decodes base64; this package
// validates mime, size, fidelity and dimensions.
type AssetInput struct {
	DesignID string
	Name     string
	Mime     string
	Data     []byte
	Fidelity string
	AddedAt  string
}

// Repository is the storage seam. Implementations own revision and
// updated_at; callers never supply them.
type Repository interface {
	List(ctx context.Context, filter ListFilter) ([]Summary, error)
	Get(ctx context.Context, id string) (Document, error)
	Create(ctx context.Context, input CreateInput) (Document, error)
	Save(ctx context.Context, id string, input UpdateInput) (Document, error)
	Delete(ctx context.Context, id string) error

	ListAssets(ctx context.Context, designID string) ([]Asset, error)
	ListAllAssets(ctx context.Context) ([]Asset, error)
	GetAsset(ctx context.Context, designID, assetID string) (Asset, []byte, error)
	CreateAsset(ctx context.Context, input AssetInput) (Asset, error)
	DeleteAsset(ctx context.Context, designID, assetID string) error
}

// normalizeName trims a required display name.
func normalizeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", fmt.Errorf("%w: name is required", ErrInvalid)
	}
	return name, nil
}

// normalizeBinding turns an optional binding into the stored representation:
// nil for unbound, otherwise trimmed non-empty text.
func normalizeBinding(raw *string) *string {
	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil
	}
	return &value
}

// normalizeFrames validates that frames is an array of JSON objects while
// keeping every object opaque, and returns a compacted copy plus its length.
func normalizeFrames(raw json.RawMessage) (json.RawMessage, int, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage("[]"), 0, nil
	}
	var frames []json.RawMessage
	if err := json.Unmarshal(trimmed, &frames); err != nil {
		return nil, 0, fmt.Errorf("%w: frames must be an array", ErrInvalid)
	}
	for i, frame := range frames {
		object := bytes.TrimSpace(frame)
		if len(object) == 0 || object[0] != '{' {
			return nil, 0, fmt.Errorf("%w: frame %d must be an object", ErrInvalid, i)
		}
	}
	compact, err := json.Marshal(frames)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: frames are not valid JSON", ErrInvalid)
	}
	return compact, len(frames), nil
}

// normalizeTheme validates that theme is a JSON object, defaulting to {}.
func normalizeTheme(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage("{}"), nil
	}
	if trimmed[0] != '{' || !json.Valid(trimmed) {
		return nil, fmt.Errorf("%w: theme must be a JSON object", ErrInvalid)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, trimmed); err != nil {
		return nil, fmt.Errorf("%w: theme is not valid JSON", ErrInvalid)
	}
	return compact.Bytes(), nil
}

// Options bounds. The keys are a closed vocabulary so a typo is a 400 rather
// than silently ignored chrome; the size ceiling keeps one document's options
// from becoming a storage slab.
const (
	maxOptionsBytes    = 4 << 10
	minOptionRadius    = 0
	maxOptionRadius    = 48
	optionDisplaySans  = "sans"
	optionDisplaySerif = "serif"
	optionKeyRadius    = "radius"
	optionKeyDisplay   = "display"
)

// normalizeOptions validates the design-owned chrome object and defaults to {}.
//
// Accepted keys: radius (number 0–48 inclusive), display ("sans"|"serif").
// Absent or null means {}, so a client that predates options keeps working.
// Valid values are stored verbatim: the server never re-serialises the object,
// so the client keeps ownership of the JSON it authored.
func normalizeOptions(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage("{}"), nil
	}
	if len(trimmed) > maxOptionsBytes {
		return nil, fmt.Errorf("%w: options must be at most 4 KiB", ErrInvalid)
	}
	if trimmed[0] != '{' || !json.Valid(trimmed) {
		return nil, fmt.Errorf("%w: options must be a JSON object", ErrInvalid)
	}
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &entries); err != nil {
		return nil, fmt.Errorf("%w: options must be a JSON object", ErrInvalid)
	}
	// Sorted so a document with several bad keys always reports the same one.
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := bytes.TrimSpace(entries[key])
		switch key {
		case optionKeyRadius:
			var radius float64
			invalid := bytes.Equal(value, []byte("null")) || json.Unmarshal(value, &radius) != nil
			if invalid || radius < minOptionRadius || radius > maxOptionRadius {
				return nil, fmt.Errorf("%w: options.radius must be a number between 0 and 48", ErrInvalid)
			}
		case optionKeyDisplay:
			var display string
			invalid := bytes.Equal(value, []byte("null")) || json.Unmarshal(value, &display) != nil
			if invalid || (display != optionDisplaySans && display != optionDisplaySerif) {
				return nil, fmt.Errorf("%w: options.display must be %q or %q", ErrInvalid, optionDisplaySans, optionDisplaySerif)
			}
		default:
			return nil, fmt.Errorf("%w: options.%s is not a supported option", ErrInvalid, key)
		}
	}
	return json.RawMessage(trimmed), nil
}
