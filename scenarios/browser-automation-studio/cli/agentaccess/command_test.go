package agentaccess

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"browser-automation-studio/cli/internal/appctx"
	"github.com/vrooli/cli-core/cliapp"
)

func TestBrowserCommandDiscoversAndQualifiesOnlyAnExistingSession(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		if r.Header.Get("X-Agent-Identity-Token") != "trusted-run-token" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "trusted-grant" {
			t.Errorf("browser context headers missing or wrong: %v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/browser/harnesses":
			_, _ = w.Write([]byte(`[{"id":"bas-typed-flow","version":"1","replay_fidelity":"typed-flow"},{"id":"browser-use","version":"1","pinned_version":"1.0.3","replay_fidelity":"unqualified"}]`))
		case "/api/v1/browser/sessions/session-123/harnesses/browser-use/qualify":
			if r.Method != http.MethodPost {
				t.Errorf("qualification method = %s", r.Method)
			}
			_, _ = w.Write([]byte(`{"adapter_id":"browser-use","outcome":"unsupported","limits":["no same-session borrowed adapter is implemented"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv(identityTokenEnv, "trusted-run-token")
	t.Setenv(grantReceiptEnv, "trusted-grant")
	t.Setenv("BAS_BROWSER_CLI_TEST_CONFIG", t.TempDir())
	app, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name: "bas-browser-cli-test", DefaultAPIBase: server.URL, APIPrefix: "/api/v1", AllowAnonymous: true,
		ConfigDirEnvVars: []string{"BAS_BROWSER_CLI_TEST_CONFIG"},
	})
	if err != nil {
		t.Fatal(err)
	}
	command := Commands(&appctx.Context{Core: app}).Commands[0]

	var listing bytes.Buffer
	rc, err := cliapp.NewTestRunContextFromArgs(command.Args, []string{"harnesses"}, app, &listing, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := command.RunCtx(rc); err != nil {
		t.Fatal(err)
	}
	var adapters []Harness
	if err := json.Unmarshal(listing.Bytes(), &adapters); err != nil {
		t.Fatalf("harness output is not JSON: %v", err)
	}
	if len(adapters) != 2 || adapters[1].PinnedVersion != "1.0.3" {
		t.Fatalf("harness output = %#v", adapters)
	}

	var qualification bytes.Buffer
	rc, err = cliapp.NewTestRunContextFromArgs(command.Args, []string{"qualify", "--session", "session-123", "--adapter", "browser-use"}, app, &qualification, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := command.RunCtx(rc); err != nil {
		t.Fatal(err)
	}
	var result Qualification
	if err := json.Unmarshal(qualification.Bytes(), &result); err != nil {
		t.Fatalf("qualification output is not JSON: %v", err)
	}
	if result.Outcome != "unsupported" || result.AdapterID != "browser-use" {
		t.Fatalf("qualification = %#v", result)
	}
	want := []string{"GET /api/v1/browser/harnesses", "POST /api/v1/browser/sessions/session-123/harnesses/browser-use/qualify"}
	if len(paths) != len(want) {
		t.Fatalf("requests = %#v", paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("request[%d] = %q, want %q", i, paths[i], want[i])
		}
	}
}

func TestBrowserSessionCommandsUseBrokerRoutesAndDoNotSendIdentityInBody(t *testing.T) {
	want := map[string]string{
		"POST /api/v1/browser/sessions":                                           `"start_url":"https://example.test"`,
		"POST /api/v1/browser/sessions/session-123/attach":                        `"permission_ref":"profile-1"`,
		"POST /api/v1/browser/sessions/session-123/observe":                       `"page_id":"main"`,
		"POST /api/v1/browser/sessions/session-123/act":                           `"instruction":{"index":1`,
		"POST /api/v1/browser/sessions/session-123/task":                          `"operation":"click"`,
		"GET /api/v1/browser/sessions/session-123/trace":                          "",
		"POST /api/v1/browser/sessions/session-123/trace/draft":                   `"assertions":[`,
		"POST /api/v1/browser/sessions/session-123/trace/promote":                 `"assertions":[`,
		"POST /api/v1/browser/sessions/session-123/workflows/workflow-123/replay": `"workflow_version":3`,
		"POST /api/v1/browser/sessions/session-123/close":                         "",
	}
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		expected, ok := want[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		seen[key] = true
		if r.Header.Get("X-Agent-Identity-Token") != "trusted-run-token" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "trusted-grant" {
			t.Errorf("trusted context missing on %s", key)
		}
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "trusted-run-token") || strings.Contains(string(body), "trusted-grant") {
			t.Errorf("identity material leaked into body for %s", key)
		}
		if expected != "" && !strings.Contains(string(body), expected) {
			t.Errorf("body for %s = %s, want substring %q", key, body, expected)
		}
		if key == "POST /api/v1/browser/sessions" && strings.Contains(string(body), "permission_ref") {
			t.Errorf("caller-selected permission profile reached BAS: %s", body)
		}
		if key == "POST /api/v1/browser/sessions" && !strings.Contains(string(body), "consent-from-env") {
			t.Errorf("consent receipt was not attached by trusted context")
		}
		if strings.HasSuffix(key, "/attach") && (!strings.Contains(string(body), "consent-from-env") || !strings.Contains(string(body), "approval-from-env")) {
			t.Errorf("attach receipts missing from trusted context")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost || strings.HasSuffix(key, "/close") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if key == "POST /api/v1/browser/sessions" {
			_, _ = w.Write([]byte(`{"session_id":"session-123","permission_ref":"am-derived-profile","identity_profile":{"id":"private-profile"},"outcome":{"success":true}}`))
			return
		}
		_, _ = w.Write([]byte(`{"session_id":"session-123","outcome":{"success":true}}`))
	}))
	t.Cleanup(server.Close)
	t.Setenv(identityTokenEnv, "trusted-run-token")
	t.Setenv(grantReceiptEnv, "trusted-grant")
	t.Setenv(consentReceiptEnv, "consent-from-env")
	t.Setenv(approvalReceiptEnv, "approval-from-env")
	t.Setenv("BAS_BROWSER_CLI_OPS_TEST_CONFIG", t.TempDir())
	app, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{Name: "bas-browser-ops-test", DefaultAPIBase: server.URL, APIPrefix: "/api/v1", AllowAnonymous: true, ConfigDirEnvVars: []string{"BAS_BROWSER_CLI_OPS_TEST_CONFIG"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := &appctx.Context{Core: app}
	dir := t.TempDir()
	write := func(name, value string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	var out bytes.Buffer
	if err := open(ctx, write("open.json", `{"start_url":"https://example.test"}`), &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "am-derived-profile") || strings.Contains(out.String(), "private-profile") {
		t.Fatalf("open output exposed trusted profile details: %s", out.String())
	}
	if err := attach(ctx, "session-123", write("attach.json", `{"permission_ref":"profile-1","tab_id":"tab-1"}`), &out); err != nil {
		t.Fatal(err)
	}
	if err := observe(ctx, "session-123", write("observe.json", `{"page_id":"main"}`), &out); err != nil {
		t.Fatal(err)
	}
	if err := act(ctx, "session-123", write("act.json", `{"instruction":{"index":1,"node_id":"node-1","action":{"type":"ACTION_TYPE_CLICK"}}}`), &out); err != nil {
		t.Fatal(err)
	}
	if err := runTask(ctx, "session-123", write("task.json", `{"operation":"click","selector":"button#go"}`), &out); err != nil {
		t.Fatal(err)
	}
	if err := trace(ctx, "session-123", &out); err != nil {
		t.Fatal(err)
	}
	if err := draftTrace(ctx, "session-123", write("draft-trace.json", `{"name":"preview task","assertions":[{"type":"ACTION_TYPE_ASSERT"}]}`), &out); err != nil {
		t.Fatal(err)
	}
	if err := promote(ctx, "session-123", write("promote.json", `{"project_id":"project-1","name":"saved flow","assertions":[{"type":"ACTION_TYPE_ASSERT"}]}`), &out); err != nil {
		t.Fatal(err)
	}
	if err := replayFlow(ctx, "session-123", write("replay.json", `{"workflow_id":"workflow-123","workflow_version":3,"parameters":{"query":"hello"}}`), &out); err != nil {
		t.Fatal(err)
	}
	if err := closeSession(ctx, "session-123"); err != nil {
		t.Fatal(err)
	}
	for route := range want {
		if !seen[route] {
			t.Errorf("route %q was not called", route)
		}
	}
}

func TestBrowserOpenRejectsCallerSelectedPermissionReference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "open.json")
	if err := os.WriteFile(path, []byte(`{"permission_ref":"caller-choice","start_url":"https://example.test"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := open(nil, path, io.Discard); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("caller-selected permission_ref was accepted: %v", err)
	}
}

func TestDraftTraceRequiresExistingSessionAndTypedAssertions(t *testing.T) {
	if err := draftTrace(nil, "../foreign", "", io.Discard); err == nil {
		t.Fatal("unsafe session ID must be rejected before request parsing")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(path, []byte(`{"name":"preview"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := draftTrace(nil, "session-1", path, io.Discard); err == nil || !strings.Contains(err.Error(), "typed assertion") {
		t.Fatalf("missing assertions error = %v", err)
	}
	path = filepath.Join(dir, "project.json")
	if err := os.WriteFile(path, []byte(`{"project_id":"project-1","name":"preview","assertions":[{}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := draftTrace(nil, "session-1", path, io.Discard); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("caller project authority must not be accepted on a draft: %v", err)
	}
}

func TestBrowserRequestRejectsOversizeAndIdentityFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(path, []byte(`{"permission_ref":"profile-1","grant_receipt":"forbidden"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var input openInput
	if err := readStrictRequest(path, &input); err == nil {
		t.Fatal("identity/grant fields in CLI request file must be rejected")
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), maxRequestBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := readStrictRequest(path, &input); err == nil {
		t.Fatal("oversized CLI request must be rejected")
	}
}

func TestBrowserQualificationRequiresSessionAdapterAndTrustedContext(t *testing.T) {
	t.Setenv(identityTokenEnv, "")
	t.Setenv(grantReceiptEnv, "")
	if err := qualify(nil, "session-1", "browser-use", &bytes.Buffer{}); err == nil {
		t.Fatal("expected missing app context / trusted credentials to fail closed")
	}
	if err := qualify(nil, "../session", "browser-use", &bytes.Buffer{}); err == nil {
		t.Fatal("expected unsafe path segment to be rejected")
	}
}
