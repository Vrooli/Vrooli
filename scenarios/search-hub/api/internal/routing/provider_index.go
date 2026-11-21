package routing

import (
	"context"
	"encoding/hex"
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
	// Scores is the dense score for each retained provider id. It is diagnostic
	// metadata only; provider selection still consumes the ordered shortlist.
	Scores map[string]float64
}

// ProviderDescriptionIndex turns registered natural-language descriptions into
// a bounded classifier input. It is deliberately a read-only routing aid: the
// registry remains the source of truth and this index is rebuilt when a
// descriptor's identity or description changes.
type ProviderDescriptionIndex interface {
	Shortlist(ctx context.Context, request ProviderDescriptionQuery, profiles []ProviderProfile) ([]ProviderProfile, DescriptionIndexResult)
}

// ProviderDescriptionAttribution contains source-owned rights and quality
// evidence references. Digests bind the referenced bytes; the router never
// infers or manufactures these values from the query or provider registry.
type ProviderDescriptionAttribution struct {
	CallerID              string
	RightsEvidenceURI     string
	RightsEvidenceSHA256  string
	QualityEvidenceURI    string
	QualityEvidenceSHA256 string
}

// ProviderDescriptionIdentity is resolved by the owning caller/runtime
// composition. Tenant, operation, experiment arm, recipe, and evidence must
// come from trusted server state, not user query fields.
type ProviderDescriptionIdentity struct {
	OperationID           string
	TenantID              string
	ArmID                 string
	RecipeID              string
	RightsReceiptURI      string
	QualityEvidenceURI    string
	QualityEvidenceSHA256 string
}

// ProviderDescriptionQuery is the complete caller contract passed to a
// semantic provider-description index. Keeping identity and query together
// prevents an adapter from silently dropping tenant, rights, or quality
// attribution while forwarding the text.
type ProviderDescriptionQuery struct {
	OperationID string
	TenantID    string
	ArmID       string
	RecipeID    string
	Text        string
	Limit       int
	Attribution ProviderDescriptionAttribution
}

// ProviderSemanticContextResolver supplies owner-controlled operation,
// tenant, arm, recipe, and evidence identities for this request. CallerID is
// filled by the router from api-core's verified principal after this resolver
// succeeds. A missing resolver means semantic routing is unavailable.
type ProviderSemanticContextResolver interface {
	ResolveProviderSemanticContext(context.Context) (VerifiedProviderSemanticContext, error)
}

// ProviderRightsDecisionVerifier is implemented at the trusted owner boundary.
// Its implementation must call the canonical W2 rights-decision/v1 verifier
// against freshly resolved source facts and reject unknown or mismatched
// receipts. Evidence references alone are not proof.
type ProviderRightsDecisionVerifier interface {
	VerifyCurrentAllowedProviderRights(context.Context, ProviderDescriptionIdentity) (receiptSHA256 string, err error)
}

// VerifiedProviderSemanticContext can only be created after the canonical
// rights receipt has been rechecked against current owner facts and allowed.
// Its fields stay private so routing cannot mistake raw receipt bytes for a
// verified current decision.
type VerifiedProviderSemanticContext struct {
	identity          ProviderDescriptionIdentity
	rightsEvidenceSHA string
	verified          bool
}

// VerifyProviderSemanticContext is the construction path for an owner
// resolver. The resolver must obtain identity and current owner facts from
// trusted source state, then use a verifier backed by the canonical W2
// rights-decision/v1 API. The verifier must reject unknown or mismatched data.
func VerifyProviderSemanticContext(ctx context.Context, identity ProviderDescriptionIdentity, verifier ProviderRightsDecisionVerifier) (VerifiedProviderSemanticContext, error) {
	if strings.TrimSpace(identity.RightsReceiptURI) == "" || strings.TrimSpace(identity.QualityEvidenceURI) == "" || !isSHA256Hex(identity.QualityEvidenceSHA256) {
		return VerifiedProviderSemanticContext{}, fmt.Errorf("provider semantic evidence identity is incomplete")
	}
	if verifier == nil {
		return VerifiedProviderSemanticContext{}, fmt.Errorf("provider rights receipt verifier unavailable")
	}
	digest, err := verifier.VerifyCurrentAllowedProviderRights(ctx, identity)
	if err != nil {
		return VerifiedProviderSemanticContext{}, fmt.Errorf("embedding rights receipt is not current and allowed: %w", err)
	}
	if !isSHA256Hex(digest) {
		return VerifiedProviderSemanticContext{}, fmt.Errorf("provider rights verifier returned an invalid receipt digest")
	}
	return VerifiedProviderSemanticContext{identity: identity, rightsEvidenceSHA: strings.ToLower(digest), verified: true}, nil
}

