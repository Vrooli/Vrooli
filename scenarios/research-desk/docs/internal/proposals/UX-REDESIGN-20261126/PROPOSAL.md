# Research Desk — full UX redesign study

**Date:** 2026-11-26
**Author:** design study for the operator
**Scope:** the whole `research-desk` UI (12 routes, 6 journeys), same features, new experience
**Method:** current UI captured from the running scenario; feature inventory read from
`experience/` and `docs/concepts/EXPERIENCE.md`; proposal built against the capabilities in
`research/experience-engineering` (EX1, EX3, EX7, EX8, EX9, EX11, EX12, EX2/EX6)
**Visual study:** [`index.html`](index.html) (open beside this document)

---

## 1. The one-sentence problem

The desk has excellent *data* and an honest *contract*, but its UI reads like a generated admin
report: every fact has the same visual weight, every collection is a bordered card list, and the
five-second question — *does anything need me, is anything moving, what changed* — is answered
by reading three sections and counting badges.

The redesign must make the system **legible before it is readable**: the shape of the screen
answers the question, then the words confirm it.

What is already right and must be kept:

- files are the state, and `unknown` is a real state;
- every write is confirmed with the exact text and destination;
- operator words are quoted verbatim;
- status is never colour alone;
- the desk never enters the supervisor loop;
- every feature in `experience/` works.

This is a **presentation and interaction redesign**, not a new product. No new authoritative
state, no database, no new supervisor burden.

---

## 2. Diagnosis — what makes the current UI feel basic

Captured 2026-11-26 at 390×844 and 1440×1200 from the running scenario. Images in
[`assets/`](assets/) are `captured` (fact), not mockups.

| # | Defect | Where it shows | Why it matters |
|---|---|---|---|
| 1 | **No system shape.** Now is a vertical stack of five equal sections (status, needs-you, supervisors, since, overlaps, packets, counts). | `/` | The operator must read to learn whether anything is wrong. There is no "instrument" that shows the state of the system. |
| 2 | **State is a badge, not a material.** `live`, `limited`, `prepared` are all the same pill with a dot and a hue. | everywhere | Colour and a dot are doing the work that shape, position, motion and words should do. At a glance, limited and live look alike. |
| 3 | **The commission row is a paragraph, not a strip.** Three lines of title (`Research commission … — Topic — Topic`), then three badges, then a topic chip, then three lines of facts. | Now, Commissions | The one record the operator acts on most is the hardest to scan. IDs and titles repeat what the route already says. |
| 4 | **"What changed" is a list of nouns, not a briefing.** Eight topics, each with `N changes in the last 24 hours (no look recorded)`. | Now | It reports *volume* but not *meaning*. The operator still has to open each topic to learn what happened. |
| 5 | **Decisions have no triage.** 19 items and 5 older lists, all the same size, each with a `Sending` badge, title, topic and a one-line recommendation. | Needs you | A decision feed must rank by consequence and age, and make the recommended answer one tap. Delivery state is louder than the decision. |
| 6 | **Launch is a long form, not a flight plan.** All five sections expanded at once; 51 topics in one list; the "map" at the top is four anchor links. | Launch | The most consequential action is presented as paperwork. The operator cannot see the packet they are about to send. |
| 7 | **Desktop is a phone column.** `max-inline-size: 60rem`, one column at 1440. | all pages | Wide screens show empty margins while lists that could be side-by-side stay stacked. |
| 8 | **Trust is hidden behind a tiny `Why?` link.** | Topics, Commission | The desk's best idea — evidence for every derived value — is the least visible thing on the page. |
| 9 | **Type hierarchy is flat.** Row titles are large and bold; labels are small; headings are compact. | all pages | Large repeated titles create noise. Data, labels, prose and controls need distinct voices. |
| 10 | **No motion language.** The only moving element is the launch progress dot. | Launch progress | A live research system should *feel* alive without decoration: a pulse line, a freshness arc, numbers that roll. |
| 11 | **Density is inverted.** Rows are tall and sparse, yet the page is long and the important things are still below the fold. | all pages | An operations desk needs high signal density, not tall cards. |
| 12 | **Empty, partial, stale and offline states are sentences.** | all pages | These are designed states in the contract but read as fallbacks. They should be first-class compositions. |

The visual study's "Today, captured" section numbers these against the real screenshot.

---

## 3. Design thesis

**A calm instrument, not a dashboard.**

The desk is a place the operator visits for 20 seconds between other work. It should behave like
a well-made instrument panel: one glance gives the reading, one touch gives the detail, and the
instrument never pretends to know something it does not.

Five rules govern every screen:

1. **Answer before detail.** Each surface opens with its answer in a sentence and a shape; the
   records that prove it come after.
2. **Time is the primary axis.** Research is movement. Every commission gets a lane, every topic
   gets a sparkline, every change gets an age. The desk is a record of time, not a table of nouns.
3. **State has a material.** Live is a moving line. Paused is a held line. Limited is a line that
   hits a ceiling. Halted is a broken line. Ended is a faded line. Unknown is a dotted line.
   Colour reinforces; shape and words carry.
4. **Evidence is a first-class surface, not a link.** Every derived number has an inline
   provenance stamp; tapping opens a sheet with source path, rule, time and the raw value.
5. **Every outward action is a dispatch.** The confirmation is a preview of the exact message,
   its destination and when it will be read — not a generic modal.

