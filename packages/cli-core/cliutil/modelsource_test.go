package cliutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/agentcatalog/harness"
)

func TestModelSourceOwnSubscriptionLeavesLaunchUntouched(t *testing.T) {
	args := []string{"resume", "thread"}
	env := []string{"CODEX_HOME=/existing", "OPENCODE_CONFIG_CONTENT={\"mine\":true}"}
	for _, runner := range []string{"codex", "opencode", "claude-code", "grok"} {
		gotArgs, gotEnv, err := ResolveModelSource(runner, ModelSource{Kind: "subscription"}, args, env)
		if err != nil || !reflect.DeepEqual(gotArgs, args) || !reflect.DeepEqual(gotEnv, env) {
			t.Fatalf("%s own account changed: %v %v %v", runner, gotArgs, gotEnv, err)
		}
	}
}

func TestModelSourceCredentialOnlyInChildEnvironment(t *testing.T) {
	source := ModelSource{ID: "Vrooli credits", Kind: "vrooli_credits", Model: "coding", BaseURL: "https://example.invalid/api/v1/ai", SecretEnv: "VROOLI_SESSION_KEY"}
	for _, runner := range []string{"codex", "opencode", "grok", "claude-code", "antigravity"} {
		args, env, err := ResolveModelSource(runner, source, []string{"run"}, []string{"VROOLI_SESSION_KEY=session-secret"})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.Join(args, " "), "session-secret") || strings.Contains(environmentValue(env, "OPENCODE_CONFIG_CONTENT"), "session-secret") {
			t.Fatal("credential entered command arguments or configuration")
		}
		if environmentValue(env, "VROOLI_SESSION_KEY") != "session-secret" {
			t.Fatal("credential was not supplied to child")
		}
		if runner == "opencode" {
			var config map[string]any
			if err := json.Unmarshal([]byte(environmentValue(env, "OPENCODE_CONFIG_CONTENT")), &config); err != nil || config["model"] != "vrooli-source/coding" {
				t.Fatal("invalid per-session model configuration", err)
			}
		}
		if runner == "grok" && (environmentValue(env, "GROK_MODELS_BASE_URL") != source.BaseURL || environmentValue(env, "XAI_API_KEY") != "xai-session-secret") {
			t.Fatalf("Grok did not receive scoped source: %#v", env)
		}
		if runner == "grok" {
			if strings.Contains(environmentValue(env, "GROK_CONFIG"), "session-secret") || strings.Contains(environmentValue(env, "GROK_CONFIG"), "env_key") {
				t.Fatal("Grok config overlay must not carry the credential value or unsupported model table")
			}
		}
		if runner == "claude-code" && (environmentValue(env, "ANTHROPIC_BASE_URL") != source.BaseURL || environmentValue(env, "ANTHROPIC_API_KEY") != "session-secret") {
			t.Fatal("Claude Code did not receive the scoped source through per-process environment")
		}
		if runner == "antigravity" && (environmentValue(env, "GOOGLE_GEMINI_BASE_URL") != source.BaseURL || environmentValue(env, "GEMINI_API_KEY") != "session-secret" || !strings.Contains(strings.Join(args, " "), "gemini-api://example.invalid/models/coding")) {
			t.Fatal("Antigravity did not receive its scoped native endpoint, model URI, and per-process API key")
		}
	}
}

func TestAntigravityModelSourceUsesScopedEndpointEnvironment(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	resourceDir := filepath.Join(root, "resources", "antigravity")
	if err := os.MkdirAll(resourceDir, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"antigravity","harness":{"id":"agy","aliases":["antigravity"],"binary":"agy","display_name":"Antigravity","icon":"antigravity","model_source_protocol":"gemini-generate-content","model_source_limitation":"","home_isolation":{"kind":"none"}}}`
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_RESOURCE_ROOT", root)
	source := ModelSource{ID: "Vrooli credits", Kind: "vrooli_credits", Model: "fixture-model", BaseURL: "https://door.example/api/v1/ai", SecretEnv: "VROOLI_CODING_SESSION_KEY"}
	args, env, err := ResolveModelSource("agy", source, []string{"--model", "saved-model"}, []string{"VROOLI_CODING_SESSION_KEY=session-secret"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(args, []string{"--model", "gemini-api://door.example/models/fixture-model"}) {
		t.Fatalf("Antigravity model selector = %q", args)
	}
	if got := environmentValue(env, "GEMINI_API_KEY"); got != "session-secret" {
		t.Fatalf("Antigravity did not receive its scoped key: %q", got)
	}
	if got := environmentValue(env, "GOOGLE_GEMINI_BASE_URL"); got != source.BaseURL {
		t.Fatalf("Antigravity custom endpoint = %q, want %q", got, source.BaseURL)
	}
	if strings.Contains(strings.Join(args, " "), "session-secret") {
		t.Fatal("session key entered Antigravity arguments")
	}
}

func TestAntigravitySourceCapabilityIsQualifiedByHarnessData(t *testing.T) {
	descriptor, ok := harness.Active().Lookup("antigravity")
	if !ok || descriptor.ModelSourceProtocol != "gemini-generate-content" || descriptor.ModelSourceLimitation != "" {
		t.Fatalf("Antigravity protocol must be data-defined and its qualification limitation cleared: %+v", descriptor)
	}
	if !SupportsModelSource("antigravity") {
		t.Fatal("Antigravity with a qualified production adapter must be offered as supported")
	}
	source := ModelSource{Kind: "vrooli_credits", Model: "coding", BaseURL: "https://example.invalid/api/v1/ai", SecretEnv: "VROOLI_SESSION_KEY"}
	if _, _, err := ResolveModelSource("antigravity", source, nil, []string{"VROOLI_SESSION_KEY=session-secret"}); err != nil {
		t.Fatalf("qualified Antigravity adapter was rejected: %v", err)
	}
}

func TestModelSourceFailsBeforeSpawnOnMissingKeyOrUnsafeEndpoint(t *testing.T) {
	source := ModelSource{ID: "credits", Kind: "vrooli_credits", Model: "coding", BaseURL: "https://example.invalid/api/v1/ai", SecretEnv: "VROOLI_SESSION_KEY"}
	if _, _, err := ResolveModelSource("opencode", source, nil, nil); err == nil {
		t.Fatal("missing key accepted")
	}
	for _, address := range []string{"http://example.invalid/v1", "https://secret@example.invalid/v1", "https://example.invalid/v1?key=secret", "file:///etc/passwd"} {
		source.BaseURL = address
		if _, _, err := ResolveModelSource("opencode", source, nil, []string{"VROOLI_SESSION_KEY=x"}); err == nil {
			t.Fatalf("unsafe address accepted: %s", address)
		}
	}
}

func TestModelSourceReplacesNativeModelOverride(t *testing.T) {
	source := ModelSource{ID: "credits", Kind: "vrooli_credits", Model: "approved", BaseURL: "https://door.example/api/v1/ai", SecretEnv: "SESSION_KEY"}
	for _, harness := range []string{"codex", "opencode"} {
		args, _, err := ResolveModelSource(harness, source, []string{"exec", "-m", "saved-model", "--model=other-model"}, []string{"SESSION_KEY=synthetic"})
		if err != nil {
			t.Fatal(err)
		}
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "saved-model") || strings.Contains(joined, "other-model") {
			t.Fatalf("old native model survived: %v", args)
		}
		if !strings.Contains(joined, "--model approved") && !strings.Contains(joined, "--model vrooli-source/approved") {
			t.Fatalf("chosen source model absent: %v", args)
		}
	}
}
