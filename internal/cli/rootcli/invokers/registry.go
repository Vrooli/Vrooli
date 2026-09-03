// Package invokers is the registry of every process that builds an argv for
// the vrooli CLI. One test parses each registered argv through the real root
// parser, so a change to the CLI's globals or command tree fails a test
// instead of a boot.
//
// Every entry calls the production builder (the argv catalog in
// repo-contract-go/cliinvoke, a rendered native-service definition, a pinned
// shell fixture, or a CI workflow scan). Nothing here retypes an argv: a
// retyped argv drifts silently, which is the failure this package exists to
// catch.
package invokers

import (
	"github.com/vrooli/repo-contract-go/cliinvoke"
)

// Invoker is one argv producer.
type Invoker struct {
	// Name identifies the site; Owner is the repo-relative file that builds
	// the argv in production.
	Name  string
	Owner string
	// Argv returns the production argv after the binary. Builders that read
	// files or render definitions return the read or render error.
	Argv func() ([]string, error)
}

// Runners are the files that execute an Invocation whose argv is supplied by
// a caller (the typed client, the autoheal check executor, the loop's invoke
// wrapper). They produce no argv of their own, so they are not owners; the
// owners are the files that call them.
var Runners = []string{
	"packages/vrooli-cli-go/client.go",
	"scenarios/vrooli-autoheal/api/internal/checks/executor.go",
	"scenarios/vrooli-autoheal/cli/loop/main.go",
	"scenarios/vrooli-autoheal/cli/loop/invoke.go",
}

func static(name, owner string, argv []string) Invoker {
	return Invoker{Name: name, Owner: owner, Argv: func() ([]string, error) { return argv, nil }}
}

const (
	loopOwner          = "scenarios/vrooli-autoheal/cli/loop/lifecycle.go"
	loopPortsOwner     = "scenarios/vrooli-autoheal/cli/loop/ports.go"
	loopPreflightOwner = "scenarios/vrooli-autoheal/cli/loop/preflight.go"
	autohealScenario   = "vrooli-autoheal"
)

// All returns every registered invoker. Catalog-backed entries are listed
// first, then rendered native definitions, then file-backed fixtures.
func All() ([]Invoker, error) {
	items := []Invoker{
		static("autoheal-loop/lifecycle-start", loopOwner, cliinvoke.ScenarioLifecycle("start", autohealScenario, true)),
		static("autoheal-loop/lifecycle-restart", loopOwner, cliinvoke.ScenarioLifecycle("restart", autohealScenario, true)),
		static("autoheal-loop/status-json", loopPortsOwner, cliinvoke.ScenarioStatusJSON(autohealScenario)),
		static("autoheal-loop/port", loopPortsOwner, cliinvoke.ScenarioPort(autohealScenario, "API_PORT")),
		static("autoheal-loop/preflight-version", loopPreflightOwner, cliinvoke.VersionJSON()),
		static("runtime-supervisor/run", "internal/runtimesupervisor/service.go", cliinvoke.RuntimeSupervisorRun()),
		static("runtime-supervisor/recovery-restart", "internal/runtimesupervisor/recovery_controller.go", cliinvoke.ScenarioRestartInstance("example-scenario", "live")),
		static("agent-recover/setup", "internal/cli/vroolicli/agent.go", cliinvoke.ScenarioSetup("example-scenario")),
		static("autoheal-watchdog/loop-rebuild", "internal/safeguards/autoheal-watchdog/handler.go", cliinvoke.ScenarioSetup(autohealScenario)),
		static("agent-recover/start-best-effort", "internal/cli/vroolicli/agent.go", cliinvoke.ScenarioLifecycle("start", "example-scenario", true)),
		static("cli-core/port-detector/port", "packages/cli-core/cliutil/port_detector.go", cliinvoke.ScenarioPortJSON("example-scenario", "API_PORT")),
		static("cli-core/port-detector/status", "packages/cli-core/cliutil/port_detector.go", cliinvoke.ScenarioStatusJSON("example-scenario")),
		static("cli-core/scenario-app/preflight-start", "packages/cli-core/cliapp/scenario_app.go", cliinvoke.ScenarioLifecycle("start", "example-scenario", false)),
		static("cli-core/scenario-app/preflight-setup", "packages/cli-core/cliapp/scenario_app.go", cliinvoke.ScenarioLifecycle("setup", "example-scenario", false)),
		static("autoheal-api/agent-recover", "scenarios/vrooli-autoheal/api/main.go", cliinvoke.AgentRecover("example-scenario", "unhealthy", autohealScenario)),
		static("autoheal-api/readiness-inspection", "scenarios/vrooli-autoheal/api/internal/checks/system/boot_recovery_readiness.go", cliinvoke.SetupStatusReadiness()),
		static("autoheal-cli/watchdog-install", "scenarios/vrooli-autoheal/cli/domains/watchdog/register.go", cliinvoke.Setup(false)),
		static("autoheal-cli/watchdog-install-json", "scenarios/vrooli-autoheal/cli/domains/watchdog/register.go", cliinvoke.Setup(true)),
		static("autoheal-cli/diagnose-port", "scenarios/vrooli-autoheal/cli/operations.go", cliinvoke.DiagnosePort("18080", "example-scenario")),
	}
	rendered, err := renderedInvokers()
	if err != nil {
		return nil, err
	}
	items = append(items, rendered...)
	bootstrap, err := bootstrapInvokers()
	if err != nil {
		return nil, err
	}
	items = append(items, bootstrap...)
	ci, err := workflowInvokers()
	if err != nil {
		return nil, err
	}
	return append(items, ci...), nil
}
