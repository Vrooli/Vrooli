# API Endpoints — Data Backup Manager

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/data-backup-manager/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/data-backup-manager/v1/errors/errors.proto`):

```json
{ "code": "<canonical_code>", "message": "<human readable>", "details": [...] }
```

Canonical REST codes used today: `invalid_request` (400),
`not_found` (404), `internal` (500). Add to the proto enum when a new
REST-exception failure mode appears.

> **Status — planned contract.** Everything below the System section
> except the `notes` worked example describes the **intended** API
> surface for the locked design (targets, destinations, plans, runs,
> restores). The operations are committed; precise proto message fields
> firm up when the `.proto` files are authored under
> `packages/proto/schemas/data-backup-manager/v1/<domain>/`. Each is a
> Connect-RPC service method, not a REST/JSON literal. The `notes`
> section is the retained template worked example, not product scope.

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
| **CLI** | `data-backup-manager status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/data-backup-manager/v1/health/health.proto`
and mirrors `api-core/health.Response` field-for-field. For the real
product, `/health` additionally surfaces overdue/failed-backup posture
derived from run history.

---

## Targets (planned)

Self-registration of backup sources. All methods are Connect-RPC on
`vrooli.data_backup_manager.v1.targets.TargetsService`.

| Operation | Purpose | Auth | Notes |
|---|---|---|---|
| `RegisterTarget` | Idempotently upsert a target keyed by `owner + name`. | scenario/operator | Inputs: owner, name, source kind, locator, optional quiesce-hook refs. Re-registration on boot is the normal path. |
| `DeregisterTarget` | Remove a target by `owner + name`. | scenario/operator | Stops future runs from including it; does not delete existing snapshots. |
| `ListTargets` | List the catalog, optionally filtered by owner or source kind. | operator | Includes last-success-per-target. |
| `GetTarget` | Fetch one target by `owner + name`. | operator | `not_found` when absent. |

Source kind is one of: `filesystem`, `sqlite`, `postgres`, `redis`,
`qdrant`, `object_storage`. Requirements: OT-P0-001, OT-P0-002.

## Destinations (planned)

Backup destinations, each a kopia repository. Connect-RPC on
`vrooli.data_backup_manager.v1.destinations.DestinationsService`.

| Operation | Purpose | Auth | Notes |
|---|---|---|---|
| `CreateDestination` | Register a destination (backend `filesystem` or `s3`, cap, vault secret refs). | operator | Encrypted by default; rejects a path under the storage root it would protect (separate-root rule). |
| `ListDestinations` | List destinations with usage-versus-cap. | operator | Usage comes from kopia repo stats. |
| `GetDestination` | Fetch one destination, including current usage. | operator | |
| `UpdateDestination` | Change cap or metadata. | operator | Cap default is alert + block. |
| `DeleteDestination` | Remove a destination. | operator | Does not implicitly delete the underlying repository. |
| `CheckDestination` | Dry-run reachability/writability (P1, OT-P1-003). | operator | |

Requirements: OT-P0-003, OT-P0-007, OT-P0-008.

## Plans (planned)

Many-to-many bindings of targets to destinations with schedule and
retention. Connect-RPC on
`vrooli.data_backup_manager.v1.plans.PlansService`.

| Operation | Purpose | Auth | Notes |
|---|---|---|---|
| `CreatePlan` | Bind targets ↔ destinations with a schedule and retention. | operator | A target may belong to multiple plans. |
| `ListPlans` | List plans with next-run and membership. | operator | |
| `GetPlan` | Fetch one plan. | operator | |
| `UpdatePlan` | Change membership, schedule, or retention. | operator | GFS retention is OT-P1-002. |
| `DeletePlan` | Remove a plan. | operator | |

Requirements: OT-P0-004, OT-P0-005.

## Runs (planned)

Executions of plans and their history. Connect-RPC on
`vrooli.data_backup_manager.v1.runs.RunsService`.

| Operation | Purpose | Auth | Notes |
|---|---|---|---|
| `StartRun` | Trigger a plan run on demand. | operator/scenario | The in-process scheduler triggers the same path on cadence. |
| `ListRuns` | List run history, filterable by plan or target. | operator | Status: success / partial_failed / failed. |
| `GetRun` | Fetch one run with per-target outcomes and snapshot refs. | operator | |

Requirements: OT-P0-005, OT-P0-009, OT-P0-010.

## Restores (planned)

Restore and verified restore. Connect-RPC on
`vrooli.data_backup_manager.v1.restores.RestoresService`.

| Operation | Purpose | Auth | Notes |
|---|---|---|---|
| `StartRestore` | Restore a target's snapshot to a chosen location. | operator | Mode `restore`. |
| `VerifyRestore` | Test-restore to scratch and checksum; records last-verified. | operator | Mode `verify` — the gate before removing data from git. |
| `ListRestores` | List restore/verify history. | operator | Includes verify pass/fail and last-verified-per-target. |
| `GetRestore` | Fetch one restore/verify record. | operator | |

Requirements: OT-P0-006.

---

## Notes (CRUD reference)

The `notes` domain is the canonical worked example. Copy its layering
when adding the first non-trivial mutation in your scenario.

### `POST /vrooli.data_backup_manager.v1.notes.NotesService/ListNotes`

List notes through the generated Connect-RPC service, newest-first.

| | |
|---|---|
| **Auth** | None (template default; scenarios add auth as needed) |
| **Response** | `ListNotesResponse { notes: Note[] }` (capped at 100 by `notes.Service`) |
| **Errors** | `500 internal` — repository read failure |
| **CLI** | `data-backup-manager notes list` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.notes.NotesService/ListNotes" \
  -H 'Content-Type: application/json' \
  -d '{}'
```

