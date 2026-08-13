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

// defaultMaxFanoutWidth bounds automatic routing after the classifier is
// uncertain. It is intentionally aligned with the router's normal worker
// budget: widening beyond one bounded wave spends latency without improving
// the recall signal that the federated eval tier measures.
const defaultMaxFanoutWidth = 6

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

// defaultAutoExternalThreshold is the confidence floor a web-shaped query must
// clear before the router folds SCOPE_EXTERNAL providers into automatic routing
// (OT-P2-002). It is deliberately HIGHER than the widen threshold: opting into a
// rate-limited / paid external corpus should require real confidence, not the
// over-fetch-on-uncertainty default that governs project-scope widening.
const defaultAutoExternalThreshold = 0.60

// autoExternalThreshold resolves the web-shaped confidence floor for external
// auto-routing, honoring an env override.
func autoExternalThreshold() float64 {
	if raw := strings.TrimSpace(os.Getenv("SEARCH_HUB_AUTO_EXTERNAL_THRESHOLD")); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil && v >= 0 && v <= 1 {
			return v
		}
	}
	return defaultAutoExternalThreshold
}

func maxFanoutWidth() int {
	if raw := strings.TrimSpace(os.Getenv("SEARCH_HUB_CLASSIFIER_MAX_FANOUT_WIDTH")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 && v <= 128 {
			return v
		}
	}
	return defaultMaxFanoutWidth
}

// ProviderProfile is the registry-derived view of one routable provider leaf
// that the classifier reasons over. The router builds these from ACTIVE
// descriptors and the classifier routes purely on the natural-language
// Description — never on hardcoded provider knowledge.
type ProviderProfile struct {
	ProviderID  string
	Type        string
	Group       string
	Description string
	// OmittedProviderIDs is populated only on the first shortlisted profile.
	// It is prompt metadata, not a candidate: the classifier must not invent
	// ids, but the prompt should name every bounded-fallback omission.
	OmittedProviderIDs []string
}

// ClassifyResult is the classifier's routing decision over a query.
type ClassifyResult struct {
	// ProviderIDs is the ranked set of provider leaf ids the query should route
	// to (most-relevant first). It may name ids that do not exist; the router
	// intersects it with the live registry before fan-out.
	ProviderIDs []string
	// Types is retained only for in-process compatibility with callers compiled
	// against the pre-leaf classifier seam. New classifier responses populate
	// ProviderIDs; widenPolicy interprets a legacy token as a type only when it
	// is not an exact provider id.
	Types []string
	// Confidence is the classifier's self-reported confidence in [0,1]. Below
	// classifierWidenThreshold the router widens (see widenPolicy).
	Confidence float64
	// Rationale is a short human-readable explanation surfaced by --explain.
	Rationale string
	// WebShaped is the classifier's GENERIC judgment that the query wants fresh,
	// live, external-web information (e.g. current events, "latest", real-time
	// facts) rather than only the project corpus. It is a property of the QUERY,
	// not of any specific provider — the router uses it (when the operator has
	// opted in) to fold SCOPE_EXTERNAL providers back into automatic routing,
	// without the router ever knowing which provider is "the web". This keeps the
	// thin-router invariant intact (OT-P2-002).
	WebShaped bool
}

// Classifier maps a free-text query to provider leaf ids it should route to,
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

// buildProfiles emits one profile per active provider leaf. Output is sorted by
// provider id so prompts stay deterministic and every registrant's description
// remains available to the classifier without a type-level collapse.
func buildProfiles(active []*registryv1.ProviderDescriptor) []ProviderProfile {
	out := make([]ProviderProfile, 0, len(active))
	for _, p := range active {
		if p.GetEndpoint() == nil {
			continue // not callable (defensive; List already filters to ACTIVE)
		}
		id := strings.TrimSpace(p.GetProviderId())
		if id == "" || strings.TrimSpace(p.GetType()) == "" {
			continue
		}
		out = append(out, ProviderProfile{
			ProviderID:  id,
			Type:        strings.TrimSpace(p.GetType()),
			Group:       strings.TrimSpace(p.GetProviderGroup()),
			Description: strings.TrimSpace(p.GetDescription()),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProviderID < out[j].ProviderID })
	return out
}

func availableProviderIDs(profiles []ProviderProfile) []string {
	ids := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		ids = append(ids, profile.ProviderID)
	}
	return ids
}

// widenPolicy turns a raw classifier result into concrete provider leaves.
// Confident results retain only the selected leaves. Uncertain results add
// siblings that share the selected leaf's type, bounded by maxWidth; this
// preserves recall without silently reverting to fleet-wide fan-out.
func widenPolicy(result ClassifyResult, profiles []ProviderProfile, maxWidth int) (chosen []string, widened, boundReached bool) {
	if maxWidth <= 0 {
		maxWidth = defaultMaxFanoutWidth
	}
	byID := make(map[string]ProviderProfile, len(profiles))
	for _, profile := range profiles {
		byID[profile.ProviderID] = profile
	}
	selected := make([]string, 0, len(result.ProviderIDs))
	seen := make(map[string]struct{}, len(result.ProviderIDs))
	rawIDs := result.ProviderIDs
	if len(rawIDs) == 0 {
		rawIDs = result.Types
	}
	for _, id := range rawIDs {
		id = strings.TrimSpace(id)
		if _, ok := byID[id]; ok {
			if _, duplicate := seen[id]; duplicate {
				continue
			}
			seen[id] = struct{}{}
			selected = append(selected, id)
			continue
		}
		// Legacy type tokens are accepted only as a compatibility bridge. The
		// production prompt and wire contract use exact provider ids.
		for _, profile := range profiles {
			if profile.Type != id {
				continue
			}
			if _, duplicate := seen[profile.ProviderID]; duplicate {
				continue
			}
			seen[profile.ProviderID] = struct{}{}
			selected = append(selected, profile.ProviderID)
		}
	}

	if result.Confidence >= widenThreshold() && len(selected) > 0 {
		chosen = append([]string(nil), selected...)
		if len(chosen) > maxWidth {
			chosen = chosen[:maxWidth]
			boundReached = true
		}
		return chosen, false, boundReached
	}

	// No usable selection has no sibling anchor. Use a deterministic bounded
	// fallback so classifier uncertainty and malformed model output cannot turn
	// into an unbounded route.
	if len(selected) == 0 {
		for _, profile := range profiles {
			selected = append(selected, profile.ProviderID)
			if len(selected) == maxWidth {
				break
			}
		}
		return selected, true, len(profiles) > len(selected)
	}

	chosen = append(chosen, selected...)
	types := make(map[string]struct{}, len(selected))
	for _, id := range selected {
		types[byID[id].Type] = struct{}{}
	}
	siblingTotal := len(selected)
	for _, profile := range profiles {
		if _, sibling := types[profile.Type]; sibling {
			if _, already := seen[profile.ProviderID]; !already {
				siblingTotal++
			}
		}
	}
	boundReached = siblingTotal > maxWidth
	for _, profile := range profiles {
		if _, sibling := types[profile.Type]; !sibling {
			continue
		}
		if _, already := seen[profile.ProviderID]; already {
			continue
		}
		chosen = append(chosen, profile.ProviderID)
		seen[profile.ProviderID] = struct{}{}
		if len(chosen) == maxWidth {
			break
		}
	}
	return chosen, true, boundReached
}
