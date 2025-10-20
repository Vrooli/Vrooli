# App Monitor UX Revamp Plan

## 1. Goals and Design Tenets
- Deliver a chrome-lite experience that feels like a dedicated browser: no traditional header, no permanent sidebar, and a persistent bottom nav with high-signal actions.
- Consolidate application, resource, and browser-tab management into a single, full-screen switcher dialog that emphasizes discoverability, history, and quick commands.
- Enable asymmetric browsing: scenarios/resources behave as managed assets with history-first ordering, while outside URLs mimic conventional browser tab semantics.
- Simplify the codebase by collapsing redundant routes, centralizing data fetching, and removing legacy layout/preview logic that bloats `Layout`, `AppsView`, `ResourcesView`, and `LogsView`.
- Improve mobile ergonomics with bottom-nav access, responsive grids (2-up cards), and touch-first overlays.

## 2. High-Level Architecture
### 2.1 App Shell
- Replace `Layout.tsx` with a new `Shell` component that:
  - Owns router outlet rendering for "content panes" (App preview, resource detail, etc.).
  - Manages global overlays (tab switcher, quick actions dialog) and bottom navigation state.
  - Hosts shared providers (Zustand stores, device emulation, debug beacons) without embedding layout markup.
- Update `App.tsx` to:
  - Keep route paths for deep-link friendliness (`/apps/:appId/preview`, `/resources/:resourceId`, `/?view=tabs`, etc.).
  - Interpret query/route params to toggle overlays instead of mounting standalone pages.

### 2.2 Overlays & Dialogs
- **Tab Switcher Overlay** (`TabSwitcherDialog`): full-viewport, focus-trapped component that replaces sidebar, history popover, apps/resources list pages.
- **Quick Actions Overlay** (`StatusActionsDialog`): bottom-sheet / modal triggered by ellipsis button; shows system status, uptime, quick controls.
- Overlay host uses React portals to render above everything else while keeping scroll locking consistent.

## 3. State Management
- Preserve existing Zustand stores for apps/resources but extract reusable selectors/hooks for filtering, sorting, and action dispatch.
- Introduce new `useBrowserTabsStore` for outside URLs:
  - State shape: `{ activeTabId, tabs: TabRecord[], history: TabHistoryEntry[], screenshotCache: Record<tabId, ScreenshotMeta> }`.
  - Actions: `openTab(url)`, `closeTab(id)`, `activateTab(id)`, `recordScreenshot(id, dataURL)`, `persist()`.
  - Persistence: lightweight localStorage sync (deferred to background effect) to survive reloads.
- Add `useOverlayStore` or co-locate overlay state inside `Shell` using React context to avoid prop drilling.
- Extend `useAppsStore` to expose derived data (history ordering, counts) via selectors to prevent duplication.

## 4. UI Composition
### 4.1 Bottom Navigation
- Component: `BottomNav.tsx` with two buttons.
  - **Tabs button**: icon with badge count (active scenarios/resources + outside URLs). Opens `TabSwitcherDialog`.
  - **More button**: ellipsis icon; opens `StatusActionsDialog`.
  - Handles haptic-friendly hit areas and accessibility (aria labels, focus outline, keyboard shortcuts).

### 4.2 Tab Switcher Dialog
- Layout sections:
  1. Search bar + sort icon (borrowing behavior from `AppsView`).
  2. Segment control (Apps | Resources | Web). Use shared button group component with robust keyboard focus.
  3. Content grid: square cards (2 per row on ≤600px). Each card includes header (name/status), cached image slot, metadata chips (status, view count), inline quick actions (start/stop, open preview, open logs).
  4. Optional secondary sections: "History" (apps/resources) vs "All", separated with headings; Web view shows active tabs first, then history list.
- Reuse skeleton/loading states; ensure virtualization or batching if dataset large.

### 4.3 Card Data Sources
- **Apps**: Use `useAppsStore`. History ordering replicates `HistoryMenu` logic (last viewed timestamp, view count). "All" uses alphabetical sorting.
- **Resources**: Use `useResourcesStore`. History derived from existing telemetry (if available) or fallback to alphabetical. Provide start/stop/refresh buttons with inline feedback.
- **Web Tabs**: `useBrowserTabsStore`. Display screenshot or favicon; include close button.

### 4.4 Screenshots & Icons
- Scenarios/URLs: integrate with `useIframeBridge` to trigger screenshot capture on blur/overlay open; store as data URL in tab store (now backed by the surface media cache).
- Resources: show package icon (existing assets) or fallback emoji; evaluate thumbnail capture once proxy data is available.
- Provide placeholder skeleton while screenshot loads; degrade gracefully if capture fails.

## 5. App Preview Integrations
- Refactor `AppPreviewView` to incorporate logs overlay:
  - Extract log fetching/rendering from `LogsView` into `useAppLogs(appId)` + `AppLogsPanel` component.
  - Add preview toolbar action to toggle logs mode; when active, iframe area replaced by logs panel (with stream selectors, filters, auto-scroll).
  - Keep device emulation and iframe bridge features intact.
