# Experience Design

## Purpose Of This Document

The UI decision for Research Desk: the comparison, the primary surface, the shell, and what
the design is accountable for. States, actions and data come from
[`SYSTEM.md`](SYSTEM.md); this file only says how they are shown. `experience/` is the
machine-readable contract and wins on disagreement. The UI is built after the API and CLI
(G10) and replaces the generated placeholder. The operator uses it on a phone through the
tunnel hostname; nothing links to `localhost`.

## The Comparison

A phone status desk such as a CI dashboard or pager app: one glance answers "does anything
need me, is everything moving", one tap reaches detail. Dense legible rows, state as a word
plus colour, relative times with the absolute time on tap, operator words in a quoted style
distinct from derived text, claims shown apart from observations.

## The Primary Surface

**Now** (`/`), top to bottom; the first three fill the first screen at 390 px (rulings V5):

1. **Status line**: "1 live · 2 paused · 0 limited", because only live is working (rulings
   F4), and the most recent research write anywhere with its topic (rulings P8). With
   nothing live but windows open, a notice says so ("Nothing is working right now").
2. **Needs you**: the count and at most three titles; each opens that item on the Needs you
   page. When items are open and nothing is delivered, one warning line links to the switch.
3. **Supervisors**: a group each for live, limited and paused: one row per commission with
   its topics, the state rule with its evidence on tap ("Why?"), the supervisor's claim in
   a quoted style, its last write ("at least …" when the walk was cut short) and what
   changed in the chosen period. A window still open after its allowance ended carries an
   "allowance ended" badge.
4. **Since**: a period (since the last look, the last day, the last week; with no recorded
   look the API reports the last 24 hours and the line says so), the topics written in it
   with what changed, and one "Caught up" action that marks everything seen after a
   confirmation (rulings V7, P7).
5. **Lenses down and unreadable files**: one notice per source that cannot be reached,
   and one list of research-home files that exist but cannot be read. What they feed
   reads `unknown`, never empty.
6. **Overlaps**: one row per pair of commissions live, paused or limited now whose time on
   the same topics overlapped, with the topics they shared. History lives on the commission
   page. An edit after a commission ended ("touched after end") never makes an overlap.
