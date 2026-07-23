package main

import (
	"testing"

	clitest "agent-manager/cli/internal/testutil"

	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

func TestHTTPToWSURLAndWebSocketMessageHandling(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"http://localhost:18800/api/v1/ws", "ws://localhost:18800/api/v1/ws"},
		{"https://example.test/ws", "wss://example.test/ws"},
		{"ws://example.test/ws", "ws://example.test/ws"},
	} {
		got, err := httpToWSURL(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("httpToWSURL(%q)=%q, %v", tc.in, got, err)
		}
	}
	if _, err := httpToWSURL("ftp://example.test/ws"); err == nil {
		t.Fatal("unsupported scheme accepted")
	}

	app := &App{}
	for _, message := range []WebSocketMessage{
		{Type: "connected"},
		{Type: "pong"},
		{Type: "run_event", Payload: []byte(`{"sequence":2,"eventType":"tool.call","data":{"tool":"rg"}}`)},
		{Type: "run_progress", Payload: []byte(`{"phase":"executing","percentComplete":50,"currentAction":"testing"}`)},
		{Type: "run_status", Payload: []byte(`{"status":"running"}`)},
		{Type: "run_status", Payload: []byte(`{"status":"completed"}`)},
		{Type: "unknown", Payload: []byte(`{"diagnostic":"safe"}`)},
	} {
		if err := app.handleWSMessage(message, "run-1"); err != nil {
			t.Fatalf("handle %s: %v", message.Type, err)
		}
	}
	if err := app.handleWSMessage(WebSocketMessage{Type: "run_event", Payload: []byte(`{`)}, "run-1"); err == nil {
		t.Fatal("malformed event payload accepted")
	}
	if err := app.handleWSMessage(WebSocketMessage{Type: "run_progress", Payload: []byte(`{`)}, "run-1"); err == nil {
		t.Fatal("malformed progress payload accepted")
	}
	if err := app.handleWSMessage(WebSocketMessage{Type: "run_status", Payload: []byte(`{`)}, "run-1"); err == nil {
		t.Fatal("malformed status payload accepted")
	}
	if err := app.handleWSMessage(WebSocketMessage{Type: "run_event", RunID: "other"}, "run-1"); err != nil {
		t.Fatalf("foreign run filter: %v", err)
	}
}

func TestRolePolicyFormattingAndInputValidation(t *testing.T) {
	if policyRunnerLabel("RUNNER_TYPE_CLAUDE_CODE") != "claude-code" || policySelectionLabel("MODEL_SELECTION_TYPE_FALLBACK") != "fallback" {
		t.Fatal("policy enum labels were not normalized")
	}
	printPolicyStatus(nil)
	printPolicyStatus(&apipb.RolePolicyStatus{Ready: true, Path: "policy.json", ActiveDigest: "sha256:active", Requirement: &apipb.RolePolicyRequirement{Required: true, Reason: "safety"}, LastReloadAttempt: &apipb.RolePolicyReloadAttempt{Diagnostic: &apipb.RolePolicyDiagnostic{Code: "INVALID", Message: "bad", Cause: "root"}}})
	printPolicyDiagnostic(nil)
	printPolicyDiagnostic(&apipb.RolePolicyDiagnostic{Code: "INVALID", Message: "bad", Cause: "bad"})
	app := &App{}
	if err := app.cmdPolicy(nil); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"explain"}, {"explain", "other", "id"}, {"status", "--bad"}, {"unknown"}} {
		if err := app.cmdPolicy(args); err == nil {
			t.Fatalf("policy %v accepted", args)
		}
	}
}

func TestPermissionPolicyFormattingAndInputValidation(t *testing.T) {
	printPermissionPolicyStatus(nil)
	printPermissionPolicyDiagnostic(nil)
	printPermissionPolicyPlan(nil)
	printPermissionPolicyReconcile(nil)
	if jsonOutput, authorized, err := parsePermissionPolicyFlags("policy", []string{"--json"}, false); err != nil || !*jsonOutput || *authorized {
		t.Fatalf("ordinary flags json=%v authorized=%v err=%v", jsonOutput, authorized, err)
	}
	if _, authorized, err := parsePermissionPolicyFlags("policy", []string{"--i-was-explicitly-authorized"}, true); err != nil || !*authorized {
		t.Fatalf("reconcile authorization err=%v", err)
	}
	app := &App{}
	if err := app.cmdPermissionPolicy(nil); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"unknown"}, {"status", "--bad"}, {"reconcile"}} {
		if err := app.cmdPermissionPolicy(args); err == nil {
			t.Fatalf("permission policy %v accepted", args)
		}
	}
}

func TestRunnerCommandsRenderNonEmptyServiceResponses(t *testing.T) {
	server := clitest.NewRecordingServer(t, `{
  "runners":[{"runnerType":"RUNNER_TYPE_CODEX","available":true,"message":"available and ready","capabilities":{"supportsToolRestriction":true,"toolRestrictionMappings":{"shell":"sandbox-exec"}}}],
  "success":true,"latencyMs":12,"details":{"binary":"codex"}
}`)
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	app := &App{services: NewServices(api)}
	for _, args := range [][]string{{"list"}, {"tools"}, {"probe", "codex"}, {"help"}} {
		if err := app.cmdRunner(args); err != nil {
			t.Fatalf("runner %v: %v", args, err)
		}
	}
	if err := app.cmdRunner([]string{"probe"}); err == nil {
		t.Fatal("probe without runner type succeeded")
	}
	if err := app.runnerTools([]string{"--unexpected"}); err == nil {
		t.Fatal("invalid tools flag succeeded")
	}
}
