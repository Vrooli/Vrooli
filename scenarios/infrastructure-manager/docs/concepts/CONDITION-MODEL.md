# Condition Model

## Purpose Of This Document

This is the **single canonical source** for the **Condition axis**: the model that answers *"for the cells that are instrumented, is the platform actually in band?"*

It is the sibling of [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md), which owns the Coverage axis (is this dimension instrumented at all?). Read that file first — the projections, cell grid, denominator/numerator split, and denominator-confidence are defined there and are **not** restated here.

Two things this document owns that are easy to look for in the wrong place:

- **Banding** — how a reading becomes in-band or out-of-band. The *bar* itself lives in [`SETPOINT-MODEL.md`](SETPOINT-MODEL.md); the *evaluation* lives here.
- **Actuation efficacy** — whether a completed fix actually returned its sensor to band.

The **Trust** sub-axis — whether a reading can be believed at all — is large enough to warrant its own document: [`TRUST-MODEL.md`](TRUST-MODEL.md). Condition depends on it completely: **an untrusted reading never produces a band verdict.**

## Why Condition Is A Distinct Axis

Coverage measures instrumentation and says nothing about the platform. A fully-instrumented, fully-broken platform reads 100% on coverage. The boundary between the two axes is precise:

> **Coverage says "we can see this." Condition says "and here is what we see."**

The failure this separation prevents is the one that makes reliability boards useless: a single blended "health score" in which improving instrumentation makes the number go *down* (because newly-visible problems appear) and deleting a sensor makes it go *up*. Under that shape, the rational move for anyone graded on the score is to instrument less. Keeping the axes separate removes the incentive entirely — closing an open-loop cell raises coverage, and whatever it reveals lands on condition where it belongs.

### Condition is not Validate

`test-genie` already checks whether things work. It does not cover this, and the boundary is the same one `meta-optimization-manager` draws:

> **Validate covers what can be *provoked*. Condition covers what can only be *observed*.**

A validator constructs a scenario and asserts an outcome. It is structurally incapable of reaching a scenario that has been up for 29 of the last 30 days, a heal that succeeds but loops, a claim that is honest at grant time and 4× over-reserved by Tuesday, or a backup that has never been restored. None of those can be provoked; all of them can be observed.

## The Population Is Derived, Never Authored

Condition has **no space doc and no curated denominator of its own.** Its population is computed:

> The Condition population is the set of **legs** backing every cell that resolved `NOW` in the live coverage join.

A **leg** is the smallest unit the owner can name and measure independently — the leg unit per projection is declared in [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md) § The Projections (a check, a claim, a plan, a phase, a saturation window).

Three consequences follow, and they are why the axis is shaped this way:

1. **Condition cannot drift out of sync with Coverage.** There is no second list to maintain. Add a projection, register a check, ship a backup plan — the Condition population extends automatically.
2. **Condition only ever asks about things claimed to be instrumented.** A `MISSING` or `IN-REACH` cell has no leg to be in bad condition. Coverage owns the "we cannot see this" case; Condition never duplicates it.
3. **The axis is honest by construction about its own scope.** Its population is exactly the set of claims the board is currently making.

> **A cached leg population is an architectural defect, not an optimization.** Operating-model rule 6 forbids enumeration anywhere in this team's surfaces, and a stored leg list is the central capability-health roster the team's retired instrumentation roadmap explicitly rejected in its Gap 11. If the derivation is slow, the fix is a faster derivation.

## Signals Are Owner-Measured, Never Self-Declared

**The control layer measures the legs it operates. Plant elements declare nothing about their own reliability.**

This is inherited verbatim from `meta-optimization-manager`'s condition model and is the load-bearing architectural decision of this axis:

- **The owner is in the call path.** Autoheal already probes the element; capacity already tracks the claim. Degradation is a byproduct of work the owner is doing anyway. The element itself sees only the requests it served — never the probe that timed out before reaching it, and never the fact that nobody watched it at all.
- **Self-reported health is the weakest evidence class this project recognizes.** On the attestation contract, an element's claim about itself is `DECLARED_UNVERIFIED`; an owner's observation of the leg it operated is `DERIVED`. Building an axis on the weaker basis would be a strange choice for the one scenario whose entire job is evidence quality.
- **Enforcement is tractable.** Owner-measured binds **six control layers**. Element-declared would bind every scenario in the fleet — a migration whose completion date is indistinguishable from never.

## Banding

A **band evaluation** joins a trusted reading against its cell's current deadband and yields one of:

| Verdict | Meaning |
|---|---|
| `IN_BAND` | The reading is inside the deadband. No finding. |
| `OUT_OF_BAND` | The reading is outside the deadband, and any sustain requirement has been met. |
| `PENDING_SUSTAIN` | Outside the deadband, but the target requires a sustained excursion that has not yet been met. Reported, not yet a finding. |
| `NEEDS_BASELINE` | The target bands on a *direction* across cycles and no prior baseline exists. Explicitly not a pass. |
| `NOT_EVALUATED` | The reading carries a non-`VALID` trust verdict. Neither in band nor out of band — it is not evidence. |

### Band verdicts are computed, never stored

This is the invariant that keeps a stale board structurally impossible:

> **Store readings. Never store a band verdict.**

A band verdict is a statement about the *target*: given this value, was it inside the deadband? Ask again tomorrow against a tightened deadband and you get a correct new answer from the same stored value. So tightening a target **re-grades its own history** rather than stranding judgments made against a bar nobody uses any more.

