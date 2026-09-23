# Experience Design

## Purpose Of This Document

Record the experience decision for Nooch (`nutrition-planner`) after the v2.0
redesign: what people will compare it to, which surface matters most, how each
surface is composed on desktop and on a phone, and what the design is accountable
for. It is the prose companion to the machine-readable contract in
[`../../experience/`](../../experience/index.json); where the two disagree,
`experience/` wins because a validator reads it.

This document does not restate tokens, type scales, or spacing — those are
binding in the root [`DESIGN.md`](../../DESIGN.md). It does not restate
behaviour — that lives in the specification's R-sections
([`../reference/product-specification.md`](../reference/product-specification.md)).
It summarizes composition and links to the surface-by-surface build notes in
[`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md) §6 and the visual
reading guide in [`../reference/mockups/README.md`](../reference/mockups/README.md).
The `ui/` source layout is [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md).

## The Comparison

Nooch should feel like **a beautifully designed cookbook in a warm kitchen**
(R05.1). People will judge it against well-made cooking and recipe apps, calm
meal planners, and the thing it replaces — opening the fridge and deciding, or
rereading the same notes. The approved concept mockups set the bar:

- **Decision first.** Today answers "what am I eating next, and what do I do?"
  with one meal and one dominant action, not a wall of nutrient gauges.
- **Food carries the colour.** Photorealistic meals and kitchens create the
  atmosphere on Today, Week, Meals, Explore, and the equipment scene; Groceries
  and Kitchen inventory emphasize readable information.
- **Editorial and quiet.** An editorial serif for titles, meal names, and the
  cooking timer; a readable sans for controls and metadata; hairlines before
  boxes; one restrained icon family; no glass over busy photos, pervasive
  gradients, oversized metric dashboards, or slogans.
- **Honest, not clinical.** Unknown reads as unknown, never zero; totals name
  their scope; "Fits your current rules", never "Safe for your allergy".
- **Low ceremony.** A name is enough to save a meal; a swap needs no reason;
  inventory is optional.

## The Primary Surface

**Today** is the surface people open first. Its hero is the next relevant
planned occurrence — not always dinner (R08.1): a local date and slot eyebrow,
the meal title in large serif, a one-line description, active and total time,
**Start cooking** as the dominant action, **Swap meal**, and a quiet occurrence
menu. Below it sit a seven-day strip, **Ready for tonight** (Check ingredients,
Open grocery list), and a collapsible **Today overview** of the rest of the day's
meals, recurring items, and supplements with explicit Planned, Recorded, or
Expected scope (R08.3).

The hero is **one component with three media treatments** (R17.1), chosen
deterministically and never by generating an image:

| Treatment | When | Mockup |
|---|---|---|
| Immersive scene | An approved scene composition matches the recipe revision, appearance, and viewport | `today-sunroom-light.png`, `today-evening-kitchen-dark.png` |
| Editorial photo | Only an ordinary photo exists (or the person chose Editorial) | `today-editorial-light.png` |
| Minimal | No reliable image; typography and an ingredient summary | none — built from the editorial layout |

All three keep the same regions, order, and actions; a failed image falls back
without moving the title or actions. An empty account sees Choose a meal and
Plan my week, never the demo meal, and a meal on another date is labelled with
its day and slot, never "Tonight".

## Information Architecture

Exactly **five primary destinations**, in this order on desktop and phone:
Today, Week, Meals, Groceries, Kitchen — plus Settings from the header (R01.2,
decision D-031).

| Surface | Route (R03.1) | Question it answers | Contract page |
|---|---|---|---|
| Today | `/today` (and `/`) | What am I eating next, and what do I do? | `today` |
| Week | `/week` | Does this week's plan fit my life? | `week` |
| Meals — Your meals | `/meals` | What have I saved, and what can I plan? | `meals` |
| Meals — Explore | `/meals/explore` | What new meals fit my rules, kitchen, and week? | `explore` |
| Recipe detail | `/meals/:id` | How do I make this, and what is in it? | `recipe` |
| Cooking | `/cook/:sessionId` | What do I do right now? | `cooking` |
| Groceries | `/groceries` | What do I need, and what have I picked up? | `groceries` |
| Kitchen | `/kitchen` | What do I have, and how do I cook? | `kitchen` |
| Onboarding | `/setup` | How do I set up my food, kitchen, and rhythm? | `onboarding` |
| Settings | `/settings` | What applies everywhere — appearance, artwork, generation, integrations, units, data? | `settings` |

`/nutrition` and `/transfer` retired as primary routes (D-031): nutrition
analysis now lives in the Week **Nutrition** view, the Today overview, and the
recipe **Nutrition** tab; targets and supplements live in Kitchen ›
Preferences; export, print, import, and restore live in Settings › **Data &
exports**. Their contract pages remain as deprecated tombstones.

URLs carry harmless UI context — week, view, tab, mode, planning slot — and
never allergy lists, credentials, signed media URLs, or recipe payloads. Back
works through details, editors, and overlays, and returning restores filters,
scroll, selected date, and slot (R03.1). Context is explicit (R03.2): Explore
opened from an empty Wednesday dinner shows **Planning Wednesday dinner**, and
viewing, scaling, or shopping never mutates a plan by itself.

## Composition By Medium

Desktop composes in space; the phone composes in sequence and thumb reach. The
content model of each surface is defined once; each medium composes it with its
own components where the interaction changes (a breakpoint hook, decision D-036)
and with CSS reflow where only arrangement changes. Container width decides
(R06.1): below 640 px compact with bottom tabs; 640–1023 px intermediate;
1024 px and up the desktop shell; the Week board needs about 136 px per day.

| Surface | Archetype | Desktop | Phone |
|---|---|---|---|
| Today | Dashboard | Hero with text in the scene's safe area; week strip; Ready for tonight; overview | Short scene or photo (about a quarter of the viewport), then title, times, stacked full-width actions within the first screen; compact strip |
| Week | Data table | Seven-day board of configurable slot rows; occurrence panel; Prep for the week | Day selector, Day / All week, stacked slot cards; actions in sheets |
| Meals | List / feed | Three-column photo grid (two at intermediate widths) | One readable card per row; chips scroll in their row |
| Explore | List / feed | Context banner, three highlighted suggestions, short secondary sections | One-column cards with the reason visible; secondary rows with See all |
| Recipe | Reading surface | Ingredient column beside a wider method; moderate photo; map in its own scroll region | Photo about 160–220 px, summary and actions, ingredients and method stacked |
| Cooking | Immersive single task | Step rail, active step, timer card, footer actions | One step, step-list sheet, timer card or tray, footer in the thumb zone |
| Groceries | List + sidebar | List with the From your plan / Before you shop sidebar | Full-width list; supplementary detail in sheets |
| Kitchen — On hand | Data table | Inventory table plus Use soon, equipment, and preference panels | Card rows and a use-soon strip |
| Kitchen — Equipment | Immersive picker | Scene beside the tile grid | Short scene above the category selector and two-column tiles |
| Kitchen — Preferences, Settings, Onboarding | Form / settings | Grouped sections | Summary rows that drill into sub-screens; full-screen steps |

The per-surface content models, states, corrections to the mockups, and the
acceptance cases that prove each surface are in
[`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md) §6. Do not
duplicate them here.

