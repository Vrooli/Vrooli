package focus

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// defaultFocusLimit bounds GetFocus when the caller passes 0.
const defaultFocusLimit = 5

// Service is the focus application surface.
type Service interface {
	GetFocus(ctx context.Context, limit int, projection Projection) (FocusResult, error)
	ListGaps(ctx context.Context, filter GapFilter) ([]Gap, error)
	GetGap(ctx context.Context, id string) (Gap, error)
	AddGapNote(ctx context.Context, id, approach string) (Gap, error)
	ListCondition(ctx context.Context) ([]Gap, error)
	ListConditionReport(ctx context.Context) (ConditionReport, error)
	ExplainCondition(ctx context.Context, providerID string) (Gap, error)
}

type service struct {
	source          GapSource
	conditionSource GapSource
	repo            Repository
	insights        ProviderInsights
	conditionReader ProgramConditionReader
}

// Deps wires the focus Service. Repo is optional (nil disables persistence; the
// service still surfaces live-derived gaps but AddGapNote then errors).
type Deps struct {
	Source          GapSource
	ConditionSource GapSource
	Repo            Repository
	Insights        ProviderInsights
	ConditionReader ProgramConditionReader
}

// NewService constructs the focus Service.
func NewService(d Deps) Service {
	return &service{source: d.Source, conditionSource: d.ConditionSource, repo: d.Repo, insights: d.Insights, conditionReader: d.ConditionReader}
}

var _ Service = (*service)(nil)

// allGaps reads the live-derived gaps and overlays the owned registry, returning
// the merged set in a stable order (derived first by projection, then
// registry-only). Degrades gracefully: a nil/erroring source or repo contributes
// nothing rather than failing the read.
func (s *service) allGaps(ctx context.Context) ([]Gap, error) {
	var derived, registry []Gap
	var errs []error
	if s.source != nil {
		d, err := s.source.DerivedGaps(ctx)
		derived = d
		if err != nil {
			errs = append(errs, err)
		}
	}
	if s.repo != nil {
		r, err := s.repo.List(ctx)
		if err == nil {
			registry = r
		} else {
			errs = append(errs, fmt.Errorf("focus registry unavailable: %w", err))
		}
	}
	return mergeGaps(derived, registry), errors.Join(errs...)
}

