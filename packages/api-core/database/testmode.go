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
