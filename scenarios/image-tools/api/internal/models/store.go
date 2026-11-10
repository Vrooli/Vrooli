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
	if m.CapabilityLabels.CommercialUse != CommercialUseYes {
		return false
	}
	if v, ok := overlay[m.ID]; ok {
		return v
	}
	return m.Enabled
}
