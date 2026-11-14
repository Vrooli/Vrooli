# UI Architecture

## Purpose Of This Document

Describe the canonical layout of the `ui/` source tree for scenarios generated
from the `react-vite` template, and the **slot taxonomy** that lets external
tools (notably `react-component-library`'s adoption resolver) place components
without asking the user for a path.

## Source Layout

```
ui/src/
├── api/            # api-client slot — Connect-RPC wrappers
├── app/            # app-bootstrap — Providers composition and route table
├── components/     # shared-component slot — cross-cutting components
│   └── ui/         # ui-primitive slot — headless primitives (kebab-case files)
├── consts/         # consts slot — strings + selectors registries
├── features/       # feature slot — per-feature folders (one subfolder per feature)
│   └── <feature>/  # feature-component slot — components inside a feature
├── hooks/          # hook slot — reusable React hooks
├── i18n/           # i18n bootstrap
│   ├── catalog.ts  # assembles each locale's files into one typed catalog
│   └── locales/    # i18n-strings slot — <locale>/<namespace>.json, one file per catalog subtree
├── layout/         # layout-shell + layout-nav slots — AppShell, Sidebar, TopBar, BottomNav
├── lib/            # lib-util slot — framework-agnostic utilities
├── pages/          # page slot — routed pages mounted under <Outlet />
├── test-utils/     # test-util slot — render helpers, factories, a11y
└── theme/          # theme-token slot — ThemeProvider + tokens.css
```

## Slots Are A Contract

Every directory above maps to a named slot in `ui/manifest.json`. The manifest
declares the directory **and** a default path pattern (e.g.
`{dir}/{ComponentName}.tsx`), so external tools can compute the canonical
filesystem path for a new file given just the component's name and slot.

A component library that publishes `"slot": "layout-nav"` and ships
`SidebarShell` knows — without any per-scenario configuration — that the file
should land at `ui/src/layout/SidebarShell.tsx`. Override the slot's `dir` in
a scenario-level overlay if you've reorganized; the resolver merges that
overlay before computing the new path.

## Component Canon

Generated scenarios start with a small adopted-provenance canon under
`ui/src/components/ui/`: button, card, data table, empty state, input, select,
status badge, sidebar shell, and bottom navigation. Each file carries a
`@vrooliComponent*` JSDoc block so ui-health and react-component-library can
classify the surface as governed rather than unknown local code.

When adding a shared component, search and adopt from the registry first:

```bash
react-component-library components list --json
react-component-library adoptions resolve-path <component-id> notification-hub
react-component-library adoptions apply <component-id> notification-hub <adopted-path>
```

Use scenario-local custom components for genuinely scenario-specific surfaces,
not for generic tables, buttons, navigation shells, form controls, or status
badges that the canon already provides.

## Adoption Resolver Flow

1. Library declares the component's slot (e.g. `"slot": "layout-nav"`).
2. Resolver looks up the slot in this scenario's UI manifest (this file's JSON
   sibling).
3. Resolver substitutes path-pattern tokens (`{dir}`, `{ComponentName}`,
   `{kebab-name}`, `{camelName}`, `{feature}`, `{locale}`) and returns the path.
4. Scenarios with no manifest fall through to a heuristic (scan for the slot's
   expected dir name) and then a final fallback
   (`ui/src/components/<ComponentName>.tsx`). Both flag warnings on the
   resulting adoption record.

## Shell And Shared Surfaces

The shell and the pages around the decision page follow
[UX_DIRECTION.md](UX_DIRECTION.md). Shared building blocks live in
`components/`, not `components/ui/` (that folder holds adopted library
copies): `Page` (the `h1`, description, actions and reading measure of every
routed page), `BackLink`, `ListGroup`/`ListItem`/`ListRow` (grouped, navigable
lists with an urgency bar), `SectionTabs` (route-linked segmented switch),
`FilterChips`, `Skeleton`, `QueryNotice` (`QueryError`, `EmptyNotice`,
`InlineNote`), `Time` (relative time with the exact time in the tooltip) and
`Facts`. Rules:

