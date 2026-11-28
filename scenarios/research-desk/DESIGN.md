---
id: research-desk
name: Research Desk
description: Phone-first lens, launcher and courier over the research home, in the Daylight design language built on the Vrooli default kit.
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
  cardRadiusMaximum: "1.75rem"
  defaultMode: "light"
  supportedModes: ["light", "dark", "system"]
  responsiveBaseline: "mobile-first"
  dominantPalette: "light-canvas-electric-blue-lime-and-pastel-tiles"
---

# Research Desk Design

This file is the source of truth for Research Desk's UI decisions. The shared
vocabulary comes from the Vrooli default design kit, generated into
`ui/src/design-tokens.css`; it is not edited by hand. The desk's intentional
expression layer is `ui/src/theme/daylight.css`, which remaps the semantic
tokens the component library already reads and adds the named roles this
product needs. `ui/src/styles.css` owns document and product anatomy only.
Components come from the React Component Library.

## Intent

The operator supervises long-running research from a phone through the tunnel,
in short glances: does anything need me, is everything moving, what changed.
The desk answers those first, and every other screen is one tap from them.

The interface is a **bright day-at-a-glance app**, not a terminal or a report.
It should feel optimistic, soft, confident and almost borderless: a light grey
canvas, white cards, one saturated electric-blue block, a single lime signal,
black pill actions and big numbers with quiet labels. It should not look like
an AI demo, a dark instrument panel, an editorial magazine or a generic
dashboard.

Everything outward (launch, controls, Say, answers, verdicts, the courier
switch) still goes through one confirmation that shows the exact text and
where it goes. Danger and warning roles are kept for those moments and for
failures, so they still mean something.

## Layout

- **Phone.** One column, 20 px page margins. The home is a vertical stack: the
  greeting, the blue needs-you block, live work, the week, since yesterday,
  topics to watch and four quick actions. The bottom bar holds Home, Inbox
  (with its count), Topics and Runs; Launch, Reviews and Desk stay in the menu.
- **Desktop.** The same surfaces in a bento: the blue hero on the left, live
  work in the middle, the open-state counts on the right, then the week,
  since-yesterday and topics-to-watch cards, with quick actions across the
  bottom. The library shell is dressed as a floating white pill console rather
  than a full-height sidebar.
- Nothing scrolls sideways at 390 px or in landscape. Long ids and paths wrap
  anywhere; wide content (tables, code, file text) scrolls inside its own box.
- Facts sit label over value on a phone and in two columns from 48rem.
- A card frames a repeated record or a region with its own heading. There is
  no card inside a card: a region inside a card uses tone and dividers.

## Color

The palette and its roles live in `theme/daylight.css`:

| Role | Token | Use |
|---|---|---|
| Canvas | `--rd-canvas` (`#F3F4F6`) | Page ground |
| Surface | `--rd-surface` (`#FFFFFF`) | Every card and sheet |
| Ink | `--rd-ink` (`#0F1115`) | Primary text and black pill actions |
| Muted | `--rd-muted` (`#5F6774`) | Labels and metadata |
| Blue | `--rd-blue` (`#1B6BFF`) | The one hero block, rings, active day, key links |
| Lime | `--rd-lime` (`#D8F84C`) | A new item or one positive highlight |
| Success | `--rd-green` (`#16A34A`) | Live dot and landed work |
| Warning | `--rd-amber` (`#F59E0B`) | Waiting, over allowance or limited |
| Danger | `--rd-red` (`#EF4444`) | Halted and destructive |
| Lavender | `--rd-lavender` (`#F2E9FF`) | Review and attention tiles |
| Peach | `--rd-peach` (`#FFE9D9`) | After-end and attention tiles |
| Mint | `--rd-mint` (`#E6F7EE`) | Landed tiles and healthy delivery |

Rules:

- Blue is a **block**, not a border: one hero per screen, plus rings and the
  active day. Never a blue outline around every card.
- The primary action is **ink**, not blue. `--color-primary` is remapped to
  ink so the library's primary button is the black pill; `--rd-blue` is for
  brand and data.
- Amber, red and green appear as **dots, rings and words**, never as five
  competing pills on every row. A state is always a word plus colour.
- Lime is used at most once per screen. Pastels are for icon tiles and one
  secondary card, never for status.
