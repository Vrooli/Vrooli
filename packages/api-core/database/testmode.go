package database

import "context"

// testModeKey is the unexported context key that carries the "this request
// should be served from the test pool" signal across the request path.
type testModeKey struct{}

// WithTestMode returns a context that, when passed to a *RoutedDB method,
// causes the RoutedDB to route the call to its installed test pool (if any).
// In the absence of a test pool the routing falls back to the primary pool.
func WithTestMode(ctx context.Context) context.Context {
	return context.WithValue(ctx, testModeKey{}, true)
}

// IsTestMode reports whether ctx has been marked as test-mode by WithTestMode.
func IsTestMode(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(testModeKey{}).(bool)
	return v
}

// RoutingPartition is an opaque, comparable key for in-memory work and event
// delivery. A pinned pool's exact generation is distinct from primary storage,
// unpinned test traffic, other owners and replacement pools with the same lease
// ID. Keep the key in memory; it is not a durable identity or authorization.
type RoutingPartition struct {
	testMode bool
	pin      testPoolPin
}

// ContextPartition retains the routing identity already carried by ctx. It
// grants no storage access and does not prove the lease is still live: every
// actual database operation must continue through RoutedDB's pin validation.
// Clearing or replacing a lease never reclassifies its old key as primary.
func ContextPartition(ctx context.Context) RoutingPartition {
	if ctx != nil {
		if pin, ok := ctx.Value(testPoolPinKey{}).(testPoolPin); ok {
			return RoutingPartition{testMode: true, pin: pin}
		}
	}
	return RoutingPartition{testMode: IsTestMode(ctx)}
}
