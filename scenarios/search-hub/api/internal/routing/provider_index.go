package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
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
	Dropped   []string
}

// ProviderDescriptionIndex turns registered natural-language descriptions into
// a bounded classifier input. It is deliberately a read-only routing aid: the
// registry remains the source of truth and this index is rebuilt when a
// descriptor's identity or description changes.
type ProviderDescriptionIndex interface {
	Shortlist(ctx context.Context, query string, profiles []ProviderProfile, limit int) ([]ProviderProfile, DescriptionIndexResult)
}

type descriptionVector struct {
	ID          string                     `json:"id"`
	Description string                     `json:"description"`
	Fingerprint string                     `json:"fingerprint"`
	Vector      []float64                  `json:"vector"`
	Metadata    aisearch.EmbeddingMetadata `json:"metadata"`
}

// EmbeddingDescriptionIndex is the production in-process provider-description
// index. It uses the shared ai-go Embedder contract, caches description vectors
// by descriptor fingerprint, and embeds only the query on steady state. A
// failed embedding call never blocks search: it returns the full bounded
// fallback input with the stable routing_index_unavailable reason.
type EmbeddingDescriptionIndex struct {
	embedder  aisearch.Embedder
	mu        sync.Mutex
	entries   map[string]descriptionVector
	cachePath string
	loaded    bool
	dropped   []string
}

func NewEmbeddingDescriptionIndex(embedder aisearch.Embedder) *EmbeddingDescriptionIndex {
	return &EmbeddingDescriptionIndex{embedder: embedder, entries: make(map[string]descriptionVector)}
}

// NewPersistentEmbeddingDescriptionIndex retains the in-memory constructor's
// behavior while adding a durable cache for production. The cache contains
// only provider descriptions and embedding metadata, never query text.
func NewPersistentEmbeddingDescriptionIndex(embedder aisearch.Embedder, cachePath string) *EmbeddingDescriptionIndex {
	index := NewEmbeddingDescriptionIndex(embedder)
	index.cachePath = strings.TrimSpace(cachePath)
	return index
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
		return fallbackDescriptionProfiles(query, profiles, limit, "routing_index_unavailable"), unavailableDescriptionIndexResult(query, profiles, limit)
	}

	queryVector, err := i.embedder.Embed(ctx, query)
	if err != nil || len(queryVector) == 0 {
		return fallbackDescriptionProfiles(query, profiles, limit, "routing_index_unavailable"), unavailableDescriptionIndexResult(query, profiles, limit)
	}
	if err := i.refresh(ctx, profiles); err != nil {
		return fallbackDescriptionProfiles(query, profiles, limit, "routing_index_unavailable"), unavailableDescriptionIndexResult(query, profiles, limit)
	}

	i.mu.Lock()
	scored := make([]scoredProfile, 0, len(profiles))
	result.Dropped = append(result.Dropped, i.dropped...)
	for _, profile := range profiles {
		entry, ok := i.entries[profile.ProviderID]
		if !ok {
			continue
		}
		scored = append(scored, scoredProfile{profile: profile, score: cosine(queryVector, entry.Vector)})
	}
	i.mu.Unlock()
	if len(scored) == 0 {
		return fallbackDescriptionProfiles(query, profiles, limit, "routing_index_unavailable"), unavailableDescriptionIndexResult(query, profiles, limit)
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
	i.mu.Lock()
	if !i.loaded {
		i.loadLocked()
		i.loaded = true
	}
	previous := make(map[string]descriptionVector, len(i.entries))
	for id, entry := range i.entries {
		previous[id] = entry
	}
	i.mu.Unlock()

	refreshed := make(map[string]descriptionVector, len(profiles))
	for _, profile := range profiles {
		fingerprint := profileFingerprint([]ProviderProfile{profile})
		if entry, ok := previous[profile.ProviderID]; ok && entry.Fingerprint == fingerprint && embeddingMetadataCompatible(entry.Metadata, currentEmbeddingMetadata(entry.Vector)) {
			refreshed[profile.ProviderID] = entry
			continue
		}
		text := strings.TrimSpace(strings.Join([]string{profile.ProviderID, profile.Type, profile.Group, profile.Description}, " — "))
		vector, err := i.embedder.Embed(ctx, text)
		if err != nil || len(vector) == 0 {
			continue
		}
		refreshed[profile.ProviderID] = descriptionVector{
			ID: profile.ProviderID, Description: profile.Description, Fingerprint: fingerprint,
			Vector: vector, Metadata: currentEmbeddingMetadata(vector),
		}
	}
	i.mu.Lock()
	i.entries = refreshed
	i.dropped = nil
	for _, profile := range profiles {
		if _, ok := refreshed[profile.ProviderID]; !ok {
			i.dropped = append(i.dropped, profile.ProviderID)
		}
	}
	i.persistLocked()
	i.mu.Unlock()
	return nil
}

