package profiles

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestSaveUsesCurrentProfile(t *testing.T) {
	var captured []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profiles/demo":
			io.WriteString(w, `{"id":"demo","tiers":[2],"swaps":{"a":"b"},"secrets":{},"settings":{"env":{"X":"Y"}}}`)
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/profiles/demo":
			data, _ := io.ReadAll(r.Body)
			captured = data
			w.Write(data)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))
	if err := cmd.Save([]string{"demo"}); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if !bytes.Contains(captured, []byte(`"swaps":{"a":"b"}`)) {
		t.Fatalf("expected swap to persist, got %s", string(captured))
	}
}

func TestDiffRequiresHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/profiles/demo/versions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		io.WriteString(w, `{"profile_id":"demo","versions":[{"version":1}]}`)
	}))
	defer server.Close()
	cmd := New(testAPIClient(server.URL))
	err := cmd.Diff([]string{"demo"})
	if err == nil || !strings.Contains(err.Error(), "nothing to diff") {
		t.Fatalf("expected diff error, got %v", err)
	}
}

func TestRollbackUsesSelectedVersion(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/profiles/demo/versions":
			io.WriteString(w, `{"profile_id":"demo","versions":[{"version":2,"tiers":[3],"swaps":{},"secrets":{},"settings":{}}]}`)
		case r.URL.Path == "/api/v1/profiles/demo" && r.Method == http.MethodPut:
			data, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(data, &payload)
			w.Write(data)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))
	if err := cmd.Rollback([]string{"demo", "--version", "2"}); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	if payload == nil || payload["tiers"] == nil {
		t.Fatalf("expected rollback payload to include tiers, got %v", payload)
	}
}

func TestAnalyzeRequiresProfileID(t *testing.T) {
	cmd := New(nil)
	if err := cmd.Analyze([]string{}); err == nil || !strings.Contains(err.Error(), "profile id is required") {
		t.Fatalf("expected profile id required error, got %v", err)
	}
}

func TestSecretsIdentifyCallsEndpoint(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		io.WriteString(w, `{}`)
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))
	if err := cmd.Secrets([]string{"identify", "demo"}); err != nil {
		t.Fatalf("secrets identify failed: %v", err)
	}
	if path != "/api/v1/profiles/demo/secrets" {
		t.Fatalf("expected secrets identify path, got %s", path)
	}
}

func TestSecretsTemplatePrintsEnvTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"template":"FOO=bar\nBAZ=qux\n"}`)
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	origStdout := os.Stdout
	t.Cleanup(func() { os.Stdout = origStdout })
	os.Stdout = w

	if err := cmd.Secrets([]string{"template", "demo", "--format", "env"}); err != nil {
		t.Fatalf("secrets template failed: %v", err)
	}
	_ = w.Close()
	out, _ := io.ReadAll(r)
	if !strings.Contains(string(out), "FOO=bar") || !strings.Contains(string(out), "BAZ=qux") {
		t.Fatalf("expected env template output, got %s", string(out))
	}
}

func TestSecretsValidatePosts(t *testing.T) {
	var method string
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		io.WriteString(w, `{}`)
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))
	if err := cmd.Secrets([]string{"validate", "demo"}); err != nil {
		t.Fatalf("secrets validate failed: %v", err)
	}
	if method != http.MethodPost || path != "/api/v1/profiles/demo/secrets/validate" {
		t.Fatalf("expected POST to validate endpoint, got %s %s", method, path)
	}
}

func testAPIClient(base string) *cliutil.APIClient {
	return cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: cliutil.APIBaseOptions{DefaultBase: base}}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: base} },
		func() string { return "" },
	)
}
