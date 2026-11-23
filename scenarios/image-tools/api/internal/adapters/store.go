package adapters

import "image-tools/internal/catalogstate"

type SQLExecutor = catalogstate.SQLExecutor
type Store = catalogstate.Overlay

func NewStore(db SQLExecutor) *Store { return catalogstate.NewOverlay(db, catalogstate.Adapters) }

// EffectiveEnabled reports whether adapter a is enabled given the overlay (an
// explicit override wins; otherwise the seed default).
func EffectiveEnabled(a Adapter, overlay map[string]bool) bool {
	if v, ok := overlay[a.ID]; ok {
		return v
	}
	return a.Enabled
}

// SeedState reports each adapter's seed enabled flag for the override
// supersession rule (catalogstate.Overlay.Reconcile). Operator-registered
// entries count as seed.
func (r *Registry) SeedState(customs []Adapter) catalogstate.SeedState {
	custom := make(map[string]Adapter, len(customs))
	for _, a := range customs {
		custom[a.ID] = a
	}
	return func(id string) (bool, bool) {
		if a, ok := r.ByID(id); ok {
			return a.Enabled, true
		}
		if a, ok := custom[id]; ok {
			return a.Enabled, true
		}
		return false, false
	}
}
