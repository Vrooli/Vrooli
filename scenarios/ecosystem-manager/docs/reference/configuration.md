# Configuration — Ecosystem Manager

How Ecosystem Manager is configured — env vars consumed by the
binaries, the `.vrooli/service.json` manifest, the PostgreSQL schema,
the on-disk auto-steer profiles registry, and the per-user CLI config
file.

The lifecycle (`vrooli scenario start ecosystem-manager`, `make start`)
sets every required variable automatically. You only need this
reference when running a binary by hand or adding a new variable.

| | |
|---|---|
| **Lifecycle port** | `30500` (API + UI dashboard) |
| **Dashboard** | `http://localhost:30500` |
| **Database** | PostgreSQL `vrooli_ecosystem_manager` |

## Environment variables

### Required at runtime (set by the lifecycle)

| Variable | Range / value | Purpose |
|---|---|---|
| `API_PORT` | `15000-19999` | Port for the Go API server (lifecycle port `30500`) |
| `UI_PORT` | `21110` (declared) | Port for the production UI server |
| `POSTGRES_*` | — | Connection params for `vrooli_ecosystem_manager`, read by `database.Connect` in [CODE: api/pkg/server/server.go] (`initializeDatabase`) |

### Optional overrides

| Variable | Default | Purpose |
|---|---|---|
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated allowed origins for API CORS. Empty falls back to `*`. Consumed via `allowedOrigins` in [CODE: api/pkg/server/server.go] |
| `OPENROUTER_API_KEY` | unset | Enables OpenRouter-backed recycler runs and model discovery; absent → that path is degraded, not fatal |

### Scenario-prefixed CLI variables

`cli-core` derives env vars from the scenario name. The prefix is the
id upper-cased with hyphens → underscores: `ECOSYSTEM_MANAGER`.
First-found wins, in this order:

| Purpose | Variables |
|---|---|
| API base URL | `ECOSYSTEM_MANAGER_API_BASE`, `ECOSYSTEM_MANAGER_API_URL`, `API_BASE_URL`, `VITE_API_BASE_URL`, `VROOLI_API_BASE` |
| API port | `ECOSYSTEM_MANAGER_API_PORT` |
| Config dir | `ECOSYSTEM_MANAGER_CONFIG_DIR`, `VROOLI_CLI_CONFIG_DIR` |
| HTTP timeout | `ECOSYSTEM_MANAGER_HTTP_TIMEOUT`, `VROOLI_HTTP_TIMEOUT` |

> The extra `API_BASE_URL` / `VITE_API_BASE_URL` aliases are declared via
> `ExtraAPIEnvVars` in [CODE: cli/app.go].

## Service manifest (`.vrooli/service.json`)

Single source of truth for the lifecycle. Schema version `2.0.0`.

| Section | Owns |
|---|---|
| `service` | name (`ecosystem-manager`), displayName (`Ecosystem Manager`), description, version, tags, category, maintainers, repository |
| `ports` | `api` (`API_PORT`, range `15000-19999`) and `ui` (`UI_PORT`, `21110`) |
| `hostTools` | optional `bats` for shell-test introspection during analysis |
| `cli` | command name, `go_module` adapter (`module_dir: cli`), per-OS install scripts, freshness inputs |
| `lifecycle.health` | API `/health` (critical) + UI `/health` (non-critical), 15s startup grace |
| `lifecycle.setup` | build the API binary, CLI, UI bundle; ensure `queue/*` directories |
| `dependencies.resources` | shared resources (below) |
| `dependencies.scenarios` | scenario dependencies (below) |

### Resource dependencies

| Resource | Required | Role |
|---|---|---|
| `postgres` | yes | Task queue history + analytics + auto-steer state |
| `qdrant` | yes | Semantic search / similarity matching |
| `claude-code` | yes | AI for resource/scenario generation |
| `ollama` | yes | Local models (`llama3.1:8b`, `llama3.2:3b`) |
| `minio` | yes | Object storage |
| `openrouter` | no (`try_start`) | Optional cloud model provider for recycler/model discovery |
| `redis` | no (disabled) | Optional caching layer |

