# Long-session performance audit — 2026-10-15–16

The real heartbeat investigation is lighter to navigate after bounded collapsed
previews, asynchronous virtual-row measurement, and less repeated search/action
work. Desktop MessagesPane commits averaged **3.1 → 2.4ms**. Mobile long tasks
fell from **1,186ms to 581ms and 612ms** in two final captures. The separate
cold-load Lighthouse gate remains red; this is not a claim of complete performance
certification.

## Scope and method

Bounded W3 implementation repair, not a contract/readiness audit or a phased
plan. Prior art: [September audit](2026-09-16-messages-and-markdown-journeys.md).
Performance Health owns trace analysis; its BAS capture engine drove the journeys.
All comparison captures used production React profiling builds, not Vite dev.

Real session: `af38f804-b18f-4fcd-b910-49bd44bed50b`, **Investigate prompt-manager
heartbeat authentication issues**, **574 events**, largest source **35,135 chars**.
No measured transcript was fabricated. The regression tests use synthetic source
only to protect bounds and complete-message behavior.

Operator boundary: do not create sessions, send messages/input, archive, remove,
or delete conversations. Navigation, scrolling, reader/search controls and unsent
mobile draft typing were authorized. Viewing can advance a seen cursor. The native
browser proof blocked prohibited HTTP/WS operations and recorded **zero attempts**
and **zero page errors**. Its 77-character draft was restored without submitting.

Journey: select existing session, scroll 45 frames each direction, open a long
message, scroll and find in the reader, search the session, type/restore an unsent
mobile draft, then restore the view. Desktop 1440×900; mobile capture 390×844.
Use the app's existing `?session=<id>` deep link when a shared active-pane change
or fresh-profile sidebar visibility would otherwise select the wrong session.

## Measured results

| Measure | Before | Final | Interpretation |
| --- | ---: | ---: | --- |
| Desktop MessagesPane average commit | 3.1ms | 2.4ms | 23% lower |
| Desktop MessagesPane maximum commit | 48.8ms | 30.5ms | 38% lower in these captures |
| Desktop long tasks | 1,213ms | 832ms | 31% lower |
| Mobile long tasks | 1,186ms | 581 / 612ms | 48–51% lower across two final captures |
| Mobile MessagesPane average commit | 3.1ms | 2.7 / 2.6ms | 13–16%; near the expected noise band |
| Mobile maximum mounted transcript nodes | 2,186 | 1,249 | 43% fewer |
| Mobile style recalculation | 2,594.2ms | 2,132 / 2,191.6ms | 16–18% lower; substantial work remains |

BAS outcomes confirm the same **574-event** target, **18 mounted rows**, final
scroll position **92,339px**, and **23,998 rendered reader characters** before
and after. TerminalPane background commits were 37 before and 33 in the first
final mobile capture. Live traffic and font/highlighter startup still introduce
noise. No statistical significance or zero-jank claim is made. Increased commit
counts are not treated as a failure when individual work is cheaper.

The initial, broader scroll-only trace recorded 4,087ms of long tasks and about
50% dropped frames. It established symptoms; it is not compared with the shorter
combined journey above. Performance Health's trace-duration field contains a
clock-origin anomaly, so elapsed duration is not used as an acceptance number.

## Confirmed causes and final changes

- `ui/src/components/messages/MessageRow.tsx:96` and `messages/outline.ts:40`:
  the 400px clip previously parsed/rendered the entire hidden source. Rows now
  render at most 6,000 source characters/80 lines. A budget-cut code fence is a
  plain excerpt, so no code-copy control misrepresents an incomplete block.
  Full event text still feeds the reader, message copy, export, snippets, search,
  handoff and playback. The reader footer is present whenever source is omitted.
