package deletiongrant

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

type flagContext struct {
	cliapp.OperationContext
	flags  map[string]string
	values map[string][]string
}

func (c flagContext) Flag(name string) string         { return c.flags[name] }
func (c flagContext) FlagValues(name string) []string { return c.values[name] }

func TestRequestSendsEveryPathAndTheSessionHandle(t *testing.T) {
	t.Setenv(webConsoleSessionEnv, "wc-1")
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != requestPath {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"pending","decision_id":"d1","paths":["/a","/b"],"await":{"park_command":"agent-manager run park <run-id> --producer operator-decision --key d1 --timeout 12h"}}`))
	}))
	defer server.Close()
	h := &handlers{httpClient: server.Client(), baseURL: server.URL}
	got, err := h.requestCall(flagContext{flags: map[string]string{"reason": "orphaned copy"}, values: map[string][]string{"path": {"/b", "/a"}}})
	if err != nil {
		t.Fatal(err)
	}
	if body["reason"] != "orphaned copy" || body["web_console_session_id"] != "wc-1" || len(body["paths"].([]any)) != 2 {
		t.Fatalf("body = %+v", body)
	}
	report := requestReport(nil, got)
	if !strings.Contains(report.Result[0], "d1") || len(report.Changes) != 2 || len(report.NextCommand) != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestRequestRequiresPathAndReason(t *testing.T) {
	h := &handlers{httpClient: http.DefaultClient, baseURL: "http://127.0.0.1:0"}
	if _, err := h.requestCall(flagContext{flags: map[string]string{"reason": "r"}}); err == nil {
		t.Fatal("request without --path succeeded")
	}
	if _, err := h.requestCall(flagContext{values: map[string][]string{"path": {"/a"}}}); err == nil {
		t.Fatal("request without --reason succeeded")
	}
}

func TestRequestSurfacesTheRefusal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "/d is the root x declares protected", http.StatusBadRequest)
	}))
	defer server.Close()
	h := &handlers{httpClient: server.Client(), baseURL: server.URL}
	_, err := h.requestCall(flagContext{flags: map[string]string{"reason": "r"}, values: map[string][]string{"path": {"/d"}}})
	if err == nil || !strings.Contains(err.Error(), "declares protected") {
		t.Fatalf("err = %v, want the API refusal", err)
	}
}

func TestRequestReadsPathsFileWithInventoryColumns(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"pending","decision_id":"d"}`))
	}))
	defer srv.Close()
	list := filepath.Join(t.TempDir(), "paths.txt")
	if err := os.WriteFile(list, []byte("# fixtures\n/home/u/a\t24K\n\n/home/u/b \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := &handlers{httpClient: srv.Client(), baseURL: srv.URL}
	ctx := flagContext{flags: map[string]string{"reason": "r", "paths-file": list}, values: map[string][]string{"path": {"/home/u/c"}}}
	if _, err := h.requestCall(ctx); err != nil {
		t.Fatal(err)
	}
	paths, _ := got["paths"].([]any)
	if len(paths) != 3 || paths[0] != "/home/u/c" || paths[1] != "/home/u/a" || paths[2] != "/home/u/b" {
		t.Fatalf("sent paths %v, want --path then the file's paths without columns or comments", paths)
	}
}
