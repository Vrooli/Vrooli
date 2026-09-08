package scenario

import (
	"context"
	"fmt"
	"strings"

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