- `ui/src/hooks/useVirtualList.ts:264`: row-ref attachment forced a layout read
  for every newly mounted item, then ResizeObserver measured it again. Native
  delivered border-box sizes now supply measurements; direct bounds are the
  compatibility fallback. A new regression observed **16 forced reads before,
  zero after**. First measurements publish confirmation even when the estimate
  was exact, so saved-position restoration does not wait forever. Key changes
  also invalidate geometry while measured heights stay attached to their items.
- `ui/src/components/messages/MessagesReader.tsx:86`: one scan now owns counting
  and painting find matches. Query, reply, source and render-mode changes reset
  navigation without an extra reset commit. A refresh regression protects against
  an invalid “3 of 1” active match. Removed an unused font-clamp forwarding API.
- `ui/src/components/MessageJumpList.tsx:434`: memoization follows search hits,
  not a new search-envelope object on every parent commit. Existing redundant
  conditions, unused imports and void-callback lint debt were removed in the
  touched components.
- `ui/src/components/MessagesPane.tsx:794`: one event-aware snippet handler
  replaces an accumulating per-event callback map and its cleanup effect.
- Shared owner: `scenarios/react-component-library/library/services/LayerManager/
  versions/2.0.2/LayerManager.tsx:45` publishes `data-rcl-overlay-canvas` from its
  existing sheet/drawer stack on push/dispose. BottomSheet **1.3.5**,
  FullPageDrawer **1.4.5**, and ResponsiveDialog **1.3.5** use that flag instead
  of root descendant queries. Nested menus preserve an underlying sheet's canvas;
  the last sheet/drawer closing clears it. No second tracker or mutation observer.

A browser-only experiment removing unbounded root `html:has(...)` rules reduced
style work from 2,720.8ms to 1,086.5ms. Narrowing them to direct portal descendants
preserved behavior but failed to reproduce that reduction after adoption, so that
approach was superseded by the lifecycle flag. The final table reports deployed
combined results, not the isolated experiment's larger delta.

Shared assets were edited only through `draft-begin`, `content-set`, and
`draft-publish`. The strict package/declaration build verified artifact
`sha256:b900bb8d8bb4cc0a123e494a0b82e95a5686278f51eb5ceb8f533024963ad19b`.
Web Console refreshed the existing local package through Scenario Dependency
Analyzer; no raw install or direct release/generated-file edit was used.

## Validation and limits

- Final focused Web Console cohort: 13 test files, **216 tests** (completion
  output retained in `/tmp/wc-perf-final-focused.log`). Shared canvas lifecycle
  and portal behavior: **5 PASS**. Focused component ESLint has no errors; the
  virtualizer retains an advisory exhaustive-deps warning for its intentional
  ref-geometry `sizeVersion` invalidation. Selector and string freshness pass.
- Native touch-browser proof on the ordinary production build opens real search
  and account drawers, then the 23,998-character reader. HTML, body and each
  actual panel agree at **rgb(23,32,51)**. Draft restoration, zero prohibited
  attempts and zero page errors are recorded in `/tmp/wc-perf-browser-proof.json`.
  Screenshots: `/tmp/wc-perf-mobile-{messages,search,reader}.png`.
- Test Genie **20261016-040746-e5211c49**, requested phase `performance` only:
  **FAIL**, `PERF_LIGHTHOUSE_BELOW_ERROR_THRESHOLD`, score **49 < 75**.
  Build, declared-budget, profiling and surface-resolution capabilities are clean.
  Thresholds were preserved. The phase preflight confirmed generated Proto and
  shared-package freshness.
- Retained Lighthouse diagnostic: `/tmp/wc-perf-lighthouse-diagnostic.json`,
  score **59**, mobile form factor, FCP **2.4s**, LCP **25.1s**, TBT **390ms**,
  CLS **0.031**, transfer **4,192KiB**, script evaluation about **11.9s**.
  This is additional diagnosis, not a replacement passing gate. The owner runner
  ignores the scenario's declared desktop form factor; filed as
  **knw-1792123967704779496**. Cold loading, eager background session hydration,
  large bundles, and live terminal activity need a separate targeted follow-up.
  The whole conversation-map subscription in Workspace remains architectural
  debt documented in the earlier audit; it was not claimed fixed here.
