package codecs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeOpenCodeCatalog writes a minimal opencode-style models.json cache to a
// temp file and returns its path. The shape mirrors ~/.cache/opencode/models.json:
// a top-level map of provider -> { "models": { "<id>": {...} } }.
func writeOpenCodeCatalog(t *testing.T) string {
	t.Helper()
	const catalog = `{
      "openrouter": {
        "models": {
          "deepseek/deepseek-v4-pro": {},
          "google/gemini-3.5-flash": {},
          "anthropic/claude-opus-4.8": {}
        }
      },
      "anthropic": {
        "models": {
          "claude-opus-4-8": {}
        }
      }
    }`
	dir := t.TempDir()
	path := filepath.Join(dir, "models.json")
	if err := os.WriteFile(path, []byte(catalog), 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
	return path
}

func TestValidateOpenCodeModel(t *testing.T) {
	catalog := writeOpenCodeCatalog(t)
	pulled := []string{"ollama/gemma4:12b", "ollama/qwen3.5:4b"}

	tests := []struct {
		name       string
		model      string
		ollama     []string
		catalog    string
		wantErr    bool
		errSnippet string
	}{
		{name: "empty model accepted (cli default)", model: "", catalog: catalog},
		{name: "cloud model present in catalog", model: "openrouter/deepseek/deepseek-v4-pro", catalog: catalog},
		{name: "cloud high tier present", model: "openrouter/anthropic/claude-opus-4.8", catalog: catalog},
		{
			name:       "cloud model absent from known provider rejected",
			model:      "openrouter/x-ai/grok-code-fast-1",
			catalog:    catalog,
			wantErr:    true,
			errSnippet: "not available from provider",
		},
		{name: "unknown provider degrades to accept", model: "vercel/deepseek/deepseek-v4-pro", catalog: catalog},
		{name: "missing catalog file degrades to accept", model: "openrouter/x-ai/grok-code-fast-1", catalog: filepath.Join(t.TempDir(), "nope.json")},
		{name: "ollama pulled model accepted", model: "ollama/gemma4:12b", ollama: pulled, catalog: catalog},
		{
			name:       "ollama model not pulled rejected",
			model:      "ollama/qwen3.6:27b",
			ollama:     pulled,
			catalog:    catalog,
			wantErr:    true,
			errSnippet: "not pulled locally",
		},
		{name: "ollama empty list degrades to accept", model: "ollama/qwen3.6:27b", ollama: nil, catalog: catalog},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOpenCodeModel(tc.model, tc.ollama, tc.catalog)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for model %q, got nil", tc.model)
				}
				if tc.errSnippet != "" && !strings.Contains(err.Error(), tc.errSnippet) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.errSnippet)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for model %q: %v", tc.model, err)
			}
		})
	}
}
