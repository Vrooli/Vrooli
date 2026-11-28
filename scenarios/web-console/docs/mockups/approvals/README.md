# Approvals inbox — mockups (L1, mockup-first HUMAN GATE)

These are **static mockups only** (self-contained HTML + the real `--wc-*` palette).
No product code, no `.tsx`/`.ts`, no RCL release edits. The operator ratifies the
**concept** (already ratified) and **these mockups** before any inbox code is written
(`experiential-ui-design` §2, L1 of the experiential ladder).

## Concept

One shared surface for every human approval in the fleet, regardless of which
scenario raised it. The producers all ride the same domain-neutral substrate
(`packages/api-core/operatordecision`): `agent-manager` (run review / authorization
scope / effort decision), `deployment-manager` (release gate), `swarm-manager`
(mutation proposal), `vrooli-autoheal` (remediation). Because every decision is the
same shape on the wire, the inbox models the **list + detail once and composes it
twice** (desktop master-detail, mobile card-stack → full-screen sheet).

The unifying idea: **"the one place that asks you, and never anywhere else."**
The surface is a *triage queue* — it answers "what needs me, how urgent, what does
the system recommend, and can I undo it" at a glance, then lets the operator answer
in one thumb-reach (mobile) or one click (desktop). It is deliberately NOT a re-skin
of agent-manager's basic approvals page; that page is a reference for *what data and
actions exist*, never for styling.

## Surface archetype

- **Desktop: List / feed → master-detail.** Filterable list left, decision detail
  right. (`experiential-ui-design` §3 List row + a detail pane.)
- **Mobile: List / feed → card-stack that drills into a full-screen / bottom-sheet
  detail.** Not a reflow of the desktop two-pane; a distinct tree (`useMediaQuery`
  already exists in the app, see `ArchiveDrawer.tsx:3`). The inbox is the *closest
  analog to `ArchiveDrawer`* (list + filters + search + selected detail + RCL
  `SwipeActions`), so the mockups copy that interaction grammar.

## Content model — modeled ONCE, composed twice

This is lifted verbatim from the canonical substrate so the inbox renders exactly
what producers emit — no second vocabulary.

### List item (one pending/resolved decision)

Source of truth: `operatordecision.PendingAsk` + the `decision_requested` `Facts`
payload (`packages/api-core/operatordecision/facts.go`, `pending.go`).

| Field | From | Shown as |
|---|---|---|
| `source` | event type `<source>.<kind>.decision_requested.v1` | source chip + icon (agent-manager / deployment-manager / swarm-manager / vrooli-autoheal) |
| `kind` | event type | sub-label under source (run-review, release, mutation, remediation) |
| `subject` (`Facts.Subject`) | facts | list row title |
| `question` (`Facts.Question`) | facts | one-line clamp in list, full in detail |
| `urgency` (`""`/low/normal/high/critical) | facts | left urgency rail + ordering |
| `reversible` (bool) | facts | "Reversible" / "Not reversible" pill — gates whether a default can auto-apply |
| `deadline` | facts | relative "in 2h" / "expires soon" countdown; drives `expired` styling |
| `state` (pending/approved/rejected) | `PendingAsk.State` | history filter + resolved/denied styling |
| `raisedAt` / `resolvedAt` / `resolvedBy` | `PendingAsk` | relative time; "you approved · 4m ago" |

### Decision detail

| Region | From |
|---|---|
| Header: source chip, subject, urgency, reversible pill, deadline countdown | list fields |
| Question (full `Facts.Question`) | facts |
| **Options** (`Facts.Options[]`, >=2) rendered as answer buttons; the `Facts.Recommended` one is the **primary** action and carries `Facts.RecommendationReason` | facts |
| Optional **note field** (free text the operator adds when answering) — carried back as the resolution note | ui-local |
| `default_answer` (shown only when `reversible`) as "if you do nothing by the deadline: <label>" | facts |
| **Context panel** (pluggable, per `source`) | `context_url` + source-specific facts (see below) |
| Open-full-context link (`Facts.ContextURL`) | facts |

### Per-source context panel contract (pluggable)

The detail reserves ONE slot that renders a per-`source` panel. The panel is chosen
by `source`; its body is driven by whatever extra facts the producer attached. Two
distinct panels are shown in the mockups so the pluggable idea is visible:

1. **deployment-manager / release** — artifact digest, target environments,
   evidence disposition (which gates passed / SKIPPED), rollback note.
2. **agent-manager / run-review** — diff summary (files/±lines), requested
   authorization scope, and the run/effort link.

(swarm-manager mutation and vrooli-autoheal remediation reuse the same slot; their
panels are sketched in the list mix but the two above are the fully-realized
examples the gate asks for.)

## Per-medium entry model (operator-settled — do NOT change without re-ratification)

- **Mobile top bar is already crowded** (unread badge, title, +, devices, settings,
  account). So:
  - **Primary entrance = a pending-approvals banner in the existing `BannerRegion`**,
    shown ONLY when `pendingCount > 0` ("N approvals need you" → opens the inbox).
    Thumb-reachable, lives in the one top-chrome owner the app already has
    (`components/banners/BannerRegion.tsx`), tone `warning`/`danger` by max urgency.
  - **Persistent access when empty = a badged entry inside an existing menu**
    (account / overflow), NOT a new always-on top-bar button.
  - **Mobile detail = full-screen / bottom-sheet** with the answer action in the
    thumb zone; approve/reject on list cards via RCL `SwipeActions` (the same
    primitive `ArchiveDrawer.tsx` uses).
