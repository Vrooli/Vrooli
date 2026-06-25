// Package authoring is the API handler for the AuthoringService — the guided
// composer wizard. It is the proto translation edge over internal/authoring; all
// business logic lives there behind seams.
package authoring

import (
	"context"
	"log"

	internalauthoring "plan-manager/internal/authoring"
	"plan-manager/internal/clock"
	"plan-manager/internal/module"
	internalplans "plan-manager/internal/plans"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	authoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/authoring/authoring_v1connect"
)

// Module returns the authoring domain's contribution to the API: the generated
// AuthoringService Connect-RPC handler, backed by the owned SQLite session store,
// the production autofill seams (each a LookPath-guarded CommandRunner shelling
// git-control-tower / prompt-manager / code-facts, degrading gracefully when the
// dependency is absent), and a PlanWriter adapter that writes the produced plan
// THROUGH the plans domain. All wired here at the production edge; never imported
// into internal/authoring.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	plansSvc := internalplans.NewService(internalplans.Deps{
		Repo:  internalplans.NewSQLiteRepository(db, clk),
		Clock: clk,
	})
	runner := internalauthoring.DefaultRunner()
	svc := internalauthoring.NewService(internalauthoring.Deps{
		Store:        internalauthoring.NewSQLiteStore(db, clk),
		Writer:       planWriter{svc: plansSvc},
		Anchor:       internalauthoring.NewCommandAnchorAutofiller(runner),
		RequiredRead: internalauthoring.NewCommandRequiredReadingSource(runner),
		References:   internalauthoring.NewCommandReferenceExtractor(runner),
		Clock:        clk,
	})
	connectPath, connectHandler := authoringconnect.NewAuthoringServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "authoring",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the authoring domain's SQL contribution (the
// authoring_sessions table) so the modules registry collects endpoints and
// schema from one symbol per handler package.
func Schema() string { return internalauthoring.Schema() }

// planWriter adapts the plans domain Service to authoring's PlanWriter write
// seam (the method names differ; the types are the shared plans model). Finalize
// writes the produced plan THROUGH the plans SSOT — authoring never owns the
// record.
type planWriter struct{ svc internalplans.Service }

func (w planWriter) CreatePlan(ctx context.Context, p internalplans.Plan) (internalplans.Plan, error) {
	return w.svc.Create(ctx, p)
}

// Endpoints is the machine-readable description of the authoring module's public
// surface; one entry per RPC (the global parity test enforces this).
var Endpoints = []module.EndpointDescriptor{
	endpoint("authoring_start_session", authoringconnect.AuthoringServiceStartSessionProcedure, "Start an authoring session", "Begins a guided authoring session for a plan title/slug, seeding the section skeleton (PM-AUTHOR-001)."),
	endpoint("authoring_get_section", authoringconnect.AuthoringServiceGetSectionProcedure, "Get a section", "Returns one section's current state."),
	endpoint("authoring_submit_section", authoringconnect.AuthoringServiceSubmitSectionProcedure, "Submit a section", "Records authored content for a section and re-validates it (PM-AUTHOR-002)."),
	endpoint("authoring_next", authoringconnect.AuthoringServiceNextProcedure, "Next section", "Returns the next section that still needs author input, or signals the session is structurally complete."),
	endpoint("authoring_validate_structure", authoringconnect.AuthoringServiceValidateStructureProcedure, "Validate structure", "Runs the structure-validation gate over the whole session — rejects empty mandatory sections + an empty regression anchor (PM-AUTHOR-002)."),
	endpoint("authoring_autofill", authoringconnect.AuthoringServiceAutofillProcedure, "Autofill mechanical sections", "Orchestrates the mechanical-section autofill behind seams (regression anchor / required reading / references). Degrades gracefully, never a false fill (OT-P0-002)."),
	endpoint("authoring_finalize", authoringconnect.AuthoringServiceFinalizeProcedure, "Finalize the plan", "Runs the structure gate then writes the produced plan through the plans domain, returning the persisted plan."),
}

func endpoint(id, path, summary, description string) module.EndpointDescriptor {
	return module.EndpointDescriptor{
		ID:          id,
		Path:        path,
		Method:      "POST",
		Summary:     summary,
		Description: description,
		Category:    "authoring",
	}
}
