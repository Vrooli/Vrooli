package versions

import (
	"context"
	"fmt"
	"log"
	"time"

	"connectrpc.com/connect"

	"react-component-library/internal/versionledger"
	"react-component-library/internal/versions"

	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
)

// Deps wires the seams the Connect versions handler needs.
type Deps struct {
	Service versions.Service
	Logger  *log.Logger
	Ledger  *versionledger.Repository
}

func (h *connectHandler) ListRetireCandidates(ctx context.Context, req *connect.Request[versionsv1.ListRetireCandidatesRequest]) (*connect.Response[versionsv1.ListRetireCandidatesResponse], error) {
	if h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version lifecycle is not configured"))
	}
	items, err := h.deps.Ledger.RetireCandidates(ctx, req.Msg.GetComponentId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &versionsv1.ListRetireCandidatesResponse{}
	for _, item := range items {
		out.Candidates = append(out.Candidates, &versionsv1.RetireCandidate{ComponentId: item.ComponentID, LibraryId: item.LibraryID, Version: item.Version, Status: item.Status})
	}
	return connect.NewResponse(out), nil
}

func cleanupScopeFromProto(scope *versionsv1.CleanupScope) versionledger.CleanupScope {
	if scope == nil {
		return versionledger.CleanupScope{}
	}
	return versionledger.CleanupScope{ComponentID: scope.GetComponentId(), LibraryID: scope.GetLibraryId(), OlderThanDays: int(scope.GetOlderThanDays())}
}

func cleanupItemToProto(item versionledger.CleanupItem) *versionsv1.CleanupItem {
	return &versionsv1.CleanupItem{
		Version:  &versionsv1.RetireCandidate{ComponentId: item.Candidate.ComponentID, LibraryId: item.Candidate.LibraryID, Version: item.Candidate.Version, Status: item.Candidate.Status},
		Eligible: item.Eligible, Reason: item.Reason, AdoptionCount: int32(item.AdoptionCount), DependencyCount: int32(item.DependencyCount), AgeDays: int32(item.AgeDays),
	}
}

