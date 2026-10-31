package config

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

// TestMain overrides the OpenRouter cloud-model role resolver so Ensure tests
// run hermetically (no resource-openrouter binary required). The returned slug
// stands in for whatever `code.default` would resolve to in production.
func TestMain(m *testing.M) {
	resolveCloudModel = func(context.Context, string) (string, error) {
		return "deepseek/deepseek-v4-flash", nil
	}
	os.Exit(m.Run())
}

func readFile(path string) ([]byte, error) { return os.ReadFile(path) }

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fptr(f float64) *float64 { return &f }
func iptr(i int) *int         { return &i }

func parse(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, data)
	}
	return m
}

func sampleOllama() *OllamaProvider {
	return &OllamaProvider{
		BaseURL:    "http://localhost:11434/api",
		ChatModel:  "gemma4:12b",
		SmallModel: "gemma4:12b",
		NumCtx:     16384,
		Sampling:   Sampling{Temperature: fptr(0.1), TopP: fptr(0.9), TopK: iptr(40)},
	}
}

func TestRender_FreshConfig(t *testing.T) {
	out, err := Render(nil, Inputs{
		Provider: "openrouter", ChatModel: "deepseek/deepseek-v4-flash", CompletionModel: "deepseek/deepseek-v4-flash",
		Repoint: true, Ollama: sampleOllama(),
	})
	if err != nil {
		t.Fatal(err)
	}
	m := parse(t, out)
	if m["model"] != "openrouter/deepseek/deepseek-v4-flash" {
		t.Errorf("model=%v", m["model"])
	}
	if m["$schema"] != schemaURL {
		t.Errorf("schema=%v", m["$schema"])
	}
	prov := m["provider"].(map[string]any)["ollama"].(map[string]any)
	if prov["npm"] != ollamaNPM || prov["name"] != ollamaName {
		t.Errorf("provider meta wrong: %v", prov)
	}
	entry := prov["models"].(map[string]any)["gemma4:12b"].(map[string]any)
	inner := entry["options"].(map[string]any)["options"].(map[string]any)
	if inner["num_ctx"].(float64) != 16384 {
		t.Errorf("num_ctx=%v", inner["num_ctx"])
	}
	if inner["temperature"].(float64) != 0.1 || inner["top_p"].(float64) != 0.9 || inner["top_k"].(float64) != 40 {
		t.Errorf("sampling not written: %v", inner)
	}
	lim := entry["limit"].(map[string]any)
	if lim["context"].(float64) != 16384 || lim["output"].(float64) != 8192 {
		t.Errorf("limit=%v", lim)
	}
}

func TestRender_PreservesPermissionAndUnknownKeys(t *testing.T) {
	existing := []byte(`{
  "$schema": "https://opencode.ai/config.json",
  "model": "openrouter/deepseek/deepseek-v4-flash",
  "permission": {"bash": {"git push*": "deny", "rm -rf /*": "deny"}},
  "operatorCustomKey": {"keep": true}
}`)
	out, err := Render(existing, Inputs{
		Provider: "openrouter", ChatModel: "deepseek/deepseek-v4-flash", CompletionModel: "deepseek/deepseek-v4-flash",
		Ollama: sampleOllama(),
	})
	if err != nil {
		t.Fatal(err)
	}
	m := parse(t, out)
	perm := m["permission"].(map[string]any)["bash"].(map[string]any)
	if perm["git push*"] != "deny" || perm["rm -rf /*"] != "deny" {
		t.Errorf("permission.bash not preserved: %v", perm)
	}
	if m["operatorCustomKey"].(map[string]any)["keep"] != true {
		t.Errorf("unknown key dropped: %v", m["operatorCustomKey"])
	}
}

func TestRender_Idempotent(t *testing.T) {
	in := Inputs{Provider: "openrouter", ChatModel: "deepseek/deepseek-v4-flash", CompletionModel: "deepseek/deepseek-v4-flash", Repoint: true, Ollama: sampleOllama()}
	first, err := Render(nil, in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Render(first, in)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("not idempotent:\nfirst=%s\nsecond=%s", first, second)
	}
}