// GetFocus returns the ranked next-best gaps (impact × importance), optionally
// filtered by projection, capped at limit (default when limit<=0).
func (s *service) GetFocus(ctx context.Context, limit int, projection Projection) (FocusResult, error) {
	if projection != "" && OwnerFor(projection) == "" {
		return FocusResult{}, fmt.Errorf("focus: unknown projection %q", projection)
	}
	gaps, liveErr := s.allGaps(ctx)
	var degraded []error
	if liveErr != nil {
		degraded = append(degraded, fmt.Errorf("live focus data unavailable: %w", liveErr))
	}
	gaps = rollupSharedCauses(gaps)
	providerSignals := map[string]ProviderInsight{}
	if s.insights == nil {
		degraded = append(degraded, errors.New("search-hub insights unavailable: provider insights reader is not configured"))
	} else if signals, err := s.insights.Insights(ctx); err != nil {
		degraded = append(degraded, fmt.Errorf("search-hub insights unavailable: %w", err))
	} else {
		for _, signal := range signals {
			for _, key := range []string{signal.ProviderID, signal.ProviderGroup} {
				key = strings.ToLower(strings.TrimSpace(key))
				if key != "" {
					providerSignals[key] = signal
				}
			}
		}
	}
	items := make([]FocusItem, 0, len(gaps))
	for _, g := range gaps {
		if projection != "" && g.Projection != projection {
			continue
		}
		imp, impo, pri, why := scoreWithInsights(g, providerSignals)
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
	result := FocusResult{Items: items}
	if len(degraded) > 0 {
		result.Degraded = true
		result.DegradedReason = errors.Join(degraded...).Error()
	}
	return result, nil
}

// rollupSharedCauses replaces multiple cell-level findings with one
// actionable cause item when they carry the same explicit cause key. The
// explicit key is supplied by the producer that understands the failure
// signal; focus never guesses that unrelated provider failures share a cause.
func rollupSharedCauses(gaps []Gap) []Gap {
	groups := make(map[string][]Gap)
	for _, gap := range gaps {
		key := strings.TrimSpace(gap.CauseKey)
		if key == "" {
			continue
		}
		groups[key] = append(groups[key], gap)
	}
	if len(groups) == 0 {
		return gaps
	}
	rolled := make([]Gap, 0, len(gaps))
	seen := make(map[string]struct{}, len(groups))
	for _, gap := range gaps {
		key := strings.TrimSpace(gap.CauseKey)
		if key == "" {
			rolled = append(rolled, gap)
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		members := groups[key]
		if len(members) < 2 {
			rolled = append(rolled, members[0])
			continue
		}
		rolled = append(rolled, sharedCauseGap(key, members))
	}
	return rolled
}

func sharedCauseGap(key string, members []Gap) Gap {
	cellIDs := make([]string, 0, len(members))
	providers := make([]string, 0)
	seenCells := map[string]struct{}{}
	seenProviders := map[string]struct{}{}
	for _, member := range members {
		cellID := member.SourceCellID
		if cellID == "" {
			cellID = member.ID
		}
		if _, ok := seenCells[cellID]; !ok {
			seenCells[cellID] = struct{}{}
			cellIDs = append(cellIDs, cellID)
		}
		for _, provider := range member.ProviderIDs {
			if _, ok := seenProviders[provider]; ok {
				continue
			}
			seenProviders[provider] = struct{}{}
			providers = append(providers, provider)
		}
	}
	sort.Strings(cellIDs)
	sort.Strings(providers)
	slug := strings.NewReplacer("/", "-", " ", "-").Replace(key)
	return Gap{
		ID: "cause/" + slug, Axis: AxisEmpirical, Projection: ProjectionAnswer,
		Title: "shared cause: " + key, Global: true, ProviderIDs: providers,
		EvidenceSource: "focus", EvidenceLocator: "focus://cause/" + key,
		ConditionStatus: "degraded", CauseKey: key, AffectedCellIDs: cellIDs,
		AffectedCellCount: len(cellIDs), Recurrence: len(cellIDs),
		Notes: []string{fmt.Sprintf("affects %d denominator cell(s)", len(cellIDs))},
	}
}

// ListGaps returns the merged registry, filtered.
func (s *service) ListGaps(ctx context.Context, filter GapFilter) ([]Gap, error) {
	gaps, _ := s.allGaps(ctx) // source failures are represented by availability gaps
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

func (s *service) ListCondition(ctx context.Context) ([]Gap, error) {
	report, err := s.ListConditionReport(ctx)
	return report.Gaps, err
}

func (s *service) ListConditionReport(ctx context.Context) (ConditionReport, error) {
	var gaps []Gap
	if s.conditionSource != nil {
		gaps, _ = s.conditionSource.DerivedGaps(ctx)
	} else {
		gaps, _ = s.allGaps(ctx)
	}
	var out []Gap
	for _, gap := range gaps {
		if gap.Axis == AxisEmpirical && (strings.HasPrefix(gap.ID, "condition/") || strings.HasPrefix(gap.ID, "source/condition/") || strings.HasPrefix(gap.ID, "source/maturity/")) {
			out = append(out, gap)
		}
	}
	// Condition is an observational surface. Sibling focus sources may be
	// unavailable while Search Hub condition evidence is still usable; do not
	// turn that partial result into a transport error. A condition/maturity
	// source failure is retained as its own source/<name>/availability gap above.
	report := ConditionReport{Gaps: out}
	if s.conditionReader != nil {
		condition, err := s.conditionReader.ReadCondition(ctx)
		if err == nil {
			report.Instrumentation = ConditionInstrumentation{
				Healthy: condition.Healthy, Degraded: condition.Degraded,
				Dormant: condition.Dormant, Uninstrumented: condition.Uninstrumented,
				Unavailable: condition.Unavailable, Instrumented: condition.Instrumented,
				Total: condition.Total, FilteredOut: condition.FilteredOut,
			}
		}
	}
	return report, nil
}

func (s *service) ExplainCondition(ctx context.Context, providerID string) (Gap, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return Gap{}, fmt.Errorf("condition: provider id required")
	}
	id := providerID
	if !strings.HasPrefix(id, "condition/") {
		id = "condition/" + id
	}
	var gap Gap
	var err error
	if s.conditionSource != nil {
		var gaps []Gap
		gaps, err = s.conditionSource.DerivedGaps(ctx)
		for _, candidate := range gaps {
			if candidate.ID == id {
				gap = candidate
				break
			}
		}
		if gap.ID != "" {
			err = nil
		} else if err == nil {
			err = fmt.Errorf("focus: gap %q not found", id)
		}
	} else {
		gap, err = s.GetGap(ctx, id)
	}
	if err != nil {
		return Gap{}, err
	}
	if gap.Axis != AxisEmpirical {
		return Gap{}, fmt.Errorf("condition: %q is not an observed condition leg", providerID)
	}
	return gap, nil
}

// GetGap returns one merged gap by id.
func (s *service) GetGap(ctx context.Context, id string) (Gap, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Gap{}, fmt.Errorf("focus: empty gap id")
	}
	gaps, _ := s.allGaps(ctx) // source failures are represented by availability gaps
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
		if r.Axis == "" {
			r.Axis = AxisCoverage
		}
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
