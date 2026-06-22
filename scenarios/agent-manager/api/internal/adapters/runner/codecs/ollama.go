// Package codecs — ollama.go provides a cached, injectable lister for the
// locally-pulled Ollama models so codecs can advertise them honestly in
// Capabilities().SupportedModels.
//
// Capabilities() is on the hot path (flag validation calls it per run), so
// the lister caches the /api/tags result for a short TTL: the first call (or
// the first after the TTL elapses) performs one bounded HTTP probe, and every
// call in between returns the cached slice. When Ollama is unreachable the
// lister degrades to the last-known list (empty on a cold miss) and never
// hard-fails — local models simply don't surface until the daemon answers.
//
// Model ids are emitted in the uniform `ollama/<name>` form used by both the
// codex codec (which strips the prefix and drives `--oss --local-provider
// ollama`) and the opencode codec (whose provider block resolves the same
// slug). Tests inject a stub fetcher so no network access is required.
package codecs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// ollamaModelPrefix is the uniform provider prefix used for locally-served
// Ollama models across the codecs.
const ollamaModelPrefix = "ollama/"

// ollamaListTTL bounds how often the lister probes /api/tags.
const ollamaListTTL = 60 * time.Second

// ollamaProbeTimeout bounds a single /api/tags probe so a slow or wedged
// daemon never stalls the Capabilities() hot path.
const ollamaProbeTimeout = 2 * time.Second

// ollamaLister discovers locally-pulled Ollama models with a TTL cache.
// The zero value is not usable; construct via newOllamaLister.
type ollamaLister struct {
	ttl   time.Duration
	fetch func(ctx context.Context) ([]string, error)
	nowFn func() time.Time

	mu        sync.Mutex
	cached    []string
	fetchedAt time.Time
}

// newOllamaLister returns a lister that probes the local Ollama daemon's
// /api/tags endpoint (resolved from OLLAMA_HOST, defaulting to
// http://localhost:11434).
func newOllamaLister() *ollamaLister {
	l := &ollamaLister{ttl: ollamaListTTL, nowFn: time.Now}
	l.fetch = defaultOllamaFetch
	return l
}

// newOllamaListerForTest returns a lister with an injected fetcher and clock
// so tests exercise the cache/degrade behaviour without network access.
func newOllamaListerForTest(fetch func(ctx context.Context) ([]string, error), ttl time.Duration, nowFn func() time.Time) *ollamaLister {
	return &ollamaLister{ttl: ttl, fetch: fetch, nowFn: nowFn}
}

// list returns the cached `ollama/<name>` model ids, refreshing once the TTL
// has elapsed. It always returns a fresh copy and never errors: a failed
// probe degrades to the last-known list (empty on a cold miss).
func (l *ollamaLister) list() []string {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.nowFn()
	if l.fetchedAt.IsZero() || now.Sub(l.fetchedAt) >= l.ttl {
		ctx, cancel := context.WithTimeout(context.Background(), ollamaProbeTimeout)
		defer cancel()
		models, err := l.fetch(ctx)
		// Stamp the attempt regardless of outcome so a down daemon is probed
		// at most once per TTL rather than on every Capabilities() call.
		l.fetchedAt = now
		if err == nil {
			l.cached = models
		} else if l.cached == nil {
			l.cached = []string{}
		}
	}

	out := make([]string, len(l.cached))
	copy(out, l.cached)
	return out
}

// defaultOllamaFetch performs one bounded GET against the local Ollama
// daemon's /api/tags endpoint and returns the pulled models as
// `ollama/<name>` ids, sorted for deterministic advertisement.
func defaultOllamaFetch(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ollamaBaseURL()+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama /api/tags: %s", resp.Status)
	}

	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	out := make([]string, 0, len(payload.Models))
	for _, m := range payload.Models {
		name := strings.TrimSpace(m.Name)
		if name == "" {
			continue
		}
		out = append(out, ollamaModelPrefix+name)
	}
	sort.Strings(out)
	return out, nil
}

// ollamaBaseURL resolves the Ollama daemon base URL from OLLAMA_HOST,
// tolerating a bare host (no scheme) or a host without a port — mirroring the
// resource-side `ollama::base_url` bash helper so Go and shell agree.
func ollamaBaseURL() string {
	const fallback = "http://localhost:11434"
	raw := strings.TrimSpace(os.Getenv("OLLAMA_HOST"))
	if raw == "" {
		return fallback
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fallback
	}
	if u.Port() == "" {
		u.Host = u.Host + ":11434"
	}
	return u.Scheme + "://" + u.Host
}

// splitOllamaModel reports whether a model id targets the local Ollama
// provider and returns the bare model name (prefix stripped) to hand to the
// upstream CLI's `-m` flag.
func splitOllamaModel(model string) (bare string, isOllama bool) {
	model = strings.TrimSpace(model)
	if strings.HasPrefix(model, ollamaModelPrefix) {
		return strings.TrimPrefix(model, ollamaModelPrefix), true
	}
	return model, false
}
