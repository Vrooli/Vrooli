# Performance — Source Ledger

This document records performance budgets, current measurements, known
constraints, and regression procedures.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| UI build | 5-10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |
| API health | responsive under lifecycle health timeout | `/health` check | active |
| UI health | responsive under lifecycle health timeout | `/health` check | active |
| Wake per-entry excerpt | first of line or character ceiling | wake response counters | active |
| Wake whole view | first of line or character ceiling | `lines_used`, `chars_used` | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Warm recall p95 (5 sequential runs) | 1.16 s | `vrooli-memory recall recall "durable shared memory and scope" --scope agent-memory --limit 5 --json`; steady samples 1.12–1.16 s after a 1.87 s cold sample | 2026-08-07 |
| Warm wake p95 (5 sequential runs) | 0.93 s | `vrooli-memory wake --scope agent-memory --budget 120 --json`; samples 0.85–0.93 s | 2026-08-07 |
| Direct source-ledger recall p95, agent-memory | 0.21 s | Five managed Connect requests after phase 19; samples 0.19–0.24 s | 2026-08-08 |
| Direct source-ledger recall p95, team:marketing-crew | 0.09 s | Five managed Connect requests over 305 imported entries | 2026-08-08 |
| Direct source-ledger recall p95, phase10-validation-20260806 | 0.08 s | Five managed Connect requests | 2026-08-08 |
| Direct source-ledger wake p95, agent-memory | 0.06 s | Managed Connect request with 256-line budget | 2026-08-08 |
| vrooli-memory client recall p95, agent-memory | 1.31 s | Five `vrooli-memory recall` requests; samples 1.21–1.31 s after source-ledger cutover | 2026-08-08 |

### Federation budget investigation — 2026-11-04

Search Hub's 7-day `source-ledger.scopes` provider telemetry recorded 2,204
routed calls, 1,941 degraded calls (88.1%), 1,142 timeout reasons, and 4,007
ms provider p95. Source Ledger's scenario health was green during the check.
One end-to-end `vrooli-memory recall` command took 5.134 seconds. A retained
fixed-query-embedding benchmark measured 1.638 seconds p95 for SQL/vector
recall at 22,509 nodes. The latter excludes live query embedding and is not a
like-for-like measurement, so it cannot identify which segment dominates the
current provider timeout.

The request currently permits embedding execution for up to 15 seconds, while
Search Hub's configured provider leg budget is four seconds. This is a
cross-scenario deadline mismatch, not proof that embedding alone causes the
timeout. The recall queries now restrict latest-assignment work to the active
scope and add indexes for latest-assignment lookup, mark lookup, and scoped
time ordering. The owner build and lifecycle restart completed healthy on
2026-11-04 (build identity
`sha256:eec7ed5d39d6f9ed449b1e27089e2f8b001c09c1ea77a8923b3f9b3fa6325738`).
No post-change recall timing or SQL plan was captured, so no latency
improvement is claimed. The next useful measurement separates embedding,
candidate loading, vector scoring, and serialization under the same query and
corpus conditions, then compares provider p95 against the four-second budget.

### Three-times corpus measurement

**2026-10-28 qualification:** The earlier benchmark copied only the main
SQLite file, so committed data in an active WAL could be omitted; the WAL
state at the time of the 2026-08-08 measurement is unknown. The historical
numbers below are retained, but do not establish a complete live-corpus
baseline. The benchmark now takes a consistent SQLite snapshot including WAL
data and uses that same snapshot for both comparisons. A current scale-one
screen and its limits are retained in the research cohort's
`evidence/search-and-usage/recall-wal-snapshot-benchmark.json`.

The disposable Go benchmark `api/cmd/measure-recall` copies the live database,
retains the 1,294-entry compaction canopy, and adds packed-blob synthetic
entries until the measured scope is exactly three times its current size. It
never writes the authority. On 2026-08-08, `agent-memory` measured 3,507
entries and 1,294 summaries at baseline, then 10,521 entries and 1,294
summaries at three times size. Direct recall p95 was 0.188 s at baseline and
0.377 s at three times size, both inside the 1.50 s post-extraction ceiling;
the fixed fixture returned the same top entry ID
`0005522f-b40d-4cde-b1b6-d9a671cfd590`. No SQL prefilter or ANN index was
needed. The measurement uses the same packed vector representation as the
live corpus, so JSON decoding overhead is not hidden in the fixture.

These are the pre-extraction baselines against the live managed `vrooli-memory`
scenario after the compaction canopy was restored. They are not fixture or
empty-forest measurements.

## Post-Extraction Ceilings

| Surface | Ceiling | Rationale |
|---|---:|---|
| Warm wake p95 | 1.20 s | Allows a bounded cached discovery/network hop over the 0.93 s baseline. |
| Warm recall p95 | 1.50 s | Allows a bounded cached discovery/network hop over the 1.16 s baseline. |

## Context ceilings

The default bounded-view policy is 96 lines / 12,000 characters for the whole
wake and 2 lines / 200 characters for each entry. Both ceilings are enforced
at excerpt time and again at admission time. The first reached unit governs;
the response records both counters and the number refused. Pins are ordered
first but are subject to the same whole-view backstop.

The character ceiling is necessary because journal entries are commonly
single-line JSON: line counting alone would treat a 5,000-character entry as
one line and would not bound prompt cost.

The ceiling is a regression gate, not a promise that the first cold request is
warm. Cold-start and provider enrichment timings remain separately observable.

## Known Constraints

- Vite production builds may process the generated module graph and take
  several minutes.
- Performance budgets for real product workflows must be defined after
  domains and UX flows are known.

## Regression Procedure

1. Run the server-owned comprehensive suite and record its run id.
2. Capture relevant API/UI command timing.
3. For UI interaction regressions, use `ui/perf/README.md` and the
   provided capture template.
4. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
