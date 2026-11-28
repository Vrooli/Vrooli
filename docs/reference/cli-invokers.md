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
| `lighting/resource-permission-apply` | `internal/resources/openrgb_device_setup.go` | `vrooli runtime lighting permissions apply --plan-id aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --sudo-mode error --json` | `runtime` |
| `lighting/resource-permission-rollback` | `internal/resources/openrgb_device_setup.go` | `vrooli runtime lighting permissions rollback --plan-id aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --sudo-mode error --json` | `runtime` |
| `lighting/devices` | `scenarios/lighting-control/api/internal/setup/cli.go` | `vrooli runtime lighting devices --json` | `runtime` |
| `lighting/permission-plan` | `scenarios/lighting-control/api/internal/setup/cli.go` | `vrooli runtime lighting permissions plan --device-id device-id --json` | `runtime` |
| `lighting/permission-apply` | `scenarios/lighting-control/api/internal/setup/cli.go` | `vrooli runtime lighting permissions apply --plan-id aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --sudo-mode error --json` | `runtime` |
| `lighting/permission-rollback` | `scenarios/lighting-control/api/internal/setup/cli.go` | `vrooli runtime lighting permissions rollback --plan-id aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --sudo-mode error --json` | `runtime` |
| `lighting-native/start` | `internal/app/runtime/lighting_native.go` | `vrooli scenario start lighting-control` | `scenario start` |
| `lighting-native/ensure-started` | `internal/lightingruntime/native_scenario.go` | `vrooli scenario start lighting-control` | `scenario start` |
| `lighting/prepare` | `internal/app/runtime/lighting_native_prepare.go` | `vrooli runtime lighting prepare --json` | `runtime` |
| `lighting/status` | `scenarios/lighting-control/api/internal/setup/cli.go` | `vrooli runtime lighting status --json` | `runtime` |
| `lighting/disable` | `scenarios/lighting-control/api/internal/setup/cli.go` | `vrooli runtime lighting disable --json` | `runtime` |
| `lighting/setup` | `scenarios/lighting-control/api/internal/setup/cli.go` | `vrooli runtime lighting setup --source managed --version 1.1 --sha256 aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --device-policy-id policy --json` | `runtime` |
| `lighting/enable` | `scenarios/lighting-control/api/internal/setup/cli.go` | `vrooli runtime lighting enable --source managed --version 1.1 --sha256 aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --device-policy-id policy --json` | `runtime` |
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
| `ci/test:68` | `.github/workflows/test.yml` | `vrooli scenario setup test-genie` | `scenario setup` |
| `ci/test:98` | `.github/workflows/test.yml` | `vrooli scenario setup test-genie` | `scenario setup` |
| `ci/test:186` | `.github/workflows/test.yml` | `vrooli scenario setup typescript-code-graph` | `scenario setup` |
| `ci/test:229` | `.github/workflows/test.yml` | `vrooli capability conformance` | `capability` |
| `ci/test:308` | `.github/workflows/test.yml` | `vrooli scenario start structure-health --timeout 120` | `scenario start` |
| `ci/test:311` | `.github/workflows/test.yml` | `vrooli hygiene --contract-only --fail-on error --json` | `hygiene` |
| `ci/test:340` | `.github/workflows/test.yml` | `vrooli scenario start tidiness-manager --timeout 120` | `scenario start` |
| `ci/test:343` | `.github/workflows/test.yml` | `vrooli hygiene --tidiness-only --fail-on error --json` | `hygiene` |
| `ci/test:370` | `.github/workflows/test.yml` | `vrooli scenario start security-health --timeout 120` | `scenario start` |
| `ci/test:371` | `.github/workflows/test.yml` | `vrooli scenario start test-genie --timeout 120` | `scenario start` |
| `ci/test:482` | `.github/workflows/test.yml` | `vrooli help` | `help` |
| `ci/test:483` | `.github/workflows/test.yml` | `vrooli scenario --help` | `scenario` |
| `ci/test:484` | `.github/workflows/test.yml` | `vrooli scenario start hello-plugin --timeout 120 --json` | `scenario start` |
| `ci/test:485` | `.github/workflows/test.yml` | `vrooli scenario status hello-plugin --json` | `scenario status` |
| `ci/test:486` | `.github/workflows/test.yml` | `vrooli scenario stop hello-plugin --json` | `scenario stop` |
| `ci/test:542` | `.github/workflows/test.yml` | `vrooli credentials store init` | `credentials` |
| `ci/test:592` | `.github/workflows/test.yml` | `vrooli credentials doctor --format json` | `credentials` |
| `ci/test:628` | `.github/workflows/test.yml` | `vrooli credentials store init` | `credentials` |
| `ci/test:638` | `.github/workflows/test.yml` | `vrooli credentials doctor --format json` | `credentials` |
| `ci/test:692` | `.github/workflows/test.yml` | `vrooli credentials doctor --format json` | `credentials` |

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
| `agent-manager` | `scenarios/agent-manager/api/internal/maintenance/attached_identity.go:41` | `exec.CommandContext(ctx, "vrooli", args...)` |
| `agent-manager` | `scenarios/agent-manager/api/internal/maintenance/executor_scope.go:412` | `exec.CommandContext(ctx, "vrooli", "runtime", "executor-scope", "--json")` |
| `agent-manager` | `scenarios/agent-manager/api/internal/orchestration/identity_renewal.go:205` | `exec.CommandContext(ctx, "vrooli", args...)` |
| `ai-gateway` | `scenarios/ai-gateway/api/internal/routing/capacity.go:90` | `exec.CommandContext(ctx, "vrooli", args...)` |
| `browser-automation-studio` | `scenarios/browser-automation-studio/bas/seeds/seed.go:137` | `exec.Command("vrooli", "scenario", "port", scenario, "API_PORT")` |
| `device-control` | `scenarios/device-control/api/internal/native/x11/session.go:30` | `exec.CommandContext(ctx, "vrooli", "host", "desktop-session", "--session-id", sessionID, "--peer-pid", strconv.Itoa(peer.PID)` |
| `integration-hub` | `scenarios/integration-hub/api/credential_cli.go:16` | `exec.CommandContext(ctx, "vrooli", "credentials", "provision", "--identity", request.Identity, "--field", request.Field)` |
| `integration-hub` | `scenarios/integration-hub/api/credential_cli.go:25` | `exec.CommandContext(ctx, "vrooli", "credentials", "status", "--identity", identity, "--field", field, "--format", "json")` |
| `integration-hub` | `scenarios/integration-hub/api/credential_cli.go:38` | `exec.CommandContext(ctx, "vrooli", "credentials", "delete", "--identity", identity, "--field", field, "--yes")` |
| `performance-health` | `scenarios/performance-health/api/internal/capture/build_controller.go:162` | `exec.CommandContext(ctx, "vrooli", "scenario", "restart", scenario)` |
| `performance-health` | `scenarios/performance-health/api/internal/startup/runner.go:167` | `exec.CommandContext(ctx, "vrooli", "scenario", "restart", scenario)` |
| `portal` | `scenarios/portal/api/internal/surfaces/applications.go:35` | `exec.CommandContext(ctx, "vrooli", "scenario", "list", "--json")` |
| `portal` | `scenarios/portal/api/internal/surfaces/applications.go:37` | `exec.CommandContext(ctx, "vrooli", "scenario", action, name, "--json")` |
| `react-component-library` | `scenarios/react-component-library/cli/domains/page/capture.go:380` | `exec.CommandContext(ctx, "vrooli", "scenario", "status", name, "--json")` |
| `scenario-authenticator` | `scenarios/scenario-authenticator/cli/domains/serviceprincipals/host.go:81` | `exec.CommandContext(ctx, "vrooli", args...)` |
| `scenario-dependency-analyzer` | `scenarios/scenario-dependency-analyzer/api/internal/dependencyhealth/integration_conformance.go:132` | `exec.CommandContext(statusCtx, "vrooli", "scenario", "status", scenario, "--json")` |
| `structure-health` | `scenarios/structure-health/api/internal/packs/targetpack/targetpack.go:155` | `exec.Command("vrooli", "package", "dependents", id, "--json")` |
| `test-genie` | `scenarios/test-genie/api/internal/validationbroker/identity.go:174` | `exec.CommandContext(ctx, "vrooli", "scenario", "freshness", scenario, "--inputs", "--json")` |
| `vrooli-onboarding` | `scenarios/vrooli-onboarding/api/readiness.go:263` | `exec.CommandContext(ctx, "vrooli", "release-authority", "status", "--format", "json")` |
| `vrooli-orchestrator` | `scenarios/vrooli-orchestrator/api/orchestrator.go:204` | `exec.Command("vrooli", "resource", resourceName, "start")` |
| `vrooli-orchestrator` | `scenarios/vrooli-orchestrator/api/orchestrator.go:233` | `exec.Command("vrooli", "resource", resourceName, "stop")` |
| `vrooli-orchestrator` | `scenarios/vrooli-orchestrator/api/orchestrator.go:262` | `exec.Command("vrooli", "scenario", "run", scenarioName)` |
| `vrooli-orchestrator` | `scenarios/vrooli-orchestrator/api/orchestrator.go:291` | `exec.Command("vrooli", "scenario", "stop", scenarioName)` |
| `vrooli-orchestrator` | `scenarios/vrooli-orchestrator/api/orchestrator.go:348` | `exec.Command("vrooli", "resource", "list")` |
| `vrooli-orchestrator` | `scenarios/vrooli-orchestrator/api/orchestrator.go:359` | `exec.Command("vrooli", "scenario", "list")` |
| `workflow-health` | `scenarios/workflow-health/api/handlers/validation/evidence.go:74` | `exec.CommandContext(ctx, "vrooli", "scenario", "freshness", scenario, "--json")` |
