package session

import "context"

// PreSpawnFunc runs with a freshly assigned session id at the single moment
// after the id is known but BEFORE the session's PTY environment is built. It
// exists so a caller that needs a per-session side effect reflected in the
// spawned process environment (e.g. minting and stashing a human-owner
// delegation that the env builder then reads back by session id) can run it at
// exactly the right point in the create flow. It must be synchronous and
// best-effort: it is invoked inline and its result is never inspected, so it
// can never fail or block a session launch.
type PreSpawnFunc func(sessionID string)

type preSpawnHookKey struct{}

// WithPreSpawnHook returns a context carrying a hook the manager invokes with
// the new session id immediately before building the PTY environment. A nil fn
// is a no-op. It is deliberately context-scoped (not manager state) so the hook
// is bound to exactly one create request and cannot leak across concurrent
// creates.
func WithPreSpawnHook(ctx context.Context, fn PreSpawnFunc) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, preSpawnHookKey{}, fn)
}

// preSpawnHookFromContext returns the create-scoped pre-spawn hook when one is
// present.
func preSpawnHookFromContext(ctx context.Context) (PreSpawnFunc, bool) {
	fn, ok := ctx.Value(preSpawnHookKey{}).(PreSpawnFunc)
	return fn, ok && fn != nil
}
