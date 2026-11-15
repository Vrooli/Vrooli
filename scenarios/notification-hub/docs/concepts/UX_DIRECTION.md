# Hub UX direction

The current design target for the Chimesail web UI (`ui/`). It covers the
whole app: shell, navigation, inbox, updates, delivery, history, settings and
the states between them. The decision page keeps its own contract in
[DECISION_PACKETS.md](DECISION_PACKETS.md); this document decides how the rest
of the app frames it. The visual proposal is
[`../mockups/hub-ux/index.html`](../mockups/hub-ux/index.html) (open it in a
browser; it follows the system theme and has a light/dark toggle).

Status: **adopted by the owning implementer, agent-authored.** Nothing here
has been ratified by the operator; the amendments table at the end records
that provenance. Experiential maturity: the shell, Needs you, Updates,
Delivery, Delivery record and History are built to this direction (L2: the
phone and desktop shells and the Delivery page are composed separately, lists
are reflowed by design). Settings and Reach follow in a second pass.

## 1. Who this is for, and the one job

One operator, mostly on an installed iPhone PWA reached through a tunnel,
sometimes on a desktop. The Hub is where agents ask them for decisions and tell
them things. Every visit is one of four intents, in this order of frequency:

1. **What needs me?** Open the app, see the decisions waiting, pick one.
2. **Decide.** Read, answer, move to the next (the packet page owns this).
3. **What happened?** Glance at updates and past decisions.
4. **Tune how I am reached.** Devices, quiet hours, escalation (rare).

The current UI serves intent 4 as loudly as intent 1. The inbox opens on a
search box and a security disclaimer; the decisions are below the fold.
The redesign makes the app answer "what needs me" before anything else and
makes everything else quieter.

## 2. Audit: what is wrong today

Captured from a build of the source at 390x844 and 1280x800, light, dark and
Arabic, against fixture data (scratchpad; the live Hub was never written to).

| Area | Finding |
|---|---|
| First view (phone) | Inbox opens on title, search label, search box, "Search all request statuses", then a 300px permission-explainer card. The first decision starts at about y=670 of 844. With no permission request it still starts at about y=250, after a search field the operator rarely needs. |
| Information scent | No count anywhere. A nav tab named "Needs you" never says how many. Rows carry no urgency or deadline even though `Ask.urgency`, `deadline` and `state=escalated` exist. |
| Navigation | Five peer tabs, plus a hamburger menu that repeats the same five destinations. Delivery (a diagnostic) has the same weight as the inbox. Desktop: a 360px sidebar of five centered text links with no icons or badges, which leaves a wide, empty column; the content then floats at a different width on every page (`max-w-3xl`, `max-w-4xl`, none). |
| Shell | Shell code special-cases routes (`reach`, `delivery`) to swap headers, and `styles.css` carries ~250 lines of overrides of the library shell (hard-coded `#e4edff`, hidden icons, `!important`). A second header on drill-in pages reads "<- Delivery  Delivery". |
| Hierarchy | Every page is `h2` plus bordered boxes of equal weight. State is a tiny gray chip. A row has up to nine pieces of meta (state, source, absolute time with time zone, counts) at the same size. |
| Copy | Internal vocabulary leaks: "configured recipient", "transport evidence", "returned for the configured recipient", "returned asks; older matches may be omitted", "This does not indicate producer health", "Request's scope". Scope disclaimers repeat on every list. Dates show `Nov 13, 2026, 12:08 PM EST` everywhere. |
| Updates | Each item is a card with a five-row definition list (source, created, urgency, sensitivity) and a prose state sentence. The API Health card is appended at the bottom of a notification feed. |
| History | Seven status filters wrap to three lines and take a third of the first screen. |
| Delivery | The page opens with a glossary of seven states. Analytics are fetched but never drawn; the page shows only their error. The detail page is key-value dumps whose header is the raw notification id. |
| Signed out | Every list says "couldn't be loaded" with a link to Settings; none offers sign-in. Only the decision page does. |
| States | Loading is the text "Loading..." in four different places; errors are bordered boxes with underlined links; empty states differ per page. |
| Touch and safe areas | Mostly fine (44px targets, dvh shell). Underlined text links as the only action on a row. Filter chips are 44px tall but crowd. |
| RTL | Mirrors correctly in the shell. The chevron is the only direction-aware glyph on rows; the "<-" back arrows are hard-coded text swapped by reading `document.documentElement.dir`. |

