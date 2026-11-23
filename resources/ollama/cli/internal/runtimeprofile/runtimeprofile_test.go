package runtimeprofile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/activityedge"
)

// TestResourceManifestAppliesTheProfileToTheDaemon binds the daemon
// environment in resource.json to profile.json. Ollama reads these at serve
// time, so every runner receives the declared context and CPU thread budget
// without any caller sending an option.
func TestResourceManifestAppliesTheProfileToTheDaemon(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		ManagedService struct {
			Environment map[string]string `json:"environment"`
		} `json:"managed_service"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	want, err := Current().ServerEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range want {
		if got := manifest.ManagedService.Environment[name]; got != value {
			t.Errorf("resource.json managed_service.environment.%s = %q, want %q from runtimeprofile/profile.json", name, got, value)
		}
	}
}

func TestProfileDeclaresEveryOllamaRunnerOption(t *testing.T) {
	// Ollama's api.Runner fields: a change in any of them restarts the runner.
	for _, key := range []string{"num_ctx", "num_batch", "num_gpu", "main_gpu", "use_mmap", "num_thread"} {
		if !Current().IsLoadOption(key) {
			t.Errorf("profile request_load_options is missing runner option %s", key)
		}
	}
	if Current().IsLoadOption("temperature") || Current().IsLoadOption("num_predict") {
		t.Fatal("sampling options must stay caller-controlled")
	}
}

func TestDecodeRejectsVariantsThatSetNonLoadOptionsAndUnexplainedModels(t *testing.T) {
	base := map[string]any{}
	if err := json.Unmarshal(profileJSON, &base); err != nil {
		t.Fatal(err)
	}
	variant := cloneJSON(t, base)
	variant["variants"] = map[string]any{"fast": map[string]any{"temperature": 0}}
	if _, err := Decode(mustJSON(t, variant)); err == nil {
		t.Fatal("variant setting a sampling option was accepted")
	}
	model := cloneJSON(t, base)
	model["models"] = map[string]any{"m:latest": map[string]any{"context_tokens": 4096}}
	if _, err := Decode(mustJSON(t, model)); err == nil {
		t.Fatal("model override without a reason was accepted")
	}
	untagged := cloneJSON(t, base)
	untagged["models"] = map[string]any{"m": map[string]any{"context_tokens": 4096, "reason": "x"}}
	if _, err := Decode(mustJSON(t, untagged)); err == nil {
		t.Fatal("untagged model reference was accepted")
	}
}

func TestContextTokensAppliesModelOverridesAndImplicitLatestTag(t *testing.T) {
	p := Current()
	if got := p.ContextTokens("nomic-embed-text"); got != p.Models["nomic-embed-text:latest"].ContextTokens {
		t.Fatalf("nomic-embed-text context = %d", got)
	}
	if got := p.ContextTokens("llama3.2:1b"); got != p.Server.ContextTokens {
		t.Fatalf("undeclared model context = %d, want server %d", got, p.Server.ContextTokens)
	}
}

func TestRequestOptionsSendOnlyContextsAboveTheServerDefault(t *testing.T) {
	p := Current()
	for ref, model := range p.Models {
		got := p.RequestOptions(ref)
		if model.ContextTokens > p.Server.ContextTokens {
			if got["num_ctx"] != model.ContextTokens || len(got) != 1 {
				t.Errorf("%s request options = %v, want num_ctx %d", ref, got, model.ContextTokens)
			}
		} else if len(got) != 0 {
			t.Errorf("%s request options = %v, want none (the daemon applies its own limit)", ref, got)
		}
	}
	if got := p.RequestOptions("llama3.2:1b"); len(got) != 0 {
		t.Fatalf("undeclared model request options = %v", got)
	}
}

func TestEdgeNormalizerStripsLoadOptionsAndKeepsEverythingElse(t *testing.T) {
	var logged []string
	n := NewEdgeNormalizer(func(format string, args ...any) { logged = append(logged, format) })
	declared := Current().ContextTokens("qwen3.5:9b")
	body := `{"model":"qwen3.5:9b","messages":[{"role":"user","content":"hi"}],"stream":true,"options":{"num_ctx":` + strconv.Itoa(declared) + `,"num_thread":4,"num_gpu":99,"temperature":0.7,"top_p":0.95,"seed":12345678901234567}}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	if _, err := n.Rewrite(req); err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(req.Body)
	var got map[string]json.RawMessage
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("normalized body is not JSON: %s", raw)
	}
	var options map[string]json.Number
	decoder := json.NewDecoder(bytes.NewReader(got["options"]))
	decoder.UseNumber()
	if err := decoder.Decode(&options); err != nil {
		t.Fatal(err)
	}
	if options["num_ctx"] != json.Number(strconv.Itoa(declared)) {
		t.Fatalf("num_ctx = %v, want the declared %d: %s", options["num_ctx"], declared, raw)
	}
	if _, ok := options["num_thread"]; ok {
		t.Fatalf("num_thread survived: %s", raw)
	}
	if _, ok := options["num_gpu"]; ok {
		t.Fatalf("num_gpu survived: %s", raw)
	}
	if options["temperature"] != "0.7" || options["top_p"] != "0.95" || options["seed"] != "12345678901234567" {
		t.Fatalf("sampling options changed: %v", options)
	}
	if string(got["stream"]) != "true" || !strings.Contains(string(got["messages"]), `"content":"hi"`) {
		t.Fatalf("non-option fields changed: %s", raw)
	}
	if req.ContentLength != int64(len(raw)) {
		t.Fatalf("content length %d, body %d", req.ContentLength, len(raw))
	}
	if len(logged) != 1 {
		t.Fatalf("expected one normalization log line, got %d", len(logged))
	}
}

