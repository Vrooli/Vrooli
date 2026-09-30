package vision

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/testutil/hubmocks"
)

func TestRedactNavigationActionPreservesStructureAndRemovesSensitiveValues(t *testing.T) {
	const secret = "BAS_SYNTHETIC_PASSWORD_7f3e"
	action := map[string]interface{}{
		"type":     "input",
		"selector": "#password",
		"input": map[string]interface{}{
			"text":  secret,
			"value": secret,
		},
		"url":       "https://example.test/login?token=" + secret + "&keep=1",
		"reasoning": "enter password=" + secret,
	}

	redacted := redactNavigationAction(action)
	if got := redacted["selector"]; got != "#password" {
		t.Fatalf("selector = %v, want #password", got)
	}
	if got := redacted["input"].(map[string]interface{})["value"]; got != redactedNavigationValue {
		t.Fatalf("input value = %v, want redaction marker", got)
	}
	if got := redacted["url"].(string); strings.Contains(got, secret) || !strings.Contains(got, "keep=1") {
		t.Fatalf("redacted URL = %q, secret leaked or ordinary query lost", got)
	}
	if got := redacted["reasoning"].(string); strings.Contains(got, secret) {
		t.Fatalf("redacted reasoning leaked secret: %q", got)
	}
	wire, err := json.Marshal(redacted)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wire), secret) {
		t.Fatalf("serialized action leaked secret: %s", wire)
	}
}

func TestRedactNavigationActionKeepsOrdinaryInput(t *testing.T) {
	action := map[string]interface{}{
		"type":     "input",
		"selector": "#search",
		"input":    map[string]interface{}{"text": "browser automation"},
	}
	redacted := redactNavigationAction(action)
	input := redacted["input"].(map[string]interface{})
	if got := input["text"]; got != "browser automation" {
		t.Fatalf("ordinary text = %v, want unchanged", got)
	}
}

func TestRedactNavigationActionRedactsAssignmentSecretsInGenericDetails(t *testing.T) {
	const secret = "BAS_GENERIC_DETAIL_SECRET_7f3e"
	action := map[string]interface{}{
		"type":   "evaluate",
		"result": "password=" + secret + " status=ok",
		"text":   "token=" + secret,
	}

	redacted := redactNavigationAction(action)
	if got := redacted["result"]; got != "password=[REDACTED] status=ok" {
		t.Fatalf("generic result = %v, want assignment redacted", got)
	}
	if got := redacted["text"]; got != "token=[REDACTED]" {
		t.Fatalf("generic text = %v, want assignment redacted", got)
	}
	wire, err := json.Marshal(redacted)
	if err != nil {
		t.Fatalf("marshal redacted action: %v", err)
	}
	if strings.Contains(string(wire), secret) {
		t.Fatalf("generic detail secret leaked: %s", wire)
	}
}

func TestRedactNavigationURLRedactsCredentialsAndSensitiveQuery(t *testing.T) {
	const secret = "BAS_SYNTHETIC_TOKEN_91ab"
	got := redactNavigationURL("https://alice:" + secret + "@example.test/?access_token=" + secret + "&page=2")
	if strings.Contains(got, secret) || strings.Contains(got, "alice") {
		t.Fatalf("URL leaked credentials: %q", got)
	}
	if !strings.Contains(got, "page=2") || !strings.Contains(got, "access_token=%5BREDACTED%5D") {
		t.Fatalf("URL lost safe query or marker: %q", got)
	}
}

