package routing

import (
	"context"
	"os"
	"sort"
	"strconv"
	"strings"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// defaultWidenThreshold is the confidence floor below which the router broadens
// the classifier's chosen type set rather than narrowing it — the make-or-break
// "widen on uncertainty" policy (plan §4 Risk): a wrong-narrow route silently
// drops results, a wrong-broad route only adds noise the reranker (Phase 6)
// removes, so when the classifier is unsure we over-fetch.
//
// 0.45 is empirically calibrated to qwen3:1.7b, whose self-reported confidence
// clusters in a narrow 0.40–0.50 band: a focused, confident selection lands at
// ~0.45–0.50 (honored), while genuinely unsure answers sit at ~0.40 (widened).
// Override with SEARCH_HUB_CLASSIFIER_WIDEN_THRESHOLD when swapping models whose
// calibration differs (the seam exists because small-model confidence is poorly
// calibrated and a true cross-encoder, Phase 6, will recalibrate this).
const defaultWidenThreshold = 0.45

// widenThreshold resolves the confidence floor, honoring the env override and
// falling back to defaultWidenThreshold for an unset/invalid/out-of-range value.
func widenThreshold() float64 {
	if raw := strings.TrimSpace(os.Getenv("SEARCH_HUB_CLASSIFIER_WIDEN_THRESHOLD")); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil && v >= 0 && v <= 1 {
			return v
		}
	}
	return defaultWidenThreshold
}

// ProviderProfile is the registry-derived view of one routable leaf type that
// the classifier reasons over. The router builds these from ACTIVE descriptors
// and the classifier routes purely on the natural-language Description — never
// on hardcoded provider knowledge (plan §3.2: "the router routes on
// descriptions"). Many leaves can share a type; Description then joins theirs.
type ProviderProfile struct {
	Type        string
	Group       string
	Description string
}

// ClassifyResult is the classifier's routing decision over a query.
type ClassifyResult struct {
	// Types is the ranked set of provider `type` tokens the query should route
	// to (most-relevant first). It may name types that do not exist; the router
	// intersects it with the live registry before fan-out.
	Types []string
	// Confidence is the classifier's self-reported confidence in [0,1]. Below
	// classifierWidenThreshold the router widens (see widenPolicy).
	Confidence float64
	// Rationale is a short human-readable explanation surfaced by --explain.
	Rationale string
}

// Classifier maps a free-text query to the provider `type`s it should route to,
// reading only the registered provider descriptions. It is a swappable seam
// (production: local Ollama qwen3:1.7b via OllamaClassifier; tests: a
// deterministic fake) so the router stays unit-testable without a model.
type Classifier interface {
	// Classify returns the ranked types to route to for query, given the
	// available provider profiles. An error means the classifier could not
	// decide (model down, bad output); the router treats that as "widen to all"
	// + degraded rather than failing the query.
	Classify(ctx context.Context, query string, profiles []ProviderProfile) (ClassifyResult, error)
	// Available reports whether the backing model is reachable. The hot query
	// path does not call this (it relies on Classify's error for degradation);
	// it exists for the Phase 7 Status surface.
	Available(ctx context.Context) bool
}

// buildProfiles collapses the active provider descriptors into one profile per
// distinct type (the classifier routes on types, not individual leaves). When
// several leaves share a type their descriptions are joined so the classifier
// sees the full corpus picture for that type. Output is type-sorted for
// deterministic prompts.
func buildProfiles(active []*registryv1.ProviderDescriptor) []ProviderProfile {
	byType := make(map[string]*ProviderProfile)
	for _, p := range active {
		if p.GetEndpoint() == nil {
			continue // not callable (defensive; List already filters to ACTIVE)
		}
		t := strings.TrimSpace(p.GetType())
		if t == "" {
			continue
		}
		prof, ok := byType[t]
		if !ok {
			byType[t] = &ProviderProfile{Type: t, Group: p.GetProviderGroup(), Description: strings.TrimSpace(p.GetDescription())}
			continue
		}
		if d := strings.TrimSpace(p.GetDescription()); d != "" {
			if prof.Description == "" {
				prof.Description = d
			} else {
				prof.Description += " " + d
			}
		}
	}
	out := make([]ProviderProfile, 0, len(byType))
	for _, prof := range byType {
		out = append(out, *prof)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}

// availableTypes returns the distinct, type-sorted set of routable types in the
// active set — the universe the classifier's answer is intersected against and
// the fallback "widen to everything" set.
func availableTypes(active []*registryv1.ProviderDescriptor) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(active))
	for _, p := range active {
		if p.GetEndpoint() == nil {
			continue
		}
		t := strings.TrimSpace(p.GetType())
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// widenPolicy turns a raw ClassifyResult into the concrete type set the router
// fans out to, applying widen-on-uncertainty. It intersects the classifier's
// chosen types with what actually exists, then — if the classifier was
// unconfident or nothing matched — broadens to every available type. It returns
// the chosen types (type-sorted) and whether widening kicked in (for --explain).
func widenPolicy(result ClassifyResult, available []string) (chosen []string, widened bool) {
	avail := make(map[string]struct{}, len(available))
	for _, t := range available {
		avail[t] = struct{}{}
	}
	seen := make(map[string]struct{})
	for _, t := range result.Types {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := avail[t]; !ok {
			continue // classifier named a type no provider serves — drop it
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		chosen = append(chosen, t)
	}
	if result.Confidence < widenThreshold() || len(chosen) == 0 {
		// Uncertain (or no usable match): over-fetch across every type and let
		// the reranker (Phase 6) cut the noise. Recall over precision.
		return append([]string(nil), available...), true
	}
	sort.Strings(chosen)
	return chosen, false
}
