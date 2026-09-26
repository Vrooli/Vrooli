// Package selector is the one deployment-targeting grammar of the CLI. Every
// command that acts on a deployment accepts exactly one of the canonical
// forms — --deployment <id> (or a positional id); --scenario with
// --environment; --scenario with --domain; --scenario with --host — and
// resolves it through the DeploymentsService. More than one match is the
// server's typed deployment_selector_ambiguous refusal (exit 2, candidates
// listed); the CLI never picks one.
package selector

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	deploymentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/identity"

	"scenario-to-cloud/cli/internal/apierr"
)

// Flags holds the parsed selector facets.
type Flags struct {
	deployment  *string
	scenario    *string
	environment *string
	domain      *string
	host        *string
}

// Register adds the selector flags to fs.
func Register(fs *flag.FlagSet) *Flags {
	return &Flags{
		deployment:  fs.String("deployment", "", "Deployment id"),
		scenario:    fs.String("scenario", "", "Scenario id (pair with --environment, --domain or --host)"),
		environment: fs.String("environment", "", "Environment name (with --scenario)"),
		domain:      fs.String("domain", "", "Edge domain (with --scenario)"),
		host:        fs.String("host", "", "Target host (with --scenario)"),
	}
}

// Usage is the selector grammar for help text.
const Usage = "(--deployment <id> | <id> | --scenario <id> --environment <env> | --scenario <id> --domain <domain> | --scenario <id> --host <host>)"

// Selector is one canonical selector.
type Selector struct {
	ID          string
	ScenarioID  string
	Environment string
	Domain      string
	Host        string
}

// Any reports whether any facet was given.
func (f *Flags) Any() bool {
	return strings.TrimSpace(*f.deployment) != "" || strings.TrimSpace(*f.scenario) != "" ||
		strings.TrimSpace(*f.environment) != "" || strings.TrimSpace(*f.domain) != "" || strings.TrimSpace(*f.host) != ""
}

// Selector builds the selector from the flags and an optional positional
// id. Grammar violations are local refusals (exit 2) so no request is sent.
func (f *Flags) Selector(positional []string) (Selector, error) {
	sel := Selector{
		ID: strings.TrimSpace(*f.deployment), ScenarioID: strings.TrimSpace(*f.scenario),
		Environment: strings.TrimSpace(*f.environment), Domain: strings.TrimSpace(*f.domain), Host: strings.TrimSpace(*f.host),
	}
	switch {
	case len(positional) > 1:
		return Selector{}, apierr.Refused("expected one deployment selector %s, got %d positional arguments", Usage, len(positional))
	case len(positional) == 1:
		if sel.ID != "" && sel.ID != strings.TrimSpace(positional[0]) {
			return Selector{}, apierr.Refused("positional id %q conflicts with --deployment %q", positional[0], sel.ID)
		}
		sel.ID = strings.TrimSpace(positional[0])
	}
	if sel.ID != "" {
		if sel.ScenarioID != "" || sel.Environment != "" || sel.Domain != "" || sel.Host != "" {
			return Selector{}, apierr.Refused("--deployment cannot be combined with --scenario/--environment/--domain/--host")
		}
		return sel, nil
	}
	if sel.ScenarioID == "" {
		return Selector{}, apierr.Refused("a deployment selector is required %s", Usage)
	}
	facets := 0
	for _, v := range []string{sel.Environment, sel.Domain, sel.Host} {
		if v != "" {
			facets++
		}
	}
	if facets != 1 {
		return Selector{}, apierr.Refused("--scenario must be paired with exactly one of --environment, --domain or --host")
	}
	return sel, nil
}

// Proto renders the selector as the wire message.
func (s Selector) Proto() *deploymentsv1.DeploymentSelector {
	return &deploymentsv1.DeploymentSelector{Id: s.ID, ScenarioId: s.ScenarioID, Environment: s.Environment, Domain: s.Domain, Host: s.Host}
}

// String renders the selector for human output.
func (s Selector) String() string {
	if s.ID != "" {
		return "deployment " + s.ID
	}
	parts := []string{"scenario=" + s.ScenarioID}
	if s.Environment != "" {
		parts = append(parts, "environment="+s.Environment)
	}
	if s.Domain != "" {
		parts = append(parts, "domain="+s.Domain)
	}
	if s.Host != "" {
		parts = append(parts, "host="+s.Host)
	}
	return strings.Join(parts, " ")
}

// Resolve maps the selector to exactly one DeploymentRef through the service.
// An id selector is resolved too, so the caller always holds the canonical
// identity (scenario, environment, target) for output.
func Resolve(ctx context.Context, client deploymentsv1connect.DeploymentsServiceClient, sel Selector) (*identityv1.DeploymentRef, error) {
	resp, err := client.ResolveDeployment(ctx, connect.NewRequest(&deploymentsv1.ResolveDeploymentRequest{Selector: sel.Proto()}))
	if err != nil {
		return nil, err
	}
	if resp.Msg.GetRef() == nil || resp.Msg.GetRef().GetId() == "" {
		return nil, fmt.Errorf("resolve %s: server returned no deployment reference", sel)
	}
	return resp.Msg.GetRef(), nil
}

// ResolveID returns the deployment id for a selector. The id form is
// returned as given (the server validates it on the next call); every other
// form is resolved through the service so ambiguity is the server's typed
// refusal.
func ResolveID(ctx context.Context, client deploymentsv1connect.DeploymentsServiceClient, sel Selector) (string, error) {
	if sel.ID != "" {
		return sel.ID, nil
	}
	ref, err := Resolve(ctx, client, sel)
	if err != nil {
		return "", err
	}
	return ref.GetId(), nil
}

// TargetKey renders the target identity the way the API keys it:
// machine:<id> for an enrolled machine, host:<host> otherwise.
func TargetKey(ref *identityv1.DeploymentRef) string {
	if ref == nil || ref.GetTarget() == nil {
		return ""
	}
	t := ref.GetTarget()
	if t.GetMachineId() != "" {
		return "machine:" + t.GetMachineId()
	}
	if t.GetLocator().GetHost() != "" {
		return "host:" + t.GetLocator().GetHost()
	}
	return ""
}

// Identity renders the canonical identity line shared by every command's
// human output.
func Identity(ref *identityv1.DeploymentRef) string {
	if ref == nil {
		return ""
	}
	line := fmt.Sprintf("deployment: %s  scenario: %s  environment: %s", ref.GetId(), ref.GetScenarioId(), ref.GetEnvironment())
	if key := TargetKey(ref); key != "" {
		line += "  target: " + key
		if transport := ref.GetTarget().GetTransport(); transport != "" {
			line += " (" + transport + ")"
		}
	}
	return line
}
