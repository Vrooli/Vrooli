package manifestvalidation

import "context"

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
type ProtoSurface struct {
	Services []ProtoService
}

type ProtoService struct {
	Name    string
	Methods []string
}

// HasMethod reports whether the named service exposes the named method.
func (p ProtoSurface) HasMethod(service, method string) bool {
	for _, s := range p.Services {
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

// HasService reports whether the named service exists in any proto file.
func (p ProtoSurface) HasService(service string) bool {
	for _, s := range p.Services {
		if s.Name == service {
			return true
		}
	}
	return false
}
