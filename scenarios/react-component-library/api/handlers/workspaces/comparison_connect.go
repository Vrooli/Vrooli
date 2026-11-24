package workspaces

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"connectrpc.com/connect"
	diffv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff"
	visualv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacevisualcomparison"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacevisualcomparison/workspacevisualcomparison_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"react-component-library/internal/module"
	domain "react-component-library/internal/workspaces"
)

func (h *Handler) CompareWorkspaceCaptureReceipts(ctx context.Context, req *connect.Request[visualv1.CompareWorkspaceCaptureReceiptsRequest]) (*connect.Response[visualv1.WorkspaceVisualComparison], error) {
	m := req.Msg
	if !uuidLikeWorkspaceID(m.GetWorkspaceId()) || !uuidLikeWorkspaceID(m.GetStateId()) || m.GetBaselineReceiptId() == "" || m.GetCurrentReceiptId() == "" || m.GetBaselineReceiptId() == m.GetCurrentReceiptId() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace/state ids and distinct baseline/current receipt ids are required"))
	}
	if h.comparer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("workspace visual comparison is unavailable"))
	}
	baseline, err := h.repository.GetCaptureReceipt(ctx, m.GetWorkspaceId(), m.GetStateId(), m.GetBaselineReceiptId())
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, domain.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("capture receipt was not found in this workspace state"))
	}
	if err != nil {
		return nil, h.mapError("load baseline capture receipt", err)
	}
	current, err := h.repository.GetCaptureReceipt(ctx, m.GetWorkspaceId(), m.GetStateId(), m.GetCurrentReceiptId())
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, domain.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("capture receipt was not found in this workspace state"))
	}
	if err != nil {
		return nil, h.mapError("load current capture receipt", err)
	}
	result, err := h.comparer.Compare(ctx, baseline, current)
	if err != nil {
		h.logger.Printf("workspaces: compare capture receipts: %v", err)
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("could not compare the selected capture receipts"))
	}
	resultJSON, err := protojson.Marshal(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not encode visual comparison"))
	}
	record, err := h.repository.CreateVisualComparison(ctx, m.GetWorkspaceId(), m.GetStateId(), baseline.ID, current.ID, string(resultJSON), result.GetHeatmapRef(), time.Now().UTC().Format(time.RFC3339Nano))
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, domain.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("capture receipt was not found in this workspace state"))
	}
	if err != nil {
		return nil, h.mapError("save visual comparison", err)
	}
	return connect.NewResponse(visualComparisonMessage(record)), nil
}

func (h *Handler) ListWorkspaceVisualComparisons(ctx context.Context, req *connect.Request[visualv1.ListWorkspaceVisualComparisonsRequest]) (*connect.Response[visualv1.ListWorkspaceVisualComparisonsResponse], error) {
	m := req.Msg
	if !uuidLikeWorkspaceID(m.GetWorkspaceId()) || !uuidLikeWorkspaceID(m.GetStateId()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace and state ids are invalid"))
	}
	w, err := h.repository.Get(ctx, m.GetWorkspaceId())
	if err != nil {
		return nil, h.mapError("load workspace visual comparisons", err)
	}
	if !workspaceHasState(w, m.GetStateId()) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("workspace state was not found"))
	}
	items, err := h.repository.ListVisualComparisons(ctx, m.GetWorkspaceId(), m.GetStateId())
	if err != nil {
		return nil, h.mapError("list workspace visual comparisons", err)
	}
	response := &visualv1.ListWorkspaceVisualComparisonsResponse{Comparisons: make([]*visualv1.WorkspaceVisualComparison, 0, len(items))}
	for _, item := range items {
		var result diffv1.DiffResult
		if err := protojson.Unmarshal([]byte(item.ResultJSON), &result); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("stored workspace visual comparison is invalid"))
		}
		canonical, err := protojson.Marshal(&result)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("could not encode stored visual comparison"))
		}
		item.ResultJSON = string(canonical)
		response.Comparisons = append(response.Comparisons, visualComparisonMessage(item))
	}
	return connect.NewResponse(response), nil
}