- Dark mode is a lower-glare translation of the same roles, not an inverted
  screenshot. Every role keeps a contrast floor of 4.5:1 for text.

## Typography

**Plus Jakarta Sans**, bundled locally (weights 400–800, latin and latin-ext)
with `Manrope` and the platform sans as fallbacks. Body 15 px, metadata 12 px,
micro 11 px. Display and metric numbers are tight (`-0.03em` to `-0.055em`)
and tabular; labels are quiet. There is no monospace voice for the interface;
monospace is reserved for ids, paths and exact text.

- Display / hero metric: 40–56 px, 700.
- Page title: 28 px, 700, `-0.03em`.
- Card title: 15–17 px, 700, `-0.02em`.
- Body: 15 px / 23 px, 400–500.
- Metadata: 11–12 px, 500–600, muted.

## Shape, spacing, shadow

- Card radius 24 px (`--radius-card`), hero and sheet 28–32 px, tile 16 px,
  pills 999 px.
- Card padding 16 px on a phone, 20–24 px on a wide screen; gap 12 px phone,
  20 px desktop.
- One soft shadow for cards: `0 1px 2px rgba(15,23,42,.04), 0 12px 30px
  rgba(15,23,42,.05)` (`--elev-raised`). The blue hero gets a blue-tinted
  shadow (`--rd-shadow-hero`). A border is the exception, not the grammar.
- No card inside a card. List rows use a single 1 px `--rd-line-soft` divider.
- Motion is 140–220 ms ease-out. Card hover lifts 1 px; the live dot may
  breathe slowly; reduced motion stops both. No parallax, gradients blobs or
  floating decoration beyond the hero's faint concentric halo and lime dot.

## Components

Library components first. The desk uses:

- **Card** `1.4.0` — `variant="surface"` for product cards (borderless, soft
  shadow, panel radius), `variant="pastel"` with a semantic `tone` for one
  deliberate tint, `variant="inset"` for a region inside a surface,
  `padding` for raw children and `interactive` for a card that is a control.
- **Tabs** `1.3.8` — `variant="segmented"` for Inbox's Focus/List switch and
  the commission sections. The track is the sunken surface; the selected
  thumb is a raised surface. `variant="underline"` remains for navigation.
- **Button** `2.2` — primary is the ink pill; secondary is the white pill;
  ghost is quiet text. On the blue hero, the white pill is a link styled with
  `--rd-on-hero-pill`, not a second primary button.
- **PageHeader**, **DescriptionList/EvidenceFact**, **StatusBadge**,
  **EmptyState**, **AsyncBoundary**, **Switch**, **Select**, **Input**,
  **BottomSheet**, **Timeline** and **Alert** from the library, dressed by the
  Daylight tokens.
- **Daylight compositions** in `ui/src/components/daylight.tsx`: `Panel` (the
  shared Card in its surface variant), `PanelHead`, `IconTile`, `Ring`,
  `StatPill`, `StatusPill`, `RowLine`, `WeekStrip` and `MiniBars`. These are
  compositions of library parts and tokens, not a second component library.
  `IconTile`, `Ring` and `WeekStrip` are the first candidates to hand back to
  the library when another scenario needs them.

Surface grammar:

- **Hero** — one saturated blue block: label, a big number, the single most
  important question, a white pill action and a quiet count. The only
  decoration is a faint concentric ring and one lime dot.
- **Ring** — a conic donut with the number in the hole, a soft track and a
  blue/amber/red arc by state.
- **Icon tile** — a 32–48 px rounded square in a soft grey or pastel tone,
  with one thin line icon and an optional state dot.
- **Segmented pills** — a grey track with a white or ink active pill.
- **Stat pill** — a big number and a small label in a white card; zero states
  are shown quietly, not hidden.
- **Decision card / focus deck** — a blue card with progress, the question,
  the recommendation and two large buttons; the full list is one tap away.
- **Topic card** — a white card with a pastel icon tile, title, open/runs
  metadata, a seven-day pulse and the evidence for its last write.
- **Rows** — icon tile, title, one meta line, a state word and a trailing
  action; no five-badge rows.

## What We Deliberately Avoid

- Dark terminal, mono type as the main voice, hairline grids, neon on black.
- Editorial serif, warm paper, “flight strips”, “constellations”.
- Five status colours as pills on every row.
- Card-in-card nesting and 1 px borders everywhere.
- Decorative charts with no reading, or charts that cannot be read as a table.
- A blue outline on every card; blue is a block.

