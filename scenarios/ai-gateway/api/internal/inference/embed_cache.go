package inference

import (
	"time"

	"ai-gateway/internal/rolecatalog"
)

// defaultEmbeddingAuthorizationTTL is how long a candidate's resolution and
// rights authorization is reused by embedding requests. It is short because
// the resource re-verifies the exact policy, model and rights evidence on every
// call; the cache only removes the gateway-side re-resolution spawns.
const defaultEmbeddingAuthorizationTTL = 10 * time.Second

type embeddingCacheEntry struct {
	value   embeddingCandidate
	expires time.Time
}

// embeddingCacheKey separates what changes the authorization outcome:
// the candidate, the locality profile and the privacy class.
func embeddingCacheKey(request EmbeddingRequest, candidate rolecatalog.Candidate) string {
	return candidate.Provider + "|" + candidate.ResourceRole + "|" + request.Profile.String() + "|" + request.PrivacyClass.String()
}

func (r *ResourceRepository) embeddingTTL() time.Duration {
	if r.authorizationTTL != nil {
		return *r.authorizationTTL
	}
	return defaultEmbeddingAuthorizationTTL
}

func (r *ResourceRepository) now() time.Time {
	if r.clock != nil {
		return r.clock()
	}
	return time.Now()
}

func (r *ResourceRepository) cachedEmbeddingCandidate(key string) (embeddingCandidate, bool) {
	r.embedMu.Lock()
	defer r.embedMu.Unlock()
	entry, ok := r.embedCache[key]
	if !ok {
		return embeddingCandidate{}, false
	}
	if !r.now().Before(entry.expires) {
		delete(r.embedCache, key)
		return embeddingCandidate{}, false
	}
	return entry.value, true
}

func (r *ResourceRepository) storeEmbeddingCandidate(key string, value embeddingCandidate) {
	ttl := r.embeddingTTL()
	if ttl <= 0 {
		return
	}
	r.embedMu.Lock()
	defer r.embedMu.Unlock()
	if r.embedCache == nil {
		r.embedCache = map[string]embeddingCacheEntry{}
	}
	r.embedCache[key] = embeddingCacheEntry{value: value, expires: r.now().Add(ttl)}
}

// forgetEmbeddingCandidate drops every cached authorization of a candidate.
func (r *ResourceRepository) forgetEmbeddingCandidate(candidate rolecatalog.Candidate) {
	r.embedMu.Lock()
	defer r.embedMu.Unlock()
	prefix := candidate.Provider + "|" + candidate.ResourceRole + "|"
	for key := range r.embedCache {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(r.embedCache, key)
		}
	}
}
