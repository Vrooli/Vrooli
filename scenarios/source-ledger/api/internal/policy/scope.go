package policy

import "context"

type (
	scopeContextKey  struct{}
	configContextKey struct{}
)

// NormalizeScope is the single defaulting rule shared by handlers, services,
// and CLI-facing integration code. Empty requests address the original
// agent-memory ledger; named scopes remain explicit all the way to storage.
func NormalizeScope(raw string) Scope {
	if raw == "" {
		return AgentMemory
	}
	return Scope(raw)
}

func WithScope(ctx context.Context, raw string) context.Context {
	return context.WithValue(ctx, scopeContextKey{}, NormalizeScope(raw))
}

func ScopeFromContext(ctx context.Context) Scope {
	if scope, ok := ctx.Value(scopeContextKey{}).(Scope); ok && scope != "" {
		return scope
	}
	return AgentMemory
}

func WithConfig(ctx context.Context, config Config) context.Context {
	return context.WithValue(ctx, configContextKey{}, config)
}

func ConfigFromContext(ctx context.Context) (Config, bool) {
	config, ok := ctx.Value(configContextKey{}).(Config)
	return config, ok
}

func ResolveContext(ctx context.Context, registry *Registry) (context.Context, error) {
	if registry == nil {
		return ctx, nil
	}
	config, err := registry.Resolve(ctx, string(ScopeFromContext(ctx)))
	if err != nil {
		return ctx, err
	}
	return WithConfig(ctx, config), nil
}
