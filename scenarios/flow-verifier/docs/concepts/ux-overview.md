# UX Overview

## Purpose Of This Document

Map every visible route in the `flow-verifier` UI to the data it depends on, the
backend endpoints it consumes, and a textual description of what a user sees on
the screen. This is the canonical contract for the UI surface — when a new route
is added, this document is updated in the same change.

The UI is a React 18 + Vite SPA at `scenarios/flow-verifier/ui/`. It uses
`react-router-dom` v7 for routing, `@tanstack/react-query` for server-state
caching, `@xyflow/react` + `dagre` for the state-graph visualization, and
`recharts` for the run-outcome timeline. All screens consume the JSON HTTP API
exposed by `scenarios/flow-verifier/api/` (see
[`reference/api-endpoints.md`](../reference/api-endpoints.md)) via the typed
client in `ui/src/api/inventory.ts`.

## Route Map

| Path | Page component | Primary data source |
|---|---|---|
| `/` | `Dashboard` (composed in `App.tsx`) | `GET /health`, `GET /api/v1/runs` |
| `/flows` | `InventoryCard` (`features/inventory/`) | `GET /api/v1/flows`, `GET /api/v1/runs` |
| `/flows/:flowId` | `FlowDetailPage` (`features/flow-detail/`) | `GET /api/v1/flows/:id`, `GET /api/v1/runs?flowId=` |
| `/runs/:runId` | `RunDetailPage` (`features/run-detail/`) | `GET /api/v1/runs/:id` |

Every route is wrapped in a per-route `<ErrorBoundary>` (`components/ErrorBoundary.tsx`)
so a render-time crash inside one screen leaves the shell and navigation intact.
The shell, navigation, and locale switcher live in `components/AppShell.tsx` and
persist across navigations.

## `/` — Dashboard

**Purpose.** First-touch landing surface. Confirms the API is reachable and
gives an at-a-glance view of recent verification activity across all flows.

**Data dependencies.**
- `GET /health` (via `features/health/HealthCard.tsx`) — surfaces the API
  `status` field; renders an error chip on non-200.
- `GET /api/v1/runs` (via `features/timeline/TimelineCard.tsx`, default
  `limit=200`) — client-side bucketing by `YYYY-MM-DD` of `finishedAt`, stacked
  by status (`passed` / `failed` / `error`).

**Screenshot description.** The shell header is the eyebrow + nav row
(`Dashboard / Flows`) on a tinted-blur background. The main panel is a vertical
stack: the `HealthCard` (compact, status-chip + last-checked timestamp), then
the `TimelineCard` (stacked bar chart, ~256 px tall, with `passed`/`failed`/`error`
legend). When the API has fewer than two distinct days of runs, the
`TimelineCard` collapses to an empty-state strip instead of rendering the chart.

**Empty / loading / error states.**
- `HealthCard`: `health-loading` while in flight, `health-error` on failure.
- `TimelineCard`: `timeline-loading`, `timeline-error`, `timeline-empty`,
  `timeline-chart` (see `features/timeline/TimelineCard.tsx`).

## `/flows` — Inventory

**Purpose.** Browse every flow the verifier has discovered under the current
root, with the most recent run status for each, and trigger a verification
(per row or for the whole list).

**Data dependencies.**
- `GET /api/v1/flows?root=<root>` — flow list.
- `GET /api/v1/runs` — joined client-side by `flowId` to surface the most
  recent run.
- `POST /api/v1/verifications` — triggered by both the per-row "Verify" button
  and the "Verify all" button.