### Scenario dependencies

| Scenario | Required | Role |
|---|---|---|
| `agent-manager` | yes | Centralized agent orchestration — executes all agent runs |
| `prompt-manager` | no | Steering prompts for Auto-Steer (graceful degradation) |
| `visited-tracker` | no | Campaign tracking, proxied via `/api/visited-tracker/*` |

> `scenario-completeness-scoring` is intentionally not listed as a required
> scenario dependency. It is the fast cached status reader over shared
> maturity/freshness packages, while Ecosystem Manager computes its current
> PRD completion metric locally from target `PRD.md` files.

## Runtime settings

`GET/PUT /api/settings` manages persisted operator settings. The queue
processor always starts inactive after process restart; activating it is
an explicit operator action.

| Field | Default | Purpose |
|---|---|---|
| `active` | `false` | Enables queue processing when the processor is also running. |
| `slots` | `1` | Concurrent task slots. |
| `cooldown_seconds` | `30` | Delay between automatic queue actions. |
| `execution_limit` | `0` | Auto-stop after this many completed executions; `0` means unlimited. |
| `importance_aware_scheduling` | `false` | Default-off queue ordering: when enabled, pending scenario tasks are ranked by derived scenario importance x maturity gap, with task priority as the tie-breaker. Missing signals degrade to neutral. |

## Database & schema bootstrap

PostgreSQL database `vrooli_ecosystem_manager`. The schema is declared
in [CODE: initialization/postgres/schema.sql] and applied idempotently
(`CREATE EXTENSION/TABLE IF NOT EXISTS`). `database.Connect` reads
`POSTGRES_*` env vars and configures a 25/5 connection pool
([CODE: api/pkg/server/server.go], `initializeDatabase`).

| Table | Purpose |
|---|---|
| `task_executions` | Execution history for analytics & learning |
| `operation_metrics` | Aggregate operation analytics |
| `profile_executions` | Auto-steer run records |
| `profile_execution_state` | Live auto-steer run state |
| `steering_queue_state` | Steering queue persistence |
| `execution_feedback_entries` | Per-run feedback entries |

Filesystem state lives alongside the database: the task queue is YAML
under `queue/<status>/`, and auto-steer profiles under
`profiles/<id-or-name>/profile.json`.

## Profiles registry

Auto-steer profiles are stored on disk (not in Postgres), under
[CODE: profiles]:

```
profiles/
  metadata.json                  # index of all profiles (id → name, path)
  <id-or-name>/profile.json      # one profile definition
```

`metadata.json` is the index; the API resolves a profile id/name to its
`profile.json`. Run state for an active profile execution is the
Postgres half (`profile_execution_state` / `steering_queue_state`).
Repo path: `scenarios/ecosystem-manager/profiles/`.

## CLI config file

The CLI persists per-user config to a JSON file. Resolution order
(first match wins):

1. `${ECOSYSTEM_MANAGER_CONFIG_DIR}/config.json`
2. `${XDG_CONFIG_HOME}/vrooli/ecosystem-manager/config.json`
3. `~/.vrooli/config/ecosystem-manager/config.json`
4. `~/.config/vrooli/ecosystem-manager/config.json`

File shape:

```json
{
  "api_base": "http://localhost:30500/api",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file:

```bash
ecosystem-manager configure api_base http://localhost:30500/api
```

### API-base resolution precedence

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base`)
4. Vrooli lifecycle port detection
5. Compile-time default (empty unless set in [CODE: cli/app.go])

## Test/CI configuration

| File | Owns |
|---|---|
| [`.vrooli/testing.json`](../../.vrooli/testing.json) | Test categories run by `vrooli scenario test` |
| [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) | API endpoint manifest (kept in sync by hand — no proto codegen) |
| [`.vrooli/service.json`](../../.vrooli/service.json) | Lifecycle, ports, deps |

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — endpoint reference
- [`cli-commands.md`](cli-commands.md) — CLI command reference
- [`ui-manifest.md`](ui-manifest.md) — UI structure (pre-adoption status)
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — env/port/lifecycle fixes
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system design
