# Known Problems & Technical Debt

## Open issues

No cutover-specific open issue is known.

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
