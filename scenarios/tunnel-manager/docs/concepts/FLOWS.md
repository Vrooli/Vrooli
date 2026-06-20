# Flows — Tunnel Manager

This document is the canonical workflow and state-transition map for
the scenario. Use it when behavior depends on ordered states, retries,
cancellation, stale completion, background jobs, polling, or mutually
exclusive UI modes.

## Purpose Of This Document

Use this document to answer:

- Which user/system workflows matter?
- Which workflows have explicit states and events?
- Which transitions are illegal?
- Which tests prove workflow correctness?
- Which flows are known but not modeled yet?

Plain CRUD with no meaningful ordering constraints does not need a
workflow model.

## Flow Inventory

`health` is a stateless reporting domain and ships no workflows. List
each real stateful flow your domains add below, with its owner, trigger,
outcome, statefulness, and validation level.

> Status: product implementation is in progress. Exposure lease
> creation/revocation, CORE reconcile, lease reaping, route probes, and
> recovery evaluation are implemented in their owning domains. Exposure
> and probes run at boot + periodic ticks by default; recovery evaluation
> uses the same scheduler shape but is opt-in with
> `TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED` because it can restart
> cloudflared. These flows do not yet have `flow/` contracts or Quint
> models; target validation levels remain the desired next maturity step.
> Owning domains are authoritative per
> [`DOMAINS.md`](DOMAINS.md).

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Expose-on-demand (lease) | `exposure` | API/CLI/operator/another scenario requests exposure with a TTL. | Scenario reachable at a public URL; LEASED route + lease persisted. | Ordered: ensure route → ensure running → push ingress → probe; partial-failure rollback. | Target Level 4–5 (ordered, externally-effecting). |
| Core reconcile | `exposure` | Boot + periodic tick + manual RPC. | Every `api-core/coreset` scenario is a CORE route, always exposed, never reaped. | Idempotent converge: add missing CORE routes/ingress, never auto-expire CORE. | Implemented with service tests + scheduler tests; target Level 3 model remains. |
| Lease expiry / reap | `exposure` | Scheduler tick finds `expires_at` < now, or manual `Reconcile`. | Expired LEASED routes + ingress removed unless also CORE; lease marked expired. | Time-driven; skip-if-CORE guard; safe re-run. | Implemented with service tests + scheduler tests; target Level 3 model remains. |
| Probe cycle | `probes` | Boot + periodic tick + manual RPC. | Internal/external reachability results are persisted for each enabled route. | Concurrent bounded HTTP probes; best-effort per-result persistence; safe re-run. | Implemented with service tests + scheduler tests; target Level 3 model remains. |
| Auto-recovery evaluation | `recovery` | Opt-in boot + periodic tick (`TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED`) + manual RPC. | cloudflared restarted after thresholded `/ready` failures; recovery event logged. | Backoff + circuit breaker; states open/closed (half-open planned); single-owner restart. | Implemented with policy tests + scheduler tests; target Level 5 model remains. |
| app-monitor exposure-query | `exposure` | app-monitor "open in new tab" calls `IsExposed` / `ExposeAndGetURL`. | Tunnel URL returned (creating a short lease if needed). | Read-or-create; reuses Expose-on-demand when not yet exposed. | Target Level 3 (consumer change is a separate task). |

## Flow Details

Document each real flow here with its owner domain, trigger, inputs,
ordered steps, outputs, failure modes, retry/cancel behavior, tests, and
generated subpackages. The worked example below shows the expected shape.

### Expose-on-demand (lease)

- Owner domain: `exposure` (delegates to `routes`, `internal/lifecycle`,
  `config`, `probes`).
- Trigger: `Expose(scenario, ttl)` from API/CLI/operator/another scenario.
- Inputs: scenario name, TTL, requested-by.