func TestRender_DropsRetiredModel(t *testing.T) {
	existing := []byte(`{
  "provider": {"ollama": {"models": {"qwen2.5-coder:7b": {"options":{}}, "keepme:1b": {"options":{}}}}}
}`)
	out, err := Render(existing, Inputs{Ollama: sampleOllama()})
	if err != nil {
		t.Fatal(err)
	}
	m := parse(t, out)
	models := m["provider"].(map[string]any)["ollama"].(map[string]any)["models"].(map[string]any)
	if _, ok := models["qwen2.5-coder:7b"]; ok {
		t.Errorf("retired model not dropped: %v", models)
	}
	if _, ok := models["keepme:1b"]; !ok {
		t.Errorf("unrelated model must be preserved: %v", models)
	}
	if _, ok := models["gemma4:12b"]; !ok {
		t.Errorf("managed model missing: %v", models)
	}
}

func TestRender_MigrateLegacy(t *testing.T) {
	existing := []byte(`{"model":"openrouter/qwen/qwen3-coder","small_model":"openrouter/qwen3-coder"}`)
	out, err := Render(existing, Inputs{
		MigrateLegacy: true,
		LegacyTargets: []string{"openrouter/qwen3-coder", "openrouter/qwen/qwen3-coder"},
		LegacyChat:    "openrouter/deepseek/deepseek-v4-flash",
		LegacySmall:   "openrouter/deepseek/deepseek-v4-flash",
	})
	if err != nil {
		t.Fatal(err)
	}
	m := parse(t, out)
	if m["model"] != "openrouter/deepseek/deepseek-v4-flash" || m["small_model"] != "openrouter/deepseek/deepseek-v4-flash" {
		t.Errorf("legacy not migrated: model=%v small=%v", m["model"], m["small_model"])
	}
}

func TestRender_MergeDoesNotClobberExistingModel(t *testing.T) {
	existing := []byte(`{"model":"openrouter/operator-pick","small_model":"openrouter/operator-pick"}`)
	out, err := Render(existing, Inputs{Provider: "openrouter", ChatModel: "deepseek/deepseek-v4-flash", CompletionModel: "deepseek/deepseek-v4-flash"})
	if err != nil {
		t.Fatal(err)
	}
	m := parse(t, out)
	if m["model"] != "openrouter/operator-pick" {
		t.Errorf("merge clobbered operator model: %v", m["model"])
	}
}

func TestRender_RepointForcesModel(t *testing.T) {
	existing := []byte(`{"model":"openrouter/deepseek/deepseek-v4-flash"}`)
	out, err := Render(existing, Inputs{Provider: "ollama", ChatModel: "gemma4:12b", CompletionModel: "gemma4:12b", Repoint: true})
	if err != nil {
		t.Fatal(err)
	}
	if parse(t, out)["model"] != "ollama/gemma4:12b" {
		t.Errorf("repoint failed: %v", parse(t, out)["model"])
	}
}

func TestRender_SamplingOmittedWhenAbsent(t *testing.T) {
	op := sampleOllama()
	op.Sampling = Sampling{} // none pinned
	out, err := Render(nil, Inputs{Provider: "openrouter", ChatModel: "x", CompletionModel: "x", Ollama: op})
	if err != nil {
		t.Fatal(err)
	}
	inner := parse(t, out)["provider"].(map[string]any)["ollama"].(map[string]any)["models"].(map[string]any)["gemma4:12b"].(map[string]any)["options"].(map[string]any)["options"].(map[string]any)
	if _, ok := inner["temperature"]; ok {
		t.Errorf("temperature must be omitted when not pinned: %v", inner)
	}
	if inner["num_ctx"].(float64) != 16384 {
		t.Errorf("num_ctx still required: %v", inner)
	}
}