## Shell Configuration

The v1 shell (sidebar density; four destinations in the v1 specification and six nav
items in the prototype code, including a Home route) is superseded.

- **Desktop header:** the serif wordmark (the configured display name, `Nooch`) at left; the five destinations
  as text links with an underline marking the current one; the appearance
  control and the Settings gear at right; a hairline below; no sidebar. Content
  max width about 1,440 px (Week up to about 1,600 px).
- **Phone:** a compact header (wordmark, appearance, Settings) and **five
  labelled bottom tabs** with safe-area padding whose height is reserved in the
  document flow; toasts sit above the tab bar and never cover primary actions.
- **Focused cooking:** no destinations; Exit cooking and View full recipe stay
  visible, and exiting keeps the session and timers.
- **Global actions** (R03.3): one labelled primary action per destination —
  Start cooking, Plan my week, Add meal, Add item, Add ingredient — and no
  unlabelled floating plus that means different things on different pages.

The shell comes from the Vrooli component library where it fits; the
nutrition-specific pieces are listed in R05.3 and
[`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md).

## Appearance Model

Three independent settings (R05.2, R17.1): **Appearance** — Light, Evening, or
Follow device (the default); **Meal artwork** — Immersive scenes, Editorial
photos, or Minimal; **Image generation** — Off (the default), Ask each time, or
Automatic within budget. They never collapse into one premium switch. Changing
appearance changes palette and artwork only — never the recipe, servings, dates,
occurrences, layout, or actions — and never starts generation. The explicit
choice persists locally for the first paint and in the account, with no bright
flash on an evening reload. A dark breakfast is still breakfast. Tokens and
theme behaviour are binding in [`DESIGN.md`](../../DESIGN.md).

## Cross-Mockup Decisions

The fifteen concepts disagree in places; the resolutions are recorded in
[`../reference/mockups/README.md`](../reference/mockups/README.md) §"Cross-mockup
decisions" and decisions D-027 and D-028 in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md). In short: underline for
the current destination and for in-page tabs; segmented controls only for mode
switches; one selection token per appearance for chips and segments; the header
always carries appearance and Settings; one icon per destination from one family,
everywhere; icon plus small-caps labels for storage and aisle groups in both
appearances; five labelled phone tabs; heart for favourite and bookmark for save;
distinct cooktop and oven; the four-step Sesame tofu bowl fixture across recipe,
map, and cooking; an explicit Mark step done; functional copy instead of slogans.

## What The Design Is Accountable For

In order:

1. **The next action is one screen away.** On a phone, Today's title and Start
   cooking, a recipe's Start cooking, and the cooking step's Mark step done and
   Next step are reached without scrolling past decoration.
2. **It looks like the approved mockups** in Light and Evening, on desktop and
   phone, including the editorial, minimal, failed-image, empty, and error
   states — not only the ideal photograph (OT-P0-021).
3. **Every derived number is honest about its scope.** Unknown, zero, partial,
   and stale are different facts; a dinner never reads as the whole day; no
   grocery total appears without price coverage.
4. **Required rules are visible and not negotiable.** Diet, exclusions, allergen
   evidence, and kitchen feasibility filter before ranking in Today swaps,
   Explore, and planning alike, and a preference never makes an ineligible meal
   look compliant.
5. **Nothing pretends.** Viewing a step does not complete it, checking a grocery
   row does not purchase it, a timer's expiry does not finish a meal, a scene
   marker does not toggle equipment, and an unavailable capability shows an
   explanation instead of a dead button.

Under those, the design meets a WCAG 2.2 AA target (R07): keyboard paths and
visible focus in both appearances, accessible names on icon actions, correct tab
and dialog semantics, semantic tables or labelled lists for ingredients and
inventory, Move/Copy alternatives to every drag, equipment tiles as the complete
alternative to scene markers, 44 px targets, reduced motion, 200 % zoom without
page-level horizontal scrolling, and timer announcements only on state changes.

## States

Every surface declares the lifecycle states in the `DESIGN.md` UX-state
contract plus its own (for example `media-editorial`, `media-minimal`,
`swap-impact`, `plan-preview`, `shop-mode`, `plan-changed`, `timer-elapsed`,
`equipment-none`, `rule-change-preview`). A load failure is never an empty
account, a skeleton appears only while real data loads, and pending edits
survive errors. The full state list for each page is in its contract file under
[`../../experience/pages/`](../../experience/index.json).

## Contract Status

The contract is authored at full depth but describes surfaces that are not built
yet, so every redesigned page and journey is `draft` with `aspirational` claims
(decision D-035). As each surface is rebuilt to its mockup, its page becomes
`active`, gains a BAS case, and has its checkable claims promoted to `machine`
tier. See [`../../experience/README.md`](../../experience/README.md).

## Cross-References

- [`../../experience/index.json`](../../experience/index.json) — the typed page and journey contract
- [`../reference/mockups/README.md`](../reference/mockups/README.md) — the approved mockups and how to read them
- [`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md) — surface specifications, build order, and verification
- [`../reference/product-specification.md`](../reference/product-specification.md) — R03, R05–R16, R17, R26, R27
- [`../../DESIGN.md`](../../DESIGN.md) — the binding design language and tokens
- [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md) — the `ui/` source tree and shared components
- [`FLOWS.md`](FLOWS.md) — the journeys and state machines these surfaces carry
- [`DOMAINS.md`](DOMAINS.md) — which domain backs each surface
- [`../reference/component-library-gaps.md`](../reference/component-library-gaps.md) — what the shared library does not yet provide
- [`../../PRD.md`](../../PRD.md) — the operational targets these surfaces claim against
