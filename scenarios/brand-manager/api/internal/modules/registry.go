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
	"brand-manager/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	applyH "brand-manager/handlers/apply"
	assetsH "brand-manager/handlers/assets"
	assignmentsH "brand-manager/handlers/assignments"
	brandsH "brand-manager/handlers/brands"
	designH "brand-manager/handlers/design"
	discoveryH "brand-manager/handlers/discovery"
	generationH "brand-manager/handlers/generation"
	healthH "brand-manager/handlers/health"
	notesH "brand-manager/handlers/notes" // EXAMPLE-DOMAIN:notes
	localdb "brand-manager/internal/database"

	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/apply"
	assetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assets"
	assignmentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments"
	brandsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands"
	designv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/design"
	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/discovery"
	generationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/generation"
	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/notes" // EXAMPLE-DOMAIN:notes
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, applyH.Endpoints...)
	out = append(out, assetsH.Endpoints...)
	out = append(out, assignmentsH.Endpoints...)
	out = append(out, brandsH.Endpoints...)
	out = append(out, designH.Endpoints...)
	out = append(out, discoveryH.Endpoints...)
	out = append(out, generationH.Endpoints...)
	out = append(out, notesH.Endpoints...) // EXAMPLE-DOMAIN:notes
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
		{Module: "apply", File: applyv1.File_brand_manager_v1_apply_apply_proto},
		{Module: "assets", File: assetsv1.File_brand_manager_v1_assets_assets_proto},
		{Module: "assignments", File: assignmentsv1.File_brand_manager_v1_assignments_assignments_proto},
		{Module: "brands", File: brandsv1.File_brand_manager_v1_brands_brands_proto},
		{Module: "design", File: designv1.File_brand_manager_v1_design_design_proto},
		{Module: "discovery", File: discoveryv1.File_brand_manager_v1_discovery_discovery_proto},
		{Module: "generation", File: generationv1.File_brand_manager_v1_generation_generation_proto},
		{Module: "notes", File: notesv1.File_brand_manager_v1_notes_notes_proto}, // EXAMPLE-DOMAIN:notes
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
		apidb.SchemaProviderFunc(assetsH.Schema),
		apidb.SchemaProviderFunc(assignmentsH.Schema),
		apidb.SchemaProviderFunc(brandsH.Schema),
		apidb.SchemaProviderFunc(notesH.Schema), // EXAMPLE-DOMAIN:notes
	}
}
