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

---

<!-- Prior content preserved below by the requirements_readme fixer. -->

# Requirements Registry

Organize requirement modules by PRD operational targets, keeping the filesystem structure aligned with the "what" articulated in the PRD. Create folders such as `01-<target-name>/` as needed (numbers preserve ordering but do **not** imply priority).

## Lifecycle
1. Operational targets in PRD map to folders here.
2. `requirements/index.json` imports each module; tests auto-sync their status when they run.
3. Coverage summaries live in `coverage/phase-results/` after each test phase.

## Contributor Notes
- Add folders/modules that match your scenario’s PRD targets (P0/P1/P2) instead of reusing other scenarios’ names.
- Tag tests with `[REQ:ID]` so auto-sync can update status.
- Never add compatibility shims (duplicate folders or alias imports) during migrations—let things fail temporarily instead of adding debt.
- Keep this README under 100 lines and link to shared docs (`docs/testing/guides/requirement-tracking-quick-start.md`) for schema details.
