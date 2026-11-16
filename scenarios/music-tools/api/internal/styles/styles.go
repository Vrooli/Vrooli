package styles

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
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
	Lyrics      string    `json:"lyrics,omitempty"`
	Params      Params    `json:"params"`
	Builtin     bool      `json:"builtin,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type Compiled struct {
	StyleID string
	Caption string
	Lyrics  string
	Params  Params
}

var (
	ErrBuiltinCollision = errors.New("style id collides with a built-in style")
	ErrInvalid          = errors.New("invalid style")
	ErrDecode           = errors.New("decode style")
	ErrBuiltinImmutable = errors.New("built-in styles are immutable")
)

const InstrumentalLyrics = "[instrumental]"
const DefaultStyleScope = "composition"
const InitialDefaultStyleID = "launch-instrumental"

// ResolveLyrics preserves authored text and supplies the preexisting default
// only for an absent/empty legacy style. It does not certify audible content.
func ResolveLyrics(text string) string {
	if text == "" {
		return InstrumentalLyrics
	}
	return text
}

var builtins = []Style{{
	ID: "launch-trap", Name: "Launch Trap",
	Description: "Dark, restless trap with a cold metallic lead and tight drums.",
	Caption:     "dark trap instrumental, 144 bpm, cold detuned metallic lead, menacing and restless, tight syncopated hi-hats, deep sub bass, punchy 808 kick, sparse arrangement, cinematic tension",
	Lyrics:      InstrumentalLyrics,
	Params:      Params{BPM: 144, KeyScale: "G minor", Duration: 45, Steps: 8, Guidance: 1, Variant: "acestep-v15-turbo"},
	Builtin:     true,
}, {
	ID: "launch-club", Name: "Launch Club", Description: "Original harder club instrumental; SA3 prompt-only tempo intent. New takes require audition.",
	Caption: "Original instrumental hard dance and glossy club pop, 172 BPM. An immediate, catchy synth hook over distorted rolling bass, tight pounding kick, sharp claps and busy syncopated percussion. Stacked bright synths, punchy edits, short builds and explosive drops; energetic and rhythmically intricate. Clean, wide, polished mix. Instrumental throughout, no vocals.", Lyrics: InstrumentalLyrics, Params: Params{Duration: 48, Steps: 8, Guidance: 1}, Builtin: true,
}, {
	ID: "launch-instrumental", Name: "Launch Instrumental", Description: "A long-form original instrumental with an evolving hook, layered counter-melodies and rhythmic detail, based on the operator-approved launch score direction.",
	Caption: "Original instrumental electronic dance pop with an immediate distinctive interlocking synth melody, bright glassy plucks answering a more rounded low melodic phrase. Fast punchy breakbeat drums, sharply articulated snare, shuffling syncopated hats and a deep elastic bass line. Bold colorful modern club production with layered counter-melodies, unexpected rhythmic fills and crisp transitions. The hook evolves with variations instead of repeating an identical short loop; sections lift, briefly clear space, then return with new layers. Approximately 160–172 BPM energy. Exciting, playful, vivid and confident. No corporate inspiration music, cinematic trailer drums, generic EDM risers, lyrics, singing, speech or vocal samples.",
	Lyrics:  InstrumentalLyrics, Params: Params{Duration: 110, Steps: 8, Guidance: 1, Variant: "stable-audio-3-medium"}, Builtin: true,
}}

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Store struct {
	mu        sync.RWMutex
	custom    map[string]Style
	db        SQLExecutor
	defaultID string
}

func NewStore() *Store { return &Store{custom: map[string]Style{}, defaultID: InitialDefaultStyleID} }
func NewStoreWithDB(db SQLExecutor) (*Store, error) {
	s := &Store{custom: map[string]Style{}, db: db, defaultID: InitialDefaultStyleID}
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
	style.Lyrics = ResolveLyrics(style.Lyrics)
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

// DefaultID returns the currently configured composition style. A missing
// setting uses the versioned initial default so fresh and upgraded databases
// share the same behavior without a special migration.
func (s *Store) DefaultID(ctx context.Context) (string, error) {
	if s.db == nil {
		s.mu.RLock()
		id := s.defaultID
		s.mu.RUnlock()
		return id, nil
	}
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT style_id FROM music_style_defaults WHERE scope=?`, DefaultStyleScope).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return InitialDefaultStyleID, nil
	}
	if err != nil {
		return "", err
	}
	if _, ok := s.Get(id); !ok {
		return "", fmt.Errorf("configured default style %q no longer exists", id)
	}
	return id, nil
}

