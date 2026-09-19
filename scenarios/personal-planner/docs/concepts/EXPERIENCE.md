# Experience Design

## Purpose Of This Document

This document records the UI decision for Personal Planner: the product
people will compare it to, the one surface that matters most and its
layout at phone and desktop widths, how the library shell is configured
(not redrawn), what the design is held accountable for, and the
information architecture of the whole app. It is the prose companion to
the machine-readable `experience/` specs and sits under the binding
design contract in [`../../DESIGN.md`](../../DESIGN.md). The direction
here is the operator-approved **Observatory** (decision D08).

## The Comparison

**Compared to:** the calm, editorial tier of personal planners —
Sunsama, Amie, Structured, and Fantastical — not the operational-console
tier (Linear, Jira) and not a raw calendar grid (Google/Outlook).

**Why this bar:** the personal-planning market is crowded, and feature
parity is table stakes. The durable differentiator is **beauty and calm
paired with honest accounting** (see [`../business/MONETIZATION.md`](../business/MONETIZATION.md)).
Competitors either look beautiful but plan naively (no remaining-effort
model, no capacity math, no explainable forecasts) or plan seriously but
feel like enterprise tools. Personal Planner must feel like a place you
*want* to open each morning — the Observatory landscape and expressive
serif typography are the wedge — while the numbers underneath never lie
green. If we match the competition's beauty and beat it on truthfulness,
we win the segment; if we match its planning and lose on beauty, we do
not get the daily open that makes planning valuable.

**Explicitly not:** a generic white-card/blue-button dashboard (the
operator rejected exactly that), a marketing site, or a neon "AI product."

## The Primary Surface

**Today** is the surface that matters most. It answers two questions:
"What can I usefully do now?" and "What needs my attention?"

**Desktop (~1440–1600px):**

```
┌──────┬───────────────────────────────────────────────────────────┐
│ rail │  Today                                    [Auto][Day][Night]│
│      │  Monday, September 21               (serif date heading)    │
│ �―Today│ ┌───────────────────────────┐  TODAY'S CAPACITY           │
│  Plan │ │ UP NEXT · STARTS NOW  45m  │   3h planned  4h avail  1h  │
│  Goals│ │ Draft the launch story    │   ────────────────────────  │
│  Focus│ │ • Cadence                 │  DEMO COMMITMENT            │
│  Rev. │ │ [▶ Start focus][Open draft]│  Promised Thu → Forecast Fri│
│      │ └───────────────────────────┘                             │
│      │  Your day  09 ─ 10● ─ 11 ─ 12 ─ 13 … 19  (real time scale) │
│      │  [Draft][Test release]  [Review]  [Unscheduled]  [Dinner]  │
│      │  ░░░░░░░░ observatory landscape (decorative, below work) ░░ │
└──────┴───────────────────────────────────────────────────────────┘
```

- Left rail (~120–152px), the five destinations, wordmark, settings.
- A single prominent **next-action surface** (eyebrow, task title, source
  chip, duration, session objective, one dominant **Start focus**, quieter
  **Open source**) beside a quieter **capacity** block (aligned tabular
  numerals, scoped totals) and a secondary **commitment notice**.
- A wide **Today timeline** with a true time scale, a now indicator, and
  fixed/flexible/unscheduled/protected distinctions.
- The Observatory scene occupies the lower band, decorative and below the
  work.

**Phone:** a vertical agenda. Order: date heading → next-action surface →
compact capacity line → commitment notice → the day as a scrollable
agenda list. Bottom navigation for the five destinations; primary actions
near the bottom edge with safe-area insets. Not a shrunken desktop
timeline.

**States this surface must show (distinct copy each):** loading, empty
(no tasks — offer capture/routine/connect, never a fake plan), partial
("3h known + 2 unestimated tasks"), over-capacity, nothing-fits (explain
the limiting window/estimate), stale forecast, source-unavailable
(last-known projection + freshness), saving-failed (preserve input), and
a running focus session. Both Day and Night appearances.

## Shell Configuration

Personal Planner uses the react-component-library **navigated console**
archetype (a left rail of destinations + a scrolling main pane), styled
by the Observatory theme. The shell is *configured*, never redrawn:

- `ui/manifest.json`: declare `shell.archetype` = navigated console,
  `shell.asset`, `shell.entry`, `shell.export`.
- `ui/src/layout/AppShell.tsx`: three shell constants — `density`
  (comfortable/editorial, not compact), `mobileNav` (bottom navigation),
  `mainMode` (scrolling).
- `ui/src/layout/navItems.tsx`: the five destinations (Today, Plan, Goals,
  Focus, Review); Settings and profile near the rail bottom.
- `ui/src/layout/BrandMark.tsx`: the "Planner / Observatory" wordmark
  until a permanent brand is chosen through brand-manager.
- The **Auto/Day/Night** control and any quick theme access live in the
  shell's `utility` slot; Settings owns the full appearance model.

If the navigated-console archetype cannot carry the editorial/place-making
composition (e.g. the decorative full-bleed scene band), record a scoped
`shell-ejection` in [`../reference/component-library-gaps.md`](../reference/component-library-gaps.md)
naming the exact `ui/src/` files that own the exception — pre-1.0
availability alone does not justify a forced or ejected shell.

## What The Design Is Accountable For

The design is accountable for these, and machine/manual evidence must
prove them (plan §6.8, §25.4):

- **Truthful time geometry:** block width ∝ duration (a 30/60/90-minute
  trio renders 1:2:3); labels and accessible names read the same times as
  the API (fixture F14).
- **Honest state legibility:** a reviewer can distinguish available time
  from reserve, promise from forecast, planned from actual, source status
  from planner state, and an empty view from a failed query.
- **The scene never competes:** contrast passes in the brightest and
  darkest landscape regions; the art blocks nothing; switching Day↔Night
  changes palette/scene but not card order, block placement, capacity
  values, or scroll position.
- **Accessibility floor:** WCAG 2.2 AA, keyboard-only planning/focus/apply/
  share, reduced motion, 200% zoom, ~44px targets.
- **Calm under load:** a busy week, a long title, many overlapping fixed
  events, a commitment at risk, and a running session all remain readable.

## Information Architecture

Primary navigation: **Today, Plan, Goals, Focus, Review**. Global search /
quick capture is reachable everywhere. **Settings** holds availability,
appearance, focus preferences, integrations, sharing management,
notifications, data, and learning controls.

- **Today** — the daily surface described above.
- **Plan** — one record set projected through **Day, Week, Month, Agenda,
  Timeline, Capacity, and Commitments** perspectives (not seven stores).
- **Goals** — outcomes, milestone status, linked work, and time allocation.
- **Focus** — openable without a task for spontaneous work; the focused
  screen centers objective, timer/mode, next step, source link, next stop
  time, and pause/end.
- **Review** — daily and weekly review with coverage, carry-forward, and
  inspectable insights.

Commitments have a dedicated filter/view inside Plan and contextual access
from Today; promote them to primary navigation only if usage demonstrates
the need. Deep links from a source app open the full planner with item ID,
date, and source filter while capacity stays aware of all authorized busy
time.

## Cross-References

- [`../../DESIGN.md`](../../DESIGN.md) — the binding design contract (Observatory)
- [`../../experience/README.md`](../../experience/README.md) — machine-readable UX specs
- [`DOMAINS.md`](DOMAINS.md) — capabilities behind each surface
- [`FLOWS.md`](FLOWS.md) — the journeys these surfaces host
- [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md) — shell/slot taxonomy and components
- [`../guides/choosing-ui.md`](../guides/choosing-ui.md) — decide/configure/adopt/build
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — why beauty is the wedge
