// Package modules is the single registration point for the scenario's
// API modules' static metadata. Both api/main.go and
// api/cmd/gen-endpoints/main.go import this package to enumerate
// domains uniformly.
//
// The runtime Module(...) constructors stay inline in main.go's
// server.New(...) call — they need live deps (db handle, clock, logger)
// and abstracting them is needless ceremony. This package only handles
// the static side: the Endpoints slice each handler exports for
// codegen, and the Schema() function each handler re-exports for
// EnsureSchemas.
//
// Adding a domain: add two lines below — one in AllEndpoints, one in
// AllSchemas. The runtime constructor lands in main.go's server.New
// call as a third line. Three central lines per new domain, no other
// central registry mutations.
package modules

import (
	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/module"

	capsH "backdrop-studio/handlers/capabilities"
	catalogH "backdrop-studio/handlers/catalog"
	composeH "backdrop-studio/handlers/compose"
	generatorsH "backdrop-studio/handlers/generators"
	legibilityH "backdrop-studio/handlers/legibility"
	releaseH "backdrop-studio/handlers/release"
	renderH "backdrop-studio/handlers/render"
	scaffoldH "backdrop-studio/handlers/scaffold"
	surfacesH "backdrop-studio/handlers/surfaces"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	healthH "backdrop-studio/handlers/health"
	localdb "backdrop-studio/internal/database"

	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/catalog"
	composev1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/compose"
	generatorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/generators"
	legibilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/legibility"
	releasev1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release"
	renderv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/render"
	scaffoldv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/scaffold"
	surfacesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/surfaces"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, capsH.Endpoints...)
	out = append(out, catalogH.Endpoints...)
	out = append(out, composeH.Endpoints...)
	out = append(out, generatorsH.Endpoints...)
	out = append(out, renderH.Endpoints...)
	out = append(out, legibilityH.Endpoints...)
	out = append(out, releaseH.Endpoints...)
	out = append(out, scaffoldH.Endpoints...)
	out = append(out, surfacesH.Endpoints...)
	return out
}

// ProtoFileEntry pairs a domain module's name with the proto
// FileDescriptor whose RPCs that module exposes via Connect-RPC. The
// global parity test in registry_test.go walks every entry and asserts
// each rpc method in the FileDescriptor has exactly one matching
// EndpointDescriptor in AllEndpoints().
//
// Adding a Connect-RPC domain: append one line below. The global parity
// test then covers it automatically — there is no per-domain parity
// test to write.
//
// REST-exception-only domains (none in the template today) are simply
// not listed here; the global test never inspects them, and the
// gen-endpoints validateTransport pass enforces their RESTException
// tags at codegen time.
type ProtoFileEntry struct {
	Module string
	File   protoreflect.FileDescriptor
}

// AllProtoFiles returns the proto FileDescriptor backing each
// Connect-mounted domain module, in registration order.
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "catalog", File: catalogv1.File_backdrop_studio_v1_catalog_catalog_proto},
		{Module: "compose", File: composev1.File_backdrop_studio_v1_compose_compose_proto},
		{Module: "generators", File: generatorsv1.File_backdrop_studio_v1_generators_generators_proto},
		{Module: "render", File: renderv1.File_backdrop_studio_v1_render_render_proto},
		{Module: "legibility", File: legibilityv1.File_backdrop_studio_v1_legibility_legibility_proto},
		{Module: "release", File: releasev1.File_backdrop_studio_v1_release_release_proto},
		{Module: "scaffold", File: scaffoldv1.File_backdrop_studio_v1_scaffold_scaffold_proto},
		{Module: "surfaces", File: surfacesv1.File_backdrop_studio_v1_surfaces_surfaces_proto},
	}
}

// AllSchemas returns every domain's schema provider plus the system
// schema (always first; cross-cutting infrastructure runs before any
// domain table). Consumed by main.go's database.EnsureSchemas call.
//
// Order matters: system → health → … (domains alphabetical).
// Postgres scenarios that put `CREATE EXTENSION ...` in system.sql rely
// on system running before any domain that references the extension.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(healthH.Schema),
		apidb.SchemaProviderFunc(catalog.Schema),
	}
}
