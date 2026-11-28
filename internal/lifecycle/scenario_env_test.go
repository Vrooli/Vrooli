package lifecycle

import (
	"slices"
	"testing"

	"github.com/vrooli/vrooli/internal/values"
)

// callerEnv is the environment of an agent shell that restarts a scenario: it
// carries host settings the scenario legitimately shares, and authentication,
// credential and identity settings that belong to the caller.
func callerEnv(extra ...string) []string {
	return append([]string{
		"PATH=/usr/bin:/bin",
		"HOME=/home/operator",
	}, extra...)
}

func TestScenarioStepEnvNeverInheritsCallerSecuritySettings(t *testing.T) {
	refused := map[string]string{
		"VROOLI_AUTH_MODE":                          "personal_local",
		"VROOLI_AUTH_PROVIDERS":                     "scenario_authenticator",
		"VROOLI_AUTH_SCENARIO_JWKS_URL":             "http://127.0.0.1:1/jwks",
		"VROOLI_AUTH_OWNER_BINDINGS_FILE":           "/tmp/owners.json",
		"VROOLI_AUTH_LOCAL_TOKEN_FILE":              "/tmp/token",
		"VROOLI_CLOUDFLARE_ACCESS_AUDIENCE":         "other-application",
		"VROOLI_CLOUDFLARE_ACCESS_JWKS_URL":         "http://127.0.0.1:1/certs",
		"VROOLI_AGENT_IDENTITY_TOKEN":               "agent-token",
		"VROOLI_AGENT_SESSION":                      "vrooli-agent-codex-1",
		"VROOLI_RUN_ID":                             "run-1",
		"VROOLI_HARNESS_SESSION_ID":                 "harness-1",
		"VROOLI_API_TOKEN":                          "control-plane-token",
		"VROOLI_BRIDGE_API_TOKEN":                   "bridge-token",
		"VROOLI_TEST_MODE_FORCE_ENABLE":             "true",
		"CLAUDE_CODE_SESSION_ID":                    "claude-session",
		"CLAUDE_CODE_MESSAGING_TOKEN":               "messaging-token",
		"CODEX_THREAD_ID":                           "codex-thread",
		"WC_WEB_CONSOLE_SESSION_ID":                 "terminal-1",
		"OPENROUTER_API_KEY":                        "sk-caller",
		"GITHUB_TOKEN":                              "ghp-caller",
		"AWS_SECRET_ACCESS_KEY":                     "aws-secret",
		"PGPASSWORD":                                "db-password",
		"npm_config_//registry.example/:_authToken": "npm-token",
		"npm_config__auth":                          "npm-basic",
	}
	extra := make([]string, 0, len(refused))
	for name, value := range refused {
		extra = append(extra, name+"="+value)
	}
	env := lifecycleStepEnvFrom(callerEnv(extra...), phasesDevelop, nil)
	for name := range refused {
		if got := values.EnvValue(env, name); got != "" {
			t.Errorf("%s reached the scenario process as %q; caller security settings must never be inherited", name, got)
		}
	}
	if values.EnvValue(env, "PATH") == "" || values.EnvValue(env, "HOME") != "/home/operator" {
		t.Fatalf("host settings lost while refusing caller security settings: %v", env)
	}
}

func TestScenarioStepEnvDropsUnlistedCallerVariables(t *testing.T) {
	env := lifecycleStepEnvFrom(callerEnv(
		"TEST_GENIE_SQLITE_PATH=/tmp/other.db",
		"NOTIFICATION_HUB_API_URL=http://127.0.0.1:1",
		"NODE_OPTIONS=--require /tmp/hook.js",
		"VROOLI_OUTPUT=json",
		"VROOLI_PROTO_LOCK_HELD=true",
		"JOURNAL_STREAM=8:123",
		"TMUX=/tmp/tmux-1000/default,1,0",
	), phasesDevelop, nil)
	for _, name := range []string{"TEST_GENIE_SQLITE_PATH", "NOTIFICATION_HUB_API_URL", "NODE_OPTIONS", "VROOLI_OUTPUT", "VROOLI_PROTO_LOCK_HELD", "JOURNAL_STREAM", "TMUX"} {
		if got := values.EnvValue(env, name); got != "" {
			t.Errorf("%s=%q reached the scenario process; only the documented pass-through set may be inherited", name, got)
		}
	}
}

