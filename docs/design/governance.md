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

## Compatibility And Extensions

Design kits should expose official-style top-level token groups (`colors`, `typography`, `rounded`, `spacing`, and `components`) so generic `DESIGN.md` tools and agents can understand the file. Vrooli-specific blocks such as `tokens` and `constraints` are allowed as extensions for adapters and generation policy, but they must be additive and consistent with the official-style groups.

Unknown Markdown sections are allowed, but the Vrooli-required UX state contract must remain present. Keep feedback, loading, error, empty, stale, partial, retry, and disabled-state guidance in the canonical `DESIGN.md`, not only in component code or scenario docs.

## Validation Expectations

Run local validation before merging design-kit changes:

```bash
template-manager design validate --all
```

Validation should reject missing kit files, invalid adapter copy rules, missing official-style token groups, and missing Vrooli UX state guidance. Official Google `DESIGN.md` tooling is alpha and optional for local development, but design-kit authors should keep the top-level token shape compatible where practical.
