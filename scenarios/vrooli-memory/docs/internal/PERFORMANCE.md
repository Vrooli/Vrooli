# Performance — Vrooli Memory

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

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet. | n/a | n/a | 2026-07-27 |

## Scenario-Specific Budgets (designed, not yet measured)

These follow from the architecture rather than from measurement. They are the
targets implementation should be held to, and the reason each exists is stated
so a future agent can tell a real regression from a wrong budget.

| Operation | Budget | Why this number | Requirement |
|---|---|---|---|
| `wake` | Bounded output, independent of corpus size | This is the whole point of the budget model — output size is a function of the configured line budget, not of how much has been remembered. A `wake` whose cost grows with corpus size means `cover()` selection is walking the journal instead of the frontier. | `VROOLIME-P0-008` |
| `note` (write) | Perceptibly immediate to the calling agent | The write path makes inference calls (classify, derive, embed). If those are synchronous and slow, agents stop writing and the whole scenario fails on adoption. Consider appending first and enriching asynchronously if the budget cannot be met. | `VROOLIME-P0-002` |
| `recall` | Comparable to other search-hub providers | Memory is one federated provider among many; a slow provider degrades every cross-corpus query it participates in. The provider descriptor declares a latency budget like any other. | `VROOLIME-P0-003`, `VROOLIME-P0-009` |
| Compaction pass | Off the request path entirely | It is a scheduled background sweep by decision (D-007 / ARCHITECTURE deviations). No user-facing operation should ever block on summarization. | `VROOLIME-P0-007` |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- **Clustering cost grows with frontier size, not corpus size.** This is the
  structural reason the frontier has a target at all. If clustering ever scans
  the full journal rather than the frontier, the scenario stops scaling and the
  budget model is broken.
- **Inference dominates the write path.** Classification, facet-text derivation,
  and embedding are all ai-gateway calls. Local model latency, not application
  code, sets the floor on `note`.
- **Embedding count multiplies with facet spaces.** `VROOLIME-P1-005` proposes
  ~3 embeddings per memory; each additional facet space is a proportional
  increase in both write-time inference and index size. The count is currently a
  guess (see `PROBLEMS.md`) and should be settled against measured clustering
  quality rather than raised speculatively.
- **Re-summarization is repeated work.** A node collapsed, then later absorbed
  into a higher summary, is summarized more than once. Generation count is
  tracked on summary nodes so this cost is visible.

## Regression Procedure

1. Run `make test`.
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
