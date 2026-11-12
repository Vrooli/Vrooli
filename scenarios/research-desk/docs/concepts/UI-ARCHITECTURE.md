# UI Architecture

## Shared contract

Read the [UI Architecture shared guide](/scenarios/template-manager/docs/concepts/UI-ARCHITECTURE.md). Template Manager owns
the common contract and its implementation references.

## Scenario details

`ui/src/` is organised by role, not by feature:

- `api/`: one small client module per API domain (`activity.ts`, `launch.ts`,
  `operator.ts`, `desk.ts`) over one shared Connect transport (`client.ts`). Every RPC the
  UI calls is named in exactly one of these files, so renaming or moving an RPC between
  domains costs one edit. `api/requests.test.ts` asserts the exact wire request for every
  write.
- `pages/`: one file per route (`app/routes.tsx`); the commission and launch pages keep
  their panels in `pages/commission/` and `pages/launch/`.
- `components/`: shared pieces: `QueryView` (loading, failed, stale, loaded), `Confirm`
  (every outward action, showing the exact text), `Fact` (a value with its evidence on
  tap, or `unknown` with the reason), `When`/`DerivedWhen` (relative time; "at least" when
  the API's walk was cut short), `Unreadable` (files that exist but cannot be read, shown
  as unknown rather than empty), `EntryField` (operator words with the 4,000-character
  entry count), `CourierHealth` (the courier lens from the overview), `Markdown` (a small
  local renderer; see `docs/reference/component-library-gaps.md`), `HtmlOpen` (review
  HTML opened in its own tab, since the file route refuses framing), `WouldWrite` (the
  exact entry a write's dry run returns and when it is read; the UI never rebuilds the
  file grammar), `ResearchWrite` (a topic's last research write with its source and note),
  `Changes` (what changed, worded by the API's baseline kind).
- `lib/`: pure helpers (`format`, `errorMessage`, `spec` for the launch form and its
  session-storage copy, `choices` for the blocker choices with the desk's recommendation
  selected, `paths`, `states` for state words, which states have a window open and which
  packets launch as they are, `entry` for the entry limit, `needsYou` for item anchors,
  `overlaps`, `topics` for the filter and sort, `period` for the Since periods, `words`
  for dropping quoted commands from API prose, `useDebounced`).
- `styles.css`: the `rd-*` document rules, built only from design tokens.

Reads poll (30 s, 2 s while a launch runs) and keep the last good read on screen when a
refresh fails. Writes are direct client calls inside `Confirm`, which then invalidate the reads they
change; nothing is optimistic. A confirmation's exact text comes from the same RPC with
`dry_run` set, which the API never writes. The launch form lives in `sessionStorage`
(this tab only); blocker choices travel in the Preflight, Build and Launch requests, never
in the spec. Research-home files are fetched from the API's read-only file route on the
page's own origin, so the desk works unchanged through the tunnel hostname.
