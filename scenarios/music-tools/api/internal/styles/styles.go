package styles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	stylespb "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/styles"
	"google.golang.org/protobuf/encoding/protojson"
)

type Params struct {
	BPM      int     `json:"bpm"`
	KeyScale string  `json:"key_scale"`
	Duration int     `json:"duration"`
	Steps    int     `json:"steps"`
	Guidance float64 `json:"guidance"`
	Variant  string  `json:"variant"`
}

type Style struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Caption     string    `json:"caption"`
	Params      Params    `json:"params"`
	Builtin     bool      `json:"builtin,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type Compiled struct {
	StyleID string
	Caption string
	Params  Params
}

var ErrBuiltinCollision = errors.New("style id collides with a built-in style")
var ErrInvalid = errors.New("invalid style")
var ErrBuiltinImmutable = errors.New("built-in styles are immutable")

var builtins = []Style{{
	ID: "launch-trap", Name: "Launch Trap",
	Description: "Dark, restless trap with a cold metallic lead and tight drums.",
	Caption:     "dark trap instrumental, 144 bpm, cold detuned metallic lead, menacing and restless, tight syncopated hi-hats, deep sub bass, punchy 808 kick, sparse arrangement, cinematic tension",
	Params:      Params{BPM: 144, KeyScale: "G minor", Duration: 45, Steps: 8, Guidance: 1, Variant: "acestep-v15-turbo"},
	Builtin:     true,
}}

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Store struct {
	mu     sync.RWMutex
	custom map[string]Style
	db     SQLExecutor
}

func NewStore() *Store { return &Store{custom: map[string]Style{}} }
func NewStoreWithDB(db SQLExecutor) (*Store, error) {
	s := &Store{custom: map[string]Style{}, db: db}
	rows, err := db.QueryContext(context.Background(), `SELECT id, json, created_at, updated_at FROM style`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, raw, createdAt, updatedAt string
		if err := rows.Scan(&id, &raw, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		style, err := decodeStyle([]byte(raw))
		if err != nil {
			return nil, fmt.Errorf("styles: load %q: %w", id, err)
		}
		style.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		style.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		style.Builtin = false
		s.custom[id] = style
	}
	return s, rows.Err()
}
func Builtins() []Style { out := make([]Style, len(builtins)); copy(out, builtins); return out }

func (s *Store) Create(style Style) (Style, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	style.ID, style.Name, style.Caption = strings.TrimSpace(style.ID), strings.TrimSpace(style.Name), strings.TrimSpace(style.Caption)
	if style.ID == "" || style.Name == "" || style.Caption == "" {
		return Style{}, ErrInvalid
	}
	for _, builtin := range builtins {
		if builtin.ID == style.ID {
			return Style{}, ErrBuiltinCollision
		}
	}
	if _, exists := s.custom[style.ID]; exists {
		return Style{}, fmt.Errorf("style %q already exists", style.ID)
	}
	now := time.Now().UTC()
	style.CreatedAt, style.UpdatedAt = now, now
	s.custom[style.ID] = style
	if s.db != nil {
		raw, err := encodeStyle(style)
		if err != nil {
			delete(s.custom, style.ID)
			return Style{}, err
		}
		if _, err := s.db.ExecContext(context.Background(), `INSERT INTO style(id, json, created_at, updated_at) VALUES (?, ?, ?, ?)`, style.ID, string(raw), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
			delete(s.custom, style.ID)
			return Style{}, err
		}
	}
	return style, nil
}

func (s *Store) Get(id string) (Style, bool) {
	for _, builtin := range builtins {
		if builtin.ID == id {
			return builtin, true
		}
	}
	s.mu.RLock()
	style, ok := s.custom[id]
	s.mu.RUnlock()
	return style, ok
}

// Delete removes a custom style. Built-ins are intentionally read-only.
func (s *Store) Delete(id string) error {
	for _, builtin := range builtins {
		if builtin.ID == id {
			return ErrBuiltinImmutable
		}
	}
	s.mu.Lock()
	if _, ok := s.custom[id]; !ok {
		s.mu.Unlock()
		return fmt.Errorf("style %q not found", id)
	}
	delete(s.custom, id)
	if s.db != nil {
		if _, err := s.db.ExecContext(context.Background(), `DELETE FROM style WHERE id=?`, id); err != nil {
			s.mu.Unlock()
			return err
		}
	}
	s.mu.Unlock()
	return nil
}

// Export returns the portable JSON representation of a style. Runtime fields
// are omitted so an import creates a fresh custom record.
func (s *Store) Export(id string) ([]byte, error) {
	style, ok := s.Get(id)
	if !ok {
		return nil, fmt.Errorf("style %q not found", id)
	}
	style.Builtin, style.CreatedAt, style.UpdatedAt = false, time.Time{}, time.Time{}
	return encodeStyle(style)
}

// Import creates a custom style from portable JSON.
func (s *Store) Import(data []byte) (Style, error) {
	style, err := decodeStyle(data)
	if err != nil {
		return Style{}, err
	}
	style.Builtin = false
	style.CreatedAt = time.Time{}
	style.UpdatedAt = time.Time{}
	return s.Create(style)
}

func encodeStyle(style Style) ([]byte, error) {
	return protojson.Marshal(&stylespb.Style{Id: style.ID, Name: style.Name, Description: style.Description, Caption: style.Caption, Bpm: int32(style.Params.BPM), Keyscale: style.Params.KeyScale, Duration: int32(style.Params.Duration), InferenceSteps: int32(style.Params.Steps), GuidanceScale: style.Params.Guidance, Variant: style.Params.Variant, Builtin: style.Builtin})
}

func decodeStyle(data []byte) (Style, error) {
	var wire stylespb.Style
	if err := protojson.Unmarshal(data, &wire); err != nil {
		var legacy Style
		if legacyErr := json.Unmarshal(data, &legacy); legacyErr != nil {
			return Style{}, fmt.Errorf("decode style: %w", err)
		}
		return legacy, nil
	}
	return Style{ID: wire.Id, Name: wire.Name, Description: wire.Description, Caption: wire.Caption, Params: Params{BPM: int(wire.Bpm), KeyScale: wire.Keyscale, Duration: int(wire.Duration), Steps: int(wire.InferenceSteps), Guidance: wire.GuidanceScale, Variant: wire.Variant}, Builtin: wire.Builtin}, nil
}

func (s *Store) List() []Style {
	out := Builtins()
	s.mu.RLock()
	for _, style := range s.custom {
		out = append(out, style)
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Store) Compile(id string) (Compiled, error) {
	style, ok := s.Get(id)
	if !ok {
		return Compiled{}, fmt.Errorf("style %q not found", id)
	}
	if style.Params.BPM <= 0 || style.Params.Duration <= 0 {
		return Compiled{}, fmt.Errorf("%w: style %q requires bpm and duration", ErrInvalid, id)
	}
	return Compiled{StyleID: style.ID, Caption: style.Caption, Params: style.Params}, nil
}
