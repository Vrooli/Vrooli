package facts

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
	"strings"

	"connectrpc.com/connect"

	internalfacts "code-facts/internal/facts"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

type Deps struct {
	Service         *internalfacts.Service
	Index           IndexController
	IndexAuthorizer IndexAuthorizer
	Logger          *log.Logger
}

type IndexAuthorizer interface {
	AuthorizeIndexControl(context.Context, string, string) error
}

const IndexControlTokenHeader = "X-Code-Facts-Control-Token" // #nosec G101 -- protocol header name, not a credential.

type StaticIndexAuthorizer struct{ token string }

type CompositeIndexAuthorizer struct {
	static  *StaticIndexAuthorizer
	dynamic func(string) bool
}

func NewStaticIndexAuthorizer(token string) *StaticIndexAuthorizer {
	return &StaticIndexAuthorizer{token: strings.TrimSpace(token)}
}

func NewCompositeIndexAuthorizer(token string, dynamic func(string) bool) *CompositeIndexAuthorizer {
	return &CompositeIndexAuthorizer{static: NewStaticIndexAuthorizer(token), dynamic: dynamic}
}

func (authorizer *CompositeIndexAuthorizer) AuthorizeIndexControl(ctx context.Context, operation, presented string) error {
	if authorizer != nil && authorizer.dynamic != nil && authorizer.dynamic(strings.TrimSpace(presented)) {
		return nil
	}
	if authorizer == nil || authorizer.static == nil {
		return fmt.Errorf("index mutation authorization is not configured")
	}
	return authorizer.static.AuthorizeIndexControl(ctx, operation, presented)
}

func (authorizer *StaticIndexAuthorizer) AuthorizeIndexControl(_ context.Context, operation, presented string) error {
	expected := ""
	if authorizer != nil {
		expected = authorizer.token
	}
	presented = strings.TrimSpace(presented)
	if expected == "" {
		return fmt.Errorf("index mutation authorization is not configured")
	}
	if len(expected) != len(presented) || subtle.ConstantTimeCompare([]byte(expected), []byte(presented)) != 1 {
		return fmt.Errorf("index mutation %s is not authorized", operation)
	}
	return nil
}

type IndexController interface {
	Status(context.Context) (*factsv1.IndexStatus, error)
	Reconcile(context.Context, string) (*factsv1.IndexControlResponse, error)
	Reindex(context.Context, string) (*factsv1.IndexControlResponse, error)
	Cancel(context.Context, string) (*factsv1.IndexControlResponse, error)
	Promote(context.Context, string) (*factsv1.IndexControlResponse, error)
	Rollback(context.Context, string) (*factsv1.IndexControlResponse, error)
	Cleanup(context.Context, bool) (*factsv1.IndexControlResponse, error)
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Service == nil {
		d.Service = internalfacts.NewService()
	}
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) DescribeCodeFacts(ctx context.Context, req *connect.Request[factsv1.DescribeCodeFactsRequest]) (*connect.Response[factsv1.CodeFactsReport], error) {
	report, err := h.deps.Service.Describe(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) Search(ctx context.Context, req *connect.Request[factsv1.SearchRequest]) (*connect.Response[factsv1.SearchResponse], error) {
	response, err := h.deps.Service.Search(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) DescribeFleetImports(ctx context.Context, req *connect.Request[factsv1.DescribeFleetImportsRequest]) (*connect.Response[factsv1.DescribeFleetImportsResponse], error) {
	report, err := h.deps.Service.DescribeFleetImports(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) ListSurfaces(ctx context.Context, req *connect.Request[factsv1.ListSurfacesRequest]) (*connect.Response[factsv1.ListSurfacesResponse], error) {
	report, err := h.deps.Service.Surfaces(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) CheckProtoAdoption(ctx context.Context, req *connect.Request[factsv1.CheckProtoAdoptionRequest]) (*connect.Response[factsv1.ProofReport], error) {
	report, err := h.deps.Service.ProtoAdoption(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) CheckEndpointProof(ctx context.Context, req *connect.Request[factsv1.CheckEndpointProofRequest]) (*connect.Response[factsv1.ProofReport], error) {
	report, err := h.deps.Service.EndpointProof(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) GetCacheStatus(ctx context.Context, req *connect.Request[factsv1.GetCacheStatusRequest]) (*connect.Response[factsv1.CacheStatus], error) {
	report, err := h.deps.Service.CacheStatus(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) InspectCache(ctx context.Context, req *connect.Request[factsv1.InspectCacheRequest]) (*connect.Response[factsv1.CacheStatus], error) {
	report, err := h.deps.Service.InspectCache(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) ClearCache(ctx context.Context, req *connect.Request[factsv1.ClearCacheRequest]) (*connect.Response[factsv1.ClearCacheResponse], error) {
	report, err := h.deps.Service.ClearCache(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) GetIndexStatus(ctx context.Context, _ *connect.Request[factsv1.GetIndexStatusRequest]) (*connect.Response[factsv1.IndexStatus], error) {
	if h.deps.Index == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("index controls are not configured"))
	}
	status, err := h.deps.Index.Status(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(status), nil
}

func (h *connectHandler) ReconcileIndex(ctx context.Context, req *connect.Request[factsv1.ReconcileIndexRequest]) (*connect.Response[factsv1.IndexControlResponse], error) {
	if err := h.authorizeIndex(ctx, "reconcile", req.Header().Get(IndexControlTokenHeader)); err != nil {
		return nil, err
	}
	response, err := h.index().Reconcile(ctx, req.Msg.GetGeneration())
	return indexResponse(response, err)
}

func (h *connectHandler) Reindex(ctx context.Context, req *connect.Request[factsv1.ReindexRequest]) (*connect.Response[factsv1.IndexControlResponse], error) {
	if err := h.authorizeIndex(ctx, "reindex", req.Header().Get(IndexControlTokenHeader)); err != nil {
		return nil, err
	}
	if !req.Msg.GetConfirmed() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("reindex requires confirmed=true"))
	}
	response, err := h.index().Reindex(ctx, req.Msg.GetGeneration())
	return indexResponse(response, err)
}

func (h *connectHandler) CancelIndexJob(ctx context.Context, req *connect.Request[factsv1.CancelIndexJobRequest]) (*connect.Response[factsv1.IndexControlResponse], error) {
	if err := h.authorizeIndex(ctx, "cancel", req.Header().Get(IndexControlTokenHeader)); err != nil {
		return nil, err
	}
	if req.Msg.GetJobId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("job_id is required"))
	}
	response, err := h.index().Cancel(ctx, req.Msg.GetJobId())
	return indexResponse(response, err)
}

func (h *connectHandler) PromoteIndexGeneration(ctx context.Context, req *connect.Request[factsv1.PromoteIndexGenerationRequest]) (*connect.Response[factsv1.IndexControlResponse], error) {
	if err := h.authorizeIndex(ctx, "promote", req.Header().Get(IndexControlTokenHeader)); err != nil {
		return nil, err
	}
	if !req.Msg.GetConfirmed() || req.Msg.GetGeneration() == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("promotion requires generation and confirmed=true"))
	}
	response, err := h.index().Promote(ctx, req.Msg.GetGeneration())
	return indexResponse(response, err)
}

func (h *connectHandler) RollbackIndexGeneration(ctx context.Context, req *connect.Request[factsv1.RollbackIndexGenerationRequest]) (*connect.Response[factsv1.IndexControlResponse], error) {
	if err := h.authorizeIndex(ctx, "rollback", req.Header().Get(IndexControlTokenHeader)); err != nil {
		return nil, err
	}
	if !req.Msg.GetConfirmed() || req.Msg.GetGeneration() == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("rollback requires generation and confirmed=true"))
	}
	response, err := h.index().Rollback(ctx, req.Msg.GetGeneration())
	return indexResponse(response, err)
}

