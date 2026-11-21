package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const hookPolicy = `{"runner":"codex","excluded_models":["gpt-6-astra"],"model_aliases":{"astra":{"canonical_model":"gpt-6-astra","provider":"codex"}},
  "usage_tiers":{"model_map":{"a":"b"},"session_model":"b"}}`

// hookCall is what Codex sends before spawn_agent (observed on Codex 0.167).
func hookCall(model string) string {
	return `{"session_id":"s1","hook_event_name":"PreToolUse","tool_name":"collaborationspawn_agent","tool_input":{"task_name":"t","model":"` + model + `","fork_turns":"none"}}`
}

func TestModelHookDeniesOnlyACleanlyReadExcludedModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("VROOLI_AGENT_POLICY_HOME", home)
	policyRoot := t.TempDir()
	t.Setenv("VROOLI_ROOT", policyRoot)
	for _, key := range []string{"VROOLI_RESOURCE_ROOT", "PROJECT_ROOT", "VROOLI_SOURCE_ROOT"} {
		t.Setenv(key, "")
	}
	policy := filepath.Join(policyRoot, "resources", "codex", "model-policy.json")
	write := func(body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(policy), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(policy, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name   string
		policy string // "" leaves the file absent
		input  string
		deny   bool
		runner string // "" means codex; a runner nobody has a policy for stands in for a missing file
	}{
		{"excluded model by name", hookPolicy, hookCall("gpt-6-astra"), true, ""},
		{"excluded model by alias", hookPolicy, hookCall("astra"), true, ""},
		{"allowed model", hookPolicy, hookCall("gpt-6-luna"), false, ""},
		{"no model named", hookPolicy, `{"tool_name":"collaborationspawn_agent","tool_input":{"task_name":"t"}}`, false, ""},
		{"another tool", hookPolicy, `{"tool_name":"Bash","tool_input":{"model":"gpt-6-astra"}}`, false, ""},
		{"invalid input", hookPolicy, `{not json`, false, ""},
		{"empty input", hookPolicy, ``, false, ""},
		{"missing policy", "", hookCall("gpt-6-astra"), false, "ghost"},
		{"invalid policy", `{"excluded_models":`, hookCall("gpt-6-astra"), false, ""},
		{"wrong schema policy", `{"roles":{}}`, hookCall("gpt-6-astra"), false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(policy)
			if tc.policy != "" {
				write(tc.policy)
			}
			var stderr strings.Builder
			runner := tc.runner
			if runner == "" {
				runner = "codex"
			}
			err := run([]string{"model-hook", "--runner", runner}, strings.NewReader(tc.input), &strings.Builder{}, &stderr)
			var exit exitError
			denied := errors.As(err, &exit) && exit.code == modelHookDenyExit
			if denied != tc.deny || (err != nil && !denied) {
				t.Fatalf("err = %v (deny=%v), want deny=%v; stderr=%s", err, denied, tc.deny, stderr.String())
			}
			if !tc.deny && tc.input != "" && strings.Contains(tc.name, "polic") && !strings.Contains(stderr.String(), "allowed") {
				t.Fatalf("a broken policy allowed the call silently: %q", stderr.String())
			}
		})
	}
	log, err := os.ReadFile(filepath.Join(home, "excluded-model-spawns.jsonl"))
	if err != nil || strings.Count(string(log), "\n") != 2 || !strings.Contains(string(log), `"model":"gpt-6-astra"`) {
		t.Fatalf("denied attempts were not recorded: %q %v", log, err)
	}
}

// A failure to record must not change the decision.
func TestModelHookStillDeniesWhenTheLogCannotBeWritten(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_AGENT_POLICY_HOME", filepath.Join(blocker, "sub"))
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	if err := os.MkdirAll(filepath.Join(root, "resources", "codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "codex", "model-policy.json"), []byte(hookPolicy), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	err := run([]string{"model-hook"}, strings.NewReader(hookCall("gpt-6-astra")), &strings.Builder{}, &stderr)
	var exit exitError
	if !errors.As(err, &exit) || !strings.Contains(stderr.String(), "could not record") {
		t.Fatalf("err = %v, stderr = %q", err, stderr.String())
	}
}
