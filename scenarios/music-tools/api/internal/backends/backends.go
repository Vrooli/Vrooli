// Package backends owns the music provider seam. Composition currently has
// one local ACE-Step provider; the registry keeps future MIR and remote lanes
// from changing callers.
package backends

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

type Request struct {
	Operation string
	ModelID   string
	InputKeys []string
	Params    map[string]string
	Progress  func(float64, string)
}
type Result struct {
	OutputRef string
	Tier      string
	Meta      map[string]string
}
type Provider interface {
	Name() string
	Operations() []string
	Standalone() bool
	IsCloud() bool
	Available(context.Context) bool
	Execute(context.Context, Request) (Result, error)
}
type Registry struct{ byOperation map[string][]Provider }

var (
	ErrNoOperations      = errors.New("backends: provider has no operations")
	ErrMissingStandalone = errors.New("backends: operation has no standalone provider")
	ErrNoProvider        = errors.New("backends: no provider for operation")
	ErrNoneAvailable     = errors.New("backends: no available provider")
)

func New() *Registry { return &Registry{byOperation: map[string][]Provider{}} }
func (r *Registry) Register(provider Provider) error {
	ops := provider.Operations()
	if len(ops) == 0 {
		return fmt.Errorf("%w: %s", ErrNoOperations, provider.Name())
	}
	for _, op := range ops {
		r.byOperation[op] = append(r.byOperation[op], provider)
	}
	return nil
}
func (r *Registry) Validate() error {
	var missing []string
	for op, providers := range r.byOperation {
		found := false
		for _, p := range providers {
			if p.Standalone() && !p.IsCloud() {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, op)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("%w: %v", ErrMissingStandalone, missing)
	}
	return nil
}
func (r *Registry) Select(ctx context.Context, operation string) (Provider, error) {
	providers := r.byOperation[operation]
	if len(providers) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoProvider, operation)
	}
	for _, p := range providers {
		if p.Standalone() && !p.IsCloud() && p.Available(ctx) {
			return p, nil
		}
	}
	for _, p := range providers {
		if p.IsCloud() && p.Available(ctx) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNoneAvailable, operation)
}
func (r *Registry) Operations() []string {
	out := make([]string, 0, len(r.byOperation))
	for op := range r.byOperation {
		out = append(out, op)
	}
	sort.Strings(out)
	return out
}
