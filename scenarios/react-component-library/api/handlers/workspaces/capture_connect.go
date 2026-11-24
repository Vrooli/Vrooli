package workspaces

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	"react-component-library/internal/componenttests"
	domain "react-component-library/internal/workspaces"
)

func (h *Handler) CaptureWorkspaceState(ctx context.Context, req *connect.Request[workspacev1.CaptureWorkspaceStateRequest]) (*connect.Response[workspacev1.CaptureWorkspaceStateResponse], error) {
	m := req.Msg
	input := captureStateRequest{ExpectedRevision: m.GetExpectedRevision(), PageID: m.GetPageId(), DisplayID: m.GetDisplayId(), StateID: m.GetStateId(), AttemptID: m.GetAttemptId()}
	workspaceID := m.GetWorkspaceId()
	if !uuidLikeWorkspaceID(workspaceID) || input.ExpectedRevision < 1 || input.PageID == "" || input.DisplayID == "" || input.StateID == "" || !validCaptureAttemptID(input.AttemptID) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace, expected revision, page, display, state, and UUID attempt id are required"))
	}
	if h.capture == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("BAS workspace capture is unavailable"))
	}
	workspace, err := h.repository.Get(ctx, workspaceID)
	if err != nil {
		return nil, h.mapError("load workspace for capture", err)
	}
	page, display, state := locateWorkspaceState(workspace, input)
	if page == nil || display == nil || state == nil {
		return nil, connect.NewError(connect.CodeAborted, errors.New("selected workspace state no longer exists; reload before capture"))
	}
	// Preserve idempotent replay of a completed receipt before enforcing the
	// revision guard. Capturing advances the aggregate revision, but a client
	// retry with the original attempt identity must still receive that receipt.
	token := workspaceCaptureToken(workspaceID, input)
	stored, evidenceErr := h.repository.GetCaptureEvidence(ctx, workspaceID, input.StateID)
	if evidenceErr == nil {
		if evidence, ok := decodeWorkspaceCaptureEvidence(stored.EvidenceJSON); ok && stored.RequestToken == token && state.Status == workspacev1.StateStatus_STATE_STATUS_CAPTURED && state.CaptureId == stored.CaptureID {
			return connect.NewResponse(captureWorkspaceResponse(workspace.Revision, state.CaptureId, evidence)), nil
		}
	} else if !errors.Is(evidenceErr, sql.ErrNoRows) {
		return nil, h.mapError("load workspace capture evidence", evidenceErr)
	}
	if workspace.Revision != input.ExpectedRevision {
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("workspace revision changed from %d to %d before capture", input.ExpectedRevision, workspace.Revision))
	}
	if workspace.GetProvenance() == workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("curated workspaces are immutable; fork one before capturing"))
	}
	if display.Width < 100 || display.Width > 4000 || display.Height < 100 || display.Height > 4000 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("selected display dimensions are outside BAS capture limits"))
	}
	// A retry can arrive from a new tab after sessionStorage lost its idempotency
	// key. Join the server's active attempt so BAS reconciles the same durable
	// request token instead of starting a second workflow for the same state.
	active, activeErr := h.repository.GetActiveCaptureAttempt(ctx, workspaceID, input.StateID, input.ExpectedRevision)
	if activeErr == nil {
		input.AttemptID = active.AttemptID
		if !validCaptureAttemptID(input.AttemptID) || active.RequestToken != workspaceCaptureToken(workspaceID, input) {
			return nil, connect.NewError(connect.CodeInternal, errors.New("active workspace capture attempt identity is inconsistent"))
		}
	} else if !errors.Is(activeErr, sql.ErrNoRows) {
		return nil, h.mapError("load active workspace capture attempt", activeErr)
	}
	token = workspaceCaptureToken(workspaceID, input)
	stored, evidenceErr = h.repository.GetCaptureEvidence(ctx, workspaceID, input.StateID)
	if evidenceErr == nil {
		if evidence, ok := decodeWorkspaceCaptureEvidence(stored.EvidenceJSON); ok && stored.RequestToken == token && state.Status == workspacev1.StateStatus_STATE_STATUS_CAPTURED && state.CaptureId == stored.CaptureID {
			return connect.NewResponse(captureWorkspaceResponse(workspace.Revision, state.CaptureId, evidence)), nil
		}
	} else if !errors.Is(evidenceErr, sql.ErrNoRows) {
		return nil, h.mapError("load workspace capture evidence", evidenceErr)
	}
	attempt, _, err := h.repository.BeginCaptureAttempt(ctx, workspaceID, input.StateID, input.ExpectedRevision, input.AttemptID, token)
	if errors.Is(err, domain.ErrCaptureInProgress) {
		return nil, captureAttemptConnectError(connect.CodeAlreadyExists, "another attempt for this state is still running", "in_progress")
	}
	if err != nil {
		return nil, h.mapError("reserve workspace state capture attempt", err)
	}
	if attempt.State == "failed" {
		return nil, captureAttemptConnectError(connect.CodeUnavailable, attempt.Error, "failed")
	}
	if attempt.State == "completed" {
		return nil, captureAttemptConnectError(connect.CodeFailedPrecondition, "capture completed but its evidence receipt is unavailable; use a new attempt", "completed")
	}
	result, err := h.capture.CaptureSurface(ctx, buildWorkspaceCaptureRequest(workspaceID, input, display.GetWidth(), display.GetHeight(), token))
	if err != nil {
		var terminal componenttests.CaptureTerminalError
		if errors.As(err, &terminal) {
			_ = h.repository.FinishCaptureAttempt(ctx, workspaceID, input.StateID, input.ExpectedRevision, input.AttemptID, "failed", terminal.Error())
			return nil, captureAttemptConnectError(connect.CodeUnavailable, terminal.Error(), "failed")
		}
		return nil, captureAttemptConnectError(connect.CodeDeadlineExceeded, err.Error(), "in_progress")
	}
	screenshotRef, accessibilityRef := captureArtifactReferences(result)
	if screenshotRef == "" || accessibilityRef == "" || result.GetExecutionId() == "" {
		message := "BAS completed without the required screenshot and accessibility evidence references"
		_ = h.repository.FinishCaptureAttempt(ctx, workspaceID, input.StateID, input.ExpectedRevision, input.AttemptID, "failed", message)
		return nil, captureAttemptConnectError(connect.CodeUnavailable, message, "failed")
	}
	evidence := workspaceCaptureEvidence{SchemaVersion: workspaceCaptureEvidenceSchema, RequestToken: token, AttemptID: input.AttemptID, SourceRevision: workspace.Revision, PageID: input.PageID, DisplayID: input.DisplayID, StateID: input.StateID, ExecutionID: result.GetExecutionId(), CapturedAt: time.Now().UTC().Format(time.RFC3339Nano), Screenshot: workspaceCaptureArtifact{Reference: screenshotRef, Width: display.Width, Height: display.Height, Scope: "selected workspace composition"}, Accessibility: workspaceCaptureArtifact{Reference: accessibilityRef, Scope: "workspace studio page; BAS accessibility artifact is not scoped to the composition screenshot"}}
	encoded, err := json.Marshal(evidence)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not encode workspace capture evidence"))
	}
	saved, err := h.repository.AttachCaptureEvidence(ctx, workspaceID, input.StateID, input.ExpectedRevision, input.AttemptID, token, result.GetExecutionId(), string(encoded), evidence.CapturedAt)
	if errors.Is(err, domain.ErrConflict) {
		return nil, connect.NewError(connect.CodeAborted, errors.New("workspace changed while BAS was capturing; evidence was not attached"))
	}
	if err != nil {
		return nil, h.mapError("save workspace capture evidence", err)
	}
	return connect.NewResponse(captureWorkspaceResponse(saved.Revision, result.GetExecutionId(), evidence)), nil
}

