package manifestvalidation

import (
	"context"
	"strings"
)

// scenarioPathCtxKey carries an optional explicit scenario root directory through
// the validation call so the filesystem manifest loader can read a scenario that
// lives outside the repo scenarios/ tree (e.g. Test Genie's deep template
// validation against a temp-generated scenario) without changing the Validator
// interface or its many test mocks. When absent, resolution falls back to the
// repo-contract path keyed by scenario name.
type scenarioPathCtxKey struct{}

// WithScenarioPath attaches an explicit scenario root directory to ctx. The
// connect handler threads the request path here; empty paths are ignored so
// in-registry callers keep the name-based resolution.
func WithScenarioPath(ctx context.Context, path string) context.Context {
	path = strings.TrimSpace(path)
	if path == "" {
		return ctx
	}
	return context.WithValue(ctx, scenarioPathCtxKey{}, path)
}

// scenarioPathFrom returns the explicit scenario root attached to ctx, or "".
func scenarioPathFrom(ctx context.Context) string {
	path, _ := ctx.Value(scenarioPathCtxKey{}).(string)
	return path
}

// includeExecutionCtxKey carries the request's include_execution switch through
// the validation call. It mirrors the scenario-path threading above: the connect
// handler attaches the request field here so the runtime CLI probe stays opt-in
// without widening the Validator interface (and its many test mocks). When
// absent the default static-only path runs — the probe never executes.
type includeExecutionCtxKey struct{}

// WithIncludeExecution records the caller's execution request on ctx. The
// connect handler threads ValidateScenarioRequest.include_execution here;
// providers that only inspect leave it unset and keep the static path.
func WithIncludeExecution(ctx context.Context, include bool) context.Context {
	return context.WithValue(ctx, includeExecutionCtxKey{}, include)
}

// includeExecutionFrom reports whether the caller requested execution. Absent
// (the default) means static-only.
func includeExecutionFrom(ctx context.Context) bool {
	include, _ := ctx.Value(includeExecutionCtxKey{}).(bool)
	return include
}
