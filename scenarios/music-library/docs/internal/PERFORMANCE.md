# Performance — Music Library

Budgets for a listener-facing application, and the constraints behind them.

## Purpose Of This Document

Use this document to answer:

- What latency does each listener-facing surface owe?
- What is slow because it is genuinely expensive, versus slow because it is queued?
- Which numbers are measured?

## Honesty statement

**Nothing has been measured.** No implementation exists. Budgets below are targets
derived from what the surfaces need to feel correct, not observations.

## Budgets

| Surface | Budget | Why |
|---|---|---|
| Transport response (play, pause, seek, skip) | < 100 ms | Below this it feels instant; above it feels broken |
| Next-track selection | < 200 ms, fully overlapped with current playback | Must never be audible as a gap |
| Gapless boundary | 0 audible gap | Binary — either correct or the feature does not exist |
| Library browse and search | < 200 ms | Interactive |
| Explanation of a recommendation | < 200 ms, same response as the recommendation | An explanation fetched separately will drift from what it explains |
| Comparison pair presentation | < 300 ms | Elicitation only works if it stays effortless |
| Preference model refit | Seconds, background, never blocking playback | Refits are frequent |
| Library scan, incremental | Background, resumable, progress-reporting | Never blocks the interface |
| Decomposition of a new library | Hours to days, background, resumable | Bounded by `music-tools`, not by this scenario |

## Current Measurements

| Measurement | Value | Confidence |
|---|---|---|
| — | none taken | — |

## Known Constraints

### The explanation must be computed with the ranking

A recommendation and its explanation come from the same scoring pass. Computing
them separately would let them disagree, which is worse than no explanation —
the whole product claim is that the reasoning is truthful. This makes explanation a
latency constraint on ranking rather than a separate feature.

### Ranking cost grows with the candidate pool, and the pool grows on purpose

Generation deliberately grows the candidate pool as a defence against feedback-loop
collapse. That means ranking cost is not stationary. Scoring must stay sub-linear in
pool size through vector-index retrieval, with exhaustive scoring reserved for a
retrieved shortlist.

### Uncertainty is more expensive than a point estimate

A Bayesian preference model is the reason exploration and explanation can be honest,
and it costs more than a point-estimate rating. Refits are therefore background,
incremental where possible, and never on the playback path.

### First run is genuinely slow, and honesty is the mitigation

A new library must be decomposed before ranking means anything, and that is bounded
by `music-tools` throughput at library scale — plausibly overnight or longer. This
cannot be engineered away here. The mitigation is truthful progress reporting and
making the comparison surface useful before decomposition finishes, not a spinner
that implies imminent completion.

### Transcoding is on the playback path

Format conversion for browser playback happens per rendition and is cached under an
LRU budget. A cache miss must not exceed the transport budget, so first-play of an
uncommon format needs a fast path or a pre-warm.

## Regression Procedure

1. Measure transport latency under a **cold** attribute cache and a **full** queue —
   the worst realistic case, not the best.
2. Measure next-track selection with the candidate pool at its budgeted maximum,
   not at seed size.
3. Assert the gapless boundary by audio analysis of the rendered output, not by
   timing instrumentation. Timing can be correct while the output is not.
4. Measure ranking and explanation as a single operation. Never benchmark them apart.
5. Re-measure after any preference-model change; representation changes move
   everything downstream.
6. Track first-run decomposition wall-clock as a product metric — it is the largest
   number a new listener will experience.

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — the preference model
- [`../concepts/DATA.md`](../concepts/DATA.md) — what is cached and what is regenerable
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — degraded behaviour
- [`PROBLEMS.md`](PROBLEMS.md) — cold start and unbuilt player fundamentals
