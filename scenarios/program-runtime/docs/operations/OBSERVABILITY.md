# Observability — Program Runtime

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us users are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `/health` status | health | API | API and dependency reachability | healthy for local development |
| UI health endpoint | health | UI server | UI bundle/server reachability | responds during lifecycle health check |
| test-genie result | validation | `make test` | scenario correctness evidence | all required phases pass |
| Typed inference usage | usage | ai-gateway `Usage` | enforce and explain per-session inference budgets | `cost_micros`, input tokens, and output tokens remain within configured ceilings |
| Delegated-run spend | usage | agent-manager run receipts | enforce and explain the separate delegated-run budget | delegated usage remains within its configured ceiling |
| Binding reachability | diagnostic | `program-runtime bindings doctor` | distinguish absent manifests from a stopped owner | unreachable count is investigated, not treated as unbound |
| Binding exercise | empirical | `program-runtime bindings condition --json` | report local runtime use and fleet-wide serving receipts without conflating their populations | inspect `ledger_exercise` and `receipt_exercise` separately; never add them |
| Friction evidence | empirical | `programs mine`, `mine-refusals`, `mine-unresolved` | feed recurring runtime failures to meta-optimization-manager | durable shapes have a recent locator |

### Binding exercise bases

`bindings condition` reports two independent instrumentation summaries:

- `ledger_exercise` has basis `local_invocation_ledger`. It counts bindings
  Program Runtime attempted through its governed execution path, including
  operator sweeps. This is the repeatable local exercise signal.
- `receipt_exercise` has basis `fleet_receipt_aggregate`. It counts bindings
  matched to durable receipts served across the fleet, including calls made
  outside Program Runtime.

The populations overlap and neither contains the other, so they must never be
summed. Each summary carries its own instrumented-binding count, total-binding
denominator, and invocation count. Per-binding condition verdicts use the
receipt basis and name that basis in their typed `exercise` field. The
deprecated top-level `instrumented_bindings` field remains only as a
receipt-basis compatibility alias for older clients.

`bindings sweep --dry-run --json` shows the read-effect population without
calling it; `bindings sweep --execute --json` invokes only reachable,
argument-populatable read bindings and records `PROVENANCE_OPERATOR` with
program id `sweep`. That provenance contributes to ledger exercise but is
excluded from the default friction-mining corpus.

### Durable declared-program accounting — implementation target

Qualification consumes runtime-owned usage, never a program's stdout claim.
The existing persisted `Program` receipt must retain typed token and marginal
charge totals, accounting completeness and charge-measurement status before its
dedicated session is reclaimed. Reuse the existing program repository and
session/invocation/delegation owners; do not create another usage ledger, scheduler
or lifecycle. AM's qualification owner consumes this receipt and retains its
existing candidate, program, request and caller identity checks.

Settle actual executed work, including nested libraries, inference and delegated
children, exactly once. A dedicated declared-program session may supply its full
totals; a reused session must not charge previous or concurrent submissions to
this program. An explicit owner-observed local execution with no metered work can
have measured zero. Missing, unpriced, pending or interrupted work stays unknown;
contract budgets, empty numeric defaults and caller-authored `usage` are not proof.
Preserve explicit charge-presence/basis evidence where the existing accounting
path currently discards it. Do not count nested totals twice or settle while a
child may still run. Retained usage must survive session reclamation and API
restart, without replaying effects or allocating a replacement session.

The scoped regression gate covers local measured zero; measured inference and
delegation; missing/unpriced charge; nested attribution; unfinished children;
reused-session isolation; persistence failure and reclaimed/restarted reads.
AM must reject forged stdout usage and accept only the matching runtime-owned
receipt. Exercise production wiring, not only manually supplied test callbacks.
Use existing storage/schema governance for data-preserving adoption; no runtime
migration shim. Source proof is not live qualification or campaign activation.

### Governed terminal waits — implementation target

The exact `test-genie/validation/wait` binding needs a server-owned long-wait
profile in the existing bridge and kernel invocation seams. Ordinary calls keep
their 90-second bridge/100-second kernel bounds. The owner-wait profile uses a
12-minute bridge ceiling and 13-minute kernel ceiling, both within PRT's existing
15-minute server write ceiling and TG's 20-minute ceiling. Derive these values in
the existing budgets package and its kernel envelope; do not duplicate literals,
accept arbitrary caller timeout overrides, or raise global/default limits.

The BAS composition uses at most an 11-minute terminal wait per retained receipt:
the declared producer can run for 10 minutes, so observing it needs terminalization
headroom. Compatible admitted owner budgets (including queue and cleanup) and
async execution must fit the declared wall allowance. Validate those bounds before
effects; the original AM/PRT deadline still limits every call and is never renewed. This does not support a
single 30-minute wait. Preserve exact program/receipt/wait IDs, grants and pending
handles across cancellation or runtime interruption; never replay promotion or
producer effects. No private polling loop, new suspension engine or scheduler.

Prove the actual kernel → bridge → TG notification-based wait path with scaled
test budgets: one wait outlasts the ordinary cap and returns a terminal receipt;
cancellation and restart preserve handles without a second producer admission.
Tests must also show ordinary bindings retain their old bounds and unknown
bindings cannot opt into the longer profile. Source tests do not prove adoption.
Source acceptance may use separate authentic kernel/bridge and TG owner
regressions; clearly label simulated transports. The joined live owner path
remains required before campaign activation. Do not add a shipping test facade
or bypass module ownership solely to combine fixtures in one source test.

## Logs

| Log | Source | How To Read | Details |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses deterministic clock seam in tests. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |

## Metrics

| Metric | Status | Details |
|---|---|---|
| Product activation | deferred | Define after PRD users and workflows are real. |
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Performance budgets | deferred | Define in `../internal/PERFORMANCE.md`. |

## Runtime Configuration And Counters

| Setting/counter | Source | Meaning |
|---|---|---|
| `PROGRAM_RUNTIME_INFERENCE_CEILING_MICROS` | API environment | Default per-session inference monetary ceiling; zero disables the default ceiling. |
| `PROGRAM_RUNTIME_DELEGATION_CEILING_MICROS` | API environment | Default per-session delegation ceiling; unmeasured agent-manager charges are never fabricated as zero. |
| `PROGRAM_RUNTIME_SUBMISSION_DEADLINE` | runner configuration | Supervisor deadline for one program submission. |
| session idle reclamation interval | API supervisor | One-minute scan for expired sessions. |
| telemetry outbox drain interval | telemetry store | Retry cadence for pending event delivery. |
| dead-letter window | telemetry schema/retention | Seven-day retention for undeliverable events. |
| program/refusal/unresolved retention | retention worker | 90 days for program evidence; 30 days for refusal and unresolved-name evidence. |
| skipped manifests | `bindings doctor` | Malformed or unreadable manifests kept out of the callable registry. |
| unreachable scenarios | `bindings doctor` | Bound scenarios whose discovery probe cannot resolve a live owner. |
| dormant bindings | binding condition response | Callable bindings with no recent invocation evidence. |
| dead-letter events | telemetry health | Events that exhausted retry policy and require operator inspection. |

## Alerts / Health

The generated scenario has lifecycle health checks for API and UI. Add
deployment-specific alerts only when deployment target and operator
expectations are known.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| Product usage telemetry | Cannot validate monetization or adoption. | Add before public launch or monetization review. |
| Delegation cost receipt | agent-manager currently omits a per-run charge receipt. | Revisit PRT-P1-011 when agent-manager publishes an explicit charge contract. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
