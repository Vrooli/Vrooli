package envkit

import "testing"

func TestResourceDropsRelativeIdentity(t *testing.T) {
	got := ForChildWithPlatform(Env{"SCENARIO_NAME=parent", "SCENARIO_PATH=/parent", "POSTGRES_HOST=db"}, Resource, Platform{})
	if contains(got, "SCENARIO_PATH") || !contains(got, "POSTGRES_HOST") {
		t.Fatalf("resource child environment = %#v", got)
	}
}

func TestSameScenarioKeepsRelativeIdentity(t *testing.T) {
	got := ForChildWithPlatform(Env{"API_PORT=1234", "SCENARIO_NAME=parent"}, SameScenario, Platform{})
	if !contains(got, "API_PORT") || !contains(got, "SCENARIO_NAME") {
		t.Fatalf("same-scenario child environment = %#v", got)
	}
}

func TestDelegatedAgentOnlyCarriesDelegatedCredential(t *testing.T) {
	got := ForChildWithPlatform(Env{"SCENARIO_NAME=parent", "VROOLI_AGENT_IDENTITY_TOKEN=secret", "CLAUDE_CODE_SESSION_ID=session"}, DelegatedAgent, Platform{})
	if !contains(got, "VROOLI_AGENT_IDENTITY_TOKEN") || contains(got, "SCENARIO_NAME") || contains(got, "CLAUDE_CODE_SESSION_ID") {
		t.Fatalf("delegated child environment = %#v", got)
	}
}

func TestIdentityCredentialsDoNotCrossNonDelegatedBoundaries(t *testing.T) {
	parent := Env{
		"API_PORT=1234",
		"SCENARIO_NAME=parent",
		"VROOLI_AGENT_IDENTITY_TOKEN=secret",
		"CLAUDE_CODE_SESSION_ID=session",
		"CODEX_THREAD_ID=thread",
	}
	for _, relationship := range []Relationship{SameScenario, ForeignScenario, Resource} {
		got := ForChildWithPlatform(parent, relationship, Platform{})
		if contains(got, "VROOLI_AGENT_IDENTITY_TOKEN") || contains(got, "CLAUDE_CODE_SESSION_ID") || contains(got, "CODEX_THREAD_ID") {
			t.Fatalf("%v inherited identity credentials: %#v", relationship, got)
		}
	}
}

func TestDelegatedAgentPreservesOnlyItsExplicitToken(t *testing.T) {
	got := WithOverlayWithPlatform(
		Env{"VROOLI_AGENT_IDENTITY_TOKEN=parent", "CLAUDE_CODE_SESSION_ID=parent-session"},
		DelegatedAgent,
		Env{"VROOLI_AGENT_IDENTITY_TOKEN=child"},
		Platform{},
	)
	if !contains(got, "VROOLI_AGENT_IDENTITY_TOKEN") || !containsValue(got, "VROOLI_AGENT_IDENTITY_TOKEN", "child") || contains(got, "CLAUDE_CODE_SESSION_ID") {
		t.Fatalf("delegated identity boundary = %#v", got)
	}
}

func TestWindowsFoldsEnvironmentNames(t *testing.T) {
	got := WithOverlayWithPlatform(Env{"Path=parent", "PATH=overlay"}, Resource, nil, Platform{CaseInsensitive: true})
	if len(got) != 1 || got[0] != "PATH=overlay" {
		t.Fatalf("windows environment = %#v", got)
	}
}

func TestWindowsTreatsRelativeNamesCaseInsensitively(t *testing.T) {
	got := ForChildWithPlatform(Env{"scenario_name=parent", "Api_Port=1234"}, Resource, Platform{CaseInsensitive: true})
	if contains(got, "scenario_name") || contains(got, "Api_Port") {
		t.Fatalf("windows relative identity leaked = %#v", got)
	}
}

func TestOverlayReplacesRelativeIdentity(t *testing.T) {
	got := WithOverlayWithPlatform(Env{"SCENARIO_NAME=parent"}, ForeignScenario, Env{"SCENARIO_NAME=child"}, Platform{})
	if len(got) != 1 || got[0] != "SCENARIO_NAME=child" {
		t.Fatalf("foreign environment = %#v", got)
	}
}

func contains(env Env, key string) bool {
	for _, entry := range env {
		if len(entry) > len(key) && entry[:len(key)] == key && entry[len(key)] == '=' {
			return true
		}
	}
	return false
}

func containsValue(env Env, key, value string) bool {
	for _, entry := range env {
		if entry == key+"="+value {
			return true
		}
	}
	return false
}