func TestPlaywrightCallbacksRedactLiveAndRecoveredNavigationData(t *testing.T) {
	const secret = "BAS_SYNTHETIC_NAV_SECRET_2c91"
	log := logrus.New()
	log.SetOutput(io.Discard)
	wsHub := hubmocks.New()
	nav := NewPlaywrightVisionNavigator(log, WithPlaywrightHub(wsHub))
	session := &NavigationSession{
		NavigationID: "nav_redaction",
		SessionID:    "session_redaction",
		StartedAt:    time.Now(),
		Status:       StatusNavigating,
	}
	nav.mu.Lock()
	nav.activeNavigations[session.NavigationID] = session
	nav.mu.Unlock()

	err := nav.HandleStepCallback(t.Context(), &NavigationStep{
		NavigationID: session.NavigationID,
		StepNumber:   1,
		Action: map[string]interface{}{
			"type":     "input",
			"selector": "#password",
			"value":    secret,
			"input":    map[string]interface{}{"value": secret},
		},
		Reasoning:  "enter password=" + secret,
		CurrentURL: "https://example.test/login?access_token=" + secret,
		Error:      "token=" + secret,
	})
	if err != nil {
		t.Fatalf("HandleStepCallback() error = %v", err)
	}

	nav.mu.RLock()
	snapshot := nav.activeNavigations[session.NavigationID].Snapshot()
	nav.mu.RUnlock()
	if got := snapshot.Steps[0].Value; got != redactedNavigationValue {
		t.Fatalf("step value = %q, want redaction marker", got)
	}
	for _, got := range []string{snapshot.Steps[0].URL, snapshot.Steps[0].Description, snapshot.Steps[0].Error} {
		if strings.Contains(got, secret) {
			t.Fatalf("step history leaked secret: %q", got)
		}
	}
	wire, err := json.Marshal(wsHub.LastBroadcastEnvelope())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wire), secret) {
		t.Fatalf("step envelope leaked secret: %s", wire)
	}

	err = nav.HandleCompleteCallback(t.Context(), &NavigationResult{
		NavigationID:  session.NavigationID,
		Status:        StatusCompleted,
		FinalURL:      "https://example.test/done?token=" + secret,
		Summary:       "completed password=" + secret,
		ExtractedData: map[string]interface{}{"password": secret, "safe": "ok"},
	})
	if err != nil {
		t.Fatalf("HandleCompleteCallback() error = %v", err)
	}
	nav.mu.RLock()
	snapshot = nav.activeNavigations[session.NavigationID].Snapshot()
	nav.mu.RUnlock()
	if got := snapshot.ExtractedData["password"]; got != redactedNavigationValue {
		t.Fatalf("extracted password = %v, want redaction marker", got)
	}
	if snapshot.ExtractedData["safe"] != "ok" {
		t.Fatalf("safe extracted data changed: %v", snapshot.ExtractedData["safe"])
	}
	wire, err = json.Marshal(wsHub.LastBroadcastEnvelope())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wire), secret) {
		t.Fatalf("completion envelope leaked secret: %s", wire)
	}
}

func TestClaudeCodeBroadcastRedactsToolInput(t *testing.T) {
	const secret = "BAS_SYNTHETIC_CLAUDE_SECRET_58d1"
	log := logrus.New()
	log.SetOutput(io.Discard)
	wsHub := hubmocks.New()
	nav := NewClaudeCodeVisionNavigator(log, WithClaudeCodeHub(wsHub))
	session := &claudeCodeSession{NavigationSession: &NavigationSession{
		NavigationID: "nav_claude_redaction",
		SessionID:    "session_claude_redaction",
		StartedAt:    time.Now(),
	}}
	nav.broadcastStep(session, &claudeStreamEvent{
		Name:  "mcp__claude-in-chrome__form_input",
		Input: json.RawMessage(`{"ref":"#password","text":"` + secret + `"}`),
	}, 1, "enter password="+secret)

	if wsHub.BroadcastEnvelopeCount() != 1 {
		t.Fatalf("broadcast count = %d, want 1", wsHub.BroadcastEnvelopeCount())
	}
	wire, err := json.Marshal(wsHub.LastBroadcastEnvelope())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wire), secret) {
		t.Fatalf("Claude broadcast leaked secret: %s", wire)
	}
	if !strings.Contains(string(wire), redactedNavigationValue) {
		t.Fatalf("Claude broadcast omitted redaction marker: %s", wire)
	}
}