7. **Prepared packets**: packets that launch as they are (prepared, or `unknown` with the
   operator's never-started assertion available), each one tap from launching.
8. **Every commission**: state counts, each opening the filtered list.

Phone (360–430 px): one column; 44 px targets; ids and paths wrap; tables and code scroll
inside their own box; bottom tabs Now, Needs you (with a count badge), Topics and Commissions;
Launch (also the first button on Now), Reviews and Desk in the menu. Desktop: the same pages in a wider column with
a sidebar. Every read has the same four states: loading, failed (unreachable, not
authorized, or the API's reason verbatim), stale (the earlier read kept on screen under a
"Not current" notice with its time) and loaded. A lens down marks only its fields
`unknown` with the reason.

## Shell Configuration

| Setting | Value | Why |
|---|---|---|
| Kit | `vrooli-default` | Neutral operations tone. |
| `mobileNav` | `menu-tabs` | Four tabs keep every label whole at 390 px; Launch is the first button on Now, and Reviews and Desk are occasional. |
| `mainMode` | `scroll` | Lists and details; no pinned composer. |
| Locale | English only | The template's Japanese and Arabic files were untranslated placeholders and were removed rather than shipped. |

## What The Design Is Accountable For

1. Within five seconds the operator knows whether anything needs them and whether every
   expected commission is moving.
2. Every value is derived from evidence or says `unknown`; claims and observations are
   never blended. Each fact offers "Why?": the source, the reason and the time it was read.
3. Launching, steering, answering and control (pause, resume, wind down, stop, discard)
   never need a chat message; the manual path is one tap away. When the desk refuses to
   type into a window, winding down through the operator file still works and is offered.
4. Nothing outward is one tap. Every write is confirmed in a dialog that shows the exact
   text and where it goes (the operator file, or the keys typed into the supervisor's
   window); discard requires a reason; a failure keeps the dialog open with the reason.

## Information Architecture

| Surface | Route | Answers / actions |
|---|---|---|
| Now | `/` | As above. |
| Needs you | `/needs-you` | One feed of collapsed rows: title, where it came from, delivery state and the recommended answer; the body opens on tap. Each row's anchor is its commission plus its ID (`#<commission>/<id>`, `global/<id>` for the global file), so a link from Now scrolls to it and opens it. Answer with an offered option or in your own words; the confirmation shows the exact entry the API's dry run returns and when the supervisor reads it. Shows sent, not sent and why (the API's reason code, never matched prose), answered, and seen. An item whose ID was already answered or delivered reads "ID reused, not delivered" and offers no answer. A global item without a commission says what it says to do and links where it is answered (rulings P6), and is never shown as answerable. Secret items say where the secret goes and never offer a text box. Unreadable files are listed, never shown as an empty feed. The courier's real state (last pass, last delivery, read-back), its confirmed self-test, and the global courier switch; with items open and nothing delivered, one warning line. |
| Topics | `/topics`, `/topics/:topic` | A filter box; sorted by last research write (run folders), its source file on tap, and "written while paused" when it came under a paused or ended commission (rulings V6, P8); mode, open workstreams, commissions, start-up read size with bloat flagged. Per topic, the research write, the mission edit and the last review apart, each with its file; workstreams, commissions over time and changes since the last look. |
| Commissions | `/commissions`, `/commissions/:id` | All commissions by state (ended, stopped and unknown folded). One commission in four tabs: **Status** (facts with evidence, including when it ended and any "touched after end", which is never activity; changes since the last look or since a picked time, with the API's comparison line and Mark as seen; sessions; launch timeline), **Steer** (Say a direction or wind down, with the API's word on when it is read: at the next unit, after resume when paused, not until steering is enabled, or never once ended, shown as a warning when it waits (rulings V4); the confirmation shows the dry run's exact entry; a count against the 4,000-character entry limit that says to name a file past it; the operator file with seen marks; pause, resume, stop, discard, re-send stance, where a refused control shows the API's reason, the keystrokes to type by hand, and "Wind down instead", the path that always works; enable steering on older charters; the courier switch), **Reading** (the start-up read set and its sizes) and **Files** (state, ledger, cards, operator file). |
| Launch | `/launch`, `/launch/:id` | Prepared packets launch as they are; an `unknown` packet the API can assert was never started has a Launch button behind that assertion, and its two manual messages stay hidden until it is made. Builder: topics as a plain list in the page scroll, per-topic workstream order (titles, not bare IDs) and notes, emphasis, stance quotes filled from the newest commission on those topics with their source (editable), allowance, models (the lower-powered default pre-selected, a cost warning on costlier ones), slug. The form survives leaving the page (this tab). Live preflight: size gates naming what to cut; each earlier commission that blocks, once, with the desk's choice already selected (supersede or run alongside, with why) and changed on the same screen (several may be superseded at once); quiet earlier commissions folded; read-first offered ticked. No command-line text anywhere: "Relaunch as a new commission" is a button wherever it applies. The default launch is tick, tick, tick, Launch, Confirm; the confirmation names what is superseded and Web Console access in plain words. Progress step by step; a failed start offers Discard (reason required) and Relaunch. When automatic launch is unavailable the same screen ends in the manual steps. |
| Reviews | `/reviews`, `/reviews/:id` | Review sessions; REVIEW.md and REVIEW.html (sandboxed) with media; a verdict per topic recorded verbatim. |
| File | `/files/*` | Any research-home file: Markdown, sandboxed HTML, images, text. Paths outside the home are shown as copyable text, never as a link that would fail. |
| Desk | `/desk` | Reachable optional scenarios and what is `unknown` because of each; the sources; Web Console access in plain words (whether an automatic launch can type into a supervisor window, and why not); the courier's real state; automatic launch availability; the confirmed launch and courier self-tests (rulings F15 restored the courier test); the theme. |

The commission page uses tabs so that a phone never scrolls through steering controls to
read status. Reviews and the Desk are occasional, so they sit in the menu rather than the
tab bar.

## Cross-References

- `path:scenarios/research-desk/docs/concepts/SYSTEM.md` — states (§4), launch (§5), control (§6), courier (§7; rules in FILE-CONTRACT §5), changes (§8)
- `path:scenarios/research-desk/experience/README.md` — the typed contract
- `path:scenarios/research-desk/DESIGN.md` — the token contract
