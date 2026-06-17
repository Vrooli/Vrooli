# API Endpoints — Image Tools

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/image-tools/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/image-tools/v1/errors/errors.proto`):

```json
{ "code": "<canonical_code>", "message": "<human readable>", "details": [...] }
```

Canonical REST codes used today: `invalid_request` (400),
`not_found` (404), `internal` (500). Add to the proto enum when a new
REST-exception failure mode appears.

> **Status:** Phase 1 exposes the spine surfaces — `jobs` (durable async
> job lifecycle) and `models` (model registry read + enable/disable).
> Image **operation** endpoints (generate / enhance / analyze) enqueue
> work on the job system and land from Phase 2; they will appear here as
> they ship.

---

## System

### `GET /health`

Service health check. Returns API readiness plus dependency status.
Also mounted at `/api/v1/health` for client callers.
This is an operational REST exception by design: lifecycle systems,
load balancers, and curl probes must be able to read it without a Connect
client.

| | |
|---|---|
| **Auth** | None |
| **Response** | `Response { status: string, readiness: bool, service: string, timestamp: string, version: string, uptime_seconds: int64, dependencies: map<string, DependencyStatus> }` |
| **Errors** | None — always returns 200 with `status: "unhealthy"` if a dependency fails |
| **CLI** | `image-tools status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/image-tools/v1/health/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Jobs (durable async lifecycle)

The `jobs` domain is the read + control surface over the server-owned
durable job system (`api/internal/jobs`). Jobs survive client
disconnects: a submit returns a job id + ETA immediately, work runs under
a server-lifetime context, callers block ONCE on `WaitJob` (never poll),
and progress streams over `WatchJob`. GPU jobs are serialized; CPU jobs
run concurrently. Job *submission* is op-specific and lands with each
operation domain from Phase 2.

Proto: `packages/proto/schemas/image-tools/v1/jobs/jobs.proto`.

### `POST /vrooli.image_tools.v1.jobs.JobsService/GetJob`

Return the current record for one job.

| | |
|---|---|
| **Request** | `GetJobRequest { id: string }` |
| **Response** | `GetJobResponse { job: Job }` |
| **Errors** | `not_found` — no job with that id |
| **CLI** | `image-tools jobs get <id>` |

### `POST /vrooli.image_tools.v1.jobs.JobsService/WaitJob`

Block server-side until the job reaches a terminal state, then return it.
A client disconnect does NOT affect the job. This is the canonical wait
verb — never poll `GetJob` in a loop.

| | |
|---|---|
| **Request** | `WaitJobRequest { id: string }` |
| **Response** | `WaitJobResponse { job: Job (terminal) }` |
| **Errors** | `not_found` — no job with that id<br>`canceled` — the client canceled the wait (the job continues server-side) |
| **CLI** | `image-tools jobs wait <id>` |

### `POST /vrooli.image_tools.v1.jobs.JobsService/ListJobs`

Return recent jobs, newest first.

| | |
|---|---|
| **Request** | `ListJobsRequest { limit: int32 (0 = server default) }` |
| **Response** | `ListJobsResponse { jobs: Job[] }` |
| **CLI** | `image-tools jobs list [--limit <n>]` |

### `POST /vrooli.image_tools.v1.jobs.JobsService/CancelJob`

Abort a job: a running job's context is canceled; a still-queued job is
marked canceled immediately. Returns the post-cancel record.

| | |
|---|---|
| **Request** | `CancelJobRequest { id: string }` |
| **Response** | `CancelJobResponse { job: Job }` |
| **Errors** | `not_found` — no job with that id |
| **CLI** | `image-tools jobs cancel <id>` |

### `POST /vrooli.image_tools.v1.jobs.JobsService/WatchJob` (server stream)

Stream progress events for a job until it reaches a terminal state. The
latest known event is replayed first so a late subscriber is never blind.
Backs SSE-style live progress in the UI.

| | |
|---|---|
| **Request** | `WatchJobRequest { id: string }` |
| **Response** | `stream ProgressEvent { job_id, state, progress, message, at }` |
| **Errors** | `not_found` — no job with that id |
| **CLI** | `image-tools jobs watch <id>` |

---

## Models (registry read + enable/disable)

The `models` domain is the read and enable/disable surface over the
declarative model registry (`api/internal/models`). The license-verified
seed catalog is the read-only baseline; runtime enable/disable state is
overlaid in SQLite. Heavier management (checksum-pinned download,
disk-space awareness, custom local entries, removal) lands in a later
phase.

