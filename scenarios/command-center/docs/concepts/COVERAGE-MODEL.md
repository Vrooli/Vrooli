# Coverage Model

**Status:** contract canon for this scenario. The worked precedent is `path:scenarios/meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md`; read that as an example of the same contract, not as a second contract.

## Purpose

Invariant 3 of the instrument contract states the rule this document implements: *three axes, never merged.* `path:docs/agent-system/TARGET_MODEL.md` names them **coverage**, **condition** and **empirical**, and is explicit about why they must stay apart — "coverage sees only existence, and a capability that is built, green on every gate, and silently degrading reports as healthy supply on every surface."

Every number this board shows answers all three before it is allowed on screen:

| Axis | The question | Named here as |
|---|---|---|
| **Coverage** | Is there a sensor for this at all? | `coverage` |
| **Condition** | Does the sensor that exists still work, and is this particular reading usable? | `trust` |
| **Empirical** | Did the work actually move this outcome? | `empirical` |

The three are independent. A metric can have a perfect sensor whose last three fetches failed. A metric can have a flawless cached value for a pipeline that was never built. A metric can be measured perfectly, on a fresh reading, and be going the wrong way. Folding any pair into one status makes both unreadable, which is exactly what the previous design did with a single `dataSource: live | partial | gap` field.

**On naming.** This scenario calls the condition axis `trust`, because at this board's granularity condition is a property of a reading rather than of a capability: a source that answers with a stale value, an impossible value or a mismatched unit is a sensor-channel fault, and that is what the verdict records. The vocabulary follows `infrastructure-manager`'s trust verdicts rather than inventing a third set of words. Where this document says *trust*, `TARGET_MODEL.md` says *condition*.

## Axis 1 — Coverage

Does supply exist to answer this outcome?

| Status | Meaning | Typical cause |
|---|---|---|
| `NOW` | A live source answers it today. | The owning team has an instrument and it exposes this reading. |
| `IN-REACH` | Substrate exists — an endpoint, a table, a declared stub — but the pipeline was never built. | The data is collected somewhere; nobody wired the aggregation. |
| `MISSING` | No substrate. A true gap. | Nothing anywhere collects this. |
| `UNREGISTERED` | The objective set names an outcome for which this registry has no row at all. | Nobody has yet decided what would measure it. |

`UNREGISTERED` is the addition this scenario needs beyond the standard three, and it is the most important one. The outcomes charter already carries the concept under the flag `pending-command-center` — "the metric is not in the registry at all." A metric nobody named renders nowhere, which means the board is silent about exactly the holes nobody has thought of yet. Counting the unregistered set is what stops the board from being confidently blind.

Coverage is authored and joined, never inferred from whether a fetch succeeded. **A source outage must never move a cell to `MISSING`.** A cell absent from a live join keeps its authored status; fabricating `MISSING` from an unreachable owner would turn every incident into an apparent coverage collapse, during exactly the incidents the board exists to surface.

## Axis 2 — Condition (trust)

Does the supply that exists still work? At reading granularity: is this reading usable?

| Verdict | Meaning |
|---|---|
| `VALID` | Observed inside the source's TTL, units agree, value is within a plausible domain. |
| `CACHED` | The last trustworthy value, held past its TTL because the current fetch failed. Carries its age. |
| `UNAVAILABLE` | The source could not be read and there is no cached value. Carries the reason verbatim. |
| `UNTRUSTED` | A reading arrived but cannot be believed — unit mismatch, an impossible value, or a source reporting a shelved sensor. |

Two rules bind this axis:

- **Only `VALID` contributes to an aggregate.** A count computed across a mix of valid and cached readings is not a count.
- **`VALID` requires a fresh observation.** Trust `VALID` may not be emitted without an `observedAt` inside `ttlSeconds` (`CC-P0-004`). A number with no age is the shape of every frozen-dashboard failure.

An `UNTRUSTED` reading routes to the *instrument's owner*, not to the plant. A unit mismatch is a sensor-integrity finding against whoever exposes the reading; it is not evidence that the underlying work is going badly.

## Axis 3 — Empirical

Coverage says a sensor exists. Condition says the reading is good. Neither says whether the work this board exists to grade is *producing* anything — and for a `production-ledger` instrument, outcome evidence is not an extra, it is one of the three things the archetype is defined to return.

The evidence already exists in a specified form. The outcomes charter's **prediction ledger** binds every portfolio decision to a named metric, a direction and a horizon date, later scored against measured evidence. Those verdicts are this board's empirical axis.

| Verdict | Meaning |
|---|---|
| `NONE` | No prediction rides on this metric. Not a deficiency — most metrics carry none. |
| `PENDING` | A prediction names this metric and its horizon has not passed. Carries the target and the remaining horizon. |
| `HIT` | A matured prediction's stated direction was observed. |
| `MISS` | A matured prediction's stated direction was not observed. |
| `UNMEASURABLE` | The horizon passed and the sensor did not exist to score it. |

