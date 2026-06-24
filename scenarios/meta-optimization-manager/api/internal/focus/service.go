package focus

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// defaultFocusLimit bounds GetFocus when the caller passes 0.
const defaultFocusLimit = 5

// Service is the focus application surface.
type Service interface {
	GetFocus(ctx context.Context, limit int, projection Projection) ([]FocusItem, error)
	ListGaps(ctx context.Context, filter GapFilter) ([]Gap, error)
	GetGap(ctx context.Context, id string) (Gap, error)
	AddGapNote(ctx context.Context, id, approach string) (Gap, error)
}

type service struct {
	source GapSource
	repo   Repository
}

// Deps wires the focus Service. Repo is optional (nil disables persistence; the
// service still surfaces live-derived gaps but AddGapNote then errors).
type Deps struct {
	Source GapSource
	Repo   Repository
}

// NewService constructs the focus Service.
func NewService(d Deps) Service {
	return &service{source: d.Source, repo: d.Repo}
}

var _ Service = (*service)(nil)

// allGaps reads the live-derived gaps and overlays the owned registry, returning
// the merged set in a stable order (derived first by projection, then
// registry-only). Degrades gracefully: a nil/erroring source or repo contributes
// nothing rather than failing the read.
func (s *service) allGaps(ctx context.Context) ([]Gap, error) {
	var derived, registry []Gap
	if s.source != nil {
		d, err := s.source.DerivedGaps(ctx)
		if err == nil {
			derived = d
		}
	}
	if s.repo != nil {
		r, err := s.repo.List(ctx)
		if err == nil {
			registry = r
		}
	}
	return mergeGaps(derived, registry), nil
}

// GetFocus returns the ranked next-best gaps (impact × importance), optionally
// filtered by projection, capped at limit (default when limit<=0).
func (s *service) GetFocus(ctx context.Context, limit int, projection Projection) ([]FocusItem, error) {
	if projection != "" && OwnerFor(projection) == "" {
		return nil, fmt.Errorf("focus: unknown projection %q", projection)
	}
	gaps, err := s.allGaps(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]FocusItem, 0, len(gaps))
	for _, g := range gaps {
		if projection != "" && g.Projection != projection {
			continue
		}
		imp, impo, pri, why := score(g)
		items = append(items, FocusItem{Gap: g, Impact: imp, Importance: impo, Priority: pri, Rationale: why})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority > items[j].Priority
		}
		return items[i].Gap.ID < items[j].Gap.ID // stable tiebreak
	})
	if limit <= 0 {
		limit = defaultFocusLimit
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

// ListGaps returns the merged registry, filtered.
func (s *service) ListGaps(ctx context.Context, filter GapFilter) ([]Gap, error) {
	gaps, err := s.allGaps(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Gap, 0, len(gaps))
	for _, g := range gaps {
		if filter.Projection != "" && g.Projection != filter.Projection {
			continue
		}
		if filter.CellID != "" && g.SourceCellID != filter.CellID {
			continue
		}
		if filter.Status != "" && g.Status != filter.Status {
			continue
		}
		out = append(out, g)
	}
	return out, nil
}

// GetGap returns one merged gap by id.
func (s *service) GetGap(ctx context.Context, id string) (Gap, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Gap{}, fmt.Errorf("focus: empty gap id")
	}
	gaps, err := s.allGaps(ctx)
	if err != nil {
		return Gap{}, err
	}
	for _, g := range gaps {
		if g.ID == id {
			return g, nil
		}
	}
	return Gap{}, fmt.Errorf("focus: gap %q not found", id)
}

// AddGapNote appends an explored approach to a gap — the one focus write verb.
// The gap must exist in the merged set (derived or registry). Appending to a
// derived gap materializes a registry row keyed by the same id (so the team's
// thinking survives); appending to an existing registry row extends it.
func (s *service) AddGapNote(ctx context.Context, id, approach string) (Gap, error) {
	id = strings.TrimSpace(id)
	approach = strings.TrimSpace(approach)
	if id == "" {
		return Gap{}, fmt.Errorf("focus: empty gap id")
	}
	if approach == "" {
		return Gap{}, fmt.Errorf("focus: empty approach")
	}
	if s.repo == nil {
		return Gap{}, fmt.Errorf("focus: registry unavailable; cannot persist note")
	}
	// The gap must exist in the merged set (the source of truth for what is a gap).
	base, err := s.GetGap(ctx, id)
	if err != nil {
		return Gap{}, err
	}
	// Overlay the existing registry row (if any) so we append, never clobber.
	row, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return Gap{}, fmt.Errorf("focus: read registry row %q: %w", id, err)
	}
	if !ok {
		// Materialize a registry row from the derived gap's identity.
		row = Gap{
			ID:           base.ID,
			Projection:   base.Projection,
			Title:        base.Title,
			Status:       base.Status,
			SourceCellID: base.SourceCellID,
			Global:       base.Global,
			Notes:        append([]string(nil), base.Notes...),
		}
	}
	row.Approaches = appendUnique(row.Approaches, approach)
	if err := s.repo.Upsert(ctx, row); err != nil {
		return Gap{}, fmt.Errorf("focus: persist note: %w", err)
	}
	return s.GetGap(ctx, id)
}

// mergeGaps overlays the registry onto the live-derived gaps: a registry row
// sharing a derived gap's id contributes its accumulated approaches/follow-ups
// and any extra notes; a registry-only row (global gap) is appended. Derived
// projection/status/title stay authoritative (they are the live truth).
func mergeGaps(derived, registry []Gap) []Gap {
	byID := make(map[string]*Gap, len(derived)+len(registry))
	order := make([]string, 0, len(derived)+len(registry))
	for i := range derived {
		cp := derived[i]
		byID[cp.ID] = &cp
		order = append(order, cp.ID)
	}
	for i := range registry {
		r := registry[i]
		if base, ok := byID[r.ID]; ok {
			base.Approaches = mergeUnique(base.Approaches, r.Approaches)
			base.FollowUps = mergeUnique(base.FollowUps, r.FollowUps)
			base.Notes = mergeUnique(base.Notes, r.Notes)
			continue
		}
		cp := r
		byID[cp.ID] = &cp
		order = append(order, cp.ID)
	}
	out := make([]Gap, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out
}

// appendUnique appends s to list iff not already present (case-sensitive).
func appendUnique(list []string, s string) []string {
	for _, e := range list {
		if e == s {
			return list
		}
	}
	return append(list, s)
}

// mergeUnique returns a ∪ b preserving a's order then b's new items.
func mergeUnique(a, b []string) []string {
	seen := make(map[string]bool, len(a))
	out := make([]string, 0, len(a)+len(b))
	for _, e := range a {
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	for _, e := range b {
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	return out
}
