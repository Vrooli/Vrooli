# UI Manifest Reference

Stable reference for the slots declared in this scenario's own
[`ui/manifest.json`](../../ui/manifest.json). Mirrors that file. Update both
when adding or renaming a slot.

## Contract

Read from `ui/manifest.json`.

| Field | Value |
|---|---|
| `contract.kind` | `scenario-ui` |
| `contract.schema` | `scenario-ui-manifest/v2` |
| `contract.template` | `react-vite` |
| `schemaVersion` | `2.0.0` |
| `templateStructureVersion` | `react-vite/v2` |
| `targets` | `react-vite` |
| `idiom` | `react-typescript` |
| `provides` | `design-tokens`, `ui-provider` |
| `gates` | `types`, `tokens`, `lifecycle` |
| `slotCoverage` | `full` |

## Slots

A slot's `dir` is a declared destination, not a directory that necessarily
exists. Several below have no folder in `ui/src/` yet; the resolver creates one
when the first file lands.

| Slot | `dir` | Path pattern | Requires `feature`? |
|---|---|---|---|
| `ui-primitive` | `ui/src/components/ui` | `{dir}/{kebab-name}.tsx` | no |
| `shared-component` | `ui/src/components` | `{dir}/{ComponentName}.tsx` | no |
| `ui-component` | `ui/src/components` | `{dir}/{ComponentName}.tsx` | no |
| `layout-shell` | `ui/src/layout` | `{dir}/{ComponentName}.tsx` | no |
| `layout-nav` | `ui/src/layout` | `{dir}/{ComponentName}.tsx` | no |
| `page` | `ui/src/pages` | `{dir}/{ComponentName}.tsx` | no |
| `feature` | `ui/src/features/{feature}` | `{dir}` (folder) | yes |
| `feature-component` | `ui/src/features/{feature}` | `{dir}/{ComponentName}.tsx` | yes |
| `hook` | `ui/src/hooks` | `{dir}/{camelName}.ts` | no |
| `api-client` | `ui/src/api` | `{dir}/{camelName}.ts` | no |
| `lib-util` | `ui/src/lib` | `{dir}/{camelName}.ts` | no |
| `consts` | `ui/src/consts` | `{dir}/{camelName}.ts` | no |
| `i18n-strings` | `ui/src/i18n/locales` | `{dir}/{locale}.json` | no |
| `theme-token` | `ui/src/theme` | `{dir}/{kebab-name}.css` | no |
| `test-util` | `ui/src/test-utils` | `{dir}/{camelName}.ts` | no |
| `service` | `ui/src/services` | `{dir}/{camelName}.ts` | no |
| `provider` | `ui/src/providers` | `{dir}/{ComponentName}.tsx` | no |
| `adapter` | `ui/src/adapters` | `{dir}/{camelName}.ts` | no |
| `pattern` | `ui/src/patterns` | `{dir}/{ComponentName}.tsx` | no |
| `ui-pattern` | `ui/src/patterns` | `{dir}/{ComponentName}.tsx` | no |
| `page-template` | `ui/src/templates` | `{dir}/{ComponentName}.tsx` | no |
| `fixture` | `ui/src/fixtures` | `{dir}/{camelName}.ts` | no |
| `visual-recipe` | `ui/src/visual` | `{dir}/{kebab-name}.ts` | no |
| `motion` | `ui/src/motion` | `{dir}/{camelName}.ts` | no |

`ui-component` is a compatibility alias for `shared-component` in the catalog
vocabulary; both resolve to `ui/src/components`. `ui-pattern` is the same
alias relationship to `pattern`.

`defaults.slot` is `shared-component` and `defaults.fallbackDir` is
`ui/src/components`. Components that publish no slot resolve through them.

## Files

Beside slots, `files` names the individual files tooling reads or writes.
Each entry is `{ "path": "...", "description": "...", "defaultLocale"?,
"managedRegion"?: { "begin", "end" } }`. Paths are scenario-relative and may
carry `{locale}`.

| Key | Default path | Read or written by |
|---|---|---|
| `designTokens` | `ui/src/design-tokens.css` | generation (base + kit adapter); `react-component-library adoptions tokens-sync` inside the `rcl:tokens` managed region; the scenario-token-requirements gate |
| `tailwindTheme` | `ui/tailwind.theme.json` | generation (kit adapter) |
| `tokenMap` | `ui/token-map.json` | react-component-library adoption preflight and preview |
| `localeCatalogue` | `ui/src/i18n/locales/{locale}.json` (`en`) | `pnpm strings:gen`; `adoptions link` merges library strings |
| `selectorRegistry` | `ui/src/consts/selectors.ts` | `adoptions link` composes the library import; `pnpm selector:manifest` |
| `librarySelectors` | `ui/src/consts/selectors.library.ts` | written by `adoptions link` |
| `appEntry` | `ui/src/main.tsx` | `adoptions link` mounts the library strings provider |
| `stringsRegistry` | `ui/src/consts/strings.generated.ts` | `pnpm strings:gen` |

A scenario overlay may change a declared path but may not introduce a key. A
tool that finds no declaration falls back to the default path above.

## Path-Pattern Tokens

| Token | Meaning | Example |
|---|---|---|
| `{dir}` | The slot's `dir` value. | `ui/src/components` |
| `{ComponentName}` | PascalCase. | `Button`, `SidebarShell` |
| `{componentName}` / `{camelName}` | camelCase. | `useGamepad`, `errorMessage` |
| `{kebab-name}` | kebab-case. | `bottom-nav`, `error-boundary` |
| `{feature}` | Feature folder; must be supplied when `requiresFeature: true`. | `health` (the one that exists); `inventory`, `instance`, `request`, `findings`, `provider` (the ones [`../concepts/UI-ARCHITECTURE.md`](../concepts/UI-ARCHITECTURE.md) plans) |
| `{locale}` | Locale code. Only used by `i18n-strings`. | `en`, `ja`, `ar` |

## Resolution Order (Adoption Resolver)

1. **Explicit override** — caller supplied a path.
2. **Scenario manifest**: `ui/manifest.json`, mirrored by this file, resolves
   the slot and substitutes tokens.
3. **Heuristic** — manifest missing or slot missing; scan `ui/src/` for a
   matching directory name. Warning attached.
4. **Fallback** — `ui/src/components/<ComponentName>.tsx`. Warning attached.

## Overlays

A scenario may override individual slot `dir` values inside
`.vrooli/ui-manifest.json` in the scenario root. Compute Manager has no such
overlay: `ui/manifest.json` is used as generated. The overlay must not
introduce new slot names; those live on the template manifest the scenario
was generated from. (Overlay loader tracked in
`scenarios/react-component-library/PRD.md`.)

## Cross-References

- Concept: [`UI-ARCHITECTURE.md`](../concepts/UI-ARCHITECTURE.md)
- Schema: `.vrooli/schemas/scenario-ui-manifest.schema.json`
- Resolver: `scenarios/react-component-library/api/internal/adoptions/pathresolver.go`
