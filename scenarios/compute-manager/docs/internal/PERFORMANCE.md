# Performance — Compute Manager

This document records performance budgets, current measurements, known
constraints, and regression procedures.

> **Status: implementation measurements are still incomplete.** The scenario
> now has provider, provisioning and bounded loop implementations, but no
> provider-live fleet measurement has been captured. Targets below remain
> forward-looking unless explicitly marked observed.
>
> **This document must be updated with real numbers once the first
> provider-backed instance is provisioned.** That rewrite is a requirement, not a
> courtesy. `vrooli-bridge` shows the failure to avoid: its
> `PERFORMANCE.md` still opens by declaring the scenario unbuilt with no
> measurements long after real data existed, so a stale disclaimer became
> the most confident-sounding statement in the file. The first real
> create, enroll, meter and destroy cycle invalidates the header of this
> document. Replace the disclaimer, populate Current Measurements, and
> strike the "target" label from every budget that has been measured,
> leaving the label only where it is still honest.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

Performance here is unusual: **latency is money**. Most scenarios treat a
slow path as a user-experience problem. In this one, a late destroy is a
provider charge, a late meter transition is a billing error, and a slow
reconciliation sweep is a window in which an orphan bills unseen. The
budgets are therefore split into three groups by what a miss costs.

### Group 1: cost-critical, where lateness is money

| Surface | Budget (target) | Rationale | Status |
|---|---|---|---|
| Expiry sweeper lateness (past expiry to destroy issued) | within one sweep interval, and the interval itself well inside the provider's minimum billable unit | An instance past expiry is pure waste. The sweep interval is the real budget, because lateness is bounded by it. | target (unbuilt) |
| Instance-side first-boot timer accuracy | powers off at its own expiry with no control-plane contact | The backstop for `OT-P0-004`. Its whole value is that it holds when the sweeper does not run at all, so its budget is independence, not speed. | target (unbuilt) |
| Meter transition recording lag (state change to usage record) | recorded in the same durable transaction as the transition | Usage is metered from transitions this scenario caused, never from an observer loop. If the record can lag behind the transition, the two can disagree and the disagreement is a billing error. | target (unbuilt) |
| Heartbeat re-reservation interval | strictly shorter than the reservation window, with margin | The upstream reservation window is hard-coded to ten minutes, shorter than an hour of compute. If a re-reservation is late, running cost goes unreserved. | target (unbuilt, blocked upstream) |
| Reconciliation sweep interval | short enough that an orphan's undetected cost stays below the alarm threshold | The sweep interval, not the sweep duration, sets how long an unaccounted instance can bill unseen. | target (unbuilt) |
| Settlement on teardown | settled in the destroy path, not on a later batch | A settlement that can be dropped is a charge that never happens. | target (unbuilt) |

### Group 2: correctness-critical, where ordering matters more than speed

| Surface | Budget (target) | Rationale | Status |
|---|---|---|---|
| Intent durability before any provider call | intent and its idempotency key committed before the provider client is reachable | `OT-P0-002`. This is an ordering guarantee, not a latency budget. It may be slow; it may never be skipped for speed. | target (unbuilt) |
| Reservation before any provider call | reservation obtained server-side before the provider client is reachable; a refusal short-circuits with zero provider calls | `OT-P0-006`. Same shape: correctness first, and a refused reservation must cost nothing. | target (unbuilt) |
| Idempotent replay | replaying a known idempotency key returns the original intent without a second provider call | A replay that races into a duplicate create is a duplicate hourly bill. | target (unbuilt) |

### Group 3: ordinary responsiveness

| Surface | Budget (target) | Rationale | Status |
|---|---|---|---|
| Connect-RPC metadata reads (list, describe from local state) | sub-100ms server-side, excluding any provider call | SQLite-backed reads of our own records. Reads should never be the bottleneck. | target (unbuilt) |
| Request accepted to intent persisted | single-digit milliseconds, dominated by one SQLite commit | The synchronous part of a create request before the slow provider work begins. | target (unbuilt) |
| Provider create call to running state | minutes, set entirely by the provider | Not our budget. It is recorded so that our own overhead can be shown to be negligible against it, and so a regression in our code is not blamed on the provider or the reverse. | target (unbuilt, provider-bound) |
| Full bidirectional sweep duration at fleet size N | bounded by provider list pagination and rate limits, not by local work | The sweep compares provider inventory against local records in both directions. Local comparison is cheap; the provider API is the ceiling. | target (unbuilt) |
| Inventory dashboard time to first meaningful render | figures visible without waiting on a provider round trip | The dashboard reads our own records. Cost and expiry are the two facts that must never wait on a third party to appear. | target (unbuilt) |
| UI build | 5-10 minutes accepted for the current Vite module graph | Inherited platform constraint, not specific to this scenario. | inherited |
| API and UI health | responsive under the lifecycle health timeout | `/health` checks via lifecycle. | inherited |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None. Nothing is implemented, so nothing has been measured. | n/a | n/a | 2026-09-03 |

