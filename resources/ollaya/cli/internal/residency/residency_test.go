package residency

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/activityedge"
)

func shippedPolicyPath(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "model-policy.json")
}

func decideRequest(t *testing.T, path, body string) *http.Request {
	t.Helper()
	return httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
}

func rewritten(t *testing.T, edge *Edge, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	if _, err := edge.Rewrite(r); err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if r.ContentLength != int64(len(raw)) {
		t.Fatalf("content length %d, body %d bytes", r.ContentLength, len(raw))
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatalf("rewritten body is not JSON: %v (%s)", err, raw)
	}
	return top
}

// The shipped policy reaches live traffic: a digest-pinned decision for a
// model with no keep_alive of its own carries the idle unload, a large
// bake-off model keeps its own shorter one, and the caller's questions and
// state reach the daemon unchanged.
func TestEdgeStampsTheShippedPolicyOnEveryDecision(t *testing.T) {
	live := &Live{Path: shippedPolicyPath(t)}
	edge := &Edge{KeepAlive: live.KeepAliveSeconds}
	questions := `{"z":{"type":"noul"},"a":{"type":"choice","criteria":{"<b>":"","&":""}}}`
	top := rewritten(t, edge, decideRequest(t, "/api/decide", `{"model":"coba:1b@sha256:`+strings.Repeat("a", 64)+`","state":"x < y","questions":`+questions+`,"keep_alive":"30m"}`))
	if string(top["keep_alive"]) != "300" {
		t.Fatalf("keep_alive = %s, want the policy's 300 s idle unload over the caller's 30m", top["keep_alive"])
	}
	if string(top["questions"]) != questions || string(top["state"]) != `"x < y"` {
		t.Fatalf("questions=%s state=%s; the caller's values must reach the daemon unchanged", top["questions"], top["state"])
	}
	top = rewritten(t, edge, decideRequest(t, "/api/decide", `{"model":"kev:4b","state":"x","questions":{}}`))
	if string(top["keep_alive"]) != "120" {
		t.Fatalf("kev:4b keep_alive = %s, want its own placement.keep_alive (2m)", top["keep_alive"])
	}
}

func TestEdgeLeavesUnloadsOtherRoutesAndUnreadableBodiesAlone(t *testing.T) {
	edge := &Edge{KeepAlive: func(string) (int, bool) { return 300, true }}
	for _, unload := range []string{`0`, `"0"`, `"0s"`} {
		top := rewritten(t, edge, decideRequest(t, "/api/decide", `{"model":"laya","keep_alive":`+unload+`}`))
		if string(top["keep_alive"]) != unload {
			t.Fatalf("explicit unload %s became %s; an unload must stay an unload", unload, top["keep_alive"])
		}
	}
	for _, request := range []*http.Request{
		decideRequest(t, "/api/show", `{"model":"laya"}`),
		httptest.NewRequest(http.MethodGet, "/api/ps", nil),
		decideRequest(t, "/api/decide", `not json`),
		decideRequest(t, "/api/decide", `{"state":"no model"}`),
	} {
		before := ""
		if request.Body != nil {
			raw, _ := io.ReadAll(request.Body)
			before = string(raw)
			request.Body = io.NopCloser(strings.NewReader(before))
		}
		if _, err := edge.Rewrite(request); err != nil {
			t.Fatal(err)
		}
		if request.Body == nil {
			continue
		}
		after, _ := io.ReadAll(request.Body)
		if string(after) != before {
			t.Fatalf("%s %s body changed: %q -> %q", request.Method, request.URL.Path, before, after)
		}
	}
	unknown := &Edge{KeepAlive: func(string) (int, bool) { return 0, false }}
	top := rewritten(t, unknown, decideRequest(t, "/api/decide", `{"model":"laya","state":"x","questions":{}}`))
	if _, present := top["keep_alive"]; present {
		t.Fatalf("keep_alive = %s, want the daemon default when the policy decides nothing", top["keep_alive"])
	}
}

func TestEdgeForwardsAnOversizedBodyUnread(t *testing.T) {
	edge := &Edge{KeepAlive: func(string) (int, bool) { return 300, true }}
	body := `{"model":"laya","state":"` + strings.Repeat("x", maxInspectedBody) + `"}`
	request := decideRequest(t, "/api/decide", body)
	if _, err := edge.Rewrite(request); err != nil {
		t.Fatal(err)
	}
	after, _ := io.ReadAll(request.Body)
	if string(after) != body {
		t.Fatalf("oversized body changed (%d -> %d bytes); it must pass through for the daemon to refuse", len(body), len(after))
	}
}

// Startup reconciliation puts models loaded before the edge started onto the
// policy schedule and never loads anything.
func TestReconcileLoadedTouchesOnlyResidentModels(t *testing.T) {
	var mu sync.Mutex
	var posted []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ps":
			_, _ = w.Write([]byte(`{"models":[{"name":"coba:1b","device":"cuda:0"},{"name":"kev:4b"},{"model":"coba:1b"}]}`))
		case "/api/decide":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			mu.Lock()
			posted = append(posted, body)
			mu.Unlock()
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	edge := &Edge{KeepAlive: (&Live{Path: shippedPolicyPath(t)}).KeepAliveSeconds}
	if err := edge.ReconcileLoaded(context.Background(), activityedge.Upstream{BaseURL: server.URL, Client: server.Client()}); err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{"coba:1b": 300, "kev:4b": 120}
	if len(posted) != len(want) {
		t.Fatalf("posted %v, want one keep_alive request per loaded model", posted)
	}
	for _, body := range posted {
		model, _ := body["model"].(string)
		if body["keep_alive"] != want[model] || body["state"] != nil || body["questions"] != nil {
			t.Fatalf("reconcile body %v, want a stateless keep_alive %v for %s", body, want[model], model)
		}
	}
}

// A live edge picks up a policy edit without a restart and keeps the last
// valid policy when an edit does not parse.
func TestLivePolicyReloadsEditsAndSurvivesABadOne(t *testing.T) {
	shipped, err := os.ReadFile(shippedPolicyPath(t))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "model-policy.json")
	if err := os.WriteFile(path, shipped, 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	live := &Live{Path: path, Now: func() time.Time { return now }}
	if got, _ := live.KeepAliveSeconds("coba:1b"); got != 300 {
		t.Fatalf("initial keep_alive = %d", got)
	}
	edited := strings.Replace(string(shipped), `"idle_unload_seconds": 300`, `"idle_unload_seconds": 1800`, 1)
	if err := os.WriteFile(path, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	now = now.Add(liveCheckInterval)
	if got, _ := live.KeepAliveSeconds("coba:1b"); got != 1800 {
		t.Fatalf("keep_alive after edit = %d, want 1800", got)
	}
	if err := os.WriteFile(path, []byte(`{"half":`), 0o600); err != nil {
		t.Fatal(err)
	}
	now = now.Add(liveCheckInterval)
	if got, ok := live.KeepAliveSeconds("coba:1b"); !ok || got != 1800 {
		t.Fatalf("keep_alive after a bad edit = %d, %v; want the last valid policy", got, ok)
	}
}
