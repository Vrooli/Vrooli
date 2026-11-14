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
  the API marks the time `at_least`), `Unreadable` (files that exist but cannot be read, shown
  as unknown rather than empty; `UnparsedHeadings` lists a heading the API could not parse
  with its line), `EntryField` (operator words counted against the entry
  limit the API sends with the page's data), `CourierHealth` (the courier lens from the overview), `Markdown` (a small
  local renderer, with `InlineMarkdown` for one line such as a workstream title; see `docs/reference/component-library-gaps.md`), `HtmlOpen` (review
  HTML opened in its own tab, since the file route refuses framing), `WouldWrite` (the
  exact entry a write's dry run returns and when it is read; the UI never rebuilds the
  file grammar), `ResearchWrite` (a topic's last research write with its source and note),
  `Changes` (what changed, worded by the API's baseline kind), `RouteError` (a page that fails
  to render shows its error inside the shell with the tabs still working; the router catches
  render errors before the app-level `ErrorBoundary`).
- `lib/`: pure helpers (`format`, `errorMessage`, `spec` for the launch form and its
  session-storage copy, `choices` for the blocker choices with the desk's recommendation
  selected, `paths`, `states` for state words, which states have a window open and which
  packets launch as they are, `entry` for counting an entry against the API's limit, `needsYou` for row anchors, the waiting rows and their count by the API's row kind (item or older list),
  `overlaps`, `topics` for the filter and sort, `period` for the Since periods, `useDebounced`).
- `styles.css`: the `rd-*` document rules, built only from design tokens.

Reads poll (30 s, 2 s while a launch runs) and keep the last good read on screen when a
refresh fails. Writes are direct client calls inside `Confirm`, which then invalidate the reads they
change; nothing is optimistic. A confirmation's exact text comes from the same RPC with
`dry_run` set, which the API never writes. The launch form lives in `sessionStorage`
(this tab only); blocker choices travel in the Preflight, Build and Launch requests, never
in the spec. Research-home files are fetched from the API's read-only file route on the
page's own origin, so the desk works unchanged through the tunnel hostname.
