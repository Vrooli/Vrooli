// Package research is the API's L2/L3 deep-research Connect-RPC surface. It
// mounts the generated ResearchService handler and exports the static
// EndpointDescriptor for codegen. Like livesearch it owns no SQLite tables of
// its own (L2/L3 capture reuses the findings store), so it exposes no Schema().
package research

import (
	"log"

	"web-search/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	researchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/research/research_v1connect"

	"web-search/internal/evaluation"
	internalresearch "web-search/internal/research"
)

// Module returns the research domain's contribution to the API: the generated
// Connect-RPC service handler. main.go owns the fetcher / searcher / synthesizer
// / agent-manager / findings wiring and injects the constructed service; tests
// build their own service over fakes.
func Module(svc *internalresearch.Service, logger *log.Logger, registries ...*evaluation.MethodRegistry) module.Module {
	var registry *evaluation.MethodRegistry
	if len(registries) > 0 {
		registry = registries[0]
	}
	connectPath, connectHandler := researchconnect.NewResearchServiceHandler(NewConnectHandler(Deps{
		Service: svc, Registry: registry,
		Logger: logger,
	}))
	return module.Module{
		Name: "research",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Endpoints describes the research module's public surface. Each Connect-RPC
// method path references the generated *Procedure constant, so renaming an RPC
// in research.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{ID: "research_answer", Path: researchconnect.ResearchServiceAnswerProcedure, Method: "POST", Summary: "Answer with explicit evidence requirements", Description: "Reuse sufficient stored evidence before live research; return complete citations and typed gaps.", Category: "research"},
	{ID: "research_wait", Path: researchconnect.ResearchServiceWaitResearchProcedure, Method: "POST", Summary: "Wait once for research", Description: "Attach to the durable Agent Manager workflow without polling or canceling on timeout.", Category: "research"},
	{ID: "research_cancel", Path: researchconnect.ResearchServiceCancelResearchProcedure, Method: "POST", Summary: "Cancel research", Description: "Explicitly cancels the owner-held Agent Manager execution and preserves its terminal cause.", Category: "research"},
	{
		ID:          "research_l2",
		Path:        researchconnect.ResearchServiceRunL2Procedure,
		Method:      "POST",
		Summary:     "L2 fetch, read, and single-pass cited synthesis",
		Description: "Fetches the top-N result pages (HTTP-first, escalating to a browser-automation-studio capture for JS-heavy pages), extracts readable text, and produces one always-cited synthesis grounded in full page content (richer than L1 snippet synthesis, at higher latency). With --capture, distills the synthesis into the findings store (L2 capture is opt-in).",
		Category:    "research",
	},
	{
		ID:          "research_l3",
		Path:        researchconnect.ResearchServiceRunL3Procedure,
		Method:      "POST",
		Summary:     "Start an L3 iterative research-and-reconcile run",
		Description: "Spawns an agent-manager run that gathers nearby findings, researches the gap with L2 tools, and reconciles (distill, supersede outdated, flag low-confidence contradictions). Returns a run handle to poll. L3 auto-captures findings.",
		Category:    "research",
	},
	{
		ID:          "research_status",
		Path:        researchconnect.ResearchServiceGetResearchStatusProcedure,
		Method:      "POST",
		Summary:     "Poll an L3 research run",
		Description: "Returns the current status (and final summary when complete) of an L3 research run by id.",
		Category:    "research",
	},
	{
		ID:          "research_gather",
		Path:        researchconnect.ResearchServiceGatherRelatedFindingsProcedure,
		Method:      "POST",
		Summary:     "Bounded GATHER of findings near a query",
		Description: "Returns the findings semantically near the query for the GATHER step of the research-and-reconcile loop. The sweep is bounded server-side (hard cap 20) — the L3 agent uses this instead of a free-form findings search, so it never scans the whole store.",
		Category:    "research",
	},
	{ID: "research_capture_status", Path: researchconnect.ResearchServiceGetCaptureStatusProcedure, Method: "POST", Summary: "Inspect research attempt capture delivery", Description: "Returns bounded pending, delivered, and failed Memory capture counts.", Category: "research"},
	{ID: "research_evidence_receipt", Path: researchconnect.ResearchServiceGetEvidenceReceiptProcedure, Method: "POST", Summary: "Read an evidence receipt", Description: "Returns owner-issued receipt metadata by opaque receipt ID.", Category: "research"},
	{ID: "research_evidence_passage", Path: researchconnect.ResearchServiceGetEvidencePassageProcedure, Method: "POST", Summary: "Read an evidence passage", Description: "Returns a bounded, integrity-addressed passage by opaque passage ID.", Category: "research"},
	{ID: "research_evidence_assessment", Path: researchconnect.ResearchServiceGetEvidenceAssessmentProcedure, Method: "POST", Summary: "Read a claim assessment", Description: "Returns an owner-issued claim assessment by opaque assessment ID.", Category: "research"},
	{ID: "research_method_current", Path: researchconnect.ResearchServiceGetMethodReleaseProcedure, Method: "POST", Summary: "Read current research method release", Description: "Returns the owner-selected method revision and evaluation identity.", Category: "research"},
	{ID: "research_method_promote", Path: researchconnect.ResearchServicePromoteMethodProcedure, Method: "POST", Summary: "Promote an evaluated research method", Description: "Promotes only a matching passing evaluation under an explicit grant and current-hash compare-and-swap.", Category: "research"},
	{ID: "research_method_rollback", Path: researchconnect.ResearchServiceRollbackMethodProcedure, Method: "POST", Summary: "Rollback research method", Description: "Rolls back to a known release using current-hash compare-and-swap.", Category: "research"},
	{ID: "research_method_suspend", Path: researchconnect.ResearchServiceSuspendMethodProcedure, Method: "POST", Summary: "Suspend research method", Description: "Suspends a method revision while retaining its release history.", Category: "research"},
}
