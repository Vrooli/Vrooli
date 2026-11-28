package actor

import "testing"

func TestNamePrefersTheExplicitActorThenTheAgentThenTheUser(t *testing.T) {
	env := map[string]string{"IMAGE_TOOLS_ACTOR": "operator", "AI_AGENT": "claude-code_agent", "USER": "matt"}
	get := func(k string) string { return env[k] }
	if got := fromEnv(get); got != "operator" {
		t.Fatalf("explicit = %q", got)
	}
	delete(env, "IMAGE_TOOLS_ACTOR")
	if got := fromEnv(get); got != "claude-code_agent" {
		t.Fatalf("agent = %q", got)
	}
	delete(env, "AI_AGENT")
	if got := fromEnv(get); got != "matt" {
		t.Fatalf("user = %q", got)
	}
	delete(env, "USER")
	if got := fromEnv(get); got != "cli" {
		t.Fatalf("fallback = %q", got)
	}
}
