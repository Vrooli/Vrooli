# Known Problems & Technical Debt

## Open issues

No cutover-specific open issue is known.

## UX issues

- **Resolved 2026-07-22 — detail-view consistency:** Goal and backlog detail tabs now share `CompactTabBar`; cross-lens actions render after the tabs and only from Overview/Info. Goal files use the same full-width editable workspace as backlog files.
- **Resolved 2026-07-22 — nested surfaces:** Milestones render as individual persisted disclosures with flat dividers rather than a card surrounding a grid of cards.
- **Resolved 2026-07-22 — plan empty state:** The absent-plan state provides a direct `plan.author` launch action and takes the operator to Activity to follow the execution.
- **Intentional integrity guard:** `goal.json` is viewable but protected from raw file mutations because it is canonical goal graph state. All goal artifacts, including migrated material, remain editable through the file workspace.

## E2E verification

- 2026-07-22: BAS execution `3b484aad-32c9-4622-942a-45ffb99126b7` completed the read-only Plan Workshop policy journey in
  [`bas/cases/plan-workshop/open-policy.json`](../../bas/cases/plan-workshop/open-policy.json). It opened the Graph workspace
  settings drawer, selected Plan Workshop, and verified the explicit-review policy states that there are no automatic rounds
  or readiness controls. Screenshot evidence is stored at
  `/home/matthalloran8/.vrooli/data/vrooli/browser-automation-studio/recordings/3b484aad-32c9-4622-942a-45ffb99126b7/artifacts/screenshots/00001--adhoc--open-graph-workspace--ACTION_TYPE_NAVIGATE.png`.

## Historical notes

Prior operating-mode and agent-operation implementation notes were removed with
the runtime. Their persisted data is retained as read-only provenance, and
migration evidence is kept under
[operations/migration](../operations/migration/).