// --- Ensure orchestration (fake Resolver) -------------------------------------

type fakeResolver struct {
	installed []string
	listErr   error
	role      RoleResolution
	roleErr   error
}

func (f fakeResolver) InstalledModels(ctx context.Context) ([]string, error) {
	return f.installed, f.listErr
}

func (f fakeResolver) LocalRole(ctx context.Context, role string) (RoleResolution, error) {
	return f.role, f.roleErr
}

func TestEnsure_CloudKeyKeepsOpenRouterButWritesOllamaBlock(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/opencode.json"
	changed, err := Ensure(context.Background(), EnsureOptions{
		ConfigPath:     path,
		Defaults:       DefaultDefaults(func(string) string { return "" }),
		HaveOpenRouter: true,
		Resolver: fakeResolver{
			installed: []string{"gemma4:12b"},
			role:      RoleResolution{Model: "gemma4:12b", Sampling: Sampling{Temperature: fptr(0.1)}},
		},
	})
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	data, _ := readFile(path)
	m := parse(t, data)
	if m["model"] != "openrouter/deepseek/deepseek-v4-flash" {
		t.Errorf("active model should stay cloud: %v", m["model"])
	}
	if _, ok := m["provider"].(map[string]any)["ollama"]; !ok {
		t.Errorf("ollama provider block should be written when reachable")
	}
}

