package conflicts

import (
	"context"
	"fmt"
	"sort"
)

// Resolver is one pluggable resolver. v0.1 resolvers record operator
// intent only; actual mutation lives in the apply phase (which is
// unimplemented in v0.1). Resolvers that defer to apply return
// ErrResolverDeferred so the service can surface "ready for `apply`"
// to the CLI.
type Resolver interface {
	Name() string
	Description() string
	HandlesKinds() []FixKind
	RequiresApply() bool

	// Resolve is called with the conflict and the operator-selected
	// Fix. Implementations either complete the resolution synchronously
	// (no apply dependency) or return ErrResolverDeferred.
	Resolve(ctx context.Context, c Conflict, fix Fix) error
}

// ErrResolverDeferred is the typed sentinel returned by resolvers that
// require the apply phase to run.
type ErrResolverDeferred struct {
	Resolver string
	Reason   string
}

func (e ErrResolverDeferred) Error() string {
	return fmt.Sprintf("resolver %q deferred: %s", e.Resolver, e.Reason)
}

// ResolverRegistry is the plug-in registry for resolvers.
type ResolverRegistry struct {
	resolvers []Resolver
}

// NewResolverRegistry returns a registry holding the given resolvers in
// alphabetical order by Name().
func NewResolverRegistry(in ...Resolver) *ResolverRegistry {
	out := append([]Resolver(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return &ResolverRegistry{resolvers: out}
}

// Lookup returns the resolver whose Name matches, or nil.
func (r *ResolverRegistry) Lookup(name string) Resolver {
	for _, res := range r.resolvers {
		if res.Name() == name {
			return res
		}
	}
	return nil
}

// All returns the registered resolvers in deterministic order.
func (r *ResolverRegistry) All() []Resolver {
	out := make([]Resolver, len(r.resolvers))
	copy(out, r.resolvers)
	return out
}

// Describe returns one descriptor per registered resolver.
func (r *ResolverRegistry) Describe() []ResolverDescriptor {
	out := make([]ResolverDescriptor, 0, len(r.resolvers))
	for _, res := range r.resolvers {
		out = append(out, ResolverDescriptor{
			Name:          res.Name(),
			Description:   res.Description(),
			Stability:     "beta",
			HandlesKinds:  res.HandlesKinds(),
			RequiresApply: res.RequiresApply(),
		})
	}
	return out
}
