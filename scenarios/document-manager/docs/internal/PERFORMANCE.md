# Performance — Document Manager

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
| Retrieval query (`DOC-P0-023`) | **p95 under 800 ms** at up to **250,000 unit vectors** in one corpus | retrieval benchmark over a seeded corpus | planned |
| Tier-1 parse, per document | **p95 under 250 ms** end to end, *including* subprocess spawn | derivation benchmark against a fixture corpus | planned |
| Composer preview render (P2) | **p95 under 1.5 s** for a 20-block spec, *including* subprocess spawn | render benchmark against a fixture spec | planned |
| Batch render, per document (P2) | **p95 under 10 s** at any declared target | render benchmark | planned |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet. | n/a | n/a | 2026-08-05 |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Performance budgets for real product workflows must be defined after
  domains and UX flows are known.

### The two numbers that are not guesses

Both budgets above exist to make a *migration trigger* mechanical rather
than a judgement call. Neither is a target to optimise toward; each is a
line that, once crossed, selects a already-decided replacement.

**Retrieval — 250,000 unit vectors.** `DOC-P0-023` ships an in-process
linear scan over `float32` blobs. Its only precedent in this repo is
`vrooli-memory` at 8,181 vectors, and this scenario's premise is document
corpora: 10,000 documents at roughly 200 units each is ~2M vectors,
about 250× that precedent. The 250k line is deliberately set well below
where a linear scan becomes hopeless, so the swap happens while it is
still cheap. **Crossing it is not a bug — it is the trigger to move the
semantic half behind `ai-go/search`'s `VectorStore` onto Qdrant.** That
interface is adopted from the first commit precisely so the swap is
configuration rather than surgery; see the retrieval row in
[`DECISIONS.md`](DECISIONS.md) and the `qdrant` row in
[`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md).

**Render — two budgets, because there are two experiences.** The write
spine has one number that is felt and one that is merely waited on, and
conflating them would set the wrong bar for both. **Preview** sits inside
an edit loop: a chat turn that changes a heading and takes four seconds
to show it makes the Composer feel broken regardless of how fast the
batch path is, so 1.5 s is an interaction budget and its trigger is to
render the *affected region* rather than the whole document. **Batch** is
a background production step where 10 s per document is unremarkable, and
its trigger is a progress surface rather than an optimisation. Both are
stated *including* spawn for the same reason tier-1 parse is: no renderer
has a Go binding either, so every render is a subprocess, and the only
number a user experiences is the one with spawn in it. Neither is
measurable until a renderer is selected — see the render-toolchain rows
in [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md).

**Tier-1 parse — 250 ms including spawn.** `anydoc`'s published ~4.4 ms
is measured in-process in Rust. We have no Go binding, so every parse is
a subprocess and a text-native PDF costs two of them. The budget is
stated *including* spawn because that is the only number a user
experiences. Crossing it is the trigger to promote the handlers to a
long-lived process rather than to accept a slower free tier — the free
tier's speed is a product claim, not an implementation detail.

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
