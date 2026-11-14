# Requirements Registry

Organize requirement modules by PRD operational targets, keeping the filesystem structure aligned with the "what" articulated in the PRD. Create folders such as `01-<target-name>/` as needed (numbers preserve ordering but do **not** imply priority).

This scenario's registry is organized by criticality, one module per PRD tier:
`01-must-ship/` (P0, 14), `02-post-launch/` (P1, 9), and `03-future/` (P2, 5).
The template's starter `01-foundation/module.json` was removed once these
PRD-generated modules existed, per `docs/START-HERE.md` Gate 2.

## Lifecycle
1. Operational targets in PRD map to folders here. Every `OT-*` has exactly one
   `SIG-*` requirement and vice versa — confirm with `business-health matrix show signal-inbox`.
2. `requirements/index.json` imports each module; tests auto-sync their status when they run.
3. Coverage summaries live in `coverage/phase-results/` after each test phase.

## Validation
Each requirement carries a `validation` array declaring the evidence that will
prove it. Every entry has a `type` (`test` or `manual`), a `status`, and for
tests a `phase` (`unit`, `integration`, `ui`, `search`). Entries state the
*intended* evidence and deliberately assert no `ref` until the referenced test
exists — a `ref` pointing at a file that has not been written is a false claim
of coverage, not a placeholder.

While the scenario is unimplemented, every requirement is `status: planned` and
every validation entry is `status: planned`. That is accurate rather than
pending cleanup. Auto-sync flips a validation entry once a test tagged
`[REQ:SIG-…]` runs against it; do not hand-edit statuses to reflect intent.

Two rules the validation entries encode on purpose:
- Prefer `test` over `manual`. `manual` is reserved for judgment that cannot be
  automated — reviewing classification quality, confirming a platform's terms,
  or sign-off on an action carrying unrecoverable risk.
- A requirement whose failure mode is silent gets an explicit negative test.
  `SIG-P0-003` (empty extraction) and `SIG-P0-014` (retry after a soft block)
  both exist because the bad outcome otherwise looks like success.

## Contributor Notes
- Add folders/modules that match your scenario’s PRD targets (P0/P1/P2) instead of reusing other scenarios’ names.
- Tag tests with `[REQ:ID]` so auto-sync can update status.
- Never add compatibility shims (duplicate folders or alias imports) during migrations—let things fail temporarily instead of adding debt.
- Keep this README under 100 lines. Use `scenarios/test-genie/docs/reference/requirement-schema.md` for schema details and `scenarios/test-genie/docs/phases/business/requirements-sync.md` for auto-sync behavior.