func TestEdgeNormalizerRateLimitsItsLog(t *testing.T) {
	var lines []string
	clock := time.Date(2026, 11, 21, 2, 0, 0, 0, time.UTC)
	n := NewEdgeNormalizer(func(format string, args ...any) { lines = append(lines, format) })
	n.Now = func() time.Time { return clock }
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(`{"model":"m","options":{"num_gpu":99}}`))
		if _, err := n.Rewrite(req); err != nil {
			t.Fatal(err)
		}
	}
	clock = clock.Add(edgeLogInterval)
	req := httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(`{"model":"m","options":{"num_gpu":99}}`))
	if _, err := n.Rewrite(req); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("log lines = %d, want first sighting plus one interval summary", len(lines))
	}
}

func TestEdgeNormalizerLeavesUnaffectedRequestsByteIdentical(t *testing.T) {
	n := NewEdgeNormalizer(nil)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/embed", `{"model":"nomic-embed-text","input":["a"],  "options":{"temperature":0}}`},
		{http.MethodPost, "/api/generate", `not json`},
		{http.MethodPost, "/api/show", `{"model":"m","options":{"num_ctx":1}}`},
		{http.MethodPost, "/v1/chat/completions", `{"model":"m"}`},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		if _, err := n.Rewrite(req); err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(req.Body)
		if string(raw) != tc.body {
			t.Fatalf("%s %s body changed: %q", tc.method, tc.path, raw)
		}
	}
}

func TestEdgeNormalizerPassesOversizedBodiesThroughIntact(t *testing.T) {
	n := NewEdgeNormalizer(nil)
	payload := bytes.Repeat([]byte("x"), maxNormalizedBody+10)
	req := httptest.NewRequest(http.MethodPost, "/api/generate", bytes.NewReader(payload))
	if _, err := n.Rewrite(req); err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(req.Body)
	if !bytes.Equal(raw, payload) {
		t.Fatalf("oversized body length %d, want %d unchanged", len(raw), len(payload))
	}
}

