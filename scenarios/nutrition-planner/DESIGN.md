---
id: vrooli-default
version: 0.3.0
name: Nooch — Warm Kitchen
description: A calm cookbook in a warm kitchen — editorial serif titles, photorealistic food, warm ivory and forest days, olive-charcoal and amber evenings — over the Vrooli token plumbing.
components:
  button-primary:
    tokenSource: design-tokens.css
  button-primary-loading:
    tokenSource: design-tokens.css
  button-disabled:
    tokenSource: design-tokens.css
  input-error:
    tokenSource: design-tokens.css
  alert-error:
    tokenSource: design-tokens.css
  toast-success:
    tokenSource: design-tokens.css
  empty-state:
    tokenSource: design-tokens.css
  skeleton:
    tokenSource: design-tokens.css
  inline-progress:
    tokenSource: design-tokens.css
  retry-action:
    tokenSource: design-tokens.css
constraints:
  letterSpacing: "0"
  cardRadiusMaximum: "1.25rem"
  defaultMode: "system"
  supportedModes: ["light", "dark", "system"]
  responsiveBaseline: "mobile-first"
  dominantPalette: "warm-ivory-forest-terracotta-day-and-olive-charcoal-ivory-amber-evening"
---

# Nooch — Warm Kitchen Design

`DESIGN.md` is the source of truth for scenario UI decisions. Stack-specific adapters may translate these tokens into CSS, Tailwind, egui, native mobile themes, or future targets, but adapters must not redefine the design language.

**Warm Kitchen is a deliberate product theme, not the default operational
console** (decision D-030). It keeps the Vrooli token plumbing — the
`design-tokens.css` sources, the spacing scale, semantic status roles, the
component-library shell — and intentionally departs from the console's
"compact, operational, no atmospheric imagery" defaults because the product
owner approved this direction and beauty is a stated product requirement
(OT-P0-021). A meal planner is opened every day by choice; the calm, appetizing
look is what earns that habit, and the habit is what makes the honest planning
loop valuable. Do **not** substitute a stock dashboard and call it equivalent.

The visual target is the set of approved concept mockups in
[`docs/reference/mockups/`](docs/reference/mockups/README.md). The values below are
starting tokens from the specification (R05.1); tune them only after measured
contrast checks against real captures, preserving the direction. Never sample
colours from a generated image.

## How To Read This Document

This file mixes two kinds of guidance, and the distinction matters.

- **Binding contract** (must follow): the tokens, colour roles, typography scale, spacing, radius, motion rules, status semantics, responsive transformations, accessibility floors, the appearance model (Light, Evening, Follow device), the media-treatment rules, the component grammar in "Component Language", and the overall "calm cookbook in a warm kitchen" feel target.
- **Illustrative examples** (shape, not checklist): any concrete list of components, layouts, page surfaces, settings controls, or copy. They communicate *shape and feel*, not the feature set. The surface set is owned by [`docs/concepts/EXPERIENCE.md`](docs/concepts/EXPERIENCE.md) and `experience/`.

Concrete rule of thumb: this design tells you *how* a surface looks, behaves, and feels — not *which* surfaces the product must include.

## Intent

Nooch should feel like a beautifully designed cookbook in a warm kitchen. It
answers one question per screen — what am I eating next, does my week fit my
life, what can I make, what do I need, what do I have — with one clear primary
action and supporting detail on request.

- **Food carries the colour; chrome stays quiet.** Photorealistic food and
  kitchen imagery supply warmth on Today, Week, Meals, Explore, recipe detail,
  cooking, and the equipment scene. Groceries and Kitchen inventory emphasize
  readable lists.
- **Editorial, not clinical.** Serif titles, generous whitespace, hairlines
  before boxes, restrained line icons.
- **Honest.** Unknown values read as unknown, scope is always named, and
  nothing decorative ever implies a fact (a photo never proves ingredients,
  nutrition, or portions).

Avoid: glass panels over busy photography, pervasive gradients, neon, oversized
metric dashboards, decorative slogans, stock-dashboard card walls, and
generated text inside images.

## Layout

- **Desktop:** a single header row (serif wordmark (the configured display name, `Nooch`) at left, the five
  destinations as text links, appearance control and Settings at right, hairline
  below). No sidebar. Content max width ~1440 px; Week may use ~1600 px when its
  cells stay readable. Side padding 24–40 px.
- **Phone:** compact header (wordmark, appearance, Settings), page title in large
  serif, one column, and **five labelled bottom tabs** (Today, Week, Meals,
  Groceries, Kitchen) whose height is reserved in document flow and which respect
  the safe area. Side padding 16–20 px.
- **Focused cooking** replaces the destinations with **Exit cooking** and keeps
  appearance and Settings.
- Layout bands follow container width (R06.1): below 640 px compact; 640–1023 px
  intermediate; 1024–1279 px desktop where navigation fits; 1280 px and above full
  desktop. At 200 % zoom the compact layout may take over.
- Use cards for repeatable objects (meal cards, day cells, sidebar panels, timer,
  tiles) and intentionally framed tools. Page structure stays unframed; separate
  list rows with hairlines.

## Color

One semantic token system with two appearances (R05.1). Screens never hard-code
colours; feature code uses tokens only.

| Semantic token | Light | Evening |
| --- | --- | --- |
| Canvas | `#F7F4EE` | `#191D16` |
| Surface | `#FFFDFA` | `#24291F` |
| Raised surface | `#FFFFFF` | `#2D3327` |
| Primary text | `#19372A` | `#F4F0E5` |
| Secondary text | `#596358` | `#BDC4B4` |
| Primary action | `#A9472D` | `#E9B564` |
| On primary | `#FFFFFF` | `#211C13` |
| Selection surface | `#E3E9DA` | amber-tinted, starting from `#E9B564` (D-040) |
| Selection text | `#203F2D` | `#211C13` |
| Column highlight | `#E3E9DA` | `#39402D` |
| Divider | `#DCDDD3` | `#454C3D` |
| Focus | `#2E6A4C` | `#F5CA80` |
| Error emphasis | `#A33332` | `#FFB4AA` |

Derived roles (define as tokens; tune by contrast):

- **Column highlight** tints the selected or current day column and day cell
  (Week board, week strips); it is not a control selection.
- **Navigation indicator:** forest (`Primary text`) underline by day, amber
  (`Primary action`) by evening.
- **Status pills:** *Use soon* — warm peach fill with dark text by day, amber
  outline by evening; *Available* — sage fill by day and olive by evening (the
  `Column highlight` values); *Out* and
  *Unknown* — neutral with a text label. Status is never colour alone.
- **Tag chips:** one neutral chip style with a leading icon; meaning comes from
  the label and icon, not a per-tag colour (D-028).
- **Scene text support:** a restrained scrim may improve contrast over imagery but
  never replaces an asset's approved safe-text region (R17.4).

Dividers are not automatically sufficient control boundaries; test control
contrast separately. Disabled components keep their explanation readable.

## Typography

- **Two families only.** A self-hosted, licensed editorial serif for the
  wordmark, page titles, section titles, meal and recipe names, and large numerals
  (the cooking timer); the UI sans (Inter, already in the kit) for navigation,
  controls, chips, metadata, instructions, and forms. The serif is chosen by
  side-by-side comparison with the mockups (OFL candidates: Newsreader, Source
  Serif 4, Fraunces); record the licence and subsets. Similar-metric system
  fallbacks; never block content on font loading.
- **Scale (R05.1):** desktop page title 36–44 px; Today meal title 42–58 px as
  space permits; phone page title 28–32 px; phone meal title 28–34 px; section
  title 22–28 px; card title 18–22 px; body 16 px; metadata 13–14 px; minimum
  nonessential metadata 12 px. Wrap long names instead of shrinking them.
- **Eyebrows** (`TODAY · WEDNESDAY`, `USE SOON`) and storage/aisle group labels
  are small caps with modest letter spacing — the one sanctioned exception to zero
  letter spacing.
- **Numerals:** tabular numerals for timers, quantities, and amounts so digits do
  not shift.
- Support user font scaling and 200 % zoom without clipping.

## Spacing, Radius, Elevation

- 4 px base; groups of 8 / 12 / 16 / 24 / 32 / 48.
- Controls 44–48 px tall; touch targets at least 44 × 44 CSS px.
- Radii: controls and chips 8–10 px (chips may be fully rounded); cards 10–14 px;
  panels 14–18 px; an inset hero 16–20 px (maximum 1.25 rem).
- Shadows are subtle and only express elevation (sheets, menus, the timer card
  over imagery). Prefer spacing and hairlines to extra containers.

## Component Language

Binding grammar resolved from the mockups (D-027, D-028); the reusable
components themselves are listed in [`docs/concepts/UI-ARCHITECTURE.md`](docs/concepts/UI-ARCHITECTURE.md).

- **Primary action:** one filled button per region in the `Primary action`
  token (Start cooking, Plan my week, Add meal, Add item, Add ingredient, Next
  step). Secondary actions are outlined (action hue or neutral). Onward text
  links use the action hue with a trailing arrow ("Review groceries →").
- **Primary navigation and in-page tabs:** semibold label with a 2 px underline
  indicator. No pill-style active navigation.
- **Mode switches** (Meals / Nutrition / Time & cost, Day / All week, Review /
  Shop, Appliances / Cookware / Tools, Reading / Recipe map, Planned / Recorded /
  Expected, per serving / per yield) are `SegmentedControl`s; **filters** are
  `FilterChip`s. Both use the single `Selection surface` token per appearance (D-040); the
  Equipment mockup's forest-filled segment is not reproduced.
- **Appearance control:** an icon button reflecting the current setting that opens
  Light / Evening / Follow device — never a blind toggle.
- **Favourite vs save:** heart for a personal favourite; bookmark for saving an
  unsaved Explore suggestion; both with explicit accessible names.
- **Icons:** one line-icon family (`lucide-react`) with one icon per destination
  used everywhere (Today home, Week weekly calendar, Meals fork and knife,
  Groceries shopping bag, Kitchen cooking pot); custom equipment icons in the same
  stroke weight; distinct cooktop and oven icons.
- **Lists:** round checkboxes for shopping rows; square checkboxes for ingredient
  and prep checklists; right-aligned tabular amounts; small realistic ingredient
  icons are decorative (text carries meaning).
- **Decorative line art:** at most one botanical sprig per view, `aria-hidden`,
  never meaningful, light appearance only unless a matching evening treatment is
  drawn.
- **Copy:** functional and short. No slogans or script lettering.

Text must not overflow or overlap at any width. Fixed-format controls (day
cells, tiles, timer, nav items, badges) have stable dimensions so dynamic content
cannot shift layout.

## Imagery And Media Treatments

MealHero and meal cards render **one layout** with one of four treatments chosen
deterministically (R17.1): **Scene** (approved finished meal-in-scene
composition matching appearance and viewport), **Cutout** (approved transparent
subject in a compatible scene), **Editorial** (an ordinary licensed or user photo
in a deliberate rounded frame), **Minimal** (typography and ingredient summary on
a tinted panel). Presentation preference (Immersive, Editorial, Minimal),
appearance, and generation permission are three separate settings.

- Live text sits only inside an asset's approved safe-text region; long titles
  may switch to a quiet panel or the editorial treatment.
- Scale a composed scene as one image using its focal point and crop bounds;
  never cover food and table layers independently (R06.3).
- Phone Today artwork takes roughly 22–30 % of the initial viewport (160–260 px);
  phone recipe photos about 160–220 px. Artwork shrinks or disappears before the
  primary action does.
- Reserve dimensions; failed or missing media falls back to the next treatment
  without a broken-image icon or layout shift.
- Never bake text, UI, logos, or watermarks into assets. Generated imagery is
  captioned *Serving inspiration*; ordinary user photos are not labelled.

## Appearance Behavior

- **Light, Evening, Follow device.** Follow device is the first-run default unless
  a stored preference exists.
- Persist an explicit choice locally (for first paint) and in the account; apply
  it before first paint so an evening reload never flashes light. Match native
  control colour scheme.
- Appearance changes presentation only: never the meal, servings, date, slot, or
  plan, and never a generation job. A dark breakfast is still breakfast.
- Themes change colour, not structure: every surface keeps the same components
  and layout in both appearances.

## Motion

Reduced motion replaces spatial slides, bounces, and parallax with instant or
brief opacity changes. Otherwise use 120–220 ms state transitions. No animated
kitchen activity or simulated steam. Timer announcements happen on state changes
and expiry, not every second. Selecting an equipment tile may fade a scene layer;
it never moves other controls or resets scroll.

## Responsiveness

Design mobile-first; the phone gets complete capability, composed for sequence
and thumb reach rather than shrunk from desktop (D-036). Where the interaction
model changes, a breakpoint hook selects a different component tree off the same
data; where only arrangement changes, use CSS reflow.

- Today: text-left/scene-right hero → short scene above solid content with
  actions right after the title.
- Week: seven-column board only when each day gets ~136 px plus the label column
  → otherwise day selector, Day / All week, stacked slot cards; never a shrunken
  seven-column grid.
- Meals and Explore: three / two / one columns; no miniature two-column food
  cards on phones.
- Recipe: ingredient column beside method → stacked sections with a compact photo.
- Cooking: step rail + step + timer → single step, step-list sheet, footer actions
  in the thumb zone.
- Groceries: list + planning sidebar → full-width list, supplementary detail in
  sheets.
- Kitchen: rows + summary panels → rows and a use-soon strip; Equipment scene
  beside tiles → short scene above the category selector and two-column tiles.
- Desktop dialogs and panels → sheets or full pages that preserve state and
  history; long forms are full-screen editors with sticky actions above the
  keyboard.

## Customization

Support Light, Evening, and Follow device from the start, plus font scaling,
reduced motion, RTL, and the presentation preference (Immersive, Editorial,
Minimal). Customization is implemented through tokens and stateful preferences,
not one-off rewrites. The Settings surface covers Appearance, Meal artwork,
Generation & usage, Integrations, Notifications, Units/currency/timezone, Data &
exports, and Account — governed in style by this design and in content by the
product's users.

## Workflow Ergonomics

Design from the user's flow. For each surface, know the primary repeated action
(Start cooking on Today, Add to <day> in Explore, check a row in Shop mode), the
highest-risk action (apply a replan, confirm purchases, change an allergy), the
most common comparison (planned versus recorded, need versus stock), and the
first thing a new user must understand (what fits their rules). Remember
filters, tabs, Review/Shop mode, and scroll position across navigation.

## Feedback & State

Every user-triggered operation needs visible state. Loading, submitting, saving, syncing, refreshing, empty, partial, stale, success, validation-error, request-error, permission-denied, offline, and retry states are part of the design contract, not implementation polish.

Buttons that start asynchronous work acknowledge the click immediately, show a
busy state, prevent harmful duplicate submission, and restore a usable state when
finished. "Saved" means the durable operation succeeded, not that a component
updated. Offline indicators distinguish cached data, local pending actions, and
server-confirmed saves. Toasts sit above the phone tab bar and safe area and never
cover primary actions. Load failure is never presented as an empty account.

## UX-State Contract

Every major surface declares the lifecycle states that users can encounter:
`idle`, `pending`, `success`, `loading`, `saving`, `syncing`, `refreshing`,
`empty`, `partial`, `stale`, `validation-error`, `request-error`, `retrying`,
`permission-denied`, and `offline`. A scenario may add a more specific state
(for example `no-eligible-meals`, `unknown-cost`, `timer-elapsed`,
`plan-changed-while-shopping`), but it must keep the generic state legible and
actionable. The state must be represented in the accessibility tree with a role,
name, status, or alert that explains what changed and what the user can do next.

Experience-manager's state coverage check uses this section as the contract
source. Pages that do not encounter a state may omit it with a documented
reason in their experience spec; a blank or silent state is never an omission
reason.

## Baseline Capture Matrix

The default machine-evidence run uses a bounded covering set of at most 12
captures per active surface. It includes every declared viewport, Light and
Evening desktop coverage, English and Arabic direction coverage, both motion
preferences, and rest/hover/focus-visible/pressed interaction states. The
matrix is a covering set rather than a Cartesian product so capture cost stays
predictable; a claim scoped to an additional axis (for example the phone
Evening Today hero) opts into the smallest targets needed for that claim. The
redesign's visual acceptance additionally compares R27.5 captures side by side
with the concept mockups.

## Request Lifecycle

For every network call, long-running task (plan generation, import, generation
job, calendar sync), file operation, or mutation, design idle, pending, success,
failure, retrying, and unavailable states. Optimistic updates are allowed only
where rollback is clear and revision-aware (checking a shopping row, selecting an
equipment tile); plan apply, purchase confirmation, import, and generation always
wait for canonical success. Background sync exposes freshness and pending state.

## Accessibility

WCAG 2.2 AA target (R07): visible focus in both appearances, accessible names on
every icon action, correct tab roles and keyboard behaviour, dialogs that restore
focus, semantic tables or labelled lists for ingredients and inventory, 44 px
targets, Move/Copy alternatives to drag, equipment tiles as the complete
alternative to scene markers, non-colour status cues, contrast verified on real
captures including text over imagery.

## Do's and Don'ts

### Do

- Start UI work by reading this file, the mockups guide, and the surface's section in `docs/internal/REDESIGN_PLAN.md`.
- Use tokens for every colour, size, and radius; keep feature code free of raw palette classes.
- Compose phone and desktop deliberately; keep the primary action in reach.
- Let food photography carry warmth and keep chrome quiet.
- Keep unknown, zero, partial, and stale visually and textually distinct.
- Design empty, loading, partial, error, offline, and nonideal-media states for every surface.

### Don't

- Build a stock dashboard or a card wall and call it this design.
- Make the phone a cramped desktop, or show a seven-column grid on a phone.
- Put text, UI, or slogans inside images, or glass panels over busy photography.
- Use colour as the only signal.
- Let a component library or an asset become a second source of design truth.
- Change structure between appearances.
