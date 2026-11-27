# Research Desk — redesign 02

**Built from your references.** 26 November 2026.

The first study was the wrong lane. It used the visual vocabulary of AI demos — dark
instrument panels, mono labels, hairlines, editorial serif, a space map — and you were right to
call it slop. The references you sent point somewhere much better: bright, soft, rounded,
confident, almost borderless, with one saturated block of colour and a lot of air.

This study resets the visual language completely. The product model is still the one in
`experience/`; every feature is still there. What changes is how it looks, reads and feels.

**Visual study:** [`index.html`](index.html)

---

## 1. What the references actually say

Three different products, one design language:

| Reference | What it contributes |
|---|---|
| **Insuriva tablet** | A bento dashboard with one electric-blue hero, white cards on a light grey canvas, acid-lime highlights, huge tabular numbers, a pill top-nav, soft shadows and generous padding. |
| **Fashion mobile** | A white / near-white feed, black pill actions, a pastel hero card, a rounded search bar, soft grey icon tiles, a two-column card grid, circular black plus buttons, and very little colour. |
| **Nutrition mobile** | White cards on light grey, a bright blue summary card, rounded bar charts, segmented pill tabs, thin rounded progress bars, coloured macro dots, friendly line icons and big numbers with small labels. |

**The rules they share**

1. **Light canvas, white cards.** The page is a soft grey (`#F3F4F6`); work sits on pure white.
2. **Almost no borders.** Separation comes from tone, radius, spacing and a barely-there shadow.
3. **Large radii.** Cards 24–28 px, hero 28–32 px, tiles 16–20 px, pills 999 px.
4. **One colour block per screen.** A large electric-blue card, or a pastel hero — not a rainbow of badges.
5. **Black pill actions.** Primary actions are black (or white on blue), short, and confident.
6. **Soft grey tiles for icons.** 36–48 px rounded squares with one thin line icon; colour is reserved for state and the hero.
7. **Big numbers, small labels.** 40–56 px numbers with 11–13 px labels; no dense mono captions.
8. **Air.** 16–24 px card padding, 12–16 px gaps, 20 px phone margins.
9. **One accent beyond the hero.** Acid lime for a highlight, one pastel for a secondary block; not five status colours competing.
10. **Friendly, simple iconography.** Thin rounded strokes, circular icon buttons, no technical ornament.

**What this means for Research Desk**

- The desk becomes a **bright day-at-a-glance app**, not a terminal or a report.
- The blue block is the **needs-you hero**, because that is the one thing the operator must act on.
- The recurring data shape is a **ring or a rounded bar**, not a sparkline lane or a flight strip.
- Topic identity comes from **soft icon tiles**, not monogram chips or mono IDs.
- The operator's 19 decisions become a **focus deck**, not a 19-row scroll.
- IDs, paths and evidence stay available, but they are **secondary**, not the visual language.

---

## 2. The visual system

### Colour

| Role | Token | Use |
|---|---|---|
| Canvas | `#F3F4F6` | Page background |
| Surface | `#FFFFFF` | Every card and sheet |
| Ink | `#0F1115` | Primary text, black pills |
| Muted | `#6B7280` | Labels, metadata |
| Primary | `#1B6BFF` | The hero card, active tab, key data |
| Primary soft | `#E9F0FF` | Selected tiles, mini bars |
| Signal | `#D8F84C` | A new item, a positive highlight |
| Success | `#16A34A` | Live dot, landed work |
| Warning | `#F59E0B` | Limited, waiting, over allowance |
| Danger | `#EF4444` | Halted, destructive |
| Lavender | `#F2E9FF` | Review / archive tiles |
| Peach | `#FFE9D9` | After-end / attention tiles |
| Mint | `#E6F7EE` | Landed tiles |

Rules: blue is a **block**, not a border. Amber/red/green appear as **dots, rings and words**. Lime is used at most once per screen. Pastels are for **icon tiles and one secondary card**, never for status.

### Type

- **Family:** a geometric humanist sans. Recommend **Plus Jakarta Sans** (or **Manrope** / **DM Sans**) for both display and UI; tabular numerals for every number.
- **Display:** 40–56 px, 700, `-0.04em` tracking.
- **H1:** 28 px, 650, `-0.03em`.
- **H2:** 20 px, 650.
- **Body:** 14–16 px, 400–500, 1.45.
- **Metadata:** 11–12 px, 500, muted.
- **Micro:** 10 px, 600, sentence case or short uppercase only for a single eyebrow.
- **Numbers:** tabular, 40–56 px, tightly tracked; the figure is the hero, the label is quiet.

### Shape, spacing, shadow