// TestStatusDetectsPlantedDrift is the canary for the runtime signal: a log
// with one model started under two different thread settings, a runner that
// ignored the CPU budget and an oversized embedding must read as drift, and the
// same log with consistent starts must read ok.
func TestStatusDetectsPlantedDrift(t *testing.T) {
	dir := t.TempDir()
	models := filepath.Join(dir, "models")
	writeManifest(t, models, "registry.ollama.ai/library/qwen3.5/4b", "aaaa1111")
	blob := filepath.Join(models, "blobs", "sha256-aaaa1111")
	start := func(at, extra string) string {
		return `time=` + at + ` level=INFO source=llama_server.go:406 msg="starting llama-server" cmd="/x/llama-server --model ` + blob + ` --host 127.0.0.1 -c 8192 -np 1 --flash-attn on -b 1024 -ub 1024` + extra + ` --context-shift"`
	}
	drifted := strings.Join([]string{
		start("2026-11-21T01:00:00-05:00", " -t 4"),
		"system_info: n_threads = 4 (n_threads_batch = 4) / 32",
		start("2026-11-21T01:01:00-05:00", ""),
		"system_info: n_threads = 16 (n_threads_batch = 16) / 32",
		`time=2026-11-21T01:01:30-05:00 level=INFO source=sched.go:557 msg="predicted to exceed available memory, evicting"`,
		"srv    send_error: task id = 1, error: input (2233 tokens) is too large to process. increase the physical batch size (current batch size: 2048)",
	}, "\n") + "\n"
	logPath := filepath.Join(dir, "service.log")
	if err := os.WriteFile(logPath, []byte(drifted), 0o600); err != nil {
		t.Fatal(err)
	}
	windowStart, _ := time.Parse(time.RFC3339, "2026-11-21T00:30:00-05:00")
	windowEnd, _ := time.Parse(time.RFC3339, "2026-11-21T02:00:00-05:00")
	report, err := ScanLogs(Current(), []string{logPath}, "", windowStart, windowEnd)
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "drift" || report.OptionDriftRestarts != 1 || report.ThreadBudgetMisses != 1 || report.Evictions != 1 || report.EmbedInputTooLarge != 1 || report.RunnerStarts != 2 {
		t.Fatalf("planted drift not detected: %+v", report)
	}
	if len(report.Runners) != 1 || report.Runners[0].Model != "qwen3.5:4b" {
		t.Fatalf("runner attribution = %+v", report.Runners)
	}

	consistent := strings.Join([]string{
		start("2026-11-21T01:00:00-05:00", ""),
		"system_info: n_threads = 4 (n_threads_batch = 4) / 32",
		start("2026-11-21T01:10:00-05:00", ""),
		"system_info: n_threads = 4 (n_threads_batch = 4) / 32",
	}, "\n") + "\n"
	if err := os.WriteFile(logPath, []byte(consistent), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err = ScanLogs(Current(), []string{logPath}, "", windowStart, windowEnd)
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "ok" || report.RunnerStarts != 2 || report.OptionDriftRestarts != 0 {
		t.Fatalf("consistent starts misread: %+v", report)
	}
}

func TestStatusIgnoresEventsOutsideTheWindow(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "service.log")
	content := `time=2026-11-20T01:00:00-05:00 level=INFO msg="x"` + "\nsrv    send_error: input (3000 tokens) is too large to process\n"
	if err := os.WriteFile(logPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	windowStart, _ := time.Parse(time.RFC3339, "2026-11-21T00:00:00-05:00")
	report, err := ScanLogs(Current(), []string{logPath, filepath.Join(dir, "missing.log")}, "", windowStart, windowStart.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if report.EmbedInputTooLarge != 0 || report.Verdict != "ok" {
		t.Fatalf("out-of-window event counted: %+v", report)
	}
}

// TestOllamaResourceCodeSetsLoadOptionsOnlyThroughTheProfile is the
// conformance check for the resource's own code: no non-test Go file outside
// this package may name a runner load option or a per-request runner flag.
// Scenario and other resource callers are covered by the AI Gateway
// conformance rule AI_OLLAMA_RUNTIME_OPTIONS.
func TestOllamaResourceCodeSetsLoadOptionsOnlyThroughTheProfile(t *testing.T) {
	pattern := loadOptionLiteralPattern(Current())
	root := filepath.Join("..", "..")
	var findings []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "runtimeprofile" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(raw), "\n") {
			if pattern.MatchString(line) {
				findings = append(findings, path+":"+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) > 0 {
		t.Fatalf("runner load options set outside runtimeprofile (callers would restart shared runners):\n%s", strings.Join(findings, "\n"))
	}
}

func loadOptionLiteralPattern(p Profile) *regexp.Regexp {
	var alternatives []string
	for _, key := range p.RequestLoadOptions {
		alternatives = append(alternatives, `"`+key+`"`, `"`+strings.ReplaceAll(key, "_", "-")+`"`, `--`+strings.ReplaceAll(key, "_", "-")+`\b`)
	}
	return regexp.MustCompile(strings.Join(alternatives, "|"))
}

func writeManifest(t *testing.T, models, name, digest string) {
	t.Helper()
	path := filepath.Join(models, "manifests", filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"layers":[{"mediaType":"application/vnd.ollama.image.model","digest":"sha256:` + digest + `"}]}`
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
}

func cloneJSON(t *testing.T, in map[string]any) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(mustJSON(t, in), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestEdgeNormalizerAppliesDeclaredContextAndDropsUndeclaredOnes(t *testing.T) {
	n := NewEdgeNormalizer(nil)
	declared := Current().ContextTokens("qwen3.5:4b")
	for _, tc := range []struct{ body, want string }{
		{`{"model":"qwen3.5:4b","prompt":"x"}`, `"num_ctx":` + strconv.Itoa(declared)},
		{`{"model":"qwen3.5:4b","prompt":"x","options":{"num_ctx":8192}}`, `"num_ctx":` + strconv.Itoa(declared)},
		{`{"model":"llama3.2:1b","prompt":"x","options":{"num_ctx":32768}}`, ""},
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(tc.body))
		if _, err := n.Rewrite(req); err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(req.Body)
		if tc.want == "" {
			if strings.Contains(string(raw), "num_ctx") {
				t.Fatalf("undeclared model kept a caller context: %s", raw)
			}
			continue
		}
		if !strings.Contains(string(raw), tc.want) {
			t.Fatalf("body %s lacks %s", raw, tc.want)
		}
	}
}

func TestEdgeNormalizerAppliesPolicyKeepAlive(t *testing.T) {
	n := NewEdgeNormalizer(nil)
	n.KeepAlive = func(model string) (int, bool) {
		switch model {
		case "qwen3.5:4b":
			return -1, true
		case "unmanaged:1b":
			return 0, false
		}
		return 600, true
	}
	for _, tc := range []struct {
		name, path, body, want string
	}{
		{"idle model gets the policy dwell", "/api/chat", `{"model":"qwen3.5:9b","messages":[]}`, "600"},
		{"caller pin is replaced", "/api/generate", `{"model":"qwen3.5:9b","prompt":"x","keep_alive":-1}`, "600"},
		{"caller duration is replaced", "/api/generate", `{"model":"qwen3.5:9b","prompt":"x","keep_alive":"24h"}`, "600"},
		{"resident model stays loaded", "/api/embed", `{"model":"qwen3.5:4b","input":["a"],"keep_alive":"5m"}`, "-1"},
		{"explicit numeric unload is honored", "/api/generate", `{"model":"qwen3.5:9b","keep_alive":0}`, "0"},
		{"explicit duration unload is honored", "/api/generate", `{"model":"qwen3.5:9b","keep_alive":"0s"}`, `"0s"`},
		{"policy without an answer leaves the caller alone", "/api/chat", `{"model":"unmanaged:1b","keep_alive":"5m"}`, `"5m"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			if _, err := n.Rewrite(req); err != nil {
				t.Fatal(err)
			}
			raw, _ := io.ReadAll(req.Body)
			var got map[string]json.RawMessage
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("body is not JSON: %s", raw)
			}
			if string(got["keep_alive"]) != tc.want {
				t.Fatalf("keep_alive = %s, want %s (body %s)", got["keep_alive"], tc.want, raw)
			}
		})
	}
}

