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
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/designcapture"
	"react-component-library/internal/designcritique"
	"react-component-library/internal/designinference"
	"react-component-library/internal/module"
	"react-component-library/internal/versionledger"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	localdb "react-component-library/internal/database"

	activityH "react-component-library/handlers/activity"
	adoptionsH "react-component-library/handlers/adoptions"
	brandH "react-component-library/handlers/brand"
	catalogH "react-component-library/handlers/catalog"
	componentsH "react-component-library/handlers/components"
	componentTestsH "react-component-library/handlers/componenttests"
	depsH "react-component-library/handlers/deps"
	healthH "react-component-library/handlers/health"
	inventoryH "react-component-library/handlers/inventory"
	libraryhealthH "react-component-library/handlers/libraryhealth"
	pageH "react-component-library/handlers/page"
	previewH "react-component-library/handlers/preview"
	reviewH "react-component-library/handlers/review"
	searchH "react-component-library/handlers/search"
	sketchH "react-component-library/handlers/sketch"
	themesH "react-component-library/handlers/themes"
	versionsH "react-component-library/handlers/versions"
	workflowsH "react-component-library/handlers/workflows"
	workspacesH "react-component-library/handlers/workspaces"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/activity"
	adoptionintakev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptionintake"
	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
	componenttestsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests"
	depsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/deps"
	experiencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/experience"
	libraryhealthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/libraryhealth"
	pagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/page"
	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
	themesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/themes"
	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workflows"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	workspaceassistv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspaceassist"
	workspacecandidatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacecandidate"
	workspacecapturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacecapture"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, activityH.Endpoints...)
	out = append(out, adoptionsH.Endpoints...)
	out = append(out, brandH.Endpoints...)
	out = append(out, catalogH.Endpoints...)
	out = append(out, componentsH.Endpoints...)
	out = append(out, componentTestsH.Endpoints...)
	out = append(out, depsH.Endpoints...)
	out = append(out, healthH.Endpoints...)
	out = append(out, inventoryH.Endpoints...)
	out = append(out, libraryhealthH.Endpoints...)
	out = append(out, pageH.Endpoints...)
	out = append(out, previewH.Endpoints...)
	out = append(out, reviewH.Endpoints...)
	out = append(out, searchH.Endpoints...)
	out = append(out, sketchH.Endpoints...)
	out = append(out, themesH.Endpoints...)
	out = append(out, versionsH.Endpoints...)
	out = append(out, workflowsH.Endpoints...)
	out = append(out, workspacesH.Endpoints...)
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
		{Module: "activity", File: activityv1.File_react_component_library_v1_activity_activity_proto},
		{Module: "adoption-intake", File: adoptionintakev1.File_react_component_library_v1_adoptionintake_adoption_intake_proto},
		{Module: "adoptions", File: adoptionsv1.File_react_component_library_v1_adoptions_adoptions_proto},
		{Module: "catalog", File: catalogv1.File_react_component_library_v1_catalog_catalog_proto},
		{Module: "components", File: componentsv1.File_react_component_library_v1_components_components_proto},
		{Module: "component-tests", File: componenttestsv1.File_react_component_library_v1_componenttests_component_tests_proto},
		{Module: "deps", File: depsv1.File_react_component_library_v1_deps_deps_proto},
		{Module: "experience", File: experiencev1.File_react_component_library_v1_experience_experience_proto},
		{Module: "inventory", File: inventoryv1.File_ui_health_v1_inventory_inventory_proto},
		{Module: "library-health", File: libraryhealthv1.File_react_component_library_v1_libraryhealth_library_health_proto},
		{Module: "page", File: pagev1.File_react_component_library_v1_page_page_proto},
		{Module: "preview", File: previewv1.File_react_component_library_v1_preview_preview_proto},
		{Module: "review", File: reviewv1.File_react_component_library_v1_review_review_proto},
		{Module: "search", File: searchv1.File_react_component_library_v1_search_search_proto},
		{Module: "themes", File: themesv1.File_react_component_library_v1_themes_themes_proto},
		{Module: "versions", File: versionsv1.File_react_component_library_v1_versions_versions_proto},
		{Module: "versions-lifecycle", File: versionsv1.File_react_component_library_v1_versions_lifecycle_proto},
		{Module: "workflows", File: workflowsv1.File_react_component_library_v1_workflows_workflows_proto},
		{Module: "workspaces", File: workspacev1.File_react_component_library_v1_workspace_workspace_proto},
		{Module: "workspace-assist", File: workspaceassistv1.File_react_component_library_v1_workspaceassist_workspace_assist_proto},
		{Module: "workspace-candidate", File: workspacecandidatev1.File_react_component_library_v1_workspacecandidate_workspace_candidate_proto},
		{Module: "workspace-capture", File: workspacecapturev1.File_react_component_library_v1_workspacecapture_workspace_capture_proto},
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
		apidb.SchemaProviderFunc(catalogcoverage.Schema),
		apidb.SchemaProviderFunc(designcapture.Schema),
		apidb.SchemaProviderFunc(designcritique.Schema),
		apidb.SchemaProviderFunc(designinference.Schema),
		apidb.SchemaProviderFunc(versionledger.Schema),
		apidb.SchemaProviderFunc(activityH.Schema),
		apidb.SchemaProviderFunc(adoptionsH.Schema),
		apidb.SchemaProviderFunc(componentsH.Schema),
		apidb.SchemaProviderFunc(depsH.Schema),
		apidb.SchemaProviderFunc(healthH.Schema),
		apidb.SchemaProviderFunc(reviewH.Schema),
		apidb.SchemaProviderFunc(searchH.Schema),
		apidb.SchemaProviderFunc(themesH.Schema),
		apidb.SchemaProviderFunc(workflowsH.Schema),
		apidb.SchemaProviderFunc(workspacesH.Schema),
	}
}
