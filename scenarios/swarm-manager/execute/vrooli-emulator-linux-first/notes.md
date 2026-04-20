# Archive Note — `execute/vrooli-emulator-linux-first`

**Retired: 2026-04-18.** This item was an XL "build the emulator Linux-first" umbrella whose deliverables were fully decomposed by `research/emulator-extraction-and-service-plan` into 9 concrete sibling items under the `emulator-platform` initiative. With decomposition complete, the umbrella carried no unique direct scope; continuing to track it would have inflated the backlog with a redundant aggregate.

## Status semantics

The backlog data model has no `archived` status; valid values are `backlog, researching, ready, queued, in_progress, completed, failed`. This item is therefore marked `completed` to remove it from active queues, but **completed here means "retired" and not "shipped"**. No code was written or merged under this item — every code-facing deliverable lives in a sibling (see below).

This distinction matters for:

- **Rollup counts**: `emulator-platform` initiative will show `completed=3`, but only two of those (`research/emulator-extraction-and-service-plan`, `execute/scaffold-vrooli-emulator-scenario`) represent actually-shipped work. This one is a retirement.
- **Release notes / audit trails**: do not attribute any emulator feature to this item; attribute to the sibling(s) that actually built it.

## Round history

- **Round 1 d1=B (retire)** — user unambiguously chose retire, reinforced via "Archive this backlog item" on d2–d5.
- **Round 2 d1=A (split rehoming)** — PRD fill-out + operator runbook + operational baseline values-as-docs → `chore/vrooli-emulator-documentation`; Phase 1 integration smoke harness + distro validation matrix → `execute/emulator-acceptance-tests-phase-1`.
- **Round 2 d2=A (comprehensive rewire)** — `execute/adopt-vrooli-emulator-in-deployment-flows.depends_on` rewired to the comprehensive Phase 1 set (5 siblings added, this item removed).
- **Round 2 d3=A (archive, not delete)** — retirement mechanism: preserve plan + workshop record. Implemented as `status=completed` + this note.

## Sibling owners of the original scope

| Original deliverable | Owning sibling |
|---|---|
| Scaffold + lifecycle baseline | `execute/scaffold-vrooli-emulator-scenario` (completed) |
| `/api/v1/sessions/` contract + headless mode + app_path | `execute/scaffold-vrooli-emulator-scenario` / standalone UI |
| External-url embed protocol | `execute/vrooli-emulator-external-url-endpoint` |
| Iframe embed in scenario-to-desktop | `execute/scenario-to-desktop-emulator-iframe-embed` |
| Livedesktop removal | `chore/scenario-to-desktop-remove-livedesktop` |
| Smoketest delegation | `execute/smoketest-delegate-display-to-emulator` |
| Standalone UI | `execute/vrooli-emulator-standalone-ui` |
| Phase 1 acceptance tests + integration smoke + distro matrix | `execute/emulator-acceptance-tests-phase-1` |
| PRD + runbook + operational baseline docs | `chore/vrooli-emulator-documentation` |
| Remote-backend research | `research/vrooli-emulator-remote-backend-spike` |

## Verification artifacts

- `execute/adopt-vrooli-emulator-in-deployment-flows.depends_on` no longer references this item; see that item's `depends_on` for the rewired comprehensive Phase 1 set.
- `execute/vrooli-emulator-remote-node-backend.depends_on` (initiative: `trusted-node-bridge`) was discovered during the Phase D grep sweep — the retirement plan missed this second dependent edge. Rewired on 2026-04-18 to replace `execute/vrooli-emulator-linux-first` with `execute/emulator-acceptance-tests-phase-1` (Phase 1 readiness gate; transitively requires scaffold).
- `chore/vrooli-emulator-documentation/plan.md` and `execute/emulator-acceptance-tests-phase-1/plan.md` each contain an "Absorbed scope" section crediting this plan as the rehoming source.

See `plan.md` §9 Contract Decisions for the full authoritative record.
