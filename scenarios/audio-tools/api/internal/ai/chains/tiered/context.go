package tiered

import "context"

// FallbackCallback is the per-request hook invoked by the coordinator when
// a request is served from a tier other than its first-priority tier.
type FallbackCallback func(ev FallbackEvent)

type ctxKey struct{}

// WithOnFallback attaches a per-request fallback callback to ctx. The
// Coordinator invokes it (in addition to any Options.OnFallback configured
// at construction time) whenever a successful response originates from a
// non-primary tier. This is the seam Connect handlers use to bridge the
// chain-internal fallback signal to the response header / response trailer
// without threading anything through the chain Options on every request.
func WithOnFallback(ctx context.Context, cb FallbackCallback) context.Context {
	if cb == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, cb)
}

func onFallbackFromContext(ctx context.Context) FallbackCallback {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(ctxKey{}).(FallbackCallback); ok {
		return v
	}
	return nil
}