UI and CLI code should normally use the generated client instead of
calling this path by hand.

### `POST /vrooli.data_backup_manager.v1.notes.NotesService/CreateNote`

Create a note through the generated Connect-RPC service.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `CreateNoteRequest { title: string (required), body: string (optional) }` |
| **Response** | `CreateNoteResponse { note: Note }` |
| **Errors** | `invalid_argument` — missing/whitespace-only title<br>`internal` — repository write failure |
| **CLI** | `data-backup-manager notes create --title <title> [--body <body>]` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.notes.NotesService/CreateNote" \
  -H 'Content-Type: application/json' \
  -d '{"title":"first","body":"hello"}'
```

Title validation (non-empty after whitespace trim) lives in
`internal/notes/service.go`, **not** the handler. The Connect handler
only translates `notes.ErrInvalidNote` into `invalid_argument`.

### `POST /vrooli.data_backup_manager.v1.notes.NotesService/GetNote`

Fetch a note by id through the generated Connect-RPC service.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `GetNoteRequest { id: string }` |
| **Response** | `GetNoteResponse { note: Note }` |
| **Errors** | `not_found` — no note with that id<br>`internal` — repository read failure |
| **CLI** | `data-backup-manager notes get <id>` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.notes.NotesService/GetNote" \
  -H 'Content-Type: application/json' \
  -d '{"id":"abc123"}'
```

`notes.ErrNoteNotFound` returned by the service is translated into the
typed `not_found` Connect error at the handler edge.

### `POST /api/v1/notes/{id}/attachments`

Upload opaque file bytes through the documented REST multipart exception.
The response is still proto-typed metadata.

| | |
|---|---|
| **Auth** | None (template default) |
| **Path params** | `id` — note identifier |
| **Request** | `multipart/form-data` with `file` part |
| **Response** | `UploadAttachmentResponse { attachment: Attachment }` |
| **Errors** | `400 invalid_request` — malformed multipart or missing file<br>`404 not_found` — no note with that id<br>`500 internal` — blob or metadata persistence failure |
| **CLI** | `data-backup-manager notes attach <id> --file <path>` |

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/notes/abc123/attachments" \
  -F file=@./example.png
```

### `Note` shape

| Field | Type | Notes |
|---|---|---|
| `id` | string (UUID) | Server-generated |
| `title` | string | Required, non-empty after trim |
| `body` | string | Optional |
| `created_at` | `google.protobuf.Timestamp` | Server-set on create |
| `updated_at` | `google.protobuf.Timestamp` | Server-set on create / future update |
| `attachment_keys` | `string[]` | Keys of uploaded note attachments |

Defined in `packages/proto/schemas/data-backup-manager/v1/notes/notes.proto`.

---

## Adding a new endpoint

For a new domain, copy the notes vertical slice first, then replace it
once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/data-backup-manager/v1/<domain>/`, then run
   `make generate`.
2. Implement the generated handler method in
   `handlers/<domain>/connect_handler.go`; keep it thin.
3. Update endpoint metadata in `handlers/<domain>/module.go`.
4. If the endpoint has a CLI mirror, update
   `api/cmd/gen-endpoints/cli_commands_seed.json`.
5. Run `make endpoints`; do not edit
   [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) by hand.
6. Update this document and add tests for the touched layers.
7. Add a row to [`internal/SEAMS.md`](../internal/SEAMS.md) if you
   introduced a new interface that production wires once and tests
   substitute.

The CI gate enforces endpoint-manifest freshness and command-seed
consistency.

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars (e.g., `API_PORT`)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#proto-as-the-canonical-contract) — proto bridge details
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns
