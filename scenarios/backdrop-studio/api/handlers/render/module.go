package render

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/module"
	internalrender "backdrop-studio/internal/render"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	render_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/render"
	renderconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/render/render_v1connect"
	shared_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/shared"
)

type handler struct {
	store   *internalrender.Store
	catalog *catalog.Store
}

func Module(db *sql.DB, store *internalrender.Store) module.Module {
	h := &handler{store: store, catalog: catalog.NewStore(db)}
	return module.Module{Name: "render", Mount: func(r *mux.Router) {
		path, svc := renderconnect.NewRenderServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

func (h *handler) Submit(ctx context.Context, req *connect.Request[render_v1.SubmitRequest]) (*connect.Response[render_v1.RenderJob], error) {
	if req.Msg.GetStyle() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("render: style is required"))
	}
	s := req.Msg.GetStyle()
	style := catalog.Style{ID: s.GetId(), Strategy: s.GetStrategy(), Subject: s.GetSubject(), Placements: s.GetPlacements(), Regions: regions(s.GetRegions()), Treatments: s.GetTreatments()}
	if s.GetScaffold() != nil {
		style.Scaffold = &catalog.ScaffoldBinding{Preset: s.GetScaffold().GetPreset(), Conditioner: s.GetScaffold().GetConditioner(), ParamsJSON: s.GetScaffold().GetParamsJson()}
	}
	if s.GetGeneration() != nil {
		style.Generation = &catalog.GenerationBlock{Role: s.GetGeneration().GetRole(), Profile: s.GetGeneration().GetProfile(), PromptTemplate: s.GetGeneration().GetPromptTemplate(), Negative: s.GetGeneration().GetNegative(), Model: s.GetGeneration().GetModel(), ProviderURL: s.GetGeneration().GetProviderUrl(), Credential: s.GetGeneration().GetCredential()}
	}
	if style.Subject == "" {
		stored, err := h.catalog.GetStyle(ctx, style.ID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		style = stored
	}
	job, err := h.store.SubmitWithContext(ctx, style, req.Msg.GetPlacement(), req.Msg.GetSeed(), int(req.Msg.GetCandidateCount()), req.Msg.GetBrandTokens())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(toProto(job)), nil
}

func (h *handler) GetJob(_ context.Context, req *connect.Request[render_v1.GetJobRequest]) (*connect.Response[render_v1.RenderJob], error) {
	job, err := h.store.Get(req.Msg.GetJobId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(toProto(job)), nil
}
func (h *handler) ListCandidates(_ context.Context, req *connect.Request[render_v1.ListCandidatesRequest]) (*connect.Response[render_v1.ListCandidatesResponse], error) {
	job, err := h.store.Get(req.Msg.GetJobId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	out := &render_v1.ListCandidatesResponse{}
	for _, c := range job.Candidates {
		out.Candidates = append(out.Candidates, candidateProto(c))
	}
	return connect.NewResponse(out), nil
}
func (h *handler) SelectCandidate(_ context.Context, req *connect.Request[render_v1.SelectCandidateRequest]) (*connect.Response[render_v1.RenderJob], error) {
	job, err := h.store.Select(req.Msg.GetJobId(), req.Msg.GetCandidateId(), req.Msg.GetActor())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(toProto(job)), nil
}

func regions(in []*shared_v1.ReservedRegion) []catalog.Region {
	out := make([]catalog.Region, 0, len(in))
	for _, r := range in {
		out = append(out, catalog.Region{X: r.GetX(), Y: r.GetY(), Width: r.GetWidth(), Height: r.GetHeight(), Kind: r.GetKind(), TextColor: r.GetTextColor()})
	}
	return out
}
func toProto(job internalrender.Job) *render_v1.RenderJob {
	out := &render_v1.RenderJob{Id: job.ID, StyleId: job.StyleID, Status: job.Status, Seed: job.Seed, ExecutionPath: job.ExecutionPath, SelectedCandidateId: job.SelectedCandidateID, SelectedBy: job.SelectedBy}
	for _, c := range job.Candidates {
		out.Candidates = append(out.Candidates, candidateProto(c))
	}
	return out
}
func candidateProto(c internalrender.Candidate) *render_v1.Candidate {
	return &render_v1.Candidate{Id: c.ID, JobId: c.JobID, ImagePng: c.PNG, Width: int32(c.Width), Height: int32(c.Height), Strategy: c.Strategy, ExecutionPath: c.ExecutionPath, TreatmentApplied: c.TreatmentApplied, Seed: c.Seed, ConditioningSubmitted: c.ConditioningSubmitted, DisclosureRequired: c.DisclosureRequired, Prompt: c.Prompt, ProvenanceJson: c.ProvenanceJSON}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "render_submit", Path: renderconnect.RenderServiceSubmitProcedure, Method: http.MethodPost, Summary: "Submit a reproducible render", Category: "render"},
	{ID: "render_job_get", Path: renderconnect.RenderServiceGetJobProcedure, Method: http.MethodPost, Summary: "Get render job status", Category: "render"},
	{ID: "render_candidates_list", Path: renderconnect.RenderServiceListCandidatesProcedure, Method: http.MethodPost, Summary: "List render candidates", Category: "render"},
	{ID: "render_candidate_select", Path: renderconnect.RenderServiceSelectCandidateProcedure, Method: http.MethodPost, Summary: "Select a render candidate", Category: "render"},
}
