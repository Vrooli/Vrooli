package render

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"backdrop-studio/internal/brandpalette"
	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/imageengine"
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
		style.Generation = &catalog.GenerationBlock{PromptTemplate: s.GetGeneration().GetPromptTemplate(), Negative: s.GetGeneration().GetNegative(), Model: s.GetGeneration().GetModel(), ProviderURL: s.GetGeneration().GetProviderUrl(), Credential: s.GetGeneration().GetCredential()}
	}
	if style.Subject == "" {
		stored, err := h.catalog.GetStyle(ctx, style.ID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		style = stored
	}
	tokens, err := h.brandTokens(ctx, req.Msg.GetBrandId(), req.Msg.GetBrandTokens())
	if err != nil {
		return nil, err
	}
	surface, err := h.resolveSurface(ctx, style, req.Msg.GetSurfaceId())
	if err != nil {
		return nil, err
	}
	job, err := h.store.SubmitWithContext(ctx, internalrender.Request{
		Style:       style,
		Surface:     surface,
		Placement:   req.Msg.GetPlacement(),
		Seed:        req.Msg.GetSeed(),
		Count:       int(req.Msg.GetCandidateCount()),
		BrandTokens: tokens,
	})
	if err != nil {
		var rejected *internalrender.QualityRejectedError
		if errors.As(err, &rejected) {
			// The render succeeded and the result is unusable. Retrying is
			// pointless — the same style at the same seed produces the same
			// image — so this is a precondition failure, not a transient one.
			return nil, connect.NewError(connect.CodeFailedPrecondition, rejected)
		}
		var unresolved *imageengine.UnresolvedSlotError
		if errors.As(err, &unresolved) {
			// Neither a bound brand nor the style itself defines this ink, so
			// no retry helps: the caller must bind a brand that defines the
			// slot, or the catalog must declare a default for it.
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("render: style %q: %w", style.ID, err))
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(toProto(job)), nil
}

// resolveSurface picks the delivery geometry for one render.
//
// A named surface must exist and must permit one of the style's placements —
// rendering an App Store portrait for a style that only declares
// `feature_graphic` produces an asset no store will accept. When the caller
// names none, the first permitted surface is chosen and echoed back on the job
// so the choice is visible rather than assumed.
func (h *handler) resolveSurface(ctx context.Context, style catalog.Style, surfaceID string) (internalrender.Surface, error) {
	surfaces, err := h.catalog.ListSurfaces(ctx)
	if err != nil {
		return internalrender.Surface{}, connect.NewError(connect.CodeInternal, fmt.Errorf("render: list surfaces: %w", err))
	}
	if len(surfaces) == 0 {
		return internalrender.Surface{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("render: no surfaces are seeded, so there is no delivery geometry to render at"))
	}
	if id := strings.TrimSpace(surfaceID); id != "" {
		for _, surface := range surfaces {
			if surface.ID != id {
				continue
			}
			if !permitsAny(surface.Placements, style.Placements) {
				return internalrender.Surface{}, connect.NewError(connect.CodeInvalidArgument,
					fmt.Errorf("render: surface %q permits %v but style %q declares %v", id, surface.Placements, style.ID, style.Placements))
			}
			return internalrender.Surface{ID: surface.ID, Width: surface.Width, Height: surface.Height}, nil
		}
		return internalrender.Surface{}, connect.NewError(connect.CodeNotFound, fmt.Errorf("render: surface %q not found", id))
	}
	// No surface named: pick the largest permitted `product` surface, falling
	// back to the largest permitted surface of any kind.
	//
	// "First permitted, in id order" was the obvious rule and the wrong one —
	// it resolved a full-bleed landing-page style to `web.auth-panel` at
	// 640x900 purely because that id sorts early, so the committed render
	// matrix showed hero styles at a portrait panel size. This scenario exists
	// to make landing-page heroes; the default should say so.
	best, found := internalrender.Surface{}, false
	bestArea, bestIsProduct := 0, false
	for _, surface := range surfaces {
		if !permitsAny(surface.Placements, style.Placements) {
			continue
		}
		area := surface.Width * surface.Height
		isProduct := surface.Kind == "product"
		if found && !betterDefault(isProduct, area, bestIsProduct, bestArea) {
			continue
		}
		best = internalrender.Surface{ID: surface.ID, Width: surface.Width, Height: surface.Height}
		bestArea, bestIsProduct, found = area, isProduct, true
	}
	if found {
		return best, nil
	}
	return internalrender.Surface{}, connect.NewError(connect.CodeFailedPrecondition,
		fmt.Errorf("render: no seeded surface permits any placement style %q declares (%v)", style.ID, style.Placements))
}

// betterDefault ranks a candidate default surface against the incumbent: a
// product surface always beats a store one, and within a kind the larger area
// wins, because a bigger master can be derived down but never up.
func betterDefault(isProduct bool, area int, bestIsProduct bool, bestArea int) bool {
	if isProduct != bestIsProduct {
		return isProduct
	}
	return area > bestArea
}

func permitsAny(surfacePlacements, stylePlacements []string) bool {
	for _, want := range stylePlacements {
		for _, have := range surfacePlacements {
			if want == have {
				return true
			}
		}
	}
	return false
}

// brandTokens resolves the palette for one render. Explicit tokens win, because
// a caller who states an ink means it; otherwise a named brand is fetched from
// brand-manager, the single palette authority. Neither is required — a style
// renders from its own ink defaults when no brand is bound.
func (h *handler) brandTokens(ctx context.Context, brandID string, explicit map[string]string) (map[string]string, error) {
	if len(explicit) > 0 {
		return explicit, nil
	}
	if strings.TrimSpace(brandID) == "" {
		return nil, nil
	}
	baseURL, resolveErr := brandpalette.ResolveBaseURL(ctx)
	if resolveErr != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("render: brand %q was requested but %w", brandID, resolveErr))
	}
	tokens, fetchErr := brandpalette.Fetch(ctx, http.DefaultClient, baseURL, brandID)
	if fetchErr != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("render: resolve brand %q: %w", brandID, fetchErr))
	}
	return tokens, nil
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
	out := &render_v1.RenderJob{Id: job.ID, StyleId: job.StyleID, SurfaceId: job.SurfaceID, Status: job.Status, Seed: job.Seed, ExecutionPath: job.ExecutionPath, SelectedCandidateId: job.SelectedCandidateID, SelectedBy: job.SelectedBy}
	for _, c := range job.Candidates {
		out.Candidates = append(out.Candidates, candidateProto(c))
	}
	return out
}

