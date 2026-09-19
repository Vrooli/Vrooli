package candidates

import (
	"log"
	"net/http"
	"time"

	internalcandidates "brand-manager/internal/candidates"
	"brand-manager/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	candsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/candidates/candidates_v1connect"
)

// imageJobDeadlineClearer removes the per-request server write deadline for the
// three procedures that wait on image-tools jobs. api-core installs a 30 s
// WriteTimeout, which is right for every ordinary request, but explore, refine
// and pick run for as long as generation, vectorize and rasterize take: past
// 30 s the finished response would be dropped and the client would see
// "unavailable: unexpected EOF" even though the work succeeded. The service
// bounds each explore round itself (exploreBudget).
type imageJobDeadlineClearer struct{ next http.Handler }

func (h imageJobDeadlineClearer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case candsconnect.CandidatesServiceExploreCandidatesProcedure,
		candsconnect.CandidatesServiceRefineCandidateProcedure,
		candsconnect.CandidatesServicePickCandidateProcedure:
		_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	}
	h.next.ServeHTTP(w, r)
}

// Module returns the candidates domain's contribution to the API.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger, svc *internalcandidates.Service) module.Module {
	connectPath, connectHandler := candsconnect.NewCandidatesServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "candidates",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: imageJobDeadlineClearer{next: connectHandler}})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the candidates schema.
func Schema() string { return internalcandidates.Schema() }

// Endpoints describes the candidates module's public surface.
var Endpoints = []module.EndpointDescriptor{
	{ID: "candidates_explore", Path: candsconnect.CandidatesServiceExploreCandidatesProcedure, Method: "POST", Summary: "Explore logo candidates", Description: "Generates N concepts × M variations as proposed candidates.", Category: "candidates", Request: &module.Schema{Type: "object", Properties: map[string]string{"brand_id": "string", "concepts": "array<string>", "variations": "int32"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"candidates": "array<LogoCandidate>"}}, Errors: []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Generation or storage failure"}}},
	{ID: "candidates_import", Path: candsconnect.CandidatesServiceImportCandidateProcedure, Method: "POST", Summary: "Import a candidate", Description: "Adopts an existing asset as a proposed candidate.", Category: "candidates", Request: &module.Schema{Type: "object", Properties: map[string]string{"brand_id": "string", "asset_id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"candidate": "LogoCandidate"}}, Errors: []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Asset not found"}}},
	{ID: "candidates_list", Path: candsconnect.CandidatesServiceListCandidatesProcedure, Method: "POST", Summary: "List candidates", Description: "Returns a brand's candidates, newest first.", Category: "candidates", Request: &module.Schema{Type: "object", Properties: map[string]string{"brand_id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"candidates": "array<LogoCandidate>"}}, Errors: []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Repository read failure"}}},
	{ID: "candidates_get", Path: candsconnect.CandidatesServiceGetCandidateProcedure, Method: "POST", Summary: "Get a candidate", Description: "Returns one candidate by id.", Category: "candidates", Request: &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"candidate": "LogoCandidate"}}, Errors: []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Candidate not found"}}},
	{ID: "candidates_refine", Path: candsconnect.CandidatesServiceRefineCandidateProcedure, Method: "POST", Summary: "Refine a candidate", Description: "Derives a new proposed candidate from a parent via instruction, mask, background removal or vectorize.", Category: "candidates", Request: &module.Schema{Type: "object", Properties: map[string]string{"candidate_id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"candidate": "LogoCandidate"}}, Errors: []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Image or storage failure"}}},
	{ID: "candidates_pick", Path: candsconnect.CandidatesServicePickCandidateProcedure, Method: "POST", Summary: "Pick a candidate", Description: "Promotes a candidate to the brand mark (vectorizing a raster pick first).", Category: "candidates", Request: &module.Schema{Type: "object", Properties: map[string]string{"candidate_id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"candidate": "LogoCandidate", "mark_asset_id": "string"}}, Errors: []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Candidate not found"}}},
	{ID: "candidates_reject", Path: candsconnect.CandidatesServiceRejectCandidateProcedure, Method: "POST", Summary: "Reject a candidate", Description: "Marks a candidate rejected.", Category: "candidates", Request: &module.Schema{Type: "object", Properties: map[string]string{"candidate_id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"candidate": "LogoCandidate"}}, Errors: []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Candidate not found"}}},
	{ID: "candidates_restore", Path: candsconnect.CandidatesServiceRestoreCandidateProcedure, Method: "POST", Summary: "Restore a candidate", Description: "Returns a rejected or superseded candidate to proposed.", Category: "candidates", Request: &module.Schema{Type: "object", Properties: map[string]string{"candidate_id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"candidate": "LogoCandidate"}}, Errors: []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Candidate not found"}}},
}