type descriptionVector struct {
	ID          string                     `json:"id"`
	Description string                     `json:"description"`
	Fingerprint string                     `json:"fingerprint"`
	Vector      []float64                  `json:"vector"`
	Vectors     [][]float64                `json:"vectors,omitempty"`
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

func (i *EmbeddingDescriptionIndex) Shortlist(ctx context.Context, request ProviderDescriptionQuery, profiles []ProviderProfile) ([]ProviderProfile, DescriptionIndexResult) {
	query, limit := request.Text, request.Limit
	result := DescriptionIndexResult{Total: len(profiles), Scores: make(map[string]float64, len(profiles))}
	if limit <= 0 {
		limit = defaultMaxFanoutWidth
	}
	if !validProviderDescriptionQuery(request) {
		unavailable := unavailableDescriptionIndexResult(query, profiles, limit)
		unavailable.Reason = "routing_identity_unavailable"
		return fallbackDescriptionProfiles(query, profiles, limit, unavailable.Reason), unavailable
	}
	if i == nil || i.embedder == nil {
		return fallbackDescriptionProfiles(query, profiles, limit, "routing_index_unavailable"), unavailableDescriptionIndexResult(query, profiles, limit)
	}

	queryVector, err := embedDescriptionQuery(ctx, i.embedder, query)
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
		score := descriptionVectorScore(queryVector, entry)
		result.Scores[profile.ProviderID] = score
		scored = append(scored, scoredProfile{profile: profile, score: score})
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

func validProviderDescriptionQuery(request ProviderDescriptionQuery) bool {
	values := []string{
		request.OperationID, request.TenantID, request.ArmID, request.RecipeID,
		request.Text, request.Attribution.CallerID,
		request.Attribution.RightsEvidenceURI, request.Attribution.RightsEvidenceSHA256,
		request.Attribution.QualityEvidenceURI, request.Attribution.QualityEvidenceSHA256,
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return request.Limit > 0 && isSHA256Hex(request.Attribution.RightsEvidenceSHA256) && isSHA256Hex(request.Attribution.QualityEvidenceSHA256)
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
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
		if entry, ok := previous[profile.ProviderID]; ok && entry.Fingerprint == fingerprint && len(entry.Vectors) > 0 && embeddingMetadataCompatible(entry.Metadata, currentEmbeddingMetadata(entry.Vector)) {
			refreshed[profile.ProviderID] = entry
			continue
		}
		vectors := make([][]float64, 0)
		for _, text := range descriptionEmbeddingSegments(profile) {
			vector, err := embedDescriptionDocument(ctx, i.embedder, text)
			if err != nil || len(vector) == 0 {
				continue
			}
			vectors = append(vectors, vector)
		}
		if len(vectors) == 0 {
			continue
		}
		refreshed[profile.ProviderID] = descriptionVector{
			ID: profile.ProviderID, Description: profile.Description, Fingerprint: fingerprint,
			Vector: vectors[0], Vectors: vectors, Metadata: currentEmbeddingMetadata(vectors[0]),
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

func embedDescriptionQuery(ctx context.Context, embedder aisearch.Embedder, query string) ([]float64, error) {
	if taskEmbedder, ok := embedder.(aisearch.TaskEmbedder); ok {
		return taskEmbedder.EmbedQuery(ctx, query)
	}
	return embedder.Embed(ctx, query)
}

func embedDescriptionDocument(ctx context.Context, embedder aisearch.Embedder, document string) ([]float64, error) {
	if taskEmbedder, ok := embedder.(aisearch.TaskEmbedder); ok {
		return taskEmbedder.EmbedDocument(ctx, document)
	}
	return embedder.Embed(ctx, document)
}

// ProviderDescriptionSegments returns the same bounded, identity-enriched
// text units indexed by the live routing shortlist. The Search Hub fleet source
// uses this owner implementation when building a versioned Qdrant generation.
func ProviderDescriptionSegments(profile ProviderProfile) []string {
	return descriptionEmbeddingSegments(profile)
}

// EmbedProviderDescription embeds one source-owned description segment with
// the same document task prefix used by the live router index.
func EmbedProviderDescription(ctx context.Context, embedder aisearch.Embedder, document string) ([]float64, error) {
	if embedder == nil {
		return nil, fmt.Errorf("provider-description embedder is unavailable")
	}
	return embedDescriptionDocument(ctx, embedder, document)
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
	text := strings.ToLower(strings.Join([]string{
		profile.ProviderID,
		profile.Type,
		profile.Group,
		profile.Description,
		strings.Join(profile.AnswerSpaces, " "),
		strings.Join(profile.Intents, " "),
		strings.Join(profile.PositiveExamples, " "),
	}, " "))
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
		fmt.Fprintf(&b, "%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00", profile.ProviderID, profile.Type, profile.Group, profile.Description, strings.Join(profile.AnswerSpaces, "\x01"), strings.Join(profile.Intents, "\x01"), strings.Join(profile.PositiveExamples, "\x01"), strings.Join(profile.Exclusions, "\x01"))
	}
	return b.String()
}

// Version 5 records the task-prefixed Nomic recipe plus positive route-profile
// segments enriched with registry identity context. Bumping this value is
// intentional: vectors generated with the legacy symmetric recipe must never
// be compared with task-prefixed query vectors or prior representations after
// a restart.
const descriptionIndexPolicyVersion = "5"

// ProviderDescriptionFleetRecipeVersion is the immutable recipe component
// for the source-owned segment composer used by fleet generations.
const ProviderDescriptionFleetRecipeVersion = "provider-description-segments/v" + descriptionIndexPolicyVersion

// descriptionEmbeddingSegments keeps each registered routing description's
// distinct route examples and responsibilities searchable without storing
// provider-specific rules in the router. Each sentence carries the registry
// identity context so queries that name a provider family, leaf type, or
// scenario can match the same declared routing vocabulary used by the lexical
// fallback. A single long embedding can dilute a decisive sentence; scoring
// the best segment preserves the descriptor's natural-language intent while
// remaining bounded by the registry data.
func descriptionEmbeddingSegments(profile ProviderProfile) []string {
	identity := providerIdentityText(profile)
	if profileHasPositiveRouteProfile(profile) {
		segments := make([]string, 0, len(profile.AnswerSpaces)+len(profile.Intents)+len(profile.PositiveExamples))
		for _, value := range profile.AnswerSpaces {
			segments = append(segments, identity+"; answer space: "+value)
		}
		for _, value := range profile.Intents {
			segments = append(segments, identity+"; intent: "+value)
		}
		for _, value := range profile.PositiveExamples {
			segments = append(segments, identity+"; positive example: "+value)
		}
		return segments
	}
	description := strings.TrimSpace(profile.Description)
	if description == "" {
		return nil
	}
	parts := strings.FieldsFunc(description, func(r rune) bool {
		return r == '.' || r == '!' || r == '?' || r == '\n'
	})
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len([]rune(part)) >= 12 {
			segments = append(segments, part)
		}
	}
	if len(segments) == 0 {
		segments = []string{description}
	}
	if identity == "" {
		return segments
	}
	enriched := make([]string, 0, len(segments))
	for _, segment := range segments {
		enriched = append(enriched, identity+"; description: "+segment)
	}
	return enriched
}

func profileHasPositiveRouteProfile(profile ProviderProfile) bool {
	return len(profile.AnswerSpaces) > 0 || len(profile.Intents) > 0 || len(profile.PositiveExamples) > 0
}

func providerIdentityText(profile ProviderProfile) string {
	context := make([]string, 0, 3)
	if id := strings.TrimSpace(profile.ProviderID); id != "" {
		context = append(context, "provider: "+id)
	}
	if typ := strings.TrimSpace(profile.Type); typ != "" {
		context = append(context, "type: "+typ)
	}
	if group := strings.TrimSpace(profile.Group); group != "" {
		context = append(context, "group: "+group)
	}
	return strings.Join(context, "; ")
}

func descriptionVectorScore(query []float64, entry descriptionVector) float64 {
	best := cosine(query, entry.Vector)
	for _, vector := range entry.Vectors {
		if score := cosine(query, vector); score > best {
			best = score
		}
	}
	return best
}

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
