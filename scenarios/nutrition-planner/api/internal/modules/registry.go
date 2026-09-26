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
	costH "nutrition-planner/handlers/cost"
	inventoryH "nutrition-planner/handlers/inventory"
	inventoryDomain "nutrition-planner/internal/inventory"
	"nutrition-planner/internal/module"
	supplementDomain "nutrition-planner/internal/supplement"

	capsH "nutrition-planner/handlers/capabilities"
	catalogH "nutrition-planner/handlers/catalog"
	diagnosticsH "nutrition-planner/handlers/diagnostics"
	eligibilityH "nutrition-planner/handlers/eligibility"
	jobsH "nutrition-planner/handlers/jobs"
	nutritionH "nutrition-planner/handlers/nutrition"
	planningH "nutrition-planner/handlers/planning"
	portabilityH "nutrition-planner/handlers/portability"
	profileH "nutrition-planner/handlers/profile"
	recipeH "nutrition-planner/handlers/recipe"
	routineH "nutrition-planner/handlers/routine"
	supplementH "nutrition-planner/handlers/supplement"
	workspaceH "nutrition-planner/handlers/workspace"
	entitlementsDomain "nutrition-planner/internal/entitlements"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	healthH "nutrition-planner/handlers/health"
	localdb "nutrition-planner/internal/database"

	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/catalog"
	costv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/cost"
	eligibilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/eligibility"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/inventory"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/jobs"
	nutritionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/nutrition"
	planningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/planning"
	portabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/portability"
	profilev1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/profile"
	recipev1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/recipe"
	routinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/routine"
	supplementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/supplement"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/workspace"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, capsH.Endpoints...)
	out = append(out, diagnosticsH.Endpoints...)
	out = append(out, workspaceH.Endpoints...)
	out = append(out, recipeH.Endpoints...)
	out = append(out, catalogH.Endpoints...)
	out = append(out, costH.Endpoints...)
	out = append(out, inventoryH.Endpoints...)
	out = append(out, nutritionH.Endpoints...)
	out = append(out, supplementH.Endpoints...)
	out = append(out, routineH.Endpoints...)
	out = append(out, portabilityH.Endpoints...)
	out = append(out, profileH.Endpoints...)
	out = append(out, eligibilityH.Endpoints...)
	out = append(out, jobsH.Endpoints...)
	out = append(out, planningH.Endpoints...)
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
		{Module: "workspace", File: workspacev1.File_nutrition_planner_v1_workspace_workspace_proto},
		{Module: "recipe", File: recipev1.File_nutrition_planner_v1_recipe_recipe_proto},
		{Module: "catalog", File: catalogv1.File_nutrition_planner_v1_catalog_catalog_proto},
		{Module: "cost", File: costv1.File_nutrition_planner_v1_cost_cost_proto},
		{Module: "inventory", File: inventoryv1.File_nutrition_planner_v1_inventory_inventory_proto},
		{Module: "nutrition", File: nutritionv1.File_nutrition_planner_v1_nutrition_nutrition_proto},
		{Module: "supplement", File: supplementv1.File_nutrition_planner_v1_supplement_supplement_proto},
		{Module: "routine", File: routinev1.File_nutrition_planner_v1_routine_routine_proto},
		{Module: "portability", File: portabilityv1.File_nutrition_planner_v1_portability_portability_proto},
		{Module: "profile", File: profilev1.File_nutrition_planner_v1_profile_profile_proto},
		{Module: "eligibility", File: eligibilityv1.File_nutrition_planner_v1_eligibility_eligibility_proto},
		{Module: "jobs", File: jobsv1.File_nutrition_planner_v1_jobs_jobs_proto},
		{Module: "planning", File: planningv1.File_nutrition_planner_v1_planning_planning_proto},
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
		apidb.SchemaProviderFunc(workspaceH.Schema),
		apidb.SchemaProviderFunc(recipeH.Schema),
		apidb.SchemaProviderFunc(catalogH.Schema),
		apidb.SchemaProviderFunc(costH.Schema),
		apidb.SchemaProviderFunc(inventoryDomain.Schema),
		apidb.SchemaProviderFunc(nutritionH.Schema),
		apidb.SchemaProviderFunc(supplementDomain.Schema),
		apidb.SchemaProviderFunc(routineH.Schema),
		apidb.SchemaProviderFunc(profileH.Schema),
		apidb.SchemaProviderFunc(planningH.Schema),
		apidb.SchemaProviderFunc(portabilityH.Schema),
		apidb.SchemaProviderFunc(jobsH.Schema),
		apidb.SchemaProviderFunc(entitlementsDomain.Schema),
	}
}
