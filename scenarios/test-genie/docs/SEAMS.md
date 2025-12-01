# Test Genie Seams

This scenario exists to erase the historical dependency on `scripts/scenarios/testing`. Every loop should harden the local boundaries that let the Go API own orchestration, while the bash harness shrinks into a compatibility shim.

## Suite Orchestrator (API)

| Seam | Purpose | Current Status | Next Steps |
|------|---------|----------------|------------|
| `SuiteOrchestrator.goPhases` | Central registry that lets the API replace bash phase scripts incrementally | ✅ Structure phase already lived here; ✅ Dependencies phase now implemented in Go so the API validates runtimes/package managers/resources without shelling out | Migrate the remaining phases (unit/integration/business/performance) and keep the bash scripts as thin adapters until they can be deleted |
| Failure instrumentation | Gives API callers a machine-readable way to understand why a phase failed | ✅ PhaseExecutionResult now includes classification/remediation/observations plus aggregated dependency failure reports | Port the remaining bash-backed phases so every failure path is classified; feed this metadata into the UI dashboards and CLI |
| `commandLookup` injection | Allows phase runners to stub OS command lookups during tests without spawning shells | ✅ Implemented while porting the dependencies phase so the orchestrator can be tested deterministically | Extend this seam to other side-effecting helpers (e.g., filesystem probes, external executors) so future phases can be unit-tested without temp scripts |

### Notes

- The Go dependencies phase intentionally mirrors the expectations from `scripts/scenarios/testing/shell/dependencies.sh`: detect which runtimes a scenario needs, verify package managers, and report required resources from `.vrooli/service.json`.
- Bash phase files still exist because the scenario’s own test suite invokes them directly. They now act as the “legacy seam”; the API is the forward-looking seam.

## Scenario Test Harness

| Seam | Purpose | Observations | Gap |
|------|---------|--------------|-----|
| `test/lib/orchestrator.sh` | Scenario-local bash orchestrator used by lifecycle `make test` | Still the entrypoint for lifecycle steps; mirrors CLI UX; provides feature parity for agents that have not switched to the API yet | Needs to call into the API (or the Go binary) once the other phases land so we only maintain logic in one place |
| Phase scripts | Concrete assertions for structure/dependencies/... | Duplicated logic between bash and Go (structure + dependencies). Bash remains the oracle for lifecycle, Go for API | After Go parity lands for each phase, turn scripts into smoke proxies (call API endpoints) or remove them entirely |

## Outstanding Opportunities

1. **Phase Registry Metadata** – The API hard-codes ordering weights. Capturing metadata (category, default timeout, dependency graph) in a declarative registry would let future scenarios extend phases without editing the orchestrator.
2. **External Command Isolation** – The CLI/business phases still shell out aggressively. Porting them will require abstractions over process execution, filesystem writes, and network connectivity. Introduce seam structs (e.g., `type Commander interface { Run(ctx, cmd, args...) }`) before attempting large migrations.
3. **Requirements Awareness** – Neither the Go nor bash orchestrators currently enforce `[REQ:ID]` tagging. Once the API can parse requirement registries it should block suites that drift from the registry layout.