- The shell (`layout/`) knows nothing about routes. A page draws its own title
  and back link; there is no header extension point.
- Four destinations (Needs you, Updates, History, Settings). Delivery is the
  second tab of Updates, so `isCurrent` in `layout/navItems.tsx` keeps Updates
  current on `/delivery` and `/delivery/:id`.
- The Needs-you badge and the installed app's badge come from `useNeedsYou`
  (the Hub's exact `total_awaiting` from a one-row read plus pending permission
  requests, refreshed at the inbox cadence). Both decision lists
  share `RequestList` (`features/asks/HistoryPage.tsx`): the view (sort, urgency
  and sender filters) is parsed from and written to the address by
  `features/asks/askListView.ts`, sent to the Hub on every read, and rendered by
  `ListControls` with the facet counts the Hub returns. The queue defaults to
  `ORDER_ATTENTION`, History to the server's newest-first. Tests that script `ListAsks` page by page use `asksInOrder` /
  `listAsksFor` from `test-utils` so that count read does not consume the script.
- Catalogs are split by owner: `i18n/locales/<locale>/<namespace>.json`, where the
  file name is the dotted key path of the subtree it holds (`pages.dashboard.json`
  is `pages.dashboard.*`). Edit only your own files; a new namespace needs a file
  in every locale directory (the parity test names a missing one). Tests read
  catalogs through `i18n/catalog` (`en`, `ja`, `ar`), not the JSON.
- Every list renders loading (`SkeletonRows`), failure (`QueryError`, which shows
  the sign-in card when the cause is a missing session), empty (`EmptyNotice`)
  and a one-line footnote about its scope; none writes its own.
- Times are relative in lists and exact in `title` and `dateTime`
  (`lib/ago.ts`); deadlines read "Due in 2 hours".

## Decision Page (`features/packet`)

The decision page (`/asks/:askId`) is one feature folder. The product contract
is [DECISION_PACKETS.md](DECISION_PACKETS.md); this section is only the map.

```
features/packet/
├── PacketPage.tsx        # route element: loads the ask, composes everything below
├── PacketCover.tsx       # question, big picture, recommendation, technical details
├── DecisionText.tsx      # prose with safe links; decisionLinks.tsx holds the pure parsing
├── model/                # pure TypeScript, no React
│   ├── questions.ts      # question kinds (unknown type -> text), limits
│   ├── draft.ts          # the unsent answer (options, responses, notes on files)
│   ├── draftStore.ts     # per-owner, per-ask browser copy + the frozen attempt
│   └── answer.ts         # draft -> AnswerPayload, required rule, pre-fill, explainer pre-fill
├── form/                 # the Hub's own answer form (the only thing that answers)
│   ├── AnswerPanel.tsx   # stacked on wide screens, guided steps + review on a phone
│   ├── MainOptions.tsx, QuestionField.tsx, ChoiceQuestion.tsx, VariantQuestion.tsx,
│   │   SectionReviewQuestion.tsx, PinnedNotes.tsx, ReviewSummary.tsx
│   └── ActionBar.tsx, SecondaryActions.tsx
├── viewer/               # attachments: tiles, path matching, the manifest, the viewer host
├── explainer/            # the sealed HTML page: document builder, bridge, frame, kit/
├── context/              # "I need more context": the sheet and the waiting/updated notices
├── status/               # decision status and what happened to the answer
├── useAnswerFlow.ts      # send once, freeze the attempt, only ever read back
├── usePacketDraft.ts, useNextDecision.ts, useDecisionLinks.ts
└── mocks/                # test factories (never imported by production code)
```

Rules the folder keeps, each covered by tests:

- **Nothing a packet part does can stop an answer.** Attachments, the explainer
  and typed questions all degrade (a tile that fails, a frame that shows a
  fallback, an unknown question type drawn as text); "Other" and Send are
  always reachable, and a draft that no longer matches a revised ask is
  filtered when it is built, never sent as invalid.
