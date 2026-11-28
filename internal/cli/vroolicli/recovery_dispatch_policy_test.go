package vroolicli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
)

func TestRecoveryNativeSessionCommandDispatchPolicy(t *testing.T) {
	contexts := []struct {
		name, key, value string
		agent            bool
	}{
		{"unattached-native", cliutil.AgentSessionEnv, "vrooli-op-coding-agent-recovery-probe.service", true},
		{"attached-token", "VROOLI_AGENT_IDENTITY_TOKEN", "fixture-token-not-consumed", true},
		{"sandbox", "VROOLI_SANDBOX_ID", "fixture-sandbox", true},
		{"operator", cliutil.AgentSessionEnv, "", false},
	}
	requests := []struct {
		args         []string
		agentBlocked bool
	}{
		{[]string{"--json", "resource", "stop", "postgres"}, false},
		{[]string{"resource", "--json", "stop-all"}, true},
		{[]string{"--json", "resource", "stop-all"}, true},
		{[]string{"--quiet", "scenario", "stop-all", "--json"}, true},
		{[]string{"--json", "stop"}, true},
		{[]string{"--json", "cleanup", "orphans"}, true},
		{[]string{"--json", "orphans", "kill"}, true},
		{[]string{"--json", "locks", "clean", "--all"}, true},
		{[]string{"--json", "scenario", "restart", "agent-manager"}, false},
		{[]string{"--json", "scenario", "wait", "agent-manager"}, false},
		{[]string{"--json", "cleanup", "locks"}, false},
		{[]string{"--json", "locks", "clean"}, false},
		{[]string{"--json", "cleanup", "orphans", "--dry-run"}, false},
		{[]string{"--json", "resource", "status", "postgres"}, false},
	}
	for _, caller := range contexts {
		for _, request := range requests {
			t.Run(caller.name+"/"+strings.Join(request.args, "_"), func(t *testing.T) {
				for _, key := range []string{cliutil.AgentSessionEnv, "VROOLI_SANDBOX_ID", "VROOLI_SANDBOX_MERGED", "VROOLI_SWARM_MANAGER_SESSION_ID", "VROOLI_AGENT_IDENTITY_TOKEN", "VROOLI_AGENT_MANAGER_API_BASE"} {
					t.Setenv(key, "")
				}
				t.Setenv(caller.key, caller.value)
				app := newRuntimeTestApp(t, t.TempDir())
				called := 0
				// Replace every selected handler: a policy failure can never reach
				// a real resource/scenario controller or host operation in this test.
				fake := func(_ *AppContext, _ []string) error { called++; return nil }
				app.registry = rootcli.NewRegistry(map[topcli.CommandID]rootcli.Handler[*AppContext]{
					topcli.CommandResource: fake,
					topcli.CommandScenario: fake,
					topcli.CommandStop:     fake,
					topcli.CommandCleanup:  fake,
					topcli.CommandOrphans:  fake,
					topcli.CommandLocks:    fake,
				}, map[scenariocli.CommandID]rootcli.Handler[*AppContext]{})
				var stdout, stderr bytes.Buffer
				// cmd/vrooli calls configuredApp().Runner() directly, bypassing
				// App.Run. Exercise that same boundary with only fake handlers.
				exit := app.Runner().Run(request.args, &stdout, &stderr)
				if caller.agent && request.agentBlocked {
					var result struct {
						Code string `json:"code"`
					}
					if err := json.Unmarshal(stderr.Bytes(), &result); err != nil {
						t.Fatalf("policy error is not JSON: %v: %s", err, stderr.String())
					}
					if exit == 0 || called != 0 || result.Code != clipolicy.CodeDestructiveVrooliMaintenanceBlocked {
						t.Fatalf("destructive request reached fake dispatch: exit=%d called=%d code=%q", exit, called, result.Code)
					}
				} else if exit != 0 || called != 1 {
					t.Fatalf("ordinary request changed: exit=%d called=%d stderr=%q", exit, called, stderr.String())
				}
			})
		}
	}
}
