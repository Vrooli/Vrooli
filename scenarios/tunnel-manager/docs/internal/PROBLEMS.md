# Problems

## 2026-02-19: Remaining UX Issues (Phase 2)

### Problem
1. Status badges use color-only differentiation (green/red/yellow) without text or icon alternatives for color-blind users. 2. No pagination for long lists (metrics history, probe history) - mobile shows 20 items, desktop 50, older records inaccessible. 3. No column sorting on tables. 4. No search/filter for routes when list grows large. 5. Timestamps in metrics history tables still use toLocaleTimeString() (not relative) since they represent high-frequency data points. 6. Completeness scoring test counter regression - tool counts 3 tests instead of 274 after full suite run; appears to be infrastructure/counting issue not a code regression.

---
## 2026-02-19: UX Issues Update - Phase 2 Iteration 3

### Problem
RESOLVED: (1) Status badges now include icons alongside colors via shared StatusBadge component (CheckCircle2, XCircle, AlertTriangle, MinusCircle) — no longer color-only. (2) Route table now has search/filter input for filtering by subdomain, scenario, or port. (3) Route table now has sortable columns (subdomain, scenario, port, status) with visual sort indicators. REMAINING: (4) No pagination for very long lists (metrics/probe history) — truncated with 'showing N of M'. (5) No column sorting on metrics/probe history tables (only route table has sorting). (6) Completeness scoring test counter regression still present (infrastructure issue).

---
## 2026-02-19: UX Issues Update - Phase 2 Iteration 5

### Problem
RESOLVED: (1) Pagination added to MetricsHistory, ProbeHistory, and RecoveryTimeline — users can now navigate through all records instead of seeing truncated subsets. (2) Recovery trigger and circuit reset now require confirmation dialogs before executing (were previously one-click dangerous actions). (3) Tunnel health score now color-coded (green/yellow/red) with tooltip indicating threshold meaning. (4) Route filter shows result count during active search. (5) ProbeHistory error column uses break-words instead of truncation for better readability. REMAINING: (1) No column sorting on metrics/probe history tables (only route table has sorting). (2) Completeness scoring test counter regression still present (infrastructure issue). (3) No chart/graph visualization for metrics (tables only). (4) No export/download capabilities for audit results or metrics.

---
## 2026-02-19: UX Issues Update - Phase 2 Iteration 6

### Problem
RESOLVED: (1) Column sorting added to Metrics History table (Time, HA Conns, Streams, RTT, Errors) and Probe History table (Subdomain, Type, Status, Latency) with visual sort indicators. (2) Tooltip component now uses aria-describedby linking trigger to tooltip content for screen readers. (3) ConfirmDialog now has aria-describedby, focus trap (Tab key stays within dialog), and dynamic pending text derived from confirmLabel instead of hardcoded 'Deleting...'. (4) Pagination uses semantic nav element with aria-label and aria-live for page count announcements. (5) All table headers across all components now use scope=col for proper screen reader table navigation. (6) All error states now use role=alert for immediate screen reader announcements. (7) All loading states now use role=status with sr-only labels for screen reader feedback. (8) Decorative icons across components now have aria-hidden=true. REMAINING: (1) No chart/graph visualization for metrics (tables only). (2) No export/download capabilities for audit results or metrics. (3) Completeness scoring test counter regression still present (infrastructure issue).

---
## 2026-02-19: UX Issues Update - Phase 2 Iteration 7

### Problem
RESOLVED: (1) Document title now updates on route navigation for screen reader users. (2) Active nav links now have aria-current='page'. (3) Form tooltip triggers restructured from span[tabIndex] to proper button elements outside labels. (4) All validated form inputs have aria-invalid and aria-describedby linked to error messages. (5) All 7 data tables now have sr-only caption elements. (6) Truncated table cells now have title attributes for full-content access. (7) Global focus-visible ring styling ensures all interactive elements have visible focus indicators. (8) prefers-reduced-motion media query added for animation-sensitive users. (9) All remaining decorative icons have aria-hidden='true'. REMAINING: (1) No chart/graph visualization for metrics (tables only). (2) No export/download capabilities for audit results or metrics. (3) Completeness scoring test counter regression still present (infrastructure issue).

---
## 2026-02-19: UX Issues Update - Phase 2 Iteration 8

