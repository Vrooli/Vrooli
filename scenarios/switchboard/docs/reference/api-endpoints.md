# API Endpoints — Switchboard

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/switchboard/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/switchboard/v1/shared/errors.proto`):

```json
{ "code": "<canonical_code>", "message": "<human readable>", "details": [...] }
```

Canonical REST codes used today: `invalid_request` (400),
`not_found` (404), `internal` (500). Add to the proto enum when a new
REST-exception failure mode appears.

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
| **Errors** | None — always returns 200 and reports an unhealthy status if a dependency fails |
| **CLI** | `switchboard status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/switchboard/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Domain endpoints — `<domain>`

Each product domain exposes its endpoints under
`POST /vrooli.switchboard.v1.<domain>.<Domain>Service/<Method>`
for proto-typed Connect-RPC calls, with REST exceptions (such as
multipart uploads) mounted at explicit REST paths. Document your
domain's endpoints here as you build them — one section per RPC, with
its auth, request/response proto shapes, error codes, and CLI mirror.

The scaffold ships one fully worked CRUD vertical slice as a copyable
reference (see the fenced example below); `template-manager detemplate
<scenario>` removes it once your real domains are green.

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The `notes` domain is the canonical worked example. Copy its layering
when adding the first non-trivial mutation in your scenario, then
remove it.

#### `POST /vrooli.switchboard.v1.notes.NotesService/ListNotes`

List notes through the generated Connect-RPC service, newest-first.

| | |
|---|---|
| **Auth** | None (template default; scenarios add auth as needed) |
| **Response** | `ListNotesResponse { notes: Note[] }` (capped at 100 by `notes.Service`) |
| **Errors** | `500 internal` — repository read failure |
| **CLI** | `switchboard notes list` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.switchboard.v1.notes.NotesService/ListNotes" \
  -H 'Content-Type: application/json' \
  -d '{}'
```

UI and CLI code should normally use the generated client instead of
calling this path by hand.

#### `POST /vrooli.switchboard.v1.notes.NotesService/CreateNote`

Create a note through the generated Connect-RPC service.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `CreateNoteRequest { title: string (required), body: string (optional) }` |
| **Response** | `CreateNoteResponse { note: Note }` |
| **Errors** | `invalid_argument` — missing/whitespace-only title<br>`internal` — repository write failure |
| **CLI** | `switchboard notes create --title <title> [--body <body>]` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.switchboard.v1.notes.NotesService/CreateNote" \
  -H 'Content-Type: application/json' \
  -d '{"title":"first","body":"hello"}'
```

Title validation (non-empty after whitespace trim) lives in
`internal/notes/service.go`, **not** the handler. The Connect handler
only translates `notes.ErrInvalidNote` into `invalid_argument`.

#### `POST /vrooli.switchboard.v1.notes.NotesService/GetNote`

Fetch a note by id through the generated Connect-RPC service.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `GetNoteRequest { id: string }` |
| **Response** | `GetNoteResponse { note: Note }` |
| **Errors** | `not_found` — no note with that id<br>`internal` — repository read failure |
| **CLI** | `switchboard notes get <id>` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.switchboard.v1.notes.NotesService/GetNote" \
  -H 'Content-Type: application/json' \
  -d '{"id":"abc123"}'
```

`notes.ErrNoteNotFound` returned by the service is translated into the
typed `not_found` Connect error at the handler edge.

#### `POST /api/v1/notes/{id}/attachments`

Upload opaque file bytes through the documented REST multipart exception.
The response is still proto-typed metadata.

| | |
|---|---|
| **Auth** | None (template default) |
| **Path params** | `id` — note identifier |
| **Request** | `multipart/form-data` with `file` part |
| **Response** | `UploadAttachmentResponse { attachment: Attachment }` |
| **Errors** | `400 invalid_request` — malformed multipart or missing file<br>`404 not_found` — no note with that id<br>`500 internal` — blob or metadata persistence failure |
| **CLI** | `switchboard notes attach <id> --file <path>` |

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/notes/abc123/attachments" \
  -F file=@./example.png
```

#### `Note` shape

| Field | Type | Notes |
|---|---|---|
| `id` | string (UUID) | Server-generated |
| `title` | string | Required, non-empty after trim |
| `body` | string | Optional |
| `created_at` | `google.protobuf.Timestamp` | Server-set on create |
| `updated_at` | `google.protobuf.Timestamp` | Server-set on create / future update |
| `attachment_keys` | `string[]` | Keys of uploaded note attachments |

