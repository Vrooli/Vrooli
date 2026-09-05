# CLI invokers

Every process that builds an argv for the `vrooli` CLI is registered in
`internal/cli/rootcli/invokers`. The test `TestEveryInvokerResolvesThroughTheRootParser`
parses each argv below through the real root parser and fails on an unknown
command or on a retired global flag, so a change to the CLI's argv contract
fails a test instead of a boot. `TestEveryDirectSpawnIsRegistered` fails when a
Go file builds a `cliinvoke.Invocation` without being a registered owner.

This page is generated: run `go test ./internal/cli/rootcli/invokers -run TestDocIsCurrent -update`
after changing the registry.

| Invoker | Owner | Argv | Resolves to |
|---|---|---|---|
| `autoheal-loop/lifecycle-start` | `scenarios/vrooli-autoheal/cli/loop/lifecycle.go` | `vrooli scenario start vrooli-autoheal --best-effort` | `scenario start` |
| `autoheal-loop/lifecycle-restart` | `scenarios/vrooli-autoheal/cli/loop/lifecycle.go` | `vrooli scenario restart vrooli-autoheal --best-effort` | `scenario restart` |
| `autoheal-loop/status-json` | `scenarios/vrooli-autoheal/cli/loop/ports.go` | `vrooli scenario status vrooli-autoheal --json` | `scenario status` |
| `autoheal-loop/port` | `scenarios/vrooli-autoheal/cli/loop/ports.go` | `vrooli scenario port vrooli-autoheal API_PORT` | `scenario port` |
| `autoheal-loop/preflight-version` | `scenarios/vrooli-autoheal/cli/loop/preflight.go` | `vrooli version --json` | `version` |
| `runtime-supervisor/run` | `internal/runtimesupervisor/service.go` | `vrooli runtime supervisor run` | `runtime` |
| `runtime-supervisor/recovery-restart` | `internal/runtimesupervisor/recovery_controller.go` | `vrooli scenario restart example-scenario --instance live` | `scenario restart` |
| `agent-recover/setup` | `internal/cli/vroolicli/agent.go` | `vrooli scenario setup example-scenario` | `scenario setup` |
| `autoheal-watchdog/loop-rebuild` | `internal/safeguards/autoheal-watchdog/handler.go` | `vrooli scenario setup vrooli-autoheal` | `scenario setup` |
| `agent-recover/start-best-effort` | `internal/cli/vroolicli/agent.go` | `vrooli scenario start example-scenario --best-effort` | `scenario start` |
| `cli-core/port-detector/port` | `packages/cli-core/cliutil/port_detector.go` | `vrooli scenario port example-scenario API_PORT --json` | `scenario port` |
| `cli-core/port-detector/status` | `packages/cli-core/cliutil/port_detector.go` | `vrooli scenario status example-scenario --json` | `scenario status` |
| `cli-core/scenario-app/preflight-start` | `packages/cli-core/cliapp/scenario_app.go` | `vrooli scenario start example-scenario` | `scenario start` |
| `cli-core/scenario-app/preflight-setup` | `packages/cli-core/cliapp/scenario_app.go` | `vrooli scenario setup example-scenario` | `scenario setup` |
| `autoheal-api/agent-recover` | `scenarios/vrooli-autoheal/api/main.go` | `vrooli agent recover --scenario example-scenario --reason unhealthy --requester vrooli-autoheal` | `agent` |
| `autoheal-api/readiness-inspection` | `scenarios/vrooli-autoheal/api/internal/checks/system/boot_recovery_readiness.go` | `vrooli setup status --json --phase readiness` | `setup` |
| `autoheal-cli/watchdog-install` | `scenarios/vrooli-autoheal/cli/domains/watchdog/register.go` | `vrooli setup` | `setup` |
| `autoheal-cli/watchdog-install-json` | `scenarios/vrooli-autoheal/cli/domains/watchdog/register.go` | `vrooli setup --json` | `setup` |
| `autoheal-cli/diagnose-port` | `scenarios/vrooli-autoheal/cli/operations.go` | `vrooli diagnose-port 18080 example-scenario` | `diagnose-port` |
| `bridge-bootstrap/line-7` | `scenarios/vrooli-bridge/bootstrap/argv-fixture.txt` | `vrooli setup --result-file /tmp/setup-result` | `setup` |
| `bridge-bootstrap/line-9` | `scenarios/vrooli-bridge/bootstrap/argv-fixture.txt` | `vrooli setup --bootstrap-only --environment development --resources none --scenarios none --include-optional --sudo-mode ask --result-file /tmp/setup-result` | `setup` |
| `bridge-bootstrap/line-11` | `scenarios/vrooli-bridge/bootstrap/argv-fixture.txt` | `vrooli setup --environment development --resources none --scenarios none --include-optional --sudo-mode ask --credential-passphrase-stdin --result-file /tmp/setup-result` | `setup` |
| `ci/deploy:57` | `.github/workflows/deploy.yml` | `vrooli build` | `build` |
| `ci/publish-plugin:27` | `.github/workflows/publish-plugin.yml` | `vrooli scenario test scenario-to-plugin --preset comprehensive` | `scenario test` |
| `ci/test:60` | `.github/workflows/test.yml` | `vrooli scenario setup test-genie` | `scenario setup` |
| `ci/test:67` | `.github/workflows/test.yml` | `vrooli scenario setup test-genie` | `scenario setup` |
| `ci/test:162` | `.github/workflows/test.yml` | `vrooli capability conformance` | `capability` |
| `ci/test:241` | `.github/workflows/test.yml` | `vrooli scenario start structure-health --timeout 120` | `scenario start` |
| `ci/test:244` | `.github/workflows/test.yml` | `vrooli hygiene --contract-only --fail-on error --json` | `hygiene` |
| `ci/test:267` | `.github/workflows/test.yml` | `vrooli scenario start tidiness-manager --timeout 120` | `scenario start` |
| `ci/test:270` | `.github/workflows/test.yml` | `vrooli hygiene --tidiness-only --fail-on error --json` | `hygiene` |
| `ci/test:297` | `.github/workflows/test.yml` | `vrooli scenario start security-health --timeout 120` | `scenario start` |
| `ci/test:298` | `.github/workflows/test.yml` | `vrooli scenario start test-genie --timeout 120` | `scenario start` |
| `ci/test:409` | `.github/workflows/test.yml` | `vrooli help` | `help` |
| `ci/test:410` | `.github/workflows/test.yml` | `vrooli scenario --help` | `scenario` |
| `ci/test:411` | `.github/workflows/test.yml` | `vrooli scenario start hello-plugin --timeout 120 --json` | `scenario start` |
| `ci/test:412` | `.github/workflows/test.yml` | `vrooli scenario status hello-plugin --json` | `scenario status` |
| `ci/test:413` | `.github/workflows/test.yml` | `vrooli scenario stop hello-plugin --json` | `scenario stop` |
| `ci/test:469` | `.github/workflows/test.yml` | `vrooli credentials store init` | `credentials` |
| `ci/test:519` | `.github/workflows/test.yml` | `vrooli credentials doctor --format json` | `credentials` |
| `ci/test:555` | `.github/workflows/test.yml` | `vrooli credentials store init` | `credentials` |
| `ci/test:565` | `.github/workflows/test.yml` | `vrooli credentials doctor --format json` | `credentials` |
| `ci/test:619` | `.github/workflows/test.yml` | `vrooli credentials doctor --format json` | `credentials` |

