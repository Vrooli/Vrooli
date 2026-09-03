# UI Architecture

## Purpose Of This Document

Describe the canonical layout of the `ui/` source tree for scenarios generated
from the `react-vite` template, and the **slot taxonomy** that lets external
tools (notably `react-component-library`'s adoption resolver) place components
without asking the user for a path.

## Compute Manager UI

> **Status: designed, not built.** The `ui/` tree is still the generated
> template. `ui/src/pages/` holds `DashboardPage.tsx` and `SettingsPage.tsx`,
> and `ui/src/features/` holds one folder, `health/`. None of the five surfaces
> described below exists as code, no feature folder named here has been
> created, and no library component named here has been linked
> (`docs/reference/component-library-gaps.md` records no ejections and no
> gaps because nothing has been attempted yet). Everything in this section is
> the intended shape, matching the experience contract in `experience/`, which
> is itself status `draft` with every claim tier `aspirational`.

### Surfaces And Feature Folders

The five surfaces are the ones named in
[`EXPERIENCE.md`](EXPERIENCE.md#information-architecture). Each maps to one
feature folder under `ui/src/features/<domain>/`, named after the backing
domain in [`DOMAINS.md`](DOMAINS.md) rather than after the route, so the UI
folder and the API package share a name.

| Surface | Route | Feature folder | Backing domains |
|---|---|---|---|
| Inventory | `/` | `ui/src/features/inventory/` | `presentation`, `reconcile`, `expiry` |
| Instance | `/instances/:id` | `ui/src/features/instance/` | `instance`, `meter`, `expiry`, `enroll` |
| Request capacity | `/request` | `ui/src/features/request/` | `intent`, `instance`, `meter`, `provider` |
| Findings | `/findings` | `ui/src/features/findings/` | `reconcile` |
| Settings | `/settings` | `ui/src/features/provider/` mounted by `ui/src/pages/SettingsPage.tsx` | `provider`, `reconcile`, `expiry` |

`ui/src/features/health/` stays as generated. It is the worked example of a
scenario-owned feature built from library parts, and the `health` domain is a
real domain rather than template residue.

Typed Connect-RPC clients belong in `ui/src/api/<domain>.ts`, one per domain,
matching the `api-client` slot.

### Library Components The Two Hardest Surfaces Need

The inventory table and the findings list carry the claims in
`experience/pages/dashboard.json` and `experience/pages/findings.json`, so they
are the two surfaces where the choice between linking and building matters.
Ask the library first, and record anything that does not fit in
[`../reference/component-library-gaps.md`](../reference/component-library-gaps.md)
rather than forking quietly.

| Surface part | Candidate library asset | What the claim demands of it |
|---|---|---|
| Inventory table | `DataTable/1` (or `Table/1` if the toolbar is unwanted) | `dashboard-tabular-numerals`: cost and lifetime columns must render with tabular numerals so figures compare down the column. `dashboard-state-not-colour-only`: an expiring-soon row must be distinguishable by shape or label, not hue alone |
| Liability summary | `StatCard/1` with `RollingNumber/0` for the accruing figure | `dashboard-cost-visible-without-scroll`: total liability sits in the first viewport at every width |
| Unaccounted badge | `NotificationBadge/1` or `StatusBadge/1`, linked to `/findings` | Must render "unknown" distinctly from "zero". This is the `partial` state, and unknown-rendered-as-zero is the failure the surface exists to prevent |
| Expiry countdown | `RelativeTime/1`, with `FreshnessArc/0` for observation age | The `stale` state labels figures with their observation time instead of showing them as current |
| Findings list | `FindingList/1` | `findings-direction-is-explicit`: each row states which side is missing, not merely that something differs |
| Quarantine control | `QuarantineBadge/1` plus `Button/2` | `findings-no-automatic-resolution`: quarantine is the only action on the page, and no control there destroys anything |
| Last-sweep status | `StatusIndicator/1` with `RelativeTime/1` | `findings-empty-is-trustworthy`: a clean sweep and a sweep that has not run must not render the same way |
| Every async region | `AsyncBoundary/1`, `LoadingState/1`, `EmptyState/1`, `ErrorState/1`, `ExperienceSurface/1` | The `state-covered` claims on both pages: each declared state needs a distinct, readable surface |
| Destroy confirmation | `UndoableDestructiveAction/1` or a scenario-local form | `instance-destroy-is-deliberate` requires typing the instance name. A generic confirm dialog does not satisfy it, and if no library asset takes a typed-confirmation token, that is a gap to record |

None of these is linked yet. Run the suggestion pass before writing any of
them by hand:

```bash
react-component-library adoptions suggest compute-manager --json
```

## Source Layout

```
ui/src/
├── api/            # api-client slot — Connect-RPC wrappers
├── app/            # app-bootstrap — Providers composition and route table
├── components/     # shared-component slot — cross-cutting components
│   └── ui/         # ui-primitive slot — empty by design; primitives come from the library
├── consts/         # consts slot — strings + selectors registries
├── features/       # feature slot — per-feature folders (one subfolder per feature)
│   └── <feature>/  # feature-component slot — components inside a feature
├── i18n/           # i18n bootstrap
│   └── locales/    # i18n-strings slot — one JSON per locale
├── layout/         # layout-shell + layout-nav slots — AppShell config, navItems, BrandMark
├── lib/            # lib-util slot — framework-agnostic utilities
├── pages/          # page slot — routed pages mounted under <Outlet />
├── test-utils/     # test-util slot — render helpers, factories, a11y
└── theme/          # theme-token slot — ThemeProvider; tokens live in the generated design-tokens.css
```

Some slots declare a directory that does not exist yet. `ui/manifest.json`
declares `hook` at `ui/src/hooks`, plus `service`, `provider`, `adapter`,
`pattern`, `page-template`, `fixture`, `visual-recipe` and `motion`; none of
those directories is in the tree, because the resolver creates a slot's
directory when the first file lands in it. A slot is a declared destination,
not a promise that the folder is there.

## Slots Are A Contract

Every directory above maps to a named slot in `ui/manifest.json`. The manifest
declares the directory **and** a default path pattern (e.g.
`{dir}/{ComponentName}.tsx`), so external tools can compute the canonical
filesystem path for a new file given just the component's name and slot.

A component library that publishes `"slot": "layout-nav"` and ships
`SidebarShell` knows — without any per-scenario configuration — that the file
should land at `ui/src/layout/SidebarShell.tsx`. Override the slot's `dir` in
a scenario-level overlay if you've reorganized; the resolver merges that
overlay before computing the new path.

## The Shell Is A Library Import

`ui/src/layout/AppShell.tsx` mounts `AppShell` from
`@vrooli/react-component-library/AppShell/2` and passes it three things: the
navigation items from `navItems.tsx`, a router adapter (`renderLink` and
`onNavigate` wired to react-router), and three settings — `density`
(`sidebar` or `rail`), `mobileNav` (`tabs` or `drawer`) and `mainMode`
(`scroll` or `fill`). The library shell composes `SidebarShell` for the
desktop column and `BottomNav` for the phone, owns the skip link, the
landmarks and the safe areas, and measures the viewport itself. Every
generated scenario therefore improves when the shell does, through
`react-component-library adoptions reconverge`.

The template draws no chrome of its own. Two things in `layout/` are
scenario-owned on purpose: the navigation data and the brand mark.

## Adopting From The Library

Pages are composed from linked library components: `PageHeader/2` at the top
of every page, `SettingsList/1` and `Select/1` on Settings, `EmptyState/1`
for the home placeholder, `Card/1`, `Button/2`, `StatusBadge/1` and
`ExperienceSurface/1` inside feature components. A linked adoption is a
package import and leaves no file behind.

Before writing any shared UI, ask the library what it has for your routes and
link it:

```bash
react-component-library adoptions suggest compute-manager --json
react-component-library adoptions link <component-id> compute-manager
react-component-library adoptions obligations compute-manager --json
```

Use scenario-local components for genuinely scenario-specific surfaces (the
feature folders under `ui/src/features/`), not for generic tables, buttons,
navigation, form controls, or status badges the library already provides.
`features/health/HealthCard.tsx` is the worked example of a scenario-owned
feature built from library parts. When a local component turns out to be
generic, hand it back with `react-component-library components ingest`.

## Files Are Declared Too

Beside `slots`, the manifest's `files` section names the scenario files tooling
reads or writes: `designTokens` (with the `rcl:tokens` managed region markers),
`tailwindTheme`, `tokenMap`, `localeCatalogue` (a `{locale}` pattern with its
default), `selectorRegistry`, `librarySelectors`, `appEntry` and
`stringsRegistry`. `react-component-library adoptions link`, `tokens-sync` and
`obligations` resolve those paths from here, so moving a file is a manifest
edit rather than a library change. An overlay may change a declared path but
may not add keys.

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

## Experience Contract

The generated `experience/` folder is the UX-intent contract for the route
table in `ui/src/app/routes.tsx`. Keep those two surfaces aligned:

- add an L0 page spec when a new user-facing route is added;
- remove or deprecate a page spec when a route is removed;
- promote a page from L0 by adding priorities, elements, claims, bindings, and
  states before calling the route production-ready;
- keep `data-testid` selectors in code aligned with
  `experience/pages/*.json::bindings`. All five page specs already declare
  bindings, so the selectors are the ones the UI is built to match.

Run `experience-manager spec validate compute-manager --json` after route or
selector changes. The generated notes page spec was example-domain content and
was removed by `template-manager detemplate compute-manager`; the five page
specs left are this scenario's own.

## Extending The Manifest

- **Add a slot.** Add an entry to `ui/manifest.json`. Keep its `dir` inside
  `ui/src/` and pick a pattern that matches your file-naming convention. The
  schema (`scenario-ui-manifest/v2`) does not enum-restrict slot names — open
  set on purpose.
- **Override a slot in a single scenario.** Drop a partial manifest at
  `.vrooli/ui-manifest.json` in the scenario root; the resolver merges it over
  the template manifest. The overlay may override existing slots, but it may
  not invent new slot names.
- **Add a `postApply` action** (auto barrel-export, route-register,
  i18n-merge). Reserve this for a future schema revision.
  Document the intent in the consuming scenario's PRD until then.

## Cross-References

- Schema: `.vrooli/schemas/scenario-ui-manifest.schema.json` (`$id:
  scenario-ui-manifest/v2`)
- Manifest: `ui/manifest.json` (`contract.schema` is `scenario-ui-manifest/v2`,
  `schemaVersion` is `2.0.0`)
- Slot reference: [`ui-manifest.md`](../reference/ui-manifest.md)
- Adoption resolver: `scenarios/react-component-library/api/internal/adoptions/pathresolver.go`