## 3. Concept

**The Hub is a desk, not a dashboard.** One pile that needs you, then a quiet
log. The unifying idea, applied in order, is *attention ladder*:

1. Things that need you are large, warm and countable.
2. Things that happened are small, calm and chronological.
3. Things that tune the Hub are reachable but never in the way.

It is structural, not decorative: it sets the order of destinations, how loud
a row is, how much text a row spends, and what the badge says. There is no
illustration or ornament. The one accent is the signal amber the packet mockup
already uses for "recommended", reused for "due soon"; the primary blue marks
action and the current place; everything else is neutral.

## 4. Information architecture

Model the content once, compose it for each medium.

| Destination | Route | Holds | Rank |
|---|---|---|---|
| **Needs you** (inbox) | `/` and `/asks` | Open decisions (pending + escalated), by default most urgent first and grouped under headings that say why; pending native permission requests as a compact section above them; sort and filters; search (on demand) | 1 |
| **Updates** | `/updates` | What agents told you, newest first, each with whether it reached you and a link to the decision if it carried one | 2 |
| **Delivery** | `/delivery`, `/delivery/:id` | Did it reach me: recent records with state, per-channel reliability, Hub status, the state glossary. A sibling of Updates, not a top-level tab. | 3 |
| **History** | `/history` | Every decision with its outcome, searchable, filterable by status, urgency and sender, sortable | 3 |
| **Settings** | `/settings`, `/settings/reach` | Appearance, language, session, devices, quiet hours, escalation, event sources | 4 |
| Decision page | `/asks/:id` | The packet (owned by `features/packet`) | - |

Decisions behind the table:

- **Four destinations, not five.** Delivery moves under Updates as a second
  tab of a shared segmented switcher (`Updates | Delivery`). Both answer "what
  did the Hub send, and did it land". `/delivery` and `/delivery/:id` keep
  working; the nav item "Updates" stays current on all of them. Nothing is
  removed.
- **History keeps its own tab** because "what did I decide, and what became of
  it" is a distinct, frequent intent and the packet page links back to it.
- **Settings is last** and holds the account and the quiet-hours/reach
  surfaces. On a phone its tab is the fourth and rightmost, within thumb reach.
- **Health leaves the feed.** The API health card moves to Delivery as the
  "Hub status" strip. It is a delivery-health fact, not an update.

### The count

The Needs-you tab carries a count badge: pending and escalated decisions plus
pending native permission requests, refreshed every 60 s. The decision count is
the Hub's exact `ListAsksResponse.total_awaiting`, which every page carries, so
the badge is one `ListAsks(limit=1, stateFilter=awaiting)` read plus
`ListNativeConsents`, with no cap and no `50+`. The same number is written to
the installed app's badge with `navigator.setAppBadge` whenever the app is
foregrounded, so the home-screen icon agrees with the tab; the call is guarded
and never throws or logs where the Badging API is missing or refused. While the
app is closed, the Hub's web push carries `"awaiting": N` and the service worker
sets or clears the badge from it.

### Order, sort and filters

The owner reported that Needs you looked randomly ordered and could not be sorted
or filtered. The earlier attention order ignored urgency, so every high-urgency
decision, with a deadline days away, sorted below normal ones due in a day.

- **Default order.** Needs you opens on `ORDER_ATTENTION`, named "Needs
  attention first": escalated, then asks closing within 6 hours (the nearest
  close first), then urgency (critical, high, normal, low), then the earliest
  deadline (none last), then newest. The operator chose the 6-hour band as
  "closing-soon" on 2026-11-14, after the list still read as unexplained. The
  list says so in one plain line under the sort menu ("Needs attention:
  escalated, closing within 6 hours, then urgency, then close time"; each other
  sort has its own line), and groups under headings (Escalated, Closing within
  6 hours, Critical, High, Normal, Low) so the order explains itself; a list
  that is one group has no heading. A row in the band shows "Closes in 3 hours"
  with a clock, in place of "Due ...". The Hub names the window and marks the
  rows, so the app holds no copy of the number. History opens newest first.
- **Sort.** One menu, labelled Sort, on both lists: Needs attention first, Due
  soonest, Newest first, Oldest first. Every order accepts every status.
