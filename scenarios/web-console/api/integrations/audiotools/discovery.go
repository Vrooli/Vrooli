package audiotools

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// ScenarioResolver resolves the audio-tools base URL via api-core/discovery,
// which consults the running lifecycle to find the audio-tools API port.
// Used in production when the audio-tools scenario is declared as a
// dependency in .vrooli/service.json.
type ScenarioResolver struct {
	Slug    string        // scenario slug, e.g., "audio-tools"
	Timeout time.Duration // per-resolve timeout (default 5s)
}

func (r ScenarioResolver) Resolve() (string, error) {
	timeout := r.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	slug := r.Slug
	if slug == "" {
		slug = "audio-tools"
	}
	return discovery.ResolveScenarioURLDefault(ctx, slug)
}

// EnvResolver resolves the audio-tools base URL from an environment variable.
// Used in development and tests. Production deployments use the
// api-core/discovery.ResolveScenarioURLDefault resolver instead.
type EnvResolver struct {
	EnvVar  string // e.g., "AUDIO_TOOLS_URL"
	Default string // e.g., "http://localhost:15000"
}

func (r EnvResolver) Resolve() (string, error) {
	if v := strings.TrimSpace(os.Getenv(r.EnvVar)); v != "" {
		return v, nil
	}
	if r.Default != "" {
		return r.Default, nil
	}
	return "", fmt.Errorf("audiotools: %s not set and no default provided", r.EnvVar)
}

// CachedResolver wraps another resolver with a TTL cache; on transport
// failure callers invoke Invalidate to force the next Resolve to re-query.
// Matches interoperability-steer §12 (short-lived captured URLs).
type CachedResolver struct {
	Inner URLResolver
	TTL   time.Duration

	mu        sync.Mutex
	value     string
	expiresAt time.Time
}

func (r *CachedResolver) Resolve() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.value != "" && time.Now().Before(r.expiresAt) {
		return r.value, nil
	}
	v, err := r.Inner.Resolve()
	if err != nil {
		return "", err
	}
	r.value = v
	ttl := r.TTL
	if ttl == 0 {
		ttl = 30 * time.Second
	}
	r.expiresAt = time.Now().Add(ttl)
	return v, nil
}

func (r *CachedResolver) Invalidate() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.value = ""
	r.expiresAt = time.Time{}
}

// PingContext is a lightweight readiness probe over the resolver, useful for
// capability-registry probes.
func (c *Client) PingContext(ctx context.Context) error {
	if err := c.Ensure(); err != nil {
		return err
	}
	return nil // Future expansion calls Health endpoint with timeout.
}