### The ownable signature: **the pulse line**

A single continuous line is the identity. It begins in the mark (three ticks and a notch), runs
through the duty header, becomes the 24-hour activity ribbon on Now, and becomes the stint
timeline on a commission. It is still when nothing is live; it advances only when the desk has
evidence of work. The line is the product's promise: *one system, honestly observed*.

### Voice

Plain, factual, short. The UI speaks in three registers, visibly different:

- **Instrument** (labels, numbers): terse, mono, tabular.
- **Briefing** (what changed): complete sentences, no jargon, grouped by meaning.
- **Operator** (quotes, answers, verdicts): verbatim, set apart, never paraphrased.

---

## 4. Three complete directions

Each direction covers all 12 routes and preserves every feature. They differ in what the desk
*is*: an instrument, a publication, or a map.

### Direction A — **The Bridge** (recommended)

> A ship's bridge for a fleet of research commissions. Day Watch is warm paper and ink; Night
> Watch is graphite and signal amber. Everything is a reading, a strip or a dispatch.

**Feeling:** precise, quiet, expensive. Braun instrument × air-traffic flight strips × a
well-set data table. Not sci-fi; no glowing gradients.

**Signature elements**

| Element | What it is | Replaces |
|---|---|---|
| **Pulse ribbon** | 24 h / 7 d activity lanes, one per open commission, with ticks for research writes, a live sweep, a now line, allowance caps and quiet gaps. | The status sentence and the separate supervisor groups. |
| **Flight strip** | A commission as a horizontal ticket: state material on the left edge, human title, mono id, route chips, allowance meter, last write, change count. | The tall commission card. |
| **Decision ticket** | A needs-you item as a boarding-pass-like slip: consequence line, age, recommendation, two large answers, courier stamp. | The collapsed `<details>` row. |
| **Read-set gauge** | Segmented budget bar (charter / state / ledger / mission / operator) against ceilings, with growth since last unit. | The `22 KB (about 5,387 tokens)` sentence. |
| **Evidence sheet** | Bottom sheet with the source path, rule, observed time, raw value and "open file". | The `Why?` link and inline evidence list. |

**Navigation**

- Phone: four tabs — **Now**, **Needs you** (count), **Topics**, **Commissions** — plus a
  persistent **Launch** action in the top bar and a **⌘K** search sheet. Reviews and Desk live in
  the overflow, as today. The tab bar is 56 px, labels always whole, safe-area aware.
- Desktop: a 224 px left rail with a live duty summary (needs-you count, live count, freshness),
  then the same destinations, Reviews and Desk, and a bottom primary Launch. Content is a
  12-column grid, max 1440 px, with list/detail two-pane views.
- Command palette: jump to a topic, commission or file; "pause …", "wind down …", "mark caught
  up", "launch". Every command shows its permission and confirmation consequence. This is how
  the desk becomes powerful without adding tabs.

**Now — the Watch**

```
┌ duty header ──────────────────────────────────────────────┐
│ Research Desk        ● 2 live   ! 19 need you    fresh 12s │
│ One line: "Two windows working. One is over its allowance.   │
│ No halt. The newest write was 3h ago in job-application."   │
├ pulse ribbon ─────────────────────────────────────────────┤
│ 6h   [ 12h ]  24h   7d                        now ─┐        │
│ SCRYR        ▁▂▅▇▅▂▁▂▅▇▇▅▂▁▂▅▇▅▂▁▂▅▇  live 13h left        │
│ managed-br.  ▁▁▂▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁  limited · ended 45m  │
│ pr3-failure  ▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁  prepared             │
│ job-applic.  ▁▂▃▂▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁  quiet 3h             │
│ + 49 more                                            →      │
├ needs you ─────────────────────────────────────────────────┤
│ 19 need you · 5 older lists      Courier: delivering        │
│ ┌───────────────────────────────────────────────────────┐  │
│ │▌ Authorize one corrected free-sandbox upload?          │  │
│ │  crypto-infrastructure · asked 6h ago · delivered       │  │
│ │  ✓ Recommended: Permit exactly one fresh POST           │  │
│ │                                        [ Answer ]       │  │
│ ├───────────────────────────────────────────────────────┤  │
│ │▌ Which route should M1 prepare for the synthetic …?     │  │
│ │  medical-practice-it-modernization · asked 9h ago       │  │
│ └───────────────────────────────────────────────────────┘  │
├ since last look ──────────────────────────────────────────┤
│ Since 08:14 yesterday                        [ Mark seen ] │
│ 3 topics wrote after their commission ended   →            │
│ 2 workstreams landed                          →            │
│ 1 review is due: experience-engineering       →            │
├ fleet ────────────────────────────────────────────────────┤
│ LIVE 2                                                     │
│ ▌SCRYR engine      scr yr-20261124   13h left  last 9h    │
│ ▌Managed browser   steward-…         45m over   last 6h    │
│ PAUSED 1  ·  LIMITED 1  ·  HALTED 0                        │
├ signals ──────────────────────────────────────────────────┤
│ web-console available · notification-hub available · 1 file │
│ unreadable                                       [ Desk ]  │
└───────────────────────────────────────────────────────────┘
```

- The **pulse ribbon** is the hero. It answers "what is moving, what is quiet, what overlaps"
  in one shape. Quiet gaps are drawn as dotted lanes, not blanks, so "no signal" is not "no
  work". The legend says so.
