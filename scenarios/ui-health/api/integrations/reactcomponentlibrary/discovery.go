package reactcomponentlibrary

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// URLResolver returns the current react-component-library base URL.
// Implementations typically wrap api-core/discovery.ResolveScenarioURLDefault.
type URLResolver interface {
	Resolve() (string, error)
}

// ScenarioResolver resolves the react-component-library base URL via
// api-core/discovery, which consults the running lifecycle to find the
// scenario's API port. This is the production path.
type ScenarioResolver struct {
	Slug    string        // scenario slug, defaults to "react-component-library"
	Timeout time.Duration // per-resolve timeout, defaults to 5s
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
		slug = "react-component-library"
	}
	return discovery.ResolveScenarioURLDefault(ctx, slug)
}

// EnvResolver resolves the base URL from an environment variable. Used in
// development and tests where the lifecycle CLI isn't available.
type EnvResolver struct {
	EnvVar  string
	Default string
}

func (r EnvResolver) Resolve() (string, error) {
	if v := strings.TrimSpace(os.Getenv(r.EnvVar)); v != "" {
		return v, nil
	}
	if r.Default != "" {
		return r.Default, nil
	}
	return "", fmt.Errorf("reactcomponentlibrary: %s not set and no default provided", r.EnvVar)
}

// CachedResolver wraps another resolver with a TTL cache. Callers invoke
// Invalidate on transport failure to force the next Resolve to re-query
// (interop-steer §12 — captured URLs are short-lived).
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

// DefaultResolver returns a CachedResolver around ScenarioResolver with an
// EnvResolver fallback for the UI_HEALTH_RCL_URL escape hatch.
func DefaultResolver() *CachedResolver {
	return &CachedResolver{
		Inner: chainResolver{
			Primary:  ScenarioResolver{},
			Fallback: EnvResolver{EnvVar: "UI_HEALTH_RCL_URL"},
		},
	}
}

// chainResolver tries Primary first; on failure falls back to Fallback.
type chainResolver struct {
	Primary  URLResolver
	Fallback URLResolver
}

func (c chainResolver) Resolve() (string, error) {
	v, err := c.Primary.Resolve()
	if err == nil {
		return v, nil
	}
	if c.Fallback == nil {
		return "", err
	}
	if v, ferr := c.Fallback.Resolve(); ferr == nil {
		return v, nil
	}
	return "", err
}
