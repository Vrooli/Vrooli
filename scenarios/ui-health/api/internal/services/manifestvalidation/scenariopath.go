package manifestvalidation

import (
	"context"
	"strings"
)

// scenarioPathCtxKey carries an optional explicit scenario root directory through
// the validation call so the service can validate a scenario that lives outside
// the repo scenarios/ tree (e.g. Test Genie's deep template validation against a
// temp-generated scenario) without changing the Validator interface or its test
// mocks. When absent, the scenario root falls back to
// <RepoRoot>/scenarios/<scenario>. The template manifest is always read from the
// repo (templates live in the repo), so RepoRoot stays required either way.
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
