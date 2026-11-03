package cliutil

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
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
	for _, runner := range []string{"codex", "opencode"} {
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
