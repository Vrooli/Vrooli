# Asset graph workspace report — historical pointer

This dated report is no longer the source of truth for the React Component
Library hierarchy or Preview composition model. Its durable architectural
facts now live in:

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — hierarchy,
  dependency direction, and the separation of subject, frame, harness, and
  fixture roles.
- [`../concepts/STORY-CONTRACT.md`](../concepts/STORY-CONTRACT.md) — the
  declarative story contract and author decision tree.
- [`../guides/asset-preview-composition.md`](../guides/asset-preview-composition.md)
  — canonical frame and shared-harness inventories, compatibility rules, and
  evidence requirements.
- [`../guides/preview-composition-migration.md`](../guides/preview-composition-migration.md)
  — migration classification, rollback, and adoption-boundary rules.

The measurements and asset lists in the original 2026-08-13 report were a
point-in-time investigation. They are intentionally not repeated here because
catalog counts and version inventories are derived projections that change as
assets are published. Recreate current measurements from the catalog index and
record them with the validation run that needs them.

Disposition: reduced to a historical pointer after the hierarchy and
composition facts were assigned canonical owners. The report may be removed
after downstream documentation-link checks no longer require a historical
anchor.