The **trust verdict is the deliberate exception and is stored** — see [`TRUST-MODEL.md`](TRUST-MODEL.md) and [`DATA.md`](DATA.md). A trust verdict is a statement about the *observation*, and nothing can reconstruct after the fact whether a check was saturated at the moment of the read.

### `NEEDS_BASELINE` is not a pass

Some targets band on a trend across cycles rather than an absolute level. A single reading cannot decide those. Reporting them as in-band because nothing contradicts them is the quiet version of the dead-deadband failure, so they get their own verdict and appear on the error surface as *needing a prior baseline*.

## Reading History

Uptime over thirty days *is* history. It cannot be computed from a live probe, which is why this scenario persists readings where `meta-optimization-manager` persists nothing.

The team's retired instrumentation roadmap named the cost of not storing them in its Gap 11: **an outage becomes indistinguishable from missing data after the fact.** A board that cannot tell "the element was down" from "we were not looking" cannot support any windowed target at all.

Retention is therefore a **correctness constraint, not housekeeping**: the floor is the longest window any cell declares, plus margin, and it is derived from the setpoint rather than hardcoded. Trimming below that floor silently converts a measurable cell into an unmeasurable one — which must then be *reported* as unmeasurable, never quietly reported as in band. See [`DATA.md`](DATA.md) § Retention And Deletion.

## Actuation Efficacy

Update-protocol rule 5 of the setpoint states the contract exactly:

> A finding that creates downstream work names its sensor and the expected in-band return. The first heartbeat after that work completes re-reads the sensor and records the result on the finding: returned in band, or did not. **A fix that does not move the sensor re-opens the finding — the fix author does not grade the fix; the sensor does.**

Efficacy is a distinct axis because condition alone cannot express it. A finding that was out of band, is now in band, and would have been in band anyway is indistinguishable from a successful fix unless the expected return was named *before* the work started.

| Field | Meaning |
|---|---|
| `sensor_ref` | The cell whose reading grades this finding |
| `expected_return` | The band verdict the fix is predicted to produce |
| `observed_return` | What the sensor actually said on the first read after the work landed |
| `verdict` | `MOVED` · `DID_NOT_MOVE` · `AWAITING_WORK` · `UNMEASURABLE` |

`DID_NOT_MOVE` re-opens the finding. `UNMEASURABLE` — the sensor became untrusted or unavailable between the finding and the re-read — is explicitly **not** a pass, and routes to the instrument rather than back to the plant.

This is the strongest single defence this scenario offers against reliability theatre, and today it is a sentence in a document with no mechanism behind it.

## Cascade Discipline

The team's cascade rule orders the layers, and ranking must honour it:

> Layer order (inner → outer): **sensor-channel integrity**, host/process substrate, capability availability, efficiency and performance trends, measurement improvement.

Sensor-channel integrity is innermost. A performance finding raised while the alarm channel is saturated is premature by construction. The `focus` domain therefore ranks trust findings above condition findings whenever both are present, **and states that it is doing so** — a reordering the reader cannot see is indistinguishable from a ranking bug.

## Load-Bearing Constants

Judgment constants are named, documented and auditable rather than buried in code:

- **`readDeadline = 10s` per source** — a slower source is an honest `UNAVAILABLE`, not a hang. Matches `meta-optimization-manager`'s `numeratorDeadline`, which was widened from a 3s assumption after live owners legitimately took several seconds to aggregate a fleet read.
- **`saturationWindow = 24h`** — inherited from the team's sensor-integrity rules. See [`TRUST-MODEL.md`](TRUST-MODEL.md).
- **`retentionFloor = longest declared window + margin`** — derived from the setpoint, never hardcoded.
- **`conditionCoverageFloor`** — deliberately unset. Setting it to whatever coverage happens to be today would report in band while the defect stands, which is precisely the dead-deadband failure `FRAMEWORK_HEALTH.md` § "Deadband rule" names.

## Current State

Recorded as of 2026-08-19, as data.

| Fact | Value |
|---|---|
| Cells with a live band evaluation | availability readings are re-evaluated against the current setpoint; other projections remain explicitly unavailable until their owner source joins |
| Reading history retained | none |
| Efficacy records | none; update-protocol rule 5 has no mechanism behind it |
| Alarm-channel state at last reading | 1,058 critical events / 24h against a ≤500 deadband — out of band |
| Team loop status | `paused-manual` since 2026-07-24 — no reading is a live baseline until it resumes |

## Governing Principles

- **Store readings; never store a band verdict.** Tightening a target re-grades its own history.
- **An untrusted reading is never banded.** It is neither in band nor out of band; it is not evidence.
- **The leg population is derived per read, never cached.** A stored roster is an architectural defect.
- **Owners measure; elements never self-declare.** Self-reported health is the weakest evidence class here.
- **`NEEDS_BASELINE` and `UNMEASURABLE` are not passes.** A verdict that cannot be reached is reported, never assumed favourable.
- **Sensor-channel integrity outranks everything.** And the board says so when it reorders.

## Cross-References

- [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md) — projections, cells, denominator-confidence.
- [`TRUST-MODEL.md`](TRUST-MODEL.md) — the closed trust vocabulary this axis depends on.
- [`SETPOINT-MODEL.md`](SETPOINT-MODEL.md) — the deadbands banding evaluates against.
- [`DATA.md`](DATA.md) — reading storage, retention floor, and what is deliberately not persisted.
- [`DOMAINS.md`](DOMAINS.md) — the `condition` and `focus` domains that implement this.
- `scenarios/meta-optimization-manager/docs/concepts/CONDITION-MODEL.md` — the sibling instrument's condition axis and the precedent for derived populations.
