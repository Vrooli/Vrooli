package audiotools

import (
	"context"

	"github.com/vrooli/api-core/discovery"
)

// URLResolver resolves the base URL of the audio-tools API at call
// time. A typical adopter wires DefaultResolver() in production and
// passes a fixed-URL resolver in tests.
type URLResolver interface {
	ResolveURL(ctx context.Context) (string, error)
}

type defaultResolver struct{}

func (defaultResolver) ResolveURL(ctx context.Context) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, "audio-tools")
}

// DefaultResolver wraps api-core's per-scenario lookup for
// "audio-tools".
func DefaultResolver() URLResolver { return defaultResolver{} }

// FixedURLResolver returns a URLResolver that always returns the
// given URL. Use this in tests with httptest.
type FixedURLResolver string

// ResolveURL implements URLResolver.
func (f FixedURLResolver) ResolveURL(_ context.Context) (string, error) {
	return string(f), nil
}