Three rules bind this axis:

- **Empirical never merges into coverage or condition** (`CC-P0-014`). A `MISS` is not a sensor fault and must never dim a reading or change its ink; a metric can be `NOW` + `VALID` + `MISS`, and that combination is the single most useful thing this board can say.
- **`UNMEASURABLE` routes to the gap surface.** The charter's prediction rule already requires that an unmeasurable verdict name the missing sensor and route to `outcome-gap`. That makes empirical a *source* of ranked findings, not just a display field.
- **The field exists from the first payload; the values will be sparse for a while.** The charter is explicit that scoring is sparse by design until stable metrics exist. Carrying the field from day one is what stops it being retrofitted into a schema that has hardened around two axes — which is precisely the mistake `dataSource` made.

The rendering of this axis is deliberately quiet: a metric under an open prediction shows its target and remaining horizon beside the value (`CC-P2-001`), and nothing else on the board changes. An outcome going badly is information, not an alarm.

## The composition

Coverage and trust compose into the four renderings the board uses. **Empirical composes into nothing** — it is displayed beside a figure, never folded into how the figure is drawn. The composition is a display concern and lives in [PROVENANCE-MODEL.md](PROVENANCE-MODEL.md); the point here is that the *payload* carries all three axes as separate fields and the display derives the rendering. The API never emits a pre-composed "ink" — that would be the merge the invariant forbids, moved one layer down.

| Coverage | Trust | Value present? | What the board draws |
|---|---|---|---|
| `NOW` | `VALID` | measured | solid |
| `NOW` | `CACHED` | last measured | dimmed, with age |
| `NOW` | `UNAVAILABLE` | none | the metric's frame, with the stated reason |
| `NOW` | `UNTRUSTED` | measured but unbelievable | the frame plus the integrity finding; never the number alone |
| `IN-REACH` | — | authored sample | hollow |
| `MISSING` | — | authored sample | dotted |
| `UNREGISTERED` | — | none | nothing on a room; counted in the self-report |

## Denominators and confidence

Where the board reports a count against a total — "4 of 12 outcome categories measurable" — the denominator is authored by the objective set, not by this scenario, and the reading carries a **denominator-confidence** with a rationale:

- `authoritative` — the total is defensible and complete
- `partial` — the total covers part of the space and states which part
- `sketch` — the total is a working estimate and is labelled as one

The honesty is recursive: a board reads "X of Y against a `partial`-confidence denominator," so it structurally cannot imply a completeness it has not earned. **No ratio appears without its denominator-confidence.**

Because Director Swarm's archetype is `production-ledger`, this board prefers counts with stated denominators over percentages, and forbids synthesised composite scores entirely. See [INSTRUMENT-MODEL.md](INSTRUMENT-MODEL.md) § Archetype.

## The third axis: source-team maturity

Coverage answers *is there a sensor*. It does not answer *why not*, and the two most common answers carry completely different work:

- **No pipeline.** The team has an instrument; this particular reading was never wired. One team, one piece of plumbing.
- **No instrument.** The team that would own the reading has no aggregator at all, or none declared. That is not plumbing; it is a missing control loop, and filing six pipeline gaps against it would imply six pieces of work where there is one.

Every reading therefore carries its source team's declared instrument state, read live from that team's record (`CC-P0-008`). See [SOURCE-MAP.md](SOURCE-MAP.md) for the current fleet and what each room inherits from it.

## Open-loop self-report

Every `MISSING` cell carries `firstObservedMissing` and the days it has stood. Every `UNREGISTERED` outcome is counted with the same dating. This includes this scenario's own blind spots — an instrument that reports everyone else's holes and not its own is not honest, it is flattering.

A gap that is *declared and dated* stays a finding. A gap that is merely declared becomes furniture; that is how the previous registry's red chips stopped meaning anything to anyone.

## Governing principles

1. Coverage, condition and empirical never merge, and none merges into a display status.
2. A source outage never changes coverage; a failed outcome never changes either.
3. `VALID` requires a fresh observation; nothing else may claim it.
4. Only `VALID` readings aggregate.
5. No ratio without its denominator-confidence; no composite score at all.
6. Every gap is dated, including the ones in this scenario.
7. Numerators are computed live and never stored, so the board is fresh or honestly unavailable.

## Cross-references

- [PROVENANCE-MODEL.md](PROVENANCE-MODEL.md) — how coverage and condition are rendered
- [INSTRUMENT-MODEL.md](INSTRUMENT-MODEL.md) § The six invariants — invariant 3, which this document implements
- [DATA.md](DATA.md) — the wire shape of a reading
- [SOURCE-MAP.md](SOURCE-MAP.md) — the fleet the coverage axis is joined against
- `path:scenarios/meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md` — the worked precedent
