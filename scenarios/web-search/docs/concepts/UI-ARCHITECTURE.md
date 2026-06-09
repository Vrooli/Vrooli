# UI Architecture

## Purpose Of This Document

Describe the canonical layout of the `ui/` source tree for scenarios generated
from the `react-vite` template, and the **slot taxonomy** that lets external
tools (notably `react-component-library`'s adoption resolver) place components
without asking the user for a path.

## Intended Surfaces (web-search)

> **Scaffold status (2026-06-09):** None of the surfaces below are built
> yet. They are the *intended* UI from `PRD.md` (OT-P0-008, OT-P1-007).
> The template `notes` feature is still present and will be removed once
> the first real feature lands. The generic slot/manifest conventions in
> the rest of this document apply unchanged.

web-search's UI is a vrooli-default operational-console surface
(light + dark, WCAG AA) made of these planned feature folders under
`ui/src/features/`:

- **search** (`features/search/`) — the primary view: a search box and a
  blended results list. Results render as distinct provider groups —
  **live** (`web-search.live`) and **learnings** (`web-search.learnings`)
  — each hit **cited** (source + retrieval date). **Disputed** findings
  render *with* a visible "sources conflict" warning and both conflicting
  sources; a default query shows learnings without ever firing a live web
  call. Conveys status through both color and text (no color-only
  signals). [OT-P0-008]
- **query history** — replay/compare prior queries (within the search
  feature or a sibling). [OT-P0-008]
- **ops panel** (`features/health/` or a sibling ops feature) — SearXNG
  engine reachability, live-web cache hit-rate, and current budget
  remaining (token-bucket window). Reads live backend state. [OT-P0-008]
- **findings** (`features/findings/`) — findings-management surface:
  browse/search the learnings corpus, edit, supersede, flag, and manually
  add a finding; shows confidence, age (decayed score), status, and
  citations. [OT-P0-008]
- **research** (`features/research/`) — start/track L2/L3 research runs and
  view the resulting cited brief. *(P1)*
- **dispute review queue** (within `features/research/` or `features/findings/`)
  — a dedicated panel listing all flagged contradictions with status,
  conflicting sources, and resolution controls (resolve, re-research,
  dismiss). *(P1, OT-P1-007)*

All surfaces are honest about uncertainty: explicit abstain messages and
"sources disagree" labels rather than false confidence. These features
map onto the standard `feature` / `feature-component` slots below; no
new slot types are required.

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
