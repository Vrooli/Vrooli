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
	internalvalidation "plan-manager/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	authoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/authoring/authoring_v1connect"
)

// Module returns the authoring domain's contribution to the API: the generated
// AuthoringService Connect-RPC handler, backed by the owned SQLite session store,
// the production discovery seams (each a LookPath-guarded CommandRunner shelling
// git-control-tower / search-hub, degrading gracefully when the dependency is
// absent), and a PlanWriter adapter that writes the produced plan
// THROUGH the plans domain. All wired here at the production edge; never imported
// into internal/authoring.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger, storePath string) module.Module {
	maturity := internalplans.NewFilesystemMaturityReader()
	plansSvc := internalplans.NewService(internalplans.Deps{
		Repo:  internalplans.NewSQLiteRepository(db, clk),
		Clock: clk,
		// The SAME mirror store the plans module wires: finalize publishes the
		// durable markdown mirror exactly like a direct plans Create. Omitting
		// it here was the silent-mirror hole — wizard-finalized plans got no
		// mirror file and a default "unknown" status.
		Mirror:   internalplans.NewDefaultOSMirrorStore(),
		Maturity: maturity,
	})
	runner := internalauthoring.DefaultRunner()
	svc := internalauthoring.NewService(internalauthoring.Deps{
		Store:     internalauthoring.NewSQLiteStore(db, clk),
		Writer:    planWriter{svc: plansSvc},
		Reader:    planWriter{svc: plansSvc},
		Anchor:    internalauthoring.DefaultAnchorIntentDeriver(),
		Skills:    internalauthoring.NewCommandSkillPackDiscoverer(runner),
		Commands:  newCLIHealthCommandValidator(),
		Resolver:  internalvalidation.NewFileResolver(""),
		Renderer:  planRenderer{},
		Posture:   posturePreparer{maturity: maturity},
		StorePath: storePath,
		Clock:     clk,
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

func (w planWriter) GetPlan(ctx context.Context, idOrSlug string) (internalplans.Plan, error) {
	return w.svc.Get(ctx, idOrSlug, internalplans.WorkspaceScope{})
}

func (w planWriter) RenderPlan(ctx context.Context, idOrSlug string) (string, error) {
	rendered, err := w.svc.Render(ctx, idOrSlug, internalplans.WorkspaceScope{}, internalplans.RenderOptions{})
	if err != nil {
		return "", err
	}
	return rendered.Markdown, nil
}

// planRenderer adapts the plans-domain deterministic renderer to the authoring
// PlanRenderer seam so the wizard can offer a render-preview before finalize.
// It is the SAME renderer the plans domain uses (no second renderer).
type planRenderer struct{}

func (planRenderer) Render(p internalplans.Plan) string {
	return internalplans.RenderMarkdown(p)
}

func (planRenderer) RenderDraft(p internalplans.Plan, sessionID string) string {
	return internalplans.RenderMarkdownWithOptions(p, internalplans.RenderOptions{AuthoringSessionID: sessionID})
}

// posturePreparer adapts the plans-domain posture resolver to authoring's
// PosturePreparer seam so the wizard's PreviewPlan stamps the SAME work posture
// (greenfield default OR brownfield from scenario maturity) that finalize/Create
// applies — preview and persisted render agree. It uses the same MaturityReader
// the plans service uses.
type posturePreparer struct{ maturity internalplans.MaturityReader }

func (p posturePreparer) PreparePosture(ctx context.Context, plan internalplans.Plan) internalplans.Plan {
	posture, source, detail := internalplans.ResolvePosture(ctx, plan, p.maturity)
	plan.WorkPosture = posture
	plan.WorkPostureSource = source
	plan.WorkPostureDetail = detail
	return plan
}

// Endpoints is the machine-readable description of the authoring module's public
// surface; one entry per RPC (the global parity test enforces this).
var Endpoints = []module.EndpointDescriptor{
	endpoint("authoring_start_session", authoringconnect.AuthoringServiceStartSessionProcedure, "Start an authoring session", "Begins a guided authoring session for a plan title/slug, seeding the section skeleton (PM-AUTHOR-001)."),
	endpoint("authoring_get_session", authoringconnect.AuthoringServiceGetSessionProcedure, "Get full session state", "Explicit full-state read: returns the whole authoring session graph. Normal mutations return only focused progress + a mutation summary, so callers hydrate full state deliberately here."),
	endpoint("authoring_get_section", authoringconnect.AuthoringServiceGetSectionProcedure, "Get a section", "Returns one section's current state."),
	endpoint("authoring_submit_section", authoringconnect.AuthoringServiceSubmitSectionProcedure, "Submit a section", "Records authored content for a section and re-validates it (PM-AUTHOR-002)."),
	endpoint("authoring_submit_fields", authoringconnect.AuthoringServiceSubmitFieldsProcedure, "Submit fields in batch", "Applies a batch of section/phase-field writes under one session lock with one save: per-item independent apply with accepted/rejected + violations + a parse summary per item — one call can carry a single field, a whole phase, or the whole plan."),
	endpoint("authoring_next", authoringconnect.AuthoringServiceNextProcedure, "Next section", "Returns the next section that still needs author input, or signals the session is structurally complete."),
	endpoint("authoring_continue", authoringconnect.AuthoringServiceContinueAuthoringProcedure, "Continue authoring", "Returns the single recommended next wizard action across sections, phases, validation review, and finalize states."),
	endpoint("authoring_validate_structure", authoringconnect.AuthoringServiceValidateStructureProcedure, "Validate structure", "Runs the structure-validation gate over the whole session — rejects empty mandatory sections + an empty regression anchor (PM-AUTHOR-002)."),
	endpoint("authoring_autofill", authoringconnect.AuthoringServiceAutofillProcedure, "Autofill mechanical sections", "Orchestrates the mechanical-section autofill behind seams (regression anchor / required reading / references). Degrades gracefully, never a false fill (OT-P0-002)."),
	endpoint("authoring_submit_relevant_context_item", authoringconnect.AuthoringServiceSubmitRelevantContextItemProcedure, "Submit relevant context", "Records one global or phase-scoped setup item in the authoring session."),
	endpoint("authoring_list_relevant_context", authoringconnect.AuthoringServiceListRelevantContextProcedure, "List relevant context", "Returns accepted context items from the authoring session without changing wizard position."),
	endpoint("authoring_update_relevant_context_item", authoringconnect.AuthoringServiceUpdateRelevantContextItemProcedure, "Update relevant context item", "Replaces one accepted global or phase-scoped context item in place (by id) so a bad item found in preview is corrected without deleting the phase/session. Legal only before finalize."),
	endpoint("authoring_remove_relevant_context_item", authoringconnect.AuthoringServiceRemoveRelevantContextItemProcedure, "Remove relevant context item", "Deletes one accepted context item (by id) before finalize, recomputing structure violations so a resulting gate is reported with its recovery step."),
	endpoint("authoring_discover_skill_pack", authoringconnect.AuthoringServiceDiscoverSkillPackProcedure, "Discover skill pack", "Runs prompt-manager skill discovery for decomposed concepts and auto-upserts the returned skill pack into global relevant context."),
	endpoint("authoring_add_phase", authoringconnect.AuthoringServiceAddPhaseProcedure, "Add phase draft", "Appends one structured phase draft so agents do not submit all phases as a markdown blob."),
	endpoint("authoring_move_phase", authoringconnect.AuthoringServiceMovePhaseProcedure, "Move phase draft", "Reorders one structured phase draft before or after another without rewriting authored phase content."),
	endpoint("authoring_get_phase", authoringconnect.AuthoringServiceGetPhaseProcedure, "Get phase draft", "Returns one structured phase draft plus the API-owned guided step for the next missing phase field."),
	endpoint("authoring_submit_phase_field", authoringconnect.AuthoringServiceSubmitPhaseFieldProcedure, "Submit phase field", "Records one phase-native field and validates references/acceptance immediately."),
	endpoint("authoring_next_phase", authoringconnect.AuthoringServiceNextPhaseProcedure, "Next phase draft", "Returns the first structured phase draft that still needs author input."),
	endpoint("authoring_preview_plan", authoringconnect.AuthoringServicePreviewPlanProcedure, "Preview the rendered plan", "Renders the in-progress session to its markdown review artifact without persisting, so the plan can be reviewed before finalize."),
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
