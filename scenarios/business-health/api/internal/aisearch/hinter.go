package aisearch

import (
	"context"
	"sync"
	"time"

	"business-health/internal/wizard"
)

// WizardHinter adapts the intent-search service to the wizard's Hinter
// seam: before scaffolding, each proposed operational target is queried
// against the fleet corpus and strong matches surface as "similar
// capability exists in scenario X" pointers. Degrades silently — any
// backend failure returns no hints and never blocks the interview.
type WizardHinter struct {
	search func(context.Context, string, int, SearchMode) (*SearchResponse, error)
}

// NewWizardHinter wires the hook (nil service = permanent no-op).
func NewWizardHinter(service *Service) *WizardHinter {
	if service == nil {
		return &WizardHinter{}
	}
	return &WizardHinter{search: func(ctx context.Context, query string, limit int, mode SearchMode) (*SearchResponse, error) {
		return service.Search(ctx, query, limit, mode)
	}}
}

// maxHintsPerTarget caps the pointers per proposed target — the hook
// exists to prevent duplicate capability work, not to pad the interview
// with noise. Confidence comes from the engine's regime-aware Weak label
// (an absolute score floor would be wrong across score regimes: cosine
// ~0.5-0.7 vs RRF-blend ~0.03).
const maxHintsPerTarget = 2
const maxConcurrentHintSearches = 3
const wizardHintBudget = 2500 * time.Millisecond

func (h *WizardHinter) Hints(parent context.Context, scenario string, targets []wizard.OTAnswer) []wizard.CapabilityHint {
	if h == nil || h.search == nil || parent == nil || len(targets) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, wizardHintBudget)
	defer cancel()
	results := make([][]wizard.CapabilityHint, len(targets))
	sem := make(chan struct{}, maxConcurrentHintSearches)
	var wg sync.WaitGroup
	for i, t := range targets {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func(i int, t wizard.OTAnswer) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			query := t.Title
			if t.Description != "" {
				query += " — " + t.Description
			}
			resp, err := h.search(ctx, query, 3, ModeAI)
			if err != nil || resp == nil {
				return // silent degradation by contract
			}
			added := 0
			for _, hit := range resp.Results {
				if hit.Weak || hit.Scenario == scenario || added >= maxHintsPerTarget {
					continue
				}
				added++
				results[i] = append(results[i], wizard.CapabilityHint{
					Scenario:   hit.Scenario,
					Capability: hit.Title,
					Anchor:     hit.Anchor,
					Score:      hit.Score,
				})
			}
		}(i, t)
	}
	wg.Wait()
	var out []wizard.CapabilityHint
	for _, hints := range results {
		out = append(out, hints...)
	}
	return out
}