func fallbackDescriptionProfiles(query string, profiles []ProviderProfile, limit int, _ string) []ProviderProfile {
	scored := make([]scoredProfile, 0, len(profiles))
	for _, profile := range profiles {
		scored = append(scored, scoredProfile{profile: profile, score: lexicalDescriptionScore(query, profile)})
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
	out := make([]ProviderProfile, 0, len(scored))
	for _, item := range scored {
		out = append(out, item.profile)
	}
	return out
}

func unavailableDescriptionIndexResult(query string, profiles []ProviderProfile, limit int) DescriptionIndexResult {
	selected := fallbackDescriptionProfiles(query, profiles, limit, "routing_index_unavailable")
	result := DescriptionIndexResult{Available: false, Reason: "routing_index_unavailable", Total: len(profiles), Returned: len(selected)}
	seen := make(map[string]struct{}, len(selected))
	for _, profile := range selected {
		seen[profile.ProviderID] = struct{}{}
	}
	for _, profile := range profiles {
		if _, ok := seen[profile.ProviderID]; !ok {
			result.Omitted = append(result.Omitted, profile.ProviderID)
		}
	}
	return result
}

func lexicalDescriptionScore(query string, profile ProviderProfile) float64 {
	terms := strings.Fields(strings.ToLower(query))
	text := strings.ToLower(strings.Join([]string{profile.ProviderID, profile.Type, profile.Group, profile.Description}, " "))
	var score float64
	for _, term := range terms {
		term = strings.Trim(term, " ,.!?:;()[]{}\"'")
		if term != "" && strings.Contains(text, term) {
			score++
		}
	}
	return score
}

func profileFingerprint(profiles []ProviderProfile) string {
	var b strings.Builder
	for _, profile := range profiles {
		fmt.Fprintf(&b, "%s\x00%s\x00%s\x00%s\x00", profile.ProviderID, profile.Type, profile.Group, profile.Description)
	}
	return b.String()
}

const descriptionIndexPolicyVersion = "1"

func currentEmbeddingMetadata(vector []float64) aisearch.EmbeddingMetadata {
	model := strings.TrimSpace(os.Getenv("SEARCH_HUB_DESCRIPTION_EMBED_MODEL"))
	if model == "" {
		model = aisearch.DefaultEmbedModel
	}
	return aisearch.EmbeddingMetadata{Role: "search-hub.provider-description", Model: model, Dimensions: len(vector), PolicySchemaVersion: descriptionIndexPolicyVersion}
}

func embeddingMetadataCompatible(stored, current aisearch.EmbeddingMetadata) bool {
	return stored.Role == current.Role && stored.Model == current.Model && stored.Dimensions == current.Dimensions && stored.PolicySchemaVersion == current.PolicySchemaVersion
}

func (i *EmbeddingDescriptionIndex) loadLocked() {
	if i.cachePath == "" {
		return
	}
	blob, err := os.ReadFile(i.cachePath)
	if err != nil {
		return
	}
	var entries map[string]descriptionVector
	if json.Unmarshal(blob, &entries) == nil {
		for id, entry := range entries {
			if embeddingMetadataCompatible(entry.Metadata, currentEmbeddingMetadata(entry.Vector)) {
				i.entries[id] = entry
			}
		}
	}
}

func (i *EmbeddingDescriptionIndex) persistLocked() {
	if i.cachePath == "" {
		return
	}
	blob, err := json.Marshal(i.entries)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(i.cachePath), 0o755); err != nil {
		return
	}
	tmp := i.cachePath + ".tmp"
	if err := os.WriteFile(tmp, blob, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, i.cachePath)
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
