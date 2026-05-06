# Design Governance

`DESIGN.md` is a generation contract. Changes to a design kit can affect every future scenario generated from templates that consume that kit.

## Ownership

Design-kit changes should be reviewed as platform-level changes. They affect generator output, template expectations, UI consistency, and downstream component libraries.

## Versioning

Use semantic versioning in `templates/design/<kit-id>/metadata.json`.

- Patch: wording clarifications or token additions that do not alter existing behavior.
- Minor: additive tokens, new adapters, or compatible component guidance.
- Major: token renames, removed tokens, meaning changes, or adapter copy-path changes that require scenario edits.

## Generated Scenarios

Generated scenarios receive a root-level `DESIGN.md`. Scenario authors may add local product-specific notes to that file, but they should preserve the inherited design intent unless the scenario intentionally adopts a different design kit.

Do not create `docs/DESIGN_LANGUAGE.md` for new scenarios. The canonical file is `DESIGN.md` at the scenario root.