func (s *Store) SetDefault(ctx context.Context, id string) error {
	if _, err := s.Compile(id); err != nil {
		return err
	}
	if s.db != nil {
		_, err := s.db.ExecContext(ctx, `INSERT INTO music_style_defaults (scope,style_id,updated_at) VALUES (?,?,?) ON CONFLICT(scope) DO UPDATE SET style_id=excluded.style_id,updated_at=excluded.updated_at`, DefaultStyleScope, id, time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			return fmt.Errorf("save default style: %w", err)
		}
	}
	s.mu.Lock()
	s.defaultID = id
	s.mu.Unlock()
	return nil
}

// Delete removes a custom style. Built-ins are intentionally read-only.
func (s *Store) Delete(id string) error {
	for _, builtin := range builtins {
		if builtin.ID == id {
			return ErrBuiltinImmutable
		}
	}
	s.mu.Lock()
	configuredDefault := s.defaultID
	if s.db != nil {
		var storedDefault string
		if err := s.db.QueryRowContext(context.Background(), `SELECT style_id FROM music_style_defaults WHERE scope=?`, DefaultStyleScope).Scan(&storedDefault); err == nil {
			configuredDefault = storedDefault
		} else if !errors.Is(err, sql.ErrNoRows) {
			s.mu.Unlock()
			return fmt.Errorf("read configured default before delete: %w", err)
		}
	}
	if configuredDefault == id {
		s.mu.Unlock()
		return fmt.Errorf("cannot delete the configured default style")
	}
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
	return protojson.Marshal(&stylespb.Style{Id: style.ID, Name: style.Name, Description: style.Description, Caption: style.Caption, Lyrics: style.Lyrics, Bpm: int32(style.Params.BPM), Keyscale: style.Params.KeyScale, Duration: int32(style.Params.Duration), InferenceSteps: int32(style.Params.Steps), GuidanceScale: style.Params.Guidance, Variant: style.Params.Variant, Builtin: style.Builtin})
}

func decodeStyle(data []byte) (Style, error) {
	var wire stylespb.Style
	if err := protojson.Unmarshal(data, &wire); err != nil {
		var legacy Style
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if legacyErr := decoder.Decode(&legacy); legacyErr != nil {
			return Style{}, fmt.Errorf("%w: portable: %v; legacy: %v", ErrDecode, err, legacyErr)
		}
		if trailingErr := decoder.Decode(new(any)); trailingErr != io.EOF {
			return Style{}, fmt.Errorf("%w: legacy document must contain one JSON value", ErrDecode)
		}
		legacy.Lyrics = ResolveLyrics(legacy.Lyrics)
		return legacy, nil
	}
	return Style{ID: wire.Id, Name: wire.Name, Description: wire.Description, Caption: wire.Caption, Lyrics: ResolveLyrics(wire.Lyrics), Params: Params{BPM: int(wire.Bpm), KeyScale: wire.Keyscale, Duration: int(wire.Duration), Steps: int(wire.InferenceSteps), Guidance: wire.GuidanceScale, Variant: wire.Variant}, Builtin: wire.Builtin}, nil
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
	if style.Params.BPM < 0 || style.Params.Duration <= 0 {
		return Compiled{}, fmt.Errorf("%w: style %q requires nonnegative bpm and positive duration", ErrInvalid, id)
	}
	if style.Params.Steps < 0 || style.Params.Steps > 100 {
		return Compiled{}, fmt.Errorf("%w: style %q inference_steps must be 0..100 (0 is unset)", ErrInvalid, id)
	}
	if math.IsNaN(style.Params.Guidance) || math.IsInf(style.Params.Guidance, 0) || style.Params.Guidance < 0 || style.Params.Guidance > 20 {
		return Compiled{}, fmt.Errorf("%w: style %q guidance_scale must be finite 0..20 (0 is unset)", ErrInvalid, id)
	}
	return Compiled{StyleID: style.ID, Caption: style.Caption, Lyrics: ResolveLyrics(style.Lyrics), Params: style.Params}, nil
}
