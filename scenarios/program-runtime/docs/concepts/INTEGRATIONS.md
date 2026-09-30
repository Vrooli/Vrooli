# Integrations — Program Runtime

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, persistence-backed domains | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| CPython + IPython | host runtime | yes | `sessions`, `programs` | Host requirement; kernel sidecar under `kernel/` | Session creation fails with a stated reason; the API stays healthy and other domains keep serving. |
| Proto descriptor image | build artifact | yes | `bindings` | `packages/proto/gen/descriptor/image.binpb` | Binding generation fails closed — no callables are produced rather than an unvalidated surface. |
| Scenario CLI manifests | repo contract | yes | `bindings`, `actspace` | `cli/manifest.json` per scenario, `.vrooli/schemas/cli-manifest.schema.json` | A scenario without a manifest contributes no bindings; this is reported, not hidden. |

## Vrooli Resources

This scenario declares no Vrooli resource. That is a real decision, not
scaffold residue: the kernel needs a CPython interpreter, which is a
**host requirement** resolved through the existing host-requirement path,
not a long-lived shared service with a port and a health check. Resources
are singletons; kernels are per-session child processes.

Every third-party package — Python or otherwise — is found and installed
through **Scenario Dependency Analyzer**. No raw `pip`, `pnpm add`, or
`go get`, and no hand-edited `.vrooli/dependencies/approved-dependencies.json`.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None | deliberate | SQLite is embedded; the kernel interpreter is a host requirement, not a resource. | A domain needs shared, cross-scenario state or a service with its own lifecycle. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `ai-gateway` | required (P1) | Typed inference (`classify`, `extract`, `judge`) inside programs. The single inference front door; this scenario never contacts a provider directly. | Connect RPC via `api-core/discovery`; usage accounting feeds the separate inference budget target OT-P1-010. |
| `agent-manager` | required (P1) | Delegated agent runs spawned from a program, and the consumer of this scenario's friction events. | Connect RPC. agent-manager must also **subscribe** to this scenario's events; it is currently the only event subscriber in the fleet. |
| `vrooli-events` | required (P0) | Typed telemetry for submissions, invocations, and failures. | `packages/api-core/eventbus`. Emission is automatic for every scenario; subscription is opt-in by the reader. |
| `search-hub` | required (P1) | In-kernel capability discovery so the callable surface is not preloaded into agent context. | Connect RPC; degrades to a stated reason. |
| `workspace-sandbox` | optional (P1) | Copy-on-write filesystem isolation for a session. | Discovery-resolved `GET /api/v1/sandboxes/{id}/workspace` today; no shared typed resolver exists yet. A resolved root is pinned as the kernel cwd. Safety from accidents, not from adversaries. |
| `meta-optimization-manager` | consumer, not a dependency | Reads the Act denominator and numerator this scenario owns. The arrow points inward: it pulls, this scenario never pushes. | `space --projection act --json` plus the binding-registry RPC. |

## Live contract snapshots

The binding registry is backed by the shared `packages/proto/descriptorimage`
source. It validates the descriptor image and CLI-manifest stamps between
requests, publishes an immutable generation, and keeps the last known-good
generation when a reload is malformed. New kernels obtain binding specs from
the current generation; an existing kernel keeps the generation it received so
an in-flight program has a stable contract. Health metadata reports the digest,
generation, load time, artifact time, and any reload error. No scenario restart
is required after a staged proto or manifest publication.

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None | deliberate | All outbound calls terminate at Vrooli-owned scenarios. Model providers are reached only through `ai-gateway`, which owns credentials and provider policy. | Revisit only if a capability cannot be expressed as a governed Vrooli operation. |

## External Obligations

Three pieces of required work sit outside this scenario's boundary. Each is
listed with an owner because in every case this scenario can reach 100% green
while the obligation stays unmet and the value stays unrealised. Mirrors
`PRD.md` §External obligations; keep the two in step.

| Obligation | Owner | Unblocks | If it stalls |
|---|---|---|---|
| Carry the live program-event subscription and delivered-event health | `agent-manager` | Friction analysis reading program evidence | Revisit if the subscription is removed or delivery health becomes unavailable |
| Raise `cli/manifest.json` coverage past 58/128 | fleet-wide; surfaced by `cli-health`, ranked by `meta-optimization-manager` `focus next` | The ceiling on the whole Act surface | Act coverage caps near 45% of the fleet however complete this scenario is |

None of the three blocks launch. The first two bound the realised value of
work that is otherwise complete; the third bounds a number this scenario
reports honestly rather than one it can raise.

## Identity Propagation — Known Gap

Agent identity reaches event receipts today because agent-manager sets
identity environment variables at spawn, and the shared Go packages
(`cli-core`, `api-core`) carry that identity into receipts with no
per-scenario work.

**That inheritance does not reach this scenario's kernel.** The chain
`agent → program-runtime → program → ai-gateway` crosses two hops it does
not traverse today, and the hard part is structural: the kernel is a
**non-Go sidecar**, so it cannot inherit propagation that lives in Go
shared packages. Identity must be carried explicitly across the sidecar
boundary and re-attached to outbound inference calls.

This is deliberately unsolved at design time and is **not a launch
blocker**. Bindings, handles, governance, and program-failure telemetry
are all correct without it. What it blocks is trustworthy attribution of
*in-program inference* back to the originating agent. Resolve it before
treating in-program inference telemetry as evidence.

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| CPython kernel | spawn or handshake failure | Session creation fails with a stated reason; existing sessions are unaffected; the API stays healthy. | `sessions` lifecycle tests |
| Descriptor image | missing or unparsable | Binding generation fails closed and reports why. No partial or unvalidated callable surface is exposed. | `bindings` registry tests |
| `ai-gateway` | unreachable or timeout | Inference callables raise inside the program with a stated reason. Non-inference bindings keep working. | `programs` inference tests |
| `agent-manager` | unreachable | Delegation callables raise; the rest of the program continues. | `programs` delegation tests |
| `search-hub` | unreachable | Discovery degrades to a stated reason; direct binding calls are unaffected. | `bindings` discovery tests |
| `vrooli-events` | unreachable | Emission is fire-and-forget; program execution never blocks on telemetry. | `telemetry` tests |
| `workspace-sandbox` | unreachable or workspace identifier unknown | Sessions without a workspace use private scratch storage. Declared identifiers are rejected; an explicit absolute path may use local validation only and is not copy-on-write isolated. | `sessions` workspace resolver tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
