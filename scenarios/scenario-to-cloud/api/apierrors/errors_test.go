package apierrors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestWriteProducesTypedEnvelope [REQ:STC-P0-017] proves the wire shape every
// client decodes: {"error":{code,message,retryable,next_action,details}} in
// snake_case with the status derived from the code.
func TestWriteProducesTypedEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	err := New(CodeDeploymentSelectorAmbiguous, "two matches").
		WithRetryable(false).
		WithDetail("candidates", []string{"a", "b"}).
		WithNextAction(NextAction{Owner: "scenario-to-cloud", Kind: "selector", Reference: "id", Label: "Select by id"})
	Write(rec, err)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q", ct)
	}
	var body map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	e := body["error"]
	if e["code"] != CodeDeploymentSelectorAmbiguous {
		t.Fatalf("code = %v", e["code"])
	}
	if e["message"] != "two matches" {
		t.Fatalf("message = %v", e["message"])
	}
	if e["retryable"] != false {
		t.Fatalf("retryable = %v", e["retryable"])
	}
	next, ok := e["next_action"].(map[string]any)
	if !ok || next["kind"] != "selector" || next["reference"] != "id" {
		t.Fatalf("next_action = %v", e["next_action"])
	}
	details, ok := e["details"].(map[string]any)
	if !ok || details["candidates"] == nil {
		t.Fatalf("details = %v", e["details"])
	}
	for _, forbidden := range []string{"nextAction", "hint", "http_status", "HTTPStatus"} {
		if _, present := e[forbidden]; present {
			t.Fatalf("wire body must not carry %q: %v", forbidden, e)
		}
	}
}

// TestFromHTTPRoundTrip [REQ:STC-P0-017] proves a Go client reads back the
// exact code, retryability and next action the server wrote.
func TestFromHTTPRoundTrip(t *testing.T) {
	rec := httptest.NewRecorder()
	Write(rec, New(CodeRequestKeyConflict, "different plan").WithRetryable(false).
		WithNextAction(NextAction{Owner: "scenario-to-cloud", Kind: "operation", Reference: "op-1", Label: "Inspect"}).
		WithDetail("existing_operation_id", "op-1"))

	got := FromHTTP(rec.Code, rec.Body.Bytes())
	if got.Code != CodeRequestKeyConflict || got.Message != "different plan" || got.Retryable {
		t.Fatalf("decoded = %+v", got)
	}
	if got.NextAction == nil || got.NextAction.Reference != "op-1" {
		t.Fatalf("next action = %+v", got.NextAction)
	}
	if got.Details["existing_operation_id"] != "op-1" {
		t.Fatalf("details = %v", got.Details)
	}
	if got.Status() != http.StatusConflict {
		t.Fatalf("status = %d", got.Status())
	}
	if ExitCodeFor(got.Code) != ExitRefused {
		t.Fatalf("exit code = %d, want refused", ExitCodeFor(got.Code))
	}
}

// TestFromHTTPUntypedBodyStillYieldsStableCode keeps clients off free text:
// a legacy or proxy body maps to a code by status and keeps the raw body.
func TestFromHTTPUntypedBodyStillYieldsStableCode(t *testing.T) {
	got := FromHTTP(http.StatusNotFound, []byte("<html>gateway</html>"))
	if got.Code != CodeDeploymentNotFound {
		t.Fatalf("code = %q", got.Code)
	}
	if got.Details["body"] != "<html>gateway</html>" {
		t.Fatalf("details = %v", got.Details)
	}
	empty := FromHTTP(http.StatusBadGateway, nil)
	if empty.Code != CodeInternal || empty.Details != nil {
		t.Fatalf("empty body decoded to %+v", empty)
	}
}

// TestWriteWrapsUntypedErrors proves an unexpected error still reaches the
// client as the typed internal code and never as a bare string.
func TestWriteWrapsUntypedErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	Write(rec, fmt.Errorf("wrapped: %w", errors.New("disk on fire")))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	got := FromHTTP(rec.Code, rec.Body.Bytes())
	if got.Code != CodeInternal {
		t.Fatalf("code = %q", got.Code)
	}
	if got.Details["cause"] != "wrapped: disk on fire" {
		t.Fatalf("cause = %v", got.Details)
	}
}

// TestIsAndAsUnwrapChains proves typed errors survive fmt.Errorf wrapping.
func TestIsAndAsUnwrapChains(t *testing.T) {
	wrapped := fmt.Errorf("handler: %w", New(CodeFenceStale, "fence 3 < 4"))
	if !Is(wrapped, CodeFenceStale) {
		t.Fatalf("Is failed to see code through wrap")
	}
	if As(wrapped).Status() != http.StatusConflict {
		t.Fatalf("status = %d", As(wrapped).Status())
	}
	if As(nil) != nil {
		t.Fatalf("As(nil) must be nil")
	}
}

// TestEveryCodeHasStatusAndExitCode guards the code registry: a code without
// a status would surface as a 500 and a refusal as a failure.
func TestEveryCodeHasStatusAndExitCode(t *testing.T) {
	for code := range statusByCode {
		if StatusFor(code) == http.StatusInternalServerError && code != CodeInternal {
			t.Fatalf("code %q maps to 500", code)
		}
	}
	if StatusFor("made_up_code") != http.StatusInternalServerError {
		t.Fatalf("unknown code must map to 500")
	}
	if ExitCodeFor("") != ExitOK || ExitCodeFor(CodeNeedsInput) != ExitPending || ExitCodeFor(CodeInternal) != ExitFailed {
		t.Fatalf("exit code contract broken")
	}
}
