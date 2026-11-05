// Package looks owns the Look / Style Library (IMG-P1-012): the seeded built-in
// Looks (seed.go), the SQLite store for operator-created custom Looks, the
// pure Look→request compiler (compiler.go, the compose-seam), and the
// deterministic preview renderer (preview.go).
//
// A Look is a named transformation recipe — an ordered chain of deterministic
// ops and/or AI ops plus a prompt template and default params — that generalizes
// presets (OT-P1-010) and recipes (OT-P1-004) into one persisted entity. The
// built-in set is read-only; this store records ONLY custom Looks, merged on top
// of the built-ins at read time (a custom id may never shadow a built-in).
package looks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"image-tools/internal/ai"
	"image-tools/internal/ops"

	"github.com/vrooli/api-core/schedule"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Store errors. Callers map these to Connect codes (NotFound, InvalidArgument,
// FailedPrecondition) for actionable messages.
var (
	ErrNotFound        = errors.New("looks: no such Look")
	ErrBuiltinReadOnly = errors.New("looks: built-in Looks are read-only")
	ErrIDCollision     = errors.New("looks: id already exists")
	ErrInvalid         = errors.New("looks: invalid Look")
)

// SQLExecutor is the narrow database surface the store depends on (satisfied by
// both *sql.DB in tests and *database.RoutedDB in production).
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store persists custom Looks and merges them with the read-only built-ins.
type Store struct {
	db       SQLExecutor
	clock    schedule.Clock
	builtins map[string]*looksv1.Look
	order    []string // built-in ids in seed order
}

// NewStore constructs the looks store over db using the system schedule.
func NewStore(db SQLExecutor) *Store { return NewStoreWithClock(db, schedule.System()) }

// NewStoreWithClock constructs the store with an injected clock (tests).
func NewStoreWithClock(db SQLExecutor, clk schedule.Clock) *Store {
	s := &Store{db: db, clock: clk, builtins: map[string]*looksv1.Look{}}
	for _, b := range BuiltinLooks() {
		s.builtins[b.GetId()] = b
		s.order = append(s.order, b.GetId())
	}
	return s
}

// List returns built-ins (in seed order) then custom Looks (by created_at),
// filtered to kind when kind != LOOK_KIND_UNSPECIFIED.
func (s *Store) List(ctx context.Context, kind looksv1.LookKind) ([]*looksv1.Look, error) {
	out := make([]*looksv1.Look, 0, len(s.order))
	for _, id := range s.order {
		b := s.builtins[id]
		if kind == looksv1.LookKind_LOOK_KIND_UNSPECIFIED || b.GetKind() == kind {
			out = append(out, proto.Clone(b).(*looksv1.Look))
		}
	}
	custom, err := s.listCustom(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range custom {
		if kind == looksv1.LookKind_LOOK_KIND_UNSPECIFIED || c.GetKind() == kind {
			out = append(out, c)
		}
	}
	return out, nil
}

// Get returns one Look (built-in or custom) by id.
func (s *Store) Get(ctx context.Context, id string) (*looksv1.Look, error) {
	if b, ok := s.builtins[id]; ok {
		return proto.Clone(b).(*looksv1.Look), nil
	}
	look, _, err := s.getCustom(ctx, id)
	if err != nil {
		return nil, err
	}
	return look, nil
}

// Create persists a new custom Look. The id is slugified from the name when
// empty; it must not collide with a built-in or an existing custom id.
func (s *Store) Create(ctx context.Context, look *looksv1.Look) (*looksv1.Look, error) {
	if look == nil {
		return nil, fmt.Errorf("%w: nil Look", ErrInvalid)
	}
	id := strings.TrimSpace(look.GetId())
	if id == "" {
		id = slugify(look.GetName())
	} else {
		id = slugify(id)
	}
	if id == "" {
		return nil, fmt.Errorf("%w: a name (or id) is required", ErrInvalid)
	}
	if _, ok := s.builtins[id]; ok {
		return nil, fmt.Errorf("%w: %q collides with a built-in Look", ErrIDCollision, id)
	}
	if _, _, err := s.getCustom(ctx, id); err == nil {
		return nil, fmt.Errorf("%w: %q", ErrIDCollision, id)
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	// Server-controlled fields.
	now := s.clock.Now().UTC().Format(time.RFC3339)
	look = proto.Clone(look).(*looksv1.Look)
	look.Id = id
	look.Builtin = false
	look.ThumbnailRef = ""
	look.CreatedAt = now
	look.UpdatedAt = now
	if look.GetKind() == looksv1.LookKind_LOOK_KIND_UNSPECIFIED {
		look.Kind = looksv1.LookKind_LOOK_KIND_CUSTOM
	}
	if err := Validate(look); err != nil {
		return nil, err
	}
	if err := s.upsert(ctx, look); err != nil {
		return nil, err
	}
	return look, nil
}

// Update replaces a custom Look's mutable fields (name/description/kind/steps/
// prompt_template/params). Built-ins are read-only.
func (s *Store) Update(ctx context.Context, look *looksv1.Look) (*looksv1.Look, error) {
	if look == nil {
		return nil, fmt.Errorf("%w: nil Look", ErrInvalid)
	}
	id := look.GetId()
	if _, ok := s.builtins[id]; ok {
		return nil, fmt.Errorf("%w: %q", ErrBuiltinReadOnly, id)
	}
	existing, _, err := s.getCustom(ctx, id)
	if err != nil {
		return nil, err
	}
	merged := proto.Clone(look).(*looksv1.Look)
	merged.Builtin = false
	merged.ThumbnailRef = existing.GetThumbnailRef() // preserved; changed only by RenderPreview
	merged.CreatedAt = existing.GetCreatedAt()
	merged.UpdatedAt = s.clock.Now().UTC().Format(time.RFC3339)
	if merged.GetKind() == looksv1.LookKind_LOOK_KIND_UNSPECIFIED {
		merged.Kind = looksv1.LookKind_LOOK_KIND_CUSTOM
	}
	if err := Validate(merged); err != nil {
		return nil, err
	}
	if err := s.upsert(ctx, merged); err != nil {
		return nil, err
	}
	return merged, nil
}

// Delete removes a custom Look. Built-ins cannot be deleted.
func (s *Store) Delete(ctx context.Context, id string) error {
	if _, ok := s.builtins[id]; ok {
		return fmt.Errorf("%w: %q", ErrBuiltinReadOnly, id)
	}
	res, err := s.db.ExecContext(ctx, "DELETE FROM look WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("looks: delete %q: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: %q", ErrNotFound, id)
	}
	return nil
}

// SetThumbnail records a rendered preview blob key on a custom Look. Built-ins
// are read-only — the caller returns the rendered ref without persisting it.
func (s *Store) SetThumbnail(ctx context.Context, id, ref string) error {
	if _, ok := s.builtins[id]; ok {
		return fmt.Errorf("%w: %q", ErrBuiltinReadOnly, id)
	}
	look, _, err := s.getCustom(ctx, id)
	if err != nil {
		return err
	}
	look.ThumbnailRef = ref
	look.UpdatedAt = s.clock.Now().UTC().Format(time.RFC3339)
	return s.upsert(ctx, look)
}

// IsBuiltin reports whether id names a built-in Look.
func (s *Store) IsBuiltin(id string) bool {
	_, ok := s.builtins[id]
	return ok
}

func (s *Store) upsert(ctx context.Context, look *looksv1.Look) error {
	raw, err := protojson.Marshal(look)
	if err != nil {
		return fmt.Errorf("looks: marshal %q: %w", look.GetId(), err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO look (id, json, thumbnail_ref, created_at, updated_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET json=excluded.json, thumbnail_ref=excluded.thumbnail_ref, updated_at=excluded.updated_at`,
		look.GetId(), string(raw), look.GetThumbnailRef(), look.GetCreatedAt(), look.GetUpdatedAt())
	if err != nil {
		return fmt.Errorf("looks: persist %q: %w", look.GetId(), err)
	}
	return nil
}

func (s *Store) listCustom(ctx context.Context) ([]*looksv1.Look, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT json FROM look ORDER BY created_at, id")
	if err != nil {
		return nil, fmt.Errorf("looks: list custom: %w", err)
	}
	defer rows.Close()
	var out []*looksv1.Look
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("looks: scan custom: %w", err)
		}
		look := &looksv1.Look{}
		if err := protojson.Unmarshal([]byte(raw), look); err != nil {
			return nil, fmt.Errorf("looks: decode custom: %w", err)
		}
		out = append(out, look)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("looks: iterate custom: %w", err)
	}
	return out, nil
}

func (s *Store) getCustom(ctx context.Context, id string) (*looksv1.Look, bool, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, "SELECT json FROM look WHERE id = ?", id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, fmt.Errorf("%w: %q", ErrNotFound, id)
	}
	if err != nil {
		return nil, false, fmt.Errorf("looks: get %q: %w", id, err)
	}
	look := &looksv1.Look{}
	if err := protojson.Unmarshal([]byte(raw), look); err != nil {
		return nil, false, fmt.Errorf("looks: decode %q: %w", id, err)
	}
	return look, true, nil
}

// Validate enforces the structural rules a usable Look must satisfy: a name, at
// least one step, and every step naming a real operation tagged with the right
// engine (deterministic ⇒ internal/ops, AI ⇒ internal/ai catalog).
func Validate(look *looksv1.Look) error {
	if strings.TrimSpace(look.GetName()) == "" {
		return fmt.Errorf("%w: a name is required", ErrInvalid)
	}
	if len(look.GetSteps()) == 0 {
		return fmt.Errorf("%w: a Look needs at least one step", ErrInvalid)
	}
	for i, step := range look.GetSteps() {
		op := step.GetOperation()
		if op == "" {
			return fmt.Errorf("%w: step %d has no operation", ErrInvalid, i)
		}
		switch step.GetKind() {
		case looksv1.StepKind_STEP_KIND_DETERMINISTIC:
			if !ops.Has(op) {
				return fmt.Errorf("%w: step %d %q is not a deterministic op", ErrInvalid, i, op)
			}
		case looksv1.StepKind_STEP_KIND_AI:
			if !ai.Has(op) {
				return fmt.Errorf("%w: step %d %q is not an AI op", ErrInvalid, i, op)
			}
		default:
			return fmt.Errorf("%w: step %d has an unspecified kind", ErrInvalid, i)
		}
	}
	return nil
}

// slugify lowercases name and collapses non-alphanumeric runs to single
// hyphens, trimming leading/trailing hyphens — the id form for a custom Look.
func slugify(name string) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastHyphen = false
		default:
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
