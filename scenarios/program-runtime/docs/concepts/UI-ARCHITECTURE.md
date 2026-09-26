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
react-component-library adoptions resolve-path <component-id> program-runtime
react-component-library adoptions apply <component-id> program-runtime <adopted-path>
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

Run `experience-manager spec validate program-runtime --json` after route or
 selector changes. Page specifications must describe real routes and are
removed with their route implementation.

## Console Shape — An IDE For Programs An Agent Wrote

This console's primary user never opens it. An agent submits programs over the
CLI; a human arrives afterward to understand what happened. That makes the
interface forensic rather than task-oriented, and the layout follows an IDE
rather than a CRUD admin panel:

| IDE concept | Surface here | Feature folder |
|---|---|---|
| Symbol explorer | Binding registry — bound tree plus unbound reasons | `features/bindings/` |
| File history | Program corpus | `features/programs/` |
| Editor | Program detail — replay, fork, compose | `features/programs/` |
| Debug session + watch window | Sessions and kernel variables | `features/sessions/` |
| Output pane | Live execution stream | `features/sessions/` |
| — | Measures (statistics) | `features/measures/` |

### Editor

Program source renders in Monaco, not a formatted code block. Monaco is already
governed in the fleet (`react-component-library` ships `@monaco-editor/react`
and `monaco-editor`; see its `ComponentEditor*.tsx` for a working reference).
Install it through Scenario Dependency Analyzer like any other package.

Monaco is **lazy-loaded**, mounted only when a program is opened. List views —
corpus rows, empty-state teaching samples, measure result snippets — render with
`shiki` instead, so the dashboard never pays for the editor bundle.

Three modes share one component:

| Mode | Source | Markers | Decorations |
|---|---|---|---|
| `replay` | a historical program, read-only | historical facts | real per-line fetched/materialized counts |
| `fork` | an edited copy, resubmittable | predictive, from live pre-flight validation | none until the fork runs |
| `compose` | a new program | predictive | none until it runs |

Editor behaviors that carry scenario meaning, and are therefore not optional
polish:

- **Hover** resolves the binding's typed signature, declared `effect`, and
  required permissions from the proto descriptor, and states whether the current
  session's grants satisfy it.
- **Markers** come from the same pre-flight validator the API uses to refuse a
  call. Never a UI-side reimplementation — see `../internal/DECISIONS.md`.
- **Inline decorations** report per line what the call fetched versus what it
  materialized. This is the product thesis at line granularity.
- **Gutter** distinguishes the three failure classes — refused, invalid,
  errored — by icon and text, never by color alone.

### Accessibility Consequence

A code editor is the hardest surface to keep accessible, so the editor's meaning
must survive without it. Every gutter marker also exists as an entry in an
accessible list adjacent to the editor, and no failure is conveyed only by a
squiggle. Tracked as claims on the `program` page spec.

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
