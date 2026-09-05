# Experience Architecture Audit

## Last Updated
2026-04-06 (Phase 18 iter3)

## Scenario Purpose
Vrooli Events is the central event bus for inter-scenario communication. It provides a durable SQLite event store with SSE pub/sub and a real-time analytics dashboard. Users come here to monitor, debug, and govern event flows between scenarios.

## Personas Identified

| Persona | Primary Job | Current Flow | Ideal Flow |
|---------|-------------|--------------|------------|
| Platform Operator | Monitor event throughput and system health | Open dashboard → check health indicator → view stream page | Same, plus analytics charts with historical trends |
| Scenario Developer | Debug inter-scenario event flow | Open dashboard → filter by source/type → inspect event payload → click correlation ID → trace page (Phase 18) | Same, plus direct event deep-linking |
| Security/Compliance | Audit policy violations | Policies page → Compliance page → violation log | Policy management UI → violation log → compliance scorecard with trend data |

## Friction Analysis

### Mechanical Friction
- **No direct event link**: Cannot deep-link to a specific event by ID. Hash routing only supports page-level navigation plus query params.
- ~~**Manual filter clearing**: Must clear each filter field individually. No "reset filters" button.~~ **Fixed (Phase 18 iter3)**: Both StreamPage and EventLogPage now show a "Reset" button when any filter is active.

### Cognitive Friction
- ~~**Stream vs Event Log distinction**: "Stream" shows live SSE events, "Event Log" queries stored events. The names don't clearly communicate this difference.~~ **Fixed (Phase 18 iter3)**: Renamed to "Live Stream" and "Event History" — "Live" signals real-time SSE, "History" signals stored/queryable events.
- **Settings page is read-only**: Shows retention configuration but provides no controls to change it (API supports changes but UI doesn't wire them up).

### Discoverability Friction
- ~~**No onboarding**: New users see an empty stream page with no events.~~ **Fixed (Phase 18)**: Quick-start hint with curl example now shows when stream is empty.
- **Hidden health details**: Health indicator in sidebar shows green/red but doesn't show subscriber count or store stats without opening a different page.

## Cross-Navigation (Phase 18)

The following cross-page navigation links were added:
- **EventDetail → Correlation Traces**: Clicking a correlation ID in the event detail panel navigates to `/traces?cid=<id>`, auto-populating the search.
- **Scenario Metrics → Event Log**: Clicking a scenario name navigates to `/events?source=<name>`, auto-populating the source filter.
- **Circuit Breakers (empty) → Policies**: The empty state now links directly to the Policies page instead of just mentioning it.

Pages that accept query params:
- `CorrelationTracePage`: `?cid=<correlationId>` — auto-fills and triggers search
- `EventLogPage`: `?type=&source=&target=&cid=&limit=` — all filters URL-persisted for deep-linking/shareability
- `StreamPage`: `?type=&source=` — filters URL-persisted

## Navigation Integrity
- **Hash-based routing** via react-router-dom HashRouter: 15 routes (10 top-level + 2 parameterized + index + wildcard + layout).
- **No back/forward issues**: Hash changes work with browser history. Cross-navigation links use `useNavigate()`.
- **Query param deep-linking**: Full filter persistence for StreamPage (`?type=&source=`), EventLogPage (`?type=&source=&target=&cid=&limit=`), and CorrelationTracePage (`?cid=`). All filters are shareable URLs.
- **Label→destination match**: All sidebar labels match their page content.
- **Sub-page back buttons**: PolicyEditorPage and SubscriptionHealthPage have explicit "Back" buttons that return to their parent list pages.

## Shared Components (Phase 18 iter2)

- **EmptyState**: Shared component with icon, title, description, and optional action link. Applied to PoliciesPage, SubscriptionsPage, ScenarioMetricsPage for consistent empty-state UX with contextual guidance text.
- **Spinner**: Animated SVG spinner with label. Applied to all 10 data-fetching pages replacing plain "Loading..." text. Provides visual motion feedback during API calls.

## Priority Improvements (Remaining)
1. ~~**Add "Quick Start" hint on empty stream**~~ — Done (Phase 18).
2. ~~**Rename pages**~~ — **Done (Phase 18 iter3)**: "Live Stream" and "Event History".
3. ~~**Add reset-filters button**~~ — **Done (Phase 18 iter3)**: Reset button appears when any filter is active.
4. **Wire up retention settings** — allow editing from the Settings page.
5. ~~**Persist all filters in URL**~~ — **Done (Phase 18 iter3)**: StreamPage and EventLogPage sync all filters to URL query params.
6. ~~**Add loading skeletons** — replace "Loading..." text with skeleton components for smoother UX.~~ **Done (Phase 18 iter2)**: All pages now use Spinner component with animated SVG and contextual label.