**Screenshot description.** A two-column layout: a small root-path input + Reload
+ "Verify all" controls strip on top, then a table of flow rows. Each row is a
`<Link>` to `/flows/:flowId` plus a status chip (color-coded `passed` /
`failed` / `error`) plus a per-row "Verify" button (disabled while
`Verify all` is pending — see the `ui-health` skill (load via
`prompt-manager skill read ui-health`) §1 for the scope-discipline
rationale.

**Empty / loading / error states.** `inventory-loading`, `inventory-error`,
`inventory-empty`, `inventory-card`.

## `/flows/:flowId` — Flow Detail

**Purpose.** Open the model behind a single flow. Three tabs:
1. **Graph** — `StateGraph.tsx` renders the state machine as a `@xyflow/react`
   graph; nodes = states, edges = transitions; layout via `dagre`. Selecting a
   node highlights incident edges. Keyboard navigable.
2. **Traces** — `TracePlayer.tsx` picks a named trace from
   `flow/generated/artifact.json.traces` and walks through states with
   play / pause / step / reset. The shared step index drives the graph
   highlight via a parent state prop (no Context — scope discipline).
3. **History** — embeds `TimelineCard` scoped via `flowId` prop, plus a list of
   runs linking to `/runs/:runId`. When a failing run is selected,
   `CounterexampleDiff.tsx` renders expected (model) vs actual (counterexample)
   transitions side-by-side.

**Data dependencies.**
- `GET /api/v1/flows/:id` — surfaces `transitionMatrix`, `traces`, language,
  status, schema version, initial state, states/events/transitions.
- `GET /api/v1/runs?flowId=:id` — drives the History tab and the embedded
  `TimelineCard`.

**Screenshot description.** Header with the flow id (monospace), language chip,
status chip, schema version, and a "Back to inventory" link. A horizontal
tab row under the header (`Graph / Traces / History`). The Graph tab is a
full-width canvas (~480 px tall) with the state machine laid out left-to-right.
The Traces tab is a transport-control strip (drop-down + play/pause/step
buttons) above a step-by-step transition table; the graph mirrors the selected
step. The History tab is the `TimelineCard` (scoped to this flow) followed by
a table of runs.

**Empty / loading / error states.** `flow-detail-loading`, `flow-detail-error`,
`flow-detail-missing` (no `:flowId`), `flow-detail-page`. Each tab has its own
empty state (`flow-detail-graph-empty`, `flow-detail-traces-empty`,
`flow-detail-history-empty`).

## `/runs/:runId` — Run Detail

**Purpose.** Inspect a single verification run, including its counterexample
when one exists.

**Data dependencies.**
- `GET /api/v1/runs/:id` — id, flow id, flow path, root, mode, status,
  startedAt, finishedAt, durationMs, optional errorMessage, optional
  `counterexample` (JSON string).

**Screenshot description.** A single-column page. Header surfaces the run id
(monospace), a `<Link>` to the parent flow, the run mode, a color-coded status
(`passed` = emerald, `failed` = red, otherwise amber), and the duration in ms.
A "Back to inventory" link sits on the right of the header. Below the header is
a definition-list grid (`<dl>`) with started / finished / root / flow path and,
when present, an `errorMessage` row in red. The bottom of the page is the
collapsible counterexample tree: a top-level `<details open>` element titled
"Counterexample" wrapping a recursive `JsonNode` (auto-opens up to depth 2;
deeper levels render but stay collapsed by default to keep large blobs fast —
see plan risk #5 for the >500 KB rationale). When the JSON fails to parse, the
parse error is surfaced inline (`run-detail-counterexample-parse-error`).

**Empty / loading / error states.** `run-detail-loading`, `run-detail-error`
(with back-link), `run-detail-missing` (no `:runId`), `run-detail-page`,
`run-detail-no-counterexample` (passed run with no `counterexample`),
`run-detail-counterexample`, `run-detail-counterexample-parse-error`.

## Cross-References

- [`reference/api-endpoints.md`](../reference/api-endpoints.md) — JSON contract
  for every endpoint consumed above.
- [`reference/cli-commands.md`](../reference/cli-commands.md) — the CLI surface
  that produces the rows visible in `/flows` and `/runs/:id`.
- [`concepts/ARCHITECTURE.md`](ARCHITECTURE.md) — overall scenario architecture
  (API / CLI / UI split).
- [`concepts/FLOWS.md`](FLOWS.md) — the underlying flow model the UI visualizes.
- `ui-health` skill (load via `prompt-manager skill read ui-health`)
  — scope-discipline rules every screen follows (useReducer over Context,
  parent-driven step state for the trace player, etc.).
- `ui-health` skill (load via `prompt-manager skill read ui-health`)
  — basename + router slot the UI binds to so the proxied URL works under any
  deployment.
