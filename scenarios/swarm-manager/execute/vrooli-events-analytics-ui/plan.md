# Implementation Plan: Build The vrooli-events Analytics And Compliance Dashboard

## Required Reading

```bash
prompt-manager skill read react-coherence ux api-steer test seam-discovery-and-enforcement documentation-health
swarm-manager backlog file-get --kind execute --name vrooli-events-core-runtime --path plan.md
swarm-manager initiatives file-get --name vrooli-events --path orchestration-summary.md
```

Then orient yourself in the existing scaffold:

```bash
ls scenarios/vrooli-events/ui/src/pages/
ls scenarios/vrooli-events/ui/src/components/
cat scenarios/vrooli-events/ui/src/router.tsx
cat scenarios/vrooli-events/ui/src/lib/api.ts
cat scenarios/vrooli-events/api/routes.go
cat scenarios/vrooli-events/requirements/05-ui-dashboard/module.json
cat scenarios/vrooli-events/PRD.md
```

## 1. Purpose

Complete the vrooli-events analytics and compliance dashboard so it satisfies the description in this backlog item and the `OT-P2-*` operational targets in the PRD. The UI scaffold already exists (12 pages, typed API client, layout/nav, react-query, SSE plumbing). This item closes the remaining gaps so the dashboard ships a coherent, charted, editable, and verifiably-tested experience.

## 2. Problem Statement

The vrooli-events scenario has a substantial UI in place from prior scaffolding work, but the dashboard does not yet match the description in this backlog item:

- **No charts.** AnalyticsPage shows stat cards only, ScenarioMetricsPage shows a sortable table only. The description specifies "per-scenario call volume and error rate **charts**" and PRD `REQ-UI-002` calls for a "time-series chart". No charting library is installed in `scenarios/vrooli-events/ui/package.json`.
- **Read-only retention/storage settings.** `SettingsPage.tsx` displays hard-coded "30 days / 2 GB / 6h" values and explicitly says "API-configurable retention is planned for a future release." The description requires "retention and storage settings management" (i.e., editable).
- **Requirement status drift.** `requirements/05-ui-dashboard/module.json` marks REQ-UI-001..006 as `status: "complete"` but the validation refs are infrastructure tests (`utils.test.ts`, `errors.test.ts`, `selectors.test.ts`) that do not enforce chart rendering or feature behavior. The "complete" claim does not match implementation.
- **Coverage gaps.** Some pages have `.test.ts` files, others (e.g., `StreamPage.tsx`, `EventLogPage.tsx`) do not. No end-to-end smoke verifies the golden paths (event ingest → SSE render, policy CRUD round-trip, etc.).

Fixing these gaps gives operators a usable analytics + compliance surface and aligns the codebase with its own PRD claims.

## 3. Scope

### In Scope
- Add a charting library (Recharts unless decision changes it) to `scenarios/vrooli-events/ui` and use it on AnalyticsPage (throughput) and ScenarioMetricsPage (per-scenario volume + error rate).
- Make retention/storage settings editable end-to-end:
  - Backend: settings table in SQLite + GET/PUT `/api/v1/settings` handlers, applied at runtime by the pruner.
  - UI: replace the read-only `SettingsPage` panels with form controls + mutation.
- Audit existing pages (Stream, EventLog, Policies, PolicyEditor, CircuitBreaker, Subscriptions, SubscriptionHealth, Compliance, CorrelationTrace, ScenarioMetrics, Analytics, Settings) for regressions and gaps surfaced during this work; fix what blocks the description.
- Reconcile `requirements/05-ui-dashboard/module.json` so each `status: "complete"` is backed by validation that actually exercises the feature.
- Add tests covering the new chart components, the settings mutation, and the previously-untested pages (StreamPage, EventLogPage); plus a small E2E smoke if Playwright is selected.
- Update `scenarios/vrooli-events/docs/` references for any new endpoints.