func TestEdgeNormalizerKeepAliveMatchLeavesBodyByteIdentical(t *testing.T) {
	n := NewEdgeNormalizer(nil)
	n.KeepAlive = func(string) (int, bool) { return 600, true }
	body := `{"model":"m",  "prompt":"x","keep_alive":600}`
	req := httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(body))
	if _, err := n.Rewrite(req); err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(req.Body)
	if string(raw) != body {
		t.Fatalf("matching body changed: %q", raw)
	}
}

// fakeDaemon serves the two endpoints residency reassertion uses and records
// every load-only request it receives.
type fakeDaemon struct {
	loaded       []string
	capabilities map[string][]string
	requests     chan map[string]any
}

func (d *fakeDaemon) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/ps":
		models := make([]map[string]string, 0, len(d.loaded))
		for _, name := range d.loaded {
			models = append(models, map[string]string{"name": name, "model": name})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"models": models})
	case "/api/show":
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{"capabilities": d.capabilities[body.Model]})
	case "/api/generate", "/api/embed":
		body := map[string]any{"endpoint": r.URL.Path}
		_ = json.NewDecoder(r.Body).Decode(&body)
		d.requests <- body
		_, _ = io.WriteString(w, `{"done":true}`)
	default:
		http.NotFound(w, r)
	}
}

