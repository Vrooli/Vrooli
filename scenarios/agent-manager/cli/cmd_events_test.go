package main

import (
	"testing"

	clitest "agent-manager/cli/internal/testutil"

	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
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

func TestPermissionPolicyCommandsRenderHealthyOperationalResponses(t *testing.T) {
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		switch request.Path {
		case "/api/v1/permission-policy/catalog":
			return `{"status":{"ready":true,"path":"policy.json","activeDigest":"active"},"catalog":{"metadata":{"catalogId":"permissions"},"schemaVersion":1}}`
		case "/api/v1/permission-policy/validate":
			return `{"valid":true,"candidateDigest":"candidate","activeDigest":"active"}`
		case "/api/v1/permission-policy/reload":
			return `{"activated":true,"status":{"ready":true,"path":"policy.json","activeDigest":"active"}}`
		case "/api/v1/permission-policy/reconcile":
			return `{"result":{"success":true}}`
		case "/api/v1/permission-policy/doctor":
			return `{"healthy":true,"summary":"all enforced","status":{"ready":true,"path":"policy.json","activeDigest":"active"},"plan":{"catalogDigest":"active","hardEnforcementSatisfied":true}}`
		default:
			return `{}`
		}
	})
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	app := &App{services: NewServices(api)}
	for _, args := range [][]string{{"catalog"}, {"validate"}, {"reload"}, {"reconcile", "--i-was-explicitly-authorized"}, {"doctor"}} {
		if err := app.cmdPermissionPolicy(args); err != nil {
			t.Fatalf("permission-policy %v: %v", args, err)
		}
	}
}

func TestRolePolicyCommandsRenderHealthyOperationalResponses(t *testing.T) {
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		switch request.Path {
		case "/api/v1/role-policy/validate":
			return `{"valid":true,"candidateDigest":"candidate","activeDigest":"active"}`
		case "/api/v1/role-policy/reload":
			return `{"activated":true,"status":{"ready":true,"path":"roles.json","activeDigest":"active"}}`
		case "/api/v1/role-policy/explain":
			return `{"targetType":"profile","targetId":"profile-1","summary":"resolved","snapshot":{"catalogDigest":"active","roleRef":"code.default","selectedIndex":0,"candidates":[{"runnerType":"RUNNER_TYPE_CODEX","selectionType":"MODEL_SELECTION_TYPE_EXPLICIT","model":"gpt-test"}]}}`
		case "/api/v1/role-policy/catalog":
			return `{"status":{"ready":true,"path":"roles.json","activeDigest":"active"},"catalog":{"metadata":{"catalogId":"roles"},"schemaVersion":1,"defaultRole":"code.default","roles":[{"roleRef":"code.default","intent":"implement","description":"make safe code changes","candidates":[{"runnerType":"RUNNER_TYPE_CODEX","resourceRole":"coding"}]}]}}`
		default:
			return `{}`
		}
	})
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	app := &App{services: NewServices(api)}
	for _, args := range [][]string{{"validate"}, {"reload"}, {"catalog"}, {"explain", "profile", "profile-1"}} {
		if err := app.cmdPolicy(args); err != nil {
			t.Fatalf("role-policy %v: %v", args, err)
		}
	}
}

func TestPermissionPolicyPlanReportsDriftAndHardEnforcementGaps(t *testing.T) {
	printPermissionPolicyPlan(&apipb.PermissionPolicyPlan{
		CatalogDigest: "digest", HardEnforcementSatisfied: false,
		MissingHardEnforcementRuleIds: []string{"rule-1"},
		Resources: []*apipb.PermissionPolicyResourceResult{{
			RunnerType: domainpb.RunnerType_RUNNER_TYPE_CODEX, Scope: "workspace", Status: "drifted", Drift: true, Error: "mapping missing",
		}},
	})
}

func TestPermissionPolicyStatusReportsOptionalCatalogAndReloadDiagnostic(t *testing.T) {
	printPermissionPolicyStatus(&apipb.PermissionPolicyStatus{
		Ready: false, Path: "policy.json", ActiveDigest: "",
		LastReloadAttempt: &apipb.PermissionPolicyReloadAttempt{Diagnostic: &apipb.PermissionPolicyDiagnostic{Code: "INVALID", Message: "missing rule"}},
	})
}

func TestRolePolicyDiagnosticReportsDistinctCause(t *testing.T) {
	printPolicyDiagnostic(&apipb.RolePolicyDiagnostic{Code: "INVALID", Message: "validation failed", Cause: "missing resource role"})
}

func TestRunnerProbeRendersFailureEvidence(t *testing.T) {
	server := clitest.NewRecordingServer(t, `{"result":{"success":false,"latencyMs":42,"error":"authentication expired","details":{"binary":"codex"}}}`)
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	if err := (&App{services: NewServices(api)}).cmdRunner([]string{"probe", "codex"}); err != nil {
		t.Fatal(err)
	}
}

func TestRunnerListRendersUnavailableLongDiagnostic(t *testing.T) {
	server := clitest.NewRecordingServer(t, `{"runners":[{"runnerType":"RUNNER_TYPE_OPENCODE","available":false,"message":"this deliberately long diagnostic demonstrates that unavailable runner output remains bounded"}]}`)
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	if err := (&App{services: NewServices(api)}).cmdRunner([]string{"list"}); err != nil {
		t.Fatal(err)
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
