# UI Architecture

## Purpose Of This Document

Describe the canonical layout of the `ui/` source tree for scenarios generated
from the `react-vite` template, and the **slot taxonomy** that lets external
tools (notably `react-component-library`'s adoption resolver) place components
without asking the user for a path.

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
a scenario-level overlay if you've reorganized; the resolver merges that
overlay before computing the new path.

## Component Canon

Generated scenarios start with a small adopted-provenance canon under
`ui/src/components/ui/`: button, card, data table, empty state, input, select,
status badge, sidebar shell, and bottom navigation. Each file carries a
`@vrooliComponent*` JSDoc block so ui-health and react-component-library can
classify the surface as governed rather than unknown local code.

When adding a shared component, search and adopt from the registry first:

```bash
react-component-library components list --json
react-component-library adoptions resolve-path <component-id> hello-mobile
react-component-library adoptions apply <component-id> hello-mobile <adopted-path>
```

Use scenario-local custom components for genuinely scenario-specific surfaces,
not for generic tables, buttons, navigation shells, form controls, or status
badges that the canon already provides.

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
  `experience/pages/*.json::bindings` once bindings exist.

Run `experience-manager spec validate hello-mobile --json` after route or
selector changes. The generated notes page spec is example-domain content and
is removed by `template-manager detemplate hello-mobile`.

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
  scenario-ui-manifest/v1`)
- Manifest: `ui/manifest.json`
- Slot reference: [`ui-manifest.md`](../reference/ui-manifest.md)
- Adoption resolver: `scenarios/react-component-library/api/internal/adoptions/pathresolver.go`