func TestScenarioStepEnvPassesDocumentedHostAllowlist(t *testing.T) {
	passed := map[string]string{
		"PATH":                         "/usr/bin:/bin",
		"HOME":                         "/home/operator",
		"USER":                         "operator",
		"LANG":                         "C.UTF-8",
		"LC_ALL":                       "C.UTF-8",
		"TZ":                           "UTC",
		"XDG_RUNTIME_DIR":              "/run/user/1000",
		"DBUS_SESSION_BUS_ADDRESS":     "unix:path=/run/user/1000/bus",
		"DISPLAY":                      ":1",
		"XAUTHORITY":                   "/run/user/1000/gdm/Xauthority",
		"SSH_AUTH_SOCK":                "/run/user/1000/keyring/ssh",
		"GOPATH":                       "/home/operator/go",
		"GOROOT":                       "/opt/go",
		"GOWORK":                       "off",
		"PNPM_HOME":                    "/home/operator/.local/share/pnpm",
		"npm_config_child_concurrency": "4",
		"http_proxy":                   "http://proxy.internal:3128",
		"SSL_CERT_FILE":                "/etc/ssl/certs/ca.pem",
		"CUDA_VISIBLE_DEVICES":         "0",
		"VROOLI_ROOT":                  "/srv/vrooli",
		"VROOLI_SOURCE_ROOT":           "/srv/vrooli",
		"VROOLI_STORAGE_ROOT":          "/tmp/isolated-storage",
		"VROOLI_SANDBOX_MERGED":        "/tmp/sandbox/merged",
		"VROOLI_API_PORT":              "8092",
	}
	caller := make([]string, 0, len(passed))
	for name, value := range passed {
		caller = append(caller, name+"="+value)
	}
	env := lifecycleStepEnvFrom(caller, phasesDevelop, nil)
	for name, want := range passed {
		got := values.EnvValue(env, name)
		if name == "PATH" {
			if got == "" {
				t.Errorf("PATH was dropped")
			}
			continue
		}
		if got != want {
			t.Errorf("%s = %q, want the host value %q passed through", name, got, want)
		}
	}
}

func TestScenarioStepEnvDeclaredAuthModeWinsOverCaller(t *testing.T) {
	env := lifecycleStepEnvFrom(
		callerEnv("VROOLI_AUTH_MODE=personal_local", "VROOLI_AUTH_PROVIDERS="),
		phasesDevelop,
		map[string]string{"VROOLI_AUTH_MODE": "shared_provider", "VROOLI_AUTH_PROVIDERS": "cloudflare_access,scenario_authenticator"},
	)
	if got := values.EnvValue(env, "VROOLI_AUTH_MODE"); got != "shared_provider" {
		t.Fatalf("VROOLI_AUTH_MODE = %q, want the declared shared_provider", got)
	}
	if got := values.EnvValue(env, "VROOLI_AUTH_PROVIDERS"); got != "cloudflare_access,scenario_authenticator" {
		t.Fatalf("VROOLI_AUTH_PROVIDERS = %q, want the declared providers", got)
	}
}

func TestDroppedSecurityEnvNamesReportsOnlySecurityVariablesByName(t *testing.T) {
	names := droppedSecurityEnvNames(callerEnv(
		"VROOLI_AUTH_MODE=personal_local",
		"VROOLI_AGENT_IDENTITY_TOKEN=secret-value",
		"TEST_GENIE_SQLITE_PATH=/tmp/x.db",
		"LANG=C.UTF-8",
	))
	want := []string{"VROOLI_AGENT_IDENTITY_TOKEN", "VROOLI_AUTH_MODE"}
	if !slices.Equal(names, want) {
		t.Fatalf("refused security names = %v, want %v", names, want)
	}
}
