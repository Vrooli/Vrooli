# UI Architecture

## Shared contract

Read the [UI Architecture shared guide](/scenarios/template-manager/docs/concepts/UI-ARCHITECTURE.md). Template Manager owns
the common contract and its implementation references — the shell/slot
taxonomy, the archetype catalog, the density and navigation conventions,
and the rule that a scenario *configures* the react-component-library
shell rather than redrawing it. This document records only what is
specific to Personal Planner; it does not restate the shared contract.

## Scenario details

Personal Planner's UI is the operator-approved **Observatory**
experience (decision D08). Its look, feel, motion, color, appearance
model, and time-geometry rules are governed by the binding design
contract in [`../../DESIGN.md`](../../DESIGN.md); the surface set and
information architecture are owned by
[`EXPERIENCE.md`](EXPERIENCE.md). This file describes how that
experience maps onto the shared shell: nothing here is built yet — the
layout below is the *configuration target*, not shipped code.

### Shell archetype and configuration

Personal Planner uses the react-component-library **navigated-console**
archetype — a left rail of destinations plus a scrolling main pane —
styled by the Observatory theme. The shell is **configured, not
redrawn**. The configuration lives in `ui/src/layout/` and
`ui/manifest.json`, and touches these files:

| File | Role | Observatory setting |
|---|---|---|
| `ui/manifest.json` | Declares the shell to the platform. | `shell.archetype` = navigated console; plus `shell.asset`, `shell.entry`, `shell.export`. |
| `ui/src/layout/AppShell.tsx` | Three shell constants that tune the archetype. | `density` = editorial/comfortable (**not** compact — this is the deliberate departure from the operational-console default); `mobileNav` = bottom navigation; `mainMode` = scrolling. |
| `ui/src/layout/navItems.tsx` | The primary destinations shown in the rail. | The five destinations below; Settings and profile sit near the rail bottom. |
| `ui/src/layout/BrandMark.tsx` | The wordmark in the rail. | The "Planner / Observatory" wordmark until a permanent brand is chosen through brand-manager. |

Because these are shell *constants and data*, not a bespoke layout, the
Observatory theme rides on the shared shell plumbing (tokens, spacing
scale, semantic status roles) while expressing the editorial,
place-making feel. If the navigated-console archetype ever cannot carry
the editorial composition — most plausibly the decorative full-bleed
scene band — record a scoped `shell-ejection` in
[`../reference/component-library-gaps.md`](../reference/component-library-gaps.md)
naming the exact `ui/src/` files that own the exception. Pre-1.0
availability alone does not justify a forced or ejected shell.

### Destinations

`navItems.tsx` declares exactly five primary destinations, in order:

- **Today** — the primary surface; answers "what can I usefully do
  now?" and "what needs my attention?" with a next-action surface,
  scoped capacity, a commitment notice, and the truthful Today timeline.
  Fully usable manually before any automation, provider, or model is
  configured.
- **Plan** — one record set projected through Day, Week, Month, Agenda,
  Timeline, Capacity, and Commitments perspectives (not seven stores).
- **Goals** — outcomes, milestone status, linked work, and time
  allocation.
- **Focus** — the focused working screen (openable without a task for
  spontaneous work): objective, timer/mode, next step, source link, next
  stop time, pause/end.
- **Review** — daily and weekly review with coverage, carry-forward, and
  inspectable insights.

Commitments have a dedicated filter/view inside Plan and contextual
access from Today; they are promoted to primary navigation only if usage
demonstrates the need. Global search / quick capture is reachable
everywhere; **Settings** (in the rail bottom, not the primary five)
holds availability, appearance, focus preferences, integrations, sharing
management, notifications, data, and learning controls.

### The appearance control

The **Auto / Day / Night** appearance control lives in the shell's
`utility` slot, rendered as a segmented control with an accessible
selected state (top-right on desktop). It switches only the palette and
scene — never card order, block placement, capacity values, scroll
position, the scheduling timezone, the current date, or the plan. Auto
uses editable transition hours (default 07:00 / 19:00) and **defers an
automatic transition during a focus session** until the next break or
session boundary. Settings owns the full appearance model (art-free /
reduced-scenery / subdued-night options); the utility-slot control is the
quick access to it. See [`../../DESIGN.md`](../../DESIGN.md) for the
binding appearance behavior and color roles.

### The Observatory scene band

The decorative day/night landscape occupies a **lower scene band**,
rendered below the working area and behind an independent error boundary.
It is a replaceable decorative asset: all meaningful, accessible content
lives in real UI above it; no task text, buttons, date, or timeline is
ever baked into the image; and it must never reduce the contrast or
legibility of any working surface (contrast is verified over the
brightest and darkest scene regions as a release gate). If the artwork
fails to load, the themed surfaces above it stay fully readable. The
scene is not a fixed overlay obscuring scrollable work.

### Illustrative reusable domain components

Personal Planner builds **reusable domain components** rather than a
separate task card per screen, so the same domain rules are exercised by
both the full planner and a compact source-app widget (plan §22.2). The
names below are illustrative shape, not a required inventory; adapt them
to the repository's component and naming conventions, and separate
presentation from mutation hooks:

| Component | Responsibility |
|---|---|
| `NextActionCard` | The Today next-action surface: criterion/eyebrow, available window, estimate, source chip, one dominant *Start focus* and a quieter *Open source*, and an alternative action. |
| `CapacitySummary` | Explicit scope, usable time, planned time, reserve, and unknown demand with aligned tabular numerals; a details affordance — the same values the API returns (INV-16). |
| `TimeHorizon` | The proportional timeline: block width ∝ duration (truthful time geometry), a now marker, density controls, and an accessible list equivalent. |
| `WorkItemRow` / card | Status, source, remaining effort, target/commitment badges, and allocation summary for a work item. |
| `CommitmentStatus` | Original vs current promise, forecast, risk state, definition of done, and a history link — promise and forecast kept visibly distinct. |
| `ProposalDiff` | Move/add/remove operations, rationale reason codes, conflicts, selected-subset validation, and apply — original vs proposed geometry labeled. |
| `FocusPanel` | Current intention, authoritative phase/time from a confirmed timestamp, pause/finish, and interruption capture. |
| `HistoryDrawer` | Observed changes, user explanations, provenance, and revision comparisons — evidence separated from interpretation. |
| `InsightCard` | Evidence sample size and coverage, a suggested setting change, a preview, and accept/dismiss (acceptance is a separate auditable setting command). |
| `SharePreview` | The exact server-side viewer projection and the selected field mask, so an owner sees precisely what a recipient will see. |

Every major surface must declare and legibly render the lifecycle states
from the UX-state contract in [`../../DESIGN.md`](../../DESIGN.md)
(loading, empty, partial, stale, over-capacity, nothing-fits,
source-unavailable, saving-failed, a running focus session, …) in both
Day and Night appearances, and must distinguish the honest states this
product depends on: available time vs reserve, promise vs forecast,
planned vs actual, source status vs planner state, and an empty view vs a
failed query.

### Cross-references

- [`../../DESIGN.md`](../../DESIGN.md) — the binding Observatory design contract
- [`EXPERIENCE.md`](EXPERIENCE.md) — UI decision, primary surface, information architecture
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and surface roles
- [`DOMAINS.md`](DOMAINS.md) — the capabilities behind each destination
- [`../reference/component-library-gaps.md`](../reference/component-library-gaps.md) — recorded shell exceptions