func (h *Handler) GetWorkspaceStateCapture(ctx context.Context, req *connect.Request[workspacev1.GetWorkspaceStateCaptureRequest]) (*connect.Response[workspacev1.CaptureWorkspaceStateResponse], error) {
	m := req.Msg
	if !uuidLikeWorkspaceID(m.GetWorkspaceId()) || !uuidLikeWorkspaceID(m.GetStateId()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace and state ids are invalid"))
	}
	evidence, err := h.repository.GetCaptureEvidence(ctx, m.GetWorkspaceId(), m.GetStateId())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("workspace state has no current capture evidence"))
	}
	if err != nil {
		return nil, h.mapError("load workspace state capture evidence", err)
	}
	workspace, err := h.repository.Get(ctx, m.GetWorkspaceId())
	if err != nil {
		return nil, h.mapError("load workspace state for capture evidence", err)
	}
	var currentState *workspacev1.WorkspaceState
	for _, page := range workspace.GetPages() {
		for _, display := range page.GetDisplays() {
			for _, state := range display.GetStates() {
				if state.GetId() == m.GetStateId() {
					currentState = state
					break
				}
			}
		}
	}
	if currentState == nil || currentState.GetStatus() != workspacev1.StateStatus_STATE_STATUS_CAPTURED || currentState.GetCaptureId() != evidence.CaptureID {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("workspace state has no current capture evidence"))
	}
	decoded, ok := decodeWorkspaceCaptureEvidence(evidence.EvidenceJSON)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("stored workspace capture evidence is invalid"))
	}
	return connect.NewResponse(captureWorkspaceResponse(workspace.GetRevision(), evidence.CaptureID, decoded)), nil
}

