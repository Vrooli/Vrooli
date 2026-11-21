package workspaces

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"connectrpc.com/connect"
	aiinference "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	"google.golang.org/protobuf/encoding/protojson"

	"react-component-library/internal/designassist"
	"react-component-library/internal/designinference"
)

// Assistant is the durable inference seam. designinference.Service satisfies
// it; tests substitute a fake so no gateway is needed.
type Assistant interface {
	Run(context.Context, string, designinference.Request) (designinference.Operation, error)
}

var assistIdempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,199}$`)

// WithAssistant attaches the inference runner used by AssistWorkspace.
func (h *Handler) WithAssistant(assistant Assistant) *Handler {
	h.assistant = assistant
	return h
}

// AssistWorkspace proposes a patch against one page. It never writes the
// aggregate: the studio applies the returned operations as one undoable
// command, so a discarded proposal leaves storage untouched.
func (h *Handler) AssistWorkspace(ctx context.Context, req *connect.Request[workspacev1.AssistWorkspaceRequest]) (*connect.Response[workspacev1.AssistWorkspaceResponse], error) {
	if h.assistant == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("workspace assistance is not configured"))
	}
	if err := requireID(req.Msg.GetId()); err != nil {
		return nil, err
	}
	if !assistIdempotencyPattern.MatchString(req.Msg.GetIdempotencyKey()) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("idempotency_key must be 8 to 200 letters, digits, dots, colons, underscores or dashes"))
	}
	workspace, err := h.repository.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, h.mapError("load workspace for assistance", err)
	}
	// The revision is the staleness proof the client also checks locally. A
	// proposal computed against another revision could target a layer that no
	// longer exists, so the RPC refuses instead of guessing.
	if workspace.GetRevision() != req.Msg.GetBaseRevision() {
		return nil, connect.NewError(connect.CodeAborted,
			fmt.Errorf("workspace changed: current revision is %d", workspace.GetRevision()))
	}

	doc, _, err := assistDocument(workspace, req.Msg.GetPageId(), req.Msg.GetActiveStates())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	scope := designassist.Scope{
		Kind:    designassist.ScopeKind(req.Msg.GetScope().GetKind()),
		FrameID: req.Msg.GetScope().GetDisplayId(),
		NodeID:  req.Msg.GetScope().GetNodeId(),
	}
	inferenceRequest, allowed, err := designassist.Prepare(doc, scope, req.Msg.GetPrompt())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	operation, err := h.assistant.Run(ctx, req.Msg.GetIdempotencyKey(), inferenceRequest)
	if err != nil {
		if errors.Is(err, designinference.ErrConflict) {
			return nil, connect.NewError(connect.CodeAlreadyExists,
				errors.New("that idempotency key is already bound to a different request"))
		}
		h.logger.Printf("workspaces: assist dispatch: %v", err)
		return nil, connect.NewError(connect.CodeUnavailable,
			errors.New("the AI gateway could not be reached; nothing was changed"))
	}

	result := &workspacev1.WorkspaceAssistOperation{Id: operation.ID, State: operation.State, Detail: operation.Detail}
	if operation.ResponseJSON != "" {
		var response aiinference.RunResponse
		if err := protojson.Unmarshal([]byte(operation.ResponseJSON), &response); err != nil {
			h.logger.Printf("workspaces: read inference response: %v", err)
			return nil, connect.NewError(connect.CodeInternal, errors.New("the AI gateway response could not be read"))
		}
		result.Provider, result.Model = response.Provider, response.Model
		if response.Error != nil && response.Error.Message != "" {
			result.Detail = response.Error.Message
		}
		if operation.State == "completed" {
			proposal, err := designassist.DecodeProposal(inferenceRequest.Schema, response.ValueJson)
			if err != nil {
				return nil, connect.NewError(connect.CodeFailedPrecondition,
					fmt.Errorf("the model returned an unusable patch: %.300s", err.Error()))
			}
			if err := designassist.Validate(doc, allowed, proposal); err != nil {
				return nil, connect.NewError(connect.CodeFailedPrecondition,
					fmt.Errorf("the model returned a patch this page cannot apply: %.300s", err.Error()))
			}
			encoded, err := json.Marshal(proposal)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.New("the proposal could not be encoded"))
			}
			return connect.NewResponse(&workspacev1.AssistWorkspaceResponse{
				Revision:     workspace.GetRevision(),
				Operation:    result,
				ProposalJson: string(encoded),
			}), nil
		}
	}

	switch operation.State {
	case "dispatch_unknown":
		return nil, connect.NewError(connect.CodeUnavailable,
			errors.New("the AI gateway did not answer; nothing was changed"))
	case "failed":
		detail := strings.TrimSpace(operation.Detail)
		if detail == "" {
			detail = "the AI gateway refused the request"
		}
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("the AI gateway could not produce a patch: "+detail))
	default:
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("the AI gateway returned no patch"))
	}
}

// assistDocument projects one page into the assistant's view. A display is a
// frame; the state the client says it is showing supplies that frame's spec.
// Reading only stored bytes keeps the model context identical for every caller.
func assistDocument(
	workspace *workspacev1.Workspace,
	pageID string,
	active []*workspacev1.WorkspaceAssistState,
) (designassist.Document, string, error) {
	page, err := selectPage(workspace, pageID)
	if err != nil {
		return designassist.Document{}, "", err
	}
	chosen := make(map[string]string, len(active))
	for _, entry := range active {
		id := strings.TrimSpace(entry.GetDisplayId())
		if id == "" {
			return designassist.Document{}, "", errors.New("active_states entries require a display_id")
		}
		chosen[id] = entry.GetStateId()
	}
	if len(page.GetDisplays()) == 0 {
		return designassist.Document{}, "", errors.New("that page has no displays to compose")
	}

	frames := make([]map[string]any, 0, len(page.GetDisplays()))
	for _, display := range page.GetDisplays() {
		state, err := selectState(display, chosen[display.GetId()])
		if err != nil {
			return designassist.Document{}, "", err
		}
		spec, err := stateSpec(page, state)
		if err != nil {
			return designassist.Document{}, "", fmt.Errorf("read composition for display %q: %w", display.GetId(), err)
		}
		frames = append(frames, map[string]any{
			"id":     display.GetId(),
			"name":   deviceName(display.GetDevice()),
			"device": deviceSpelling(display.GetDevice()),
			"spec":   spec,
		})
	}
	framesJSON, err := json.Marshal(frames)
	if err != nil {
		return designassist.Document{}, "", fmt.Errorf("encode page frames: %w", err)
	}
	theme := workspace.GetThemeJson()
	if strings.TrimSpace(theme) == "" {
		theme = "{}"
	}
	doc, err := designassist.ParseDocument(framesJSON, json.RawMessage(theme))
	if err != nil {
		return designassist.Document{}, "", fmt.Errorf("the page composition cannot be assisted: %w", err)
	}
	doc.Name = page.GetName()
	return doc, page.GetId(), nil
}

func selectPage(workspace *workspacev1.Workspace, pageID string) (*workspacev1.WorkspacePage, error) {
	if len(workspace.GetPages()) == 0 {
		return nil, errors.New("that workspace has no pages")
	}
	if strings.TrimSpace(pageID) == "" {
		return workspace.GetPages()[0], nil
	}
	for _, page := range workspace.GetPages() {
		if page.GetId() == pageID {
			return page, nil
		}
	}
	return nil, fmt.Errorf("page %q is not part of this workspace", pageID)
}

func selectState(display *workspacev1.WorkspaceDisplay, stateID string) (*workspacev1.WorkspaceState, error) {
	if len(display.GetStates()) == 0 {
		return nil, fmt.Errorf("display %q has no states", display.GetId())
	}
	if strings.TrimSpace(stateID) == "" {
		return display.GetStates()[0], nil
	}
	for _, state := range display.GetStates() {
		if state.GetId() == stateID {
			return state, nil
		}
	}
	return nil, fmt.Errorf("state %q is not part of display %q", stateID, display.GetId())
}

// stateSpec reads the composition a state renders. New and migrated states
// store it under the `spec` key; a bare node is also accepted so an older
// imported state still assists. A state with neither falls back to the page
// composition, which is what the studio renders in that case.
func stateSpec(page *workspacev1.WorkspacePage, state *workspacev1.WorkspaceState) (json.RawMessage, error) {
	raw := strings.TrimSpace(state.GetArgsJson())
	if raw != "" && raw != "null" && raw != "{}" {
		var payload struct {
			Spec json.RawMessage `json:"spec"`
		}
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			return nil, fmt.Errorf("args_json is not an object: %w", err)
		}
		if len(payload.Spec) > 0 {
			return payload.Spec, nil
		}
		var node map[string]any
		if err := json.Unmarshal([]byte(raw), &node); err == nil {
			if _, hasID := node["id"]; hasID {
				if _, hasType := node["type"]; hasType {
					return json.RawMessage(raw), nil
				}
			}
		}
	}
	pageSpec := strings.TrimSpace(page.GetSpecJson())
	if pageSpec == "" || pageSpec == "null" || pageSpec == "{}" {
		return nil, errors.New("the page has no composition yet")
	}
	var nested struct {
		Spec json.RawMessage `json:"spec"`
	}
	if err := json.Unmarshal([]byte(pageSpec), &nested); err == nil && len(nested.Spec) > 0 {
		return nested.Spec, nil
	}
	return json.RawMessage(pageSpec), nil
}

func deviceSpelling(device workspacev1.DisplayDevice) string {
	switch device {
	case workspacev1.DisplayDevice_DISPLAY_DEVICE_TABLET:
		return "tablet"
	case workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE:
		return "phone"
	default:
		return "desktop"
	}
}

func deviceName(device workspacev1.DisplayDevice) string {
	switch device {
	case workspacev1.DisplayDevice_DISPLAY_DEVICE_TABLET:
		return "Tablet"
	case workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE:
		return "Phone"
	default:
		return "Desktop"
	}
}
