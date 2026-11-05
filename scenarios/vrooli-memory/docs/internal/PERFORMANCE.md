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
| 395-source Claude Code reconciliation | 393 unchanged sources skipped before inference; 2 new entries appended | Live `harness import` + `import-status` run `9823e3d0-97e3-4199-ac25-fa5ef6f1d1d2` | 2026-07-27 |

## Scenario-Specific Budgets (designed, not yet measured)

These follow from the architecture rather than from measurement. They are the
targets implementation should be held to, and the reason each exists is stated
so a future agent can tell a real regression from a wrong budget.

| Operation | Budget | Why this number | Requirement |
|---|---|---|---|
| `wake` | Bounded output, independent of corpus size | This is the whole point of the budget model — output size is a function of the configured line budget, not of how much has been remembered. A `wake` whose cost grows with corpus size means `cover()` selection is walking the journal instead of the frontier. | `VMEM-P0-008` |
| `note` (write) | Perceptibly immediate to the calling agent | The write path makes inference calls (classify, derive, embed). If those are synchronous and slow, agents stop writing and the whole scenario fails on adoption. Consider appending first and enriching asynchronously if the budget cannot be met. | `VMEM-P0-002` |
| unchanged harness import | No inference calls | The importer checks the content-addressed import key before classification and three embeddings. Reconciliation cost is a local indexed lookup per source, not a full re-enrichment. | `VMEM-P0-011` |
| `recall` | Comparable to other search-hub providers | Memory is one federated provider among many; a slow provider degrades every cross-corpus query it participates in. The provider descriptor declares a latency budget like any other. | `VMEM-P0-003`, `VMEM-P0-009` |
| Compaction pass | Off the request path entirely | It is a scheduled background sweep by decision (D-007 / ARCHITECTURE deviations). No user-facing operation should ever block on summarization. | `VMEM-P0-007` |

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
- **Embedding count multiplies with facet spaces.** `VMEM-P1-005` proposes
  ~3 embeddings per memory; each additional facet space is a proportional
  increase in both write-time inference and index size. The count is currently a
  guess (see `PROBLEMS.md`) and should be settled against measured clustering
  quality rather than raised speculatively.
- **Import is asynchronous and single-flight per runtime.** A caller receives a durable run ID immediately and observes counters through `GetImportStatus`. This prevents client timeout from being mistaken for failure and prevents duplicate concurrent scans.
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