func (h *connectHandler) CleanupIndex(ctx context.Context, req *connect.Request[factsv1.CleanupIndexRequest]) (*connect.Response[factsv1.IndexControlResponse], error) {
	if err := h.authorizeIndex(ctx, "cleanup", req.Header().Get(IndexControlTokenHeader)); err != nil {
		return nil, err
	}
	if !req.Msg.GetDryRun() && !req.Msg.GetConfirmed() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("cleanup mutation requires confirmed=true"))
	}
	response, err := h.index().Cleanup(ctx, req.Msg.GetDryRun())
	return indexResponse(response, err)
}

func (h *connectHandler) authorizeIndex(ctx context.Context, operation, presented string) error {
	if h.deps.IndexAuthorizer == nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("index mutation authorization is not configured"))
	}
	if err := h.deps.IndexAuthorizer.AuthorizeIndexControl(ctx, operation, presented); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return nil
}

func (h *connectHandler) index() IndexController {
	if h.deps.Index == nil {
		return unavailableIndexController{}
	}
	return h.deps.Index
}

func indexResponse(response *factsv1.IndexControlResponse, err error) (*connect.Response[factsv1.IndexControlResponse], error) {
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(response), nil
}

type unavailableIndexController struct{}

func (unavailableIndexController) unavailable() (*factsv1.IndexControlResponse, error) {
	return nil, fmt.Errorf("index controls are not configured")
}

func (unavailableIndexController) Status(context.Context) (*factsv1.IndexStatus, error) {
	return nil, fmt.Errorf("index controls are not configured")
}

func (controller unavailableIndexController) Reconcile(context.Context, string) (*factsv1.IndexControlResponse, error) {
	return controller.unavailable()
}

func (controller unavailableIndexController) Reindex(context.Context, string) (*factsv1.IndexControlResponse, error) {
	return controller.unavailable()
}

func (controller unavailableIndexController) Cancel(context.Context, string) (*factsv1.IndexControlResponse, error) {
	return controller.unavailable()
}

func (controller unavailableIndexController) Promote(context.Context, string) (*factsv1.IndexControlResponse, error) {
	return controller.unavailable()
}

func (controller unavailableIndexController) Rollback(context.Context, string) (*factsv1.IndexControlResponse, error) {
	return controller.unavailable()
}

func (controller unavailableIndexController) Cleanup(context.Context, bool) (*factsv1.IndexControlResponse, error) {
	return controller.unavailable()
}
