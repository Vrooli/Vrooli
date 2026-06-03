package runner

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestSanitizedBaseEnv_RemovesCrossScenarioVars(t *testing.T) {
	t.Setenv("API_PORT", "18800")
	t.Setenv("VROOLI_SCENARIO", "agent-manager")
	t.Setenv("VROOLI_PROCESS_ID", "pid-123")
	t.Setenv("VROOLI_STEP", "start-api")
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CODEX_HOME", "/home/u/.vrooli/state/vrooli/web-console/sessions/codex/abc")
	t.Setenv("SAFE_ENV_KEY", "keep-me")

	env := SanitizedBaseEnv()
	keys := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, found := strings.Cut(entry, "=")
		if found {
			keys = append(keys, key)
		}
	}

	if slices.Contains(keys, "API_PORT") {
		t.Fatal("expected API_PORT to be removed from runner environment")
	}
	if slices.Contains(keys, "VROOLI_SCENARIO") {
		t.Fatal("expected VROOLI_SCENARIO to be removed from runner environment")
	}
	if slices.Contains(keys, "VROOLI_PROCESS_ID") {
		t.Fatal("expected VROOLI_PROCESS_ID to be removed from runner environment")
	}
	if slices.Contains(keys, "VROOLI_STEP") {
		t.Fatal("expected VROOLI_STEP to be removed from runner environment")
	}
	if slices.Contains(keys, "CLAUDECODE") {
		t.Fatal("expected CLAUDECODE to be removed from runner environment")
	}
	if slices.Contains(keys, "CODEX_HOME") {
		t.Fatal("expected CODEX_HOME to be removed so codex defaults to writable $HOME/.codex inside the sandbox")
	}
	if !slices.Contains(env, fmt.Sprintf("%s=%s", "SAFE_ENV_KEY", "keep-me")) {
		t.Fatal("expected safe env var to remain available")
	}
}

func TestAppendEnvMap_IncludesProvidedVars(t *testing.T) {
	base := []string{"PATH=/usr/bin"}
	extras := map[string]string{
		"CUSTOM_API_PORT": "16544",
		"CUSTOM_API_URL":  "http://localhost:16544",
	}

	env := AppendEnvMap(base, extras)
	joined := strings.Join(env, "\n")

	if !strings.Contains(joined, "CUSTOM_API_PORT=16544") {
		t.Fatal("expected custom port env var to be present")
	}
	if !strings.Contains(joined, "CUSTOM_API_URL=http://localhost:16544") {
		t.Fatal("expected custom URL env var to be present")
	}
}