func TestReconcileLoadedResidencyRepairsOnlyLoadedManagedModels(t *testing.T) {
	daemon := &fakeDaemon{
		loaded:       []string{"gemma4:12b", "nomic-embed-text:latest", "unmanaged:1b"},
		capabilities: map[string][]string{"nomic-embed-text:latest": {"embedding"}},
		requests:     make(chan map[string]any, 4),
	}
	server := httptest.NewServer(daemon)
	defer server.Close()
	n := NewEdgeNormalizer(nil)
	n.KeepAlive = func(model string) (int, bool) {
		switch model {
		case "gemma4:12b":
			return 600, true
		case "nomic-embed-text:latest":
			return -1, true
		default:
			return 0, false
		}
	}
	upstream := activityedge.Upstream{BaseURL: server.URL, Client: server.Client()}
	if err := n.ReconcileLoadedResidency(context.Background(), upstream); err != nil {
		t.Fatalf("ReconcileLoadedResidency() error = %v", err)
	}
	want := map[string]map[string]any{
		"gemma4:12b":              {"endpoint": "/api/generate", "model": "gemma4:12b", "keep_alive": float64(600)},
		"nomic-embed-text:latest": {"endpoint": "/api/embed", "model": "nomic-embed-text:latest", "keep_alive": float64(-1), "input": []any{}},
	}
	for range want {
		got := <-daemon.requests
		model, _ := got["model"].(string)
		if !reflect.DeepEqual(got, want[model]) {
			t.Errorf("reconciliation request = %v, want %v", got, want[model])
		}
	}
	select {
	case extra := <-daemon.requests:
		t.Fatalf("reconciled an unmanaged or absent model: %v", extra)
	default:
	}
}

