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

## Tunnel Manager UI surfaces

> The experience contract is now authored under `experience/`, but all pages
> are intentionally `draft` until the visual implementation reconciles. The
> target is the Exposure Command Center: a meaningful Exposure Constellation
> on Overview, a guided bounded exposure flow, explicit duration and
> authentication scope, and a visible `/public/*` boundary. See
> [`../concepts/UX-DIRECTION.md`](../concepts/UX-DIRECTION.md).

| Surface | Feature folder | Shows |
|---|---|---|
| **Overview** | `ui/src/features/overview/` | Exposure Constellation, setup readiness, route counts, actionable findings, and the dominant expose action. |
| **Exposure** | `ui/src/features/exposure/` | Searchable route inventory and guided choose/configure/confirm lease workflow with duration, policy boundary, and verification. |
| **Recovery & Events** | `ui/src/features/recovery/` | Auto-recovery state, breaker/backoff risk, next operator action, guarded manual recover/force action, and recovery event details. |
| **Metrics** | `ui/src/features/metrics/` | Local→ingress→public route journey diagnostics, cloudflared metrics, probe history, classifications, and scrape/probe actions. |
| **Audit** | `ui/src/features/audit/` | Governance: port compliance, status filters, ownership context, and remediation hints. |
| **Drift** | `ui/src/features/drift/`, `ui/src/features/routes/` | Live ingress versus manifest comparison and explicit adoption, ignore, or prune actions. |
| **Settings / Setup** | `ui/src/pages/SettingsPage.tsx`, `ui/src/api/config.ts` | Local/remote mode, Cloudflare credential readiness, final config policy guidance, write-only Cloudflare credential save/clear through the Vrooli credential authority, field source status, missing fields, local config path, sync dry-run, sync apply, theme, and locale. |

The scaffold `health` feature lives at `ui/src/features/health/`.

### Durable seams (preserve across UI rewrites)

The visual layout is replaceable, but these seams are binding and must
survive any redesign:

- **i18n** — `SUPPORTED_LOCALES`, `useTranslation`, `setLocale`; strings
  in `ui/src/i18n/locales/<locale>.json` (`i18n-strings` slot).
- **Accessibility** — `role`, `aria-*`, and `data-testid` selectors;
  WCAG AA contrast (operators need quick, color-coded glances).
- **Design tokens** — vrooli-default tokens (`bg-app-background`,
  `rounded-panel`, etc.) for light/dark; status semantics
  green/yellow/red.
- **Feature-folder pattern** — one folder per domain under
  `ui/src/features/<domain>/`, following the proto → API → CLI → UI
  vertical slice.
- **Experience contract** — `experience/index.json` is the route and journey
  registry; page specs remain draft until the implementation and evidence are
  ready. Do not add machine claims or fake bindings for the future visual
  before it exists.

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