Defined in `packages/proto/schemas/switchboard/v1/notes/notes.proto`.
<!-- EXAMPLE-DOMAIN:notes END -->

---

## Adding a new endpoint

For a new domain, copy the worked vertical slice in the fenced example
above first, then replace it once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/switchboard/v1/<domain>/`, then run
   `make generate`.
2. Implement the generated handler method in
   `handlers/<domain>/connect_handler.go`; keep it thin.
3. Update endpoint metadata in `handlers/<domain>/module.go`.
4. If the endpoint has a CLI mirror, bind it (or list it in `omitted[]`
   with a reason) in `cli/manifest.json` — the single source of truth for
   the CLI surface.
5. Run `make endpoints`; do not edit
   [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) by hand.
6. Update this document and add tests for the touched layers.
7. Add a row to [`internal/SEAMS.md`](../internal/SEAMS.md) if you
   introduced a new interface that production wires once and tests
   substitute.

The CI gate enforces endpoint-manifest freshness and the API↔CLI mapping
contract (every Connect endpoint is bound or omitted in `cli/manifest.json`).

## Planned surface (not yet implemented)

> **Status.** No domain code exists. Everything below is the intended Connect
> surface, authored documentation-first so the proto contracts can be reviewed
> before they are written. The `notes` reference above describes the template's
> example domain, which is real today and is deleted by
> `template-manager detemplate switchboard` once a real domain is green.
> Domain order is fixed in `docs/concepts/DOMAINS.md`.

All services live under `packages/proto/schemas/switchboard/v1/<domain>/`.

### `channels` — descriptors, availability, adapters

| Method | Purpose | Notes |
|---|---|---|
| `ListChannels` | Every descriptor with its availability state and reason | The console's channel catalogue. Ordered by `setup.friction` |
| `GetChannel` | One descriptor plus its live adapter state | — |
| `ReloadDescriptors` | Re-read and re-validate `data/channels/` | An invalid descriptor is an error here, never a silent skip |
| `ProbeAvailability` | Re-evaluate `requires` against host facts | Reaches `vrooli-bridge` and `tunnel-manager`. Never on the message path |

### `conversations` — threads, messages, media

| Method | Purpose | Notes |
|---|---|---|
| `ListThreads` | Threads across every channel, filterable | The unified conversation list |
| `GetThread` | One thread with its roster and paged messages | — |
| `AcceptInbound` | **Adapter-facing.** Submit a normalised envelope | De-duplicates on `(channel_id, remote_message_id)` and returns whether it was accepted or was a duplicate. This is the entry point for the whole scenario |
| `SendMessage` | Send on an existing thread | Egress goes back through the ingressing adapter |

### `agents` — bindings and roster

| Method | Purpose | Notes |
|---|---|---|
| `ListAgentBindings` | The roster projection, with each agent's channels | Agent detail is read from `prompt-manager`, never stored |
| `BindAgent` | Bind an agent to an address on a channel | Rejects an ambiguous binding rather than choosing |
| `UnbindAgent` | Remove a binding | Threads survive, marked orphaned |

### `trust` — contacts, tiers, resolution

| Method | Purpose | Notes |
|---|---|---|
| `ListContacts` | Known contacts and their tiers | An unrecognised sender is `stranger` and has no row until seen |
| `SetContactTier` | Assign a tier | **Refuses any attempt to reach an owner-only scope from a lower tier.** Structural, not validated |
| `ResolveScope` | Compute effective scope for a hypothetical turn | Read-only. Exists so the console can show the consequence of a tier change before it is saved |
| `ListResolutionLog` | The audit record of permission decisions | Append-only; outlives the threads it describes |

### `turns` — arbitration, budgets, approvals

| Method | Purpose | Notes |
|---|---|---|
| `GetTurn` | One turn and its state | States per the `FLOWS.md` machine |
| `ListApprovals` / `DecideApproval` | The owner's approval queue | Owner-only. Requests expire; they never wait indefinitely |
| `GetThreadBudget` / `SetThreadCap` | Turn budget and spend cap | Both fail closed |

### Conventions this surface must follow

- Every method is proto-owned; no hand-written REST except where the template
  already documents a multipart exception.
- **No method accepts or returns a channel identifier as a behavioral switch.**
  Channel identity appears as data, never as a branch condition.
- Errors map through the sentinel scheme in
  `docs/internal/ERROR-HANDLING.md`. A refusal is a typed outcome, not an error.
- No endpoint ever returns a credential value.

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars (e.g., `API_PORT`)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#proto-as-the-canonical-contract) — proto bridge details
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns
