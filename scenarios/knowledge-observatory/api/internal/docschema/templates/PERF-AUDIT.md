---
date: YYYY-MM-DD
scenario: <scenario-slug>
interactions:
  - <interaction-label-1>
  - <interaction-label-2>
traces:
  before: /tmp/<scenario>/perf/trace.before.json
  after: /tmp/<scenario>/perf/trace.after.json
  capture_script: /tmp/<scenario>/perf/capture.js
status: open
related_skill_run: scenario-performance-audit
---

# Perf audit: <short title>

> Drop the audit findings into the sections below. The frontmatter and the
> "## Per-component aggregation" table are validated by
> `knowledge-observatory docs audit <scenario>`. Other sections are
> recommended but not enforced — keep them, the format is what makes audits
> comparable across scenarios over time.

## Framing

- User complaint (verbatim if possible):
- Environment (browser / desktop / proxy / etc):
- Reproduction trigger (which click, which drag, which list state):

## Methodology

- Profile-mode build verified (look for `onProfilerRender` and `*Impl` in served bundle):
- Capture script: `<absolute path to capture.js>`
- Interactions exercised, in order:
- Capture configuration (viewport, timeout, etc):

## Per-component aggregation

Copy the Phase 5 aggregator output here. The table must have at least the
following columns; extra columns are allowed.

| component | count | total(ms) | avg(μs) | max(μs) |
|---|---|---|---|---|
| <component-1> | 0 | 0.0 | 0 | 0 |

## Long-task summary

| metric | before | after | delta |
|---|---|---|---|
| count |   |   |   |
| total(ms) |   |   |   |
| max(ms) |   |   |   |

## Findings

For each finding:

- **What:** file:line-anchored summary.
- **Evidence:** numbers from the per-component table or long-task summary.
- **Hypothesis:** likely cause (broken memo, O(N²) recompute, etc.).
- **Suggested next step:** out of scope to implement here.

## Recommendations + outcome

| # | Recommendation | Status | Notes |
|---|---|---|---|
| 1 |  | open / fixed / deferred |  |

## New dependencies

List any packages that the recommendations would add (so the user can
authorize them up front):

- (none)
