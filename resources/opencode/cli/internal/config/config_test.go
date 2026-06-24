package config

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

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