- **One answer, sent once.** `useAnswerFlow` freezes the payload as an attempt
  before it leaves; after any uncertain response the page only reads the
  decision back.
- **The explainer is untrusted.** `explainer/document.ts` builds the `srcdoc`
  (policy first, kit, bridge); `explainer/bridge.ts` validates every message
  from the frame; a pre-fill only edits the draft.
- **The sticky action bar is a direct child of the tall page content** (a sticky
  element cannot leave its parent), so the aside holding the form is
  `display: contents` below the wide layout.
- **Opacity modifiers work on every theme color.** The theme colors are CSS
  variables holding hex values, so `tailwind.config.ts` wraps each in an opacity
  function that emits `color-mix(in srgb, var(--color-x) NN%, transparent)`.
  Write `bg-app-warning/15`, not a per-component tint class.
- **Files open in the shared viewer, and the Hub only hosts it.**
  `viewer/AttachmentViewer.tsx` gives `@vrooli/file-viewer` a manifest source
  (`viewer/manifest.ts`: the ask's stored copies on the content route, read
  through the session fetch so a 401 reaches the sign-in path) and fills its
  footer slot with what is specific to a decision: the "changed since" warning,
  previous/next between attachments, and "Comment here". The viewer owns
  rendering, folders, anchors and its own close control. It is loaded lazily
  (its PDF, 3D and document engines stay out of the main chunk), and
  `theme/viewerTokens.css` maps the Hub palette onto its `--fv-*` channel
  triplets (`viewerTokens.test.ts` fails if a triplet drifts from the palette).
  Spreadsheets and Mermaid need optional engines the Hub does not install, so
  they degrade to their download link.
- **Generated proto types are read from source.** `vite.config.ts` and
  `tsconfig.json` alias `@vrooli/proto-types` to `packages/proto/gen/typescript`
  (and `@bufbuild/protobuf` to this app's copy, because the source sits outside
  the package). The installed `file:` copy of the proto package goes stale
  whenever the proto is regenerated, which made a new field type-check in one
  consumer and be missing at runtime in another. Vitest, `tsc` and the build all
  read the same generated files, so they cannot drift. The `package.json`
  entry stays (removing it is a dependency change).
- **Tests run in a jsdom environment with Node's `AbortController`**
  (`src/test-environment.ts`), because Node's `Request` rejects jsdom's signal
  and every React Router data navigation builds one.

## Experience Contract

The generated `experience/` folder is the UX-intent contract for the route
table in `ui/src/app/routeTable.tsx`. Keep those two surfaces aligned:

- add an L0 page spec when a new user-facing route is added;
- remove or deprecate a page spec when a route is removed;
- promote a page from L0 by adding priorities, elements, claims, bindings, and
  states before calling the route production-ready;
- keep `data-testid` selectors in code aligned with
  `experience/pages/*.json::bindings` once bindings exist.

Run `experience-manager spec validate notification-hub --json` after route or
selector changes. The generated example page spec is scaffold content and
is removed by `template-manager detemplate notification-hub`.

## Extending The Manifest

- **Add a slot.** Add an entry to `ui/manifest.json`. Keep its `dir` inside
  `ui/src/` and pick a pattern that matches your file-naming convention. The
  schema (`scenario-ui-manifest/v2`) does not enum-restrict slot names — open
  set on purpose.
- **Override a slot in a single scenario.** Drop a partial manifest at
  `.vrooli/ui-manifest.json` in the scenario root; the resolver merges it over
  the template manifest. The overlay may override existing slots, but it may
  not invent new slot names.
- **Add a `postApply` action** (auto barrel-export, route-register,
  i18n-merge). Reserve this for a future schema revision.
  Document the intent in the consuming scenario's PRD until then.

## Cross-References

- Schema: `.vrooli/schemas/scenario-ui-manifest.schema.json` (`$id:
  scenario-ui-manifest/v1`)
- Manifest: `ui/manifest.json`
- Slot reference: [`ui-manifest.md`](../reference/ui-manifest.md)
- Adoption resolver: `scenarios/react-component-library/api/internal/adoptions/pathresolver.go`
