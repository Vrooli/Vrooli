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
// A missing catalog entry is a hard refusal: target-aware HTTP must never
// become an ungoverned escape hatch around the CLI manifest.
func NewService(registrySvc registry.Service, presence internal.Presence, pusher internal.Pusher, broker *internal.Broker) internal.Service {
	root, rootErr := repocontract.FindRepoRootFromEnvOrCWD()
	var catalog scopecatalog.Catalog
	if rootErr == nil {
		catalog, rootErr = scopecatalog.BuildResilient(root)
	}
	return internal.NewService(nodeReader{registry: registrySvc}, presence, pusher, broker, internal.WithAdmission(func(request internal.Request, node internal.TargetNode) error {
		if rootErr != nil {
			return fmt.Errorf("scenario proxy catalog unavailable: %w", rootErr)
		}
		for _, scope := range catalog.Scopes {
			serviceName := scope.Service
			if index := strings.LastIndex(request.Service, "."); index >= 0 {
				serviceName = request.Service[index+1:]
			}
			if scope.Scenario != request.Scenario || scope.Service != serviceName || scope.Method != request.Method {
				continue
			}
			if !scope.RunEligible {
				return fmt.Errorf("scenario method %s.%s is not run-eligible", request.Service, request.Method)
			}
			required, ok := scopecatalog.TransportScope(scope.Value)
			if !ok || !scopecatalog.Resolve(node.Scopes, scope.Value) || !scopecatalog.Resolve(node.Scopes, required) {
				return fmt.Errorf("target node lacks scope %s for %s.%s", scope.Value, request.Service, request.Method)
			}
			return nil
		}
		return fmt.Errorf("scenario method %s.%s is not in the governed catalog", request.Service, request.Method)
	}))
}

func splitProcedure(path string) (service, method string, err error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || strings.TrimSpace(parts[len(parts)-1]) == "" || strings.TrimSpace(parts[len(parts)-2]) == "" {
		return "", "", fmt.Errorf("scenario proxy path must end in service/method")
	}
	return strings.Join(parts[:len(parts)-1], "/"), parts[len(parts)-1], nil
}
