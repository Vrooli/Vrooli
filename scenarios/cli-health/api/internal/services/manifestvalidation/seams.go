package manifestvalidation

import (
	"context"
	"fmt"
	"strings"

	measures "github.com/vrooli/measures-go"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// MeasureSchemaReader resolves the proto-derived param schema for a measure
// command's binding (the Phase 0 descriptor reader). manifestscan.SchemaSource
// and manifestscan.DescriptorSchemaReader satisfy it. The seam is optional: when
// the service is constructed without one, measure validation is skipped (so a
// scenario with no measure blocks, and unit tests that don't exercise measures,
// are unaffected). Tests inject a stub to drive specific proto shapes.
type MeasureSchemaReader interface {
	RequestParams(service, method string) ([]measures.ParamSchema, error)
}

// ManifestLoader fetches the raw cli/manifest.json bytes for a scenario.
// Returns (nil, "", os.ErrNotExist) when the file is absent so the service
// can emit a `manifest.missing` warning without confusing it with I/O
// failures. Seam so tests can inject fixture bytes without a temp dir.
type ManifestLoader interface {
	Load(ctx context.Context, scenario string) (raw []byte, path string, err error)
}

// SchemaValidator JSON-schema-validates raw manifest bytes against
// .vrooli/schemas/cli-manifest.schema.json. Returns one finding per
// schema violation; nil if the manifest validates. Seam so unit tests can
// drive specific schema failures deterministically.
type SchemaValidator interface {
	Validate(ctx context.Context, raw []byte) ([]Finding, error)
}

// ProtoLoader loads the proto descriptor set for a scenario. The result is
// a flat list of services (service name + method names) so the validation
// service stays decoupled from descriptorpb. Seam so tests can stub the
// proto surface without invoking buf.
type ProtoLoader interface {
	Load(ctx context.Context, scenario string) (ProtoSurface, error)
}

// ProtoSurface is a slimmed view of a scenario's proto: just the services
// and their methods. Anything richer (message types, options) would belong
// in a v2 of this surface.
//
// Services are the scenario's OWN proto services — fully coverage-checked
// (every method must be bound or omitted). Shared are cross-scenario contracts
// the scenario may SERVE but does not own (today: the shared, token-gated
// search-hub.v1.control.SearchControlService that any search provider
// implements). A scenario may bind shared services from its CLI, but it is not
// required to expose every shared RPC — so shared services are bindable but NOT
// orphan-checked.
type ProtoSurface struct {
	Services []ProtoService
	Shared   []ProtoService
	// Requests carries descriptor-backed request messages keyed by
	// Service.Method. It is optional so lightweight validator seams can keep
	// using service-only fixtures.
	Requests          map[string]protoreflect.MessageDescriptor
	RequestCandidates map[string][]ProtoRequestCandidate
}

// ProtoRequestCandidate keeps enough provenance to apply the same
// own-package-first rule as program-runtime when short service names collide.
type ProtoRequestCandidate struct {
	Request protoreflect.MessageDescriptor
	Source  string
	Shared  bool
}

type ProtoService struct {
	Name     string
	FullName string
	Methods  []string
}

// HasMethod reports whether the named service (own OR shared) exposes the named
// method.
func (p ProtoSurface) HasMethod(service, method string) bool {
	return hasMethodIn(p.Services, service, method) || hasMethodIn(p.Shared, service, method)
}

func hasMethodIn(svcs []ProtoService, service, method string) bool {
	for _, s := range svcs {
		if s.Name != service && s.FullName != service {
			continue
		}
		for _, m := range s.Methods {
			if m == method {
				return true
			}
		}
	}
	return false
}

// HasAnyMethod reports whether the scenario declares any of its OWN proto RPC
// methods. Shared cross-scenario services it merely re-serves are excluded, so
// this answers "does this scenario have a proto-driven CLI surface of its own
// that a cli/manifest.json must cover?".
func (p ProtoSurface) HasAnyMethod() bool {
	for _, s := range p.Services {
		if len(s.Methods) > 0 {
			return true
		}
	}
	return false
}

// HasService reports whether the named service exists in any proto file (own OR
// shared).
func (p ProtoSurface) HasService(service string) bool {
	for _, s := range p.Services {
		if s.Name == service || s.FullName == service {
			return true
		}
	}
	for _, s := range p.Shared {
		if s.Name == service || s.FullName == service {
			return true
		}
	}
	return false
}

// ResolveRequest applies own-package-first resolution to a short service and
// method name. It returns an error naming all candidates when the declaration
// is ambiguous rather than silently selecting descriptor iteration order.
func (p ProtoSurface) ResolveRequest(service, method string) (protoreflect.MessageDescriptor, error) {
	candidates := p.RequestCandidates[service+"."+method]
	if len(candidates) == 0 {
		return p.Requests[service+"."+method], nil
	}
	var own, shared []ProtoRequestCandidate
	for _, candidate := range candidates {
		if candidate.Shared {
			shared = append(shared, candidate)
		} else {
			own = append(own, candidate)
		}
	}
	selected := own
	if len(selected) == 0 {
		selected = shared
	}
	if len(selected) > 1 {
		labels := make([]string, 0, len(selected))
		for _, candidate := range selected {
			labels = append(labels, candidate.Source)
		}
		return nil, fmt.Errorf("service %s.%s is ambiguous across %s", service, method, strings.Join(labels, ", "))
	}
	if len(selected) == 1 {
		return selected[0].Request, nil
	}
	return nil, nil
}
