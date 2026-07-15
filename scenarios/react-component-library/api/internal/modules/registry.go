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
	"react-component-library/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	localdb "react-component-library/internal/database"

	adoptionsH "react-component-library/handlers/adoptions"
	componentsH "react-component-library/handlers/components"
	depsH "react-component-library/handlers/deps"
	healthH "react-component-library/handlers/health"
	inventoryH "react-component-library/handlers/inventory"
	previewH "react-component-library/handlers/preview"
	themesH "react-component-library/handlers/themes"
	versionsH "react-component-library/handlers/versions"
	workflowsH "react-component-library/handlers/workflows"

	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
	depsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/deps"
	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	themesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/themes"
	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workflows"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, adoptionsH.Endpoints...)
	out = append(out, componentsH.Endpoints...)
	out = append(out, depsH.Endpoints...)
	out = append(out, healthH.Endpoints...)
	out = append(out, inventoryH.Endpoints...)
	out = append(out, previewH.Endpoints...)
	out = append(out, themesH.Endpoints...)
	out = append(out, versionsH.Endpoints...)
	out = append(out, workflowsH.Endpoints...)
	return out
}

// ProtoFileEntry pairs a domain module's name with the proto
// FileDescriptor whose RPCs that module exposes via Connect-RPC. The
// global parity test in registry_test.go walks every entry and asserts
// each rpc method in the FileDescriptor has exactly one matching
// EndpointDescriptor in AllEndpoints().
type ProtoFileEntry struct {
	Module string
	File   protoreflect.FileDescriptor
}

// AllProtoFiles returns the proto FileDescriptor backing each
// Connect-mounted domain module, in registration order.
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "adoptions", File: adoptionsv1.File_react_component_library_v1_adoptions_adoptions_proto},
		{Module: "components", File: componentsv1.File_react_component_library_v1_components_components_proto},
		{Module: "deps", File: depsv1.File_react_component_library_v1_deps_deps_proto},
		{Module: "inventory", File: inventoryv1.File_ui_health_v1_inventory_inventory_proto},
		{Module: "preview", File: previewv1.File_react_component_library_v1_preview_preview_proto},
		{Module: "themes", File: themesv1.File_react_component_library_v1_themes_themes_proto},
		{Module: "versions", File: versionsv1.File_react_component_library_v1_versions_versions_proto},
		{Module: "workflows", File: workflowsv1.File_react_component_library_v1_workflows_workflows_proto},
	}
}

// AllSchemas returns every domain's schema provider plus the system
// schema (always first; cross-cutting infrastructure runs before any
// domain table). Consumed by main.go's database.EnsureSchemas call.
//
// Order matters: system → domains (alphabetical).
// Postgres scenarios that put `CREATE EXTENSION ...` in system.sql rely
// on system running before any domain that references the extension.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(adoptionsH.Schema),
		apidb.SchemaProviderFunc(componentsH.Schema),
		apidb.SchemaProviderFunc(depsH.Schema),
		apidb.SchemaProviderFunc(healthH.Schema),
		apidb.SchemaProviderFunc(themesH.Schema),
		apidb.SchemaProviderFunc(workflowsH.Schema),
	}
}
