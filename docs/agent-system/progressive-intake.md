# Progressive Work-Product Intake

Adaptive capture is for agents and interactive workflows. It preserves useful
input without making an incomplete observation look published. Automation that
requires a deterministic success/failure boundary continues to use strict
`create` commands.

Every adaptive endpoint returns one of two dispositions:

- `published`: the supplied payload was complete and valid; the canonical
  artifact is now visible to its normal readers.
- `draft`: the supplied input was durably retained for repair, but is private:
  it must not enter search, a learning corpus, a bug inbox, or publication
  events.

The response shape is stable across work products: `accepted` contains only
safe canonical normalization; `needs` names missing fields; `invalid` preserves
the raw invalid value and its correction guidance; `warnings` carries
non-blocking information; and `next_action` is an argv array for the precise
repair command. Unknown enum values are never guessed or rewritten.

`edit`/repair merges supplied fields with the retained raw draft. If the merged
payload becomes valid, it publishes exactly once under the same draft ID. A
still-incomplete repair remains a draft. Retrying either request is safe: a
published artifact is not duplicated.

Current ownership:

- Swarm Manager owns learning-record capture (`records capture`) while strict
  `records create` stays fail-fast.
- Scenario QA owns taxonomy-complete bug intake through `prompt-manager team
  bug-capture scenario-qa`; its drafts never use `bug-inbox/*` topics.
- Plan Manager forwards the complete typed contracts through its `log bug-add`
  and `log record-add` commands. Its ledger preserves the downstream
  disposition, diagnostics, repair argv, and sync provenance rather than
  fabricating taxonomy or record fields.
