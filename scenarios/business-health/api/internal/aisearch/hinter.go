package aisearch

import (
	"context"
	"time"

	"business-health/internal/wizard"
)

// WizardHinter adapts the intent-search service to the wizard's Hinter
// seam: before scaffolding, each proposed operational target is queried
// against the fleet corpus and strong matches surface as "similar
// capability exists in scenario X" pointers. Degrades silently — any
// backend failure returns no hints and never blocks the interview.
type WizardHinter struct {
	service *Service
}

// NewWizardHinter wires the hook (nil service = permanent no-op).
func NewWizardHinter(service *Service) *WizardHinter {
	return &WizardHinter{service: service}
}

// maxHintsPerTarget caps the pointers per proposed target — the hook
// exists to prevent duplicate capability work, not to pad the interview
// with noise. Confidence comes from the engine's regime-aware Weak label
// (an absolute score floor would be wrong across score regimes: cosine
// ~0.5-0.7 vs RRF-blend ~0.03).
const maxHintsPerTarget = 2

func (h *WizardHinter) Hints(scenario string, targets []wizard.OTAnswer) []wizard.CapabilityHint {
	if h == nil || h.service == nil {
		return nil
	}
	var out []wizard.CapabilityHint
	for _, t := range targets {
		query := t.Title
		if t.Description != "" {
			query += " — " + t.Description
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		resp, err := h.service.Search(ctx, query, 3, ModeAI)
		cancel()
		if err != nil || resp == nil {
			continue // silent degradation by contract
		}
		added := 0
		for _, hit := range resp.Results {
			if hit.Weak || hit.Scenario == scenario || added >= maxHintsPerTarget {
				continue
			}
			added++
			out = append(out, wizard.CapabilityHint{
				Scenario:   hit.Scenario,
				Capability: hit.Title,
				Anchor:     hit.Anchor,
				Score:      hit.Score,
			})
		}
	}
	return out
}
