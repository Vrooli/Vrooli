# Requirements

Requirement modules live here, one folder per group of operational
targets. Every requirement links back to a PRD operational target via
`prd_ref` and carries at least one validation entry pointing at its
proof.

| Module | Priority | Covers |
|---|---|---|
| `01-must-ship/` | P0 | Identity registry and versioning, canon import, spec composition and validation, render lifecycle, provenance, cost, the asset library, the conformance and credential gates, the reference surface, and the workbench. |
| `02-post-launch/` | P1 | Multi-frame binding, video, capture, compositing, automated scoring, spend budgets, character sheets, federation, look reuse, regeneration, and agent invocation. |
| `03-future/` | P2 | Persona voice, test variants, drift monitoring, and desktop packaging. |

## Validation

- Statuses are earned, not asserted: auto-sync updates them from
  `[REQ:ID]`-tagged test results on comprehensive suite runs. Every
  requirement here is `planned` because no implementation exists yet.
- Replace the scaffolded validation stubs with test-typed entries — a
  `ref` to the test file plus the `[REQ:ID]` tag — as behavior lands.
- Validate the registry with
  `vrooli scenario requirements validate asset-studio`, or
  `business-health validate scenario asset-studio` for the full
  business-contract check.
- Inspect traceability with `business-health matrix show asset-studio`.
- Coverage summaries land in `coverage/phase-results/` after each phase.
- Do not hand-edit PRD checkboxes; they sync from this registry.

## Contributor notes

- Add modules that match this scenario's own PRD targets; do not reuse
  another scenario's module names.
- Never add compatibility shims (duplicate folders or alias imports)
  during migrations — let things fail temporarily instead of adding debt.
- Schema details: `scenarios/test-genie/docs/reference/requirement-schema.md`.
  Auto-sync behavior: `scenarios/test-genie/docs/phases/business/requirements-sync.md`.
