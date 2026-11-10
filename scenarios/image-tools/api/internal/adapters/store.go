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
