package agentaccess

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

var screenshotTestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,100}$`)

func validScreenshotTestID(value string) bool {
	return screenshotTestIDPattern.MatchString(value)
}

// HarnessTaskRequest is the subset of common browser harness operations that
// can be represented safely by BAS typed actions. Unknown JSON fields fail
// closed through decodeJSON. In particular this contract has no script, CDP,
// launch, close, browser object, or driver endpoint field.
type HarnessTaskRequest struct {
	Operation string   `json:"operation"`
	TestID    string   `json:"test_id,omitempty"`
	WaitUntil string   `json:"wait_until,omitempty"`
	SettleMs  int      `json:"settle_ms,omitempty"`
	Selector  string   `json:"selector,omitempty"`
	Ref       string   `json:"ref,omitempty"`
	Value     string   `json:"value,omitempty"`
	URL       string   `json:"url,omitempty"`
	Key       string   `json:"key,omitempty"`
	Modifiers []string `json:"modifiers,omitempty"`
	X         int32    `json:"x,omitempty"`
	Y         int32    `json:"y,omitempty"`
	DeltaX    int32    `json:"delta_x,omitempty"`
	DeltaY    int32    `json:"delta_y,omitempty"`
}

func (h *HTTPHandler) runHarnessTask(w http.ResponseWriter, r *http.Request) {
	var request HarnessTaskRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	sessionID := chi.URLParam(r, "sessionID")
	switch strings.TrimSpace(request.Operation) {
	case "receipt":
		result, err := h.Broker.SettlementReceipt(r.Context(), sessionID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeResponse(w, map[string]any{"settlement_status": result.Status, "receipt": result.Receipt}, nil)
	case "observe":
		if request.TestID != "" || request.WaitUntil != "" || request.SettleMs != 0 || request.Selector != "" || request.Ref != "" || request.Value != "" || request.URL != "" || request.Key != "" || len(request.Modifiers) != 0 || request.X != 0 || request.Y != 0 || request.DeltaX != 0 || request.DeltaY != 0 {
			writeErrorStatus(w, http.StatusBadRequest, errors.New("observe does not accept target fields"))
			return
		}
		result, err := h.Broker.Observe(r.Context(), sessionID, ObserveRequest{})
		writeResponse(w, result, err)
	case "screenshot":
		if request.TestID == "" || !validScreenshotTestID(request.TestID) || request.Selector != "" || request.Ref != "" || request.Value != "" || request.URL != "" || (request.WaitUntil != "" && request.WaitUntil != "networkidle") || request.SettleMs < 0 || request.SettleMs > 5000 {
			writeErrorStatus(w, http.StatusBadRequest, errors.New("screenshot requires a bounded test_id and no other target fields"))
			return
		}
		result, err := h.Broker.Observe(r.Context(), sessionID, ObserveRequest{TestID: request.TestID, WaitUntil: request.WaitUntil, SettleMs: request.SettleMs})
		writeResponse(w, result, err)
	case "read", "locate":
		if request.TestID != "" || request.WaitUntil != "" || request.SettleMs != 0 {
			writeErrorStatus(w, http.StatusBadRequest, errors.New("accessibility read does not accept screenshot fields"))
			return
		}
		result, err := h.Broker.AccessibilityTree(r.Context(), sessionID)
		writeResponse(w, result, err)
	case "navigate", "click", "fill", "type":
		if request.TestID != "" || request.WaitUntil != "" || request.SettleMs != 0 {
			writeErrorStatus(w, http.StatusBadRequest, errors.New("action does not accept screenshot fields"))
			return
		}
		instruction, targetURL, err := harnessTaskInstruction(request)
		if err != nil {
			writeError(w, err)
			return
		}
		result, err := h.Broker.Act(r.Context(), sessionID, ActRequest{
			Instruction: instruction,
			TargetURL:   targetURL,
		})
		writeResponse(w, result, err)
	default:
		writeErrorStatus(w, http.StatusBadRequest, errors.New("unsupported harness operation"))
	}
}

func harnessTaskInstruction(request HarnessTaskRequest) (contracts.CompiledInstruction, string, error) {
	selector := strings.TrimSpace(request.Selector)
	ref := strings.TrimSpace(request.Ref)
	if ref != "" {
		if selector != "" || len(ref) > 96 || !strings.HasPrefix(ref, "e") || strings.TrimLeft(ref[1:], "0123456789") != "" || len(ref) < 2 {
			return contracts.CompiledInstruction{}, "", errors.New("ref must be an opaque accessibility reference")
		}
		// Only the Broker/driver understand this sentinel. The caller never sends
		// or receives a CSS selector when using an accessibility reference.
		selector = "@bas-ref:" + ref
	}
	var action *basactions.ActionDefinition
	switch strings.TrimSpace(request.Operation) {
	case "navigate":
		if strings.TrimSpace(request.URL) == "" || selector != "" || request.Ref != "" || request.Value != "" {
			return contracts.CompiledInstruction{}, "", errors.New("navigate requires only url")
		}
		action = &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: request.URL}}}
		return contracts.CompiledInstruction{Action: action}, request.URL, nil
	case "click":
		if selector == "" || request.URL != "" || request.Value != "" {
			return contracts.CompiledInstruction{}, "", errors.New("click requires only selector")
		}
		action = &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK, Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: selector}}}
	case "fill", "type":
		if selector == "" || request.URL != "" {
			return contracts.CompiledInstruction{}, "", errors.New("fill/type requires selector and optional value")
		}
		clearFirst := strings.TrimSpace(request.Operation) == "fill"
		action = &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_INPUT, Params: &basactions.ActionDefinition_Input{Input: &basactions.InputParams{Selector: selector, Value: request.Value, ClearFirst: &clearFirst}}}
	case "scroll":
		if selector != "" || ref != "" || request.URL != "" || request.Value != "" || request.Key != "" || len(request.Modifiers) != 0 ||
			request.X < 0 || request.X > 10_000 || request.Y < 0 || request.Y > 10_000 ||
			request.DeltaX < -10_000 || request.DeltaX > 10_000 || request.DeltaY < -10_000 || request.DeltaY > 10_000 ||
			(request.DeltaX == 0 && request.DeltaY == 0) {
			return contracts.CompiledInstruction{}, "", errors.New("scroll requires bounded coordinates and a nonzero bounded delta")
		}
		behavior := basactions.ScrollBehavior_SCROLL_BEHAVIOR_AUTO
		x, y, deltaX, deltaY := request.X, request.Y, request.DeltaX, request.DeltaY
		action = &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_SCROLL, Params: &basactions.ActionDefinition_Scroll{Scroll: &basactions.ScrollParams{
			X: &x, Y: &y, DeltaX: &deltaX, DeltaY: &deltaY, Behavior: &behavior,
		}}}
	case "keyboard":
		if request.Key == "" || len(request.Key) > 32 || selector != "" || ref != "" || request.URL != "" || request.Value != "" ||
			request.X != 0 || request.Y != 0 || request.DeltaX != 0 || request.DeltaY != 0 || len(request.Modifiers) > 1 {
			return contracts.CompiledInstruction{}, "", errors.New("keyboard requires one key and at most one modifier")
		}
		key := request.Key
		press := basactions.KeyAction_KEY_ACTION_PRESS
		keyboard := &basactions.KeyboardParams{Key: &key, Action: &press}
		if len(request.Modifiers) == 1 {
			switch strings.ToLower(strings.TrimSpace(request.Modifiers[0])) {
			case "ctrl":
				keyboard.Modifiers = []basactions.KeyboardModifier{basactions.KeyboardModifier_KEYBOARD_MODIFIER_CTRL}
			case "meta":
				keyboard.Modifiers = []basactions.KeyboardModifier{basactions.KeyboardModifier_KEYBOARD_MODIFIER_META}
			default:
				return contracts.CompiledInstruction{}, "", errors.New("keyboard modifier must be ctrl or meta")
			}
		}
		action = &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_KEYBOARD, Params: &basactions.ActionDefinition_Keyboard{Keyboard: keyboard}}
	default:
		return contracts.CompiledInstruction{}, "", errors.New("operation is not an action")
	}
	return contracts.CompiledInstruction{Action: action}, "", nil
}
