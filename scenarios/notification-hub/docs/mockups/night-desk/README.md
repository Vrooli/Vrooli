# Night Desk — the decision room

A desktop + phone design proposal for the operator side of Notification Hub.

Night Desk is one system with two compositions. Desktop puts the queue, the
evidence and the answer in one field. Phone puts one decision under your thumb.
Neither is a reflow of the other. The visual language, **Night Signal**, comes
from the product mark: deep space, moonlight, and one cyan signal.

- `index.html` — the case study (open this first)
- `desktop.html` — 1440 desktop composition
- `mobile.html` — 390 phone briefing
- `mobile-detail.html` — 390 phone decision
- `focus.html` — desktop focus review, one ask carrying three questions

Each screen supports light and dark: add `?theme=light` or `?theme=dark`, or use
the theme control. Screenshots are in `shots/` at 1440×940 and 390×844 (plus
320 px phone checks).

## The concept

**A quiet instrument for the work that needs you.** The desk answers three
questions before you arrive: how many decisions exist, which ones close, and
which are one cause wearing two hats. Then it gets out of the way.

- **One accent, one meaning.** Cyan is attention and action. Everything else is
  moonlight and hairline. No left accent bars, no gradient text, no card grid.
- **A count, not a pile.** Decisions only; status updates live in Updates.
- **One decision per cause.** Two checks with one cause are one card. One ask can
  already carry several questions, so three grants become one pass.
- **Every row answers cold.** Title, source, age, recommendation, effect and the
  cost of waiting — enough to answer without opening anything.
- **Show the close.** Answer, owner run, result, or a visible stall.

## What is real, what is derived, what is proposed

The case study carries the full table. In short:

- **Live:** the count, deadlines and closing-soon, source facets, title, big
  picture, recommendation and reason, options, “if you do nothing”, recurrence,
  and multi-question asks with per-question recommendations.
- **Derived:** the day/week/later bands, and “one cause” from the Hub's existing
  subject-similarity check.
- **Proposed:** a declared `decision_family` (already in the proto, not yet
  populated), a persisted readiness projection, semantic “related” suggestions,
  an execution/aftermath projection, and optional impact facts.

Read `PROPOSALS.md` for the field-by-field design, the code paths, and the
implementation order.

## Fidelity

These are `rendered` design proposals, not captured screenshots of the running
Hub. Content follows the 24 November backlog: two guard decisions with one
cause, three Portal grants, an Agent Manager consent, a research deferral and a
review. No number on these screens is invented; where a real field does not
exist, the element is left out or marked in the data table.

The font is Inter shipped with this folder (`fonts/Inter.woff2`, SIL OFL), so the
proposal renders the same everywhere. The product should ship the same face for
`--font-sans` rather than relying on the host's fallback.

These are design documents, not sendable decision pages. `notification-hub pages
check` expects a single self-contained page with the kit's tokens; to check one
of these, inline `night.css` and map the `--moon`/`--cyan` tokens to the kit's
`--hub-*` names first.
