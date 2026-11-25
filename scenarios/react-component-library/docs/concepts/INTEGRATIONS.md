# Integrations — React Component Library

Behavioral claims are current only within the specific checks in the register. Other guidance and unverified descriptions below are **design intent**, not claims of current implementation. See the [behavior claim register](../internal/TESTING.md#behavior-claim-register).

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, notes reference | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| Agent Manager | scenario | no | Assisted extraction and adoption workflows | `.vrooli/service.json`, capability availability probe, generated Connect `StartWorkflowExecution`, `WaitWorkflowExecution`, `GetWorkflowExecutionResult`, and `CancelWorkflowExecution` | Assisted workflows remain unavailable; direct catalog and adoption APIs remain available. |

## Vrooli Resources

The generated template does not declare external Vrooli resources. Add
resources to `.vrooli/service.json` only when a real scenario domain
requires them.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None yet. | not-applicable | SQLite is embedded by default. | Add when PRD/requirements demand shared resource behavior. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| Agent Manager | optional | Provides server-side execution for attributable assisted extraction and adoption workflows. | Generated `agent_manager.v1.AgentManagerService` Connect procedures listed above; `vrooli scenario start agent-manager --json` |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | No external third-party service is required; Agent Manager is a Vrooli scenario dependency. | Add when a non-Vrooli service is introduced. |

## Capabilities surface (planned D · Nocturne)

The Capabilities page is the human projection of this dependency inventory. It does not add a dependency or own health; it renders each integration's state, what it enables, the last check and the recovery command. A missing service is a visible finding, not a silently skipped check. See [WORKBENCH-REDESIGN.md](WORKBENCH-REDESIGN.md#capabilities).

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| Agent Manager | lifecycle status unavailable or dispatch error | Assisted workflow is stored as unavailable; direct catalog and adoption APIs remain available. | workflow service tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness

## Provider-neutral integration presentation

`IntegrationCard@0.1.4` is the governed presentation pattern for a named
connection. It accepts provider and account labels, connection status, scopes,
bindings, usage, freshness, next-action text, and an action slot. It owns no
provider protocol, credential authority, entitlement, or network behavior.

Use `StatusBadge` for the shared connection vocabulary and pass lifecycle
controls through the `actions` slot. Consumers own authorization and mutation
callbacks, advertise only supported operations, and keep runtime dependency
health in a separate component. The released stories cover connected,
checking, attention, expired, disconnected, insufficient-scope,
provider-unavailable, offline, and unknown states.
