package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

// daemon fakes the Ollaya HTTP API with the models it holds. A non-empty
// servedDigest makes pulls store that digest instead of the requested one, the
// way a registry would if the pin were ignored.
func daemon(t *testing.T, installed map[string]string, servedDigest ...string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			var models []map[string]string
			for name, digest := range installed {
				models = append(models, map[string]string{"name": name, "digest": digest})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"models": models})
		case "/api/pull":
			var body struct{ Model string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			name, digest, _ := strings.Cut(body.Model, "@sha256:")
			if len(servedDigest) > 0 {
				digest = servedDigest[0]
			}
			installed[upstream.InstalledName(name)] = digest
			_, _ = w.Write([]byte(`{"status":"success"}`))
		case "/api/decide":
			_, _ = io.Copy(io.Discard, r.Body)
			_, _ = w.Write([]byte(`{"model":"laya:en","answers":{"q":{"type":"noul","noul":0.9}},"usage":{"input_tokens":12,"output_tokens":0},"state_truncated":false,"eval_duration":1000000}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func handlers(t *testing.T, env map[string]string, stdin string) (*Handlers, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	values := map[string]string{"OLLAYA_MODEL_POLICY_PATH": filepath.Join(filepath.Dir(file), "..", "..", "..", "model-policy.json")}
	for key, value := range env {
		values[key] = value
	}
	var stdout, stderr bytes.Buffer
	return &Handlers{GetEnv: func(key string) string { return values[key] }, Stdin: strings.NewReader(stdin), Stdout: &stdout, Stderr: &stderr, HTTP: http.DefaultClient}, &stdout, &stderr
}

func TestRolesReportLocalityAndUnconfiguredHostedRoles(t *testing.T) {
	h, stdout, _ := handlers(t, map[string]string{"EDENAI_API_KEY": "k"}, "")
	if err := h.Roles([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Roles []struct {
			Role, Locality, Status string
			Capabilities           []string
		}
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	status := map[string]string{}
	for _, role := range report.Roles {
		status[role.Role] = role.Locality + "/" + role.Status
		if len(role.Capabilities) != 1 || role.Capabilities[0] != "decide" {
			t.Fatalf("%s capabilities = %v, want [decide]", role.Role, role.Capabilities)
		}
	}
	want := map[string]string{"decide.local": "local/available", "decide.jev.edenai": "remote/available", "decide.jev.typesafe": "remote/unconfigured", "decide.clef.cloudflare": "remote/unconfigured"}
	for role, expected := range want {
		if status[role] != expected {
			t.Fatalf("%s = %s, want %s", role, status[role], expected)
		}
	}
}

func TestResolveFailsTypedWhenTheRoleCannotServe(t *testing.T) {
	h, _, stderr := handlers(t, nil, "")
	if err := h.Resolve([]string{"--role", "decide.jev.typesafe", "--json"}); err == nil {
		t.Fatal("hosted role without its key must not resolve")
	}
	if !strings.Contains(stderr.String(), upstream.ProviderErrorMarker+`{"code":"unconfigured"`) {
		t.Fatalf("stderr = %q, want the unconfigured marker", stderr.String())
	}

	server := daemon(t, map[string]string{"laya:latest": "x"})
	h, _, stderr = handlers(t, map[string]string{"OLLAYA_URL": server.URL}, "")
	if err := h.Resolve([]string{"--role", "decide.local", "--json"}); err == nil || !strings.Contains(stderr.String(), "laya:en is not pulled") {
		t.Fatalf("err=%v stderr=%q, want a router missing its targets to be unconfigured", err, stderr.String())
	}
}

func TestResolveAndDecideLocalRole(t *testing.T) {
	server := daemon(t, map[string]string{"laya:latest": "a", "laya:en": "b", "laya:multilingual": "c"})
	h, stdout, _ := handlers(t, map[string]string{"OLLAYA_URL": server.URL}, "")
	if err := h.Resolve([]string{"--role", "decide.local", "--json"}); err != nil {
		t.Fatal(err)
	}
	var resolved map[string]any
	_ = json.Unmarshal(stdout.Bytes(), &resolved)
	if resolved["role"] != "decide.local" || resolved["model"] != "laya" || resolved["locality"] != "local" {
		t.Fatalf("resolve = %v", resolved)
	}

	h, stdout, _ = handlers(t, map[string]string{"OLLAYA_URL": server.URL}, `{"state":"x","questions":{"q":{"type":"noul"}}}`)
	if err := h.Decide([]string{"--role", "decide.local", "--input-stdin", "--json"}); err != nil {
		t.Fatal(err)
	}
	var result upstream.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Upstream != "local" || result.Model != "laya:en" || *result.Usage.InputTokens != 12 {
		t.Fatalf("decide = %+v", result)
	}
}

func TestPullVerifiesEveryPinnedDigest(t *testing.T) {
	installed := map[string]string{}
	server := daemon(t, installed)
	h, stdout, _ := handlers(t, map[string]string{"OLLAYA_URL": server.URL}, "")
	if err := h.PullModels([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	var report struct{ Pulled []string }
	_ = json.Unmarshal(stdout.Bytes(), &report)
	if strings.Join(report.Pulled, ",") != "laya,laya:en,laya:multilingual" {
		t.Fatalf("pulled = %v, want the router before its targets", report.Pulled)
	}
	drifting := daemon(t, map[string]string{}, "0000000000000000000000000000000000000000000000000000000000000000")
	h, _, _ = handlers(t, map[string]string{"OLLAYA_URL": drifting.URL}, "")
	if err := h.PullModels([]string{"--model", "laya:en"}); err == nil || !strings.Contains(err.Error(), "do not match their pinned digests") {
		t.Fatalf("err = %v, want a pulled digest that differs from the pin to fail", err)
	}
}