func visualComparisonMessage(item domain.VisualComparison) *visualv1.WorkspaceVisualComparison {
	return &visualv1.WorkspaceVisualComparison{ComparisonId: item.ID, WorkspaceId: item.WorkspaceID, StateId: item.StateID, BaselineReceiptId: item.BaselineReceiptID, CurrentReceiptId: item.CurrentReceiptID, CreatedAt: item.CreatedAt, ResultJson: item.ResultJSON}
}

func (h *Handler) CompareWorkspaceSourceBaseline(ctx context.Context, req *connect.Request[visualv1.CompareWorkspaceSourceBaselineRequest]) (*connect.Response[visualv1.WorkspaceSourceVisualComparison], error) {
	m := req.Msg
	if !uuidLikeWorkspaceID(m.GetWorkspaceId()) || !uuidLikeWorkspaceID(m.GetStateId()) || m.GetCurrentReceiptId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace, state, and current receipt ids are required"))
	}
	if h.comparer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("workspace visual comparison is unavailable"))
	}
	baseline, err := h.repository.GetAcceptedSourceBaseline(ctx, m.GetWorkspaceId(), m.GetStateId())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("accept a source baseline before comparing drift"))
	}
	if err != nil {
		return nil, h.mapError("load accepted source baseline", err)
	}
	asset, sourceBytes, err := h.repository.GetWorkspaceAsset(ctx, m.GetWorkspaceId(), baseline.AssetID)
	if err != nil {
		return nil, h.mapError("load source baseline image", err)
	}
	if asset.Mime != "image/png" && asset.Mime != "image/jpeg" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("accepted source image format cannot be compared"))
	}
	digest := sha256.Sum256(sourceBytes)
	if hex.EncodeToString(digest[:]) != baseline.SHA256 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("source baseline image digest no longer matches its recorded source"))
	}
	current, err := h.repository.GetCaptureReceipt(ctx, m.GetWorkspaceId(), m.GetStateId(), m.GetCurrentReceiptId())
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, domain.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("current capture receipt was not found"))
	}
	if err != nil {
		return nil, h.mapError("load current capture receipt", err)
	}
	fetcher, ok := h.comparer.(interface {
		FetchScreenshot(context.Context, domain.CaptureReceipt) ([]byte, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("capture screenshot resolver is unavailable"))
	}
	currentBytes, err := fetcher.FetchScreenshot(ctx, current)
	if err != nil {
		h.logger.Printf("workspaces: resolve current source comparison capture: %v", err)
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("current capture image is unavailable"))
	}
	imageComparer, ok := h.comparer.(ImageBytesCompareRunner)
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("source image comparison is unavailable"))
	}
	result, err := imageComparer.CompareImages(ctx, sourceBytes, currentBytes)
	if err != nil {
		h.logger.Printf("workspaces: compare source baseline: %v", err)
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("source visual comparison failed"))
	}
	resultJSON, err := protojson.Marshal(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not encode source comparison result"))
	}
	item, err := h.repository.CreateSourceVisualComparison(ctx, domain.SourceVisualComparison{WorkspaceID: m.GetWorkspaceId(), StateID: m.GetStateId(), SourceBaselineID: baseline.ID, SourceAssetID: baseline.AssetID, CurrentReceiptID: current.ID, ResultJSON: string(resultJSON), HeatmapRef: result.GetHeatmapRef(), CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("accepted source baseline or current capture is unavailable"))
	}
	if err != nil {
		return nil, h.mapError("persist source visual comparison", err)
	}
	return connect.NewResponse(sourceVisualComparisonMessage(item)), nil
}

