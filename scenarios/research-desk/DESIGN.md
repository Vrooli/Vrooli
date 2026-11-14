---
id: research-desk
name: Research Desk
description: Phone-first lens, launcher and courier over the research home, in the Vrooli default design language (0.2.0).
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
  cardRadiusMaximum: "0.5rem"
  defaultMode: "light"
  supportedModes: ["light", "dark", "system"]
  responsiveBaseline: "mobile-first"
  dominantPalette: "neutral-operational-with-blue-cyan-and-semantic-status"
---

# Research Desk Design

This file is the source of truth for Research Desk's UI decisions. The tokens come from the
Vrooli default design language and are generated into `ui/src/design-tokens.css`; they are
not edited by hand. Rules that belong only to this scenario live in `ui/src/styles.css` as
`rd-*` classes built from those tokens. Components come from the React Component Library.

## Intent

The operator supervises long-running research from a phone through the tunnel, in short
glances: does anything need me, is everything moving, what changed. The desk answers those
first, and every other screen is one tap from them. It should feel calm, dense and
technical: state as a word plus colour, the operator's own words set apart from what the
desk derived, and nothing decorative.

Everything outward (launch, controls, Say, answers, verdicts, the courier switch) goes
through one confirmation that shows the exact text and where it goes. The danger and warning
roles are kept for those moments and for failures, so they still mean something.

## Layout

- One column, at most 60rem wide, centred. The library app shell gives bottom navigation on
  a phone and a sidebar on a wide screen; there are no split panes or inspectors.
- Nothing scrolls sideways at 390 px or in landscape. Long ids and paths wrap anywhere; wide
  content (tables, code, file text) scrolls inside its own box.
- Facts sit label over value on a phone and side by side from 48rem.
- Cards frame only repeated records (a commission, a needs-you item, a review). A needs-you
  item is one collapsed row (title, where it came from, delivery state, the recommended
  answer) whose body opens on tap.
- The launch bar sticks to the bottom only once topics are chosen, and never covers content
  with the bottom navigation.

## Color

Neutral slate surfaces dominate. Blue marks commands and the selected place, cyan technical
emphasis, green work that is done or running well, amber what waits on the operator or is
not being delivered, red failures and destructive confirmations. Colour never carries state
alone: every state is also a word (`live`, `paused`, `limited`, `stopped or ended`,
`unknown`). Light is the default; dark and system are first-class.

## Typography

Inter or the platform sans stack for the interface; monospace for commission and topic ids,
paths and file text. Body text is 16 px, and every phone field stays at 16 px so iOS does not
zoom. Headings are compact; there is no hero type. Letter spacing is zero.

## Words

- The operator's words are quoted as written; derived text is plain.
- A value the desk cannot derive reads `unknown`, in muted italics, with a "Why?" that shows
  the evidence. Stale reads are labelled stale.
- Times are relative, with the absolute UTC time on tap.
- The API's sentences (refusals, blockers, verdicts, delivery reasons) are shown as sent; the
  UI never rewrites or matches on them. Nothing the operator reads contains command-line
  text.

## Components

Library components first; a gap the desk has to cover is recorded in
`docs/reference/component-library-gaps.md`. A write never happens in one tap or from a deep
link. Discarding a bad start asks for a reason. No text box ever takes a secret.

## Responsiveness

Phone first. At phone width every control, path link and in-text link is at least 44 px tall
whatever pointer the browser reports. Bottom chrome respects the safe area. Nothing important
hides behind hover.

## Preferences

The theme (light, dark, system) is chosen on the Desk page and remembered. Text size follows
the browser's own setting because the tokens are in rem. Reduced motion stops the one moving
element (the current-step dot). Locales are wired with text direction, though only English
ships.

## Feedback & State

Every operator action shows its state at once: the confirm button shows busy and cannot be
pressed twice, and the result (or the API's refusal, in its own words) appears beside the
action. A failed form keeps what was typed. Lists and pages have loading, empty, partial and
error states rather than blank space; an optional scenario that is down shows `unknown` for
what it feeds and the rest of the page still works. Toasts are not used; a failure that
blocks stays on the page with a retry.

## UX-State Contract

Every major surface declares the lifecycle states that users can encounter:
`idle`, `pending`, `success`, `loading`, `saving`, `syncing`, `refreshing`,
`empty`, `partial`, `stale`, `validation-error`, `request-error`, `retrying`,
`permission-denied`, and `offline`. A page may add a more specific state, but it keeps the
generic state legible and actionable. The state is represented in the accessibility tree
with a role, name, status or alert that explains what changed and what the operator can do
next.

Experience Manager's state coverage check uses this section as the contract source. A page
that never meets a state may omit it with a documented reason in its experience spec; a blank
or silent state is never a reason.

## Request Lifecycle

Reads refresh every 30 seconds and when the window regains focus (the optional-scenario
lens every 60 seconds), and say when what is shown is stale. Writes are never optimistic:
the confirmation shows the dry-run text the API returned, and the screen changes only after
the API has written. A slow step shows a visible pending state in a stable layout. A failure
says what happened, what is still safe, and offers retry.

## Accessibility

Visible focus on every control, readable contrast in light and dark, status never by colour
alone, and 44 px targets. Live counts and refusals are announced through `status` and `alert`
roles.
