package sketch

import (
	"context"

	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"

	"connectrpc.com/connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
)

// WorkspaceAssistPort is the narrow handler surface registered on
// WorkspaceAssistService.
type WorkspaceAssistPort interface {
	ProposeSketch(context.Context, *connect.Request[sharedv1.ProposeSketchRequest]) (*connect.Response[sharedv1.ProposeSketchResponse], error)
	InferSketch(context.Context, *connect.Request[sketchv1.InferSketchRequest]) (*connect.Response[sketchv1.SketchInferenceOperation], error)
	GetSketchInference(context.Context, *connect.Request[sketchv1.GetSketchInferenceRequest]) (*connect.Response[sketchv1.SketchInferenceOperation], error)
	RefineCandidate(context.Context, *connect.Request[sketchv1.RefineCandidateRequest]) (*connect.Response[sketchv1.RefineCandidateResponse], error)
}

type workspaceAssistHandler struct{ port WorkspaceAssistPort }

func (h workspaceAssistHandler) ProposeSketch(ctx context.Context, req *connect.Request[sharedv1.ProposeSketchRequest]) (*connect.Response[sharedv1.ProposeSketchResponse], error) {
	return h.port.ProposeSketch(ctx, req)
}

func (h workspaceAssistHandler) InferSketch(ctx context.Context, req *connect.Request[sketchv1.InferSketchRequest]) (*connect.Response[sketchv1.SketchInferenceOperation], error) {
	return h.port.InferSketch(ctx, req)
}

func (h workspaceAssistHandler) GetSketchInference(ctx context.Context, req *connect.Request[sketchv1.GetSketchInferenceRequest]) (*connect.Response[sketchv1.SketchInferenceOperation], error) {
	return h.port.GetSketchInference(ctx, req)
}

func (h workspaceAssistHandler) RefineCandidate(ctx context.Context, req *connect.Request[sketchv1.RefineCandidateRequest]) (*connect.Response[sketchv1.RefineCandidateResponse], error) {
	return h.port.RefineCandidate(ctx, req)
}

// WorkspaceCandidatePort is the narrow handler surface registered on
// WorkspaceCandidateService.
type WorkspaceCandidatePort interface {
	ListCandidates(context.Context, *connect.Request[sharedv1.SketchTarget]) (*connect.Response[sketchv1.ListCandidatesResponse], error)
	PlaceCandidateAsset(context.Context, *connect.Request[sketchv1.PlaceCandidateAssetRequest]) (*connect.Response[sharedv1.CandidateResponse], error)
	MapCandidateRegions(context.Context, *connect.Request[sketchv1.MapCandidateRegionsRequest]) (*connect.Response[sharedv1.CandidateResponse], error)
	SaveCandidate(context.Context, *connect.Request[sketchv1.SaveCandidateRequest]) (*connect.Response[sharedv1.CandidateResponse], error)
	GetCandidate(context.Context, *connect.Request[sharedv1.CandidateReference]) (*connect.Response[sharedv1.CandidateResponse], error)
	RenderCandidate(context.Context, *connect.Request[sharedv1.RenderCandidateRequest]) (*connect.Response[sharedv1.RenderCompositionResponse], error)
	RenderSketch(context.Context, *connect.Request[sketchv1.RenderSketchRequest]) (*connect.Response[sharedv1.RenderCompositionResponse], error)
}

type workspaceCandidateHandler struct{ port WorkspaceCandidatePort }

func (h workspaceCandidateHandler) ListCandidates(ctx context.Context, req *connect.Request[sharedv1.SketchTarget]) (*connect.Response[sketchv1.ListCandidatesResponse], error) {
	return h.port.ListCandidates(ctx, req)
}

func (h workspaceCandidateHandler) PlaceCandidateAsset(ctx context.Context, req *connect.Request[sketchv1.PlaceCandidateAssetRequest]) (*connect.Response[sharedv1.CandidateResponse], error) {
	return h.port.PlaceCandidateAsset(ctx, req)
}

func (h workspaceCandidateHandler) MapCandidateRegions(ctx context.Context, req *connect.Request[sketchv1.MapCandidateRegionsRequest]) (*connect.Response[sharedv1.CandidateResponse], error) {
	return h.port.MapCandidateRegions(ctx, req)
}

func (h workspaceCandidateHandler) SaveCandidate(ctx context.Context, req *connect.Request[sketchv1.SaveCandidateRequest]) (*connect.Response[sharedv1.CandidateResponse], error) {
	return h.port.SaveCandidate(ctx, req)
}

func (h workspaceCandidateHandler) GetCandidate(ctx context.Context, req *connect.Request[sharedv1.CandidateReference]) (*connect.Response[sharedv1.CandidateResponse], error) {
	return h.port.GetCandidate(ctx, req)
}

func (h workspaceCandidateHandler) RenderCandidate(ctx context.Context, req *connect.Request[sharedv1.RenderCandidateRequest]) (*connect.Response[sharedv1.RenderCompositionResponse], error) {
	return h.port.RenderCandidate(ctx, req)
}

func (h workspaceCandidateHandler) RenderSketch(ctx context.Context, req *connect.Request[sketchv1.RenderSketchRequest]) (*connect.Response[sharedv1.RenderCompositionResponse], error) {
	return h.port.RenderSketch(ctx, req)
}
