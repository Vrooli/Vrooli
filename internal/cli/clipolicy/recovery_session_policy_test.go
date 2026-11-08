package clipolicy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func TestNativeSessionMarkerKeepsDestructiveCommandsGuarded(t *testing.T) {
	for _, env := range [][]string{{cliutil.AgentSessionEnv + "=vrooli-agent-codex-probe.scope"}, {cliutil.AgentSessionEnv + "=vrooli-op-coding-agent-recovery-probe.service"}} {
		for _, argv := range [][]string{{"vrooli", "stop-all"}, {"vrooli", "cleanup", "orphans"}, {"vrooli", "resource", "stop", "postgres"}} {
			got := ClassifyAgentCommand(argv, env)
			if got.Allowed || got.Code != CodeDestructiveVrooliMaintenanceBlocked {
				t.Errorf("native marker %v command %v bypassed guard: %+v", env, argv, got)
			}
		}
		for _, argv := range [][]string{{"vrooli", "scenario", "restart", "agent-manager"}, {"vrooli", "scenario", "wait", "agent-manager", "--json"}, {"vrooli", "cleanup", "locks"}} {
			if got := ClassifyAgentCommand(argv, env); !got.Allowed {
				t.Errorf("ordinary maintenance %v denied: %+v", argv, got)
			}
		}
	}
	for _, env := range [][]string{nil, {"HOME=/operator", "PATH=/usr/bin"}, {cliutil.AgentSessionEnv + "="}} {
		if got := ClassifyAgentCommand([]string{"vrooli", "scenario", "restart", "agent-manager"}, env); !got.Allowed {
			t.Errorf("ordinary operator maintenance denied: %+v", got)
		}
		if got := ClassifyAgentCommand([]string{"vrooli", "stop-all"}, env); !got.Allowed {
			t.Errorf("empty/non-agent context changed: %+v", got)
		}
	}
}

func TestUnavailableAMLaunchPassesAClassifiableNativeMarker(t *testing.T) {
	for _, runner := range []string{"opencode", "codex"} {
		t.Run(runner, func(t *testing.T) {
			for _, key := range agentContextEnvKeys {
				t.Setenv(key, "")
			}
			t.Setenv(cliutil.AgentSessionEnv, "")
			t.Setenv(cliutil.ModelSourceEnvironment, "")
			t.Setenv("VROOLI_HOME", t.TempDir())
			t.Setenv("CODEX_HOME", t.TempDir())
			t.Setenv("HOME", t.TempDir())
			previous := cliutil.DefaultEditorLeaseRecorder
			cliutil.DefaultEditorLeaseRecorder = nil
			t.Cleanup(func() { cliutil.DefaultEditorLeaseRecorder = previous })
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "fixture owner unavailable", http.StatusServiceUnavailable)
			}))
			defer server.Close()
			ran := false
			result, err := cliutil.LaunchCodingAgentResult(context.Background(), cliutil.AgentLaunchRequest{Agent: runner, APIBase: server.URL, WorkingDir: t.TempDir(), Environment: []string{"HOME=" + os.Getenv("HOME"), "PATH=/usr/bin:/bin"}, Args: []string{"--safe-fixture"}, AttachTimeout: time.Second, LookPath: func(string) (string, error) { return "/fixture/not-executed", nil }, RunChild: func(_ context.Context, _ string, _ []string, environment []string, _ io.Reader, _, _ io.Writer) error {
				ran = true
				values := map[string]string{}
				keys := []string{}
				for _, entry := range environment {
					for i := range entry {
						if entry[i] == '=' {
							values[entry[:i]] = entry[i+1:]
							keys = append(keys, entry[:i])
							break
						}
					}
				}
				if values[cliutil.AgentSessionEnv] == "" || values[cliutil.AgentManagerIdentityTokenEnv] != "" {
					t.Fatal("fixture did not reach an unattached native session")
				}
				stop := ClassifyAgentCommand([]string{"vrooli", "stop-all"}, environment)
				restart := ClassifyAgentCommand([]string{"vrooli", "scenario", "restart", "agent-manager"}, environment)
				if path := os.Getenv("VROOLI_RECOVERY_FALLBACK_PROBE_OUT"); path != "" {
					b, _ := json.MarshalIndent(map[string]any{"runner": runner, "actual_launcher_environment_keys": keys, "session_marker_present": true, "identity_token_present": false, "stop_all_decision": stop, "ordinary_restart_decision": restart, "host_action_executed": false, "model_executed": false}, "", "  ")
					if err := os.WriteFile(path+"-"+runner+".json", append(b, '\n'), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if stop.Allowed {
					t.Error("actual unattached launcher child was treated as operator")
				}
				if !restart.Allowed {
					t.Error("ordinary supported restart was denied")
				}
				return nil
			}})
			if err != nil || !ran || result.Tier != "tier-4" {
				t.Fatalf("fallback launch fixture result=%+v ran=%v err=%v", result, ran, err)
			}
		})
	}
}
