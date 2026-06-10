# API Endpoints — Ecosystem Manager

Human-readable reference for the Ecosystem Manager API.

> **Transport note (pre-proto scenario).** Ecosystem Manager predates
> the current Vrooli standard (proto + Connect-RPC). It serves
> **REST/JSON over `gorilla/mux`**. There is no `.proto` contract and
> no generated client — handler structs are hand-written, so API/UI/CLI
> shapes can drift. This is tracked drift, not the target architecture.
> See [`../internal/COHERENCE-NOTES.md`](../internal/COHERENCE-NOTES.md)
> and [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) for the
> drift / migration notes.

| | |
|---|---|
| **Base URL** | `http://localhost:30500/api` |
| **Dashboard** | `http://localhost:30500` |
| **Health** | `http://localhost:30500/health` |
| **Transport** | REST/JSON over `gorilla/mux` |
| **Auth** | None (local-only operational tool) |

Routes are registered in [CODE: api/pkg/server/server.go] (~lines
360–532). Each group has a `register<Group>Routes(api *mux.Router)`
method mounted under the `/api` subrouter. CORS, request-ID, recovery,
and request-logging middleware wrap the whole router. A WebSocket
endpoint is mounted at the router root (`/ws`), outside `/api`.

Errors are returned as JSON with the relevant HTTP status code
(`400` malformed request, `404` not found, `409`/`429` conflict or
rate-limited, `500` internal). The exact envelope is handler-defined
(no shared proto error type).

---

## System

### `GET /health`

Service health check used by the lifecycle, load balancers, and curl
probes. Returns `200` with a status body.

| | |
|---|---|
| **Auth** | None |
| **CLI** | `ecosystem-manager status` |

```bash
curl "http://localhost:30500/health"
```

### `WS /ws`

WebSocket channel for live queue/task/log updates pushed to the UI.
Mounted at the router root (not under `/api`).
Handled by [CODE: api/pkg/server/server.go] (`a.wsManager.HandleWebSocket`).

---

## Tasks & Queue

Tasks model resource/scenario **generation** and **improvement** work.
The on-disk queue is stored as YAML under `queue/<status>/`; execution
history and analytics persist to Postgres (`task_executions`,
`operation_metrics`). Handlers:
[CODE: api/pkg/server/server.go] (`registerTaskRoutes`,
`registerPromptRoutes`, `registerQueueRoutes`).

### Tasks

| Method & path | Purpose |
|---|---|
| `GET /api/tasks` | List tasks (filter by `status`, `type`) |
| `POST /api/tasks` | Create a task |
| `GET /api/tasks/active-targets` | Targets currently being worked on |
| `GET /api/tasks/{id}` | Get task detail |
| `PUT /api/tasks/{id}` | Update a task |
| `DELETE /api/tasks/{id}` | Delete a task |
| `PUT /api/tasks/{id}/status` | Update task status |
| `PUT /api/tasks/{id}/queue-position` | Reorder a task in the queue |
| `GET /api/tasks/{id}/logs` | Task log lines |
| `GET /api/tasks/{id}/executions` | Execution history for the task |
| `GET /api/tasks/{id}/executions/bulk-analysis` | Aggregate analysis across executions |
| `GET /api/tasks/{id}/executions/{execution_id}/prompt` | Prompt used by an execution |
| `GET /api/tasks/{id}/executions/{execution_id}/output` | Raw agent output |
| `GET /api/tasks/{id}/executions/{execution_id}/metadata` | Execution metadata |
| `GET /api/executions` | Global execution history (all tasks) |

### Prompts

| Method & path | Purpose |
|---|---|
| `GET /api/tasks/{id}/prompt` | Task prompt template |
| `GET /api/tasks/{id}/prompt/assembled` | Fully assembled prompt |
| `POST /api/prompt-viewer` | Render an ad-hoc prompt preview |
| `GET /api/prompts` · `POST /api/prompts` | List / create prompt files |
| `GET\|PUT /api/prompts/{path:.*}` | Read / update a prompt file by path |

### Queue & processes

| Method & path | Purpose |
|---|---|
| `GET /api/queue/status` | Queue depth + processor state |
| `GET /api/queue/resume-diagnostics` | Why a paused queue is/isn't resuming |
| `POST /api/queue/trigger` | Kick the processor once |
| `POST /api/queue/start` · `POST /api/queue/stop` | Start / stop the processor |
| `POST /api/queue/reset-rate-limit` | Clear the rate-limit cooldown |
| `GET /api/processes/running` | Currently running agent processes |
| `POST /api/queue/processes/terminate` | Terminate a running process |
| `POST /api/maintenance/state` | Set maintenance mode |

```bash
curl -X POST http://localhost:30500/api/tasks \
  -H 'Content-Type: application/json' \
  -d '{"type":"scenario","operation":"generate","target":"my-app"}'
```

---

## Auto-Steer

Auto-Steer drives iterative, phase-based agent runs against a task using
a tunable **profile**. Profiles are stored on the filesystem under
`profiles/<id-or-name>/profile.json`, indexed by `profiles/metadata.json`.
Run state and history persist to Postgres (`profile_executions`,
`profile_execution_state`, `steering_queue_state`,
`execution_feedback_entries`). Handlers:
[CODE: api/pkg/server/server.go] (`registerAutoSteerRoutes`).

