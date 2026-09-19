# Money Ledger UX architecture

Money Ledger is the source-of-truth console for typed money events, their
provenance, and the read-time position derived from them. The UI never edits
or deletes a posting. A correction is a linked reversing posting, and every
figure that can affect a decision states its currency and basis beside it.

## State vocabulary

Every page uses the shared `ExperienceSurface` state contract. These states
are distinct and must not be collapsed into a generic empty or zero value:

| State | Meaning | Rendering rule |
|---|---|---|
| default/idle | The surface is ready and has no active mutation. | Show current data and its basis. |
| loading/pending | The first request is in flight. | Keep the page shell and announce busy state. |
| refreshing/syncing | Existing data is being refreshed. | Retain existing data and show a non-blocking status. |
| saving | A user mutation is in flight. | Disable the submitted action and announce progress. |
| success | A mutation completed. | Announce the result and refresh the affected read surface. |
| empty | No record exists yet. | Give a first action; never show an empty table as a zero balance. |
| partial | Some declared sources are unavailable or incomplete. | Show the figure, missing source, reason, and age. |
| stale | Data exists but its freshness window has elapsed. | Show the value with its age and stale label. |
| validation-error | A form is incomplete or invalid. | Put a text error near the form and retain entered values. |
| request-error | The server rejected or could not complete a request. | Announce the error and keep a retry/action affordance. |
| retrying | A retry is active. | Announce retry progress; do not synthesize a result. |
| permission-denied | The caller cannot perform the action. | Explain the role boundary in text. |
| offline | The API cannot be reached. | Preserve context where possible and name the unavailable source. |

`partial`, `stale`, `unavailable`, and genuine zero are four different facts:
an unavailable or missing source has a named gap; stale data has an age; a
partial result has an explicit completeness label; only a successfully
computed zero is rendered as zero.

## Financial rendering rules

- Amounts are stored and submitted in minor units, but displayed with the
  locale-aware currency formatter and ISO currency code.
- Money columns use tabular numerals. Outflows are described as outflow or
  expense as well as using their negative sign where applicable.
- Basis is visible in the same row or immediately adjacent summary. It is
  never available only through hover, colour, or a tooltip.
- Source, fetch age, partial status, and failure reason are text. Colour is
  supplemental and never the only signal.
- Journal rows expose date, account, description, signed amount, currency,
  basis, source, and reversal link. Statements expose currency and coverage
  beside every aggregate.

## Surface specifications

| Route | Purpose and primary action | Data sources | Degradation and mobile strategy |
|---|---|---|---|
| `/` | Read position, runway, goal verdicts, declare a goal, and inspect the observed runway/burn trend. | Books, Position, Goals, Journal postings. | Partial position or posting reads show a named gap; no postings are rendered as undefined rather than zero; mobile stacks cards/charts and keeps the visible trend table horizontally scrollable. |
| `/accounts` | Create books/accounts, inspect balances, and make a paired transfer. | Books, Accounts, Postings. | Balance gaps say unavailable rather than zero; the account table scrolls inside its container on mobile. |
| `/journal` | Record a manual event and reverse an existing posting. | Books, Accounts, Postings, audit trail. | Errors stay beside the form; the journal table scrolls inside its container on mobile. |
| `/adapters` | Register, run, and import through typed adapters. | Adapter registry and receipts. | Availability, failure reason, last-success age, and missing-input impact remain visible. |
| `/statements` | Select a period, inspect aggregates, and export JSON. | Statement RPC and adapter availability. | Partial coverage is named beside each figure; the breakdown scrolls inside its container. |
| `/settings` | Change locale/theme and review goal presentation. | Local settings and goal metadata. | Controls remain stacked and keyboard reachable. |

Desktop uses the sidebar and multi-column cards; tablet keeps the sidebar only
where space permits; mobile uses safe-area-aware bottom navigation and
single-column cards. The page body must never acquire horizontal overflow.

## Accessibility contract

Forms use explicit `label`/`id` associations, errors use alert semantics,
successful asynchronous outcomes use live regions, and focus indicators use
the shared visible focus ring. Buttons, navigation items, and form controls
provide at least a 44px target at mobile widths. Summaries have text/table
equivalents; no chart is the only representation of a value. The runway/burn
trend uses the shared Cartesian chart adoption, with a visible period table,
source/basis note, selectable window, and explicit gap state; it never
interpolates unavailable intervals.

See [`../../experience/index.json`](../../experience/index.json),
`ui/src/consts/selectors.ts`, and
[`../reference/cli-commands.md`](../reference/cli-commands.md).
