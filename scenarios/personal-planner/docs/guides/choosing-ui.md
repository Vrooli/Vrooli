# Choosing the UI

## Purpose Of This Document

Walk a scenario author through the one path from a generated scaffold to a
product-shaped UI: decide, configure, adopt, build. Gate 5 in
`path:scenarios/personal-planner/docs/START-HERE.md` is the checklist; this guide is the reasoning and the
reference for what the shell and the library can do.

## Why the order matters

Generated scenarios that skipped the first step all ended up looking the same:
a labelled sidebar, a top bar holding a theme select, and a home page of stat
tiles. Not because anyone chose that, but because the scaffold rendered it and
nothing asked whether it fit. The shell is now a library import with three
settings, and the home page is a placeholder the orientation gate refuses.
Both exist to make the decision unavoidable, not to make it for you.

## 1. Decide

For Personal Planner this decision is already made and recorded in
[`../concepts/EXPERIENCE.md`](../concepts/EXPERIENCE.md) and the binding
[`../../DESIGN.md`](../../DESIGN.md) (the operator-approved **Observatory**
direction, decision D08). The three questions and their answers:

1. **What will people compare this to?** The calm, editorial tier of
   personal planners — **Sunsama, Amie, Structured, Fantastical** — not the
   operational-console tier (Linear/Jira) and not a raw calendar grid. That
   sets a high bar for beauty and daily-open appeal; the wedge is matching
   that beauty *and* beating it on honest accounting. Explicitly **not** a
   generic white-card/blue-button dashboard, a marketing site, or a neon
   "AI product."
2. **What is the one surface that matters most?** **Today** — the page
   people open first and stay on. It answers "What can I usefully do now?"
   and "What needs my attention?" via a prominent next-action surface, a
   quiet capacity block, a commitment notice, and a real-time-scale
   timeline, with the Observatory scene decorative below the work. See
   `EXPERIENCE.md` for its desktop and phone layouts and its full state
   list (loading, empty, partial, over-capacity, nothing-fits, stale
   forecast, source-unavailable, saving-failed, running session — in both
   Day and Night).
3. **What is the design accountable for?** In order: **truthful time
   geometry** (block width ∝ duration), **honest state legibility**
   (available vs reserve, promise vs forecast, planned vs actual, source
   status vs planner state, empty view vs failed query), and **the scene
   never competing** with legible, accessible work. These become the claims
   in `experience/`.

Because this decision is settled, do not re-derive it — build to it. The
primary surface is **Today**; the shell is the **navigated console**
archetype (a left rail of destinations + a scrolling main pane) styled by
the **Observatory** theme, with **five destinations**: Today, Plan, Goals,
Focus, Review (plus Settings and profile near the rail bottom). The gate
checks that the question was asked; here it is answered.

## 2. Configure

The shell is `AppShell` from the component library. `ui/src/layout/AppShell.tsx`
sets three constants and plugs in the router; it draws nothing.

| Setting | Options | Choose |
|---|---|---|
| `density` | `sidebar` · `rail` | `sidebar` for a tool with several peer surfaces (icon and label, resizable). `rail` when one surface needs the width (icon over a short label, fixed narrow column). |
| `mobileNav` | `tabs` · `drawer` | `tabs` for three to five destinations. `drawer` when there are more, or when the phone needs the same nav labels as the desktop. |
| `mainMode` | `scroll` · `fill` | `scroll` pads and scrolls every page for you. `fill` hands the page the whole pane so it can pin its own header and composer and scroll only its transcript. |

Destinations live in `ui/src/layout/navItems.tsx`: key, path, label key,
icon. The shell renders the same list as the desktop column and the phone tab
bar, so the two cannot drift. The mark in `ui/src/layout/BrandMark.tsx` is
yours to replace.

The shell also gives you, without asking: a skip link, one primary navigation
landmark and one mobile landmark, `main` with an id, safe-area handling on the
phone, a resizable column that remembers its width, forced-colours and
reduced-motion behaviour, and right-to-left layout.

