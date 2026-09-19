---
id: vrooli-default
version: 0.2.0
name: Observatory
description: Calm, editorial, place-making planning UI with coordinated day and night landscape environments over the Vrooli token plumbing.
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
  cardRadiusMaximum: "0.875rem"
  defaultMode: "system"
  supportedModes: ["light", "dark", "system"]
  responsiveBaseline: "mobile-first"
  dominantPalette: "warm-ivory-day-and-midnight-plum-night-with-stable-source-accents"
---

# Observatory Design

`DESIGN.md` is the source of truth for scenario UI decisions. Stack-specific adapters may translate these tokens into CSS, Tailwind, egui, native mobile themes, or future targets, but adapters must not redefine the design language.

**Observatory is a deliberate product theme, not the default operational
console.** It is built on the Vrooli token plumbing (the `design-tokens.css`
sources, the `space-*` scale, semantic status roles, and the shared component
library shell) but intentionally departs from the console's "avoid atmospheric
backgrounds / compact-operational-only" defaults because the operator selected
and approved this direction (decision D08). Beauty is a stated product
requirement here: the coordinated day/night landscape is the wedge that earns
repeated daily use, and repeated use is what makes the honest planning loop
valuable. Do **not** substitute a stock component-library dashboard and call it
equivalent. The recommended tokens below are starting points to tune and
verify against real screenshots (plan §6.3) — not colors sampled from a
generated image.

## How To Read This Document

This file mixes two kinds of guidance, and the distinction matters.

- **Binding contract** (must follow): the tokens, color roles, typography scale, spacing, radius, motion rules, status-color semantics, responsive transformations, accessibility floors, the appearance model (Auto/Day/Night with coordinated scenes), the truthful time-geometry rule, and the overall "calm, editorial, place-making" feel target. These define the design language and must be respected.
- **Illustrative examples** (shape, not checklist): any concrete list of components, layouts, page surfaces, settings controls, or copy. These exist to communicate *shape and feel*, not to enumerate the features your scenario must (or must not) ship. Implement every feature the product actually needs, even if it is not listed here.

Concrete rule of thumb: this design tells you *how* a surface should look, behave, and feel — not *which* surfaces the product must include (the surface set is owned by [`docs/concepts/EXPERIENCE.md`](docs/concepts/EXPERIENCE.md) and `experience/`).

## Intent

Observatory helps a person inspect a realistic day, choose the next useful
action, work, record a small correction when needed, and see how that affects
the plan — inside an interface calm enough to return to every day. The feeling
target is **editorial and unhurried**: expressive serif headings, generous
whitespace, tabular numerals for time, and a quiet landscape that creates a
sense of place without ever competing with text, inputs, or the timeline.

The recurring environment is a small hillside observatory overlooking a valley,
lake, and layered distant mountains. Day shows warm sunlit terrain and
atmospheric depth; night depicts the same buildings, trees, shoreline, and
viewpoint as dark silhouettes with restrained settlement lights. The scene sits
mostly below the working area. It should never feel like a marketing hero, a
neon AI product, or a decorative dashboard — and it must never reduce the
legibility of the working surfaces above it.

There is no approved permanent product name, logo, or slogan. "Planner" and
"Observatory" are replaceable configuration; any decorative slogan that appears
in a generated concept is optional and off by default in the working UI.

## Layout

The reference desktop composition (~1440–1600px, plan §6.2):

- A **narrow left navigation rail** (~120–152px): wordmark at top, the five
  primary destinations below (Today, Plan, Goals, Focus, Review), profile/
  settings near the bottom. Source apps are contextual labels, not a second
  navigation hierarchy.
- A main canvas with ~32–48px outer padding on a responsive grid (never fixed
  image coordinates).
- An expressive **serif date heading** ("Today" + the readable local date),
  ~42–72px by width with a maximum so the date never pushes work below the fold.
- The **Auto / Day / Night** appearance control at top right with an accessible
  selected state.
