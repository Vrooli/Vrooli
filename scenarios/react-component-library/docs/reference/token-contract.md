# Design-token contract for adopted assets

Only the boundaries checked below are current-behavior claims; other recommendations are **design intent**. See the [behavior claim register](../internal/TESTING.md#behavior-claim-register).

## Shared authority

The consumer-resolved BaseStyles/1 release authors shared visual values. `ui/scripts/design-token-generate.mjs` parses its static CSS template and generates `ui/src/design-tokens.css`, `ui/src/theme/tailwind.theme.json`, and the Tokens module projection. `ui/token-map.json` supplies utility aliases rather than duplicate values. Run `node ui/scripts/design-token-generate.mjs --check` from the scenario root. It checks drift, missing aliases and app/library collisions; `ui/scripts/design-token-generate.test.mjs` checks parsing and generation.

The generator copies root/theme declarations, not component-specific descendant selectors. A Tokens change is generated with `--tokens-out` to a temporary file, then written and published through the Tokens draft lifecycle. The generator refuses to overwrite an immutable Tokens release. Follow the [edit, rebuild and look loop](../guides/asset-update-flow.md#edit-rebuild-and-look).

## App-local split

`ui/src/app-tokens.css` authors application geometry in the `--layout-*` family and the legacy keyboard-inset value. It cannot redeclare a BaseStyles token. Tailwind aliases may reference either set. `ui/src/lib/designTokens.ts` is a compatibility export, not a value authority.

The 2026-09-08 migration chose library values for shared-name conflicts, promoted missing shared roles to BaseStyles, retained application geometry separately, and removed unused or invalid declarations. Its dated classification is retained with the make-the-library-observable plan artifacts. Current counts come from the token check.

## Consumer adoption boundary

Other scenarios own their token mapping and template-declared destination. Token preflight and explicit managed-region sync/prune operations are checked by token-related tests in `api/internal/adoptions/` and `api/internal/components/indexer_test.go`. Generated-token ownership guards protect this scenario's generated projection. This does not certify every consumer palette or kit overlay.

The previous `_base`/Go-generator ownership description is superseded for this scenario's runtime values.

## Host viewport contract

The `--rcl-viewport-height`, `--rcl-safe-top/right/bottom/left`, and `--rcl-keyboard-inset` names let hosts report usable viewport facts. BaseStyles supplies defaults. Correct host reporting is an integration obligation, not an assertion that every consumer fulfills it. See [style ownership](style-ownership.md).
