package agentaccess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestPromoteTraceHandlesUserAttachedSessionWithoutOpeningAnotherLifecycle(t *testing.T) {
	profile := sessionpermission.Profile{ID: "profile", ProfileID: "run-profile", Effect: sessionpermission.EffectAct, Approval: sessionpermission.ApprovalNone, ProfileMode: "user_attached", URLAllow: []string{"https://example.test/*"}, View: true, Control: true}
	policy := &bridgePolicyRecorder{profile: profile}
	now := time.Now().UTC()
	trace := Trace{ID: "trace-1", SessionID: "session-1", PermissionRef: "run-profile", Events: []TraceEvent{{
		Sequence: 1, SessionID: "session-1", Kind: "action", PermissionClass: "act", EffectClass: "ui", URL: "https://example.test", OccurredAt: now,
		Instruction: &contracts.CompiledInstruction{Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK, Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: "#continue"}}}},
		Outcome:     &contracts.StepOutcome{Success: true},
	}, {Sequence: 2, SessionID: "session-1", Kind: "observation", PermissionClass: "observe", EffectClass: "none", URL: "https://example.test", OccurredAt: now.Add(time.Millisecond)}}}
	e := &entry{sessionID: "session-1", permissionRef: "run-profile", backend: string(sessionpermission.BackendUserAttached), leaseID: "lease-1", ownerID: "owner-1", callerID: "run:1", tabURL: "https://example.test/account", profile: profile, policy: policy, trace: trace}
	b := &Broker{profiles: bridgeProfileResolver{profile: profile, authorizer: policy}, caller: bridgeCaller("run:1"), entries: map[string]*entry{e.sessionID: e}}
	result, err := b.PromoteTrace(context.Background(), e.sessionID, PromoteRequest{
		Name:       "Account flow",
		Assertions: []*actionsv1.ActionDefinition{{Type: actionsv1.ActionType_ACTION_TYPE_ASSERT, Params: &actionsv1.ActionDefinition_Assert{Assert: &actionsv1.AssertParams{Selector: "#confirmation", Mode: basev1.AssertionMode_ASSERTION_MODE_VISIBLE}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ActionCount != 1 || len(result.Definition.GetNodes()) != 2 {
		t.Fatalf("promoted flow = %#v", result)
	}
	if len(policy.requests) != 1 || policy.requests[0].Kind != "trace_promote" || policy.requests[0].CurrentURL != e.tabURL {
		t.Fatalf("promotion authorization not bound to attached session URL: %#v", policy.requests)
	}
	if result.Definition.GetMetadata().GetLabels()["trace_id"] != trace.ID {
		t.Fatalf("trace provenance missing from promoted flow metadata: %#v", result.Definition.GetMetadata())
	}
}

func TestHTTPTraceDraftReturnsDeterministicPreviewWithoutWorkflowCatalog(t *testing.T) {
	profile := sessionpermission.Profile{ID: "profile", ProfileID: "run-profile", Effect: sessionpermission.EffectAct, Approval: sessionpermission.ApprovalNone, ProfileMode: "user_attached", URLAllow: []string{"https://example.test/*"}, View: true, Control: true}
	policy := &bridgePolicyRecorder{profile: profile}
	now := time.Now().UTC()
	trace := Trace{ID: "trace-draft-route", SessionID: "session-1", PermissionRef: "run-profile", Events: []TraceEvent{
		{Sequence: 1, SessionID: "session-1", Kind: "action", PermissionClass: "act", EffectClass: "ui", URL: "https://example.test", OccurredAt: now,
			Instruction: &contracts.CompiledInstruction{Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK, Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: "#continue"}}}}, Outcome: &contracts.StepOutcome{Success: true}},
		{Sequence: 2, SessionID: "session-1", Kind: "observation", PermissionClass: "observe", EffectClass: "none", URL: "https://example.test", OccurredAt: now.Add(time.Millisecond)},
	}}
	e := &entry{sessionID: "session-1", permissionRef: "run-profile", backend: string(sessionpermission.BackendUserAttached), leaseID: "lease-1", ownerID: "owner-1", callerID: "run:1", tabURL: "https://example.test/account", profile: profile, policy: policy, trace: trace}
	broker := &Broker{profiles: bridgeProfileResolver{profile: profile, authorizer: policy}, caller: bridgeCaller("run:1"), entries: map[string]*entry{e.sessionID: e}}
	router := chi.NewRouter()
	(&HTTPHandler{Broker: broker}).RegisterRoutes(router) // no workflow catalog or project-scope verifier is configured
	req := httptest.NewRequest(http.MethodPost, "/browser/sessions/session-1/trace/draft", strings.NewReader(`{"name":"Account flow","assertions":[{"type":"ACTION_TYPE_ASSERT","assert":{"selector":"#confirmation","mode":"ASSERTION_MODE_VISIBLE"}}]}`))
	req.Header.Set("X-Agent-Identity-Token", "opaque-token")
	req.Header.Set("X-Vrooli-Browser-Grant-Receipt", "opaque-grant")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("draft status=%d body=%s", response.Code, response.Body.String())
	}
	var result struct {
		DraftDigest     string          `json:"draft_digest"`
		TraceID         string          `json:"trace_id"`
		SessionID       string          `json:"session_id"`
		WorkflowID      string          `json:"workflow_id"`
		WorkflowVersion int32           `json:"workflow_version"`
		Definition      json.RawMessage `json:"definition"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode draft response: %v", err)
	}
	var definition workflowsv1.WorkflowDefinitionV2
	if err := protojson.Unmarshal(result.Definition, &definition); err != nil {
		t.Fatalf("decode protobuf JSON workflow definition: %v", err)
	}
	if result.DraftDigest == "" || result.TraceID != trace.ID || result.SessionID != e.sessionID || result.WorkflowID != "" || result.WorkflowVersion != 0 || len(definition.GetNodes()) != 2 {
		t.Fatalf("draft response persisted or omitted deterministic preview fields: %+v", result)
	}
	if len(policy.requests) != 1 || policy.requests[0].Kind != "trace_promote" || policy.requests[0].SessionID != e.sessionID || policy.requests[0].LeaseID != e.leaseID {
		t.Fatalf("draft authorization was not tied to the existing session lease: %+v", policy.requests)
	}
}

func TestRedactInstructionParameterizesNavigationAndSensitiveInput(t *testing.T) {
	input := contracts.CompiledInstruction{
		Metadata: map[string]string{"authorization_receipt": "receipt-secret", "kept": "yes"},
		Action:   &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_INPUT, Params: &actionsv1.ActionDefinition_Input{Input: &actionsv1.InputParams{Selector: "#password", Value: "actual-password", IsSensitive: boolPtr(true)}}},
	}
	redacted, parameters, err := redactInstruction(input, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got := redacted.Action.GetInput().GetValue(); got != "{{@params/input_7}}" {
		t.Fatalf("input value = %q", got)
	}
	if len(parameters) != 1 || parameters[0].Name != "input_7" || parameters[0].Kind != "sensitive" {
		t.Fatalf("parameters = %#v", parameters)
	}
	encoded, err := json.Marshal(redacted)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "actual-password") || strings.Contains(string(encoded), "receipt-secret") {
		t.Fatalf("redacted trace contains sensitive material: %s", encoded)
	}
	if input.Action.GetInput().GetValue() != "actual-password" {
		t.Fatal("redaction mutated the input instruction")
	}
	if input.Metadata["authorization_receipt"] != "receipt-secret" {
		t.Fatal("redaction mutated the caller metadata")
	}
	if len(redacted.Metadata) != 0 {
		t.Fatalf("freeform instruction metadata survived draft redaction: %#v", redacted.Metadata)
	}
}

func TestRedactInstructionParameterizesNavigationURL(t *testing.T) {
	instruction := contracts.CompiledInstruction{Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_NAVIGATE, Params: &actionsv1.ActionDefinition_Navigate{Navigate: &actionsv1.NavigateParams{Url: "https://example.test/private?token=secret"}}}}
	redacted, parameters, err := redactInstruction(instruction, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := redacted.Action.GetNavigate().GetUrl(); got != "{{@params/navigation_2}}" {
		t.Fatalf("url = %q", got)
	}
	if len(parameters) != 1 || parameters[0].Kind != "url" {
		t.Fatalf("parameters = %#v", parameters)
	}
	encoded, _ := json.Marshal(redacted)
	if strings.Contains(string(encoded), "secret") || strings.Contains(string(encoded), "private") {
		t.Fatalf("trace retained navigation URL: %s", encoded)
	}
}

func TestRedactInstructionRejectsArbitraryProtocolAndScriptActions(t *testing.T) {
	for _, action := range []*actionsv1.ActionDefinition{
		{Type: actionsv1.ActionType_ACTION_TYPE_EVALUATE, Params: &actionsv1.ActionDefinition_Evaluate{Evaluate: &actionsv1.EvaluateParams{Expression: "fetch('/secret')"}}},
		{Type: actionsv1.ActionType_ACTION_TYPE_COOKIE_STORAGE, Params: &actionsv1.ActionDefinition_CookieStorage{CookieStorage: &actionsv1.CookieStorageParams{Operation: actionsv1.CookieOperation_COOKIE_OPERATION_SET, Key: strPtr("password"), Value: strPtr("secret")}}},
	} {
		if _, _, err := redactInstruction(contracts.CompiledInstruction{Action: action}, 1); err == nil {
			t.Fatalf("action %s was accepted for automatic promotion", action.Type)
		}
	}
}

func TestCompileTraceDraftIsCanonicalAndInterleavesPostActionAssertions(t *testing.T) {
	trace := testCompileTrace(&actionsv1.ActionDefinition{
		Type:   actionsv1.ActionType_ACTION_TYPE_CLICK,
		Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: `[data-testid="continue"]`}},
	})
	req := PromoteRequest{Name: "Continue", Assertions: []*actionsv1.ActionDefinition{visibleAssertion(`[data-testid="continue"]`)}}
	first, err := compileTraceDraft(trace, trace.SessionID, req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := compileTraceDraft(trace, trace.SessionID, req)
	if err != nil {
		t.Fatal(err)
	}
	if first.DraftDigest == "" || first.DraftDigest != second.DraftDigest || !proto.Equal(first.Definition, second.Definition) {
		t.Fatalf("identical trace compiled nondeterministically: digest %q vs %q", first.DraftDigest, second.DraftDigest)
	}
	if first.DraftDigest != "d2eeac2e445553888735f0ecda0502c617f6dd01764070355e4f607ed3a3a0c3" {
		t.Fatalf("canonical V2 draft digest changed: %s", first.DraftDigest)
	}
	if first.ActionCount != 1 || len(first.Definition.GetNodes()) != 2 || len(first.Definition.GetEdges()) != 1 {
		t.Fatalf("action/assertion sequence = nodes %d edges %d", len(first.Definition.GetNodes()), len(first.Definition.GetEdges()))
	}
	if first.Definition.GetNodes()[0].GetAction().GetClick() == nil || first.Definition.GetNodes()[1].GetAction().GetAssert() == nil {
		t.Fatalf("post-action assertion was not interleaved: %v", first.Definition.GetNodes())
	}
	if len(first.Evidence) != 1 || first.Evidence[0].ActionSequence != 1 || first.Evidence[0].ObservationSequence != 2 || first.Evidence[0].PermissionClass != "act" || first.Evidence[0].EffectClass != "ui" {
		t.Fatalf("trace evidence was not preserved: %+v", first.Evidence)
	}
	if first.Evidence[0].ObservedOrigin != "https://example.test" {
		t.Fatalf("unexpected observed origin: %+v", first.Evidence[0])
	}
}

func TestCompileTraceDraftParameterizesInputsAndNavigationWithoutRetainingSecrets(t *testing.T) {
	now := time.Now().UTC()
	nav := &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_NAVIGATE, Params: &actionsv1.ActionDefinition_Navigate{Navigate: &actionsv1.NavigateParams{Url: "https://example.test/start?token=secret"}}}
	input := &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_INPUT, Params: &actionsv1.ActionDefinition_Input{Input: &actionsv1.InputParams{Selector: "#email", Value: "person@example.test", IsSensitive: boolPtr(true)}}}
	trace := Trace{ID: "trace-fixture", SessionID: "session-fixture", PermissionRef: "permission-fixture", Events: []TraceEvent{
		compileEvent(1, "action", "https://example.test", "https://example.test", "act", "navigation", now, nav),
		compileEvent(2, "observation", "https://example.test", "", "observe", "none", now.Add(time.Millisecond), nil),
		compileEvent(3, "action", "https://example.test", "", "act", "ui", now.Add(2*time.Millisecond), input),
		compileEvent(4, "observation", "https://example.test", "", "observe", "none", now.Add(3*time.Millisecond), nil),
	}}
	result, err := compileTraceDraft(trace, trace.SessionID, PromoteRequest{Name: "Lookup", Assertions: []*actionsv1.ActionDefinition{
		visibleAssertion("#email"), {Type: actionsv1.ActionType_ACTION_TYPE_ASSERT, Params: &actionsv1.ActionDefinition_Assert{Assert: &actionsv1.AssertParams{Selector: "#email", Mode: basev1.AssertionMode_ASSERTION_MODE_TEXT_CONTAINS, Expected: &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: "private page text"}}, FailureMessage: strPtr("private page text failure")}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	nodes := result.Definition.GetNodes()
	if len(nodes) != 4 || nodes[0].GetAction().GetNavigate().GetUrl() != "{{@params/navigation_1}}" || nodes[2].GetAction().GetInput().GetValue() != "{{@params/input_3}}" {
		t.Fatalf("input/navigation parameters are not stable: %v", nodes)
	}
	if len(result.RequiredParameters) != 3 || result.RequiredParameters[0].Kind != "url" || result.RequiredParameters[1].Kind != "sensitive" || result.RequiredParameters[2].Kind != "assertion-text" {
		t.Fatalf("required parameters = %+v", result.RequiredParameters)
	}
	if got := nodes[3].GetAction().GetAssert().GetExpected().GetStringValue(); got != "{{@params/assertion_text_3}}" {
		t.Fatalf("assertion text was not parameterized: %q", got)
	}
	if nodes[3].GetAction().GetAssert().FailureMessage != nil {
		t.Fatal("freeform assertion failure message survived draft compilation")
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"token=secret", "person@example.test", "private page text", "private page text failure"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("draft retained %q: %s", secret, encoded)
		}
	}
}

func TestCompileTraceDraftSupportsRedactedStableAttributeAssertions(t *testing.T) {
	for _, mode := range []basev1.AssertionMode{
		basev1.AssertionMode_ASSERTION_MODE_ATTRIBUTE_EQUALS,
		basev1.AssertionMode_ASSERTION_MODE_ATTRIBUTE_CONTAINS,
	} {
		t.Run(mode.String(), func(t *testing.T) {
			trace := testCompileTrace(&actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK, Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: "#continue"}}})
			assertion := &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_ASSERT, Params: &actionsv1.ActionDefinition_Assert{Assert: &actionsv1.AssertParams{
				Selector: "#continue", Mode: mode, AttributeName: strPtr("data-testid"),
				Expected: &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: "private-customer-marker"}},
			}}}
			result, err := compileTraceDraft(trace, trace.SessionID, PromoteRequest{Name: "Confirm", Assertions: []*actionsv1.ActionDefinition{assertion}})
			if err != nil {
				t.Fatal(err)
			}
			compiled := result.Definition.GetNodes()[1].GetAction().GetAssert()
			if compiled.GetAttributeName() != "data-testid" || compiled.GetMode() != mode || compiled.GetExpected().GetStringValue() != "{{@params/assertion_text_1}}" {
				t.Fatalf("attribute assertion was not preserved and parameterized: %+v", compiled)
			}
			if len(result.RequiredParameters) != 1 || result.RequiredParameters[0].Kind != "assertion-text" {
				t.Fatalf("attribute assertion parameter metadata = %+v", result.RequiredParameters)
			}
			encoded, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), "private-customer-marker") {
				t.Fatalf("attribute value leaked into draft: %s", encoded)
			}
		})
	}

	trace := testCompileTrace(&actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK, Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: "#continue"}}})
	unsafe := &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_ASSERT, Params: &actionsv1.ActionDefinition_Assert{Assert: &actionsv1.AssertParams{
		Selector: "#continue", Mode: basev1.AssertionMode_ASSERTION_MODE_ATTRIBUTE_EQUALS, AttributeName: strPtr("value"),
		Expected: &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: "secret"}},
	}}}
	if _, err := compileTraceDraft(trace, trace.SessionID, PromoteRequest{Name: "Confirm", Assertions: []*actionsv1.ActionDefinition{unsafe}}); err == nil || !strings.Contains(err.Error(), "stable trace allowlist") {
		t.Fatalf("secret-bearing attribute was not rejected: %v", err)
	}
}

func TestCompileTraceDraftRejectsForgedIncompleteAndUnsupportedTraceEvents(t *testing.T) {
	base := testCompileTrace(&actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK, Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: "#continue"}}})
	assertion := []*actionsv1.ActionDefinition{visibleAssertion("#continue")}
	cases := []struct {
		name   string
		mutate func(*Trace)
		req    []*actionsv1.ActionDefinition
	}{
		{name: "wrong session", mutate: func(trace *Trace) { trace.SessionID = "another-session" }},
		{name: "cross-session event", mutate: func(trace *Trace) { trace.Events[1].SessionID = "another-session" }},
		{name: "duplicate sequence", mutate: func(trace *Trace) { trace.Events[1].Sequence = 1 }},
		{name: "reordered events", mutate: func(trace *Trace) { trace.Events[0], trace.Events[1] = trace.Events[1], trace.Events[0] }},
		{name: "duplicate event timestamp", mutate: func(trace *Trace) { trace.Events[1].OccurredAt = trace.Events[0].OccurredAt }},
		{name: "reversed event timestamp", mutate: func(trace *Trace) { trace.Events[1].OccurredAt = trace.Events[0].OccurredAt.Add(-time.Nanosecond) }},
		{name: "missing sequence", mutate: func(trace *Trace) { trace.Events[1].Sequence = 0 }},
		{name: "missing observation", mutate: func(trace *Trace) { trace.Events = trace.Events[:1] }},
		{name: "incomplete observation", mutate: func(trace *Trace) { trace.Events[1].URL = "" }},
		{name: "redirect origin", mutate: func(trace *Trace) { trace.Events[1].URL = "https://outside.test" }},
		{name: "opaque ref", mutate: func(trace *Trace) { trace.Events[0].Instruction.Action.GetClick().Selector = "@bas-ref:e123" }},
		{name: "unstable selector", mutate: func(trace *Trace) { trace.Events[0].Instruction.Action.GetClick().Selector = "button:nth-child(2)" }},
		{name: "unsupported effect", mutate: func(trace *Trace) { trace.Events[0].EffectClass = "unsupported" }},
		{name: "missing permission evidence", mutate: func(trace *Trace) { trace.Events[0].PermissionClass = "" }},
		{name: "failed action", mutate: func(trace *Trace) { trace.Events[0].Outcome.Success = false }},
		{name: "forged event kind", mutate: func(trace *Trace) { trace.Events[1].Kind = "audio_observation" }},
		{name: "unstable assertion", req: []*actionsv1.ActionDefinition{visibleAssertion("button:nth-child(2)")}},
		{name: "text assertion without expected value", req: []*actionsv1.ActionDefinition{{Type: actionsv1.ActionType_ACTION_TYPE_ASSERT, Params: &actionsv1.ActionDefinition_Assert{Assert: &actionsv1.AssertParams{Selector: "#continue", Mode: basev1.AssertionMode_ASSERTION_MODE_TEXT_EQUALS}}}}},
		{name: "missing assertion", req: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trace := base
			trace.Events = append([]TraceEvent(nil), base.Events...)
			for i := range trace.Events {
				if base.Events[i].Instruction != nil && base.Events[i].Instruction.Action != nil {
					instruction := *base.Events[i].Instruction
					instruction.Action = proto.Clone(base.Events[i].Instruction.Action).(*actionsv1.ActionDefinition)
					trace.Events[i].Instruction = &instruction
				}
				if base.Events[i].Outcome != nil {
					outcome := *base.Events[i].Outcome
					trace.Events[i].Outcome = &outcome
				}
			}
			if tc.mutate != nil {
				tc.mutate(&trace)
			}
			reqAssertions := assertion
			if tc.req != nil || tc.name == "missing assertion" {
				reqAssertions = tc.req
			}
			if _, err := compileTraceDraft(trace, "session-fixture", PromoteRequest{Name: "Test", Assertions: reqAssertions}); err == nil {
				t.Fatal("invalid trace compiled to a promotable draft")
			}
		})
	}
}

func TestDraftTraceRejectsRevokedSessionBeforeCompilation(t *testing.T) {
	closedEntry := &entry{sessionID: "session-fixture", closed: true, trace: testCompileTrace(&actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK, Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: "#continue"}}})}
	broker := &Broker{entries: map[string]*entry{"session-fixture": closedEntry}}
	if _, err := broker.DraftTrace(context.Background(), "session-fixture", PromoteRequest{Name: "Test"}); err == nil {
		t.Fatal("revoked session was allowed to compile a trace")
	}
}

func TestDraftTraceRequiresExactEntryPermissionBinding(t *testing.T) {
	profile := sessionpermission.Profile{ID: "profile", ProfileID: "run-profile", Effect: sessionpermission.EffectAct, Approval: sessionpermission.ApprovalNone, ProfileMode: "user_attached", URLAllow: []string{"https://example.test/*"}, View: true, Control: true}
	policy := &bridgePolicyRecorder{profile: profile}
	trace := testCompileTrace(&actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK, Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: "#continue"}}})
	trace.PermissionRef = "forged-permission"
	e := &entry{sessionID: trace.SessionID, permissionRef: "run-profile", backend: string(sessionpermission.BackendUserAttached), leaseID: "lease-1", ownerID: "owner-1", callerID: "run:1", tabURL: "https://example.test/account", profile: profile, policy: policy, trace: trace}
	broker := &Broker{profiles: bridgeProfileResolver{profile: profile, authorizer: policy}, caller: bridgeCaller("run:1"), entries: map[string]*entry{e.sessionID: e}}
	if _, err := broker.DraftTrace(context.Background(), e.sessionID, PromoteRequest{Name: "Test", Assertions: []*actionsv1.ActionDefinition{visibleAssertion("#continue")}}); err == nil {
		t.Fatal("draft compiled against a trace with a mismatched permission binding")
	}
}

func testCompileTrace(action *actionsv1.ActionDefinition) Trace {
	now := time.Now().UTC()
	targetOrigin := ""
	effectClass := "ui"
	if nav := action.GetNavigate(); nav != nil {
		targetOrigin, effectClass = "https://example.test", "navigation"
	}
	return Trace{ID: "trace-fixture", SessionID: "session-fixture", PermissionRef: "permission-fixture", Events: []TraceEvent{
		compileEvent(1, "action", "https://example.test", targetOrigin, "act", effectClass, now, action),
		compileEvent(2, "observation", "https://example.test", "", "observe", "none", now.Add(time.Millisecond), nil),
	}}
}

func compileEvent(sequence uint64, kind, origin, targetOrigin, permission, effect string, at time.Time, action *actionsv1.ActionDefinition) TraceEvent {
	event := TraceEvent{Sequence: sequence, SessionID: "session-fixture", Kind: kind, URL: origin, TargetOrigin: targetOrigin, PermissionClass: permission, EffectClass: effect, OccurredAt: at}
	if action != nil {
		event.Instruction = &contracts.CompiledInstruction{Action: action}
		event.Outcome = &contracts.StepOutcome{Success: true}
	}
	return event
}

func visibleAssertion(selector string) *actionsv1.ActionDefinition {
	return &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_ASSERT, Params: &actionsv1.ActionDefinition_Assert{Assert: &actionsv1.AssertParams{Selector: selector, Mode: basev1.AssertionMode_ASSERTION_MODE_VISIBLE}}}
}

func boolPtr(v bool) *bool    { return &v }
func strPtr(v string) *string { return &v }