- An upper working row: one prominent **next-action surface** (~58–62% of
  usable width) and a quieter capacity/commitment region in the remainder.
- A wide horizontal **Today timeline** below the working row with a real time
  scale, a now indicator, fixed/flexible distinctions, and open space that
  *means* something.
- The lower ~20–30% may hold the scene when there is room; content can extend
  beyond it, and the art is never a fixed overlay obscuring scrollable work.

At 1024–1279px, shrink or collapse the rail, reduce the heading, and stack the
commitment summary below capacity. At tablet/phone widths, stack the action and
capacity regions and switch to the mobile pattern below. Never truncate the
sole accessible representation of a task title.

Use cards for repeated records, focused tools, modals, and the next-action
surface — not to wrap whole page sections. Keep page-level structure unframed
unless the content is a true object or tool.

## Color

Two coordinated appearances share one set of semantic roles. Recommended
starting tokens (tune and verify; plan §6.3):

| Token | Day | Night |
|---|---|---|
| Canvas | `#F5F0E6` warm ivory | `#141321` midnight plum |
| Raised surface | `#FFFDF7` | `#201E30` |
| Primary text | `#192622` forest ink | `#F5EEDF` warm cream |
| Secondary text | `#565E59` | `#C0B9CB` |
| Primary action | `#A64327` burnt sienna (near-white text) | `#F2B75A` warm amber (near-black text) |
| Divider | `#D7CEC0` | `#444054` |
| Cadence source accent (example) | `#7055AE` | `#B69DE7` |
| Daily source accent (example) | `#456B58` | `#A6C6B5` |
| Attention | deep burnt orange (with label/icon) | warm amber (with label/icon) |

Semantic status roles are preserved from the platform contract and paired with
label/icon/shape, never color alone: success (completed work), warning/amber
(pending attention, at-risk commitments), destructive/red (failed, blocked),
and a technical/info accent. Source colors (e.g. Cadence purple, Daily green)
stay **semantically stable across appearances**, even as shades shift for
contrast.

The night next-action surface may remain warm ivory with dark text (matching
the approved concept) but must use its **own on-surface tokens** rather than
inheriting light-mode white text; offer a subdued night-surface setting for
low-light comfort. Avoid neon, uncontrolled glass transparency, decorative
gradient blobs, and charts-as-decoration.

## Typography

- **Headings:** an expressive editorial serif (Fraunces or an equivalent;
  verify availability and licensing locally) for the date heading and section
  titles. Reserve the largest sizes for the Today date.
- **Body & controls:** a highly readable UI sans (Inter or the platform sans
  stack). Base body 16px for input safety; dense panels may use 14px where
  scanning benefits, with legible controls and usable targets.
- **Numerals:** **tabular/aligned numerals** for timers, capacity totals, and
  every time value, so digits do not shift as they change.
- Letter spacing is zero by default. Support user font-size scaling.

## Time Geometry And Visual Semantics

This is a binding correctness rule, not styling (plan §6.6, fixture F14):

- On a linear timeline, **position and width represent time accurately**. A
  45-minute interval cannot occupy 90 minutes to fit its text — use an adjacent
  label, popover, stacked collision lane, or an agenda representation instead.
- Day/week scales use actual instants when DST makes a nonstandard day; a
  collapsed nonworking interval carries an explicit break marker and never
  silently distorts the scale.
- Represent fixed commitments with an anchor/lock and clear boundary; flexible
  sessions with move affordances; pending proposals with labeled original vs
  proposed geometry; actual activity as a distinct comparison layer; forecast
  ranges labeled as scenario ranges. Color supplements shape and text.
- Distinguish **unscheduled**, **unavailable**, and **reserved breathing room**
  time — white space is not automatically usable capacity. Show a legend when
  relevant.

## Scene Assets

Treat the landscape as a **replaceable decorative asset with all meaningful,
accessible content outside it** (plan §6.4):

