package resourcehandlers

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/resources"
)

func TestRecoveryResourceNormalizationBoundary(t *testing.T) {
	spellings := func(name string) []string {
		return []string{name, strings.ToUpper(name), strings.ToUpper(name[:1]) + name[1:], " \t" + name + "\n", "\u2003" + strings.ToUpper(name) + "\u00a0"}
	}
	for _, agent := range []bool{true, false} {
		for _, command := range spellings("resource") {
			for _, operation := range []string{"stop", "stop-all"} {
				for _, spelling := range spellings(operation) {
					t.Run(fmt.Sprintf("agent=%t/%q/%q", agent, command, spelling), func(t *testing.T) {
						for _, key := range []string{cliutil.AgentSessionEnv, "VROOLI_SANDBOX_ID", "VROOLI_SANDBOX_MERGED", "VROOLI_SWARM_MANAGER_SESSION_ID", "VROOLI_AGENT_IDENTITY_TOKEN", "VROOLI_AGENT_MANAGER_API_BASE"} {
							t.Setenv(key, "")
						}
						if agent {
							t.Setenv(cliutil.AgentSessionEnv, "native-resource-normalization-fixture")
						}
						ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
						controllers, effects := 0, 0
						deps := rootcli.HandlerDeps[testContext]{
							Stdout: func(c testContext) io.Writer { return c.stdout },
							ResourceController: func(testContext) (*resources.Controller, error) {
								controllers++
								return &resources.Controller{}, nil
							},
						}
						const target = "  ExampleCase/STOP-ALL  "
						final := map[string]rootcli.ResourceHandler[testContext]{
							"stop": func(_ testContext, _ *resources.Controller, args []string) error {
								req, err := resourcecli.ParseControlRequest("stop", args)
								if err != nil {
									return err
								}
								if req.Name != target {
									t.Fatalf("target argument changed: %q", req.Name)
								}
								effects++
								return nil
							},
							"stop-all": func(_ testContext, _ *resources.Controller, args []string) error {
								if _, err := resourcecli.ParseStopAllRequest(args); err != nil {
									return err
								}
								effects++
								return nil
							},
						}
						registry := rootcli.NewRegistry(map[topcli.CommandID]rootcli.Handler[testContext]{
							topcli.CommandResource: rootHandlerWithCommands(deps, final),
						}, map[scenariocli.CommandID]rootcli.Handler[testContext]{})
						runner := rootcli.NewRunner(rootcli.RunnerConfig[testContext]{
							Registry:   registry,
							NewContext: func(rootcli.GlobalOptions, io.Writer, io.Writer, *slog.Logger) testContext { return ctx },
							NewLogger: func(rootcli.GlobalOptions, io.Writer) (*slog.Logger, func()) {
								return slog.New(slog.NewTextHandler(io.Discard, nil)), func() {}
							},
							ResolveRoot: func() (string, error) { return t.TempDir(), nil },
							BeforeDispatch: func(_ testContext, parsed rootcli.ParsedArgs) error {
								argv := append([]string{"vrooli", parsed.Command}, parsed.Args...)
								return clipolicy.NewCommandPolicyError(clipolicy.ClassifyAgentCommand(argv, os.Environ()))
							},
						})
						args := []string{command, "--json", spelling}
						if operation == "stop" {
							args = append(args, target)
						}
						exit := runner.Run(args, ctx.stdout, ctx.stderr)
						if agent {
							if exit == 0 || controllers != 0 || effects != 0 || !strings.Contains(ctx.stderr.String(), clipolicy.CodeDestructiveVrooliMaintenanceBlocked) {
								t.Fatalf("resource spelling bypass: exit=%d controllers=%d fakeEffects=%d stderr=%s", exit, controllers, effects, ctx.stderr.String())
							}
						} else if exit != 0 || controllers != 1 || effects != 1 {
							t.Fatalf("operator grammar changed: exit=%d controllers=%d fakeEffects=%d stderr=%s", exit, controllers, effects, ctx.stderr.String())
						}
					})
				}
			}
		}
	}
}
