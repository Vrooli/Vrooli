# UI Architecture

## The governing idea

The workbench is a **specimen sheet**, not a gallery.

Imagery is the content under judgement, so the interface around it stays
achromatic. The only saturated colour on screen should be inside the artwork
(`UIX-006`). A chrome that competes with the specimens makes them impossible to
evaluate — and a tool for judging visual work that cannot itself be judged
against is self-defeating.

Two consequences that are easy to get wrong:

- **A specimen frame paints its own declared ground.** It must not inherit the
  page background, or the same candidate reads differently in light and dark
  themes and neither reading is trustworthy.
- **Parameters sit beside every specimen** in a monospace face, so a result
  always reads as a recipe rather than an accident (`UIX-004`).

## Surfaces

| Route | Purpose | Primary requirement |
|---|---|---|
| `/` — Catalog | Browse and filter styles by axis; the entry point | `CAT-003`, `UIX-001` |
| `/styles/:id` — Style | Inspect and edit one style; fork it | `CAT-005`, `CAT-006` |
| `/compose` — Composer | Bind a style to a brief; review the resolved plan before spending | `CMP-001`, `UIX-002` |
| `/renders/:id` — Candidates | Judge a candidate set against the gate; select one | `RND-003`, `UIX-003` |
| `/renders/:id/placements` — Placement preview | See the selected candidate in every declared placement, desktop and mobile | `UIX-005` |
| `/sweep` — Contact sheet | Render a grid across two axes | `RND-006` |
| `/backdrops` — Released | The consumer-facing library | `REL-004` |
| `/settings` | Brand selection, locale, execution preferences | template floor |

## The composer is a two-stage surface

This is the interaction that matters most, because it is where money is spent.

1. **Resolve.** The operator picks a style and writes a brief. The UI shows the
   fully resolved plan — strategy, ordered operations, merged parameters,
   conditioning inputs, expected execution path — with nothing executed yet.
2. **Render.** Only then does work begin.

Separating the two makes the cost of a decision visible before it is incurred,
and turns an unexpected result into something readable rather than mysterious.
It mirrors `image-tools`' own explain-before-execute pattern deliberately.

## Judging candidates

Each candidate tile carries, without interaction:

- the image, framed against its own declared ground
- the measured worst-pixel contrast as a **number and a text label** — never
  colour alone (`UIX-003`)
- the strategy, seed, and execution path
- the full resolved plan, expandable and copyable

The gate verdict must survive being read by someone who cannot distinguish the
pass and fail hues. A contrast tool that communicates its verdicts by colour
would fail exactly the users it exists to protect.

## Refusals are a first-class UI state

Whenever release is unavailable, the surface names the specific unmet condition
and, where one exists, the amendment that would pass (`UIX-002`):

| Condition | What the UI shows |
|---|---|
| Contrast below threshold | Measured ratio, threshold, region — plus the minimum scrim that would pass |
| Unresolved palette slot | The slot name and the active brand |
| Restrictive adapter licence | The adapter and its restriction |
| Missing alt text | The field, with "mark decorative" as an explicit alternative |
| No selection | Which candidates are eligible |

A disabled control with no stated reason is a defect, not a style choice.

## Shared primitives

Adopt from `react-component-library` via `react-component-library adoptions
apply` rather than hand-rolling: buttons, cards, data tables, empty states,
inputs, selects, status badges, sidebar shell, bottom navigation.

Genuinely new to this scenario, and therefore built here:

- **Specimen frame** — a canvas with a declared ground, an aspect lock, and a
  parameter caption
- **Contrast verdict** — numeric ratio, text label, threshold, and pass state
- **Copy-safe overlay** — the region drawn over a specimen, toggleable
- **Placement preview** — a candidate composited into a layout wireframe with
  representative copy at a chosen viewport
- **Axis filter** — multi-axis facet selection over the catalog

If any of these prove generally useful, promote them to
`react-component-library` rather than letting a second scenario copy them.

## Responsive and accessibility floors

- The template's responsive shell floors are durable seams: full viewport height,
  overflow-contained main content, desktop sidebar, fixed safe-area mobile bottom
  navigation, Settings owning locale.
- WCAG 2.2 AA throughout. Visible keyboard focus on every control.
- Specimen canvases carry meaningful `aria-label`s describing the treatment and
  subject, not "image".
- `prefers-reduced-motion` respected; the workbench has no ambient animation by
  default, because motion beside a still specimen corrupts the comparison.

## Related

- `../../DESIGN.md` — tokens, motion, and status semantics
- `../../experience/` — L0 route specs
- `ARCHITECTURE.md` — why the composer resolves before it executes