func (h *connectHandler) PlanCleanup(ctx context.Context, req *connect.Request[versionsv1.PlanCleanupRequest]) (*connect.Response[versionsv1.PlanCleanupResponse], error) {
	if h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version lifecycle is not configured"))
	}
	items, hash, err := h.deps.Ledger.PlanCleanup(ctx, cleanupScopeFromProto(req.Msg.GetScope()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &versionsv1.PlanCleanupResponse{PlanHash: hash}
	for _, item := range items {
		out.Items = append(out.Items, cleanupItemToProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CleanupVersions(ctx context.Context, req *connect.Request[versionsv1.CleanupVersionsRequest]) (*connect.Response[versionsv1.CleanupVersionsResponse], error) {
	if h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version lifecycle is not configured"))
	}
	items, hash, retired, err := h.deps.Ledger.CleanupVersions(ctx, cleanupScopeFromProto(req.Msg.GetScope()), req.Msg.GetPlanHash(), req.Msg.GetConfirm())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	out := &versionsv1.CleanupVersionsResponse{PlanHash: hash, RetiredCount: int32(retired), Applied: req.Msg.GetConfirm()}
	for _, item := range items {
		out.Items = append(out.Items, cleanupItemToProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CleanupDraft(ctx context.Context, req *connect.Request[versionsv1.CleanupDraftRequest]) (*connect.Response[versionsv1.CleanupDraftResponse], error) {
	if h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version lifecycle is not configured"))
	}
	item, err := h.deps.Ledger.CleanupDraft(ctx, req.Msg.GetComponentId(), int(req.Msg.GetOlderThanDays()), req.Msg.GetConfirm())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&versionsv1.CleanupDraftResponse{Item: cleanupItemToProto(item), Applied: req.Msg.GetConfirm() && item.Eligible}), nil
}

func (h *connectHandler) ListVersionLedger(ctx context.Context, req *connect.Request[versionsv1.ListVersionLedgerRequest]) (*connect.Response[versionsv1.ListVersionLedgerResponse], error) {
	if h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version ledger is not configured"))
	}
	items, err := h.deps.Ledger.ListWindow(ctx, req.Msg.GetLibraryId(), req.Msg.GetWindow())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &versionsv1.ListVersionLedgerResponse{}
	for _, item := range items {
		out.Rows = append(out.Rows, &versionsv1.VersionLedgerRow{LibraryId: item.LibraryID, Version: item.Version, CreatedAt: item.CreatedAt.Format(time.RFC3339Nano), ReleasedAt: item.ReleasedAt.Format(time.RFC3339Nano), RetiredAt: item.RetiredAt.Format(time.RFC3339Nano), LifecycleState: item.LifecycleState, GatePassCount: int32(item.GatePassCount), GateFailCount: int32(item.GateFailCount), TestRuns: int32(item.TestRuns), TestPassRate: item.TestPassRate, AdoptionCurrent: int32(item.AdoptionCurrent), AdoptionPeak: int32(item.AdoptionPeak), FileCount: int32(item.FileCount), LinesOfCode: int32(item.LinesOfCode), DependencyCount: int32(item.DependencyCount)})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) transition(ctx context.Context, req *connect.Request[versionsv1.VersionLifecycleRequest], state string) (*connect.Response[versionsv1.VersionLifecycleResponse], error) {
	if h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version lifecycle is not configured"))
	}
	item, err := h.deps.Ledger.Transition(ctx, req.Msg.GetComponentId(), req.Msg.GetVersion(), state, req.Msg.GetConfirm())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&versionsv1.VersionLifecycleResponse{Version: &versionsv1.RetireCandidate{ComponentId: item.ComponentID, LibraryId: item.LibraryID, Version: item.Version, Status: item.Status}, LifecycleState: state}), nil
}

func (h *connectHandler) DeprecateVersion(ctx context.Context, req *connect.Request[versionsv1.VersionLifecycleRequest]) (*connect.Response[versionsv1.VersionLifecycleResponse], error) {
	return h.transition(ctx, req, "deprecated")
}

func (h *connectHandler) ArchiveVersion(ctx context.Context, req *connect.Request[versionsv1.VersionLifecycleRequest]) (*connect.Response[versionsv1.VersionLifecycleResponse], error) {
	return h.transition(ctx, req, "archived")
}

func (h *connectHandler) RetireVersion(ctx context.Context, req *connect.Request[versionsv1.VersionLifecycleRequest]) (*connect.Response[versionsv1.VersionLifecycleResponse], error) {
	return h.transition(ctx, req, "retired")
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListVersions(ctx context.Context, req *connect.Request[versionsv1.ListVersionsRequest]) (*connect.Response[versionsv1.ListVersionsResponse], error) {
	out, err := h.deps.Service.List(ctx, versions.ListQuery{
		ComponentID: req.Msg.ComponentId,
		Limit:       int(req.Msg.Limit),
	})
	if err != nil {
		h.deps.Logger.Printf("versions.ListVersions: %v", err)
		return nil, mapErr(err)
	}
	resp := &versionsv1.ListVersionsResponse{Versions: make([]*versionsv1.Version, 0, len(out))}
	for _, v := range out {
		resp.Versions = append(resp.Versions, versionToProto(v, false))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetVersion(ctx context.Context, req *connect.Request[versionsv1.GetVersionRequest]) (*connect.Response[versionsv1.GetVersionResponse], error) {
	v, err := h.deps.Service.Get(ctx, req.Msg.ComponentId, req.Msg.Version)
	if err != nil {
		connectErr := mapErr(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("versions.GetVersion(%q,%q): %v", req.Msg.ComponentId, req.Msg.Version, err)
		}
		return nil, connectErr
	}
	resp := &versionsv1.GetVersionResponse{Version: versionToProto(v, false)}
	if req.Msg.IncludeContent {
		resp.Content = v.Content
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) DiffVersions(ctx context.Context, req *connect.Request[versionsv1.DiffVersionsRequest]) (*connect.Response[versionsv1.DiffVersionsResponse], error) {
	result, err := h.deps.Service.Diff(ctx, versions.DiffInput{
		ComponentID: req.Msg.ComponentId,
		From:        req.Msg.From,
		To:          req.Msg.To,
	})
	if err != nil {
		connectErr := mapErr(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("versions.DiffVersions: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(diffToProto(result)), nil
}

func mapErr(err error) *connect.Error {
	if mapped := versions.ToConnectError(err); mapped != nil {
		return mapped
	}
	return connect.NewError(connect.CodeInternal, err)
}
