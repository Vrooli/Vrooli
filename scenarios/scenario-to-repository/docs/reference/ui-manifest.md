# UI Manifest Reference

Stable reference for the slots declared in
`templates/scenarios/react-vite/ui/manifest.json`. Mirrors the file. Update
both when adding or renaming a slot.

## Contract

| Field | Value |
|---|---|
| `kind` | `scenario-ui` |
| `schema` | `scenario-ui-manifest/v1` |
| `template` | `react-vite` |

## Slots (v1)

| Slot | `dir` | Path pattern | Requires `feature`? |
|---|---|---|---|
| `ui-primitive` | `ui/src/components/ui` | `{dir}/{kebab-name}.tsx` | no |
| `shared-component` | `ui/src/components` | `{dir}/{ComponentName}.tsx` | no |
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

`defaults.slot` is `shared-component` — components that publish no slot resolve
through this slot.

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
| `{feature}` | Feature folder; must be supplied when `requiresFeature: true`. | `health`, `<your-domain>` |
| `{locale}` | Locale code. Only used by `i18n-strings`. | `en`, `ja`, `ar` |

## Resolution Order (Adoption Resolver)

1. **Explicit override** — caller supplied a path.
2. **Template manifest** — this file resolves the slot and substitutes tokens.
3. **Heuristic** — manifest missing or slot missing; scan `ui/src/` for a
   matching directory name. Warning attached.
4. **Fallback** — `ui/src/components/<ComponentName>.tsx`. Warning attached.

## Overlays

Scenarios may override individual slot `dir` values inside
`.vrooli/ui-manifest.json` in the scenario root. The overlay must not introduce
new slot names — those live on the template manifest. (Overlay loader tracked
in `scenarios/react-component-library/PRD.md`.)

## Cross-References

- Concept: [`UI-ARCHITECTURE.md`](../concepts/UI-ARCHITECTURE.md)
- Schema: `.vrooli/schemas/scenario-ui-manifest.schema.json`
- Resolver: `scenarios/react-component-library/api/internal/adoptions/pathresolver.go`