- Card radius 24; hero 28; phone sheet 32; tile 16–18; input and buttons 999.
- Padding: 20 px phone page, 16–20 px cards, 24–32 px desktop cards.
- Gap: 12 px phone, 20–24 px desktop.
- Shadow: `0 1px 2px rgba(15,23,42,.04), 0 12px 30px rgba(15,23,42,.05)` — visible only as depth, never as a border.
- No card inside a card. No dividers except a single 1 px `#F1F2F4` between list rows.

### Components

- **Hero card** — a saturated blue or pastel block with one big number, one sentence, one white/black pill.
- **Ring meter** — a 40–120 px donut; the number sits in the hole; the track is `#EDF0F4`; the arc is blue, amber, grey or red by state.
- **Icon tile** — a 36–48 px soft-grey or pastel rounded square with one line icon; a state dot in the corner.
- **Segmented pills** — a grey track with a white or black active pill.
- **Stat pill / stat card** — a big number with a small label, always in a white card.
- **Decision card** — a white card with an icon tile, a question, a recommendation strip, and two large buttons.
- **Focus card** — a blue decision card with `3 of 19`, one question, a recommendation and a white `Use this` pill.
- **Topic card** — a white card with a pastel icon tile, title, meta and a 7-day mini bar row.
- **Week strip** — seven day circles with activity dots and a blue active day.
- **Action sheet / bottom bar** — a white sheet with a black primary pill and a grey secondary; one row of actions.

### Motion

- 180–240 ms ease-out; card lift 1 px on press; active pill slides; ring fills on load; numbers count up once.
- One continuous motion only: the live dot breathes slowly. Reduced motion stops it.
- No parallax, no gradient blobs, no floating decoration.

### What we deliberately avoid

- Dark terminal, mono type as the main voice, hairline grids, neon on black.
- Editorial serif, warm paper, "flight strips", "constellations".
- Five status colours as pills on every row.
- Card-in-card nesting and 1 px borders everywhere.
- Decorative charts with no reading, or charts that cannot be read as a table.

---

## 3. Three structural models, one visual language

The references fix the look. The remaining product question is structure: how should the desk
organise the day? Three answers, all in the same system.

### A · Daylight — recommended

> A bright day-at-a-glance app. One blue hero for the decisions, white cards for the work, a
> ring for every allowance, a week strip for the pulse. The console you would actually enjoy
> opening on a phone.

**Home (phone).** Header with date, `Good morning.`, `2 running · 19 waiting`, and a round bell.
Then a **blue hero**: `19` waiting, `+3 new`, the single most important question, a white
`Answer` pill and `5 older lists`. Then **Live now** (two rows with icon tiles, state dots and
allowance rings), **This week** (a seven-day strip, rounded activity bars and two digest rows),
**Since yesterday** (three pastel-tiled sentences with one black `Catch up`), and four soft
**quick-action tiles** (Launch, Topics, Runs, Reviews). Bottom nav: Home, Inbox (19), Topics, Runs.

**Home (desktop).** A top pill-nav, then a bento: the blue hero on the left (4 columns), Live
now in the middle (5), three white stat pills on the right (3), This week below the middle,
Since on the right, and a Topics-to-watch card under the stats. It is the Insuriva composition,
translated to research.

**Inbox.** A **focus deck**: one decision at a time in a blue card — `3 of 19`, the question, the
recommendation, `Use this` / `In my words`, a thin progress bar, `Skip`. Below it, `Up next` with
three white cards, an `Older lists` row, a courier row and `Answered today` with read
receipts. A `Focus / List` segmented control switches to the full list. This is how 19 decisions stop feeling like a wall.

**Commission.** A white hero with a large **allowance ring** (`13h left`), three small stats
(`Last write 9h`, `187 changes`, `24h window`), a state pill, and four segmented tabs:
Overview / Steer / Reading / Files. Overview is a clean stint timeline and a `Changes` card.
Steer is a message composer with a black send pill, then big pill controls (Pause, Wind down)
and a `More` sheet for Stop/Discard. Reading is a read-set ring with a segmented budget bar.
Files is a two-column grid of white file cards with pastel icon tiles.

**Launch.** A four-step flow with a segmented progress pill: **Scope → Words → Window → Check**.
Scope is a two-column grid of selectable topic cards (icon tile, last write, state dot, check
circle), with a sticky white summary bar (`3 topics · 24h · luna`) and a black
`Review & launch`. Words uses quote cards on lavender. Window uses big duration pills. Check is
three large status cards (clear / resolve / blocked) and a final blue `Ready to launch` card.
The confirmation is a white sheet with the exact text in a soft-grey block and one black button.

