package models

import "image-tools/internal/catalogstate"

type SQLExecutor = catalogstate.SQLExecutor
type Store = catalogstate.Overlay

func NewStore(db SQLExecutor) *Store { return catalogstate.NewOverlay(db, catalogstate.Models) }

// EnabledWithOverlay returns an EnabledFunc that reports a model's effective
// enabled state: an explicit overlay override when present, else the seed
// default. Unknown ids report false. Pass the result to Registry.Select and use
// EffectiveEnabled for the wire shape so the two never disagree.
func (r *Registry) EnabledWithOverlay(overlay map[string]bool) EnabledFunc {
	return func(id string) bool {
		if m, ok := r.byID[id]; ok {
			return EffectiveEnabled(m, overlay)
		}
		return false
	}
}

// EffectiveEnabled reports whether model m is enabled given the overlay (an
// explicit override wins; otherwise the seed default).
func EffectiveEnabled(m Model, overlay map[string]bool) bool {
	// Rights are not decided here: the shared rights owner gates enable,
	// install, selection and execution against the operator's current context.
	if v, ok := overlay[m.ID]; ok {
		return v
	}
	return m.Enabled
}

// SeedState reports each model's seed enabled flag for the override
// supersession rule (catalogstate.Overlay.Reconcile). Operator-registered
// entries count as seed: their stored flag is the baseline an override is
// compared against.
func (r *Registry) SeedState(customs []Model) catalogstate.SeedState {
	custom := make(map[string]Model, len(customs))
	for _, m := range customs {
		custom[m.ID] = m
	}
	return func(id string) (bool, bool) {
		if m, ok := r.byID[id]; ok {
			return m.Enabled, true
		}
		if m, ok := custom[id]; ok {
			return m.Enabled, true
		}
		return false, false
	}
}
