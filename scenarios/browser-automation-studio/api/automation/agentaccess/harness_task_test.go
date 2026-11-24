package agentaccess

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestHarnessTaskMapsOnlyTypedBrokerActions(t *testing.T) {
	tests := []struct {
		name      string
		request   HarnessTaskRequest
		wantType  basactions.ActionType
		wantURL   string
		wantError string
	}{
		{name: "navigate", request: HarnessTaskRequest{Operation: "navigate", URL: "https://example.test"}, wantType: basactions.ActionType_ACTION_TYPE_NAVIGATE, wantURL: "https://example.test"},
		{name: "click", request: HarnessTaskRequest{Operation: "click", Selector: "button#go"}, wantType: basactions.ActionType_ACTION_TYPE_CLICK},
		{name: "click opaque ref", request: HarnessTaskRequest{Operation: "click", Ref: "e123456"}, wantType: basactions.ActionType_ACTION_TYPE_CLICK},
		{name: "fill", request: HarnessTaskRequest{Operation: "fill", Selector: "input[name=q]", Value: "search"}, wantType: basactions.ActionType_ACTION_TYPE_INPUT},
		{name: "type", request: HarnessTaskRequest{Operation: "type", Selector: "input[name=q]", Value: "search"}, wantType: basactions.ActionType_ACTION_TYPE_INPUT},
		{name: "bounded scroll", request: HarnessTaskRequest{Operation: "scroll", X: 900, Y: 800, DeltaY: 1600}, wantType: basactions.ActionType_ACTION_TYPE_SCROLL},
		{name: "keyboard shortcut", request: HarnessTaskRequest{Operation: "keyboard", Key: "k", Modifiers: []string{"ctrl"}}, wantType: basactions.ActionType_ACTION_TYPE_KEYBOARD},
		{name: "launch forbidden", request: HarnessTaskRequest{Operation: "launch"}, wantError: "operation is not an action"},
		{name: "close forbidden", request: HarnessTaskRequest{Operation: "close"}, wantError: "operation is not an action"},
		{name: "evaluate forbidden", request: HarnessTaskRequest{Operation: "evaluate", Value: "document.cookie"}, wantError: "operation is not an action"},
		{name: "selector required", request: HarnessTaskRequest{Operation: "click"}, wantError: "click requires only selector"},
		{name: "ref format rejected", request: HarnessTaskRequest{Operation: "click", Ref: "eabc"}, wantError: "ref must be an opaque accessibility reference"},
		{name: "no mismatched fields", request: HarnessTaskRequest{Operation: "navigate", URL: "https://example.test", Value: "extra"}, wantError: "navigate requires only url"},
		{name: "scroll outside bounds", request: HarnessTaskRequest{Operation: "scroll", DeltaY: 10001}, wantError: "bounded"},
		{name: "keyboard unknown modifier", request: HarnessTaskRequest{Operation: "keyboard", Key: "k", Modifiers: []string{"alt"}}, wantError: "ctrl or meta"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			instruction, targetURL, err := harnessTaskInstruction(test.request)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("error=%v want substring %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if instruction.Action == nil || instruction.Action.Type != test.wantType || targetURL != test.wantURL {
				t.Fatalf("instruction=%+v target=%q", instruction, targetURL)
			}
			if test.request.Ref != "" {
				click := instruction.Action.GetClick()
				if click == nil || click.Selector != "@bas-ref:"+test.request.Ref {
					t.Fatalf("opaque ref was not preserved as a private sentinel: %+v", instruction.Action)
				}
			}
			if test.request.Operation == "scroll" {
				params := instruction.Action.GetScroll()
				if params == nil || params.GetX() != test.request.X || params.GetY() != test.request.Y || params.GetDeltaY() != test.request.DeltaY {
					t.Fatalf("scroll params were not mapped exactly: %+v", instruction.Action)
				}
			}
			if test.request.Operation == "keyboard" {
				params := instruction.Action.GetKeyboard()
				if params == nil || params.GetKey() != test.request.Key || params.GetAction() != basactions.KeyAction_KEY_ACTION_PRESS || len(params.GetModifiers()) != 1 {
					t.Fatalf("keyboard params were not mapped exactly: %+v", instruction.Action)
				}
			}
		})
	}
}

func TestScreenshotTargetsUseOnlyBoundedStableTestIDs(t *testing.T) {
	for _, test := range []struct {
		value string
		valid bool
	}{{"scene-canvas-pixels", true}, {"scene:canvas.v2", true}, {"", false}, {"canvas' [x]", false}, {strings.Repeat("a", 101), false}} {
		if got := validScreenshotTestID(test.value); got != test.valid {
			t.Errorf("validScreenshotTestID(%q)=%t want %t", test.value, got, test.valid)
		}
	}
}

func TestHarnessTaskReturnsSessionBoundLPBSReceiptWithoutCallerLookupFields(t *testing.T) {
	b, e, _ := closedEvidenceFixture(t)
	b.caller = bridgeCaller(e.callerID)
	e.identityToken = "signed-run"
	e.settlementResult = BrowserSettlementResult{Status: "settled", Receipt: []byte(`{"reservation_id":"lpbs-1","status":"settled","browser_session_minutes":2,"browser_actions":4,"browser_bytes":8192}`)}
	run := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/browser/sessions/"+e.sessionID+"/task", strings.NewReader(body))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("sessionID", e.sessionID)
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		req = req.WithContext(WithBrowserAgentCredentials(ctx, e.identityToken, e.grantReceipt))
		response := httptest.NewRecorder()
		(&HTTPHandler{Broker: b}).runHarnessTask(response, req)
		return response
	}
	response := run(`{"operation":"receipt"}`)
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"settlement_status":"settled"`) || !strings.Contains(response.Body.String(), `"reservation_id":"lpbs-1"`) {
		t.Fatalf("session receipt response = %d %s", response.Code, response.Body.String())
	}
	forged := run(`{"operation":"receipt","business_account_id":"other","reservation_id":"other"}`)
	if forged.Code != 400 {
		t.Fatalf("caller lookup fields status = %d, want 400: %s", forged.Code, forged.Body.String())
	}
}

func TestEphemeralAccessibilityRefsCannotBecomeDeterministicFlowSelectors(t *testing.T) {
	instruction, _, err := harnessTaskInstruction(HarnessTaskRequest{Operation: "click", Ref: "e123"})
	if err != nil {
		t.Fatal(err)
	}
	if !instructionUsesEphemeralAccessibilityRef(instruction) {
		t.Fatal("opaque ref action could be promoted as a stable selector")
	}
	selectorAction, _, err := harnessTaskInstruction(HarnessTaskRequest{Operation: "click", Selector: "button[data-testid=continue]"})
	if err != nil {
		t.Fatal(err)
	}
	if instructionUsesEphemeralAccessibilityRef(selectorAction) {
		t.Fatal("ordinary typed selector incorrectly marked as ephemeral")
	}
}