**Topics.** A search bar, three filter pills (`All`, `Needs attention`, `Quiet`), a wide lavender
`Needs attention` card, then a two-column grid of topic cards with pastel icon tiles, a title,
`3 open · 4 runs`, a seven-day mini bar row and a state dot. This is the fashion reference,
applied to research.

**Reviews.** A two-column grid of pastel review cards; the review itself is a clean reading page
with a sticky black `Record verdict` pill and verdict quote cards.

**Desk.** Settings rows in white cards: icon tile, label, status dot, chevron. Theme is a
segmented pill (Light / Dark / System). A green `Everything connected` card sits at the top;
sources become a clean row list with `Start` text buttons; self-tests are black pills.

**States.** Loading is soft grey skeleton blocks with the same radii. Empty is a large pastel
icon circle, one sentence and one black pill. Error is a white card with a red icon tile and the
API's sentence in full. Stale is a small `Updated 3m ago` grey pill; partial is an amber dot and
`Some sources are offline` row. None of these are full-page dramas.

**Why it wins.** It is the closest to the references, it answers the five-second question with a
single blue block, it makes the decision queue feel finite, and it gives the desk a distinct,
optimistic identity without a themed metaphor.

### B · Index — the quiet alternate

> A white, list-first research feed. Almost monochrome; one pastel hero, black pills, soft grey
> tiles, and a topic grid that looks like a well-made shopping app.

**Home.** A white / grey feed: a lavender hero `19 decisions waiting` with a black `Start` pill;
a row of four category tiles; a horizontal `Live now` row of two cards; a `Since last look`
list; a `Topics` two-column preview. No blue block, no bento. More air, fewer numbers.

**Topics.** The full two-column grid is the centrepiece. A `Needs attention` lavender card at
the top; then topic cards with pastel icon tiles, a title, a last-write line and a mini bar row.
This is the fashion reference at its purest.

**Commission.** A clean detail page: big title, one row of scrollable stat pills, a segmented tab
bar, a white timeline, and a sticky bottom action bar (black `Pause`, white `Message`).

**Launch.** A checkout: four black/white segmented steps, large selectable cards, a white order
summary with the packet details, and a black `Launch` bar.

**When to choose it.** If you want the quietest, most premium, most "consumer app" version. It is
beautiful for reading and browsing; it is slower than Daylight for a quick operational scan, and
it hides the live state in a horizontal row.

### C · Week — the optional time board

> The desk as a week. A seven-day strip, coloured commission blocks, a now line, and a decision
> strip on top. Research is time; this makes time visible.

**Home.** A day strip (M–S with dates, today blue), a blue `19 decisions` strip, then a week
agenda: commission blocks as rounded pastel cards spanning days, each with a state dot, a
duration bar and topic chips. On phone, the same data becomes a vertical agenda grouped by day.

**Commission.** The detail page gains a **time ruler**: the stint as a horizontal band with
launch, units, writes, controls and now; the allowance as a ring. Launch becomes `Schedule a
window`: a duration picker and a calendar preview.

**When to use it.** As a `Week` view inside Daylight or Index, not as the default. It is the best
view for overlaps, windows and "what is running when"; it is the worst for answering 19
decisions quickly.

---

## 4. Comparison

| Criterion | A · Daylight | B · Index | C · Week |
|---|---|---|---|
| Five-second "does anything need me?" | **5/5** — blue hero | 4/5 — lavender hero | 3/5 — decisions are a strip |
| Five-second "what is moving?" | **5/5** — live rings + week strip | 4/5 — live row | **5/5** — blocks on time |
| Speed to answer 19 decisions | **5/5** — focus deck | 4/5 — list | 3/5 — decisions are secondary |
| Reading long research | 4/5 | **5/5** | 3/5 |
| Phone | **5/5** | **5/5** | 3/5 |
| Desktop | **5/5** — bento | 4/5 — feed | **5/5** — board |
| Closest to your references | **5/5** | **5/5** | 4/5 |
| Distinctiveness | 4/5 | 4/5 | **5/5** |
| Build cost | M | M | L |
| Risk | Low | Low | Medium |

**Recommendation:** build **A · Daylight**, take **B's topic grid** and white-card discipline
into it, and add **C's week board** as an optional `Week` view after the core is proved.

---

## 5. Feature parity

The 34 feature groups from `UX-REDESIGN-20261126/PROPOSAL.md §9` remain the baseline. The new
homes:

