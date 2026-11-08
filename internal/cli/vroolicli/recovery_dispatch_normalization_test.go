package vroolicli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
)

func recoveryCommandSpellings(name string) []string {
	mixed := []byte(name)
	for i := range mixed {
		if i%2 == 0 && mixed[i] >= 'a' && mixed[i] <= 'z' {
			mixed[i] -= 'a' - 'A'
		}
	}
	return []string{name, strings.ToUpper(name), string(mixed), " \t" + name + "\n", "\u2003" + string(mixed) + "\u00a0"}
}

func setRecoveryNormalizationCaller(t *testing.T, agent bool) {
	t.Helper()
	for _, key := range []string{cliutil.AgentSessionEnv, "VROOLI_SANDBOX_ID", "VROOLI_SANDBOX_MERGED", "VROOLI_SWARM_MANAGER_SESSION_ID", "VROOLI_AGENT_IDENTITY_TOKEN", "VROOLI_AGENT_MANAGER_API_BASE"} {
		t.Setenv(key, "")
	}
	if agent {
		t.Setenv(cliutil.AgentSessionEnv, "native-fallback-normalization-fixture")
	}
}

func TestRecoveryNativeScenarioNormalizationBoundary(t *testing.T) {
	for _, agent := range []bool{true, false} {
		for _, command := range recoveryCommandSpellings("scenario") {
			for _, operation := range recoveryCommandSpellings("stop-all") {
				t.Run(fmt.Sprintf("agent=%t/%q/%q", agent, command, operation), func(t *testing.T) {
					setRecoveryNormalizationCaller(t, agent)
					app := newRuntimeTestApp(t, t.TempDir())
					called := 0
					fakeEffect := func(ctx *AppContext, args []string) error {
						if _, err := scenariocli.ParseStopAllRequest(ctx.Globals.JSON, args); err != nil {
							return err
						}
						called++
						return nil
					}
					// Keep actual App.Runner, scenario RootHandler and stop-all
					// parser. Only the final actuator is fake, even for the
					// operator controls that must reach it to prove acceptance.
					app.registry = rootcli.NewRegistry(map[topcli.CommandID]rootcli.Handler[*AppContext]{
						topcli.CommandScenario: topLevelHandlerDefinitions()[topcli.CommandScenario],
					}, map[scenariocli.CommandID]rootcli.Handler[*AppContext]{scenariocli.CommandStopAll: fakeEffect})
					var stdout, stderr bytes.Buffer
					exit := app.Runner().Run([]string{"--quiet", command, operation, "--json"}, &stdout, &stderr)
					if agent {
						if exit == 0 || called != 0 || !strings.Contains(stderr.String(), clipolicy.CodeDestructiveVrooliMaintenanceBlocked) {
							t.Fatalf("accepted spelling bypassed policy: exit=%d fake=%d stderr=%s", exit, called, stderr.String())
						}
					} else if exit != 0 || called != 1 {
						t.Fatalf("operator spelling not accepted by actual parser: exit=%d fake=%d stderr=%s", exit, called, stderr.String())
					}
				})
			}
		}
	}
}

func TestRecoveryNormalizedSafeScenarioOperationsPreserveArguments(t *testing.T) {
	for _, operation := range []string{"restart", "wait"} {
		for _, spelling := range recoveryCommandSpellings(operation) {
			t.Run(fmt.Sprintf("%s/%q", operation, spelling), func(t *testing.T) {
				setRecoveryNormalizationCaller(t, true)
				app := newRuntimeTestApp(t, t.TempDir())
				path := t.TempDir() + "/Mixed Case/STOP-ALL"
				called := 0
				handlers := app.buildScenarioHandlerMap()
				handlers[scenariocli.CommandRestart] = func(ctx *AppContext, args []string) error {
					req, err := scenariocli.ParseRestartRequest(ctx.Globals.JSON, args)
					if err != nil {
						return err
					}
					if req.Name != "agent-manager" || req.Options.CustomPath != path {
						t.Fatalf("target/path changed: name=%q path=%q", req.Name, req.Options.CustomPath)
					}
					called++
					return nil
				}
				handlers[scenariocli.CommandWait] = func(ctx *AppContext, args []string) error {
					req, err := scenariocli.ParseWaitRequest(ctx.Globals.JSON, args)
					if err != nil {
						return err
					}
					if req.Name != "agent-manager" || req.TimeoutSeconds != 17 {
						t.Fatalf("target/flag changed: %+v", req)
					}
					called++
					return nil
				}
				app.registry = rootcli.NewRegistry(map[topcli.CommandID]rootcli.Handler[*AppContext]{
					topcli.CommandScenario: topLevelHandlerDefinitions()[topcli.CommandScenario],
				}, handlers)
				args := []string{"--json", " ScEnArIo ", spelling, "agent-manager"}
				if operation == "restart" {
					args = append(args, "--path", path)
				} else {
					args = append(args, "--timeout", "17")
				}
				var stdout, stderr bytes.Buffer
				if exit := app.Runner().Run(args, &stdout, &stderr); exit != 0 || called != 1 {
					t.Fatalf("safe maintenance changed: exit=%d fake=%d stderr=%s", exit, called, stderr.String())
				}
			})
		}
	}
}