- `pnpm run type-check` still reports four pre-existing test-fixture errors in
  `machines/MachineList.test.tsx` and `desktop/desktopTransport.test.ts` (also
  recorded by the preceding keyboard/launch work). No error names the changed
  production sources. These are local fixture types, not stale generated output.
- Desktop floating voice toolbar intercepted the view toggle in a fresh profile.
  Measured flows used the same control's DOM click; filed separately as
  **knw-1792123204122572156** through `report-bug`.
- Native iOS keyboard/OS rendering and complete scenario certification are not
  claimed. Stopped captures `f6043953-1329-43da-baa4-e2f2bb582b18` and
  `fac8e17f-cd6b-473b-8c6b-937100178c98`, plus one unavailable trace, are excluded.

Two final source/behavior reviews checked full-text retention, partial code-copy
semantics, measurement confirmation, key identity, find resets, nested overlay
lifecycle, draft restoration, production mode and evidence attribution. Findings
within the repair boundary were fixed and covered by regressions. Remaining
cold-load/measurement and pre-existing fixture debt is explicitly retained above.

## Durable capture artifacts

Desktop baseline: Performance Health audit trace:
`/home/matthalloran8/.vrooli/data/vrooli/browser-automation-studio_shadow/captures/e8149728-a899-4434-ae05-f83d6d5c635a/e92a354c-3e2d-4ea4-9296-968e19f971ae/performance/performance.json`.

- `baseline-mobile`: execution `ff3ccb32-854a-4247-bc95-5f504588df5e`; trace `/home/matthalloran8/.vrooli/data/vrooli/browser-automation-studio/captures/853e45c7-a409-487a-95b2-c51d04e795fd/ff3ccb32-854a-4247-bc95-5f504588df5e/performance/performance.json`.
- `final2-mobile`: execution `80527e58-bd4d-4314-8809-c13892956d3b`; trace `/home/matthalloran8/.vrooli/data/vrooli/browser-automation-studio/captures/9a00f8fb-4a9e-46ef-9e87-6cbcf58a7419/80527e58-bd4d-4314-8809-c13892956d3b/performance/performance.json`.
- `final3-mobile`: execution `afa04801-ee72-40f1-b3da-469730e4c5d4`; trace `/home/matthalloran8/.vrooli/data/vrooli/browser-automation-studio/captures/3bfc1e39-f247-43ff-9ccc-089087445e8b/afa04801-ee72-40f1-b3da-469730e4c5d4/performance/performance.json`.
- `final2-desktop`: execution `85b9a410-3771-4604-ad68-481a79c26261`; trace `/home/matthalloran8/.vrooli/data/vrooli/browser-automation-studio/captures/cb1cf3d2-1bd4-48e5-a275-26b8335897cf/85b9a410-3771-4604-ad68-481a79c26261/performance/performance.json`.

Analyses and comparison: `/tmp/wc-perf-*-analysis.json` and `/tmp/wc-perf-mobile-comparison.json`. Full traces remain in the BAS-owned capture locations above.

Final runtime proof: `vrooli scenario status web-console` reports **running / healthy**;
`web-console --api-base http://localhost:16382 status` reports **Ready: true**,
database connected. Production bundles contain no profiling-renderer markers.
Work record: **67b8f5d5-e3be-420c-861a-626815e05689**.


## Authorized follow-up implementation — 2026-10-16

Operator requested implementation and validation of all recorded follow-ups.
Bounded W3 repair extends to the Performance Health Lighthouse owner. Targets:
respect declared desktop/mobile measurement settings without changing thresholds;
prioritize active-session history and bound background hydration with shared load
ownership; isolate navigation and playback subscriptions from unrelated event
updates; keep the draggable voice toolbar clear of essential navigation controls;
repair machine/desktop fixture types. Preserve unread counts, paging, playback,
full message source and the existing read-only real-session interaction boundary.
Validate focused regressions, scoped Test Genie phases, deployed real-data
interaction and startup traces, ordinary pointer clicks and mobile unsent drafts.
Physical iPhone validation requires a verified reachable Apple target; record the
actual inventory disposition and never substitute emulation for physical proof.
Keep findings and final evidence here; leave production Web Console healthy.


