// Package capabilities projects the desk's optional dependencies from
// .vrooli/service.json into the shared capability registry. The manifest is
// the only list of dependencies and the registry the only table of whether
// each can be reached: launch and the courier dial a peer only
// through Reach. Every dependency is optional.
package capabilities

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/vrooli/api-core/discovery"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"

	"research-desk/internal/home"
)

type (
	Def      = capabilityregistry.Def
	Registry = capabilityregistry.Registry
	Status   = capabilityregistry.Status
)

// The registry's words: a dependency that answers, and one whose check
// could not tell (it timed out).
const (
	StatusAvailable = capabilityregistry.StatusAvailable
	StatusUnknown   = capabilityregistry.StatusUnknown
)

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
		defs[i].OperatorCommand = StartCommand(defs[i].DependencySlug) + " --json"
	}
	return defs, nil
}

// StartCommand is the command that starts a scenario dependency, for the CLI
// to print beside a reason that names no command.
func StartCommand(slug string) string { return "vrooli scenario start " + slug }

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

// Check implements capabilityregistry.Checker. A check that timed out is
// unknown, not unavailable: a busy host can delay a healthy scenario.
func (c HealthChecker) Check(ctx context.Context) (capabilityregistry.Status, string) {
	ctx, cancel := context.WithTimeout(ctx, CheckTimeout)
	defer cancel()
	port, err := c.Resolver.ResolveScenarioPort(ctx, c.Slug, "API_PORT")
	switch {
	case err != nil && timedOut(err):
		return capabilityregistry.StatusUnknown, fmt.Sprintf("%s was not reached just now (timed out after %s finding its port)", c.Slug, CheckTimeout)
	case err != nil:
		return capabilityregistry.StatusUnavailable, fmt.Sprintf("%s is not running: %v", c.Slug, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d/health", port), nil)
	if err != nil {
		return capabilityregistry.StatusUnknown, err.Error()
	}
	resp, err := c.Client.Do(req)
	switch {
	case err != nil && timedOut(err):
		return capabilityregistry.StatusUnknown, fmt.Sprintf("%s was not reached just now (timed out after %s); its lifecycle lists it running", c.Slug, CheckTimeout)
	case err != nil:
		return capabilityregistry.StatusUnavailable, fmt.Sprintf("%s health on port %d failed: %v", c.Slug, port, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return capabilityregistry.StatusUnavailable, fmt.Sprintf("%s health on port %d answered %d", c.Slug, port, resp.StatusCode)
	}
	return capabilityregistry.StatusAvailable, fmt.Sprintf("%s is healthy", c.Slug)
}

// timedOut reports whether err is a deadline rather than an answer.
func timedOut(err error) bool {
	var nerr net.Error
	var derr *discovery.Error
	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &nerr) && nerr.Timeout()) || (errors.As(err, &derr) && derr.Kind == discovery.ErrTimeout)
}

// ErrNotReached marks a dependency whose state is unknown because its check
// timed out: neither available nor down.
var ErrNotReached = errors.New("not reached just now")

// notReached carries the check's own sentence and matches ErrNotReached.
type notReached struct{ reason string }

func (e notReached) Error() string        { return e.reason }
func (e notReached) Is(target error) bool { return target == ErrNotReached }

// StartHint is the command that starts slug for the CLI to print, or "" when
// its lifecycle lists it running (or the lookup timed out): starting a
// running scenario is never the next step.
func StartHint(ctx context.Context, resolver PortResolver, slug string) string {
	ctx, cancel := context.WithTimeout(ctx, CheckTimeout)
	defer cancel()
	if _, err := resolver.ResolveScenarioPort(ctx, slug, "API_PORT"); err == nil || timedOut(err) {
		return ""
	}
	return StartCommand(slug)
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
// available, or the registry's reason (carrying the command that starts it). A
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
		case state.Status == capabilityregistry.StatusUnknown:
			return "", notReached{state.Message}
		case state.Status != capabilityregistry.StatusAvailable:
			return "", withHint(fmt.Errorf("%s is %s: %s", slug, state.Status, state.Message), StartHint(ctx, resolver, slug))
		}
		ctx, cancel := context.WithTimeout(ctx, CheckTimeout)
		defer cancel()
		port, err := resolver.ResolveScenarioPort(ctx, slug, "API_PORT")
		if err != nil && timedOut(err) {
			return "", notReached{slug + " was not reached just now (timed out finding its port)"}
		}
		if err != nil {
			return "", withHint(fmt.Errorf("%s stopped answering (%v)", slug, err), StartCommand(slug))
		}
		return fmt.Sprintf("http://localhost:%d", port), nil
	}
}

func withHint(err error, command string) error {
	if command == "" {
		return err
	}
	return home.WithCommand(err, command)
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
