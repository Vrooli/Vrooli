# API Endpoints — Device Studio

## Availability and planned product surface

The project workspace RPCs and scaffold System/Notes operations below are callable. Broader Device Studio domains remain design intent: versioned device/scene/presentation/motion libraries, shot compile/preview, render jobs/cancel/results, creation attempts, scoped reviews and export. Each later domain owns its generated proto request/response names.

Compilation returns structured incompatibility reasons, supported repair choices and pinned input revisions; it does not launch AI. Render/creation admission is a distinct operation with idempotency, policy revision, explicit funding route and a cost bound when applicable. Reads never spawn agents. Long work returns a durable job identity; unknown remote outcome remains reconciling and cannot be replayed as a fresh paid request. Artifact access uses authorized project references, never arbitrary host paths. See [flows](../concepts/FLOWS.md), [data](../concepts/DATA.md), and [security](../internal/SECURITY.md).

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/device-studio/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/device-studio/v1/shared/errors.proto`):

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
| **Errors** | 200 when ready; 503 with readiness false when a critical dependency such as the database fails |
| **CLI** | `device-studio status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/device-studio/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Projects

Project data is stored in the scenario-owned SQLite database. Empty projects are valid. Names are trimmed, required, limited to 120 Unicode characters, and reject control characters. Lists default to 50 and cap at 100; page tokens carry a stable `(created_at,id)` descending cursor.

`ProjectsService` endpoints use Connect-RPC at `/vrooli.device_studio.v1.projects.ProjectsService/<Method>`:

| Method | Request / response | Errors | CLI |
|---|---|---|---|
| `ListProjects` | `ListProjectsRequest { page_size: int32, page_token: string }` → `ListProjectsResponse { projects: Project[], next_page_token: string }` | `invalid_argument` for malformed token; `internal` for storage failure | `device-studio projects list` |
| `CreateProject` | `CreateProjectRequest { name: string }` → `CreateProjectResponse { project: Project }` | `invalid_argument` for invalid name; `internal` for storage failure | `device-studio projects create --name <name>` |
| `GetProject` | `GetProjectRequest { id: string }` → `GetProjectResponse { project: Project }` | `not_found`, `invalid_argument`, `internal` | `device-studio projects get <id>` |
| `RenameProject` | `RenameProjectRequest { id: string, name: string, expected_revision: int64 }` → `RenameProjectResponse { project: Project }` | `not_found`, `invalid_argument`; `aborted` with `ProjectConflict { project_id, expected_revision, current_revision }` on stale revision; `internal` for storage failure | `device-studio projects rename <id> --name <name> --expected-revision <revision>` |

The bounded first-media slice supports local PNG/JPEG originals up to 24 MiB, 24 million pixels, and 20,000 pixels per side. It fully decodes uploads, rejects non-identity EXIF orientation and unsupported embedded color metadata, and stores immutable bytes separately from typed project metadata. `original_filename` is sanitized display metadata; internal storage keys and host paths never leave the API.

| Method | Request / response | Errors | CLI |
|---|---|---|---|
| `ListSourceMedia` | `ListSourceMediaRequest { project_id, page_size, page_token }` → `ListSourceMediaResponse { source_media[], next_page_token }` | `not_found`, `invalid_argument`, `internal` | `device-studio media list <project-id>` |
| `GetSourceMedia` | `GetSourceMediaRequest { project_id, id }` → `GetSourceMediaResponse { source_media }` | `not_found`, `invalid_argument`, `internal` | `device-studio media get <project-id> <id>` |
| Multipart import | `POST /api/v1/projects/{project_id}/media`, one `file` part → protojson `UploadSourceMediaResponse` | `not_found`, `invalid_request`, `internal` | `device-studio media import <project-id> --file <path>` |
| Source bytes | `GET /api/v1/projects/{project_id}/media/{media_id}/content` → opaque PNG/JPEG bytes | `not_found`, `internal` | UI/browser fetch by opaque IDs |

`Project` contains `id`, `name`, `created_at`, `updated_at`, and `revision`. Creation sets revision 1. Rename uses a single conditional SQLite `UPDATE ... RETURNING`; the write succeeds only when `expected_revision` matches and increments revision atomically.

## Domain endpoints — `<domain>`

Each product domain exposes its endpoints under
`POST /vrooli.device_studio.v1.<domain>.<Domain>Service/<Method>`
for proto-typed Connect-RPC calls, with REST exceptions (such as
multipart uploads) mounted at explicit REST paths. Document your
domain's endpoints here as you build them — one section per RPC, with
its auth, request/response proto shapes, error codes, and CLI mirror.

The scaffold ships one fully worked CRUD vertical slice as a copyable
reference (see the fenced example below); `template-manager detemplate
<scenario>` removes it once your real domains are green.

---

## Adding a new endpoint

For a new domain, copy the worked vertical slice in the fenced example
above first, then replace it once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/device-studio/v1/<domain>/`, then run
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

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars (e.g., `API_PORT`)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#contracts-and-data-flow) — proto bridge details
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns
