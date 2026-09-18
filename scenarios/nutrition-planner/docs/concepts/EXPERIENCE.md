# Experience Design

## Purpose Of This Document

Record the UI decision for Nutrition Planner: what people will compare it to, which surface
matters most, how that surface lays out at each width, and what the design is accountable
for. It is the prose companion to the machine-readable contract in `experience/`; where the
two disagree, `experience/` wins because a validator reads it.

This document does not describe the `ui/` source tree. That is
[`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md). The binding token, motion, and status-semantics
contract lives in the root `DESIGN.md`; this document must not restate or edit it.

## The Comparison

The comparable product is **a calm daily meal-decision assistant** — closer to a focused
personal planner than to a data dashboard.

People will judge Daily against the thing it replaces: opening a fridge and deciding, or
opening a notes app and rereading the same meals. They may also compare its recipe views to
well-made cooking apps and its summaries to a simple habit tracker. Those comparisons set
the bar:

- **Decision first, not data first.** The first screen answers one question — what should I
  eat next, and what do I do? — with one recommendation, not a wall of nutrient gauges.
- **Low ceremony.** A returning user sees the next meal immediately; a name-only meal is
  savable; a swap does not require writing a reason.
- **Calm, capable, slightly playful.** Generous whitespace, a strong cobalt primary action,
  warm lime highlights, white surfaces, dark blue text, and restrained metadata. Reassuring
  copy such as "Dinner, decided." and "Amount not set". No streak pressure, no shame.
- **Honest, not clinical.** A missing value reads as unknown, not as zero, and the language
  prefers "Fits your current rules" over "Safe for your allergy."

## The Primary Surface

**Today** is the one surface people open first and stay on. It answers "what should I eat
next, and what do I do?" with minimal navigation.

The hero is the **next-meal card**. It contains the meal name and one-line description; an
optional image or ingredient composition; estimated active preparation time (elapsed time
lives in details); energy, protein, portion cost, and effort, **each with its own unknown
state**; a concise, calculation-backed reason (for example, "Uses your opened rice; 6
minutes of active prep"); a primary **Let's make it** action, a secondary **Swap meal**
action, and a quieter **See the recipe map** action.

The hero is the only place a number is allowed to lead. Every number it shows is backed by
an evaluation, and a total with unknown components reads "$3.10 known + unpriced items" or
equivalent. A missing quantity or price never renders as `$0.00` or `0 min`. A known zero is
valid only when it was actually asserted (a zero-cost ingredient, zero active minutes) and
stays visually distinct from a missing value.

Below the hero is an easy-night card with a lightning icon and "Not a cooking kind of
night?", opening an effort-focused swap scoped to the selected meal.

**Layout.** Desktop uses a roughly **1.8:1 split**: the primary meal card on the left and
compact context cards on the right (the selected scope, kitchen rules, and the wider week).
At narrower widths the main card and supporting content **stack**, the main action stays
visible without traversing an analytics panel, and mobile keeps a persistent four-item
navigation area. The page body never scrolls horizontally; only an explicitly wide recipe
map scrolls inside its own region.

**States.** Today must render these distinctly, never as a fake empty or zeroed dashboard:

| State | What the user sees | Recovery |
|---|---|---|
| Loading | Stable shell and bounded skeletons; no sample food shown as saved data. | — |
| No configured plan | A short invitation to generate or choose a meal. | Generate draft / Choose meal / Add favorite |
| No eligible meals | Rule-conflict summary with counts. | Add a matching meal / Review restrictions; no automatic relaxation |
| Plan exists, data incomplete | The meal stays usable with unknown metrics. | Fill relevant detail / Continue without a completeness claim |
| Archived recipe still scheduled | The pinned revision remains available. | Replace a future occurrence if desired; preserve history |
| Account load failed | A clear load error and retry; edits are not allowed against a fake empty account. | Retry |
| Save failed after edit | Edited state remains visible with a pending marker. | Retry / Export pending changes / Resolve conflict |
| Provider unavailable | Existing local records still work. | Manual entry / Cached results labeled by age / Retry |

The selected scope control is **Meal / Day** and can widen to the week. Planned and recorded
totals are shown separately; an open or unrecorded slot contributes unknown future intake,
not zero or assumed compliance. There is no universal "all nutrients covered" indicator.

## Shell Configuration

| Setting | Value | Why |
|---|---|---|
| Kit | `vrooli-default` | The shared shell archetype plus the design-token contract gives a calm, accessible, responsive frame. The cobalt/white/lime direction is expressed **through the design tokens**, not by redrawing chrome; `DESIGN.md` is the source of truth and is not edited here. |
| `density` | `sidebar` | Desktop has five stable destinations plus secondary ones; a persistent sidebar keeps route meaning legible and matches the repository's app pattern. |
| `mobileNav` | `tabs` | Mobile gets a persistent four-item bottom navigation (**Today**, **Your week**, **Groceries**, **Meals**) placed above the safe-area inset. Kitchen and transfer are reached from context, not the bar. |
| `mainMode` | `scroll` | Today is a decision surface read top to bottom; the hero must not be trapped in a fixed pane. Dialogs and the recipe map own their own scroll regions. |

The application shell, page headers, settings rows, empty states, cards, buttons, badges,
and async regions come from `react-component-library` as linked package imports. The shell is
configured in `ui/src/layout/AppShell.tsx` and never redrawn locally; **Settings owns every
preference**.

## What The Design Is Accountable For

Three things, in order, that the UI must make true:

1. **The next decision is one screen away.** A returning user can see the next meal, read a
   calculation-backed reason, and start or swap it without navigating or filling a form.
2. **Every derived number is honest about its scope.** Unknown, zero, partial, and stale are
   four different facts, each rendered with its basis and completeness; no total is presented
   as exact when contributors are missing, and no source is silently converted to zero.
3. **Required rules are visible and not negotiable.** Diet, exclusions, allergen evidence,
   and kitchen feasibility are shown as reasons with remedies; a preference weight can never
   make an ineligible meal look compliant.

Under those three, the design is also accountable for WCAG 2.2 AA: keyboard paths, zoom and
reflow, visible focus, meaningful accessible names on icon-only actions, non-color status
cues, 44 px targets for frequent touch actions, contained and restored dialog focus, and
reduced-motion support. Live regions announce save errors and import outcomes without
narrating every slider movement.

**Progressive disclosure.** Show only material caveats near an action. Provenance,
individual nutrient lines, planner diagnostics, and source comparisons belong in details
panels. Do not expose database, provider routing, optimization weights, or job internals in
ordinary eating flows. When an advanced feature is absent in the selected release, omit it or
name its fallback; never show a fake progress animation, a fake live price, or a success
toast without a result.

**Binding versus illustrative.** `DESIGN.md` is binding for tokens, color roles, typography,
spacing, radius, motion, status semantics, responsive transformations, and accessibility
floors. Any concrete component list or example layout, in this document or `DESIGN.md`, is
illustrative — implement every control the product actually needs, styled to that floor.

## Information Architecture

Four primary destinations, two secondary contexts, and the settings surface. Route state is
bookmarkable where useful; a refresh preserves the meaningful destination, selected week, and
recipe rather than reopening an arbitrary default day.

| Surface | Route | The question it answers |
|---|---|---|
| Today | `/today` | What should I eat next, and what do I do? |
| Your week | `/week?start=YYYY-MM-DD` | What does this week look like, and what can I change? |
| Groceries | `/groceries?plan=<id>` | What do I need to buy, what do I have, and what will it cost? |
| Meals | `/meals` | What can I cook, and does it fit my setup? |
| Meal detail | `/meals/:id` | What is this meal's recipe, revision, and readiness? |
| Your kitchen | `/kitchen` | What are my food rules, appliances, routine, and priorities? |
| Nutrition | `/nutrition` | How do planned and recorded totals compare with my targets? |
| Import / export | `/data` | How do I move, back up, or print my data? |
| Settings | `/settings` | What changes behaviour for everything — theme, locale, font scale, accessibility, notifications, workspace? |

**Primary destinations** are Today, Your week, Groceries, and Meals. **Your kitchen** and
**Import / export** are secondary: kitchen opens setup and preferences, and transfer is
globally accessible and also available contextually from Meals, Your week, Groceries, and a
recipe. Nutrition analysis, inventory/price details, and supplements are secondary
destinations reached from relevant summaries or Your kitchen. Keep the four primary
destinations stable; do not add a top-level tab per domain entity.

## Cross-References

- [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md) — the `ui/` source tree and shell layout
- [`DOMAINS.md`](DOMAINS.md) — which domain backs each surface
- [`FLOWS.md`](FLOWS.md) — the journeys these surfaces carry
- [`../reference/product-specification.md`](../reference/product-specification.md) — §4, §5, §7–§10, §12.3
- [`../../experience/index.json`](../../experience/index.json) — the typed experience contract
- [`../../DESIGN.md`](../../DESIGN.md) — the binding token contract
- [`../reference/component-library-gaps.md`](../reference/component-library-gaps.md) — what the shared library does not yet provide
- [`../../PRD.md`](../../PRD.md) — the operational targets these surfaces claim against
