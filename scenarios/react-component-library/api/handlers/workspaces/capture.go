package workspaces

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	"react-component-library/internal/httprest"
	domain "react-component-library/internal/workspaces"
)

const (
	workspaceCaptureEvidenceSchema     = "rcl-workspace-state-capture/v1"
	workspaceCaptureReadySelector      = `[data-testid="workspace-state-composition"][data-capture-ready="true"]`
	workspaceCaptureScreenshotSelector = `[data-testid="workspace-state-composition"]`
)

var captureAttemptIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type WorkspaceStateCaptureClient interface {
	CaptureSurface(context.Context, *capturev1.CaptureRequest) (*capturev1.CaptureResponse, error)
}

type captureStateRequest struct {
	ExpectedRevision int32  `json:"expectedRevision"`
	PageID           string `json:"pageId"`
	DisplayID        string `json:"displayId"`
	StateID          string `json:"stateId"`
	AttemptID        string `json:"attemptId"`
}

type workspaceCaptureArtifact struct {
	Reference string `json:"reference"`
	Width     int32  `json:"width,omitempty"`
	Height    int32  `json:"height,omitempty"`
	Scope     string `json:"scope"`
}

type workspaceCaptureEvidence struct {
	SchemaVersion  string                   `json:"schemaVersion"`
	RequestToken   string                   `json:"requestToken"`
	AttemptID      string                   `json:"attemptId"`
	SourceRevision int32                    `json:"sourceRevision"`
	PageID         string                   `json:"pageId"`
	DisplayID      string                   `json:"displayId"`
	StateID        string                   `json:"stateId"`
	ExecutionID    string                   `json:"executionId"`
	CapturedAt     string                   `json:"capturedAt"`
	Screenshot     workspaceCaptureArtifact `json:"screenshot"`
	Accessibility  workspaceCaptureArtifact `json:"accessibility"`
}

func mountStateCapture(router *mux.Router, handler *Handler) {
	router.HandleFunc("/api/workspaces/{id}/states/{stateId}/capture-receipts/{receiptId}/screenshot", handler.getCaptureReceiptScreenshot).Methods(http.MethodGet)
}

func buildWorkspaceCaptureRequest(workspaceID string, input captureStateRequest, width, height int32, token string) *capturev1.CaptureRequest {
	query := url.Values{}
	query.Set("workspace", workspaceID)
	query.Set("page", input.PageID)
	query.Set("display", input.DisplayID)
	query.Set("state", input.StateID)
	query.Set("revision", strconv.FormatInt(int64(input.ExpectedRevision), 10))
	return &capturev1.CaptureRequest{
		Url: "scenario=react-component-library,path=/workspace-capture/" + url.PathEscape(workspaceID) + "?" + query.Encode(),
		Captures: []capturev1.CaptureType{
			capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT,
			capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY,
		},
		Dimensions:         &capturev1.Dimensions{Width: &width, Height: &height},
		WaitFor:            &capturev1.WaitFor{Spec: &capturev1.WaitFor_Selector{Selector: workspaceCaptureReadySelector}},
		ScreenshotSelector: workspaceCaptureScreenshotSelector,
		RequestToken:       token,
		Direction:          "ltr",
	}
}

func workspaceCaptureToken(workspaceID string, input captureStateRequest) string {
	identity := fmt.Sprintf("%s\x00%d\x00%s\x00%s\x00%s\x00%s", workspaceID, input.ExpectedRevision, input.PageID, input.DisplayID, input.StateID, input.AttemptID)
	digest := sha256.Sum256([]byte(identity))
	return "rcl-state-" + hex.EncodeToString(digest[:])
}

func validCaptureAttemptID(id string) bool { return captureAttemptIDPattern.MatchString(id) }

func locateWorkspaceState(workspace *workspacev1.Workspace, input captureStateRequest) (*workspacev1.WorkspacePage, *workspacev1.WorkspaceDisplay, *workspacev1.WorkspaceState) {
	if workspace == nil {
		return nil, nil, nil
	}
	for _, page := range workspace.GetPages() {
		if page.GetId() != input.PageID {
			continue
		}
		for _, display := range page.GetDisplays() {
			if display.GetId() != input.DisplayID {
				continue
			}
			for _, state := range display.GetStates() {
				if state.GetId() == input.StateID {
					return page, display, state
				}
			}
		}
	}
	return nil, nil, nil
}

func decodeWorkspaceCaptureEvidence(raw string) (workspaceCaptureEvidence, bool) {
	var evidence workspaceCaptureEvidence
	if raw == "" || json.Unmarshal([]byte(raw), &evidence) != nil || evidence.SchemaVersion != workspaceCaptureEvidenceSchema {
		return workspaceCaptureEvidence{}, false
	}
	return evidence, true
}

func captureArtifactReferences(response *capturev1.CaptureResponse) (screenshot, accessibility string) {
	if response == nil {
		return "", ""
	}
	for _, artifact := range response.GetArtifacts() {
		if artifact == nil || strings.TrimSpace(artifact.GetReference()) == "" {
			continue
		}
		switch artifact.GetType() {
		case capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT:
			screenshot = artifact.GetReference()
		case capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY:
			accessibility = artifact.GetReference()
		}
	}
	return screenshot, accessibility
}

func uuidLikeWorkspaceID(id string) bool {
	return len(id) > 0 && len(id) <= 128 && strings.TrimSpace(id) == id && !strings.ContainsAny(id, `/\\`)
}

func (h *Handler) captureWorkspaceError(w http.ResponseWriter, operation string, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		httprest.WriteError(w, http.StatusNotFound, "not_found", "workspace was not found")
		return
	}
	if errors.Is(err, domain.ErrCurated) {
		httprest.WriteError(w, http.StatusConflict, "failed_precondition", err.Error())
		return
	}
	h.logger.Printf("workspaces: %s: %v", operation, err)
	httprest.WriteError(w, http.StatusInternalServerError, "internal", "workspace storage failure")
}
