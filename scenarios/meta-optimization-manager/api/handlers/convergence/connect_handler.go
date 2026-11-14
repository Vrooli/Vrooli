package convergence

import (
	"context"
	"log"

	internalconv "meta-optimization-manager/internal/convergence"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	convergencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/convergence"
)

// Deps wires the seams the Connect convergence handler needs.
type Deps struct {
	Service internalconv.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the ConvergenceService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetConvergenceStatus(ctx context.Context, _ *connect.Request[convergencev1.GetConvergenceStatusRequest]) (*connect.Response[convergencev1.GetConvergenceStatusResponse], error) {
	status, err := h.deps.Service.GetConvergenceStatus(ctx)
	if err != nil {
		h.deps.Logger.Printf("convergence.GetConvergenceStatus: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &convergencev1.GetConvergenceStatusResponse{
		Templates:  make([]*convergencev1.TemplateFitness, 0, len(status.Templates)),
		References: make([]*convergencev1.ReferenceHealth, 0, len(status.References)),
	}
	for _, tf := range status.Templates {
		resp.Templates = append(resp.Templates, fitnessToProto(tf))
	}
	for _, h := range status.References {
		resp.References = append(resp.References, referenceToProto(h))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetTemplateFitness(ctx context.Context, req *connect.Request[convergencev1.GetTemplateFitnessRequest]) (*connect.Response[convergencev1.GetTemplateFitnessResponse], error) {
	fitness, err := h.deps.Service.GetTemplateFitness(ctx, req.Msg.GetTemplate())
	if err != nil {
		h.deps.Logger.Printf("convergence.GetTemplateFitness: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &convergencev1.GetTemplateFitnessResponse{Templates: make([]*convergencev1.TemplateFitness, 0, len(fitness))}
	for _, tf := range fitness {
		resp.Templates = append(resp.Templates, fitnessToProto(tf))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListReferences(ctx context.Context, req *connect.Request[convergencev1.ListReferencesRequest]) (*connect.Response[convergencev1.ListReferencesResponse], error) {
	refs, err := h.deps.Service.ListReferences(ctx, eligibilityFromProto(req.Msg.GetEligibility()))
	if err != nil {
		h.deps.Logger.Printf("convergence.ListReferences: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &convergencev1.ListReferencesResponse{References: make([]*convergencev1.ReferenceHealth, 0, len(refs))}
	for _, rh := range refs {
		resp.References = append(resp.References, referenceToProto(rh))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetConvergenceTrend(ctx context.Context, req *connect.Request[convergencev1.GetConvergenceTrendRequest]) (*connect.Response[convergencev1.GetConvergenceTrendResponse], error) {
	points, err := h.deps.Service.GetConvergenceTrend(ctx, req.Msg.GetTemplate())
	if err != nil {
		h.deps.Logger.Printf("convergence.GetConvergenceTrend: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &convergencev1.GetConvergenceTrendResponse{Points: make([]*convergencev1.FitnessTrendPoint, 0, len(points))}
	for _, p := range points {
		resp.Points = append(resp.Points, &convergencev1.FitnessTrendPoint{
			Template:             p.Template,
			At:                   timestamppb.New(p.At),
			PerReplicaCost:       int32(p.PerReplicaCost),
			CoordinatedEditCount: int32(p.CoordinatedEditCount),
		})
	}
	return connect.NewResponse(resp), nil
}

func fitnessToProto(tf internalconv.TemplateFitness) *convergencev1.TemplateFitness {
	return &convergencev1.TemplateFitness{
		Template:                 tf.Template,
		PerReplicaCost:           int32(tf.PerReplicaCost),
		DriftSurfaceCount:        int32(tf.DriftSurfaceCount),
		CommentOnlyContractCount: int32(tf.CommentOnlyContractCount),
		CoordinatedEditCount:     int32(tf.CoordinatedEditCount),
		Tier:                     tierToProto(tf.Tier),
	}
}

func referenceToProto(h internalconv.ReferenceHealth) *convergencev1.ReferenceHealth {
	out := &convergencev1.ReferenceHealth{
		Scenario:          h.Scenario,
		StaleFromTemplate: h.StaleFromTemplate,
		CleanOnAllTools:   h.CleanOnAllTools,
		StabilityDays:     int32(h.StabilityDays),
		Breadth:           int32(h.Breadth),
		Eligibility:       eligibilityToProto(h.Eligibility),
	}
	if !h.LastTemplateSync.IsZero() {
		out.LastTemplateSync = timestamppb.New(h.LastTemplateSync)
	}
	return out
}