| Surface | Everything preserved |
|---|---|
| Home | status counts, newest write, needs-you count and top questions, live/paused/limited groups, since-last-look + caught up, lenses down, unreadable files, overlaps, prepared packets, state counts |
| Inbox | the full needs-you feed: body, options, own words, exact entry, sent/not sent/answered/read, reused ID, global items, secret items, older lists, courier health, self-test, global switch; focus and list modes |
| Topics | filter, newest-write sort, mode, open workstreams, runs, read size/bloat, written-while-paused, source on tap; topic dossier: write, mission edit, last review, workstreams, runs, changes |
| Runs | grouped states, filters, prepared/unknown packets; commission: facts + evidence, changes + mark seen, sessions, launch timeline, steer (say/wind down/entry limit/destination/read timing), operator log + read marks + unparsed headings, pause/resume/stop/discard/re-send stance, enable steering, refused controls + manual keys, courier switch, read-set gauge, files |
| Launch | prepared/unknown, topics max, workstream order, notes, emphasis, quotes + source, allowance, models + cost hint, slug, form persistence, preflight, blockers + recommendation, quiet folded, read-first, existing packet STATE/read set, confirmation, progress, failed start discard/relaunch, manual messages |
| Reviews / Review | list, REVIEW.md/HTML/media, verdict per topic, unparsed headings |
| Desk | capabilities, sources, start, Web Console access, courier state, auto-launch, self-tests, theme |
| Cross-cutting | 30 s / 60 s reads, focus refresh, stale, non-optimistic writes, dry run, pending, retry, 44 px targets, status words + shape, unknown, reduced motion, locale direction, no command-line text |

Nothing is dropped. Each implementation phase diffs the served inventory against the 34-group
baseline; a lost feature becomes a keep/drop card.

---

## 6. Build

**Phase 0 — Baseline (1–2 days).** EX9 capture of the current 12 routes at 390/1440, light/dark;
EX8 inventory; 5-second test; time-to-answer and time-to-launch baselines.

**Phase 1 — System (1 week).** Tokens: canvas, ink, blue, lime, pastels, radii, type, shadow.
RCL assets with contracts: `HeroCard`, `RingMeter`, `IconTile`, `SegmentedPills`, `DecisionCard`,
`FocusDeck`, `TopicCard`, `WeekStrip`, `StatPill`, `ActionSheet`, `SourceRow`, `FileCard`.
Stories at 320/390/1440, light/dark, long German labels, reduced motion.

**Phase 2 — Home + Inbox (1.5 weeks).** The blue hero, live rings, week strip, since card, quick
actions; focus deck and list; courier sheet. The 5-second test must pass here before anything
else ships.

**Phase 3 — Runs (1.5 weeks).** Commission hero + ring, segmented tabs, stint timeline, changes,
message composer, controls, operator log, read-set ring, files.

**Phase 4 — Launch (1.5 weeks).** Four-step flow, topic grid, quote cards, duration pills,
clearance cards, ready card, confirmation sheet, progress. Manual/unknown paths.

**Phase 5 — Library and rooms (1.5 weeks).** Topics grid + dossier, reviews, Desk, File. Optional
Week board.

**Phase 6 — Proof (1 week).** EX1 critic at 390/1440 light/dark; `--phases experience`; served
journeys at 320–1440; parity diff; two operator reviews through Notification Hub.

The same experience-engineering capabilities apply: EX9 baseline, EX8 inventory, EX11
compositing, EX12 region feedback, EX1 critic, EX7 contracts, EX3 studio, EX2/EX6 gate. The
new components are built in RCL, not copied locally.

---

## 7. Decisions

1. **Structure.** A · Daylight (recommended), B · Index, or C · Week as the default? C can be an
   optional view under either.
2. **Accent.** Electric blue + lime (recommended, closest to the references), or monochrome +
   lavender (Index), or a warmer peach/orange.
3. **Inbox default.** Focus deck or list? The deck is faster for 19; the list is faster for
   scanning and searching.
4. **Brand latitude.** May the desk use the blue/lime palette and a simple ring mark? Staying
   inside the current default kit will lose most of the reference's character.

No answer is a real answer: the current desk keeps working and no commission is touched.

---

## 8. Provenance

- References: the three images the operator supplied on 2026-11-26.
- Current state: `../UX-REDESIGN-20261126/assets/captured-*.png` (`captured`).
- Feature baseline: `../UX-REDESIGN-20261126/PROPOSAL.md §9` (34 groups).
- Product contract: `scenarios/research-desk/experience/`, `docs/concepts/EXPERIENCE.md`,
  `docs/concepts/SYSTEM.md`, `DESIGN.md`.
- The phone and desktop screens in `index.html` are `sketch`: HTML/CSS, not RCL components. The
  build target is `rendered` in RCL; the proof is `composited` through EX11.