func TestEnsure_NoKeyLocalReachableSelfHealsToOllama(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/opencode.json"
	// Seed an existing cloud config so the repoint path is exercised.
	writeFile(t, path, `{"model":"openrouter/deepseek/deepseek-v4-flash"}`)
	_, err := Ensure(context.Background(), EnsureOptions{
		ConfigPath:     path,
		Defaults:       DefaultDefaults(func(string) string { return "" }),
		HaveOpenRouter: false,
		Resolver:       fakeResolver{installed: []string{"gemma4:12b"}, role: RoleResolution{Model: "gemma4:12b"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := readFile(path)
	if parse(t, data)["model"] != "ollama/gemma4:12b" {
		t.Errorf("expected self-heal to ollama, got %v", parse(t, data)["model"])
	}
}

func TestEnsure_OllamaUnreachableNoBlockOnFresh(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/opencode.json"
	_, err := Ensure(context.Background(), EnsureOptions{
		ConfigPath:     path,
		Defaults:       DefaultDefaults(func(string) string { return "" }),
		HaveOpenRouter: true,
		Resolver:       fakeResolver{listErr: errors.New("connection refused")},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := readFile(path)
	m := parse(t, data)
	if _, ok := m["provider"]; ok {
		t.Errorf("no provider block expected when Ollama unreachable on a fresh config: %v", m["provider"])
	}
}

func TestEnsure_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/opencode.json"
	opts := EnsureOptions{
		ConfigPath:     path,
		Defaults:       DefaultDefaults(func(string) string { return "" }),
		HaveOpenRouter: true,
		Resolver:       fakeResolver{installed: []string{"gemma4:12b"}, role: RoleResolution{Model: "gemma4:12b", Sampling: Sampling{Temperature: fptr(0.1)}}},
	}
	if _, err := Ensure(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	changed, err := Ensure(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Errorf("second Ensure must be a no-op (idempotent)")
	}
}

func TestEnsure_CloudRoleResolutionFailsLoud(t *testing.T) {
	prev := resolveCloudModel
	resolveCloudModel = func(context.Context, string) (string, error) {
		return "", errors.New("resource-openrouter not found")
	}
	defer func() { resolveCloudModel = prev }()

	dir := t.TempDir()
	_, err := Ensure(context.Background(), EnsureOptions{
		ConfigPath:     dir + "/opencode.json",
		Defaults:       DefaultDefaults(func(string) string { return "" }),
		HaveOpenRouter: true,
		Resolver:       fakeResolver{installed: []string{"gemma4:12b"}},
	})
	if err == nil {
		t.Fatal("expected Ensure to fail when the cloud role cannot be resolved (no concrete fallback)")
	}
}

func TestEnsure_ResolvesCloudModelFromRole(t *testing.T) {
	prev := resolveCloudModel
	var gotRole string
	resolveCloudModel = func(_ context.Context, role string) (string, error) {
		gotRole = role
		return "vendor/some-resolved-model", nil
	}
	defer func() { resolveCloudModel = prev }()

	dir := t.TempDir()
	path := dir + "/opencode.json"
	if _, err := Ensure(context.Background(), EnsureOptions{
		ConfigPath:     path,
		Defaults:       DefaultDefaults(func(string) string { return "" }),
		HaveOpenRouter: true,
		Resolver:       fakeResolver{installed: []string{"gemma4:12b"}},
	}); err != nil {
		t.Fatal(err)
	}
	if gotRole != "code.default" {
		t.Errorf("expected role code.default, got %q", gotRole)
	}
	data, _ := readFile(path)
	if parse(t, data)["model"] != "openrouter/vendor/some-resolved-model" {
		t.Errorf("active model should be the resolved cloud slug: %v", parse(t, data)["model"])
	}
}

func TestOllamaBaseURL(t *testing.T) {
	cases := map[string]string{
		"":                    "http://localhost:11434",
		"myhost":              "http://myhost:11434",
		"myhost:9999":         "http://myhost:9999",
		"https://gpu.box:443": "https://gpu.box:443",
		"http://10.0.0.5":     "http://10.0.0.5:11434",
	}
	for in, want := range cases {
		got := ollamaBaseURL(func(k string) string {
			if k == "OLLAMA_HOST" {
				return in
			}
			return ""
		})
		if got != want {
			t.Errorf("ollamaBaseURL(%q)=%q want %q", in, got, want)
		}
	}
}

func TestRender_ProjectsExtraOllamaModels(t *testing.T) {
	op := sampleOllama()
	op.ExtraModels = []LocalModel{{
		Model:     "chat:9b",
		NumCtx:    8192,
		MaxOutput: 1024,
		Sampling:  Sampling{Temperature: fptr(0.7), TopP: fptr(0.95)},
	}}
	out, err := Render(nil, Inputs{Provider: "openrouter", ChatModel: "x", CompletionModel: "x", Ollama: op})
	if err != nil {
		t.Fatal(err)
	}
	models := parse(t, out)["provider"].(map[string]any)["ollama"].(map[string]any)["models"].(map[string]any)
	entry, ok := models["chat:9b"].(map[string]any)
	if !ok {
		t.Fatalf("extra model missing from provider block: %v", models)
	}
	inner := entry["options"].(map[string]any)["options"].(map[string]any)
	if inner["num_ctx"].(float64) != 8192 || inner["temperature"].(float64) != 0.7 || inner["top_p"].(float64) != 0.95 {
		t.Errorf("extra model options = %v", inner)
	}
	limit := entry["limit"].(map[string]any)
	if limit["context"].(float64) != 8192 || limit["output"].(float64) != 1024 {
		t.Errorf("extra model limit = %v", limit)
	}
	if _, ok := models["gemma4:12b"]; !ok {
		t.Error("default local model must still be present")
	}
}

func TestRender_ExtraModelPreservesOperatorName(t *testing.T) {
	existing := []byte(`{"provider":{"ollama":{"models":{"chat:9b":{"name":"Operator Name","custom":true}}}}}`)
	op := sampleOllama()
	op.ExtraModels = []LocalModel{{Model: "chat:9b"}}
	out, err := Render(existing, Inputs{Provider: "openrouter", ChatModel: "x", CompletionModel: "x", Ollama: op})
	if err != nil {
		t.Fatal(err)
	}
	entry := parse(t, out)["provider"].(map[string]any)["ollama"].(map[string]any)["models"].(map[string]any)["chat:9b"].(map[string]any)
	if entry["name"] != "Operator Name" || entry["custom"] != true {
		t.Errorf("unmanaged keys not preserved: %v", entry)
	}
}

func TestParseLocalModelsFiltersAndRanks(t *testing.T) {
	raw := []byte(`{"roles":[
	  {"role":"chat.default","model":"model-a","preference":30,"capabilities":["generate","chat","summarize"],"context_window_tokens":32768,"sampling":{"temperature":0.7,"top_p":0.95,"has_temperature":true,"has_top_p":true}},
	  {"role":"summarize.default","model":"model-a","preference":40,"capabilities":["generate","chat"],"context_window_tokens":32768,"sampling":{"temperature":0.3,"has_temperature":true}},
	  {"role":"embedding.default","model":"model-b","preference":10,"capabilities":["embedding"],"context_window_tokens":8192},
	  {"role":"chat.missing","model":"model-c","preference":20,"capabilities":["chat"],"context_window_tokens":32768},
	  {"role":"chat.uncensored","model":"model-d","preference":80,"capabilities":["generate","chat"],"context_window_tokens":32768,"max_tokens":2048,"sampling":{"temperature":0.7,"has_temperature":true}}
	]}`)
	got, err := parseLocalModels(raw, []string{"model-a", "model-b", "model-d"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Model != "model-a" || got[1].Model != "model-d" {
		t.Fatalf("models = %+v, want model-a then model-d", got)
	}
	if got[0].Sampling.Temperature == nil || *got[0].Sampling.Temperature != 0.7 {
		t.Errorf("model-a should inherit chat.default sampling, got %+v", got[0].Sampling)
	}
	if got[0].NumCtx != 32768 || got[1].NumCtx != 32768 {
		t.Errorf("projected context = %d/%d, want 32768", got[0].NumCtx, got[1].NumCtx)
	}
	if got[1].MaxOutput != 2048 {
		t.Errorf("model-d max output = %d, want 2048", got[1].MaxOutput)
	}
}

func TestRender_CapsProjectedContext(t *testing.T) {
	op := sampleOllama()
	op.ExtraModels = []LocalModel{{Model: "chat:9b", NumCtx: 262144}}
	out, err := Render(nil, Inputs{Provider: "openrouter", ChatModel: "x", CompletionModel: "x", Ollama: op})
	if err != nil {
		t.Fatal(err)
	}
	entry := parse(t, out)["provider"].(map[string]any)["ollama"].(map[string]any)["models"].(map[string]any)["chat:9b"].(map[string]any)
	if entry["limit"].(map[string]any)["context"].(float64) != maxProjectedContext {
		t.Errorf("projected context not capped: %v", entry["limit"])
	}
}

type listingResolver struct {
	fakeResolver
	extras []LocalModel
}

func (r listingResolver) LocalModels(context.Context, []string) ([]LocalModel, error) {
	return r.extras, nil
}

func TestEnsure_ProjectsPolicyModelsIntoProviderBlock(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/opencode.json"
	_, err := Ensure(context.Background(), EnsureOptions{
		ConfigPath:     path,
		Defaults:       DefaultDefaults(func(string) string { return "" }),
		HaveOpenRouter: true,
		Resolver: listingResolver{
			fakeResolver: fakeResolver{installed: []string{"gemma4:12b", "chat:9b"}, role: RoleResolution{Model: "gemma4:12b"}},
			extras:       []LocalModel{{Model: "chat:9b", MaxOutput: 2048, Sampling: Sampling{Temperature: fptr(0.7)}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	models := parse(t, mustRead(t, path))["provider"].(map[string]any)["ollama"].(map[string]any)["models"].(map[string]any)
	if _, ok := models["chat:9b"]; !ok {
		t.Fatalf("policy model not projected into provider block: %v", models)
	}
}

func TestEnsure_ProjectionFailureIsNonFatal(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/opencode.json"
	_, err := Ensure(context.Background(), EnsureOptions{
		ConfigPath:     path,
		Defaults:       DefaultDefaults(func(string) string { return "" }),
		HaveOpenRouter: true,
		Resolver:       failingLister{fakeResolver: fakeResolver{installed: []string{"gemma4:12b"}, role: RoleResolution{Model: "gemma4:12b"}}},
	})
	if err != nil {
		t.Fatalf("projection failure must not fail ensure: %v", err)
	}
	if _, ok := parse(t, mustRead(t, path))["provider"].(map[string]any)["ollama"]; !ok {
		t.Error("ollama block should still be written")
	}
}

type failingLister struct{ fakeResolver }

func (f failingLister) LocalModels(context.Context, []string) ([]LocalModel, error) {
	return nil, errors.New("policy unavailable")
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := readFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestEnsure_CloudProviderKeepsLocalModelInOllamaBlock(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/opencode.json"
	_, err := Ensure(context.Background(), EnsureOptions{
		ConfigPath:     path,
		Defaults:       DefaultDefaults(func(string) string { return "" }),
		HaveOpenRouter: true,
		Resolver:       fakeResolver{installed: []string{"gemma4:12b"}, role: RoleResolution{Model: "gemma4:12b"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m := parse(t, mustRead(t, path))
	if m["model"] != "openrouter/deepseek/deepseek-v4-flash" {
		t.Errorf("cloud provider should stay active, got %v", m["model"])
	}
	models := m["provider"].(map[string]any)["ollama"].(map[string]any)["models"].(map[string]any)
	if _, ok := models["gemma4:12b"]; !ok {
		t.Errorf("local model missing from ollama block: %v", models)
	}
	if _, ok := models["deepseek-v4.1-flash"]; ok {
		t.Errorf("cloud model must not be advertised as a local Ollama model: %v", models)
	}
}

func TestEnsure_OpenCodeGoKeepsLocalModelInOllamaBlock(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/opencode.json"
	_, err := Ensure(context.Background(), EnsureOptions{
		ConfigPath:     path,
		Defaults:       DefaultDefaults(func(string) string { return "" }),
		HaveOpenCodeGo: true,
		Resolver:       fakeResolver{installed: []string{"gemma4:12b"}, role: RoleResolution{Model: "gemma4:12b"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m := parse(t, mustRead(t, path))
	if m["model"] != "opencode-go/deepseek-v4.1-flash" {
		t.Errorf("OpenCode Go should stay active, got %v", m["model"])
	}
	models := m["provider"].(map[string]any)["ollama"].(map[string]any)["models"].(map[string]any)
	if _, ok := models["gemma4:12b"]; !ok {
		t.Errorf("local model missing from ollama block: %v", models)
	}
}

func TestRender_OpenAICompatibleProviderBlocks(t *testing.T) {
	out, err := Render(nil, Inputs{
		Provider: "orcarouter", ChatModel: "openai/gpt-4o-mini", CompletionModel: "openai/gpt-4o-mini", Repoint: true,
		OpenAIProviders: []OpenAIProvider{
			{ID: "orcarouter", Name: "OrcaRouter", NPM: "@ai-sdk/openai-compatible", BaseURL: "https://api.orcarouter.ai/v1"},
			{ID: "abliteration", Name: "Abliteration", NPM: "@ai-sdk/openai-compatible", BaseURL: "https://api.abliteration.ai/v1"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	m := parse(t, out)
	providers := m["provider"].(map[string]any)
	for _, id := range []string{"orcarouter", "abliteration"} {
		block := providers[id].(map[string]any)
		if block["npm"] != "@ai-sdk/openai-compatible" {
			t.Errorf("%s npm = %v", id, block["npm"])
		}
		if block["options"].(map[string]any)["baseURL"] == "" {
			t.Errorf("%s baseURL missing", id)
		}
	}
	if m["model"] != "orcarouter/openai/gpt-4o-mini" {
		t.Errorf("model = %v", m["model"])
	}
}
