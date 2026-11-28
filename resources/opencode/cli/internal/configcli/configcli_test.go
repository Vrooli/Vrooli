package configcli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/opencode/cli/internal/config"
)

func TestProviderModelsReadsChatModelsFromTheOwningResource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell fixture")
	}
	dir := t.TempDir()
	script := `#!/bin/sh
cat <<'JSON'
{"models":[
  {"model":"chat-a","display_name":"Chat A","endpoints":["chat"],"capabilities":["chat"]},
  {"model":"embed","endpoints":["embeddings"],"capabilities":["embeddings"]},
  {"model":"chat-b","capabilities":["chat"]},
  {"model":"chat-a","endpoints":["chat"]},
  {"model":" "}
]}
JSON
`
	path := filepath.Join(dir, "resource-testprovider")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	models, err := providerModels(context.Background(), "testprovider")
	if err != nil {
		t.Fatal(err)
	}
	if want := []config.ProviderModel{{ID: "chat-a", Name: "Chat A"}, {ID: "chat-b"}}; !slices.Equal(models, want) {
		t.Fatalf("models = %v, want %v", models, want)
	}
}

func TestProviderModelsReportsAMissingCatalog(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := providerModels(context.Background(), "missingprovider"); err == nil {
		t.Fatal("expected a missing provider CLI to be reported")
	}
}

func TestSelectLocalModelRequiresProjectionAndPreservesConfigMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")
	content := `{"model":"openrouter/paid","permission":{"bash":{"git push*":"deny"}},"provider":{"ollama":{"models":{"huihui_ai/granite4.1-abliterated:8b":{"name":"Granite"}}}}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	h := &Handlers{GetEnv: func(key string) string {
		if key == "OPENCODE_CONFIG_DIR" {
			return dir
		}
		return ""
	}, Stdout: &output}
	if err := h.SelectLocalModel("huihui_ai/granite4.1-abliterated:8b"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	want := "ollama/huihui_ai/granite4.1-abliterated:8b"
	if doc["model"] != want || doc["small_model"] != want {
		t.Fatalf("active model/small_model = %v/%v, want %q", doc["model"], doc["small_model"], want)
	}
	if doc["permission"].(map[string]any)["bash"].(map[string]any)["git push*"] != "deny" {
		t.Fatalf("permission config changed: %v", doc["permission"])
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config mode = %o, want 600", got)
	}
	if err := h.SelectLocalModel("not-projected:latest"); err == nil {
		t.Fatal("unprojected model must not be selected")
	}
}

func TestSelectLocalModelDoesNotWriteAfterCancellation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")
	content := `{"model":"openrouter/paid","provider":{"ollama":{"models":{"qwen3.5:9b":{"name":"Qwen"}}}}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h := &Handlers{Stdout: &strings.Builder{}, GetEnv: func(key string) string {
		if key == "OPENCODE_CONFIG_DIR" {
			return dir
		}
		return ""
	}}
	if err := h.SelectLocalModelContext(ctx, "qwen3.5:9b"); !errors.Is(err, context.Canceled) {
		t.Fatalf("SelectLocalModelContext error = %v, want context cancellation", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != content {
		t.Fatalf("config after cancellation = %s err=%v, want unchanged", data, err)
	}
}

func TestSelectLocalModelPreservesConfigSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires filesystem symlinks")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "managed-opencode.json")
	path := filepath.Join(dir, "opencode.json")
	content := `{"model":"openrouter/paid","provider":{"ollama":{"models":{"qwen3.5:9b":{"name":"Qwen"}}}}}`
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	h := &Handlers{Stdout: &strings.Builder{}, GetEnv: func(key string) string {
		if key == "OPENCODE_CONFIG_DIR" {
			return dir
		}
		return ""
	}}
	if err := h.SelectLocalModel("qwen3.5:9b"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config link info = %v err=%v, symlink was replaced", info, err)
	}
	link, err := os.Readlink(path)
	if err != nil || link != target {
		t.Fatalf("config link target = %q err=%v, want %q", link, err, target)
	}
	data, err := os.ReadFile(target)
	if err != nil || !strings.Contains(string(data), `"model": "ollama/qwen3.5:9b"`) {
		t.Fatalf("config target = %s err=%v, model selection did not reach target", data, err)
	}
}

func TestPrepareLocalModelCommitsProjectionAndSelectionTogether(t *testing.T) {
	dir := t.TempDir()
	installFakeOllamaCLI(t, dir)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	configDir := t.TempDir()
	path := filepath.Join(configDir, "opencode.json")
	content := `{"model":"openrouter/paid","permission":{"bash":{"git push*":"deny"}}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	h := &Handlers{GetEnv: func(key string) string {
		if key == "OPENCODE_CONFIG_DIR" {
			return configDir
		}
		return ""
	}, Stdout: &strings.Builder{}}
	if err := h.PrepareLocalModelContext(context.Background(), "target:8b"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["model"] != "ollama/target:8b" || doc["small_model"] != "ollama/target:8b" {
		t.Fatalf("selected models = %v/%v", doc["model"], doc["small_model"])
	}
	models := doc["provider"].(map[string]any)["ollama"].(map[string]any)["models"].(map[string]any)
	if _, ok := models["target:8b"]; !ok {
		t.Fatal("selected model was not projected into the picker")
	}
	if doc["permission"].(map[string]any)["bash"].(map[string]any)["git push*"] != "deny" {
		t.Fatal("existing OpenCode permissions were not preserved")
	}
}

func TestPrepareLocalModelProjectionFailureDoesNotRepointActiveModel(t *testing.T) {
	dir := t.TempDir()
	installFakeOllamaCLI(t, dir)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	configDir := t.TempDir()
	path := filepath.Join(configDir, "opencode.json")
	content := `{"model":"openrouter/paid","small_model":"openrouter/cheap"}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	h := &Handlers{GetEnv: func(key string) string {
		if key == "OPENCODE_CONFIG_DIR" {
			return configDir
		}
		return ""
	}, Stdout: &strings.Builder{}}
	err := h.PrepareLocalModelContext(context.Background(), "not-projected:8b")
	if err == nil || !strings.Contains(err.Error(), "not in the prepared OpenCode catalog") {
		t.Fatalf("PrepareLocalModelContext error = %v, want unprojected-model refusal", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != content {
		t.Fatalf("config after failed projection = %s err=%v, want unchanged", data, err)
	}
}

func installFakeOllamaCLI(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "resource-ollama")
	script := `#!/bin/sh
case "$1:$2" in
  models:list) printf '%s\n' '{"models":["target:8b"]}' ;;
  policy:resolve) printf '%s\n' '{"model":"target:8b"}' ;;
  policy:models) printf '%s\n' '{"models":[{"model":"target:8b"}]}' ;;
  policy:roles) printf '%s\n' '{"roles":[{"role":"code.local","model":"target:8b","capabilities":["chat","generate"],"context_window_tokens":16384,"rights_decision":{"status":"allowed","use":"local_inference"}}]}' ;;
  *) exit 2 ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}