- The **duty line** is a sentence plus three chips, not a row of counts. The sentence is
  generated from the same data; tapping a clause highlights the matching lanes.
- **Needs you** shows the top 2–3 tickets by consequence and age, not the first 3 in file order.
  Consequence is honest: "blocks this commission's next unit" when the supervisor is paused, or
  "delivered 6h ago, not yet read" when it is not. No invented impact.
- **Since last look** is a briefing, grouped by meaning: *wrote after end*, *landed*, *waiting*,
  *stalled*, *review due*. Each group opens the exact records. "Mark seen" is a quiet action and
  never looks like an approval.
- **Fleet** is grouped by state, but live is visually first and paused/limited are visibly
  "open but idle". Halted gets a red broken strip with "Resume or wind down" beside it.
- **Signals** is a thin footer, not a stack of amber alerts. An optional lens being down is not
  an emergency; it is a fact about one field.
- **Overlaps** appear as a joined pair of strips under the fleet, with the shared topics. No
  separate section until there is a pair.

**Needs you — the Queue**

- Header: `19 waiting · 5 older lists`, courier chip, filter chips: **All / Blocking / Questions
  / Notices / Older / Answered**.
- Grouped by commission, with a sticky group header showing the commission state. This makes
  "five questions from one stalled commission" visible as one cluster.
- Each **decision ticket** shows: title; `asked 6h ago · delivered · not read`; the body when
  opened; the recommendation as a filled block with its reason; options as full-width rows with
  a radio state; "answer in my own words" as a secondary expander; the exact dispatch preview on
  confirm.
- **Delivery is a stamp, not a badge**: `delivered`, `not delivered · hub unreachable`,
  `answered · read 09:12`. The reason is a tap away.
- **Older lists** are visually distinct archive slips: readable in place, never answerable,
  always counted separately.
- **Courier** is a single sheet at the bottom: real state, last pass, last delivery, read-back,
  self-test, global switch. Never mixed into the decision list.

**Topics — the Library**

- Desktop: a **research index** data table: Topic (title + mono id), Mode, Last write (+ 7-day
  sparkline), Workstreams, Runs, Read set (segmented mini gauge), Flags. Sortable, filterable,
  with saved views: *All*, *Needs attention* (written while paused/ended, bloated read set,
  waiting decisions), *Quiet* (no write in 7 days).
- Phone: dense two-line rows with the sparkline at the right, flags as words under the title.
- Topic page: a **dossier** with sticky header (title, mode, last write, `Commission` button) and
  four tabs: **Brief** (research write, mission edit, last review as three dated markers, each
  with source), **Workstreams** (open/active/landed, each with its status and order), **Runs**
  (commissions over time as a mini timeline), **Changes** (since last look).
- Read set appears as a **gauge** with `21 KB / 32 KB` and the largest file named; over ceiling
  is amber, never a wall of text.

**Commissions — the Runs**

- Filter chips with counts; sort by last write or allowance end; search. Ended, stopped and
  unknown fold into one quiet group at the bottom.
- Each row is a **flight strip**. The allowance meter is a thin line; when it passes, the strip
  gets an `allowance ended` cap and a small `over by 45m` stamp.
- Commission page:
  - **Header**: human title, mono id, state seal, route chips, last write, allowance meter,
    session identity with evidence.
  - **Overview**: the **stint timeline** — launch, composer ready, stance sent, ready seen, goal
    sent, goal verified, then units and writes, operator entries and their read receipts, controls
    and their confirmations. This is the story of the commission, not a pile of facts. Facts sit
    under it as a definition list with inline provenance.
  - **Steer**: a control rail. The **Say** composer shows `Goes to OPERATOR.md · read at next unit`
    (or `after resume`, `not until steering is enabled`, `never once ended`) as a live destination
    line, then a dispatch preview with the exact entry. Primary is Pause/Resume; Wind down is a
    first-class secondary; Stop and Discard live in a menu with their consequences. The operator
    file is a log with read marks; unparsed headings are flagged with line numbers. Enable
    steering shows the exact line it adds. Per-commission courier switch.
  - **Reading**: the read-set gauge, the file table, growth since the last unit, and the API's
    size gates with "what to cut first".
  - **Files**: a file ledger with type, size, modified, inline preview; markdown, sandboxed HTML,
    images, text. Paths outside the home are copyable, never dead links.
- Every derived value has a **provenance stamp**: a small `ⓘ 09:12` that opens the evidence sheet.
  The claim quote is visually distinct from observed facts.

**Launch — the Flight Plan**

A four-step builder with a persistent plan bar. It is still one screen, but it reads as a route,
not a form.

1. **Route** — searchable topic list with state, last write and open workstreams. Selecting a
   topic adds a route chip and immediately runs the **clean-start verdict**: live / paused /
   limited / stopped / ended / prepared / unknown, what this launch does about it (supersede,
   run alongside, read first), with the desk's recommendation selected and changeable in place.
   This moves the most important safety information to the moment of choice.
2. **Cards** — per selected topic, workstream order as a reorderable list with title, ID and
   status; notes; "mission order" and "clear"; counts against the cap.
