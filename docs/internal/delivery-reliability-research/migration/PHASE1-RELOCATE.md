# Phase 1 — relocate delivery-team homes to `.vrooli/work/delivery/`

Status: DONE · 2026-10-23 · all 9 homes relocated + enrollments repointed + team.json updated

Executed after the web-console authorization binding bug was fixed (see the reorg memo /
`agent_authorization.go`), which let an owner-approved `effort-metadata` grant perform the
repoints. 6 enrolled efforts repointed via `ReconcileEffortMetadata`; 3 non-enrolled teams
moved directly (no destination_ref to repoint). Boards verified reading the new homes.
The per-team recipe below is retained for reference / rollback.

Phase 1 moves each delivery team's working ledger ("goal home") out of the product
scenario (or scattered `docs/internal/` folder) into the central
`.vrooli/work/delivery/<team-id>/`, and repoints the agent-manager effort enrollment so
the supervisor reads the new location. **The file set is unchanged in Phase 1** (still
`GOAL.md`, `QUEUE.md`, `ACCEPTANCE.md`, `FEEDBACK.md`, `epochs/`, …); the
CHARTER/STATUS/LOG modernization is Phase 2.

## Historical scope

This is the completed relocation record, not a procedure to rerun. The Phase 1
operator-only limitation was superseded by scoped owner-approved metadata reconciliation.
Current boards/writers use CHARTER/STATUS/LOG/FEEDBACK. `effort migrate` retains explicit
legacy import with recoverable publication and operator-edit protection.

Use the [current work entry point](../../../../.vrooli/work/README.md) and
[owner reference](../../../../scenarios/agent-manager/docs/reference/effort-supervision.md).
Preserve source history and native enrollment identity; do not copy the old `GOAL.md`
sentinel or rebuild a second writable home from these historical mappings.

## Target teams (old → new)

| Team id | effortRef | Old goal home | New home |
|---|---|---|---|
| browser-automation-studio-delivery | `effort:browser-automation-studio-rehabilitation` | `scenarios/browser-automation-studio/docs/internal/goal` | `.vrooli/work/delivery/browser-automation-studio-delivery` |
| desktop-packaging-delivery | `effort:desktop-packaging-performance` | `scenarios/scenario-to-desktop/docs/internal/goal` | `.vrooli/work/delivery/desktop-packaging-delivery` |
| marketing-i16-delivery | `effort:marketing-I16` | `scenarios/content-desk/docs/internal/goal` | `.vrooli/work/delivery/marketing-i16-delivery` |
| nooch-delivery | `nooch-selected-delivery-2026-10-03` | `scenarios/nutrition-planner/docs/internal/goal` | `.vrooli/work/delivery/nooch-delivery` |
| notification-hub-delivery | `effort:notification-hub-redesign` | `scenarios/notification-hub/docs/internal/goal` | `.vrooli/work/delivery/notification-hub-delivery` |
| personal-planner-delivery | `effort:personal-planner-I18` | `scenarios/personal-planner/docs/internal/goal` | `.vrooli/work/delivery/personal-planner-delivery` |
| rcl-prompt-manager-ux-design-campaign | `effort:react-component-library-rehabilitation` | `scenarios/react-component-library/docs/internal/goal` | `.vrooli/work/delivery/rcl-prompt-manager-ux-design-campaign` |
| experience-quality-delivery | `effort:experience-quality` | `docs/internal/experience-quality-delivery` | `.vrooli/work/delivery/experience-quality-delivery` |
| external-knowledge-delivery | `effort:external-knowledge-search` | `docs/internal/external-knowledge-delivery` | `.vrooli/work/delivery/external-knowledge-delivery` |

> The remaining delivery teams (aquila-*, video/song efforts, qualification-*,
> fleet-storage, landing-page, unattended) are intentionally **not** moved: they either
> have no in-repo goal home or already keep their artifacts under `~/.vrooli/`, so they
> never polluted product source.

## Current disposition

All nine mapped product homes use the current file set. Six enrolled efforts point
at their existing CHARTER through revision-checked metadata reconciliation; three
preparation-only homes remain unenrolled. Other finite teams retain their managed
owners or historical qualification records, as listed at the work entry point.

Rollback is an owner reconciliation and preservation operation, not the inverse of
these historical moves: newer source edits, commissions and evidence must survive.
Do not revive the retired runtime format or silently redirect a current enrollment.
