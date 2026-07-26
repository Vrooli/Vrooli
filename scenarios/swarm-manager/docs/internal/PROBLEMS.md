# Known Problems & Technical Debt

## Open issues

No cutover-specific open issue is known.

## Measures projection convergence

The operator-facing Stats projection is canonical: it retains the incremental
event-log replay, goal-scoped snapshot, trends, ETA, review, operating-mode,
record, and session analysis. The `measures` API/CLI remains a programmatic
contract, but its individual computations have not yet all been refactored to
delegate to that projection. Do not retire a Stats field or its UI until the
measure contract has a behaviorally equivalent, projection-backed adapter and
parity tests. Sandbox adoption remains correctly owned by
`agent-manager metrics-sandbox-adoption`.

## Remaining Connect migration

The 15-command `measures` group is fully Connect-backed and is the completed
rank-3 migration slice. The typed services that already preserve a command's
full behavior are likewise Connect-backed. The remaining local bindings are
intentional until their contracts cover the complete existing operation:
backlog (rank 1), execution and review (rank 2), sessions and captures (rank
4), scenarios and settings (rank 5), operations/agent-manager/portfolio (rank
6), and the greenfield records, proposals, search, prompts, and queue surfaces
(ranks 7–10). In particular, `backlog list`, `backlog get`, and
`backlog delete` are now typed, while current
`BacklogService.CreateItem` remains a cross-scenario triage contract rather
than a replacement for attachment-aware operator creation; the seam is
documented in `SEAMS.md`.

## UX issues

- **Resolved 2026-07-22 — detail-view consistency:** Goal and backlog detail tabs now share `CompactTabBar`; cross-lens actions render after the tabs and only from Overview/Info. Goal files use the same full-width editable workspace as backlog files.
- **Resolved 2026-07-22 — nested surfaces:** Milestones render as individual persisted disclosures with flat dividers rather than a card surrounding a grid of cards.
- **Resolved 2026-07-22 — plan empty state:** The absent-plan state provides a direct `plan.author` launch action and takes the operator to Activity to follow the execution.
- **Resolved 2026-07-23 — mobile detail deep links:** Direct mobile links retained the persisted open desktop sidebar, which is a full-screen overlay below 768px and hid the detail pane. `AppShell` now collapses the sidebar on entering the mobile breakpoint.
- **Intentional integrity guard:** `goal.json` is viewable but protected from raw file mutations because it is canonical goal graph state. All goal artifacts, including migrated material, remain editable through the file workspace.

## E2E verification

- 2026-07-22: BAS execution `3b484aad-32c9-4622-942a-45ffb99126b7` completed the read-only Plan Workshop policy journey in
  [`bas/cases/plan-workshop/open-policy.json`](../../bas/cases/plan-workshop/open-policy.json). It opened the Graph workspace
  settings drawer, selected Plan Workshop, and verified the explicit-review policy states that there are no automatic rounds
  or readiness controls. Screenshot evidence is stored at
  `/home/matthalloran8/.vrooli/data/vrooli/browser-automation-studio/recordings/3b484aad-32c9-4622-942a-45ffb99126b7/artifacts/screenshots/00001--adhoc--open-graph-workspace--ACTION_TYPE_NAVIGATE.png`.
- 2026-07-23: BAS execution `46403ad3-a14e-42e9-8c36-bf6cd4ac2b3a` captured `/goals/ai-image-generation-foundation` at 390x844 after a 5-second settle. The goal detail is visible with the sidebar collapsed and no console output. Desktop confirmation is `c57632af-5608-4527-835f-69199e1d7428` at 1440x900.

## Historical notes

Prior operating-mode and agent-operation implementation notes were removed with
the runtime. Their persisted data is retained as read-only provenance, and
migration evidence is kept under
[operations/migration](../operations/migration/).