### Follow-up implementation and baseline evidence

- Hydration now shares `refreshConversationSession` ownership with panes, loads
  the active session first, and uses two background workers with retry/error and
  paging metadata preserved. Foreground selection can bypass busy background
  workers. Completed histories are not reloaded when the session list changes.
- Workspace no longer subscribes to the conversation map. Sidebar and unread
  badges own navigation summaries; handoff subscribes while open; playback
  observes its selected session and reads fresh event/queue data for actions.
  Removed the root's signature cache and whole-workspace event scans.
- Settings, appearance, fleet and archive modules download on first use. The
  deferred boundary shows loading feedback and stays mounted after first open,
  preserving surface state and exit animations.
- Floating toolbar placement reserves the measured top chrome plus the view
  control row, for stored positions, dragging and docking. Height-only viewport
  changes also clamp docked tabs. Report: `knw-1792123204122572156`.
- Performance Health forwards desktop/mobile device settings, explicit screen
  and throttling settings, categories, skipped audits, Chrome flags and timeout.
  A report with the wrong measured form factor is rejected. No thresholds were
  changed. Report: `knw-1792123967704779496`.
- Machine and desktop test fixtures now satisfy their production types. Touched
  production files also received focused lint cleanup rather than suppressions.

Regression-before evidence: `/tmp/wc-followup-regression-before.log` demonstrates
both the extra Workspace render on a background reply and six history requests
for five sessions (including duplicate foreground loading). The repaired focused
cohort passed 119 tests across 15 files before final scenario validation.

Successful baseline startup trace:
`/home/matthalloran8/.vrooli/data/vrooli/browser-automation-studio_shadow/captures/70996ff6-5f37-451c-9723-ac77574cd07f/4a78aa65-22be-4b7d-b63c-e13cd67d41e4/performance/performance.json`.
Performance Health analysis: `/tmp/wc-followup-startup-before-analysis.json`.
It waits for the actual existing heartbeat Messages view, rather than measuring
only the launch screen. The assertion-free flow uses the supported session deep
link so sidebar origin-bucket visibility does not select a different session.

Guarded production-browser baseline (`/tmp/wc-followup-browser-before.json`):
31 sessions, 32 history requests, peak 31 concurrent requests, 3,002,534 downloaded
JavaScript bytes on both viewports. Desktop workspace/messages ready at
2,349/4,452ms; mobile at 1,507/2,529ms. These readiness times are single local
observations, not throttled Lighthouse metrics or statistical guarantees.
Both read the real 574-event session and recorded zero prohibited operations and
page errors. Corrected desktop Lighthouse baseline scored 62, accessibility 94;
minimums remain 75 and 90 respectively.

Excluded evidence: the default mount-only capture completed before the lazy
Workspace loaded; one capture was interrupted by a measurement-owner restart;
malformed edges and a hidden sidebar origin bucket caused failed startup flows;
an intermediate candidate's callback-order error was caught and repaired.
None is counted as a passing trace or a before/after performance comparison.

Physical iPhone route probe (`scenario-to-ios ios targets --json`) reports Linux
simulator unsupported, no usable macOS bridge established, and bridge inventory
unavailable with `unauthenticated`. This does not prove a phone is disconnected;
it means this environment has no verified authorized physical-device path.
Operator clarified: “nah don't worry about that just do your best with what's available”.
Physical-device execution is therefore outside this follow-up acceptance boundary. Chromium
mobile browser verification cannot certify the native iOS keyboard, momentum
scrolling or installed-PWA system canvas.