### Profiles & templates

| Method & path | Purpose |
|---|---|
| `POST /api/auto-steer/profiles` | Create a profile |
| `GET /api/auto-steer/profiles` | List profiles |
| `GET\|PUT\|DELETE /api/auto-steer/profiles/{id}` | Read / update / delete a profile |
| `GET /api/auto-steer/templates` | Built-in profile templates |

### Execution control

| Method & path | Purpose |
|---|---|
| `POST /api/auto-steer/execution/start` | Begin a steered run |
| `POST /api/auto-steer/execution/evaluate` | Score the current iteration |
| `POST /api/auto-steer/execution/advance` | Advance to the next phase |
| `POST /api/auto-steer/execution/seek` | Jump to a specific phase |
| `POST /api/auto-steer/execution/reset` | Reset the run |
| `GET /api/auto-steer/execution/{taskId}` | Current execution state for a task |
| `GET /api/auto-steer/metrics/{taskId}` | Live metrics for a task's run |

### History, feedback & analytics

| Method & path | Purpose |
|---|---|
| `GET /api/auto-steer/history` | List past executions |
| `GET /api/auto-steer/history/{executionId}` | One execution's detail |
| `POST /api/auto-steer/history/{executionId}/feedback` | Submit run-level feedback |
| `POST /api/auto-steer/history/{executionId}/feedback/entries` | Append a feedback entry |
| `GET /api/auto-steer/analytics/{profileId}` | Aggregate analytics for a profile |

---

## Settings, Discovery & Insights

### Settings

Handlers: [CODE: api/pkg/server/server.go] (`registerSettingsRoutes`).

| Method & path | Purpose |
|---|---|
| `GET /api/settings` | Read settings |
| `PUT /api/settings` | Update settings |
| `POST /api/settings/reset` | Reset to defaults |
| `GET /api/settings/recycler/models` | Models available to the task recycler |

### Discovery

Read-only views over the local resource/scenario landscape.
Handlers: [CODE: api/pkg/server/server.go] (`registerDiscoveryRoutes`).

| Method & path | Purpose |
|---|---|
| `GET /api/resources` | List local resources |
| `GET /api/scenarios` | List scenarios |
| `GET /api/resources/{name}/status` | One resource's status |
| `GET /api/scenarios/{name}/status` | One scenario's status |
| `GET /api/operations` | Supported operation types |
| `GET /api/categories` | Target categories |

### Insights

AI-generated improvement reports and applicable suggestions, scoped to a
task or to the whole system. Handlers:
[CODE: api/pkg/server/server.go] (`registerInsightRoutes`).

| Method & path | Purpose |
|---|---|
| `GET /api/tasks/{id}/insights` | Task insight reports |
| `GET /api/tasks/{id}/insights/preview` | Preview the insight prompt |
| `POST /api/tasks/{id}/insights/generate` | Generate a new insight report |
| `GET /api/tasks/{id}/insights/{report_id}` | One report |
| `POST .../insights/{report_id}/suggestions/{suggestion_id}/status` | Set a suggestion's status |
| `POST .../insights/{report_id}/suggestions/{suggestion_id}/apply` | Apply a suggestion |
| `GET /api/insights/system` | System-level insights |
| `POST /api/insights/system/generate` | Generate system-level insights |

### Importance

Derived scenario importance scores for fleet maintenance ordering. The
composer combines scenario-dependency-analyzer centrality, core-set proximity,
recent swarm-manager operations, and the `system_required` manifest floor.
Missing optional inputs degrade to neutral scores and are listed in the
response.

| Method & path | Purpose |
|---|---|
| `GET /api/importance` | Cached derived importance scores for all scenarios |
| `GET /api/importance?refresh=true` | Force a fresh read of optional inputs |

### Logs, Skills & visited-tracker proxy

| Method & path | Purpose |
|---|---|
| `GET /api/logs` | API log lines |
| `GET /api/skills` · `POST /api/skills/sync` | List / sync skills |
| `GET\|DELETE\|POST /api/visited-tracker/...` | Proxy endpoints into the visited-tracker scenario |

---

## Adding a new endpoint

1. Implement the handler on its handler struct in
   [CODE: api/pkg/server/server.go] (or a dedicated handlers package).
2. Register the route in the matching `register<Group>Routes` method in
   [CODE: api/pkg/server/server.go], choosing the HTTP method(s) and a
   path under `/api`.
3. If the endpoint should map to the CLI, add the command per
   [`cli-commands.md`](cli-commands.md#adding-a-new-command).
4. Update [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) so the
   endpoint manifest stays in sync; document the endpoint here.
5. Add handler tests next to the handler.

> Because this scenario has no proto contract, there is no codegen step
> and no generated client to update — but there is also no compiler-
> enforced API/UI/CLI parity. Keep this doc, `endpoints.json`, and the
> CLI in lockstep by hand. New API surfaces should prefer proto +
> Connect-RPC per the project standard; see
> [`../internal/COHERENCE-NOTES.md`](../internal/COHERENCE-NOTES.md).

---

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars, ports, database, profiles registry
- [`ui-manifest.md`](ui-manifest.md) — UI structure (pre-adoption status)
- [`../internal/COHERENCE-NOTES.md`](../internal/COHERENCE-NOTES.md) — REST-vs-proto drift note
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — known issues
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system design