func (h *Handler) ListWorkspaceSourceBaselineComparisons(ctx context.Context, req *connect.Request[visualv1.ListWorkspaceSourceBaselineComparisonsRequest]) (*connect.Response[visualv1.ListWorkspaceSourceBaselineComparisonsResponse], error) {
	m := req.Msg
	if !uuidLikeWorkspaceID(m.GetWorkspaceId()) || !uuidLikeWorkspaceID(m.GetStateId()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace and state ids are invalid"))
	}
	w, err := h.repository.Get(ctx, m.GetWorkspaceId())
	if err != nil {
		return nil, h.mapError("load workspace source comparison history", err)
	}
	if !workspaceHasState(w, m.GetStateId()) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("workspace state was not found"))
	}
	items, err := h.repository.ListSourceVisualComparisons(ctx, m.GetWorkspaceId(), m.GetStateId())
	if err != nil {
		return nil, h.mapError("list source visual comparisons", err)
	}
	response := &visualv1.ListWorkspaceSourceBaselineComparisonsResponse{Comparisons: make([]*visualv1.WorkspaceSourceVisualComparison, 0, len(items))}
	for _, item := range items {
		response.Comparisons = append(response.Comparisons, sourceVisualComparisonMessage(item))
	}
	return connect.NewResponse(response), nil
}

func sourceVisualComparisonMessage(item domain.SourceVisualComparison) *visualv1.WorkspaceSourceVisualComparison {
	return &visualv1.WorkspaceSourceVisualComparison{ComparisonId: item.ID, WorkspaceId: item.WorkspaceID, StateId: item.StateID, SourceBaselineId: item.SourceBaselineID, SourceAssetId: item.SourceAssetID, CurrentReceiptId: item.CurrentReceiptID, CreatedAt: item.CreatedAt, ResultJson: item.ResultJSON}
}

var visualConnectEndpoints = []module.EndpointDescriptor{
	{ID: "workspaces_visual_comparisons_create", Path: visualconnect.WorkspaceVisualComparisonServiceCompareWorkspaceCaptureReceiptsProcedure, Method: "POST", Summary: "Compare two captured workspace states", Category: "workspaces", Request: &module.Schema{Type: "object", Properties: map[string]string{"workspace_id": "string", "state_id": "string", "baseline_receipt_id": "string", "current_receipt_id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"comparison_id": "string", "result_json": "DiffResult protojson"}}, Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Receipt ids are missing or identical"}, {Status: 404, Code: "not_found", Description: "The selected receipt was not found"}, {Status: 503, Code: "unavailable", Description: "Image comparison is unavailable"}}},
	{ID: "workspaces_visual_comparisons_list", Path: visualconnect.WorkspaceVisualComparisonServiceListWorkspaceVisualComparisonsProcedure, Method: "POST", Summary: "List visual comparisons for a workspace state", Category: "workspaces", Request: &module.Schema{Type: "object", Properties: map[string]string{"workspace_id": "string", "state_id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"comparisons": "array<WorkspaceVisualComparison>"}}},
	{ID: "workspaces_source_visual_comparison_create", Path: visualconnect.WorkspaceVisualComparisonServiceCompareWorkspaceSourceBaselineProcedure, Method: "POST", Summary: "Compare an accepted source image with a captured workspace state", Category: "workspaces", Request: &module.Schema{Type: "object", Properties: map[string]string{"workspace_id": "string", "state_id": "string", "current_receipt_id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"comparison_id": "string", "source_baseline_id": "string", "source_asset_id": "string", "result_json": "DiffResult protojson"}}},
	{ID: "workspaces_source_visual_comparisons_list", Path: visualconnect.WorkspaceVisualComparisonServiceListWorkspaceSourceBaselineComparisonsProcedure, Method: "POST", Summary: "List comparisons against accepted source baselines", Category: "workspaces", Request: &module.Schema{Type: "object", Properties: map[string]string{"workspace_id": "string", "state_id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"comparisons": "array<WorkspaceSourceVisualComparison>"}}},
}
