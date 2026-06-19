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

> The 5 operator surfaces below are the production information architecture,
> organized around the exposure lifecycle (the prior UI sprawled to 7
> score-chasing pages contradicting the PRD's "minimal UI" — see
> [`../internal/DECISIONS.md`](../internal/DECISIONS.md)). The current
> production-readiness redesign has started with the first-run
> Settings/Setup workflow, config-readiness summary on Overview, and
> Exposure, Diagnostics, Recovery, and Audit operator workflow polish.

| Surface | Feature folder | Shows |
|---|---|---|
| **Overview** | `ui/src/features/routes/` | Tunnel + exposure health at a glance: the manifest (routes, tiers, leases), color-coded status. |
| **Exposure** | `ui/src/features/exposure/` | Lease management and reconcile workflow — request / extend / revoke leased exposure; CORE vs LEASED filtering; search; route-classification badges from probes. |
| **Recovery & Events** | `ui/src/features/recovery/` | Auto-recovery state, breaker/backoff risk, next operator action, guarded manual recover/force action, and recovery event details. |
| **Metrics** | `ui/src/features/metrics/` | cloudflared Prometheus summary + time-series (HA connections, RTT, errors, active streams), probe history, route classifications, current diagnostic-signal limits, and scrape/probe actions. Backed by the `tunnel` and `probes` domains. |
| **Audit** | `ui/src/features/audit/` | Port-compliance summary, status filters, findings, and remediation hints for mismatched / missing / ranged UI ports. |
| **Settings / Setup** | `ui/src/pages/SettingsPage.tsx`, `ui/src/api/config.ts` | Local/remote mode, Cloudflare credential readiness, missing fields, local config path, sync dry-run, sync apply, theme, and locale. |

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