func candidateProto(c internalrender.Candidate) *render_v1.Candidate {
	return &render_v1.Candidate{Id: c.ID, JobId: c.JobID, ImagePng: c.PNG, Width: int32(c.Width), Height: int32(c.Height), Strategy: c.Strategy, ExecutionPath: c.ExecutionPath, TreatmentApplied: c.TreatmentApplied, Seed: c.Seed, ConditioningSubmitted: c.ConditioningSubmitted, DisclosureRequired: c.DisclosureRequired, Prompt: c.Prompt, ProvenanceJson: c.ProvenanceJSON, QualityJson: c.QualityJSON}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "render_submit", Path: renderconnect.RenderServiceSubmitProcedure, Method: http.MethodPost, Summary: "Submit a reproducible render", Category: "render"},
	{ID: "render_job_get", Path: renderconnect.RenderServiceGetJobProcedure, Method: http.MethodPost, Summary: "Get render job status", Category: "render"},
	{ID: "render_candidates_list", Path: renderconnect.RenderServiceListCandidatesProcedure, Method: http.MethodPost, Summary: "List render candidates", Category: "render"},
	{ID: "render_candidate_select", Path: renderconnect.RenderServiceSelectCandidateProcedure, Method: http.MethodPost, Summary: "Select a render candidate", Category: "render"},
}