```
request Expose(scenario, ttl)
  │
  ▼
[1] ensure route exists in manifest (routes) — LEASED tier, fixed UI port
  │
  ▼
[2] ensure scenario is running (delegate → internal/lifecycle ensure-running)
  │
  ▼
[3] push ingress (config: remote = Cloudflare API v4 hot-reload, local = config.yml + restart)
  │
  ▼
[4] persist lease (exposure: leases row, expires_at = now + ttl)
  │
  ▼
[5] probe internal + external (probes) → confirm reachable
  │
  ▼
return public_url + lease
```

- Outputs: `public_url`, lease record; or typed error.
- Failure modes: scenario won't start (lifecycle), ingress push rejected
  (Cloudflare API / config), probe never goes green (scenario-down vs
  tunnel-down — classified by `probes`).
- Retry/cancel: each step is idempotent so a failed Expose can be retried;
  on irrecoverable failure, roll back the LEASED route/ingress created in
  this attempt (CORE routes are never touched).

### Core reconcile

- Owner domain: `exposure`.
- Trigger: API boot, periodic scheduler tick, or manual
  `ExposureService.Reconcile`. Coreset changes converge on the next tick
  or manual reconcile.

```
load core set (api-core/coreset)
  │
  ▼
for each core scenario:
    ensure CORE route exists (routes) and ingress present (config)
  │
  ▼
for each CORE route whose scenario left the coreset:
    demote/remove (only if not separately leased)
  │
  ▼
guarantee: no CORE route is ever auto-expired
```

- Outputs: converged manifest + ingress for the core tier.
- Failure modes: coreset seam unavailable (skip-and-alert, do not tear down
  existing CORE routes); Cloudflare API error (retry next tick).
- Retry/cancel: fully idempotent; safe to run on every tick.

### Lease expiry / reap

- Owner domain: `exposure`.
- Trigger: scheduler tick or manual `ExposureService.Reconcile`.

```
for each lease where expires_at < now and status = active:
    if scenario is ALSO in core set → keep route, mark lease expired (no teardown)
    else → remove ingress (config) + remove LEASED route (routes), mark lease expired
```

- Outputs: expired leases reaped; budget freed.
- Failure modes: ingress removal fails (leave lease active, retry next tick
  — never orphan a route without its ingress).
- Retry/cancel: idempotent; skip-if-CORE guard prevents reaping always-on
  scenarios.

### Probe cycle

- Owner domain: `probes` (reads `routes`, writes probe history).
- Trigger: API boot, periodic scheduler tick, or manual
  `ProbesService.RunProbes`.
- Inputs: enabled routes from the exposure manifest.

```
load enabled routes
  │
  ▼
for each route:
    GET http://localhost:<port><health_path>
    GET https://<subdomain>.<domain><health_path>
  │
  ▼
persist internal + external results
  │
  ▼
classify latest internal/external pairs
```

- Outputs: probe history rows and route classifications.
- Failure modes: route manifest read failure aborts the cycle; individual
  probe transport/status failures persist as probe results; individual
  persistence failures are best-effort and do not fail the full cycle.
- Retry/cancel: scheduler retries on the next tick; HTTP requests receive
  context cancellation from API shutdown.

### Auto-recovery evaluation

- Owner domain: `recovery` (reads `tunnel`/`probes`; single authoritative
  owner of cloudflared restart).
- Trigger: manual `Recover`, or background `Evaluate` when
  `TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED` is set. The current
  automatic signal is `/ready`; HA-connection driven evaluation is
  deferred until the tunnel metrics scheduler feeds that state.

```
detect (recovery: /ready; future: tunnel HA connections + probes)
  │
  ▼
classify (probes): healthy | tunnel-down | scenario-down | config-drift
  │
  ▼
if circuit breaker OPEN → alert only, do not act
else select action by class:
    tunnel-down / HA=0   → restart cloudflared (systemctl)
    config-drift         → re-push ingress (config Sync)
    scenario-down        → defer to exposure/lifecycle, not a tunnel restart
  │
  ▼
apply with exponential backoff; record recovery_events row (trigger, action, outcome)
  │
  ▼
re-probe → success closes breaker; repeated failure opens breaker (alert-only)
```

