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

// TestCallerSessionStaysWithTheCaller covers a scenario started from an agent
// terminal. It must not inherit the terminal's session socket, token, or state
// directories; web-console used an inherited WC_SESSION_STATE_ROOT to find the
// caller's tmux server.
func TestCallerSessionStaysWithTheCaller(t *testing.T) {
	parent := Env{
		"HOME=/home/op",
		"PATH=/usr/bin",
		"SSH_AUTH_SOCK=/run/agent.sock",
		"CLAUDE_CODE_URL=http://localhost:8100",
		"TMUX=/sock,1,0",
		"CLAUDECODE=1",
		"CLAUDE_CODE_MESSAGING_SOCKET=/run/cc.sock",
		"CLAUDE_CODE_MESSAGING_TOKEN=secret",
		"CODEX_HOME=/state/web-console/sessions/codex/abc",
		"VROOLI_AGENT_SESSION=1",
		"WC_WEB_CONSOLE_SESSION_ID=abc",
		"WC_SESSION_STATE_ROOT=/state/web-console/sessions",
	}
	callerSession := []string{"TMUX", "CLAUDECODE", "CLAUDE_CODE_MESSAGING_SOCKET", "CLAUDE_CODE_MESSAGING_TOKEN", "CODEX_HOME", "VROOLI_AGENT_SESSION", "WC_WEB_CONSOLE_SESSION_ID", "WC_SESSION_STATE_ROOT"}
	for _, relationship := range []Relationship{ForeignScenario, Resource} {
		got := ForChildWithPlatform(parent, relationship, Platform{})
		for _, key := range callerSession {
			if contains(got, key) {
				t.Errorf("%v inherited caller session variable %s", relationship, key)
			}
		}
		for _, key := range []string{"HOME", "PATH", "SSH_AUTH_SOCK", "CLAUDE_CODE_URL"} {
			if !contains(got, key) {
				t.Errorf("%v dropped host variable %s", relationship, key)
			}
		}
	}
	same := ForChildWithPlatform(parent, SameScenario, Platform{})
	if !contains(same, "WC_SESSION_STATE_ROOT") || !contains(same, "TMUX") {
		t.Fatalf("same-scenario child lost its own session environment: %#v", same)
	}
	// A delegated agent keeps the session its launcher composed for it: the
	// agent home is derived from WC_SESSION_STATE_ROOT + WC_WEB_CONSOLE_SESSION_ID.
	delegated := ForChildWithPlatform(parent, DelegatedAgent, Platform{})
	if !contains(delegated, "WC_SESSION_STATE_ROOT") || !contains(delegated, "WC_WEB_CONSOLE_SESSION_ID") {
		t.Fatalf("delegated agent lost the session its launcher composed: %#v", delegated)
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

func TestIndependentOwnersDoNotInheritExecutionAttribution(t *testing.T) {
	parent := Env{
		"CODEX_AGENT_TAG=parent-run", "CLAUDE_AGENT_TAG=other-parent", "FUTURE_PROVIDER_AGENT_TAG=future-parent",
		"VROOLI_RUN_ID=parent-run", "VROOLI_HARNESS_SESSION_ID=parent-session", "VROOLI_AGENT_MANAGER_STORE_ID=parent-store",
		"VROOLI_RUNTIME_INSTANCE_ID=serving-instance", "VROOLI_RUNTIME_GENERATION=serving-generation", "VROOLI_PROCESS_ID=serving-process",
		"VROOLI_LIFECYCLE_MANAGED=true", "VROOLI_STEP=serve", "VROOLI_PHASE=develop", "LIFECYCLE_PHASE=develop",
		"VROOLI_VARIANT=parent-variant", "VROOLI_STORAGE_NAMESPACE=parent-namespace",
		"API_BASE_URL=http://parent", "API_BASE=http://parent", "WS_PORT=123", "VITE_API_BASE_URL=http://parent",
		"PATH=/bin", "HOME=/home/operator", "VROOLI_ROOT=/repo", "VROOLI_HOME=/runtime", "POSTGRES_HOST=db", "OPENAI_API_KEY=fixture-oauth-availability",
	}
	for _, relationship := range []Relationship{Resource, ForeignScenario} {
		got := ForChildWithPlatform(parent, relationship, Platform{})
		for _, key := range []string{"CODEX_AGENT_TAG", "CLAUDE_AGENT_TAG", "FUTURE_PROVIDER_AGENT_TAG", "VROOLI_RUN_ID", "VROOLI_HARNESS_SESSION_ID", "VROOLI_AGENT_MANAGER_STORE_ID", "VROOLI_RUNTIME_INSTANCE_ID", "VROOLI_RUNTIME_GENERATION", "VROOLI_PROCESS_ID", "VROOLI_LIFECYCLE_MANAGED", "VROOLI_STEP", "VROOLI_PHASE", "LIFECYCLE_PHASE", "VROOLI_VARIANT", "VROOLI_STORAGE_NAMESPACE", "API_BASE_URL", "API_BASE", "WS_PORT", "VITE_API_BASE_URL"} {
			if contains(got, key) {
				t.Errorf("boundary %v inherited execution attribution %s", relationship, key)
			}
		}
		for _, key := range []string{"PATH", "HOME", "VROOLI_ROOT", "VROOLI_HOME", "POSTGRES_HOST", "OPENAI_API_KEY"} {
			if !contains(got, key) {
				t.Errorf("boundary %v removed host routing/access %s", relationship, key)
			}
		}
	}
	same := ForChildWithPlatform(parent, SameScenario, Platform{})
	for _, key := range []string{"CODEX_AGENT_TAG", "VROOLI_RUN_ID", "VROOLI_RUNTIME_INSTANCE_ID", "API_BASE_URL"} {
		if !contains(same, key) {
			t.Errorf("same owner lost %s", key)
		}
	}
	delegated := ForChildWithPlatform(parent, DelegatedAgent, Platform{})
	if !contains(delegated, "CODEX_AGENT_TAG") || !contains(delegated, "VROOLI_RUN_ID") || !contains(delegated, "VROOLI_HARNESS_SESSION_ID") || contains(delegated, "VROOLI_RUNTIME_INSTANCE_ID") {
		t.Fatal("delegated launcher changed composed caller-session versus service ownership semantics")
	}
}

func TestIndependentOwnerOverlayReplacesOnlyExplicitChildAttribution(t *testing.T) {
	for _, platform := range []Platform{{}, {CaseInsensitive: true}} {
		parent := Env{"CODEX_AGENT_TAG=parent", "VROOLI_RUN_ID=parent", "CODEX_HOME=/parent", "VROOLI_AGENT_IDENTITY_TOKEN=old", "VROOLI_AGENT_MANAGER_STORE_ID=old-store", "CODEX_THREAD_ID=old-thread"}
		got := WithOverlayWithPlatform(parent, Resource, Env{"CODEX_AGENT_TAG=child", "VROOLI_RUN_ID=child", "CODEX_HOME=/child", "VROOLI_AGENT_IDENTITY_TOKEN=verified-child", "VROOLI_AGENT_MANAGER_STORE_ID=child-store"}, platform)
		for _, pair := range [][2]string{{"CODEX_AGENT_TAG", "child"}, {"VROOLI_RUN_ID", "child"}, {"CODEX_HOME", "/child"}, {"VROOLI_AGENT_IDENTITY_TOKEN", "verified-child"}, {"VROOLI_AGENT_MANAGER_STORE_ID", "child-store"}} {
			if !containsValue(got, pair[0], pair[1]) {
				t.Errorf("platform %+v did not retain explicit child %s", platform, pair[0])
			}
		}
		if contains(got, "CODEX_THREAD_ID") {
			t.Fatal("new owner inherited old thread")
		}
	}
}

func TestWindowsIndependentOwnerFoldsAttributionAndChildOverlay(t *testing.T) {
	got := WithOverlayWithPlatform(Env{"cOdEx_AgEnT_tAg=parent", "vrooli_run_id=parent", "wc_web_console_session_id=parent", "vrooli_runtime_instance_id=host", "scenario_path=/host", "vrooli_agent_identity_token=old", "CodeX_Home=/parent", "Path=parent"}, Resource, Env{"CODEX_HOME=C:\\child", "VROOLI_AGENT_IDENTITY_TOKEN=child", "PATH=child"}, Platform{CaseInsensitive: true})
	if len(got) != 3 || !containsValue(got, "CODEX_HOME", "C:\\child") || !containsValue(got, "VROOLI_AGENT_IDENTITY_TOKEN", "child") || !containsValue(got, "PATH", "child") {
		t.Fatal("Windows attribution was inherited or explicit child entries were duplicated/lost")
	}
}