### Out of Scope
- Core runtime, pub/sub, ingestion, query, SSE protocol — owned by completed `execute/vrooli-events-core-runtime`.
- Discovery package event emission and policy cache — owned by `execute/discovery-event-emission-and-policy-cache`.
- Policy CRUD API and enforcement middleware — owned by `execute/vrooli-events-policy-api-and-middleware` (status `failed`; not this item's responsibility to revive).
- Authentication on the API — deferred per PRD; do not add auth in this item.
- Notification-hub integration — separate item.
- Prometheus `/metrics` (PRD OT-P2-009) — out of scope here.

## 4. Dependencies

| Dependency | Kind | Status | What It Provides |
|---|---|---|---|
| `execute/vrooli-events-core-runtime` | execute | **completed** | SQLite event store, SSE pub/sub, ingest/query API, health endpoint |
| `execute/vrooli-events-policy-api-and-middleware` | execute | failed | Policy/subscription handlers already on disk (`handlers_policy*.go`, `handlers_subscription*.go`); UI consumes them — verify endpoints respond correctly during this item |

No external scenario or resource dependency. Embedded SQLite only.

## 5. Current Technical Context

### UI scaffold already on disk (`scenarios/vrooli-events/ui/`)
- React 18 + Vite 5 + TypeScript 5 + Tailwind 3 + react-router-dom 7 (HashRouter)
- `@tanstack/react-query` v5 for data fetching, polling intervals in `lib/constants.ts` (`HEALTH_POLL_INTERVAL_MS=10s`, `METRICS_POLL_INTERVAL_MS=5s`)
- Typed API client `lib/api.ts` (242 LoC): `fetchHealth`, `fetchEvents`, `fetchPolicies`, `fetchPolicy`, `fetchViolations`, `fetchSubscription(s)`, `fetchSubscriptionHealth`, `subscribeSSE`, `overrideCircuitBreaker`
- Pages (`src/pages/`, ~1,949 LoC total): StreamPage, AnalyticsPage, EventLogPage, ScenarioMetricsPage, CorrelationTracePage, PoliciesPage, PolicyEditorPage, CircuitBreakerPage, SubscriptionsPage, SubscriptionHealthPage, CompliancePage, SettingsPage
- Components: Layout (sidebar nav + health badge), EventTable, EventDetail, StatCard, Panel, PageHeader, StatusBadge, EmptyState, ErrorAlert, ErrorBoundary, Spinner, plus `components/ui/` (button)
- `consts/selectors.ts` declares a selector manifest (used by infra tests)
- `@vrooli/api-base` and `@vrooli/iframe-bridge` linked from `packages/`
- No charting library installed

### Backend already on disk (`scenarios/vrooli-events/api/`)
- Routes registered in `routes.go`:
  - Events: `POST /api/v1/events`, `GET /api/v1/events`, `GET /api/v1/events/subscribe`
  - Policies: `POST/GET/PUT/DELETE /api/v1/policies[/{id}]`, `GET /api/v1/policies/subscribe`, `GET /api/v1/policies/violations`, `POST /api/v1/policies/evaluate`, `POST /api/v1/policies/{id}/override`
  - Subscriptions: full CRUD plus `/health`, `/test`, `/deliver`
  - `GET /health`
- No `/api/v1/settings` endpoint exists yet — this item adds it.

### Codebase patterns to follow
- Go API: handler factory on `*Server`, real SQLite for tests (no mocks), `gofumpt` + `golangci-lint`, Go 1.22+ net/http ServeMux.
- React UI: react-query for server state, lucide-react icons, CSS variables for theming (`var(--surface-*)`, `var(--text-*)`, `var(--status-*)`), `data-testid` for table/form selectors.
- Charts: Recharts is the de facto standard across Vrooli scenarios (used by `agent-manager`, `deployment-manager`, `system-monitor`, `funnel-builder`, `scenario-auditor`, `social-media-scheduler`, `tech-tree-designer`, `ai-chatbot-manager`).

## 6. Target End State

A running `scenarios/vrooli-events` scenario where:
1. AnalyticsPage shows a time-series throughput chart (events over time) plus existing stat cards.
2. ScenarioMetricsPage shows a bar chart of per-scenario call volume and a comparable error-rate visualization, plus the existing sortable table.
3. SettingsPage is a working form: operators can edit retention window, size cap, and pruning interval; values persist in SQLite, are read by the pruner on its next cycle, and survive scenario restart.
4. CompliancePage, CircuitBreakerPage, PoliciesPage, SubscriptionsPage continue to function and pass tests; any regressions surfaced during the audit are fixed.
5. `requirements/05-ui-dashboard/module.json` accurately reflects implementation — every `status: "complete"` has at least one validation entry that exercises the rendered behavior.
6. Test suite: `pnpm --dir scenarios/vrooli-events/ui test` and `go test ./... -timeout 300s` in `scenarios/vrooli-events/api/` both pass; lint/format clean.

## 7. Implementation Strategy

### Decisions Pending Workshop

| Decision | Status |
|---|---|
| Scope strategy (audit-and-complete vs greenfield) | Pending (d1) |
| Charting library | Pending (d2) |
| Editable retention/settings approach | Pending (d3) |
| Throughput chart data source (client-aggregate vs backend rollup) | Pending (d4) |
| Test strategy (unit only vs unit + Playwright smoke) | Pending (d5) |

### Phased work (assumes recommended decisions; revise after workshop)

**Phase 1 — Backend: settings persistence** (gated by d3)
1. Add `app_settings` table to SQLite (key/value text or typed columns for `retention_days`, `retention_bytes`, `prune_interval_seconds`).
2. Migrations on startup: insert defaults if absent (30, 2*1024^3, 21600).
3. Implement `internal/settings/` (Get/Set with validation: positive ints, sane upper bounds).
4. Wire pruner to read live settings each cycle (no restart required).
5. Handlers: `GET /api/v1/settings`, `PUT /api/v1/settings` (returns updated state, 200; validation errors 400 with structured payload).
6. Integration tests with real SQLite: round-trip update, pruner picks up new values, validation rejections.

**Phase 2 — Backend: throughput rollup** (only if d4 selects backend rollup)
1. `GET /api/v1/events/stats?bucket=minute|hour&since=...&until=...` returns `[{bucket_start, count}]`.
2. SQL: `SELECT strftime(...) AS bucket, COUNT(*) FROM events WHERE created_at BETWEEN ... GROUP BY bucket`.
3. Tests with real SQLite covering bucketing, time range filters, empty range.

**Phase 3 — UI: charting library + Analytics chart** (gated by d2)
1. Install chosen library in `scenarios/vrooli-events/ui/package.json`.
2. Add `<ThroughputChart />` component (in `src/components/charts/`). Source data per d4.
3. Mount on AnalyticsPage above the existing stat cards. Loading/error/empty states reuse existing primitives.
4. Component tests using `@testing-library/react` (assert chart svg, axis labels, data points rendered for fixture).

**Phase 4 — UI: per-scenario charts**
1. `<ScenarioVolumeChart />` (bar): outbound/inbound counts per scenario.
2. `<ScenarioErrorRateChart />` (bar or horizontal): error rate per scenario, color-coded by threshold (matches table's `> 0.1` red).
3. Mount on ScenarioMetricsPage above the existing table; both share the same `useQuery` cache key.
4. Component tests.

**Phase 5 — UI: editable SettingsPage** (gated by d3)
1. Add `fetchSettings`, `updateSettings` to `lib/api.ts`.
2. Replace static panels in `SettingsPage.tsx` with a controlled form (number inputs, units shown alongside).
3. `useMutation` with optimistic UI; on success invalidate `["settings"]`; show success toast (or inline confirmation banner).
4. Validation mirrors backend rules; show inline error from 400 response.
5. Tests: render, fill form, submit, assert mutation called with correct body, assert error path.

**Phase 6 — Audit + reconcile requirements**
1. Walk every page in the router and verify it loads, renders an empty state, and renders with fixture data without console errors.
2. Replace `requirements-coverage.test.ts`'s factory-shape assertions with assertions that actually render the page and verify chart/form presence (or update validation refs to point at the new component tests).
3. For any `status: "complete"` requirement still unbacked by a feature-level test, either add the test or downgrade the status (not silently leave it as "complete").

**Phase 7 — Tests for previously-untested pages**
1. Add `StreamPage.test.ts` covering: filter input updates URL query string, pause toggles SSE delivery, clear empties the buffer.
2. Add `EventLogPage.test.ts` covering: query params propagate to `fetchEvents`, table rows render, error state shows ErrorAlert.
3. (Optional, gated by d5) One Playwright spec hitting the running scenario for the golden path: dashboard loads → live stream shows a synthetically ingested event → SettingsPage edit persists.

**Phase 8 — Polish, lint, docs**
1. `pnpm --dir scenarios/vrooli-events/ui lint && pnpm --dir scenarios/vrooli-events/ui type-check && pnpm --dir scenarios/vrooli-events/ui test`.
2. `cd scenarios/vrooli-events/api && gofumpt -w . && golangci-lint run && go test ./... -timeout 300s`.
3. Update `scenarios/vrooli-events/docs/reference/api-endpoints.md` with the new `/api/v1/settings` (and stats endpoint if d4=B).
4. Update `scenarios/vrooli-events/docs/reference/configuration.md` to reflect that retention is now runtime-configurable.

## 8. Contract Decisions

### `GET /api/v1/settings`
Response 200:
```json
{
  "retentionDays": 30,
  "retentionBytes": 2147483648,
  "pruneIntervalSeconds": 21600
}
```

### `PUT /api/v1/settings`
Request body (partial updates allowed):
```json
{ "retentionDays": 14 }
```
Response 200: full updated settings (same shape as GET).
Response 400: `{"error":"retentionDays must be > 0","code":"INVALID_SETTINGS"}`

### `GET /api/v1/events/stats` (only if d4=B)
Query: `bucket=minute|hour|day` (default `minute`), `since` (RFC3339 or unix), `until` (optional, default now), `source` (optional glob).
Response 200: `[{"bucketStart":"2026-04-18T13:00:00Z","count":42}, ...]`

### Charts (assumes d2=Recharts)
- `<ThroughputChart />` — `<LineChart>` or `<AreaChart>`, X = time bucket, Y = event count.
- `<ScenarioVolumeChart />` — `<BarChart>`, X = scenario, Y = count, two series (outbound, inbound).
- `<ScenarioErrorRateChart />` — `<BarChart>` of error %, threshold band at 10%.

## 9. Data Model

```sql
CREATE TABLE IF NOT EXISTS app_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
-- bootstrap on startup if missing:
-- ('retention_days', '30'), ('retention_bytes', '2147483648'), ('prune_interval_seconds', '21600')
```

Pruner reads these on each tick; no in-memory cache invalidation needed (one read per cycle is cheap).

## 10. Testing Plan

| Layer | Test | How |
|---|---|---|
| Backend | settings CRUD round-trip | Real SQLite, httptest.Server |
| Backend | settings validation rejects negative/zero/huge values | Table-driven |
| Backend | pruner picks up new retention without restart | Insert old events, change retention, run prune, assert |
| Backend (if d4=B) | events/stats bucketing | Real SQLite with seeded events across buckets |
| UI | ThroughputChart renders for fixture data and empty data | `@testing-library/react` + jest-dom |
| UI | ScenarioVolumeChart and ScenarioErrorRateChart render expected bars | Snapshot-free assertions on accessible names / data-testids |
| UI | SettingsPage form submit calls updateSettings, optimistic update, error path | `userEvent` + mocked `fetch` |
| UI | StreamPage filter persistence in URL, pause/clear behavior | `userEvent`, mock `EventSource` |
| UI | EventLogPage query param propagation | Mocked fetch, assert request URL |
| UI | requirements-coverage suite asserts feature presence (not just factory shapes) | Render page, assert chart svg exists, assert form fields exist |
| E2E (if d5 includes Playwright) | Dashboard golden path | Start scenario via `make start`, drive Playwright against it |

## 11. Rollout / Validation Checklist
- [ ] `pnpm --dir scenarios/vrooli-events/ui install` succeeds with the new chart library
- [ ] `pnpm --dir scenarios/vrooli-events/ui type-check && pnpm --dir scenarios/vrooli-events/ui lint && pnpm --dir scenarios/vrooli-events/ui test` all pass
- [ ] `cd scenarios/vrooli-events/api && go build ./... && go test ./... -timeout 300s` passes
- [ ] `gofumpt -l scenarios/vrooli-events/api` reports nothing; `golangci-lint run` clean
- [ ] Manually verify `make start` brings up scenario; UI loads at the dev URL; AnalyticsPage shows chart; SettingsPage saves and persists across restart; CompliancePage table renders
- [ ] Every `status: "complete"` in `requirements/05-ui-dashboard/module.json` either has a feature-level validation test or has been downgraded
- [ ] User does NOT run `vrooli scenario restart vrooli-events` from inside this Claude Code session — code is written to disk; user restarts manually

## 12. Risks + Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Adding a chart library bloats the UI bundle | Slower first paint | Recharts tree-shakes well; verify bundle size after; lazy-load AnalyticsPage and ScenarioMetricsPage if needed |
| Pruner reading settings every cycle could cause SQLite contention | Theoretical write blocking | Settings table is tiny and read-only on the hot path; one read per 6h cycle is negligible |
| Editable retention can be set to absurd values that fill disk or wipe history | Operational footgun | Validation: 1 ≤ retentionDays ≤ 3650, retentionBytes ≥ 1MB ≤ 100GB, pruneIntervalSeconds ≥ 60 |
| Existing requirements marked "complete" hide bugs | Drift between docs and code | Phase 6 reconciles status; CI test asserts feature-level validation refs exist |
| Failed `vrooli-events-policy-api-and-middleware` may mean policy endpoints behave incorrectly | Compliance/Policies pages render against bad data | Smoke-test the read paths during the audit; if endpoints are broken, file a separate fix item rather than expanding scope here |
| Throughput rollup endpoint duplicates aggregation logic with client | Maintenance burden | Pick ONE source per d4; do not implement both |
| Restarting the scenario from inside the agent session would kill the process running it | Crash, lost work | Per `feedback_no_restart_active_scenario`: write code only, instruct user to restart manually |

## 13. Non-goals / Prohibited Patterns
- Do NOT add authentication or authorization on any endpoint in this item.
- Do NOT modify the discovery package — that's `execute/discovery-event-emission-and-policy-cache`.
- Do NOT touch the SQLite schema for `events` or `store_meta` — only add `app_settings`.
- Do NOT install both Recharts and another chart library; pick one (per d2).
- Do NOT mock SQLite in backend tests — use real SQLite.
- Do NOT issue `vrooli scenario restart vrooli-events` from inside this session.
- Do NOT create a `lib/` folder in the scenario — use the v2.0 service.json lifecycle that already exists.
- Do NOT silently leave a `status: "complete"` requirement that has no feature-level test backing it.
- Do NOT compute proxy URLs client-side; use the existing `lib/api.ts` helpers.

## 14. Definition of Done
- [ ] Charts render on AnalyticsPage (throughput) and ScenarioMetricsPage (volume + error rate).
- [ ] SettingsPage edits retention/size/interval; values persist in SQLite; pruner uses live values.
- [ ] `GET /api/v1/settings` and `PUT /api/v1/settings` implemented and tested.
- [ ] If d4=B: `GET /api/v1/events/stats` implemented and tested.
- [ ] StreamPage and EventLogPage have unit/component tests covering golden paths.
- [ ] requirements-coverage suite asserts rendered features, not just factory shapes.
- [ ] `requirements/05-ui-dashboard/module.json` accurately reflects implementation.
- [ ] All UI tests pass; all API tests pass; lint and format clean.
- [ ] Docs updated under `scenarios/vrooli-events/docs/reference/`.
- [ ] Manual smoke confirms the dashboard works end-to-end against a freshly restarted scenario (user-initiated restart).