## Words

- The operator's words are quoted as written; derived text is plain.
- A value the desk cannot derive reads `unknown`, in muted italics, with a
  “Why?” that shows the evidence. Stale reads are labelled stale.
- Times are relative, with the absolute UTC time on tap.
- The API's sentences (refusals, blockers, verdicts, delivery reasons) are
  shown as sent; the UI never rewrites or matches on them. Nothing the operator
  reads contains command-line text.
- Generated commission titles of the form “Research commission <id> — <what it
  is>” display the part after the dash; the id stays in the eyebrow and the
  full title remains the link's title.

## Responsiveness

Phone first. At phone width every control, path link and in-text link is at
least 44 px tall whatever pointer the browser reports. Bottom chrome respects
the safe area. Nothing important hides behind hover. The desktop bento
collapses to one column below 48rem; the two-column topic and launch grids
collapse to one.

## Preferences

The theme (light, dark, system) is chosen on the Desk page and remembered. Text
size follows the browser's own setting because the tokens are in rem. Reduced
motion stops the one moving element (the current-step dot and card hover).
Locales are wired with text direction, though only English ships.

## Feedback & State

Every operator action shows its state at once: the confirm button shows busy
and cannot be pressed twice, and the result (or the API's refusal, in its own
words) appears beside the action. A failed form keeps what was typed. Lists and
pages have loading, empty, partial and error states rather than blank space; an
optional scenario that is down shows `unknown` for what it feeds and the rest
of the page still works. Toasts are not used; a failure that blocks stays on
the page with a retry.

Loading is a soft grey skeleton with the card's own radii. Empty is a large
pastel icon circle, one sentence and one black pill. Error is a white card with
a red icon tile and the API's sentence in full. Stale is a small
“Updated 3m ago” grey pill; partial is an amber dot and one row.

## UX-State Contract

Every major surface declares the lifecycle states that users can encounter:
`idle`, `pending`, `success`, `loading`, `saving`, `syncing`, `refreshing`,
`empty`, `partial`, `stale`, `validation-error`, `request-error`, `retrying`,
`permission-denied`, and `offline`. A page may add a more specific state, but it
keeps the generic state legible and actionable. The state is represented in
the accessibility tree with a role, name, status or alert that explains what
changed and what the operator can do next.

Experience Manager's state coverage check uses this section as the contract
source. A page that never meets a state may omit it with a documented reason in
its experience spec; a blank or silent state is never a reason.

## Request Lifecycle

Reads refresh every 30 seconds and when the window regains focus (the optional
scenario lens every 60 seconds), and say when what is shown is stale. Writes are
never optimistic: the confirmation shows the dry-run text the API returned, and
the screen changes only after the API has written. A slow step shows a visible
pending state in a stable layout. A failure says what happened, what is still
safe, and offers retry.

## Accessibility

Visible focus on every control, readable contrast in light and dark, status
never by colour alone, and 44 px targets. Live counts and refusals are announced
through `status` and `alert` roles. Hero text is solid white on blue and on the
darker translucent question panel, not a low-opacity white that would fall
below 4.5:1. The white pill on blue is a fixed white surface with ink text in
both themes.

## Component Library Maturity

The redesign resolved these shared-library issues rather than copying around
them:

| Change | Why |
|---|---|
| `Card 1.4.0` | `surface`, `inset` and `pastel` variants, raw-child padding and an interactive lift; the product card no longer needs a local fork to drop its border. |
| `Tabs 1.3.7` | Per-instance tab/panel ids (two strips could collide) and a narrow-rail story that no longer overflows its own frame. |
| `Tabs 1.3.8` | Segmented track and selected thumb use the sunken/raised surfaces, so the control is legible in light and dark without a consumer override. |
| `page-section 0.1.1` | Already forwards `className`; the scenario now imports it. |
| Specifier resolver | A version whose entry filename predates the catalog name and has a `story.tsx` could not resolve; the resolver now honours `component.json.entry` before its legacy fallbacks. |
| Validation | A stale `AsyncOptionsField` type fixture blocked the package build; it now points at the current release. |

Known gaps handed back to the library: `IconTile`, `Ring` and `WeekStrip` are
still scenario-local compositions. The next generic consumer should ingest
them rather than copy them.
