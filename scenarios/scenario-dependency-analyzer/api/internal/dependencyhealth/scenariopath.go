package dependencyhealth

import (
	"context"
	"path/filepath"
	"strings"
)

// scenarioPathCtxKey carries an optional explicit scenario root directory through
// the dependency-health evaluation so the stages can read service.json, surfaces,
// and release-age policy for a scenario that lives outside the repo scenarios/
// tree (e.g. Test Genie's deep template validation against a temp-generated
// scenario) without changing the native ValidateDependencyHealth request shape.
// When absent, the scenario dir falls back to <scenariosDir>/<scenario>.
type scenarioPathCtxKey struct{}

// withScenarioPath attaches an explicit scenario root directory to ctx. The
// shared ValidateScenario adapter threads the request path here; empty paths are
// ignored so in-registry callers keep name-based resolution.
func withScenarioPath(ctx context.Context, path string) context.Context {
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

// scenarioDir resolves the on-disk root for scenario, preferring an explicit
// path supplied via withScenarioPath and otherwise joining the configured
// scenarios directory with the scenario name.
func (h *connectHandler) scenarioDir(ctx context.Context, scenario string) string {
	if explicit := scenarioPathFrom(ctx); explicit != "" {
		return explicit
	}
	return filepath.Join(h.resolveScenariosDir(), scenario)
}