3. **Words** — emphasis; stance quotes as quote cards with their source ("newest commission on
   these topics"), editable; add quote; a **packet preview** that renders the stance message as it
   will be sent.
4. **Window** — allowance presets (6/12/24/48/72 h) and a range, model selectors with the cheaper
   default and a cost hint, slug, read-first toggles.
5. **Clearance** — the live preflight: size gates naming what to cut, blockers with the
   recommended choice already selected, quiet earlier commissions folded, read-first. A
   **clearance checklist** (green/amber/red) and one Launch button. The final confirmation names
   what is superseded and the Web Console access in plain words.

Prepared packets are **ready-to-fly strips** at the top: Launch as is / Inspect packet. An
`unknown` packet shows the never-started assertion in place; the two manual messages stay hidden
until it is made. If automatic launch is unavailable, the same screen ends with the manual card:
two messages, copy buttons, and a plain-language reason.

Launch progress becomes a **mission timeline** (not a generic stepper): each step has a time, a
log line on tap, and the manual fallback. A failed step offers Discard (reason required) and
Relaunch as a new commission.

**Reviews — the Evidence Room**

- Index as a ledger: date, topics, verdict count, REVIEW.md / REVIEW.html, open in new tab.
- Review page: a reading room. Desktop two-pane: document and verdict rail. Phone: document then
  verdict cards. A verdict is a **stamp card** recording the operator's words verbatim; unparsed
  headings are listed with their line, never dropped.

**Desk — Systems**

- A systems board: each optional lens as a strip with material state (available / unavailable /
  unknown), what it feeds, which fields read `unknown`, and a Start action when its lifecycle is
  not running.
- Web Console access in plain words, courier state, automatic launch availability, launch and
  courier self-tests, theme (light / dark / system) with a live preview.
- The desk's own state: last read, cache age, build identity.

**States**

| State | Bridge treatment |
|---|---|
| loading | Skeleton lanes and strips with labels "reading the research home"; no spinner-only screen. |
| empty | A composed quiet state: "No window is working" + newest write + Launch; "Nothing is waiting on you" with the last time something was. |
| partial | The affected field shows dotted `unknown` and a reason; the Signals row names the lens; the rest of the page is unaffected. |
| stale | A `Not current · 3 min ago` pill in the duty header; content dims slightly but remains readable; Retry. |
| request-error | A signal-lost composition with the API's sentence verbatim and Retry; navigation still works. |
| offline | "You are offline. Showing the last read at 09:12." No fake live state. |
| permission-denied | Answer buttons disabled with a banner; reads still work. |
| validation-error | Clearance checklist marks the failing gate and names the largest part to cut; Launch disabled. |
| saving/submitting | Dispatch sheet shows busy, cannot double-submit; the page keeps its layout. |

**Accessibility & platform**

- WCAG 2.2 AA; 44 px targets at phone width; visible focus; 16 px phone fields.
- Status always word + shape + colour; charts have an "as list" alternative with the same data.
- Reduced motion freezes the pulse and disables number rolls; no information is motion-only.
- Light/dark first-class; 320, 360, 390, 768, 1024, 1440 tested; landscape; safe areas.
- RTL-safe logical properties; English ships, locale hooks remain.
- Every region uses RCL `ExperienceSurface` with a stable `surfaceId` so the critic and the
  experience phase can address it.

---

### Direction B — **The Desk** (editorial briefing)

> A private correspondent's desk. The desk writes you a short, beautiful brief each time you
> arrive; decisions are reply slips; commissions are dossiers.

**Feeling:** warm, human, literate. Ivory paper, ink, one vermilion accent, a serif display face,
hairline rules and marginalia. The opposite of a terminal — for an operator who reads on a phone
in the morning and wants the story, not the telemetry.

**Now — the front page**

- **Masthead**: `RESEARCH DESK · Thursday 26 November`; a one-line edition summary: "Two windows
  open · nothing halted · nineteen decisions waiting."
- **Lead**: the most consequential change or blocker as two sentences, with a `Read the evidence`
  footnote and an inline action. Not a hero image; a hero *thought*.
- **Waiting on you**: a stack of **reply slips** — perforated edge, the question, the
  recommendation printed like a reply, two reply buttons. The recommendation is one tap.
- **The fleet**: a compact **ledger** with state words, last write, allowance meter and a
  one-line note; live rows have a small moving rule.
- **Since your last visit**: a three-column digest on desktop, one column on phone, grouped by
  meaning with topic names as links.
- **Marginalia**: overlaps, lens-down notices, unreadable files as small notes in the margin —
  present but not shouting.
- **Commission a study** is a letter: choose topics, write your words, see the charter preview,
  sign and send.

**Commission — the dossier**: a cover page with the state seal and route; sections as *Contents*
(Overview / Correspondence / Reading / Files). Steering is **Correspondence**: the operator's
entries as letters with read receipts; controls as "instruct the supervisor" actions. The
timeline is a **log book** with dated entries.

**Why it is strong:** the most pleasurable to read; the most distinctive; excellent for long
research, reviews, and "what happened while I was away"; naturally slows the operator down in a
good way.

**Why it is not the default:** it optimises reading over triage. Nineteen decisions are slower to
scan than in the Bridge; the fleet is less scannable; data density and live state are harder to
keep honest in a paper metaphor; the visual language (serif, warm paper) is a bigger brand change.

**What to steal from it:** the **briefing voice** for "Since last look" and the empty/idle states;
the **reply slip** as the visual parent of the decision ticket; the **dossier** section rhythm for
long-form topic and review pages.

**The Desk across the routes**

| Route | The Desk treatment |
|---|---|
| Now | The front page: masthead, edition line, lead, reply slips, fleet ledger, digest, marginalia. |
| Needs you | The correspondence queue: reply slips grouped by commission, with a full reply sheet for the body and options. |
| Topics | The reading list: an index of topic cards with a one-line "what this is" and a last-write stamp; filters become "editions". |
| Topic | A dossier: cover, abstract (mission essence), workstreams as a contents list, runs as a chronology, changes as annotations. |
| Commissions | The ledger: a compact table of every commission, grouped by state; each row opens its dossier. |
| Commission | A bound dossier with Contents (Overview / Correspondence / Reading / Files); the stint timeline is a log book; controls are instructions to the supervisor. |
| Launch | "Commission a study": a letter with topic lines, your words, the charter preview and a signature action; clearance is a checklist attached to the letter. |
| Launch progress | A dispatch log: dated entries, the manual fallback as a telegram, discard/relaunch as a new commission. |
| Reviews | The review supplement: an index of issues, each with its topics and verdict count. |
| Review | The reading room: the document and its media, with verdict stamps in the margin. |
| Desk | The colophon: systems, sources, self-tests, theme, edition metadata. |
| File | A reading view with a paper margin and a copyable path. |

### Direction C — **The Constellation** (spatial observatory)

> The research portfolio as a night sky. Topics are stars, commissions are orbits, decisions are
> beacons. You navigate by shape and relationship, then drop into ordinary lists for precision.

**Feeling:** cinematic, exploratory, alive. Deep observatory blue-black, fine star fields, one
warm beacon colour, restrained glow. Built for the "which topics are alive, which overlap, what
is drifting" question that lists answer poorly.

**The map**

- Nodes = topics. Size = open workstreams; brightness = recency of research writes; ring =
  a live commission; dashed halo = stalled/quiet; amber beacon = needs you; red fracture = halted;
  dotted = unknown. Edges = commissions that cover more than one topic, so overlaps become visible
  geometry rather than a separate section.
- A **time scrubber** at the bottom replays the last 7 days; pressing play shows waves of activity,
  making stalls and bursts obvious.
- Tap a node: a **lens sheet** with the topic's write, workstreams, runs and decisions.
- Tap a beacon: the decision ticket, answerable in place.
- Launch: **plot a course** — select stars, see the route light up, then the same clearance
  checklist as the Bridge.
- Commission: an **orbit card** with the stint timeline as a trail.

**Why it is strong:** the most memorable and the best at portfolio-level relationships, overlap
and drift. It makes the research system feel like a living thing, which it is.

**Why it is not the default:** precision and speed for 19 decisions are worse; mobile real estate
is hostile to a map; accessibility needs a complete non-spatial equivalent (which is the real
cost); it is a new interaction model to learn. It is a superb **optional Map view** for desktop,
not the front door.

**What to steal from it:** the node/edge model for an optional **Map** tab on Topics; the time
scrubber as an alternate view of the pulse ribbon on desktop; the beacon treatment for
needs-you markers on the map.

**The Constellation across the routes**

| Route | The Constellation treatment |
|---|---|
| Now | The map is the front door, with a signal stack of the top decisions and a 7-day scrubber. A list toggle renders every reading as a table. |
| Needs you | Beacons on the map; the signal stack opens the same decision tickets as the Bridge. |
| Topics | The map is the index; a "List" toggle gives the research index for precision. |
| Topic | A lens sheet over the map: the write, workstreams, runs and decisions, with the node held in place. |
| Commissions | An orbit rail: each commission as a trajectory with its topics as nodes, plus a list. |
| Commission | An orbit dossier: the stint timeline drawn as a trail, with the same Overview / Steer / Reading / Files sections. |
| Launch | "Plot a course": select stars, see the route light up, then the same clearance checklist and packet preview. |
| Launch progress | The trajectory: steps as points, the manual fallback as a plotted alternative. |
| Reviews | Select a cluster to scope a review; the review itself is the reading room. |
| Review | The reading room, with a map thumbnail of the reviewed topics. |
| Desk | The observatory settings: systems, sources, self-tests, theme; the map's legend and fidelity notes. |
| File | An inspector sheet over the map, with the file rendered as in the other directions. |

---

## 5. Comparison

| Criterion | A · The Bridge | B · The Desk | C · The Constellation |
|---|---|---|---|
| Five-second "does anything need me?" | **5/5** — duty line + beacon tickets | 4/5 — masthead + reply slips | 3/5 — beacons require reading the map |
| Five-second "is anything moving?" | **5/5** — pulse ribbon | 4/5 — fleet ledger + live rules | **5/5** — brightness and orbits |
| Speed to answer 19 decisions | **5/5** — ranked queue, one-tap recommendation | 3/5 — reply slips are rich but slower | 3/5 — beacons are scattered |
| Reading long research | 4/5 — dossier tabs, good prose | **5/5** — editorial reading room | 3/5 — map is not a reader |
| Phone ergonomics | **5/5** | **5/5** | 2/5 |
| Desktop use of space | **5/5** — rail + two-pane + wall | 4/5 — two-column editorial | **5/5** — full-bleed map |
| Distinctiveness / award potential | **5/5** — pulse line + flight strips | **5/5** — editorial voice | **5/5** — spatial identity |
| Build cost | M | M–L (new type/brand system) | L–XL (map, accessibility equivalent) |
| Risk | Low–M | M (brand shift, density) | High (new interaction model) |
| Fits the existing design tokens / RCL | **High** | Medium (new type/tokens) | Medium–High (map asset needed) |
| Preserves all features cleanly | **Yes** | Yes | Yes, but some become secondary |

**Recommendation: build A · The Bridge, with B's briefing voice inside "Since last look" and
C's map as an optional desktop view after the core ships.** This gives the operator an
award-winning, ownable default without betting the product on a new interaction model.

---

## 6. What the experience-engineering capabilities contribute

This proposal is not a picture of a redesign; it is the first customer of the EE pipeline.

| Capability | Use in this redesign | Artifact / label |
|---|---|---|
| **EX9 capture kernel** | Capture today's 12 routes at 390/1440, light/dark, plus the six journey states, into one `ux-capture.v1` bundle. The captures in `assets/` are the interim baseline; the bundle replaces them. | `captured` |
| **EX8 feature inventory** | Before any implementation, take the served inventory of every route, action and component; after each phase, diff it; every lost feature becomes a keep/drop card. This is the guarantee behind "same features". | inventory + decision cards |
| **EX11 redesign round trip** | Edit the frozen page's outline into the Bridge layouts, compositing unchanged regions and rendering changed regions from real RCL components fed the captured data. The output is the implementer's spec. | `composited` |
| **EX12 region feedback** | The operator reviews the proposal on a phone, taps a region, writes one sentence; the comment binds to the outline node and routes to the owning implementation task. The same component powers in-app point-and-correct after launch. | region IDs + corrections |
| **EX1 UX critic** | Score every built screen at 390 and 1440, light/dark. The critic's first job is the defects in §2: stacked drawer actions, phone overflow, contrast, 44 px targets, crowded nav, oversized type. | ranked findings with screenshots |
| **EX7 usage contracts** | New components ship with contracts: BottomNav max 4; DecisionTicket max 2 options + own words; PulseRibbon max 6 lanes then "show all"; FlightStrip label budgets; every overlay uses the shared action bar (one row, one primary). | dev warnings, stories, static call-site check |
| **EX3 studio** | Build the new primitives in the RCL studio; the studio revises a page from this capture bundle. No local one-off copies of tabs, dropdowns, nav or action bars. | RCL assets + adoption report |
| **EX2/EX6 experience phase** | `vrooli scenario test research-desk --phases experience` becomes the redesign gate. `not-evaluated` never passes; false-pass/false-fail measured on the journey corpus. | one verdict with evidence |
| **present-ux-decisions** | This study follows the method: decision first, today beside the proposal, one card per option, what approval does, no answer as a real choice, fidelity labels and provenance at the bottom. | decision page |

### Proposed new RCL assets (with contracts)

| Asset | Contract sketch |
|---|---|
| `PulseRibbon` | 1–6 lanes, then "show all"; each lane has state, ticks, allowance cap and quiet gap; an `asList` prop renders the accessible table; no motion under reduced motion. |
| `StateSeal` | Word + shape + colour; 8 states; sizes sm/md; never icon-only. |
| `FlightStrip` | Title ≤ 2 lines, mono id optional, route chips ≤ 4 + overflow, one meter, one primary action; whole strip is one link; no nested interactive targets. |
| `DecisionTicket` | One question, ≤ 2 offered options + own words, one recommendation with reason, delivery stamp; the answer action is never one tap (confirmation). |
| `ReadSetGauge` | 2–6 segments, each with label, bytes, ceiling; over-ceiling is a word + pattern, not only colour. |
| `EvidenceSheet` | Source path, rule, observed time, raw value, open-file; one sheet per value. |
| `BriefingDigest` | 1–5 groups, each a sentence + count + expandable records; never a raw change list. |
| `ClearanceChecklist` | 1–12 checks, each pass/warn/fail + reason + fix; fail blocks Launch with a word. |
| `MissionTimeline` | Reuses `Timeline`; steps with time, state, detail; manual fallback slot. |

### 6a. What the desk itself gains from the same capabilities

The desk is a launcher, lens and courier over research. Some of that research *is* UX work, and
the operator's decisions about it are visual. The redesign makes the desk the place those
decisions are answered, rather than a text inbox that links elsewhere.

- **Decision tickets can carry evidence.** When a needs-you item references a `ui` evidence
  record, the ticket renders the shared `HubCharts.evidence.uiViewer`: a `captured` before and a
  `composited` after at the same viewport, fidelity chips, a compare slider and tap-to-comment
  regions. A text-only item renders exactly as today. The item body is never replaced by the
  picture; the picture is the thing the question is about.
- **Reviews become evidence rooms.** A review session can carry EX9 capture bundles for the
  topics it covers; REVIEW.html can show before/after; a verdict may bind to a region ID, so
  "the drawer footer stacks" is recorded against the exact element, not inferred later.
- **The courier carries the packet.** A `decision` delivery may include an HTML packet as an
  attachment. The desk shows the packet inline with "open full packet"; the answer still goes
  back through the operator file. A desk outage still loses no answer.
- **The answer returns with the region.** For a visual item, the dispatch sheet appends the
  operator's words plus the region reference (`region <id>: <text>`) in the same operator-file
  entry, so the supervisor and the implementer see exactly what was meant.
- **The desk is corrected the same way.** With the EX12 root-provider overlay on the served
  desk, the operator long-presses any element and writes one sentence; the correction carries
  the outline node, component identity, screenshot crop, build and viewport, and routes to the
  owning task. This is the mechanism that keeps the redesign from drifting after launch.

This is the same contract as today — files are the state, the desk is never in the loop, and
nothing new is required of a supervisor — with a richer evidence surface where it matters.


---

## 7. Implementation plan

Assumes the API changes are read-only projections; no new authoritative state.

**Phase 0 — Evidence and baseline (1–2 days)**
- EX9 capture bundle of the current 12 routes and six journeys.
- EX8 inventory of routes, actions, components, states.
- Baseline measures: 5-second test, time to answer, time to launch with defaults, operator-caught
  defects in the last review, accessibility/axe, 320/390/1440 overflow.
- Token audit: which Bridge tokens already exist; which need a brand-manager/RCL addition.

**Phase 1 — Primitives and tokens (1 week)**
- Build `StateSeal`, `FlightStrip`, `DecisionTicket`, `ReadSetGauge`, `EvidenceSheet`,
  `PulseRibbon`, `BriefingDigest`, `ClearanceChecklist` in RCL with stories at 320/390/1440,
  light/dark, long German labels, reduced motion, and contracts.
- Fix the shell: 4-tab nav contract, top Launch, command palette, desktop rail.
- Read-only `ActivitySeries` projection (writes per commission and per day) with honest
  `unknown` when a write cannot be dated; cap the read set.

**Phase 2 — Now and Needs you (1.5 weeks)**
- The Watch: duty header, pulse ribbon, decision shelf, briefing, fleet strips, signals, overlaps,
  prepared packets, state counts.
- The Queue: filters, groups, tickets, answer flow, dispatch sheet, courier sheet, older lists,
  answered, permission-denied, delivery reasons.
- This is the phase that must pass the 5-second test before anything else ships.

**Phase 3 — Commissions (1.5 weeks)**
- Runs index with filters and folded states.
- Commission dossier: header, stint timeline, overview facts, steer rail, operator-file log,
  reading gauge, files ledger.
- All controls confirmed; all refusals show the API's reason and the wind-down alternative.

**Phase 4 — Launch (1.5 weeks)**
- Four-step flight plan, clean-start verdict at topic selection, packet preview, clearance,
  plan bar, prepared and unknown packets, manual fallback.
- Launch progress as mission timeline; discard/relaunch.

**Phase 5 — Library and rooms (1.5 weeks)**
- Topics table/list, saved views, sparklines, topic dossier.
- Reviews index and reading room; verdict stamps.
- Desk systems board, self-tests, theme; File viewer.
- Optional desktop `Map` view (Constellation) behind a flag.

**Phase 6 — Proof and polish (1 week)**
- EX1 critic clean or every finding explained; EX2/EX6 experience phase.
- Served journeys at 320/390/768/1024/1440, light/dark, reduced motion, keyboard, screen reader
  spot checks.
- EX8 after-diff: zero unexplained features lost.
- Two operator batch reviews through Notification Hub with EX12 region comments, bound to exact
  versions.
- Performance: first meaningful paint under 1.5 s on the tunnel; no layout shift when numbers roll.

**Sequencing constraints**

- EX9/EX11/EX12 are ideal but not blockers for the first two phases: use the interim captures and
  CSS/RCL sketches, label them honestly, and replace them with composited bundles as they land.
- Do not start implementation until the operator picks a direction and the feature inventory is
  signed off. Taste calls are cheap here; rework is not.

---

## 8. Acceptance — how we know it is better

| Measure | Today | Target |
|---|---|---|
| Five-second test: "does anything need you?" | not measured | 5/5 in a 5-person test, phone and desktop |
| Five-second test: "what is working?" | not measured | 5/5 |
| Five-second test: "what changed?" | not measured | ≥ 4/5 |
| Median time to answer one needs-you item | not measured | < 30 s from opening the app |
| Median time to launch 3 topics with defaults | not measured | < 60 s |
| Operator-caught UX defects per review | baseline from EE corpus | ≤ 2, then trending to 0 |
| Feature parity | 12 routes / 6 journeys | 100% of the EX8 inventory, or a keep/drop card for each difference |
| Critic | not run | no unexplained findings at 390/1440, light/dark |
| Accessibility | AA claimed | AA verified by axe + manual keyboard/reader spot checks |
| Overflow | 390 only | no horizontal overflow at 320/360/390/768/1024/1440 |

---

## 9. Feature parity map

Every item below exists today and must exist after the redesign. "New home" names the Bridge
surface; the Desk and Constellation keep the same data with different presentation.

| # | Feature (current) | New home in the Bridge | Preserved |
|---|---|---|---|
| 1 | Live / halted / paused / limited counts | Duty header chips + fleet groups | yes |
| 2 | Newest research write anywhere, with note | Duty sentence + pulse lane marker | yes |
| 3 | Needs-you count and top titles | Decision shelf (ranked) | yes |
| 4 | "Items open, nothing delivered" warning | Courier chip + Queue banner | yes |
| 5 | Supervisors grouped by state | Fleet strips, state groups | yes |
| 6 | Commission topics, state rule/evidence, claim, last write, changes, allowance-ended badge | Flight strip + evidence sheet + allowance meter | yes |
| 7 | Since period (look/day/week), topics written, Caught up | Briefing digest + period control + Mark seen | yes |
| 8 | Lenses down notices | Signals footer + Desk systems | yes |
| 9 | Unreadable files | Signals footer + inline unknown rows | yes |
| 10 | Overlaps among open commissions | Joined strips + map edges | yes |
| 11 | Prepared packets | Ready-to-fly shelf | yes |
| 12 | State counts | Fleet footer / Runs filters | yes |
| 13 | Commission list grouping/filter | Runs chips + folded groups | yes |
| 14 | Commission facts with evidence | Overview definition list + evidence sheets | yes |
| 15 | Changes + mark as seen | Overview briefing + Mark seen | yes |
| 16 | Sessions and launch timeline | Stint timeline + session fact | yes |
| 17 | Say / wind down, read timing, entry limit | Steer composer + destination line + counter | yes |
| 18 | Operator file with read marks, unparsed headings | Operator log | yes |
| 19 | Pause / resume / stop / discard / re-send stance | Control rail + overflow menu | yes |
| 20 | Refused control reason, keystrokes, wind down instead | Inline refusal + manual sheet | yes |
| 21 | Enable steering with dry-run line | Steer banner + exact-line sheet | yes |
| 22 | Per-commission courier switch | Steer / courier row | yes |
| 23 | Reading read-set sizes and bloat | Read-set gauge + file table | yes |
| 24 | Files panel | Files ledger | yes |
| 25 | Needs-you feed: expanded body, options, own words, exact entry, sent/not sent/answered/read, reused ID, global items, secret items, older lists, courier health, self-test, global switch | Queue tickets + dispatch sheet + courier sheet | yes |
| 26 | Launch: prepared, unknown assertion, topics max, workstream order, notes, emphasis, stance quotes + source, allowance range/default, models + cost hint, slug, form persistence, preflight, blockers + recommendation, quiet folded, read-first, existing packet STATE/read set, confirmation, progress steps, failed start discard/relaunch, manual messages | Four-step flight plan + clearance + mission timeline | yes |
| 27 | Reviews list; REVIEW.md/HTML + media; verdict per topic; unparsed headings | Reviews ledger + reading room + verdict stamps | yes |
| 28 | File viewer: markdown, sandboxed HTML, images, text; outside-home paths copyable | File ledger + preview | yes |
| 29 | Desk: capabilities, sources, Web Console access, courier state, auto-launch, self-tests, theme | Systems board | yes |
| 30 | Request lifecycle: 30 s / 60 s polls, focus refresh, stale, non-optimistic writes, dry run, pending, retry | Same, with freshness arc and dispatch sheet | yes |
| 31 | Accessibility: 44 px, status words, unknown, reduced motion, locale direction | Same, plus chart alternatives and surface IDs | yes |
| 32 | No command-line text; API sentences as sent | Same; provenance sheets show paths as data, not instructions | yes |
| 33 | Topics index: filter, newest-write sort, mode, open workstreams, runs, read size and bloat, written-while-paused, source file on tap | Library table/list + saved views | yes |
| 34 | Topic dossier: research write, mission edit and last review apart, workstreams, commissions over time, changes since last look | Topic dossier tabs | yes |

Additive (not replacements): command palette, saved topic views, optional desktop map, time
scrubber, read-set growth sparkline. These are optional and can ship later.

---

## 10. Decisions I need from you

1. **Direction.** A · The Bridge (recommended), B · The Desk, or C · The Constellation as the
   default? (C can ship as an optional view under any choice.)
2. **Light or night first.** The Bridge is designed light-first ("Day Watch") with a full "Night
   Watch" dark mode. Do you want the desk to open light (calmer, paper) or dark (current habit)?
3. **The pulse ribbon.** Is a time-axis ribbon the right hero, or would you rather the hero be a
   ranked decision queue with the ribbon below it? Both preserve every feature; this is a taste
   and priority call.
4. **Brand latitude.** May the desk get its own mark (the pulse line) and a display face, or
   should it stay entirely inside the current default kit? The proposal works either way; the
   former is what makes it award-winning.

No answer is a real answer: the current desk keeps working, and no commission is affected.

---

## 11. Provenance of this study

- Current-state captures: `assets/captured-*.png`, taken 2026-11-26 ~03:02Z (dark) and ~03:34Z
  (light default) from `http://127.0.0.1:20004` at 390×844 and 1440×1200. These are `captured`
  (fact), not an EX9 bundle.
- Feature inventory: `scenarios/research-desk/experience/pages/*.json`,
  `experience/journeys/*.json`, `docs/concepts/EXPERIENCE.md`, `docs/concepts/SYSTEM.md`.
- Design constraints: `scenarios/research-desk/DESIGN.md`, `ui/src/styles.css`,
  `docs/reference/component-library-gaps.md`.
- EE capabilities: `research/topics/experience-engineering/MISSION.md`, `knowledge/MAP.md`,
  `skills/present-ux-decisions/SKILL.md`, EX9/EX11/EX12 briefs.
- Visual study: `index.html`. The phone mockups there are `sketch` (CSS, not RCL components); the
  implementation target is `rendered` / `composited` through the EE pipeline.