- Outputs: restored tunnel/ingress + a `recovery_events` audit row.
- Failure modes: repeated failures trip the circuit breaker into alert-only
  to avoid restart storms on foundational infra (explicit operator
  decision, see [`../internal/DECISIONS.md`](../internal/DECISIONS.md)).
- Retry/cancel: backoff between attempts; breaker bounds total attempts.
  `vrooli-autoheal` must be alert-only so the two never duel.

### app-monitor exposure-query

- Owner domain: `exposure` (the app-monitor-side consumer change is a
  separate task, not this scenario).
- Trigger: app-monitor "open in new tab" calls `IsExposed(scenario)` or
  `ExposeAndGetURL(scenario)`.

```
IsExposed(scenario)?
  ├─ yes → return existing public_url
  └─ no  → ExposeAndGetURL: run Expose-on-demand (short lease) → return new public_url
```

- Outputs: a tunnel URL the new tab opens.
- Failure modes: same as Expose-on-demand when a new lease is needed.
- Retry/cancel: caller may retry; reuses the existing route if already
  exposed (no duplicate route).

## State Machines

List each modeled flow's states, illegal transitions, and how they are
enforced. Plain CRUD with no ordering constraints does not appear here.

> No `*.flow.json` contracts or Quint models exist yet. The exposure
> service currently enforces the rules in Go with unit tests; formal flow
> contracts remain a maturity follow-up.

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| `exposure` / Expose-on-demand | requested, route_ensured, running_ensured, ingress_pushed, lease_active, probed_ok, failed | ingress before route, lease_active before ingress, probe before ingress, terminal-state escape | Implemented in `api/internal/exposure` with service, module, and scheduler tests; formal flow contract remains a maturity follow-up. |
| `exposure` / lease lifecycle | active, extended, expired, revoked | extend after expired/revoked, reap a CORE-backed scenario's route, resurrect a revoked lease | Implemented in `api/internal/exposure` service/repository tests; formal flow contract remains a maturity follow-up. |
| `recovery` / circuit breaker | closed, open | act while open unless forced; restart before threshold; restart during backoff window | Implemented in `api/internal/recovery` with service + scheduler tests; formal flow contract remains planned. |

## Maturity Ladder

Temporal workflows mature in layers. Do not skip the executable layers
to add a standalone formal document.

| Level | Name | What exists |
|---|---|---|
| 0 | Unmodeled risk | Lifecycle behavior exists only inside handlers, components, callbacks, or jobs. |
| 1 | Inventory | The flow is listed here with owner, source links, risk, and next step. |
| 2 | Workflow model | State/status values, event values, `Transition`, and `CheckInvariants` live beside the owning domain or feature. |
| 3 | Matrix + traces | Tests cover every state/event pair and replay representative traces against production transition logic. |
| 4 | Declarative contract | A domain-local `*.flow.json` declares states, events, transitions, invariants, and named traces. |
| 5 | Checked formal model | Quint/TLA+ or an equivalent tool is generated from the contract, checked, and replayed by production tests. |

## Production Shape

Three (Go) or four (UI) files per flow at the top of the feature folder,
plus one `generated/` sibling. Everything in `generated/` is codegen output.

Every flow lives in a `flow/` subdirectory next to its consumer with
conventional file names. API domains that own durable lifecycle state use:

```text
api/internal/<domain>/
  flow/
    flow.json                   # hand: source of truth (schema v6)
    transition.go               # hand: wrapper (package flow)
    flow_test.go                # hand: thin replay delegation (package flow)
    generated/
      model.qnt
      artifact.json
      runtime.go                # package generated
      replay.go
```

UI features that own client-side modes use:

```text
ui/src/features/<domain>/
  flow/
    flow.json                   # hand: source of truth (schema v6)
    transition.ts               # hand: wrapper
    fixtures.ts                 # hand: replay fixtures
    flow.test.ts                # hand: thin replay delegation
    generated/
      model.qnt
      artifact.json
      runtime.ts
      replay.helper.ts
```

Every flow uses the same file names. The `flow/` directory IS the unit;
the contract no longer declares any output paths or module names.

