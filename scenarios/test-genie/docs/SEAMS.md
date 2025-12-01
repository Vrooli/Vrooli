# Test Genie Seams

This scenario exists to erase the historical dependency on `scripts/scenarios/testing`. Every loop should harden the local boundaries that let the Go API own orchestration, while the bash harness shrinks into a compatibility shim.

## Suite Orchestrator (API)

| Seam | Purpose | Current Status | Next Steps |
|------|---------|----------------|------------|
| `SuiteOrchestrator.goPhases` | Central registry that lets the API replace bash phase scripts incrementally | ✅ Structure phase already lived here; ✅ Dependencies phase now implemented in Go so the API validates runtimes/package managers/resources without shelling out | Migrate the remaining phases (unit/integration/business/performance) and keep the bash scripts as thin adapters until they can be deleted |
| Failure instrumentation | Gives API callers a machine-readable way to understand why a phase failed | ✅ PhaseExecutionResult now includes classification/remediation/observations plus aggregated dependency failure reports | Port the remaining bash-backed phases so every failure path is classified; feed this metadata into the UI dashboards and CLI |
| `commandLookup` injection | Allows phase runners to stub OS command lookups during tests without spawning shells | ✅ Implemented while porting the dependencies phase so the orchestrator can be tested deterministically; new exported override helpers let both Go unit tests and orchestrator HTTP tests substitute fake executors without reaching into package internals | Extend this seam to other side-effecting helpers (e.g., filesystem probes, external executors) so future phases can be unit-tested without temp scripts |
| `internal/orchestrator/{phases,workspace,requirements}` packages | Splits the monolithic `suite` package into explicit seams for runners, filesystem manifests, and requirements syncing so orchestration logic can evolve independently of queue/persistence code | ✅ All phase runners now live in `phases`, filesystem + config handling moved to `workspace`, and the Node requirements bridge lives in `requirements`; `suite` now consumes the orchestrator engine through interfaces | Rewire the CLI/bash shims to consume the Go orchestrator via these packages so we can delete duplicate logic in `scripts/scenarios/testing` |

### Notes

- The Go dependencies phase intentionally mirrors the expectations from `scripts/scenarios/testing/shell/dependencies.sh`: detect which runtimes a scenario needs, verify package managers, and report required resources from `.vrooli/service.json`.
- Bash phase files still exist because the scenario’s own test suite invokes them directly. They now act as the “legacy seam”; the API is the forward-looking seam.

## Scenario Test Harness

| Seam | Purpose | Observations | Gap |
|------|---------|--------------|-----|
| `test/lib/orchestrator.sh` | Scenario-local bash orchestrator used by lifecycle `make test` | Still the entrypoint for lifecycle steps; mirrors CLI UX; provides feature parity for agents that have not switched to the API yet | Needs to call into the API (or the Go binary) once the other phases land so we only maintain logic in one place |
| Phase scripts | Concrete assertions for structure/dependencies/... | Duplicated logic between bash and Go (structure + dependencies). Bash remains the oracle for lifecycle, Go for API | After Go parity lands for each phase, turn scripts into smoke proxies (call API endpoints) or remove them entirely |

## API Runtime & HTTP Surface

| Seam | Purpose | Current Status | Next Steps |
|------|---------|----------------|------------|
| `runtime.LoadConfig` | Encapsulates lifecycle env parsing (ports, DB URL, scenarios root) so HTTP handlers never chase globals or stringly-typed paths | ✅ Lives in `internal/app/runtime` now; config resolution is completely isolated from the HTTP package and can be swapped for fixtures in tests | Accept overrides from CLI/test flags so we can spin up the server against temp DBs without exporting fake env vars |
| `runtime.BuildDependencies` | Single bootstrap point for DB connections, schema migration, orchestrator wiring, and scenario directory repos | ✅ New runtime package returns a `Bootstrapped` struct so CLI/worker processes can reuse the same wiring without importing HTTP code | Split orchestration/bootstrap so the CLI can reuse the same seams when it migrates off `scripts/scenarios/testing` |
| `httpserver.Dependencies` (`suiteRequestQueue` / `suite.ExecutionHistory` / `suiteExecutor` / `scenarioDirectory` / `phaseCatalog`) | Keeps the HTTP layer transport-focused and lets tests/integration harnesses swap concrete implementations without rewriting handlers | ✅ Server construction moved into `internal/app/httpserver`; tests now live beside the transport package and inject no-op loggers or fake repos as needed | Introduce adapters that proxy to the legacy bash runner so CLI consumers can switch to HTTP without waiting for every phase rewrite |
| `httpserver.Logger` | Lets transport code write telemetry without binding to the global `log` package (critical when sharing the runner inside other processes) | ✅ Server defaults to `log.Default()` but accepts injected loggers so tests and future embedders can silence or redirect transport logging | Feed structured logs into the UI and CLI so queue/runner events show up without tailing files |
| `suite.ExecutionHistoryService` | Converts DB-backed execution records into orchestrator payloads so HTTP/CLI callers never read persistence structs directly | ✅ HTTP now depends on the `ExecutionHistory` interface instead of the repository, and new tests stub the interface without touching sqlmock | Reuse the same seam when the CLI embeds the runner so bash components never need to import database types |

## Outstanding Opportunities

1. **Phase Registry Metadata** – The API hard-codes ordering weights. Capturing metadata (category, default timeout, dependency graph) in a declarative registry would let future scenarios extend phases without editing the orchestrator.
2. **External Command Isolation** – The CLI/business phases still shell out aggressively. Porting them will require abstractions over process execution, filesystem writes, and network connectivity. Introduce seam structs (e.g., `type Commander interface { Run(ctx, cmd, args...) }`) before attempting large migrations.
3. **Requirements Awareness** – Neither the Go nor bash orchestrators currently enforce `[REQ:ID]` tagging. Once the API can parse requirement registries it should block suites that drift from the registry layout.
