// Package codecs — ollama.go provides a cached, injectable lister for the
// locally-pulled Ollama models so codecs can advertise them honestly in
// Capabilities().SupportedModels.
//
// Capabilities() is on the hot path (flag validation calls it per run), so
// the lister caches the result for a short TTL: the first call (or the first
// after the TTL elapses) shells out once to the Ollama probe SSOT
// (`resource-ollama models list --json`) — the single authority for Ollama
// model discovery — and every call in between returns the cached slice. When
// the SSOT is unreachable (CLI absent / daemon down) the lister degrades to
// the last-known list (empty on a cold miss) and never hard-fails — local
// models simply don't surface until the probe answers. This mirrors the
// existing exec-to-resource-ollama pattern in
// adapters/recommendation/ollama_extractor.go; no raw /api/tags HTTP path
// lives here anymore.
//
// Model ids are emitted in the uniform `ollama/<name>` form used by both the
// codex codec (which strips the prefix and drives `--oss --local-provider
// ollama`) and the opencode codec (whose provider block resolves the same
// slug). Tests inject a stub fetcher so no subprocess is required.
package codecs

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

// ollamaModelPrefix is the uniform provider prefix used for locally-served
// Ollama models across the codecs.
const ollamaModelPrefix = "ollama/"

// ollamaListTTL bounds how often the lister shells out to the probe SSOT.
const ollamaListTTL = 60 * time.Second

// ollamaProbeTimeout bounds a single `resource-ollama models list` probe so a
// slow or wedged daemon never stalls the Capabilities() hot path. Slightly
// larger than the old direct-HTTP budget to absorb subprocess spawn.
const ollamaProbeTimeout = 5 * time.Second

// ollamaSSOTCommand is the probe SSOT binary every Ollama discovery flows
// through (resolved on PATH, as in ollama_extractor.go).
const ollamaSSOTCommand = "resource-ollama"

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

// newOllamaLister returns a lister that discovers installed models by shelling
// out to the probe SSOT (`resource-ollama models list --json`); it never opens
// a daemon HTTP connection itself.
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

// defaultOllamaFetch shells out once to the probe SSOT
// (`resource-ollama models list --json`) and returns the pulled models as
// `ollama/<name>` ids, sorted for deterministic advertisement. Mirrors
// ollama_extractor.go's exec discipline: a bounded context, JSON decode that
// tolerates unknown fields, and graceful failure (the caller degrades to the
// last-known list — never a crash). It is intentionally the ONLY Ollama
// discovery path in agent-manager.
func defaultOllamaFetch(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, ollamaSSOTCommand, "models", "list", "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseOllamaListJSON(out)
}

// parseOllamaListJSON decodes the `resource-ollama models list --json` payload
// ({"models":["gemma4:12b",...]}) into sorted `ollama/<name>` ids. Unknown
// fields are tolerated (DiscardUnknown discipline).
func parseOllamaListJSON(data []byte) ([]string, error) {
	var payload struct {
		Models []string `json:"models"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(data), &payload); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(payload.Models))
	for _, name := range payload.Models {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out = append(out, ollamaModelPrefix+name)
	}
	sort.Strings(out)
	return out, nil
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