- Author matched day/night versions from the same master composition; export
  responsive sizes and a modern compressed format with a compatible fallback.
- **Never bake task text, buttons, the date, or the timeline into the image.**
- Bundle or cache assets locally; a third-party image host is never a
  prerequisite for opening Today, and decorative assets must not block the
  first task view. Provide an art-free setting and a reduced-scenery intensity.
- If artwork is unavailable early, a deliberate tonal placeholder is acceptable
  for R0, but the approved scenic experience is a **visual release gate for R1**.
  Never regenerate a new image on every visit.

## Appearance Behavior

Support **Auto, Day, Night** (plan §6.5):

- Default Auto uses the user's local display timezone with editable transition
  hours (suggested 07:00 / 19:00, shown as defaults). It requires no location
  access and claims no true sunrise/sunset. System appearance may be an optional
  alternative Auto policy; astronomical transitions are later refinement.
- Manual override persists per user. Theme changes never alter scheduling
  timezone, data selection, the current date, or the plan.
- **During a focus session, defer an automatic transition** until the next
  break or session boundary. A "Night" appearance chosen at 10:00 is valid.
- Respect reduced motion; otherwise use a restrained opacity/color transition
  that never moves content.

## Components

Controls should be predictable and optimized for calm repeated work: icon
buttons for familiar actions, segmented controls for modes (Auto/Day/Night is
one), toggles for binary settings, inputs/steppers for numeric values, menus
for option sets, tabs for sibling Plan perspectives, and one dominant primary
action per surface (the next-action surface has a single "Start focus" primary
and a quieter "Open source"). Illustrative reusable domain components live in
[`docs/concepts/UI-ARCHITECTURE.md`](docs/concepts/UI-ARCHITECTURE.md).

Text must not overflow or overlap at any width. Fixed-format controls (timeline
blocks, capacity counters, nav items, badges) need stable dimensions or
responsive constraints so dynamic content cannot shift the layout.

## Responsiveness

Design mobile-first; mobile provides complete capability, not a read-only
fallback (plan §6.7). Phone Today defaults to a **vertical agenda**, the
next-action surface, a compact capacity line, and a commitment notice, with
bottom navigation for the five destinations. A compressed horizontal desktop
timeline is **not** the primary phone experience. Common transformations:

- Left rail → mobile bottom navigation.
- Desktop modal → mobile bottom sheet or full-screen panel (forms, review
  steps, filters, multi-step decisions).
- Desktop split/timeline → mobile day-grouped list or stepwise panels with
  preserved context.
- Hover affordance → visible or long-press-safe alternative.

Use safe-area padding; primary touch targets ≥44px; support optional
left/right-handed placement for frequent bottom actions.

## Customization

Support light (Day), dark (Night), and system-following Auto from the start,
plus font-size scaling, reduced motion, RTL, an art-free/reduced-scenery
setting, and a subdued night-surface option. Customization is implemented
through tokens and stateful preferences, not one-off rewrites. Build a full
settings surface for everything the product needs — appearance, availability,
focus preferences, integrations, sharing management, notifications, data, and
learning controls — governed in *style* by this design and in *content* by the
product's users.

## Workflow Ergonomics

Design from the user's flow. For each major surface, identify the primary
repeated action (Start focus on Today), the highest-risk action (Apply a plan
change), the most common comparison (planned vs actual, promise vs forecast),
and the first thing a new user must understand (what fits today). Experienced
users move with short pointer travel, predictable keyboard focus, remembered
filters, and stable navigation; new users get enough structure and progressive
disclosure to act without documentation.

## Feedback & State

Every user-triggered operation needs visible state. Loading, submitting,
saving, syncing, refreshing, empty, partial, stale, success, validation-error,
request-error, permission-denied, offline, and retry states are part of the
design contract, not implementation polish.

