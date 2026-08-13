package routing

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	aisearch "github.com/vrooli/ai-go/search"
)

// DescriptionIndexResult describes the routing index decision. Omitted is
// intentionally explicit: operators can see which providers were not sent to
// the classifier when the bounded shortlist is active.
type DescriptionIndexResult struct {
	Available bool
	Reason    string
	Total     int
	Returned  int
	Omitted   []string
}

// ProviderDescriptionIndex turns registered natural-language descriptions into
// a bounded classifier input. It is deliberately a read-only routing aid: the
// registry remains the source of truth and this index is rebuilt when a
// descriptor's identity or description changes.
type ProviderDescriptionIndex interface {
	Shortlist(ctx context.Context, query string, profiles []ProviderProfile, limit int) ([]ProviderProfile, DescriptionIndexResult)
}

type descriptionVector struct {
	id          string
	description string
	vector      []float64
}

// EmbeddingDescriptionIndex is the production in-process provider-description
// index. It uses the shared ai-go Embedder contract, caches description vectors
// by descriptor fingerprint, and embeds only the query on steady state. A
// failed embedding call never blocks search: it returns the full bounded
// fallback input with the stable routing_index_unavailable reason.
type EmbeddingDescriptionIndex struct {
	embedder aisearch.Embedder
	mu       sync.Mutex
	entries  map[string]descriptionVector
	version  string
}

func NewEmbeddingDescriptionIndex(embedder aisearch.Embedder) *EmbeddingDescriptionIndex {
	return &EmbeddingDescriptionIndex{embedder: embedder, entries: make(map[string]descriptionVector)}
}

func (i *EmbeddingDescriptionIndex) Shortlist(ctx context.Context, query string, profiles []ProviderProfile, limit int) ([]ProviderProfile, DescriptionIndexResult) {
	result := DescriptionIndexResult{Total: len(profiles)}
	if limit <= 0 {
		limit = defaultMaxFanoutWidth
	}
	if len(profiles) <= limit {
		result.Available = true
		result.Returned = len(profiles)
		return append([]ProviderProfile(nil), profiles...), result
	}
	if i == nil || i.embedder == nil {
		return fallbackDescriptionProfiles(profiles, limit, "routing_index_unavailable"), unavailableDescriptionIndexResult(profiles, limit)
	}

	queryVector, err := i.embedder.Embed(ctx, query)
	if err != nil || len(queryVector) == 0 {
		return fallbackDescriptionProfiles(profiles, limit, "routing_index_unavailable"), unavailableDescriptionIndexResult(profiles, limit)
	}
	if err := i.refresh(ctx, profiles); err != nil {
		return fallbackDescriptionProfiles(profiles, limit, "routing_index_unavailable"), unavailableDescriptionIndexResult(profiles, limit)
	}

	i.mu.Lock()
	scored := make([]scoredProfile, 0, len(profiles))
	for _, profile := range profiles {
		entry, ok := i.entries[profile.ProviderID]
		if !ok {
			continue
		}
		scored = append(scored, scoredProfile{profile: profile, score: cosine(queryVector, entry.vector)})
	}
	i.mu.Unlock()
	if len(scored) == 0 {
		return fallbackDescriptionProfiles(profiles, limit, "routing_index_unavailable"), unavailableDescriptionIndexResult(profiles, limit)
	}
	sort.SliceStable(scored, func(a, b int) bool {
		if scored[a].score == scored[b].score {
			return scored[a].profile.ProviderID < scored[b].profile.ProviderID
		}
		return scored[a].score > scored[b].score
	})
	if len(scored) > limit {
		scored = scored[:limit]
	}
	shortlist := make([]ProviderProfile, 0, len(scored))
	selected := make(map[string]struct{}, len(scored))
	for _, item := range scored {
		shortlist = append(shortlist, item.profile)
		selected[item.profile.ProviderID] = struct{}{}
	}
	result.Available = true
	result.Returned = len(shortlist)
	for _, profile := range profiles {
		if _, ok := selected[profile.ProviderID]; !ok {
			result.Omitted = append(result.Omitted, profile.ProviderID)
		}
	}
	return shortlist, result
}

type scoredProfile struct {
	profile ProviderProfile
	score   float64
}

func (i *EmbeddingDescriptionIndex) refresh(ctx context.Context, profiles []ProviderProfile) error {
	fingerprint := profileFingerprint(profiles)
	i.mu.Lock()
	if fingerprint == i.version {
		i.mu.Unlock()
		return nil
	}
	i.mu.Unlock()

	refreshed := make(map[string]descriptionVector, len(profiles))
	for _, profile := range profiles {
		text := strings.TrimSpace(strings.Join([]string{profile.ProviderID, profile.Type, profile.Group, profile.Description}, " — "))
		vector, err := i.embedder.Embed(ctx, text)
		if err != nil || len(vector) == 0 {
			return fmt.Errorf("embed provider description %q: %w", profile.ProviderID, err)
		}
		refreshed[profile.ProviderID] = descriptionVector{id: profile.ProviderID, description: profile.Description, vector: vector}
	}
	i.mu.Lock()
	i.entries = refreshed
	i.version = fingerprint
	i.mu.Unlock()
	return nil
}

func fallbackDescriptionProfiles(profiles []ProviderProfile, limit int, _ string) []ProviderProfile {
	if len(profiles) > limit {
		profiles = profiles[:limit]
	}
	out := append([]ProviderProfile(nil), profiles...)
	return out
}

func unavailableDescriptionIndexResult(profiles []ProviderProfile, limit int) DescriptionIndexResult {
	result := DescriptionIndexResult{Available: false, Reason: "routing_index_unavailable", Total: len(profiles), Returned: minInt(len(profiles), limit)}
	for _, profile := range profiles[result.Returned:] {
		result.Omitted = append(result.Omitted, profile.ProviderID)
	}
	return result
}

func profileFingerprint(profiles []ProviderProfile) string {
	var b strings.Builder
	for _, profile := range profiles {
		fmt.Fprintf(&b, "%s\x00%s\x00%s\x00%s\x00", profile.ProviderID, profile.Type, profile.Group, profile.Description)
	}
	return b.String()
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return -1
	}
	var dot, an, bn float64
	for n := range a {
		dot += a[n] * b[n]
		an += a[n] * a[n]
		bn += b[n] * b[n]
	}
	if an == 0 || bn == 0 {
		return -1
	}
	return dot / (math.Sqrt(an) * math.Sqrt(bn))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