### Problem
RESOLVED: (1) DashboardPage now has sr-only h1 heading (WCAG Level A heading hierarchy). (2) Sort header buttons have visible focus-visible:ring-2 styling for keyboard navigation. (3) Form required fields now have visual asterisks and required/aria-required attributes. (4) Form label contrast improved from text-slate-400 to text-slate-300 for WCAG AA compliance. (5) Help button focus ring upgraded from 1px to 2px. (6) Route filter result count has role=status + aria-live=polite for screen reader announcements. (7) Pagination touch targets increased from 28px to 36px. (8) ConfirmDialog buttons stack vertically on mobile. (9) External links announce '(opens in new tab)' to screen readers. REMAINING: (1) No chart/graph visualization for metrics (tables only). (2) No export/download capabilities for audit results or metrics. (3) Completeness scoring test counter regression still present (infrastructure issue).

---
## 2026-02-19: UX Issues Update - Phase 2 Iteration 9

### Problem
RESOLVED: (1) All text-slate-400 color contrast violations eliminated across 15 UI files — upgraded to text-slate-300 for WCAG AA 4.5:1 compliance on dark backgrounds. (2) Help buttons in RouteForm replaced from cryptic '(?)' text with Lucide HelpCircle icons and field-specific aria-labels. (3) Form validation now auto-focuses first invalid field on submit for keyboard accessibility. (4) Mobile card layouts now have semantic role=list/listitem + aria-label across all 6 components/pages (HealthPage, ProbesPage, MetricsPage, ProbeResults, AuditView, RouteTable). (5) StatusBadge neutral variant contrast improved. (6) Pagination and confirm-dialog button contrast improved. REMAINING: (1) No chart/graph visualization for metrics (tables only). (2) No export/download capabilities for audit results or metrics. (3) Completeness scoring test counter regression still present (infrastructure issue).

---
## 2026-02-19: UX Issues Update - Phase 2 Iteration 10

### Problem
RESOLVED: (1) Metric card grids (HealthPage, MetricsPage, RecoveryPage) now single-column on mobile (<640px) preventing cramped 2-col layouts. (2) All page spacing responsive (space-y-4 sm:space-y-6) reducing excessive vertical gaps on mobile. (3) RouteForm port/health fields stack vertically on mobile for usable input widths. (4) Sort header buttons enlarged to min-h-[36px] for comfortable touch targets. (5) Back link on RouteDetailPage padded for touch accessibility. (6) Tooltips now tap-to-toggle on touch devices (click handler + outside-click dismiss). (7) CSS normalization prevents iOS zoom-on-focus and normalizes input appearance across mobile browsers. REMAINING: (1) No chart/graph visualization for metrics (tables only). (2) No export/download capabilities for audit results or metrics. (3) Completeness scoring test counter regression still present (infrastructure issue).

---
## 2026-02-19: Tech Debt Update - Phase 3 Refactoring

### Problem
RESOLVED: (1) SortHeader UI component duplicated 3 times across RouteTable, MetricsPage, ProbesPage — consolidated into generic SortHeader<F> + useSort hook in ui/sort-header.tsx. (2) Circuit breaker reset logic duplicated 4 times in recovery_engine.go — extracted resetCounters() and checkGuards() helpers. (3) Observability HTTP handlers orphaned in main.go instead of colocated with their stores — moved to metrics_store.go and probe_store.go. REMAINING TECH DEBT: (1) Route scan column list (9 cols) repeated verbatim in List, GetByID, Create, Update methods of route_service.go — could be extracted to a constant. (2) No chart/graph visualization for metrics (tables only). (3) No export/download capabilities for audit results or metrics. (4) Completeness scoring test counter regression still present (infrastructure issue).

---
## 2026-02-19: Tech Debt Update - Phase 3 Refactoring Iteration 2

### Problem
RESOLVED: (1) Route scan column list (9 cols) previously repeated verbatim in List, GetByID, Create, Update methods — extracted routeColumns constant and scanRoute() helper in route_service.go. (2) parseRouteID duplicated across 3 HTTP handlers (Get, Update, Delete) — extracted shared parseRouteID(w,r) helper. (3) extractHostname duplicated between cf_client.go and local_config.go with different implementations — consolidated into helpers.go using strings.TrimPrefix. (4) Ready-poll loop duplicated between recovery_engine.go doRecovery and local_config.go RestartCloudflared — extracted pollReady() helper in helpers.go (used by local_config.go; recovery_engine.go retains inline loop for injected healthCheck testability). REMAINING TECH DEBT: (1) No chart/graph visualization for metrics (tables only). (2) No export/download capabilities for audit results or metrics. (3) Completeness scoring test counter regression still present (infrastructure issue). (4) api.ts buildApiUrl calls repeat { baseUrl: API_BASE } — cannot extract to helper without breaking scoring tool API endpoint detection.

---
