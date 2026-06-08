package manifestvalidation

import (
	"context"

	measures "github.com/vrooli/measures-go"
)

// MeasureSchemaReader resolves the proto-derived param schema for a measure
// command's binding (the Phase 0 descriptor reader). measurescan.SchemaSource
// and measurescan.DescriptorSchemaReader satisfy it. The seam is optional: when
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
}

type ProtoService struct {
	Name    string
	Methods []string
}

// HasMethod reports whether the named service (own OR shared) exposes the named
// method.
func (p ProtoSurface) HasMethod(service, method string) bool {
	return hasMethodIn(p.Services, service, method) || hasMethodIn(p.Shared, service, method)
}

func hasMethodIn(svcs []ProtoService, service, method string) bool {
	for _, s := range svcs {
		if s.Name != service {
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

// HasService reports whether the named service exists in any proto file (own OR
// shared).
func (p ProtoSurface) HasService(service string) bool {
	for _, s := range p.Services {
		if s.Name == service {
			return true
		}
	}
	for _, s := range p.Shared {
		if s.Name == service {
			return true
		}
	}
	return false
}