- Record last viewed timestamp & view count updates when preview loads or logs panel opened, reusing existing telemetry methods.

## 6. Route & Navigation Strategy
- Consolidate routes:
  - `/` → `Shell` with preview route or blank state.
  - `/apps/:appId/preview` remains primary deep link.
  - `/resources/:resourceId` renders resource detail view in shell content pane.
  - `/tabs` (optional) or `?overlay=tabs` toggles tab switcher; map old `/apps`, `/resources`, `/logs` to new overlay states via redirects or effect.
- Update navigation helpers (`useNavigate`) to set overlay state rather than pushing to legacy pages.
- Remove `HistoryMenu`, `AppsView`, `ResourcesView`, `LogsView` default exports after migration; replace with small fallback wrappers that call new overlay logic if needed during phased rollout.

## 7. Styling & Theming
- Create new CSS modules or Tailwind-esque utility wrappers for modern layout.
- Deprecate `Layout.css`, `AppsView.css`, `ResourcesView.css`, `LogsView.css`, migrating reusable tokens into a shared file (`styles/layout.css`, `styles/cards.css`).
- Ensure matrix background and global effects remain optional toggles; adjust to full-screen overlay context.
- Implement responsive breakpoints for bottom nav spacing and grid columns.

## 8. Cleanup & Deletions
- Remove unused components:
  - `Layout.tsx`, `HistoryMenu.tsx`, `AppsView.tsx`, `ResourcesView.tsx`, `LogsView.tsx`, associated skeletons if superseded.
  - Sidebar-specific icons/components, collapsed-state logic, `menuItems` arrays.
- Prune corresponding styles, tests, and storybook/demo files if any.
- Update docs (`README.md`, architecture docs) to describe new UX.
- Delete debug-specific features (sidebar toggles, history popovers) that are obsolete.

## 9. Implementation Phases
1. **Scaffold Shell** ✅ *(done)*
   - Shell wrapper with bottom nav and overlay state now wraps legacy layout.
   - Tab switcher and actions overlays replaced placeholder content.
2. **Port Apps/Resources data pipelines** ✅ *(done)*
   - Shared catalog hooks/utilities power both legacy views and the tab switcher.
   - Scenario/resource grids render via new reusable components.
3. **Build Browser Tabs store** ✅ *(done)*
  - Zustand store manages active/history web tabs; dialog supports open/close/reopen flows.
  - Screenshot capture for scenarios/web tabs now lands via the surface media cache (resources still pending).
4. **Integrate Screenshots** ✅ *(done)*
  - Scenario and web tab cards consume cached bridge snapshots; history reopen restores thumbnails.
  - Follow-up: evaluate resource capture once a safe source of imagery exists.
5. **Refactor Logs** ✅ *(done)*
  - Inline logs overlay, new `useAppLogs` hook, toolbar toggle, and `/logs` redirects keep deep links working.
  - Follow-up: remove `LogsView` + styles once the redirect has baked; add analytics for overlay usage.
6. **Replace Legacy Routes** *(next)*
  - Finish collapsing legacy layout/sidebar artifacts, remove unused menu plumbing, and promote query-driven overlays everywhere.
7. **Styling & Polish** *(ongoing)*
   - Continue responsive/a11y refinements as new surfaces stabilize.
8. **Cleanup & Docs** *(final)*
   - Remove dead files, update tests/docs, run lint/test suites before wrap.

## 10. Testing Strategy
- Unit-test new stores/hooks (app filtering, browser tabs actions, logs hook).
- Integration/UI tests (Playwright/Vitest + Testing Library) for overlay flows: open/close tabs dialog, filter/search, open resource actions, toggle logs.
- Regression tests for `AppPreviewView` (device emulation, bridge interactions).
- Manual QA checklist: mobile bottom nav usability, keyboard navigation, screen reader labels, screenshot caching behavior, resource action success/failure feedback.

## 11. Observability & Telemetry
- Update client debug beacons currently fired in `Layout` to new shell lifecycle (route changes, overlay toggles) to retain instrumentation.
- Capture analytics events for tab creation, outside URL usage, logs toggle.

## 12. Risks & Mitigations
- **State divergence**: ensure overlays subscribe to store slices to avoid stale counts; leverage selectors + memoization.
- **Screenshot storage bloat**: limit cache size, compress images, and expire old entries.
- **Accessibility regressions**: invest in focus management utilities and run axe audits.
- **Performance**: implement virtualization or lazy rendering if grids exceed threshold; throttle screenshot capture.

## 13. Outstanding Questions
- Should outside tabs sync with backend for team-wide visibility or remain local-only?
- Do we need per-resource history metadata to mimic app behavior, or is alphabetical enough?
- How aggressively should we expire screenshot cache to balance fidelity vs storage?

## 14. Definition of Done
- Old sidebar/header routes removed; `Shell` + overlays fully drive navigation.
- Tab switcher presents apps, resources, and web tabs with search/sort/filter parity to legacy views.
- Logs accessible within preview, no standalone logs route.
- All documentation and tests updated; lint/test/build pipelines pass.
- Manual QA sign-off on desktop + mobile experiences.
