# UI Architecture

## Purpose Of This Document

Describe the canonical layout of the `ui/` source tree for scenarios generated
from the `react-vite` template, and the **slot taxonomy** that lets external
tools (notably `react-component-library`'s adoption resolver) place components
without asking the user for a path.

## Tunnel Manager UI (planned)

> Status: documentation-first (Phase 1). The UI below is **designed, not
> built** — only the template scaffold + the fenced `notes` example exist.
> See [`PRD.md`](../../PRD.md) (OT-P1-004) and the "UI reimagined as 5
> surfaces" decision in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

The dashboard is intentionally minimal and glanceable: operators need quick
status reads, not complex interactions. It is organized as exactly **five
surfaces** (not score-chasing pages), each routed under `<Outlet />`:

| Surface | Purpose | Backing domains |
|---|---|---|
| **Overview** | Tunnel health at a glance: cloudflared status, core/leased route grid, current recovery state, active alerts. | `tunnel`, `routes`, `exposure`, `recovery` |
| **Exposure** | Unified route table — per row: tier badge (CORE/LEASED), internal + external status, local port, public URL, lease TTL countdown. Actions: request-expose, pin/unpin-core, extend/revoke lease, probe-now. | `routes`, `exposure`, `probes` |
| **Recovery & Events** | Recovery-attempt timeline, circuit-breaker state (closed/open/half-open), manual-recover action. | `recovery` |
| **Metrics** | cloudflared time-series (HA connections, request errors, RTT, active streams) + per-route probe history. | `tunnel`, `probes` |
| **Audit** | Port-compliance findings: scenarios whose `service.json` UI port is missing, ranged, or mismatched vs the manifest. | `audit` |

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

Typed Connect-RPC clients live in `ui/src/api/<domain>.ts`. Real-time
updates use redis pub/sub when available and fall back to HTTP polling
otherwise (see [`INTEGRATIONS.md`](INTEGRATIONS.md)).

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