The workflow owns state/status values, events, `Transition`, and
`CheckInvariants`. It should be pure or nearly pure. Effects live
outside the workflow behind seams: repositories, BlobStore, clocks,
timers, HTTP clients, or UI API modules.

The `*.flow.json` contract is the source of truth. Level 5 generated
Quint models, formal artifacts, and Go/TypeScript declarations are
checked-in source artifacts for reviewability, but they are refreshed
and checked by the `flow-verifier` scenario CLI; the
scenario lifecycle runs `make temporal-models` (which calls
`flow-verifier verify check`) before the normal test
suite. A Quint file by itself is not accepted: the model must typecheck,
test, verify named invariants, emit deterministic artifacts, and those
artifacts must replay against the production Go/TypeScript transition
functions.

The generated declarations keep state/event topology and formal
freshness metadata out of hand-maintained test lists. They also provide
pure status-transition helpers generated from the `*.flow.json`
transition matrix. For TypeScript flows, the same declarations can own
the discriminated state/event union shape and replay fixture contract.
Production workflow wrappers call those helpers for abstract validity
and next-status outcomes, while keeping payload validation, side-effect
orchestration, and rich state construction in hand-authored code. API
replay tests get expected paths, hashes, invariants, and generated checks
from `generated/<folder>/runtime.go`; UI replay tests import the same metadata
from `generated/<folder>/runtime.ts`. The generated `replay.{go,helper.ts}`
files own the assertion calls; the hand-authored top-level test simply binds
the wrapper's transition function and the fixtures and invokes
`RunReplay`/`runFormalReplay` once.

Formal artifacts use schema v6 coverage metadata. Matrix completeness,
terminal transition checks, named trace coverage, and generated MBT trace
coverage are separate fields. Do not treat generated trace
`allPairsCovered` as required proof of correctness; replay tests require
the complete transition matrix and named traces, while generated trace
coverage reports how much the model explorer happened to visit.

Schema v6 `flow.json` files carry no path or module information. The
`replay` block declares only `transition.function` (plus
`transition.statusAccessor` for TS or `transition.stateType` /
`transition.statusField` for Go). Everything else is derived from the
flow directory.

Go flows emit `flow/generated/replay.go` and require a hand-authored
`flow/flow_test.go` (package `flow`) that calls `generated.RunReplay`.
TypeScript flows emit `flow/generated/replay.helper.ts` and require a
hand-authored `flow/flow.test.ts` that calls
`runFormalReplay({ transition, fixtures })` at module top level.
`flow-verifier verify check` byte-compares every generated file and runs an
AST-level lint over the hand-authored test, so a silent bypass — missing
import, stubbed transition, or call buried inside a guarded block —
fails the check.

To scaffold a new flow:

```bash
flow-verifier flows new ui/src/features/<feature> --flow-id <flow-id> --lang ts --root .
flow-verifier flows new api/internal/<domain>     --flow-id <flow-id> --lang go --root .
```

The scaffold writes the hand-authored files and immediately runs
`generate`, so `check` is green from the moment it returns.

To add or rename a state/event:

1. Edit the owning `*.flow.json`.
2. Regenerate that flow with `flow-verifier verify run --flow <flow-id>`.
3. Update only payload-specific wrapper branches that need new runtime
   data; the abstract transition table is generated.
4. Update the UI replay fixture module. The generated formal replay fixture
   interface should make missing state/event fixtures a type error.
5. Run `make temporal-models` and the scenario tests.

## Deferred / Unmodeled Flows

| Flow | Risk | Next Step |
|---|---|---|
| Exposure/recovery formal models | Runtime code and unit tests enforce the rules, but no executable `flow/` contracts or Quint models exist yet. | Scaffold `flow/` contracts for Expose-on-demand, lease lifecycle, and the recovery circuit breaker if those workflows gain additional ordering risk or regressions. |
| Mode switch (remote↔local) | Implemented and covered by service/API tests, but not yet promoted to a formal temporal model. | Model only if mode migration grows additional ordering risk beyond the current sync/switch service tests. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
