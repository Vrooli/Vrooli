package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	agentharness "github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliutil"
)

func receiptTestEnv(endpoint string) func(string) string {
	values := map[string]string{
		hookReceiptURLKey:   endpoint,
		hookReceiptTokenKey: "per-launch-nonce-secret",
		hookInvocationIDKey: "87e26b23-1cc0-4ae7-b39c-93dd352b2bd9",
		identityTokenKey:    "run-identity-secret",
	}
	return func(key string) string { return values[key] }
}

func receiptTestDecision(action agentharness.DecisionAction) agentharness.Decision {
	return agentharness.Decision{
		Action: action, ProviderID: "scenario-dependency-analyzer",
		EvaluatorID:          "scenario-dependency-analyzer:dependency-addition",
		EvaluationDurationMS: 17, Reason: "private evaluator reason",
	}
}

func receiptTestEvent() agentharness.ToolEvent {
	return agentharness.ToolEvent{Context: map[string]string{"session_id": "session-123"}}
}

func TestPersistEvaluatorReceiptPostsOnlyBoundedEvaluatorFacts(t *testing.T) {
	var got evaluatorReceipt
	var postCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		postCount++
		if r.Method != http.MethodPost || r.URL.Path != hookReceiptPath {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get(cliutil.HeaderAgentIdentityToken) != "run-identity-secret" || r.Header.Get("Authorization") != "" || r.Header.Get("X-Vrooli-Hook-Receipt-Token") != "per-launch-nonce-secret" {
			t.Errorf("auth headers did not match owner contract: %v", r.Header)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode receipt: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	decision := receiptTestDecision(agentharness.ActionDeny)
	now := time.Date(2026, 11, 17, 4, 5, 6, 0, time.FixedZone("offset", -5*60*60))
	if err := persistEvaluatorReceipt(receiptTestEvent(), "codex", decision, receiptTestEnv(server.URL+hookReceiptPath), server.Client(), now); err != nil {
		t.Fatal(err)
	}
	if postCount != 1 {
		t.Fatalf("receipt posts = %d, want one", postCount)
	}
	if !validUUID(got.ReceiptID) || got.InvocationID != "87e26b23-1cc0-4ae7-b39c-93dd352b2bd9" || got.SessionID != "session-123" || got.Runner != "codex" || got.EvaluatorID != decision.EvaluatorID || got.DurationMS != 17 || !got.OccurredAt.Equal(now.UTC()) {
		t.Fatalf("receipt = %+v", got)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private evaluator reason", "command", "tool_input", "rm -rf", "run-identity-secret", "per-launch-nonce-secret"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("receipt leaked %q: %s", forbidden, raw)
		}
	}
}

func TestPersistEvaluatorReceiptRejectsUntrustedOrMalformedInputs(t *testing.T) {
	cases := []struct {
		name      string
		event     agentharness.ToolEvent
		decision  agentharness.Decision
		mutateEnv func(map[string]string)
	}{
		{name: "allowed decision", event: receiptTestEvent(), decision: receiptTestDecision(agentharness.ActionAllow)},
		{name: "missing session", event: agentharness.ToolEvent{}, decision: receiptTestDecision(agentharness.ActionDeny)},
		{name: "control characters in session", event: agentharness.ToolEvent{Context: map[string]string{"session_id": "s\n1"}}, decision: receiptTestDecision(agentharness.ActionDeny)},
		{name: "foreign evaluator", event: receiptTestEvent(), decision: agentharness.Decision{Action: agentharness.ActionDeny, ProviderID: "p", EvaluatorID: "other:evaluator"}},
		{name: "missing launch nonce", event: receiptTestEvent(), decision: receiptTestDecision(agentharness.ActionDeny), mutateEnv: func(env map[string]string) { delete(env, hookReceiptTokenKey) }},
		{name: "missing invocation id", event: receiptTestEvent(), decision: receiptTestDecision(agentharness.ActionDeny), mutateEnv: func(env map[string]string) { delete(env, hookInvocationIDKey) }},
		{name: "malformed URL", event: receiptTestEvent(), decision: receiptTestDecision(agentharness.ActionDeny), mutateEnv: func(env map[string]string) { env[hookReceiptURLKey] = "http://user:pass@localhost/path" }},
		{name: "remote HTTP URL", event: receiptTestEvent(), decision: receiptTestDecision(agentharness.ActionDeny), mutateEnv: func(env map[string]string) { env[hookReceiptURLKey] = "http://example.com" + hookReceiptPath }},
		{name: "wrong route", event: receiptTestEvent(), decision: receiptTestDecision(agentharness.ActionDeny), mutateEnv: func(env map[string]string) { env[hookReceiptURLKey] = "https://example.com/other" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := map[string]string{
				hookReceiptURLKey:   "https://example.com" + hookReceiptPath,
				hookReceiptTokenKey: "nonce",
				hookInvocationIDKey: "87e26b23-1cc0-4ae7-b39c-93dd352b2bd9",
				identityTokenKey:    "identity",
			}
			if tc.mutateEnv != nil {
				tc.mutateEnv(values)
			}
			err := persistEvaluatorReceipt(tc.event, "codex", tc.decision, func(key string) string { return values[key] }, nil, time.Now())
			if err == nil {
				t.Fatal("expected invalid receipt inputs to be rejected")
			}
		})
	}
}

func TestEvaluatorReceiptFailureDoesNotChangeDenial(t *testing.T) {
	decision := receiptTestDecision(agentharness.ActionDeny)
	var stdout, stderr strings.Builder
	err := persistEvaluatorReceipt(receiptTestEvent(), "codex", decision, func(string) string { return "" }, nil, time.Now())
	if err == nil {
		t.Fatal("missing config should fail receipt persistence")
	}
	decisionErr := answerHook("codex", decision, &stdout, &stderr)
	var denied exitError
	if !errors.As(decisionErr, &denied) || denied.code != codexDenyExit {
		t.Fatalf("receipt failure changed enforcement result: err=%v", decisionErr)
	}
	if !strings.Contains(stderr.String(), "Blocked by Vrooli agent policy") || strings.Contains(stderr.String(), "receipt") || strings.Contains(stdout.String(), "private evaluator reason") {
		t.Fatalf("receipt failure changed denial or surfaced receipt details: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