There are no real performance numbers, and there is no partial data
either. The API, the CLI, the provider adapters, the reconciler and the
expiry sweeper do not exist beyond documentation and generated template
code, so there is nothing that could have produced a number. Every value
in the Budgets tables is a target awaiting first measurement.

The first measurements will come in two waves. The fake provider carries
wave one: intent durability, reservation ordering, idempotent replay,
sweep duration at synthetic fleet sizes, and metadata read latency can
all be measured with no API key and no money spent. Wave two needs one
real provisioned instance, and it is the wave that produces every number
in Group 1, because lateness only means something against a real hourly
bill.

## Known Constraints

- **Nothing is measured, so nothing is confirmed.** The most important
  constraint on this document is its own emptiness. Do not cite a budget
  here as evidence of anything.
- **The provider dominates every wall-clock the operator sees.** Create
  latency is minutes and belongs to the provider. Our own overhead must
  be shown to be negligible against it rather than tuned in isolation.
- **Provider rate limits, not local work, bound the reconciliation
  sweep.** Both directions of the sweep need provider inventory. Fleet
  size therefore turns into API calls, and a faster sweep interval buys
  detection speed at the cost of rate-limit headroom.
- **Hourly rounding creates a cost cliff, not a cost curve.** Hetzner
  rounds a partial hour up to a full hour. Destroying at fifty-nine
  minutes and at one minute cost the same, so an optimization that shaves
  seconds off a destroy path is worth nothing, while one that keeps a
  workload inside an hour boundary is worth the whole hour. Any timing
  work here must be aimed at the boundary, not at the millisecond.
- **Provider billing data lags by hours to more than a day.** It is a
  reconciliation signal and never a control. No fast path may wait on it,
  and no budget above may be validated against it in near-real-time.
- **SQLite is a single writer.** The expiry sweeper, the reconciler and
  the request path share one database. Contention between a long sweep
  and a live create is the first scaling limit to look for, and the sweep
  should not hold a write transaction across provider calls.
- **The reservation window is ten minutes upstream and hard-coded.**
  Until it is parameterised or covered by heartbeat re-reservation, the
  heartbeat budget in Group 1 cannot be met by design rather than by
  tuning.
- **A stopped instance still bills on most providers**, which is why
  there is no pause. No amount of performance work makes a power-off
  cheap, so the fast path out of cost is destroy and there is no second
  option to optimize.

## Regression Procedure

Until the first real instance exists, steps 1 and 2 are the only ones
that can run.

1. Run `make test`. This is the only step available today.
2. Once the fake provider exists, capture wave-one numbers against it:
   intent commit time, reservation ordering, idempotent replay, metadata
   read latency, and full sweep duration at several synthetic fleet
   sizes. Record them in Current Measurements with the date and the fleet
   size, because a sweep number without a fleet size means nothing.
3. Once one real instance has been created, enrolled, metered and
   destroyed, rewrite the header of this document, populate Current
   Measurements with wave-two numbers, and strike the "target" label from
   every budget that now has an observation behind it.
4. For every Group 1 budget, record the miss as well as the target. A
   late destroy has a currency value; write it down. Lateness measured in
   money is the only unit that makes these budgets comparable to each
   other.
5. Compare metered usage against the provider's own statement on the
   `OT-P1-003` daily cadence. A divergence beyond threshold is a
   correctness finding, not a performance one, and belongs in
   [`PROBLEMS.md`](PROBLEMS.md).
6. Record accepted constraints here and unresolved debt in
   [`PROBLEMS.md`](PROBLEMS.md). A budget that is repeatedly missed is
   debt, not a constraint, until an explicit decision moves it.

## Cross-References

- [`../../PRD.md`](../../PRD.md): the operational targets the Group 1 budgets defend
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md): the meter, reconcile and expiry domains these budgets belong to
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md): why lateness here is money, and the rounding problem
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md): signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md): release checklist
- [`TESTING.md`](TESTING.md): coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md): unresolved performance debt
