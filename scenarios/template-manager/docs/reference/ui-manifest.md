# UI Manifest Reference

Shared reference for scenarios using the React-Vite UI contract. Template
Manager owns this document; scenario docs link here and describe only local differences.

## Contract

The [template UI manifest](../../../../templates/scenarios/react-vite/ui/manifest.json)
and [schema](../../../../.vrooli/schemas/scenario-ui-manifest.schema.json)
are authoritative for the current schema version, slot names, directories,
default slot, and path patterns. Read those declarations when resolving a path;
do not maintain a copied slot table in each scenario.

## Slots

A slot identifies a UI building block and its destination. A slot that declares
`requiresFeature` needs a feature name. The manifest's default slot handles
components with no declared slot. Scenario overlays can change existing slot
paths; extending the slot vocabulary belongs in the template manifest.

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
2. **Template manifest** — resolve the slot and substitute tokens. A manifest
   declaring full slot coverage rejects an unknown slot instead of guessing.
3. **Heuristic** — manifest missing or slot missing; scan `ui/src/` for a
   matching directory name. Warning attached.
4. **Fallback** — `ui/src/components/<ComponentName>.tsx`. Warning attached.

## Overlays

Scenarios may override individual slot `dir` values inside
`.vrooli/ui-manifest.json` in the scenario root. The loader merges the overlay over the template manifest. The overlay must
not introduce new slot names — those live on the template manifest.

## Cross-References

- Concept: [`UI-ARCHITECTURE.md`](../concepts/UI-ARCHITECTURE.md)
- Schema: `.vrooli/schemas/scenario-ui-manifest.schema.json`
- Resolver: `scenarios/react-component-library/api/internal/adoptions/pathresolver.go`

### Shared selectors

Selector registries import `@vrooli/ui-selectors`; their generated manifests use
its portable v1 contract. `files.selectorRegistry` identifies the application
registry, `files.librarySelectors` identifies its generated library definitions,
and `files.appEntry` identifies the provider mount. Overrides stay in the existing
scenario `.vrooli/ui-manifest.json`. The shared Go loader is now
`github.com/vrooli/api-core/uimanifest` and is used by library adoption, BAS, and
UI Health. Run `selector:manifest` after changing definitions; `selector:check`
verifies freshness. See [the shared selector contract](../../../../packages/ui-selectors/README.md).
