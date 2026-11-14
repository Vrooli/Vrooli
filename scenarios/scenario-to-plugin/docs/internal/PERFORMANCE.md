# Performance — Scenario to Plugin

Performance budgets, the constraints that set them, and how a regression
is caught.

The governing observation: **this scenario is not latency-sensitive, and
pretending otherwise would push it toward the wrong tradeoffs.** A
publication happens rarely and matters enormously. Correctness, evidence
completeness, and fail-closed behavior outrank speed at every gate. The
one place performance genuinely matters is interactive: an operator
reading a readiness board or a finding list should never wait.

## Purpose Of This Document

Use this document to answer:

- What is fast enough, and what is deliberately allowed to be slow?
- Which constraints are inherent and which are ours to fix?
- How is a regression detected?

## Budgets

### Interactive surfaces — tight

| Surface | Budget | Rationale |
|---|---|---|
| Readiness board (fleet, ~130 scenarios) | p95 < 500 ms | The operator's entry point. A slow board discourages the check that prevents bad publications. |
| Package detail / gate ladder | p95 < 300 ms | Read from local records; no external call. |
| Finding list (up to ~500 findings) | p95 < 300 ms | Local rows. Findings carry offsets, not bodies, so payloads stay small. |
| Rehearsal journey view | p95 < 400 ms | Reference resolution only; logs stream separately. |
| Publication history | p95 < 300 ms | Local rows. |
| Any read-only API call | p95 < 250 ms | These never touch the network or the sandbox. |
| UI first contentful paint | < 1.5 s on a cold load | Design-kit floor. |

### Pipeline stages — deliberately unbudgeted for latency

| Stage | Expectation | Why no tight budget |
|---|---|---|
| Composition | Seconds | Bounded by artifact size and digest computation. |
| Conformance | Seconds | Every rule runs even after the first failure, so one pass yields the full finding list. Trading completeness for speed here would make remediation iterative. |
| Attestation | Tens of seconds to minutes | Dominated by third-party scanners and Sigstore. Not ours to optimize, and rushing it means scanning less. |
| Rehearsal | **Minutes** | Real sandbox provisioning, a real install run twice, and real command execution. This is the point. |
| Publication | Seconds to minutes | Network-bound, plus confirmation-by-retrieval which deliberately adds a round trip. |

The correct budget for a pipeline stage is a **timeout**, not a latency
target — a bound past which the stage is declared `unavailable` rather
than allowed to hang.

| Stage | Timeout | On timeout |
|---|---|---|
| Conformance | 5 min | `failed` — a rule set that cannot finish is a bug. |
| Attestation scan | 15 min | `unavailable` — not a package failure; the scanner did not answer. |
| Rehearsal (total) | 30 min | `unavailable`, sandbox torn down. Never `failed`: the package was not judged. |
| Rehearsal (single command) | 2 min, overridable per declaration | Command recorded as timed out; rehearsal continues to collect the rest. |
| Channel push | 10 min | `failed` for that channel; other channels unaffected. |
| Confirmation retrieval | 2 min, retried | Publication stays `unconfirmed`, which is alertable. |

`unavailable` versus `failed` is the important distinction and it is a
correctness property, not a cosmetic one. A timed-out rehearsal must never
be recorded as a package failure — the package was not judged, and marking
it failed would attribute an infrastructure problem to an author.

## Current Measurements

**None.** This scenario is pre-implementation; no measurement exists.

Every number above is a design budget, not an observation. Do not cite
them as measured performance, and replace this section with real
percentiles from a real run before any release claims performance
characteristics.

## Known Constraints

| Constraint | Nature | Response |
|---|---|---|
| Sandbox provisioning dominates rehearsal wall-clock | Inherent | Do not pool or reuse sandboxes to save time. Reuse would weaken the isolation claim, which is the entire value of the stage. |
| Scanners and Sigstore are third-party | External | Budget as timeouts; report `unavailable` rather than degrading the verdict. |
| Rehearsals are serialized per package | Ours | Sequential by design: a package's stages are ordered. Different packages may rehearse concurrently, bounded by sandbox capacity. |
| Fleet readiness is O(scenarios) | Ours | Cache derived readiness; it is recomputable and safe to prune. Recompute on declaration change, not per request. |
| Digest computation over artifact trees | Inherent | Linear in artifact size; artifacts are small (text plus a small install script). |
| Publication history grows monotonically | Ours | Never pruned, by design. Index by plugin and channel; the table stays small because publications are rare. |

## Regression Procedure

1. Confirm the regression is real: re-run against the same fixture package
   on an otherwise idle host. Rehearsal timings are noisy because they
   involve real sandbox provisioning.
2. Separate interactive from pipeline. An interactive regression is a
   defect; a pipeline regression may be a third-party change.
3. For interactive regressions, check for an N+1 across the pipeline
   domains first — the gate ladder joins six domains and is the most
   likely place a per-package query multiplies.
4. For rehearsal regressions, check `sandboxes_leaked` before profiling.
   Accumulated sandboxes degrade the host and look like a slowdown.
5. **Never resolve a regression by weakening a gate, shortening a rule
   set, reusing sandboxes, or skipping confirmation-by-retrieval.** If the
   only available fix trades away a check, the correct outcome is to
   accept the slower path and record why here.
6. Record the finding in [`PROBLEMS.md`](PROBLEMS.md) with the measurement.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — the histograms and gauges named above
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md) — stage ordering, cancellation, and teardown
- [`../concepts/DATA.md`](../concepts/DATA.md) — retention, which bounds table growth
- [`PROBLEMS.md`](PROBLEMS.md) — open performance gaps
