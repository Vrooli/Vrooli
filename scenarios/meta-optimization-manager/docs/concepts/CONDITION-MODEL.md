# Condition Model

## Purpose Of This Document

This is the **single canonical source** for the **Condition axis**: the model that answers *"is the supply the board counts still working?"*

It is the sibling of [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md), which owns the **Coverage axis** (does the capability exist?) and names the **Empirical axis** (what happens when agents work?). Read that file first — the projections, denominator/numerator split, attestation contract, and status legend are defined there and are **not** restated here.

This document describes the **ideal** instrumentation, not what exists today. That is deliberate and matches how every other denominator in this system is authored: the model is written at full maturity so the distance between it and reality is *measurable* rather than invisible. Current adoption is recorded in [§ Current State](#current-state) as data, never by trimming the model down to what already works.

## Why Condition Is A Distinct Axis

The board has, until now, measured **supply** and **outcomes** and nothing in between:

| Axis | Question | Shape | Owner of the model |
|---|---|---|---|
| **Coverage** | Does the capability *exist*? | `now / total` against a curated denominator | [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md) |
| **Condition** | Does the capability that exists still *work*? | per-leg verdict + instrumentation coverage | this document |
| **Empirical** | What *happens* when agents work? | trend over observed runs, no denominator | [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md) § "Coverage Is Not The Only Axis" |

A coverage gap says *"this does not exist."* An empirical signal says *"agents are hurting."* Condition says the third thing, which neither can express: ***"this exists, is counted as supply, and is silently broken or silently unused."***

### Condition is not Validate

The obvious objection is that `test-genie` already checks whether things work. It does not cover this, and the boundary is precise:

> **Validate covers what can be *provoked*. Condition covers what can only be *observed*.**

A validator constructs a scenario and asserts an outcome. That is the right instrument for anything reproducible on demand. It is structurally incapable of reaching:

- **Quality decay with no oracle.** A search provider that returns results that are *worse*, not absent. Relevance has no assertable ground truth.
- **Runtime-only failure.** A binding that generates cleanly and errors on every real invocation. Validation proves the binding exists; only invocation proves it works.
- **Third-party state.** A provider whose circuit breaker keeps flipping. You cannot provoke a vendor outage in a test.
- **Pass-but-degrading.** A test phase that still passes and now takes forty minutes. Green, and quietly destroying the loop it belongs to.
- **Human-loop abandonment.** A skill that exists, scores healthy, and that agents stop following mid-task.
- **Built, green, and unused.** A capability nothing calls. Every gate passes. Nothing is wrong. Nothing is happening either.

That last class is the dominant observed failure mode in this project, and no existing surface reports it. It gets a first-class status below (`DORMANT`) rather than being folded into degradation, because the correct response — drive adoption, or retire — is different from the response to a broken leg.

Where a projection's numerator **already** folds a serving signal into cell status, that is Condition-as-implemented in one place, not an argument that Condition is redundant. See [§ Relationship To The Coverage Numerator](#relationship-to-the-coverage-numerator).

## The Population Is Derived, Never Authored

Condition has **no space doc and no curated denominator.** Its population is computed:

> The Condition population is the set of **contributor legs** backing every cell that resolved `NOW` in the live coverage join.

A **leg** is the smallest unit the owner can name and measure independently:

| Projection | Leg unit | Named by |
|---|---|---|
| Answer | provider | `search-hub` provider id |
| Validate | phase / provider | `test-genie` catalog entry |
| Guide | skill | `prompt-manager` skill id |
| Act | binding | `program-runtime` binding id |

Three consequences follow, and they are the reason the axis is shaped this way:

1. **Condition cannot drift out of sync with Coverage.** There is no second list to maintain. Add a projection, register a provider, ship a binding — the Condition population extends automatically.
2. **Condition only ever asks about things claimed to work.** A `MISSING` or `IN-REACH` cell has no leg to be in bad condition. Coverage owns the "doesn't exist" case; Condition never duplicates it.
3. **The axis is honest by construction about its own scope.** Its population is exactly the set of claims the board is currently making.

## Signals Are Owner-Measured, Not Contributor-Declared

**The projection owner measures its contributor legs. Contributors declare nothing about their own condition.**

This is the load-bearing architectural decision of this model, and it is the opposite of the intuitive design. The rationale:

- **The owner is in the call path.** Degradation of a leg is a byproduct of fan-out the owner already performs. The contributor sees only the requests it handled; it cannot see the ones that timed out before reaching it, the ones whose results were discarded as poor, or the fact that nobody called it at all.
- **Self-reported health is the weakest evidence class this project recognizes.** On the attestation contract in `COVERAGE-MODEL.md`, a contributor's claim about itself is `DECLARED_UNVERIFIED`. An owner's observation of the leg it called is `DERIVED`. Building a whole axis on the weaker basis would be a strange choice.
- **Enforcement is tractable.** Owner-measured means the contract binds **four scenarios**. Contributor-declared would bind every scenario that ever registers with anything — a fleet migration whose completion date is indistinguishable from never.
- **It preserves the registration boundary.** `search-hub` and `test-genie` deliberately hold no code-level knowledge of who registers with them; registration is scanning and declaration. Contributors must correspondingly hold no knowledge that they are being measured. Owner-side measurement is the only shape that keeps both halves ignorant of each other.

**The one exception is Freshness**, where the authoritative timestamp sometimes lives with the contributor (only it knows when it last rebuilt its index). The rule is: **the owner still reports the signal**, sourcing it from the contributor's existing control surface where necessary. The read seam stays singular even when the fact does not originate with the owner.

## The Three Signal Families

Every leg is characterized along three independent families. A leg may be healthy in one and not another, and the combinations are meaningful — a leg that serves perfectly and is never called is a different finding from one that is called constantly and fails half the time.

### 1. Serving — does the leg answer, and how badly does it fail?

| Signal | Definition |
|---|---|
| `serving.failure_rate` | Fraction of invocations of this leg that returned an error, over a window. |
| `serving.degradation_rate` | Fraction that returned an answer the owner classified as degraded — partial, fallback, timed-out-with-partial, breaker-open. Distinct from failure: the caller got something. |
| `serving.latency` | Distribution (at minimum p50 and p95) of the leg's response time, over a window. |

Latency is a first-class serving signal, not an optimization concern. A leg that answers correctly but slowly enough to change how agents work is degrading the loop, and no other axis will ever say so.

### 2. Freshness — is what it serves current?

| Signal | Definition |
|---|---|
| `freshness.age` | Time since the leg's underlying artifact was last rebuilt, reindexed, or regenerated. |
| `freshness.drift` | Where the owner can compare a declaration against reality, the divergence between them. Absent where no such comparison exists — never fabricated as zero. |

Freshness catches the failure mode where every request succeeds and every answer is stale. Serving signals are structurally blind to it.

### 3. Exercise — is anyone actually calling it?

| Signal | Definition |
|---|---|
| `exercise.invocations` | Count of invocations of this leg over a window. |
| `exercise.distinct_callers` | Number of distinct callers. Separates "load-bearing for many" from "one caller's private path." |
| `exercise.last_invoked_at` | Timestamp of the most recent invocation, or explicitly never. |

Exercise is the cheapest family to source and the one with no existing coverage anywhere. It is also the input to an existing policy: `path:docs/agent-system/DEPRECATION_POLICY.md` sets staleness windows for skills, agents, and teams but currently resolves them by *reference-grep and heartbeat presence*. Exercise signals replace that proxy with measured invocation, and extend the same policy to providers, phases, and bindings, which have no staleness rule today.

## Status Vocabulary

**The vocabulary is closed.** An unrecognized token coerces to `UNINSTRUMENTED` — the conservative reading, which claims nothing.

| Status | Meaning |
|---|---|
| `HEALTHY` | Signals are present and within their declared band. |
| `DEGRADED` | Signals are present and out of band. The verdict **must** name the offending signal and its value. A bare `DEGRADED` is not a valid verdict. |
| `DORMANT` | Serving signals are healthy or absent, and `exercise.invocations` is zero over the window. The capability exists, works as far as anyone knows, and nothing uses it. |
| `UNINSTRUMENTED` | The owner declares no signal in this family for this leg. **This is not a synonym for healthy.** |
| `UNAVAILABLE` | The owner could not be read. The reason is stated verbatim, as with every other cross-scenario read in this system. |

### The uninstrumented-is-not-healthy rule

This is the honesty invariant of the entire axis, and it is the same discipline the rest of the model already enforces — an unaudited coverage cell is authored `PARTIAL`, never `COVERED`; an unmeasured surface reports `UNAVAILABLE` with a reason, never `0`.

> **A leg with no signal is never reported as `HEALTHY`, is never counted in a healthy total, and never contributes to a condition percentage as a passing member.**

Without this rule the axis inverts on contact with reality: a fleet with no instrumentation at all would report perfect condition, and the incentive to instrument would be exactly backwards.

### Instrumentation coverage is reported with every condition verdict

Every condition number is reported as a triple, never as a bare count:

```
<verdict distribution> of <instrumented legs> instrumented, of <total NOW legs>
```

So a reader sees `2 DEGRADED, 1 DORMANT, 8 HEALTHY of 11 instrumented, of 17 total legs` — and can never mistake the six uninstrumented legs for six healthy ones. This mirrors the recursive honesty of denominator-confidence in `COVERAGE-MODEL.md`: the board reports its own blindness alongside its findings.

**Instrumentation coverage is itself a first-class gap.** An uninstrumented leg backing a `NOW` cell is a rankable finding on `focus next`, because an unmeasurable claim is a weaker claim than a measured one.

## Relationship To The Coverage Numerator

Condition and Coverage must not be folded together, and the reason is that folding them destroys both numbers: coverage becomes volatile (a transient vendor outage drops the Answer percentage, and the trend line stops meaning anything), while genuine sustained breakage gets averaged into a supply figure where nobody reads it.

The ideal separation:

1. **Condition reports beside Coverage by default.** A `DEGRADED` leg does **not** change its cell's coverage status. Coverage answers "does this exist"; the answer is still yes.
2. **Sustained degradation promotes to a coverage downgrade.** A leg continuously `DEGRADED` beyond `conditionSustainedWindow` is no longer credible supply, and its cell downgrades `NOW → IN-REACH` — the substrate exists, but it must be repaired before it counts. The promotion is *stated on the cell*, so a downgrade is always traceable to the signal and window that caused it.
3. **`DORMANT` never downgrades coverage.** An unused capability is not absent supply. It is a different finding with a different response, and conflating the two would push the system toward deleting working capability it simply failed to adopt.

### Load-bearing constants

Following the discipline in `COVERAGE-MODEL.md` § "Load-bearing constants" — judgment constants are named, documented, and auditable rather than buried:

- **`conditionSustainedWindow`** — how long a leg must stay `DEGRADED` before it downgrades its coverage cell. Must be long enough to survive an ordinary vendor incident and short enough that chronic breakage cannot hide indefinitely. Revisit once there is a real distribution of degradation durations to read.
- **`conditionExerciseWindow`** — the lookback over which zero invocations yields `DORMANT`. Must exceed the natural cadence of the slowest legitimate caller; a capability exercised only by a monthly audit is not dormant at a weekly window.
- **`conditionBands`** — the per-signal thresholds separating `HEALTHY` from `DEGRADED`. These are per-projection, because a 5% failure rate means something different for a search provider than for a governed destructive binding.

Every constant above is deliberately unset in this document. They are owner-calibrated against real distributions, and authoring a number here before that data exists would be exactly the fabricated precision this model rejects elsewhere.

### Current known asymmetry

`test-genie`'s numerator **already** gates on serving signals: a provider with ledger `failureRate > 0` or `autofix.pending > 0` yields `IN-REACH` rather than `NOW` (`COVERAGE-MODEL.md` § "Per-Cell Numerator Semantics"). That is Condition-as-implemented, predating this model, and it gates immediately rather than after a sustained window.

Under the ideal above, that behavior converges to report-beside plus sustained-promotion. This is a **behavior change to a live number** and therefore a decision, not a refactor: it would raise the reported Validate coverage while introducing a Validate condition line. It is recorded here as the target state so the inconsistency is visible and deliberate rather than discovered later.

## The Ideal Signal Set, Per Owner

This is the target-state contract. Each owner declares these signals as **Measures** — the existing federated, typed, parameterized, search-hub-matchable query layer (`path:docs/concepts/MEASURES.md`) — so that condition is answerable in natural language without opening a UI, and so no new transport is invented for it.

Measures are the right carrier for three reasons beyond convenience. They are **declared**, so an owner's instrumentation is inspectable without running anything. They are **parameterized by a time window**, which every signal here requires. And they are **already federated and centrally indexed**, which is what lets the board read condition in one call instead of polling the fleet.

Legend: **●** declared and live · **◐** partially present · **○** not declared.

### Answer — `search-hub`, leg = provider

| Family | Required signal | State |
|---|---|---|
| Serving | per-provider degradation rate | ● `metrics provider-degradation-rate`, scoped by provider id |
| Serving | per-provider failure rate | ◐ fleet-level `degraded-query-rate` exists; per-provider failure distinct from degradation does not |
| Serving | per-provider latency p50/p95 | ◐ `metrics federated-latency` is federation-level, not per-leg |
| Freshness | provider index age | ○ |
| Exercise | per-provider invocation count, distinct callers, last-invoked | ○ |

`search-hub` is the reference implementation of the Serving family. `provider-degradation-rate` — a per-leg signal owned by the projection owner, scoped by contributor id — is the shape every other owner should copy.

### Validate — `test-genie`, leg = phase / provider

| Family | Required signal | State |
|---|---|---|
| Serving | per-provider failure rate | ● in `SelfHealth`; currently gates the numerator |
| Serving | per-provider pending autofix | ● in `SelfHealth`; currently gates the numerator |
| Serving | per-phase duration trend | ○ — the pass-but-slower blind spot |
| Serving | per-phase flake rate, distinct from failure rate | ○ — a phase that fails intermittently and a phase that fails consistently need different responses |
| Freshness | phase catalog freshness | ○ |
| Exercise | per-phase invocation count, last-invoked | ○ |

### Guide — `prompt-manager`, leg = skill

| Family | Required signal | State |
|---|---|---|
| Serving | per-skill health score | ◐ graph health scores exist and feed the numerator; they measure authored quality, not serving behavior |
| Serving | per-skill abandonment rate | ○ — read, then not followed to completion |
| Freshness | per-skill age against the canon it cites | ○ |
| Exercise | per-skill read count, distinct callers, last-read | ○ |

Guide currently has the **least** condition visibility of the four and the most prose-shaped supply — the combination where silent decay is most likely and least detectable. Its exercise signal is also the highest-value single measure in this table, because it is the direct input to the skill staleness window already written into `DEPRECATION_POLICY.md`.

### Act — `program-runtime`, leg = binding

| Family | Required signal | State |
|---|---|---|
| Serving | per-binding invocation outcome (success / refused / failed) | ○ |
| Serving | per-binding latency p50/p95 | ○ |
| Freshness | binding registry regeneration age vs. descriptor and manifest mtimes | ○ |
| Exercise | per-binding call count, distinct callers, last-invoked | ○ |

**Per-binding invocation outcome is the highest-leverage unbuilt measure in this document.** It is simultaneously the Act condition signal and the friction evidence the improvement loop wants, and it is the only instrument that can distinguish the bindings that are genuinely callable from the ones that merely resolve. A registry reporting a four-digit binding count with no invocation data is a supply claim with no condition evidence behind it whatsoever.

Note the dependency: `program-runtime`'s existing `programs.mine` measure is structurally incapable of returning a non-empty result while its corpus is process memory. **Durable retention of the program corpus is a precondition for Act condition measurement**, not an independent durability preference.

### Fleet-wide exercise

Serving and Freshness are necessarily per-owner. **Exercise is not** — it has a single universal source. `vrooli.events.receipt.v1` is emitted target-side by every scenario served by `api-core/server.Run`, caller-agnostic, carrying outcome, status, duration, and a policy-declared projection (`path:docs/concepts/VROOLI_EVENTS_PLATFORM_CONTRACT.md`).

The ideal is therefore that **no owner implements Exercise independently.** One aggregate over the receipt stream, keyed by target scenario and operation, serves the Exercise family for all four projections at once. Owners contribute only the mapping from their leg ids to the operations that serve them.

This is the cheapest and broadest signal in the model, and its prerequisite is stated plainly in [§ Current State](#current-state).

## How The Board Consumes Condition

Condition joins through the same seam every other signal uses — the multiplexed `GapSource` fan-out behind `focus next` — as one additional named source. The properties it inherits are the ones that matter:

- **Independent degradation.** An unreadable condition source produces a visible availability entry; the other sources keep ranking. It can never take the board down.
- **One read, not a fleet fan-out.** Condition reads the **central measures index** rather than polling scenarios. Instrumentation is federated; collection is not.
- **Ranked beside everything else.** A degraded leg, a dormant capability, an uninstrumented leg, a coverage gap, and an empirical friction cluster all arrive on one ordered surface. A reader still needs only `focus next`.

The ranking intent, stated so implementations do not have to guess it:

| Finding | Intent |
|---|---|
| `DEGRADED`, sustained | Highest. Counted supply that is actively failing is worse than supply that was never claimed. |
| `DEGRADED`, transient | Ranked, damped by duration. Incidents should be visible without drowning structural findings. |
| `UNINSTRUMENTED` leg on a `NOW` cell | Ranked by the leg's coverage weight. An unverifiable claim is a weaker claim. |
| `DORMANT` | Ranked low and never auto-escalating. It routes to an adoption-or-retirement judgment, and per `DEPRECATION_POLICY.md` § "The mandatory roadmap check", a low-usage entity may be load-bearing for roadmapped work. This axis supplies evidence to that check; it never substitutes for it. |

## Enforcement

**Bind the contract to the four projection owners, never to contributors.**

`measures-health` is the enforcement point: it already hosts the central measures index, already grades declared-measure coverage per stateful domain, and already owns the vocabulary in which these signals are expressed. Extending it to grade *"each projection owner declares the required signals for its leg unit"* is within its existing charter and does not require a new validator.

The grading contract:

- A missing signal is `UNINSTRUMENTED` and **visible**, not a hard failure. An owner is never blocked from shipping because an ideal signal does not exist yet; it is simply reported as a gap and ranked.
- A declared signal that returns nothing is worse than an absent one and is graded as such — it asserts instrumentation that does not exist.
- A waiver naming the signal and the reason is a stated position and grades accordingly, matching the `measures.omitted` precedent.

The surface being four scenarios rather than the whole fleet is what makes this contract enforceable at all, and it follows directly from the owner-measured decision above.

## Current State

Recorded as of 2026-08-11, as data. This section is the measured distance from the model above and is expected to change; the model is not expected to change with it.

| Fact | Value |
|---|---|
| Scenarios declaring any measure | 14 of 118 |
| Total declared measures, fleet-wide | 97 |
| Projection owners with a per-leg serving signal | 1 of 4 (`search-hub`) |
| Projection owners whose numerator already gates on condition | 1 of 4 (`test-genie`) |
| Projection owners with any freshness signal | 0 of 4 |
| Projection owners with any exercise signal | 0 of 4 |
| Condition source registered on the board | none |
| Overall instrumentation coverage | effectively `0%` — reported as `UNINSTRUMENTED`, never as healthy |

Two scenarios outside the projection owners already demonstrate the target shape and are worth copying rather than reinventing:

- **`ai-gateway`** measures its provider legs over `route_events`: failure rate, fallback rate, breaker-open, capacity rejections, latency p95. This is the most complete Serving family in the project.
- **`agent-manager`** declares 23 measures whose shape — retry rate, tool failure rate, help-recovery rate, repeated-work rate, file-reread rate — is squarely **Empirical**, not Condition. It is a useful contrast: the same mechanism serves both axes, and the axis a measure belongs to is determined by whether its subject is a *leg of supply* or an *agent outcome*.

### Stated prerequisites

These are known blockers on the model above. They are recorded so the gap is scheduled rather than rediscovered:

1. **Fleet-wide Exercise needs a counting surface.** `vrooli-events` exposes `events query` by type, source, and correlation id, but no aggregate or count operation. It also ships no `cli/manifest.json` and declares no measures of its own, so it is neither bindable nor measurable today. The single cheapest signal in this model is blocked on its owner growing a counting surface.
2. **Exercise attribution is bounded by agent identity.** Only verified `agent-manager` identity claims may set a receipt's subject and agent correlation. `exercise.invocations` and `exercise.last_invoked_at` are unaffected, but `exercise.distinct_callers` degrades to a count of *attributable* callers and must report the unattributed remainder rather than silently undercounting.
3. **Act condition needs a durable program corpus.** Stated above under the Act table.
4. **The Validate gating asymmetry is an open decision**, not an implementation gap. Stated above under [§ Relationship To The Coverage Numerator](#relationship-to-the-coverage-numerator).

## Governing Principles

- **Uninstrumented is never healthy.** The axis reports its own blindness alongside its findings, or it is worse than having no axis.
- **Owners measure legs; contributors declare nothing.** Observation beats self-report, and it keeps the registration boundary intact in both directions.
- **The population is derived, never authored.** Condition asks only about claims the board is already making, and can never drift from them.
- **Existence and condition stay separable.** Supply that is broken is reported as broken supply, not as absent supply — until it has been broken long enough that the distinction stops being meaningful.
- **Surfaces, does not decide.** Condition ranks candidates and reports numbers. Whether to repair, adopt, or retire a leg stays an agentic and operator judgment, routed through the policies that already own those decisions.

## Cross-References

- [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md) — the Coverage axis, the projections, the attestation contract, and the status legend this document builds on.
- [`DOMAINS.md`](DOMAINS.md), [`ARCHITECTURE.md`](ARCHITECTURE.md).
- `path:docs/concepts/RECURSIVE_SELF_IMPROVEMENT.md` — the narrative spine this axis sits inside.
- `path:docs/concepts/MEASURES.md` — the federated metrics layer every condition signal is declared in; its contract is `packages/measures-go` and its enforcer is `measures-health`.
- `path:docs/concepts/VROOLI_EVENTS_PLATFORM_CONTRACT.md` — the receipt contract backing the fleet-wide Exercise family.
- `path:docs/agent-system/DEPRECATION_POLICY.md` — the policy that consumes `DORMANT` findings, including the mandatory roadmap check that bounds them.
- The four space docs, which own the Coverage denominators whose `NOW` cells define this axis's population: `search-hub/docs/spaces/answer-space.md`, `test-genie/docs/spaces/validate-space.md`, `prompt-manager/docs/spaces/guide-space.md`, `program-runtime/docs/spaces/act-space.md`.
