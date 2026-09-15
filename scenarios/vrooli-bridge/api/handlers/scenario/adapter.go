package scenario

import (
	"context"
	"fmt"
	"strings"

	"vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/registry"
	internal "vrooli-bridge/internal/scenario"

	"github.com/vrooli/api-core/scopecatalog"
	repocontract "github.com/vrooli/repo-contract-go"
)

type nodeReader struct{ registry registry.Service }

func (r nodeReader) GetTarget(ctx context.Context, id string) (internal.TargetNode, error) {
	node, err := r.registry.Get(ctx, id)
	if err != nil {
		return internal.TargetNode{}, err
	}
	return internal.TargetNode{ID: node.ID, Scopes: append([]string(nil), node.Scopes...), Revoked: node.Revoked()}, nil
}

// OnboardTarget is the SSH identity an owner last onboarded a node through, so
// a re-onboard remedy can be a command rather than a description.
type OnboardTarget struct {
	Host string
	User string
}

// versionFacts answers TargetFacts from the registry, the control plane's own
// commit, and the node's onboarding history.
type versionFacts struct {
	registry     registry.Service
	controlPlane func(context.Context) (string, error)
	onboardedAs  func(context.Context, string) (OnboardTarget, bool)
}

// NewVersionFacts builds the failure-path version context. controlPlane and
// onboardedAs may be nil; the corresponding facts are then omitted.
func NewVersionFacts(registrySvc registry.Service, controlPlane func(context.Context) (string, error), onboardedAs func(context.Context, string) (OnboardTarget, bool)) TargetFacts {
	return versionFacts{registry: registrySvc, controlPlane: controlPlane, onboardedAs: onboardedAs}
}

func (v versionFacts) VersionFacts(ctx context.Context, nodeID, scenario string) VersionFacts {
	var facts VersionFacts
	if v.controlPlane != nil {
		if commit, err := v.controlPlane(ctx); err == nil {
			facts.ControlPlaneRevision = shortRevision(commit)
		}
	}
	node, err := v.registry.Get(ctx, nodeID)
	if err != nil {
		return facts
	}
	facts.TargetRevision = shortRevision(node.Revision)
	// The node already carries the control plane's revision, so the source is
	// not behind: the scenario's running process predates the update (a
	// re-onboard ships a tree but does not restart running scenarios).
	if scenario != "" && facts.ControlPlaneRevision != "" && strings.TrimSuffix(facts.TargetRevision, "+dirty") == strings.TrimSuffix(facts.ControlPlaneRevision, "+dirty") {
		facts.UpdatePath = "restart"
		facts.UpdateCommand = fmt.Sprintf("vrooli-bridge relay call --node-id %s --scenario %s --command \"scenario restart\"", node.ID, scenario)
		return facts
	}
	observation, reported := node.ProvisioningObservation()
	if reported && observation.State == "ready" && !onboard.IsWorkingTreeRevision(node.Revision) {
		facts.UpdatePath = "provision"
		facts.UpdateCommand = "vrooli-bridge provision sync " + node.ID
		return facts
	}
	// Provisioning cannot run on this node (no helper, no git checkout, or an
	// agent too old to say), so re-running onboarding is the update path.
	facts.UpdatePath = "reonboard"
	command := "vrooli-bridge onboard connect"
	if target, ok := v.lookupOnboardTarget(ctx, node.ID); ok {
		command += " --host " + target.Host
		if target.User != "" {
			command += " --user " + target.User
		}
	} else {
		command += " --host <host>"
	}
	if onboard.IsWorkingTreeRevision(node.Revision) {
		command += " --source working-tree"
	}
	facts.UpdateCommand = command
	return facts
}

func (v versionFacts) lookupOnboardTarget(ctx context.Context, nodeID string) (OnboardTarget, bool) {
	if v.onboardedAs == nil {
		return OnboardTarget{}, false
	}
	target, ok := v.onboardedAs(ctx, nodeID)
	return target, ok && target.Host != ""
}

// shortRevision keeps a revision readable in a sentence: a 12-character commit
// with the working-tree marker preserved, so a dirty node still reads as one.
func shortRevision(revision string) string {
	revision = strings.TrimSpace(revision)
	base, dirty := strings.CutSuffix(revision, "+dirty")
	if len(base) > 12 {
		base = base[:12]
	}
	if dirty {
		return base + "+dirty"
	}
	return base
}

// NewService wires catalog-derived per-method authorization into the proxy.
// A missing procedure entry is a hard refusal: target-aware HTTP must never
// become an ungoverned escape hatch around the CLI manifest.
func NewService(registrySvc registry.Service, presence internal.Presence, pusher internal.Pusher, broker *internal.Broker) internal.Service {
	root, rootErr := repocontract.FindRepoRootFromEnvOrCWD()
	var catalog scopecatalog.Catalog
	if rootErr == nil {
		catalog, rootErr = scopecatalog.BuildResilient(root)
	}
	return newServiceWithCatalog(registrySvc, presence, pusher, broker, catalog, rootErr)
}

func newServiceWithCatalog(registrySvc registry.Service, presence internal.Presence, pusher internal.Pusher, broker *internal.Broker, catalog scopecatalog.Catalog, catalogErr error) internal.Service {
	return internal.NewService(nodeReader{registry: registrySvc}, presence, pusher, broker, internal.WithAdmission(func(request internal.Request, node internal.TargetNode) error {
		return admit(catalog, catalogErr, request, node)
	}))
}

func admit(catalog scopecatalog.Catalog, catalogErr error, request internal.Request, node internal.TargetNode) error {
	if catalogErr != nil {
		return fmt.Errorf("scenario proxy catalog unavailable: %w", catalogErr)
	}
	serviceName := request.Service[strings.LastIndex(request.Service, ".")+1:]
	for _, scope := range catalog.Scopes {
		if scope.Scenario != request.Scenario || scope.Service != serviceName || scope.Method != request.Method {
			continue
		}
		required, ok := scopecatalog.TransportScope(scope.Value)
		if !ok || !scopecatalog.Resolve(node.Scopes, scope.Value) || !scopecatalog.Resolve(node.Scopes, required) {
			return fmt.Errorf("target node lacks scope %s for %s.%s", scope.Value, request.Service, request.Method)
		}
		// run_eligible controls prompt-manager action invocation. Owner
		// identity is established by the HTTP handler; the node namespace grant
		// and Bridge transport grant authorize this proxy call.
		return nil
	}
	return fmt.Errorf("scenario procedure %s/%s is not governed", request.Service, request.Method)
}

func splitProcedure(path string) (service, method string, err error) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" || !strings.Contains(parts[0], ".") {
		return "", "", fmt.Errorf("scenario proxy accepts Connect procedures; %q is not one", path)
	}
	return parts[0], parts[1], nil
}