Proto: `packages/proto/schemas/image-tools/v1/models/models.proto`.

### `POST /vrooli.image_tools.v1.models.ModelsService/ListModels`

Return catalog entries, optionally filtered to one operation, with
effective (overlay-aware) enabled state.

| | |
|---|---|
| **Request** | `ListModelsRequest { operation: string (optional) }` |
| **Response** | `ListModelsResponse { models: Model[] }` |
| **Errors** | `invalid_argument` — unknown operation filter |
| **CLI** | `image-tools models list [--operation <op>]` |

### `POST /vrooli.image_tools.v1.models.ModelsService/GetModel`

Return one catalog entry by id with effective enabled state.

| | |
|---|---|
| **Request** | `GetModelRequest { id: string }` |
| **Response** | `GetModelResponse { model: Model }` |
| **Errors** | `not_found` — no model with that id |
| **CLI** | `image-tools models get <id>` |

### `POST /vrooli.image_tools.v1.models.ModelsService/ListOperations`

Return the registry operation vocabulary in declaration order.

| | |
|---|---|
| **Response** | `ListOperationsResponse { operations: string[] }` |
| **CLI** | `image-tools models operations` |

### `POST /vrooli.image_tools.v1.models.ModelsService/SelectModel`

Preview which enabled model would run for an operation on the probed host
(honoring the per-op default and any override) without executing. Host
facts come from `vrooli host inventory` via `vrooli-cli-go`.

| | |
|---|---|
| **Request** | `SelectModelRequest { operation: string (required), override_id: string (optional) }` |
| **Response** | `SelectModelResponse { model: Model, gpu_viable: bool, reason: string, warnings: string[] }` |
| **Errors** | `invalid_argument` — unknown operation or invalid override<br>`failed_precondition` — no enabled model can run for the op on this host<br>`internal` — host probe or model-state load failure |
| **CLI** | `image-tools models select <operation> [--override <id>]` |

### `POST /vrooli.image_tools.v1.models.ModelsService/SetModelEnabled`

Toggle a model's runtime-enabled state, persisted in the SQLite overlay
over the read-only seed catalog.

| | |
|---|---|
| **Request** | `SetModelEnabledRequest { id: string, enabled: bool }` |
| **Response** | `SetModelEnabledResponse { model: Model }` |
| **Errors** | `not_found` — no model with that id<br>`internal` — model-state persistence failure |
| **CLI** | `image-tools models enable <id>` / `image-tools models enable <id> --disable` |

### `POST /vrooli.image_tools.v1.models.ModelsService/ListBlocklist`

Return the license-encumbered models excluded from the catalog, with the
reason each is excluded.

| | |
|---|---|
| **Response** | `ListBlocklistResponse { entries: BlocklistEntry[] }` |
| **CLI** | `image-tools models blocklist` |

---

## Adding a new endpoint

The endpoint surface is generated from the proto contracts and the API
module registry — never hand-edited into `.vrooli/endpoints.json`.

1. **Author the proto.** Add the RPC + messages to
   `packages/proto/schemas/image-tools/v1/<domain>/<domain>.proto`. Prefer
   a Connect-RPC method; only use a REST path with an allowed `RESTReason`
   (multipart upload, webhook receiver, third-party shape, ops probe). Run
   `cd packages/proto && make generate && make lint`.
2. **Implement the handler.** Add `api/handlers/<domain>/` with a
   `connect_handler.go` implementing the generated `*ServiceHandler`
   interface and a `module.go` exporting `Module(...)`, `Schema()`, and the
   `Endpoints` descriptors (reference the generated `*Procedure` constants
   so a renamed RPC breaks the build).
3. **Register it.** Add the domain to `api/internal/modules/registry.go`
   (`AllEndpoints`, `AllProtoFiles`, `AllSchemas`) and wire its runtime
   `Module(...)` into `main.go`'s `server.New(...)` call.
4. **Mirror the CLI.** Add the command(s) to `cli/manifest.json` + a
   `cli/domains/<domain>/` package, and add matching rows to
   `api/cmd/gen-endpoints/cli_commands_seed.json`. Run `make endpoints`.

The global parity test (`TestProtoConnectParity`) asserts every proto RPC
has exactly one endpoint descriptor; `gen-endpoints` rejects unjustified
REST paths and unseeded CLI commands.

## Cross-references

- [`cli-commands.md`](cli-commands.md) — the CLI commands these endpoints mirror
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — domain-module + transport architecture
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — the testability seams behind these handlers
- `packages/proto/schemas/image-tools/v1/` — the wire-contract source of truth
