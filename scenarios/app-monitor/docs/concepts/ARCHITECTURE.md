# App Monitor Architecture

## Product and ownership boundary

App Monitor is a legacy implementation reference while [Portal](../../../portal/docs/concepts/SUCCESSOR-PATH.md) matures as the operator-facing suite surface. Monitoring and host-operation authority remain with their existing owners; see the current [scenario scope](../../README.md). App Monitor’s API delegates scenario start, stop, and restart to the Vrooli CLI rather than owning scenario lifecycle state.

## Runtime shape

- [`api/main.go`](../../api/main.go) runs the shared preflight, loads environment-backed configuration, constructs the server, and starts it.
- [`api/server.go`](../../api/server.go) composes the Gin router, HTTP handlers, services, and optional database, Redis, and Docker clients. Handlers adapt HTTP requests; services coordinate behavior; repository implementations own persistence details.
- [`api/services/app_service.go`](../../api/services/app_service.go) delegates scenario lifecycle commands to the Vrooli CLI. PostgreSQL, when configured, backs application records and view statistics through [`api/repository/postgres.go`](../../api/repository/postgres.go) and the scenario-owned schema.
- Workspace presets use JSON files under App Monitor’s data directory through the shared `api-core/storage` resolver, not the application database; see [`api/repository/preset_fs.go`](../../api/repository/preset_fs.go).
- Redis Pub/Sub on `app-events` feeds WebSocket updates when Redis is available. Without it, the WebSocket remains available for client messages but receives no Redis events. Docker integration is also optional and is composed by the server.

## UI and preview boundaries

The React router in [`ui/src/App.tsx`](../../ui/src/App.tsx) keeps the single-preview URL as a compatibility redirect into the workspace. A workspace pane owns its selected scenario, URL history, bridge state, iframe and logs mode; the workspace store owns pane collection, focus, layout and persisted workspace preferences. The [preview workspace guide](../preview-workspace.md) describes those routes and state boundaries. The [seam map](../internal/SEAMS.md) records the more detailed bridge, pane-reset, self-embedding, and rendering contracts.

App Monitor’s document API scans its supported root documents and the `docs/` tree; the UI presents those groups separately. The manifest’s `docs/concepts/ARCHITECTURE.md` path is the maintained architecture entry. A short root-file pointer remains for clients that still request the former `ARCHITECTURE.md` path.

## Configuration, persistence, and host assumptions

The scenario lifecycle supplies `API_PORT`. Database, Redis, Docker, CORS, WebSocket-origin, rate-limit, and optional API-key settings are read from configuration and environment; their implementation defaults belong to [`api/config/config.go`](../../api/config/config.go) and [`api/middleware/security.go`](../../api/middleware/security.go). When optional services are unavailable, startup continues with reduced functionality and the dependent operations can fail or omit their data.

[`api/services/metrics_service.go`](../../api/services/metrics_service.go) collects host-level CPU, memory, disk, and network readings concurrently and caches the latest result for five seconds. Memory uses Vrooli host inventory; CPU and root-disk probes use Linux-specific interfaces (`/proc/stat`, `top`, and `df /`). This source describes observed implementation, not a cross-platform support promise; non-Linux runtime support has not been verified here.

## Source map

| Concern | Owner source | Consumer path |
|---|---|---|
| API startup, routes, and optional services | `api/main.go`, `api/server.go`, `api/config/config.go` | App Monitor HTTP API and health endpoints |
| Scenario lifecycle and application data | `api/services/app_service.go`, `api/repository/postgres.go` | App handlers and UI application actions |
| Workspace preset persistence | `api/repository/preset_fs.go`, `api-core/storage` | Workspace preset API and UI |
| Preview and workspace interaction | `ui/src/App.tsx`, `ui/src/features/preview-workspace/` | Single-preview compatibility route and workspace panes |
| Host metrics and event stream | `api/services/metrics_service.go`, `api/handlers/websocket.go` | System metrics API and dashboard WebSocket |

This page is the architecture hub; it links to the focused preview guide and seam map instead of repeating their detailed transition and bridge rules.