Buttons that start async work acknowledge the click immediately, show a busy
state, prevent harmful duplicate submission, and restore a usable state when
finished. **Start focus, Apply a proposal, create a share, and connect a
provider show pending state until canonical success — never a false optimistic
timer.** Forms preserve input on failure, place field-level validation near the
control, and show a form-level summary when submit fails. Lists, timelines, and
capacity panels have purposeful loading/empty/partial/error states, not blank
space. Error messages explain what happened, what is still safe, and the next
action — without exposing stack traces, secrets, tokens, raw provider errors,
or hidden record IDs.

Distinguish the honest states this product depends on: **available time vs
reserve**, **promise vs forecast**, **planned vs actual**, **source status vs
planner state**, and **an empty view vs a failed query**. Each needs distinct,
legible copy.

## UX-State Contract

Every major surface declares the lifecycle states users can encounter:
`idle`, `pending`, `success`, `loading`, `saving`, `syncing`, `refreshing`,
`empty`, `partial`, `stale`, `validation-error`, `request-error`, `retrying`,
`permission-denied`, and `offline`. A scenario may add a more specific state
(e.g. `at-risk`, `beyond-horizon`, `over-capacity`, `nothing-fits`) but must
keep the generic state legible and actionable. The state must appear in the
accessibility tree with a role, name, status, or alert that explains what
changed and what the user can do next. Experience-manager's state-coverage
check uses this section as the contract source.

## Baseline Capture Matrix

The default machine-evidence run uses a bounded covering set of at most 12
captures per active surface: every declared viewport, light (Day) and dark
(Night) desktop coverage, English and Arabic direction coverage, both motion
preferences, and rest/hover/focus-visible/pressed interaction states. The
matrix is a covering set, not a Cartesian product, so capture cost stays
predictable; a claim scoped to an additional axis (e.g. landscape contrast in
the brightest/darkest regions) opts into the smallest targets needed for it.

## Request Lifecycle

For every network call, long-running local task (proposal generation, provider
sync, forecast refresh), file operation, or resource mutation, design the
lifecycle deliberately: idle, pending, success, failure, retrying, and
disabled/unavailable. Slow operations show progress, skeletons, or queued
status with stable layout; if progress is unknown, show an indeterminate but
visible pending state. Optimistic updates are allowed **only** when rollback is
clear (simple reversible fields with revision-aware rollback); proposal apply,
source completion, share creation, and timer start always wait for canonical
success. Background sync exposes freshness, last-updated time, stale data, and
reconnection status when the result affects decisions.

## Accessibility

Target **WCAG 2.2 AA as a release gate** (plan §6.7). Verify ≥4.5:1 normal-text
contrast in both appearances *and* over the brightest and darkest landscape
regions, relevant non-text contrast, visible focus, disabled/hover/active
states, and target-size rules (~44px where practical). Every drag operation has
a keyboard and menu alternative; task start/stop needs no precise pointer;
announce meaningful timer-state changes without narrating every second. Ensure
sensible focus return from dialogs, semantic headings, reflow, text zoom to
200%, honored reduced motion, RTL, and no color-only status.

## Do's and Don'ts

### Do

- Reproduce the Observatory composition and atmosphere, then fix mockup
  shortcuts (crowded labels, example arithmetic) against truthful behavior.
- Keep the scene below the working area and entirely decorative; all meaning
  lives in real UI above it.
- Make position and width on the timeline mean real time.
- Support Auto/Day/Night, reduced scenery, art-free, and subdued-night settings.
- Preserve source colors as stable semantic accents across appearances.
- Design honest empty/over-capacity/nothing-fits/stale states with distinct copy.

### Don't

- Substitute a stock operational dashboard and call it equivalent to Observatory.
- Bake text, the date, buttons, or the timeline into the landscape image.
- Require a third-party image host or regenerate art on every visit.
- Stretch a 45-minute block to 90 minutes to fit a label.
- Move content during a theme transition, or transition mid-focus-session.
- Let the scene reduce contrast or legibility of any working surface.
