package manifest

// DefaultSignalWeights returns the day-one weights documented in
// docs/concepts/SIGNAL_LADDER.md::Default Weights Summary. Returning
// a fresh map per call so callers can mutate freely without affecting
// other callers.
func DefaultSignalWeights() map[string]float64 {
	return map[string]float64{
		"path-token":      1.5,
		"import-cluster":  1.0,
		"symbol-glossary": 0.9,
		"importer-voting": 0.8,
		"test-coupling":   0.7,
		"git-co-edit":     0.6,
	}
}

// DefaultThresholds returns the day-one tier thresholds (auto_place
// >=0.85, suggest >=0.55) documented in
// docs/concepts/SIGNAL_LADDER.md.
func DefaultThresholds() []Threshold {
	return []Threshold{
		{Tier: "auto_place", MinValue: 0.85},
		{Tier: "suggest", MinValue: 0.55},
	}
}

// EffectiveWeights computes the per-domain effective signal weights by
// layering, in order:
//
//  1. day-one defaults (DefaultSignalWeights),
//  2. manifest-level overlay (m.SignalWeights),
//  3. per-domain overrides (DomainSpec.SignalWeightOverrides).
//
// A weight of 0 disables the signal for that domain; absent keys
// inherit from the next layer down. Pass an empty string for domain
// to get the manifest-level overlay only.
func EffectiveWeights(m ManifestDefinition, domain string) map[string]float64 {
	out := DefaultSignalWeights()
	for k, v := range m.SignalWeights.Weights {
		out[k] = v
	}
	if domain == "" {
		return out
	}
	for _, d := range m.Domains {
		if d.Name != domain {
			continue
		}
		for k, v := range d.SignalWeightOverrides.Weights {
			out[k] = v
		}
		return out
	}
	return out
}

// EffectiveThresholds merges the manifest's threshold list with the
// day-one defaults. Manifest entries win on tier-name collision.
func EffectiveThresholds(m ManifestDefinition) []Threshold {
	merged := make(map[string]float64)
	for _, t := range DefaultThresholds() {
		merged[t.Tier] = t.MinValue
	}
	for _, t := range m.Thresholds {
		merged[t.Tier] = t.MinValue
	}
	out := make([]Threshold, 0, len(merged))
	for tier, v := range merged {
		out = append(out, Threshold{Tier: tier, MinValue: v})
	}
	// Stable order by descending MinValue (so auto_place comes before suggest).
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].MinValue > out[i].MinValue {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
