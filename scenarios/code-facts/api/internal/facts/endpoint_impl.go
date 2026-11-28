package facts

import (
	"sort"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

type payloadRole string

const (
	payloadRoleRequest  payloadRole = "request"
	payloadRoleResponse payloadRole = "response"
	payloadRoleError    payloadRole = "error"
)

type endpointImplementation struct {
	EndpointID string
	SurfaceID  string
	Language   string
	Framework  string
	Route      endpointRouteEvidence
	Handler    endpointHandlerEvidence
	Payloads   map[payloadRole]payloadImplementationEvidence
	Evidence   []*factsv1.Evidence
}

type endpointRouteEvidence struct {
	Path   string
	Method string
	Status factsv1.EvidenceStatus
	Range  *factsv1.SourceRange
}

type endpointHandlerEvidence struct {
	Expression string
	Symbol     string
	Enclosing  string
}

type payloadImplementationEvidence struct {
	Role          payloadRole
	ProtoFullName string
	Transport     string
	Status        factsv1.EvidenceStatus
	Range         *factsv1.SourceRange
	Message       string
	Evidence      *factsv1.Evidence
}

type endpointAdapterContext struct {
	target    *factsv1.TargetContext
	facts     []*factsv1.GenericFact
	endpoints []endpointDeclaration
}

type endpointAdapter interface {
	ID() string
	Language() string
	Supports(ctx endpointAdapterContext) bool
	ExtractEndpointImplementations(ctx endpointAdapterContext) []endpointImplementation
}

type endpointAdapterRegistry []endpointAdapter

func defaultEndpointAdapters() endpointAdapterRegistry {
	return endpointAdapterRegistry{
		goEndpointAdapter{},
		tsExpressEndpointAdapter{},
	}
}

func (r endpointAdapterRegistry) ExtractEndpointImplementations(ctx endpointAdapterContext) []endpointImplementation {
	adapters := append(endpointAdapterRegistry(nil), r...)
	sort.SliceStable(adapters, func(i, j int) bool { return adapters[i].ID() < adapters[j].ID() })
	var out []endpointImplementation
	for _, adapter := range adapters {
		if adapter == nil || !adapter.Supports(ctx) {
			continue
		}
		out = append(out, adapter.ExtractEndpointImplementations(ctx)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].EndpointID != out[j].EndpointID {
			return out[i].EndpointID < out[j].EndpointID
		}
		if out[i].Language != out[j].Language {
			return out[i].Language < out[j].Language
		}
		return out[i].Framework < out[j].Framework
	})
	return out
}

type goEndpointAdapter struct{}

func (goEndpointAdapter) ID() string       { return "go.http" }
func (goEndpointAdapter) Language() string { return "go" }

func (goEndpointAdapter) Supports(ctx endpointAdapterContext) bool {
	for _, fact := range ctx.facts {
		if fact.GetAttributes()["language"] == "go" {
			return true
		}
	}
	return false
}

type tsExpressEndpointAdapter struct{}

func (tsExpressEndpointAdapter) ID() string       { return "ts.express" }
func (tsExpressEndpointAdapter) Language() string { return "typescript" }

func (tsExpressEndpointAdapter) Supports(ctx endpointAdapterContext) bool {
	for _, fact := range ctx.facts {
		attrs := fact.GetAttributes()
		if attrs["language"] == "typescript" && attrs["router_framework"] == "express" {
			return true
		}
	}
	return false
}

func (tsExpressEndpointAdapter) ExtractEndpointImplementations(ctx endpointAdapterContext) []endpointImplementation {
	var out []endpointImplementation
	for _, endpoint := range ctx.endpoints {
		if endpoint.RESTException == nil {
			continue
		}
		scope := tsExpressRouteProofScope(endpoint, ctx.facts)
		impl := endpointImplementation{
			EndpointID: endpoint.ID,
			SurfaceID:  "api",
			Language:   "typescript",
			Framework:  "express",
			Route: endpointRouteEvidence{
				Path:   endpoint.Path,
				Method: endpoint.Method,
				Status: scope.route.GetStatus(),
				Range:  scope.route.GetRange(),
			},
			Handler: endpointHandlerEvidence{
				Expression: scope.handler,
				Symbol:     scope.symbol,
				Enclosing:  scope.enclosing,
			},
			Payloads: map[payloadRole]payloadImplementationEvidence{},
			Evidence: []*factsv1.Evidence{
				scope.route,
			},
		}
		for _, payload := range declaredPayloadRoles(endpoint) {
			ev := tsExpressPayloadEvidence(string(payload.role), payload.declaration, ctx.facts, scope)
			impl.Payloads[payload.role] = payloadImplementationEvidence{
				Role:          payload.role,
				ProtoFullName: payload.declaration.ProtoFullName,
				Transport:     payload.declaration.Transport,
				Status:        ev.GetStatus(),
				Range:         ev.GetRange(),
				Message:       ev.GetMessage(),
				Evidence:      ev,
			}
			impl.Evidence = append(impl.Evidence, ev)
		}
		out = append(out, impl)
	}
	return out
}

func (goEndpointAdapter) ExtractEndpointImplementations(ctx endpointAdapterContext) []endpointImplementation {
	var out []endpointImplementation
	for _, endpoint := range ctx.endpoints {
		if endpoint.RESTException == nil {
			continue
		}
		scope := routeProofScope(endpoint, ctx.facts)
		impl := endpointImplementation{
			EndpointID: endpoint.ID,
			SurfaceID:  "api",
			Language:   "go",
			Framework:  routeFramework(scope),
			Route: endpointRouteEvidence{
				Path:   endpoint.Path,
				Method: endpoint.Method,
				Status: scope.route.GetStatus(),
				Range:  scope.route.GetRange(),
			},
			Handler: endpointHandlerEvidence{
				Expression: scope.handler,
				Symbol:     scope.symbol,
				Enclosing:  scope.enclosing,
			},
			Payloads: map[payloadRole]payloadImplementationEvidence{},
			Evidence: []*factsv1.Evidence{
				scope.route,
			},
		}
		for _, payload := range declaredPayloadRoles(endpoint) {
			ev := goPayloadEvidence(string(payload.role), payload.declaration, ctx.facts, scope)
			impl.Payloads[payload.role] = payloadImplementationEvidence{
				Role:          payload.role,
				ProtoFullName: payload.declaration.ProtoFullName,
				Transport:     payload.declaration.Transport,
				Status:        ev.GetStatus(),
				Range:         ev.GetRange(),
				Message:       ev.GetMessage(),
				Evidence:      ev,
			}
			impl.Evidence = append(impl.Evidence, ev)
		}
		out = append(out, impl)
	}
	return out
}

func routeFramework(scope endpointProofScope) string {
	if scope.framework != "" {
		return scope.framework
	}
	return "go.http"
}

type declaredPayloadRole struct {
	role        payloadRole
	declaration payloadDeclaration
}

func declaredPayloadRoles(endpoint endpointDeclaration) []declaredPayloadRole {
	payloads := endpoint.RESTException.ProtoPayloads
	if payloads == nil {
		return nil
	}
	out := []declaredPayloadRole{
		{role: payloadRoleResponse, declaration: payloads.Response},
		{role: payloadRoleError, declaration: payloads.Error},
	}
	if payloads.Request.Transport != "" && payloads.Request.Conformance != "none" && payloads.Request.ProtoFullName != "" {
		out = append(out, declaredPayloadRole{role: payloadRoleRequest, declaration: payloads.Request})
	}
	return out
}