What it deliberately does not have is a top bar. Preferences live in Settings.
If a product needs a global strip, pass `header`; it appears at every width.

If the shell cannot do what your primary surface needs, do not fork it. Record
the gap in `docs/reference/component-library-gaps.md` and eject with a reason:

```bash
react-component-library adoptions eject navigation.app-shell personal-planner --reason "..."
```

An ejection is a declared debt the library can pay back; a silent fork is not.

## 3. Adopt

Ask the library what it has for your route list, then link it. A linked
adoption is a package import (`@vrooli/react-component-library/<Asset>/<major>`)
and leaves no file behind; the version follows the library through
`adoptions reconverge`.

```bash
react-component-library adoptions suggest personal-planner --json
react-component-library adoptions link <component-id> personal-planner
react-component-library adoptions preflight <component-id> personal-planner
react-component-library adoptions obligations personal-planner --json
```

`link` writes to the files `ui/manifest.json` declares under `files`: the
asset's selectors into `selectorRegistry`/`librarySelectors`, any tokens the
ramp lacks into the `designTokens` managed region, and the asset's
`defineStrings` entries into the default `localeCatalogue`. After a link, run
`pnpm selector:manifest`, and `pnpm strings:gen` if the catalogue changed, so
the generated registries match before `pnpm type-check`.

What the template already links, and what to reach for next:

| Need | Library asset | Notes |
|---|---|---|
| Page title, description, actions | `PageHeader/2` | Every routed page opens with one. `level={2}` for a section; `density="compact"` inside a pane. |
| Settings rows | `SettingsList/1` + `Select/1` | Group by what a setting changes. Rows label their control for you; `RadioGroup/1` `variant="card"` when a choice needs a sentence of consequence. |
| Nothing here yet | `EmptyState/1` | Say what would fill it. Never an empty box. |
| Loading, empty, partial, error | `AsyncPanel/1` or `ExperienceSurface/1` | Every declared region renders through one so the state is machine-checkable. |
| A record or a tool | `Card/1` | Repeated records and framed tools only. Do not wrap page sections in cards. `CardTitle as="h2"` when the card sits directly under the page title. |
| Status | `StatusBadge/1` | Categorical states. A rank needs its own encoding. |
| Actions | `Button/2`, `IconButton/3` | Variants and sizes are props; a Tailwind class in `className` will not override them. |
| List beside detail | `MasterDetail/1` | Stacks on a phone with a back affordance. |

Before importing an asset, check its maturity: the library's
`docs/guides/component-quality-rubric.md` says how to tell a real component
from a scaffold, and `components list --json` reports the declared maturity.
Do not adopt anything below `implemented` into a page users will see.

## 4. Build

Build only what is left, under `ui/src/features/<name>/`, token-bound. The
tokens are Tailwind utilities generated from the kit: colours as `app-*`,
radii as `rounded-control` / `rounded-panel` / `rounded-pill`, spacing as
`gap-space-sm` / `p-space-md` and so on, type as `text-body` / `text-caption`.
A pixel value typed by hand is a sign the scale is missing a step; add the
token, do not add the pixel.

`features/health/HealthCard.tsx` is the worked example: a scenario-owned
feature composed from `Card`, `StatusBadge` and `Button`, with its own
selectors, strings and tests. When a local component turns out to be generic,
hand it back:

```bash
react-component-library components ingest personal-planner ui/src/features/<name>/<Component>.tsx <Component>
```

## Cross-references

- `path:scenarios/personal-planner/docs/START-HERE.md` — Gate 5, the checklist this guide explains
- `path:scenarios/personal-planner/docs/concepts/UI-ARCHITECTURE.md` — where files go and why
- `path:scenarios/personal-planner/docs/concepts/EXPERIENCE.md` — where the decision is written
- `path:scenarios/personal-planner/DESIGN.md` — the token contract the kit provides
- `path:scenarios/personal-planner/experience/README.md` — turning the decision into typed claims
