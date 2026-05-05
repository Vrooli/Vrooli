# API Endpoints — {{SCENARIO_DISPLAY_NAME}}

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/{{SCENARIO_ID}}/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/{{SCENARIO_ID}}/v1/errors/errors.proto`):

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
| **Errors** | None — always returns 200 with `status: "unhealthy"` if a dependency fails |
| **CLI** | `{{SCENARIO_ID}} status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/{{SCENARIO_ID}}/v1/health/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Notes (CRUD reference)

The `notes` domain is the canonical worked example. Copy its layering
when adding the first non-trivial mutation in your scenario.

### `POST /vrooli.{{SCENARIO_ID_SNAKE}}.v1.notes.NotesService/ListNotes`

List notes through the generated Connect-RPC service, newest-first.

| | |
|---|---|
| **Auth** | None (template default; scenarios add auth as needed) |
| **Response** | `ListNotesResponse { notes: Note[] }` (capped at 100 by `notes.Service`) |
| **Errors** | `500 internal` — repository read failure |
| **CLI** | `{{SCENARIO_ID}} notes list` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.{{SCENARIO_ID_SNAKE}}.v1.notes.NotesService/ListNotes" \
  -H 'Content-Type: application/json' \
  -d '{}'
```

UI and CLI code should normally use the generated client instead of
calling this path by hand.

### `POST /vrooli.{{SCENARIO_ID_SNAKE}}.v1.notes.NotesService/CreateNote`

Create a note through the generated Connect-RPC service.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `CreateNoteRequest { title: string (required), body: string (optional) }` |
| **Response** | `CreateNoteResponse { note: Note }` |
| **Errors** | `invalid_argument` — missing/whitespace-only title<br>`internal` — repository write failure |
| **CLI** | `{{SCENARIO_ID}} notes create --title <title> [--body <body>]` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.{{SCENARIO_ID_SNAKE}}.v1.notes.NotesService/CreateNote" \
  -H 'Content-Type: application/json' \
  -d '{"title":"first","body":"hello"}'
```

Title validation (non-empty after whitespace trim) lives in
`internal/notes/service.go`, **not** the handler. The Connect handler
only translates `notes.ErrInvalidNote` into `invalid_argument`.

### `POST /vrooli.{{SCENARIO_ID_SNAKE}}.v1.notes.NotesService/GetNote`

Fetch a note by id through the generated Connect-RPC service.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `GetNoteRequest { id: string }` |
| **Response** | `GetNoteResponse { note: Note }` |
| **Errors** | `not_found` — no note with that id<br>`internal` — repository read failure |
| **CLI** | `{{SCENARIO_ID}} notes get <id>` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.{{SCENARIO_ID_SNAKE}}.v1.notes.NotesService/GetNote" \
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
| **CLI** | `{{SCENARIO_ID}} notes attach <id> --file <path>` |

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

Defined in `packages/proto/schemas/{{SCENARIO_ID}}/v1/notes/notes.proto`.

---

## Adding a new endpoint

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/{{SCENARIO_ID}}/v1/<domain>/`. Run
   `make generate` to refresh generated types and Connect stubs.
2. If this is a new domain: create `internal/<domain>/{types,repository,sqlite,service}.go`
   following the notes layout.
3. Add the generated service implementation in
   `handlers/<domain>/connect_handler.go`. Keep it thin: call the
   service, translate typed sentinels to Connect errors, return generated
   proto responses.
4. Mount it in `handlers/<domain>/module.go` with
   `connectx.RegisterServices`.
5. Keep [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json)
   current by updating endpoint metadata and running `make endpoints`.
   Any endpoint with a `cli_mapping` also needs a matching
   `api/cmd/gen-endpoints/cli_commands_seed.json` row.
6. Update this document.
7. Add tests at every layer per [`internal/TESTING.md`](../internal/TESTING.md).
8. Add a row to [`internal/SEAMS.md`](../internal/SEAMS.md) if you
   introduced a new interface that production wires once and tests
   substitute.

The CI gate enforces (5) — every `cli_mapping` must name a registered
CLI command, and every endpoint must declare its error codes.

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars (e.g., `API_PORT`)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#proto-as-the-canonical-contract) — proto bridge details
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns
