# Requirements

Requirement modules live here, one folder per group of operational
targets. Every requirement links back to a PRD operational target via
`prd_ref` and carries at least one validation entry pointing at its
proof.

- Statuses are earned, not asserted: auto-sync updates them from
  `[REQ:ID]`-tagged test results on comprehensive suite runs.
- Replace scaffolded manual validation stubs with test-typed entries
  (a `ref` to the test file plus the `[REQ:ID]` tag) as behavior lands.
- Validate with `business-health validate scenario <scenario>`; inspect
  traceability with `business-health matrix show <scenario>`.