- **Desktop: persistent affordance via the `FloatingToolbar`** (desktop-only, has
  room) — a badged inbox button alongside Settings/AI/Machines. Opens the
  master-detail inbox (as a `FullPageDrawer`, matching `ArchiveDrawer`).

## Deep link (no React Router)

A push/chat link lands directly on a specific decision's detail. The app has no
router; the intent is parsed at boot into a store slice (same pattern as
`ArchiveDrawer`'s `initialSessionId`). Mockup `mobile-e-deeplink.html` and the
desktop detail show this "landed on one decision" state.

## State set covered

empty · loading (skeletons) · error · pending · resolved/history filter ·
expired/denied (`--wc-error-*`).

## Palette / tokens used (and where they were found)

All values are the **operational `--wc-*` palette**, `R G B` triples from
`scenarios/web-console/ui/src/styles.css` (lines noted), consumed as
`rgb(R G B / a)`. The brand hexes mirror them in
`scenarios/web-console/ui/src/design-tokens.css`.

| Token | Value (sRGB) | styles.css |
|---|---|---|
| `--wc-surface-base` | `2 6 23` (slate-950) | :75 |
| `--wc-surface-raised` | `15 23 42` (slate-900) | :76 |
| `--wc-surface-input` | `30 41 59` (slate-800) | :77 |
| `--color-surface-raised` | `#172033` (card) | design-tokens.css |
| `--wc-text-primary` | `248 250 252` | :80 |
| `--wc-text-secondary` | `226 232 240` | :81 |
| `--wc-text-muted` | `148 163 184` | :82 |
| `--wc-text-faint` | `100 116 139` | :83 |
| `--wc-border-default` | `255 255 255 / 0.1` | :86 |
| `--wc-border-hover` | `255 255 255 / 0.3` | :87 |
| `--wc-accent` | `34 211 238` (cyan-400) | :90 |
| `--wc-accent-fg` | `15 23 42` | :91 |
| `--wc-accent-border` | `6 182 212 / 0.4` | :92 |
| `--wc-error-text` | `252 165 165` | :96 |
| `--wc-error-detail` | `248 113 113` | :97 |
| `--wc-error-border` | `239 68 68 / 0.3` | :98 |
| `--wc-error-surface` | `69 10 10 / 0.5` | :99 |
| `--wc-success` | `74 222 128` | :108 |
| `--wc-warning` | `245 178 63` | :109 |
| radii | `--radius-panel .5rem`, `--radius-control .375rem`, `--radius-pill 9999px` | design-tokens.css |
| font | `system-ui, -apple-system, …` (`--font-sans`) | design-tokens.css |
| tap target | `--tap-target-min 44px` | design-tokens.css |

Per-source accents reuse existing semantic tokens (no new colors invented):
deployment = `--wc-warning` (release gate = at-risk), agent-manager = `--wc-accent`,
swarm = `--provenance-sample #b7a6ff` (already in design-tokens.css), autoheal =
`--wc-success` for a healthy remediation / `--wc-error-detail` when it is escalated.

## How to view

Open each file directly in a browser:

- Desktop master-detail (pending, mixed sources): `file://.../desktop-a-master-detail.html`
- Desktop empty state: `file://.../desktop-b-empty.html`
- Desktop loading skeletons: `file://.../desktop-c-loading.html`
- Desktop FloatingToolbar entry + error state: `file://.../desktop-d-entry-error.html`
- Mobile banner entrance (in BannerRegion): `file://.../mobile-a-banner.html`
- Mobile card-stack list + swipe actions: `file://.../mobile-b-list-swipe.html`
- Mobile full-screen detail (thumb-zone action): `file://.../mobile-c-detail.html`
- Mobile badged menu entry (empty / persistent): `file://.../mobile-d-menu-empty.html`
- Mobile deep-link-landed-on-detail: `file://.../mobile-e-deeplink.html`

(Absolute paths are listed in the report to the orchestrator.)

## Open design questions for the operator (decide at this checkpoint)

1. **Mobile answer: swipe vs button.** Mockups show RCL `SwipeActions` for
   approve/reject on list cards (matching `ArchiveDrawer`) AND a thumb-zone button
   in the detail. Is swipe-to-approve acceptable for an *irreversible* decision, or
   should irreversible ones require the detail + an explicit confirm (swipe disabled
   on `reversible:false` cards)? Mockup assumes **swipe only on reversible cards;
   irreversible must open detail**.
2. **Desktop entry location.** Mockups put the badged inbox button in the
   `FloatingToolbar`. Should it ALSO appear as a fixed item (not draggable) so it
   never docks off-screen, or is the toolbar's dock behavior acceptable for an
   approvals affordance?
3. **New RCL primitive?** The decision detail's **answer-options block** (recommended
   primary + secondary options + inline note + reversible/default hint) is reused on
   desktop and mobile and across all sources. Should this become a shared RCL
   `DecisionCard`/`AnswerOptions` primitive (drafted via the RCL draft flow), or stay
   a web-console-local component composed from existing RCL `Button`/`Banner`? The
   mockups are built from existing primitives; flagging the candidate, not building it.
4. **Default-answer visibility.** Where should the `default_answer` + deadline
   ("if you do nothing, we will <X> at <time>") sit — in the detail only, or also as
   a line on the list card for reversible+deadlined decisions? Mockup shows it in
   detail + a subtle countdown chip on the card.
5. **Resolved/history retention.** How far back does the history filter show resolved
   decisions, and does it page? Mockup shows a flat "Resolved" filter with relative
   times; no paging decision made.