## Runners

These files execute an invocation whose argv a registered owner supplies; they produce no argv of their own.

- `packages/vrooli-cli-go/client.go`
- `scenarios/vrooli-autoheal/api/internal/checks/executor.go`
- `scenarios/vrooli-autoheal/cli/loop/invoke.go`
- `scenarios/vrooli-autoheal/cli/loop/main.go`

## Scenario-level callers not yet migrated

These scenarios spawn `vrooli` with their own resolution and are outside the
registry; migrating them onto `repo-contract-go/cliinvoke` is a follow-up.

| Scenario | File:line | Call |
|---|---|---|
| `agent-manager` | `scenarios/agent-manager/api/internal/adapters/capacity/broker.go:67` | `exec.CommandContext(ctx, "vrooli", args...)` |
| `agent-manager` | `scenarios/agent-manager/api/internal/runsignal/catalog.go:412` | `exec.Command("vrooli", "help")` |
| `ai-gateway` | `scenarios/ai-gateway/api/internal/routing/capacity.go:87` | `exec.CommandContext(ctx, "vrooli", args...)` |
| `backdrop-studio` | `scenarios/backdrop-studio/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `browser-automation-studio` | `scenarios/browser-automation-studio/bas/seeds/seed.go:137` | `exec.Command("vrooli", "scenario", "port", scenario, "API_PORT")` |
| `deployment-manager` | `scenarios/deployment-manager/api/internal/capabilities/registry.go:64` | `exec.CommandContext(ctx, "vrooli", "scenario", "status", slug, "--json")` |
| `device-control` | `scenarios/device-control/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `document-manager` | `scenarios/document-manager/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `hello-mobile` | `scenarios/hello-mobile/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `hello-python` | `scenarios/hello-python/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `image-tools` | `scenarios/image-tools/api/internal/ai/capacity.go:66` | `exec.CommandContext(ctx, "vrooli", args...)` |
| `infrastructure-manager` | `scenarios/infrastructure-manager/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `money-ledger` | `scenarios/money-ledger/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `music-library` | `scenarios/music-library/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `music-tools` | `scenarios/music-tools/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `performance-health` | `scenarios/performance-health/api/internal/capture/build_controller.go:159` | `exec.CommandContext(ctx, "vrooli", "scenario", "restart", scenario)` |
| `performance-health` | `scenarios/performance-health/api/internal/startup/runner.go:167` | `exec.CommandContext(ctx, "vrooli", "scenario", "restart", scenario)` |
| `persona` | `scenarios/persona/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `program-runtime` | `scenarios/program-runtime/api/internal/capabilities/registry.go:47` | `exec.Command("vrooli", "scenario", "status", scenario, "--json")` |
| `prompt-manager` | `scenarios/prompt-manager/api/main.go:951` | `exec.CommandContext(ctx, "vrooli", "agent", "recover", "--scenario", scenario, "--reason", reason, "--requester", "prompt-manager")` |
| `react-component-library` | `scenarios/react-component-library/api/internal/capabilities/registry.go:190` | `exec.CommandContext(ctx, "vrooli", "scenario", "status", slug, "--json")` |
| `scenario-dependency-analyzer` | `scenarios/scenario-dependency-analyzer/api/internal/dependencyhealth/integration_conformance.go:132` | `exec.CommandContext(statusCtx, "vrooli", "scenario", "status", scenario, "--json")` |
| `scenario-to-android` | `scenarios/scenario-to-android/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `scenario-to-ios` | `scenarios/scenario-to-ios/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `scenario-to-plugin` | `scenarios/scenario-to-plugin/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `source-ledger` | `scenarios/source-ledger/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", c.Slug, "--json")` |
| `storage-manager` | `scenarios/storage-manager/api/handlers/storage/module.go:701` | `exec.CommandContext(ctx, "vrooli", append([]string{commandName}, commandArgs...)` |
| `structure-health` | `scenarios/structure-health/api/internal/packs/targetpack/targetpack.go:155` | `exec.Command("vrooli", "package", "dependents", id, "--json")` |
| `switchboard` | `scenarios/switchboard/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `token-economy` | `scenarios/token-economy/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `treasury` | `scenarios/treasury/api/internal/capabilities/registry.go:43` | `exec.Command("vrooli", "scenario", "status", "audio-tools", "--json")` |
| `vrooli-onboarding` | `scenarios/vrooli-onboarding/api/v2_readiness.go:119` | `exec.CommandContext(ctx, "vrooli", "release-authority", "status", "--format", "json")` |
| `vrooli-orchestrator` | `scenarios/vrooli-orchestrator/api/orchestrator.go:204` | `exec.Command("vrooli", "resource", resourceName, "start")` |
| `vrooli-orchestrator` | `scenarios/vrooli-orchestrator/api/orchestrator.go:233` | `exec.Command("vrooli", "resource", resourceName, "stop")` |
| `vrooli-orchestrator` | `scenarios/vrooli-orchestrator/api/orchestrator.go:262` | `exec.Command("vrooli", "scenario", "run", scenarioName)` |
| `vrooli-orchestrator` | `scenarios/vrooli-orchestrator/api/orchestrator.go:291` | `exec.Command("vrooli", "scenario", "stop", scenarioName)` |
| `vrooli-orchestrator` | `scenarios/vrooli-orchestrator/api/orchestrator.go:348` | `exec.Command("vrooli", "resource", "list")` |
| `vrooli-orchestrator` | `scenarios/vrooli-orchestrator/api/orchestrator.go:359` | `exec.Command("vrooli", "scenario", "list")` |
