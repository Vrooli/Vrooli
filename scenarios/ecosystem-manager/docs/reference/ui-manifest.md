# UI Manifest Reference — Ecosystem Manager

> **Status: pre-adoption / not-applicable.** Ecosystem Manager predates
> the slot-based scenario-UI manifest and the adoption resolver. There
> is **no `.vrooli/ui-manifest.json`** in this scenario, and no overlay
> of the template manifest. The headings below are kept for contract
> parity with sibling reference docs, but they describe the canonical
> system (sourced from the react-vite template) and what adopting it
> here would entail — they do **not** describe a manifest this scenario
> currently ships.

The scenario's UI is a hand-organized React app under
[CODE: ui/src] (`components/`, `contexts/`, `hooks/`, `stores/`,
`lib/`, `consts/`, `types/`). It is not resolved through the slot
manifest.

## Contract

The canonical contract lives on the template at
`templates/scenarios/react-vite/ui/manifest.json`. A scenario adopts it
by overlaying slot `dir` values in `.vrooli/ui-manifest.json`.

| Field | Canonical value | Ecosystem Manager |
|---|---|---|
| `kind` | `scenario-ui` | — (no manifest) |
| `schema` | `scenario-ui-manifest/v1` | — (no manifest) |
| `template` | `react-vite` | structurally react-vite-like, not manifest-declared |

## Slots (v1)

These are the template-defined slots an adopting scenario would resolve
through. Ecosystem Manager does **not** declare any of them today.

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

`defaults.slot` is `shared-component`. Some of these directories
(`components`, `hooks`, `lib`, `consts`) already exist in
[CODE: ui/src]; others (`pages`, `features`, `api`, `i18n`, `theme`)
do not.

## Path-Pattern Tokens

| Token | Meaning | Example |
|---|---|---|
| `{dir}` | The slot's `dir` value | `ui/src/components` |
| `{ComponentName}` | PascalCase | `QueuePanel` |
| `{componentName}` / `{camelName}` | camelCase | `useQueueStatus` |
| `{kebab-name}` | kebab-case | `task-card` |
| `{feature}` | Feature folder (required when `requiresFeature: true`) | `auto-steer` |
| `{locale}` | Locale code (only `i18n-strings`) | `en` |

## Resolution Order (Adoption Resolver)

For an adopting scenario, the resolver picks a path in this order:

1. **Explicit override** — caller supplied a path.
2. **Template manifest** — resolves the slot and substitutes tokens.
3. **Heuristic** — manifest/slot missing; scan `ui/src/` for a matching
   directory name (warning attached).
4. **Fallback** — `ui/src/components/<ComponentName>.tsx` (warning attached).

For Ecosystem Manager today, only steps 3–4 (heuristic / fallback) would
apply, since no manifest is present — which is exactly why adopting the
manifest is the recommended next step rather than relying on heuristics.

## Overlays

An adopting scenario overrides individual slot `dir` values in
`.vrooli/ui-manifest.json` at the scenario root; overlays may not
introduce new slot names. **Ecosystem Manager ships no such overlay.**

Adopting the manifest here would mean:

1. Reorganizing [CODE: ui/src] toward the template's slot directories
   (notably introducing `pages/` and `features/` where appropriate).
2. Adding `.vrooli/ui-manifest.json` with only the `dir` overrides this
   scenario needs.
3. Validating against `.vrooli/schemas/scenario-ui-manifest.schema.json`.

## Cross-References

- Concept: [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md)
- Template contract: `templates/scenarios/react-vite/ui/manifest.json`
- Schema: `.vrooli/schemas/scenario-ui-manifest.schema.json`
- [`api-endpoints.md`](api-endpoints.md) — API the UI consumes
- [`cli-commands.md`](cli-commands.md) — CLI equivalents
- [`configuration.md`](configuration.md) — ports and env vars
