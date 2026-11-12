// Package capabilities projects the desk's optional dependencies from
// .vrooli/service.json into the shared capability registry. The manifest is
// the only list of dependencies and the registry the only table of whether
// each can be reached (rulings E5): launch and the courier dial a peer only
// through Reach. Every dependency is optional (charter I5).
package capabilities

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/vrooli/api-core/discovery"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

type (
	Def      = capabilityregistry.Def
	Registry = capabilityregistry.Registry
	Status   = capabilityregistry.Status
)

// StatusAvailable is the registry's word for a dependency that answers.
const StatusAvailable = capabilityregistry.StatusAvailable

// Load projects the manifest at path. Each scenario dependency gets the
// operator command that starts it.
func Load(path string) ([]Def, error) {
	defs, err := capabilityregistry.ProjectManifest(path, nil)
	if err != nil {
		return nil, fmt.Errorf("project %s: %w", path, err)
	}
	for i := range defs {
		if defs[i].DependencyKind != capabilityregistry.DependencyScenario {
			continue
		}
		defs[i].ActionKind = capabilityregistry.ActionKindScenarioStart
		defs[i].ActionLabel = "Start " + defs[i].DependencySlug
		defs[i].OperatorCommand = "vrooli scenario start " + defs[i].DependencySlug + " --json"
	}
	return defs, nil
}

// PortResolver finds a scenario's API port.
type PortResolver interface {
	ResolveScenarioPort(ctx context.Context, scenario, portKey string) (int, error)
}

// HealthChecker reports a scenario available when its API port resolves and
// its /health answers 2xx.
type HealthChecker struct {
	Slug     string
	Resolver PortResolver
	Client   *http.Client
}

// Check implements capabilityregistry.Checker.
func (c HealthChecker) Check(ctx context.Context) (capabilityregistry.Status, string) {
	ctx, cancel := context.WithTimeout(ctx, CheckTimeout)
	defer cancel()
	port, err := c.Resolver.ResolveScenarioPort(ctx, c.Slug, "API_PORT")
	if err != nil {
		return capabilityregistry.StatusUnavailable, fmt.Sprintf("%s is not running: %v; start it with `vrooli scenario start %s`", c.Slug, err, c.Slug)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d/health", port), nil)
	if err != nil {
		return capabilityregistry.StatusUnknown, err.Error()
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return capabilityregistry.StatusUnavailable, fmt.Sprintf("%s health on port %d failed: %v", c.Slug, port, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return capabilityregistry.StatusUnavailable, fmt.Sprintf("%s health on port %d answered %d", c.Slug, port, resp.StatusCode)
	}
	return capabilityregistry.StatusAvailable, fmt.Sprintf("%s is healthy", c.Slug)
}

// CheckTimeout bounds one health check (port lookup and /health) and the
// port lookup of a reach; peer.CallTimeout bounds each call after it.
const CheckTimeout = 3 * time.Second

// NewRegistry builds the registry over defs with a health check per scenario
// dependency. Results are cached for 15 s.
func NewRegistry(defs []Def, resolver PortResolver) *Registry {
	client := &http.Client{Timeout: CheckTimeout}
	checkers := make(map[string]capabilityregistry.Checker, len(defs))
	for _, def := range defs {
		if def.DependencyKind == capabilityregistry.DependencyScenario {
			checkers[def.ID] = HealthChecker{Slug: def.DependencySlug, Resolver: resolver, Client: client}
		}
	}
	return capabilityregistry.New(defs, checkers, 15*time.Second)
}

// DefaultResolver is the local discovery ladder.
func DefaultResolver() PortResolver {
	return discovery.NewResolver(discovery.ResolverConfig{})
}

// Reach returns a scenario's API base URL when the registry says it is
// available, or the registry's reason (with the command that starts it). A
// nil registry reaches nothing and says why.
func Reach(registry *Registry, resolver PortResolver, slug string) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		if registry == nil {
			return "", fmt.Errorf("%s: the capability registry is not loaded (.vrooli/service.json not found; set VROOLI_ROOT)", slug)
		}
		state, ok := Find(registry.Resolve(ctx), slug)
		switch {
		case !ok:
			return "", fmt.Errorf("%s is not a dependency in .vrooli/service.json", slug)
		case state.Status != capabilityregistry.StatusAvailable:
			return "", fmt.Errorf("%s is %s: %s", slug, state.Status, state.Message)
		}
		ctx, cancel := context.WithTimeout(ctx, CheckTimeout)
		defer cancel()
		port, err := resolver.ResolveScenarioPort(ctx, slug, "API_PORT")
		if err != nil {
			return "", fmt.Errorf("%s stopped answering (%v); start it with `vrooli scenario start %s`", slug, err, slug)
		}
		return fmt.Sprintf("http://localhost:%d", port), nil
	}
}

// Find is the state of the dependency named slug.
func Find(states []capabilityregistry.State, slug string) (capabilityregistry.State, bool) {
	for _, s := range states {
		if s.DependencySlug == slug || s.ID == slug {
			return s, true
		}
	}
	return capabilityregistry.State{}, false
}