func (h *Handler) ListWorkspaceCaptureReceipts(ctx context.Context, req *connect.Request[workspacev1.ListWorkspaceCaptureReceiptsRequest]) (*connect.Response[workspacev1.ListWorkspaceCaptureReceiptsResponse], error) {
	m := req.Msg
	if !uuidLikeWorkspaceID(m.GetWorkspaceId()) || !uuidLikeWorkspaceID(m.GetStateId()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace and state ids are invalid"))
	}
	w, err := h.repository.Get(ctx, m.GetWorkspaceId())
	if err != nil {
		return nil, h.mapError("load workspace capture receipts", err)
	}
	if !workspaceHasState(w, m.GetStateId()) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("workspace state was not found"))
	}
	items, err := h.repository.ListCaptureReceipts(ctx, m.GetWorkspaceId(), m.GetStateId())
	if err != nil {
		return nil, h.mapError("list workspace capture receipts", err)
	}
	response := &workspacev1.ListWorkspaceCaptureReceiptsResponse{Receipts: make([]*workspacev1.WorkspaceCaptureReceipt, 0, len(items))}
	for _, item := range items {
		evidence, ok := decodeWorkspaceCaptureEvidence(item.EvidenceJSON)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, errors.New("stored workspace capture receipt is invalid"))
		}
		response.Receipts = append(response.Receipts, &workspacev1.WorkspaceCaptureReceipt{Id: item.ID, WorkspaceId: item.WorkspaceID, StateId: item.StateID, SourceRevision: item.SourceRevision, CaptureId: item.CaptureID, AttemptId: item.AttemptID, CapturedAt: item.CapturedAt, Evidence: captureEvidenceMessage(evidence)})
	}
	return connect.NewResponse(response), nil
}

func captureWorkspaceResponse(revision int32, captureID string, e workspaceCaptureEvidence) *workspacev1.CaptureWorkspaceStateResponse {
	return &workspacev1.CaptureWorkspaceStateResponse{WorkspaceRevision: revision, CaptureId: captureID, Evidence: captureEvidenceMessage(e)}
}

func captureEvidenceMessage(e workspaceCaptureEvidence) *workspacev1.WorkspaceCaptureEvidence {
	return &workspacev1.WorkspaceCaptureEvidence{SchemaVersion: e.SchemaVersion, RequestToken: e.RequestToken, AttemptId: e.AttemptID, SourceRevision: e.SourceRevision, PageId: e.PageID, DisplayId: e.DisplayID, StateId: e.StateID, ExecutionId: e.ExecutionID, CapturedAt: e.CapturedAt, Screenshot: &workspacev1.WorkspaceCaptureArtifact{Reference: e.Screenshot.Reference, Width: e.Screenshot.Width, Height: e.Screenshot.Height, Scope: e.Screenshot.Scope}, Accessibility: &workspacev1.WorkspaceCaptureArtifact{Reference: e.Accessibility.Reference, Scope: e.Accessibility.Scope}}
}

func captureAttemptConnectError(code connect.Code, message, state string) error {
	e := connect.NewError(code, errors.New(message))
	e.Meta().Set("attempt-state", state)
	return e
}