- **Filters.** A Filters button opens a sheet (a dialog on a wide screen) of
  urgency and sender chips, each with the number of decisions it would show. The
  counts come from the Hub in the same read (`ListAsksResponse.facets`) and apply
  every filter but their own, so choosing High leaves the urgency counts alone and
  narrows the senders. Choosing applies at once and the sheet stays open; the
  button reads "Show N decisions". Filters in force are chips under the sort menu,
  each removable in one tap, and the header reads "N of M waiting".
- **In the address.** `sort`, `urgency` (repeatable) and `sender` (repeatable)
  are query parameters beside `q` and `state`, so a reload, Back and a shared
  link keep the view. The default sort is left out.
- **Sender.** Rows and the sender filter name who is asking, not the raw source.
  The Hub derives it from who it verified: a service it sent for ("Research
  Desk", the name the Sources screen shows), else the label the `ask` command
  found for its own session, else "Agent run", else "Unnamed sender". An agent
  normally passes nothing; `ask --from "Treasury steward"` is an optional override
  for a name that reads better. It is display text only and never grants access.
- **No due filter.** Due soonest is a sort; a "due within" filter would answer
  the same question twice and add a second facet to every request.

## 5. Navigation model

### Phone: sequence

- A **four-tab bar** pinned to the bottom (thumb zone), icon over label, count
  badge on Needs you, safe-area padding below. No hamburger: the tabs are the
  whole menu, so the shell stops using the `menu-tabs` mode.
- **No brand bar.** The page title is the first thing on screen and carries the
  page identity (a large title, `PageHeader`). The brand lives in Settings and
  on the sign-in card. This returns about 56px to the content and removes a
  bar that says "Chimesail" on every screen.
- **Drill-in pages** (decision, delivery record, reach) show a back link as
  the `PageHeader` leading slot, not a second shell header. Back goes to where
  the operator came from (`location.state.from`, already the contract for the
  decision page) and falls back to the parent list.
- Lists scroll; the tab bar and (on the decision page) its action bar stay.
  The scroll position and focus of a list are restored on return (existing
  behaviour, kept).

### Desktop: space

- A **left sidebar** at a readable default width (15rem), icon + label, count
  badge, current item filled. Footer: the signed-in owner, or the way to sign in
  (the theme choice stays in Settings).
  The library's drag-to-resize stays, but the default drops from 360px to 15rem
  (bounded 12.5-20rem) under a new storage key, so an old stored width cannot
  bring the wide column back.
- One **content measure per page type**: lists 44rem, record and form pages
  44rem, delivery overview 64rem (two columns). The decision page keeps its
  wide two-column layout.
- The same four destinations in the same order. Keyboard: `g` then `i/u/h/s`
  is a candidate for later; not part of this pass.

The shell has **no per-route knowledge**. Pages declare their own header
(`PageHeader` with `leading` for back). There is no shell extension point:
editors are routes with their own back link, so the former quiet-editor header
context is gone.

## 6. Page by page

### Needs you (`/`, `/asks`)

Intent: pick the next decision. Archetype: list / feed.

- Title **Needs you**, with the count as subtitle ("3 decisions waiting", or
  "All caught up"). Header action: search icon button (opens the field; the
  field is shown whenever `?q=` is set). No search label text.
- If native permission requests are pending, a **compact section above the
  list**: one row per request (title, "Review in Agent Manager" as the row
  action, expiry as a relative time). The authority disclaimer collapses to a
  single quiet line under the section title; the long copy moves to an
  expandable "Why can't I approve here?" so it does not take the fold. The
  rule it protects is unchanged (this component has no mutation client).
- **Decision rows** (see section 7): title, source, time ago, the due-in chip
  when there is a deadline, an Escalated chip when escalated, attachment and
  question counts, "Asked for context". A one-line preview of the big picture
  appears from the `md` breakpoint up. Whole row is the link.
- Empty: "All caught up" with the last-checked time and a link to History.
  Signed out: the sign-in card in place of the list. Error: the shared error
  state with retry. Loading: three skeleton rows.
- Footer link: "Search every status" goes to History with the query.

### Updates (`/updates`)

Intent: glance at what the Hub sent. Archetype: list / feed, with the shared
Updates | Delivery switcher under the title.

- A row is: title, two-line body, source, time ago, and a **delivery chip**
  (Delivered, Held, Failed, ...). Chip tone carries the state: success for
  delivered, danger for failed and unroutable, neutral for held and
  suppressed, info for pending and routed. The meaning sentence becomes the
  chip's tooltip and the detail page copy instead of repeating on every row.
- Row action: opens the decision when the update carries one, else the
  delivery record. A secondary text link "Open source" appears when there is a
  context URL. Urgency shows only when it is not normal. Sensitivity shows as
  a small lock chip when present.
- The "showing up to 20" and "matched against up to 200 asks" notices become a
  single footnote at the end of the list. Linked-decision lookup failure
  becomes a quiet one-line note, not two status paragraphs.

### Delivery (`/delivery`)

Intent: did it reach me, and is the Hub healthy. Archetype: dashboard (two
columns on desktop, one sequence on phone).

- Phone order: a one-line **summary** ("36 delivered, 3 failed in the last
  day"), then recent records, then Hub status and per-channel reliability
  collapsed under "Channel reliability", then "What the states mean"
  collapsed.
- Desktop: records on the left (measure 36rem), a right rail holding Hub
  status (the existing health card, restyled as a strip), channel reliability
  (delivered, failed, failure rate, latency per channel, drawn from the
  analytics the page already fetches) and the glossary.
- Analytics failure is an inline notice above the records, not the page
  headline; records stay the first thing.

### Delivery record (`/delivery/:id`)

Intent: see the story of one notification. Title: the notification's title;
the id is a secondary mono line with a copy button. A **vertical timeline**
merges the receipt and attempts (attempt number, channel, outcome, when, next
retry). The state sentence, source, urgency, sensitivity are a compact
description list. The three current branches (unknown status, no receipt,
receipt present) become three variants of one layout, with the same copy
meaning ("A returned receipt records adapter acceptance. It does not establish
device arrival or human reading.").

### History (`/history`)

Intent: find what I decided. Archetype: list / feed with search and filters.

- Search field always visible (it is the archive).
- Status filter is **one scrolling row of chips** (All, Open, Answered,
  Defaulted, Expired, Withdrawn, Dismissed), single line on a phone, wrapping
  on desktop. URL contract `?state=` unchanged.
- Rows share the decision row component; the outcome chip shows the answer
  ("Answered: Approve") and when. Late-answer availability stays a line.

### Settings and Reach

Owned by the second implementer. They inherit: `PageHeader`, the content
measure, the shared state components, tokens and the copy voice below. The
shape the direction asks for: summary rows that drill in on a phone (Appearance
and language, Session, Devices and channels, Quiet hours, Escalation order,
Event sources), a two-pane layout on desktop, and no long scroll wall. See
"Handoff to the settings implementer".

### Sign-in and session

A single **session card** (`features/session`) replaces the bare form: the
Chimesail mark, "Sign in to see what needs you", email, password, error line.
It renders in place of any list that fails with Unauthenticated, and on the
decision page (hub-ui keeps using `SignInForm`; its props and test ids stay).
The sign-out confirmation is the library `AlertDialog/2`, the same confirmation
the settings screens use: a short title, one sentence on what it clears, a note
that other sign-ins (Cloudflare Access, issued tokens) stay separate, and
"Sign out of Hub" beside "Keep session". Focus starts on the safe choice, and a
failed sign-out is said inside the dialog. The earlier full-screen page with a
display-size heading is gone.

## 7. Shared components and tokens

Reuse the library first; the table records the fit check against the shipped
contracts (`dist/components/*/versions/*`).

| Need | Decision |
|---|---|
| Shell | `AppShell` **2.1.4** (same props as 2.1.2; nav badges are supported by both). `mobileNav="tabs"`, `density="sidebar"`, a fixed-width sidebar, `utility` for the account footer. All `styles.css` overrides of shell internals are deleted; the only kept rule is the iOS standalone one that subtracts the top safe-area inset from the shell height. |
| Page title | `PageHeader` 2.0.1 through `components/Page.tsx` (`h1`, description, actions, reading measure 44rem or 64rem). A global rule keeps its actions beside the title on a phone instead of dropping them to a row. `components/BackLink.tsx` is the way back on drill-in pages. |
| Facts | `components/Facts.tsx`: label and value on one grid, one row each. |
| Status | `StatusBadge` 1.2.5 for delivery and decision states (tones neutral, success, warning, danger, info). |
| Empty and error | `components/QueryNotice.tsx`: `QueryError` (sign-in card when the cause is Unauthenticated, else a named failure, "nothing was lost" and a retry), `EmptyNotice`, `InlineNote`. RCL `ErrorState` and `EmptyState` have no signed-out branch and carry their own title scale; the wrappers keep one voice. |
| Segmented switcher | `Tabs` 1.3.6 has a `segmented` variant but selects by callback; the Hub needs route links. Local `components/SectionTabs.tsx` (link-based, `aria-current`). RCL gap: a link-based segmented control. |
| Filter chips | Same gap: RCL `Chip` is a button. Local `components/FilterChips.tsx` (links, scrolling row). |
| Row | Local `components/ListRow.tsx` (`ListGroup`, `ListItem`, `ListRow`): link row with leading, title, meta, trailing, direction-aware chevron, 56px minimum. RCL `List` has no navigable-row contract the inbox needs (selection, return focus). RCL gap, noted. |
| Time | Local `components/Time.tsx`: relative ("12 min ago", locale-aware via `Intl.RelativeTimeFormat`) in text, absolute with zone in `title` and `dateTime`. RCL `RelativeTime` takes no locale or format. `features/delivery/deliveryView.formatDeliveryTime` stays as the absolute formatter. |
| Loading | Skeleton rows, not the word "Loading": `components/Skeleton.tsx` (`SkeletonRows`) over the RCL `Skeleton/1` primitive, announced once. |
| Banners | Not used. A notice is a quiet inline line, a warning row or a `StatusBadge`. |
| Colors, radii, type | The palette in `theme/tokens.css` and `design-tokens.css` does not change (the file viewer's `viewerTokens.test.ts` pins it). New semantic aliases only: `--hub-due` (signal amber) maps to the existing warning. Type steps use the existing `--text-*` tokens: title 24, section 18, row title 16/600, meta 13, chip 12. Radius: rows flat with hairline dividers on phone, `--radius-panel` panels on desktop. |
| Opacity tints | `bg-app-warning/15` and friends via `tailwind.config.ts`; no per-component tint classes. |
| Motion | 120-180 ms `--dur-quick` on press, focus and chip state; nothing animates layout; all of it off under `prefers-reduced-motion` (library shell already does this). |

Rules for every list page: skeleton while pending, one error state with retry
(and sign-in when the cause is Unauthenticated), one empty state, a footnote
for scope, `aria-busy` while refetching, focus returns to the row opened.

## 8. Copy voice

Plain, second person, short, calm. Sentence case. A title says what the page
is for; a subtitle says the state. Never explain the system's internals to the
operator when a fact is enough.

| Instead of | Say |
|---|---|
| Decisions | Needs you |
| Awaiting answer | Waiting for you |
| Information sent to the configured recipient, with delivery facts shown as transport evidence. | What agents told you, and whether it reached you. |
| Showing up to 20 notifications returned for the configured recipient. | Showing the latest 20. |
| Requests couldn't be loaded. Your requests haven't been cleared. | Couldn't load your decisions. Nothing was dismissed. |
| No Hub questions awaiting an answer | All caught up |
| Native permission requests | Permission requests |
| Delivered: A delivery adapter accepted the payload | Delivered. The channel accepted it. |
| Nov 13, 2026, 12:08 PM EST | 12 min ago (with the full time on long press / hover) |

Every string goes through the catalog in en, ja and ar; keys keep their
current names where the meaning is unchanged, so translations are edited, not
re-created.

## 9. Accessibility, device and RTL commitments

- Landmarks: one `main`, one `nav` per shell, one `h1` per page (the
  `PageHeader` title; the current `h2` page titles become `h1`). Skip link
  stays.
- Targets: rows, tabs, chips and icon buttons are at least 44px; the tab bar
  is 56px plus the home-indicator inset.
- Focus: one visible ring, `--color-focus`, on composed controls (a search
  field's ring surrounds the icon and the input).
- Contrast: unchanged palette; chip text is `--color-foreground` on a tint, not
  the tone color on a tint.
- RTL: logical properties throughout (`ps-`, `ms-`, `border-s`), the chevron
  and back arrow are icons that flip with `rtl:rotate-180`; the hard-coded
  arrow characters and the `document.documentElement.dir` reads are removed.
- iOS PWA: the shell keeps its dvh sizing with the status-bar inset removed;
  nothing else positions against `100vh`; bottom bar uses `padding-bottom:
  env(safe-area-inset-bottom)`.
- Long machine identifiers (scenario names, ids, paths) wrap with
  `[overflow-wrap:anywhere]`, never forcing horizontal scroll. Verified with
  the fixture's 60-character scenario name.

## 10. Settings and Reach as built

Settings is a set of real routes under `/settings`, each one screen, so the
back button, a reload and a shared link all land where the reader was.

| Route | Holds |
|---|---|
| `/settings` | The list: six rows in three groups (Reaching you: devices, quiet hours, escalation; This app: appearance and language, session; Advanced: event sources), each with a one-line plain-language state |
| `/settings/reach`, `/settings/reach/devices/:id`, `.../channels/:channel` | This browser's push state and the one action that fixes it; devices; one device's channels; the email address and content approvals of one channel; apps per device |
| `/settings/quiet-hours` | Schedules (days, hours, time zone, critical allowed) edited in a sheet |
| `/settings/escalation` | The channel order, reordered with touch-sized controls |
| `/settings/appearance`, `/settings/session`, `/settings/sources` | Theme and language; who is signed in; which apps may notify and their lowest urgency |

A phone shows the list as a page and each setting over it with a back link.
From 68rem up the list stays at 17rem on the left and the open setting at the
44rem measure on the right; `/settings` opens the first setting there. Owner
screens render the sign-in card in place when signed out; if the session ends
mid-edit the draft stays mounted and hidden and returns for the same owner only.
Every save is confirmed from a fresh read, not from the write's answer; a
partial save names the days or channel that did not land and keeps the draft.
Anything that widens what leaves the Hub (a new destination, added content
approvals) is confirmed in plain words first; removals ask first too.

Quiet hours may cross midnight. Hours that end earlier than they start are
overnight: they start on each chosen day and end the next morning, and the
editor says so beside the end time. Only equal start and end are refused.

## 11. API deltas

Landed (Hub, 2026-11-14), and used by the UI:

- `ListAsksResponse.total_awaiting`: exact count of pending + escalated asks the
  reader may see, set on every page, ignoring query, filter and cursor. Native
  permission requests are not counted; the UI adds them.
- `ListAsksRequest.order`: `ORDER_ATTENTION` (escalated, then closing within the
  window, then urgency, then earliest deadline with none last, then newest; the
  window is measured at the first page's time, carried in the cursor), `ORDER_DUE`, `ORDER_NEWEST`
  and `ORDER_OLDEST`, each valid with any state filter. A cursor is valid only
  for the order, filters and search it came from.
- `ListAsksRequest.urgencies` and `senders` (server-side filters), and
  `ListAsksResponse.facets` (urgency and sender counts that ignore their own
  filter, on every page).
- `Ask.sender` and `AskRequest.sender` (declared "from" label, unverified).
- The Hub's web push JSON carries `"awaiting": N` (0 clears the badge); the
  service worker calls `setAppBadge` / `clearAppBadge`, guarded for browsers
  without them.

Done on the Hub for the settings screens: overnight quiet hours. `SetQuietWindow`
accepts `end < start` (equal times are still refused); `inQuietWindow` matches the
start weekday from `start` and the next weekday until `end`, wrapping Saturday
night into Sunday morning. No proto change.

## 12. Decisions and amendments

IDs are descriptive and author-scoped. Disposition is the owner's, not the
operator's.

| ID | Source, date | Surfaces | Change and what is kept | Why and tradeoff | Evidence | Disposition |
|---|---|---|---|---|---|---|
| `hubux-four-tabs` | agent (ux-owner), 2026-11-14 | shell, delivery, updates | Delivery moves under Updates as a switcher; four tabs. Every route and capability stays. | Diagnostics out of the primary nav; cost: one more tap to Delivery. | mockup, nav tests | adopted |
| `hubux-no-hamburger-no-brand-bar` | agent, 2026-11-14 | shell | Phone shell loses the brand bar and the menu that duplicated the tabs. | Returns ~56px, one nav not two; cost: the brand mark leaves the phone chrome (it stays in Settings and sign-in). | before/after captures | adopted |
| `hubux-count-badge` | agent, 2026-11-14 | shell, inbox | Exact count of waiting decisions on the tab and app icon, from the Hub's `total_awaiting`; the queue is in attention order. | The tab finally says how many and the list leads with what is most pressing; cost: one more one-row read per minute. | tests, captures | adopted |
| `hubux-sort-filter` | agent, 2026-11-14 | inbox, history, API | Visible sort and server-side urgency and sender filters with facet counts; attention order now ranks urgency before deadline and groups under headings; rows name the sender, not the raw source. | The owner could not tell why decisions were in the order shown or narrow them; cost: a sender field agents should set, and two facet queries per read (a few ms at this size). | Go and Vitest tests, captures | adopted |
| `hubux-closing-soon` | operator, 2026-11-14 | inbox, API | Attention order puts pending asks closing within 6 hours ahead of urgency (nearest close first), measured at the first page's time carried in the cursor; one plain line says the order; rows in the band read "Closes in ...". | An ask that will default before the next look cannot wait behind ones that merely rank higher; cost: a low-urgency ask due in an hour outranks a critical one due tomorrow, by design. | tests, live read | adopted |
| `hubux-inbox-first-fold` | agent, 2026-11-14 | inbox | Search collapses to an action, disclaimer collapses, rows show urgency. | The first decision sits in the first fold; cost: search takes one tap on the queue (History keeps it open). | before/after captures | adopted |
| `hubux-health-to-delivery` | agent, 2026-11-14 | updates, delivery | API Health leaves Updates and becomes Hub status on Delivery. | Health is a delivery fact; no capability removed. | tests | adopted |
| `hubux-analytics-shown` | agent, 2026-11-14 | delivery | The analytics already fetched are drawn as channel reliability. | A fetched value that was never shown. | captures | adopted |
| `hubux-signin-in-place` | agent, 2026-11-14 | all lists | Unauthenticated list errors show the sign-in card. | The cause is named and fixable in place; cost: none. | tests | adopted |
| `settings-routed-screens` | agent (settings-owner), 2026-11-14 | settings, reach | One long page and the quiet-editor header override become a list plus one route per setting; two panes from 68rem. Every setting, device, channel and capability stays. | Phone drill-in with no scroll wall; deep links and back work; the list doubles as navigation on desktop. Cost: more routes. | tests, captures | adopted |
| `settings-quiet-schedules` | agent (settings-owner), 2026-11-14 | quiet hours | Seven per-day windows become schedules (days, hours, zone, critical) with a sheet editor; per-day storage is unchanged and days sharing settings are grouped. Per-day review, staged removal and result screens are replaced by a read-back that names unsaved days. | Setting weekday hours is one edit, not five; time zone is a list, not free text. Cost: cannot show one day with a different zone inside a schedule (it becomes its own schedule). | tests, captures | adopted |
| `settings-escalation-reorder` | agent (settings-owner), 2026-11-14 | escalation | Edit order, add channel and clear review screens become one list with move/remove buttons and a sticky save bar. A saved channel no device offers is flagged and kept, no longer blocking save. | Faster; unrelated reorders are not held hostage by one stale entry. Cost: none removed silently. | tests | adopted |
| `settings-reach-confirm-on-widen` | agent (settings-owner), 2026-11-14 | reach | The mandatory review step for every channel change becomes one confirmation when a save widens exposure (new destination, added approvals); narrowing saves directly. Enrollment keeps an explanation and confirm before the browser prompt. | Same safety where it matters, fewer taps elsewhere. | tests | adopted |
| `settings-quiet-overnight` | agent (settings-owner), 2026-11-14 | quiet hours, Hub | Hours may end earlier than they start (overnight); the Hub reads them from the start weekday into the next morning, and the editor says so. Equal times stay refused. | A night shift is the common case; the old rule made people split it by hand. Cost: overlapping days now share a morning, so critical passes only when every active window allows it. | Go table tests, tests | adopted |
| `settings-integration-lock-removed` | agent (settings-owner), 2026-11-14 | settings | The permanently locked "Event integration" card is removed along with its unused client. | It offered nothing and read like an error. | tests | adopted |

Implementation status and the before/after capture index are recorded in the
engineering hand-off for this change, not here, so this file does not rot.
