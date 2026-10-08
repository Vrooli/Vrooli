---
name: "test-genie-improve"
description: "Regulate Test Genie admission, cache, retry, wait, and receipt reliability from bounded producer evidence without lowering validation gates."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["test-genie","improve","admission","cache","retry","receipt"]
  icon: "gauge"
  status: "active"
  revision: 1
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["test-genie", "agent-manager"]
    commands: ["test-genie validation", "test-genie runs", "program-runtime library run"]
  origin: {kind: "authored"}
---
## Practice focus: Test Genie Improve

Regulate Test Genie from its durable receipts and Agent Manager friction. Fix
the owning invariant when clients repeatedly compensate for admission, cache,
retry, wait, or receipt defects.

Required reading: `prompt-manager skill read improvement-do-and-dont`,
`prompt-manager skill read scenario-work-ladder`, and
`prompt-manager skill read test-genie`.

### Scope and setpoint

In scope: admission decisions, dedup/cache outcomes, retry classifications,
single-wait behavior, receipt completeness, and retention. Out of scope:
lowering a phase gate, relabelling unavailable as pass, or repairing consumer
logic to hide a producer defect.

| Row | Desired band | Route |
|---|---|---|
| receipt-terminal | Every accepted intent reaches terminal or a typed active state | Missing transition is a Test Genie invariant defect. |
| receipt-evidence | Every terminal receipt names bounded producer evidence | Missing locator is a receipt contract defect. |
| admission-classified | Every refusal is terminal or retryable with a reason | Missing class is an admission contract defect. |
| retry-amplification | Zero repeated submissions for one dedup key | Fix attach/explain behavior before client prose. |
| cache-honesty | Cache reuse identifies source evidence and freshness | Unattributed reuse is a cache defect. |
| wait-friction | Zero recurring polling fingerprints | Improve the wait/receipt operation, then simplify skills. |

### Cycle

Run `test-genie.validation-digest` with a bounded limit. Read Agent Manager
friction for `test-genie`. Select the highest safety or terminality gap, route
it with the table, apply one coherent change, and repeat the same sensor. Record
before, after, unavailable rows, and the next gap.

### Output expectations

Report each row as `in_band`, `out_of_band`, or `unavailable` with evidence.
Never claim improvement from a lower gate, fewer tests, or discarded receipts.

### Troubleshooting & Edge Cases

- Sparse data: enforce invariants; do not claim a trend.
- Stale server: restart through `make stop` and `make start` before measuring.
- Producer storage unavailable: preserve the failed read as evidence.