func TestEdgeNormalizerReassertsResidencyAfterCompatibleRequests(t *testing.T) {
	daemon := &fakeDaemon{loaded: []string{"qwen3.5:9b", "nomic-embed-text:latest"}, requests: make(chan map[string]any, 8)}
	server := httptest.NewServer(daemon)
	defer server.Close()
	upstream := activityedge.Upstream{BaseURL: server.URL, Client: server.Client()}

	n := NewEdgeNormalizer(nil)
	n.ReassertDelay = 20 * time.Millisecond
	n.KeepAlive = func(model string) (int, bool) {
		switch model {
		case "nomic-embed-text":
			return -1, true
		case "unmanaged:1b":
			return 0, false
		}
		return 600, true
	}
	body := `{"model":"qwen3.5:9b","messages":[{"role":"user","content":"hi"}],"keep_alive":120}`
	// A burst of three compatible requests yields one reassertion.
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
		complete, err := n.Rewrite(req)
		if err != nil || complete == nil {
			t.Fatalf("compatible request: completion=%v err=%v, want a residency follow-up", complete != nil, err)
		}
		if raw, _ := io.ReadAll(req.Body); string(raw) != body {
			t.Fatalf("compatible body changed: %s", raw)
		}
		complete(upstream)
	}
	select {
	case got := <-daemon.requests:
		if got["endpoint"] != "/api/generate" || got["model"] != "qwen3.5:9b" || got["keep_alive"] != float64(600) || got["prompt"] != nil || got["messages"] != nil {
			t.Fatalf("reassertion = %v, want a load-only generate with keep_alive 600", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no residency reassertion after the burst")
	}
	select {
	case extra := <-daemon.requests:
		t.Fatalf("burst produced a second reassertion: %v", extra)
	case <-time.After(100 * time.Millisecond):
	}

	// An untagged resident model is matched against /api/ps by canonical name,
	// and an embeddings request reasserts through an empty /api/embed because
	// embedding models refuse /api/generate.
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", strings.NewReader(`{"model":"nomic-embed-text","input":"x"}`))
	complete, err := n.Rewrite(req)
	if err != nil || complete == nil {
		t.Fatalf("embeddings: completion=%v err=%v", complete != nil, err)
	}
	complete(upstream)
	select {
	case got := <-daemon.requests:
		input, isList := got["input"].([]any)
		if got["endpoint"] != "/api/embed" || got["keep_alive"] != float64(-1) || !isList || len(input) != 0 {
			t.Fatalf("resident reassertion = %v, want an empty /api/embed with keep_alive -1", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no reassertion for the resident model")
	}

	// No follow-up when the policy has no answer or the body names no model.
	for _, body := range []string{`{"model":"unmanaged:1b"}`, `{"messages":[]}`, `not json`} {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
		if complete, err := n.Rewrite(req); err != nil || complete != nil {
			t.Fatalf("body %s: completion=%v err=%v, want none", body, complete != nil, err)
		}
	}
	// Native requests carry keep_alive in the body and need no follow-up.
	native := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"model":"qwen3.5:9b","messages":[]}`))
	if complete, err := n.Rewrite(native); err != nil || complete != nil {
		t.Fatalf("native request: completion=%v err=%v, want none", complete != nil, err)
	}
}

func TestEdgeNormalizerNeverReloadsAnUnloadedModel(t *testing.T) {
	daemon := &fakeDaemon{loaded: []string{"nomic-embed-text:latest"}, requests: make(chan map[string]any, 1)}
	server := httptest.NewServer(daemon)
	defer server.Close()
	n := NewEdgeNormalizer(nil)
	n.KeepAlive = func(string) (int, bool) { return 600, true }
	if err := n.reassertResidency("qwen3.5:9b", defaultLoadOnly, activityedge.Upstream{BaseURL: server.URL, Client: server.Client()}); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-daemon.requests:
		t.Fatalf("reasserted an unloaded model: %v", got)
	default:
	}
}

// TestPolicyContextWindowsNeverExceedTheRunner keeps one answer to "how much
// context does this model get": the runner context here. The policy's
// context_window_tokens, which the gateway admits requests against, may only
// narrow it (a model's trained limit); a larger window would admit prompts the
// runner silently truncates.
func TestPolicyContextWindowsNeverExceedTheRunner(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var policy struct {
		Models map[string]struct {
			ContextWindowTokens int `json:"context_window_tokens"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	for model, declared := range policy.Models {
		if runner := Current().ContextTokens(model); declared.ContextWindowTokens > runner {
			t.Errorf("model-policy.json %s context_window_tokens %d exceeds its runner context %d from runtimeprofile/profile.json", model, declared.ContextWindowTokens, runner)
		}
	}
}

// Ollama serves one model under every spelling it accepts, so each spelling
// must get that model's runner options; a miss would load a second runner at
// the server default context.
func TestEverySpellingOllamaAcceptsGetsTheModelsRunnerOptions(t *testing.T) {
	want := fmt.Sprint(Current().RequestOptions("qwen3.5:9b"))
	for _, ref := range []string{"Qwen3.5:9B", "registry.ollama.ai/library/qwen3.5:9b", "library/qwen3.5:9b"} {
		if got := fmt.Sprint(Current().RequestOptions(ref)); got != want {
			t.Errorf("RequestOptions(%q) = %s, want %s", ref, got, want)
		}
	}
}
