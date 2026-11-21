# UI Architecture

## Purpose Of This Document

Describe the canonical layout of the `ui/` source tree for scenarios generated
from the `react-vite` template, and the **slot taxonomy** that lets external
tools (notably `react-component-library`'s adoption resolver) place components
without asking the user for a path.

## Tunnel Manager UI

> Status: experience contract authored; visual implementation still pending.
> The existing routes are documented as draft experience pages under
> `experience/`. The intended destination is the Exposure Command Center
> described in [`UX-DIRECTION.md`](UX-DIRECTION.md), with an Exposure
> Constellation hero, guided bounded exposure flow, explicit `/public/*`
> security boundary, and route-level diagnostics. Current work is hardening
> the existing implementation around real setup,
> exposure, diagnostics, recovery, and audit workflows. The setup/readiness
> slice is wired to the config API and surfaces Cloudflare credential
> availability, local/remote mode, sync preview, and sync apply actions. The
> Exposure slice now composes the exposure and probes APIs for search,
> CORE/LEASED filtering, reconcile feedback, lease actions, and current
> route-classification badges. The Diagnostics/Metrics slice now summarizes
> latest cloudflared metrics, per-route classifications, current
> classification limits, probe history, and manual scrape/probe actions. The
> Audit slice now summarizes fixed-port compliance with status filtering and
> remediation hints for each finding. The Recovery slice now summarizes the
> state machine, breaker/backoff risk, next operator action, manual recovery
> force semantics, and durable event details.

The dashboard should be glanceable without being shallow: operators need a
quick system picture plus direct paths to the next safe action. The current
route set is organized around the following draft experience pages under
`<Outlet />`; navigation may later group them into the five destination areas
described by the UX direction:

| Surface | Purpose | Backing domains |
|---|---|---|
| **Overview** | Exposure Constellation, readiness summary, actionable findings, and a direct expose action. | `tunnel`, `routes`, `exposure`, `recovery`, `config` |
| **Exposure** | Searchable route collection plus guided choose/configure/confirm flow with duration, access boundary, and verification result. | `routes`, `exposure`, `probes`, `config` |
| **Recovery & Events** | Recovery state machine, breaker/backoff summary, next operator action, guarded manual recover/force action, and recovery-attempt timeline. | `recovery` |
| **Metrics** | Route journey diagnostics from local listener through ingress to public probe, plus cloudflared metric summary and probe history. | `tunnel`, `probes` |
| **Audit** | Governance surface for fixed-port compliance, ownership, and remediation guidance. | `audit`, `routes`, `config` |
| **Drift** | Live ingress versus manifest comparison with explicit adoption, ignore, and destructive prune decisions. | `config`, `routes` |
| **Settings / Setup** | Mode visibility, credential source, missing `CLOUDFLARE_*` fields, local config path, metrics endpoint, sync dry-run, sync apply, and local/remote mode switching. | `config` |

### Feature-folder mapping

Each surface maps to a feature folder under `ui/src/features/<domain>/`,
matching the domain source paths in [`DOMAINS.md`](DOMAINS.md):

| Surface | Feature folder |
|---|---|
| Overview | composed from `features/tunnel/`, `features/routes/`, `features/exposure/`, `features/recovery/` |
| Exposure | `ui/src/features/exposure/` (+ `features/routes/`, `features/probes/`) |
| Recovery & Events | `ui/src/features/recovery/` |
| Metrics | `ui/src/features/metrics/` (the `tunnel` domain's UI surface) + `features/probes/` |
| Audit | `ui/src/features/audit/` |
| Settings / Setup | `ui/src/pages/SettingsPage.tsx` + `ui/src/api/config.ts` |

Typed Connect-RPC clients live in `ui/src/api/<domain>.ts`. Real-time
updates use redis pub/sub when available and fall back to HTTP polling
otherwise (see [`INTEGRATIONS.md`](INTEGRATIONS.md)).

### Experience contract and visual seams

The draft page and journey contract is under `experience/`. Keep its route
registry, page purposes, states, claims, sketches, and journey IDs aligned with
the implementation as the redesign progresses. `UX-DIRECTION.md` is the
human-readable companion for the Exposure Constellation, publicness boundary,
guided expose flow, route detail, and visual evidence requirements.

### Durable seams kept from the template

The 5-surface IA reuses the template's durable UI seams unchanged:
i18n (`ui/src/i18n/`, no hardcoded strings), accessibility selectors
(`data-testid`, roles, `aria-*` registered in `ui/src/consts/`), and design
tokens / `ThemeProvider` (`ui/src/theme/`, light/dark, WCAG AA contrast,
vrooli-default kit). New work adds feature folders and slot entries; it does
not fork these cross-cutting seams.

## Source Layout

```
ui/src/
├── api/            # api-client slot — Connect-RPC wrappers
├── app/            # app-bootstrap — Providers composition and route table
├── components/     # shared-component slot — cross-cutting components
│   └── ui/         # ui-primitive slot — headless primitives (kebab-case files)
├── consts/         # consts slot — strings + selectors registries
├── features/       # feature slot — per-feature folders (one subfolder per feature)
│   └── <feature>/  # feature-component slot — components inside a feature
├── hooks/          # hook slot — reusable React hooks
├── i18n/           # i18n bootstrap
│   └── locales/    # i18n-strings slot — one JSON per locale
├── layout/         # layout-shell + layout-nav slots — AppShell, Sidebar, TopBar, BottomNav
├── lib/            # lib-util slot — framework-agnostic utilities
├── pages/          # page slot — routed pages mounted under <Outlet />
├── test-utils/     # test-util slot — render helpers, factories, a11y
└── theme/          # theme-token slot — ThemeProvider + tokens.css
```

## Slots Are A Contract

Every directory above maps to a named slot in `ui/manifest.json`. The manifest
declares the directory **and** a default path pattern (e.g.
`{dir}/{ComponentName}.tsx`), so external tools can compute the canonical
filesystem path for a new file given just the component's name and slot.

A component library that publishes `"slot": "layout-nav"` and ships
`SidebarShell` knows — without any per-scenario configuration — that the file
should land at `ui/src/layout/SidebarShell.tsx`. Override the slot's `dir` in
a scenario-level overlay if you've reorganized; the resolver will pick up the
new path automatically.

## Adoption Resolver Flow

1. Library declares the component's slot (e.g. `"slot": "layout-nav"`).
2. Resolver looks up the slot in this scenario's UI manifest (this file's JSON
   sibling).
3. Resolver substitutes path-pattern tokens (`{dir}`, `{ComponentName}`,
   `{kebab-name}`, `{camelName}`, `{feature}`, `{locale}`) and returns the path.
4. Scenarios with no manifest fall through to a heuristic (scan for the slot's
   expected dir name) and then a final fallback
   (`ui/src/components/<ComponentName>.tsx`). Both flag warnings on the
   resulting adoption record.

## Extending The Manifest

- **Add a slot.** Add an entry to `ui/manifest.json`. Keep its `dir` inside
  `ui/src/` and pick a pattern that matches your file-naming convention. The
  schema (`scenario-ui-manifest/v1`) does not enum-restrict slot names — open
  set on purpose.
- **Override a slot in a single scenario.** Drop a partial manifest at
  `.vrooli/ui-manifest.json` in the scenario root; the resolver will read it
  as an overlay over the template manifest. (Overlay support tracked in
  scenarios/react-component-library's PRD.)
- **Add a `postApply` action** (auto barrel-export, route-register,
  i18n-merge). Reserved for a future schema bump (`scenario-ui-manifest/v2`).
  Document the intent in the consuming scenario's PRD until then.

## Cross-References

- Schema: `.vrooli/schemas/scenario-ui-manifest.schema.json` (`$id:
  scenario-ui-manifest/v1`)
- Manifest: `ui/manifest.json`
- Slot reference: [`ui-manifest.md`](../reference/ui-manifest.md)
- Adoption resolver: `scenarios/react-component-library/api/internal/adoptions/pathresolver.go`
