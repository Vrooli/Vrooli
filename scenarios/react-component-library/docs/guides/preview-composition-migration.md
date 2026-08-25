# Preview composition history

This document is retained as a historical pointer only. The preview-composition
migration workflow is retired.

The repository is greenfield at the current Story Contract v4 boundary:

- `story.json` files are authored directly against the v4 contract.
- Invalid or obsolete shapes are corrected in source rather than converted at
  runtime.
- No story migration command, compatibility parser, migration ledger, or
  repository-wide backfill remains in the active implementation.
- Released asset bytes remain immutable; version cleanup is a separate,
  hash-bound retention operation.

Use the current authoring and review guidance instead:

- [Story Contract](../concepts/STORY-CONTRACT.md)
- [Asset preview composition](asset-preview-composition.md)
- [Asset update flow](asset-update-flow.md)

The checked-in inventory and state files under `docs/evidence/` are evidence of
the current authored corpus. They are not migration instructions. Counts in
those files are time-dependent measurements and must not be copied into tests
or permanent prose as fixed catalog requirements.
