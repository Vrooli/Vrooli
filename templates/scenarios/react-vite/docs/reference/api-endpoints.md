# API Endpoints — {{SCENARIO_DISPLAY_NAME}}

Human-readable reference for the HTTP API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. **When you add or change an endpoint, update both.** The CI
gate fails if the JSON drifts from the registered handlers or from
the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/{{SCENARIO_ID}}/v1/<domain>/<file>.proto`.
Tests, handlers, UI clients, and CLI handlers all consume the
generated types — no hand-written struct mirror exists to drift.

Errors share a single envelope shape (`packages/proto/schemas/{{SCENARIO_ID}}/v1/errors/errors.proto`):

```json
{ "code": "<canonical_code>", "message": "<human readable>", "details": [...] }
```

Canonical codes used today: `invalid_request` (400), `not_found` (404),
`internal` (500). Add to the proto enum when a new failure mode appears.

---

## System

### `GET /health`

Service health check. Returns API readiness plus dependency status.
Also mounted at `/api/v1/health` for client callers.

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

### `GET /api/v1/notes`

List notes, newest-first.

| | |
|---|---|
| **Auth** | None (template default; scenarios add auth as needed) |
| **Response** | `ListNotesResponse { notes: Note[] }` (capped at 100 by `notes.Service`) |
| **Errors** | `500 internal` — repository read failure |
| **CLI** | `{{SCENARIO_ID}} notes list` |

```bash
curl "http://localhost:${API_PORT}/api/v1/notes"
```

Empty results serialise to `{}` (proto3 default-omission); UI clients
should treat absent `notes` as the empty array.

### `POST /api/v1/notes`

Create a note.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `CreateNoteRequest { title: string (required), body: string (optional) }` |
| **Response** | `CreateNoteResponse { note: Note }` |
| **Errors** | `400 invalid_request` — missing/whitespace-only title, malformed JSON, or unknown fields (request decode uses `DisallowUnknownFields`)<br>`500 internal` — repository write failure |
| **CLI** | `{{SCENARIO_ID}} notes create --title <title> [--body <body>]` |

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/notes" \
  -H 'Content-Type: application/json' \
  -d '{"title":"first","body":"hello"}'
```

Title validation (non-empty after whitespace trim) lives in
`internal/notes/service.go`, **not** the handler. The handler only
translates `notes.ErrInvalidNote` into the 400 envelope.

### `GET /api/v1/notes/{id}`

Fetch a note by id.

| | |
|---|---|
| **Auth** | None (template default) |
| **Path params** | `id` — note identifier (UUID) |
| **Response** | `GetNoteResponse { note: Note }` |
| **Errors** | `404 not_found` — no note with that id<br>`500 internal` — repository read failure |
| **CLI** | `{{SCENARIO_ID}} notes get <id>` |

```bash
curl "http://localhost:${API_PORT}/api/v1/notes/abc123"
```

`notes.ErrNoteNotFound` returned by the service is translated into the
typed 404 envelope at the handler edge.

### `Note` shape

| Field | Type | Notes |
|---|---|---|
| `id` | string (UUID) | Server-generated |
| `title` | string | Required, non-empty after trim |
| `body` | string | Optional |
| `created_at` | string (RFC3339) | Server-set on create |
| `updated_at` | string (RFC3339) | Server-set on create / future update |

Defined in `packages/proto/schemas/{{SCENARIO_ID}}/v1/notes/notes.proto`.

---

## Adding a new endpoint

1. Add or extend the `.proto` message in
   `packages/proto/schemas/{{SCENARIO_ID}}/v1/<domain>/`. Run
   `make generate` to refresh the Go and TypeScript types.
2. If this is a new domain: create `internal/<domain>/{types,repository,sqlite,service}.go`
   following the notes layout.
3. Add the handler in `handlers/<domain>/handler.go`. Keep it thin —
   parse, validate via `DecodeJSON[T]`, call the service, translate
   typed sentinels to error envelopes, write the proto response.
4. Register the route in `internal/server/routes.go`.
5. Update [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) — path,
   method, summary, error codes (matching `httpx.Code*` constants the
   handler actually emits), and a `cli_mapping` pointing at the real
   subcommand.
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
