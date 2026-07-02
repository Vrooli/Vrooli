// Package plans is the API handler for the PlansService — the structured-plan
// SSOT domain. It is the proto translation edge over internal/plans; all
// business logic lives in internal/plans behind seams.
package plans

import (
	"log"

	"plan-manager/internal/clock"
	"plan-manager/internal/module"
	internalplans "plan-manager/internal/plans"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans/plans_v1connect"
)

// Module returns the plans domain's contribution to the API: the generated
// PlansService Connect-RPC handler, backed by the SQLite plans store over the
// ~/.vrooli home store and the os-backed plan-source reader (the fallback
// import seam). Wired here at the production edge; never imported into
// internal/plans.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	svc := internalplans.NewService(internalplans.Deps{
		Repo:     internalplans.NewSQLiteRepository(db, clk),
		Clock:    clk,
		Reader:   internalplans.OSSourceReader{},
		Mirror:   internalplans.NewDefaultOSMirrorStore(),
		Maturity: internalplans.NewFilesystemMaturityReader(),
	})
	connectPath, connectHandler := plansconnect.NewPlansServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "plans",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalplans.Schema so the modules registry collects
// endpoints and schema from one symbol per handler package.
func Schema() string { return internalplans.Schema() }

// Endpoints is the machine-readable description of the plans module's public
// surface. Each Connect-RPC method path references a generated *Procedure
// constant, so renaming an RPC in plans.proto breaks this at compile time; the
// global parity test asserts every RPC has exactly one entry.
var Endpoints = []module.EndpointDescriptor{
	endpoint("plans_list", plansconnect.PlansServiceListPlansProcedure, "List plans", "Returns plans, optionally filtered by status (OT-P0-001)."),
	endpoint("plans_get", plansconnect.PlansServiceGetPlanProcedure, "Get a plan", "Returns one plan by id or slug."),
	endpoint("plans_create", plansconnect.PlansServiceCreatePlanProcedure, "Create a plan", "Persists a new structured plan; server assigns id/content-hash/status/timestamps."),
	endpoint("plans_update", plansconnect.PlansServiceUpdatePlanProcedure, "Update a plan", "Replaces authored fields; computed fields are recomputed server-side."),
	endpoint("plans_archive", plansconnect.PlansServiceArchivePlanProcedure, "Archive a plan", "Soft-archives a plan (kept, hidden by default)."),
	endpoint("plans_render", plansconnect.PlansServiceRenderMarkdownProcedure, "Render a plan to markdown", "Deterministically renders the structured record to its markdown projection."),
	endpoint("plans_add_phase", plansconnect.PlansServiceAddPhaseProcedure, "Add a phase", "Appends a first-class phase to a plan."),
	endpoint("plans_update_phase", plansconnect.PlansServiceUpdatePhaseProcedure, "Update a phase", "Replaces a phase's authored fields."),
	endpoint("plans_list_relevant_context", plansconnect.PlansServiceListRelevantContextProcedure, "List relevant context", "Returns accepted setup context for a plan or one phase."),
	endpoint("plans_update_relevant_context", plansconnect.PlansServiceUpdateRelevantContextProcedure, "Update relevant context", "Replaces one structured setup item without rewriting the whole plan."),
	endpoint("plans_remove_relevant_context", plansconnect.PlansServiceRemoveRelevantContextProcedure, "Remove relevant context", "Removes one structured setup item from a plan or phase."),
	endpoint("plans_list_references", plansconnect.PlansServiceListReferencesProcedure, "List references", "Returns structured code/doc/req references for a plan or one phase."),
	endpoint("plans_update_reference", plansconnect.PlansServiceUpdateReferenceProcedure, "Update reference", "Replaces one structured reference without rewriting the whole plan."),
	endpoint("plans_remove_reference", plansconnect.PlansServiceRemoveReferenceProcedure, "Remove reference", "Removes one structured reference from a plan or phase."),
	endpoint("plans_get_graph", plansconnect.PlansServiceGetGraphProcedure, "Get the plan graph", "Returns supersession/dependency edges, optionally scoped to one plan (OT-P1-002)."),
	endpoint("plans_link_supersession", plansconnect.PlansServiceLinkSupersessionProcedure, "Link supersession", "Records that one plan supersedes another (PM-GRAPH-001)."),
	endpoint("plans_link_dependency", plansconnect.PlansServiceLinkDependencyProcedure, "Link dependency", "Records that one plan depends on another."),
	endpoint("plans_import", plansconnect.PlansServiceImportPlanProcedure, "Import a markdown plan", "Adopts a markdown plan from a fallback read location into the structured model."),
	endpoint("plans_migrate", plansconnect.PlansServiceMigratePlanProcedure, "Migrate a plan to canonical", "Ensures a fallback-resolved plan resides in the canonical home store."),
	endpoint("plans_reconcile", plansconnect.PlansServiceReconcilePlansProcedure, "Reconcile plans", "Repairs rendered mirrors and non-destructively adopts legacy markdown plans in bulk."),
	endpoint("plans_list_templates", plansconnect.PlansServiceListTemplatesProcedure, "List plan templates", "Returns the per-surface plan templates (CLI/proto/UI)."),
	endpoint("plans_create_from_template", plansconnect.PlansServiceCreateFromTemplateProcedure, "Create from template", "Pre-scaffolds a plan from a template."),
}

// endpoint is a small constructor that keeps the Endpoints slice readable. All
// PlansService RPCs are POST Connect calls in the "plans" category.
func endpoint(id, path, summary, description string) module.EndpointDescriptor {
	return module.EndpointDescriptor{
		ID:          id,
		Path:        path,
		Method:      "POST",
		Summary:     summary,
		Description: description,
		Category:    "plans",
	}
}
